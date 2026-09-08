import pytest
from pathlib import Path
from fastapi.testclient import TestClient

from main import app
from ingestion.pii_scrubber import PIIScrubber
from ingestion.pdf_fallback import LocalPDFLayoutEngine
from agents.extractor_agent import FinancialExtractorAgent
from agents.critic_agent import CriticAuditAgent

FIXTURES_DIR = Path("tests/fixtures")

@pytest.fixture
def test_client():
    return TestClient(app)

def test_e2e_clean_deal_auto_approves(test_client):
    pdf_path = FIXTURES_DIR / "deal_clean_enterprise_saas.pdf"
    assert pdf_path.exists(), "Clean deal PDF fixture must exist"

    with open(pdf_path, "rb") as f:
        resp = test_client.post(
            "/api/v1/process-document",
            files={"file": ("deal_clean_enterprise_saas.pdf", f, "application/pdf")},
            data={"deal_name": "Acme Enterprise Cloud"}
        )

    assert resp.status_code == 200, f"Processing failed: {resp.text}"
    data = resp.json()

    assert data["deal_name"] == "Acme Enterprise Cloud"
    assert data["pii_sanitized"] is True
    assert data["raw_text_pages_count"] >= 3

    # Financial verification
    assert len(data["financials"]) >= 1
    fin = data["financials"][0]
    assert fin["revenue"]["value"] == 40_000_000.0
    assert fin["revenue"]["citation"]["page_number"] == 1
    assert "revenue" in fin["revenue"]["citation"]["exact_quote"].lower()

    # EBITDA verification
    assert fin["ebitda"]["value"] == 10_000_000.0
    assert fin["ebitda"]["citation"]["page_number"] == 2

    # Audit verification
    audit = data["audit"]
    assert audit["is_grounded"] is True
    assert audit["grounding_score"] >= 0.85
    assert len(audit["discrepancies"]) == 0
    assert audit["recommendation"] == "FAST_TRACK_APPROVAL"

def test_e2e_adversarial_ebitda_contradiction_triggers_hitl(test_client):
    pdf_path = FIXTURES_DIR / "deal_adversarial_ebitda_contradiction.pdf"
    assert pdf_path.exists(), "Contradictory deal PDF fixture must exist"

    with open(pdf_path, "rb") as f:
        resp = test_client.post(
            "/api/v1/process-document",
            files={"file": ("deal_adversarial_ebitda_contradiction.pdf", f, "application/pdf")},
            data={"deal_name": "Apex Logistics Software"}
        )

    assert resp.status_code == 200
    data = resp.json()

    # The Critic Agent MUST detect the contradiction between Page 1 ($18M) and Page 4 ($11.2M)
    audit = data["audit"]
    assert audit["is_grounded"] is False
    assert len(audit["discrepancies"]) >= 1
    assert audit["recommendation"] == "REQUIRE_HUMAN_REVIEW"

    # Verify conflict details
    discrepancy = audit["discrepancies"][0]
    assert "EBITDA" in discrepancy["metric_name"]
    assert discrepancy["page_a"] == 1 or discrepancy["page_b"] == 4
    assert "18" in discrepancy["claimed_value"]
    assert "11,200,000" in discrepancy["found_conflict"] or "11.2" in discrepancy["explanation"]

def test_e2e_adversarial_customer_concentration_triggers_hitl(test_client):
    pdf_path = FIXTURES_DIR / "deal_adversarial_customer_concentration.pdf"
    assert pdf_path.exists(), "Customer concentration PDF fixture must exist"

    with open(pdf_path, "rb") as f:
        resp = test_client.post(
            "/api/v1/process-document",
            files={"file": ("deal_adversarial_customer_concentration.pdf", f, "application/pdf")},
            data={"deal_name": "OmniHealth Telemed"}
        )

    assert resp.status_code == 200
    data = resp.json()

    # Verify customer concentration was identified as a severe risk
    conc_risks = [r for r in data["risks"] if r["risk_category"] == "CUSTOMER_CONCENTRATION"]
    assert len(conc_risks) >= 1
    risk = conc_risks[0]
    assert risk["concentration_pct"] == 48.5
    assert risk["severity"] == "HIGH"
    assert risk["citation"]["page_number"] == 3

    # High concentration (>25%) penalizes grounding score and triggers review
    audit = data["audit"]
    assert any("48.5%" in note or "concentration" in note.lower() for note in audit["math_reconciliation_notes"])

def test_e2e_pii_adversarial_scrubbing_guarantee(test_client):
    pdf_path = FIXTURES_DIR / "deal_pii_adversarial.pdf"
    assert pdf_path.exists(), "PII PDF fixture must exist"

    with open(pdf_path, "rb") as f:
        resp = test_client.post(
            "/api/v1/process-document",
            files={"file": ("deal_pii_adversarial.pdf", f, "application/pdf")},
            data={"deal_name": "QuantumBio Analytics"}
        )

    assert resp.status_code == 200
    data = resp.json()

    assert data["pii_sanitized"] is True
    assert data["redacted_entities_count"] >= 3

    # Verify raw page text parsing
    with open(pdf_path, "rb") as f:
        raw_pages = LocalPDFLayoutEngine.extract_page_texts(f.read(), filename="deal_pii_adversarial.pdf")

    scrubber = PIIScrubber()
    sanitized_p1, count, mapping_hash, detected = scrubber.scrub(raw_pages[1])

    # Assert no sensitive PII survives in sanitized text
    assert "987-65-4321" not in sanitized_p1
    assert "sarah.jenkins.private@gmail.com" not in sanitized_p1
    assert "555" not in sanitized_p1 or "[REDACTED_PHONE_" in sanitized_p1
    assert "[REDACTED_SSN_" in sanitized_p1
    assert "[REDACTED_EMAIL_" in sanitized_p1
    assert len(mapping_hash) == 64
