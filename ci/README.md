# Self-hosted verification

Run the project from a checked-out local workspace on a self-hosted runner:

```console
make verify-local
```

The runner needs the reviewed Go toolchain, vendored source, and any optional
platform packaging tools installed locally. It does not download Go modules or
invoke a hosted CI action.

For release verification, evaluate `build/zclone.sbom.cdx.json` using an
approved local vulnerability advisory database and retain the signed report
with the release record.

Set `ZCLONE_SKIP_NETWORK_TESTS=1` only on restricted runners that cannot bind a
loopback test server. A release runner must execute the complete verification
without that override.
