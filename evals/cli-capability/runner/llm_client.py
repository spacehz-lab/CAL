from __future__ import annotations

import json
import os
import time
import urllib.error
import urllib.request
from typing import Any

from util import elapsed_ms


DEFAULT_API = "chat_completions"
DEFAULT_BASE_URL = "https://api.openai.com/v1"
CHAT_COMPLETIONS_PATH = "/chat/completions"
REQUEST_TIMEOUT_SECONDS = 120


class LLMClient:
    def __init__(self, api: str, base_url: str, model: str, api_key: str, timeout: int = REQUEST_TIMEOUT_SECONDS, temperature: float = 0) -> None:
        self.api = api
        self.base_url = base_url.rstrip("/")
        self.model = model
        self.api_key = api_key
        self.timeout = timeout
        self.temperature = temperature

    @classmethod
    def from_env(cls) -> "LLMClient | None":
        model = os.environ.get("CAL_LLM_MODEL", "").strip()
        api_key = os.environ.get("CAL_LLM_API_KEY", "").strip()
        if not model or not api_key:
            return None
        api = os.environ.get("CAL_LLM_API", DEFAULT_API).strip() or DEFAULT_API
        base_url = os.environ.get("CAL_LLM_BASE_URL", DEFAULT_BASE_URL).strip() or DEFAULT_BASE_URL
        temperature = parse_temperature(os.environ.get("CAL_LLM_TEMPERATURE", "0"))
        return cls(api, base_url, model, api_key, temperature=temperature)

    def chat(self, system: str, user: str) -> dict[str, Any]:
        if self.api != DEFAULT_API:
            raise RuntimeError(f"unsupported LLM API: {self.api}")
        payload = {
            "model": self.model,
            "temperature": self.temperature,
            "messages": [
                {"role": "system", "content": system},
                {"role": "user", "content": user},
            ],
        }
        started = time.monotonic()
        request = urllib.request.Request(
            self.base_url + CHAT_COMPLETIONS_PATH,
            data=json.dumps(payload).encode("utf-8"),
            headers={
                "Authorization": f"Bearer {self.api_key}",
                "Content-Type": "application/json",
            },
            method="POST",
        )
        try:
            with urllib.request.urlopen(request, timeout=self.timeout) as response:
                body = response.read().decode("utf-8")
        except urllib.error.HTTPError as exc:
            detail = exc.read().decode("utf-8", errors="replace")
            raise RuntimeError(f"LLM request failed: HTTP {exc.code}: {detail[:500]}") from exc
        except urllib.error.URLError as exc:
            raise RuntimeError(f"LLM request failed: {exc.reason}") from exc

        parsed = json.loads(body)
        choices = parsed.get("choices") or []
        content = ((choices[0].get("message") or {}).get("content") if choices else "") or ""
        return {
            "api": self.api,
            "model": self.model,
            "duration_ms": elapsed_ms(started),
            "content": content,
            "usage": parsed.get("usage") or {},
        }


def parse_temperature(value: str) -> float:
    try:
        return float(value)
    except ValueError as exc:
        raise RuntimeError(f"invalid CAL_LLM_TEMPERATURE: {value}") from exc
