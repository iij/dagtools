package client

import (
	"mime"
	"net/http"
	"net/http/httputil"
	"regexp"
	"strings"
)

// DumpRequest returns the as-received wire representation of req,
// optionally including the request body, for debugging.
func DumpRequest(req *http.Request, body bool) string {
	if req == nil {
		return ""
	}
	dump, _ := httputil.DumpRequest(req, body)
	return string(dump)
}

// DumpResponse is like DumpRequest but dumps a response.
func DumpResponse(resp *http.Response, body bool) string {
	if resp == nil {
		return ""
	}
	dump, _ := httputil.DumpResponse(resp, body)
	return string(dump)
}

// GetMimeType returns MimeType from the file extension.
func GetMimeType(filename string) string {
	r, _ := regexp.Compile(`(\.[a-z]+)$`)
	s := r.FindString(filename)
	mimeType := strings.Split(mime.TypeByExtension(s), ";")[0]
	if mimeType == "" {
		return "application/octet-stream"
	}
	return mimeType
}
