# CAL Proposal Binding

Proposal Binding is the third internal Proposal stage.

It receives one planned capability and materializes provider-specific candidate
binding material.

## Input

```text
Provider
one capability plan item
relevant surface_items[]
observations
```

After Capability planning, CAL may run Binding stages in parallel per
capability id with a bounded concurrency limit.

## Output

Binding outputs candidate and probe material:

```text
candidates[]
  provider_id
  capability_id
  description
  input_constraints optional
  execution

probe_material[]
  candidate_index
  inputs
  fixtures optional
```

The candidate `capability_id` must match the current capability plan item.

## Execution Rules

For CLI executions:

```text
execution.spec.args must not include the executable path
CAL supplies the provider path
runtime inputs use placeholders such as {{source}}, {{target}}, {{format}}
```

Every placeholder used in `execution.spec.args` must have a matching runtime
input or fixture-backed input in probe material.

If the observed CLI prints the candidate's primary result to stdout, Binding
must make stdout explicit by setting `execution.spec.stdout_path_input` to the
path input that should receive stdout, usually `target`.

If a target artifact is checked later, candidate execution must produce that
target either through an argument placeholder or through `stdout_path_input`.

## Input Constraints

`input_constraints` may describe only placeholders that appear in execution.

Use constraints for documented accepted values, formats, modes, or meanings.
Do not invent enum values.

Runtime discriminators such as `format`, `algorithm`, `encoding`, and `mode`
stay in inputs and constraints. They do not alter `capability_id`.

## Boundary

Binding can conclude:

```text
This provider-specific execution may implement the planned capability.
These controlled inputs can probe it.
```

Binding cannot conclude:

```text
The execution passed.
The binding is durable.
The verification level is final.
```
