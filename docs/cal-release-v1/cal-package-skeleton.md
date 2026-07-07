# Package Skeleton

Release V1 uses package count deliberately. More packages are acceptable when each package owns a single responsibility and dependency direction stays clear.

```text
pkg/
  jsonfile/
    jsonfile.go

internal/
  model/
    provider.go
    capability.go
    execution.go
    verify.go
    evidence.go
    run.go
    trace.go
    proposal_trace.go
    progress.go
    error.go
    ids.go
    validate.go
    verify_level.go

  progress/
    progress.go

  store/
    store.go
    provider.go
    capability.go
    trace.go
    run.go
    record.go
    json.go

  config/
    config.go
    llm.go
    logging.go

  logging/
    logging.go
    rotate.go
    dir_darwin.go
    dir_linux.go
    dir_windows.go

  llm/
    client.go
    openai.go
    options.go
    errors.go

  acquisition/
    runner.go
    state.go
    request.go
    errors.go

  entry/
    registry.go
    request.go
    path.go
    path_darwin.go
    path_linux.go
    path_windows.go
    errors.go

  observe/
    contract.go
    request.go
    runner.go
    constants.go
    errors.go

    cli/
      observer.go
      usage.go
      usage_unix.go
      usage_windows.go

  proposal/
    runner.go
    pipeline.go
    request.go
    result.go
    contract.go
    select.go
    diagnostics.go
    options.go
    errors.go

    surface/
      runner.go
      prompt.go
      parse.go

    capability/
      runner.go
      prompt.go
      parse.go

    binding/
      runner.go
      prompt.go
      parse.go

    evidence/
      runner.go
      prompt.go
      parse.go

    replay/
      replay.go

    rules/
      rules.go

    policy/
      policy.go
      file.go

  probe/
    runner.go
    workdir.go
    fixture.go
    materialize.go
    classify.go

  execute/
    contract.go
    runner.go
    inputs.go

    cli/
      runner.go
      command.go

  check/
    checker.go
    request.go
    subject.go
    predicate.go
    predicate_file.go
    predicate_text.go
    predicate_bytes.go
    predicate_hash.go
    params.go

  promote/
    runner.go
    request.go
    errors.go

  tracelog/
    writer.go
    request.go
    errors.go

  run/
    runner.go
    request.go
    record.go
    outputs.go

    resolve/
      runner.go

  use/
    runner.go
    request.go
    record.go

    select/
      runner.go
      local.go
      llm.go

    plan/
      runner.go
      target.go

  eval/
    runner.go
    metrics.go
    records.go
    acquisition.go
    reuse.go

  contract/
    errors.go
    providers.go
    acquisition.go
    capabilities.go
    runs.go
    use.go
    eval.go
    daemon.go

  httpserver/
    router.go
    response.go
    sse.go
    provider_handler.go
    acquisition_handler.go
    capability_handler.go
    run_handler.go
    use_handler.go
    eval_handler.go
    daemon_handler.go

  cli/
    cli.go
    daemon.go
    providers.go
    acquisition.go
    capabilities.go
    runs.go
    use.go
    eval.go
    input.go
    render.go
    errors.go

    client/
      client.go
      stream.go
      request.go
      error.go

  cald/
    command.go

    app/
      app.go
      providers.go
      capabilities.go
      acquisition.go
      runs.go
      use.go
      eval.go
      mapping.go

    endpoint/
      endpoint.go

    daemon/
      daemon.go
```

## Package Responsibilities

`pkg/jsonfile` owns low-level atomic JSON file writing shared by foundation
packages. It must not import CAL business packages.

`model` owns CAL durable records, IDs, enum constants, record validation, and `VerifySpec` data shape. `check` owns `VerifySpec` subject, predicate, and param semantics.

`progress` owns context-scoped live progress fanout. It carries `model.ProgressEvent` from workflow runners to request-local observers such as HTTP SSE without importing transport packages.

`store` owns JSON persistence under `CAL_HOME`. It reads and writes durable records, but does not decide acquisition, run, use, or eval behavior.

`config` owns `config.json` schema, loading, defaults, writing, and validation. It does not resolve `CAL_HOME`, initialize logging, create LLM clients, resolve secrets during ordinary loading, or access store records.

`logging` owns process log setup, platform log directories, and rotation. It receives `config.LoggingConfig` from callers; it does not read `config.json` or resolve `CAL_HOME`.

`llm` owns the provider-neutral LLM client interface and OpenAI-compatible implementation. It does not know prompt semantics.

`acquisition` owns the main targeted acquisition runner:
`entry -> observe -> proposal -> probe -> promote -> tracelog`. It is an
orchestrator with stage-specific interfaces, not a generic stage framework.

`entry` owns explicit provider registration and loading. It is shared by provider registration commands and the acquisition Entry stage; it does not scan directories, observe provider capabilities, or produce candidates.

`observe` owns provider observation contracts and provider-kind dispatch. `observe/cli` owns CLI usage observation.

`proposal` owns candidate generation contracts, the four-stage proposal orchestrator, and per-capability binding/evidence concurrency. `proposal/surface`, `proposal/capability`, `proposal/binding`, and `proposal/evidence` own the internal four-stage LLM proposal flow. `proposal/replay`, `proposal/rules`, and `proposal/policy` own replay, deterministic baseline, and policy behavior. Probe materialization belongs to `probe`, not `proposal`.

`probe` owns acquisition verification orchestration for candidates: workdir, fixtures, execution, deterministic checks, and probe classification. It receives a work root from callers and does not import `store`.

`execute` owns execution contracts and input rendering. `execute/cli` owns CLI execution.

`check` owns deterministic `VerifySpec` check evaluation. It is shared by `probe` and `run`.

`promote` owns converting passed probes into durable capabilities and bindings. It defines a narrow local store interface and imports only `model`.

`tracelog` owns acquisition trace assembly and persistence. It defines a narrow
local store interface and imports only `model`.

`run` owns formal execution of promoted capabilities, including binding
resolution, provider execution, optional deterministic verification, and durable
run record persistence. `run/resolve` owns deterministic binding selection for a
known capability.

`use` owns intent-level reuse: select, plan inputs, and delegate to `run`. `use/select` owns local and optional LLM semantic selection. `use/plan` owns input completion.

`eval` owns read-only acquisition and reuse metrics.

`contract` owns request, response, and error DTOs shared by CLI, the CLI daemon client, HTTP, and cald app methods. Contract responses may embed or return `model` durable records directly, but request-only shapes, response wrappers, and transport errors belong in `contract`.

`httpserver` owns HTTP routing, request decoding, response encoding, SSE streaming adapters, and error mapping.

`cli/client` owns the CLI-facing local cald HTTP client. It reads endpoint metadata through `cald/endpoint`, sends local HTTP requests, decodes `contract` DTOs, parses SSE streams, and surfaces structured errors.

`cald/endpoint` is the small endpoint-file subpackage inside the `cald` module.
It owns the local endpoint metadata file shape, path policy, read, write, and
remove behavior. It is shared by `cald/daemon` and `cli/client` so daemon
lifecycle does not import CLI packages.

`cli` owns cobra command parsing, daemon-client calls, daemon lifecycle commands, stdout rendering, and CLI error formatting.

`cald/app` owns application service assembly and method dispatch. It is the composition root for store, config, LLM clients, proposal variants, observation drivers, execution drivers, and use-case runners. `cald/daemon` owns process lifecycle, local server startup, status, and endpoint publication.
