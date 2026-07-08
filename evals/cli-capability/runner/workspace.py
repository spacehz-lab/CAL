from __future__ import annotations

import subprocess
import time
from pathlib import Path
from typing import Any

from util import parse_json


class Workspace:
    def __init__(self, repo: Path, home: Path, calctl: str, env: dict[str, str]) -> None:
        self.repo = repo
        self.home = home
        self.calctl = calctl
        self.env = env

    def run_calctl(self, args: list[str]) -> subprocess.CompletedProcess[str]:
        return subprocess.run([self.calctl, *args], cwd=self.repo, env=self.env, text=True, capture_output=True)

    def trace_ids(self) -> set[str]:
        root = self.home / "traces"
        if root.exists():
            return {path.name for path in root.iterdir() if path.is_dir()}
        return set()

    def new_trace_id(self, before: set[str]) -> str:
        return next(iter(self.trace_ids() - before), "")

    def trace_path(self, trace_id: str) -> Path:
        return self.home / "traces" / trace_id / "trace.json"


def start_cald(cald: str, repo: Path, env: dict[str, str], run_dir: Path) -> subprocess.Popen[Any]:
    run_dir.mkdir(parents=True, exist_ok=True)
    log = open(run_dir / "cald.log", "ab")
    process = subprocess.Popen([cald, "serve"], cwd=repo, env=env, stdout=log, stderr=log)
    process._cal_log = log  # type: ignore[attr-defined]
    return process


def wait_for_cald(calctl: str, repo: Path, env: dict[str, str]) -> None:
    last = ""
    for _ in range(100):
        completed = subprocess.run([calctl, "daemon", "status", "--json"], cwd=repo, env=env, text=True, capture_output=True)
        if completed.returncode == 0:
            status_doc = parse_json(completed.stdout) or {}
            if status_doc.get("running") is True:
                return
        last = completed.stderr.strip() or completed.stdout.strip()
        time.sleep(0.1)
    raise SystemExit(f"cald did not become ready: {last}")


def stop_process(process: subprocess.Popen[Any]) -> None:
    if process.poll() is None:
        process.terminate()
        try:
            process.wait(timeout=5)
        except subprocess.TimeoutExpired:
            process.kill()
            process.wait(timeout=5)
    log = getattr(process, "_cal_log", None)
    if log is not None:
        log.close()
