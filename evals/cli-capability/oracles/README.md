# Benchmark Oracles

These scripts are benchmark scoring oracles. They are independent from CAL
runtime verification.

CAL verify specs decide whether a candidate can be promoted. These oracle
scripts decide whether the reported benchmark task succeeded on held-out inputs.
Each oracle reads one JSON object from stdin:

```json
{
  "inputs": {
    "source": "...",
    "target": "..."
  },
  "oracle": {}
}
```

Each script prints:

```json
{"passed": true, "evidence": {...}}
```

or:

```json
{"passed": false, "error": {"code": "...", "message": "..."}}
```
