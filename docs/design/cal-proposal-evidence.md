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

The Stage4 model output is a draft:

```text
verify
  method execute | contract
  checks[]
    subject
      type file | stdout | stderr | exit_code
      input file subjects only
    predicate
    params optional
```

Proposal materializes the final `VerifySpec` by deriving `level` locally:

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

## Decision Process

Evidence should internally decide the `VerifySpec` in this order:

```text
if the probe is safe, short, local, reads only probe fixtures, and writes only
declared probe outputs inside the probe workdir, use execute

if execution would install, remove, update, upgrade, clean, link, unlink, tap,
untap, edit, start services, require network, require interaction, or change
external state, use contract

contract checks are advisory and are not executed

for execute, choose the strongest built-in deterministic checks supported by
probe material and observations

CAL derives the final level from method and checks, not from the capability name
or candidate description
```

Fixture-only sample content must not become durable expected output unless
observations explicitly say the command always emits that literal.

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

The model must not output `verify.level`. CAL owns final level derivation.
Checks such as `exists` and `non_empty` must not be treated as L3 unless the
capability semantics are only artifact creation.

Probe fixtures are temporary sample inputs for acquisition. Evidence must not
promote fixture content into durable `VerifySpec` checks unless observations
explicitly say the command always emits that literal. Proposal normalizes
fixture-only literal checks out of the final verify spec and derives the verify
level from the remaining checks.

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
  semantics use contract L1.
```

`contract` is materialized as `L1`. Contract checks, if present, are advisory
and are not executed. `execute + L1` still executes the probe and must include
checks.

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

The core rule table must also declare bounded parameter values when a predicate
only supports a fixed set. Initial fixed values are:

```text
format: pdf | png | json | text
bytes_equal_transform.transform: base64_encode | base64_decode
hash_line_matches.algorithm: sha1 | sha256 and common sha-1/sha-256 aliases
```

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
