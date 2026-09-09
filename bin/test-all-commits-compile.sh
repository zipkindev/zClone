#!/bin/sh
# This compile-checks every commit between the current branch and master.
# It uses disposable worktrees and never changes the caller's checkout or Go bin.
set -eu

BRANCH=$(git rev-parse --abbrev-ref HEAD)
if [ "$BRANCH" = "master" ]; then
    echo "Don't run on master branch" >&2
    exit 2
fi

ROOT=$(git rev-parse --show-toplevel)
COMMITS=$(git -C "$ROOT" rev-list --reverse master.."$BRANCH")
WORKTREE_BASE=$(mktemp -d)
WORKTREE="$WORKTREE_BASE/checkout"

cleanup() {
    git -C "$ROOT" worktree remove --force "$WORKTREE" 2>/dev/null || true
    rmdir "$WORKTREE_BASE" 2>/dev/null || true
}
trap cleanup EXIT INT TERM

for COMMIT in $COMMITS; do
    git -C "$ROOT" worktree add --detach --quiet "$WORKTREE" "$COMMIT"
    echo "Checking $COMMIT"
    (
        cd "$WORKTREE"
        GOCACHE="$WORKTREE/.gocache" GOPROXY=off GOSUMDB=off GOFLAGS=-mod=vendor \
            go test -run '^$' ./...
    )
    git -C "$ROOT" worktree remove --force "$WORKTREE"
done

echo "All commits compiled successfully"
