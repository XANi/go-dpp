# generate version number
version=$(shell git describe --tags --long --always|sed 's/^v//')

all: dep
	gom exec go build  -ldflags "-X main.version=$(version)" dpp.go
	go fmt

compile:
	gom exec go build  -ldflags "-X main.version=$(version)" dpp.go

dep:
	gom install

version:
	@echo $(version)
