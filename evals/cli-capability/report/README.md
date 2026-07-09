# CLI Capability Paper Reports

Release-facing reports are generated from scenario-first benchmark artifacts,
not from ad-hoc experiment notes.

The HTML report renders four paper sections:

1. Experiment 1: Acquiring Capabilities From Provider Surfaces
2. Experiment 2: Verification And Failure Gating
3. Experiment 3: Capability Structure Evidence
4. Experiment 4: Repeated Held-Out Reuse

Minimum generated tables:

- acquisition evidence table
- verification/failure-gating table
- provider-to-capability and capability-to-provider structure tables
- repeated held-out reuse table
- repeated reuse method comparison
- raw summary, scores, capability model, and failure taxonomy details

Report gates use `experiment_gates`.
