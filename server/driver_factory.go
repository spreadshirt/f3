package server

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
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
	featureFlags   int
	noOverwrite    bool
	awsCredentials *credentials.Credentials
	s3PathStyle    bool
	s3Region       string
	s3Endpoint     string
	hostname       string
	bucketName     string
	bucketURL      *url.URL
}

// NewDriver returns a new FTP driver.
func (d DriverFactory) NewDriver() (ftp.Driver, error) {
	logrus.Debugf("Trying to create an aws session with: Region: %q, PathStyle: %v, Endpoint: %q", d.s3Region, d.s3PathStyle, d.s3Endpoint)
	s3Session, err := session.NewSession(&aws.Config{
		Region:           aws.String(d.s3Region),
		S3ForcePathStyle: aws.Bool(d.s3PathStyle),
		Endpoint:         aws.String(d.s3Endpoint),
		Credentials:      d.awsCredentials,
	})
	if err != nil {
		return nil, err
	}
	s3Client := s3.New(s3Session)

	cloudwatchSession, err := session.NewSession(&aws.Config{
		Endpoint:    aws.String(d.s3Endpoint),
		Region:      aws.String(d.s3Region),
		Credentials: d.awsCredentials,
	})
	if err != nil {
		return nil, err
	}

	metricsSender, err := NewCloudwatchSender(cloudwatchSession)
	if err != nil {
		return nil, err
	}

	return S3Driver{
		featureFlags: d.featureFlags,
		noOverwrite:  d.noOverwrite,
		s3:           s3Client,
		uploader:     s3manager.NewUploaderWithClient(s3Client),
		metrics:      metricsSender,
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
	factory.awsCredentials = credentials.NewStaticCredentials(accessKey, secretKey, sessionToken)

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
	factory.s3Endpoint = endpoint
	factory.s3Region = config.S3Region
	factory.s3PathStyle = config.S3UsePathStyle

	return config, factory, nil
}
