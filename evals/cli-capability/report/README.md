# CLI Capability Paper Reports

Release-facing reports are generated from scenario-first benchmark artifacts,
not from ad-hoc experiment notes.

The HTML report renders four paper sections:

1. Experiment 1: Acquiring Capabilities From Provider Surfaces
2. Experiment 2: Verification And Failure Gating
3. Experiment 3: Repeated Held-Out Reuse
4. Experiment 4: Capability Structure Evidence

Minimum generated tables:

- acquisition evidence table
- verification/failure-gating table
- reuse effectiveness table for `--reuse-profile effectiveness`
- round-level CAL reuse vs LLM one-shot table for `--reuse-profile comparison`
- provider-to-capability and capability-to-provider structure tables
- raw summary, scores, capability model, and failure taxonomy details

Report gates use `experiment_gates`.
