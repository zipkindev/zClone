#!/usr/bin/env bash

# Example script for git bisect run
#
# Copy this file into /tmp say before running as it will be
# overwritten by the bisect as it is checked in.
#
# Change the test below to find out whether zclone is working or not
#
# Run from the project root
#
# git bisect start
# git checkout master
# git bisect bad
# git checkout v1.41 (or whatever is the first good one)
# git bisect good
# git bisect run /tmp/bisect-zclone.sh

set -e

# Compile notifying git on compile failure
make || exit 125
zclone version

# Test whatever it is that is going wrong - exit with non zero exit code on failure
# commented out examples follow

# truncate -s 10M /tmp/10M
# zclone delete azure:zclone-test1/10M || true
# zclone --retries 1 copyto -vv /tmp/10M azure:zclone-test1/10M --azureblob-upload-cutoff 1M

# rm -f "/tmp/tests's.docx" || true
# zclone -vv --retries 1 copy "drive:test/tests's.docx" /tmp
