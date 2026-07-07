# Dependency Map

Dependency direction should make ownership obvious and prevent workflow packages from importing adapters or transport layers.

## Foundation

```text
model -> standard library only
progress -> model
pkg/jsonfile -> standard library only
store -> model, pkg/jsonfile
config -> pkg/jsonfile
logging -> config
llm -> standard library + OpenAI SDK only
```

`model` is the bottom layer for CAL contracts. No package in `internal` may be
imported by `model`.

`progress` is a thin fanout helper for `model.ProgressEvent`. It must not import
workflow packages, adapters, store, config, logging, or LLM.

`pkg/jsonfile` is a small shared file utility package. It must stay free of CAL
business types and must not import `internal`.

`llm` receives runtime `Options` from composition packages. It must not import
`config` or resolve environment variables.

## Acquisition

```text
acquisition -> model, progress, entry, observe, proposal, probe, promote, tracelog
entry       -> model, store
observe     -> model
observe/cli -> model, observe
proposal    -> model, llm, progress, proposal/surface, proposal/capability, proposal/binding, proposal/evidence, proposal/policy
proposal/{surface,capability} -> model, llm, proposal/policy
proposal/{binding,evidence} -> model, llm
proposal/policy -> model
proposal/{replay,rules} -> proposal, model
probe       -> model, proposal, execute, check
execute     -> model
execute/cli -> model, execute
check       -> model
promote     -> model
tracelog    -> model
```

`proposal/replay`, `proposal/rules`, and `proposal/policy` must not import `llm`; they are non-LLM proposal variants or policy helpers. They must not import `acquisition`, `probe`, `promote`, or adapters. Proposal stage packages must not import the main `proposal` package.

## Reuse

```text
run         -> model, progress, store, run/resolve, execute, check
run/resolve -> model, execute
use         -> model, progress, store, llm, use/select, use/plan, run
use/select  -> model, llm, execute
use/plan    -> standard library only
eval        -> model, store
```

`run/resolve` is deterministic binding selection for a known capability. `use/select` is semantic selection from intent and may use LLM.

## Adapters

```text
contract      -> model
httpserver    -> contract, progress, cald/app
cald/endpoint -> pkg/jsonfile
cli/client    -> contract, cald/endpoint
cli           -> contract, cli/client, cald/daemon, logging
cald/app      -> contract, model, store, config, llm, acquisition, entry, observe, observe/cli, proposal, probe, execute, execute/cli, check, promote, tracelog, run, use, eval
cald/daemon   -> cald/app, cald/endpoint, httpserver, logging
```

`httpserver` must not import acquisition, run, use, proposal, store, config, or llm directly. It talks to `cald/app`.

`cli` must not import acquisition, run, use, proposal, store, config, execute, check, or model workflow packages directly. It talks to `cli/client` or daemon lifecycle helpers.

`cli/client` must not import `cald/app`, `httpserver`, or workflow implementation packages.

`cald/endpoint` owns only local endpoint metadata file contracts and file IO. It must not import CLI or workflow implementation packages.

## Forbidden Dependencies

- `model` -> any `internal` package.
- `progress` -> acquisition, run, use, proposal, probe, promote, tracelog, store, config, logging, llm, contract, httpserver, cli, or cald.
- `store` -> acquisition, proposal, probe, promote, run, use, eval, cald/app, httpserver, cli/client, or CLI packages.
- `llm` -> config, model, store, logging, acquisition, proposal, use, contract, httpserver, cli, cald, or any `internal` workflow package.
- `acquisition` -> store, check, execute, observe/cli, proposal/surface, proposal/capability, proposal/binding, proposal/evidence, proposal/policy, llm, config, logging, contract/httpserver/cli/cald, run, use, eval.
- `contract` -> cald/app, httpserver, cli/client, cli, or workflow implementation packages.
- `cli/client` -> cald/app, httpserver, acquisition, run, use, proposal, store, config, llm, execute, or check.
- `httpserver` -> acquisition, run, use, proposal, store, config, llm, execute, or check.
- `cli` -> acquisition, run, use, proposal, probe, store, config, execute, or check.
- `proposal/*` -> acquisition, probe, promote, run, use, or adapter packages.
- `probe` -> store, acquisition, entry, observe, promote, tracelog, run, use, eval, llm, config, logging, contract/httpserver/cli/cald.
- `promote` -> store, proposal, probe, check, execute, acquisition, entry, observe, tracelog, run, use, eval, llm, config, logging, contract/httpserver/cli/cald.
- `tracelog` -> store, proposal, probe, promote, check, execute, acquisition, entry, observe, run, use, eval, llm, config, logging, contract/httpserver/cli/cald.
- `check` -> probe or run.
- `execute` -> probe or run.
- `run` -> acquisition, proposal, probe, promote, or tracelog.
- `use/select` -> run.

## Cross-Cutting Rule

The package that owns a dependency-owning struct should expose its constructor. Callers should not assemble non-trivial dependency graphs with composite literals.
