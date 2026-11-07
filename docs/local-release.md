# Local Release (no CORE_REPO_TOKEN)

The GitHub-hosted `release` workflow checks out `axisvela/rheovela` because the
core repository is private, so it requires the `CORE_REPO_TOKEN` secret. Local
releases do not need that token: they build from the sibling core checkout.

## Prerequisites

- Sibling checkouts: `../Rheovela`, `../Rheovela-open` (this repo), and
  `../Harmovela` (core's replace directives).
- Go on `PATH`.
- Optional: Docker for the server image, `gh` for uploading assets.

## Usage

```powershell
# Windows
pwsh -File scripts/release-local.ps1 v1.63.0
pwsh -File scripts/release-local.ps1 v1.63.0 -Targets linux/amd64 -SkipDocker
pwsh -File scripts/release-local.ps1 v1.63.0 -Upload
```

```bash
# Linux/macOS
bash scripts/release-local.sh v1.63.0
UPLOAD=1 bash scripts/release-local.sh v1.63.0
```

Artifacts are written to `dist/`:

- `rheo-windows-amd64.exe`
- `rheo-linux-amd64`
- `rheo-linux-arm64`
- `rheo-darwin-amd64`
- `rheo-darwin-arm64`
- `SHA256SUMS`

The scripts build from `../Rheovela/backend` (module root) with the version
stamped via `-X main.version`. The bash script delegates to core's
`release/build.sh`; the PowerShell script runs the same matrix directly so it
works without bash.

Docker is built from `../Rheovela/release/Dockerfile` with context
`../Rheovela`; pushing is left to the operator.

## Hosted CI

`CORE_REPO_TOKEN` is still required by `.github/workflows/release.yml` because
the core repository is private. To remove the token:

- make `axisvela/rheovela` public and adjust the checkout, or
- run the release on a self-hosted runner with the sibling checkouts, or
- move core binary/image publishing into a private core workflow.

The hosted workflow builds core from `Rheovela/backend` (the module root).
