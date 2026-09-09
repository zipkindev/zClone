# Zclone

Zclone is a local-first file-transfer and synchronization tool for SFTP, SMB,
cloud storage, and object stores. It provides an rsync-style workflow with a
bounded concurrent transfer queue, retries, filtering, verification, and one
consistent CLI across storage backends.

Zclone is not an implementation of rsync's block-delta protocol. For large,
partially modified files on a host that runs rsync, rsync can be the faster
choice. Zclone is intended for concurrent, portable synchronization across
heterogeneous storage services.

## Highlights

- Concurrent copy, move, sync, check, and bidirectional-sync operations.
- Support for SFTP, SMB/CIFS, WebDAV, FTP, S3-compatible stores, and many
  cloud providers.
- Optional encryption, compression, archive, cache, chunking, and union
  backends.
- Local-only, repeatable Go builds: all standard build dependencies are in
  `vendor/`; no module download is required.
- Local SBOM and source-manifest generation for release review.

## Quick start

Build Zclone without downloading Go modules:

```console
make zclone
./build/zclone version
```

Copy a local directory to a remote with bounded parallelism:

```console
zclone copy --transfers 4 --checkers 8 ./source remote:archive/source
```

Use `copy` for additive archive-style transfers. Use `sync` only when the
destination should exactly mirror the source, including deletion of
destination-only files:

```console
zclone sync --transfers 4 --checkers 8 ./source remote:mirror/source
```

Verify a remote after a copy:

```console
zclone check --download ./source remote:archive/source
```

`--download` is the strongest verification option for backends that do not
provide a common server-side hash.

## macOS installation

Create a locally installable package:

```console
make macos-installer
sudo installer -pkg build/Zclone-v0.1.0-arm64.pkg -target /
```

The installer places `zclone` in `/usr/local/bin` and adds that directory to
new login-shell PATHs through `/etc/paths.d/zclone`.

## Validation and offline supply chain

Run the local verification suite:

```console
ZCLONE_SKIP_NETWORK_TESTS=1 make verify-local
```

This builds the binary, compiles every package, runs static analysis, verifies
the vendored dependency manifest, generates a CycloneDX SBOM, verifies the GUI
asset archive, and validates the macOS code signature.

The repository ignores local credentials and machine-specific configuration.
Copy `.env.example` to `.env` only for local integration tests; never commit
the resulting file.

## Repository layout

- `backend/` — storage-provider implementations.
- `cmd/` — command-line commands and local GUI assets.
- `fs/` and `vfs/` — synchronization, filesystem, filtering, and VFS core.
- `docs/content/` — command and backend documentation.
- `vendor/` — reviewed Go dependency source used for offline builds.
- `installer/macos/` — macOS package build and install scripts.

## License and attribution

Zclone is maintained as a distinct local project. Required upstream copyright,
license, and third-party attribution are retained in [COPYING](COPYING),
[NOTICE](NOTICE), and `vendor/`.
