#!/usr/bin/env bash
# Build a self-contained macOS installer from the locally built binary.
set -euo pipefail

ROOT=$(cd "$(dirname "$0")/../.." && pwd)
BINARY=${ZCLONE_BINARY:-"$ROOT/build/zclone"}
VERSION=$(sed -n '1p' "$ROOT/VERSION")
PACKAGE_VERSION=${VERSION#v}
PACKAGE_VERSION=${PACKAGE_VERSION%%-*}
ARCH=$(uname -m)
OUTPUT=${ZCLONE_INSTALLER_OUTPUT:-"$ROOT/build/Zclone-${VERSION}-${ARCH}.pkg"}
STAGING=$(mktemp -d)

cleanup() {
	rm -rf "$STAGING"
}
trap cleanup EXIT

if [ "$(uname -s)" != "Darwin" ]; then
	echo "macOS installers must be built on macOS" >&2
	exit 2
fi
if [ ! -x "$BINARY" ]; then
	echo "Build and sign zclone first: make sign" >&2
	exit 2
fi
if ! [[ "$PACKAGE_VERSION" =~ ^[0-9]+(\.[0-9]+){0,2}$ ]]; then
	echo "VERSION must begin with a macOS package version: $VERSION" >&2
	exit 2
fi

mkdir -p "$STAGING/root/usr/local/lib/zclone" "$STAGING/scripts" "$(dirname "$OUTPUT")"
install -m 0755 "$BINARY" "$STAGING/root/usr/local/lib/zclone/zclone"
# Finder provenance is not part of the executable and otherwise becomes AppleDouble
# metadata in an unsigned package payload.
xattr -c "$STAGING/root/usr/local/lib/zclone/zclone"
install -m 0755 "$ROOT/installer/macos/scripts/postinstall" "$STAGING/scripts/postinstall"

args=(
	--root "$STAGING/root"
	--scripts "$STAGING/scripts"
	--identifier org.zclone.zclone
	--version "$PACKAGE_VERSION"
	--install-location /
	--ownership recommended
)
if [ -n "${INSTALLER_SIGN_IDENTITY:-}" ]; then
	args+=(--sign "$INSTALLER_SIGN_IDENTITY")
fi

COPYFILE_DISABLE=1 pkgbuild "${args[@]}" "$OUTPUT"
if [ -n "${INSTALLER_SIGN_IDENTITY:-}" ]; then
	pkgutil --check-signature "$OUTPUT"
else
	pkgutil --payload-files "$OUTPUT" >/dev/null
	echo "Created unsigned local-install package; set INSTALLER_SIGN_IDENTITY for distribution."
fi
echo "Created $OUTPUT"
