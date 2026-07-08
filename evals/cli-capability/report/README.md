# CLI Capability Benchmark Reports

Release-facing reports should be generated from benchmark summaries, not from
ad-hoc experiment notes.

Minimum generated tables:

- closed-loop matrix;
- held-out oracle success;
- capability model evidence;
- aggregate metrics;
- cost summary;
- failure taxonomy;
- reuse-suite baseline comparison.

The report should keep the paper questions visually separate:

- Acquisition Suite: acquisition matrix, proposal stages, probe/promote evidence,
  and failure taxonomy. Do not present non-CAL baselines as an acquisition
  comparison.
- Capability Model Suite: provider coverage, capability coverage, and
  Provider -> Capability -> Binding structure. Non-CAL methods are represented
  as having no durable structure.
- Reuse Suite: held-out use/run results, independent oracle status, and the main
  method comparison. The current runner renders direct CLI baseline results;
  LLM one-shot and provider-tool rows should appear only after those runners are
  implemented.
