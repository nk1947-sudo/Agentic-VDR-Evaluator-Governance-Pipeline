import re
import hashlib
from typing import Dict, List, Tuple

class PIIScrubber:
    """
    Enterprise PII Redaction Sandbox.
    Scrubs sensitive personal information before text is dispatched to external LLMs.
    Guarantees isolation of untrusted data and maintains cryptographic audit hashes.
    """

    PATTERNS = {
        "SSN": re.compile(r"\b\d{3}-\d{2}-\d{4}\b"),
        "EMAIL": re.compile(r"\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}\b"),
        "PHONE": re.compile(r"\b(?:\+?1[-.\s]?)?(?:\(?\d{3}\)?[-.\s]?)\d{3}[-.\s]?\d{4}\b"),
        "CREDIT_CARD": re.compile(r"\b(?:\d{4}[-\s]?){3}\d{4}\b"),
        "IP_ADDRESS": re.compile(r"\b(?:[0-9]{1,3}\.){3}[0-9]{1,3}\b"),
    }

    def __init__(self):
        self._redaction_map: Dict[str, str] = {}

    def scrub(self, text: str) -> Tuple[str, int, str, List[str]]:
        """
        Sanitizes text by replacing PII with secure tokens.
        Returns:
            (sanitized_text, total_redacted_count, token_mapping_sha256, detected_entity_types)
        """
        sanitized = text
        total_redacted = 0
        detected_types: List[str] = []
        entity_counters: Dict[str, int] = {}

        for entity_type, pattern in self.PATTERNS.items():
            matches = list(pattern.finditer(sanitized))
            if matches:
                detected_types.append(entity_type)

            # Replace from right to left to avoid altering start indices
            for match in reversed(matches):
                original_value = match.group(0)
                counter = entity_counters.get(entity_type, 0) + 1
                entity_counters[entity_type] = counter
                
                token = f"[REDACTED_{entity_type}_{counter}]"
                self._redaction_map[token] = original_value
                
                start, end = match.span()
                sanitized = sanitized[:start] + token + sanitized[end:]
                total_redacted += 1

        # Calculate cryptographic hash of redaction dictionary for compliance tracking
        map_repr = "".join(f"{k}:{v}" for k, v in sorted(self._redaction_map.items()))
        mapping_hash = hashlib.sha256(map_repr.encode("utf-8")).hexdigest()

        return sanitized, total_redacted, mapping_hash, detected_types

    def get_token_count(self) -> int:
        return len(self._redaction_map)

    def clear(self):
        self._redaction_map.clear()
