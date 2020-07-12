# generate version number
version=$(shell git describe --tags --long --always|sed 's/^v//')
binfile=dpp

all:
	go build -ldflags "-X main.version=$(version)" $(binfile).go
	-@go fmt

static:
	go build -ldflags "-X main.version=$(version) -extldflags \"-static\"" -o $(binfile).static $(binfile).go

arch:
	GOARCH=arm go build  -ldflags "-X main.version=$(version) -extldflags \"-static\"" -o $(binfile).arm $(binfile).go
	GOARCH=arm64 go build  -ldflags "-X main.version=$(version) -extldflags \"-static\"" -o $(binfile).aarch64 $(binfile).go
	GOARCH=amd64 go build  -ldflags "-X main.version=$(version) -extldflags \"-static\"" -o $(binfile).amd64 $(binfile).go
	GOARCH=386 go build  -ldflags "-X main.version=$(version) -extldflags \"-static\"" -o $(binfile).386 $(binfile).go
version:
	@echo $(version)
