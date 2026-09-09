#!/usr/bin/env bash
# Exercise a disposable SFTP sync round trip using local-only .env credentials.
set -euo pipefail

ROOT=$(cd "$(dirname "$0")/.." && pwd)
ENV_FILE=${ZCLONE_TEST_ENV:-"$ROOT/.env"}
BINARY=${ZCLONE_BINARY:-"$ROOT/build/zclone"}

if [ ! -r "$ENV_FILE" ]; then
	echo "Missing $ENV_FILE; copy .env.example and provide local test credentials" >&2
	exit 2
fi
if [ ! -x "$BINARY" ]; then
	echo "Missing executable $BINARY; run make zclone first" >&2
	exit 2
fi

set -a
. "$ENV_FILE"
set +a

: "${ZCLONE_SFTP_HOST:?ZCLONE_SFTP_HOST is required}"
: "${ZCLONE_SFTP_USER:?ZCLONE_SFTP_USER is required}"
: "${ZCLONE_SFTP_PASSWORD:?ZCLONE_SFTP_PASSWORD is required}"
ZCLONE_SFTP_PORT=${ZCLONE_SFTP_PORT:-22}

umask 077
WORK=$(mktemp -d)
CONFIG="$WORK/zclone.conf"
SOURCE="$WORK/source"
PULL="$WORK/pull"
REMOTE_NAME=zclone_sftp_test
RUN_ID="zclone-integration-$(date +%Y%m%d%H%M%S)-$$"
REMOTE_PATH="$REMOTE_NAME:$RUN_ID"
remote_removed=false

cleanup() {
	if [ "$remote_removed" = false ]; then
		"$BINARY" --config "$CONFIG" purge "$REMOTE_PATH" >/dev/null 2>&1 || true
	fi
	rm -rf "$WORK"
}
trap cleanup EXIT INT TERM

"$BINARY" --config "$CONFIG" config create "$REMOTE_NAME" sftp \
	host "$ZCLONE_SFTP_HOST" user "$ZCLONE_SFTP_USER" port "$ZCLONE_SFTP_PORT" \
	pass "$ZCLONE_SFTP_PASSWORD" --obscure --non-interactive --no-output

# Record the server host key in the temporary test configuration before transfer.
"$BINARY" --config "$CONFIG" --sftp-pin-host-key lsd "$REMOTE_NAME:" >/dev/null

mkdir -p "$SOURCE/one" "$SOURCE/two" "$SOURCE/three"
directories=(one two three)
for index in $(seq 1 12); do
	directory=${directories[$((index % 3))]}
	dd if=/dev/urandom of="$SOURCE/$directory/file-$index.bin" bs=1048576 count=1 2>/dev/null
done
printf '%s\n' "zclone SFTP integration test $RUN_ID" > "$SOURCE/manifest.txt"

"$BINARY" --config "$CONFIG" sync --transfers 8 --checkers 16 "$SOURCE" "$REMOTE_PATH"
"$BINARY" --config "$CONFIG" check --one-way "$SOURCE" "$REMOTE_PATH"
"$BINARY" --config "$CONFIG" sync --transfers 8 --checkers 16 "$REMOTE_PATH" "$PULL"
diff -qr "$SOURCE" "$PULL"
"$BINARY" --config "$CONFIG" purge "$REMOTE_PATH"
remote_removed=true

echo "SFTP push, pull, content verification, and remote cleanup passed for $RUN_ID"
