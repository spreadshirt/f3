.PHONY: clean docker test

APP		:=f3
VERSION :=$(shell git describe --tags --always|sed 's/^v//g')
GO_FLAGS:=-ldflags="-X main.Version=$(VERSION)"
SOURCES	:=$(wildcard server/*.go)
GOPATH	:=$(shell pwd)/.go
DEB_NAME:=$(APP)_$(VERSION)_amd64.deb

all: $(APP)

$(GOPATH):
	s/bootstrap

$(APP): $(APP).go $(SOURCES) $(GOPATH)
	go install $(GO_FLAGS) github.com/spreadshirt/$(APP)
	@cp $(GOPATH)/bin/$(APP) $(APP)

install: test $(APP)
ifeq ($$EUID, 0)
	@install --mode=0755 --verbose $(APP) /usr/local/bin
else
	@install --mode=0755 --verbose $(APP) $$HOME/.local/bin
endif

docker: $(APP)
	docker build -t $(APP) .

test: $(GOPATH)
	go test github.com/spreadshirt/$(APP)/server

deb: $(APP) test
	mkdir -p deb/usr/sbin
	cp $(APP) deb/usr/sbin
	fpm --force\
		--input-type dir\
		--output-type deb\
		--version $(VERSION)\
		--name $(APP)-server\
		--architecture amd64\
		--prefix /\
		--description 'An FTP to AWS s3 bridge'\
		--url 'github.com/spreadshirt/f3'\
		--deb-systemd 'deb/lib/systemd/system/$(APP)-server.service'\
		--no-deb-systemd-restart-after-upgrade\
		--chdir deb

fmt:
	@gofmt -w $(APP).go server
	@goimports -w $(APP).go server

check: vet lint

vet:
	go vet github.com/spreadshirt/$(APP)
	go vet github.com/spreadshirt/$(APP)/server

lint:
	golint github.com/spreadshirt/$(APP)
	golint github.com/spreadshirt/$(APP)/server

clean:
	rm -f $(APP)

clean-all: clean
	rm -f $(APP)_*.deb
	rm -rf .go vendor
