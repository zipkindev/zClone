---
title: "Local builds"
description: "Build and install Zclone from this source distribution."
type: page
---

# Local builds

This distribution does not publish binaries or operate a download service.
All Go dependencies required for the default build are checked into `vendor/`.

Build Zclone without contacting a Go module service:

```console
make zclone
```

The resulting executable is `build/zclone`. Install or sign it through your
normal local deployment process. To create a source archive that includes the
vendored dependencies, run `make vendorball`; this preserves the checked-in
`vendor/` directory.

If you operate a release service, configure the destination variables in the
Makefile in your release environment. They are intentionally unset in this
source distribution.
