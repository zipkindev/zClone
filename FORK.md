# Fork provenance

Zclone is an independently maintained downstream derivative of
[rclone](https://github.com/rclone/rclone). It was created to support a locally
controlled build and deployment profile, including use on restricted work
systems. It is not an official rclone distribution and is not affiliated with or
endorsed by Nick Craig-Wood or the rclone project.

## Lineage

- Upstream project: `https://github.com/rclone/rclone.git`
- Upstream branch: `master`
- Last shared commit: `6a617a379bc9883d015370cf9dc4320f6e0138e4`
- First Zclone-only commit: `9a982e3cf10307ba4778ce461d345694a04350ac`
- Downstream repository: `https://github.com/zipkindev/zClone.git`
- Downstream branch: `main`

The full pre-fork Git history is intentionally retained. Do not squash it into a
new root commit or rewrite upstream author names: that history is the most precise
record of who contributed each change. Zclone-only commits should identify their
actual authors in the normal Git metadata.

## Licensing and credit

Rclone is distributed under the MIT license. Zclone retains the upstream
copyright and permission notice in [COPYING](COPYING), adds a concise provenance
statement in [NOTICE](NOTICE), and retains dependency notices in `vendor/`,
`lib/`, and `third_party/`. Source and binary distributions must continue to
include the MIT notice as required by that license.

The rclone contributors are upstream authors, not Zclone maintainers. Zclone's
maintainers are listed separately in [MAINTAINERS.md](MAINTAINERS.md).

## Tracking upstream

A clone can track both projects with separate remotes:

```console
git remote rename origin zclone
git remote add upstream https://github.com/rclone/rclone.git
git fetch --all --prune
```

Before importing an upstream change, compare the histories and review the patch
for Zclone-specific naming, module paths, vendoring, local-only build rules, and
disabled network/update behavior:

```console
git log --left-right --cherry-pick --oneline upstream/master...zclone/main
git show <upstream-commit>
git cherry-pick -x <upstream-commit>
```

The `-x` trailer records the upstream commit selected for a cherry-pick. For a
larger merge or manual port, record the upstream commit range in the commit
message. Never attribute a downstream adaptation to an upstream author unless
that person actually authored the downstream commit; preserve upstream credit by
referencing the source commit instead.

GitHub may not display this repository in rclone's fork network if it was created
by pushing an existing clone rather than with GitHub's **Fork** action. That UI
relationship is separate from copyright compliance and Git ancestry. Preserving
history, notices, and a prominent upstream link provides the durable provenance
record.
