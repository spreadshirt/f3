.PHONY: clean

BIN		:=ftp2s3
VERSION :=$(shell git describe --tags --always|sed 's/^v//g')
GO_FLAGS:=-ldflags="-X main.Version=$(VERSION)"
SOURCES	:=$(wildcard *.go **/*.go)
GO_PATH	:=$(shell pwd)/.go

all: $(BIN)

$(BIN): $(SOURCES)
	GOPATH=$(GO_PATH) go build $(GO_FLAGS) $@.go

clean:
	rm $(BIN)
