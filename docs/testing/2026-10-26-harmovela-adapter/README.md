# Harmovela Producer Adapter

Date: 2026-10-26. Tracked locally under proc project 139. Unreleased after v1.7.4.

## Changes

- New `sdk/go/harmovela` module publishes Rheovela work-item lifecycle as
  Harmovela `task.*` events through an injected Publisher.
- Session open/ready/closed events; tenant/workspace envelope fields.
- Core keeps the consumer direction; built-in transports are out of scope.

## Verification

| Command | Result |
| --- | --- |
| `go test ./...` in `sdk/go/harmovela` | 1 package ok, exit 0 |
| `go test ./...` in `sdk/go` | 2 packages ok, exit 0 |
| `go test ./...` in repo root | 3 packages ok, exit 0 |
| `go vet ./...` in `sdk/go/harmovela` | exit 0, no findings |

## Notes

The adapter depends on the published Harmovela `v3.0.0` module; no local replace
is required. Callers provide the transport (WebSocket/gRPC/NATS/etc.).
