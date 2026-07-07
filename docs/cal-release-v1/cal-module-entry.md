# CAL Release V1 Entry

`entry/` owns explicit provider registration and loading.

It turns one user-provided `provider_path` into a durable `model.Provider`, and
it loads stored providers for acquisition. It does not scan directories,
observe provider behavior, execute providers, infer capabilities, or write
traces.

## Goal

`entry/` answers three narrow questions:

```text
provider_path -> model.Provider
provider_path -> stored Provider lookup
provider_id   -> stored Provider load
```

The first V1 implementation supports explicit provider paths only. It does not
introduce `entry_paths`, recursive directory scanning, default platform search
paths, or automatic provider discovery.

## Boundary

`entry/` owns:

- Validating `provider_path` and `provider_id` request fields.
- Expanding `$HOME` and environment variables in a provider path.
- Normalizing provider paths with absolute clean paths.
- Detecting whether a path is a supported provider entry.
- Constructing stable `model.Provider` records.
- Registering a provider record in store.
- Listing stored provider records.
- Loading a provider by id for acquisition.
- Looking up a stored provider by normalized path.
- Returning created or updated registration results.

`entry/` does not own:

- Directory scanning or recursive provider discovery.
- Default platform search paths.
- Provider observation. That belongs to `observe`.
- CLI help, manpage, or documentation capture. That belongs to `observe/cli`.
- Provider execution. That belongs to `execute`.
- Candidate generation. That belongs to `proposal`.
- Probe execution or deterministic verification. Those belong to `probe` and
  `check`.
- Capability promotion. That belongs to `promote`.
- Acquisition trace writing. That belongs to `tracelog`.
- LLM calls.
- Config loading.
- Logging setup.
- CLI, HTTP, CLI daemon client, daemon, or DTO rendering.

## Dependency Rule

```text
entry -> model
store -> model
cald/app -> entry, store
```

`entry` declares the narrow provider storage interface it consumes. It should
not import the concrete `store` package.

Forbidden:

```text
entry -> store
entry -> observe
entry -> observe/cli
entry -> proposal
entry -> probe
entry -> promote
entry -> tracelog
entry -> acquisition
entry -> execute
entry -> check
entry -> llm
entry -> config
entry -> logging
entry -> contract/httpserver/cli/cald
```

The dependency map may describe the logical composition as `entry -> model,
store`, but the code should keep the concrete dependency outside `entry`.
`cald/app` wires `store.Store` into `entry.Registry`.

## Callers

```text
cald/app -> entry
acquisition -> entry
httpserver -> cald/app -> entry
cli -> cli/client -> httpserver -> cald/app -> entry
```

Provider registration commands call `entry.Registry.Register`.

Acquisition calls `entry.Registry.Load` with a provider id before observation
starts.

CLI and HTTP code should not normalize provider paths or compute provider ids
directly. They should pass request DTO fields to the app layer.

## Files

```text
entry/
  registry.go
  request.go
  path.go
  path_darwin.go
  path_linux.go
  path_windows.go
  errors.go
```

`registry.go` owns `Registry`, `ProviderStore`, and provider persistence
methods.

`request.go` owns request and result structs.

`path.go` owns provider path normalization, provider construction, provider
name derivation, and shared path helpers.

`path_*.go` files own platform-specific checks for app bundles and executable
provider entries.

`errors.go` owns entry error codes and coded errors.

Do not add `scanner.go`, `ScanRequest`, `ScanResult`, `EntryOptions`, or
`EntryPaths` in the first V1 implementation.

## Public Shape

`request.go` owns:

```go
type RegisterRequest struct {
	ProviderPath string
}

type RegisterResult struct {
	Provider model.Provider
	Created  bool
	Updated  bool
}

type LoadRequest struct {
	ProviderID string
}

type LoadResult struct {
	Provider model.Provider
}
```

`registry.go` owns:

```go
type ProviderStore interface {
	ListProviders() ([]model.Provider, error)
	GetProvider(id string) (model.Provider, bool, error)
	SaveProvider(provider *model.Provider) error
}

type Registry struct {
	store ProviderStore
}

func NewRegistry(store ProviderStore) *Registry

func (registry *Registry) Register(ctx context.Context, req *RegisterRequest) (*RegisterResult, error)
func (registry *Registry) Load(ctx context.Context, req *LoadRequest) (*LoadResult, error)
func (registry *Registry) List(ctx context.Context) ([]model.Provider, error)
func (registry *Registry) GetByPath(ctx context.Context, providerPath string) (model.Provider, bool, error)
```

The interface is named `ProviderStore` instead of `Store` to avoid confusion
with the concrete `store.Store` package.

Use pointer parameters and pointer return values for request and result structs
because registry operations are non-trivial workflow values.

## Provider Path Resolution

`path.go` should expose only package-local helpers unless another package has a
real need for direct provider-path resolution.

Suggested internal shape:

```go
func resolveProviderPath(providerPath string) (model.Provider, error)
func normalizeProviderPath(providerPath string) (string, error)
func providerName(path string) string
func providerID(kind model.ProviderKind, path string) string
```

Resolution rules:

- Empty or whitespace-only path returns `invalid_provider_path`.
- `$HOME` and environment variables are expanded.
- The path is normalized with absolute clean path semantics.
- A supported executable file becomes `model.ProviderKindCLI`.
- A supported `.app` bundle becomes `model.ProviderKindApp`.
- Any other path returns `target_provider_not_found`.

Do not scan child entries when the input path is a normal directory. A normal
directory is not a provider.

## Provider ID

Provider id generation should use the `model` package's stable id helper. If
the helper does not exist yet, add it to `model`; do not invent a separate id
algorithm in `entry`.

The id should be stable for:

```text
platform + provider kind + normalized absolute provider path
```

`entry` may derive `Provider.Name` from the path basename:

- `/usr/local/bin/ffmpeg` -> `ffmpeg`
- `/Applications/Preview.app` -> `Preview`

`Provider.Version` stays empty in V1. Version probing would require executing
or observing the provider, which belongs outside `entry`.

## Register Flow

```text
1. Validate request.
2. Resolve provider_path into model.Provider.
3. List existing providers.
4. If provider id already exists, mark Updated=true.
5. Otherwise mark Created=true.
6. Save provider.
7. Return RegisterResult.
```

Store write failures return an error. `entry` does not write partial diagnostic
records.

## Load Flow

```text
1. Validate provider_id.
2. Call ProviderStore.GetProvider.
3. If missing, return provider_not_found.
4. Return LoadResult.
```

Acquisition should use `Load` before `observe`.

## GetByPath Flow

```text
1. Validate and normalize provider_path.
2. List providers.
3. Return the provider whose stored Path equals the normalized path.
4. Return ok=false when no stored provider matches.
```

`GetByPath` should not require the provider path to still exist on disk. It is
a stored-record lookup by normalized path, not a fresh registration attempt.

## Error Codes

`errors.go` should define semantic constants:

```go
const (
	CodeInvalidProviderPath    = "invalid_provider_path"
	CodeTargetProviderNotFound = "target_provider_not_found"
	CodeProviderNotFound       = "provider_not_found"
	CodeEntryStoreFailed       = "entry_store_failed"
)
```

Do not define `ambiguous_target_provider` or `entry_scan_failed` in V1 because
there is no directory scanning and one explicit provider path must resolve to at
most one provider.

## Platform Rules

Darwin:

- `.app` directory is `model.ProviderKindApp`.
- executable regular file is `model.ProviderKindCLI`.

Linux:

- executable regular file is `model.ProviderKindCLI`.
- app bundle support is not required.

Windows:

- executable file extensions such as `.exe`, `.bat`, `.cmd`, and `.ps1` may be
  treated as `model.ProviderKindCLI`.
- app bundle support is not required.

Keep platform checks local to `entry`. Do not let `model` know filesystem or OS
rules.

## Tests

Add direct unit tests for:

- registering an executable provider creates a provider record;
- registering the same provider again returns `Updated=true`;
- empty provider path returns `invalid_provider_path`;
- unsupported path returns `target_provider_not_found`;
- `Load` returns a provider by id;
- missing provider id returns `provider_not_found`;
- `List` passes through stored providers;
- `GetByPath` matches normalized stored paths;
- store list, get, and save errors propagate with `entry_store_failed`;
- context cancellation stops before store or filesystem work when possible.

Use temporary executable files and platform-specific tests where needed. Keep
test fixtures small and avoid scanning real system directories.
