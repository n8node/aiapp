from __future__ import annotations

import os
import threading
from typing import Any

import httpx
from fastapi import Depends, FastAPI, HTTPException, status
from pydantic import BaseModel, Field

from app.auth import require_gateway_token
from app.embedder import get_embedder, prefix_text

from fastapi.responses import StreamingResponse

app = FastAPI(title="rigintel-model-gateway", docs_url=None, redoc_url=None, openapi_url=None)
VLLM_URL = (os.environ.get("VLLM_URL") or "").rstrip("/")
CHAT_URL = (os.environ.get("CHAT_URL") or "").rstrip("/")


def chat_upstream() -> str:
    return VLLM_URL or CHAT_URL


@app.on_event("startup")
def warmup_embedder() -> None:
    threading.Thread(target=_warmup, daemon=True).start()


def _warmup() -> None:
    try:
        get_embedder()
    except Exception:
        pass


class EmbedRequest(BaseModel):
    input: str | list[str]
    model: str = "embeddings"
    input_type: str = Field(default="document")


@app.get("/live")
def live() -> dict[str, str]:
    return {"status": "ok"}


@app.get("/v1/models")
def models(_: None = Depends(require_gateway_token)) -> dict[str, Any]:
    return {
        "object": "list",
        "data": [
            {"id": "embeddings", "object": "model", "owned_by": "rigintel"},
        ],
    }


@app.post("/v1/embeddings")
def embeddings(body: EmbedRequest, _: None = Depends(require_gateway_token)) -> dict[str, Any]:
    texts = body.input if isinstance(body.input, list) else [body.input]
    if not texts or any(not isinstance(t, str) or not t.strip() for t in texts):
        raise HTTPException(status_code=status.HTTP_400_BAD_REQUEST, detail="invalid_input")
    if len(texts) > 32:
        raise HTTPException(status_code=status.HTTP_400_BAD_REQUEST, detail="batch_too_large")
    prefixed = [prefix_text(t, body.input_type) for t in texts]
    try:
        vectors = list(get_embedder().embed(prefixed))
    except Exception:
        raise HTTPException(status_code=status.HTTP_503_SERVICE_UNAVAILABLE, detail="gateway_unavailable") from None
    data = []
    for i, vec in enumerate(vectors):
        arr = [float(x) for x in vec]
        data.append({"object": "embedding", "index": i, "embedding": arr})
    return {"object": "list", "model": "intfloat/multilingual-e5-small", "data": data}


class ChatRequest(BaseModel):
    model: str = "chat"
    messages: list[dict[str, str]]
    stream: bool = False


class MediaRequest(BaseModel):
    model: str = ""
    prompt: str
    n: int = 1


def _proxy_json(path: str, payload: dict[str, Any]) -> Any:
    upstream = chat_upstream()
    if not upstream:
        raise HTTPException(status_code=status.HTTP_501_NOT_IMPLEMENTED, detail="chat_not_configured")
    try:
        with httpx.Client(timeout=180.0) as client:
            resp = client.post(upstream + path, json=payload, headers={"Content-Type": "application/json"})
    except Exception:
        raise HTTPException(status_code=status.HTTP_502_BAD_GATEWAY, detail="chat_unavailable") from None
    if resp.status_code in (404, 405, 501):
        raise HTTPException(status_code=status.HTTP_501_NOT_IMPLEMENTED, detail="media_unavailable")
    if resp.status_code >= 400:
        raise HTTPException(status_code=status.HTTP_502_BAD_GATEWAY, detail="chat_unavailable")
    try:
        return resp.json()
    except Exception:
        raise HTTPException(status_code=status.HTTP_502_BAD_GATEWAY, detail="chat_unavailable") from None


@app.post("/v1/chat/completions")
def chat(body: ChatRequest, _: None = Depends(require_gateway_token)) -> Any:
    upstream = chat_upstream()
    if not upstream:
        raise HTTPException(status_code=status.HTTP_501_NOT_IMPLEMENTED, detail="chat_not_configured")
    payload = body.model_dump()
    if body.stream:

        def gen():
            try:
                with httpx.Client(timeout=httpx.Timeout(180.0)) as client:
                    with client.stream(
                        "POST",
                        upstream + "/v1/chat/completions",
                        json=payload,
                        headers={"Content-Type": "application/json"},
                    ) as resp:
                        if resp.status_code >= 400:
                            yield b'data: {"error":"chat_unavailable"}\n\n'
                            return
                        for chunk in resp.iter_bytes():
                            if chunk:
                                yield chunk
            except Exception:
                yield b'data: {"error":"chat_unavailable"}\n\n'

        return StreamingResponse(gen(), media_type="text/event-stream")
    try:
        with httpx.Client(timeout=180.0) as client:
            resp = client.post(
                upstream + "/v1/chat/completions",
                json=payload,
                headers={"Content-Type": "application/json"},
            )
    except Exception:
        raise HTTPException(status_code=status.HTTP_502_BAD_GATEWAY, detail="chat_unavailable") from None
    if resp.status_code >= 400:
        raise HTTPException(status_code=status.HTTP_502_BAD_GATEWAY, detail="chat_unavailable")
    return resp.json()


@app.post("/v1/images/generations")
def images(body: MediaRequest, _: None = Depends(require_gateway_token)) -> Any:
    return _proxy_json("/v1/images/generations", body.model_dump())


@app.post("/v1/videos/generations")
def videos(body: MediaRequest, _: None = Depends(require_gateway_token)) -> Any:
    return _proxy_json("/v1/videos/generations", body.model_dump())
