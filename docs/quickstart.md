# CAL Quickstart

This guide runs one local loop: register a provider, acquire a verified binding,
reuse it by intent, and inspect local evidence.

## Install

```sh
make install
export PATH="$(go env GOPATH)/bin:$PATH"
```

The examples use `jq` to extract ids from JSON output.

For local development without installing:

```sh
make build
export PATH="$PWD/build/bin:$PATH"
```

## Start CAL

Use a disposable local state directory:

```sh
export CAL_HOME="$PWD/.cal-demo"
rm -rf "$CAL_HOME"
```

Start the daemon in the background:

```sh
calctl daemon start --json
calctl daemon status --json
```

Use `cald serve` directly when you want a foreground process for debugging or a
process manager.

## Configure LLM

Live acquisition and LLM-assisted `use` require an OpenAI-compatible endpoint:

```sh
export CAL_LLM_API=chat_completions
export CAL_LLM_BASE_URL="<openai-compatible-base-url>"
export CAL_LLM_MODEL="<model>"
export CAL_LLM_API_KEY="<api-key>"
```

Do not write API keys into repository files, `CAL_HOME/config.json`, traces,
proposal fixtures, or logs.

## Register A Provider

Use a real local CLI. On macOS, `plutil` is a useful first target because it is
installed by default and exposes conversion behavior through local docs:

```sh
PROVIDER_PATH="$(command -v plutil)"
PROVIDER_ID="$(calctl providers add --provider-path "$PROVIDER_PATH" --json | jq -r .id)"
calctl providers list --json
```

On another platform, choose a local CLI with useful `--help` or `man` output.

## Run Acquisition

Run targeted acquisition for the registered provider:

```sh
calctl acquisition run \
  --provider-id "$PROVIDER_ID" \
  --hint "convert plist to json" \
  --stream --json
```

The acquisition flow is:

```text
entry -> observe -> proposal -> probe -> promote -> tracelog
```

`proposal` may use live LLM output, but promotion requires local probe evidence.
The exact capability id is model-proposed and may vary.

For deterministic replay tests or demos, use:

```sh
calctl acquisition run \
  --provider-id "$PROVIDER_ID" \
  --mode replay \
  --proposal-path /path/to/proposal.json \
  --json
```

## Use The Capability

Create an input property list:

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
```

Route a user intent through promoted bindings:

```sh
calctl use \
  "convert /tmp/cal-sample.plist to json" \
  --stream --json
```

When LLM configuration is available, `use` can select a promoted binding and
derive missing inputs from the intent. You can also pass structured inputs:

```sh
calctl use \
  "convert plist to json" \
  --inputs-json '{"source":"/tmp/cal-sample.plist","target":"/tmp/cal-sample.json"}' \
  --verify \
  --json
```

If you already know the capability id, run it directly:

```sh
calctl runs create \
  --capability-id "<capability-id>" \
  --provider-id "$PROVIDER_ID" \
  --inputs-json '{"source":"/tmp/cal-sample.plist","target":"/tmp/cal-sample.json"}' \
  --verify \
  --json
```

## Inspect Evidence

```sh
calctl capabilities list --json
calctl eval --json
```

`eval` summarizes local provider, capability, binding, trace, and run records.

## Stop CAL

```sh
calctl daemon stop --json
```

## Troubleshooting

- `cald unavailable`: run `calctl daemon start --json` with the same
  `CAL_HOME`, then retry `calctl daemon status --json`.
- `invalid_request`: check that required flags such as `--provider-id` or
  `--proposal-path` are present for the selected mode.
- `proposal_failed`: check the LLM environment variables, try `--stream --json`
  for proposal-stage diagnostics, or use a provider with richer help output.
- `binding_not_found` or `no_match`: acquisition did not promote a compatible
  binding; inspect `calctl capabilities list --json` and `calctl eval --json`.
