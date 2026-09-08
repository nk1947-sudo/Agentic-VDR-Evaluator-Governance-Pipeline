package validator

import (
	"errors"
	"fmt"

	"github.com/nk1947-sudo/Agentic-VDR-Evaluator-Governance-Pipeline/services/orchestrator/models"
)

type MathReconciler struct{}

func NewMathReconciler() *MathReconciler {
	return &MathReconciler{}
}

// Reconcile performs accounting sanity checks between financial statements
func (mr *MathReconciler) Reconcile(record *models.FinancialMetricsRecord) error {
	// EBITDA cannot wildly exceed Revenue (EBITDA is an operating earnings metric)
	if record.EBITDACents > record.RevenueCents {
		return fmt.Errorf(
			"accounting violation: EBITDA ($%d cents) exceeds total revenue ($%d cents)",
			record.EBITDACents, record.RevenueCents,
		)
	}

	// If Gross Profit is reported, Gross Profit <= Revenue
	if record.GrossProfitCents > 0 && record.GrossProfitCents > record.RevenueCents {
		return errors.New("accounting violation: Gross Profit exceeds Total Revenue")
	}

	return nil
}
