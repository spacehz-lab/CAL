# Contributing

CAL is a release preview for a local Capability Acquisition Layer. Contributions
should keep the current scope small: local-first, CLI-first, evidence-gated
promotion, and intent-level reuse through `calctl use`.

## Development Setup

Build local binaries:

```sh
make build
```

Install commands into Go's install directory:

```sh
make install
```

If `calctl` is not found after install, add Go's install directory to `PATH`:

```sh
export PATH="$(go env GOPATH)/bin:$PATH"
```

## Tests

Run the default test suite:

```sh
make test
```

Run default functional end-to-end checks:

```sh
make e2e
```

Environment-dependent checks are opt-in:

```sh
CAL_LOCAL_CLI_E2E=1 go test ./tests/e2e/local_cli -count=1 -v
```

Keep live API keys in the environment only. Do not write them into repository
files, traces, logs, or committed artifacts.

## Evaluation Runners

Build before running eval assets:

```sh
make build
```

Replay mode runs without live LLM credentials:

```sh
python3 evals/cli-capability/runner/run.py \
  --mode replay \
  --level focus \
  --calctl build/bin/calctl \
  --cald build/bin/cald
```

Generated outputs belong under `evals/out/` and should not be committed.

## Change Guidelines

- Keep behavior grounded in the current architecture and docs.
- Prefer small, focused changes over broad rewrites.
- Keep LLM output behind local validation and deterministic execution.
- Do not introduce hard-coded provider-specific behavior into production paths;
  use fixtures, tests, or baseline runners for deterministic examples.
- Update docs and tests with behavior changes.
- Keep generated files, local state, logs, and secrets out of git.

## Pull Request Checklist

Before opening a pull request, run:

```sh
make test
make e2e
```

If your change affects local CLI integrations, also run the relevant opt-in
tests or explain why they were not run.
