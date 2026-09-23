# Continuous integration and self-hosted verification

GitHub Actions runs the public repository's standard CI policy:

- `.github/workflows/ci.yml` runs the authoritative Linux verification profile,
  focused compatibility regression tests, and compilation checks on Linux,
  macOS, and Windows.
- `.github/workflows/codeql.yml` performs Go CodeQL analysis on changes to
  `main`, pull requests, a weekly schedule, and manual requests.
- Action dependencies are pinned to immutable commit IDs. Dependabot proposes
  reviewed monthly updates to those pins.

The Go build itself remains vendor-only in CI. GitHub may download the selected
Go toolchain and pinned actions, but `GOPROXY=off`, `GOSUMDB=off`, and
`GOFLAGS=-mod=vendor` prevent module resolution from the network.

## Local verification

Run the project from a checked-out local workspace on a self-hosted runner:

```console
make verify-local
```

The machine needs the reviewed Go toolchain, vendored source, and any optional
platform packaging tools installed locally. Local verification does not download
Go modules or invoke a hosted CI action.

For release verification, evaluate `build/zclone.sbom.cdx.json` using an
approved local vulnerability advisory database and retain the signed report
with the release record.

Set `ZCLONE_SKIP_NETWORK_TESTS=1` only on restricted runners that cannot bind a
loopback test server. A release runner must execute the complete verification
without that override.
