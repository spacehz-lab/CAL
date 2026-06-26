# CAL Discovery Entry

Discovery Entry is the first step of Discovery.

It finds local provider entries from configured provider sources and creates or updates `Provider` records.

Entry only proves that a provider entry exists. It does not infer capabilities, propose bindings, run probes, or verify workflows.

## Input

Entry reads `provider_sources` from `CAL_HOME/config.json`.

```text
Config
  provider_sources []ProviderSource

ProviderSource
  kind path
  value string
```

Default `darwin` config:

```json
{
  "provider_sources": [
    {"kind": "path", "value": "PATH"},
    {"kind": "path", "value": "/Applications"},
    {"kind": "path", "value": "/System/Applications"},
    {"kind": "path", "value": "$HOME/Applications"}
  ]
}
```

Default `linux` config:

```json
{
  "provider_sources": [
    {"kind": "path", "value": "PATH"},
    {"kind": "path", "value": "$HOME/.local/bin"},
    {"kind": "path", "value": "/usr/local/bin"},
    {"kind": "path", "value": "/usr/bin"},
    {"kind": "path", "value": "/bin"},
    {"kind": "path", "value": "/snap/bin"}
  ]
}
```

Default `windows` config:

```json
{
  "provider_sources": [
    {"kind": "path", "value": "PATH"},
    {"kind": "path", "value": "%ProgramFiles%"},
    {"kind": "path", "value": "%ProgramFiles(x86)%"},
    {"kind": "path", "value": "%LocalAppData%\\Programs"}
  ]
}
```

`kind: "path"` is the only supported v0 provider source kind. `PATH` means the
current process `PATH`.

Other values are directory paths. They may use `$HOME` or environment variables.

## Handling

Entry handling:

```text
open CAL_HOME
-> ensure CAL_HOME exists
-> ensure config.json exists
   -> if missing: write platform default config
   -> if present: load user config
-> read provider_sources
-> expand configured paths
   -> PATH expands to the current process PATH directories
   -> other paths expand $HOME and environment variables
-> inspect each directory
-> identify provider entries
-> normalize provider entry facts
-> create or update Provider records
```

Path inspection is platform-specific. The Entry contract is only that supported provider entries found under `provider_sources` become `Provider` records.

## Output

Entry outputs:

```text
[]Provider
```

Persisted shape:

```text
CAL_HOME/
  providers/
    <provider-id>.json
```

If Entry runs as part of a full Discovery attempt, its process details may also be written into that attempt's `Trace`.

## Data Structure

Provider:

```text
Provider
  id
  name
  kind
  path
  version optional
```

Fields:

```text
id
  Deterministic local entry id.

name
  Human-readable provider entry name.

kind
  Provider entry kind.

path
  Absolute clean path to the provider entry.

version optional
  Provider version when it can be read as an entry fact.
```

Provider kinds:

```text
cli
app
```

Provider id:

```text
provider_<short_hash(platform|kind|absolute_clean_path)>
```

`Provider` is an entry fact. It is not a product family, a capability, a binding, or proof that a workflow works.

Two different entries can remain two different Providers, even when they belong to the same product.
