all: build

CWD := $(dir $(realpath $(firstword $(MAKEFILE_LIST))))
GOPATH := $(CWD)
SRC := src/cmd/dividat-driver/main.go
BIN := dividat-driver
BUCKET := dist.dividat.ch

BRANCH = $(shell git rev-parse --abbrev-ref HEAD)
ifeq ($(BRANCH), master)
	CHANNEL = internal
else
	CHANNEL = dev
endif

COMMIT := $(shell git rev-parse HEAD)

TAG := $(shell git describe --exact-match HEAD 2>/dev/null)

PACKAGE_VERSION := $(shell node -p "require('./package.json').version")

RELEASE_DIR = release/$(VERSION)

LINUX = $(RELEASE_DIR)/linux/$(BIN)-linux-amd64-$(VERSION)
DARWIN = $(RELEASE_DIR)/darwin/$(BIN)-darwin-amd64-$(VERSION)
WINDOWS = $(RELEASE_DIR)/win32/$(BIN)-win32-amd64-$(VERSION).exe

deps:
	glide install

.PHONY: build
build:
	GOPATH=$(GOPATH) go build -v -o bin/dividat-driver $(SRC)

crossbuild: check-version $(LINUX) $(DARWIN) $(WINDOWS)

$(LINUX):
	$(call build-os,linux,$@)
	upx $@
	$(call write-checksum,$@)

$(DARWIN):
	$(call build-os,darwin,$@)
	upx $@
	$(call write-checksum,$@)

$(WINDOWS):
	$(call build-os,windows,$@)
	upx $@
	$(call sign-bin,$@)
	$(call write-checksum,$@)

define build-os
	GOOS=$(1) GOARCH=amd64 GOPATH=$(GOPATH) go build -o $(2) $(SRC)
endef

define write-checksum
	cd `dirname $(1)` && shasum `basename $(1)` > `basename $(1)`.sha1
endef

define sign-bin
	osslsigncode sign \
		-pkcs12 $(CODE_SIGNING_CERT) \
		-pass $(CODE_SIGNING_PW) \
		-h sha1 \
		-n "Dividat Driver" \
		-i "https://www.dividat.com/" \
		-in $(1) \
		-out $(1)
endef

.PHONY: check-version
check-version:
ifeq ($(VERSION),)
	$(error VERSION parameter required)
endif
ifneq ($(VERSION), $(TAG))
	$(error VERSION ($(VERSION)) and annotated Git tag of HEAD ($(TAG)) must match)
endif
ifneq ($(VERSION), $(PACKAGE_VERSION))
	$(error VERSION ($(VERSION)) and `package.json` version ($(PACKAGE_VERSION)) must match)
endif

.PHONY: release
release: crossbuild
	aws s3 cp $(RELEASE_DIR) s3://$(BUCKET)/driver/$(CHANNEL)/$(VERSION)/ --recursive --dryrun \
		--acl public-read \
		--cache-control max-age=0 \
		--content-type application/octet-stream
	aws s3api put-object --dryrun \
	  --bucket $(BUCKET) \
		--key driver/$(CHANNEL)/latest/ \
		--website-redirect-location /driver/$(CHANNEL)/$(VERSION)/ \
		--acl public-read \
		--cache-control max-age=0 \

clean:
	rm -rf release/
	go clean
