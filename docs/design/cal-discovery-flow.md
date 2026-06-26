# CAL Discovery Flow

Discovery is the process that finds local provider entries and turns verified provider-specific executions into reusable capability bindings.

The stable core model is:

```text
Provider
Capability
Binding
```

Discovery also writes a `Trace` for explanation, debugging, and evaluation. `Trace` is a process log, not a core model.

`LLM proposer` participation is limited by default. CAL concentrates `LLM proposer` use in Inference and keeps Verification and Promotion deterministic unless a probe plan needs fallback proposal generation.

## Flow

```text
Entry
-> Inference
-> Verification
-> Promotion
```

### Entry

Entry finds provider entries from configured provider sources.

```text
config.provider_sources
-> inspect configured path sources
-> create/update Provider
```

Entry details are defined in `docs/design/cal-discovery-entry.md`.

### Inference

Inference observes a Provider and proposes candidate bindings.

```text
Provider
-> observations
-> candidate binding
```

Inference details are defined in `docs/design/cal-discovery-inference.md`.

### Verification

Verification tests a candidate binding with a safe probe.

```text
Trace.candidates[]
-> selected candidate
-> safe probe
-> Trace.probes[]
```

Verification details are defined in `docs/design/cal-discovery-verification.md`.

A candidate is still not reusable after Inference. It becomes promotable only after Verification passes, and reusable only after Promotion writes the durable records.

### Promotion

Promotion writes verified results into the core model.

```text
passed probe + candidate
-> create/update Capability
-> create/update Binding
```

Promotion details are defined in `docs/design/cal-discovery-promotion.md`.

## Trace

`Trace` records what happened during one discovery attempt. It is an artifact for explanation, debugging, and evaluation, not a core model.

```text
Entry / Inference / Verification / Promotion
-> Trace
```

Trace details are defined in `docs/design/cal-discovery-trace.md`.

## LLM proposer Participation

Proposal and `LLM proposer` call boundaries are defined in `docs/design/cal-discovery-proposal.md`. Live LLM adapter configuration is defined in `docs/design/cal-discovery-llm.md`.

## Target Command Slice

The target first command slice separates provider finding from targeted
capability acquisition.

Supported:

```text
calctl providers find --kind cli --json
  Find CLI provider entries from configured provider sources and write Provider records.

calctl discovery run --provider-path <provider-path> --json
  Create or update one explicit provider entry, then run targeted acquisition for that provider.

calctl discovery run --provider-id <provider-id> --json
  Service-routed targeted Inference, Verification, Promotion, and Trace writing.

calctl discovery run --provider-path <provider-path> --proposal-path <proposal.json> --json
  Replay one SOP or LLM-style proposal through the same Verification and Promotion path.

calctl runs create --capability-id <capability-id> --inputs-json <json> --json
  CLI execution for promoted bindings. Add --verify when deterministic outcome evidence is required.

calctl use <intent> --json
  Semantic reuse over promoted bindings. It does not trigger Discovery; it routes
  the intent to already promoted records and delegates execution to Run.
```

The targeted loop supports live LLM and replayed SOP or LLM-style proposal JSON
for candidate, probe-plan, and generated verifier-harness input. It can promote
multiple verified candidates from one provider attempt. It supports
`--provider-id <provider-id>` for already discovered providers and `<path>` for one
explicit provider entry path. The explicit path form first creates or updates
the Provider record, then runs the same targeted acquisition loop.
`--capability-id` is only a debug filter. Proposal replay is schema-validated and
still uses CAL execution plus deterministic script verifier results. Local
trusted or generated verifier scripts live under `CAL_HOME/verifiers/`;
production runtime does not load embedded default verifier scripts. The full
verifier contract is defined in `docs/design/cal-discovery-verification.md`.
Deterministic rules for the controlled fake CLI pattern, macOS `cupsfilter`,
and macOS `sips` are retained behind a hidden baseline/regression mode, not as
the production first attempt. It does not yet use app menu observation, AX
actions, or marker-free `soffice` help parsing.

Targeted acquisition writes a completed Trace only after promotion succeeds. If
observation, candidate proposal generation, Verification, or Promotion fails after a
provider is selected, it writes a failed Trace with the available observations,
candidates, probes, and structured error. Provider lookup errors do not create
an acquisition Trace.
