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
- Produce a standalone CLI command for the requested provider.
- Return `command` as argv only. Do not return shell strings, pipes, command
  substitution, or wrapper scripts.
- The first argv item must be the requested provider command.
- Use the supplied `target` path when the provider supports writing output to a
  file.
- If the command prints the result to stdout, set `writes_target` to `false` so
  the runner can capture stdout into `target`.
- Do not claim success. The benchmark oracle will score the output.
