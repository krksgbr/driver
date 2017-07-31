all: build

deps:
	glide install

build:
	go build -v ./cmd/dividat-driver
