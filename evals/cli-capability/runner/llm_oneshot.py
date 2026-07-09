from __future__ import annotations

import json
import platform
import shutil
import subprocess
import time
from pathlib import Path
from typing import Any

from constants import MODE_LIVE_LLM, STATUS_FAILED, STATUS_PASSED, STATUS_SKIPPED, STEP_BASELINE_LLM_ONESHOT, STEP_BASELINE_ORACLE
from llm_client import LLMClient
from oracle import OracleRunner
from reuse import materialize_inputs, reuse_rounds, runtime_inputs, summarize_inputs
from util import elapsed_ms, failure, flow_step, parse_json


HELP_TIMEOUT_SECONDS = 5
HELP_TEXT_LIMIT = 12_000
COMMAND_TIMEOUT_SECONDS = 60

CODE_LIVE_MODE_REQUIRED = "live_mode_required"
CODE_MISSING_LLM_CONFIG = "missing_llm_config"
CODE_PLATFORM_UNAVAILABLE = "platform_unavailable"
CODE_CLI_UNAVAILABLE = "cli_unavailable"
CODE_LLM_FAILED = "llm_failed"
CODE_INVALID_JSON = "invalid_json"
CODE_INVALID_COMMAND = "invalid_command"
CODE_MISSING_TARGET = "missing_target"
CODE_COMMAND_FAILED = "command_failed"


class OneShotRunner:
    def __init__(self, repo: Path, bench: Path, home: Path, mode: str, oracle: OracleRunner, client: LLMClient | None = None) -> None:
        self.repo = repo
        self.bench = bench
        self.home = home
        self.mode = mode
        self.oracle = oracle
        self.client = client if client is not None else LLMClient.from_env()
        self.prompt = (bench / "baselines" / "llm_one_shot" / "prompt.md").read_text(encoding="utf-8")

    def run_case(self, case: dict[str, Any]) -> list[dict[str, Any]]:
        rows = []
        for provider in case.get("providers") or []:
            for fixture in reuse_rounds(case):
                rows.append(self.run_fixture(case, provider, fixture))
        return rows

    def run_fixture(self, case: dict[str, Any], provider: dict[str, Any], fixture: dict[str, Any]) -> dict[str, Any]:
        row = {
            "id": f"{case.get('id', '')}:{provider.get('id', '')}:{fixture.get('id', '')}",
            "provider": provider.get("id", ""),
            "fixture_id": fixture.get("id", ""),
            "steps": [],
        }
        inputs = materialize_inputs(self.bench, self.home, case["id"], fixture)
        preflight = self.preflight(provider)
        if preflight:
            return self.skipped(row, preflight)
        user_prompt = self.build_user_prompt(case, provider, fixture, inputs)
        started = time.monotonic()
        try:
            response = self.client.chat(self.prompt, user_prompt)
        except Exception as exc:
            err = failure("baseline_llm_oneshot", CODE_LLM_FAILED, str(exc))
            return self.failed(row, err, elapsed_ms(started))
        llm_duration = response.get("duration_ms", elapsed_ms(started))
        row["llm"] = {
            "api": response.get("api", ""),
            "model": response.get("model", ""),
            "duration_ms": llm_duration,
            "usage": response.get("usage") or {},
        }
        command_spec = parse_llm_command(response.get("content", ""))
        if not command_spec:
            return self.failed(row, failure("baseline_llm_oneshot", CODE_INVALID_JSON, "LLM did not return a JSON command object"), llm_duration)
        command = command_spec.get("command")
        command_error = validate_command(command, provider.get("command", ""))
        if command_error:
            return self.failed(row, command_error, llm_duration)
        writes_target = bool(command_spec.get("writes_target"))
        row["command"] = command
        row["writes_target"] = writes_target
        return self.run_command(row, case, inputs, command, writes_target, llm_duration)

    def preflight(self, provider: dict[str, Any]) -> dict[str, str] | None:
        if self.mode != MODE_LIVE_LLM:
            return failure("baseline_llm_oneshot", CODE_LIVE_MODE_REQUIRED, "LLM one-shot baseline only runs in live_llm mode")
        if self.client is None:
            return failure("baseline_llm_oneshot", CODE_MISSING_LLM_CONFIG, "CAL_LLM_MODEL and CAL_LLM_API_KEY are required")
        platforms = provider.get("platforms") or []
        if platforms and current_platform() not in platforms:
            return failure("baseline_llm_oneshot", CODE_PLATFORM_UNAVAILABLE, f"provider is not configured for {current_platform()}")
        command = provider.get("command", "")
        if not command or not shutil.which(command):
            return failure("baseline_llm_oneshot", CODE_CLI_UNAVAILABLE, f"{command or 'command'} was not found on PATH")
        return None

    def build_user_prompt(self, case: dict[str, Any], provider: dict[str, Any], fixture: dict[str, Any], inputs: dict[str, Any]) -> str:
        payload = {
            "case": {
                "id": case.get("id", ""),
                "intent": case.get("intent", ""),
                "description": case.get("description", ""),
                "domain": case.get("domain", ""),
            },
            "provider": {
                "id": provider.get("id", ""),
                "command": provider.get("command", ""),
                "help": provider_help(provider),
            },
            "fixture": {
                "id": fixture.get("id", ""),
                "inputs": runtime_inputs(case, inputs),
            },
        }
        return json.dumps(payload, ensure_ascii=False, indent=2, sort_keys=True)

    def run_command(self, row: dict[str, Any], case: dict[str, Any], inputs: dict[str, Any], command: list[str], writes_target: bool, llm_duration: int) -> dict[str, Any]:
        started = time.monotonic()
        try:
            completed = subprocess.run(command, cwd=self.repo, text=True, capture_output=True, timeout=COMMAND_TIMEOUT_SECONDS)
        except subprocess.TimeoutExpired:
            return self.failed(row, failure("baseline_llm_oneshot", CODE_COMMAND_FAILED, "command timed out"), llm_duration)
        duration = elapsed_ms(started)
        row["duration_ms"] = llm_duration + duration
        row["inputs"] = summarize_inputs(self.home, runtime_inputs(case, inputs))
        if not writes_target:
            target = inputs.get("target")
            if not target:
                return self.failed(row, failure("baseline_llm_oneshot", CODE_MISSING_TARGET, "stdout capture requires a target input"), llm_duration)
            target_path = Path(str(target))
            target_path.parent.mkdir(parents=True, exist_ok=True)
            target_path.write_text(completed.stdout, encoding="utf-8")
        if completed.returncode != 0:
            message = completed.stderr.strip() or completed.stdout.strip() or f"exit status {completed.returncode}"
            return self.failed(row, failure("baseline_llm_oneshot", CODE_COMMAND_FAILED, message), row["duration_ms"])
        oracle = self.oracle.run(case, inputs)
        row["oracle"] = oracle
        row["status"] = STATUS_PASSED if oracle.get("passed") else STATUS_FAILED
        if not oracle.get("passed"):
            err = oracle.get("error") or {}
            row["failure"] = failure("baseline_oracle", err.get("code", ""), err.get("message", ""))
        row["steps"] = [
            flow_step(STEP_BASELINE_LLM_ONESHOT, STATUS_PASSED, duration_ms=row["duration_ms"], provider=row.get("provider", ""), llm_duration_ms=llm_duration),
            flow_step(STEP_BASELINE_ORACLE, row["status"], oracle_id=(oracle or {}).get("id", ""), failure=row.get("failure")),
        ]
        return row

    def skipped(self, row: dict[str, Any], err: dict[str, str]) -> dict[str, Any]:
        row["status"] = STATUS_SKIPPED
        row["failure"] = err
        row["steps"] = [
            flow_step(STEP_BASELINE_LLM_ONESHOT, STATUS_SKIPPED, failure=err),
            flow_step(STEP_BASELINE_ORACLE, STATUS_SKIPPED, failure=err),
        ]
        return row

    def failed(self, row: dict[str, Any], err: dict[str, str], duration_ms: int = 0) -> dict[str, Any]:
        row["status"] = STATUS_FAILED
        row["failure"] = err
        if duration_ms:
            row["duration_ms"] = duration_ms
        row["steps"] = [
            flow_step(STEP_BASELINE_LLM_ONESHOT, STATUS_FAILED, duration_ms=duration_ms, failure=err),
            flow_step(STEP_BASELINE_ORACLE, STATUS_SKIPPED, failure=err),
        ]
        return row


def provider_help(provider: dict[str, Any]) -> str:
    command = provider.get("command", "")
    attempts = []
    if provider.get("version_command"):
        attempts.append(provider["version_command"])
    attempts.extend([[command, "--help"], [command, "-h"]])
    for attempt in attempts:
        if not attempt or not shutil.which(str(attempt[0])):
            continue
        try:
            completed = subprocess.run(attempt, text=True, capture_output=True, timeout=HELP_TIMEOUT_SECONDS)
        except subprocess.TimeoutExpired:
            continue
        text = (completed.stdout + "\n" + completed.stderr).strip()
        if text:
            return text[:HELP_TEXT_LIMIT]
    return f"{command} command help unavailable"


def parse_llm_command(content: str) -> dict[str, Any] | None:
    parsed = parse_json(content.strip())
    if parsed:
        return parsed
    start = content.find("{")
    end = content.rfind("}")
    if start >= 0 and end > start:
        return parse_json(content[start : end + 1])
    return None


def validate_command(command: Any, provider_command: str) -> dict[str, str] | None:
    if not isinstance(command, list) or not command or not all(isinstance(item, str) and item for item in command):
        return failure("baseline_llm_oneshot", CODE_INVALID_COMMAND, "command must be a non-empty string array")
    if Path(command[0]).name != provider_command:
        return failure("baseline_llm_oneshot", CODE_INVALID_COMMAND, f"command must use provider command {provider_command}")
    return None


def current_platform() -> str:
    system = platform.system().lower()
    if system == "darwin":
        return "darwin"
    if system == "linux":
        return "linux"
    return system
