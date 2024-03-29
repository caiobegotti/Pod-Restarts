# Copyright 2020 Caio Begotti
# Copyright 2019 Cornelius Weig (based on the Makefile from ketall)
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# defaults for a better make UX
MAKEFLAGS += --warn-undefined-variables
MAKEFLAGS += --no-builtin-rules
SHELL := bash
.ONESHELL:
.SHELLFLAGS := -eu -o pipefail -c
.DELETE_ON_ERROR:
.DEFAULT_GOAL := help

export GO111MODULE ?= on
export GOARCH      ?= amd64
export CGO_ENABLED ?= 0

PROJECT   ?= pod-restarts
REPOPATH  ?= github.com/caiobegotti/$(PROJECT)
COMMIT    := $(shell git rev-parse HEAD)
VERSION   ?= $(shell git describe --always --tags --dirty="-WIP")
GOOS      ?= $(shell go env GOOS)
GOPATH    ?= $(shell go env GOPATH)

BUILDDIR  := out
PLATFORMS ?= darwin/amd64 windows/amd64 linux/amd64
DISTFILE  := $(BUILDDIR)/$(VERSION).tar.gz
ASSETS     := $(BUILDDIR)/pod-restarts-$(GOARCH)-darwin.tar.gz $(BUILDDIR)/pod-restarts-$(GOARCH)-linux.tar.gz $(BUILDDIR)/pod-restarts-$(GOARCH)-windows.zip
CHECKSUMS  := $(patsubst %,%.sha256,$(ASSETS))

VERSION_PACKAGE := $(REPOPATH)/pkg/pod-restarts/version

DATE_FMT = %Y-%m-%dT%H:%M:%SZ
ifdef SOURCE_DATE_EPOCH
    # GNU and BSD date require different options for a fixed date
    BUILD_DATE ?= $(shell date -u -d "@$(SOURCE_DATE_EPOCH)" "+$(DATE_FMT)" 2>/dev/null || date -u -r "$(SOURCE_DATE_EPOCH)" "+$(DATE_FMT)" 2>/dev/null)
else
    BUILD_DATE ?= $(shell date "+$(DATE_FMT)")
endif
GO_LDFLAGS :="-s -w
GO_LDFLAGS += -X $(VERSION_PACKAGE).version=$(VERSION)
GO_LDFLAGS += -X $(VERSION_PACKAGE).buildDate=$(BUILD_DATE)
GO_LDFLAGS += -X $(VERSION_PACKAGE).gitCommit=$(COMMIT)
GO_LDFLAGS +="

ifdef ZOPFLI
  COMPRESS:=zopfli -c
else
  COMPRESS:=gzip --best -k -c
endif

GO_FILES  := $(shell find . -type f -name '*.go')

.PHONY: all
all: clean lint test dev ## clean, lint, test and build a dev binary

.PHONY: test
test:
	go test ./...

.PHONY: help
help:
	@# from https://news.ycombinator.com/item?id=21812897
	@echo -e 'valid make targets:\n'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z0-9_-]+:.*?## / {printf "%-10s (%s)\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: coverage
coverage: $(BUILDDIR) ## run unit tests with coverage
	go test -coverprofile=$(BUILDDIR)/coverage.txt -covermode=atomic ./...

.PHONY: dev
dev: CGO_ENABLED := 1
dev: GO_LDFLAGS := $(subst -s -w,,$(GO_LDFLAGS))
dev: ## build the binary for the current platform
	go build -race -ldflags $(GO_LDFLAGS) -o pod-restarts $(REPOPATH)/cmd/plugin

build: $(GO_FILES) $(BUILDDIR) ## build binaries for all supported platforms
	gox -osarch="$(PLATFORMS)" -ldflags $(GO_LDFLAGS) -output="out/pod-restarts-{{.Arch}}-{{.OS}}" $(REPOPATH)/cmd/plugin

.PHONY: lint
lint: ## run golang fmt and vet
	go fmt ./pkg/... ./cmd/...
	go vet ./pkg/... ./cmd/...

.PRECIOUS: %.zip
%.zip: %.exe
	cp LICENSE $(BUILDDIR) && \
	cd $(BUILDDIR) && \
	zip $(patsubst $(BUILDDIR)/%, %, $@) $(BUILDDIR)/LICENSE $(patsubst $(BUILDDIR)/%, %, $<)

.PRECIOUS: %.gz
%.gz: %
	$(COMPRESS) "$<" > "$@"

%.tar: %
	cp LICENSE $(BUILDDIR)
	tar cf "$@" -C $(BUILDDIR) $(BUILDDIR)/LICENSE $(patsubst $(BUILDDIR)/%,%,$^)

$(BUILDDIR):
	mkdir -p "$@"

%.sha256: %
	shasum -a 256 $< > $@

.INTERMEDIATE: $(DISTFILE:.gz=)
$(DISTFILE:.gz=): $(BUILDDIR)
	git archive --prefix="pod-restarts-$(VERSION)/" --format=tar HEAD > "$@"

.PHONY: deploy
deploy: $(CHECKSUMS)
	$(RM) $(BUILDDIR)/LICENSE

.PHONY: dist
dist: $(DISTFILE) ## create a tar archive of the source code

.PHONY: compact
compact: build
	@cp LICENSE $(BUILDDIR) && \
	cd $(BUILDDIR) && \
	tar cvvfz pod-restarts-amd64-darwin.tar.gz pod-restarts-amd64-darwin LICENSE && \
	tar cvvfz pod-restarts-amd64-linux.tar.gz pod-restarts-amd64-linux LICENSE && \
	zip pod-restarts-amd64-windows.exe.zip pod-restarts-amd64-windows.exe LICENSE

.PHONY: release
release: compact ## build, compact, sha256 files for a release
	@openssl sha256 out/*.tar.gz out/*.zip

ifeq ($(OS),Windows_NT)
    THIS_OS := Windows
else
    UNAME_S := $(shell uname -s)
    ifeq ($(UNAME_S),Linux)
        THIS_OS += -D linux
    endif
    ifeq ($(UNAME_S),Darwin)
        THIS_OS += darwin
    endif
endif

.PHONE: install
install: build
	sudo cp -vaf out/pod-restarts-amd64-$(THIS_OS) $(HOME)/.krew/bin/kubectl-pod_restarts

.PHONY: clean
clean: ## clean up build directory and binaries files
	$(RM) -r $(BUILDDIR) pod-restarts

$(BUILDDIR)/pod-restarts-amd64-linux: build
$(BUILDDIR)/pod-restarts-amd64-darwin: build
$(BUILDDIR)/pod-restarts-amd64-windows.exe: build
