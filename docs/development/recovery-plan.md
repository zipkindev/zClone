# Checkpointed recovery upload plan

Implement a POSIX Python 3.9+ controller around existing zClone commands, keeping
backend and stdin behavior unchanged. Use standard-library dependencies only.
The source is an immutable seekable image or a read-only block device. A pipe
cannot provide the replay contract required for process recovery.

1. Write interruption and corruption tests before the controller.
2. Stage one bounded source range at a time as an independent gzip file. Require
   exact reads, deterministic gzip headers, and both raw and compressed SHA-256.
3. Save prepared, uploaded, and verified states to an exclusively locked local
   journal. Chain record checksums, fsync records and directory creation, discard
   only an unterminated final record, and reject other corruption.
4. Reconcile unfinished chunks by reading remote content before uploading again.
   Reuse staged files or regenerate identical bytes when staging was lost. Never
   release staging before the verified journal record is durable.
5. Publish a session-specific manifest after all chunks verify. Add restoration
   that checks layout and both hashes and publishes only a complete new image.
6. Test actual process termination at prepared, uploaded, and verified states,
   erase staging to simulate tmpfs loss, resume with a real zClone executable,
   and compare restored disposable images byte for byte.

Implementation: `bin/zclone-recover.py`. Tests:
`bin/tests/test_zclone_recover.py`. Usage: `docs/content/recovery.md`.

This first version requires the local persistent journal for upload resumption.
It reads every uploaded chunk back for verification, supports one staging slot,
requires an operator-verified stable identity for raw devices, and stops on read
errors. Automatic bad-sector recovery, provider multipart resumption, concurrent
pipelines, reconstruction of lost journals, and resumable restoration are future
work. They need separate contracts and tests.
