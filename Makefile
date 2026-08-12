.PHONY: fmt test build all

all:
	go build -o pm .

fmt:
	go fmt ./...

test:
	go test ./...

build:
	go build ./...
