.PHONY: all build test lint clean

BINARY_NAME=gw

all: build test

build:
	go build -o bin/$(BINARY_NAME) ./cmd/gw

test:
	go test -v ./...

lint:
	go vet ./...

clean:
	rm -rf bin/
