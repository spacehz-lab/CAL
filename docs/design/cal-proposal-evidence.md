# CAL Proposal Evidence

Proposal Evidence is the fourth internal Proposal stage.

It receives candidate binding material and proposes the deterministic evidence
checks that Verification should run after executing the probe.

Evidence is a planning stage. It does not execute providers and does not decide
pass/fail.

## Input

```text
Provider
candidate
probe inputs
fixtures optional
observations
```

## Output

Evidence outputs a `verify` spec:

```text
verify
  level L0 | L1 | L2 | L3
  checks[]
    subject
    predicate
    params optional
  fallback optional
```

Prompt payloads may use a compact check array form, but CAL should normalize the
accepted result to named fields before storing it in core records.

## Verification Levels

```text
L3 semantic
  Verifies the semantic result itself.
  Example: target bytes equal base64_decode(source).

L2 structural
  Verifies output structure, format, or key properties.
  Example: target exists, is non-empty, and parses as PDF.

L1 behavioral
  Verifies that an action occurred or state changed.
  Example: URL changed, DOM element appeared, AX node appeared.

L0 unsupported
  No reliable deterministic verification is available.
```

The model may suggest a level, but CAL owns final level validation. Checks such
as `exists` and `non_empty` must not be treated as L3 unless the capability
semantics are only artifact creation.

## Evidence Subjects

Initial subjects:

```text
stdout
stderr
source
target
artifact
```

Future provider kinds may add:

```text
dom
url
network
ax
screenshot
app_state
```

Subjects identify evidence available after execution. They do not identify a
fallback implementation.

## Predicates

Initial CLI predicates should stay small:

```text
exists
non_empty
format
contains
regex
bytes_equal_transform
hash_line_matches
```

Examples:

```json
{
  "level": "L3",
  "checks": [
    {
      "subject": "target",
      "predicate": "bytes_equal_transform",
      "params": {
        "source": "source",
        "transform": "base64_decode"
      }
    }
  ]
}
```

```json
{
  "level": "L3",
  "checks": [
    {
      "subject": "stdout",
      "predicate": "hash_line_matches",
      "params": {
        "source": "source",
        "algorithm": "sha1"
      }
    }
  ]
}
```

```json
{
  "level": "L2",
  "checks": [
    {"subject": "target", "predicate": "exists"},
    {"subject": "target", "predicate": "non_empty"},
    {"subject": "target", "predicate": "format", "params": {"format": "pdf"}}
  ]
}
```

## Fallback

If built-in checks cannot express the required outcome, Evidence may request a
script or plugin fallback:

```text
verify.fallback
  type script | plugin
  id
```

Fallback is not the default path. It must be reported separately in evaluation
and should not be used when built-in checks can express the evidence relation.

If neither built-in checks nor fallback can provide deterministic evidence,
Evidence should return `L0` and the candidate should not be promoted by default.

## Boundary

Evidence can conclude:

```text
These checks are a candidate verification plan.
```

Evidence cannot conclude:

```text
The candidate passed.
The binding should be promoted.
The LLM-proposed level is trusted without local validation.
```
