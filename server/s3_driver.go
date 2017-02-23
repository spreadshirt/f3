package server

import (
	"fmt"
	"io"
	"os"
	"path"

	ftp "github.com/goftp/server"
)

// S3Driver is a filesystem FTP driver.
// Implements https://godoc.org/github.com/goftp/server#Driver
type S3Driver struct {
	rootPath     string
	featureFlags int
	noOverwrite  bool
}

func (S3Driver) Init(conn *ftp.Conn) {
	conn.Serve()
}

func (d S3Driver) buildPath(pathname string) string {
	return path.Join(d.rootPath, pathname)
}

func (d S3Driver) Stat(pathname string) (ftp.FileInfo, error) {
	pathname = d.buildPath(pathname)
	file, err := os.Open(pathname)
	if err != nil {
		return nil, err
	}
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	return FileInfo{info}, nil
}

func (d S3Driver) ChangeDir(pathname string) error {
	// There is no s3 equivalent
	return notEnabled("CD")
}

func (d S3Driver) ListDir(pathname string, cb func(ftp.FileInfo) error) error {
	if d.featureFlags&featureList == 0 {
		return notEnabled("LS")
	}
	// TODO object listing
	return nil
}

func (d S3Driver) DeleteDir(pathname string) error {
	// NOTE: Bucket removal will not be implemented
	return notEnabled("RMDIR")
}

func (d S3Driver) DeleteFile(pathname string) error {
	if d.featureFlags&featureRemove == 0 {
		return notEnabled("RM")
	}
	// TODO remove s3 objects by prefix
	return nil
}

func (d S3Driver) Rename(oldPath string, newPath string) error {
	// There is no s3 equivalent
	return notEnabled("MV")
}

func (d S3Driver) MakeDir(pathname string) error {
	// There is no s3 equivalent
	return notEnabled("MKDIR")
}

func (d S3Driver) GetFile(pathname string, offset int64) (int64, io.ReadCloser, error) {
	if d.featureFlags&featureGet == 0 {
		return -1, nil, notEnabled("GET")
	}
	// TODO s3 GET object
	return 0, nil, nil
}

func (d S3Driver) PutFile(pathname string, data io.Reader, appendMode bool) (int64, error) {
	if d.featureFlags&featurePut == 0 {
		return -1, notEnabled("PUT")
	}
	// TODO s3 PUT file
	if d.noOverwrite {
		// TODO check with HEAD if object already exists
	}
	return 0, nil
}
