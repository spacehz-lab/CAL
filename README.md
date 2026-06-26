# CAL

CAL is a local Capability Acquisition Layer. It observes provider action
surfaces, promotes verified provider-specific bindings into reusable
capabilities, and lets later requests route through `calctl use`.

The current implementation is local-only and CLI-first.

## Quickstart

Install the local commands:

```sh
make install
```

If `calctl` is not found after install, add Go's install directory to `PATH`:

```sh
export PATH="$(go env GOPATH)/bin:$PATH"
```

Use an isolated local state directory:

```sh
export CAL_HOME="$PWD/.cal-demo"
rm -rf "$CAL_HOME"
```

Start CAL:

```sh
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
will create a temporary output path when the selected binding needs one. For a
deterministic call, pass structured inputs explicitly:

```sh
calctl use \
  --intent "convert plist to json" \
  --inputs-json '{"source":"/tmp/cal-sample.plist","format":"json"}' \
  --json
```

Add `--verify` when you want CAL to run the promoted verifier again and attach
fresh evidence to the run.

Inspect evidence and stop the service:

```sh
calctl eval --json
calctl daemon stop --json
```

See [docs/quickstart.md](docs/quickstart.md) for the full walkthrough, live LLM
configuration, and troubleshooting notes.

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
