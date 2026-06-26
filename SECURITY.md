# Security Policy

CAL is a local-first release preview. It is intended to run on a developer
machine and to acquire capabilities from local provider actions, starting with
CLI providers.

## Trust Boundary

CAL does not treat LLM output as proof of success. Candidate bindings, probe
plans, and generated verifier harnesses must be executed and checked locally
before a binding is promoted.

Generated verifier harnesses are local materialized code. They are not a
sandbox. Run CAL only in environments where executing local verifier scripts is
acceptable, and inspect generated verifier packages when working with untrusted
providers or prompts.

## Secrets

Keep API keys and other secrets outside the repository and outside durable CAL
artifacts.

Do not commit:

- API keys or access tokens
- raw LLM prompts or full LLM responses containing secrets
- raw `CAL_HOME` directories
- logs or traces containing private local paths or sensitive provider output
- generated outputs under `evals/out/`

Use environment variables for live LLM credentials, for example
`CAL_LLM_API_KEY`. Do not write raw keys into `CAL_HOME/config.json`, proposal
fixtures, traces, logs, or eval artifacts.

## Local Service Scope

`cald` is a local control service. It should be bound to loopback for local use.
Do not expose it as a remote multi-user service without adding authentication,
authorization, request isolation, and a stronger execution sandbox.

## Reporting Issues

Please report security issues privately to the project maintainers instead of
opening a public issue with exploit details or secrets. Include the affected
command, platform, provider type, and a minimal reproduction when possible.
