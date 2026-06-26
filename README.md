# CAL

CAL is a local Capability Acquisition Layer. It observes provider action
surfaces, promotes verified provider-specific bindings into reusable
capabilities, and lets later requests route through `calctl use`.

Status: release preview / local-only / CLI-first.

## Highlights

- Observes real local CLI providers through their local action surfaces.
- Uses an OpenAI-compatible LLM to propose capability bindings and verifier
  harnesses.
- Promotes a binding only after CAL runs the generated probe and verifier
  locally.
- Reuses promoted bindings later through `calctl use`, so callers can start
  from intent instead of provider-specific flags.
- Keeps runtime state local and reads API keys from environment variables.

## Requirements

- Go 1.23 or newer.
- `python3`, used by generated verifier harnesses.
- A local CLI with useful `--help` or `man` output.
- An OpenAI-compatible LLM endpoint for live acquisition and intent routing.

CAL is currently best tested on macOS with CLI providers.

## How It Works

```text
provider -> observe -> LLM proposal -> generated verifier -> local probe -> promoted binding -> calctl use
```

LLM output is a proposal, not proof. CAL only promotes bindings after local
verification passes. Generated verifier harnesses are local code and are not a
sandbox boundary; see [SECURITY.md](SECURITY.md) for the trust model.

## Quickstart

Install the commands, use an isolated local state directory, and start CAL:

```sh
make install
export PATH="$(go env GOPATH)/bin:$PATH"
export CAL_HOME="$PWD/.cal-demo"
rm -rf "$CAL_HOME"
calctl daemon start --json
calctl daemon status --json
```

Configure an OpenAI-compatible LLM provider:

```sh
export CAL_LLM_API=chat_completions
export CAL_LLM_BASE_URL="<openai-compatible-base-url>"
export CAL_LLM_MODEL="<model>"
export CAL_LLM_API_KEY="<api-key>"
```

Acquire a capability from a real local CLI. On macOS, `plutil` is a useful
first target:

```sh
PROVIDER_PATH="$(command -v plutil)"
calctl discovery run \
  --provider-path "$PROVIDER_PATH" \
  --json
```

Use the acquired capability by intent:

```sh
cat > /tmp/cal-sample.plist <<'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>message</key>
  <string>hello CAL</string>
</dict>
</plist>
EOF

calctl use \
  "convert /tmp/cal-sample.plist to json" \
  --json
```

`use` can infer inputs from the intent when LLM configuration is available and
will create a temporary output path when the selected binding needs one.

Inspect evidence and stop the service:

```sh
calctl eval --json
calctl daemon stop --json
```

See [docs/quickstart.md](docs/quickstart.md) for the full walkthrough, live LLM
configuration, and troubleshooting notes.

## Commands

| Command | Purpose |
| --- | --- |
| `calctl daemon` | Start, stop, and inspect the local CAL service. |
| `calctl discovery run` | Observe a provider and acquire verified bindings. |
| `calctl capabilities list` | List promoted reusable capabilities. |
| `calctl use` | Route an intent through promoted bindings. |
| `calctl eval` | Inspect local acquisition and reuse evidence. |

## Development

Build local binaries without installing them:

```sh
make build
build/bin/calctl --help
```

Run tests:

```sh
make test
make e2e
```

Generated binaries under `build/bin/` are ignored by git.
