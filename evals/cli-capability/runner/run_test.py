from __future__ import annotations

import importlib.util
import json
import os
import tempfile
import unittest
from pathlib import Path
from unittest import mock


RUNNER_PATH = Path(__file__).with_name("run.py")
SPEC = importlib.util.spec_from_file_location("cal_cli_v0_runner", RUNNER_PATH)
assert SPEC is not None
runner = importlib.util.module_from_spec(SPEC)
assert SPEC.loader is not None
SPEC.loader.exec_module(runner)


class TaskCatalogTest(unittest.TestCase):
    def test_select_rejects_unknown_provider(self) -> None:
        with tempfile.TemporaryDirectory() as temp:
            bench = Path(temp)
            (bench / "tasks.jsonl").write_text(
                json.dumps({"id": "task_a", "intent": "hash this file", "provider_candidates": ["missing"]}) + "\n",
                encoding="utf-8",
            )
            (bench / "providers.json").write_text(json.dumps({"providers": []}), encoding="utf-8")

            catalog = runner.TaskCatalog(bench)

            with self.assertRaises(SystemExit):
                catalog.select("task_a")

    def test_select_defaults_to_hash_task(self) -> None:
        with tempfile.TemporaryDirectory() as temp:
            bench = Path(temp)
            (bench / "tasks.jsonl").write_text(
                json.dumps(
                    {
                        "id": "file_hash_sha1",
                        "intent": "hash this file",
                        "provider_candidates": ["shasum"],
                    }
                )
                + "\n",
                encoding="utf-8",
            )
            (bench / "providers.json").write_text(
                json.dumps({"providers": [{"id": "shasum", "command": "shasum"}]}),
                encoding="utf-8",
            )

            selected = runner.TaskCatalog(bench).select("")

            self.assertEqual(selected[0]["id"], "file_hash_sha1")
            self.assertEqual(selected[0]["providers"][0]["id"], "shasum")

    def test_select_level_focus_and_full(self) -> None:
        with tempfile.TemporaryDirectory() as temp:
            bench = Path(temp)
            (bench / "tasks.jsonl").write_text(
                "\n".join(
                    [
                        json.dumps({"id": "focus_a", "level": "focus", "intent": "a", "provider_candidates": ["provider_a"]}),
                        json.dumps({"id": "focus_b", "level": "focus", "intent": "b", "provider_candidates": ["provider_a"]}),
                        json.dumps({"id": "full_a", "level": "full", "intent": "c", "provider_candidates": ["provider_a"]}),
                    ]
                )
                + "\n",
                encoding="utf-8",
            )
            (bench / "providers.json").write_text(
                json.dumps({"providers": [{"id": "provider_a", "command": "tool"}]}),
                encoding="utf-8",
            )
            catalog = runner.TaskCatalog(bench)

            self.assertEqual([task["id"] for task in catalog.select("", "focus")], ["focus_a", "focus_b"])
            self.assertEqual([task["id"] for task in catalog.select("", "full")], ["focus_a", "focus_b", "full_a"])
            self.assertEqual([task["id"] for task in catalog.select("full_a", "focus")], ["full_a"])


class RunnerHelpersTest(unittest.TestCase):
    def test_materialize_inputs_expands_fixture_and_work_paths(self) -> None:
        with tempfile.TemporaryDirectory() as temp:
            bench = Path(temp) / "bench"
            home = Path(temp) / "home"
            fixture = bench / "fixtures" / "file_hash_sha1" / "reuse" / "b.txt"
            fixture.parent.mkdir(parents=True)
            fixture.write_text("hello", encoding="utf-8")

            inputs = runner.materialize_inputs(
                bench,
                home,
                "file_hash_sha1",
                {
                    "id": "text_b",
                    "inputs": {
                        "source": "fixtures/file_hash_sha1/reuse/b.txt",
                        "target": "{work}/sha1.txt",
                    },
                },
            )

            self.assertEqual(inputs["source"], str(fixture.resolve()))
            self.assertEqual(inputs["target"], str(home / "benchmark" / "file_hash_sha1" / "reuse" / "text_b" / "sha1.txt"))
            self.assertTrue(Path(inputs["target"]).parent.exists())

    def test_provider_oracles_passed_requires_every_reuse_oracle(self) -> None:
        provider = {
            "candidates": [
                {
                    "reuse": [
                        {"oracle": {"passed": True}},
                        {"oracle": {"passed": False}},
                    ]
                }
            ]
        }

        self.assertFalse(runner.provider_oracles_passed(provider))

        provider["candidates"][0]["reuse"][1]["oracle"]["passed"] = True
        self.assertTrue(runner.provider_oracles_passed(provider))

    def test_provider_oracles_passed_ignores_skipped_reuse(self) -> None:
        provider = {
            "candidates": [
                {
                    "reuse": [
                        {"status": runner.STATUS_SKIPPED, "skip": {"stage": "reuse_skipped", "code": "missing_runtime_inputs"}},
                        {"oracle": {"passed": True}},
                    ]
                }
            ]
        }

        self.assertTrue(runner.provider_oracles_passed(provider))

    def test_provider_has_promoted_binding(self) -> None:
        self.assertFalse(runner.provider_has_promoted_binding({"candidates": [{"promotion": {}}]}))
        self.assertTrue(
            runner.provider_has_promoted_binding(
                {"candidates": [{"promotion": {"binding_id": "binding_a"}}]}
            )
        )

    def test_required_runtime_inputs_include_templates_and_stdout_path(self) -> None:
        candidate = {
            "execution": {
                "spec": {
                    "args": ["-a", "{{ algorithm }}", "{{source}}"],
                    "stdout_path_input": "target",
                }
            }
        }

        self.assertEqual(runner.required_runtime_inputs(candidate), ["algorithm", "source", "target"])

    def test_missing_runtime_inputs_reports_only_absent_values(self) -> None:
        candidate = {"execution": {"spec": {"args": ["{{source}}", "{{format}}"], "stdout_path_input": "target"}}}

        self.assertEqual(
            runner.missing_runtime_inputs(candidate, {"source": "a.txt", "target": "out.txt"}),
            ["format"],
        )

    def test_runtime_inputs_strip_oracle_only_expected(self) -> None:
        task = {"oracle": {"expected_input": "expected", "actual_input": "target"}}
        inputs = {"source": "source.txt", "target": "out.txt", "expected": "want.txt"}

        self.assertEqual(runner.runtime_inputs(task, inputs), {"source": "source.txt", "target": "out.txt"})
        self.assertEqual(runner.use_inputs(task, inputs), {"source": "source.txt"})

    def test_use_oracle_inputs_use_generated_target(self) -> None:
        inputs = {"source": "source.txt", "target": "fixture-target.txt", "expected": "want.txt"}
        output = {"run": {"inputs": {"source": "source.txt", "target": "generated-target.txt"}}}

        self.assertEqual(
            runner.use_oracle_inputs(inputs, output),
            {"source": "source.txt", "target": "generated-target.txt", "expected": "want.txt"},
        )

    def test_summarize_counts_direct_reuse_and_intent_use(self) -> None:
        tasks = [
            {
                "providers": [
                    {
                        "provider_path": "/bin/tool",
                        "acquisition_duration_ms": 10,
                        "candidates": [
                            {
                                "probe": {"passed": True},
                                "promotion": {"binding_id": "binding_a"},
                                "reuse": [
                                    {"duration_ms": 5, "oracle": {"passed": True}},
                                    {"status": runner.STATUS_SKIPPED, "skip": {"stage": "reuse_skipped", "code": "missing_runtime_inputs"}},
                                ],
                            }
                        ],
                    }
                ],
                "use": [
                    {
                        "duration_ms": 7,
                        "selection": {"source": "llm"},
                        "oracle": {"passed": True},
                    }
                ],
            }
        ]

        summary = runner.summarize(tasks)

        self.assertEqual(summary["held_out_reuses"], 1)
        self.assertEqual(summary["skipped_reuses"], 1)
        self.assertEqual(summary["oracle_pass_count"], 1)
        self.assertEqual(summary["held_out_uses"], 1)
        self.assertEqual(summary["use_oracle_pass_count"], 1)
        self.assertEqual(summary["run_stage_llm_calls"], 1)
        self.assertEqual(summary["oracle_use_success_rate"], 1)

    def test_summarize_live_separates_negative_evidence_from_closed_loop_failures(self) -> None:
        tasks = [
            {
                "providers": [
                    {
                        "provider_path": "/bin/tool",
                        "failure": {"stage": "acquisition_failed", "code": "candidate_proposal_failed"},
                        "candidates": [{"probe": {"status": "failed"}}],
                    }
                ],
                "use": [{"oracle": {"passed": True}}],
            }
        ]

        summary = runner.summarize(tasks, runner.MODE_LIVE_LLM)

        self.assertEqual(summary["provider_failures"], 1)
        self.assertEqual(summary["candidate_negative_evidence"], 1)
        self.assertEqual(summary["failed"], 0)
        self.assertEqual(summary["use_oracle_pass_count"], 1)

    def test_score_replay_includes_direct_reuse(self) -> None:
        summary = {
            "oracle_pass_count": 2,
            "held_out_reuses": 2,
            "use_oracle_pass_count": 1,
            "held_out_uses": 1,
            "providers_with_promoted_bindings": 1,
            "provider_available": 2,
            "probe_pass_count": 3,
            "candidate_count": 4,
            "promoted_bindings": 2,
            "provider_failures": 1,
            "candidate_negative_evidence": 1,
            "failed": 0,
            "acquisition_duration_ms": 3000,
            "reuse_duration_ms": 40,
            "use_duration_ms": 50,
            "llm_duration_ms": 0,
            "avg_acquisition_ms": 1000,
            "avg_reuse_ms": 20,
            "avg_use_ms": 50,
            "avg_llm_ms": 0,
        }

        scores = runner.score(summary, runner.MODE_REPLAY)

        self.assertEqual(scores["profile"], runner.MODE_REPLAY)
        self.assertEqual(scores["closed_loop_success_rate"], 1)
        self.assertEqual(scores["direct_reuse_success_rate"], 1)
        self.assertEqual(scores["intent_use_success_rate"], 1)
        self.assertEqual(scores["provider_yield_rate"], 0.5)
        self.assertEqual(scores["provider_acquisition_total_ms"], 3000)
        self.assertEqual(scores["replay_direct_reuse_avg_ms"], 20)

    def test_score_live_uses_intent_path_without_direct_reuse(self) -> None:
        summary = {
            "oracle_pass_count": 0,
            "held_out_reuses": 0,
            "use_oracle_pass_count": 4,
            "held_out_uses": 5,
            "providers_with_promoted_bindings": 2,
            "provider_available": 4,
            "probe_pass_count": 3,
            "candidate_count": 6,
            "promoted_bindings": 3,
            "provider_failures": 1,
            "candidate_negative_evidence": 2,
            "failed": 1,
            "acquisition_duration_ms": 300_000,
            "reuse_duration_ms": 0,
            "use_duration_ms": 7_500,
            "llm_duration_ms": 240_000,
            "avg_acquisition_ms": 60_000,
            "avg_reuse_ms": 0,
            "avg_use_ms": 1_500,
            "avg_llm_ms": 45_000,
        }

        scores = runner.score(summary, runner.MODE_LIVE_LLM)

        self.assertEqual(scores["profile"], runner.MODE_LIVE_LLM)
        self.assertEqual(scores["closed_loop_success_rate"], 0.8)
        self.assertEqual(scores["direct_reuse_success_rate"], None)
        self.assertEqual(scores["provider_yield_rate"], 0.5)
        self.assertEqual(scores["negative_evidence_count"], 2)
        self.assertEqual(scores["proposal_llm_total_ms"], 240_000)
        self.assertEqual(scores["acquisition_local_overhead_total_ms"], 60_000)
        self.assertEqual(scores["replay_direct_reuse_total_ms"], None)

    def test_render_html_shows_main_result_and_timing_before_raw_details(self) -> None:
        html = runner.render_html(
            {
                "scores": {"profile": runner.MODE_LIVE_LLM, "closed_loop_success_rate": 1.0, "proposal_llm_avg_ms": 60_000},
                "summary": {"held_out_uses": 1},
                "tasks": [],
            }
        )

        self.assertIn("<h2>Main Result</h2>", html)
        self.assertIn("<h2>Workflow Timing</h2>", html)
        self.assertIn("Closed-loop success", html)
        self.assertIn("100.0%", html)
        self.assertIn("1.0 min", html)
        self.assertIn("<summary>Raw scores</summary>", html)

    def test_failure_taxonomy_counts_provider_probe_and_reuse_failures(self) -> None:
        tasks = [
            {
                "providers": [
                    {
                        "failure": {"stage": "acquisition_failed", "code": "candidate_proposal_failed"},
                        "candidates": [
                            {
                                "probe": {"error": {"stage": "probe_failed", "code": "verification_failed"}},
                                "reuse": [
                                    {"failure": {"stage": "oracle_failure", "code": "bytes_mismatch"}},
                                    {"failure": {"stage": "oracle_failure", "code": "bytes_mismatch"}},
                                ],
                            }
                        ],
                    }
                ],
                "use": [
                    {"failure": {"stage": "use_failed", "code": "no_match"}},
                ],
            }
        ]

        self.assertEqual(
            runner.failure_taxonomy(tasks),
            [
                {"stage": "acquisition_failed", "code": "candidate_proposal_failed", "count": 1},
                {"stage": "oracle_failure", "code": "bytes_mismatch", "count": 2},
                {"stage": "probe_failed", "code": "verification_failed", "count": 1},
                {"stage": "use_failed", "code": "no_match", "count": 1},
            ],
        )

    def test_new_run_id_uses_timestamp_mode_and_hash(self) -> None:
        self.assertRegex(runner.new_run_id("replay"), r"^\d{8}-\d{6}-replay-[0-9a-f]{6}$")

    def test_new_run_id_uses_model_label_for_live_llm(self) -> None:
        self.assertRegex(
            runner.new_run_id("live_llm", "Kimi K2.7 Code"),
            r"^\d{8}-\d{6}-kimi-k2-7-code-[0-9a-f]{6}$",
        )

    def test_new_artifact_records_non_secret_live_llm_settings(self) -> None:
        args = argparse_like(mode=runner.MODE_LIVE_LLM, level="focus")

        with mock.patch.dict(
            os.environ,
            {"CAL_LLM_API": "chat_completions", "CAL_LLM_BASE_URL": "https://api.example.test/v1"},
            clear=False,
        ):
            artifact = runner.new_artifact("run_1", args, [], "test-model")

        self.assertEqual(
            artifact["llm"],
            {"api": "chat_completions", "model": "test-model", "base_url_configured": True},
        )

    def test_validate_result_accepts_live_intent_use_without_direct_reuse(self) -> None:
        artifact = {
            "summary": {
                "oracle_pass_count": 0,
                "use_oracle_pass_count": 1,
                "oracle_fail_count": 2,
                "failed": 3,
            }
        }

        runner.validate_result(runner.MODE_LIVE_LLM, artifact)

        with self.assertRaises(SystemExit):
            runner.validate_result(runner.MODE_REPLAY, artifact)

    def test_validate_result_requires_intent_use(self) -> None:
        artifact = {"summary": {"oracle_pass_count": 1, "use_oracle_pass_count": 0}}

        with self.assertRaises(SystemExit):
            runner.validate_result(runner.MODE_LIVE_LLM, artifact)


def argparse_like(**values):
    return type("Args", (), values)()


if __name__ == "__main__":
    unittest.main()
