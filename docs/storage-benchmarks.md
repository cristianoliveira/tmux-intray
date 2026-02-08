# Storage Benchmarks

Use storage benchmarks to compare TSV and SQLite backends on common operations.

## Run

```bash
make benchmarks
```

This runs benchmark scenarios in `internal/storage/benchmarks_test.go` with `-benchmem` and `-benchtime=1x` so each scenario executes once with fixed workload sizes.

## Scenarios

- Add 1,000 notifications sequentially
- List all active notifications from a 10,000 row dataset
- Filter by state, level, and session on a 10,000 row dataset
- Mark 100 notifications as read
- Dismiss 100 notifications
- Cleanup old notifications from a 10,000 dismissed row dataset
- Concurrent add workload (10 goroutines x 100 operations)
- Query a large dataset (100,000 rows)

## Interpreting results

- Compare `ns/op` for raw latency and `B/op` plus `allocs/op` for memory behavior.
- Benchmark setup (fixture generation) is excluded from measured timings.
- Use `benchmarks/benchmarks.txt` as a baseline snapshot and compare new runs against it after storage changes.
- Expect system-level variance between machines; only compare runs from similar environments when possible.
