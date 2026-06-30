# CAL Proposal Evidence

Proposal Evidence is the fourth internal Proposal stage.

It receives candidate binding material and proposes how Verification should
collect evidence.

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
  method execute | contract
  checks[]
    subject
      type file | stdout | stderr | exit_code
      input file subjects only
    predicate
    params optional
```

`level` describes confidence and promotion/use policy. `method` describes how
evidence is collected.

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

Probe fixtures are temporary sample inputs for acquisition. Evidence must not
promote fixture content into durable `VerifySpec` checks unless observations
explicitly say the command always emits that literal. Proposal normalizes
fixture-only literal checks out of the final verify spec and caps the verify
level to match the remaining checks.

## Verification Methods

```text
execute
  CAL executes the probe and evaluates built-in checks locally. Safe execute
  probes may read probe fixtures and write declared probe outputs inside the
  probe workdir.

contract
  CAL does not execute the probe. It records weak contract evidence from
  observations when execution would install, remove, update, edit, start
  services, require network, require interaction, or change external state.
  Unsafe commands with a clear observed command path and documented operation
  semantics use contract L1. Contract L0 is reserved for ambiguous observations
  where CAL cannot identify a reliable command path or operation semantics.
```

`contract` cannot exceed `L1` and must not include checks. `contract + L0` is
not promoted. `execute + L1` still executes the probe and must include checks.

## Evidence Subjects

Initial subject types:

```text
file
  Reads a file path from a named probe/run input such as source or target.

stdout
  Reads process stdout.

stderr
  Reads process stderr.

exit_code
  Reads process exit code.
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

The valid subject type, predicate, and parameter combinations are owned by the
core VerifySpec rule table. Proposal injects that rule table into the Stage4
prompt and `ValidateVerifySpec` enforces the same table before any probe or
promotion.

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

Predicate params are part of the contract and must be validated before
Verification runs. `equals`, `not_equals`, and `contains` use `params.value`;
`contains_any` uses `params.values`; `regex` uses `params.pattern`; `format`
uses `params.format`; `bytes_equal_transform` uses `params.source` and
`params.transform`; `hash_line_matches` uses `params.source` and
`params.algorithm`. File subjects must include `subject.input`, and that input
must be available in the probe material and future run inputs.

For stable literal output content, Evidence should prefer `contains` over
anchored full-file `regex`. Anchored regex is appropriate only when observations
specify the exact whole file content, including newline behavior.

Examples:

```json
{
  "level": "L3",
  "method": "execute",
  "checks": [
    {
      "subject": {"type": "file", "input": "target"},
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
  "method": "execute",
  "checks": [
    {
      "subject": {"type": "stdout"},
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
  "method": "execute",
  "checks": [
    {"subject": {"type": "file", "input": "target"}, "predicate": "exists"},
    {"subject": {"type": "file", "input": "target"}, "predicate": "non_empty"},
    {"subject": {"type": "file", "input": "target"}, "predicate": "format", "params": {"format": "pdf"}}
  ]
}
```

For a state-changing command that is well documented but unsafe to probe:

```json
{
  "level": "L1",
  "method": "contract"
}
```

If neither built-in checks nor contract evidence can provide reliable evidence,
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
