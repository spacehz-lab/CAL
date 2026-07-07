# CAL Release V1 Config

`config/` owns the local `config.json` schema, defaults, file read/write, and
validation.

It is a foundation package. It must stay small and must not become a service
locator for runtime dependencies.

## Goal

`config/` turns one CAL home directory into a validated configuration value:

```text
CAL_HOME/config.json
-> load JSON
-> apply defaults
-> validate config contract
-> Config
```

If the file is missing, callers may use `Ensure` to write the default
configuration.

## Boundary

`config/` owns:

- `config.json` filename and on-disk schema.
- Config file read/write.
- Default values.
- Field validation and normalization.
- Non-secret durable LLM settings.
- Durable logging settings.

`config/` does not own:

- `CAL_HOME` resolution. Callers pass the home path in.
- Store paths or durable record layout.
- Logger initialization, file opening, or rotation.
- LLM client construction.
- Prompt construction.
- API key lookup as normal config loading behavior.
- Daemon endpoint files or process lifecycle.

## Dependency Rule

```text
config -> pkg/jsonfile
config -> standard library
```

Allowed callers:

```text
logging -> config
cald/app -> config
```

`llm` should not need to import `config` in the first V1 pass. Prefer this
direction:

```text
cald/app -> config
cald/app -> llm
```

`cald/app` reads config, resolves runtime overrides and secrets, then passes
plain options into `llm`.

## Files

```text
config/
  config.go
  llm.go
  logging.go
```

Do not add subpackages for V1.

## Public Shape

`config.go` owns the top-level file API:

```go
const FileName = "config.json"

type File struct {
	path string
}

type Config struct {
	LLM     *LLMConfig    `json:"llm,omitempty"`
	Logging LoggingConfig `json:"logging,omitempty"`
}

func NewFile(home string) *File
func (file *File) Path() string
func (file *File) Load() (*Config, error)
func (file *File) Save(cfg *Config) error
func (file *File) Ensure() (*Config, error)

func Default() *Config
func (cfg *Config) WithDefaults() *Config
func (cfg *Config) Validate() error
```

Use pointer receivers for `File` and `Config` methods because they represent
non-trivial file/config values. `Default` returns a fresh config value so callers
cannot accidentally share mutable nested fields.

`llm.go` owns the durable LLM config section:

```go
type LLMConfig struct {
	API       LLMAPI `json:"api,omitempty"`
	BaseURL   string `json:"base_url,omitempty"`
	Model     string `json:"model,omitempty"`
	APIKeyRef string `json:"api_key_ref,omitempty"`
}

type LLMAPI string

const (
	LLMAPIResponses       LLMAPI = "responses"
	LLMAPIChatCompletions LLMAPI = "chat_completions"
)
```

`logging.go` owns the durable logging config section:

```go
type LoggingConfig struct {
	Level LoggingLevel      `json:"level,omitempty"`
	File  LoggingFileConfig `json:"file,omitempty"`
}

type LoggingFileConfig struct {
	Enabled  *bool `json:"enabled,omitempty"`
	MaxBytes int64 `json:"max_bytes,omitempty"`
	MaxFiles int   `json:"max_files,omitempty"`
}

type LoggingLevel string

const (
	LoggingLevelDebug LoggingLevel = "debug"
	LoggingLevelInfo  LoggingLevel = "info"
	LoggingLevelWarn  LoggingLevel = "warn"
	LoggingLevelError LoggingLevel = "error"
)
```

## Defaults

Initial defaults:

```text
logging.level = info
logging.file.enabled = true
logging.file.max_bytes = 10485760
logging.file.max_files = 5
```

LLM defaults should be empty in `config.json`. V1 should not invent a provider
or model when the user has not configured one.

## Validation

Top-level validation:

- Unknown fields should be rejected when loading `config.json`.
- Missing sections are allowed and receive defaults.
- Invalid JSON returns a structured error with file context.

LLM validation:

- Empty LLM config is valid.
- `api` must be empty, `responses`, or `chat_completions`.
- `base_url`, `model`, and `api_key_ref` are trimmed.
- `api_key_ref` may name a secret source, but `config` must not resolve the
  secret during ordinary load.
- Raw API keys must not be written into `config.json`.

Logging validation:

- `level` must normalize to `debug`, `info`, `warn`, or `error`.
- `warning` may be accepted and normalized to `warn`.
- `max_bytes` must be positive after defaults; zero means omitted and receives the default.
- `max_files` must be positive after defaults; zero means omitted and receives the default.
- `enabled` defaults to true when omitted.

## Secret Boundary

`config.json` may contain references to secrets, not secrets themselves.

Allowed:

```json
{
  "llm": {
    "api_key_ref": "env:OPENAI_API_KEY"
  }
}
```

Not allowed:

```json
{
  "llm": {
    "api_key": "sk-..."
  }
}
```

Runtime secret resolution belongs outside `config`, in the composition layer
that constructs the LLM client.

`CAL_LLM_*` environment variables are runtime overrides, not `config.json`
schema fields. They are read by `cald/app` when constructing runtime
`llm.Options`:

```text
CAL_LLM_API
CAL_LLM_BASE_URL
CAL_LLM_MODEL
CAL_LLM_API_KEY
```

`config` must not read these variables and must not write their values into
`config.json`.

## File Behavior

`Load`:

- Returns default config when `config.json` is missing.
- Does not create files.
- Applies defaults.
- Validates the result.

`Ensure`:

- Writes default config only when `config.json` is missing.
- Loads existing config when present.
- Creates the parent directory when needed.

`Save`:

- Applies defaults and validates before writing.
- Writes deterministic pretty JSON.
- Uses atomic write behavior.

## Tests

Required tests:

- missing config loads defaults without writing a file;
- `Ensure` creates `config.json`;
- `Save` and `Load` round-trip;
- unknown fields are rejected;
- logging defaults are applied;
- logging level normalizes `warning` to `warn`;
- invalid logging level fails;
- negative logging rotation values fail;
- empty LLM config is valid;
- supported LLM APIs pass;
- unsupported LLM API fails;
- raw API key fields are rejected by unknown-field handling;
- API key refs stay unresolved by `Load`.

## Decision

Keep `config/` at three files for V1:

```text
config.go
llm.go
logging.go
```

Do not split `home`, `env`, `secret`, or provider-specific files into this
package. Those would blur ownership and make `config` responsible for runtime
composition instead of durable configuration.
