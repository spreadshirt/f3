.PHONY: clean test clean-all

SHELL		:=bash
NAMESPACE	:=github.com/spreadshirt/f3
GO_SOURCES	:=$(wildcard cmd/f3/*.go server/*.go)
VERSION		:=$(shell git describe --tags --always | sed 's/^v//')
GO_FLAGS	:=-ldflags="-X $(NAMESPACE)/meta.Version=$(VERSION) -X $(NAMESPACE)/meta.BuildTime=$(shell date --iso-8601=seconds --utc)"

all: f3

f3: $(GO_SOURCES)
	@touch meta/meta.go
	@vgo build $(GO_FLAGS) ./cmd/f3

test: $(GO_SOURCES)
	@vgo test ./...

install: f3
ifeq ($$EUID, 0)
	@install --mode=0755 --verbose f3 /usr/local/bin
else
	@install --mode=0755 --verbose f3 $$HOME/.local/bin
endif

deb: f3 test
	mkdir -p deb/usr/sbin
	cp f3 deb/usr/sbin
	fpm --force\
		--input-type dir\
		--output-type deb\
		--version $(VERSION)\
		--name f3-server\
		--architecture amd64\
		--prefix /\
		--description 'An FTP to AWS s3 bridge'\
		--url "$(NAMESPACE)"\
		--no-deb-systemd-restart-after-upgrade\
		--chdir deb

clean:
	rm -f f3

clean-all: clean
	rm -f f3-server_*.deb
