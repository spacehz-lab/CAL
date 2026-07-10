from __future__ import annotations

import importlib.util
import json
import os
import sys
import tempfile
import unittest
from pathlib import Path
from unittest import mock


RUNNER_DIR = Path(__file__).resolve().parent
if str(RUNNER_DIR) not in sys.path:
    sys.path.insert(0, str(RUNNER_DIR))


def load_module(name: str):
    path = RUNNER_DIR / f"{name}.py"
    spec = importlib.util.spec_from_file_location(f"cal_cli_eval_{name}", path)
    assert spec is not None
    module = importlib.util.module_from_spec(spec)
    assert spec.loader is not None
    spec.loader.exec_module(module)
    return module


catalog = load_module("catalog")
constants = load_module("constants")
reuse = load_module("reuse")
acquisition = load_module("acquisition")
oracle = load_module("oracle")
llm_oneshot = load_module("llm_oneshot")
llm_client = load_module("llm_client")
summary = load_module("summary")
report = load_module("report")
export_result = load_module("export_result")
run = load_module("run")
seed = load_module("seed")
validate = load_module("validate")


class ScenarioCatalogTest(unittest.TestCase):
    def test_select_reads_scenario_cases(self) -> None:
        with tempfile.TemporaryDirectory() as temp:
            bench = Path(temp)
            write_catalog_fixture(bench)

            selected = catalog.ScenarioCatalog(bench).select("acquisition,repeated_reuse", "focus")

            self.assertEqual([case["case_key"] for case in selected], ["acquisition:case_a", "repeated_reuse:case_b"])
            self.assertEqual(selected[0]["providers"][0]["id"], "tool_a")

    def test_select_filters_by_tag_and_provider_class(self) -> None:
        with tempfile.TemporaryDirectory() as temp:
            bench = Path(temp)
            write_catalog_fixture(bench)

            selected = catalog.ScenarioCatalog(bench).select("acquisition,repeated_reuse", "focus", provider_classes="third_party_cli", scenario_tags="repeated_reuse")

            self.assertEqual([case["case_key"] for case in selected], ["repeated_reuse:case_b"])

    def test_select_rejects_unknown_experiment(self) -> None:
        with tempfile.TemporaryDirectory() as temp:
            bench = Path(temp)
            write_catalog_fixture(bench)

            with self.assertRaises(SystemExit):
                catalog.ScenarioCatalog(bench).select("unknown", "focus")


class ReuseHelpersTest(unittest.TestCase):
    def test_materialize_inputs_expands_round_and_work_paths(self) -> None:
        with tempfile.TemporaryDirectory() as temp:
            bench = Path(temp) / "bench"
            home = Path(temp) / "home"
            fixture = bench / "fixtures" / "case_a" / "reuse" / "b.txt"
            fixture.parent.mkdir(parents=True)
            fixture.write_text("hello", encoding="utf-8")

            inputs = reuse.materialize_inputs(
                bench,
                home,
                "case_a",
                {"id": "round_1", "inputs": {"source": "fixtures/case_a/reuse/b.txt", "target": "{work}/out.txt"}},
            )

            self.assertEqual(inputs["source"], str(fixture.resolve()))
            self.assertEqual(inputs["target"], str(home / "benchmark" / "case_a" / "reuse" / "round_1" / "out.txt"))
            self.assertEqual(reuse.reuse_rounds({"reuse": {"rounds": [{"id": "round_1"}]}}), [{"id": "round_1"}])


class SummaryTest(unittest.TestCase):
    def test_summarize_reports_end_to_end_and_conditional_reuse(self) -> None:
        cases = [
            case_result("case_a", ["repeated_reuse"], "provider_a", "cap_a", "binding_a", use_passed=True),
            case_result("case_b", ["repeated_reuse"], "provider_b", "cap_b", "", use_passed=False),
        ]

        result = summary.summarize(cases, constants.MODE_LIVE_LLM)
        scores = summary.score(result, constants.MODE_LIVE_LLM)

        reuse_summary = result["experiments"]["repeated_reuse"]
        self.assertEqual(reuse_summary["held_out_uses"], 2)
        self.assertEqual(reuse_summary["use_oracle_pass_count"], 1)
        self.assertEqual(reuse_summary["eligible_held_out_uses"], 1)
        self.assertEqual(reuse_summary["conditional_use_oracle_pass_count"], 1)
        self.assertEqual(reuse_summary["upstream_acquisition_failure_count"], 1)
        self.assertEqual(scores["closed_loop_success_rate"], 0.5)
        self.assertEqual(scores["conditional_reuse_success_rate"], 1.0)

    def test_capability_model_and_gates_use_paper_experiments(self) -> None:
        cases = [
            case_result("case_a", ["acquisition", "capability_structure"], "provider_a", "cap_a", "binding_a", checks={"expect_multi_binding": True, "expected_provider_count": 2}),
            case_result("case_b", ["acquisition", "capability_structure"], "provider_b", "cap_a", "binding_b"),
            case_result("case_c", ["verification_failure"], "provider_c", "cap_c", ""),
        ]

        artifact = {"cases": cases}
        summary.update_artifact_metrics(artifact, constants.MODE_REPLAY)

        self.assertEqual(artifact["capability_model"]["multi_binding_capabilities"], 1)
        self.assertEqual(artifact["capability_model"]["check_passed"], 1)
        self.assertTrue(artifact["experiment_gates"]["verification_failure"]["passed"])
        self.assertEqual(artifact["experiment_gates"]["verification_failure"]["false_promotions"], 0)

    def test_summarize_splits_acquisition_modes(self) -> None:
        cases = [
            case_result("case_a", ["acquisition"], "provider_a", "cap_a", "binding_a", acquisition_mode="intent_guided"),
            case_result("case_b", ["acquisition"], "provider_b", "cap_b", "", acquisition_mode="full_acquisition"),
        ]

        result = summary.summarize(cases, constants.MODE_LIVE_LLM)

        self.assertEqual(result["acquisition_modes"]["intent_guided"]["case_attempted"], 1)
        self.assertEqual(result["acquisition_modes"]["intent_guided"]["providers_with_promoted_bindings"], 1)
        self.assertEqual(result["acquisition_modes"]["full_acquisition"]["case_attempted"], 1)
        self.assertEqual(result["acquisition_modes"]["full_acquisition"]["providers_with_promoted_bindings"], 0)

    def test_summarize_baselines_counts_repeated_reuse_only(self) -> None:
        cases = [
            {"paper_experiments": ["acquisition"], "baselines": {"llm_oneshot": [{"duration_ms": 3, "oracle": {"passed": True}}]}},
            {"paper_experiments": ["repeated_reuse"], "baselines": {"llm_oneshot": [{"duration_ms": 11, "oracle": {"passed": True}}]}},
        ]

        result = summary.summarize(cases, constants.MODE_REPLAY)

        self.assertEqual(result["baselines"]["llm_oneshot"]["attempted"], 1)
        self.assertEqual(result["baselines"]["llm_oneshot"]["duration_ms"], 11)

    def test_discovery_coverage_matches_promoted_expected_surfaces(self) -> None:
        cases = [
            case_result(
                "suite_a",
                ["acquisition"],
                "provider_a",
                "cap_a",
                "binding_a",
                acquisition_mode="full_acquisition",
                expected_capabilities=[
                    {"key": "alpha", "surface": "alpha", "min_promoted_bindings": 1},
                    {"key": "beta", "surface": "beta", "min_promoted_bindings": 1},
                ],
                candidates=[
                    candidate_result("cap.alpha", "binding_alpha", ["alpha", "--input", "{{source}}"]),
                    candidate_result("cap.beta", "binding_beta", ["beta", "--input", "{{source}}"]),
                    candidate_result("cap.gamma", "", ["gamma", "--input", "{{source}}"]),
                ],
            )
        ]

        result = summary.discovery_coverage(cases)

        self.assertEqual(result["expected_capabilities"], 2)
        self.assertEqual(result["promoted_expected_capabilities"], 2)
        self.assertEqual(result["multi_cap_design_rate"], 1.0)
        self.assertEqual(result["multi_cap_promoted_rate"], 1.0)
        self.assertEqual(result["cases"][0]["matched_surfaces"], ["alpha", "beta"])
        self.assertEqual(result["cases"][0]["extra_promoted_surfaces"], [])


class ReportTest(unittest.TestCase):
    def test_build_flow_artifact_uses_cases(self) -> None:
        artifact = {
            "run": {"id": "run_1", "mode": constants.MODE_REPLAY},
            "cases": [case_result("case_a", ["acquisition"], "provider_a", "cap_a", "binding_a")],
            "summary": {},
            "scores": {},
        }

        flow = report.build_flow_artifact(artifact)

        self.assertEqual(flow["schema_version"], constants.FLOW_SCHEMA_VERSION)
        self.assertEqual(flow["cases"][0]["id"], "case_a")
        self.assertEqual(flow["cases"][0]["providers"][0]["acquisition_duration_ms"], 10)

    def test_render_html_shows_paper_sections(self) -> None:
        artifact = {
            "run": {
                "id": "run_1",
                "mode": constants.MODE_REPLAY,
                "selected_experiments": ["acquisition", "repeated_reuse"],
                "selected_cases": ["acquisition:case_a"],
                "jobs": 4,
            },
            "status": "completed",
            "cases": [case_result("case_a", ["acquisition", "repeated_reuse"], "provider_a", "cap_a", "binding_a", use_passed=True)],
            "summary": {
                "experiments": {
                    "repeated_reuse": {
                        "held_out_uses": 1,
                        "use_oracle_pass_count": 1,
                        "eligible_held_out_uses": 1,
                        "conditional_use_oracle_pass_count": 1,
                        "avg_use_ms": 7,
                        "run_stage_llm_calls": 0,
                        "total_tokens": 0,
                    }
                },
                "baselines": {"llm_oneshot": {"attempted": 1, "passed": 1, "success_rate": 1.0, "avg_duration_ms": 10, "llm_calls": 1, "total_tokens": 42}},
            },
            "coverage": {"distinct_case_count": 1},
            "experiment_gates": {"acquisition": {"numerator": 1, "denominator": 1, "actual": 1, "target": 0.85, "passed": True}},
            "capability_model": {"providers": {}, "capabilities": {}, "checks": []},
            "scores": {},
        }

        html = report.render_html(artifact)

        self.assertIn("Experiment 1: Acquiring Capabilities From Provider Surfaces", html)
        self.assertIn("Experiment 3: Repeated Held-Out Reuse", html)
        self.assertNotIn("Experiment 2: Verification And Failure Gating", html)
        self.assertNotIn("Experiment 4: Capability Structure Evidence", html)
        self.assertIn("Selected experiments and cases", html)
        self.assertIn("Repeated Reuse Method Comparison", html)

    def test_render_html_splits_acquisition_modes_and_shows_discovery_coverage(self) -> None:
        artifact = {
            "run": {"id": "run_1", "mode": constants.MODE_LIVE_LLM, "selected_experiments": ["acquisition"], "selected_cases": [], "jobs": 1},
            "status": "completed",
            "cases": [
                case_result("intent_case", ["acquisition"], "provider_a", "alpha", "binding_a"),
                case_result(
                    "suite_case",
                    ["acquisition"],
                    "provider_b",
                    "beta",
                    "binding_b",
                    acquisition_mode="full_acquisition",
                    expected_capabilities=[
                        {"key": "alpha", "surface": "alpha", "min_promoted_bindings": 1},
                        {"key": "beta", "surface": "beta", "min_promoted_bindings": 1},
                    ],
                    candidates=[
                        candidate_result("cap.alpha", "binding_alpha", ["alpha", "--input", "{{source}}"]),
                        candidate_result("cap.beta", "binding_beta", ["beta", "--input", "{{source}}"]),
                    ],
                ),
            ],
        }
        summary.update_artifact_metrics(artifact, constants.MODE_LIVE_LLM)

        html = report.render_html(artifact)

        self.assertIn("Intent-guided Acquisition Runs", html)
        self.assertIn("Full Acquisition Runs", html)
        self.assertIn("Full Acquisition Discovery Coverage", html)
        self.assertIn("alpha, beta", html)
        self.assertIn("10 ms / 0", html)

    def test_render_html_uses_selected_experiments_for_single_group(self) -> None:
        artifact = {
            "run": {
                "id": "run_1",
                "mode": constants.MODE_LIVE_LLM,
                "selected_experiments": ["acquisition"],
                "selected_cases": ["acquisition:case_a"],
                "jobs": 8,
            },
            "status": "completed",
            "cases": [case_result("case_a", ["acquisition"], "provider_a", "cap_a", "binding_a")],
            "summary": {"experiments": {"acquisition": {"providers_with_promoted_bindings": 1}}},
            "coverage": {"distinct_case_count": 1},
            "experiment_gates": {"acquisition": {"numerator": 1, "denominator": 1, "actual": 1, "target": 0.85, "passed": True}},
            "capability_model": {"providers": {}, "capabilities": {}, "checks": []},
            "scores": {},
        }

        html = report.render_html(artifact)

        self.assertIn("Experiment 1: Acquiring Capabilities From Provider Surfaces", html)
        self.assertNotIn("Experiment 2: Verification And Failure Gating", html)
        self.assertNotIn("Experiment 3: Repeated Held-Out Reuse", html)
        self.assertNotIn("Experiment 4: Capability Structure Evidence", html)


class ExportResultTest(unittest.TestCase):
    def test_public_artifact_removes_trace_paths_and_raw_model_outputs(self) -> None:
        raw = {
            "run": {
                "id": "run_a",
                "mode": constants.MODE_LIVE_LLM,
                "status": "completed",
                "level": "full",
                "selected_experiments": ["acquisition"],
                "selected_cases": ["acquisition:case_a"],
                "jobs": 8,
                "llm": {"api": "chat_completions", "model": "model_a", "base_url_configured": True},
            },
            "status": "completed",
            "cases": [
                {
                    **case_result("case_a", ["acquisition"], "provider_a", "cap_a", "binding_a"),
                    "shard": {"path": "/Users/example/run/shard", "home": "/Users/example/run/home"},
                }
            ],
            "summary": {
                "experiments": {"acquisition": {"provider_attempted": 1}},
                "acquisition_modes": {"intent_guided": {"provider_attempted": 1, "providers_with_promoted_bindings": 1}},
            },
            "scores": {"proposal_total_tokens": 42},
            "coverage": {},
            "capability_model": {},
            "discovery_coverage": {},
            "failure_taxonomy": [],
            "experiment_gates": {"acquisition": {"metric": "yield", "numerator": 1, "denominator": 1, "actual": 1.0, "target": 0.85, "passed": True}},
        }
        raw["cases"][0]["providers"][0]["provider_path"] = "/Users/example/tool"
        raw["cases"][0]["providers"][0]["trace_id"] = "trace_123"
        raw["cases"][0]["providers"][0]["acquisition"] = {"proposal": {"attempts": [{"raw_response": "secret"}]}}

        public = export_result.build_public_artifact(raw)
        metrics = export_result.build_metrics(public)
        text = json.dumps(public)

        self.assertNotIn("/Users/", text)
        self.assertNotIn("provider_path", text)
        self.assertNotIn("trace_", text)
        self.assertNotIn("raw_response", text)
        self.assertEqual(metrics["acquisition_gate"]["numerator"], 1)
        self.assertEqual(public["cases"][0]["providers"][0]["promoted_bindings"], 1)

    def test_public_failure_messages_redact_absolute_paths(self) -> None:
        failure = {
            "stage": "baseline_llm_oneshot",
            "code": "command_failed",
            "message": "invalid choice: '/Users/example/project/fixtures/input.txt'",
        }

        public = export_result.public_failure(failure)

        self.assertNotIn("/Users/", public["message"])
        self.assertIn("<path>", public["message"])

    def test_export_run_writes_sanitized_files(self) -> None:
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            run_dir = root / "run"
            out_dir = root / "public"
            run_dir.mkdir()
            raw = {
                "run": {"id": "run_a", "mode": constants.MODE_LIVE_LLM, "status": "completed", "level": "full", "selected_experiments": ["acquisition"], "selected_cases": [], "jobs": 1, "llm": {"model": "model_a"}},
                "status": "completed",
                "cases": [case_result("case_a", ["acquisition"], "provider_a", "cap_a", "binding_a")],
                "summary": {"experiments": {"acquisition": {"provider_attempted": 1}}, "acquisition_modes": {}},
                "scores": {},
                "coverage": {},
                "capability_model": {},
                "discovery_coverage": {},
                "failure_taxonomy": [],
                "experiment_gates": {"acquisition": {"metric": "yield", "numerator": 1, "denominator": 1, "actual": 1.0, "target": 0.85, "passed": True}},
            }
            (run_dir / "summary.json").write_text(json.dumps(raw), encoding="utf-8")

            export_result.export_run(run_dir, out_dir)

            self.assertTrue((out_dir / "artifact.public.json").exists())
            self.assertTrue((out_dir / "metrics.json").exists())
            self.assertTrue((out_dir / "report.html").exists())
            export_result.assert_public_directory(out_dir)

    def test_verification_failure_metrics_are_experiment_specific(self) -> None:
        raw = {
            "run": {
                "id": "run_v",
                "mode": constants.MODE_LIVE_LLM,
                "status": "completed",
                "level": "full",
                "selected_experiments": ["verification_failure"],
                "selected_cases": ["failure_gating:drift_hash_sha256"],
                "jobs": 8,
                "llm": {"api": "chat_completions", "model": "model_v", "base_url_configured": True},
            },
            "status": "completed",
            "cases": [case_result("drift_hash_sha256", ["verification_failure"], "drift_hash", "file.checksum", "")],
            "summary": {
                "experiments": {
                    "verification_failure": {
                        "provider_attempted": 1,
                        "candidate_count": 1,
                        "probe_fail_count": 1,
                        "probe_pass_count": 0,
                        "promoted_bindings": 0,
                        "candidate_negative_evidence": 1,
                        "avg_acquisition_ms": 12,
                        "acquisition_llm_calls": 4,
                        "total_tokens": 99,
                    }
                }
            },
            "scores": {},
            "coverage": {},
            "capability_model": {},
            "discovery_coverage": {},
            "failure_taxonomy": [],
            "experiment_gates": {
                "verification_failure": {
                    "metric": "blocked_invalid_candidate_rate",
                    "numerator": 1,
                    "denominator": 1,
                    "actual": 1.0,
                    "target": 0.95,
                    "passed": True,
                    "false_promotions": 0,
                }
            },
        }
        raw["cases"][0]["providers"][0]["provider_path"] = "/Users/example/drift-hash"

        public = export_result.build_public_artifact(raw)
        metrics = export_result.build_metrics(public)
        readme = export_result.render_readme(public, metrics, {"source_run_id": "run_v"})

        self.assertEqual(metrics["experiment"], "verification_failure")
        self.assertEqual(metrics["blocked_invalid"], 1)
        self.assertEqual(metrics["false_promotions"], 0)
        self.assertNotIn("acquisition_gate", metrics)
        self.assertIn("Verification gate", readme)
        self.assertNotIn("/Users/", json.dumps(public))

    def test_capability_structure_metrics_are_experiment_specific(self) -> None:
        raw = {
            "run": {
                "id": "run_s",
                "mode": constants.MODE_LIVE_LLM,
                "status": "completed",
                "level": "full",
                "selected_experiments": ["acquisition", "capability_structure"],
                "selected_cases": ["acquisition:file_hash_sha1"],
                "jobs": 8,
                "llm": {"api": "chat_completions", "model": "model_s", "base_url_configured": True},
            },
            "status": "completed",
            "cases": [case_result("file_hash_sha1", ["acquisition", "capability_structure"], "shasum", "sha1.hash", "binding_a")],
            "summary": {
                "experiments": {
                    "acquisition": {
                        "provider_attempted": 1,
                        "candidate_count": 1,
                        "probe_pass_count": 1,
                        "probe_fail_count": 0,
                        "promoted_bindings": 1,
                        "avg_acquisition_ms": 123,
                        "acquisition_llm_calls": 4,
                        "total_tokens": 456,
                    }
                }
            },
            "scores": {},
            "coverage": {},
            "capability_model": {
                "providers": {"shasum": ["sha1.hash", "sha256.hash"]},
                "capabilities": {"sha1.hash": ["shasum", "sha1sum"]},
                "multi_capability_providers": 1,
                "multi_binding_capabilities": 1,
                "check_passed": 2,
                "check_failed": 0,
                "check_skipped": 0,
            },
            "discovery_coverage": {},
            "failure_taxonomy": [],
            "experiment_gates": {
                "acquisition": {"metric": "yield", "numerator": 1, "denominator": 1, "actual": 1.0, "target": 0.85, "passed": True},
                "capability_structure": {"metric": "structure", "numerator": 2, "denominator": 2, "actual": 1.0, "target": 0.9, "passed": True, "skipped": 0},
            },
        }

        public = export_result.build_public_artifact(raw)
        metrics = export_result.build_metrics(public)
        readme = export_result.render_readme(public, metrics, {"source_run_id": "run_s"})

        self.assertEqual(metrics["experiment"], "capability_structure")
        self.assertEqual(metrics["capability_structure_gate"]["numerator"], 2)
        self.assertEqual(metrics["acquisition_gate"]["numerator"], 1)
        self.assertEqual(metrics["multi_capability_providers"], 1)
        self.assertEqual(metrics["multi_binding_capabilities"], 1)
        self.assertIn("Capability-structure gate", readme)


class ValidateTest(unittest.TestCase):
    def test_check_baselines_restricts_to_repeated_reuse(self) -> None:
        with self.assertRaises(SystemExit):
            validate.check_baselines("case_a", ["acquisition"], ["llm_oneshot"])
        with self.assertRaises(SystemExit):
            validate.check_baselines("case_a", ["repeated_reuse"], ["provider_tool"])
        with self.assertRaises(SystemExit):
            validate.check_baselines("case_a", ["repeated_reuse"], ["removed_baseline"])

        validate.check_baselines("case_a", ["repeated_reuse"], ["llm_oneshot"])

    def test_full_acquisition_design_requires_mostly_multi_cap_cases(self) -> None:
        cases = [
            {"acquisition_mode": "full_acquisition", "paper_experiments": ["acquisition"], "expected_capabilities": [{"key": "one", "surface": "one", "min_promoted_bindings": 1}]},
            {
                "acquisition_mode": "full_acquisition",
                "paper_experiments": ["acquisition"],
                "expected_capabilities": [
                    {"key": "one", "surface": "one", "min_promoted_bindings": 1},
                    {"key": "two", "surface": "two", "min_promoted_bindings": 1},
                ],
            },
        ]

        with self.assertRaises(SystemExit):
            validate.check_full_acquisition_design(cases)

        cases[0]["expected_capabilities"].append({"key": "two", "surface": "two", "min_promoted_bindings": 1})
        validate.check_full_acquisition_design(cases)

    def test_reuse_comparison_design_requires_eight_cases_and_ten_rounds(self) -> None:
        cases = [
            {"paper_experiments": ["repeated_reuse"], "scenario_tags": ["reuse_comparison"], "reuse": {"rounds": [{"id": "a"}]}}
        ]

        with self.assertRaises(SystemExit):
            validate.check_reuse_comparison_design(cases)

        cases = [
            {
                "paper_experiments": ["repeated_reuse"],
                "scenario_tags": ["reuse_comparison"],
                "reuse": {"rounds": [{"id": f"r{round_index}"} for round_index in range(2 if case_index < 2 else 1)]},
            }
            for case_index in range(8)
        ]

        validate.check_reuse_comparison_design(cases)


class LLMOneShotTest(unittest.TestCase):
    def test_client_reads_temperature_from_env(self) -> None:
        with mock.patch.dict(
            os.environ,
            {
                "CAL_LLM_MODEL": "kimi-k2.7-code",
                "CAL_LLM_API_KEY": "test-key",
                "CAL_LLM_TEMPERATURE": "1",
            },
            clear=False,
        ):
            client = llm_client.LLMClient.from_env()

        self.assertIsNotNone(client)
        self.assertEqual(client.temperature, 1)

    def test_runner_executes_round_command_and_scores_oracle(self) -> None:
        with tempfile.TemporaryDirectory() as temp:
            repo = Path(temp) / "repo"
            bench = Path(temp) / "bench"
            home = Path(temp) / "home"
            repo.mkdir()
            write_oneshot_fixture(bench)
            source = bench / "fixtures" / "copy_case" / "reuse" / "b.txt"
            source.write_text("hello\n", encoding="utf-8")
            runner = llm_oneshot.OneShotRunner(repo, bench, home, constants.MODE_LIVE_LLM, oracle.OracleRunner(repo, bench), CopyClient())

            result = runner.run_case(oneshot_case())[0]

            self.assertEqual(result["status"], constants.STATUS_PASSED)
            self.assertTrue(result["oracle"]["passed"])
            self.assertEqual(result["llm"]["usage"]["total_tokens"], 3)

    def test_runner_uses_baseline_provider_when_configured(self) -> None:
        case = oneshot_case()
        case["baseline_provider"] = "cp"
        case["providers"].append({"id": "cat", "command": "cat", "platforms": []})

        providers = llm_oneshot.baseline_providers(case)

        self.assertEqual([provider["id"] for provider in providers], ["cp"])


class RunContractTest(unittest.TestCase):
    def test_live_mode_limits_jobs_to_llm_jobs(self) -> None:
        args = type("Args", (), {"jobs": 8, "llm_jobs": 3, "mode": constants.MODE_LIVE_LLM})()

        self.assertEqual(run.effective_jobs(args), 3)

    def test_repeated_reuse_defaults_to_seeded_records(self) -> None:
        case = {"paper_experiments": ["repeated_reuse"]}

        self.assertFalse(run.should_run_acquisition(case, seed.REUSE_SEED_REPLAY))
        self.assertTrue(run.should_seed_records(case, seed.REUSE_SEED_REPLAY))
        self.assertTrue(run.should_run_acquisition(case, seed.REUSE_SEED_SELF))
        self.assertFalse(run.should_seed_records(case, seed.REUSE_SEED_SELF))

    def test_capability_structure_can_run_from_seeded_records(self) -> None:
        case = {"paper_experiments": ["capability_structure"]}

        self.assertFalse(run.should_run_acquisition(case, seed.REUSE_SEED_REPLAY))
        self.assertTrue(run.should_seed_records(case, seed.REUSE_SEED_REPLAY))

    def test_full_acquisition_omits_task_hint_by_default(self) -> None:
        self.assertEqual(acquisition.acquisition_hint({"acquisition_mode": "full_acquisition", "intent": "single task"}), "")
        self.assertEqual(
            acquisition.acquisition_hint({"acquisition_mode": "full_acquisition", "acquisition": {"full_hint": "discover provider surfaces"}}),
            "discover provider surfaces",
        )
        self.assertEqual(acquisition.acquisition_hint({"acquisition_mode": "intent_guided", "intent": "single task"}), "single task")

    def test_restrict_cases_to_selected_experiments(self) -> None:
        cases = [{"id": "case_a", "paper_experiments": ["acquisition", "capability_structure"]}]

        restricted = run.restrict_cases_to_selected_experiments(cases, "capability_structure")

        self.assertEqual(restricted[0]["paper_experiments"], ["capability_structure"])

    def test_reuse_effectiveness_profile_limits_rounds_and_disables_baselines(self) -> None:
        cases = [
            {
                "id": "case_a",
                "paper_experiments": ["repeated_reuse"],
                "reuse": {"rounds": [{"id": "a"}, {"id": "b"}]},
                "baselines": ["llm_oneshot"],
            }
        ]

        profiled = run.apply_reuse_profile(cases, constants.REUSE_PROFILE_EFFECTIVENESS)

        self.assertEqual([round_value["id"] for round_value in profiled[0]["reuse"]["rounds"]], ["a"])
        self.assertEqual(profiled[0]["baselines"], [])
        self.assertEqual(profiled[0]["reuse_profile"], constants.REUSE_PROFILE_EFFECTIVENESS)

    def test_reuse_comparison_profile_selects_tagged_cases(self) -> None:
        cases = [
            {"id": "case_a", "paper_experiments": ["repeated_reuse"], "scenario_tags": ["reuse_comparison"], "baselines": ["llm_oneshot"]},
            {"id": "case_b", "paper_experiments": ["repeated_reuse"], "scenario_tags": [], "baselines": ["llm_oneshot"]},
        ]

        profiled = run.apply_reuse_profile(cases, constants.REUSE_PROFILE_COMPARISON)

        self.assertEqual([case["id"] for case in profiled], ["case_a"])
        self.assertEqual(profiled[0]["baselines"], ["llm_oneshot"])

    def test_intent_reuse_does_not_pin_provider(self) -> None:
        workspace = RecordingWorkspace()
        runner = reuse.ReuseRunner(Path("/bench"), Path("/home"), workspace, oracle=None)
        case = {
            "id": "case_a",
            "intent": "copy file",
            "reuse": {"rounds": [{"id": "round_a", "inputs": {"source": "input.txt"}}]},
        }
        result = {"providers": [{"provider_id": "provider_a", "candidates": [{"promotion": {"binding_id": "binding_a"}}]}], "use": []}

        runner.run_intent_uses(case, result)

        self.assertNotIn("--provider-id", workspace.commands[0])
        self.assertIn("--strategy", workspace.commands[0])
        strategy_index = workspace.commands[0].index("--strategy")
        self.assertEqual(workspace.commands[0][strategy_index + 1], "best")

    def test_seed_capabilities_builds_promoted_binding(self) -> None:
        proposal = {
            "candidates": [
                {
                    "capability_id": "text.copy",
                    "description": "Copy text.",
                    "execution": {"kind": "cli", "spec": {"args": ["cp", "{{source}}", "{{target}}"]}},
                }
            ],
            "probe_plans": [
                {
                    "candidate_index": 0,
                    "verify": {
                        "level": "L3",
                        "method": "execute",
                        "checks": [{"subject": {"type": "file", "input": "target"}, "predicate": "exists"}],
                    },
                }
            ],
        }

        capabilities = seed.seed_capabilities(proposal, "provider_a")

        self.assertEqual(capabilities[0]["id"], "text.copy")
        binding = capabilities[0]["bindings"][0]
        self.assertEqual(binding["provider_id"], "provider_a")
        self.assertEqual(binding["state"], "promoted")
        self.assertEqual(binding["verify"]["level"], "L3")
        self.assertTrue(binding["evidence"])

    def test_seed_capability_write_merges_multiple_bindings(self) -> None:
        with tempfile.TemporaryDirectory() as temp:
            home = Path(temp)
            first = [{"id": "text.search", "description": "Search text.", "bindings": [{"id": "binding_grep", "provider_id": "provider_grep"}]}]
            second = [{"id": "text.search", "description": "Search text.", "bindings": [{"id": "binding_rg", "provider_id": "provider_rg"}]}]

            seed.write_seeded_capabilities(home, first)
            seed.write_seeded_capabilities(home, second)

            stored = json.loads((home / "capabilities" / "text.search.json").read_text(encoding="utf-8"))
            self.assertEqual([binding["id"] for binding in stored["bindings"]], ["binding_grep", "binding_rg"])

    def test_benchmark_env_prepends_eval_tools(self) -> None:
        with tempfile.TemporaryDirectory() as temp:
            bench = Path(temp) / "bench"
            (bench / "tools" / "bin").mkdir(parents=True)

            env = run.benchmark_env({"PATH": "/usr/bin"}, bench)

            self.assertTrue(env["PATH"].startswith(str(bench / "tools" / "bin")))


def write_catalog_fixture(bench: Path) -> None:
    (bench / "scenarios").mkdir(parents=True)
    (bench / "oracles").mkdir()
    (bench / "oracles" / "a.py").write_text("print('ok')\n", encoding="utf-8")
    (bench / "providers.json").write_text(json.dumps({"providers": [{"id": "tool_a", "command": "tool-a"}, {"id": "tool_b", "command": "tool-b"}]}), encoding="utf-8")
    write_jsonl(
        bench / "scenarios" / "acquisition.jsonl",
        [
            scenario_case("case_a", "tool_a", ["acquisition"], "known_cli", ["known_cli"]),
        ],
    )
    write_jsonl(
        bench / "scenarios" / "repeated_reuse.jsonl",
        [
            scenario_case("case_b", "tool_b", ["repeated_reuse"], "third_party_cli", ["third_party_cli", "repeated_reuse"]),
        ],
    )
    write_jsonl(bench / "scenarios" / "failure_gating.jsonl", [])
    write_jsonl(bench / "scenarios" / "capability_structure.jsonl", [])


def scenario_case(case_id: str, provider_id: str, experiments: list[str], provider_class: str, tags: list[str]) -> dict:
    return {
        "id": case_id,
        "level": "focus",
        "domain": "text",
        "provider_class": provider_class,
        "provider_candidates": [provider_id],
        "paper_experiments": experiments,
        "scenario_tags": tags,
        "acquisition_mode": "intent_guided",
        "failure_type": "",
        "intent": "do it",
        "acquisition": {"fixtures": [{"id": "a", "inputs": {"source": "fixtures/a.txt", "target": "{work}/out.txt"}}]},
        "reuse": {"rounds": [{"id": "round_1", "inputs": {"source": "fixtures/a.txt", "target": "{work}/out.txt"}}]},
        "oracle": {"path": "oracles/a.py"},
    }


def case_result(
    case_id: str,
    experiments: list[str],
    provider_id: str,
    capability_id: str,
    binding_id: str,
    use_passed: bool | None = None,
    checks: dict | None = None,
    acquisition_mode: str = "intent_guided",
    expected_capabilities: list[dict] | None = None,
    candidates: list[dict] | None = None,
) -> dict:
    candidate_rows = candidates or [candidate_result(capability_id, binding_id, [capability_id, "{{source}}", "{{target}}"])]
    case = {
        "id": case_id,
        "case_key": f"acquisition:{case_id}",
        "paper_experiments": experiments,
        "provider_class": "known_cli",
        "acquisition_mode": acquisition_mode,
        "domain": "text",
        "expected_capabilities": expected_capabilities or [],
        "providers": [
            {
                "id": provider_id,
                "provider_path": f"/bin/{provider_id}",
                "acquisition_duration_ms": 10,
                "candidates": candidate_rows,
            }
        ],
        "use": [],
        "baselines": {},
        "capability_layer_checks": checks or {},
        "reuse": {"rounds": [{"id": "round_1"}]},
    }
    if use_passed is not None:
        case["use"] = [{"fixture_id": "round_1", "duration_ms": 7, "selection": {"source": "llm", "binding_id": binding_id}, "oracle": {"passed": use_passed}, "status": "passed" if use_passed else "failed"}]
    return case


def candidate_result(capability_id: str, binding_id: str, args: list[str]) -> dict:
    return {
        "capability_id": capability_id,
        "execution": {"kind": "cli", "spec": {"args": args}},
        "probe": {"passed": bool(binding_id), "status": "passed" if binding_id else "failed"},
        "promotion": {"capability_id": capability_id, "binding_id": binding_id},
    }


class RecordingWorkspace:
    def __init__(self) -> None:
        self.commands: list[list[str]] = []

    def run_calctl(self, args: list[str]):
        self.commands.append(args)
        return type("Completed", (), {"returncode": 1, "stdout": '{"error":{"code":"no_match","message":"no match"}}', "stderr": ""})()


def write_oneshot_fixture(bench: Path) -> None:
    (bench / "baselines" / "llm_one_shot").mkdir(parents=True)
    (bench / "baselines" / "llm_one_shot" / "prompt.md").write_text("Return JSON", encoding="utf-8")
    (bench / "fixtures" / "copy_case" / "reuse").mkdir(parents=True)
    (bench / "oracles").mkdir(parents=True)
    (bench / "oracles" / "copy.py").write_text(
        "import json,pathlib,sys\nreq=json.load(sys.stdin)\ninputs=req['inputs']\nsrc=pathlib.Path(inputs['source']); dst=pathlib.Path(inputs['target'])\nprint(json.dumps({'passed': src.read_text()==dst.read_text()}))\n",
        encoding="utf-8",
    )


def oneshot_case() -> dict:
    return {
        "id": "copy_case",
        "intent": "copy file",
        "description": "copy",
        "domain": "text",
        "providers": [{"id": "cp", "command": "cp", "platforms": [], "version_command": ["cp", "--version"]}],
        "reuse": {"rounds": [{"id": "round_1", "inputs": {"source": "fixtures/copy_case/reuse/b.txt", "target": "{work}/out.txt"}}]},
        "oracle": {"path": "oracles/copy.py"},
    }


class CopyClient:
    def chat(self, _system: str, user: str) -> dict:
        payload = json.loads(user)
        inputs = payload["fixture"]["inputs"]
        return {
            "content": json.dumps({"command": ["cp", inputs["source"], inputs["target"]], "writes_target": True}),
            "duration_ms": 5,
            "api": "test",
            "model": "test",
            "usage": {"prompt_tokens": 1, "completion_tokens": 2, "total_tokens": 3},
        }


def write_jsonl(path: Path, rows: list[dict]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text("".join(json.dumps(row) + "\n" for row in rows), encoding="utf-8")


if __name__ == "__main__":
    unittest.main()
