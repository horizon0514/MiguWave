#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root"

app_name="MiguWave"
identity="MiguWave Local Signing"
support_dir="$HOME/Library/Application Support/$app_name/local-signing"
keychain="$support_dir/signing.keychain-db"
password_file="$support_dir/signing.keychain-password"

install=false
case "${1:-}" in
  "")
    ;;
  --install)
    install=true
    ;;
  *)
    echo "usage: $0 [--install]" >&2
    exit 2
    ;;
esac

case "$(uname -m)" in
  arm64)
    default_platform="darwin/arm64"
    ;;
  x86_64)
    default_platform="darwin/amd64"
    ;;
  *)
    default_platform="darwin/arm64"
    ;;
esac
platform="${MYGO_LOCAL_PLATFORM:-$default_platform}"

umask 077
mkdir -p "$support_dir"

if [[ ! -f "$keychain" || ! -f "$password_file" ]]; then
  if [[ -e "$keychain" || -e "$password_file" ]]; then
    echo "local signing files are incomplete; move $support_dir aside and retry" >&2
    exit 1
  fi

  work="$(mktemp -d "${TMPDIR:-/tmp}/miguwave-signing.XXXXXX")"
  trap 'find "$work" -depth -delete 2>/dev/null || true' EXIT

  keychain_password="$(openssl rand -hex 32)"
  p12_password="$(openssl rand -hex 32)"

  cat > "$work/cert.cnf" <<'EOF'
[req]
distinguished_name = dn
x509_extensions = ext
prompt = no
[dn]
CN = MiguWave Local Signing
O = MiguWave Local
[ext]
basicConstraints = critical,CA:FALSE
keyUsage = critical,digitalSignature
extendedKeyUsage = codeSigning
subjectKeyIdentifier = hash
authorityKeyIdentifier = keyid,issuer
EOF

  /usr/bin/openssl req -x509 -newkey rsa:3072 -sha256 -nodes -days 3650 \
    -config "$work/cert.cnf" \
    -keyout "$work/key.pem" \
    -out "$work/cert.pem" >/dev/null 2>&1
  /usr/bin/openssl pkcs12 -export \
    -inkey "$work/key.pem" \
    -in "$work/cert.pem" \
    -out "$work/identity.p12" \
    -passout "pass:$p12_password" \
    -name "$identity" >/dev/null 2>&1

  security create-keychain -p "$keychain_password" "$keychain"
  security set-keychain-settings -lut 21600 "$keychain"
  security unlock-keychain -p "$keychain_password" "$keychain"
  security import "$work/identity.p12" \
    -k "$keychain" \
    -P "$p12_password" \
    -T /usr/bin/codesign >/dev/null
  security set-key-partition-list \
    -S apple-tool:,apple:,codesign: \
    -s \
    -k "$keychain_password" \
    "$keychain" >/dev/null

  printf '%s\n' "$keychain_password" > "$password_file"
  chmod 600 "$password_file" "$keychain"
  echo "Created local signing identity: $identity"
fi

keychain_password="$(<"$password_file")"
security unlock-keychain -p "$keychain_password" "$keychain" >/dev/null
if ! security find-certificate -c "$identity" "$keychain" >/dev/null 2>&1; then
  echo "local signing certificate is missing from $keychain" >&2
  exit 1
fi

go tool mygo build -platform "$platform" -skip-dmg

app="$root/build/${platform//\//-}/$app_name.app"
if [[ ! -d "$app" ]]; then
  echo "expected app bundle at $app" >&2
  exit 1
fi

codesign --force --deep \
  --sign "$identity" \
  --keychain "$keychain" \
  "$app"
codesign --verify --deep --strict --verbose=2 "$app"

echo "Built: $app"

if $install; then
  target="/Applications/$app_name.app"
  if pgrep -x "$app_name" >/dev/null; then
    pkill -x "$app_name" || true
    for _ in {1..30}; do
      pgrep -x "$app_name" >/dev/null || break
      sleep 0.1
    done
  fi

  backup=""
  if [[ -e "$target" ]]; then
    backup="/tmp/$app_name.app.backup.$(date +%Y%m%d-%H%M%S)"
    mv "$target" "$backup"
  fi
  if ! ditto "$app" "$target"; then
    if [[ -n "$backup" && ! -e "$target" ]]; then
      mv "$backup" "$target"
    fi
    exit 1
  fi

  echo "Installed: $target"
  if [[ -n "$backup" ]]; then
    echo "Previous app: $backup"
  fi
fi
