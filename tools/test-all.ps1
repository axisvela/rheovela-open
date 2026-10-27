#!/usr/bin/env pwsh
$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
Push-Location $root
try {
    Write-Host "==> Go root"
    go test ./... -count=1

    Write-Host "==> Go SDK"
    Push-Location sdk/go
    try { go test ./... -count=1 } finally { Pop-Location }

    Write-Host "==> Go Harmovela adapter"
    Push-Location sdk/go/harmovela
    try { go test ./... -count=1 } finally { Pop-Location }

    Write-Host "==> Python"
    python -m py_compile sdk/python/worker.py
    python -m unittest discover -s sdk/python -v

    Write-Host "==> TypeScript"
    node --test --experimental-strip-types sdk/typescript/worker.test.ts

    Write-Host "==> Java"
    Push-Location sdk/java
    try {
        javac -d out worker/Worker.java example/Example.java
        java -cp out example.Example
    } finally { Pop-Location }

    Write-Host "==> Rust"
    Push-Location sdk/rust
    try {
        cargo test --offline
        cargo run --offline
    } finally { Pop-Location }

    Write-Host "==> Designer"
    node --test designer/model.test.mjs designer/core.test.mjs

    Write-Host "All language checks passed."
} finally {
    Pop-Location
}
