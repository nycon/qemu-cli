#!/usr/bin/env bash
set -euo pipefail

VERSION="${1:-v1.0.0}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="${ROOT}/dist"
mkdir -p "$OUT"
rm -f "$OUT"/*

targets=(
  "linux/amd64"
  "linux/arm64"
  "darwin/amd64"
  "darwin/arm64"
  "windows/amd64"
)

echo "Building qemu-cli ${VERSION}..."
for t in "${targets[@]}"; do
  GOOS="${t%/*}"
  GOARCH="${t#*/}"
  EXT=""
  [[ "$GOOS" == "windows" ]] && EXT=".exe"
  NAME="qemu-cli_${VERSION}_${GOOS}_${GOARCH}${EXT}"
  echo "  -> ${NAME}"
  CGO_ENABLED=0 GOOS="$GOOS" GOARCH="$GOARCH" go build \
    -trimpath \
    -ldflags "-s -w -X github.com/nycon/qemu-cli/cmd.Version=${VERSION}" \
    -o "${OUT}/${NAME}" \
    "$ROOT"

  if [[ "$GOOS" == "windows" ]]; then
    (cd "$OUT" && zip -q "${NAME%.exe}.zip" "$NAME" && rm -f "$NAME")
  else
    (cd "$OUT" && tar -czf "${NAME}.tar.gz" "$NAME" && rm -f "$NAME")
  fi
done

(
  cd "$OUT"
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum ./* > SHA256SUMS
  else
    shasum -a 256 ./* > SHA256SUMS
  fi
)

echo "Done: ${OUT}"
ls -lh "$OUT"
