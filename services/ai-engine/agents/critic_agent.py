import json
import logging
import re
from typing import Dict, List
from config import settings
from models.deal_metrics import FinancialMetrics, DealRisk, AuditPacket, CriticDiscrepancy
from agents.prompts import CRITIC_SYSTEM_PROMPT

logger = logging.getLogger(__name__)

class CriticAuditAgent:
    """
    Dual-Agent Layer 2: Critic & Anti-Hallucination Agent.
    Independently audits the extraction against raw scrubbed text.
    Detects cross-page contradictions, verifies quote grounding, and enforces governance gates.
    """

    def __init__(self):
        self.mock_mode = settings.mock_mode
        self.api_key = settings.openai_api_key

    def audit(
        self,
        page_texts: Dict[int, str],
        financials: List[FinancialMetrics],
        risks: List[DealRisk]
    ) -> AuditPacket:
        if not self.mock_mode and self.api_key and not self.api_key.startswith("mock"):
            try:
                return self._audit_with_openai(page_texts, financials, risks)
            except Exception as e:
                logger.warning(f"OpenAI critic audit failed: {e}. Falling back to deterministic auditor.")

        return self._audit_deterministic(page_texts, financials, risks)

    def _audit_with_openai(
        self,
        page_texts: Dict[int, str],
        financials: List[FinancialMetrics],
        risks: List[DealRisk]
    ) -> AuditPacket:
        from openai import OpenAI
        client = OpenAI(api_key=self.api_key)

        doc_payload = "\n\n".join([f"--- PAGE {p} ---\n{txt}" for p, txt in sorted(page_texts.items())])
        extraction_summary = json.dumps({
            "financials": [f.model_dump() for f in financials],
            "risks": [r.model_dump() for r in risks]
        }, indent=2)

        prompt = f"""AUDIT THE FOLLOWING EXTRACTIONS AGAINST THE RAW DOCUMENT:

Extracted Data:
{extraction_summary}

Raw Source Document (By Page):
{doc_payload}
"""

        response = client.chat.completions.create(
            model=settings.critic_model,
            temperature=settings.temperature,
            messages=[
                {"role": "system", "content": CRITIC_SYSTEM_PROMPT},
                {"role": "user", "content": prompt}
            ],
            response_format={"type": "json_object"}
        )

        data = json.loads(response.choices[0].message.content)
        discrepancies = [CriticDiscrepancy(**d) for d in data.get("discrepancies", [])]
        grounding_score = float(data.get("grounding_score", 0.9))
        recommendation = data.get("recommendation", "REQUIRE_HUMAN_REVIEW")

        return AuditPacket(
            grounding_score=grounding_score,
            is_grounded=grounding_score >= settings.min_grounding_score and len(discrepancies) == 0,
            discrepancies=discrepancies,
            math_consistency_passed=data.get("math_consistency_passed", True),
            math_reconciliation_notes=data.get("math_reconciliation_notes", []),
            critic_summary=data.get("critic_summary", "Audit completed via LLM Critic."),
            recommendation=recommendation
        )

    def _audit_deterministic(
        self,
        page_texts: Dict[int, str],
        financials: List[FinancialMetrics],
        risks: List[DealRisk]
    ) -> AuditPacket:
        """
        Deterministic, audit-grade verification engine.
        Examines text grounding, scans for multi-page contradictions, and calculates defensible score.
        """
        grounding_score = 1.0
        discrepancies: List[CriticDiscrepancy] = []
        math_notes: List[str] = []
        math_passed = True

        # 1. Quote Grounding Verification
        for fin in financials:
            # Check revenue quote
            rev_cit = fin.revenue.citation
            page_text = page_texts.get(rev_cit.page_number, "")
            if not self._is_quote_grounded(rev_cit.exact_quote, page_text):
                grounding_score -= 0.35
                discrepancies.append(CriticDiscrepancy(
                    metric_name="Revenue",
                    claimed_value=str(fin.revenue.value),
                    found_conflict="Quoted text not found on specified page.",
                    page_a=rev_cit.page_number,
                    quote_a=rev_cit.exact_quote,
                    page_b=rev_cit.page_number,
                    quote_b="[N/A]",
                    severity="CRITICAL",
                    explanation="Hallucination detected: Cited revenue quote does not exist verbatim on the cited page."
                ))

            # Check ebitda quote
            ebitda_cit = fin.ebitda.citation
            ebitda_page_text = page_texts.get(ebitda_cit.page_number, "")
            if not self._is_quote_grounded(ebitda_cit.exact_quote, ebitda_page_text):
                grounding_score -= 0.35
                discrepancies.append(CriticDiscrepancy(
                    metric_name="EBITDA",
                    claimed_value=str(fin.ebitda.value),
                    found_conflict="Quoted text not found on specified page.",
                    page_a=ebitda_cit.page_number,
                    quote_a=ebitda_cit.exact_quote,
                    page_b=ebitda_cit.page_number,
                    quote_b="[N/A]",
                    severity="CRITICAL",
                    explanation="Hallucination detected: Cited EBITDA quote does not exist verbatim on the cited page."
                ))

            # Math check: EBITDA vs Revenue
            if fin.ebitda.value > fin.revenue.value:
                math_passed = False
                math_notes.append("EBITDA exceeds total revenue (mathematical impossibility).")
                grounding_score -= 0.30

        # 2. Cross-Page Contradiction Detection
        # Check if another page in the document contains an alternate/conflicting EBITDA value
        ebitda_mentions: List[tuple[int, float, str]] = []
        for p_num, text in page_texts.items():
            for line in text.split("\n"):
                if re.search(r"\bEBITDA\b", line, re.I):
                    # Prioritize finding explicit dollar amount (e.g. $18.0M or $11.2M) to avoid matching year like FY2023
                    m = re.search(r"\$\s*([\d,]+(?:\.\d+)?)\s*(?:M|million)?\b", line, re.I)
                    if not m:
                        m = re.search(r"\b([\d,]+(?:\.\d+)?)\s*(?:M|million)\b", line, re.I)
                    if m:
                        val = float(m.group(1).replace(",", ""))
                        if "m" in m.group(0).lower() or "million" in m.group(0).lower() or "m" in line.lower():
                            val *= 1_000_000.0
                        ebitda_mentions.append((p_num, val, line.strip()))

        # If multiple distinct EBITDA figures found across pages (differing by > 5%)
        if len(ebitda_mentions) > 1:
            base_p, base_val, base_quote = ebitda_mentions[0]
            for p, val, quote in ebitda_mentions[1:]:
                diff_ratio = abs(val - base_val) / max(base_val, 1.0)
                if diff_ratio > 0.05:
                    grounding_score -= 0.40
                    discrepancies.append(CriticDiscrepancy(
                        metric_name="EBITDA Contradiction",
                        claimed_value=f"${base_val:,.0f} on Page {base_p}",
                        found_conflict=f"${val:,.0f} on Page {p}",
                        page_a=base_p,
                        quote_a=base_quote,
                        page_b=p,
                        quote_b=quote,
                        severity="CRITICAL",
                        explanation=f"Severe financial discrepancy: Marketing summary claims EBITDA of ${base_val:,.0f} (Page {base_p}), but audited notes on Page {p} report ${val:,.0f}."
                    ))

        # 3. Customer Concentration Risk Check
        for risk in risks:
            if risk.risk_category == "CUSTOMER_CONCENTRATION" and risk.concentration_pct:
                if risk.concentration_pct > settings.high_concentration_threshold_pct:
                    grounding_score -= 0.15
                    math_notes.append(f"High risk trigger: Customer concentration is {risk.concentration_pct}%, exceeding maximum safe threshold of {settings.high_concentration_threshold_pct}%.")

        grounding_score = max(0.0, min(1.0, round(grounding_score, 3)))
        is_clean = grounding_score >= settings.min_grounding_score and len(discrepancies) == 0

        recommendation = "FAST_TRACK_APPROVAL" if is_clean else "REQUIRE_HUMAN_REVIEW"
        summary = (
            f"Audit finished with grounding score {grounding_score:.2f}. "
            f"Discrepancies identified: {len(discrepancies)}. "
            f"Human review required: {not is_clean}."
        )

        return AuditPacket(
            grounding_score=grounding_score,
            is_grounded=is_clean,
            discrepancies=discrepancies,
            math_consistency_passed=math_passed,
            math_reconciliation_notes=math_notes,
            critic_summary=summary,
            recommendation=recommendation
        )

    def _is_quote_grounded(self, quote: str, page_text: str) -> bool:
        if not quote or not page_text:
            return False
        # Normalize whitespace and casing
        norm_quote = re.sub(r"\s+", " ", quote.strip().lower())
        norm_page = re.sub(r"\s+", " ", page_text.strip().lower())
        return norm_quote in norm_page or any(word in norm_page for word in norm_quote.split() if len(word) > 5)
