# Dividat Driver

[![Build Status](https://travis-ci.org/dividat/driver.svg?branch=develop)](https://travis-ci.org/dividat/driver)

Dividat drivers and hardware test suites.

## Prerequisites

For building and testing all dependencies can be installed via [Nix](https://nixos.org/nix).

Alternatively you need:

-   [Go](https://golang.org/)
-   [Dep](https://github.com/golang/dep)
-   [NodeJS](https://nodejs.org/)

For packaging:

-   [UPX](https://upx.github.io/)
-   osslsigncode (macOS: `brew install osslsigncode`)
-   A recent version of OpenSSL (tested with 1.0.2.m) (macOS: `nix-shell -p openssl`)
-   AWS CLI (macOS: `brew install awscli`)

## Compatibility

Firefox, Safari and Edge not supported as they are not yet properly implementing _loopback as a trustworthy origin_, see:

-   Firefox (tracking): <https://bugzilla.mozilla.org/show_bug.cgi?id=1376309>
-   Edge: <https://developer.microsoft.com/en-us/microsoft-edge/platform/issues/11963735/>
-   Safari: <https://bugs.webkit.org/show_bug.cgi?id=171934>

## Quick start

-   Clone repository, this will be the $GOPATH

-   In your editor settings, set $GOPATH to repo if needed

-   Install dependencies:

        make deps

-   Build and run

        make
        ./bin/dividat-driver

    Or use `go run`:

        go run src/cmd/dividat-driver/main.go start

## Tests

Run the test suite with: `make test`.

## Releasing

Currently releases can only be made for Linux (from Linux). Builds on Linux are statically linked with [musl](https://www.musl-libc.org/).

To create a release run: `make release`. You will need to be able to provide appropriate signing keys.

To deploy a new release run: `make deploy`. This can only be done if you are on `master` or `develop` branch, have correctly tagged the revision and have AWS credentials set in your environment.

## Tools

### Data replayer

Logged data can be replayed for debugging purposes.

For default settings: `npm run replay`

To replay an other recording: `npm run replay -- rec/simple.dat`

To slow down the replay: `npm run replay -- -t 100`
