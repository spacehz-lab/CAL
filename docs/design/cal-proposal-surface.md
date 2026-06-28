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
