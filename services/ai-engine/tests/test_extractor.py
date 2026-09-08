import pytest
from agents.extractor_agent import FinancialExtractorAgent

def test_extractor_deterministic():
    agent = FinancialExtractorAgent()
    pages = {
        1: "Executive Summary: Acme Cloud Solutions reported Revenue of $40.0M and ARR of $35.0M.",
        2: "Financial Profile: In FY2023, EBITDA reached $10.0M with an 80% gross margin.",
        3: "Customer Risk: Our top customer represents 14% of total ARR, indicating modest concentration."
    }

    financials, risks = agent.extract(pages, deal_name="Acme Cloud")

    assert len(financials) >= 1
    fin = financials[0]
    assert fin.revenue.value == 40_000_000.0
    assert fin.revenue.citation.page_number == 1
    assert "revenue" in fin.revenue.citation.exact_quote.lower()

    assert fin.ebitda.value == 10_000_000.0
    assert fin.ebitda.citation.page_number == 2

    assert len(risks) >= 1
    risk = risks[0]
    assert risk.risk_category == "CUSTOMER_CONCENTRATION"
    assert risk.concentration_pct == 14.0
    assert risk.citation.page_number == 3
