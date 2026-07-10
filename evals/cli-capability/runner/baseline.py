from __future__ import annotations

from pathlib import Path
from typing import Any

from constants import BASELINE_LLM_ONESHOT, MODE_REPLAY
from llm_oneshot import OneShotRunner
from oracle import OracleRunner


class BaselineRunner:
    def __init__(self, repo: Path, bench: Path, home: Path, oracle: OracleRunner, mode: str = MODE_REPLAY, env: dict[str, str] | None = None) -> None:
        self.repo = repo
        self.bench = bench
        self.home = home
        self.oracle = oracle
        self.mode = mode
        self.env = env
        self.llm_oneshot = OneShotRunner(repo, bench, home, mode, oracle, env=env)

    def run_case(self, case: dict[str, Any]) -> dict[str, Any]:
        baselines: dict[str, Any] = {}
        if BASELINE_LLM_ONESHOT in (case.get("baselines") or []):
            baselines[BASELINE_LLM_ONESHOT] = self.llm_oneshot.run_case(case)
        return baselines
