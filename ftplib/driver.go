package ftplib

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"

	ftp "github.com/goftp/server"
)

// DriverFactory builds FTP drivers.
// Implements https://godoc.org/github.com/goftp/server#DriverFactory
type DriverFactory struct {
	rootPath     string
	featureFlags int
	noOverwrite  bool
}

// NewDriverFactory returns a DriverFactory.
func NewDriverFactory(rootPath string, featureSet string, noOverwrite bool) (DriverFactory, error) {
	featureFlags, err := parseFeatureSet(featureSet)
	if err != nil {
		return DriverFactory{}, err
	}
	return DriverFactory{rootPath, featureFlags, noOverwrite}, nil
}

const (
	F_CD    = 1 << iota
	F_LS    = 1 << iota
	F_RMDIR = 1 << iota
	F_RM    = 1 << iota
	F_MV    = 1 << iota
	F_MKDIR = 1 << iota
	F_GET   = 1 << iota
	F_PUT   = 1 << iota
)

func parseFeatureSet(featureSet string) (int, error) {
	featureFlags := 0
	featureSet = strings.TrimSpace(featureSet)
	if featureSet == "" {
		return featureFlags, errors.New("Empty feature set")
	}
	features := strings.Split(featureSet, ",")
	for _, feature := range features {
		switch strings.ToLower(feature) {
		case "cd":
			featureFlags |= F_CD
		case "ls":
			featureFlags |= F_LS
		case "rmdir":
			featureFlags |= F_RMDIR
		case "rm":
			featureFlags |= F_RM
		case "mv":
			featureFlags |= F_MV
		case "mkdir":
			featureFlags |= F_MKDIR
		case "get":
			featureFlags |= F_GET
		case "put":
			featureFlags |= F_PUT
		default:
			return 0, fmt.Errorf("Unknown feature flag: %q", feature)
		}
	}
	return featureFlags, nil
}

func (d DriverFactory) NewDriver() (ftp.Driver, error) {
	return FsDriver{d.rootPath, d.featureFlags, d.noOverwrite}, nil
}

// FsDriver is a filesystem FTP driver.
// Implements https://godoc.org/github.com/goftp/server#Driver
type FsDriver struct {
	rootPath     string
	featureFlags int
	noOverwrite  bool
}

func (FsDriver) Init(conn *ftp.Conn) {
	// start as go routine and save connections into list for later management
	conn.Serve()
}

// Contains file information.
type FileInfo struct {
	os.FileInfo
}

func (f FileInfo) Owner() string {
	return "Unknown"
}
func (f FileInfo) Group() string {
	return "Unknown"
}

func (d FsDriver) buildPath(pathname string) string {
	return path.Join(d.rootPath, pathname)
}

func notEnabled(op string) error {
	return fmt.Errorf("%q is not enabled", op)
}

func (d FsDriver) Stat(pathname string) (ftp.FileInfo, error) {
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

func (d FsDriver) ChangeDir(pathname string) error {
	if d.featureFlags&F_CD == 0 {
		return notEnabled("CD")
	}
	d.rootPath = d.buildPath(pathname)
	return nil
}

func (d FsDriver) ListDir(pathname string, cb func(ftp.FileInfo) error) error {
	if d.featureFlags&F_LS == 0 {
		return notEnabled("LS")
	}
	pathname = d.buildPath(pathname)
	files, err := ioutil.ReadDir(pathname)
	if err != nil {
		return err
	}
	for _, fileInfo := range files {
		err = cb(FileInfo{fileInfo})
		if err != nil {
			return err
		}
	}
	return nil
}

func (d FsDriver) DeleteDir(pathname string) error {
	if d.featureFlags&F_RMDIR == 0 {
		return notEnabled("RMDIR")
	}
	return os.RemoveAll(d.buildPath(pathname))
}

func (d FsDriver) DeleteFile(pathname string) error {
	if d.featureFlags&F_RM == 0 {
		return notEnabled("RM")
	}
	return os.Remove(d.buildPath(pathname))
}

func (d FsDriver) Rename(oldPath string, newPath string) error {
	if d.featureFlags&F_MV == 0 {
		return notEnabled("MV")
	}
	oldPath = d.buildPath(oldPath)
	newPath = d.buildPath(newPath)
	return os.Rename(oldPath, newPath)
}

func (d FsDriver) MakeDir(pathname string) error {
	if d.featureFlags&F_MKDIR == 0 {
		return notEnabled("MKDIR")
	}
	return os.MkdirAll(d.buildPath(pathname), 0755)
}

func (d FsDriver) GetFile(pathname string, offset int64) (int64, io.ReadCloser, error) {
	if d.featureFlags&F_GET == 0 {
		return -1, nil, notEnabled("GET")
	}
	file, err := os.Open(d.buildPath(pathname))
	if err != nil {
		return -1, nil, err
	}
	_, err = file.Seek(offset, os.SEEK_SET) // SEEK_SET means seek relative to the file origin
	if err != nil {
		return -1, nil, err
	}
	info, err := file.Stat()
	if err != nil {
		return -1, nil, err
	}
	return info.Size(), file, nil
}

func (d FsDriver) PutFile(pathname string, data io.Reader, appendMode bool) (int64, error) {
	if d.featureFlags&F_PUT == 0 {
		return -1, notEnabled("PUT")
	}
	pathname = d.buildPath(pathname)
	info, err := os.Stat(pathname)
	if os.IsExist(err) {
		if info.IsDir() {
			return -1, fmt.Errorf("%q is already a directory", pathname)
		}
		if d.noOverwrite {
			return -1, fmt.Errorf("Overwrite is forbidden")
		}
	}

	mode := os.O_WRONLY | os.O_CREATE
	if appendMode {
		mode = os.O_APPEND
	}
	file, err := os.OpenFile(pathname, mode, 0644)
	if err != nil {
		return -1, err
	}
	defer file.Close()

	buf := make([]byte, 1024 * 1024)
	cnt := int64(0)
	for {
		n, err := data.Read(buf)
		if n > 0 {
			file.WriteAt(buf[:n], cnt)
			cnt += int64(n)
		}
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return -1, err
			}
		}
	}
	return cnt, nil
}
