package client

import (
	"testing"
)

func TestGetMimeType(t *testing.T) {
	mimeType := GetMimeType("dummy.txt")
	if mimeType != "text/plain" {
		t.Errorf("text/plain != %v", mimeType)
	}
	mimeType = GetMimeType("dummy.html")
	if mimeType != "text/html" {
		t.Errorf("text/html != %v", mimeType)
	}
	mimeType = GetMimeType("dummy.png")
	if mimeType != "image/png" {
		t.Errorf("image/png != %v", mimeType)
	}
	mimeType = GetMimeType("dummy.unknown")
	if mimeType != "application/octet-stream" {
		t.Errorf("application/octet-stream != %v", mimeType)
	}
}
