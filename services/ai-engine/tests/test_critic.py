import pytest
from agents.critic_agent import CriticAuditAgent
from models.deal_metrics import FinancialMetrics, MetricWithCitation, Citation, DealRisk

def test_critic_clean_deal():
    critic = CriticAuditAgent()
    pages = {
        1: "Company Overview: Enterprise SaaS revenue is $50.0M for FY2023.",
        2: "Profitability: Verified EBITDA is $12.0M with clean cash conversion.",
    }
    financials = [
        FinancialMetrics(
            fiscal_year=2023,
            revenue=MetricWithCitation(value=50_000_000.0, citation=Citation(page_number=1, exact_quote="Enterprise SaaS revenue is $50.0M for FY2023.")),
            ebitda=MetricWithCitation(value=12_000_000.0, citation=Citation(page_number=2, exact_quote="Verified EBITDA is $12.0M with clean cash conversion."))
        )
    ]
    risks = []

    audit = critic.audit(pages, financials, risks)
    assert audit.is_grounded is True
    assert audit.grounding_score >= 0.85
    assert len(audit.discrepancies) == 0
    assert audit.recommendation == "FAST_TRACK_APPROVAL"

def test_critic_contradiction_detection():
    critic = CriticAuditAgent()
    pages = {
        1: "Executive Summary: Revenue is reported at $60.0M. Management claims adjusted EBITDA of $18.0M for FY2023.",
        14: "Appendix Footnote 4: Audited financial statements confirm GAAP EBITDA is $11.2M.",
    }
    financials = [
        FinancialMetrics(
            fiscal_year=2023,
            revenue=MetricWithCitation(value=60_000_000.0, citation=Citation(page_number=1, exact_quote="Revenue is reported at $60.0M")),
            ebitda=MetricWithCitation(value=18_000_000.0, citation=Citation(page_number=1, exact_quote="adjusted EBITDA of $18.0M for FY2023"))
        )
    ]
    risks = []

    audit = critic.audit(pages, financials, risks)
    assert audit.is_grounded is False
    assert len(audit.discrepancies) > 0
    assert audit.recommendation == "REQUIRE_HUMAN_REVIEW"
    assert any("EBITDA" in d.metric_name for d in audit.discrepancies)
