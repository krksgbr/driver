# Dividat Drivers

Dividat drivers and hardware test suites (in Go!).

## Prerequisites

-   [Go](https://golang.org/)
-   [Glide](https://glide.sh/)

For packaging:

-   [UPX](https://upx.github.io/)
-   osslsigncode (macOS: `brew install osslsigncode`)
-   AWS CLI (macOS: `brew install awscli`)

## Compatibility

Firefox, Safari and Edge not supported as they are not yet properly implementing _loopback as a trustworthy origin_, see:

- Firefox (tracking): https://bugzilla.mozilla.org/show_bug.cgi?id=1376309
- Edge: https://developer.microsoft.com/en-us/microsoft-edge/platform/issues/11963735/
- Safari: https://bugs.webkit.org/show_bug.cgi?id=171934

## Quick start

-   Clone repository, this will be the $GOPATH

-   In your editor settings, set $GOPATH to repo if needed

-   Install dependencies:
        make deps

-   Build and run
        make
        ./release/dividat-driver

    Or use `go run`:

        go run src/cmd/dividat-driver/main.go start

## Notes
## Tests

Run the test suite with: `npm test`.

## Tools

-   Senso replay:
        npm install
        npm run replay -- tools/rec/simple.dat

## TODO

-   Senso discovery (<https://github.com/grandcat/zeroconf>)
