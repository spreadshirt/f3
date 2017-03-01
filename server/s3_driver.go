package server

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	ftp "github.com/klingtnet/goftp"
	"github.com/sirupsen/logrus"
)

func notEnabled(op string) error {
	return fmt.Errorf("%q is not enabled", op)
}

// S3Driver is a filesystem FTP driver.
// Implements https://godoc.org/github.com/goftp/server#Driver
type S3Driver struct {
	featureFlags int
	noOverwrite  bool
	s3           s3iface.S3API
	metrics      cloudwatchiface.CloudWatchAPI
	hostname     string
	bucketName   string
	bucketURL    *url.URL
}

func intoAwsError(err error) awserr.Error {
	return err.(awserr.Error)
}

func logAwsError(err awserr.Error) {
	logrus.Errorf("AWS Error: Code=%q Message=%q", err.Code(), err.Message())
}

// bucketCheck checks if the bucket is accessible
func (d S3Driver) bucketCheck() error {
	_, err := d.s3.HeadBucket(&s3.HeadBucketInput{
		Bucket: aws.String(d.bucketName),
	})
	if err != nil {
		err := intoAwsError(err)
		logAwsError(err)
		logrus.Errorf("Bucket %q is not accessible.", d.bucketURL)
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

	fqdn := d.fqdn(key)
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
		logrus.WithFields(logrus.Fields{"time": time.Now(), "object": fqdn}).Errorf("Stat for %q failed.\nCode: %s", fqdn, err.Code())
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

	logrus.WithFields(logrus.Fields{"time": time.Now(), "key": fqdn, "action": "STAT"}).Infof("File information for %q", fqdn)
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
		logAwsError(err)
		logrus.Errorf("Could not list %q.", fqdn)
		return err
	}

	for _, object := range resp.Contents {
		key := *object.Key
		owner := ""
		if object.Owner != nil {
			owner = object.Owner.String()
		}
		err = cb(S3ObjectInfo{
			name:    key,
			size:    *object.Size,
			owner:   owner,
			modTime: *object.LastModified,
		})
		if err != nil {
			logrus.WithFields(logrus.Fields{"time": time.Now(), "error": err}).Errorf("Could not list %q", d.fqdn(key))
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

	fqdn := d.fqdn(key)
	_, err := d.s3.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(d.bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		err := intoAwsError(err)
		logAwsError(err)
		logrus.WithFields(logrus.Fields{"time": time.Now(), "code": err.Code(), "error": err.Message()}).Error("Failed to delete object %q.", fqdn)
		return err
	}

	logrus.WithFields(logrus.Fields{"time": time.Now(), "key": fqdn, "action": "DELETE"}).Infof("Deleted %q", fqdn)
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

	fqdn := d.fqdn(key)
	timestamp := time.Now()
	resp, err := d.s3.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(d.bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		err := intoAwsError(err)
		logAwsError(err)
		if err.Code() == "NotFound" {
			logrus.WithFields(logrus.Fields{"time": timestamp, "Object": fqdn}).Errorf("Failed to get object: %q", fqdn)
		}
		return 0, nil, err
	}
	size := *resp.ContentLength
	logrus.WithFields(logrus.Fields{"time": timestamp, "operation": "GET", "object": fqdn}).Infof("Serving object: %s", fqdn)

	_, err = d.metrics.PutMetricData(&cloudwatch.PutMetricDataInput{
		Namespace: aws.String("f3"),
		MetricData: []*cloudwatch.MetricDatum{
			&cloudwatch.MetricDataum{
				MetricName: aws.String("GET"),
				Timestamp:  &timestamp,
				Unit:       aws.String("Bytes"),
				Value:      aws.Float64(float64(size)),
				Dimensions: []*cloudwatch.Dimenson{&cloudwatch.Dimension{
					Name:  aws.String("Hostname"),
					Value: aws.String(d.hostname),
				}},
			},
		},
	})
	if err != nil {
		err = intoAwsError(err)
		logAwsError(err)
		return 0, nil, err
	}

	return size, resp.Body, nil
}

// PutFile stores the object with key `key`.
// The method returns an error with no-overwrite was set and the object already exists or appendMode was specified.
func (d S3Driver) PutFile(key string, data io.Reader, appendMode bool) (int64, error) {
	if d.featureFlags&featurePut == 0 {
		return -1, notEnabled("PUT")
	}

	fqdn := d.fqdn(key)
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

	timestamp := time.Now()
	buffer, err := ioutil.ReadAll(data)
	if err != nil {
		msg := fmt.Sprintf("Failed to put object %q because reading from source failed.", fqdn)
		logrus.WithFields(logrus.Fields{"time": timestamp, "object": fqdn, "action": "PUT", "error": err}).Errorf(msg)
		return -1, err
	}
	size := int64(len(buffer))

	_, err = d.s3.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(d.bucketName),
		Key:    aws.String(key),
		Body:   bytes.NewReader(buffer),
	})
	logrus.WithFields(logrus.Fields{"time": timestamp, "key": fqdn, "action": "PUT"}).Infof("Put %q", fqdn)

	_, err := d.metrics.PutMetricData(&cloudwatch.PutMetricDataInput{
		Namespace: aws.String("f3"),
		MetricData: []*cloudwatch.MetricDatum{
			&cloudwatch.MetricDataum{
				MetricName: aws.String("PUT"),
				Timestamp:  &timestamp,
				Unit:       aws.String("Bytes"),
				Value:      aws.Float64(float64(size)),
				Dimensions: []*cloudwatch.Dimenson{&cloudwatch.Dimension{
					Name:  aws.String("Hostname"),
					Value: aws.String(d.hostname),
				}},
			},
		},
	})
	if err != nil {
		err = intoAwsError(err)
		logAwsError(err)
		return 0, nil, err
	}

	return size, err
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
		logrus.Debugf("Failed to check object %q", d.fqdn(key))
		return false
	}
	return true
}
