#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

echo "==> Go root"
go test ./... -count=1

echo "==> Go SDK"
(cd sdk/go && go test ./... -count=1)

echo "==> Go Harmovela adapter"
(cd sdk/go/harmovela && go test ./... -count=1)

echo "==> Python"
python -m py_compile sdk/python/worker.py
python -m unittest discover -s sdk/python -v

echo "==> TypeScript"
node --test --experimental-strip-types sdk/typescript/worker.test.ts

echo "==> Java"
(cd sdk/java && javac -d out worker/Worker.java example/Example.java && java -cp out example.Example)

echo "==> Rust"
(cd sdk/rust && cargo test --offline && cargo run --offline)

echo "==> Designer"
node --test designer/model.test.mjs designer/core.test.mjs designer/history.test.mjs designer/diagram.test.mjs designer/diff.test.mjs

echo "All language checks passed."
