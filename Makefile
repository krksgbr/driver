all: build

CWD = $(dir $(realpath $(firstword $(MAKEFILE_LIST))))
GOPATH = $(CWD)
SRC = src/cmd/dividat-driver/main.go
BIN = dividat-driver

BRANCH = $(shell git rev-parse --abbrev-ref HEAD)
ifeq ($(BRANCH), master)
	CHANNEL = internal
else
	CHANNEL = dev
endif

COMMIT := $(shell git rev-parse HEAD)

TAG := $(shell git describe --exact-match HEAD 2>/dev/null)

RELEASE_DIR = release/$(VERSION)

LINUX = $(RELEASE_DIR)/linux/$(BIN)-linux-amd64-$(VERSION)
DARWIN = $(RELEASE_DIR)/darwin/$(BIN)-darwin-amd64-$(VERSION)
WINDOWS = $(RELEASE_DIR)/win32/$(BIN)-win32-amd64-$(VERSION).exe

ifndef VERSION
$(error VERSION required)
endif

ifneq ($(VERSION), $(TAG))
$(error VERSION $(VERSION) and git tag of HEAD $(TAG) must match)
endif


.PHONY: build release

deps:
	glide install

build:
	GOPATH=$(GOPATH) go build -v -o bin/dividat-driver $(SRC)

crossbuild: $(LINUX) $(DARWIN) $(WINDOWS)

$(LINUX):
	GOOS=linux GOARCH=amd64 GOPATH=$(GOPATH) go build -o $@ $(SRC)
	upx $@
	cd $(dir $@) && shasum $(notdir $@) > $(notdir $@).sha1

$(DARWIN):
	GOOS=darwin GOARCH=amd64 GOPATH=$(GOPATH) go build -o $@ $(SRC)
	upx $@
	cd $(dir $@) && shasum $(notdir $@) > $(notdir $@).sha1

$(WINDOWS):
	GOOS=windows GOARCH=amd64 GOPATH=$(GOPATH) go build -o $@ $(SRC)
	upx $@
	osslsigncode sign \
		-pkcs12 $(CODE_SIGNING_CERT) \
		-pass $(CODE_SIGNING_PW) \
		-h sha1 \
		-n "Dividat Driver" \
		-i "https://www.dividat.com/" \
		-in $@ \
		-out $@
	cd $(dir $@) && shasum $(notdir $@) > $(notdir $@).sha1


release: crossbuild
	aws s3 cp $(RELEASE_DIR) s3://dist.dividat.ch/driver/$(CHANNEL)/$(VERSION)/ --recursive --dryrun \
		--acl public-read \
		--cache-control max-age=0 \
		--content-type application/octet-stream

clean:
	rm -rf release/
	go clean
