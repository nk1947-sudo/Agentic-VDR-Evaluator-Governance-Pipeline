package tests

import (
	"testing"
	"time"

	"github.com/nk1947-sudo/Agentic-VDR-Evaluator-Governance-Pipeline/services/orchestrator/models"
	"github.com/nk1947-sudo/Agentic-VDR-Evaluator-Governance-Pipeline/services/orchestrator/statemachine"
)

func TestTokenManager_GenerateAndValidate(t *testing.T) {
	secret := "test-secret-key-12345"
	tm := statemachine.NewTokenManager(secret)
	dealID := "deal-uuid-101"

	// Valid token test
	token := tm.GenerateReviewToken(dealID, 10*time.Minute)
	if err := tm.ValidateReviewToken(token, dealID); err != nil {
		t.Fatalf("expected valid token, got error: %v", err)
	}

	// Mismatched deal ID test
	if err := tm.ValidateReviewToken(token, "different-deal-id"); err == nil {
		t.Fatal("expected error on deal ID mismatch, got nil")
	}

	// Tampered signature test
	tamperedToken := token + "bad"
	if err := tm.ValidateReviewToken(tamperedToken, dealID); err == nil {
		t.Fatal("expected error on tampered signature, got nil")
	}

	// Expired token test
	expiredToken := tm.GenerateReviewToken(dealID, -1*time.Minute)
	if err := tm.ValidateReviewToken(expiredToken, dealID); err == nil {
		t.Fatal("expected error on expired token, got nil")
	}
}

func TestWorkflowMachine_CircuitBreakerEnforcement(t *testing.T) {
	tm := statemachine.NewTokenManager("test-secret")
	wm := statemachine.NewWorkflowMachine(tm)
	dealID := "deal-abc-999"

	deal := &models.Deal{
		ID:     dealID,
		Status: models.StatusSubmitted,
	}

	// Legal advance
	if err := wm.Transition(deal, models.StatusPIISanitized, "", ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := wm.Transition(deal, models.StatusAudited, "", ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := wm.Transition(deal, models.StatusAwaitingHumanReview, "", ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Try committing WITHOUT token -> MUST FAIL (Circuit Breaker)
	err := wm.Transition(deal, models.StatusCommitted, "", "Reviewer Jane")
	if err == nil {
		t.Fatal("governance breach: allowed transition to COMMITTED without review token!")
	}

	// Try committing WITHOUT reviewer -> MUST FAIL
	validToken := tm.GenerateReviewToken(dealID, 15*time.Minute)
	err = wm.Transition(deal, models.StatusCommitted, validToken, "")
	if err == nil {
		t.Fatal("governance breach: allowed transition to COMMITTED without reviewer identity!")
	}

	// Legitimate human review release -> MUST PASS
	err = wm.Transition(deal, models.StatusCommitted, validToken, "Partner John")
	if err != nil {
		t.Fatalf("unexpected error on legitimate human approval: %v", err)
	}
	if deal.Status != models.StatusCommitted {
		t.Fatalf("expected status COMMITTED, got %s", deal.Status)
	}
}

func TestGovernanceEvaluator_Triggers(t *testing.T) {
	eval := statemachine.NewGovernanceEvaluator(0.85, 0, 25.0)

	// Clean deal
	cleanResp := &models.ProcessDocumentResponse{
		Audit: models.AuditPacket{
			GroundingScore: 0.95,
			Discrepancies:  nil,
			Recommendation: "FAST_TRACK_APPROVAL",
		},
		Risks: nil,
	}
	requiresHuman, reasons := eval.Evaluate(cleanResp)
	if requiresHuman {
		t.Fatalf("expected clean deal to pass, got reasons: %v", reasons)
	}

	// Discrepancy deal
	discrepancyResp := &models.ProcessDocumentResponse{
		Audit: models.AuditPacket{
			GroundingScore: 0.70,
			Discrepancies: []models.CriticDiscrepancy{
				{MetricName: "EBITDA", ClaimedValue: "$18M", FoundConflict: "$11M", PageA: 2, PageB: 14},
			},
			Recommendation: "REQUIRE_HUMAN_REVIEW",
		},
	}
	requiresHuman, reasons = eval.Evaluate(discrepancyResp)
	if !requiresHuman {
		t.Fatal("expected discrepancy to trigger human review gate")
	}
	if len(reasons) < 2 {
		t.Fatalf("expected multiple reasons (score + discrepancy), got %v", reasons)
	}

	// High Customer Concentration deal (> 25%)
	highConc := 42.5
	concResp := &models.ProcessDocumentResponse{
		Audit: models.AuditPacket{
			GroundingScore: 0.90,
			Recommendation: "FAST_TRACK_APPROVAL",
		},
		Risks: []models.DealRisk{
			{
				RiskCategory:     "CUSTOMER_CONCENTRATION",
				ConcentrationPct: &highConc,
			},
		},
	}
	requiresHuman, reasons = eval.Evaluate(concResp)
	if !requiresHuman {
		t.Fatal("expected >25% customer concentration to trigger human review gate")
	}
}
