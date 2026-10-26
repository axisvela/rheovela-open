# Open SDK v1.4.0 Parity

Date: 2026-10-26. Tracked locally under proc project 138.
Status: unreleased after v1.7.4; will ship in the next coordinated release.

## Changes

- Python/TypeScript/Java/Rust clients expose get/list/claim/heartbeat/
  complete-with-result/request-approval/approve/reject/cancel/fail.
- Work items expose result/checkpoint/effect/attempts/max/last-error/approval.
- Workers take a rich WorkResult (success/failure/approval).
- Java/Rust stay dependency-free and carry arbitrary JSON as raw text.
- CI runs Go root, Go SDK, Python, TypeScript, Java, Rust.

## Verification

| Command | Result |
| --- | --- |
| `go test ./...` | 3 packages ok |
| `go test ./...` in `sdk/go` | 2 packages ok |
| `python -m py_compile sdk/python/worker.py` + unittest | py_compile exit 0; 12 tests OK |
| `node --test --experimental-strip-types sdk/typescript/worker.test.ts` | 15 tests pass, 0 fail |
| `javac ...` + `java example.Example` | javac exit 0; example printed `ok`, exit 0 |
| `cargo test --offline` + `cargo run --offline` | 5 tests passed; example printed `processed=2` / `ok` |

Local tooling: Go 1.25.0, Python 3.14.6, Node v24.18.0 (CI targets Node 22),
Temurin JDK 21.0.8, and the locally installed Rust toolchain.

## Notes

Worker callbacks are intentionally breaking for this feature line. Raw result
and final_args JSON must be valid when supplied to Java/Rust.

On this Windows host, foreground `java.exe` startup can hang even for
`java -version`. The Java example was verified with `Start-Process` and
redirected stdio; it exited 0 and printed `ok`.
