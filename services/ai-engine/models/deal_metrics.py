from typing import Optional, List
from pydantic import BaseModel, Field

class Citation(BaseModel):
    page_number: int = Field(..., description="Document page number where metric was identified")
    exact_quote: str = Field(..., description="Verbatim text quote from source document supporting this data")
    table_reference: Optional[str] = Field(None, description="Identifier or caption of table if metric was extracted from a table")

class MetricWithCitation(BaseModel):
    value: float = Field(..., description="Extracted numerical value (e.g. 15000000.0 for $15M)")
    unit: str = Field(default="USD", description="Unit of measurement: USD, PERCENT, MULTIPLE, etc.")
    citation: Citation

class FinancialMetrics(BaseModel):
    fiscal_year: int
    period_type: str = Field(default="FY", description="FY, LTM, or NTM")
    currency: str = Field(default="USD")
    revenue: MetricWithCitation
    arr: Optional[MetricWithCitation] = None
    gross_profit: Optional[MetricWithCitation] = None
    gross_margin_pct: Optional[MetricWithCitation] = None
    ebitda: MetricWithCitation
    adjusted_ebitda: Optional[MetricWithCitation] = None
    net_retention_rate_pct: Optional[MetricWithCitation] = None
    customer_churn_logo_pct: Optional[MetricWithCitation] = None

class DealRisk(BaseModel):
    risk_category: str = Field(..., description="CUSTOMER_CONCENTRATION, LITIGATION, REGULATORY, KEY_PERSON, SUPPLIER_DEPENDENCY")
    severity: str = Field(..., description="LOW, MEDIUM, HIGH, CRITICAL")
    description: str
    concentration_pct: Optional[float] = Field(None, description="Percentage of revenue/business concentrated in single account/factor")
    citation: Citation
    grounding_confidence: float = Field(default=1.0, ge=0.0, le=1.0)

class CriticDiscrepancy(BaseModel):
    metric_name: str
    claimed_value: str
    found_conflict: str
    page_a: int
    quote_a: str
    page_b: int
    quote_b: str
    severity: str = Field(default="HIGH", description="MEDIUM, HIGH, CRITICAL")
    explanation: str

class AuditPacket(BaseModel):
    grounding_score: float = Field(..., ge=0.0, le=1.0, description="Confidence score representing verification against source text")
    is_grounded: bool
    discrepancies: List[CriticDiscrepancy] = Field(default_factory=list)
    math_consistency_passed: bool = True
    math_reconciliation_notes: List[str] = Field(default_factory=list)
    critic_summary: str
    recommendation: str = Field(..., description="FAST_TRACK_APPROVAL or REQUIRE_HUMAN_REVIEW")

class ProcessDocumentResponse(BaseModel):
    deal_name: str
    pii_sanitized: bool
    redacted_entities_count: int
    financials: List[FinancialMetrics]
    risks: List[DealRisk]
    audit: AuditPacket
    raw_text_pages_count: int
