# CAL Proposal Capability

Proposal Capability is the second internal Proposal stage.

It receives Surface output and chooses the provider-independent capability ids
that later Binding and Evidence stages must use.

## Input

```text
Provider
kept surface_items[]
existing Capability ids
optional debug capability filter
```

`existing Capability ids` are a bounded local lookup result. They are reuse
candidates, not proof that the provider implements those capabilities.

The debug filter is not a task hint and must not change capability id rules.

## Output

Capability outputs a global plan:

```text
capabilities[]
  capability_id
  description optional
  source_surface_ids[]
  confidence high | medium | low
  rationale optional
deferred[]
  surface_id
  reason
```

## Capability Id Rules

Capability ids use exactly two lowercase dotted parts:

```text
<subject>.<operation>
```

They must be provider-independent and semantic. Do not include:

```text
provider names
executable names
command names
flags
paths
versions
formats
encodings
checksum algorithms
modes
target artifact kinds
```

Discriminators belong to Binding execution, input constraints, and Evidence
checks.

Preferred examples:

```text
file.checksum
text.encode
text.decode
document.convert
archive.create
key.generate
certificate.verify
bytes.generate
```

Avoid:

```text
file.sha1sum
text.base64_encode
document.pdf_export
archive.create_zip
```

## Reuse Rules

Reuse an existing Capability id only when the observed subject and operation are
semantically equivalent.

If no existing id matches, generate a new id following the same two-part rule.

Capability is the only Proposal stage that may choose or generate
`capability_id`. Binding and Evidence must reject or normalize away any attempt
to rename it.

## Boundary

Capability can conclude:

```text
These provider surfaces should be explored under these capability ids.
```

Capability cannot conclude:

```text
The provider-specific execution is known.
The binding works.
The outcome is verified.
```
