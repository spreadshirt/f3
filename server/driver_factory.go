package server

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	ftp "github.com/klingtnet/goftp"
	"github.com/sirupsen/logrus"
)

// DefaultFeatureSet is the driver default (set of) features
var DefaultFeatureSet []string

func init() {
	DefaultFeatureSet = []string{"ls"}
}

const (
	// DefaultRegion is the default bucket region
	DefaultRegion = "custom"
)

// DriverFactory builds FTP drivers.
// Implements https://godoc.org/github.com/goftp/server#DriverFactory
type DriverFactory struct {
	featureFlags int
	noOverwrite  bool
	s3           s3iface.S3API
	metrics      MetricsSender
	hostname     string
	bucketName   string
	bucketURL    *url.URL
}

// NewDriver returns a new FTP driver.
func (d DriverFactory) NewDriver() (ftp.Driver, error) {
	return S3Driver{
		featureFlags: d.featureFlags,
		noOverwrite:  d.noOverwrite,
		s3:           d.s3,
		metrics:      d.cloudwatch,
		bucketName:   d.bucketName,
		bucketURL:    d.bucketURL,
	}, nil
}

// FactoryConfig wraps config values required to setup an FTP driver and for the s3 backend.
type FactoryConfig struct {
	FtpFeatures    []string
	FtpNoOverwrite bool
	S3Credentials  string
	S3BucketURL    string
	S3Region       string
	S3UsePathStyle bool
}

// NewDriverFactory returns a DriverFactory.
func NewDriverFactory(config *FactoryConfig) (DriverFactory, error) {
	_, factory, err := setupS3(setupFtp(config, &DriverFactory{}, nil))
	return *factory, err
}

func setupFtp(config *FactoryConfig, factory *DriverFactory, err error) (*FactoryConfig, *DriverFactory, error) {
	if err != nil { // fallthrough
		return config, factory, err
	}
	factory.noOverwrite = config.FtpNoOverwrite

	logrus.Debugf("Trying to parse feature set: %q", config.FtpFeatures)
	featureFlags, err := parseFeatureSet(config.FtpFeatures)
	if err != nil {
		return config, factory, err
	}
	factory.featureFlags = featureFlags

	return config, factory, nil
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

func parseFeatureSet(features []string) (int, error) {
	featureFlags := 0
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

func setupS3(config *FactoryConfig, factory *DriverFactory, err error) (*FactoryConfig, *DriverFactory, error) {
	if err != nil { // fallthrough
		return config, factory, err
	}

	// credentials
	pair := strings.SplitN(config.S3Credentials, ":", 2)
	if len(pair) != 2 {
		return config, factory, fmt.Errorf("Malformed credentials, not in format: 'access_key:secret_key'")
	}
	accessKey, secretKey := pair[0], pair[1]
	sessionToken := ""

	bucketURL, err := url.Parse(config.S3BucketURL)
	if err != nil {
		return config, factory, err
	}
	factory.bucketURL = bucketURL

	// retrieve bucket name and endpoint from bucket FQDN
	pair = strings.SplitN(bucketURL.Host, ".", 2)
	if len(pair) != 2 {
		return config, factory, fmt.Errorf("Not a fully qualified bucket name (e.g. 'bucket.host.domain'): %q", bucketURL.String())
	}
	bucketName, endpoint := pair[0], fmt.Sprintf("%s://%s", bucketURL.Scheme, pair[1])
	factory.bucketName = bucketName

	logrus.Debugf("Trying to create an aws session with: Region: %q, PathStyle: %v, Endpoint: %q", config.S3Region, config.S3UsePathStyle, endpoint)
	// create an s3 session
	awsSession, err := session.NewSession(&aws.Config{
		Region:           aws.String(config.S3Region),
		S3ForcePathStyle: aws.Bool(config.S3UsePathStyle),
		Endpoint:         aws.String(endpoint),
		Credentials:      credentials.NewStaticCredentials(accessKey, secretKey, sessionToken),
	})
	if err != nil {
		return config, factory, err
	}
	factory.s3 = s3.New(awsSession)
	metricsSender, err := NewCloudwatchSender(awsSession)
	if err != nil {
		return config, factory, err
	}
	factory.metrics = metricsSender

	return config, factory, nil
}
