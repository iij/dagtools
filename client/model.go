package client

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// ErrorResponse represents error information of dagrin REST API
type ErrorResponse struct {
	XMLName   xml.Name `xml:"Error"`
	Code      string   `xml:"Code"`
	Message   string   `xml:"Message"`
	Resource  string   `xml:"Resource"`
	RequestID string   `xml:"RequestId"`
	ErrorCode int
}

func (e ErrorResponse) Error() string {
	msg := fmt.Sprintf("%d %s %s", e.ErrorCode, e.Code, e.Message)
	if e.RequestID != "" {
		msg += fmt.Sprintf(" (%s)", e.RequestID)
	}
	return msg
}

// Bucket is meta information of `Bucket` in dagrin
type Bucket struct {
	Name         string    `xml:"Name"`
	CreationDate time.Time `xml:"CreationDate"`
	Region		string		`xml:"LocationConstraint"`
}

func (b Bucket) String() string {
	return fmt.Sprintf("%s (%s)", b.Name, b.CreationDate)
}

// Object is meta information of `Object` in dagrin
type Object struct {
	Bucket       string
	Key          string
	ETag         string
	LastModified time.Time
	Size         int64
	Metadata     *ObjectMetadata
}

// ObjectSummary is summary of an Object's meta information in dagrin
type ObjectSummary struct {
	Key          string    `xml:"Key"`
	LastModified time.Time `xml:"LastModified"`
	ETag         string    `xml:"ETag"`
	Size         int64     `xml:"Size"`
	StorageClass string    `xml:"StorageClass"`
	Owner        Owner     `xml:"Owner"`
}

func (s ObjectSummary) String() string {
	return fmt.Sprintf("%s (%d Bytes, %s)", s.Key, s.Size, s.LastModified)
}

// BucketListing returns list of Bucket
type BucketListing struct {
	Owner   Owner    `xml:"Owner"`
	Buckets []Bucket `xml:"Buckets>Bucket"`
}

// ObjectListing returns list of ObjectSummary
type ObjectListing struct {
	Name           string          `xml:"Name"`
	Prefix         string          `xml:"Prefix"`
	Marker         string          `xml:"Marker"`
	MaxKeys        int             `xml:"MaxKeys"`
	Delimiter      string          `xml:"Delimiter"`
	NextMarker     string          `xml:"NextMarker"`
	IsTruncated    bool            `xml:"IsTruncated"`
	Summaries      []ObjectSummary `xml:"Contents"`
	CommonPrefixes []CommonPrefix  `xml:"CommonPrefixes"`
}

// IsEmpty returns true if no CommonPrefixes and no Summaries
func (listing *ObjectListing) IsEmpty() bool {
	return len(listing.Summaries) == 0 && len(listing.CommonPrefixes) == 0
}

func (listing *ObjectListing) String() string {
	return fmt.Sprintf("Bucket=%v, Prefix=%v, Marker=%v, MaxKeys=%v, Delimiter=%v, NextMarker=%v, IsTrucated=%v, Summaries: %v, CommonPrefixes: %v",
		listing.Name, listing.Prefix, listing.Marker, listing.MaxKeys, listing.Delimiter, listing.NextMarker, listing.IsTruncated, listing.Summaries, listing.CommonPrefixes)
}

// MultipartUploadListing returns list of MultipartUpload
type MultipartUploadListing struct {
	Bucket             string                   `xml:"Bucket"`
	Prefix             string                   `xml:"Prefix"`
	KeyMarker          string                   `xml:"KeyMarker"`
	UploadIdMarker     string                   `xml:"UploadIdMarker"`
	MaxUploads         int                      `xml:"MaxUploads"`
	Delimiter          string                   `xml:"Delimiter"`
	NextKeyMarker      string                   `xml:"NextKeyMarker"`
	NextUploadIdMarker string                   `xml:"NextUploadIdMarker"`
	IsTruncated        bool                     `xml:"IsTruncated"`
	Uploads            []MultipartUploadSummary `xml:"Upload"`
	CommonPrefixes     []string                 `xml:"CommonPrefixes>Prefix"`
}

// IsEmpty returns true if listing has no CommonPrefixes and no Uploads
func (listing *MultipartUploadListing) IsEmpty() bool {
	return len(listing.Uploads) == 0 && len(listing.CommonPrefixes) == 0
}

func (listing *MultipartUploadListing) String() string {
	return fmt.Sprintf("Bucket=%v, Prefix=%v, KeyMarker=%v, UploadIdMarker=%v, Delimiter=%v, MaxUploads=%v, NextKeyMarker=%v, NextUploadIdMarker=%v, CommonPrefixes=%v, Uploads=%v",
		listing.Bucket, listing.Prefix, listing.KeyMarker, listing.UploadIdMarker, listing.Delimiter, listing.MaxUploads, listing.NextKeyMarker, listing.NextUploadIdMarker, listing.CommonPrefixes, listing.Uploads)
}

// PartListing returns list of PartDescription
type PartListing struct {
	Bucket               string            `xml:"Bucket"`
	Key                  string            `xml:"Key"`
	UploadId             string            `xml:"UploadId"`
	PartNumberMarker     int               `xml:"PartNumberMarker"`
	MaxParts             int               `xml:"MaxParts"`
	NextPartNumberMarker int               `xml:"NextPartNumberMarker"`
	IsTruncated          bool              `xml:"IsTruncated"`
	Initiator            Owner             `xml:"Initiator"`
	Owner                Owner             `xml:"Owner"`
	StorageClass         string            `xml:"StorageClass"`
	Parts                []PartDescription `xml:"Part"`
}

// IsEmpty return true if listing has no Parts
func (listing *PartListing) IsEmpty() bool {
	return len(listing.Parts) == 0
}

// GetPart returns a part of specified number
func (listing PartListing) GetPart(partNumber int) *PartDescription {
	for _, p := range listing.Parts {
		if p.PartNumber == partNumber {
			return &p
		}
	}
	return nil
}

func (listing *PartListing) String() string {
	return fmt.Sprintf("Bucket=%v, Key=%v, UploadId=%v, PartNumberMarker=%v, MaxParts=%v, NextPartNumberMarker=%v, IsTruncated=%v, Initiator=%v, Owner=%v, StorageClass=%v, Parts=%v",
		listing.Bucket, listing.Key, listing.UploadId, listing.PartNumberMarker, listing.MaxParts, listing.NextPartNumberMarker, listing.IsTruncated, listing.Initiator, listing.Owner, listing.StorageClass, listing.Parts)
}

// CommonPrefix represents name of directory
type CommonPrefix struct {
	Prefix string `xml:"Prefix"`
}

func (prefix CommonPrefix) String() string {
	return prefix.Prefix
}

// Owner is user information of Bucket/Object owner
type Owner struct {
	ID          string `xml:"ID"`
	DisplayName string `xml:"DisplayName"`
}

func (o Owner) String() string {
	return o.DisplayName
}

// MultipartUploadSummary is multipart upload's summary
type MultipartUploadSummary struct {
	Key          string    `xml:"Key"`
	UploadId     string    `xml:"UploadId"`
	Initiator    Owner     `xml:"Initiator"`
	Owner        Owner     `xml:"Owner"`
	StorageClass string    `xml:"StorageClass"`
	Initiated    time.Time `xml:"Initiated"`
}

func (summary *MultipartUploadSummary) String() string {
	return fmt.Sprintf("Key=%v, UploadId=%v, Initiator=%v, Owner=%v, StorageClass=%v, Initiated=%v",
		summary.Key, summary.UploadId, summary.Initiator.DisplayName, summary.Owner.DisplayName, summary.StorageClass, summary.Initiated)
}

// MultipartUpload is multipart upload's identifier information
type MultipartUpload struct {
	Bucket   string `xml:"Bucket"`
	Key      string `xml:"Key"`
	UploadID string `xml:"UploadId"`
}

func (m *MultipartUpload) String() string {
	return fmt.Sprintf("{bucket: %q, key: %q, uploadId: %q}", m.Bucket, m.Key, m.UploadID)
}

type completeMultipartUploadRequest struct {
	XMLName xml.Name `xml:"CompleteMultipartUpload"`
	Parts   []*Part  `xml:"Part"`
}

// Part is meta information of a part of multipart upload.
type Part struct {
	PartNumber int    `xml:"PartNumber"`
	ETag       string `xml:"ETag"`
}

// PartDescription is more information of Part.
type PartDescription struct {
	Part
	LastModified time.Time `xml:"LastModified"`
	Size         int64     `xml:"Size"`
}

func (p Part) String() string {
	return fmt.Sprintf("{partNumber: %d, eTag: %q}", p.PartNumber, p.ETag)
}

// CompleteMultipartUploadResult is response of complete multipart upload request.
type CompleteMultipartUploadResult struct {
	Location string `xml:"Location"`
	Bucket   string `xml:"Bucket"`
	Key      string `xml:"Key"`
	ETag     string `xml:"ETag"`
}

type multipleDeletionKey struct {
	Key string `xml:"Key"`
}

type multipleDeletionRequest struct {
	XMLName xml.Name              `xml:"Delete"`
	Quiet   bool                  `xml:"Quiet"`
	Keys    []multipleDeletionKey `xml:"Object"`
}

// DeletedObject is an object that has been deleted.
type DeletedObject struct {
	Key string `xml:"Key"`
}

// MultipleDeletionError is a part of errors in a Delete Multiple Objects request.
type MultipleDeletionError struct {
	Key     string `xml:"Key"`
	Code    string `xml:"Code"`
	Message string `xml:"Message"`
}

func (e *MultipleDeletionError) String() string {
	return fmt.Sprintf("Failed to delete %q: %v %v", e.Key, e.Code, e.Message)
}

func (e *MultipleDeletionError) Error() string {
	return fmt.Sprintf("%q: %v %v", e.Key, e.Code, e.Message)
}

// MultipleDeletionResult is result of Delete Multiple Objects.
type MultipleDeletionResult struct {
	DeletedObjects []DeletedObject         `xml:"Deleted"`
	Errors         []MultipleDeletionError `xml:"Error"`
}

// HasErrors returns contains error in result.
func (res *MultipleDeletionResult) HasErrors() bool {
	return len(res.Errors) > 0
}

// ObjectMetadata is object's user metadata and HTTP metadata.
type ObjectMetadata struct {
	ContentLength           int64
	ContentType             string
	ContentMD5              string
	ContentDisposition      string
	ContentEncoding         string
	CacheControl            string
	WebsiteRedirectLocation string
	UserMetadata            *UserMetadata
}

func (m *ObjectMetadata) String() string {
	var fields []string
	if m.ContentType != "" {
		fields = append(fields, fmt.Sprintf("Content-Type: %s", m.ContentType))
	}
	if m.ContentLength > 0 {
		fields = append(fields, fmt.Sprintf("Content-Length: %d", m.ContentLength))
	}
	if m.ContentMD5 != "" {
		fields = append(fields, fmt.Sprintf("Content-MD5: %s", m.ContentMD5))
	}
	if m.ContentDisposition != "" {
		fields = append(fields, fmt.Sprintf("Content-Disposition: %s", m.ContentDisposition))
	}
	if m.ContentEncoding != "" {
		fields = append(fields, fmt.Sprintf("Content-Encoding: %s", m.ContentEncoding))
	}
	if m.CacheControl != "" {
		fields = append(fields, fmt.Sprintf("Content-Disposition: %s", m.CacheControl))
	}
	if m.WebsiteRedirectLocation != "" {
		fields = append(fields, fmt.Sprintf("x-iijgio-website-redirect-location: %s", m.WebsiteRedirectLocation))
	}
	v := strings.Join(fields, ",")
	if m.UserMetadata != nil {
		if v != "" {
			v += ","
		}
		v += m.UserMetadata.String()
	}
	return fmt.Sprintf("Metadata{%s}", v)
}

// AddUserMetadata add a user metadata.
func (m *ObjectMetadata) AddUserMetadata(name string, value string) {
	if m.UserMetadata == nil {
		var _m UserMetadata = map[string][]string{}
		m.UserMetadata = &_m
	}
	m.UserMetadata.Add(name, value)
}

// SetMetadata set metadata to HTTP header
func (m *ObjectMetadata) SetMetadata(h http.Header) {
	if m.ContentType != "" {
		h.Set("Content-Type", m.ContentType)
	}
	if m.ContentMD5 != "" {
		h.Set("Content-MD5", m.ContentMD5)
	}
	if m.ContentDisposition != "" {
		h.Set("Content-Disposition", m.ContentDisposition)
	}
	if m.ContentEncoding != "" {
		h.Set("Content-Encoding", m.ContentEncoding)
	}
	if m.CacheControl != "" {
		h.Set("Cache-Control", m.CacheControl)
	}
	if m.WebsiteRedirectLocation != "" {
		h.Set("x-iijgio-website-redirect-location", m.WebsiteRedirectLocation)
	}
	if m.UserMetadata != nil {
		for key, values := range *m.UserMetadata {
			for i := range values {
				h.Add(key, values[i])
			}
		}
	}
}

// GetUserMetadata returns user metadata
func (m *ObjectMetadata) GetUserMetadata(key string) string {
	if !strings.HasPrefix(key, "x-") {
		key = "x-iijgio-meta-" + key
	}
	if m.UserMetadata != nil {
		for _key, values := range *m.UserMetadata {
			if strings.ToLower(_key) == key {
				if len(values) > 0 {
					return values[0]
				}
			}
		}
	}
	return ""
}

// UserMetadata is object's custom metadata.
type UserMetadata map[string][]string

// Add a user metadata
func (um *UserMetadata) Add(name string, value string) {
	if um != nil {
		if !strings.HasPrefix(strings.ToLower(name), "x-") {
			name = "x-iijgio-meta-" + name
		}
		_um := *um
		if len(_um[name]) == 0 {
			_um[name] = []string{value}
		} else {
			_um[name] = append(_um[name], value)
		}
	}
}

func (um *UserMetadata) String() string {
	if um == nil {
		return ""
	}
	var _um []string
	for key, values := range *um {
		_um = append(_um, fmt.Sprintf("%s: [%s]", key, strings.Join(values, ",")))
	}
	return strings.Join(_um, ",")
}

// StorageSpace is a usage of the DAG storage.
type StorageSpace struct {
	XMLName      xml.Name `xml:"StorageSpaceInfo"`
	ContractUsed int64    `xml:"ContractUsed"`
	AccountUsed  int64    `xml:"AccountUsed"`
}

// ListTrafficResult is list of a DAG network traffic.
type ListTrafficResult struct {
	XMLName      xml.Name       `xml:"ListTrafficResult"`
	DownTraffics []*DownTraffic `xml:"DownTraffics"`
}

// DownTraffic is network traffic of the DAG storage.
type DownTraffic struct {
	ChargeDate string `xml:"ChargeDate"`
	Amount     int64  `xml:"Amount"`
}
