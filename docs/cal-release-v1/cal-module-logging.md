# CAL Release V1 Logging

`logging/` owns process log setup, platform log paths, and rotating file writes.

It applies an already loaded `config.LoggingConfig`. It does not load
`config.json` and does not resolve `CAL_HOME`.

## Goal

`logging/` turns runtime logging inputs into an installed default `slog` handler:

```text
config.LoggingConfig + name + env + stderr
-> resolve effective level
-> resolve log path
-> create rotating writer when enabled
-> install slog default handler
-> Result
```

## Boundary

`logging/` owns:

- `slog` default handler setup.
- `CAL_LOG_LEVEL` runtime override.
- Platform log directory resolution.
- Log file path construction from a stable name.
- Rotating file writer.
- Effective logging setup result.

`logging/` does not own:

- Reading or writing `config.json`.
- Resolving `CAL_HOME`.
- Creating `config.File`.
- Logging policy defaults such as level or rotation size.
- Business log event decisions.
- Store, trace, API, CLI command, daemon, or LLM behavior.
- API keys or secret handling.

## Dependency Rule

```text
logging -> config
```

Besides `internal/config`, `logging` may use only the Go standard library.

Allowed callers:

```text
cli -> logging
cald/daemon -> logging
```

`cald/app` or caller code loads config first, then passes
`config.LoggingConfig` into `logging`.

## Files

```text
logging/
  logging.go
  rotate.go
  dir_darwin.go
  dir_linux.go
  dir_windows.go
```

## Public Shape

`logging.go` owns the public API:

```go
const EnvLevel = "CAL_LOG_LEVEL"

type Options struct {
	Name   string
	Config config.LoggingConfig
	Err    io.Writer
	Env    []string
}

type Result struct {
	Level       config.LoggingLevel
	FileEnabled bool
	FilePath    string
	ConfigError error
	FileError   error
}

func Configure(opts *Options) *Result
func Path(name string) (string, error)
```

`Name` is the stable logging subject name used for the log file. It is not an
OS process id.

Examples:

```go
logging.Path("calctl")
logging.Path("cald")
```

## Defaults

Logging policy defaults live in `config`, not `logging`:

```text
logging.level = info
logging.file.enabled = true
logging.file.max_bytes = 10485760
logging.file.max_files = 5
```

`logging` owns only runtime fallbacks:

```text
Name empty -> cal
Err nil    -> io.Discard
Env nil    -> os.Environ()
```

Do not define duplicate default level, max bytes, or max files in `logging`.

## Env Override

`CAL_LOG_LEVEL` may temporarily override `config.LoggingConfig.Level`.

Supported values:

```text
debug
info
warn
error
```

Invalid override behavior:

- Effective level falls back to `error`.
- `Result.ConfigError` records the validation error.

## Path Behavior

`Path(name)` returns the log file path. It does not create a logger or write
data.

Name handling:

```text
filepath.Base(name)
empty, ".", or path separator -> cal
append ".log"
```

Platform directories:

```text
darwin  -> ~/Library/Logs/cal
linux   -> $XDG_STATE_HOME/cal/logs or ~/.local/state/cal/logs
windows -> %LocalAppData%/cal/logs or UserCacheDir/cal/logs
```

## Rotation

`rotate.go` owns the rotating writer:

```go
type rotatingWriter struct {
	path     string
	maxBytes int64
	maxFiles int
	mu       sync.Mutex
}

func newRotatingWriter(path string, maxBytes int64, maxFiles int) (*rotatingWriter, error)
func (writer *rotatingWriter) Write(p []byte) (int, error)
```

Behavior:

```text
before each write, check current size + incoming bytes
if over max_bytes, rotate files
keep max_files historical files
delete the oldest file when needed
```

`config` validates that `max_bytes` and `max_files` are positive after defaults.
`logging` should still fail cleanly when it receives unusable values.

## Constants

Own these constants in `logging`:

```go
const (
	EnvLevel    = "CAL_LOG_LEVEL"
	defaultName = "cal"
	logExt      = ".log"
)
```

Platform env constants:

```go
const envXDGStateHome = "XDG_STATE_HOME"
const envLocalAppData = "LocalAppData"
```

Do not constantize error messages.

## Tests

Required tests:

- `Configure` uses the passed config level.
- `CAL_LOG_LEVEL` overrides config level.
- Invalid `CAL_LOG_LEVEL` falls back to error and records `ConfigError`.
- `file.enabled=false` writes only to `Err`.
- `file.enabled=true` creates and writes through a rotating writer.
- `Path` sanitizes names.
- Platform log directory behavior is covered by platform-specific tests.
- Rotation rolls files and deletes the oldest file.
- `logging` does not read `config.json`.

## Decision

Use `Name` and `Path`, not `Process` and `ProcessLogPath`.

Rationale:

- `Name` describes the stable logging subject and avoids OS-process confusion.
- `Path` is short and clear inside the `logging` package.
- Config loading stays in the caller, keeping `logging` focused on applying an
  already loaded logging policy.
