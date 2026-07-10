# CAL

CAL is a local Capability Acquisition Layer. It observes provider surfaces,
proposes executable bindings, probes them locally, promotes verified bindings
into reusable capabilities, and routes later requests through those capabilities.

Status: release preview / local-only / CLI-first.

## Highlights

- Registers explicit local providers; no background provider scanning.
- Runs acquisition as a staged loop: entry, observe, proposal, probe, promote,
  and tracelog.
- Uses an OpenAI-compatible LLM for live proposal and intent selection, with
  replay and rules modes available for deterministic paths.
- Promotes a binding only after local probe evidence is recorded.
- Reuses promoted bindings through `calctl use` or explicit `calctl runs`.
- Keeps runtime state local under `CAL_HOME`; API keys are read from
  environment variables.
- Supports streaming progress for long-running acquisition, run, and use calls.

## Requirements

- Go 1.23 or newer.
- A local CLI provider with useful `--help` or `man` output.
- An OpenAI-compatible LLM endpoint for live acquisition and LLM-assisted use.
- `jq` for the copy-paste JSON examples.

CAL is currently best tested on macOS with CLI providers.

## Architecture

```text
provider -> entry -> observe -> proposal -> probe -> promote -> tracelog
                                      |
                                      v
                              reusable capability
                                      |
                                      v
                              run / use / eval
```

LLM output is a proposal, not proof. `probe` executes controlled candidate
bindings and `check` evaluates deterministic verification rules before
promotion. Durable records stay under `CAL_HOME`: providers, capabilities,
traces, runs, config, and daemon endpoint metadata.

## Quickstart

Build or install the commands:

```sh
make install
export PATH="$(go env GOPATH)/bin:$PATH"
```

Use an isolated local state directory and start the daemon:

```sh
export CAL_HOME="$PWD/.cal-demo"
rm -rf "$CAL_HOME"
calctl daemon start --json
calctl daemon status --json
```

Configure a live LLM:

```sh
export CAL_LLM_API=chat_completions
export CAL_LLM_BASE_URL="<openai-compatible-base-url>"
export CAL_LLM_MODEL="<model>"
export CAL_LLM_API_KEY="<api-key>"
```

Register a provider and run acquisition:

```sh
PROVIDER_PATH="$(command -v plutil)"
PROVIDER_ID="$(calctl providers add --provider-path "$PROVIDER_PATH" --json | jq -r .id)"
calctl acquisition run \
  --provider-id "$PROVIDER_ID" \
  --hint "convert plist to json" \
  --stream --json
```

Route an intent through promoted capabilities:

```sh
calctl use "convert /tmp/input.plist to json" --stream --json
```

Inspect local records and stop the daemon:

```sh
calctl capabilities list --json
calctl eval --json
calctl daemon stop --json
```

See [docs/quickstart.md](docs/quickstart.md) for a fuller walkthrough and
troubleshooting notes.

Release V1 architecture notes live under
[docs/cal-release-v1](docs/cal-release-v1); older design drafts are kept only
in the ignored local `backup/` tree.

## Commands

| Command | Purpose |
| --- | --- |
| `cald serve` | Run the local CAL daemon in the foreground. |
| `calctl daemon start/status/stop` | Start, inspect, or stop the local daemon. |
| `calctl providers add/list` | Register and inspect provider paths. |
| `calctl acquisition run` | Acquire verified capability bindings. |
| `calctl capabilities list` | List reusable promoted capabilities. |
| `calctl runs create` | Execute a known promoted capability. |
| `calctl use` | Route an intent through promoted capabilities. |
| `calctl eval` | Summarize acquisition and reuse evidence. |

## Development

Build local binaries without installing them:

```sh
make build
build/bin/calctl --help
build/bin/cald --help
```

Run tests:

```sh
make test
make e2e
```

Generated binaries under `build/bin/`, eval outputs, and the local `backup/`
tree are ignored by git.

## Evaluation

Executable evaluation assets live under `evals/cli-capability/`. Generated
local runs are written to ignored `evals/out/cli-capability/`; selected,
sanitized paper-facing summaries are committed under
`evals/results/cli-capability/`.

The current CLI capability benchmark is organized by paper experiment:

| Experiment | Purpose |
| --- | --- |
| `acquisition` | Acquire verified bindings from known, third-party, and uncommon CLI providers. |
| `verification_failure` | Check that invalid or drifted candidates are blocked before promotion. |
| `repeated_reuse` | Reuse verified capability records through `calctl use` on held-out tasks. |
| `capability_structure` | Measure provider-to-capability and capability-to-binding structure evidence. |

Current committed Kimi `kimi-k2.7-code` live LLM result bundles:

| Result | Headline |
| --- | --- |
| Acquisition | `21 / 22` provider acquisitions promoted verified bindings. |
| Verification failure | `5 / 5` invalid or drifted candidates were blocked. |
| Reuse effectiveness | CAL reuse passed `17 / 17` held-out uses. |
| Reuse comparison | CAL reuse passed `10 / 10`; LLM one-shot passed `7 / 10`. |
| Capability structure | Structure checks passed `10 / 10`; supporting acquisition passed `14 / 14`. |

The sanitized artifacts and per-run metrics are in
`evals/results/cli-capability/`.

Run a deterministic replay check:

```sh
make build
python3 evals/cli-capability/runner/run.py \
  --mode replay \
  --experiment acquisition,verification_failure,repeated_reuse,capability_structure \
  --level focus \
  --jobs 8 \
  --calctl build/bin/calctl \
  --cald build/bin/cald
```

See [evals/README.md](evals/README.md) for live LLM commands, reuse profiles,
and result-curation instructions.
