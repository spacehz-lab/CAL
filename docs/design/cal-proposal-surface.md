# CAL Proposal Surface

Proposal Surface is the first internal Proposal stage.

It converts raw provider observations into a bounded list of documented surfaces
that may expose reusable operations.

## Input

```text
Provider
observations
optional debug capability filter
```

The debug filter is not a task hint. It may narrow inspection during developer
diagnostics, but normal acquisition should not depend on user task intent.

Observation sources may include:

```text
CLI help
subcommand lists
manual pages
menu trees
AX trees
DOM structures
API descriptions
app or service metadata
```

## Output

Surface outputs a compact list:

```text
surface_items[]
  id
  kind
  name
  description optional
  evidence_source
  decision keep | defer | skip
  rationale optional
```

`keep` means the item is documented enough to consider for capability planning.
`defer` means the item may require deeper observation or unsafe interaction.
`skip` means the item should not enter Capability planning.

## Rules

Surface must not propose `capability_id`.

Surface must not produce candidate executions, probe inputs, or verify checks.

Surface should prefer broad command or action coverage over semantic depth, but
it must still prune:

```text
interactive-only surfaces
server/listener modes
network actions without controlled fixtures
destructive operations
metadata-only commands
aliases that do not add operation coverage
```

For complex CLIs, Surface should keep a bounded number of documented primary
commands or command families instead of enumerating every variant flag.

## Local Policy

CAL applies a local policy gate after the LLM Surface stage and before
Capability planning. This policy filters Surface items, not raw observations.
Raw observations remain process evidence.

The policy file is:

```text
CAL_HOME/proposal_policy.json
```

It is a complete policy file, not an incremental override. The default file is:

```json
{
  "surface": {
    "allowed_kinds": ["command", "subcommand", "mode", "option"],
    "skip_names": ["help", "version", "usage"],
    "skip_patterns": []
  },
  "capability": {
    "allowed_subjects": [],
    "blocked_subjects": []
  }
}
```

Surface currently consumes only:

```text
surface.allowed_kinds
surface.skip_names
surface.skip_patterns
```

Capability policy fields are reserved for the Capability stage.

## Trace Diagnostics

Surface writes parsed Stage1 decisions into the discovery trace:

```text
trace.proposal.stages[]
  name = surface
  summary.raw
  summary.keep
  summary.defer
  summary.skip
  summary.selected
  items[]
    id
    kind
    name
    decision
    rationale optional
```

`items[]` records final Stage1 decisions after local policy. A model-returned
`keep` may become `skip` when blocked by local policy, for example
`help/version/usage`.

Only `selected` `keep` items enter Capability planning. Deferred and skipped
items stay in trace diagnostics so operators can distinguish:

```text
not observed
observed but deferred
observed but skipped
selected for Capability planning
```

Surface diagnostics must not store raw LLM response text by default.

## Boundary

Surface can conclude:

```text
This observed surface is worth considering.
```

Surface cannot conclude:

```text
This is a reusable capability.
This provider binding works.
This outcome is verified.
```
