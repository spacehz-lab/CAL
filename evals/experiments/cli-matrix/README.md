# CAL CLI Matrix Experiment

This experiment records CAL acquisition and reuse behavior across a small local
CLI matrix. It is an exploratory release evidence runner, not the
stable benchmark surface.

The experiment has two acquisition modes over the same case list:

- `replay`: proposal JSON comes from `proposals/replay/*.json`, so the run is
  deterministic and does not require an LLM API key.
- `live_llm`: proposal JSON comes from the configured LLM through CAL. This mode
  does not predeclare capability ids, verifiers, proposals, or run inputs.

Both modes execute the same black-box CAL path through compiled `calctl` and
`cald` binaries:

```text
calctl discovery run --provider-path ...
-> Trace persistence
-> probe execution
-> deterministic verifier execution
-> promotion
-> calctl runs create --verify
-> calctl eval
```

## Build

Build the binaries before running the experiment:

```sh
make build
```

## Replay

Replay mode defaults to the full 10-case matrix:

```sh
python3 evals/experiments/cli-matrix/run.py \
  --mode replay \
  --level full \
  --calctl build/bin/calctl \
  --cald build/bin/cald
```

Replay mode should complete verified reuse for every selected available CLI.

## Live LLM

Live LLM mode defaults to `smoke`, but `focus` is the recommended triage level:

```sh
CAL_LLM_API=chat_completions \
CAL_LLM_BASE_URL=<openai-compatible base url> \
CAL_LLM_MODEL=<model> \
CAL_LLM_API_KEY=<api key> \
  python3 evals/experiments/cli-matrix/run.py \
    --mode live_llm \
    --level focus \
    --calctl build/bin/calctl \
    --cald build/bin/cald
```

Live mode records partial candidate failures. The runner fails only if no
selected case produces a verified reuse.

## Run Levels

```text
smoke  base64_encode
focus  textutil_html, plutil_json, jq_query, base64_encode
full   all current cases
```

For precise debugging, pass a comma-separated case list:

```sh
python3 evals/experiments/cli-matrix/run.py \
  --mode live_llm \
  --cases textutil_html,jq_query \
  --calctl build/bin/calctl \
  --cald build/bin/cald
```

## Result Artifacts

By default each run writes:

```text
evals/out/experiments/cli-matrix/<run-id>/summary.json
evals/out/experiments/cli-matrix/<run-id>/index.html
evals/out/experiments/cli-matrix/<run-id>/cald.log
evals/out/experiments/cli-matrix/<run-id>/home/
```

The generated `home/` directory contains the local `CAL_HOME` used for that
run, including traces and generated verifier packages. Generated outputs under
`evals/out/` are ignored by git.

## Current Cases

Replay mode uses proposal-local generated verifier packages, not a runtime
default verifier catalog.

| CLI | Replay capability | Replay verification |
| --- | --- | --- |
| `textutil` | `document.convert_html` | generated HTML content check |
| `plutil` | `plist.convert_json` | generated JSON parse check |
| `jq` | `json.query` | generated text equality check |
| `base64` | `text.encode_base64` | generated source-vs-target Base64 check |
| `iconv` | `text.transcode` | generated text equality check |
| `shasum` | `file.hash_sha1` | generated source-vs-target SHA-1 check |
| `md5` | `file.hash_md5` | generated source-vs-target MD5 check |
| `file` | `file.detect_type` | generated text report check |
| `zip` | `archive.create_zip` | generated ZIP parse check |
| `gzip` | `file.compress_gzip` | generated gzip parse check |
