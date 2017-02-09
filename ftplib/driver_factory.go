package ftplib

import (
	"errors"
	"fmt"
	ftp "github.com/goftp/server"
	"strings"
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
	featureChangeDir = 1 << iota
	featureList      = 1 << iota
	featureRemoveDir = 1 << iota
	featureRemove    = 1 << iota
	featureMove      = 1 << iota
	featureMakeDir   = 1 << iota
	featureGet       = 1 << iota
	featurePut       = 1 << iota
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
			featureFlags |= featureChangeDir
		case "ls":
			featureFlags |= featureList
		case "rmdir":
			featureFlags |= featureRemoveDir
		case "rm":
			featureFlags |= featureRemove
		case "mv":
			featureFlags |= featureMove
		case "mkdir":
			featureFlags |= featureMakeDir
		case "get":
			featureFlags |= featureGet
		case "put":
			featureFlags |= featurePut
		default:
			return 0, fmt.Errorf("Unknown feature flag: %q", feature)
		}
	}
	return featureFlags, nil
}

// NewDriver returns a new FTP driver.
func (d DriverFactory) NewDriver() (ftp.Driver, error) {
	return FsDriver{d.rootPath, d.featureFlags, d.noOverwrite}, nil
}
