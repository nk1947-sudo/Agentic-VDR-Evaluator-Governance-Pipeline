import logging
from fastapi import FastAPI, File, UploadFile, Form, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from config import settings
from models.deal_metrics import ProcessDocumentResponse
from models.schemas import PIIScrubRequest, PIIScrubResponse
from ingestion.pii_scrubber import PIIScrubber
from ingestion.pdf_fallback import LocalPDFLayoutEngine
from agents.extractor_agent import FinancialExtractorAgent
from agents.critic_agent import CriticAuditAgent

logging.basicConfig(level=logging.INFO, format="%(asctime)s [%(levelname)s] %(name)s: %(message)s")
logger = logging.getLogger("ai-engine")

app = FastAPI(
    title=settings.app_name,
    version=settings.app_version,
    description="Multi-Agent VDR Extraction, PII Redaction, and Anti-Hallucination Critic Service"
)

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

pii_scrubber = PIIScrubber()
extractor_agent = FinancialExtractorAgent()
critic_agent = CriticAuditAgent()

@app.get("/api/v1/health")
async def health():
    return {
        "status": "healthy",
        "service": settings.app_name,
        "mock_mode": settings.mock_mode,
        "extractor_model": settings.extractor_model,
        "critic_model": settings.critic_model
    }

@app.post("/api/v1/pii-scrub", response_model=PIIScrubResponse)
async def scrub_pii(request: PIIScrubRequest):
    sanitized_text, count, token_hash, detected = pii_scrubber.scrub(request.text)
    return PIIScrubResponse(
        sanitized_text=sanitized_text,
        redacted_count=count,
        token_mapping_hash=token_hash,
        detected_entity_types=detected
    )

@app.post("/api/v1/process-document", response_model=ProcessDocumentResponse)
async def process_document(
    file: UploadFile = File(...),
    deal_name: str = Form("Target Corp")
):
    try:
        content = await file.read()
        logger.info(f"Ingesting document: {file.filename} ({len(content)} bytes) for deal: {deal_name}")

        # 1. OCR & Layout parsing (page-indexed dictionary)
        page_texts = LocalPDFLayoutEngine.extract_page_texts(content, filename=file.filename)
        if not page_texts:
            raise HTTPException(status_code=400, detail="Unable to extract text from provided document.")

        # 2. PII Scrubbing Sandbox on each page
        scrubbed_pages = {}
        total_redacted = 0
        for p_num, text in page_texts.items():
            sanitized, count, _, _ = pii_scrubber.scrub(text)
            scrubbed_pages[p_num] = sanitized
            total_redacted += count

        logger.info(f"PII scrub complete. Sanitized {total_redacted} sensitive entities across {len(scrubbed_pages)} pages.")

        # 3. Layer 1: Extraction Agent
        financials, risks = extractor_agent.extract(scrubbed_pages, deal_name=deal_name)
        logger.info(f"Extraction complete: {len(financials)} financial statements, {len(risks)} deal risk factors.")

        # 4. Layer 2: Critic & Anti-Hallucination Audit Agent
        audit_packet = critic_agent.audit(scrubbed_pages, financials, risks)
        logger.info(f"Critic audit complete. Grounding score: {audit_packet.grounding_score:.2f}, Recommendation: {audit_packet.recommendation}")

        return ProcessDocumentResponse(
            deal_name=deal_name,
            pii_sanitized=True,
            redacted_entities_count=total_redacted,
            financials=financials,
            risks=risks,
            audit=audit_packet,
            raw_text_pages_count=len(scrubbed_pages)
        )
    except Exception as e:
        logger.error(f"Error processing document: {e}", exc_info=True)
        raise HTTPException(status_code=500, detail=str(e))

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000)
