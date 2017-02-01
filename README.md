# ftp2s3

ftp2s3 is a bridge that acts like an FTP server which accepts files but transfers them into an s3 bucket, instead of writing them to disk.

## Installation

```sh
make install
```

If you need help, run: `ftp2s3 -h`.

## Development

Make sure that a go 1.7+ distribution is available on your system.

```sh
$ git clone github.com/spreadshirt/ftp2s3.git
$ cd ftp2s3
$ s/bootstrap
$ make [test|clean|docker]
```
