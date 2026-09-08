package statemachine

import (
	"errors"
	"fmt"
	"time"

	"github.com/nk1947-sudo/Agentic-VDR-Evaluator-Governance-Pipeline/services/orchestrator/models"
)

type WorkflowMachine struct {
	tokenManager *TokenManager
}

func NewWorkflowMachine(tm *TokenManager) *WorkflowMachine {
	return &WorkflowMachine{
		tokenManager: tm,
	}
}

// Transition validates and updates the deal's workflow state
func (wm *WorkflowMachine) Transition(
	deal *models.Deal,
	target models.DealStatus,
	token string,
	reviewer string,
) error {
	current := deal.Status

	// Allowed transitions matrix
	switch current {
	case models.StatusSubmitted:
		if target != models.StatusPIISanitized && target != models.StatusRejected {
			return fmt.Errorf("invalid transition from %s to %s", current, target)
		}
	case models.StatusPIISanitized:
		if target != models.StatusAudited && target != models.StatusRejected {
			return fmt.Errorf("invalid transition from %s to %s", current, target)
		}
	case models.StatusAudited:
		if target != models.StatusAwaitingHumanReview && target != models.StatusApproved && target != models.StatusRejected {
			return fmt.Errorf("invalid transition from %s to %s", current, target)
		}
	case models.StatusAwaitingHumanReview:
		// CIRCUIT BREAKER ENFORCEMENT:
		// Moving from Awaiting Human Review requires explicit valid token and reviewer!
		if target != models.StatusCommitted && target != models.StatusRejected && target != models.StatusApproved {
			return fmt.Errorf("invalid transition from %s to %s", current, target)
		}
		if reviewer == "" {
			return errors.New("governance violation: reviewer name required to exit AWAITING_HUMAN_REVIEW")
		}
		if token == "" {
			return errors.New("governance violation: valid review token required to release circuit breaker")
		}
		if err := wm.tokenManager.ValidateReviewToken(token, deal.ID); err != nil {
			return fmt.Errorf("governance violation: %w", err)
		}
	case models.StatusApproved:
		if target != models.StatusCommitted && target != models.StatusRejected {
			return fmt.Errorf("invalid transition from %s to %s", current, target)
		}
	case models.StatusCommitted, models.StatusRejected:
		return fmt.Errorf("deal in terminal state %s cannot transition further", current)
	default:
		return fmt.Errorf("unknown current status %s", current)
	}

	deal.Status = target
	deal.UpdatedAt = time.Now()
	return nil
}
