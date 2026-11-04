#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root"

if [[ -n "$(gofmt -l *.go)" ]]; then
  echo "gofmt needs to run:" >&2
  gofmt -l *.go >&2
  exit 1
fi

go test ./...
go vet ./...
GOOS=windows GOARCH=amd64 go vet ./...

if ! command -v makensis >/dev/null 2>&1; then
  echo "makensis is required for Windows installers: brew install nsis" >&2
  exit 1
fi

go tool mygo build -platform darwin/universal,windows/amd64,windows/arm64

dmgs=(build/darwin-universal/miguwave-*.dmg)
if [[ ${#dmgs[@]} -ne 1 || ! -f "${dmgs[0]}" ]]; then
  echo "expected one universal DMG" >&2
  exit 1
fi

setups=()
for arch in amd64 arm64; do
  matches=(build/windows-$arch/miguwave-setup-*.exe)
  if [[ ${#matches[@]} -ne 1 || ! -f "${matches[0]}" ]]; then
    echo "expected one Windows installer for $arch" >&2
    exit 1
  fi
  setups+=("${matches[0]}")
done

shasum -a 256 "${dmgs[0]}" "${setups[@]}" > build/SHA256SUMS

app="build/darwin-universal/MiguWave.app"
codesign --verify --deep --strict --verbose=2 "$app"

echo
echo "Built: ${dmgs[0]}"
echo "Built: ${setups[0]}"
echo "Built: ${setups[1]}"
echo "Checksum: build/SHA256SUMS"
if security find-identity -v -p codesigning | grep -q "Developer ID Application"; then
  echo "Signature: Developer ID"
else
  echo "Signature: ad-hoc (Gatekeeper will reject this build on other Macs)"
  echo "Enable Apple Developer ID and notarization before a frictionless public release."
fi
echo "Windows installers: unsigned (SmartScreen will warn until a certificate is configured)"
