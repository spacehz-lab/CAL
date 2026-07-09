# CLI Capability Paper Scoring

This scoring contract defines the scenario-first CAL arXiv v0 benchmark.

## Case Outcomes

Each scenario case should produce trace-backed records for one or more paper
experiments:

- `acquisition`: provider was available, candidate bindings were proposed,
  probes ran, deterministic verification decided promotion, and promoted or
  blocked records were stored.
- `verification_failure`: a controlled invalid candidate was blocked, with no
  false promotion.
- `capability_structure`: promoted records were analyzed for
  `Provider -> Capability*` and `Capability -> Binding* -> Provider` structure.
- `repeated_reuse`: promoted bindings were reused on held-out rounds and scored
  by benchmark oracles.

CAL internal verify specs decide promotion. Benchmark oracles decide held-out
task success. These are separate trust boundaries.

## Required Metrics

Aggregate output should report:

- attempted cases and provider-case pairs
- provider class and domain coverage
- provider availability
- candidate count
- probe pass/fail count
- promoted binding count
- negative evidence count
- false promotion count for controlled invalid cases
- multi-capability provider count
- multi-binding capability count
- held-out reuse rounds
- end-to-end reuse success rate
- conditional reuse success rate after promotion
- direct CLI oracle results
- repeated one-shot LLM baseline results
- acquisition, use, and baseline latency
- LLM calls and tokens when available

## Paper Experiment Gates

Gates are display and readiness criteria. They must not hide failed rows.

| Experiment | Formula | Target |
| --- | --- | --- |
| Acquisition | promoted provider-case bindings / available provider-case attempts | >= 85% |
| Verification failure | false promotions on controlled invalid cases | 0 |
| Verification failure | blocked invalid candidates / controlled invalid candidates | >= 95% |
| Capability structure | passed structure checks / non-skipped structure checks | >= 90% |
| Repeated reuse | oracle-passed held-out rounds / attempted held-out rounds | >= 90% |

Repeated reuse reports two rates:

```text
end_to_end_reuse = passed held-out rounds / all planned held-out rounds
conditional_reuse = passed held-out rounds with promoted upstream binding / eligible held-out rounds
```

The paper must report both rates so acquisition failures are not hidden by a
conditional denominator.

## Baselines

Baselines are attached to repeated held-out reuse:

- `direct_cli`: hand-authored feasibility oracle and latency reference.
- `llm_oneshot`: repeated model command generation for each held-out round.

CAL may be slower on the first case because acquisition has upfront cost. The
comparison is acquire-once/reuse-many versus repeated one-shot synthesis.

