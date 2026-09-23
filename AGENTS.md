# AGENTS.md

This file provides guidance to AI coding agents (e.g. Claude Code, Codex, Cursor,
Gemini CLI, and similar tools) when working with code in this repository.

Zclone welcomes AI-assisted contributions, but the expectation is that you, the human submitter, understand every line you propose and have compiled and tested it against real zclone code - not just generated it. See the "AI-assisted contributions" section of [CONTRIBUTING.md](CONTRIBUTING.md) before opening a pull request.

## Project Overview

Zclone is a Go command-line program for syncing files and directories across
local, network, and cloud storage. It is an independently maintained derivative
of rclone, created for a locally controlled and restricted-system-friendly build
profile. Read [FORK.md](FORK.md) before changing project identity, licensing,
attribution, module paths, update behavior, or upstream synchronization policy.

## Fork and Attribution Guardrails

- Treat pre-fork code as rclone-derived work, not as original Zclone work. Keep
  [COPYING](COPYING), [NOTICE](NOTICE), and the upstream credit in the README.
- Preserve Git authorship. Do not squash the inherited history into a new root,
  rewrite contributor identities, or replace upstream copyright notices with the
  downstream maintainer's name.
- Do not describe rclone contributors as Zclone maintainers or imply that rclone
  endorses or supports Zclone. Current downstream ownership is documented in
  [MAINTAINERS.md](MAINTAINERS.md).
- When porting an upstream commit, prefer `git cherry-pick -x` when practical.
  Otherwise identify the upstream commit or range in the downstream commit
  message and describe any Zclone-specific adaptation.
- Avoid global `rclone` to `zclone` replacements. Historical text, protocol/API
  identifiers, provider integration values, migration compatibility, licenses,
  vendored files, and generated files may need to retain upstream names.
- Never edit vendored or generated files merely to change attribution. Update
  source-of-truth files and regenerate only when the task requires it.

## General Notes

**We take backwards compatibility very seriously.** PRs should not change the observable behaviour of existing commands, flags or rc API without very good reason. Zclone does not try to preserve a stable Go API but try not to change it gratuitously.

Zclone operates with a lot of different backends, so **compatibility is key**. It is the backend integration tests which guarantee that compatibility. Changes should consider both known and unknown backends and should take care not to break functionality of existing installations.

The core parts of zclone under `fs` and `vfs` need to work with all backends and **backend specific hacks won't be merged**. Fixes likely need to go in the relevant backend or if new behaviour is really needed, a new Feature flag needs to be added.

**Changes should be kept to the minimum.** Work hard to make the most elegant, smallest change you can. Do not refactor or re-order code unless necessary as this makes review more challenging. Re-use existing test scaffolding, existing or library routines (e.g. `lib`) where possible.

Make sure added tests **actually test the code you have written** and test the intention behind the change. If you are fixing a problem, write the tests first to reproduce the problem before starting on the fix.

## Build and Test Commands

```bash
# Build zclone using the reviewed vendored dependencies
make zclone

# Direct Go commands must remain offline and use vendor/
GOPROXY=off GOSUMDB=off GOFLAGS=-mod=vendor go build

# Run all unit tests (no cloud credentials needed)
make quicktest
# Run the repository's local verification profile
make verify-local

# Run tests for a specific package, preserving offline vendoring
GOPROXY=off GOSUMDB=off GOFLAGS=-mod=vendor go test -v ./backend/memory/

# Run a single test
GOPROXY=off GOSUMDB=off GOFLAGS=-mod=vendor go test -v -run TestIntegration/FsCheckWrap ./backend/memory/

# Run tests with race detector
make racequicktest

# Lint (requires golangci-lint)
golangci-lint run ./...

# Run backend integration tests (requires configured TestRemote remote)
cd backend/drive && go test -v
# Run sync/operations integration tests against a remote
cd fs/sync && go test -v -remote TestDrive:
cd fs/operations && go test -v -remote TestDrive:

# Run integration tests via test framework
GOPROXY=off GOSUMDB=off GOFLAGS=-mod=vendor go run ./fstest/test_all -backends drive
```

The reviewed verification environment uses Go 1.27.0 even though `go.mod`
declares Go 1.26.0. See [toolchain/README.md](toolchain/README.md). Tests that
bind loopback sockets may require `ZCLONE_SKIP_NETWORK_TESTS=1` on restricted
runners, but a skipped network suite is not equivalent to full verification.

Never use real work data, production remotes, or credentials for tests. Commands
such as `sync`, `move`, `delete`, and `purge` can remove data. Use temporary local
directories or an explicitly disposable test remote, and use `--dry-run` where
the command supports it.

## Continuous Integration

- `.github/workflows/ci.yml` is the required build/test workflow. Keep its Go
  version aligned with `bin/verify-local.sh`, `toolchain/README.md`, and the
  README badge.
- `.github/workflows/codeql.yml` provides static security analysis.
- Pin third-party actions to full commit IDs and retain the release tag in a
  comment. Update pins through reviewed Dependabot pull requests.
- CI may install the Go toolchain and actions, but Go module resolution must stay
  offline and use `vendor/`. Do not add `go mod download` or an online `GOPROXY`.
- Never add repository or provider credentials merely to make integration tests
  pass. Credentialed backend tests belong in an explicitly approved environment.

## Architecture

### Entry Point and Plugin Registration

`zclone.go` is the main entry point. It imports `backend/all` and `cmd/all` which use Go's `init()` pattern to register all backends and commands. Each backend calls `fs.Register()` with a `fs.RegInfo` struct during init.

### Core Interfaces (`fs/`)

The `fs` package defines the core abstractions:
- **`fs.Fs`** (`fs/types.go`): The filesystem interface every backend must implement (List, NewObject, Put, Mkdir, Rmdir).
- **`fs.Object`** (`fs/types.go`): Interface for a file/object (Open, Update, Remove, SetModTime).
- **`fs.Features`** (`fs/features.go`): Optional capabilities a backend can declare (Purge, Copy, Move, DirMove, etc.). Backends set function pointers for operations they support; nil means not supported.
- **`fs.RegInfo`** (`fs/registry.go`): Registration metadata for a backend including its name, config options, and NewFs constructor.

### Backend Structure (`backend/`)

Each backend is a single Go package (e.g., `backend/s3/`, `backend/drive/`). Key conventions:
- Main implementation in a single file (e.g., `s3.go`) - **do not** split into `fs.go`/`object.go`.
- API types go in a separate `api/types.go` file.
- Test file (e.g., `s3_test.go`) uses `fstests.Run()` from `fstest/fstests` for standardized integration tests.
- Register in `backend/all/all.go` via blank import.
- HTTP-based backends should use `lib/rest` for HTTP calls and `fs/fshttp` for the HTTP client.
- Use `lib/dircache` for directory-ID-based remotes, `lib/oauthutil` for OAuth, `lib/pacer` for rate limiting.

### Command Structure (`cmd/`)

Each command is a package under `cmd/` registered in `cmd/all/all.go` via blank import. Commands use cobra via `cmd.Main()`.

### Key Subsystems

- **`fs/operations/`**: Core file operations (Copy, Move, Delete, etc.)
- **`fs/sync/`**: Directory sync logic
- **`fs/march/`**: Parallel directory tree walker used by sync
- **`fs/filter/`**: Include/exclude filtering
- **`fs/accounting/`**: Transfer statistics and bandwidth limiting
- **`fs/config/`**: Configuration file management
- **`vfs/`**: Virtual filesystem layer (used by mount, serve)
- **`libzclone/`**: C-compatible library interface for embedding zclone
- **`fstest/`**: Integration test framework; `fstest/fstests/` has the generic backend test suite

## Commit Message Convention

Prefix with the directory of the change, then a colon: `drive: add team drive support - fixes #885`. For cross-cutting changes use a broader prefix like `fs` or `operations`.

Make the first line of your commit message a summary of the change that a user (not a developer) of zclone would like to read. So write `drive: fix server side copy of big files` instead of `drive: no longer set the MimeType in Move or Copy`. This is important because these lines go into the change log which is read by users.

## Code Commenting Style

Comments describe the code as it is now, for a future reader who has no knowledge of the change that introduced it.

Every exported type, function, field, and constant has a **godoc comment** that starts with its name and is phrased as a present-tense statement of what it is or does (`// Mkdir makes the directory (container, bucket)`)

**Document the contract** callers need - preconditions, what's returned, which sentinel errors are returned and when, and any "shouldn't return an error if it already exists" style caveats - rather than the implementation.

**Keep comments terse**: a single line for most things, with extra paragraphs (separated by blank `//` lines) reserved for genuine subtlety.

Inline comments inside function bodies should **explain why** - a non-obvious API quirk, a workaround, a gotcha, or an ordering constraint - and may cite an external reference (forum thread, vendor docs, RFC) when that's what makes the behaviour non-obvious; skip comments that merely restate what the code plainly does.

Use `FIXME` and `TODO` for known shortcomings.

**Do not write comments that narrate the change itself** or compare against the previous behaviour (no "now we also handle...", "changed to...", "previously this returned...", or references to bug/PR numbers in the code) - that context belongs in the commit message, not in source that will outlive the change.

Code comments should only refer to **current** behaviour - and shouldn't describe behaviour that is trivially deducible from reading the code.

## Linting Configuration

Uses golangci-lint v2 with config in `.golangci.yml`. Enabled linters: errcheck, govet, ineffassign, staticcheck, unused, gocritic, misspell, revive, unconvert. The `goimports` formatter is also enabled.

## Documentation

- Backend option docs come from `Help:` fields in the Go source Options structs, not from markdown files.
- Command docs are in the command source code (e.g., `cmd/ls/ls.go`).
- Don't commit autogenerated doc changes from `make backenddocs` or `make commanddocs`.
- Website docs are in `docs/content/` as markdown, built with Hugo (`make serve` to preview).
