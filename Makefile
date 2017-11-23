all: build

CWD := $(dir $(realpath $(firstword $(MAKEFILE_LIST))))
GOPATH := $(CWD)
SRC := src/cmd/dividat-driver/main.go
BIN := dividat-driver
BUCKET := dist-test.dividat.ch

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
LATEST = release/latest.json

deps:
	cd src && glide install

.PHONY: build
build:
	GOPATH=$(GOPATH) go build -ldflags "-X server.channel=dev -X server.version=$(shell git describe HEAD)" -v -o bin/dividat-driver $(SRC)

crossbuild: check-version $(LINUX) $(DARWIN) $(WINDOWS) $(LATEST)

$(LINUX):
	$(call build-os,linux,$@)
	upx $@
	$(call write-metadata,$@)

$(DARWIN):
	$(call build-os,darwin,$@)
	upx $@
	$(call write-metadata,$@)

$(WINDOWS):
	$(call build-os,windows,$@)
	upx $@
	$(call sign-bin,$@)
	$(call write-metadata,$@)

$(LATEST):
	echo "{\"version\": \"$(VERSION)\", \"commit\": \"$(COMMIT)\"}" > $@


define build-os
	GOOS=$(1) GOARCH=amd64 GOPATH=$(GOPATH) go build \
	  -ldflags "-X server.channel=$(CHANNEL) -X server.version=$(VERSION)" \
		-o $(2) $(SRC)
endef

define write-metadata
  ./tools/gen-metadata.sh signingprivatekey.pem $(1) > `dirname $(1)`/metadata.json
endef

define sign-bin
	osslsigncode sign \
		-pkcs12 $(CODE_SIGNING_CERT) \
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
release: crossbuild $(LATEST)
	aws s3 cp $(RELEASE_DIR) s3://$(BUCKET)/releases/driver/$(CHANNEL)/$(VERSION)/ --recursive \
		--acl public-read \
		--cache-control max-age=0 \
		--content-type application/octet-stream
	aws s3 cp $(LATEST) s3://$(BUCKET)/releases/driver/$(CHANNEL)/latest.json \
		--acl public-read \
		--cache-control max-age=0 \
		--content-type application/json

clean:
	rm -rf release/
	go clean
