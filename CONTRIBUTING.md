# Contributing to zclone

This is a short guide on how to contribute things to zclone.

Zclone is a downstream derivative of rclone, not an official rclone project.
Before changing project identity, upstream-derived code, or attribution, read
[FORK.md](FORK.md) and preserve [COPYING](COPYING) and [NOTICE](NOTICE).

## Reporting a bug

If you've got a question or aren't sure if you've found a bug, use the
support channel configured by the project owner instead of filing an issue.

When filing an issue, please include the following information if
possible as well as a description of the problem.  Make sure you test
with the [latest beta of zclone](/):

- Zclone version (e.g. output from `zclone version`)
- Which OS you are using and how many bits (e.g. Windows 10, 64 bit)
- The command you were trying to run (e.g. `zclone copy /tmp remote:tmp`)
- A log of the command with the `-vv` flag (e.g. output from
  `zclone -vv copy /tmp remote:tmp`)
  - if the log contains secrets then edit the file with a text editor first to
    obscure them

## Submitting a new feature or bug fix

If you find a bug that you'd like to fix, or a new feature that you'd
like to implement then please submit a pull request via GitHub.

If it is a big feature, open an issue first so it can be discussed.

To prepare a pull request, fork or clone the Zclone repository hosted by your
project owner.

Then [install Git](https://git-scm.com/downloads) and set your public contribution
[name](https://docs.github.com/en/github/getting-started-with-github/setting-your-username-in-git)
and [email](https://docs.github.com/en/github/setting-up-and-managing-your-github-user-account/setting-your-commit-email-address#setting-your-commit-email-address-in-git).

Next open your terminal, change directory to your preferred folder and initialise
your local zclone project:

```console
git clone <approved-zclone-repository-or-local-path>
cd zclone
```

Note that most of the terminal commands in the rest of this guide must be
executed from the zclone folder created above.

Now [install Go](https://golang.org/doc/install) and verify your installation:

```console
go version
```

Great, you can now compile and execute your own version of zclone:

```console
go build
./zclone version
```

(Note that you can also replace `go build` with `make`, which will include a
more accurate version number in the executable as well as enable you to specify
more build options.) Finally make a branch to add your new feature

```console
git checkout -b my-new-feature
```

And get hacking.

You may like one of the [popular editors/IDE's for Go](https://github.com/golang/go/wiki/IDEsAndTextEditorPlugins)
and a quick view on the zclone [code organisation](#code-organisation).

When ready - test the affected functionality and run the unit tests for the
code you changed

```console
cd folder/with/changed/files
go test -v
```

Note that you may need to make a test remote, e.g. `TestSwift` for some
of the unit tests.

This is typically enough if you made a simple bug fix, otherwise please read
the zclone [testing](#testing) section too.

Make sure you

- Add [unit tests](#testing) for a new feature.
- Add [documentation](#writing-documentation) for a new feature.
- [Commit your changes](#committing-your-changes) using the [commit message guidelines](#commit-messages).

When you are done with that push your changes to GitHub:

```console
git push -u origin my-new-feature
```

and open the GitHub website to [create your pull
request](https://help.github.com/articles/creating-a-pull-request/).

Your changes will then get reviewed and you might get asked to fix some stuff.
If so, then make the changes in the same branch, commit and push your updates to
GitHub.

You may sometimes be asked to [base your changes on the latest master](#basing-your-changes-on-the-latest-master)
or [squash your commits](#squashing-your-commits).

## AI-assisted contributions

You are welcome to use AI coding assistants (Claude Code, Codex, Cursor, Gemini
CLI, and similar) to help write your contribution. Zclone has an
[AGENTS.md](AGENTS.md) file at the top of the repository describing the project's
conventions; point your tool at it so the code it produces matches zclone's
style.

However, the same standard applies to every pull request whether or not a tool
was involved: **you are responsible for the code you submit.** Before you open a
pull request please make sure that:

- You understand every line of the change and can explain why it is correct. If
  a reviewer asks a question about it, you should be able to answer without
  going back to the tool.
- You have actually built and run it. At a minimum `go build` and
  `make quicktest` must pass, and for a backend change you should run the
  [backend tests](#backend-testing) against a real remote where you can.
- The change is a genuine fix or feature that you have verified solves the
  problem, not a plausible-looking guess. Unverified, AI-generated pull requests
  that do not compile, do not pass the tests, invent APIs that do not exist, or
  do not actually do what the description claims waste maintainer time and are
  likely to be closed.
- You have trimmed the comments. AI tools tend to add verbose comments that
  restate what the code plainly does or narrate the change being made; please
  cut these down to match the [code commenting style](AGENTS.md#code-commenting-style).

In short: an AI assistant is a tool to help *you* contribute, not a substitute
for understanding and testing your own work.

## Using Git and GitHub

### Committing your changes

Follow the guideline for [commit messages](#commit-messages) and then:

```console
git checkout my-new-feature      # To switch to your branch
git status                       # To see the new and changed files
git add FILENAME                 # To select FILENAME for the commit
git status                       # To verify the changes to be committed
git commit                       # To do the commit
git log                          # To verify the commit. Use q to quit the log
```

You can modify the message or changes in the latest commit using:

```console
git commit --amend
```

If you amend to commits that have been pushed to GitHub, then you will have to
[replace your previously pushed commits](#replacing-your-previously-pushed-commits).

### Replacing your previously pushed commits

Note that you are about to rewrite the GitHub history of your branch. It is good
practice to involve your collaborators before modifying commits that have been
pushed to GitHub.

Your previously pushed commits are replaced by:

```console
git push --force origin my-new-feature 
```

### Basing your changes on the latest main branch

To base your changes on the latest version of the
[Zclone main branch](https://github.com/zipkindev/zClone):

```console
git checkout main
git fetch origin
git merge --ff-only origin/main
git checkout my-new-feature
git rebase main
```

Here, `origin` means the Zclone repository or your Zclone fork. The separate
`upstream` remote is reserved for rclone; do not merge `upstream/master` as a
routine branch update. Follow [the upstream-porting guidance](FORK.md#tracking-upstream)
when intentionally importing rclone changes.

If you rebase commits that have been pushed to GitHub, then you will have to
[replace your previously pushed commits](#replacing-your-previously-pushed-commits).

### Squashing your commits

To combine your commits into one commit:

```console
git log                          # To count the commits to squash, e.g. the last 2
git reset --soft HEAD~2          # To undo the 2 latest commits
git status                       # To check everything is as expected
```

If everything is fine, then make the new combined commit:

```console
git commit                       # To commit the undone commits as one
```

otherwise, you may roll back using:

```console
git reflog                       # To check that HEAD{1} is your previous state
git reset --soft 'HEAD@{1}'      # To roll back to your previous state
```

If you squash commits that have been pushed to GitHub, then you will have to
[replace your previously pushed commits](#replacing-your-previously-pushed-commits).

Tip: You may like to use `git rebase -i master` if you are experienced or have a
more complex situation.

### Continuous Integration

Zclone is verified on a controlled self-hosted runner using the local toolchain
and committed dependency tree. See `ci/README.md`; hosted CI services are not a
build dependency of this distribution.

## Testing

### Code quality tests

If you install [golangci-lint](https://github.com/golangci/golangci-lint) then
you can run the same tests as get run in the CI which can be very helpful.

You can run them with `make check` or with `golangci-lint run ./...`.

Using these tests ensures that the zclone codebase all uses the same coding
standards. These tests also check for easy mistakes to make (like forgetting
to check an error return).

### Quick testing

zclone's tests are run from the go testing framework, so at the top
level you can run this to run all the tests.

```console
go test -v ./...
```

You can also use `make`, if supported by your platform

```console
make quicktest
```

Run quicktest locally or on the controlled self-hosted runner before proposing
a change.

### Backend testing

zclone contains a mixture of unit tests and integration tests.
Because it is difficult (and in some respects pointless) to test cloud
storage systems by mocking all their interfaces, zclone unit tests can
run against any of the backends.  This is done by making specially
named remotes in the default config file.

If you wanted to test changes in the `drive` backend, then you would
need to make a remote called `TestDrive`.

You can then run the unit tests in the drive directory.  These tests
are skipped if `TestDrive:` isn't defined.

```console
cd backend/drive
go test -v
```

You can then run the integration tests which test all of zclone's
operations.  Normally these get run against the local file system,
but they can be run against any of the remotes.

```console
cd fs/sync
go test -v -remote TestDrive:
go test -v -remote TestDrive: -fast-list

cd fs/operations
go test -v -remote TestDrive:
```

If you want to use the integration test framework to run these tests
altogether with an HTML report and test retries then from the
project root:

```console
go run ./fstest/test_all -backends drive
```

### Full integration testing

If you want to run all the integration tests against all the remotes,
then change into the project root and run

```console
make check
make test
```

The commands require locally provisioned test tools and configured remotes.
`make build_dep` intentionally does not download tools. Run the full suite on
your controlled test infrastructure.

## Code Organisation

Zclone code is organised into a small number of top level directories
with modules beneath.

- backend - the zclone backends for interfacing to cloud providers -
  - all - import this to load all the cloud providers
  - ...providers
- bin - scripts for use while building or maintaining zclone
- cmd - the zclone commands
  - all - import this to load all the commands
  - ...commands
- cmdtest - end-to-end tests of commands, flags, environment variables,...
- docs - the documentation and website
  - content - adjust these docs only, except those marked autogenerated
    or portions marked autogenerated where the corresponding .go file must be
    edited instead, and everything else is autogenerated
    - commands - these are auto-generated, edit the corresponding .go file
- fs - main zclone definitions - minimal amount of code
  - accounting - bandwidth limiting and statistics
  - asyncreader - an io.Reader which reads ahead
  - config - manage the config file and flags
  - driveletter - detect if a name is a drive letter
  - filter - implements include/exclude filtering
  - fserrors - zclone specific error handling
  - fshttp - http handling for zclone
  - fspath - path handling for zclone
  - hash - defines zclone's hash types and functions
  - list - list a remote
  - log - logging facilities
  - march - iterates directories in lock step
  - object - in memory Fs objects
  - operations - primitives for sync, e.g. Copy, Move
  - sync - sync directories
  - walk - walk a directory
- fstest - provides integration test framework
  - fstests - integration tests for the backends
  - mockdir - mocks an fs.Directory
  - mockobject - mocks an fs.Object
  - test_all - Runs integration tests for everything
- graphics - the images used in the website, etc.
- lib - libraries used by the backend
  - atexit - register functions to run when zclone exits
  - dircache - directory ID to name caching
  - oauthutil - helpers for using oauth
  - pacer - retries with backoff and paces operations
  - readers - a selection of useful io.Readers
  - rest - a thin abstraction over net/http for REST
- libzclone - in memory interface to zclone's API for embedding zclone
- vfs - Virtual FileSystem layer for implementing zclone mount and similar

## Writing Documentation

If you are adding a new feature then please update the documentation.

The documentation sources are generally in Markdown format, in conformance
with the CommonMark specification and compatible with GitHub Flavored
Markdown (GFM). The markdown format and style is checked as part of the lint
operation that runs automatically on pull requests, to enforce standards and
consistency. This is based on the [markdownlint](https://github.com/DavidAnson/markdownlint)
tool by David Anson, which can also be integrated into editors so you can
perform the same checks while writing. It generally follows Ciro Santilli's
[Markdown Style Guide](https://cirosantilli.com/markdown-style-guide), which
is good source if you want to know more.

HTML pages, served as website <zclone.org>, are generated from the Markdown,
using [Hugo](https://gohugo.io). Note that when generating the HTML pages,
there is currently used a different algorithm for generating header anchors
than what GitHub uses for its Markdown rendering. For example, in the HTML docs
generated by Hugo any leading `-` characters are ignored, which means when
linking to a header with text `--config string` we therefore need to use the
link `#config-string` in our Markdown source, which will not work in GitHub's
preview where `#--config-string` would be the correct link.

Most of the documentation are written directly in text files with extension
`.md`, mainly within folder `docs/content`. Note that several of such files
are autogenerated (e.g. the command documentation, and `docs/content/flags.md`),
or contain autogenerated portions (e.g. the backend documentation under
`docs/content/commands`). These are marked with an `autogenerated` comment.
The sources of the autogenerated text are usually Markdown formatted text
embedded as string values in the Go source code, so you need to locate these
and edit the `.go` file instead. The `MANUAL.*`, `zclone.1` and other text
files in the root of the repository are also autogenerated. The autogeneration
of files, and the website, will be done during the release process. See the
`make doc` and `make website` targets in the Makefile if you are interested in
how. You don't need to run these when adding a feature.

If you add a new general flag (not for a backend), then document it in
`docs/content/docs.md` - the flags there are supposed to be in
alphabetical order.

If you add a new backend option/flag, then it should be documented in
the source file in the `Help:` field:

- Start with the most important information about the option,
  as a single sentence on a single line.
  - This text will be used for the command-line flag help.
  - It will be combined with other information, such as any default value,
    and the result will look odd if not written as a single sentence.
  - It should end with a period/full stop character, which will be shown
    in docs but automatically removed when producing the flag help.
  - Try to keep it below 80 characters, to reduce text wrapping in the terminal.
- More details can be added in a new paragraph, after an empty line (`"\n\n"`).
  - Like with docs generated from Markdown, a single line break is ignored
    and two line breaks creates a new paragraph.
  - This text will be shown to the user in `zclone config`
    and in the docs (where it will be added by `make backenddocs`,
    normally run some time before next release).
- To create options of enumeration type use the `Examples:` field.
  - Each example value have their own `Help:` field, but they are treated
    a bit different than the main option help text. They will be shown
    as an unordered list, therefore a single line break is enough to
    create a new list item. Also, for enumeration texts like name of
    countries, it looks better without an ending period/full stop character.
- You can run `make backenddocs` to verify the resulting Markdown.
  - This will update the autogenerated sections of the backend docs Markdown
    files under `docs/content`.
  - It requires you to have [Python](https://www.python.org) installed.
  - The `backenddocs` make target runs the Python script `bin/make_backend_docs.py`,
    and you can also run this directly, optionally with the name of a backend
    as argument to only update the docs for a specific backend.
  - **Do not** commit the updated Markdown files. This operation is run as part of
    the release process. Since any manual changes in the autogenerated sections
    of the Markdown files will then be lost, we have a pull request check that
    reports error for any changes within the autogenerated sections. Should you
    have done manual changes outside of the autogenerated sections they must be
    committed, of course.
- You can run `make serve` to verify the resulting website.
  - This will build the website and serve it locally, so you can open it in
    your web browser and verify that the end result looks OK. Check specifically
    any added links, also in light of the note above regarding different algorithms
    for generated header anchors.
  - It requires you to have the [Hugo](https://gohugo.io) tool available.
  - The `serve` make target depends on the `website` target, which runs the
    `hugo` command from the `docs` directory to build the website, and then
    it serves the website locally with an embedded web server using a command
    `hugo server --logLevel info -w --disableFastRender --ignoreCache`, so you
    can run similar Hugo commands directly as well.

When writing documentation for an entirely new backend,
see [backend documentation](#backend-documentation).

If you are updating documentation for a command, you must do that in the
command source code, e.g. `cmd/ls/ls.go`. Write flag help strings as a single
sentence on a single line, without a period/full stop character at the end,
as it will be combined unmodified with other information (such as any default
value).

Note that you can use
[GitHub's online editor](https://help.github.com/en/github/managing-files-in-a-repository/editing-files-in-another-users-repository)
for small changes in the docs which makes it very easy. Just remember the
caveat when linking to header anchors, noted above, which means that GitHub's
Markdown preview may not be an entirely reliable verification of the results.

After your changes have been merged, you can verify them on
[tip.zclone.org](/). This site is updated daily with the
current state of the master branch at 07:00 UTC. The changes will be on the main
[zclone.org](/) site once they have been included in a release.

## Making a release

There are separate instructions for making a release in the RELEASE.md
file.

## Commit messages

Please make the first line of your commit message a summary of the
change that a user (not a developer) of zclone would like to read, and
prefix it with the directory of the change followed by a colon.  The
changelog gets made by looking at just these first lines so make it
good!

If you have more to say about the commit, then enter a blank line and
carry on the description.  Remember to say why the change was needed -
the commit itself shows what was changed.

Writing more is better than less.  Comparing the behaviour before the
change to that after the change is very useful.  Imagine you are
writing to yourself in 12 months time when you've forgotten everything
about what you just did and you need to get up to speed quickly.

If the change fixes an issue then write `Fixes #1234` in the commit
message.  This can be on the subject line if it will fit.  If you
don't want to close the associated issue just put `#1234` and the
change will get linked into the issue.

Here is an example of a short commit message:

```text
drive: add team drive support - fixes #885
```

And here is an example of a longer one:

```text
mount: fix hang on errored upload

In certain circumstances, if an upload failed then the mount could hang
indefinitely. This was fixed by closing the read pipe after the Put
completed.  This will cause the write side to return a pipe closed
error fixing the hang.

Fixes #1498
```

## Adding a dependency

Review new dependencies in a controlled environment, then bring the reviewed
source, `go.mod`, `go.sum`, and regenerated `vendor/` tree into Zclone together.
Run `make source-manifest` in the same change. The normal Zclone build never
downloads a module.

## Updating a dependency

Follow the same controlled-import process as for a new dependency. Check in the
module metadata, vendored source, and updated dependency manifest together.

## Updating all the dependencies

`make update` is intentionally disabled. Dependency updates are performed in a
controlled import environment and validated locally with `make verify-local`.

## Updating a backend

If you update a backend then please run the unit tests and the
integration tests for that backend.

Assuming the backend is called `remote`, make create a config entry
called `TestRemote` for the tests to use.

Now `cd remote` and run `go test -v` to run the unit tests.

Then `cd fs` and run `go test -v -remote TestRemote:` to run the
integration tests.

The next section goes into more detail about the tests.

## Writing a new backend

Choose a name.  The docs here will use `remote` as an example.

Note that in zclone terminology a file system backend is called a
remote or an fs.

### Research

- Look at the interfaces defined in `fs/types.go`
- Study one or more of the existing remotes

### Getting going

- Create `backend/remote/remote.go` (copy this from a similar remote)
  - box is a good one to start from if you have a directory-based remote (and
    shows how to use the directory cache)
  - b2 is a good one to start from if you have a bucket-based remote
- Add your remote to the imports in `backend/all/all.go`
- HTTP based remotes are easiest to maintain if they use zclone's
  [lib/rest](https://pkg.go.dev/zclone/lib/rest) module, but
  if there is a really good Go SDK from the provider then use that instead.
- Try to implement as many optional methods as possible as it makes the remote
  more usable.
- Use [lib/encoder](https://pkg.go.dev/zclone/lib/encoder) to
  make sure we can encode any path name and `zclone info` to help determine the
  encodings needed
  - `zclone purge -v TestRemote:zclone-info`
  - `zclone test info --all --remote-encoding None -vv --write-json remote.json TestRemote:zclone-info`
  - `go run cmd/test/info/internal/build_csv/main.go -o remote.csv remote.json`
  - open `remote.csv` in a spreadsheet and examine

### Guidelines for a speedy merge

- **Do** use [lib/rest](https://pkg.go.dev/zclone/lib/rest)
  if you are implementing a REST like backend and parsing XML/JSON in the backend.
- **Do** use zclone's Client or Transport from [fs/fshttp](https://pkg.go.dev/zclone/fs/fshttp)
  if your backend is HTTP based - this adds features like `--dump bodies`,
  `--tpslimit`, `--user-agent` without you having to code anything!
- **Do** follow your example backend exactly - use the same code order, function
  names, layout, structure. **Don't** move stuff around and **Don't** delete the
  comments.
- **Do not** split your backend up into `fs.go` and `object.go` (there are a few
  backends like that - don't follow them!)
- **Do** put your API type definitions in a separate file - by preference `api/types.go`
- **Remember** we have >50 backends to maintain so keeping them as similar as
  possible to each other is a high priority!

### Unit tests

- Create a config entry called `TestRemote` for the unit tests to use
- Create a `backend/remote/remote_test.go` - copy and adjust your example remote
- Make sure all tests pass with `go test -v`

### Integration tests

- Add your backend to `fstest/test_all/config.yaml`
  - Once you've done that then you can use the integration test framework from
    the project root:
  - `go run ./fstest/test_all -backends remote`

Or if you want to run the integration tests manually:

- Make sure integration tests pass with
  - `cd fs/operations`
  - `go test -v -remote TestRemote:`
  - `cd fs/sync`
  - `go test -v -remote TestRemote:`
- If your remote defines `ListR` check with this also
  - `go test -v -remote TestRemote: -fast-list`

Before a new or changed backend can be merged we require:

- A clean run of `go run ./fstest/test_all -backends remote` (please include the
  result in your pull request).
- A test account for the backend so the maintainers can add it to the
  [integration test server](https://integration.zclone.org) and keep the backend
  working as zclone evolves. A backend that we cannot test is likely to break and
  may be removed.

See the [testing](#testing) section for more information on integration tests.

### Backend documentation

Add your backend to the docs.  Keep lists of remotes in alphabetical
order of full name of remote (e.g. `drive` is ordered as `Google
Drive`) but with the local file system last.

First add a data file about your backend in
`docs/data/backends/remote.yaml` - this is used to build the overview
tables and the tiering info.

- Create it with: `bin/manage_backends.py create docs/data/backends/remote.yaml`
- Edit it to fill in the blanks. Look at the [tiers docs](//tiers/).
- Run this command to fill in the features: `bin/manage_backends.py features docs/data/backends/remote.yaml`

Next edit these files:

- `README.md` - main GitHub page
- `docs/content/remote.md` - main docs page (note the backend options are
  automatically added to this file with `make backenddocs`)
  - make sure this has the `autogenerated options` comments in (see your
    reference backend docs)
  - update them in your backend with `bin/make_backend_docs.py remote`
- `docs/content/docs.md` - list of remotes in config section
- `docs/content/_index.md` - front page of zclone.org
- `docs/layouts/chrome/navbar.html` - add it to the website navigation
- `bin/make_manual.py` - add the page to the `docs` constant

Once you've written the docs, run `make serve` and check they look OK
in the web browser and the links (internal and external) all work.

## Adding a new s3 provider

[Please see the guide in the S3 backend directory](backend/s3/README.md).

## Writing a plugin

New features (backends, commands) can also be added "out-of-tree", through Go
plugins. Changes will be kept in a dynamically loaded file instead of being
compiled into the main binary. This is useful if you can't merge your changes
upstream or don't want to maintain a fork of zclone.

### Usage

- Naming
  - Plugins names must have the pattern `libzcloneplugin_KIND_NAME.so`.
  - `KIND` should be one of `backend`, `command` or `bundle`.
  - Example: A plugin with backend support for PiFS would be called
    `libzcloneplugin_backend_pifs.so`.
- Loading
  - Supported on macOS & Linux as of now. ([Go issue for Windows support](https://github.com/golang/go/issues/19282))
  - Supported on zclone v1.50 or greater.
  - All plugins in the folder specified by variable `$ZCLONE_PLUGIN_PATH` are loaded.
  - If this variable doesn't exist, plugin support is disabled.
  - Plugins must be compiled against the exact version of zclone to work.
    (The zclone used during building the plugin must be the same as the source
    of zclone)

### Building

To turn your existing additions into a Go plugin, move them to an external repository
and change the top-level package name to `main`.

Check `zclone --version` and make sure that the plugin's zclone dependency and
host Go version match.

Then, run `go build -buildmode=plugin -o PLUGIN_NAME.so .` to build the plugin.

[Go reference](https://godoc.org/zclone/lib/plugin)

## Keeping a backend or command out of tree

Zclone was designed to be modular so it is very easy to keep a backend
or a command out of the main zclone source tree.

So for example if you had a backend which accessed your proprietary
systems or a command which was specialised for your needs you could
add them out of tree.

This may be easier than using a plugin and is supported on all
platforms not just macOS and Linux.

This is explained further in </_out_of_tree_example>
which has an example of an out of tree backend `ram` (which is a
renamed version of the `memory` backend).
