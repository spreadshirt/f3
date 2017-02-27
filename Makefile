.PHONY: clean docker test

BIN		:=ftp2s3
VERSION :=$(shell git describe --tags --always|sed 's/^v//g')
GO_FLAGS:=-ldflags="-X main.Version=$(VERSION)"
SOURCES	:=$(wildcard server/*.go)
GO_PATH	:=$(shell pwd)/.go
DEB_NAME:=$(BIN)_$(VERSION)_amd64.deb

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

test: $(GO_PATH)
	GOPATH=$(GO_PATH) go test -v github.com/spreadshirt/$(BIN)/server

deb: $(BIN) test
	mkdir -p deb/usr/sbin
	cp $(BIN) deb/usr/sbin
	fpm --force\
		--input-type dir\
		--output-type deb\
		--version $(VERSION)\
		--name $(BIN)\
		--architecture amd64\
		--prefix /\
		--description 'An FTP to AWS s3 bridge'\
		--url 'github.com/spreadshirt/ftp2s3'\
		--chdir deb

fmt:
	@gofmt -w $(BIN).go server
	@goimports -w $(BIN).go server

check: vet lint

vet:
	@GOPATH=$(GO_PATH) go vet github.com/spreadshirt/$(BIN)
	@GOPATH=$(GO_PATH) go vet github.com/spreadshirt/$(BIN)/server

lint:
	@GOPATH=$(GO_PATH) golint github.com/spreadshirt/$(BIN)
	@GOPATH=$(GO_PATH) golint github.com/spreadshirt/$(BIN)/server

clean:
	rm -f $(BIN)

clean-all: clean
	rm -rf .go vendor
