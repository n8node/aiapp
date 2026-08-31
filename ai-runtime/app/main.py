from __future__ import annotations

import os
import tempfile
from pathlib import Path

from fastapi import Depends, FastAPI, File, Form, HTTPException, UploadFile, status

from app.auth import require_extract_token
from app.schemas import ExtractResponse
from app.services.extract import extract_path

MAX_UPLOAD_BYTES = 55 * 1024 * 1024
CHUNK = 1024 * 1024

app = FastAPI(title="rigintel-ai-runtime", docs_url=None, redoc_url=None, openapi_url=None)


@app.get("/live")
def live() -> dict[str, str]:
    return {"status": "ok"}


@app.post("/internal/v1/extract", response_model=ExtractResponse)
def extract(
    file: UploadFile = File(...),
    original_name: str | None = Form(default=None),
    mime_type: str | None = Form(default=None),
    _: None = Depends(require_extract_token),
) -> ExtractResponse:
    _ = mime_type
    name = (original_name or file.filename or "upload.bin").replace("\\", "/").split("/")[-1]
    suffix = Path(name).suffix[:16]
    tmp_path = ""
    written = 0
    try:
        with tempfile.NamedTemporaryFile(prefix="extract-", suffix=suffix, delete=False) as tmp:
            tmp_path = tmp.name
            while True:
                chunk = file.file.read(CHUNK)
                if not chunk:
                    break
                written += len(chunk)
                if written > MAX_UPLOAD_BYTES:
                    raise HTTPException(status_code=status.HTTP_413_REQUEST_ENTITY_TOO_LARGE, detail="object_too_large")
                tmp.write(chunk)
        if written == 0:
            raise HTTPException(status_code=status.HTTP_400_BAD_REQUEST, detail="empty_file")
        try:
            return extract_path(tmp_path, name)
        except HTTPException:
            raise
        except Exception:
            raise HTTPException(status_code=status.HTTP_400_BAD_REQUEST, detail="extract_failed")
    finally:
        if tmp_path:
            try:
                os.unlink(tmp_path)
            except OSError:
                pass
