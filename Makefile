all: build

CWD=$(dir $(realpath $(firstword $(MAKEFILE_LIST))))

deps:
	glide install

build:
	GOPATH=$(CWD) go build -v -o release/dividat-driver src/cmd/dividat-driver/main.go
	upx release/dividat-driver
