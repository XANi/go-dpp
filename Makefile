# generate version number
version=$(shell git describe --tags --long --always|sed 's/^v//')
binfile=dpp

all:
	mkdir -p bin
	CGO_ENABLED=0 go build -ldflags "-X main.version=$(version)" -o bin/$(binfile) $(binfile).go
	-@go fmt

static:
	go build -ldflags "-X main.version=$(version) -extldflags \"-static\"" -o bin/$(binfile).static $(binfile).go

arch:
	mkdir -p bin
	CGO_ENABLED=0 GOARCH=arm go build  -ldflags "-X main.version=$(version) -extldflags \"-static\"" -o bin/$(binfile).arm $(binfile).go
	CGO_ENABLED=0 GOARCH=arm64 go build  -ldflags "-X main.version=$(version) -extldflags \"-static\"" -o bin/$(binfile).aarch64 $(binfile).go
	CGO_ENABLED=0 GOARCH=amd64 go build  -ldflags "-X main.version=$(version) -extldflags \"-static\"" -o bin/$(binfile).amd64 $(binfile).go
	CGO_ENABLED=0 GOARCH=386 go build  -ldflags "-X main.version=$(version) -extldflags \"-static\"" -o bin/$(binfile).386 $(binfile).go
version:
	@echo $(version)
