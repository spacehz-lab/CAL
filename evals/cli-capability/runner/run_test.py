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
oracle = load_module("oracle")
baseline = load_module("baseline")
llm_oneshot = load_module("llm_oneshot")
llm_client = load_module("llm_client")
summary = load_module("summary")
report = load_module("report")
run = load_module("run")
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

    def test_summarize_baselines_counts_repeated_reuse_only(self) -> None:
        cases = [
            {"paper_experiments": ["acquisition"], "baselines": {"direct_cli": [{"duration_ms": 3, "oracle": {"passed": True}}]}},
            {"paper_experiments": ["repeated_reuse"], "baselines": {"direct_cli": [{"duration_ms": 11, "oracle": {"passed": True}}]}},
        ]

        result = summary.summarize(cases, constants.MODE_REPLAY)

        self.assertEqual(result["baselines"]["direct_cli"]["attempted"], 1)
        self.assertEqual(result["baselines"]["direct_cli"]["duration_ms"], 11)


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
        self.assertIn("Experiment 4: Repeated Held-Out Reuse", html)
        self.assertIn("Selected experiments and cases", html)
        self.assertIn("Repeated Reuse Method Comparison", html)


class ValidateTest(unittest.TestCase):
    def test_check_baselines_restricts_to_repeated_reuse(self) -> None:
        with self.assertRaises(SystemExit):
            validate.check_baselines("case_a", ["acquisition"], ["direct_cli"])
        with self.assertRaises(SystemExit):
            validate.check_baselines("case_a", ["repeated_reuse"], ["provider_tool"])

        validate.check_baselines("case_a", ["repeated_reuse"], ["direct_cli"])
        validate.check_baselines("case_a", ["repeated_reuse"], ["llm_oneshot"])


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


class RunContractTest(unittest.TestCase):
    def test_live_mode_limits_jobs_to_llm_jobs(self) -> None:
        args = type("Args", (), {"jobs": 8, "llm_jobs": 3, "mode": constants.MODE_LIVE_LLM})()

        self.assertEqual(run.effective_jobs(args), 3)

    def test_should_run_acquisition_for_repeated_reuse(self) -> None:
        self.assertTrue(run.should_run_acquisition({"paper_experiments": ["repeated_reuse"]}))

    def test_benchmark_env_prepends_eval_tools(self) -> None:
        with tempfile.TemporaryDirectory() as temp:
            bench = Path(temp) / "bench"
            (bench / "tools" / "bin").mkdir(parents=True)

            env = run.benchmark_env({"PATH": "/usr/bin"}, bench)

            self.assertTrue(env["PATH"].startswith(str(bench / "tools" / "bin")))


class BaselineEnvTest(unittest.TestCase):
    def test_unavailable_command_uses_runner_env_path(self) -> None:
        with tempfile.TemporaryDirectory() as temp:
            tool = Path(temp) / "fake-tool"
            tool.write_text("#!/bin/sh\nexit 0\n", encoding="utf-8")
            tool.chmod(0o755)

            err = baseline.unavailable_command({}, ["fake-tool"], {"PATH": temp})

            self.assertIsNone(err)


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


def case_result(case_id: str, experiments: list[str], provider_id: str, capability_id: str, binding_id: str, use_passed: bool | None = None, checks: dict | None = None) -> dict:
    case = {
        "id": case_id,
        "case_key": f"acquisition:{case_id}",
        "paper_experiments": experiments,
        "provider_class": "known_cli",
        "acquisition_mode": "intent_guided",
        "domain": "text",
        "providers": [
            {
                "id": provider_id,
                "provider_path": f"/bin/{provider_id}",
                "acquisition_duration_ms": 10,
                "candidates": [
                    {
                        "capability_id": capability_id,
                        "probe": {"passed": bool(binding_id), "status": "passed" if binding_id else "failed"},
                        "promotion": {"capability_id": capability_id, "binding_id": binding_id},
                    }
                ],
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
