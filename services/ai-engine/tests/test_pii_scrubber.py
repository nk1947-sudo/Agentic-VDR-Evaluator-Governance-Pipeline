import pytest
from ingestion.pii_scrubber import PIIScrubber

def test_pii_scrubber_redacts_ssn_email_phone():
    scrubber = PIIScrubber()
    sample_text = (
        "Founder John Doe (SSN: 123-45-6789) can be reached at john.doe@example.com "
        "or cell +1 555-123-4567. Target company headquarters: 100 Main St."
    )

    sanitized, count, mapping_hash, detected = scrubber.scrub(sample_text)

    assert count >= 3
    assert "123-45-6789" not in sanitized
    assert "john.doe@example.com" not in sanitized
    assert "555-123-4567" not in sanitized
    assert "[REDACTED_SSN_" in sanitized
    assert "[REDACTED_EMAIL_" in sanitized
    assert "[REDACTED_PHONE_" in sanitized
    assert len(mapping_hash) == 64  # SHA-256 length
    assert "SSN" in detected
    assert "EMAIL" in detected
    assert "PHONE" in detected

def test_pii_scrubber_no_pii():
    scrubber = PIIScrubber()
    clean_text = "Acme Corp generated $45.0M revenue with 82% gross margins in FY2023."
    sanitized, count, mapping_hash, detected = scrubber.scrub(clean_text)

    assert count == 0
    assert sanitized == clean_text
    assert len(detected) == 0
