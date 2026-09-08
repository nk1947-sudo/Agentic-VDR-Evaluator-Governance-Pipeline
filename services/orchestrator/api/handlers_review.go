package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/nk1947-sudo/Agentic-VDR-Evaluator-Governance-Pipeline/services/orchestrator/models"
)

func (h *DealHandler) HandleReview(w http.ResponseWriter, r *http.Request) {
	// Expected URL pattern: /api/v1/deals/{id}/review
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 4 {
		http.Error(w, "Invalid deal review URL", http.StatusBadRequest)
		return
	}
	dealID := parts[3]

	deal, err := h.storage.GetDeal(dealID)
	if err != nil {
		http.Error(w, "Deal not found: "+err.Error(), http.StatusNotFound)
		return
	}

	cached, exists := h.pendingReviews[dealID]
	if !exists {
		// If deal is already committed or rejected, return its database state
		finances, _ := h.storage.GetFinancials(dealID)
		risks, _ := h.storage.GetRisks(dealID)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"deal_id":    deal.ID,
			"status":     deal.Status,
			"financials": finances,
			"risks":      risks,
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"deal_id":            deal.ID,
		"company_name":       deal.CompanyName,
		"status":             deal.Status,
		"audit":              cached.Audit,
		"financials":         cached.Financials,
		"risks":              cached.Risks,
		"redacted_entities":  cached.RedactedEntitiesCount,
	})
}

func (h *DealHandler) HandleDecision(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Expected URL pattern: /api/v1/deals/{id}/decision
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 4 {
		http.Error(w, "Invalid decision URL", http.StatusBadRequest)
		return
	}
	dealID := parts[3]

	var decision models.ReviewDecision
	if err := json.NewDecoder(r.Body).Decode(&decision); err != nil {
		http.Error(w, "Invalid decision payload: "+err.Error(), http.StatusBadRequest)
		return
	}

	deal, err := h.storage.GetDeal(dealID)
	if err != nil {
		http.Error(w, "Deal not found: "+err.Error(), http.StatusNotFound)
		return
	}

	if deal.Status != models.StatusAwaitingHumanReview {
		http.Error(w, fmt.Sprintf("Deal is not currently awaiting review (status: %s)", deal.Status), http.StatusConflict)
		return
	}

	// 1. Verify Cryptographic Review Token
	if err := h.tokenManager.ValidateReviewToken(decision.ReviewToken, dealID); err != nil {
		http.Error(w, "Governance token rejection: "+err.Error(), http.StatusForbidden)
		return
	}

	// 2. Handle Rejection
	if strings.EqualFold(decision.Action, "REJECT") {
		_ = h.workflowMachine.Transition(deal, models.StatusRejected, decision.ReviewToken, decision.ReviewerName)
		_ = h.storage.UpdateDealStatus(dealID, models.StatusRejected)
		_ = h.storage.SaveAuditLog(
			dealID, "HUMAN_REVIEW", "InvestmentCommittee", "REJECTED",
			0.0, 0, true, decision.ReviewerName, "REJECT", decision.Comment,
		)

		delete(h.pendingReviews, dealID)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"message":  "Deal rejected by reviewer.",
			"deal_id":  dealID,
			"status":   models.StatusRejected,
			"reviewer": decision.ReviewerName,
		})
		return
	}

	// 3. Handle Approval or Override
	if strings.EqualFold(decision.Action, "APPROVE") || strings.EqualFold(decision.Action, "OVERRIDE") {
		cached, exists := h.pendingReviews[dealID]
		if !exists {
			http.Error(w, "Extraction payload not found for pending deal", http.StatusInternalServerError)
			return
		}

		var records []*models.FinancialMetricsRecord
		for _, fin := range cached.Financials {
			rec, valErr := h.schemaValidator.ConvertAndValidate(dealID, &fin)
			if valErr != nil {
				http.Error(w, "Schema validation error: "+valErr.Error(), http.StatusUnprocessableEntity)
				return
			}

			// Apply human overrides if specified
			if decision.OverrideRevenue != nil && *decision.OverrideRevenue > 0 {
				rec.RevenueCents = *decision.OverrideRevenue
			}
			if decision.OverrideEBITDA != nil {
				rec.EBITDACents = *decision.OverrideEBITDA
			}

			if reconErr := h.mathReconciler.Reconcile(rec); reconErr != nil {
				http.Error(w, "Accounting reconciliation error after human override: "+reconErr.Error(), http.StatusUnprocessableEntity)
				return
			}
			records = append(records, rec)
		}

		// Commit records to PostgreSQL
		_ = h.storage.SaveFinancials(records)
		for i := range cached.Risks {
			cached.Risks[i].DealID = dealID
		}
		_ = h.storage.SaveRisks(cached.Risks)

		// Stage to mock Snowflake
		stagedPath, _ := h.snowflake.StageDealMetrics(deal, records, cached.Risks)

		// Transition state machine
		_ = h.workflowMachine.Transition(deal, models.StatusCommitted, decision.ReviewToken, decision.ReviewerName)
		_ = h.storage.UpdateDealStatus(dealID, models.StatusCommitted)

		// Record immutable audit entry
		_ = h.storage.SaveAuditLog(
			dealID, "HUMAN_APPROVAL", "InvestmentCommittee", "COMMITTED",
			cached.Audit.GroundingScore, len(cached.Audit.Discrepancies), true,
			decision.ReviewerName, decision.Action, decision.Comment,
		)

		delete(h.pendingReviews, dealID)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"message":               "Human approval verified. Deal released from circuit breaker and committed.",
			"deal_id":               dealID,
			"status":                models.StatusCommitted,
			"reviewer":              decision.ReviewerName,
			"action":                decision.Action,
			"committed_records":     len(records),
			"snowflake_staged_path": stagedPath,
		})
		return
	}

	http.Error(w, "Unknown review action: "+decision.Action, http.StatusBadRequest)
}
