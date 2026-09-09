---
title: "Bugs"
description: "Zclone Bugs and Limitations"
---

# Bugs and Limitations

## Limitations

### Directory timestamps aren't preserved on some backends

As of `v1.66`, zclone supports syncing directory modtimes, if the backend
supports it. Some backends do not support it -- see
[overview](//overview/) for a complete list. Additionally, note
that empty directories are not synced by default (this can be enabled with
`--create-empty-src-dirs`.)

### Zclone struggles with millions of files in a directory/bucket

Currently zclone loads each directory/bucket entirely into memory before
using it.  Since each zclone object takes 0.5k-1k of memory this can take
a very long time and use a large amount of memory.

Millions of files in a directory tends to occur on bucket-based remotes
(e.g. S3 buckets) since those remotes do not segregate subdirectories within
the bucket.

### Bucket-based remotes and folders

Bucket-based remotes (e.g. S3/GCS/Swift/B2) do not have a concept of
directories.  Zclone therefore cannot create directories in them which
means that empty directories on a bucket-based remote will tend to
disappear.

Some software creates empty keys ending in `/` as directory markers.
Zclone doesn't do this as it potentially creates more objects and
costs more.  This ability may be added in the future (probably via a
flag/option).

## Bugs

Bugs are stored in zclone's GitHub project:

- [Reported bugs](/)
- [Known issues](/)
