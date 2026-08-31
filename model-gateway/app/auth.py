from __future__ import annotations

import hmac
import os

from fastapi import Header, HTTPException, status

_TOKEN = os.environ.get("GATEWAY_TOKEN", "")


def require_gateway_token(authorization: str | None = Header(default=None)) -> None:
    if not _TOKEN:
        raise HTTPException(status_code=status.HTTP_503_SERVICE_UNAVAILABLE, detail="gateway_unavailable")
    scheme, _, value = (authorization or "").partition(" ")
    got = value.strip() if scheme.lower() == "bearer" else ""
    if len(got) != len(_TOKEN) or not hmac.compare_digest(got, _TOKEN):
        raise HTTPException(status_code=status.HTTP_401_UNAUTHORIZED, detail="unauthorized")
