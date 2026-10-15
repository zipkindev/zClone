---
title: "Checkpointed recovery uploads"
description: "Stage, verify, resume, and restore independently compressed image chunks."
---

`bin/zclone-recover.py` uploads an immutable image or block device as independent
gzip objects using zClone. It requires POSIX, Python 3.9 or newer, and a zClone
executable. It is a standalone helper, not a built-in zClone command.

## Upload and resume

Use a healthy persistent filesystem for the journal and either RAM or a separate
healthy disk for staging. Do not stage onto the device being recovered. The
source must remain unchanged for the entire session; use an offline source or
an immutable snapshot.

```console
python3 bin/zclone-recover.py upload /safe/source.img remote:recovery/run-001 \
  --journal /safe/state/run-001.jsonl \
  --stage-dir /safe/staging \
  --zclone ./build/zclone
```

Rerun the exact command to resume. Keep the journal even after completion. It
records source identity, destination, chunk size, compression settings, byte
ranges, sizes, and SHA-256 hashes. A lock prevents concurrent writers to the
same journal. Changing source identity or settings causes resumption to stop.

For a block device, additionally provide `--size` with the exact device capacity
in bytes and `--source-id` with its verified serial or other stable identity.
Check that identity against the actual device before every invocation. The
helper checks the supplied identity, device path and device number, and capacity;
it does not discover or authenticate a hardware serial. It opens sources for
reading only and rejects pipes and other non-seekable sources. Unreadable ranges
or unexpected short reads stop recovery; this is not a bad-sector rescue tool.

`--chunk-size` sets the uncompressed bytes per chunk (default 67108864, or 64
MiB). One chunk is staged at a time. Allow staging space for the chunk plus 1%
and 1 MiB for compression overhead. Also leave RAM for Python, zClone's transfer
buffers, and the growing journal. Compression may not reduce size. `--config`
selects the zClone configuration and `--retries` sets copy retries (default 3).

## Completion and verification

Each range is compressed independently with fixed gzip settings. Source and
compressed bytes receive separate SHA-256 hashes. The helper finishes and fsyncs
the staged file, records its preparation, uploads using `copyto`, reads remote
content with `cat`, and records successful content verification before deleting
the staged file. Verification requires downloading every chunk; account for
remote egress charges and traffic.

On restart, an unfinished remote object is checked first. If its size and hash
match, it is checkpointed without rereading the source. Otherwise its retained
staged file is uploaded again. If staging was lost, the source range is
regenerated and must match both recorded hashes before upload. RAM staging does
not survive a reboot. With a failing source, persistent staging on a separate
healthy device preserves recovered bytes more reliably.

Chunk names contain the session ID, index, and compressed hash. Completed chunks
are assumed to remain unchanged on the remote. Different sessions have distinct
chunk and manifest names. The helper never deletes remote files.

The journal uses fsynced, checksummed records. Only an incomplete final line is
discarded on restart; malformed complete records cause recovery to stop. Store
the journal outside staging on storage that honors fsync. There is no automatic
reconstruction if that journal is lost. Remote manifests are published only
when all chunks have been verified, so an interrupted session needs its local
journal to finish.

The final output prints an exact session-specific manifest path, for example
`remote:recovery/run-001/manifests/SESSION.json`. It is uploaded and read back for
verification. A failed manifest upload can be retried by rerunning the upload
command without rereading completed chunks.

## Restore

Use the exact manifest path printed by upload and a new local output filename:

```console
python3 bin/zclone-recover.py restore \
  remote:recovery/run-001/manifests/SESSION.json /safe/restored.img \
  --zclone ./build/zclone
```

Restoration checks that ranges cover the entire source without gaps or overlaps,
verifies each compressed object, and checks the size and SHA-256 of each
uncompressed range. It writes a temporary image and publishes the complete
image without overwriting an existing output. It does not restore directly to
block devices. Allow space for the complete image and one compressed chunk.
An interrupted restore must be restarted; a hard kill may leave temporary
`.partial` files to remove manually.

SHA-256 checks detect corruption; they do not authenticate an attacker-controlled
manifest. Preserve the trusted journal and restrict write access to recovery
objects. Test restoration with disposable data before starting a real recovery.

## Tests

```console
python3 -m unittest discover -s bin/tests -p test_zclone_recover.py
```

The suite includes forced process termination with disposable local remotes.
It uses `build/zclone` when present; set `ZCLONE_RECOVERY_TEST_BINARY` to another
built executable. Without one, the real-executable test is explicitly skipped.

## The streamed-upload gap

Upstream [`rcat`](https://rclone.org/commands/rclone_rcat/) consumes stdin. Its
whole upload cannot be retried when that input is no longer available; supported
multipart backends can retry individual parts during a running upload. This is
particularly relevant to `disk image → gzip → rcat`: after the process exits,
recreating an arbitrary point in one gzip stream is not a durable resume strategy.

The packaged helper changes the unit of recovery to an independently compressed,
seekable range and records verified progress durably. It uses `copyto` and `cat`;
it does not patch `rcat` or promise universal resumability for ordinary copies.
Completed chunks are retained rather than restarting the entire image. An
unfinished range may need rereading when its staged file is lost, so the source
must remain unchanged and persistently retaining staging can reduce rereads of a
failing device. A lost journal cannot be reconstructed automatically.

## QNAP Drive TNAS validation

The retained `QNAP-NAS-Recovery` runbook and diagnostic receipts record the
following validation on October 13–14, 2026, using published zClone v0.1.0.
This summary omits private credentials, host addresses, destination names, and
customer paths.

1. The released static Linux amd64 client ran on QNAP's Linux 5.10.60-qnap
   without a special QNAP rebuild. Four forced process-kill scenarios interrupted
   staging and prepared/uploaded/verified checkpoints. RAM staging was removed
   before resume; all restored the same 8,388,608-byte test image and SHA-256.
   Uploaded/verified checkpoint tests also guarded against regenerating the first
   accepted source chunk.
2. Direct QNAP uploads to Google Drive passed readback, gzip, length, and raw
   SHA-256 checks with synthetic inputs. A tandem test transferred an 8 MiB image
   as four independently compressed 2 MiB chunks under a 4,500,000-byte cloud
   budget. The TNAS downloader was stopped after accepting/draining chunk zero,
   restarted against the same persistent journal, and restored matching data.
   The producer waited at its buffer limit and continued as the receiver drained it.
3. A later metadata transfer completed 128 chunks and 68,719,476,736 bytes
   (64 GiB). The acquisition reference, TNAS reconstruction, and independent
   analysis image had the same whole-image SHA-256. The runbook recorded
   42,878,556,638 compressed payload bytes. This is one input's observed result,
   not a compression-ratio prediction for other data.

The published helper supplied checkpoint serialization and content verification.
The recovery project's separate controller supplied bounded parallel staging,
cloud-buffer backpressure, encryption, durable receiver acceptance, verified
acknowledgments, and retirement of only its newly generated temporary cloud
payloads. TNAS acceptance included compressed/raw hashes, sizes, and fsync of the
retained data. Do not apply that controller's temporary-object deletion policy to
this helper: `zclone-recover.py` never deletes remote files.

These results demonstrate tested interruption recovery and transport integrity.
They do not establish whole-host power-loss recovery, live-capture coherence,
filesystem repair, or recovery of all user files. The live source was not proven
to be a coherent snapshot; matching copies preserve the captured bytes, including
any inconsistency already in those bytes. Restoration from retired cloud payloads
requires the TNAS-retained chunks and the separately protected decryption keys.
