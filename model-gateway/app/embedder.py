from __future__ import annotations

from typing import Any

_embedder: Any = None


def get_embedder() -> Any:
    global _embedder
    if _embedder is None:
        from fastembed import TextEmbedding

        _embedder = TextEmbedding(model_name="intfloat/multilingual-e5-small")
    return _embedder


def prefix_text(text: str, input_type: str) -> str:
    t = text.strip()
    if input_type == "query":
        return "query: " + t
    return "passage: " + t
