---
title: "Install"
description: "Build and install Zclone locally."
---

# Install Zclone locally

This source distribution intentionally has no central installer, download
service, or self-update endpoint. The complete Go dependency set for the
standard build is included in `vendor/`.

From the repository root, build the executable without a Go module download:

```console
make zclone
```

The executable is written to `build/zclone`. Copy it to a location on your
`PATH` using your organization's normal deployment mechanism. On macOS, sign
the binary after building:

```console
make sign CODESIGN_IDENTITY="YOUR SIGNING IDENTITY"
```

## macOS installer

Build a locally installable macOS package with no download step:

```console
make macos-installer
```

This signs the binary with the configured local signing identity (ad-hoc by
default), creates `build/Zclone-<version>-<architecture>.pkg`, and installs the
binary in `/usr/local/lib/zclone/zclone`. Its post-install script provides a
`/usr/local/bin/zclone` symlink and `/etc/paths.d/zclone`, so it is available
in newly opened Terminal sessions. Install it with:

```console
sudo installer -pkg build/Zclone-<version>-<architecture>.pkg -target /
```

Set `INSTALLER_SIGN_IDENTITY` to a Developer ID Installer certificate when
creating a package for distribution. Without it, the package is unsigned and
contains an ad-hoc-signed binary, suitable only for local installation.

To create a release, use the Makefile targets only after setting the required
release destination variables in the release environment. They have no defaults
in this checkout.
