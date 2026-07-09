from __future__ import annotations

import subprocess
import platform
import shutil
import time
from pathlib import Path
from typing import Any

from constants import BASELINE_DIRECT_CLI, BASELINE_LLM_ONESHOT, MODE_REPLAY, STATUS_FAILED, STATUS_PASSED, STATUS_SKIPPED, STEP_BASELINE_DIRECT_CLI, STEP_BASELINE_ORACLE
from llm_oneshot import OneShotRunner
from oracle import OracleRunner
from reuse import materialize_inputs, reuse_rounds
from util import elapsed_ms, failure, flow_step, read_jsonl, render_template


class BaselineRunner:
    def __init__(self, repo: Path, bench: Path, home: Path, oracle: OracleRunner, mode: str = MODE_REPLAY) -> None:
        self.repo = repo
        self.bench = bench
        self.home = home
        self.oracle = oracle
        self.mode = mode
        self.commands = read_jsonl(bench / "baselines" / "oracle" / "commands.jsonl")
        self.llm_oneshot = OneShotRunner(repo, bench, home, mode, oracle)

    def run_case(self, case: dict[str, Any]) -> dict[str, Any]:
        baselines: dict[str, Any] = {}
        if BASELINE_DIRECT_CLI in (case.get("baselines") or []):
            baselines[BASELINE_DIRECT_CLI] = self.run_direct_cli(case)
        if BASELINE_LLM_ONESHOT in (case.get("baselines") or []):
            baselines[BASELINE_LLM_ONESHOT] = self.llm_oneshot.run_case(case)
        return baselines

    def run_direct_cli(self, case: dict[str, Any]) -> list[dict[str, Any]]:
        rows = []
        commands = [command for command in self.commands if command.get("case_id") == case["id"]]
        for command in commands:
            for fixture in reuse_rounds(case):
                rows.append(self.run_direct_cli_fixture(case, command, fixture))
        return rows

    def run_direct_cli_fixture(self, case: dict[str, Any], command: dict[str, Any], fixture: dict[str, Any]) -> dict[str, Any]:
        inputs = materialize_inputs(self.bench, self.home, case["id"], fixture)
        rendered = [render_template(str(arg), inputs) for arg in command.get("command") or []]
        skip = unavailable_command(command, rendered)
        if skip:
            return {
                "id": command.get("id", ""),
                "provider": command.get("provider", ""),
                "fixture_id": fixture.get("id", ""),
                "status": STATUS_SKIPPED,
                "failure": skip,
                "steps": [
                    flow_step(STEP_BASELINE_DIRECT_CLI, STATUS_SKIPPED, failure=skip),
                    flow_step(STEP_BASELINE_ORACLE, STATUS_SKIPPED, failure=skip),
                ],
            }
        started = time.monotonic()
        completed = subprocess.run(rendered, cwd=self.repo, text=True, capture_output=True)
        duration = elapsed_ms(started)
        result: dict[str, Any] = {
            "id": command.get("id", ""),
            "provider": command.get("provider", ""),
            "fixture_id": fixture.get("id", ""),
            "duration_ms": duration,
            "steps": [],
        }
        stdout_input = command.get("stdout_path_input")
        if stdout_input and inputs.get(stdout_input):
            target = Path(inputs[stdout_input])
            target.parent.mkdir(parents=True, exist_ok=True)
            target.write_text(completed.stdout, encoding="utf-8")
        if completed.returncode != 0:
            result["status"] = STATUS_FAILED
            result["failure"] = failure("baseline_direct_cli", "command_failed", completed.stderr.strip() or completed.stdout.strip() or f"exit status {completed.returncode}")
            result["steps"] = [
                flow_step(STEP_BASELINE_DIRECT_CLI, STATUS_FAILED, duration_ms=duration, failure=result["failure"]),
                flow_step(STEP_BASELINE_ORACLE, STATUS_SKIPPED, failure=result["failure"]),
            ]
            return result
        oracle = self.oracle.run(case, inputs)
        result["oracle"] = oracle
        result["status"] = STATUS_PASSED if oracle.get("passed") else STATUS_FAILED
        if not oracle.get("passed"):
            err = oracle.get("error") or {}
            result["failure"] = failure("baseline_oracle", err.get("code", ""), err.get("message", ""))
        result["steps"] = [
            flow_step(STEP_BASELINE_DIRECT_CLI, STATUS_PASSED, duration_ms=duration, provider=command.get("provider", "")),
            flow_step(STEP_BASELINE_ORACLE, result["status"], oracle_id=(oracle or {}).get("id", ""), failure=result.get("failure")),
        ]
        return result


def unavailable_command(command: dict[str, Any], rendered: list[str]) -> dict[str, str] | None:
    platforms = command.get("platforms") or []
    if platforms and current_platform() not in platforms:
        return failure("baseline_direct_cli", "platform_unavailable", f"baseline command is not configured for {current_platform()}")
    if not rendered or not shutil.which(rendered[0]):
        return failure("baseline_direct_cli", "cli_unavailable", f"{rendered[0] if rendered else 'command'} was not found on PATH")
    return None


def current_platform() -> str:
    system = platform.system().lower()
    if system == "darwin":
        return "darwin"
    if system == "linux":
        return "linux"
    return system
