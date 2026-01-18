# Performance

Tenet is designed for real-time validation with sub-millisecond latency.

## Benchmarks

Tested on Apple M4 Pro (14 cores, 24GB RAM):

### Run (Execution)

| Benchmark | Throughput | Latency | Memory |
|-----------|------------|---------|--------|
| Typical schema (6 defs, 3 rules) | 40,000/sec | 25 µs | 24 KB |
| Parallel (14 cores) | 69,000/sec | 14 µs | 24 KB |
| Large schema (100 defs, 50 rules) | 5,400/sec | 186 µs | 199 KB |

### Verify (Audit)

| Benchmark | Throughput | Latency | Memory |
|-----------|------------|---------|--------|
| Typical schema | 6,700/sec | 150 µs | 127 KB |
| Parallel (14 cores) | 11,800/sec | 85 µs | 130 KB |

> **Note:** Verify is ~6x slower than Run because it replays the user journey step-by-step (turn-based verification).

## Run Benchmarks Yourself

```bash
go test -bench=. -benchtime=3s -benchmem ./pkg/tenet
```

## What This Means

| Use Case | Required Latency | Tenet Run | Tenet Verify | Verdict |
|----------|------------------|-----------|--------------|---------|
| Form validation | <100ms | 0.025ms | — | ✅ |
| Real-time UI | <16ms (60 FPS) | 0.025ms | — | ✅ |
| API batch processing | 1000/sec | 40,000/sec | 6,700/sec | ✅ |
| Compliance audit | <1sec | 0.025ms | 0.15ms | ✅ |

## Static Linter

The linter is a **development-time tool** for schema authors. It performs static analysis *without* executing the schema.

```bash
tenet lint -file schema.json
```

**What it checks:**

| Check | Type | Description |
|-------|------|-------------|
| Undefined variables | Error | `{"var": "x"}` where `x` not in definitions |
| Potential cycles | Warning | Multiple rules setting the same field |
| Missing types | Warning | Definitions without `type` specified |
| Empty attestations | Warning | Attestations without `statement` |
| Temporal versions | Warning | Branches without `logic_version` |

**Use the linter for:**
- Pre-commit validation of schema files
- CI/CD pipeline checks
- Catching typos in variable names
- Identifying conflicting rules early

**Note:** Runtime cycle detection is also built into `Run()` — it warns when different rules set the same field during execution.

## Performance Considerations

1. **JSON parsing** — Each `Run()` parses JSON. For hot paths, consider caching parsed schemas.

2. **Memory allocations** — ~300 allocations per run. Acceptable for validation, not for per-frame game loops.

3. **WASM overhead** — Browser WASM is ~2-3x slower than native Go. Still sub-millisecond.

4. **Cycle detection** — Adds ~2 allocations per `set` operation. Negligible overhead.

## Best Practices

- **Batch validation**: Run once after all field changes, not after each keystroke
- **Debounce in UI**: Wait 100-200ms after user stops typing before calling `Run()`
- **Cache schemas**: Parse once, modify values, re-run
- **Lint in CI**: Run `tenet lint` in your CI pipeline to catch issues early
