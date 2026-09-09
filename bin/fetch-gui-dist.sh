#!/bin/sh

# The GUI bundle is committed to this repository so normal builds are offline.
echo "Fetching GUI assets is disabled in the local Zclone distribution." >&2
echo "Replace cmd/gui/dist.zip and cmd/gui/dist.tag from an approved local source." >&2
exit 1
