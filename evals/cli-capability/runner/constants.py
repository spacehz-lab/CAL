from __future__ import annotations


MODE_REPLAY = "replay"
MODE_LIVE_LLM = "live_llm"

SUITE_ACQUISITION = "acquisition"
SUITE_CAPABILITY_MODEL = "capability_model"
SUITE_REUSE = "reuse"
SUITES = [SUITE_ACQUISITION, SUITE_CAPABILITY_MODEL, SUITE_REUSE]

BASELINE_DIRECT_CLI = "direct_cli"
BASELINE_LLM_ONESHOT = "llm_oneshot"
BASELINE_PROVIDER_TOOL = "provider_tool"

STATUS_PASSED = "passed"
STATUS_FAILED = "failed"
STATUS_SKIPPED = "skipped"
STATUS_NOT_RUN = "not_run"

FLOW_SCHEMA_VERSION = "cli-capability-flow-v2"

STEP_PROVIDER_RESOLVE = "provider.resolve"
STEP_PROVIDER_REGISTER = "provider.register"
STEP_ACQUISITION_RUN = "acquisition.run"
STEP_STAGE_OBSERVE = "acquisition.observe"
STEP_STAGE_SURFACE = "proposal.surface"
STEP_STAGE_CAPABILITY = "proposal.capability"
STEP_STAGE_BINDING = "proposal.binding"
STEP_STAGE_EVIDENCE = "proposal.evidence"
STEP_STAGE_PROBE = "acquisition.probe"
STEP_STAGE_PROMOTE = "acquisition.promote"
STEP_DIRECT_REUSE_RUN = "direct_reuse.run"
STEP_DIRECT_REUSE_ORACLE = "direct_reuse.oracle"
STEP_INTENT_USE_SELECT = "intent_use.select"
STEP_INTENT_USE_RUN = "intent_use.run"
STEP_INTENT_USE_ORACLE = "intent_use.oracle"
STEP_BASELINE_DIRECT_CLI = "baseline.direct_cli"
STEP_BASELINE_ORACLE = "baseline.oracle"

FLOW_MATRIX_STEPS = [
    STEP_PROVIDER_RESOLVE,
    STEP_PROVIDER_REGISTER,
    STEP_ACQUISITION_RUN,
    STEP_STAGE_OBSERVE,
    STEP_STAGE_SURFACE,
    STEP_STAGE_CAPABILITY,
    STEP_STAGE_BINDING,
    STEP_STAGE_EVIDENCE,
    STEP_STAGE_PROBE,
    STEP_STAGE_PROMOTE,
    STEP_DIRECT_REUSE_ORACLE,
    STEP_INTENT_USE_ORACLE,
]
