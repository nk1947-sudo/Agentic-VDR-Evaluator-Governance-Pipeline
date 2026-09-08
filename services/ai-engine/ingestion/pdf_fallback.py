import io
import logging
from typing import Dict

logger = logging.getLogger(__name__)

class LocalPDFLayoutEngine:
    """
    Robust local layout & text extraction engine for VDR documents.
    Extracts text preserving page numbers and table structure.
    """

    @staticmethod
    def extract_page_texts(file_bytes: bytes, filename: str = "") -> Dict[int, str]:
        pages_dict: Dict[int, str] = {}

        # 1. Check if input is plain text or markdown
        if filename.endswith(".txt") or filename.endswith(".md"):
            content = file_bytes.decode("utf-8", errors="replace")
            # If document has explicit [PAGE X] or Page X markers, split by page
            raw_pages = content.split("--- PAGE ")
            if len(raw_pages) > 1:
                for idx, p in enumerate(raw_pages[1:], start=1):
                    # extract actual page number if present e.g. "1 --- \n..."
                    parts = p.split("---", 1)
                    if len(parts) == 2:
                        try:
                            p_num = int(parts[0].strip())
                            pages_dict[p_num] = parts[1].strip()
                        except ValueError:
                            pages_dict[idx] = p.strip()
                    else:
                        pages_dict[idx] = p.strip()
            else:
                pages_dict[1] = content
            return pages_dict

        # 2. Try extraction using pdfplumber (table & layout sensitive)
        try:
            import pdfplumber
            with pdfplumber.open(io.BytesIO(file_bytes)) as pdf:
                for idx, page in enumerate(pdf.pages, start=1):
                    text = page.extract_text() or ""
                    # Also extract tables if present and format as markdown tables
                    tables = page.extract_tables()
                    if tables:
                        table_strs = []
                        for table in tables:
                            formatted_rows = [" | ".join(str(cell or "").strip() for cell in row) for row in table if any(row)]
                            if formatted_rows:
                                table_strs.append("\n".join(formatted_rows))
                        if table_strs:
                            text += "\n\n[Extracted Tables]:\n" + "\n\n".join(table_strs)
                    pages_dict[idx] = text.strip()
            if pages_dict and any(pages_dict.values()):
                return pages_dict
        except Exception as e:
            logger.debug(f"pdfplumber extraction failed or not applicable: {e}")

        # 3. Fallback to pypdf
        try:
            import pypdf
            reader = pypdf.PdfReader(io.BytesIO(file_bytes))
            for idx, page in enumerate(reader.pages, start=1):
                pages_dict[idx] = (page.extract_text() or "").strip()
            if pages_dict and any(pages_dict.values()):
                return pages_dict
        except Exception as e:
            logger.warning(f"pypdf extraction failed: {e}")

        # 4. Fallback decode
        decoded = file_bytes.decode("utf-8", errors="replace")
        pages_dict[1] = decoded
        return pages_dict
