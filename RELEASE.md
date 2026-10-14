# Releasing Zclone

Zclone releases are assembled from the repository and its committed local
dependencies. Go module resolution stays offline and uses `vendor/`. GitHub Actions may
install the pinned actions and required Go toolchain.

## Independent versions and Git hook

Zclone versions are independent of rclone versions and of GitHub's fork-network
classification. Upstream synchronization is not part of versioning or releases.
`VERSION`, `fs/versiontag.go`, and `docs/layouts/partials/version.html` must match.
Existing upstream history and attribution remain intact.

Install the opt-in commit hook in each checkout:

```console
make install-hooks
```

The hook validates the version files in the Git index, so partial staging cannot
commit inconsistent versions. CI checks the working tree regardless of whether
a developer installed the hook. Version bumps are explicit, not per commit:

```console
make bump-version PART=patch
# PART may also be minor or major
```

The bump changes all three files; it does not commit, tag, or push them. Review
and commit the version change, run verification, and push it to main. Releases
use immutable downstream tags matching `VERSION` exactly:

```console
git tag -a v0.1.0 -m "Zclone v0.1.0"
git push origin v0.1.0
```

Use the current version in both commands. Never move an existing release tag.
The inherited `make retag` force-retagging target is not the GitHub release path.

## GitHub release workflow

`.github/workflows/release.yml` runs for version tags, checks that the tag matches
the source version and belongs to main, and calls the existing CI workflow.
Only after verification succeeds does it build six archives: Linux, macOS and
Windows for amd64 and arm64. Builds use Go 1.27.0, vendored dependencies, and
`CGO_ENABLED=0`; Linux binaries do not depend on host glibc. This does not replace
runtime testing on QNAP or other target platforms.

Archives include attribution, the recovery helper and instructions, the vendored
source manifest, a CycloneDX SBOM and a build identity. `SHA256SUMS` accompanies
the downloads. Executables are currently unsigned and macOS binaries are not
notarized. Versions below 1.0 are marked as GitHub prereleases. Signing and
notarization for trusted platform distribution remain operator steps below.

Publication uses the workflow's scoped GitHub token, with write permission only
in the publishing job. Assets are uploaded to a draft first; the draft becomes
public only when uploads finish. A retry may repair a draft but refuses to replace
assets of an already published release. To retry a failed tag run:

```console
gh workflow run release.yml --ref v0.1.0
```

Build a single archive locally after `make sbom`:

```console
python3 bin/package-release.py v0.1.0 linux amd64
```

## Windows size and runtime qualification

Windows archives include the `cmount` build tag and use the vendored cgofuse
Windows implementation without CGO. Mounting requires WinFsp on the destination
machine. Linux and macOS build tags are unchanged.

The ordinary executable remains the default. An optional packed x64 archive can
be built with an installed UPX tool:

```console
python3 bin/package-release.py v0.1.0 windows amd64 --compression upx
```

Packed archives have a separate `-upx` name and record the UPX version, original
size, packed size, compression method, and build tags in `build-info.json`.
Packing precedes Authenticode signing. ARM64 packing is rejected until that
platform has separate tool and runtime qualification. The publishing workflow
continues to publish the six ordinary archives; packed candidates are QA
artifacts rather than an automatic replacement for the default executable.

The required Windows CI job runs `bin/windows-build-qa.ps1`. It downloads official
UPX 5.1.1 and WinFsp 2.1.25156 installers with pinned SHA-256 checksums, builds the
base revision and candidate with identical release linker settings, and uses
`bin/compare-executables.py` to compare backend option inventories and preserve
existing commands and flags. Candidate, UPX, and Windows `compact /EXE` XPRESS8K
and LZX variants undergo disposable copy/sync/check, JSON listing, archive and
crypt roundtrips, custom HTTP template, RC, GUI, and mount read/write checks.
The baseline undergoes the same checks except mount, which is newly enabled.
CI also runs Drive, crypt, Azure Blob, GUI, and HTTP package tests and scans the
ordinary and packed candidates with the runner's Defender engine.

The comparison report records executable and ZIP sizes, hashes, startup medians, server peak working sets,
and native compression allocation measurements. Startup numbers are warm,
runner-specific observations, not cold-start performance guarantees. Uploads
include reports and candidate executables for review. Missing WinFsp support,
unavailable Defender, failed scans, or failed runtime checks fail qualification;
they are not silently skipped. Defender qualification does not establish
compatibility with every organization's endpoint-security product.

For installed disk savings on Windows 10 or later with NTFS, the ordinary
executable can be compressed transparently without changing its file contents:

```powershell
compact /C /EXE:XPRESS8K "C:\Tools\zclone.exe"
# Undo executable compression:
compact /U /EXE "C:\Tools\zclone.exe"
```

LZX favors disk savings; compare it against XPRESS8K using the QA report before
choosing it for a deployment. This reduces allocated disk space, not the logical
file size or the size of an executable copied to another machine. See the
[Microsoft compact documentation](https://learn.microsoft.com/en-us/windows-server/administration/windows-commands/compact).

For a manual run on a disposable, administrator-capable Windows QA machine with
Go 1.27.0 and Python installed:

```powershell
./bin/windows-build-qa.ps1 -BaselineRef <baseline-commit>
```

This script installs WinFsp on that QA machine. Go dependency resolution remains
offline. Provider integration tests require a separately approved disposable
remote; local QA and inventory equality alone cannot prove cloud-service parity.
The Drive v2 download tests compare the narrower implementation against the SDK
and cover API errors, cancellation, client replacement, retries, ranged reads,
selected API endpoints, and fallback to the original download URL.

Retain authentication and transfer SDK components needed by supported providers.
Do not remove an SDK based on the size of its vendored source tree. Experimental
Arrow listing and dynamic/custom templates remain available; disabling them is
a feature-profile decision, not a behavior-preserving optimization.

## Prerequisites

- The required Go toolchain and any packaging tools are installed locally.
- `vendor/`, `third_party/`, the local compatibility libraries, `go.mod`,
  `go.sum`, and the embedded GUI bundle are present.
- The signing identities and publication destination are controlled by the
  release operator.

## Local release checklist

1. Run `make verify-local` to build with vendored dependencies, run static and
   selected functional tests, verify the checked-in dependency manifest, and
   confirm that the embedded GUI archive matches its reviewed source.
   It also writes `build/zclone.sbom.cdx.json` and
   `build/zclone.sources.sha256`.
   Use the documented self-hosted profile in `ci/` and the approved local
   toolchain profile in `toolchain/` for release verification.
2. Run `make sign` for a locally identifiable development binary. It uses an
   ad-hoc macOS signature unless `CODESIGN_IDENTITY` is supplied.
3. For a distributable macOS binary, set an Apple Developer certificate in
   `CODESIGN_IDENTITY` and run `make release-sign`. Notarization remains a
   release-operator step because it requires the operator's Apple account.
4. Build the required platform artifacts with locally installed packaging tools.
   Windows artifacts must be Authenticode-signed by the release operator.
5. Run `make vendorball`, `make tarball`, and `make sign_upload` to create
   signed SHA-256 and SHA-512 release checksums. Verify them with
   `make check_sign`.
6. Publish only by supplying an explicit destination variable, such as
   `DOWNLOAD_DESTINATION`. No upstream publication service is configured.
7. Review the generated SBOM against an approved local vulnerability advisory
   database. Record the scanner version, advisory database digest, review date,
   and disposition for every finding with the release record. Do not claim a
   vulnerability-free release without that evidence.

## Technical debt review

`TODO` and `FIXME` markers are tracked implementation limitations, not proof of
an active defect. Run `make debt-report` before a release and triage any markers
touched by the release:
security and data-integrity issues block release; backend compatibility issues
require documented tests or a deferral rationale. Do not make speculative
cross-backend changes merely to remove a marker.

## Dependency changes

Online dependency update commands are intentionally disabled. Review and stage
new dependency source in a controlled environment, update `go.mod` and
`go.sum`, regenerate `vendor/` there, then bring the reviewed result into this
repository. Validate it with `make verify-local` before release.
