# generate version number
version=$(shell git describe --tags --long --always|sed 's/^v//')

all: glide.lock vendor
	go build  -ldflags "-X main.version=$(version)" dpp.go
	go fmt

compile:
	go build  -ldflags "-X main.version=$(version)" dpp.go

vendor: glide.lock
	glide install && touch vendor
glide.lock: glide.yaml
	glide update && touch glide.lock
glide.yaml:
build:
	go build  -ldflags "-X main.version=$(version)" dpp.go

version:
	@echo $(version)
