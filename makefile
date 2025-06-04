BINARY = gportage
VERSION = v0.1.0
GOARCH = amd64

build:
	GO111MODULE=on go build -o bin/$(BINARY) cmd/gportage/main.go

install:
	go install cmd/gportage/main.go

test:
	go test -v ./...

clean:
	rm -rf bin

release: build
	tar -czf gportage-$(VERSION)-$(GOOS)-$(GOARCH).tar.gz bin/$(BINARY)

.PHONY: build install test clean release