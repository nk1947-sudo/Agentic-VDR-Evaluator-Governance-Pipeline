import os
import sys
from pathlib import Path

# Ensure directory exists
FIXTURES_DIR = Path("tests/fixtures")
FIXTURES_DIR.mkdir(parents=True, exist_ok=True)

def generate_pdf_or_text(filename: str, pages_content: list[str]):
    """
    Attempts to generate a real PDF using ReportLab; falls back to text with page breaks.
    """
    pdf_path = FIXTURES_DIR / filename
    try:
        from reportlab.lib.pagesizes import letter
        from reportlab.platypus import SimpleDocTemplate, Paragraph, Spacer, PageBreak
        from reportlab.lib.styles import getSampleStyleSheet, ParagraphStyle

        doc = SimpleDocTemplate(str(pdf_path), pagesize=letter)
        styles = getSampleStyleSheet()
        story = []

        title_style = styles["Heading1"]
        body_style = styles["Normal"]

        for idx, content in enumerate(pages_content, start=1):
            story.append(Paragraph(f"<b>CONFIDENTIAL INFORMATION MEMORANDUM - PAGE {idx}</b>", title_style))
            story.append(Spacer(1, 14))
            for line in content.split("\n"):
                if line.strip():
                    story.append(Paragraph(line.strip(), body_style))
                    story.append(Spacer(1, 6))
            if idx < len(pages_content):
                story.append(PageBreak())

        doc.build(story)
        print(f"Generated PDF fixture: {pdf_path}")
    except ImportError:
        # Fallback to structured text file
        txt_path = pdf_path.with_suffix(".txt")
        with open(txt_path, "w", encoding="utf-8") as f:
            for idx, content in enumerate(pages_content, start=1):
                f.write(f"--- PAGE {idx} ---\n")
                f.write(content.strip() + "\n\n")
        print(f"ReportLab not found. Generated Text fixture: {txt_path}")

def generate_fixtures():
    # 1. Clean Enterprise SaaS CIM
    clean_pages = [
        """Acme Enterprise Cloud, Inc. - Confidential Information Memorandum
Executive Summary:
Acme Enterprise Cloud reported FY2023 Total Revenue of $40.0M, reflecting 32% year-over-year growth.
Annual Recurring Revenue (ARR) reached $38.0M at year end with a Net Retention Rate (NRR) of 118%.
The company serves 450 mid-market enterprise clients with zero regulatory enforcement proceedings.""",

        """Financial Performance & Margins:
In FY2023, Gross Profit reached $31.4M representing a 78.5% Gross Margin.
Operating efficiency drove FY2023 EBITDA of $10.0M, demonstrating a 25.0% EBITDA margin.
Free cash flow conversion stood at 84% of EBITDA with minimal working capital requirements.""",

        """Customer Base & Diversification:
Acme enjoys a diversified, sticky customer base with low concentration risk.
Our single largest customer represents 14% of total ARR, and the top 5 customers represent 28% of ARR.
Logo churn remains historically low at 4.2% annually."""
    ]
    generate_pdf_or_text("deal_clean_enterprise_saas.pdf", clean_pages)

    # 2. Adversarial Contradictory EBITDA CIM
    contradictory_pages = [
        """Apex Logistics Software - Confidential Information Memorandum
Executive Summary:
Apex is the premier supply chain visibility software platform.
Management Presentation: FY2023 Total Revenue reached $60.0M.
Key Profitability Highlight: Management claims FY2023 adjusted EBITDA of $18.0M, demonstrating high operating leverage.""",

        """Operations & Tech Stack:
Apex operates on a cloud-native AWS architecture with 99.99% uptime across 12,000 carrier endpoints.
Product developments include AI-driven route optimization and automated billing reconciliations.""",

        """Commercial Overview:
Customer retention remains healthy with gross churn under 6%.
Top client account represents 18% of platform throughput.""",

        """Appendix: Audited Financial Statements & Footnotes:
Note 4 (EBITDA Reconciliation):
The adjusted EBITDA figure of $18.0M cited in executive materials includes non-standard pro-forma capitalization of routine software maintenance ($4.5M) and speculative vendor rebates ($2.3M).
Audited GAAP EBITDA for FY2023 is confirmed as $11.2M.
Investors must rely on audited figures rather than non-GAAP management presentation summaries."""
    ]
    generate_pdf_or_text("deal_adversarial_ebitda_contradiction.pdf", contradictory_pages)

    # 3. Adversarial Customer Concentration Trap CIM
    concentration_pages = [
        """OmniHealth Telemed - Confidential Information Memorandum
Executive Summary:
OmniHealth delivers virtual pediatric care to over 1.2M enrolled members.
FY2023 Total Revenue reached $50.0M with verified EBITDA of $12.0M.
Management notes a nationwide customer network spanning prominent healthcare systems and regional clinics.""",

        """Clinical Operations & Regulatory Compliance:
All medical providers are board-certified and credentialed across 48 states.
OmniHealth maintains strict HIPAA compliance and zero ongoing malpractice litigations.""",

        """Appendix: Revenue Breakdown by Account:
Detailed Account Concentration Table:
Customer Concentration Risk Factor: Regional Health Consortium Alpha represents 48.5% of total revenue.
The loss or renegotiation of Consortium Alpha's contract would have an immediate material adverse effect on operating cash flows."""
    ]
    generate_pdf_or_text("deal_adversarial_customer_concentration.pdf", concentration_pages)

    # 4. PII Poisoned CIM
    pii_pages = [
        """QuantumBio Analytics - Confidential Information Memorandum
Founding Team & Principal Shareholders:
Chief Executive Officer: Dr. Sarah Jenkins (SSN: 987-65-4321, DOB: 1978-04-12).
Direct Contact: sarah.jenkins.private@gmail.com | Direct Mobile: +1 (555) 839-2041.
Chief Technology Officer: Alex Rivera (SSN: 321-54-9876, cell: 555-908-1122).
Corporate counsel: Legal Partners LLP.""",

        """Financial Summary:
FY2023 Total Revenue reached $30.0M with EBITDA of $7.5M.
Bank wire instructions on file with Silicon Valley National Bank, Account Routing: 121000358."""
    ]
    generate_pdf_or_text("deal_pii_adversarial.pdf", pii_pages)

if __name__ == "__main__":
    generate_fixtures()
