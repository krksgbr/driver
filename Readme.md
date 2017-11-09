# Dividat Drivers

Dividat drivers and hardware test suites (in Go!).

## Prerequisites

-   [Go](https://golang.org/)
-   [Glide](https://glide.sh/)
-   [UPX](https://upx.github.io/)

## Compatibility

Firefox, Safari and Edge not supported as they are not yet properly implementing _loopback as a trustworthy origin_, see:

- Firefox (tracking): https://bugzilla.mozilla.org/show_bug.cgi?id=1376309
- Edge: https://developer.microsoft.com/en-us/microsoft-edge/platform/issues/11963735/
- Safari: https://bugs.webkit.org/show_bug.cgi?id=171934

## Quick start

-   Clone repository, this will be the $GOPATH

-   In your editor settings, set $GOPATH to repo if needed

-   Install dependencies with glide:
        cd src && glide install && cd ..

-   Build and run
        make
        ./release/dividat-driver

    Or use `go run`:

        go run cmd/dividat-driver/main.go start

## Notes

-   Requires a special build of `diviapps`
-   Change the Senso address in `main.go`

## TODO

-   Senso discovery (<https://github.com/grandcat/zeroconf>)
-   Selfupdate (<https://github.com/inconshreveable/go-update> and possibly <https://github.com/flynn/go-tuf>)
