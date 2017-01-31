package ftplib

import (
	"errors"
	"fmt"
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
