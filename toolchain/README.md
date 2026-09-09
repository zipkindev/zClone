# Local toolchain profile

The verified Zclone build requires Go 1.27.0 with the committed `vendor/` tree.
`make verify-local` checks the installed Go version before building. Set
`GOPROXY=off`, `GOSUMDB=off`, and `GOFLAGS=-mod=vendor` for every build and
test invocation.

For an offline container build, preload a reviewed Go builder image and runtime
image into the local OCI store, then pass their immutable local names or digests
to `docker build` through `GO_IMAGE` and `RUNTIME_IMAGE`. The Dockerfile has no
default image and does not install packages by default, so this is required:

```console
docker build --pull=false \
  --build-arg GO_IMAGE=approved-zclone-builder@sha256:... \
  --build-arg RUNTIME_IMAGE=approved-zclone-runtime@sha256:... \
  --build-arg INSTALL_BUILD_DEPS=0 \
  --build-arg INSTALL_RUNTIME_DEPS=0 \
  -t zclone:local .
```

The builder image must already contain the Go toolchain, `make`, `bash`, `gawk`,
and `git`; the runtime image must already contain the required runtime libraries.

This repository intentionally does not name a remote registry, image tag, or
package mirror. The release operator records the approved local image digests
with each release alongside the SBOM and signed checksum files.
