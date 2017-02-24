package server

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
)

type S3Mock struct {
	s3iface.S3API
}

func (mock *S3Mock) HeadObject(*s3.HeadObjectInput) (*s3.HeadObjectOutput, error) {
	return nil, awserr.New("NoCode", "Head object Failed", nil)
}

func (mock *S3Mock) HeadBucket(*s3.HeadBucketInput) (*s3.HeadBucketOutput, error) {
	return nil, awserr.New("NoCode", "Head object Failed", nil)
}

func TestS3Driver(t *testing.T) {
	d := S3Driver{
		featureFlags: featureGet | featurePut | featureList | featureRemove,
		noOverwrite:  true,
		s3:           &S3Mock{},
		bucketName:   "test-bucket",
		bucketURL:    nil,
	}

	_, err := d.Stat("a-key")
	if err != nil {
		t.Error(err)
	}
}
