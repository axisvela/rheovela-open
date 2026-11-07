#!/usr/bin/env bash
# Build the core rheo release matrix from a sibling core checkout, without
# CORE_REPO_TOKEN. Run from anywhere; pass the semantic release tag.
set -euo pipefail

VERSION="${1:?usage: release-local.sh v1.63.0 [core-path]}"
CORE="${2:-$(cd "$(dirname "$0")/../../Rheovela" && pwd)}"
OUT="$(cd "$(dirname "$0")/.." && pwd)/dist"

if [[ ! "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+ ]]; then
  echo "version must look like v1.63.0 (got '$VERSION')" >&2
  exit 1
fi
if [[ ! -f "$CORE/backend/go.mod" ]]; then
  echo "core backend module not found at $CORE/backend" >&2
  exit 1
fi

RHEO_VERSION="$VERSION" "$CORE/release/build.sh"

mkdir -p "$OUT"
targets=(
  rheo-windows-amd64.exe
  rheo-linux-amd64
  rheo-linux-arm64
  rheo-darwin-amd64
  rheo-darwin-arm64
)
for name in "${targets[@]}"; do
  cp "$CORE/dist/$name" "$OUT/"
done
cp "$CORE/dist/SHA256SUMS" "$OUT/SHA256SUMS"
echo "==> release artifacts ready in $OUT"

if command -v docker >/dev/null 2>&1; then
  echo "==> building ghcr.io/axisvela/rheo:$VERSION"
  docker build -f "$CORE/release/Dockerfile" -t "ghcr.io/axisvela/rheo:$VERSION" \
    --build-arg "RHEO_VERSION=$VERSION" "$CORE"
  echo "Docker image built locally. Push with: docker push ghcr.io/axisvela/rheo:$VERSION"
else
  echo "WARN: docker not found; skipping image build" >&2
fi

if [[ "${UPLOAD:-0}" == "1" ]]; then
  files=()
  for name in "${targets[@]}"; do
    files+=("$OUT/$name")
  done
  gh release upload "$VERSION" "${files[@]}" "$OUT/SHA256SUMS" --clobber
fi
