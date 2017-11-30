all: build

CWD := $(dir $(realpath $(firstword $(MAKEFILE_LIST))))
GOPATH := $(CWD)
CHECK_VERSION_BIN := bin/check-version
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

RELEASE_DIR = release/$(VERSION)

LINUX = $(RELEASE_DIR)/linux/$(BIN)-linux-amd64-$(VERSION)
DARWIN = $(RELEASE_DIR)/darwin/$(BIN)-darwin-amd64-$(VERSION)
WINDOWS = $(RELEASE_DIR)/win32/$(BIN)-win32-amd64-$(VERSION).exe
LATEST = release/latest

deps:
	cd src && glide install

.PHONY: build
build:
	GOPATH=$(GOPATH) go build -ldflags "-X server.channel=dev -X server.version=$(shell git describe --always HEAD)" -v -o bin/dividat-driver $(SRC)

crossbuild: check-version $(LINUX) $(DARWIN) $(WINDOWS) $(LATEST)

$(LINUX):
	$(call build-os,linux,$@)
	upx $@
	$(call write-signature,$@)

$(DARWIN):
	$(call build-os,darwin,$@)
	upx $@
	$(call write-signature,$@)

$(WINDOWS):
	$(call build-os,windows,$@)
	upx $@
	$(call sign-bin,$@)
	$(call write-signature,$@)

.PHONY: $(LATEST)
$(LATEST):
	echo $(VERSION) > $@ && \
	openssl dgst -sha256 -sign $(CHECKSUM_SIGNING_CERT) $@ | openssl base64 -A -out $@.sig

define build-os
	GOOS=$(1) GOARCH=amd64 GOPATH=$(GOPATH) go build \
	  -ldflags "-X server.channel=$(CHANNEL) -X server.version=$(VERSION)" \
		-o $(2) $(SRC)
endef

define write-signature
	openssl dgst -sha256 -sign $(CHECKSUM_SIGNING_CERT) $(1) | openssl base64 -A -out $(1).sig
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

check-version: $(CHECK_VERSION_BIN)
	$(CHECK_VERSION_BIN) -channel $(CHANNEL) -version $(VERSION)

$(CHECK_VERSION_BIN):
	GOPATH=$(GOPATH) go install ./src/cmd/check-version

.PHONY: release
release: crossbuild $(LATEST)
	aws s3 cp $(RELEASE_DIR) s3://$(BUCKET)/releases/driver/$(CHANNEL)/$(VERSION)/ --recursive \
		--acl public-read \
		--cache-control max-age=0
	aws s3 cp $(LATEST) s3://$(BUCKET)/releases/driver/$(CHANNEL)/latest \
		--acl public-read \
		--cache-control max-age=0
	aws s3 cp $(LATEST).sig s3://$(BUCKET)/releases/driver/$(CHANNEL)/latest.sig \
		--acl public-read \
		--cache-control max-age=0

clean:
	rm -rf release/
	rm -f $(CHECK_VERSION_BIN)
	go clean
