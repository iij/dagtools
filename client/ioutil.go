package client

import (
	"hash"
	"io"
)

// DigestReader is io.Reader and calculate digest(md5)
type DigestReader struct {
	r io.Reader
	h hash.Hash
}

// Read specified bytes
func (r DigestReader) Read(p []byte) (n int, err error) {
	n, err = r.r.Read(p)
	if err == nil && r.h != nil {
		r.h.Write(p[0:n])
	}
	return
}

// Digest returns MD5 value of read data
func (r DigestReader) Digest() []byte {
	return r.h.Sum(nil)
}
