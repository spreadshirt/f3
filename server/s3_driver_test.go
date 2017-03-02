package server

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	ftp "github.com/klingtnet/goftp"
)

type BucketMock struct {
	objects map[string]ObjectMock
	name    string
	lock    sync.Mutex
}

func NewBucketMock(name string) *BucketMock {
	return &BucketMock{
		objects: map[string]ObjectMock{},
		name:    name,
	}
}
func (b *BucketMock) Put(key string, object ObjectMock) {
	b.lock.Lock()
	b.objects[key] = object
	b.lock.Unlock()
}

func (b *BucketMock) Get(key string) (ObjectMock, error) {
	b.lock.Lock()
	defer b.lock.Unlock()
	if object, ok := b.objects[key]; ok {
		return object, nil
	}
	return ObjectMock{}, fmt.Errorf("No object %q found in bucket %q", key, b.name)
}

func (b *BucketMock) Delete(key string) error {
	b.lock.Lock()
	defer b.lock.Unlock()
	if _, ok := b.objects[key]; !ok {
		return awserr.New("NoSuchObject", fmt.Sprintf("Object %q not found", key), nil)
	}

	delete(b.objects, key)
	return nil
}

func (b *BucketMock) Name() string {
	return b.name
}

func (b *BucketMock) List() map[string]ObjectMock {
	b.lock.Lock()
	defer b.lock.Unlock()

	m := map[string]ObjectMock{}
	for key, object := range b.objects {
		m[key] = ObjectMock{
			data:    object.data, // should be deep copied, but hey ...
			lastMod: object.lastMod,
			etag:    object.etag,
		}
	}
	return m
}

type MetricsSenderMock struct {
	MetricsSender
}

func (m MetricsSenderMock) SendPut(size int64, timestamp time.Time) error {
	return nil
}
func (m MetricsSenderMock) SendGet(size int64, timestamp time.Time) error {
	return nil
}

type S3UploaderMock struct {
	bucket *BucketMock
}

func (s *S3UploaderMock) Upload(input *s3manager.UploadInput, options ...func(*s3manager.Uploader)) (*s3manager.UploadOutput, error) {
	bucketName := aws.StringValue(input.Bucket)
	if bucketName != s.bucket.Name() {
		return nil, fmt.Errorf("Wrong bucket, expected %q but was %q", bucketName, s.bucket.Name())
	}
	key := aws.StringValue(input.Key)
	data, err := ioutil.ReadAll(input.Body)
	if err != nil {
		return nil, awserr.New("FailedToReadBody", fmt.Sprintf("Could not read data for key: %s", key), nil)
	}
	etag := fmt.Sprintf("%s", sha256.Sum256(append([]byte(key), data...)))
	s.bucket.Put(key, ObjectMock{
		data,
		time.Now(),
		etag,
	})
	return &s3manager.UploadOutput{}, nil
}

type S3Mock struct {
	s3iface.S3API
	bucket *BucketMock
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

	object, err := mock.bucket.Get(aws.StringValue(input.Key))
	if err != nil {
		return nil, awserr.New("NoSuchObject", err.Error(), err)
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

	if aws.StringValue(input.Bucket) != mock.bucket.Name() {
		return nil, awserr.New("NoSuchBucket", fmt.Sprintf("Bucket %q not found", aws.StringValue(input.Bucket)), nil)
	}
	return &s3.HeadBucketOutput{}, nil
}

func (mock *S3Mock) ListObjects(input *s3.ListObjectsInput) (*s3.ListObjectsOutput, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	contents := []*s3.Object{}
	prefix := aws.StringValue(input.Prefix)
	for key, object := range mock.bucket.List() {
		if strings.HasPrefix(key, prefix) {
			contents = append(contents, &s3.Object{
				ETag:         aws.String(object.etag),
				Key:          aws.String(key),
				LastModified: &object.lastMod,
				Size:         aws.Int64(int64(len(object.data))),
			})
		}
	}

	return &s3.ListObjectsOutput{Contents: contents}, nil
}

func (mock *S3Mock) GetObject(input *s3.GetObjectInput) (*s3.GetObjectOutput, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	object, err := mock.bucket.Get(aws.StringValue(input.Key))
	if err != nil {
		return nil, awserr.New("NoSuchObject", err.Error(), err)
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

	err := mock.bucket.Delete(aws.StringValue(input.Key))
	return &s3.DeleteObjectOutput{}, err
}

func TestS3Driver(t *testing.T) {
	bucketName := "test-bucket"
	bucketMock := NewBucketMock(bucketName)
	bucketURL := intoURL(fmt.Sprintf("https://%s.my.s3.host.com", bucketName))
	mock := S3Mock{
		bucket: bucketMock,
	}
	noOverwrite := true
	d := S3Driver{
		featureFlags: featureGet | featurePut | featureList | featureRemove,
		noOverwrite:  noOverwrite,
		s3:           &mock,
		uploader: &S3UploaderMock{
			bucket: bucketMock,
		},
		metrics:    MetricsSenderMock{},
		bucketName: bucketName,
		bucketURL:  bucketURL,
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
