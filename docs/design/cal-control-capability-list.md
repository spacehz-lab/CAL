# CAL Capability List

Capability List is the first agent-facing access module after Discovery Promotion.

It exposes the current reusable capability surface to an agent or human through a stable machine-readable catalog.

## Role

Capability List answers one question:

```text
What promoted capabilities can CAL currently offer, and what does an agent need to know before choosing one?
```

It does not choose a binding and does not execute anything.

The intended semantic online path is:

```text
agent task
-> calctl use <intent> --json
-> CAL selects a promoted capability and binding
-> CAL runs the selected binding
```

The deterministic lower-level path remains:

```text
agent task
-> calctl capabilities list --json
-> agent selects capability_id
-> calctl runs create --capability-id <capability_id> ...
```

`use` owns semantic intent matching. `run` owns deterministic binding
resolution and execution. `list` owns only capability exposure.

## Input

Default command:

```bash
calctl capabilities list --json
```

Optional exact filters:

```bash
calctl capabilities list --provider-id <provider-id> --json
calctl capabilities list --capability-id <capability-id> --json
```

Filters are deterministic. They are not semantic intent matching.

List should avoid:

```text
--intent
--semantic
LLM-backed query
embedding-backed query
```

An agent may use its own planner to select a capability from the returned catalog.
CAL List must not become a planner. CAL-owned semantic selection belongs to
Capability Use.

## Output

The output is a compact catalog, not the full persisted record.

Shape:

```json
{
  "count": 1,
  "capabilities": [
    {
      "id": "document.export_pdf",
      "description": "Create a PDF artifact from an editable document.",
      "bindings": {
        "available": 2,
        "provider_ids": [
          "provider_soffice",
          "provider_preview"
        ],
        "verifiers": [
          "file_exists",
          "file_parse_pdf"
        ]
      }
    }
  ]
}
```

Required fields per listed capability:

```text
id
bindings.available
bindings.provider_ids[]
bindings.verifiers[]
```

Optional fields:

```text
description
```

`list` should include enough information for an agent to decide whether a capability is relevant and whether it can provide the required inputs.

Capability records describe abstract reusable operations. Provider-specific
runtime inputs and accepted values belong to `Binding.input_constraints` and
the selected `Binding.execution`, not to capability-level schemas.

## Binding Exposure

List exposes binding summaries only.

It should not expose full provider-specific execution plans by default.

Allowed binding summary fields:

```text
provider ids
verifier ids
binding count
```

Do not include by default:

```text
execution spec
input constraints
raw UI selectors
paths that are not needed for selection
Trace references
candidate rationale
probe internals
```

Detailed execution data belongs to `run` internals and debugging output, not the normal list catalog.

## Selection Semantics

List never says:

```text
Use this binding.
This is the fastest binding.
This task matches this capability.
```

List can say:

```text
This capability exists.
This capability has reusable bindings.
These provider ids can realize it.
These verifier ids are available.
```

The agent or Capability Use decides whether the current task matches a
`capability_id`.

`run` later decides which binding to use.

## Source Of Truth

Capability List reads only the reusable core model:

```text
Capability
Binding
```

It must not read process-only discovery material:

```text
Trace.observations
Trace.candidates
Trace.probes
failed or ambiguous candidates
```

Only bindings promoted by Discovery Promotion are visible through List.

## Ordering

Output order must be deterministic:

```text
capability id ascending
provider id ascending inside summaries
verifier id ascending inside summaries
```

Deterministic ordering keeps agent prompts, tests, and diffs stable.

## Empty Result

If no promoted capability is available, List returns a successful empty catalog:

```json
{
  "count": 0,
  "capabilities": []
}
```

It must not trigger discovery automatically.

Discovery is an explicit background or targeted flow owned by CAL discovery, not a side effect of listing.

## Boundary

Capability List owns:

```text
reading promoted capabilities
filtering by exact provider_id or capability_id
redacting execution details
returning stable JSON for agents
```

Capability List does not own:

```text
natural-language intent understanding
semantic search
Use selection
binding resolution
binding execution
outcome verification
Discovery Inference
Discovery Verification
Discovery Promotion
Trace writing
run history scoring
```

## Agent Policy

A skill or agent policy can use List like this:

```text
Before performing low-level GUI, shell, or app actions for a task that looks reusable, call `calctl capabilities list --json`.
Choose a capability only when its id and description match the task, and its binding summary shows an available provider/verifier surface that is acceptable for the environment.
If no listed capability fits, continue normal agent behavior or trigger an explicit discovery flow.
After selecting a capability_id, call `calctl runs create`.
```

For user-facing semantic reuse, prefer `calctl use <intent>` so CAL can
select the capability and fill binding-specific control inputs. Use List
directly when the caller already wants to inspect the catalog or choose an exact
`capability_id`.
