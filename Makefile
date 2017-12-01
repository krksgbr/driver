### Release configuration #################################
# Path to folder in S3 (without slash at end)
BUCKET := s3://dist-test.dividat.ch/releases/driver2

# where the BUCKET folder is accessible for getting updates
RELEASE_URL = http://dist-test.dividat.ch.s3.amazonaws.com/releases/driver2/


### Basic setup ###########################################
# Set GOPATH to repository path
CWD = $(dir $(realpath $(firstword $(MAKEFILE_LIST))))
GOPATH = $(CWD)

# Main source to build
BIN = dividat-driver
SRC = ./src/cmd/$(BIN)/main.go

# Get version from git
VERSION = $(shell git describe --always HEAD)


### Simple build ##########################################
.PHONY: $(BIN)
$(BIN):
	GOPATH=$(GOPATH) go build \
	-ldflags "-X server.channel=none -X server.version=$(VERSION) -X update.releaseUrl=$(RELEASE_URL)" \
	-o bin/$(BIN) $(SRC)


### Cross compilation #####################################

# helper for cross compilation
define cross-build
	GOOS=$(1) GOARCH=amd64 GOPATH=$(GOPATH) go build \
	  -ldflags "-X server.channel=$(CHANNEL) -X server.version=$(VERSION) -X update.releaseUrl=$(RELEASE_URL)" \
		-o $(2) $(SRC)
endef

LINUX = $(BIN)-linux-amd64
.PHONY: $(LINUX)
$(LINUX):
	$(call cross-build,linux,bin/$(LINUX))

DARWIN = $(BIN)-darwin-amd64
.PHONY: $(DARWIN)
$(DARWIN):
	$(call cross-build,darwin,bin/$(DARWIN))

WINDOWS = $(BIN)-windows-amd64
.PHONY: $(WINDOWS)
$(WINDOWS):
	$(call cross-build,windows,bin/$(WINDOWS).exe)

crossbuild: $(LINUX) $(DARWIN) $(WINDOWS)


### Release ###############################################

# set the channel name to the branch name
CHANNEL = $(shell git rev-parse --abbrev-ref HEAD)

RELEASE_DIR = release/$(CHANNEL)/$(VERSION)

# Helper to create signature
define write-signature
	openssl dgst -sha256 -sign $(CHECKSUM_SIGNING_CERT) $(1) | openssl base64 -A -out $(1).sig
endef

# write the pointer to the latest release
LATEST = release/$(CHANNEL)/latest
.PHONY: $(LATEST)
$(LATEST):
	mkdir -p `dirname $(LATEST)`
	echo $(VERSION) > $@
	$(call write-signature,$@)

LINUX_RELEASE = $(RELEASE_DIR)/$(LINUX)-$(VERSION)
DARWIN_RELEASE = $(RELEASE_DIR)/$(DARWIN)-$(VERSION)
WINDOWS_RELEASE = $(RELEASE_DIR)/$(WINDOWS)-$(VERSION).exe

# sign and copy binaries to release folders
.PHONY: release
release: crossbuild release/$(CHANNEL)/latest
	mkdir -p $(RELEASE_DIR)

	# Linux
	cp bin/$(LINUX) $(LINUX_RELEASE)
	upx $(LINUX_RELEASE)
	$(call write-signature,$(LINUX_RELEASE))

	# Darwin
	cp bin/$(DARWIN) $(DARWIN_RELEASE)
	upx $(DARWIN_RELEASE)
	$(call write-signature,$(DARWIN_RELEASE))

	# Windows
	cp bin/$(WINDOWS).exe $(WINDOWS_RELEASE)
	upx $(WINDOWS_RELEASE)
	$(call write-signature,$(WINDOWS_RELEASE))
	osslsigncode sign \
		-pkcs12 $(CODE_SIGNING_CERT) \
		-h sha1 \
		-n "Dividat Driver" \
		-i "https://www.dividat.com/" \
		-in $(WINDOWS_RELEASE) \
		-out $(WINDOWS_RELEASE)
	

### Deploy ################################################

deploy: release
	test $(CHANNEL) = "master" -o $(CHANNEL) = "develop"

	aws s3 cp $(RELEASE_DIR) $(BUCKET)/$(CHANNEL)/$(VERSION)/ --recursive \
		--acl public-read \
		--cache-control max-age=0
	aws s3 cp $(LATEST) $(BUCKET)/$(CHANNEL)/latest \
		--acl public-read \
		--cache-control max-age=0
	aws s3 cp $(LATEST).sig $(BUCKET)/$(CHANNEL)/latest.sig \
		--acl public-read \
		--cache-control max-age=0


### Dependencies and cleanup ##############################
deps:
	cd src && glide install
	npm install

clean:
	rm -rf release/
	rm -f $(CHECK_VERSION_BIN)
	go clean
