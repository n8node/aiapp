from __future__ import annotations

from pydantic import BaseModel, Field


class ExtractResponse(BaseModel):
    engine: str
    text: str = ""
    page_count: int = 0
    confidence: float = 0.0
    warnings: list[str] = Field(default_factory=list)
