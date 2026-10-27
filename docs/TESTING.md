# Testing Guide

This page records the exact local test commands for every public language SDK and
the static designer. CI uses the same commands (see `.github/workflows/ci.yml`).

Last verified: 2026-10-27.

## Prerequisites

| Tool | Version used by CI |
|------|--------------------|
| Go | 1.25 (SDK/root); 1.26 (Harmovela adapter) |
| Python | 3.12 |
| Node.js | 22 |
| Java | 21 (Temurin) |
| Rust | stable |
| PostgreSQL | optional, for core/EE store tests |

## Language matrix

| Language | Command | CI job | Observed result |
|----------|---------|--------|-----------------|
| Go (root) | `go test ./... -count=1` | `test` | 3 packages ok |
| Go SDK | `cd sdk/go && go test ./... -count=1` | `sdk-go` | 2 packages ok |
| Go Harmovela adapter | `cd sdk/go/harmovela && go test ./... -count=1` | `harmovela-adapter` | 1 package ok |
| Python | `python -m py_compile sdk/python/worker.py` then `python -m unittest discover -s sdk/python -v` | `python` | 12 tests OK |
| TypeScript | `node --test --experimental-strip-types sdk/typescript/worker.test.ts` | `typescript` | 15 tests pass |
| Java | `javac -d out sdk/java/worker/Worker.java sdk/java/example/Example.java` then `java -cp out example.Example` | `java` | prints `ok` |
| Rust | `cd sdk/rust && cargo test --offline && cargo run --offline` | `rust` | 5 tests pass; `processed=2` |
| Designer | `node --test designer/model.test.mjs designer/core.test.mjs designer/history.test.mjs designer/diagram.test.mjs designer/diff.test.mjs` | `designer` | 29 tests pass |

## Run everything locally

Windows PowerShell:

```powershell
powershell -ExecutionPolicy Bypass -File tools/test-all.ps1
```

PowerShell 7+:

```powershell
pwsh -File tools/test-all.ps1
```

Bash:

```bash
./tools/test-all.sh
```

## Platform notes

- **Windows / PowerShell:** use `;` or `if ($?) { ... }` instead of `&&`.
- **TypeScript:** CI uses Node 22 with `--experimental-strip-types`. Newer Node
  versions may require `--experimental-transform-types`; record the exact command
  you used if it differs.
- **Java:** the example is a short-lived in-process contract check and exits
  after printing `ok`.
- **Rust:** `--offline` is used because the SDK has no external crates.
- **PostgreSQL:** core store tests gate on `RHEO_TEST_PG_DSN`; EE tests gate on
  `RHEO_EE_TEST_PG_DSN`. Use a dedicated empty database for the store contract
  suite (it creates/truncates tables).

## Cross-repo verification (optional)

```sh
# core
cd ../Rheovela && go test ./... -count=1

# EE (PostgreSQL-backed)
cd ../Rheovela-ee
RHEO_EE_TEST_PG_DSN='postgres://postgres:postgres@127.0.0.1:5433/postgres?sslmode=disable' \
  go test ./... -count=1
```
