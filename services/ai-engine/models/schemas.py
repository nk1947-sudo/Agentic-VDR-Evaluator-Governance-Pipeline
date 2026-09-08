from typing import Optional, List, Dict
from pydantic import BaseModel, Field

class PIIScrubRequest(BaseModel):
    text: str

class PIIScrubResponse(BaseModel):
    sanitized_text: str
    redacted_count: int
    token_mapping_hash: str
    detected_entity_types: List[str]

class ExtractorRequest(BaseModel):
    document_text: str
    page_texts: Dict[int, str]
    deal_name: Optional[str] = "Confidential Target Corp"

class CriticAuditRequest(BaseModel):
    page_texts: Dict[int, str]
    extracted_financials: List[Dict]
    extracted_risks: List[Dict]
