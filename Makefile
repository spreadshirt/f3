.PHONY: clean test fmt vet lint setup check-dep check-lint deb

SHELL		:=bash
GOPATH		:=$(PWD)/.go
NAMESPACE	:=github.com/spreadshirt/f3
WORKSPACE	:=$(GOPATH)/src/$(NAMESPACE)
GO_SOURCES	:=$(wildcard cmd/f3/*.go server/*.go)
GO_PACKAGES	:=$(dir $(GO_SOURCES))
VERSION		:=$(shell git describe --tags --always)
GO_FLAGS	:=-ldflags="-X $(NAMESPACE)/meta.Version=$(VERSION) -X $(NAMESPACE)/meta.BuildTime=$(shell date --iso-8601=seconds --utc)"

all: setup f3

f3: test $(GO_SOURCES)
	@cd $(WORKSPACE)\
		&& go install $(GO_FLAGS) $(NAMESPACE)/cmd/f3
	@cp $(GOPATH)/bin/$@ $(PWD)

test: setup
	@cd $(WORKSPACE)\
		&& go test $(addprefix $(NAMESPACE)/,$(GO_PACKAGES))

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
		--deb-systemd 'deb/lib/systemd/system/f3-server.service'\
		--no-deb-systemd-restart-after-upgrade\
		--chdir deb

fmt: $(GO_SOURCES)
	gofmt -w $<
	goimports -w $<

check: vet lint

vet: $(GO_SOURCES)
	go vet $(addprefix $(NAMESPACE)/,$(GO_PACKAGES))

lint: check-lint $(GO_SOURCES)
	golint $(addprefix $(NAMESPACE)/,$(GO_PACKAGES))

dep: $(WORKSPACE)
	@cd $(WORKSPACE) && dep $(ARGS)

setup: check-dep $(WORKSPACE)
	@cd $(WORKSPACE) && dep ensure

$(GOPATH):
	@mkdir -p $@

$(WORKSPACE): $(GOPATH)
	@mkdir -p $$(dirname $@)
	@ln -s $(PWD) $@

check-dep:
	@hash dep 2>/dev/null\
		|| (echo -e "dep is missing:\ngo get -u github.com/golang/dep/cmd/dep"; false)

check-lint:
	@hash golint 2>/dev/null\
		|| (echo -e "golint is missing:\ngo get -u github.com/golang/lint/golint"; false)

clean:
	rm -f f3

clean-all: clean
	rm -f f3-server_*.deb
	rm -rf .go vendor