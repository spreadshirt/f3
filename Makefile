.PHONY: clean test clean-all docker

SYSTEM:=$(shell uname)
META_PACKAGE_IMPORT_PATH := $(shell vgo list -f '{{ .ImportPath }}' ./meta)
GO_SOURCES	:=$(shell vgo list -f '{{ range $$element := .GoFiles }}{{ $$.Dir }}/{{ $$element }}{{ "\n" }}{{ end }}' ./...)
VERSION		:=$(shell git describe --tags --always | sed 's/^v//')
ifeq ($(SYSTEM), Darwin)
BUILD_DATE	:=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
else
BUILD_DATE	:=$(shell date --iso-8601=seconds --utc)
endif
GO_FLAGS	:=-ldflags="-X $(META_PACKAGE_IMPORT_PATH).Version=$(VERSION) -X $(META_PACKAGE_IMPORT_PATH).BuildTime=$(BUILD_DATE)"

all: f3

f3: $(GO_SOURCES)
	@touch meta/meta.go
	@CGO_ENABLED=0 vgo build $(GO_FLAGS) ./cmd/f3

test: $(GO_SOURCES)
	@vgo test ./...

install: f3
ifeq ($$EUID, 0)
	@install -m 0755 -v f3 /usr/local/bin
else
	@vgo install ./cmd/f3
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

docker: Dockerfile f3
	docker build -t spreadshirt/f3:$(VERSION) .

docker-push: docker
	docker login docker.io
	docker push spreadshirt/f3:$(VERSION)

clean:
	rm -f f3

clean-all: clean
	rm -f f3-server_*.deb
