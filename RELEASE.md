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
