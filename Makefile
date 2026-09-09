SHELL = bash
# Branch we are working on
BRANCH := $(or $(BUILD_SOURCEBRANCHNAME),$(lastword $(subst /, ,$(GITHUB_REF))),$(shell git rev-parse --abbrev-ref HEAD))
# Tag of the current commit, if any.  If this is not "" then we are building a release
RELEASE_TAG := $(shell git tag -l --points-at HEAD)
# Version of last release (may not be on this branch)
VERSION := $(shell cat VERSION)
# Last tag on this branch
LAST_TAG := $(shell git describe --tags --abbrev=0)
# Next version
NEXT_VERSION := $(shell echo $(VERSION) | awk -F. -v OFS=. '{print $$1,$$2+1,0}')
NEXT_PATCH_VERSION := $(shell echo $(VERSION) | awk -F. -v OFS=. '{print $$1,$$2,$$3+1}')
# If we are working on a release, override branch to master
ifdef RELEASE_TAG
	BRANCH := master
	LAST_TAG := $(shell git describe --abbrev=0 --tags $(VERSION)^)
endif
TAG_BRANCH := .$(subst /,_,$(BRANCH))
BRANCH_PATH := branch/$(BRANCH)/
# If building HEAD or master then unset TAG_BRANCH and BRANCH_PATH
ifeq ($(subst HEAD,,$(subst master,,$(BRANCH))),)
	TAG_BRANCH :=
	BRANCH_PATH :=
endif
# Make version suffix -beta.NNNN.CCCCCCCC (N=Commit number, C=Commit)
VERSION_SUFFIX := -beta.$(shell git rev-list --count HEAD).$(shell git show --no-patch --no-notes --pretty='%h' HEAD)
# TAG is current version + commit number + commit + branch
TAG := $(VERSION)$(VERSION_SUFFIX)$(TAG_BRANCH)
ifdef RELEASE_TAG
	TAG := $(RELEASE_TAG)
endif
# A branch name may contain slashes, which are valid in a release path but not
# in a single archive filename.
ARTIFACT_TAG := $(subst /,_,$(TAG))
GO_VERSION := $(shell go version)
GO_OS := $(shell go env GOOS)
ifdef BETA_SUBDIR
	BETA_SUBDIR := /$(BETA_SUBDIR)
endif
BETA_PATH := $(BRANCH_PATH)$(TAG)$(BETA_SUBDIR)
# Release endpoints are intentionally unset in the local distribution.
# Set both variables explicitly when publishing a Zclone release.
BETA_URL ?=
BETA_UPLOAD_ROOT ?=
BETA_UPLOAD := $(BETA_UPLOAD_ROOT)/$(BETA_PATH)
# Release destinations are deliberately unset.  Supply these variables when
# publishing from a controlled release environment.
WEBSITE_DESTINATION ?=
DOWNLOAD_DESTINATION ?=
PUBLIC_BETA_DESTINATION ?=
PRIVATE_BETA_DESTINATION ?=
# Pass in GOTAGS=xyz on the make command line to set build tags
ifdef GOTAGS
BUILDTAGS=-tags "$(GOTAGS)"
LINTTAGS=--build-tags "$(GOTAGS)"
endif
LDFLAGS=--ldflags "-s -X zclone/fs.Version=$(TAG)"
GO_OFFLINE=GOPROXY=off GOSUMDB=off GOFLAGS=-mod=vendor

.PHONY: zclone sign release-sign macos-installer sbom source-manifest verify-sources verify-local debt-report gui-dist verify-gui-dist test_all vars version fetch-gui fetch-gui-and-commit

zclone:
ifeq ($(GO_OS),windows)
	$(GO_OFFLINE) go run bin/resource_windows.go -version $(TAG) -syso resource_windows_`go env GOARCH`.syso
endif
	mkdir -p build
	$(GO_OFFLINE) go build -buildvcs=false -trimpath -v -o build/zclone $(LDFLAGS) $(BUILDTAGS) $(BUILD_ARGS)
ifeq ($(GO_OS),windows)
	rm resource_windows_`go env GOARCH`.syso
endif

# CODESIGN_IDENTITY may be set to an Apple Developer certificate identity.
CODESIGN_IDENTITY ?= -
sign: zclone
	codesign --force --sign "$(CODESIGN_IDENTITY)" --identifier org.zclone.zclone build/zclone
	codesign --verify --strict --verbose=2 build/zclone

# release-sign rejects the ad-hoc identity. It is for distributable macOS builds.
release-sign: zclone
	@test "$(CODESIGN_IDENTITY)" != "-" || (echo "Set CODESIGN_IDENTITY to an Apple Developer certificate" && exit 2)
	codesign --force --options runtime --timestamp --sign "$(CODESIGN_IDENTITY)" --identifier org.zclone.zclone build/zclone
	codesign --verify --strict --verbose=2 build/zclone

# Builds a macOS package that installs /usr/local/bin/zclone and /etc/paths.d/zclone.
macos-installer: sign
	INSTALLER_SIGN_IDENTITY="$(INSTALLER_SIGN_IDENTITY)" ./installer/macos/build-installer.sh

sbom:
	mkdir -p build
	$(GO_OFFLINE) go run bin/make_sbom.go

source-manifest:
	$(GO_OFFLINE) go run bin/make_sbom.go -checksums DEPENDENCY_MANIFEST.sha256

verify-sources:
	$(GO_OFFLINE) go run bin/make_sbom.go -checksums build/zclone.sources.verify.sha256
	cmp -s DEPENDENCY_MANIFEST.sha256 build/zclone.sources.verify.sha256 || (echo "Dependency manifest does not match; review changes then run make source-manifest" && exit 2)

verify-local:
	$(SHELL) ./bin/verify-local.sh

# debt-report lists open implementation limitations for release triage.
debt-report:
	mkdir -p build
	@rg -n --glob '*.go' 'TODO|FIXME' backend cmd fs lib vfs > build/TECHNICAL_DEBT.txt || true

gui-dist:
	cd cmd/gui/dist && zip -X -q ../dist.zip index.html icon.svg

verify-gui-dist:
	@tmp_dir=$$(mktemp -d); trap 'rm -rf "$$tmp_dir"' EXIT; \
		unzip -qq cmd/gui/dist.zip -d "$$tmp_dir"; \
		diff -ru cmd/gui/dist "$$tmp_dir"

fetch-gui:
	$(SHELL) ./bin/fetch-gui-dist.sh

fetch-gui-and-commit:
	$(SHELL) ./bin/fetch-gui-dist.sh --commit

test_all:
	mkdir -p build
	$(GO_OFFLINE) go build $(LDFLAGS) $(BUILDTAGS) $(BUILD_ARGS) -o build/test_all ./fstest/test_all

vars:
	@echo SHELL="'$(SHELL)'"
	@echo BRANCH="'$(BRANCH)'"
	@echo TAG="'$(TAG)'"
	@echo VERSION="'$(VERSION)'"
	@echo GO_VERSION="'$(GO_VERSION)'"
	@echo BETA_URL="'$(BETA_URL)'"

btest:
	@test -n "$(BETA_URL)" || (echo "BETA_URL must be set by the release environment" && exit 2)
	@echo "[$(TAG)]($(BETA_URL)) on branch $(BRANCH) (uploaded in 15-30 mins)" | xclip -r -sel clip
	@echo "Copied markdown of beta release to clip board"

btesth:
	@test -n "$(BETA_URL)" || (echo "BETA_URL must be set by the release environment" && exit 2)
	@echo "<a href="$(BETA_URL)">$(TAG)</a> on branch $(BRANCH) (uploaded in 15-30 mins)" | xclip -r -sel clip -t text/html
	@echo "Copied beta release in HTML to clip board"

version:
	@echo '$(TAG)'

# Full suite of integration tests
test:	zclone test_all
	-build/test_all 2>&1 | tee build/test_all.log
	@echo "Written logs in build/test_all.log"

# Quick test
#
# The timeout is raised above the go test default of 10m as the
# cmd/gitannex end to end tests can take longer than that on slow CI
# runners.
quicktest:
	ZCLONE_CONFIG="/notfound" $(GO_OFFLINE) go test $(LDFLAGS) $(BUILDTAGS) -timeout 20m ./...

racequicktest:
	ZCLONE_CONFIG="/notfound" $(GO_OFFLINE) go test $(LDFLAGS) $(BUILDTAGS) -cpu=2 -race -timeout 20m ./...

compiletest:
	ZCLONE_CONFIG="/notfound" $(GO_OFFLINE) go test $(LDFLAGS) $(BUILDTAGS) -run XXX ./...

# Do source code quality checks
check:	zclone
	@echo "-- START CODE QUALITY REPORT -------------------------------"
	@golangci-lint run $(LINTTAGS) ./...
	@bin/markdown-lint
	@echo "-- END CODE QUALITY REPORT ---------------------------------"

# Get the build dependencies
build_dep:
	@echo "Online dependency installation is disabled. Provide approved tools locally."
	@exit 2

# Get the release dependencies we only install on linux
release_dep_linux:
	@echo "Online dependency installation is disabled. Provide nfpm locally."
	@exit 2

# Update dependencies
showupdates:
	@echo "Online dependency checks are disabled in the local Zclone distribution."
	@exit 2

# Update direct dependencies only
updatedirect:
	@echo "Online dependency updates are disabled in the local Zclone distribution."
	@exit 2

# Update direct dependencies only but won't update the `go` line in go.mod
updatedirectnoupgrade:
	@echo "Online dependency updates are disabled in the local Zclone distribution."
	@exit 2

# Update direct and indirect dependencies and test dependencies
update:
	@echo "Online dependency updates are disabled in the local Zclone distribution."
	@exit 2

# Tidy the module dependencies
tidy:
	@echo "Online dependency updates are disabled in the local Zclone distribution."
	@exit 2

doc:
	@test -f zclone.1 || (echo "zclone.1 is missing" && exit 2)

zclone.1:	MANUAL.md bin/make_man.go
	$(GO_OFFLINE) go run bin/make_man.go MANUAL.md zclone.1

MANUAL.md:	bin/make_manual.py docs/content/*.md commanddocs
	./bin/make_manual.py

MANUAL.html:	MANUAL.md
	pandoc -s --from markdown-smart --to html MANUAL.md -o MANUAL.html

MANUAL.txt:	MANUAL.md
	pandoc -s --from markdown-smart --to plain MANUAL.md -o MANUAL.txt

commanddocs: zclone
	$(GO_OFFLINE) go generate ./lib/transform
	$(GO_OFFLINE) go generate ./cmd/bisync
	@doc_home=$$(mktemp -d); trap 'rm -rf "$$doc_home"' EXIT; \
		XDG_CACHE_HOME="$$doc_home/.cache" XDG_CONFIG_HOME="$$doc_home/.config" HOME="$$doc_home" USER="zclone-docs" build/zclone gendocs --config=/notfound docs/content/
	$(GO_OFFLINE) go run bin/make_bisync_docs.go ./docs/content/

backenddocs: zclone bin/make_backend_docs.py
	@doc_home=$$(mktemp -d); trap 'rm -rf "$$doc_home"' EXIT; \
		XDG_CACHE_HOME="$$doc_home/.cache" XDG_CONFIG_HOME="$$doc_home/.config" HOME="$$doc_home" USER="zclone-docs" ./bin/make_backend_docs.py

rcdocs: zclone
	@echo "RC documentation is maintained in docs/content/rc.md; automatic RC doc generation is disabled."

install: zclone
	install -d ${DESTDIR}/usr/bin
	install build/zclone ${DESTDIR}/usr/bin

clean:
	go clean ./...
	find . -name \*~ | xargs -r rm -f
	rm -rf build docs/public
	rm -f zclone fs/operations/operations.test fs/sync/sync.test fs/test_all.log test.log

website:
	rm -rf docs/public
	cd docs && hugo
	@if grep -R "raw HTML omitted" docs/public ; then echo "ERROR: found unescaped HTML - fix the markdown source" ; fi

upload_website:	website
	@test -n "$(WEBSITE_DESTINATION)" || (echo "WEBSITE_DESTINATION must be set by the release environment" && exit 2)
	zclone -v sync docs/public "$(WEBSITE_DESTINATION)"

upload_test_website:	website
	@test -n "$(WEBSITE_DESTINATION)" || (echo "WEBSITE_DESTINATION must be set by the release environment" && exit 2)
	zclone -P sync docs/public "$(WEBSITE_DESTINATION)"

validate_website: website
	find docs/public -type f -name "*.html" | xargs tidy --mute-id yes -errors --gnu-emacs yes --drop-empty-elements no --warn-proprietary-attributes no --mute MISMATCHED_ATTRIBUTE_WARN

tarball:
	git archive -9 --format=tar.gz --prefix=zclone-$(ARTIFACT_TAG)/ -o build/zclone-$(ARTIFACT_TAG).tar.gz $(TAG)

vendorball:
	mkdir -p build
	tar -zcf build/zclone-$(ARTIFACT_TAG)-vendor.tar.gz vendor

sign_upload:
	cd build && shasum -a 256 zclone* | gpg --clearsign > SHA256SUMS
	cd build && shasum -a 512 zclone* | gpg --clearsign > SHA512SUMS

check_sign:
	cd build && gpg --verify SHA256SUMS && gpg --decrypt SHA256SUMS | shasum -a 256 -c
	cd build && gpg --verify SHA512SUMS && gpg --decrypt SHA512SUMS | shasum -a 512 -c

upload:
	@test -n "$(DOWNLOAD_DESTINATION)" || (echo "DOWNLOAD_DESTINATION must be set by the release environment" && exit 2)
	zclone -P copy build/ "$(DOWNLOAD_DESTINATION)/$(TAG)"
	zclone lsf build --files-only --include '*.{zip,deb,rpm}' --include version.txt | xargs -i bash -c 'i={}; j="$$i"; [[ $$i =~ (.*)(-v[0-9\.]+-)(.*) ]] && j=$${BASH_REMATCH[1]}-current-$${BASH_REMATCH[3]}; zclone copyto -v "$(DOWNLOAD_DESTINATION)/$(TAG)/$$i" "$(DOWNLOAD_DESTINATION)/$$j"'

upload_github:
	@echo "GitHub publishing is disabled in the local Zclone distribution."
	@exit 2

cross:	doc
	$(GO_OFFLINE) go run bin/cross-compile.go -release current $(BUILD_FLAGS) $(BUILDTAGS) $(BUILD_ARGS) $(TAG)

beta:
	@test -n "$(PUBLIC_BETA_DESTINATION)" || (echo "PUBLIC_BETA_DESTINATION must be set by the release environment" && exit 2)
	$(GO_OFFLINE) go run bin/cross-compile.go $(BUILD_FLAGS) $(BUILDTAGS) $(BUILD_ARGS) $(TAG)
	zclone -v copy build/ "$(PUBLIC_BETA_DESTINATION)/$(TAG)"
	@echo "Beta release uploaded to $(PUBLIC_BETA_DESTINATION)/$(TAG)/"

privatebeta:
	@test -n "$(PRIVATE_BETA_DESTINATION)" || (echo "PRIVATE_BETA_DESTINATION must be set by the release environment" && exit 2)
	$(GO_OFFLINE) go run bin/cross-compile.go $(BUILD_FLAGS) $(BUILDTAGS) $(BUILD_ARGS) -include '^(darwin|windows|linux)/(arm64|amd64)$$' $(TAG)
	zclone -Pv copy build/ "$(PRIVATE_BETA_DESTINATION)/beta/$(TAG)"
	@echo "Private beta release uploaded to $(PRIVATE_BETA_DESTINATION)/beta/$(TAG)/"
	zclone link "$(PRIVATE_BETA_DESTINATION)/beta/$(TAG)"

log_since_last_release:
	git log $(LAST_TAG)..

compile_all:
	$(GO_OFFLINE) go run bin/cross-compile.go -compile-only $(BUILD_FLAGS) $(BUILDTAGS) $(BUILD_ARGS) $(TAG)

ci_upload:
	@test -n "$(BETA_UPLOAD_ROOT)" || (echo "BETA_UPLOAD_ROOT must be set by the release environment" && exit 2)
	sudo chown -R $$USER build
	find build -type l -delete
	gzip -r9v build
	./zclone --no-check-dest --config bin/ci.zclone.conf -v copy build/ $(BETA_UPLOAD)/testbuilds
ifeq ($(or $(BRANCH_PATH),$(RELEASE_TAG)),)
	./zclone --no-check-dest --config bin/ci.zclone.conf -v copy build/ $(BETA_UPLOAD_ROOT)/test/testbuilds-latest
endif
	@echo Beta release ready at $(BETA_URL)/testbuilds

ci_beta:
	@test -n "$(BETA_UPLOAD_ROOT)" || (echo "BETA_UPLOAD_ROOT must be set by the release environment" && exit 2)
	git log $(LAST_TAG).. > /tmp/git-log.txt
	$(GO_OFFLINE) go run bin/cross-compile.go -release beta-latest -git-log /tmp/git-log.txt $(BUILD_FLAGS) $(BUILDTAGS) $(BUILD_ARGS) $(TAG)
	zclone --no-check-dest --config bin/ci.zclone.conf -v copy --exclude '*beta-latest*' build/ $(BETA_UPLOAD)
ifeq ($(or $(BRANCH_PATH),$(RELEASE_TAG)),)
	zclone --no-check-dest --config bin/ci.zclone.conf -v copy --include '*beta-latest*' --include version.txt build/ $(BETA_UPLOAD_ROOT)$(BETA_SUBDIR)
endif
	@echo Beta release ready at $(BETA_URL)

# Fetch approved build artifacts from the configured release storage.
fetch_binaries:
	@test -n "$(BETA_UPLOAD_ROOT)" || (echo "BETA_UPLOAD_ROOT must be set by the release environment" && exit 2)
	zclone -P sync --exclude "/testbuilds/**" --delete-excluded $(BETA_UPLOAD) build/

serve:	website
	cd docs && hugo server --logLevel info -w --disableFastRender --ignoreCache

tag:	retag doc
	bin/make_changelog.py $(LAST_TAG) $(VERSION) > docs/content/changelog.md.new
	mv docs/content/changelog.md.new docs/content/changelog.md
	@echo "Edit the new changelog in docs/content/changelog.md"
	@echo "Then commit all the changes"
	@echo git commit -m \"Version $(VERSION)\" -a -v
	@echo "And finally run make retag before make cross, etc."

retag:
	@echo "Version is $(VERSION)"
	git tag -f -s -m "Version $(VERSION)" $(VERSION)

startdev:
	@echo "Version is $(VERSION)"
	@echo "Next version is $(NEXT_VERSION)"
	echo -e "package fs\n\n// VersionTag of zclone\nvar VersionTag = \"$(NEXT_VERSION)\"\n" | gofmt > fs/versiontag.go
	echo -n "$(NEXT_VERSION)" > docs/layouts/partials/version.html
	echo "$(NEXT_VERSION)" > VERSION
	git commit -m "Start $(NEXT_VERSION)-DEV development" fs/versiontag.go VERSION docs/layouts/partials/version.html

startstable:
	@echo "Version is $(VERSION)"
	@echo "Next stable version is $(NEXT_PATCH_VERSION)"
	echo -e "package fs\n\n// VersionTag of zclone\nvar VersionTag = \"$(NEXT_PATCH_VERSION)\"\n" | gofmt > fs/versiontag.go
	echo -n "$(NEXT_PATCH_VERSION)" > docs/layouts/partials/version.html
	echo "$(NEXT_PATCH_VERSION)" > VERSION
	git commit -m "Start $(NEXT_PATCH_VERSION)-DEV development" fs/versiontag.go VERSION docs/layouts/partials/version.html

winzip:
	zip -9 zclone-$(TAG).zip zclone.exe

# docker volume plugin
PLUGIN_USER ?= zclone
PLUGIN_TAG ?= latest
PLUGIN_BASE_TAG ?= latest
PLUGIN_ARCH ?= amd64
PLUGIN_IMAGE := $(PLUGIN_USER)/docker-volume-zclone:$(PLUGIN_TAG)
PLUGIN_BASE := $(PLUGIN_USER)/zclone:$(PLUGIN_BASE_TAG)
PLUGIN_BUILD_DIR := ./build/docker-plugin
PLUGIN_CONTRIB_DIR := ./contrib/docker-plugin/managed

docker-plugin-create:
	docker buildx inspect |grep -q /${PLUGIN_ARCH} || (echo "Docker buildx support for ${PLUGIN_ARCH} is required locally" && exit 2)
	rm -rf ${PLUGIN_BUILD_DIR}
	docker buildx build \
		--no-cache \
		--build-arg BASE_IMAGE=${PLUGIN_BASE} \
		--platform linux/${PLUGIN_ARCH} \
		--output ${PLUGIN_BUILD_DIR}/rootfs \
		${PLUGIN_CONTRIB_DIR}
	cp ${PLUGIN_CONTRIB_DIR}/config.json ${PLUGIN_BUILD_DIR}
	docker plugin rm --force ${PLUGIN_IMAGE} 2>/dev/null || true
	docker plugin create ${PLUGIN_IMAGE} ${PLUGIN_BUILD_DIR}

docker-plugin-push:
	docker plugin push ${PLUGIN_IMAGE}
	docker plugin rm ${PLUGIN_IMAGE}

docker-plugin: docker-plugin-create docker-plugin-push
