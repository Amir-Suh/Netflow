"""AES-256-GCM helper compatible with the Go ``internal/crypto`` package.

The Go implementation stores ciphertext and nonce as standard base64
strings (without prepending the nonce to the ciphertext) and uses
``crypto/cipher`` AEAD with no associated data. The cryptography library's
``AESGCM`` has the matching wire format.
"""

from __future__ import annotations

import base64
import os

from cryptography.hazmat.primitives.ciphers.aead import AESGCM


_ALG = "AES-256-GCM"


def _key() -> bytes:
    raw = os.environ.get("AES_256_GCM_KEY_BASE64", "")
    if not raw:
        raise RuntimeError("AES_256_GCM_KEY_BASE64 is required")
    key = base64.b64decode(raw)
    if len(key) != 32:
        raise RuntimeError(
            f"AES_256_GCM_KEY_BASE64 must decode to 32 bytes, got {len(key)}"
        )
    return key


def decrypt(ciphertext_b64: str, nonce_b64: str, algorithm: str = _ALG) -> str:
    if algorithm and algorithm != _ALG:
        raise RuntimeError(f"unsupported algorithm: {algorithm!r}")
    aesgcm = AESGCM(_key())
    ciphertext = base64.b64decode(ciphertext_b64)
    nonce = base64.b64decode(nonce_b64)
    return aesgcm.decrypt(nonce, ciphertext, None).decode("utf-8")
