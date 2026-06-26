from __future__ import annotations

import importlib.util
import sys
import unittest
from pathlib import Path


RUNNER_PATH = Path(__file__).with_name("run.py")
sys.path.insert(0, str(RUNNER_PATH.parent))
SPEC = importlib.util.spec_from_file_location("cli_matrix_runner", RUNNER_PATH)
assert SPEC is not None
runner = importlib.util.module_from_spec(SPEC)
assert SPEC.loader is not None
SPEC.loader.exec_module(runner)


class CLIMatrixRunnerTest(unittest.TestCase):
    def test_summarize_counts_direct_reuse_and_intent_use(self) -> None:
        summary = runner.summarize(
            [
                {
                    "provider_path": "/bin/tool",
                    "scan_duration_ms": 10,
                    "llm_duration_ms": 20,
                    "candidates": [
                        {
                            "probe": {"passed": True},
                            "promotion": {"binding_id": "binding_a"},
                            "reuse": {"duration_ms": 5, "verified": True},
                            "use": {"duration_ms": 7, "verified": True},
                        }
                    ],
                }
            ]
        )

        self.assertEqual(summary["candidate_count"], 1)
        self.assertEqual(summary["promoted_bindings"], 1)
        self.assertEqual(summary["verified_reuses"], 1)
        self.assertEqual(summary["intent_uses"], 1)
        self.assertEqual(summary["verified_uses"], 1)
        self.assertEqual(summary["avg_use_ms"], 7)
        self.assertEqual(summary["reuse_success_rate"], 1)
        self.assertEqual(summary["use_success_rate"], 1)

    def test_validate_result_requires_intent_use(self) -> None:
        artifact = {"summary": {"verified_reuses": 1, "verified_uses": 0}}

        with self.assertRaises(SystemExit):
            runner.validate_result(runner.MODE_REPLAY, [{}], artifact)
        with self.assertRaises(SystemExit):
            runner.validate_result(runner.MODE_LIVE_LLM, [{}], artifact)

    def test_use_intent_prefers_candidate_description(self) -> None:
        intent = runner.use_intent(
            {"cli": "tool", "use_intent": "configured intent"},
            {"capability_id": "file.hash_sha1", "description": "Compute a SHA-1 hash."},
        )

        self.assertEqual(intent, "Compute a SHA-1 hash.")

    def test_use_inputs_lets_cal_generate_target(self) -> None:
        self.assertEqual(
            runner.use_inputs({"source": "/tmp/source.txt", "target": "/tmp/target.txt", "format": "json"}),
            {"source": "/tmp/source.txt", "format": "json"},
        )

    def test_use_failure_stage_groups_selection_input_and_verification(self) -> None:
        self.assertEqual(runner.use_failure_stage("no_match"), "use_selection_failed")
        self.assertEqual(runner.use_failure_stage("missing_inputs"), "use_input_failed")
        self.assertEqual(runner.use_failure_stage("verification_failed"), "use_verification_failed")
        self.assertEqual(runner.use_failure_stage("execution_failed"), "use_execution_failed")


if __name__ == "__main__":
    unittest.main()
