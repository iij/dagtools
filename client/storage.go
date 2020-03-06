package client

import (
	"bufio"
	"bytes"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"crypto/tls"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/iij/dagtools/env"
	"github.com/iij/dagtools/ini"
)

const (
	bufferSize                int64 = 4096
	defaultMultipartChunkSize int64 = 1073741824
	defaultRetry              int   = 2
	defaultRetryInterval      int64 = 3000 // 3 sec
)

var (
	nonValueSubResources = map[string]bool{
		"acl":      true,
		"location": true,
		"policy":   true,
		"uploads":  true,
		"cors":     true,
		"delete":   true,
		"space":    true,
		"traffic":  true,
		"website":  true,
	}
	subResources = map[string]bool{
		"partNumber": true,
		"uploadId":   true,
	}
)

func init() {
	for k, v := range nonValueSubResources {
		subResources[k] = v
	}
}

// StorageClient defines a client for the DAG storage.
type StorageClient interface {
	// -----------------------
	// Low Level APIs
	// -----------------------

	// Service API methods
	GetStorageSpace(region string) (*StorageSpace, error)
	ListNetworkTraffics(backwardTo int, region string) (*ListTrafficResult, error)
	GetNetworkTraffic(date string, region string) (*DownTraffic, error)
	GetRegions() (regions *Regions, err error)

	// Bucket API methods
	ListBuckets() (*BucketListing, error)
	SelectRegionPutBucket(bucket string, region string) error
	PutBucket(bucket string) error
	DeleteBucket(bucket string) error
	GetBucketLocation(bucket string) (bucketLocation string, err error)
	DoesBucketExist(bucket string) (bool, string, error)
	GetBucketPolicy(bucket string) (io.ReadCloser, error)
	PutBucketPolicy(bucket string, policy io.Reader) error
	DeleteBucketPolicy(bucket string) error

	// Object API methods
	ListObjects(bucket, prefix, marker, delimiter string, maxKeys int) (*ObjectListing, error)
	NextListObjects(previous *ObjectListing) (*ObjectListing, error)
	PutObject(bucket, key string, data *os.File, metadata *ObjectMetadata) error
	PutObjectAt(bucket, key string, data *os.File, off, length int64, metadata *ObjectMetadata) error
	PutObjectCopy(sourceBucket, sourceKey, distBucket, distKey string, metaData *ObjectSummary) error
	GetObject(bucket, key string) (io.ReadCloser, error)
	DoesObjectExist(bucket, key string) (bool, string, error)
	GetObjectSummary(bucket, key string) (*ObjectSummary, error)
	GetObjectMetadata(bucket, key string) (*Object, error)
	DeleteObject(bucket, key string) error
	DeleteMultipleObjects(bucket string, keys []string, quiet bool) (*MultipleDeletionResult, error)

	// MultipartUpload API methods
	ListMultipartUploads(bucket, prefix, keyMarker, uploadIdMarker, delimiter string, maxUploads int) (*MultipartUploadListing, error)
	NextListMultipartUploads(previous *MultipartUploadListing) (*MultipartUploadListing, error)
	InitiateMultipartUpload(bucket, key string, metadata *ObjectMetadata) (*MultipartUpload, error)
	AbortMultipartUpload(upload *MultipartUpload) error
	CompleteMultipartUpload(upload *MultipartUpload, parts []*Part) (*CompleteMultipartUploadResult, error)
	UploadPartCopy(upload *MultipartUpload, num int, sourceBucket, sourceKey string, rangeFirst, rangeLast int64) (part *Part, err error)
	ListParts(bucket, key, uploadId string, partNumberMarker, maxParts int) (*PartListing, error)
	NextListParts(previous *PartListing) (*PartListing, error)
	UploadPart(upload *MultipartUpload, num int, data *os.File) (*Part, error)
	UploadPartAt(upload *MultipartUpload, num int, data *os.File, off, length int64) (*Part, error)

	// Utility methods
	Sign(req *http.Request) error
	SetEndpoint(endpoint string)
	GetEndpoint() string

	// -----------------------
	// High Level API
	// -----------------------
	Upload(bucket, key string, data io.Reader, metadata *ObjectMetadata) error
	UploadFile(bucket, key string, fd *os.File, metadata *ObjectMetadata) error
	ResumeUploadFile(bucket, key, uploadId string, fd *os.File, metadata *ObjectMetadata) error
}

// DefaultStorageClient implements StorageClient
type DefaultStorageClient struct {
	env        *env.Environment
	Config     StorageClientConfig
	Logger     *log.Logger
	HTTPClient NewHTTPClient
}

// StorageClientConfig defines parameters for the Client
type StorageClientConfig struct {
	Endpoint           string
	AccessKeyID        string
	SecretAccessKey    string
	Secure             bool
	Proxy              string
	MultipartChunkSize int64
	TempDir            string
	Retry              int
	RetryInterval      time.Duration
	Vendor             string
	AbortOnFailure     bool
}

// NewStorageClient returns a initiated Client of DAG storage.
func NewStorageClient(env *env.Environment) (StorageClient, error) {
	var s *ini.Section
	if env.Config.HasSection("dagrin") {
		s = env.Config.Section("dagrin")
	} else {
		s = env.Config.Section("storage")
	}
	var (
		endpoint        = s.Get("endpoint", "storage-dag.iijgio.com")
		accessKeyID     = s.Get("accessKeyId", "")
		secretAccessKey = s.Get("secretAccessKey", "")
		secure          = s.GetBool("secure", true)
		chunkSize       = s.GetInt64("multipartChunkSize", defaultMultipartChunkSize)
		retry           = s.GetInt("retry", defaultRetry)
		retryInterval   = s.GetInt64("retryInterval", defaultRetryInterval)
		abortOnFailure  = s.GetBool("abortOnFailure", true)
		vendor          = s.Get("vendor", "IIJGIO")
		proxy           = env.Config.Get("dagtools", "proxy", "")
		tempDir         = env.Config.Get("dagtools", "tempDir", os.TempDir())
	)
	config := StorageClientConfig{
		Endpoint:           endpoint,
		AccessKeyID:        accessKeyID,
		SecretAccessKey:    secretAccessKey,
		Secure:             secure,
		Proxy:              proxy,
		MultipartChunkSize: chunkSize,
		TempDir:            tempDir,
		Retry:              retry,
		RetryInterval:      time.Duration(retryInterval) * time.Millisecond,
		AbortOnFailure:     abortOnFailure,
		Vendor:             vendor,
	}
	cli := DefaultStorageClient{
		Config: config,
		Logger: env.Logger,
	}
	cli.env = env
	cli.HTTPClient = NewDefaultHTTPClient
	return &cli, nil
}

// NewHTTPClient define ClientStorageClient instance
type NewHTTPClient func(cli *DefaultStorageClient) HTTPClient

// HTTPClient defines Do method for HTTP client
type HTTPClient interface {
	Do(req *http.Request) (resp *http.Response, err error)
}

// DefaultHTTPClient implements HTTP client
type DefaultHTTPClient struct {
	c *http.Client
}

// Do wrapper of http.Client.Do
func (client *DefaultHTTPClient) Do(req *http.Request) (resp *http.Response, err error) {
	return client.c.Do(req)
}

// NewDefaultHTTPClient implements HTTPClient
func NewDefaultHTTPClient(cli *DefaultStorageClient) HTTPClient {
	tr := &http.Transport{
		DisableCompression: true,
	}
	if cli.Config.Secure {
		if cli.env.Config.GetBool("storage", "insecureSkipVerify", false) {
			tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		}
	}
	// proxy
	if cli.Config.Proxy != "" {
		if proxyURL, err := url.Parse(cli.Config.Proxy); err == nil {
			tr.Proxy = http.ProxyURL(proxyURL)
		}
	}
	httpcli := &http.Client{Transport: tr}
	defaultCli := &DefaultHTTPClient{}
	defaultCli.c = httpcli
	return defaultCli
}

// Set a endpoint address from receive region
func (cli *DefaultStorageClient) SetEndpoint(endpoint string) {
	cli.Config.Endpoint = endpoint
}

func (cli *DefaultStorageClient) GetEndpoint() string {
	return cli.Config.Endpoint
}

// ListBuckets returns list of buckets (GET Service)
func (cli *DefaultStorageClient) ListBuckets() (listing *BucketListing, err error) {
	if cli.Config.AccessKeyID == "" {
		err = errors.New("please check your access_key_id and secret_access_key and try again")
		return
	}
	if cli.env.Debug {
		cli.env.Logger.Println("Storage REST API Call: GET Service (List Buckets)")
	}
	target := cli.Config.buildURL("", "", nil)
	resp, err := cli.DoAndRetry(func() (*http.Request, error) {
		req, err := http.NewRequest("GET", target, nil)
		if err != nil {
			cli.Logger.Printf("Failed to create a new HTTP request for ListObjects. reason: %v\n", err)
			return nil, err
		}
		return req, nil
	}, &listing)
	if err != nil {
		cli.Logger.Println("Failed to execute HTTP request.", err)
		return
	}
	defer resp.Body.Close()
	return
}

//Execute Get Bucket Location and return bucket location
func (cli *DefaultStorageClient) GetBucketLocation(bucket string) (bucketLocation string, err error) {
	if cli.env.Debug {
		cli.env.Logger.Println("Storage REST API Call: GET Bucket location")
	}
	target := cli.Config.buildURL(bucket, "", map[string]string{"location": ""})
	resp, err := cli.DoAndRetry(func() (*http.Request, error) {
		req, err := http.NewRequest("GET", target, nil)
		if err != nil {
			cli.Logger.Printf("Failed to get a bucket locations. reason: %v\n", err)
			return nil, err
		}
		return req, nil
	}, &bucketLocation)
	if err != nil {
		cli.Logger.Println("Failed to execute HTTP request.", err)
		return
	}
	defer resp.Body.Close()
	return
}

func (cli *DefaultStorageClient) GetRegions() (regions *Regions, err error) {
	if cli.env.Debug {
		cli.Logger.Println("Storage REST API Call: GET Regions")
	}
	target := cli.Config.buildURL("", "", map[string]string{"regions": ""})
	accesskey := cli.Config.AccessKeyID
	cli.Config.AccessKeyID = ""
	resp, err := cli.DoAndRetry(func() (*http.Request, error) {
		req, err := http.NewRequest("GET", target, nil)
		if err != nil {
			cli.Logger.Printf("Failed to get locations. reason: %v\n", err)
			return nil, err
		}
		return req, nil
	}, &regions)
	if err != nil {
		cli.Logger.Printf("Failed to execute HTTP request. reason: %s", err)
		return nil, err
	}
	defer resp.Body.Close()
	cli.Config.AccessKeyID = accesskey
	return
}

// SelectRegionPutBucket specify a region and creates new bucket (PUT Bucket)
func (cli *DefaultStorageClient) SelectRegionPutBucket(bucket string, region string) error {
	if cli.env.Debug {
		cli.env.Logger.Printf("Storage REST API Call: PUT Bucket {bucket: %q}", bucket)
	}
	defaultLocation := cli.Config.Endpoint
	location, err := cli.GetRegions()
	if err != nil {
		cli.Logger.Printf("Failed to execute Get Regions. reason: %v\n", err)
		return err
	}
	for _, r := range location.Regions {
		if r.Name == region {
			cli.Config.Endpoint = r.Endpoint
		}
	}
	target := cli.Config.buildURL(bucket, "", nil)
	resp, err := cli.DoAndRetry(func() (*http.Request, error) {
		req, err := http.NewRequest("PUT", target, nil)
		if err != nil {
			cli.Logger.Printf("Failed to create a new HTTP request for ListObjects. reason: %v\n", err)
			return nil, err
		}
		return req, nil
	}, nil)
	cli.Config.Endpoint = defaultLocation
	if err != nil {
		cli.Logger.Printf("Failed to execute HTTP request. reason: %s", err)
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		cli.Logger.Printf("Failed to execute HTTP request.")
		return err
	}
	return nil
}

// PutBucket creates new bucket (PUT bucket)
func (cli *DefaultStorageClient) PutBucket(bucket string) error {
	if cli.env.Debug {
		cli.env.Logger.Printf("Storage REST API Call: PUT Bucket {bucket: %q}", bucket)
	}
	target := cli.Config.buildURL(bucket, "", nil)
	resp, err := cli.DoAndRetry(func() (*http.Request, error) {
		req, err := http.NewRequest("PUT", target, nil)
		if err != nil {
			cli.Logger.Printf("Failed to create a new HTTP request for ListObjects. reason: %v\n", err)
			return nil, err
		}
		return req, nil
	}, nil)
	if err != nil {
		cli.Logger.Printf("Failed to execute HTTP request. reason: %s", err)
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		cli.Logger.Printf("Failed to execute HTTP request.")
		return err
	}
	return nil
}

// DeleteBucket removes a bucket (DELETE Bucket)
func (cli *DefaultStorageClient) DeleteBucket(bucket string) error {
	if cli.env.Debug {
		cli.env.Logger.Printf("Storage REST API Call: DELETE Bucket {bucket: %q}", bucket)
	}
	bucketLocation, err := cli.GetBucketLocation(bucket)
	if err != nil {
		cli.Logger.Printf("Failed to execute Get bucket location. reason: %v\n", err)
		return err
	}
	locations, err := cli.GetRegions()
	if err != nil {
		cli.Logger.Printf("Failed to execute Get Regions. reason: %v\n", err)
		return err
	}
	defaultEndpoint := cli.Config.Endpoint
	for _, r := range locations.Regions {
		if r.Name == bucketLocation {
			cli.Config.Endpoint = r.Endpoint
		}
	}
	target := cli.Config.buildURL(bucket, "", nil)
	resp, err := cli.DoAndRetry(func() (*http.Request, error) {
		req, err := http.NewRequest("DELETE", target, nil)
		if err != nil {
			cli.Logger.Printf("Failed to create a new HTTP request for ListObjects. reason: %v\n", err)
			return nil, err
		}
		return req, nil
	}, nil)
	cli.Config.Endpoint = defaultEndpoint
	if err != nil {
		cli.Logger.Printf("Failed to execute HTTP request. reason: %s", err)
		return err
	}
	defer resp.Body.Close()
	return nil
}

// DoesBucketExist returns true if the bucket exists.
func (cli *DefaultStorageClient) DoesBucketExist(bucket string) (result bool, bucketLocation string, err error) {
	result = false
	if cli.env.Debug {
		cli.env.Logger.Printf("Storage REST API Call: HEAD Bucket {bucket: %q}", bucket)
	}
	bucketLocation, err = cli.GetBucketLocation(bucket)
	if err != nil {
		cli.Logger.Printf("Failed to execute Get bucket location. reason: %v\n", err)
		return false, bucketLocation, err
	}
	locations, err := cli.GetRegions()
	if err != nil {
		cli.Logger.Printf("Failed to execute Get Regions. reason: %v\n", err)
		return false, bucketLocation, err
	}
	defaultEndpoint := cli.Config.Endpoint
	for _, r := range locations.Regions {
		if r.Name == bucketLocation {
			cli.Config.Endpoint = r.Endpoint
		}
	}
	target := cli.Config.buildURL(bucket, "", nil)
	resp, err := cli.DoAndRetry(func() (*http.Request, error) {
		req, err := http.NewRequest("HEAD", target, nil)
		if err != nil {
			cli.Logger.Printf("Failed to create a new HTTP request for ListObjects. reason: %v\n", err)
			return nil, err
		}
		return req, nil
	}, nil)
	cli.Config.Endpoint = defaultEndpoint
	if err != nil && resp == nil {
		return false, bucketLocation, err
	}
	defer resp.Body.Close()
	sc := resp.StatusCode
	if sc == 200 || sc == 404 {
		return sc == 200, bucketLocation, nil
	}
	cli.Logger.Printf("invalid response [status: %s]", resp.Status)
	return false, bucketLocation, err
}

// GetBucketPolicy gets a policy of the specified bucket.
func (cli *DefaultStorageClient) GetBucketPolicy(bucket string) (policy io.ReadCloser, err error) {
	if cli.env.Debug {
		cli.env.Logger.Printf("Storage REST API Call: GET Bucket policy {Bucket: %s}", bucket)
	}
	bucketLocation, err := cli.GetBucketLocation(bucket)
	if err != nil {
		cli.Logger.Printf("Failed to execute Get bucket location. reason: %v\n", err)
		return nil, err
	}
	locations, err := cli.GetRegions()
	if err != nil {
		cli.Logger.Printf("Failed to execute Get Regions. reason: %v\n", err)
		return nil, err
	}
	defaultEndpoint := cli.Config.Endpoint
	for _, r := range locations.Regions {
		if r.Name == bucketLocation {
			cli.Config.Endpoint = r.Endpoint
		}
	}
	target := cli.Config.buildURL(bucket, "", map[string]string{"policy": ""})
	resp, err := cli.DoAndRetry(func() (*http.Request, error) {
		req, err := http.NewRequest("GET", target, nil)
		if err != nil {
			cli.Logger.Printf("Failed to create a new HTTP request for ListObjects. reason: %v\n", err)
			return nil, err
		}
		return req, nil
	}, nil)
	if err != nil {
		cli.Logger.Println("Failed to get a bucket policy.", err)
		return
	}
	cli.Config.Endpoint = defaultEndpoint
	switch resp.StatusCode {
	case 404:
		return
	case 200:
		policy = resp.Body
		return
	default:
		cli.Logger.Println("invalid response")
		err = errors.New("invalid response")
	}
	defer resp.Body.Close()
	return
}

// PutBucketPolicy puts a policy of the specified bucket.
func (cli *DefaultStorageClient) PutBucketPolicy(bucket string, r io.Reader) error {
	if cli.env.Debug {
		cli.env.Logger.Printf("Storage REST API Call: PUT Bucket policy {bucket: %q}", bucket)
	}
	bucketLocation, err := cli.GetBucketLocation(bucket)
	if err != nil {
		cli.Logger.Printf("Failed to execute Get bucket location. reason: %v\n", err)
		return err
	}
	locations, err := cli.GetRegions()
	if err != nil {
		cli.Logger.Printf("Failed to execute Get Regions. reason: %v\n", err)
		return err
	}
	defaultEndpoint := cli.Config.Endpoint
	for _, r := range locations.Regions {
		if r.Name == bucketLocation {
			cli.Config.Endpoint = r.Endpoint
		}
	}
	target := cli.Config.buildURL(bucket, "", map[string]string{"policy": ""})
	resp, err := cli.DoAndRetry(func() (*http.Request, error) {
		req, err := http.NewRequest("PUT", target, r)
		if err != nil {
			cli.Logger.Printf("Failed to create a new HTTP request for ListObjects. reason: %v\n", err)
			return nil, err
		}
		return req, nil
	}, nil)
	if err != nil {
		cli.Logger.Println("Failed to put a bucket policy.", err)
		return err
	}
	cli.Config.Endpoint = defaultEndpoint
	defer resp.Body.Close()
	if resp.StatusCode != 204 {
		cli.Logger.Println("invalid response")
		return errors.New("invalid response")
	}
	return nil
}

// DeleteBucketPolicy deletes a policy of the specified bucket.
func (cli *DefaultStorageClient) DeleteBucketPolicy(bucket string) error {
	if cli.env.Debug {
		cli.env.Logger.Printf("Storage REST API Call: DELETE Bucket policy {bucket: %q}", bucket)
	}
	bucketLocation, err := cli.GetBucketLocation(bucket)
	if err != nil {
		cli.Logger.Printf("Failed to execute Get bucket location. reason: %v\n", err)
		return err
	}
	locations, err := cli.GetRegions()
	if err != nil {
		cli.Logger.Printf("Failed to execute Get Regions. reason: %v\n", err)
		return err
	}
	defaultEndpoint := cli.Config.Endpoint
	for _, r := range locations.Regions {
		if r.Name == bucketLocation {
			cli.Config.Endpoint = r.Endpoint
		}
	}
	target := cli.Config.buildURL(bucket, "", map[string]string{"policy": ""})
	resp, err := cli.DoAndRetry(func() (*http.Request, error) {
		req, err := http.NewRequest("DELETE", target, nil)
		if err != nil {
			cli.Logger.Printf("Failed to create a new HTTP request for ListObjects. reason: %v\n", err)
			return nil, err
		}
		return req, nil
	}, nil)
	if err != nil {
		cli.Logger.Println("Failed to delete a bucket policy.", err)
		return err
	}
	cli.Config.Endpoint = defaultEndpoint
	defer resp.Body.Close()
	if resp.StatusCode != 204 {
		cli.Logger.Println("invalid response")
		return errors.New("invalid response")
	}
	return nil
}

// ListObjects returns list of objects (GET Bucket := List Objects)
func (cli *DefaultStorageClient) ListObjects(bucket, prefix, marker, delimiter string, maxKeys int) (listing *ObjectListing, err error) {
	if cli.env.Debug {
		cli.env.Logger.Printf("Storage REST API Call: GET Bucket (List Objects) {bucket: %q, prefix: %q, marker: %q, delimiter: %q, maxKeys: %d}",
			bucket, prefix, marker, delimiter, maxKeys)
	}
	bucketLocation, err := cli.GetBucketLocation(bucket)
	if err != nil {
		cli.Logger.Printf("Failed to execute Get bucket location. reason: %v\n", err)
		return nil, err
	}
	locations, err := cli.GetRegions()
	if err != nil {
		cli.Logger.Printf("Failed to execute Get Regions. reason: %v\n", err)
		return nil, err
	}
	defaultEndpoint := cli.Config.Endpoint
	for _, r := range locations.Regions {
		if r.Name == bucketLocation {
			cli.Config.Endpoint = r.Endpoint
		}
	}
	resp, err := cli.DoAndRetry(func() (*http.Request, error) {
		req, err := cli.NewListObjectsRequest(bucket, prefix, marker, delimiter, maxKeys)
		if err != nil {
			cli.Logger.Printf("Failed to create a new HTTP request for ListObjects. reason: %v\n", err)
			return nil, err
		}
		return req, nil
	}, &listing)
	if err != nil {
		cli.Logger.Printf("Failed to execute HTTP request. reason: %v\n", err)
		return nil, err
	}
	cli.Config.Endpoint = defaultEndpoint
	defer resp.Body.Close()
	return
}

// NewListObjectsRequest returns new a request for ListObjects
func (cli *DefaultStorageClient) NewListObjectsRequest(bucket, prefix, marker, delimiter string, maxKeys int) (*http.Request, error) {
	queries := map[string]string{
		"max-keys": "100",
	}
	if prefix != "" {
		queries["prefix"] = prefix
	}
	if marker != "" {
		queries["marker"] = marker
	}
	if delimiter != "" {
		queries["delimiter"] = delimiter
	}
	if maxKeys > 0 {
		queries["max-keys"] = strconv.Itoa(maxKeys)
	}
	target := cli.Config.buildURL(bucket, "", queries)
	req, err := http.NewRequest("GET", target, nil)
	return req, err
}

// NextListObjects returns next page of object listing.
func (cli *DefaultStorageClient) NextListObjects(listing *ObjectListing) (*ObjectListing, error) {
	return cli.ListObjects(listing.Name, listing.Prefix, listing.NextMarker, listing.Delimiter, listing.MaxKeys)
}

// PutObject uploads a file (PUT Object)
func (cli *DefaultStorageClient) PutObject(bucket, key string, f *os.File, metadata *ObjectMetadata) error {
	stat, err := f.Stat()
	if err != nil {
		return err
	}
	size := stat.Size()
	return cli.PutObjectAt(bucket, key, f, 0, size, metadata)
}

// PutObjectAt uploads n bytes from the File starting at byte offset off.
func (cli *DefaultStorageClient) PutObjectAt(bucket, key string, f *os.File, off, n int64, metadata *ObjectMetadata) error {
	if cli.env.Debug {
		cli.env.Logger.Printf("Storage REST API Call: PUT Object {bucket: %q, key: %q}", bucket, key)
	}
	bucketLocation, err := cli.GetBucketLocation(bucket)
	if err != nil {
		cli.Logger.Printf("Failed to get bucket location. reason: %v\n", err)
	}
	locations, err := cli.GetRegions()
	if err != nil {
		cli.Logger.Printf("Failed to execute Get Regions. reason: %v\n", err)
		return err
	}
	defaultEndpoint := cli.Config.Endpoint
	for _, r := range locations.Regions {
		if r.Name == bucketLocation {
			cli.Config.Endpoint = r.Endpoint
		}
	}
	target := cli.Config.buildURL(bucket, key, nil)
	resp, err := cli.DoAndRetry(func() (*http.Request, error) {
		r := io.NewSectionReader(f, off, n)
		req, err := http.NewRequest("PUT", target, r)
		if err != nil {
			cli.Logger.Printf("Failed to create a new HTTP request for ListParts. reason: %v\n", err)
			return nil, err
		}
		if metadata != nil {
			metadata.SetMetadata(req.Header)
		}
		req.ContentLength = n
		req.Header.Set("Content-Type", GetMimeType(key))
		return req, nil
	}, nil)
	if err != nil {
		return err
	}
	cli.Config.Endpoint = defaultEndpoint
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return errors.New("invalid response")
	}
	return nil
}

func (cli *DefaultStorageClient) PutObjectCopy(sourceBucket, sourceKey, destBucket, destKey string, metaData *ObjectSummary) error {
	if metaData != nil && (metaData.Size > cli.Config.MultipartChunkSize) {
		return cli.MultiPartUploadCopy(sourceBucket, sourceKey, destBucket, destKey, metaData)
	}
	if cli.env.Debug {
		cli.env.Logger.Printf("Storage REST API Call: PUT Object(copy) {source bucket: %q, source key: %q, dist bucket: %q, dist key: %q}", sourceBucket, sourceKey, destBucket, destKey)
	}
	target := cli.Config.buildURL(destBucket, destKey, nil)
	resp, err := cli.DoAndRetry(func() (*http.Request, error) {
		req, err := http.NewRequest("PUT", target, nil)
		if err != nil {
			cli.Logger.Printf("Failed to create a new HTTP request for put object(copy). reason: %v\n", err)
			return nil, err
		}
		source := "/" + sourceBucket + "/" + sourceKey
		source = url.QueryEscape(source)
		req.Header.Set("x-iijgio-copy-source", source)
		if metaData != nil {
			etag := url.QueryEscape(metaData.ETag)
			req.Header.Set("x-iijgio-copy-source-if-none-match", etag)
		}
		return req, nil
	}, nil)
	if err != nil {
		cli.Logger.Println("Failed to execute HTTP request.", err)
		return err
	}
	defer resp.Body.Close()
	return nil
}

// GetObject downloads an object (GET Object)
func (cli *DefaultStorageClient) GetObject(bucket, key string) (r io.ReadCloser, err error) {
	if cli.env.Debug {
		cli.env.Logger.Printf("Storage REST API Call: GET Object {bucket: %q, key: %q}", bucket, key)
	}
	bucketLocation, err := cli.GetBucketLocation(bucket)
	if err != nil {
		cli.Logger.Printf("Failed to get bucket location. reason: %v\n", err)
	}
	locations, err := cli.GetRegions()
	if err != nil {
		cli.Logger.Printf("Failed to execute Get Regions. reason: %v\n", err)
		return nil, err
	}
	defaultEndpoint := cli.Config.Endpoint
	for _, r := range locations.Regions {
		if r.Name == bucketLocation {
			cli.Config.Endpoint = r.Endpoint
		}
	}
	target := cli.Config.buildURL(bucket, key, nil)
	resp, err := cli.DoAndRetry(func() (*http.Request, error) {
		req, err := http.NewRequest("GET", target, nil)
		if err != nil {
			cli.Logger.Printf("Failed to create a new HTTP request for ListParts. reason: %v\n", err)
			return nil, err
		}
		return req, nil
	}, nil)
	cli.Config.Endpoint = defaultEndpoint
	if err != nil {
		cli.Logger.Println("Failed to execute HTTP request.", err)
		return
	}
	if resp.StatusCode != 200 {
		cli.Logger.Println("Failed to execute HTTP request.")
		return nil, errors.New("invalid response")
	}
	r = resp.Body
	return
}

// DoesObjectExist returns presence of an object (HEAD Object)
func (cli *DefaultStorageClient) DoesObjectExist(bucket, key string) (bool, string, error) {
	if cli.env.Debug {
		cli.env.Logger.Printf("Storage REST API Call: HEAD Object {bucket: %q, key: %q}", bucket, key)
	}
	bucketLocation, err := cli.GetBucketLocation(bucket)
	if err != nil {
		cli.Logger.Printf("Failed to get bucket Location. reason: %v\n", err)
		return false, bucketLocation, err
	}
	locations, err := cli.GetRegions()
	if err != nil {
		cli.Logger.Printf("Failed to execute Get Regions. reason: %v\n", err)
		return false, bucketLocation, err
	}
	defaultLocation := cli.Config.Endpoint
	for _, r := range locations.Regions {
		if r.Name == bucketLocation {
			cli.Config.Endpoint = r.Endpoint
		}
	}
	target := cli.Config.buildURL(bucket, key, nil)
	resp, err := cli.DoAndRetry(func() (*http.Request, error) {
		req, err := http.NewRequest("HEAD", target, nil)
		if err != nil {
			cli.Logger.Printf("Failed to create a new HTTP request for ListParts. reason: %v\n", err)
			return nil, err
		}
		return req, nil
	}, nil)
	if resp == nil && err != nil {
		return false, bucketLocation, err
	}
	cli.Config.Endpoint = defaultLocation
	defer resp.Body.Close()
	sc := resp.StatusCode
	if sc == 200 || sc == 404 {
		return sc == 200, bucketLocation, nil
	}
	cli.Logger.Printf("invalid response [status: %s]", resp.Status)
	return false, bucketLocation, err
}

// GetObjectSummary returns summary information of the Object.
func (cli *DefaultStorageClient) GetObjectSummary(bucket, key string) (os *ObjectSummary, err error) {
	if cli.env.Debug {
		cli.env.Logger.Printf("Storage REST API Call: HEAD Object {bucket: %q, key: %q}", bucket, key)
	}
	target := cli.Config.buildURL(bucket, key, nil)
	resp, err := cli.DoAndRetry(func() (*http.Request, error) {
		req, err := http.NewRequest("HEAD", target, nil)
		if err != nil {
			cli.Logger.Printf("Failed to create a new HTTP request for ListParts. reason: %v\n", err)
			return nil, err
		}
		return req, nil
	}, nil)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode == 200 {
		lastModified, _ := http.ParseTime(resp.Header.Get("Last-Modified"))
		return &ObjectSummary{
			Key:          key,
			LastModified: lastModified,
			ETag:         resp.Header.Get("ETag"),
			Size:         resp.ContentLength,
		}, nil
	}
	return
}

// GetObjectMetadata returns metadata of object. if the object does not exist, return nil.
func (cli *DefaultStorageClient) GetObjectMetadata(bucket, key string) (o *Object, err error) {
	if cli.env.Debug {
		cli.env.Logger.Printf("Storage REST API Call: HEAD Object {bucket: %q, key: %q}", bucket, key)
	}
	bucketLocation, err := cli.GetBucketLocation(bucket)
	if err != nil {
		cli.Logger.Printf("Failed to get bucket Location. reason: %v\n", err)
		return nil, err
	}
	locations, err := cli.GetRegions()
	if err != nil {
		cli.Logger.Printf("Failed to execute Get Regions. reason: %v\n", err)
		return nil, err
	}
	defaultLocation := cli.Config.Endpoint
	for _, r := range locations.Regions {
		if r.Name == bucketLocation {
			cli.Config.Endpoint = r.Endpoint
		}
	}
	target := cli.Config.buildURL(bucket, key, nil)
	resp, err := cli.DoAndRetry(func() (*http.Request, error) {
		req, err := http.NewRequest("HEAD", target, nil)
		if err != nil {
			cli.Logger.Printf("Failed to create a new HTTP request for ListParts. reason: %v\n", err)
			return nil, err
		}
		return req, nil
	}, nil)
	cli.Config.Endpoint = defaultLocation
	if resp == nil && err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 {
		return nil, nil
	}
	if resp.StatusCode == 200 {
		o = new(Object)
		o.Bucket = bucket
		o.Key = key
		o.ETag = resp.Header.Get("ETag")
		o.LastModified, _ = http.ParseTime(resp.Header.Get("Last-Modified"))
		o.Size = resp.ContentLength
		m := ObjectMetadata{
			ContentLength:           resp.ContentLength,
			ContentType:             resp.Header.Get("Content-Type"),
			ContentMD5:              resp.Header.Get("Content-MD5"),
			ContentDisposition:      resp.Header.Get("Content-Disposition"),
			ContentEncoding:         resp.Header.Get("Content-Encoding"),
			CacheControl:            resp.Header.Get("Cache-Control"),
			WebsiteRedirectLocation: resp.Header.Get("x-iijgio-website-redirect-location"),
		}
		for key, values := range resp.Header {
			_key := strings.ToLower(key)
			if strings.HasPrefix(_key, "x-iijgio-meta-") || strings.HasPrefix(_key, "x-amz-meta-") {
				for i := range values {
					m.AddUserMetadata(_key, values[i])
				}
			}
		}
		o.Metadata = &m
	}
	return
}

// DeleteObject removes an object (DELETE Object)
func (cli *DefaultStorageClient) DeleteObject(bucket, key string) (err error) {
	if cli.env.Debug {
		cli.env.Logger.Printf("Storage REST API Call: DELETE Object {bucket: %q, key: %q}", bucket, key)
	}
	bucketLocation, err := cli.GetBucketLocation(bucket)
	if err != nil {
		cli.Logger.Printf("Failed to get bucket Location. reason: %v\n", err)
		return err
	}
	locations, err := cli.GetRegions()
	if err != nil {
		cli.Logger.Printf("Failed to execute Get Regions. reason: %v\n", err)
		return err
	}
	defaultLocation := cli.Config.Endpoint
	for _, r := range locations.Regions {
		if r.Name == bucketLocation {
			cli.Config.Endpoint = r.Endpoint
		}
	}
	target := cli.Config.buildURL(bucket, key, nil)
	resp, err := cli.DoAndRetry(func() (*http.Request, error) {
		req, err := http.NewRequest("DELETE", target, nil)
		if err != nil {
			cli.Logger.Printf("Failed to create a new HTTP request for ListParts. reason: %v\n", err)
			return nil, err
		}
		return req, nil
	}, nil)
	cli.Config.Endpoint = defaultLocation
	if err != nil {
		cli.Logger.Println("Failed to execute HTTP request.", err)
		return
	}
	defer resp.Body.Close()
	return
}

// DeleteMultipleObjects removes objects (DELETE Multiple Objects)
func (cli *DefaultStorageClient) DeleteMultipleObjects(bucket string, keys []string, quiet bool) (res *MultipleDeletionResult, err error) {
	_keys := make([]multipleDeletionKey, len(keys))
	for i, key := range keys {
		_keys[i] = multipleDeletionKey{Key: key}
	}
	if cli.env.Debug {
		cli.env.Logger.Printf("Storage REST API Call: Delete Multiple Objects {bucket: %q, key: %q, quiet: %v}", bucket, keys, quiet)
	}
	bucketLocation, err := cli.GetBucketLocation(bucket)
	if err != nil {
		cli.Logger.Printf("Failed to execute Get bucket location. reason: %v\n", err)
		return nil, err
	}
	locations, err := cli.GetRegions()
	if err != nil {
		cli.Logger.Printf("Failed to execute Get Regions. reason: %v\n", err)
		return nil, err
	}
	defaultLocation := cli.Config.Endpoint
	for _, r := range locations.Regions {
		if r.Name == bucketLocation {
			cli.Config.Endpoint = r.Endpoint
		}
	}
	request := multipleDeletionRequest{Quiet: quiet, Keys: _keys}
	target := cli.Config.buildURL(bucket, "", map[string]string{"delete": ""})
	resp, err := cli.DoAndRetry(func() (*http.Request, error) {
		b, err := xml.Marshal(request)
		if err != nil {
			return nil, err
		}
		r := bytes.NewReader(b)
		req, err := http.NewRequest("POST", target, r)
		if err != nil {
			cli.Logger.Printf("Failed to create a new HTTP request for ListParts. reason: %v\n", err)
		}
		req.Header.Set("Content-Type", "text/xml")
		return req, nil
	}, &res)
	if err != nil {
		return
	}
	cli.Config.Endpoint = defaultLocation
	defer resp.Body.Close()
	return res, nil
}

// ListMultipartUploads returns list of multipart-uploads
func (cli *DefaultStorageClient) ListMultipartUploads(bucket, prefix, keyMarker, uploadIdMarker, delimiter string, maxUploads int) (listing *MultipartUploadListing, err error) {
	queries := map[string]string{
		"uploads":     "",
		"max-uploads": "100",
	}
	if prefix != "" {
		queries["prefix"] = prefix
	}
	if keyMarker != "" {
		queries["key-marker"] = keyMarker
	}
	if uploadIdMarker != "" {
		queries["upload-id-marker"] = uploadIdMarker
	}
	if delimiter != "" {
		queries["delimiter"] = delimiter
	}
	if maxUploads > 0 {
		queries["max-uploads"] = strconv.Itoa(maxUploads)
	}
	bucketLocation, err := cli.GetBucketLocation(bucket)
	if err != nil {
		cli.Logger.Printf("Failed to execute Get bucket location. reason: %v\n", err)
		return nil, err
	}
	locations, err := cli.GetRegions()
	if err != nil {
		cli.Logger.Printf("Failed to execute Get Regions. reason: %v\n", err)
		return nil, err
	}
	defaultEndpoint := cli.Config.Endpoint
	for _, r := range locations.Regions {
		if r.Name == bucketLocation {
			cli.Config.Endpoint = r.Endpoint
		}
	}
	target := cli.Config.buildURL(bucket, "", queries)
	resp, err := cli.DoAndRetry(func() (*http.Request, error) {
		req, err := http.NewRequest("GET", target, nil)
		if err != nil {
			cli.Logger.Printf("Failed to create a new HTTP request for ListObjects. reason: %v\n", err)
			return nil, err
		}
		return req, nil
	}, &listing)
	if err != nil {
		cli.Logger.Printf("Failed to execute HTTP request. reason: %v\n", err)
		return nil, err
	}
	cli.Config.Endpoint = defaultEndpoint
	defer resp.Body.Close()
	prefixes := make([]string, 0)
	for _, cm := range listing.CommonPrefixes {
		if cm != "" {
			prefixes = append(prefixes, cm)
		}
	}
	listing.CommonPrefixes = prefixes
	return
}

// NextListMultipartUploads returns next page of multipart-uploads listing
func (cli *DefaultStorageClient) NextListMultipartUploads(prev *MultipartUploadListing) (listing *MultipartUploadListing, err error) {
	if !prev.IsTruncated {
		return nil, errors.New("the listing is not truncated")
	}
	return cli.ListMultipartUploads(prev.Bucket, prev.Prefix, prev.NextKeyMarker, prev.NextUploadIdMarker, prev.Delimiter, prev.MaxUploads)
}

// InitiateMultipartUpload starts the multipart upload
func (cli *DefaultStorageClient) InitiateMultipartUpload(bucket, key string, metadata *ObjectMetadata) (res *MultipartUpload, err error) {
	if cli.env.Debug {
		cli.env.Logger.Printf("Storage REST API Call: Initiate Multipart Upload {bucket: %q, key: %q, metadata: %q}", bucket, key, metadata)
	}
	target := cli.Config.buildURL(bucket, key, map[string]string{"uploads": ""})
	resp, err := cli.DoAndRetry(func() (*http.Request, error) {
		req, err := http.NewRequest("POST", target, nil)
		if err != nil {
			cli.Logger.Printf("Failed to create a new HTTP request for ListParts. reason: %v\n", err)
			return nil, err
		}
		if metadata != nil {
			metadata.SetMetadata(req.Header)
		}
		return req, nil
	}, &res)
	if err != nil {
		cli.Logger.Println("Failed to initiate a new multipart upload.", err)
		return
	}
	defer resp.Body.Close()
	return
}

// AbortMultipartUpload cancels the multipart upload.
func (cli *DefaultStorageClient) AbortMultipartUpload(upload *MultipartUpload) error {
	if cli.env.Debug {
		cli.env.Logger.Printf("Abort Multipart Upload: %v", upload)
	}
	bucketLocation, err := cli.GetBucketLocation(upload.Bucket)
	if err != nil {
		cli.Logger.Printf("Failed to get bucket Location. reason: %v\n", err)
		return err
	}
	locations, err := cli.GetRegions()
	if err != nil {
		cli.Logger.Printf("Failed to execute Get Regions. reason: %v\n", err)
		return err
	}
	defaultLocation := cli.Config.Endpoint
	for _, r := range locations.Regions {
		if r.Name == bucketLocation {
			cli.Config.Endpoint = r.Endpoint
		}
	}
	target := cli.Config.buildURL(upload.Bucket, upload.Key, map[string]string{"uploadId": upload.UploadID})
	resp, err := cli.DoAndRetry(func() (*http.Request, error) {
		req, err := http.NewRequest("DELETE", target, nil)
		if err != nil {
			cli.Logger.Printf("Failed to create a new HTTP request for ListParts. reason: %v\n", err)
			return nil, err
		}
		return req, nil
	}, nil)
	if err != nil {
		cli.Logger.Println(err.Error())
		return err
	}
	cli.Config.Endpoint = defaultLocation
	defer resp.Body.Close()
	if resp.StatusCode != 204 {
		cli.Logger.Printf("Failed to abort the multipart upload. StatusCode: 204 != %v", resp.StatusCode)
		err = errors.New("failed to abort the multipart upload")
	}
	return nil
}

// CompleteMultipartUpload creates a storage object
func (cli *DefaultStorageClient) CompleteMultipartUpload(upload *MultipartUpload, parts []*Part) (res *CompleteMultipartUploadResult, err error) {
	if cli.env.Debug {
		cli.env.Logger.Printf("Storage REST API Call: Complete Multipart Upload {upload: %v, parts: %v}", upload, parts)
	}
	var request = completeMultipartUploadRequest{Parts: parts}
	target := cli.Config.buildURL(upload.Bucket, upload.Key, map[string]string{"uploadId": upload.UploadID})
	resp, err := cli.DoAndRetry(func() (*http.Request, error) {
		b, _ := xml.Marshal(request)
		r := bytes.NewReader(b)
		req, err := http.NewRequest("POST", target, r)
		req.Header.Set("Content-Type", "text/xml")
		if err != nil {
			cli.Logger.Printf("Failed to create a new HTTP request for ListParts. reason: %v\n", err)
			return nil, err
		}
		return req, nil
	}, &res)
	if err != nil {
		cli.Logger.Println("Failed to complete the multipart uploads.", err)
		return
	}
	defer resp.Body.Close()
	return
}

// PUT object Copy via multipart upload
func (cli *DefaultStorageClient) MultiPartUploadCopy(sourceBucket, sourceKey, destBucket, destKey string, metaData *ObjectSummary) (err error) {
	if cli.env.Debug {
		cli.env.Logger.Printf("Storage REST API Call: Multipart Upload Copy {sourceObject: %s:%s, destObject: %s:%s}", sourceBucket, sourceKey, destBucket, destKey)
	}
	var (
		num     = int(math.Ceil(float64(metaData.Size) / float64(cli.Config.MultipartChunkSize)))
		parts   = make([]*Part, 1000)
		wg      sync.WaitGroup
		upload  *MultipartUpload
		ok      = true
		channel = make(chan bool)
	)
	defer func() {
		if !ok {
			if upload != nil && cli.Config.AbortOnFailure {
				cli.AbortMultipartUpload(upload)
			}
			if err == nil {
				err = errors.New("failed to object copy")
			}
		}
	}()
	upload, err = cli.InitiateMultipartUpload(destBucket, destKey, nil)
	if err != nil {
		return err
	}
	for i := 1; i <= num; i++ {
		wg.Add(1)
		go func(i int) {
			<-channel
			defer func() {
				wg.Done()
				channel <- true
			}()
			cli.Logger.Printf("Upload a part number %d\n", i)
			rangeFirst := cli.Config.MultipartChunkSize * int64(i-1)
			rangeLast := cli.Config.MultipartChunkSize*int64(i) - 1
			cli.Logger.Printf("Part range : %d - %d\n", rangeFirst, rangeLast)
			part, err := cli.UploadPartCopy(upload, i, sourceBucket, sourceKey, rangeFirst, rangeLast)
			if part == nil || err != nil {
				ok = false
				return
			}
			parts[i-1] = part
			cli.Logger.Printf("Finished to upload Part(%s).", part)
		}(i)
	}
	concurrency := cli.env.Concurrency
	if concurrency > num {
		concurrency = num
	}
	for i := 0; i < concurrency; i++ {
		channel <- true
	}
	wg.Wait()
	if ok {
		if _, err = cli.CompleteMultipartUpload(upload, parts); err != nil {
			ok = false
		} else {
			cli.Logger.Printf("Succeeded to upload %s:%s as %s:%s\n", sourceBucket, sourceKey, destBucket, destKey)
		}
	}
	return
}

func (cli *DefaultStorageClient) UploadPartCopy(upload *MultipartUpload, num int, sourceBucket, sourceKey string, rangeFirst, rangeLast int64) (part *Part, err error) {
	if cli.env.Debug {
		cli.env.Logger.Printf("Storage REST API Call: Upload Part(copy) {upload: %v, num: %d}", upload, num)
	}
	var p Part
	target := cli.Config.buildURL(upload.Bucket, upload.Key, map[string]string{"partNumber": strconv.Itoa(num), "uploadId": upload.UploadID})
	resp, err := cli.DoAndRetry(func() (*http.Request, error) {
		req, err := http.NewRequest("PUT", target, nil)
		if err != nil {
			cli.Logger.Printf("Failed to create a new HTTP request for put object(copy). reason: %v\n", err)
			return nil, err
		}
		source := "/" + sourceBucket + "/" + sourceKey
		source = url.QueryEscape(source)
		byteRange := fmt.Sprintf("%s-%s", strconv.Itoa(int(rangeFirst)), strconv.Itoa(int(rangeLast)))
		byteRange = url.QueryEscape(byteRange)
		req.Header.Set("x-iijgio-copy-source", source)
		req.Header.Set("x-iijgio-copy-source-range", fmt.Sprintf("bytes=%s", byteRange))
		return req, nil
	}, &p)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	//part.ETag = resp.Header.Get("ETag")
	p.PartNumber = num
	part = &p
	return
}

// ListParts returns list of parts (List Parts)
func (cli *DefaultStorageClient) ListParts(bucket, key, uploadId string, partNumberMarker, maxParts int) (listing *PartListing, err error) {
	queries := map[string]string{
		"uploadId":  uploadId,
		"max-parts": "100",
	}
	if partNumberMarker > 0 {
		queries["part-number-marker"] = strconv.Itoa(partNumberMarker)
	}
	if maxParts > 0 {
		queries["max-parts"] = strconv.Itoa(maxParts)
	}
	target := cli.Config.buildURL(bucket, key, queries)
	resp, err := cli.DoAndRetry(func() (*http.Request, error) {
		req, err := http.NewRequest("GET", target, nil)
		if err != nil {
			cli.Logger.Printf("Failed to create a new HTTP request for ListParts. reason: %v\n", err)
			return nil, err
		}
		return req, nil
	}, &listing)
	if err != nil {
		cli.Logger.Printf("Failed to execute a HTTP request. reason: %v\n", err)
	}
	defer resp.Body.Close()
	return listing, err
}

// NextListParts returns next page of parts
func (cli *DefaultStorageClient) NextListParts(prev *PartListing) (listing *PartListing, err error) {
	if !prev.IsTruncated {
		return nil, errors.New("the listing is not truncated")
	}
	return cli.ListParts(prev.Bucket, prev.Key, prev.UploadId, prev.NextPartNumberMarker, prev.MaxParts)
}

// UploadPart uploads a part of the multipart upload
func (cli *DefaultStorageClient) UploadPart(upload *MultipartUpload, num int, f *os.File) (p *Part, err error) {
	stat, err := f.Stat()
	if err != nil {
		return
	}
	size := stat.Size()
	return cli.UploadPartAt(upload, num, f, 0, size)
}

// UploadPartAt uploads n bytes from the File starting at byte offset off.
func (cli *DefaultStorageClient) UploadPartAt(upload *MultipartUpload, num int, f *os.File, off, n int64) (p *Part, err error) {
	if cli.env.Debug {
		cli.env.Logger.Printf("Storage REST API Call: Upload Part {upload: %v, num: %d}", upload, num)
	}
	target := cli.Config.buildURL(upload.Bucket, upload.Key,
		map[string]string{"partNumber": strconv.Itoa(num), "uploadId": upload.UploadID})
	var r DigestReader
	resp, err := cli.DoAndRetry(func() (*http.Request, error) {
		r = DigestReader{r: io.NewSectionReader(f, off, n), h: md5.New()}
		req, err := http.NewRequest("PUT", target, r)
		if err != nil {
			cli.Logger.Printf("Failed to create a new HTTP request for ListParts. reason: %v\n", err)
			return nil, err
		}
		req.ContentLength = n
		return req, nil
	}, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	p = &Part{PartNumber: num, ETag: fmt.Sprintf(`"%x"`, r.Digest())}
	return p, nil
}

// Upload uploads data as storage object from read the io.Reader
func (cli *DefaultStorageClient) Upload(bucket, key string, r io.Reader, metadata *ObjectMetadata) (err error) {
	logger := cli.env.Logger
	logger.Printf("Uploading to %s:%s ...", bucket, key)
	var (
		count  int
		size   int64
		out    *os.File
		wg     sync.WaitGroup
		upload *MultipartUpload
		parts  = make([]*Part, 1000)
		num    = 1
		ok     = true
	)
	defer func() {
		if !ok {
			if err != nil {
				if upload != nil && cli.Config.AbortOnFailure {
					cli.AbortMultipartUpload(upload)
				}
				if err == nil {
					err = errors.New("failed to upload file(s)")
				}
			}
		}
	}()
	buf := make([]byte, bufferSize)
	uploadChannel := make(chan bool, cli.env.Concurrency)
	tmpFileWriteChannel := make(chan bool, cli.env.Concurrency+1)
	for {
		n, _ := r.Read(buf)
		if n < 1 {
			break
		}
		size += int64(n)
		if out == nil {
			tmpFileWriteChannel <- true
			out, err = ioutil.TempFile(cli.Config.TempDir, "dagtools-")
			if err != nil {
				logger.Printf("Failed to create a temporary file.")
				return
			}
		}
		if _, err = out.Write(buf[0:n]); err != nil {
			out.Close()
			return err
		}
		count++
		if cli.Config.MultipartChunkSize < size {
			// initiate mutlipart upload
			if upload == nil {
				upload, err = cli.InitiateMultipartUpload(bucket, key, metadata)
				if err != nil {
					ok = false
					return err
				}
			}
			wg.Add(1)
			go func(filename string, num int) {
				uploadChannel <- true
				defer func() {
					wg.Done()
					os.Remove(filename)
					<-tmpFileWriteChannel
					<-uploadChannel
				}()
				logger.Printf("Uploading a part (File: %v, UploadNumber: %d) ...", filename, num)
				f, err := os.Open(filename)
				if err != nil {
					logger.Printf(err.Error())
					ok = false
					return
				}
				defer f.Close()
				part, err := cli.UploadPart(upload, num, f)
				if err != nil {
					logger.Printf(err.Error())
				}
				parts[num-1] = part
				logger.Printf("Finished to write the part file: %v", filename)
			}(out.Name(), num)
			out.Close()
			out = nil
			size = 0
			num++
		}
	}
	if out != nil {
		filename := out.Name()
		defer os.Remove(filename)
		if upload == nil {
			f, err := os.Open(filename)
			if err != nil {
				return err
			}
			return cli.PutObject(bucket, key, f, metadata)
		}
		part, err := cli.UploadPart(upload, num, out)
		if err != nil {
			return err
		}
		parts[num-1] = part
	}
	wg.Wait()
	if ok && upload != nil {
		if _, err = cli.CompleteMultipartUpload(upload, parts[0:num]); err != nil {
			logger.Printf(err.Error())
			ok = false
		} else {
			logger.Printf("Succeeded to upload an object %s:%s", bucket, key)
		}
	}
	return
}

func (cli *DefaultStorageClient) ResumeUploadFile(bucket, key, uploadId string, fd *os.File, metadata *ObjectMetadata) (err error) {
	if fd == nil {
		return errors.New("no such file")
	}
	logger := cli.env.Logger
	fi, _ := fd.Stat()
	size := fi.Size()
	chunkSize := cli.Config.MultipartChunkSize
	if size <= cli.Config.MultipartChunkSize {
		return cli.PutObject(bucket, key, fd, metadata)
	}
	logger.Printf("Uploading %s -> %s:%s ...", fd.Name(), bucket, key)
	var (
		num   = int(math.Ceil(float64(size) / float64(chunkSize)))
		parts = make([]*Part, int(num))
		wg    sync.WaitGroup
		ok    = true
	)
	upload := &MultipartUpload{
		Bucket:   bucket,
		Key:      key,
		UploadID: uploadId,
	}
	listing, err := cli.ListParts(bucket, key, uploadId, 0, 1000)
	defer func() {
		if !ok {
			if upload != nil && cli.Config.AbortOnFailure {
				cli.AbortMultipartUpload(upload)
			}
			if err == nil {
				err = errors.New("failed to upload file(s)")
			}
		}
	}()
	if err != nil {
		return
	}
	ch := make(chan bool)
	for i := 1; i <= num; i++ {
		wg.Add(1)
		go func(filename string, num int) {
			<-ch
			defer func() {
				wg.Done()
				ch <- true
			}()
			partD := listing.GetPart(num)
			if partD != nil {
				parts[num-1] = &partD.Part
				return
			}
			logger.Printf("Uploading a part (UploadNumber: %d) ...", num)
			f, err := os.Open(filename)
			if err != nil {
				logger.Println(err.Error())
				ok = false
				return
			}
			defer f.Close()
			off := chunkSize * int64(num-1)
			n := chunkSize
			if off+n > size {
				n = size - off
			}
			logger.Printf("File: %s Offset: %d, Size: %d", filename, off, n)
			part, err := cli.UploadPartAt(upload, num, f, off, n)
			if part == nil || err != nil {
				ok = false
				return
			}
			parts[num-1] = part
			logger.Printf("Finished to upload Part(%s).", part)
		}(fd.Name(), i)
	}
	concurrency := cli.env.Concurrency
	if concurrency > num {
		concurrency = num
	}
	for i := 0; i < concurrency; i++ {
		ch <- true
	}
	wg.Wait()
	if ok {
		if _, err = cli.CompleteMultipartUpload(upload, parts); err != nil {
			ok = false
		} else {
			logger.Printf("Succeeded to upload %s as %s:%s", fd.Name(), bucket, key)
		}
	}
	return
}

// UploadFile uploads a file as storage object
func (cli *DefaultStorageClient) UploadFile(bucket, key string, fd *os.File, metadata *ObjectMetadata) (err error) {
	if fd == nil {
		return errors.New("no such file")
	}
	logger := cli.env.Logger
	fi, _ := fd.Stat()
	size := fi.Size()
	multipartChunkSize := cli.Config.MultipartChunkSize
	if size <= cli.Config.MultipartChunkSize {
		return cli.PutObject(bucket, key, fd, metadata)
	}
	logger.Printf("Uploading %s -> %s:%s ...", fd.Name(), bucket, key)
	var (
		num    = int(math.Ceil(float64(size) / float64(multipartChunkSize)))
		parts  = make([]*Part, int(num))
		wg     sync.WaitGroup
		upload *MultipartUpload
		ok     = true
	)
	defer func() {
		if !ok {
			if upload != nil && cli.Config.AbortOnFailure {
				cli.AbortMultipartUpload(upload)
			}
			if err == nil {
				err = errors.New("failed to upload file(s)")
			}
		}
	}()
	upload, err = cli.InitiateMultipartUpload(bucket, key, metadata)
	if err != nil {
		return
	}
	ch := make(chan bool)
	for i := 1; i <= num; i++ {
		wg.Add(1)
		go func(filename string, num int) {
			<-ch
			defer func() {
				wg.Done()
				ch <- true
			}()
			logger.Printf("Uploading a part (UploadNumber: %d) ...", num)
			f, err := os.Open(filename)
			if err != nil {
				logger.Println(err.Error())
				ok = false
				return
			}
			defer f.Close()
			off := multipartChunkSize * int64(num-1)
			n := multipartChunkSize
			if off+n > size {
				n = size - off
			}
			logger.Printf("File: %s Offset: %d, Size: %d", filename, off, n)
			part, err := cli.UploadPartAt(upload, num, f, off, n)
			if part == nil || err != nil {
				ok = false
				return
			}
			parts[num-1] = part
			logger.Printf("Finished to upload Part(%s).", part)
		}(fd.Name(), i)
	}
	concurrency := cli.env.Concurrency
	if concurrency > num {
		concurrency = num
	}
	for i := 0; i < concurrency; i++ {
		ch <- true
	}
	wg.Wait()
	if ok {
		if _, err = cli.CompleteMultipartUpload(upload, parts); err != nil {
			ok = false
		} else {
			logger.Printf("Succeeded to upload %s as %s:%s", fd.Name(), bucket, key)
		}
	}
	return
}

// GetStorageSpace returns a usage of the DAG storage.
func (cli *DefaultStorageClient) GetStorageSpace(region string) (usage *StorageSpace, err error) {
	if cli.env.Debug {
		cli.env.Logger.Println("Storage REST API Call: GET Service space")
	}
	locations, err := cli.GetRegions()
	if err != nil {
		cli.Logger.Printf("Failed to execute Get Regions. reason: %v\n", err)
		return nil, err
	}
	defaultEndpoint := cli.Config.Endpoint
	for _, r := range locations.Regions {
		if r.Name == region {
			cli.Config.Endpoint = r.Endpoint
		}
	}
	urlStr := cli.Config.buildURL("", "", map[string]string{"space": ""})
	_, err = cli.DoAndRetry(func() (*http.Request, error) {
		req, _ := http.NewRequest("GET", urlStr, nil)
		return req, nil
	}, &usage)
	if err != nil {
		cli.Logger.Println("Failed to execute HTTP request.", err)
		return
	}
	cli.Config.Endpoint = defaultEndpoint
	return
}

// ListNetworkTraffics returns list of network traffic every 1 day.
func (cli *DefaultStorageClient) ListNetworkTraffics(backwardTo int, region string) (result *ListTrafficResult, err error) {
	if cli.env.Debug {
		cli.env.Logger.Printf("Storage REST API Call: GET Service traffic (backwardTo: %v)", backwardTo)
	}
	locations, err := cli.GetRegions()
	if err != nil {
		cli.Logger.Printf("Failed to execute Get Regions. reason: %v\n", err)
		return nil, err
	}
	defaultEndpoint := cli.Config.Endpoint
	for _, r := range locations.Regions {
		if r.Name == region {
			cli.Config.Endpoint = r.Endpoint
		}
	}
	urlStr := cli.Config.buildURL("", "", map[string]string{"traffic": "", "backwardTo": strconv.Itoa(backwardTo)})
	_, err = cli.DoAndRetry(func() (*http.Request, error) {
		req, err := http.NewRequest("GET", urlStr, nil)
		if err != nil {
			cli.Logger.Printf("Failed to create a new HTTP request for ListParts. reason: %v\n", err)
			return nil, err
		}
		return req, nil
	}, &result)
	if err != nil {
		cli.Logger.Println("Failed to execute HTTP request.", err)
		return
	}
	cli.Config.Endpoint = defaultEndpoint
	return
}

// GetNetworkTraffic returns a network traffic of specified date.
func (cli *DefaultStorageClient) GetNetworkTraffic(date string, region string) (traffic *DownTraffic, err error) {
	var traffics *ListTrafficResult
	if cli.env.Debug {
		cli.env.Logger.Printf("Storage REST API Call: GET Service traffic (chargeDate: %s)", date)
	}
	locations, err := cli.GetRegions()
	if err != nil {
		cli.Logger.Printf("Failed to execute Get Regions. reason: %v\n", err)
		return nil, err
	}
	defaultEndpoint := cli.Config.Endpoint
	for _, r := range locations.Regions {
		if r.Name == region {
			cli.Config.Endpoint = r.Endpoint
		}
	}
	urlStr := cli.Config.buildURL("", "", map[string]string{"traffic": "", "chargeDate": date})
	_, err = cli.DoAndRetry(func() (*http.Request, error) {
		req, err := http.NewRequest("GET", urlStr, nil)
		if err != nil {
			cli.Logger.Printf("Failed to create a new HTTP request for ListParts. reason: %v\n", err)
			return nil, err
		}
		return req, nil
	}, &traffics)
	if err != nil {
		cli.Logger.Println("Failed to execute HTTP request.", err)
		return
	}
	cli.Config.Endpoint = defaultEndpoint
	if traffics != nil && len(traffics.DownTraffics) > 0 {
		return traffics.DownTraffics[0], nil
	}
	return nil, fmt.Errorf("not found")
}

// Sign calculates a signature string and set to the Authorization header.
func (cli *DefaultStorageClient) Sign(req *http.Request) error {
	var (
		accessKeyID     = cli.Config.AccessKeyID
		secretAccessKey = cli.Config.SecretAccessKey
	)
	if accessKeyID == "" || secretAccessKey == "" {
		return errors.New("please check your access_key_id and secret_access_key, and try again")
	}
	now := time.Now()
	date := req.Header.Get("Date")
	if date == "" {
		date = now.In(time.UTC).Format(time.RFC1123)
		req.Header.Set("Date", date)
	}
	contentType := req.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
		req.Header.Set("Content-Type", contentType)
	}
	canonHeaders := getCanonicalHeaders(req.Header)
	canonResource := getCanonicalResource(req.URL)
	headers := req.Header
	stringToSign := fmt.Sprintf("%s\n%s\n%s\n%s\n%s%s",
		req.Method,
		headers.Get("Content-MD5"),
		contentType,
		date,
		canonHeaders,
		canonResource)
	if cli.env.Debug {
		cli.Logger.Printf("StringToSign = %q", stringToSign)
	}
	// calculate the signature
	mac := hmac.New(sha1.New, []byte(secretAccessKey))
	mac.Write([]byte(stringToSign))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	if cli.env.Debug {
		cli.Logger.Printf("Signature = %q", signature)
	}
	// "Authorization" header string
	authorization := fmt.Sprintf("%s %s:%s", cli.Config.Vendor, accessKeyID, signature)
	req.Header.Set("Authorization", authorization)
	return nil
}

// DoAndRetry executes Do method and if the response is error (< 400), retries http requests.
func (cli *DefaultStorageClient) DoAndRetry(fn func() (*http.Request, error), result interface{}) (resp *http.Response, err error) {
	req, err := fn()
	if err != nil {
		return
	}
	resp, err = cli.Do(req, result)
	for retryCount := 1; retryCount <= cli.Config.Retry; retryCount++ {
		if resp != nil && resp.StatusCode < 400 {
			return resp, nil
		}
		if resp != nil && resp.StatusCode == 404 {
			// NotFound
			break
		}
		time.Sleep(cli.Config.RetryInterval)
		req, err := fn()
		if err != nil {
			continue
		}
		resp, err = cli.Do(req, result)
		if err != nil {
			cli.Logger.Printf("Failed to request. %v (retry: %d/%d)", err, retryCount, cli.Config.Retry)
		}
	}
	return
}

// Do sends an HTTP request and returns an HTTP response
func (cli *DefaultStorageClient) Do(req *http.Request, result interface{}) (resp *http.Response, err error) {
	httpcli := cli.HTTPClient(cli)
	if cli.Config.AccessKeyID != "" {
		cli.Sign(req)
	}
	req.Header.Set("User-Agent", fmt.Sprintf("dagtools/%s", cli.env.Version))
	if cli.env.Debug {
		cli.Logger.Println("Sending HTTP request ...")
	}
	cli.Logger.Printf(">> %s %s %s", req.Method, req.URL, req.Proto)
	if cli.env.Debug {
		for key, values := range req.Header {
			_key := key
			if strings.HasPrefix(_key, "X-") {
				_key = strings.ToLower(_key)
			}
			cli.Logger.Printf(">> %s: %s", _key, strings.Join(values, ","))
		}
	}
	resp, err = httpcli.Do(req)
	if err != nil || resp == nil {
		if err == nil {
			err = errors.New("unknown error")
		}
		return
	}
	if cli.env.Debug {
		cli.Logger.Println("Received HTTP response.")
	}
	cli.Logger.Printf("<< %s %s", resp.Proto, resp.Status)
	if cli.env.Debug {
		for key, values := range resp.Header {
			_key := key
			if strings.HasPrefix(_key, "X-") {
				_key = strings.ToLower(_key)
			}
			cli.Logger.Printf("<< %s: %s", _key, strings.Join(values, ","))
		}
	}
	if resp.StatusCode >= 300 {
		requestId := resp.Header.Get("x-iijgio-request-id")
		var e ErrorResponse
		if resp.Header.Get("Content-Type") == "application/xml" {
			unmarshal(resp.Body, &e)
		} else {
			e.RequestID = requestId
			e.Code = strings.Join(strings.Split(resp.Status, " ")[1:], " ")
		}
		e.ErrorCode = resp.StatusCode
		return resp, e
	}
	if resp.ContentLength < 0 && cli.env.Debug {
		cli.Logger.Printf("Illegal Content-Length detect. value = %q", resp.ContentLength)
	}
	if result != nil {
		err = unmarshal(resp.Body, result)
	}
	return resp, err
}

// Get the canonical resource string from url.URL
func getCanonicalResource(u *url.URL) string {
	var (
		cr = encodeURL(u.Path)
		qs []string
	)
	for k, v := range u.Query() {
		if subResources[k] {
			q := k
			if len(v) > 0 {
				_v := strings.Join(v, ",")
				if _v != "" {
					q += "=" + _v
				}
			}
			qs = append(qs, q)
		}
	}
	if len(qs) > 0 {
		sort.Strings(qs)
		cr += "?" + strings.Join(qs, "&")
	}
	return cr
}

// Get the canonical headers string from http.Header
func getCanonicalHeaders(h http.Header) string {
	var canonicalHeaders []string
	for key, values := range h {
		_key := strings.TrimSpace(strings.ToLower(key))
		if !(strings.HasPrefix(_key, "x-iijgio-") || strings.HasPrefix(_key, "x-amz-")) {
			continue
		}
		var _values []string
		for i := range values {
			_values = append(_values, strings.TrimSpace(values[i]))
		}
		canonicalHeaders = append(canonicalHeaders, fmt.Sprintf("%s:%s\n", _key, strings.Join(_values, ",")))
	}
	sort.Sort(sort.Reverse(sort.StringSlice(canonicalHeaders)))
	return strings.Join(canonicalHeaders, "")
}

// Unmarshal XML file into Go struct.
func unmarshal(r io.ReadCloser, o interface{}) (err error) {
	if r != nil && o != nil {
		defer r.Close()
		br := bufio.NewReader(r)
		dec := xml.NewDecoder(br)
		err = dec.Decode(o)
	}
	return
}

// Build a resource URL
func (config *StorageClientConfig) buildURL(bucket, key string, qs map[string]string) string {
	var (
		scheme = "http"
		urlStr = ""
	)
	if config.Secure {
		scheme = "https"
	}
	urlStr = fmt.Sprintf("%s://%s/", scheme, config.Endpoint)
	if bucket != "" {
		urlStr += bucket
	}
	if key != "" {
		urlStr += "/" + encodeURL(key)
	}
	if qs != nil {
		var _qs []string
		for k, v := range qs {
			q := k
			if !nonValueSubResources[k] {
				q += "=" + encodeURL(v)
			}
			_qs = append(_qs, q)
		}
		if len(_qs) > 0 {
			sort.Strings(_qs)
			urlStr += "?" + strings.Join(_qs, "&")
		}
	}
	return urlStr
}

func encodeURL(urlStr string) string {
	u := &url.URL{Path: urlStr}
	return u.String()
}
