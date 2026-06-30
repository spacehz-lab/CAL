# LLM One-Shot CLI Baseline Prompt

You are given:

- one benchmark task family;
- provider command help text or concise documentation;
- concrete runtime inputs for one held-out benchmark instance.

Return exactly one JSON object:

```json
{
  "command": ["program", "arg1", "arg2"],
  "writes_target": true
}
```

Rules:

- Do not use CAL capabilities, bindings, traces, or verify specs.
- Produce a direct CLI command for the requested provider.
- Use the supplied `target` path when the provider supports writing output to a
  file.
- If the command prints the result to stdout, set `writes_target` to `false` so
  the runner can capture stdout into `target`.
- Do not claim success. The benchmark oracle will score the output.
