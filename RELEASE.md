# Releasing Zclone

Zclone releases are assembled from the repository and its committed local
dependencies. No release target downloads tools or source code.

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
