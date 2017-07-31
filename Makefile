all: build

deps:
	glide install

build:
	go build -v -o release/dividat-driver ./cmd/dividat-driver
	upx release/dividat-driver
