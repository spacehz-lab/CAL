# CAL Discovery Entry

Discovery Entry is the narrow provider registration step before targeted
Discovery.

It turns one explicit provider entry path into one stored `Provider` record.
Entry only proves that a provider entry exists. It does not infer capabilities,
propose bindings, run probes, or verify workflows.

## Input

Entry receives one `provider_path` from the caller.

```text
provider_path string
```

The path must resolve to one supported provider entry, such as a CLI executable
or an app bundle. Directory-wide scanning is not part of the current public
contract.

## Handling

Entry handling:

```text
provider_path
-> trim and expand environment variables
-> resolve to an absolute clean path
-> inspect the explicit entry
-> normalize provider entry facts
-> create or update Provider
```

Path inspection is platform-specific. The Entry contract is only that a
supported explicit provider entry becomes a `Provider` record.

## Output

Entry outputs:

```text
Provider
```

Persisted shape:

```text
CAL_HOME/
  providers/
    <provider-id>.json
```

If Entry runs as part of a future larger discovery attempt, its process details
may also be written into that attempt's `Trace`. In the current command slice,
`calctl providers add --provider-path <path>` writes only the Provider record.

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

`Provider` is an entry fact. It is not a product family, a capability, a
binding, or proof that a workflow works.

Two different entries can remain two different Providers, even when they belong
to the same product.
