package server

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	ftp "github.com/klingtnet/goftp"
	"github.com/sirupsen/logrus"
)

// S3ObjectInfo metadata about an s3 object.
type S3ObjectInfo struct {
	name     string
	size     int64
	owner    string
	modTime  time.Time
	isPrefix bool
}

func notEnabled(op string) error {
	return fmt.Errorf("%q is not enabled", op)
}

// Name returns an s3 objects fully qualified identifier, e.g. `https://my-bucket.s3.amazonaws.com/some/prefix/objectKey`.
func (s S3ObjectInfo) Name() string {
	return s.name
}

// Size returns the objects size in bytes.
func (s S3ObjectInfo) Size() int64 {
	return s.size
}

// Mode returns `o644` for all objects because there is no file mode equivalent for s3 objects.
func (s S3ObjectInfo) Mode() os.FileMode {
	return os.FileMode(0644)
}

// IsDir is solely used for compatibility with FTP, don't rely on its return value.
func (s S3ObjectInfo) IsDir() bool {
	return s.isPrefix
}

// ModTime returns the object's date of last modification.
func (s S3ObjectInfo) ModTime() time.Time {
	return s.modTime
}

// Sys always returns `nil`.
func (s S3ObjectInfo) Sys() interface{} {
	return nil
}

// Owner returns the objects owner name if known, otherwise "Unknown" is returned.
func (s S3ObjectInfo) Owner() string {
	if s.owner == "" {
		return "Unknown"
	}
	return s.owner
}

// Group returns always "Unknown" because there is no corresponding attribute for an s3 object.
func (s S3ObjectInfo) Group() string {
	return "Unknown"
}

// S3Driver is a filesystem FTP driver.
// Implements https://godoc.org/github.com/goftp/server#Driver
type S3Driver struct {
	featureFlags int
	noOverwrite  bool
	s3           s3iface.S3API
	bucketName   string
	bucketURL    *url.URL
}

func intoAwsError(err error) awserr.Error {
	return err.(awserr.Error)
}

// bucketCheck checks if the bucket is accessible
func (d S3Driver) bucketCheck() error {
	_, err := d.s3.HeadBucket(&s3.HeadBucketInput{
		Bucket: aws.String(d.bucketName),
	})
	if err != nil {
		err := intoAwsError(err)
		logrus.Errorf("Bucket %q is not accessible.\nCode: %s", d.bucketURL, err.Code())
		return err
	}
	return nil
}

// Init initializes the FTP connection.
func (d S3Driver) Init(conn *ftp.Conn) {
	conn.Serve()
}

// Stat returns information about the object with key `key`.
func (d S3Driver) Stat(key string) (ftp.FileInfo, error) {
	if err := d.bucketCheck(); err != nil {
		return S3ObjectInfo{}, err
	}

	resp, err := d.s3.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(d.bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		err := intoAwsError(err)
		if err.Code() == "NotFound" {
			// If a client calls `ls` for a prefix (path) then `stat` is called for this prefix which will fail
			// in cases where the prefix is not an object key.
			// Returning an error would cause `ls` to fail, thus an ObjectInfo is returned which simulates a `stat` on a directory.
			return S3ObjectInfo{
				name:     key,
				isPrefix: true,
				size:     0,
				modTime:  time.Now(),
			}, nil
		}
		fqdn := d.fqdn(key)
		logrus.WithFields(logrus.Fields{"object": fqdn}).Errorf("Stat for %q failed.\nCode: %s", fqdn, err.Code())
		return S3ObjectInfo{}, err
	}

	size := int64(0)
	if resp.ContentLength != nil {
		size = *resp.ContentLength
	}
	modTime := time.Now()
	if resp.LastModified != nil {
		modTime = *resp.LastModified
	}
	return S3ObjectInfo{
		name:     key,
		isPrefix: true,
		size:     size,
		modTime:  modTime,
	}, nil
}

// ChangeDir will always return an error because there is no such operation for a cloud object storage.
func (d S3Driver) ChangeDir(key string) error {
	// There is no s3 equivalent
	logrus.Warn("ChangeDir (CD) is not supported.")
	return notEnabled("CD")
}

// ListDir call the callback function with object metadata for each object located under prefix `key`.
func (d S3Driver) ListDir(key string, cb func(ftp.FileInfo) error) error {
	if d.featureFlags&featureList == 0 {
		return notEnabled("LS")
	}

	if err := d.bucketCheck(); err != nil {
		return err
	}

	// TODO: Prefix and delimiter
	resp, err := d.s3.ListObjects(&s3.ListObjectsInput{
		Bucket: aws.String(d.bucketName),
	})
	if err != nil {
		err := intoAwsError(err)
		fqdn := d.fqdn(key)
		logrus.Errorf("Could not list %q.\nCode: %s\n", fqdn, err.Code())
		return err
	}

	for _, object := range resp.Contents {
		key := *object.Key
		err = cb(S3ObjectInfo{
			name:    key,
			size:    *object.Size,
			owner:   object.Owner.String(),
			modTime: *object.LastModified,
		})
		if err != nil {
			logrus.WithFields(logrus.Fields{"Error": err}).Errorf("Could not list %q", d.fqdn(key))
		}
	}
	return nil
}

// DeleteDir will always return an error because there is no such operation for a cloud object storage.
func (d S3Driver) DeleteDir(key string) error {
	// NOTE: Bucket removal will not be implemented
	logrus.Warn("RemoveDir (RMDIR) is not supported.")
	return notEnabled("RMDIR")
}

// DeleteFile will delete the object with key `key`.
func (d S3Driver) DeleteFile(key string) error {
	if d.featureFlags&featureRemove == 0 {
		logrus.Warn("Remove (RM) is not enabled.")
		return notEnabled("RM")
	}

	_, err := d.s3.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(d.bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		err := intoAwsError(err)
		logrus.WithFields(logrus.Fields{"Code": err.Code(), "Error": err.Message()}).Error("Failed to delete object %q.", d.fqdn(key))
		return err
	}
	return nil
}

// Rename will always return an error because there is no such operation for a cloud object storage.
func (d S3Driver) Rename(oldKey string, newKey string) error {
	// TODO: there is no direct method for s3, must be copied and removed
	logrus.Warn("Rename (MV) is not supported.")
	return notEnabled("MV")
}

// MakeDir will always return an error because there is no such operation for a cloud object storage.
func (d S3Driver) MakeDir(key string) error {
	// There is no s3 equivalent
	logrus.Warn("MakeDir (MkDir) is not supported.")
	return notEnabled("MKDIR")
}

// GetFile returns the object with key `key`.
func (d S3Driver) GetFile(key string, offset int64) (int64, io.ReadCloser, error) {
	if d.featureFlags&featureGet == 0 {
		return -1, nil, notEnabled("GET")
	}

	if d.noOverwrite && d.objectExists(key) {
		return -1, nil, fmt.Errorf("object alread exists and overwrite is not allowed")
	}

	resp, err := d.s3.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(d.bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		err := intoAwsError(err)
		if err.Code() == "NotFound" {
			fqdn := d.fqdn(key)
			logrus.WithFields(logrus.Fields{"Object": fqdn}).Errorf("Failed to get object: %q", fqdn)
		}
		return 0, nil, err
	}

	fqdn := d.fqdn(key)
	logrus.WithFields(logrus.Fields{"Operation": "GET", "Object": fqdn}).Info("Serving object", fqdn)
	return *resp.ContentLength, resp.Body, nil
}

// PutFile stores the object with key `key`.
// The method returns an error with no-overwrite was set and the object already exists or appendMode was specified.
func (d S3Driver) PutFile(key string, data io.Reader, appendMode bool) (int64, error) {
	if d.featureFlags&featurePut == 0 {
		return -1, notEnabled("PUT")
	}

	if appendMode {
		msg := fmt.Sprintf("can not append to object %q because the backend does not support appending", d.fqdn(key))
		logrus.Error(msg)
		return -1, fmt.Errorf(msg)
	}

	if d.noOverwrite && d.objectExists(key) {
		msg := fmt.Sprintf("object %q already exists and overwriting is forbidden", d.fqdn(key))
		logrus.Error(msg)
		return -1, fmt.Errorf(msg)
	}

	buffer, err := ioutil.ReadAll(data)
	if err != nil {
		fqdn := d.fqdn(key)
		msg := fmt.Sprintf("Failed to put object %q because reading from source failed.", fqdn)
		logrus.WithFields(logrus.Fields{"Object": fqdn, "Operation": "PUT", "Error": err}).Errorf(msg)
		return -1, err
	}

	d.s3.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(d.bucketName),
		Key:    aws.String(key),
		Body:   bytes.NewReader(buffer),
	})

	return 0, nil
}

// fqdn returns the fully qualified name for a object with key `key`.
func (d S3Driver) fqdn(key string) string {
	u := d.bucketURL
	u.Path = key
	return u.String()
}

// objectExists returns true if the object exists.
func (d S3Driver) objectExists(key string) bool {
	logrus.Debugf("Trying to check if object %q exists.", d.fqdn(key))
	_, err := d.s3.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(d.bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		err := intoAwsError(err)
		if err.Code() == "NotFound" {
			return false
		}
		logrus.Error("Failed to check object %q", d.fqdn(key))
		return false
	}
	return true
}
