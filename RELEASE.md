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

## Standard Arrow JSON profile

Release packaging, Make targets, cross-compilation defaults, and CI use
`arrow_json_stdlib`. Windows packaging also uses `cmount`. Arrow's supported
standard-library adapter replaces `github.com/goccy/go-json`; Arrow listing,
archive formats, authentication, and provider registrations remain enabled.
Raw Go commands must specify `-tags arrow_json_stdlib` (and `cmount` for Windows
mount support); Go module resolution must still remain offline with vendor.

Windows QA builds a same-source `cmount` executable without the JSON tag and
compares it against `cmount,arrow_json_stdlib`. This isolates the JSON change
from the PR's other size changes. Reports include executable/ZIP size, warm
startup medians, RC/GUI peak working sets, compression variants, and function
sizes read from the stripped executables' Go runtime tables.

`bin/compare-arrow-json.py` runs all Arrow listing tests under both libraries,
including JSON compatibility checks for Unicode, nulls, integer limits,
malformed input, and checked allocator cleanup. It alternates seven benchmark
samples per profile after warmup with GOMAXPROCS=2. Listing benchmarks cover
1,000 and 10,000 blobs with timestamps, lengths, booleans, and other standard
listing fields. Medians must stay within 20% for listing latency and 5% for
allocated bytes and allocation counts; the report preserves sample ranges.
Direct JSON encode/decode performance is reported separately: the Azure listing
path uses binary Arrow IPC rather than these JSON helpers. The standard adapter
can make those helpers slower and increase their allocation counts. A smaller
executable does not imply faster operations or a smaller working set.

Arrow pager tests cover pagination, XML fallback, authentication headers, and API
errors using disposable loopback fixtures. Credentialed Azure preview-service
integration still requires an approved disposable account.

## Machine-code retention findings

For the reviewed Windows x64 Go 1.27 profile, the Arrow JSON switch removes
about 1.31 MiB overall, including about 0.74 MiB of function code and 559 runtime
function entries. The remainder includes associated metadata and data. Code
measurements include alignment and are not estimates of safely removable bytes.
`bin/analyze-executable.py` records section sizes, package code, large functions,
and retained S3 methods directly from the stripped executable's runtime tables.

The linker dependency trace (`go build -ldflags "-dumpdep ..."`, with both stdout
and stderr redirected to a file) shows two sources of extra code:

- Arrow's default JSON dispatcher references regular, indented, colored, and
  debug encoder implementations. The supported standard adapter removes that
  separate dispatcher and encoder family.
- `text/template.(*state).evalField` performs dynamic `MethodByName` lookup.
  The linker conservatively retains exported methods of reachable types marked
  as used in interfaces. Its trace propagates that mark from the S3 filesystem
  type to its SDK client. The reviewed binary retains all 224 S3 API operations
  declared by the vendored SDK, including `PutBucketAnalyticsConfiguration`,
  which has no direct call in Zclone source. The trace records first-reach
  parents, not a complete runtime call graph.

SDK-generated serializers, endpoint rule dispatch, and other protocol support
also contribute many specialized functions. The large S3 endpoint resolver
supports runtime choices such as region, acceleration, and FIPS endpoints.
Generic specialization contributes roughly 1 MiB, rather than explaining the
whole approximately 35 MiB code section. Four retained YAML implementations
contribute about 0.57 MiB of code combined, suggesting a separate dependency
consolidation opportunity. Some SDK utility files import test helpers outside
`_test.go`; their package initializers are retained, but the measured `testing`
and assertion code is small. Dependency presence alone does not establish that
an entire test suite or SDK is linked. Further SDK narrowing requires a
separate design that preserves custom templates, authentication choices, and
provider behavior; disabling reflection globally would invalidate those
contracts. Debug symbols are already stripped from release executables.

## Windows size and runtime qualification

Windows archives include the `cmount` build tag and use the vendored cgofuse
Windows implementation without CGO. Mounting requires WinFsp on the destination
machine. Linux and macOS packaging use the standard Arrow JSON profile.

Release downloads use ZIP compression and preserve the ordinary executable.
Windows ordinary archives include `compress-windows.ps1` and this document.
After extracting to NTFS, the helper defaults to XPRESS4K, the performance-first
native algorithm. XPRESS8K and LZX are explicit alternatives; LZX favors disk
savings. Every helper operation verifies that the executable's SHA-256 remains
unchanged. Reapply compression after replacing the executable during an update.
ZIP extraction or copying to a different filesystem does not maintain native
compression automatically. The same full SDKs, features, and authentication
options remain available.

An optional packed x64 archive can be built with an installed UPX tool:

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
existing commands and flags. Candidate, UPX, and native XPRESS4K, XPRESS8K,
and LZX variants undergo disposable copy/sync/check, JSON listing, archive and
crypt roundtrips, custom HTTP template, RC, GUI, and mount read/write checks.
The baseline undergoes the same checks except mount, which is newly enabled.
CI also runs Drive, crypt, Azure Blob, GUI, and HTTP package tests and scans the
ordinary and packed candidates with the runner's Defender engine. Native
variants use the packaged helper under built-in Windows PowerShell; QA verifies
compression, undo, reapplication, allocated-disk savings, and content hashes.

Startup comparisons warm each variant and alternate forward/reverse execution
order across nine samples per variant, preserving every sample in the report.
XPRESS4K qualification fails if median warm startup rises more than 20% or either
RC/GUI peak working set rises more than 10% versus the ordinary executable.
XPRESS8K, LZX, and UPX measurements remain visible for comparison. These gates
identify material regressions; they do not establish the fastest algorithm on
every device. Working sets are single-process observations, and these tests do
not claim cold-start or sustained-throughput qualification. ARM64 release
archives include the helper, but native runtime/performance qualification is
currently Windows x64 only.

The comparison report records executable and ZIP sizes, hashes, startup medians, server peak working sets,
and native compression allocation measurements. Startup numbers are warm,
runner-specific observations, not cold-start performance guarantees. Uploads
include reports and candidate executables for review. Missing WinFsp support,
unavailable Defender, failed scans, or failed runtime checks fail qualification;
they are not silently skipped. Defender qualification does not establish
compatibility with every organization's endpoint-security product.

For installed disk savings on Windows 10 or later with NTFS, the ordinary
executable can be compressed transparently without changing its file contents.
From the extracted Windows release directory:

```powershell
.\compress-windows.ps1
# Prefer minimum allocated disk over the fastest native algorithm:
.\compress-windows.ps1 -Algorithm LZX
# Undo executable compression:
.\compress-windows.ps1 -Undo
```

The equivalent direct native command for the performance-first profile is
`compact /C /F /EXE:XPRESS4K "C:\Tools\zclone.exe"`; undo it with
`compact /U /F /EXE "C:\Tools\zclone.exe"`. Microsoft identifies XPRESS4K as the
fastest algorithm and LZX as the most compact. This reduces allocated disk
space, not the logical file size or the size of an executable copied to another
machine. See the
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

## Whole-executable assessment

The qualified Windows x64 executable at `b67d08e` is 77.77 MiB. Its actual PE
sections break down as follows; code sizes include function alignment:

| Portion | MiB | Share of executable |
| --- | ---: | ---: |
| Machine code | 34.63 | 44.5% |
| Read-only data, including runtime metadata | 37.81 | 48.6% |
| Initialized data | 3.75 | 4.8% |
| Unwind records, relocations, headers and other sections | 1.59 | 2.0% |

A same-source diagnostic build retaining symbols has identical function counts,
total function code bytes, function-name bytes, and PC-mapping bytes to the
qualified build. Its named data attributes approximately 21.8 MiB to the Go
runtime function table, 8.9 MiB to type information, 4.0 MiB to `go:func.*` data,
and 2.0 MiB to the string pool. These figures are diagnostic attributions within
the read-only section, not additional sections or safely removable budgets.
Some named symbols overlap, so arbitrary symbol sizes must not be added together.
Runtime tables support stack walking, garbage collection, reflection, and
diagnostics; debug-symbol stripping does not remove these runtime requirements.

Third-party libraries collectively account for approximately 21.9 MiB of
function code, and Zclone packages for approximately 6.3 MiB. The remainder
includes the Go runtime, standard library, and compiler helpers. S3 contributes
about 2.9 MiB of code, or 3.7% of the executable, excluding its associated data,
metadata, and shared dependencies. An S3-only code reduction cannot be assumed
to produce a large whole-application reduction. The repository's embedded
payload source files total approximately 0.18 MiB; the embedded GUI ZIP itself
is only 1,909 bytes. Removing GUI assets is not a meaningful size opportunity.

The Arrow tag's 559-function reduction is a net count: 563 functions disappear
and four appear. Of the removals, 541 belong to `github.com/goccy/go-json`, 13 are
compiler-generated equality helpers, and nine are standard-library helpers.
The four additions belong to Arrow's standard adapter and the standard JSON
implementation. No Zclone application functions disappear in this controlled
comparison. `bin/analyze-executable.py` reports the exact removed and added
identities and their package counts on every Windows qualification run.
These identities explain the change; runtime tests remain necessary to assess
behavior. No SDK source files are deleted or edited by the Arrow switch.

The measured options with large overall savings are compression:

| Full-feature variant | Size or allocated disk | Warm startup | RC peak working set |
| --- | ---: | ---: | ---: |
| Ordinary executable | 77.77 MiB file | 109 ms | 43.67 MiB |
| ZIP of ordinary executable | 26.70 MiB download | Extract before use | Same executable |
| NTFS LZX | 25.65 MiB allocated disk | 106 ms | 43.89 MiB |
| UPX | 23.25 MiB file | 454 ms | 135.14 MiB |

Measurements come from one qualified hosted Windows run; repeated runs varied
in timing and memory while producing identical executable hashes. The ZIP and
LZX variants preserve the ordinary executable's bytes and feature set. LZX
reduces allocated disk space by about 67%; copying or extracting the executable
elsewhere can require reapplying compression on a compatible filesystem. UPX
reduces portable executable size by about 70%, with substantial measured startup
and memory costs. It remains optional. No evidence currently supports a
similarly large reduction in the ordinary executable's logical size without
additional compatibility, runtime, or maintenance tradeoffs.

## Maintaining dependency reductions

Keep the complete reviewed dependency sources and normal upstream SDK update
path. Supported upstream build profiles and ordinary Go dead-code elimination
are preferred over maintaining a pruned SDK, replacing SDK authentication, or
rewriting application components. A count of unused SDK methods, source files,
or functions is not permission to remove them. A proposal needs a meaningful
whole-application benefit that justifies its maintenance burden. Narrow code
savings alone do not establish that benefit.

For each dependency or size-related update:

1. Record the baseline commit and executable hashes, SDK versions, reviewed Go
   version, target architecture, tags, and linker flags. Preserve a complete SDK
   reference build. Change dependency versions in the controlled environment
   described below; never prune `vendor/` by hand to make a binary smaller.
2. Regenerate the imported-package inventory, SBOM, and source manifest from the
   full dependency sources. Build both profiles with offline vendoring and the
   same toolchain/settings. The imported-package graph and linked function
   inventory answer different questions; neither alone proves feature parity.
3. Run `bin/analyze-executable.py` on the old and new executables. Review the
   exact removed/added functions, affected packages, PE sections, and overall
   archive/file/allocated-disk gains. Do not freeze a function count or assume
   every change in compiler-generated names is a feature change.
4. Run required CI, the Windows executable comparison, and affected backend
   tests against the reference. Preserve provider options, commands, flags,
   custom templates, transfers, error/retry/cancellation behavior, and mount,
   archive, crypt, RC, and GUI behavior. For S3, explicitly cover signing,
   presigned URLs, region/endpoint selection, and the credential-provider chain,
   including session/refresh behavior, roles, profiles, environment, web
   identity, SSO, container/instance credentials, and supported provider signers.
   Keep SDK-internal operations required for authentication, such as S3 Express
   `CreateSession`, even when Zclone does not call them directly.
5. Apply the existing compatibility and listing-performance gates. Compare
   repeated startup, working-set, and allocation measurements and disclose
   regressions. Credentialed cloud cases require approved disposable accounts;
   untested cases keep a new SDK reduction unqualified. Retain the reference
   profile and reports for rollback and the next update.

For example, from a reviewed checkout with the pinned toolchain installed:

```bash
export GOPROXY=off GOSUMDB=off GOFLAGS=-mod=vendor GOTOOLCHAIN=local
mkdir -p build
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go list -deps -json \
  -tags arrow_json_stdlib,cmount . > build/windows-dependencies.json
go version -m build/windows-qa/candidate.exe > build/windows-build-modules.txt
python3 bin/analyze-executable.py --baseline old-zclone.exe \
  --candidate build/windows-qa/candidate.exe --output build/size-review.json
```

Any future source transformation would require a separately reviewed,
deterministic generator, recorded upstream inputs and hashes, checks that reject
unexpected SDK structure, a reproducibility check in CI, and full SDK reference
parity tests. Silent method allowlists, patched Go runtimes, and automatic
feature or authentication removal are not part of the build profile.

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
