package server

type byteCountingReader struct {
	data io.Reader
	byteCount int
}
func (r byteCountingReader) Read(p []byte) (n int, err error) {
	if len(p) == 0 { // NOP
		return 0, nil
	}
	n, err := r.data.Read(p)
	r.byteCount += n
	return n, err
}
func (r byteCountingReader) Seek(offset int64, whence int) (int64, error) {
	// NOTE:
	// > Since the SDK uses AWS v4 signature version a digest hash of the body needs to be computed. In order to do this the SDK must read the contents of the io.ReadSeekers and seek back to the origin of the reader so that the http.Client can send the body in the request.
	// See https://github.com/aws/aws-sdk-go/issues/142
	// and https://github.com/aws/aws-sdk-go/issues/915
	// A solution: use the SDK's s3manager.Uploader to upload which also gives concurrent uploads :)
	return 0, nil
}