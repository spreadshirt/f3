package server

import (
	"os"
	"time"
)

// S3ObjectInfo metadata about an s3 object.
type S3ObjectInfo struct {
	name     string
	size     int64
	owner    string
	modTime  time.Time
	isPrefix bool
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
