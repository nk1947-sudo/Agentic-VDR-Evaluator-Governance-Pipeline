import os
from pydantic_settings import BaseSettings

class Settings(BaseSettings):
    app_name: str = "Agentic VDR Evaluator - AI Engine"
    app_version: str = "1.0.0"
    environment: str = os.getenv("ENVIRONMENT", "development")
    mock_mode: bool = os.getenv("MOCK_MODE", "false").lower() in ("true", "1", "yes")

    # LLM Settings
    openai_api_key: str = os.getenv("OPENAI_API_KEY", "mock-key")
    extractor_model: str = os.getenv("EXTRACTOR_MODEL", "gpt-4o")
    critic_model: str = os.getenv("CRITIC_MODEL", "gpt-4o")
    temperature: float = 0.0

    # Azure Document Intelligence (optional, fallback to local parser if not configured)
    azure_doc_intel_endpoint: str = os.getenv("AZURE_DOC_INTEL_ENDPOINT", "")
    azure_doc_intel_key: str = os.getenv("AZURE_DOC_INTEL_KEY", "")

    # Governance Thresholds
    min_grounding_score: float = 0.85
    max_discrepancies: int = 0
    high_concentration_threshold_pct: float = 25.0

    model_config = {
        "env_file": ".env",
        "extra": "allow"
    }

settings = Settings()
