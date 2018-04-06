### Release configuration #################################
# Path to folder in S3 (without slash at end)
BUCKET = s3://dist.dividat.ch/releases/driver2

# where the BUCKET folder is accessible for getting updates (needs to end with a slash)
RELEASE_URL = https://dist.dividat.com/releases/driver2/


### Basic setup ###########################################
# Set GOPATH to repository path
CWD = $(dir $(realpath $(firstword $(MAKEFILE_LIST))))
GOPATH = $(CWD)

# Set GOROOT to one matching go binary (Travis CI)
GOROOT := $(shell which go)/../../share/go

# Main source to build
BIN = dividat-driver
SRC = ./src/$(BIN)/main.go

# Get version from git
VERSION := $(shell git describe --always HEAD)

# set the channel name to the branch name
CHANNEL := $(shell git rev-parse --abbrev-ref HEAD)

CC := gcc
CXX := g++

# Force static linking on Linux
PCSCLITE_DIR := $(CWD)libpcsclite
UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Linux)
	STATIC_LINKING_LDFLAGS := -linkmode external -extldflags \"-static\"
	CC := musl-gcc
	CXX := musl-gcc

	LIB_PCSCLITE := $(PCSCLITE_DIR)
	export PKG_CONFIG_PATH=$(PCSCLITE_DIR)/lib/pkgconfig:$$PKG_CONFIG_PATH
endif

GO_LDFLAGS = -ldflags "$(STATIC_LINKING_LDFLAGS) -X dividat-driver/server.channel=$(CHANNEL) -X dividat-driver/server.version=$(VERSION) -X dividat-driver/update.releaseUrl=$(RELEASE_URL)"

CODE_SIGNING_CERT ?= ./keys/codesign.p12
CHECKSUM_SIGNING_CERT ?= ./keys/checksumsign.private.pem

### Simple build ##########################################
.PHONY: $(BIN)
$(BIN): deps
	GOROOT=$(GOROOT) GOPATH=$(GOPATH) CC=$(CC) CXX=$(CXX) go build $(GO_LDFLAGS) $(GO_CFLAGS) -o bin/$(BIN) $(SRC)


### Test suite ##########################################
.PHONY: test
test: $(BIN)
	npm install
	npm test


### Cross compilation #####################################

# helper for cross compilation
define cross-build
	GOOS=$(1) GOARCH=amd64 GOROOT=$(GOROOT) GOPATH=$(GOPATH) CC=$(CC) CXX=$(CXX) go build $(GO_LDFLAGS) -o $(2) $(SRC)
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

crossbuild: $(LINUX) # $(DARWIN) $(WINDOWS)


### Release ###############################################

RELEASE_DIR = release/$(CHANNEL)/$(VERSION)

# Helper to create signature
define write-signature
	openssl dgst -sha256 -sign $(CHECKSUM_SIGNING_CERT) $(1) | openssl base64 -A -out $(1).sig
	# make sure signature file exists and is non-zero
	test -s $(1).sig
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
	#cp bin/$(DARWIN) $(DARWIN_RELEASE)
	#upx $(DARWIN_RELEASE)
	#$ (call write-signature,$(DARWIN_RELEASE))

	# Windows
	#cp bin/$(WINDOWS).exe $(WINDOWS_RELEASE)
	#upx $(WINDOWS_RELEASE)
	#\$ (call write-signature,$(WINDOWS_RELEASE))
	#osslsigncode sign \
		#-pkcs12 $(CODE_SIGNING_CERT) \
		#-h sha1 \
		#-n "Dividat Driver" \
		#-i "https://www.dividat.com/" \
		#-in $(WINDOWS_RELEASE) \
		#-out $(WINDOWS_RELEASE)


### Deploy ################################################

SEMVER_REGEX = ^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(\-[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?(\+[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?$

deploy: release
	# Check if on right channel
	[[ $(CHANNEL) = "master" || $(CHANNEL) = "develop" ]]

	# Check if version is in semver format
	[[ $(VERSION) =~ $(SEMVER_REGEX) ]]

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
deps: $(LIB_PCSCLITE)
	cd src/$(BIN) && dep ensure

$(PCSCLITE_DIR):
	curl -LO https://github.com/LudovicRousseau/PCSC/archive/pcsc-1.8.23.tar.gz
	mkdir -p tmp/pcsclite && tar xzf pcsc-1.8.23.tar.gz -C tmp/pcsclite --strip-components=1
	cd tmp/pcsclite; \
		./bootstrap; \
		CC=musl-gcc ./configure \
			--enable-static \
			--prefix=$(PCSCLITE_DIR) \
			--with-systemdsystemunitdir=$(PCSCLITE_DIR)/lib/systemd/system \
			--disable-libudev \
			--disable-libusb \
			--disable-libsystemd; \
		make; \
		make install
	rm -rf tmp/pcsclite
	rm pcsc-1.8.23.tar.gz


clean:
	rm -rf release/
	rm -rf $(PCSCLITE_DIR)
	rm -f $(CHECK_VERSION_BIN)
	go clean
