---
title: "Local signing"
description: "Sign locally built Zclone binaries."
---

# Local signing

Release signing is controlled by the organization maintaining this checkout.
No public key service or hosted verification endpoint is configured here.

On macOS, sign a locally built executable with:

```console
make sign CODESIGN_IDENTITY="YOUR SIGNING IDENTITY"
```

The default signing identity is ad hoc and is useful for local verification.
Use your organization's certificate, notarization, and checksum process for
distributed artifacts.
