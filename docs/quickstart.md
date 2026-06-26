# CAL Quickstart

This guide runs one local loop: observe a CLI provider, promote a verified
binding, call it by intent, and inspect eval output.

## Install

```sh
make install
```

If your shell cannot find `calctl`, add Go's install directory to `PATH`:

```sh
export PATH="$(go env GOPATH)/bin:$PATH"
```

## Start CAL

Use a disposable local state directory:

```sh
export CAL_HOME="$PWD/.cal-demo"
rm -rf "$CAL_HOME"
```

Start the local service through `calctl`:

```sh
calctl daemon start --json
calctl daemon status --json
```

## Configure LLM

Configure an OpenAI-compatible provider:

```sh
export CAL_LLM_API=chat_completions
export CAL_LLM_BASE_URL="<openai-compatible-base-url>"
export CAL_LLM_MODEL="<model>"
export CAL_LLM_API_KEY="<api-key>"
```

Do not write API keys into repository files, `CAL_HOME/config.json`, traces,
proposal fixtures, or logs.

## Run Acquisition

Use a real local CLI. On macOS, `plutil` is a useful first target because it is
installed by default and exposes conversion behavior through its local docs:

```sh
PROVIDER_PATH="$(command -v plutil)"
calctl discovery run \
  --provider-path "$PROVIDER_PATH" \
  --json
```

On another platform, choose a local CLI that exists on your machine and has
useful `--help` or `man` output, then pass its absolute path with
`--provider-path`.

Expect `state: "succeeded"` and at least one promoted binding. The exact
capability id and verifier id are model-proposed and may vary. CAL only
promotes bindings whose generated probe passes locally.

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
  --json
```

When LLM configuration is available, `use` can extract explicit inputs from the
intent and CAL will create a temporary output path if the selected binding needs
one. You can also pass structured inputs explicitly:

```sh
calctl use \
  --intent "convert plist to json" \
  --inputs-json '{"source":"/tmp/cal-sample.plist","format":"json"}' \
  --json
```

Expected shape:

```json
{
  "status": "succeeded",
  "selection": {
    "capability_id": "<model-proposed-capability-id>"
  },
  "run": {
    "status": "succeeded",
    "verified": false
  }
}
```

Add `--verify` when you want CAL to run the promoted verifier again and attach
fresh evidence to the run.

## Inspect Evidence

```sh
calctl capabilities list --json
calctl eval --json
```

`eval` should show at least one provider, capability, promoted binding, trace,
and run.

## Stop CAL

```sh
calctl daemon stop --json
```

## Troubleshooting

- `cald executable was not found`: run `make install` and ensure
  `$(go env GOPATH)/bin` is on `PATH`.
- `candidate_proposal_failed`: check the LLM environment variables and try a
  CLI with richer help or manual output.
- `no_match`: acquisition did not promote a compatible binding; inspect
  `calctl capabilities list --json` and `calctl eval --json`.
