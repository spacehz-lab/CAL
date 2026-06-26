# CAL Discovery Inference

Discovery Inference is the second step of Discovery.

It observes a discovered `Provider` and proposes candidate bindings.

Inference does not verify candidates and does not create durable `Capability` or `Binding` records. Candidate bindings are process material recorded in `Trace`.

## Input

Inference input:

```text
Provider
hint optional
Trace optional
```

Fields:

```text
Provider
  A provider entry created by Discovery Entry.

hint optional
  A discovery hint, such as document.export_pdf or convert document.
  It can guide which observations and candidates are prioritized.

Trace optional
  The current discovery trace.
  Inference appends observations and candidates to it when present.
```

## Handling

Inference handling:

```text
select Provider
-> choose observation actions
-> collect observations
-> interpret observations
-> infer capability intent
-> select existing Capability id candidates
   -> reuse matching Capability.id
   -> or propose a new capability_id
-> propose candidate bindings
-> append observations and candidates to Trace
```

Observation sources can include:

```text
cli_output
app_hints
menu_tree
ax_tree
```

Inference may use an `LLM proposer` or deterministic parser to interpret observations. Production acquisition should treat `LLM proposer` proposals as the semantic path and keep hard-coded deterministic rules limited to baseline, fixture, and regression use unless a rule is truly generic. Proposal boundaries are defined in `docs/design/cal-discovery-proposal.md`; LLM prompt rules are defined in `docs/design/cal-discovery-llm-prompt.md`.

`LLM proposer` output is not proof. Every candidate must still pass Verification before Promotion.

## Output

Inference outputs candidate proposals through `Trace`.

```text
Trace
  observations[]
  candidates[]
```

Inference does not output:

```text
Capability
Binding
```

## Trace Data

Observation:

```text
observation
  provider_id
  type
  source optional
  content optional
  error optional
  created_at
```

Candidate:

```text
candidate
  provider_id
  capability_id
  description
  source optional
  input_constraints optional
  execution
  rationale optional
  created_at
```

Candidate fields:

```text
provider_id
  Provider that may implement the capability.

capability_id
  Proposed semantic capability id.

description
  Provider-independent description of the exact reusable operation.

source optional
  Human-readable source summary, such as "--convert-to in soffice --help".
  It is not an id or hard reference.

input_constraints optional
  Provider-specific accepted values or meanings for runtime placeholders used by
  execution. These constraints are part of the candidate Binding contract, not
  the abstract Capability.

execution
  Propose provider-specific execution.
  It is not promotable until Verification passes and is not reusable until Promotion writes durable records.

rationale optional
  Why the observation suggests this candidate.

created_at
  Candidate creation time.
```

Capability id selection is lookup-first. Inference must try to reuse an existing `Capability.id` before proposing a new one.

Before calling an `LLM proposer`, CAL should run a local Capability Catalog
Lookup over existing capabilities. The lookup takes provider observations from
CLI output, app hints, menu trees, AX trees, or other structured tree
observations and selects a bounded set of existing `Capability.id` candidates.
The first implementation may use deterministic lexical scoring over observation
text and capability ids. The selector may later be replaced or augmented by
BM25, embeddings, or hybrid retrieval without changing the proposal contract.
The first implementation must not require an embedding service or a separate
`LLM proposer` call.

The selected existing ids are a prompt budget boundary, not proof. They are
candidates for reuse and must still flow through candidate proposal generation,
Verification, and Promotion.

Minimal lookup order:

```text
1. If hint is an existing Capability.id, reuse it.
2. If the inferred capability_id exactly matches an existing Capability.id, reuse it.
3. Otherwise, run local Capability Catalog Lookup and pass selected existing ids to the `LLM proposer`.
4. Ask the `LLM proposer` to choose one selected existing Capability.id or return none.
5. Only when the result is none, propose a new capability_id.
```

When the `LLM proposer` chooses a reusable id, that choice must be constrained
to:

```text
reuse: <existing_capability_id>
none
```

New capability id rule:

```text
<domain>.<operation>
```

Capability ids must match:

```text
^[a-z0-9_]+\.[a-z0-9_]+$
```

Examples:

```text
document.export_pdf
document.open
edit.find_replace
image.resize
message.send
```

Capability ids must be provider-independent and outcome-oriented. Do not include app names, paths, menu labels, flags, versions, implementation details, or temporary task names.

Capability ids must not be broader than the candidate execution. If the
execution fixes the output type, result type, encoding, archive format, checksum
algorithm, or target artifact kind, include that result in the id:

```text
document.export_pdf
document.convert_html
plist.convert_json
image.convert_png
archive.create_zip
file.compress_gzip
file.hash_sha1
file.hash_md5
text.encode_base64
```

Use a parameterized id such as `document.convert_format` only when the execution
actually exposes the result type as a runtime input, for example `{{format}}`,
and the candidate description says the requested format is an input.

Do not add aliases, a taxonomy table, or a closed domain list in the first version.

Execution:

```text
execution
  kind
  spec
```

Execution kinds:

```text
cli
menu
ax_action
url_open
```

## Boundary

Inference can conclude:

```text
A Provider may support a capability through a candidate execution.
```

Inference cannot conclude:

```text
The candidate works.
The capability is reusable.
The candidate should be used by runtime.
```
