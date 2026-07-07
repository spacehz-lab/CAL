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
```

Old pre-v1 live LLM and CLI canary suites are archived under
`backup/tests/e2e/` with the old implementation. They are not part of the main
module test set after the `internal/` v1 switch.

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
