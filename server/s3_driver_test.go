package server

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	ftp "github.com/klingtnet/goftp"
)

type S3Mock struct {
	s3iface.S3API

	bucketName string
	objects    map[string]ObjectMock
}

type ObjectMock struct {
	data    []byte
	lastMod time.Time
	etag    string
}

func (mock *S3Mock) HeadObject(input *s3.HeadObjectInput) (*s3.HeadObjectOutput, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	key := aws.StringValue(input.Key)
	object, ok := mock.objects[key]
	if !ok {
		return nil, awserr.New("NoSuchObject", fmt.Sprintf("Object %q not found", key), nil)
	}
	return &s3.HeadObjectOutput{
		ContentLength: aws.Int64(int64(len(object.data))),
		LastModified:  aws.Time(object.lastMod),
	}, nil
}

func (mock *S3Mock) HeadBucket(input *s3.HeadBucketInput) (*s3.HeadBucketOutput, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	if *input.Bucket != mock.bucketName {
		return nil, awserr.New("NoSuchBucket", fmt.Sprintf("Bucket %q not found", *input.Bucket), nil)
	}
	return &s3.HeadBucketOutput{}, nil
}

func (mock *S3Mock) PutObject(input *s3.PutObjectInput) (*s3.PutObjectOutput, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	key := aws.StringValue(input.Key)
	data, err := ioutil.ReadAll(input.Body)
	if err != nil {
		return nil, awserr.New("FailedToReadBody", fmt.Sprintf("Could not read data for key: %s", key), nil)
	}
	etag := fmt.Sprintf("%s", sha256.Sum256(append([]byte(key), data...)))
	mock.objects[key] = ObjectMock{
		data,
		time.Now(),
		etag,
	}
	return &s3.PutObjectOutput{
		ETag: aws.String(etag),
	}, nil
}

func (mock *S3Mock) ListObjects(input *s3.ListObjectsInput) (*s3.ListObjectsOutput, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	contents := []*s3.Object{}
	prefix := aws.StringValue(input.Prefix)
	for key, obj := range mock.objects {
		if strings.HasPrefix(key, prefix) {
			contents = append(contents, &s3.Object{
				ETag:         aws.String(obj.etag),
				Key:          aws.String(key),
				LastModified: &obj.lastMod,
				Size:         aws.Int64(int64(len(obj.data))),
			})
		}
	}
	return &s3.ListObjectsOutput{Contents: contents}, nil
}

func (mock *S3Mock) GetObject(input *s3.GetObjectInput) (*s3.GetObjectOutput, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	key := aws.StringValue(input.Key)
	object, ok := mock.objects[key]
	if !ok {
		return nil, awserr.New("NoSuchObject", fmt.Sprintf("Object %q not found", key), nil)
	}
	return &s3.GetObjectOutput{
		Body:          ioutil.NopCloser(bytes.NewReader(object.data)),
		ContentLength: aws.Int64(int64(len(object.data))),
		ETag:          aws.String(object.etag),
		LastModified:  &object.lastMod,
	}, nil
}

func (mock *S3Mock) DeleteObject(input *s3.DeleteObjectInput) (*s3.DeleteObjectOutput, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	key := aws.StringValue(input.Key)
	if _, ok := mock.objects[key]; !ok {
		return nil, awserr.New("NoSuchObject", fmt.Sprintf("Object %q not found", key), nil)
	}

	delete(mock.objects, key)
	return &s3.DeleteObjectOutput{}, nil
}

func TestS3Driver(t *testing.T) {
	bucketName := "test-bucket"
	bucketURL := intoURL(fmt.Sprintf("https://%s.my.s3.host.com", bucketName))
	mock := S3Mock{
		bucketName: bucketName,
		objects:    map[string]ObjectMock{},
	}
	noOverwrite := true
	d := S3Driver{
		featureFlags: featureGet | featurePut | featureList | featureRemove,
		noOverwrite:  noOverwrite,
		s3:           &mock,
		bucketName:   bucketName,
		bucketURL:    bucketURL,
	}

	key := "some-key"
	content := bytes.NewBufferString("The contents of some-key.")
	contentLen := int64(content.Len())

	// Fails: put with append
	_, err := d.PutFile(key, content, true)
	if err == nil {
		t.Fatalf("Unsupported operation without error: PUT in append mode")
	}
	// valid put
	_, err = d.PutFile(key, content, false)
	if err != nil {
		t.Fatal(err)
	}
	// Fails: put on existing key without overwrite
	_, err = d.PutFile(key, content, false)
	if err == nil && noOverwrite {
		t.Fatal("Overwrite is not allowed but succeeded")
	}
	// get object
	respLen, respReader, err := d.GetFile(key, 0)
	if err != nil {
		t.Fatal(err)
	}
	if respLen != contentLen {
		t.Fatalf("Content lengths differ: expected %d but was: %d", contentLen, respLen)
	}
	respData, err := ioutil.ReadAll(respReader)
	if err != nil {
		t.Fatalf("Could not read response data: %s", err)
	}
	for idx, b := range content.Bytes() {
		if respData[idx] != b {
			t.Fatalf("Response contents differ from original object at byte %d", idx)
		}
	}
	// list objects
	err = d.ListDir("", func(info ftp.FileInfo) error {
		if info.Name() != key {
			return fmt.Errorf("Unexpected object: %s", info.Name())
		}
		if info.Size() != contentLen {
			return fmt.Errorf("Object %q has unexpected size: %d", key, info.Size())
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Object listing failed: %s", err)
	}
	// delete object
	err = d.DeleteFile(key)
	if err != nil {
		t.Fatalf("Deleting object %q failed: %s", key, err)
	}
}

func intoURL(s string) *url.URL {
	u, err := url.Parse(s)
	if err != nil {
		panic(err)
	}
	return u
}
