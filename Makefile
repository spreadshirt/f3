.PHONY: clean docker test

APP		:=f3
VERSION :=$(shell git describe --tags --always|sed 's/^v//g')
GO_FLAGS:=-ldflags="-X main.Version=$(VERSION)"
SOURCES	:=$(wildcard server/*.go)
GO_PATH	:=$(shell pwd)/.go
DEB_NAME:=$(APP)_$(VERSION)_amd64.deb

all: $(APP)

$(GO_PATH):
	s/bootstrap

$(APP): $(APP).go $(SOURCES) $(GO_PATH)
	GOPATH=$(GO_PATH) go build $(GO_FLAGS) github.com/spreadshirt/$(APP)

install: test $(APP)
ifeq ($$EUID, 0)
	@install --mode=0755 --verbose $(APP) /usr/local/bin
else
	@install --mode=0755 --verbose $(APP) $$HOME/.local/bin
endif

docker: $(APP)
	docker build -t $(APP) .

test: $(GO_PATH)
	GOPATH=$(GO_PATH) go test github.com/spreadshirt/$(APP)/server

deb: $(APP) test
	mkdir -p deb/usr/sbin
	cp $(APP) deb/usr/sbin
	fpm --force\
		--input-type dir\
		--output-type deb\
		--version $(VERSION)\
		--name $(APP)\
		--architecture amd64\
		--prefix /\
		--description 'An FTP to AWS s3 bridge'\
		--url 'github.com/spreadshirt/f3'\
		--chdir deb

fmt:
	@gofmt -w $(APP).go server
	@goimports -w $(APP).go server

check: vet lint

vet:
	@GOPATH=$(GO_PATH) go vet github.com/spreadshirt/$(APP)
	@GOPATH=$(GO_PATH) go vet github.com/spreadshirt/$(APP)/server

lint:
	@GOPATH=$(GO_PATH) golint github.com/spreadshirt/$(APP)
	@GOPATH=$(GO_PATH) golint github.com/spreadshirt/$(APP)/server

clean:
	rm -f $(APP)

clean-all: clean
	rm -rf .go vendor
