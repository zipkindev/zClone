# zClone v0.1.2

## Windows: choose a folder and install

Download **zClone-Updater.exe**, double-click it, choose a folder with **Browse**,
and click **Install / Update**. The window shows download and installation
progress and remembers the folder for future updates. Close running zClone
processes before updating. Existing command-line updater options remain available.

The updater downloads the latest stable Windows x64 release, checks its published
SHA-256, and applies native XPRESS4K compression before installing it. Native
compression roughly halved executable disk usage in Windows QA without the UPX
startup and memory costs. Use a filesystem that supports native executable
compression, such as NTFS. Your zClone configuration and PATH are unchanged.

For manual installation or other platforms, download the archive matching your
OS and architecture and extract it. The updater supports Windows x64; Windows
ARM64 users should use the ARM64 archive. Windows mounts require WinFsp.

## Recovery transfers: what the helper adds

A compressed stdin upload cannot resume from a durable checkpoint after its
process exits. zClone's packaged `bin/zclone-recover.py` addresses that gap with
independently compressed chunks, a persistent journal, SHA-256 verification of
uploaded content, and restart reconciliation. Rerunning the same upload command
keeps verified chunks and recreates unfinished work from the unchanged source.
The helper shipped in v0.1.0; this release adds clearer instructions and evidence.
It requires a POSIX system and Python 3.9 or newer.

The QNAP → Google Drive → TNAS recovery project tested forced interruptions and
lost RAM staging, then completed a 128-chunk / 64 GiB metadata transfer. The
source acquisition, reconstructed TNAS image, and independent analysis copy had
matching SHA-256 hashes. That project's bounded cloud buffer and receiver
acknowledgments are separate orchestration; verified transport does not establish
that a live capture was coherent or that all damaged NAS files were recovered.

See the [recovery guide](https://github.com/zipkindev/zClone/blob/v0.1.2/docs/content/recovery.md)
for upload, resume, restore, guarantees, and evidence details.

## Other changes and downloads

Product-facing names now consistently use **zClone**. Existing commands, provider
identifiers, SDKs, authentication options, and upstream attribution are preserved.

The release includes Linux, macOS, and Windows archives for amd64 and arm64,
the Windows updater, the recovery helper, attribution, SBOM, and **SHA256SUMS**.
Linux executables are statically built for environments such as QNAP. Archives
are unsigned; macOS executables are not notarized.
