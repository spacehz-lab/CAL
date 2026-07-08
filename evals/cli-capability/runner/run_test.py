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
summary = load_module("summary")
report = load_module("report")
run = load_module("run")


class SuiteCatalogTest(unittest.TestCase):
    def test_select_reads_physical_suites(self) -> None:
        with tempfile.TemporaryDirectory() as temp:
            bench = Path(temp)
            write_catalog_fixture(bench)

            selected = catalog.SuiteCatalog(bench).select("acquisition,reuse", "focus")

            self.assertEqual([case["suite"] for case in selected], ["acquisition", "reuse"])
            self.assertEqual(selected[0]["providers"][0]["id"], "tool_a")

    def test_select_rejects_unknown_provider(self) -> None:
        with tempfile.TemporaryDirectory() as temp:
            bench = Path(temp)
            write_suite_files(bench, [{"id": "case_a", "suite": "acquisition", "intent": "do", "provider_candidates": ["missing"], "oracle": {"path": "oracles/a.py"}}])
            (bench / "providers.json").write_text(json.dumps({"providers": []}), encoding="utf-8")

            with self.assertRaises(SystemExit):
                catalog.SuiteCatalog(bench)

    def test_select_rejects_unknown_suite(self) -> None:
        with tempfile.TemporaryDirectory() as temp:
            bench = Path(temp)
            write_catalog_fixture(bench)

            with self.assertRaises(SystemExit):
                catalog.SuiteCatalog(bench).select("unknown", "focus")


class ReuseHelpersTest(unittest.TestCase):
    def test_materialize_inputs_expands_fixture_and_work_paths(self) -> None:
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
                {"id": "text_b", "inputs": {"source": "fixtures/case_a/reuse/b.txt", "target": "{work}/out.txt"}},
            )

            self.assertEqual(inputs["source"], str(fixture.resolve()))
            self.assertEqual(inputs["target"], str(home / "benchmark" / "case_a" / "reuse" / "text_b" / "out.txt"))
            self.assertTrue(Path(inputs["target"]).parent.exists())

    def test_required_runtime_inputs_include_templates_and_stdout_path(self) -> None:
        candidate = {"execution": {"spec": {"args": ["{{source}}", "{{ format }}"], "stdout_path_input": "target"}}}

        self.assertEqual(reuse.required_runtime_inputs(candidate), ["format", "source", "target"])

    def test_provider_oracles_passed_ignores_skipped_reuse(self) -> None:
        provider = {"candidates": [{"reuse": [{"status": constants.STATUS_SKIPPED}, {"oracle": {"passed": True}}]}]}

        self.assertTrue(reuse.provider_oracles_passed(provider))


class SummaryTest(unittest.TestCase):
    def test_summarize_groups_by_suite_and_baseline(self) -> None:
        suites = {
            "acquisition": {
                "cases": [
                    {
                        "providers": [
                            {
                                "provider_path": "/bin/tool",
                                "acquisition_duration_ms": 10,
                                "candidates": [
                                    {
                                        "probe": {"passed": True},
                                        "promotion": {"binding_id": "binding_a"},
                                        "reuse": [{"duration_ms": 5, "oracle": {"passed": True}}],
                                    }
                                ],
                            }
                        ],
                        "use": [{"duration_ms": 7, "selection": {"source": "llm"}, "oracle": {"passed": True}}],
                        "baselines": {"direct_cli": [{"duration_ms": 3, "oracle": {"passed": True}}]},
                    }
                ]
            },
            "capability_model": {"cases": []},
            "reuse": {"cases": []},
        }

        result = summary.summarize(suites, constants.MODE_REPLAY)

        self.assertEqual(result["suites"]["acquisition"]["held_out_reuses"], 1)
        self.assertEqual(result["suites"]["acquisition"]["use_oracle_pass_count"], 1)
        self.assertEqual(result["total"]["promoted_bindings"], 1)
        self.assertEqual(result["baselines"]["direct_cli"]["passed"], 1)
        self.assertEqual(result["baselines"]["direct_cli"]["success_rate"], 1)

    def test_capability_model_counts_multi_provider_and_multi_capability(self) -> None:
        suites = {
            "acquisition": {
                "cases": [
                    case_result("case_a", "provider_a", "cap_a", "binding_a"),
                    case_result("case_b", "provider_a", "cap_b", "binding_b"),
                    case_result("case_c", "provider_b", "cap_a", "binding_c"),
                ]
            }
        }

        model = summary.capability_model(suites)

        self.assertEqual(model["multi_capability_providers"], 1)
        self.assertEqual(model["multi_binding_capabilities"], 1)


class ReportTest(unittest.TestCase):
    def test_build_flow_artifact_groups_by_suite(self) -> None:
        artifact = {
            "run": {"id": "run_1", "mode": constants.MODE_REPLAY},
            "suites": {"acquisition": {"cases": [case_result("case_a", "provider_a", "cap_a", "binding_a")]}, "capability_model": {"cases": []}, "reuse": {"cases": []}},
            "summary": {},
            "scores": {},
        }

        flow = report.build_flow_artifact(artifact)

        self.assertEqual(flow["schema_version"], constants.FLOW_SCHEMA_VERSION)
        self.assertEqual(flow["suites"]["acquisition"]["cases"][0]["id"], "case_a")

    def test_render_html_shows_suite_sections(self) -> None:
        html = report.render_html(
            {
                "run": {"id": "run_1", "mode": constants.MODE_REPLAY, "selected_suites": ["acquisition"], "selected_cases": ["acquisition:case_a"]},
                "status": "completed",
                "suites": {"acquisition": {"cases": []}, "capability_model": {"cases": []}, "reuse": {"cases": []}},
                "summary": {"total": {"held_out_uses": 1, "use_oracle_pass_count": 1}},
                "scores": {"closed_loop_success_rate": 1.0},
            }
        )

        self.assertIn("<h2>Acquisition Suite</h2>", html)
        self.assertIn("<h2>Capability Model Suite</h2>", html)
        self.assertIn("<h2>Reuse Suite</h2>", html)
        self.assertIn("<h2>Baseline / Cost Amortization</h2>", html)
        self.assertIn("100.0%", html)


class RunTest(unittest.TestCase):
    def test_new_artifact_records_non_secret_live_llm_settings(self) -> None:
        args = argparse_like(mode=constants.MODE_LIVE_LLM, level="focus")

        with mock.patch.dict(os.environ, {"CAL_LLM_API": "chat_completions", "CAL_LLM_BASE_URL": "https://api.example.test/v1"}, clear=False):
            artifact = run.new_artifact("run_1", args, [], ["acquisition"], "test-model")

        self.assertEqual(artifact["run"]["llm"], {"api": "chat_completions", "model": "test-model", "base_url_configured": True})

    def test_validate_result_uses_total_summary(self) -> None:
        artifact = {"summary": {"total": {"oracle_pass_count": 1, "use_oracle_pass_count": 1, "oracle_fail_count": 0, "use_oracle_fail_count": 0, "failed": 0}}}

        run.validate_result(constants.MODE_REPLAY, artifact)

        with self.assertRaises(SystemExit):
            run.validate_result(constants.MODE_LIVE_LLM, {"summary": {"total": {"use_oracle_pass_count": 0}}})


def write_catalog_fixture(bench: Path) -> None:
    cases = [
        {"id": "case_a", "suite": "acquisition", "level": "focus", "intent": "do", "provider_candidates": ["tool_a"], "oracle": {"path": "oracles/a.py"}},
        {"id": "case_a", "suite": "reuse", "level": "focus", "intent": "do", "provider_candidates": ["tool_a"], "oracle": {"path": "oracles/a.py"}},
    ]
    write_suite_files(bench, cases)
    (bench / "providers.json").write_text(json.dumps({"providers": [{"id": "tool_a", "command": "tool"}]}), encoding="utf-8")


def write_suite_files(bench: Path, cases: list[dict]) -> None:
    suites = {"acquisition": [], "capability_model": [], "reuse": []}
    for case in cases:
        suites[case["suite"]].append(case)
    (bench / "suites").mkdir(parents=True)
    for suite, rows in suites.items():
        (bench / "suites" / f"{suite}.jsonl").write_text("".join(json.dumps(row) + "\n" for row in rows), encoding="utf-8")


def case_result(case_id: str, provider_id: str, capability_id: str, binding_id: str) -> dict:
    return {
        "id": case_id,
        "providers": [
            {
                "id": provider_id,
                "candidates": [
                    {
                        "capability_id": capability_id,
                        "promotion": {"binding_id": binding_id},
                    }
                ],
            }
        ],
    }


def argparse_like(**values):
    return type("Args", (), values)()


if __name__ == "__main__":
    unittest.main()
