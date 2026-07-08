# CAL Tests

`tests/` contains cross-package, black-box tests that exercise CAL through its
command-line entry points or environment-dependent integrations.

Unit tests stay next to their target implementation files under `internal/`,
`cmd/`, and `scripts/`.

## Layout

```text
tests/
  e2e/
    functional/  # deterministic closed-loop CLI tests, expected to run by default
    local_cli/   # local real-CLI end-to-end checks, environment-dependent
    live_llm/    # fake local CLI + live LLM acquisition checks, opt-in
    cli_canary/  # real local CLI + live LLM acquisition canaries, opt-in
```

Old pre-v1 suites are archived under `backup/tests/e2e/` with the old
implementation. Current v1 suites live under `tests/e2e/`.

## Commands

Default functional end-to-end checks:

```sh
go test ./tests/e2e/functional -count=1
```

Local real-CLI checks:

```sh
CAL_LOCAL_CLI_E2E=1 go test ./tests/e2e/local_cli -count=1 -v
```

Live LLM checks should be opt-in and may use runtime environment overrides:

```sh
CAL_LIVE_LLM_E2E=1 \
CAL_LLM_API=chat_completions \
CAL_LLM_MODEL=<model> \
CAL_LLM_API_KEY=<api-key> \
go test ./tests/e2e/live_llm -count=1 -v
```

`CAL_LLM_API_KEY` must stay in the environment. Do not write it into repository
files, traces, logs, or committed artifacts.

Real local CLI canaries use real system tools plus live LLM acquisition. They
are intentionally separate from `live_llm` because they depend on host CLI
availability and platform-specific help output:

```sh
CAL_CLI_CANARY_E2E=1 \
CAL_LLM_API=chat_completions \
CAL_LLM_MODEL=<model> \
CAL_LLM_API_KEY=<api-key> \
go test ./tests/e2e/cli_canary -count=1 -v -timeout=30m
```
