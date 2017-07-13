# Experimental Dividat Drivers in Go

## Prerequisites

-   [Go](https://golang.org/)
-   [Glide](https://glide.sh/)

## Quick start

-   Clone repository to your `$GOPATH`:

        cd $GOPATH
        mkdir -p github.com/dividat
        cd github.com/dividat
        git clone git@github.com:dividat/driver-go.git

-   Install dependenceis with glide:
        glide Install

-   Build and run
        go build
        ./driver-go

## Notes

-   Requires a special build of `diviapps`
-   Change the Senso address in `main.go`

## TODO

-   Senso discovery (<https://github.com/grandcat/zeroconf>)
-   Selfupdate (<https://github.com/inconshreveable/go-update> and possibly <https://github.com/flynn/go-tuf>)
