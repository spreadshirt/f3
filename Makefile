.PHONY: clean docker test

BIN		:=ftp2s3
VERSION :=$(shell git describe --tags --always|sed 's/^v//g')
GO_FLAGS:=-ldflags="-X main.Version=$(VERSION)"
SOURCES	:=$(wildcard ftplib/*.go)
GO_PATH	:=$(shell pwd)/.go

all: $(BIN)

$(GO_PATH):
	s/bootstrap

$(BIN): $(BIN).go $(SOURCES) $(GO_PATH)
	GOPATH=$(GO_PATH) go build $(GO_FLAGS) github.com/spreadshirt/$(BIN)

install: test $(BIN)
ifeq ($$EUID, 0)
	@install --mode=0755 --verbose $(BIN) /usr/local/bin
else
	@install --mode=0755 --verbose $(BIN) $$HOME/.local/bin
endif

docker: $(BIN)
	docker build -t $(BIN) .

test:
	GOPATH=$(GO_PATH) go test -v ./ftplib/...

clean:
	rm $(BIN)
