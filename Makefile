.PHONY: fmt test build all

all:
	go build -o tp .

fmt:
	go fmt ./...

test:
	go test ./...

build:
	go build ./...
