package statemachine

import (
	"fmt"
	"strings"

	"github.com/nk1947-sudo/Agentic-VDR-Evaluator-Governance-Pipeline/services/orchestrator/models"
)

type GovernanceEvaluator struct {
	MinGroundingScore          float64
	MaxDiscrepancies           int
	CustomerConcentrationLimit float64
}

func NewGovernanceEvaluator(minScore float64, maxDisc int, concLimit float64) *GovernanceEvaluator {
	return &GovernanceEvaluator{
		MinGroundingScore:          minScore,
		MaxDiscrepancies:           maxDisc,
		CustomerConcentrationLimit: concLimit,
	}
}

// Evaluate checks whether an ingested deal's metrics and audit pass directly or trigger the HITL circuit breaker
func (ge *GovernanceEvaluator) Evaluate(resp *models.ProcessDocumentResponse) (bool, []string) {
	var reasons []string
	requiresHuman := false

	// 1. Audit Grounding Score Threshold Check
	if resp.Audit.GroundingScore < ge.MinGroundingScore {
		requiresHuman = true
		reasons = append(reasons, fmt.Sprintf(
			"Low grounding confidence score: %.2f (required threshold >= %.2f)",
			resp.Audit.GroundingScore, ge.MinGroundingScore,
		))
	}

	// 2. Critic Discrepancies Check
	if len(resp.Audit.Discrepancies) > ge.MaxDiscrepancies {
		requiresHuman = true
		for _, d := range resp.Audit.Discrepancies {
			reasons = append(reasons, fmt.Sprintf(
				"Critic flagged %s [%s]: %s (Claim: %s on P%d vs Conflict: %s on P%d)",
				d.MetricName, d.Severity, d.Explanation, d.ClaimedValue, d.PageA, d.FoundConflict, d.PageB,
			))
		}
	}

	// 3. Customer Concentration Risk Check
	for _, risk := range resp.Risks {
		if strings.EqualFold(risk.RiskCategory, "CUSTOMER_CONCENTRATION") && risk.ConcentrationPct != nil {
			if *risk.ConcentrationPct > ge.CustomerConcentrationLimit {
				requiresHuman = true
				reasons = append(reasons, fmt.Sprintf(
					"Severe customer concentration: %.1f%% exceeds safe investment threshold (%.1f%%)",
					*risk.ConcentrationPct, ge.CustomerConcentrationLimit,
				))
			}
		}
	}

	// 4. Recommendation check
	if resp.Audit.Recommendation == "REQUIRE_HUMAN_REVIEW" {
		requiresHuman = true
	}

	return requiresHuman, reasons
}
