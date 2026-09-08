package validator

import (
	"errors"
	"fmt"
	"math"

	"github.com/nk1947-sudo/Agentic-VDR-Evaluator-Governance-Pipeline/services/orchestrator/models"
)

type SchemaValidator struct{}

func NewSchemaValidator() *SchemaValidator {
	return &SchemaValidator{}
}

// ConvertAndValidate transforms raw float extractions into integer-exact database records and validates schema
func (v *SchemaValidator) ConvertAndValidate(
	dealID string,
	ext *models.ExtractedFinancials,
) (*models.FinancialMetricsRecord, error) {
	if dealID == "" {
		return nil, errors.New("schema validation failed: dealID is required")
	}

	if ext.FiscalYear < 1990 || ext.FiscalYear > 2050 {
		return nil, fmt.Errorf("schema validation failed: invalid fiscal year %d", ext.FiscalYear)
	}

	// 1. Validate Citations
	if ext.Revenue.Citation.PageNumber <= 0 || ext.Revenue.Citation.ExactQuote == "" {
		return nil, errors.New("audit violation: revenue metric lacks defensible citation (page/quote)")
	}
	if ext.EBITDA.Citation.PageNumber <= 0 || ext.EBITDA.Citation.ExactQuote == "" {
		return nil, errors.New("audit violation: EBITDA metric lacks defensible citation (page/quote)")
	}

	// 2. High-precision integer conversion (avoid floating point drift)
	revenueCents := int64(math.Round(ext.Revenue.Value * 100.0))
	if revenueCents <= 0 {
		return nil, errors.New("schema validation failed: revenue must be a positive integer in cents")
	}

	ebitdaCents := int64(math.Round(ext.EBITDA.Value * 100.0))

	var arrCents int64
	if ext.ARR != nil {
		arrCents = int64(math.Round(ext.ARR.Value * 100.0))
	}

	var grossProfitCents int64
	if ext.GrossProfit != nil {
		grossProfitCents = int64(math.Round(ext.GrossProfit.Value * 100.0))
	}

	// Gross margin in basis points (e.g. 78.5% -> 7850 bps)
	var grossMarginBps int
	if ext.GrossMarginPct != nil {
		grossMarginBps = int(math.Round(ext.GrossMarginPct.Value * 100.0))
		if grossMarginBps < -10000 || grossMarginBps > 10000 {
			return nil, fmt.Errorf("schema validation failed: gross margin %d bps out of safe bounds [-10000, 10000]", grossMarginBps)
		}
	}

	var adjEbitdaCents int64
	if ext.AdjustedEBITDA != nil {
		adjEbitdaCents = int64(math.Round(ext.AdjustedEBITDA.Value * 100.0))
	}

	var nrrBps int
	if ext.NetRetentionRatePct != nil {
		nrrBps = int(math.Round(ext.NetRetentionRatePct.Value * 100.0))
	}

	var churnBps int
	if ext.CustomerChurnLogoPct != nil {
		churnBps = int(math.Round(ext.CustomerChurnLogoPct.Value * 100.0))
	}

	record := &models.FinancialMetricsRecord{
		DealID:              dealID,
		FiscalYear:          ext.FiscalYear,
		PeriodType:          ext.PeriodType,
		Currency:            ext.Currency,
		RevenueCents:        revenueCents,
		ARRCents:            arrCents,
		GrossProfitCents:    grossProfitCents,
		GrossMarginBps:      grossMarginBps,
		EBITDACents:         ebitdaCents,
		AdjustedEBITDACents: adjEbitdaCents,
		NetRetentionRateBps: nrrBps,
		CustomerChurnBps:    churnBps,
		CitationPage:        ext.Revenue.Citation.PageNumber,
		CitationQuote:       ext.Revenue.Citation.ExactQuote,
	}

	return record, nil
}
