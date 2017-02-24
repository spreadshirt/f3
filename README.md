# ftp2s3

ftp2s3 is a bridge that acts like an FTP server which accepts files but transfers them into an s3 bucket, instead of writing them to disk.

## Installation

```sh
make install
```

If you need help, run: `ftp2s3 -h`.

## Example

```sh
$ ftp2s3 --features="ls,put,rm,get" --no-overwrite --ftp-addr 127.0.0.1:2121 --s3-region eu-central-1 --s3-credentials 'accesskey:secret' --s3-bucket 'https://<f3.somewhere.com>' ./ftp-credentials.txt
```

## Development

Make sure that a go 1.7+ distribution is available on your system.

```sh
$ git clone github.com/spreadshirt/ftp2s3.git
$ cd ftp2s3
$ s/make [test|clean|docker]
```

- `s/make lint` requires `golint` which can be installed by running: `go get -u github.com/golang/lint/golint`
