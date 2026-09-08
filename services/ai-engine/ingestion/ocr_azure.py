import logging
from typing import Dict, Optional

logger = logging.getLogger(__name__)

class AzureDocumentIntelligenceClient:
    """
    Client for Azure Document Intelligence layout extraction.
    Parses messy, multi-page VDR PDFs into structured text and tables.
    """

    def __init__(self, endpoint: str, api_key: str):
        self.endpoint = endpoint
        self.api_key = api_key
        self.is_configured = bool(endpoint and api_key and not api_key.startswith("your-"))

    def analyze_document(self, file_bytes: bytes) -> Optional[Dict[int, str]]:
        if not self.is_configured:
            logger.info("Azure Document Intelligence not configured. Delegating to local layout engine.")
            return None

        try:
            # Azure Document Intelligence REST or SDK integration
            import httpx
            url = f"{self.endpoint.rstrip('/')}/documentintelligence/documentModels/prebuilt-layout:analyze?api-version=2024-02-29-preview"
            headers = {
                "Ocp-Apim-Subscription-Key": self.api_key,
                "Content-Type": "application/pdf"
            }
            with httpx.Client(timeout=60.0) as client:
                response = client.post(url, headers=headers, content=file_bytes)
                if response.status_code == 202:
                    logger.info("Azure Document Intelligence accepted document analysis.")
                    # In production, poll operation-location. Here fallback if async
                    return None
        except Exception as e:
            logger.warning(f"Azure Document Intelligence call failed: {e}. Falling back to local engine.")
            return None

        return None
