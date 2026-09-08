package api

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/nk1947-sudo/Agentic-VDR-Evaluator-Governance-Pipeline/services/orchestrator/config"
	"github.com/nk1947-sudo/Agentic-VDR-Evaluator-Governance-Pipeline/services/orchestrator/models"
	"github.com/nk1947-sudo/Agentic-VDR-Evaluator-Governance-Pipeline/services/orchestrator/statemachine"
	"github.com/nk1947-sudo/Agentic-VDR-Evaluator-Governance-Pipeline/services/orchestrator/storage"
	"github.com/nk1947-sudo/Agentic-VDR-Evaluator-Governance-Pipeline/services/orchestrator/validator"
)

type DealHandler struct {
	cfg             *config.Config
	storage         storage.Storage
	tokenManager    *statemachine.TokenManager
	workflowMachine *statemachine.WorkflowMachine
	evaluator       *statemachine.GovernanceEvaluator
	schemaValidator *validator.SchemaValidator
	mathReconciler  *validator.MathReconciler
	snowflake       *storage.MockSnowflakeStaging
	// In-memory cache for pending review packets
	pendingReviews map[string]*models.ProcessDocumentResponse
}

func NewDealHandler(
	cfg *config.Config,
	st storage.Storage,
	tm *statemachine.TokenManager,
	wm *statemachine.WorkflowMachine,
	ev *statemachine.GovernanceEvaluator,
	sv *validator.SchemaValidator,
	mr *validator.MathReconciler,
	sf *storage.MockSnowflakeStaging,
) *DealHandler {
	return &DealHandler{
		cfg:             cfg,
		storage:         st,
		tokenManager:    tm,
		workflowMachine: wm,
		evaluator:       ev,
		schemaValidator: sv,
		mathReconciler:  mr,
		snowflake:       sf,
		pendingReviews:  make(map[string]*models.ProcessDocumentResponse),
	}
}

func (h *DealHandler) HandleIngest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 1. Parse Multipart Form (max 50MB)
	err := r.ParseMultipartForm(50 << 20)
	if err != nil {
		http.Error(w, "Unable to parse form: "+err.Error(), http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("document")
	if err != nil {
		http.Error(w, "Document file is required: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Failed to read file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	dealName := r.FormValue("company_name")
	if dealName == "" {
		dealName = "Confidential Target"
	}
	industry := r.FormValue("target_industry")
	if industry == "" {
		industry = "Enterprise SaaS"
	}

	// 2. Compute Document Hash & Initialize Deal
	hasher := sha256.New()
	hasher.Write(fileBytes)
	docHash := hex.EncodeToString(hasher.Sum(nil))

	dealID := uuid.New().String()
	deal := &models.Deal{
		ID:             dealID,
		CompanyName:    dealName,
		TargetIndustry: industry,
		SourceFilename: header.Filename,
		DocumentHash:   docHash,
		Status:         models.StatusSubmitted,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	_ = h.storage.SaveDeal(deal)

	// 3. Dispatch to AI Engine for OCR, PII Scrub, and Dual-Agent Audit
	aiResp, err := h.callAIEngine(fileBytes, header.Filename, dealName)
	if err != nil {
		log.Printf("AI Engine error: %v", err)
		http.Error(w, "AI Engine processing failed: "+err.Error(), http.StatusBadGateway)
		return
	}

	// Transition deal through PII and Audit stages
	_ = h.workflowMachine.Transition(deal, models.StatusPIISanitized, "", "")
	_ = h.workflowMachine.Transition(deal, models.StatusAudited, "", "")

	// 4. Governance Evaluation (Circuit Breaker Gate)
	requiresHuman, reasons := h.evaluator.Evaluate(aiResp)

	if requiresHuman {
		// CIRCUIT BREAKER TRIGGERED
		log.Printf("Deal %s triggered HITL circuit breaker. Reasons: %v", dealID, reasons)
		deal.Status = models.StatusAwaitingHumanReview
		_ = h.storage.UpdateDealStatus(dealID, models.StatusAwaitingHumanReview)

		// Generate cryptographically signed review token
		reviewToken := h.tokenManager.GenerateReviewToken(dealID, 60*time.Minute)

		// Save audit log entry
		_ = h.storage.SaveAuditLog(
			dealID, "GOVERNANCE_GATE", "CriticAuditAgent", "CIRCUIT_BREAKER_TRIGGERED",
			aiResp.Audit.GroundingScore, len(aiResp.Audit.Discrepancies), true,
			"", "AWAITING_HUMAN_REVIEW", fmt.Sprintf("Circuit breaker reasons: %v", reasons),
		)

		// Store pending extraction payload in memory for review
		h.pendingReviews[dealID] = aiResp

		reviewPacket := models.ReviewPacket{
			DealID:        dealID,
			Status:        models.StatusAwaitingHumanReview,
			Audit:         aiResp.Audit,
			Financials:    aiResp.Financials,
			Risks:         aiResp.Risks,
			ReviewToken:   reviewToken,
			RequiresHuman: true,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted) // 202 Accepted: Processing halted awaiting approval
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"message":          "Human-In-The-Loop review required. Automated database write blocked.",
			"reasons":          reasons,
			"review_packet":    reviewPacket,
			"approval_endpoint": fmt.Sprintf("/api/v1/deals/%s/decision", dealID),
		})
		return
	}

	// 5. Clean Deal - Fast Track Auto Approval & Schema Validation
	_ = h.workflowMachine.Transition(deal, models.StatusApproved, "", "")

	// Validate Schema & Accounting Sanity
	var records []*models.FinancialMetricsRecord
	for _, fin := range aiResp.Financials {
		rec, valErr := h.schemaValidator.ConvertAndValidate(dealID, &fin)
		if valErr != nil {
			http.Error(w, "Schema validation failed: "+valErr.Error(), http.StatusUnprocessableEntity)
			return
		}
		if reconErr := h.mathReconciler.Reconcile(rec); reconErr != nil {
			http.Error(w, "Accounting reconciliation failed: "+reconErr.Error(), http.StatusUnprocessableEntity)
			return
		}
		records = append(records, rec)
	}

	// Commit to Golden Database
	_ = h.storage.SaveFinancials(records)
	for i := range aiResp.Risks {
		aiResp.Risks[i].DealID = dealID
	}
	_ = h.storage.SaveRisks(aiResp.Risks)

	// Stage to Snowflake data warehouse mock
	stagedPath, _ := h.snowflake.StageDealMetrics(deal, records, aiResp.Risks)

	_ = h.workflowMachine.Transition(deal, models.StatusCommitted, "", "")
	_ = h.storage.UpdateDealStatus(dealID, models.StatusCommitted)

	_ = h.storage.SaveAuditLog(
		dealID, "AUTO_APPROVAL", "GoGovernanceOrchestrator", "COMMITTED",
		aiResp.Audit.GroundingScore, 0, false, "SystemAutoPilot", "APPROVED", "Passed all governance guardrails",
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"message":               "Deal evaluated, verified, and committed to golden database successfully.",
		"deal_id":               dealID,
		"status":                models.StatusCommitted,
		"grounding_score":       aiResp.Audit.GroundingScore,
		"committed_records":     len(records),
		"snowflake_staged_path": stagedPath,
	})
}

func (h *DealHandler) callAIEngine(fileBytes []byte, filename, dealName string) (*models.ProcessDocumentResponse, error) {
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, err
	}
	_, err = part.Write(fileBytes)
	if err != nil {
		return nil, err
	}

	_ = writer.WriteField("deal_name", dealName)
	_ = writer.Close()

	aiURL := fmt.Sprintf("%s/api/v1/process-document", h.cfg.AIEngineURL)
	req, err := http.NewRequest(http.MethodPost, aiURL, &requestBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{Timeout: 90 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ai engine returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var aiResp models.ProcessDocumentResponse
	if err := json.NewDecoder(resp.Body).Decode(&aiResp); err != nil {
		return nil, err
	}

	return &aiResp, nil
}
