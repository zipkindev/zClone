#!/usr/bin/env sh
set -eu

case "$(go version)" in
  *"go1.27.0 "*) ;;
  *)
    echo "Zclone verification requires Go 1.27.0; see toolchain/README.md" >&2
    exit 2
    ;;
esac

verification_storage="$(pwd)/build/recovery-verification"
GOTMPDIR="${GOTMPDIR:-$verification_storage/tmp}"
GOCACHE="${GOCACHE:-$verification_storage/cache}"
mkdir -p "$GOTMPDIR" "$GOCACHE"
export GOTMPDIR GOCACHE

python3 bin/version.py check
make zclone
PYTHONDONTWRITEBYTECODE=1 ZCLONE_RECOVERY_TEST_BINARY="$(pwd)/build/zclone" \
  python3 -m unittest discover -s bin/tests -p 'test_*.py'
ZCLONE_CONFIG=/notfound GOPROXY=off GOSUMDB=off GOFLAGS=-mod=vendor go test -tags arrow_json_stdlib -run '^$' ./...
GOPROXY=off GOSUMDB=off GOFLAGS=-mod=vendor go vet -tags arrow_json_stdlib ./...
if [ "${ZCLONE_SKIP_NETWORK_TESTS:-0}" != "1" ]; then
  ZCLONE_CONFIG=/notfound GOPROXY=off GOSUMDB=off GOFLAGS=-mod=vendor \
    go test -tags arrow_json_stdlib ./cmd/gui ./fs/config ./fs/operations ./fs/sync ./backend/memory ./backend/azureblob/...
fi
make sbom
make verify-sources
make verify-gui-dist
if command -v codesign >/dev/null 2>&1; then
  make sign
  codesign -dv --verbose=2 build/zclone 2>&1
fi
