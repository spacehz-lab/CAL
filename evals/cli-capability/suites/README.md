# CLI Capability Suites

The CLI Capability benchmark is physically split into three suites. Each suite
answers one paper question and writes evidence that can be rendered in the final
HTML report.

The suites share `providers.json`, `fixtures/`, `oracles/`, `proposals/replay/`,
and `baselines/`.

Planned suite files:

```text
evals/cli-capability/suites/
  acquisition.jsonl
  capability_model.jsonl
  reuse.jsonl
```

## Acquisition Suite

Question:

```text
Can CAL acquire verified provider-specific bindings from real CLI surfaces?
```

The suite covers common CLIs that large models likely know, third-party
production CLIs, and multiple business domains. Missing optional third-party
CLIs are reported as unavailable provider cases, not CAL acquisition failures.

Representative target set:

| Domain | Providers | Target capabilities |
| --- | --- | --- |
| Text and encoding | `base64`, `openssl` | `text.base64_encode`, `text.base64_decode` |
| Security and checksums | `shasum`, `openssl` | `file.checksum.sha1` |
| Archive and compression | `tar`, `zip`, `gzip`, `ditto` | `archive.create`, `file.compress` |
| Structured data | `jq`, `python3`, `plutil`, `yq` | `json.query`, `plist.convert`, `yaml.query` |
| Documents | `pandoc`, `pdftotext` | `document.convert`, `pdf.extract_text` |
| Media and images | `ffmpeg`, `magick` | `media.convert`, `image.resize`, `image.convert` |
| Search and extraction | `rg` | `text.search` |

Evidence per case:

- provider command, resolved provider path, platform, and availability;
- observation source, such as CLI help;
- proposal stage summaries for surface, capability, binding, and evidence;
- candidate count and selected candidate count;
- probe pass and fail count;
- verification level and check summary;
- promoted capability and binding ids;
- acquisition latency and proposal LLM latency when available;
- failure stage and reason.

HTML section:

```text
Acquisition Suite
-> acquisition matrix
-> proposal stage detail
-> probe and promotion evidence
-> acquisition failure taxonomy
```

## Capability Model Suite

Question:

```text
Does CAL form a capability layer above providers instead of wrapping each CLI as
one tool?
```

The suite intentionally includes providers that expose multiple capabilities and
capabilities that can be realized by multiple providers.

Representative target set:

| Evidence shape | Providers | Target capabilities |
| --- | --- | --- |
| One provider, many capabilities | `openssl` | `file.checksum.sha1`, `text.base64_encode`, `text.base64_decode` |
| One provider, many capabilities | `ffmpeg` | `media.convert`, `media.extract_metadata` |
| One provider, many capabilities | `magick` | `image.resize`, `image.convert` |
| One capability, many providers | `base64`, `openssl` | `text.base64_encode`, `text.base64_decode` |
| One capability, many providers | `shasum`, `openssl` | `file.checksum.sha1` |
| One capability, many providers | `jq`, `python3` | `json.query` |
| One capability, many providers | `tar`, `zip`, `ditto` | `archive.create` |
| One capability, many providers | `pandoc`, `python3` | `document.convert_md_html` |

Evidence per case:

- provider id, command, and resolved path;
- promoted capability ids per provider;
- promoted binding ids per capability;
- binding execution kind;
- verification level distribution;
- provider-to-capability mapping;
- capability-to-provider mapping;
- oracle pass count for promoted bindings when held-out reuse exists.

HTML section:

```text
Capability Model Suite
-> provider coverage table
-> capability coverage table
-> provider -> capability -> binding table
```

## Reuse Suite

Question:

```text
Can promoted bindings be reused on held-out inputs without another acquisition
step?
```

The suite uses held-out fixtures that differ from acquisition probe fixtures.
Task success is decided by benchmark oracles, not by CAL's internal verification
alone.

Representative target set:

| Domain | Providers | Held-out target |
| --- | --- | --- |
| Checksums | `shasum`, `openssl` | hash unseen files |
| Encoding | `base64`, `openssl` | encode and decode unseen text or bytes |
| Structured data | `jq`, `python3`, `plutil` | query or convert unseen JSON/plist files |
| Archive | `tar`, `zip` | archive unseen files |
| Documents | `pandoc`, `pdftotext` | convert or extract unseen document content |
| Media and images | `ffmpeg`, `magick` | convert or resize unseen media/image fixtures |
| Search | `rg` | search unseen text trees |

Evidence per case:

- acquisition fixture id and reuse fixture id;
- explicit held-out marker;
- `calctl use` selection status;
- shortlist size and selected capability, binding, and provider ids;
- runtime run status;
- CAL verification status;
- independent oracle status;
- use, run, and oracle latency;
- failure stage and reason.

HTML section:

```text
Reuse Suite
-> held-out reuse matrix
-> Use resolver detail
-> CAL verify vs independent oracle
-> cost amortization vs baselines
```

## Baseline Comparison

The suites compare CAL against three baselines:

| Baseline | Purpose | Reuse expectation |
| --- | --- | --- |
| Direct CLI oracle | Shows task feasibility and latency with a hand-authored correct command. | No reusable binding. |
| LLM one-shot CLI | Tests model memory and one-off command synthesis from provider help plus task input. | Repeats LLM work per task. |
| Provider tool baseline | Treats the CLI as a generic tool and lets the model choose arguments each time. | Repeats tool selection and argument synthesis per task. |
| CAL replay/live | Acquires once, promotes verified bindings, then reuses them through `use` and `run`. | Reuse should amortize acquisition cost. |

The main comparison is repeated-task amortization, not single-task latency:

```text
method / repeated tasks / LLM calls / tokens / total latency / oracle successes
-> average cost per oracle-verified success
```

CAL may be slower on the first task because acquisition has an upfront cost. The
paper claim depends on later held-out reuse reducing repeated LLM calls, repeated
prompt tokens, and repeated command-synthesis failures.
