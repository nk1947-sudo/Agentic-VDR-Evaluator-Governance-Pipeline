package tests

import (
	"testing"

	"github.com/nk1947-sudo/Agentic-VDR-Evaluator-Governance-Pipeline/services/orchestrator/models"
	"github.com/nk1947-sudo/Agentic-VDR-Evaluator-Governance-Pipeline/services/orchestrator/validator"
)

func TestSchemaValidator_ValidConversion(t *testing.T) {
	sv := validator.NewSchemaValidator()
	mr := validator.NewMathReconciler()

	arrVal := 38_000_000.0
	gmVal := 78.5 // 78.5%

	ext := &models.ExtractedFinancials{
		FiscalYear: 2023,
		PeriodType: "FY",
		Currency:   "USD",
		Revenue: models.MetricWithCitation{
			Value: 40_000_000.0,
			Unit:  "USD",
			Citation: models.Citation{
				PageNumber: 1,
				ExactQuote: "FY2023 revenue reached $40.0M",
			},
		},
		ARR: &models.MetricWithCitation{
			Value: arrVal,
			Unit:  "USD",
			Citation: models.Citation{
				PageNumber: 1,
				ExactQuote: "ARR is $38.0M",
			},
		},
		GrossMarginPct: &models.MetricWithCitation{
			Value: gmVal,
			Unit:  "PERCENT",
			Citation: models.Citation{
				PageNumber: 2,
				ExactQuote: "Gross margin 78.5%",
			},
		},
		EBITDA: models.MetricWithCitation{
			Value: 12_000_000.0,
			Unit:  "USD",
			Citation: models.Citation{
				PageNumber: 2,
				ExactQuote: "EBITDA achieved $12.0M",
			},
		},
	}

	record, err := sv.ConvertAndValidate("deal-123", ext)
	if err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}

	// Verify integer conversion in cents & bps
	if record.RevenueCents != 40_000_000_00 {
		t.Errorf("expected revenue cents 4000000000, got %d", record.RevenueCents)
	}
	if record.EBITDACents != 12_000_000_00 {
		t.Errorf("expected EBITDA cents 1200000000, got %d", record.EBITDACents)
	}
	if record.GrossMarginBps != 7850 {
		t.Errorf("expected gross margin bps 7850, got %d", record.GrossMarginBps)
	}

	// Accounting reconciliation check
	if err := mr.Reconcile(record); err != nil {
		t.Fatalf("unexpected accounting reconciliation error: %v", err)
	}
}

func TestSchemaValidator_MissingCitationFails(t *testing.T) {
	sv := validator.NewSchemaValidator()

	ext := &models.ExtractedFinancials{
		FiscalYear: 2023,
		Revenue: models.MetricWithCitation{
			Value: 20_000_000.0,
			Unit:  "USD",
			Citation: models.Citation{
				PageNumber: 0, // INVALID
				ExactQuote: "",
			},
		},
		EBITDA: models.MetricWithCitation{
			Value: 5_000_000.0,
			Unit:  "USD",
			Citation: models.Citation{
				PageNumber: 2,
				ExactQuote: "EBITDA was $5.0M",
			},
		},
	}

	_, err := sv.ConvertAndValidate("deal-123", ext)
	if err == nil {
		t.Fatal("expected citation error for missing revenue page/quote, got nil")
	}
}

func TestMathReconciler_EBITDAExceedsRevenueFails(t *testing.T) {
	mr := validator.NewMathReconciler()

	record := &models.FinancialMetricsRecord{
		DealID:       "deal-1",
		RevenueCents: 10_000_000_00, // $10M
		EBITDACents:  15_000_000_00, // $15M (Impossible)
	}

	if err := mr.Reconcile(record); err == nil {
		t.Fatal("expected math reconciliation error when EBITDA > Revenue, got nil")
	}
}
