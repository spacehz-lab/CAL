# CAL Release V1 Store

`store/` owns local JSON persistence under one CAL home directory.

It is a foundation package. It stores durable `model` records and nothing else.

## Goal

`store/` turns one caller-provided home directory into a small record store:

```text
CAL_HOME/
  providers/
    <provider-id>.json
  capabilities/
    <capability-id>.json
  traces/
    <trace-id>/
      trace.json
  runs/
    <run-id>.json
```

Bindings stay embedded inside `Capability` records. Do not add a top-level
`bindings/` directory in the first V1 pass.

`traces/` stores trace records and trace-scoped artifacts.

## Boundary

`store/` owns:

- Store construction from an explicit home path.
- Fixed durable record directories.
- Directory creation.
- Record path safety.
- JSON decode and encode.
- Atomic JSON writes.
- Stable sorted list operations.
- Record validation before returning or saving records.

`store/` does not own:

- `CAL_HOME` resolution from environment or platform defaults.
- `config.json` loading.
- Logging setup.
- LLM client construction.
- Acquisition, proposal, probe, promotion, run, use, or eval behavior.
- Probe work directory policy.
- Binding selection.
- Data migration or version negotiation.

Probe work directories may live under `traces/<trace-id>/`, but the policy for
creating and cleaning them belongs to `probe/`, not `store/`.

## Dependency Rule

```text
store -> model
store -> pkg/jsonfile
store -> standard library
```

Forbidden:

```text
store -> config
store -> logging
store -> llm
store -> acquisition
store -> proposal
store -> probe
store -> promote
store -> run
store -> use
store -> eval
store -> contract/httpserver/cli/cald
```

`pkg/jsonfile` is allowed because it owns only low-level JSON file writing and
does not import CAL business packages.

## Files

```text
store/
  store.go
  provider.go
  capability.go
  trace.go
  run.go
  record.go
  json.go
```

Do not add subpackages for V1.

## Public Shape

`store.go` owns store construction and directory setup:

```go
type Store struct {
	root string
}

func New(root string) (*Store, error)
func (store *Store) Root() string
func (store *Store) Ensure() error
```

Record APIs should stay explicit, not generic:

```go
func (store *Store) ListProviders() ([]model.Provider, error)
func (store *Store) GetProvider(id string) (model.Provider, bool, error)
func (store *Store) SaveProvider(provider *model.Provider) error

func (store *Store) ListCapabilities() ([]model.Capability, error)
func (store *Store) GetCapability(id string) (model.Capability, bool, error)
func (store *Store) SaveCapability(capability *model.Capability) error

func (store *Store) ListTraces() ([]model.Trace, error)
func (store *Store) GetTrace(id string) (model.Trace, bool, error)
func (store *Store) SaveTrace(trace *model.Trace) error

func (store *Store) ListRuns() ([]model.Run, error)
func (store *Store) GetRun(id string) (model.Run, bool, error)
func (store *Store) SaveRun(run *model.Run) error
```

Use pointer parameters for `Save*` methods because records can be large and are
mutable workflow outputs. Return values may stay values because callers receive
independent decoded records.

Use `Save`, not `Put`, for the public API. `Save` means create or replace one
durable record.

## File Behavior

`Ensure` creates:

```text
providers/
capabilities/
traces/
runs/
```

`List*` behavior:

- Missing directory returns an empty slice.
- Non-JSON files are ignored.
- Subdirectories are ignored except `ListTraces`, which reads
  `traces/<trace-id>/trace.json`.
- Returned records are sorted by `ID`.
- Invalid JSON returns an error.
- Invalid records return an error with file context.

`Get*` behavior:

- Missing record returns `(zero, false, nil)`.
- Invalid record id returns an error.
- Invalid JSON or invalid record returns an error.

`Save*` behavior:

- Nil record pointers return an error.
- Record validation runs before writing.
- Record id must be path-safe.
- Parent directories are created as needed.
- Writes are atomic within the target directory.
- JSON is indented for inspectability.

## Record ID Safety

Record ids must be safe as filename or path segment input:

- non-empty after trimming
- not `.`
- not `..`
- must not contain `/`
- must not contain `\`

Capability ids such as `document.convert` remain valid filenames.

## Constants

Store-owned directory and filename strings are persisted layout contract values.
Define them as constants in `store/`:

```go
const (
	providersDir    = "providers"
	capabilitiesDir = "capabilities"
	tracesDir       = "traces"
	runsDir         = "runs"
	traceFileName   = "trace.json"
	jsonExt         = ".json"
)
```

Keep these constants private unless another package has a real need to reference
the layout contract directly.

## Error Policy

`store/` should return ordinary Go errors with enough context for callers to
wrap or render them.

It should not create API error DTOs, structured CLI output, or log messages.
Those belong to adapter and composition packages.
