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
                json.dumps({"id": "task_a", "capability_goal": "file.hash_sha1", "provider_candidates": ["missing"]}) + "\n",
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
                        "capability_goal": "file.hash_sha1",
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
                        json.dumps({"id": "focus_a", "level": "focus", "capability_goal": "a", "provider_candidates": ["provider_a"]}),
                        json.dumps({"id": "focus_b", "level": "focus", "capability_goal": "b", "provider_candidates": ["provider_a"]}),
                        json.dumps({"id": "full_a", "level": "full", "capability_goal": "c", "provider_candidates": ["provider_a"]}),
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

    def test_candidate_matches_task_accepts_parameterized_capability(self) -> None:
        task = {"capability_goal": "file.hash_sha1", "accepted_capability_ids": ["file.hash_algorithm"]}

        self.assertTrue(runner.candidate_matches_task(task, {"capability_id": "file.hash_sha1"}))
        self.assertTrue(runner.candidate_matches_task(task, {"capability_id": "file.hash_algorithm"}))
        self.assertFalse(runner.candidate_matches_task(task, {"capability_id": "file.hash_md5"}))

    def test_input_overrides_are_provider_and_capability_specific(self) -> None:
        task = {
            "reuse": {
                "input_overrides": {
                    "file.hash_algorithm": {
                        "shasum": {"algorithm": "1"},
                        "openssl": {"algorithm": "sha1"},
                    }
                }
            }
        }

        self.assertEqual(runner.input_overrides(task, "shasum", "file.hash_algorithm"), {"algorithm": "1"})
        self.assertEqual(runner.input_overrides(task, "openssl", "file.hash_algorithm"), {"algorithm": "sha1"})
        self.assertEqual(runner.input_overrides(task, "shasum", "file.hash_sha1"), {})

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
                ]
            }
        ]

        self.assertEqual(
            runner.failure_taxonomy(tasks),
            [
                {"stage": "acquisition_failed", "code": "candidate_proposal_failed", "count": 1},
                {"stage": "oracle_failure", "code": "bytes_mismatch", "count": 2},
                {"stage": "probe_failed", "code": "verification_failed", "count": 1},
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

    def test_validate_result_is_strict_for_replay_and_tolerant_for_live(self) -> None:
        artifact = {"summary": {"oracle_pass_count": 1, "oracle_fail_count": 0, "failed": 1}}

        runner.validate_result(runner.MODE_LIVE_LLM, artifact)

        with self.assertRaises(SystemExit):
            runner.validate_result(runner.MODE_REPLAY, artifact)


def argparse_like(**values):
    return type("Args", (), values)()


if __name__ == "__main__":
    unittest.main()
