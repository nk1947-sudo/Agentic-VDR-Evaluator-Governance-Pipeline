package models

import (
	"time"
)

type DealStatus string

const (
	StatusSubmitted           DealStatus = "SUBMITTED"
	StatusPIISanitized        DealStatus = "PII_SANITIZED"
	StatusAudited             DealStatus = "AUDITED"
	StatusAwaitingHumanReview DealStatus = "AWAITING_HUMAN_REVIEW"
	StatusApproved            DealStatus = "APPROVED"
	StatusCommitted           DealStatus = "COMMITTED"
	StatusRejected            DealStatus = "REJECTED"
)

type Deal struct {
	ID             string     `json:"id"`
	CompanyName    string     `json:"company_name"`
	TargetIndustry string     `json:"target_industry"`
	SourceFilename string     `json:"source_filename"`
	DocumentHash   string     `json:"document_hash"`
	Status         DealStatus `json:"status"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type Citation struct {
	PageNumber     int    `json:"page_number"`
	ExactQuote     string `json:"exact_quote"`
	TableReference string `json:"table_reference,omitempty"`
}

type MetricWithCitation struct {
	Value    float64  `json:"value"`
	Unit     string   `json:"unit"`
	Citation Citation `json:"citation"`
}

type ExtractedFinancials struct {
	FiscalYear           int                 `json:"fiscal_year"`
	PeriodType           string              `json:"period_type"`
	Currency             string              `json:"currency"`
	Revenue              MetricWithCitation  `json:"revenue"`
	ARR                  *MetricWithCitation `json:"arr,omitempty"`
	GrossProfit          *MetricWithCitation `json:"gross_profit,omitempty"`
	GrossMarginPct       *MetricWithCitation `json:"gross_margin_pct,omitempty"`
	EBITDA               MetricWithCitation  `json:"ebitda"`
	AdjustedEBITDA       *MetricWithCitation `json:"adjusted_ebitda,omitempty"`
	NetRetentionRatePct  *MetricWithCitation `json:"net_retention_rate_pct,omitempty"`
	CustomerChurnLogoPct *MetricWithCitation `json:"customer_churn_logo_pct,omitempty"`
}

// FinancialMetricsRecord represents the integer-exact database representation
type FinancialMetricsRecord struct {
	ID                  string    `json:"id"`
	DealID              string    `json:"deal_id"`
	FiscalYear          int       `json:"fiscal_year"`
	PeriodType          string    `json:"period_type"`
	Currency            string    `json:"currency"`
	RevenueCents        int64     `json:"revenue_cents"`
	ARRCents            int64     `json:"arr_cents"`
	GrossProfitCents    int64     `json:"gross_profit_cents"`
	GrossMarginBps      int       `json:"gross_margin_bps"` // 10,000 bps = 100.00%
	EBITDACents         int64     `json:"ebitda_cents"`
	AdjustedEBITDACents int64     `json:"adjusted_ebitda_cents"`
	NetRetentionRateBps int       `json:"net_retention_rate_bps"`
	CustomerChurnBps    int       `json:"customer_churn_bps"`
	CitationPage        int       `json:"citation_page"`
	CitationQuote       string    `json:"citation_quote"`
	CreatedAt           time.Time `json:"created_at"`
}

type DealRisk struct {
	ID                 string   `json:"id"`
	DealID             string   `json:"deal_id"`
	RiskCategory       string   `json:"risk_category"`
	Severity           string   `json:"severity"`
	Description        string   `json:"description"`
	ConcentrationPct   *float64 `json:"concentration_pct,omitempty"`
	Citation           Citation `json:"citation"`
	GroundingConfidence float64  `json:"grounding_confidence"`
}

type CriticDiscrepancy struct {
	MetricName   string `json:"metric_name"`
	ClaimedValue string `json:"claimed_value"`
	FoundConflict string `json:"found_conflict"`
	PageA        int    `json:"page_a"`
	QuoteA       string `json:"quote_a"`
	PageB        int    `json:"page_b"`
	QuoteB       string `json:"quote_b"`
	Severity     string `json:"severity"`
	Explanation  string `json:"explanation"`
}

type AuditPacket struct {
	GroundingScore          float64             `json:"grounding_score"`
	IsGrounded              bool                `json:"is_grounded"`
	Discrepancies           []CriticDiscrepancy `json:"discrepancies"`
	MathConsistencyPassed  bool                `json:"math_consistency_passed"`
	MathReconciliationNotes []string            `json:"math_reconciliation_notes"`
	CriticSummary           string              `json:"critic_summary"`
	Recommendation          string              `json:"recommendation"`
}

type ProcessDocumentResponse struct {
	DealName             string                `json:"deal_name"`
	PIISanitized         bool                  `json:"pii_sanitized"`
	RedactedEntitiesCount int                   `json:"redacted_entities_count"`
	Financials           []ExtractedFinancials `json:"financials"`
	Risks                []DealRisk            `json:"risks"`
	Audit                AuditPacket           `json:"audit"`
	RawTextPagesCount    int                   `json:"raw_text_pages_count"`
}

type ReviewDecision struct {
	Action          string             `json:"action"` // APPROVE, REJECT, OVERRIDE
	ReviewerName    string             `json:"reviewer_name"`
	ReviewToken     string             `json:"review_token"`
	Comment         string             `json:"comment"`
	OverrideRevenue *int64             `json:"override_revenue_cents,omitempty"`
	OverrideEBITDA  *int64             `json:"override_ebitda_cents,omitempty"`
}

type ReviewPacket struct {
	DealID        string                `json:"deal_id"`
	Status        DealStatus            `json:"status"`
	Audit         AuditPacket           `json:"audit"`
	Financials    []ExtractedFinancials `json:"financials"`
	Risks         []DealRisk            `json:"risks"`
	ReviewToken   string                `json:"review_token"`
	RequiresHuman bool                  `json:"requires_human_review"`
}
