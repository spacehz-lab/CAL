# CAL Discovery Flow

Discovery is the process that finds local provider entries and turns verified
provider-specific executions into reusable capability bindings.

The stable core model is:

```text
Provider
Capability
Binding
```

Discovery also writes a `Trace` for explanation, debugging, and evaluation.
`Trace` is a process log, not a core model.

`LLM proposer` participation is limited to Proposal. Verification and Promotion
must remain deterministic CAL-owned work.

## Flow

```text
Entry
-> Proposal
-> Verification
-> Promotion
```

### Entry

Entry registers one explicit provider entry path.

```text
provider_path
-> inspect explicit entry
-> create/update Provider
```

Entry details are defined in `docs/design/cal-discovery-entry.md`.

### Proposal

Proposal observes a Provider and proposes candidate binding material.

```text
Provider
-> observations
-> Surface
-> Capability
-> Binding
-> Evidence
-> Trace.candidates[]
```

Proposal details are defined in `docs/design/cal-discovery-proposal.md`.

### Verification

Verification tests a candidate binding with a safe probe.

```text
Trace.candidates[]
-> selected candidate
-> execute safe probe
-> collect evidence
-> evaluate verify.checks
-> Trace.probes[]
```

Verification details are defined in `docs/design/cal-discovery-verification.md`.

A candidate is still not reusable after Proposal. It becomes promotable only
after Verification passes, and reusable only after Promotion writes the durable
records.

### Promotion

Promotion writes verified results into the core model.

```text
passed probe + candidate + verify level
-> create/update Capability
-> create/update Binding
```

Promotion details are defined in `docs/design/cal-discovery-promotion.md`.

## Trace

`Trace` records what happened during one discovery attempt. It is an artifact
for explanation, debugging, and evaluation, not a core model.

```text
Entry / Proposal / Verification / Promotion
-> Trace
```

Trace details are defined in `docs/design/cal-discovery-trace.md`.

## Proposal Participation

Proposal and `LLM proposer` call boundaries are defined in
`docs/design/cal-discovery-proposal.md`. Live LLM adapter configuration is
defined in `docs/design/cal-discovery-llm.md`.

The live LLM Proposal implementation may use four internal stages:

```text
Surface
-> Capability
-> Binding
-> Evidence
```

These are not public Discovery lifecycle stages. Deterministic parsers and
replay fixtures may produce equivalent final Proposal material without running
all four internal stages.

## Target Command Slice

The target first command slice separates provider registration from targeted
capability acquisition.

Supported:

```text
calctl providers add --provider-path <provider-path> --json
  Register one explicit provider entry path and write its Provider record.

calctl providers get --provider-path <provider-path> --json
  Check whether one explicit provider path is already registered.

calctl discovery run --provider-id <provider-id> --json
  Service-routed targeted Proposal, Verification, Promotion, and Trace writing.

calctl discovery run --provider-id <provider-id> --proposal-path <proposal.json> --json
  Replay one SOP or LLM-style proposal through the same Verification and Promotion path.

calctl runs create --capability-id <capability-id> --inputs-json <json> --json
  CLI execution for promoted bindings. Add --verify when deterministic outcome evidence is required.

calctl use <intent> --json
  Semantic reuse over promoted bindings. It does not trigger Discovery; it routes
  the intent to already promoted records and delegates execution to Run.
```

The targeted loop supports live Proposal and replayed SOP or LLM-style proposal
JSON for candidate, probe, and verify-spec input. It can promote multiple
verified candidates from one provider attempt. It supports
`--provider-id <provider-id>` for already registered providers. Provider path
bootstrap belongs to `calctl providers add --provider-path`.

`--capability-id` is only a debug filter. Proposal replay is schema-validated
and still uses CAL execution plus deterministic verification. New proposals
should prefer `method=execute` with built-in `verify.checks`. Use
`method=contract` only for weak evidence when real probing would change state,
require network, require interaction, or be otherwise unsafe. Contract verify
specs do not carry checks.

Synchronous targeted discovery may take longer than ordinary control calls, so
the client uses an extended timeout. Individual execute probes still have their
own shorter timeout so one slow command does not dominate verification.

Targeted acquisition writes a completed Trace only after promotion succeeds. If
observation, Proposal, Verification, or Promotion fails after a provider is
selected, it writes a failed Trace with the available observations, candidates,
probes, and structured error. Provider lookup errors do not create an
acquisition Trace.
