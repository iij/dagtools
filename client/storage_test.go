package client

import (
	"errors"
	"net/http"
	"os"
	"testing"

	"io"
	"io/ioutil"
	"strings"
	"time"

	"github.com/iij/dagtools/env"
	"github.com/iij/dagtools/ini"
	"github.com/golang/mock/gomock"
)

func newMockEnvironment() env.Environment {
	config := ini.Config{Filename: "dummy.ini", Sections: make(map[string]ini.Section)}
	config.Set("storage", "accessKeyId", "SAMPLE00000000000000")
	config.Set("storage", "secretAccessKey", "Sample0000000000000000000000000000000000")
	config.Set("storage", "retry", "0")
	config.Set("storage", "retryInterval", "500")
	e := env.Environment{Config: &config}
	e.Init()
	return e
}

func newMock() (client *DefaultStorageClient) {
	e := newMockEnvironment()
	_client, _ := NewStorageClient(&e)
	client = _client.(*DefaultStorageClient)
	return
}

func newHTTPClientMock(t *testing.T) (client *DefaultStorageClient, mock *MockHTTPClient) {
	client = newMock()
	mockcli := &mockHTTPCli{}
	client.HTTPClient = mockcli.newMockHTTPClient
	ctrl := gomock.NewController(t)
	mock = NewMockHTTPClient(ctrl)
	mockcli.mock = mock
	return
}

func newAnonymousMockEnvironment() env.Environment {
	config := ini.Config{Filename: "dummy.ini", Sections: make(map[string]ini.Section)}
	config.Set("storage", "accessKeyId", "")
	config.Set("storage", "secretAccessKey", "")
	e := env.Environment{Config: &config}
	e.Init()
	return e
}

type mockHTTPCli struct {
	mock *MockHTTPClient
}

func (mock *mockHTTPCli) newMockHTTPClient(cli *DefaultStorageClient) HTTPClient {
	return mock.mock
}

func assertEquals(t *testing.T, msg string, actual, expected interface{}) {
	if actual != expected {
		t.Errorf("%v (%q != %q)", msg, actual, expected)
	}
}

func TestBuildSimpleURL(t *testing.T) {
	config := StorageClientConfig{
		Endpoint:        "storage-dag.iijgio.com",
		AccessKeyID:     "",
		SecretAccessKey: "",
		Secure:          true,
	}
	url := config.buildURL("mybucket", "example", nil)
	assertEquals(t, "Failed to build an object url.", url, "https://storage-dag.iijgio.com/mybucket/example")
}

func TestBuildURLWithSpecialCharacter(t *testing.T) {
	config := StorageClientConfig{
		Endpoint:        "storage-dag.iijgio.com",
		AccessKeyID:     "",
		SecretAccessKey: "",
		Secure:          true,
	}
	url := config.buildURL("mybucket", "example#foo?bar", nil)
	assertEquals(t, "Failed to build an object url.", url, "https://storage-dag.iijgio.com/mybucket/example%23foo%3Fbar")
}

func TestBuildQueryContainedURL(t *testing.T) {
	config := StorageClientConfig{
		Endpoint:        "storage-dag.iijgio.com",
		AccessKeyID:     "",
		SecretAccessKey: "",
		Secure:          true,
	}
	url := config.buildURL("mybucket", "example", map[string]string{"b": "foo", "a": "bar"})
	assertEquals(t, "Failed to build an object url contained query parameters.", url, "https://storage-dag.iijgio.com/mybucket/example?a=bar&b=foo")
}

func TestBuildNoValueQueryContainedURL(t *testing.T) {
	config := StorageClientConfig{
		Endpoint:        "storage-dag.iijgio.com",
		AccessKeyID:     "",
		SecretAccessKey: "",
		Secure:          true,
	}
	url := config.buildURL("mybucket", "example", map[string]string{"foo": "bar", "policy": ""})
	assertEquals(t, "Failed to build an object url contained a query parameter that has no value.", url, "https://storage-dag.iijgio.com/mybucket/example?foo=bar&policy")
}

func TestNewListObjectsRequest(t *testing.T) {
	client := newMock()
	req, err := client.NewListObjectsRequest("mybucket", "foo/", "foo/bar", "/", 1000)
	assertEquals(t, "Failed to create a new request.", err, nil)
	assertEquals(t, "The created url is invalid. see if contains all parameters or orders parameters by key.", req.URL.String(), "https://storage-dag.iijgio.com/mybucket?delimiter=/&marker=foo/bar&max-keys=1000&prefix=foo/")
}

func TestNewListObjectsRequest2(t *testing.T) {
	client := newMock()
	req, err := client.NewListObjectsRequest("mybucket", "foo#bar/", "foo", "/", 1000)
	assertEquals(t, "Failed to create a new request.", err, nil)
	assertEquals(t, "The created url is invalid. see if contains all parameters or orders parameters by key.", req.URL.String(), "https://storage-dag.iijgio.com/mybucket?delimiter=/&marker=foo&max-keys=1000&prefix=foo%23bar/")
}

func TestSign(t *testing.T) {
	client := newMock()
	url := client.Config.buildURL("mybucket", "example", map[string]string{"acl": ""})
	assertEquals(t, "The object acl's url is invalid.", url, "https://storage-dag.iijgio.com/mybucket/example?acl")
	req, _ := http.NewRequest("GET", url, nil)
	date := "Mon, 15 Jun 2015 12:18:51 GMT"
	req.Header.Set("Date", date)
	client.Sign(req)
	assertEquals(t, "Must not re-set a Date header.", req.Header.Get("Date"), date)
	assertEquals(t, "Should set a default content type when request did not set a Content-Type header.", req.Header.Get("Content-Type"), "application/octet-stream")
	assertEquals(t, "A calculated signature was unmatched. please recheck the string-to-sign", req.Header.Get("Authorization"), "IIJGIO SAMPLE00000000000000:tPscjiXKgO45q1/qc2ryPNsLlT0=")
}

func TestNoSign(t *testing.T) {
	e := newMockEnvironment()
	e.Config.Set("storage", "accessKeyId", "")
	_client, _ := NewStorageClient(&e)
	client := _client.(*DefaultStorageClient)
	url := client.Config.buildURL("mybucket", "example", map[string]string{"acl": ""})
	assertEquals(t, "The object acl's url is invalid", url, "https://storage-dag.iijgio.com/mybucket/example?acl")
	req, _ := http.NewRequest("GET", url, nil)
	client.Sign(req)
	assertEquals(t, "Should not set a Date header when requested by anonymous.", req.Header.Get("Date"), "")
	assertEquals(t, "Should not set a Content-Type header when requested by anonymous.", req.Header.Get("Content-Type"), "")
	assertEquals(t, "Should not set an Authorization header when requested by anonymous.", req.Header.Get("Authorization"), "")
}

func TestSignWithCanonicalHeaders(t *testing.T) {
	client := newMock()
	url := client.Config.buildURL("mybucket", "example", nil)
	assertEquals(t, "The object acl's url is invalid.", url, "https://storage-dag.iijgio.com/mybucket/example")
	req, _ := http.NewRequest("PUT", url, nil)
	req.Header.Add("x-iijgio-meta-a2", "a2")
	req.Header.Add("x-iijgio-meta-a1", "a1")
	req.Header.Add("x-iijgio-meta-b", "b")
	date := "Mon, 15 Jun 2015 12:18:51 GMT"
	req.Header.Set("Date", date)
	client.Sign(req)
	assertEquals(t, "Must not re-set a Date header.", req.Header.Get("Date"), date)
	assertEquals(t, "Should set a default content type when request did not set a Content-Type header.", req.Header.Get("Content-Type"), "application/octet-stream")
	assertEquals(t, "A calculated signature was unmatched. please recheck the string-to-sign", req.Header.Get("Authorization"), "IIJGIO SAMPLE00000000000000:4B+FN8T+r5zm2K7H2VwqbfTwzp0=")
}

func TestGetUndecodedCanonicalResource(t *testing.T) {
	var req *http.Request
	url := "https://storage-dag.iijgio.com/mybucket/日本語"
	req, _ = http.NewRequest("PUT", url, nil)
	canonResource := getCanonicalResource(req.URL)
	assertEquals(t, "The CanonicalResource is invalid.", canonResource, "/mybucket/%E6%97%A5%E6%9C%AC%E8%AA%9E")
}

func TestHTTPCliDo(t *testing.T) {
	client, mock := newHTTPClientMock(t)
	mockresp := &http.Response{
		Body:       NewEmptyBody(),
		StatusCode: 200,
	}
	url := "https://storage-dag.iijgio.com/mybucket/example"
	req, _ := http.NewRequest("PUT", url, nil)
	mock.EXPECT().Do(gomock.Any()).Return(mockresp, nil)

	resp, _ := client.Do(req, nil)
	assertEquals(t, "Should return StatusCode 200 series at normal end.", resp.StatusCode, 200)
}

func TestHTTPCliDoErr(t *testing.T) {
	client, mock := newHTTPClientMock(t)
	url := "https://storage-dag.iijgio.com/mybucket/example"
	req, _ := http.NewRequest("GET", url, nil)
	mock.EXPECT().Do(gomock.Any()).Return(nil, errors.New("dummy"))

	_, err := client.Do(req, nil)
	assertEquals(t, "Pass through HTTP error check.", err.Error(), "dummy")
}

func TestHTTPClientDoErrCode(t *testing.T) {
	client, mock := newHTTPClientMock(t)
	mockresp := &http.Response{Body: NewEmptyBody()}
	mockresp.StatusCode = 300
	url := "https://storage-dag.iijgio.com/mybucket/example"
	req, _ := http.NewRequest("PUT", url, nil)
	mock.EXPECT().Do(gomock.Any()).Return(mockresp, nil)

	resp, _ := client.Do(req, nil)
	assertEquals(t, "Pass through HTTP StatusCode check.", resp.StatusCode, 300)
}

func TestHTTPClientDoErrCodeWithXML(t *testing.T) {
	client, mock := newHTTPClientMock(t)
	mockresp := &http.Response{Body: NewEmptyBody()}
	mockresp.StatusCode = 300
	mockresp.Header = map[string][]string{"Content-Type": {"application/xml"}}
	url := "https://storage-dag.iijgio.com/mybucket/example"
	req, _ := http.NewRequest("PUT", url, nil)
	mock.EXPECT().Do(gomock.Any()).Return(mockresp, nil)

	resp, _ := client.Do(req, nil)
	assertEquals(t, "Pass through HTTP StatusCode check.", resp.StatusCode, 300)
	assertEquals(t, "Pass through HTTP response xml check.", mockresp.Header.Get("Content-Type"), "application/xml")
}

func TestListBucketsApi(t *testing.T) {
	client, mock := newHTTPClientMock(t)
	mockresp := &http.Response{
		Body: NewBodyWithString(`<?xml version="1.0" encoding="UTF-8"?>
<ListAllMyBucketsResult>
  <Owner>
    <ID>bcaf1ffd86f461ca5fb16fd081034f</ID>
    <DisplayName>webfile</DisplayName>
  </Owner>
  <Buckets>
    <Bucket>
      <Name>quotes</Name>
      <CreationDate>2006-02-03T16:45:09.000Z</CreationDate>
    </Bucket>
    <Bucket>
      <Name>samples</Name>
      <CreationDate>2006-02-03T16:41:58.000Z</CreationDate>
    </Bucket>
  </Buckets>
</ListAllMyBucketsResult>`),
	}
	mock.EXPECT().Do(gomock.Any()).Return(mockresp, nil)

	listing, err := client.ListBuckets()
	assertEquals(t, "Should return nil at normal end.", err, nil)
	buckets := listing.Buckets
	assertEquals(t, "Should return 2 buckets.", len(buckets), 2)
	bucket1 := buckets[0]
	assertEquals(t, "Should return name of first bucket.", bucket1.Name, "quotes")
	assertEquals(t, "Should return creationDate of first bucket", bucket1.CreationDate, time.Date(2006, time.February, 3, 16, 45, 9, 0, time.UTC))

	bucket2 := buckets[1]
	assertEquals(t, "Should return name of second bucket.", bucket2.Name, "samples")
	assertEquals(t, "Should return creationDate of second bucket", bucket2.CreationDate, time.Date(2006, time.February, 3, 16, 41, 58, 0, time.UTC))
}

func TestPutBucketApi(t *testing.T) {
	client, mock := newHTTPClientMock(t)
	mockresp := &http.Response{
		Body: NewEmptyBody(),
	}
	mockresp.StatusCode = 200
	mock.EXPECT().Do(gomock.Any()).Return(mockresp, nil)

	err := client.PutBucket("mybucket")
	assertEquals(t, "Should return nil at normal end.", err, nil)
}

func TestDeleteBucketApi(t *testing.T) {
	client, mock := newHTTPClientMock(t)
	mockresp := &http.Response{
		Body: NewEmptyBody(),
	}
	mock.EXPECT().Do(gomock.Any()).Return(mockresp, nil)

	err := client.DeleteBucket("mybucket")
	assertEquals(t, "Should return nil at normal end.", err, nil)
}

func TestDoesBucketExistApi(t *testing.T) {
	client, mock := newHTTPClientMock(t)
	mockresp := &http.Response{
		Body:       NewEmptyBody(),
		StatusCode: 200,
	}
	mock.EXPECT().Do(gomock.Any()).Return(mockresp, nil)

	resp, err := client.DoesBucketExist("mybucket")
	assertEquals(t, "Should return nil at normal end.", err, nil)
	assertEquals(t, "Should return true response at normal end.", resp, true)
}

func TestGetBucketPolicyApi(t *testing.T) {
	client, mock := newHTTPClientMock(t)
	mockresp := &http.Response{
		Body:       NewBodyWithString("dummy"),
		StatusCode: 200,
	}
	mock.EXPECT().Do(gomock.Any()).Return(mockresp, nil)

	policy, _ := client.GetBucketPolicy("mybucket")
	raw, _ := ioutil.ReadAll(policy)
	body := string(raw)
	assertEquals(t, "Policy is not include in Response Body.", body, "dummy")
}

func TestPutBucketPolicyApi(t *testing.T) {
	client, mock := newHTTPClientMock(t)
	mockresp := &http.Response{
		Body:       NewEmptyBody(),
		StatusCode: 204,
	}
	mock.EXPECT().Do(gomock.Any()).Return(mockresp, nil)

	err := client.PutBucketPolicy("mybucket", nil)
	assertEquals(t, "Should return nil at normal end.", err, nil)
}

func TestDeleteBucketPolicyApi(t *testing.T) {
	client, mock := newHTTPClientMock(t)
	mockresp := &http.Response{
		Body: NewEmptyBody(),
	}
	mockresp.StatusCode = 204
	mock.EXPECT().Do(gomock.Any()).Return(mockresp, nil)

	err := client.DeleteBucketPolicy("mybucket")
	assertEquals(t, "Should return nil at normal end.", err, nil)
}

func TestListObjectsApi(t *testing.T) {
	client, mock := newHTTPClientMock(t)
	mockresp := &http.Response{
		Body: NewBodyWithString(`<?xml version="1.0" encoding="UTF-8"?>
<ListBucketResult>
  <Name>bucket</Name>
  <Prefix/>
  <Marker/>
  <MaxKeys>1000</MaxKeys>
  <IsTruncated>false</IsTruncated>
  <Contents>
    <Key>my-image.jpg</Key>
    <LastModified>2009-10-12T17:50:30.000Z</LastModified>
    <ETag>&quot;fba9dede5f27731c9771645a39863328&quot;</ETag>
    <Size>434234</Size>
    <StorageClass>STANDARD</StorageClass>
    <Owner>
      <ID>8a6925ce4a7f21c32aa379004fef</ID>
      <DisplayName>mtd@dag.iijgio.com</DisplayName>
    </Owner>
  </Contents>
  <Contents>
    <Key>my-third-image.jpg</Key>
    <LastModified>2009-10-12T17:50:30.000Z</LastModified>
    <ETag>&quot;1b2cf535f27731c974343645a3985328&quot;</ETag>
    <Size>64994</Size>
    <StorageClass>STANDARD</StorageClass>
    <Owner>
      <ID>8a69b1ddee97f21c32aa379004fef</ID>
      <DisplayName>mtd@dag.iijgio.com</DisplayName>
    </Owner>
  </Contents>
</ListBucketResult>`),
	}
	mock.EXPECT().Do(gomock.Any()).Return(mockresp, nil)

	_, err := client.ListObjects("mybucket", "", "", "", 1)
	assertEquals(t, "Should return nil at normal end.", err, nil)
}

func TestPutObjectAtApi(t *testing.T) {
	client, mock := newHTTPClientMock(t)
	mockresp := &http.Response{
		Body: NewEmptyBody(),
	}
	mockresp.StatusCode = 200
	mock.EXPECT().Do(gomock.Any()).Return(mockresp, nil)

	f, openerr := os.OpenFile("test_file/test.txt", 0, 0644)
	assertEquals(t, "Can not Open test File.", openerr, nil)
	stat, _ := f.Stat()
	size := stat.Size()

	err := client.PutObjectAt("mybucket", "", f, 0, size, nil)
	assertEquals(t, "Should return nil at normal end.", err, nil)
}

func TestGetObjectApi(t *testing.T) {
	client, mock := newHTTPClientMock(t)
	mockresp := &http.Response{
		Body:       NewBodyWithString("dummy"),
		StatusCode: 200,
	}
	mock.EXPECT().Do(gomock.Any()).Return(mockresp, nil)

	resp, err := client.GetObject("mybucket", "")
	assertEquals(t, "Should return nil at normal end.", err, nil)
	raw, _ := ioutil.ReadAll(resp)
	body := string(raw)
	assertEquals(t, "Should return response body.", body, "dummy")
}

func TestDoesObjectExistApi(t *testing.T) {
	client, mock := newHTTPClientMock(t)
	mockresp := &http.Response{
		Body: NewEmptyBody(),
	}
	mockresp.StatusCode = 200
	mock.EXPECT().Do(gomock.Any()).Return(mockresp, nil)

	resp, err := client.DoesObjectExist("mybucket", "")
	assertEquals(t, "Should return nil at normal end.", err, nil)
	assertEquals(t, "Should return true response at normal end.", resp, true)
}

func TestGetObjectSummaryApi(t *testing.T) {
	client, mock := newHTTPClientMock(t)
	mockresp := &http.Response{
		Body: NewEmptyBody(),
	}
	mockresp.StatusCode = 200
	mockresp.ContentLength = 1
	mock.EXPECT().Do(gomock.Any()).Return(mockresp, nil)

	summary, err := client.GetObjectSummary("mybucket", "")
	assertEquals(t, "Should return nil at normal end.", err, nil)
	assertEquals(t, "ContentLength is not set in ObjectMetadata.", summary.Size, int64(1))
}

func TestGetObjectMetadataApi(t *testing.T) {
	client, mock := newHTTPClientMock(t)
	mockresp := &http.Response{
		Body: NewEmptyBody(),
	}
	mockresp.StatusCode = 200
	mockresp.ContentLength = 1
	mock.EXPECT().Do(gomock.Any()).Return(mockresp, nil)

	object, err := client.GetObjectMetadata("mybucket", "")
	assertEquals(t, "Should return nil at normal end.", err, nil)
	assertEquals(t, "BucketName is not set in Object.", object.Bucket, "mybucket")
	assertEquals(t, "ObjectKey is not set in Object.", object.Key, "")
	assertEquals(t, "ContentLength is not set in Object.", object.Size, int64(1))
	assertEquals(t, "ContentLength is not set in ObjectMetadata.", object.Metadata.ContentLength, int64(1))
}

func TestDeleteObjectApi(t *testing.T) {
	client, mock := newHTTPClientMock(t)
	mockresp := &http.Response{
		Body: NewEmptyBody(),
	}
	mock.EXPECT().Do(gomock.Any()).Return(mockresp, nil)

	err := client.DeleteObject("mybucket", "")
	assertEquals(t, "Should return nil at normal end.", err, nil)
}

func TestDeleteMultipleObjectsApi(t *testing.T) {
	client, mock := newHTTPClientMock(t)
	mockresp := &http.Response{
		Body: NewBodyWithString(`<?xml version="1.0" encoding="UTF-8"?>
<DeleteResult xmlns="http://acs.dag.iijgio.com/doc/2006-03-01/">
  <Deleted>
    <Key>sample1.txt</Key>
  </Deleted>
  <Error>
    <Key>sample2.txt</Key>
    <Code>AccessDenied</Code>
    <Message>Access Denied</Message>
  </Error>
</DeleteResult>`),
	}
	mock.EXPECT().Do(gomock.Any()).Return(mockresp, nil)

	keys := []string{"foo", "bar"}
	_, err := client.DeleteMultipleObjects("mybucket", keys, true)
	assertEquals(t, "Should return nil at normal end.", err, nil)
}

func TestInitiateMultipartUploadApi(t *testing.T) {
	client, mock := newHTTPClientMock(t)
	mockresp := &http.Response{
		Body: NewBodyWithString(`<?xml version="1.0" encoding="UTF-8"?>
<InitiateMultipartUploadResult xmlns="http://acs.dag.iijgio.com/doc/2006-03-01/">
  <Bucket>mybucket</Bucket>
  <Key>myobject</Key>
  <UploadId>VXBsb2FkIElEIGZvciA2aWWpbmcncyBteS1tb3ZpZS5tMnRzIHVwbG9hZA</UploadId>
</InitiateMultipartUploadResult>`),
	}
	mock.EXPECT().Do(gomock.Any()).Return(mockresp, nil)

	_, err := client.InitiateMultipartUpload("mybucket", "", nil)
	assertEquals(t, "Should return nil at normal end.", err, nil)
}

func TestAbortMultipartUploadApi(t *testing.T) {
	client, mock := newHTTPClientMock(t)
	mockresp := &http.Response{
		Body: NewEmptyBody(),
	}
	mockresp.StatusCode = 204
	mock.EXPECT().Do(gomock.Any()).Return(mockresp, nil)

	upload := &MultipartUpload{}
	err := client.AbortMultipartUpload(upload)
	assertEquals(t, "Should return nil at normal end.", err, nil)
}

func TestCompleteMultipartUploadApi(t *testing.T) {
	client, mock := newHTTPClientMock(t)
	mockresp := &http.Response{
		Body: NewBodyWithString(`<?xml version="1.0" encoding="UTF-8"?>
<CompleteMultipartUploadResult xmlns="http://acs.dag.iijgio.com/doc/2006-03-01/">
  <Location>http://mybucket.storage-dag.iijgio.com/myobject</Location>
  <Bucket>mybucket</Bucket>
  <Key>myobject</Key>
  <ETag>"3858f62230ac3c915f300c664312c11f-9"</ETag>
</CompleteMultipartUploadResult>`),
	}
	mock.EXPECT().Do(gomock.Any()).Return(mockresp, nil)

	upload := &MultipartUpload{}
	var parts []*Part
	_, err := client.CompleteMultipartUpload(upload, parts)
	assertEquals(t, "Should return nil at normal end.", err, nil)
}

func TestUploadPartAtApi(t *testing.T) {
	client, mock := newHTTPClientMock(t)
	mockresp := &http.Response{
		Body: NewEmptyBody(),
	}
	mock.EXPECT().Do(gomock.Any()).Return(mockresp, nil)

	f, err := os.OpenFile("test_file/test.txt", 0, 0644)
	assertEquals(t, "Can not Open test File.", err, nil)

	upload := &MultipartUpload{}
	part, err := client.UploadPartAt(upload, 1, f, 1, 1)
	assertEquals(t, "Should return nil at normal end.", err, nil)
	assertEquals(t, "Should return PartNumber 1 at normal end.", part.PartNumber, 1)
}

func TestListBucketsApiHTTPErr(t *testing.T) {
	client, mock := newHTTPClientMock(t)
	mock.EXPECT().Do(gomock.Any()).Return(nil, errors.New("dummy"))

	_, err := client.ListBuckets()
	assertEquals(t, "Pass through HTTP error check.", err.Error(), "dummy")
}

func TestPutBucketApiHTTPErr(t *testing.T) {
	client, mock := newHTTPClientMock(t)
	mock.EXPECT().Do(gomock.Any()).Return(nil, errors.New("dummy"))

	err := client.PutBucket("mybucket")
	assertEquals(t, "Pass through HTTP error check.", err.Error(), "dummy")
}

func TestDeleteBucketApiHTTPErr(t *testing.T) {
	client, mock := newHTTPClientMock(t)
	mock.EXPECT().Do(gomock.Any()).Return(nil, errors.New("dummy"))

	err := client.DeleteBucket("mybucket")
	assertEquals(t, "Pass through HTTP error check.", err.Error(), "dummy")
}

func TestDoesBucketExistApiHTTPErr(t *testing.T) {
	client, mock := newHTTPClientMock(t)
	mock.EXPECT().Do(gomock.Any()).Return(nil, errors.New("dummy"))

	resp, _ := client.DoesBucketExist("mybucket")
	assertEquals(t, "Pass through HTTP error check.", resp, false)
}

func TestGetBucketPolicyApiHTTPErr(t *testing.T) {
	client, mock := newHTTPClientMock(t)
	mock.EXPECT().Do(gomock.Any()).Return(nil, errors.New("dummy"))

	_, err := client.GetBucketPolicy("mybucket")
	assertEquals(t, "Pass through HTTP error check.", err.Error(), "dummy")
}

func TestPutBucketPolicyApiHTTPErr(t *testing.T) {
	client, mock := newHTTPClientMock(t)
	mock.EXPECT().Do(gomock.Any()).Return(nil, errors.New("dummy"))

	err := client.PutBucketPolicy("mybucket", nil)
	assertEquals(t, "Pass through HTTP error check..", err.Error(), "dummy")
}

func TestDeleteBucketPolicyApiHTTPErr(t *testing.T) {
	client, mock := newHTTPClientMock(t)
	mock.EXPECT().Do(gomock.Any()).Return(nil, errors.New("dummy"))

	err := client.DeleteBucketPolicy("mybucket")
	assertEquals(t, "Pass through HTTP error check.", err.Error(), "dummy")
}

func TestListObjectsApiHTTPErr(t *testing.T) {
	client, mock := newHTTPClientMock(t)
	mockresp := &http.Response{
		Body:          NewEmptyBody(),
		StatusCode:    500,
		ContentLength: 0,
	}
	mock.EXPECT().Do(gomock.Any()).Return(mockresp, errors.New("dummy"))

	_, err := client.ListObjects("mybucket", "", "", "", 1)
	assertEquals(t, "Pass through HTTP error check.", err.Error(), "dummy")
}

func TestPutObjectAtApiHTTPErr(t *testing.T) {
	client, mock := newHTTPClientMock(t)
	mockresp := &http.Response{
		Body:          NewEmptyBody(),
		StatusCode:    500,
		ContentLength: 0,
	}
	mock.EXPECT().Do(gomock.Any()).Return(mockresp, errors.New("dummy"))

	f, openerr := os.OpenFile("test_file/test.txt", 0, 0644)
	assertEquals(t, "Can not Open test File.", openerr, nil)
	stat, _ := f.Stat()
	size := stat.Size()

	err := client.PutObjectAt("mybucket", "", f, 0, size, nil)
	assertEquals(t, "Pass through HTTP error check.", err.Error(), "dummy")
}

func TestGetObjectApiHTTPErr(t *testing.T) {
	client, mock := newHTTPClientMock(t)
	mockresp := &http.Response{Body: NewEmptyBody()}
	mock.EXPECT().Do(gomock.Any()).Return(mockresp, errors.New("dummy"))

	_, err := client.GetObject("mybucket", "")
	assertEquals(t, "Pass through HTTP error check.", err.Error(), "dummy")
}

func TestDoesObjectExistApiHTTPErr(t *testing.T) {
	client, mock := newHTTPClientMock(t)
	mockresp := &http.Response{Body: NewEmptyBody()}
	mock.EXPECT().Do(gomock.Any()).Return(mockresp, errors.New("dummy"))

	resp, err := client.DoesObjectExist("mybucket", "")
	assertEquals(t, "Pass through HTTP error check.", err.Error(), "dummy")
	assertEquals(t, "Pass through HTTP error check.", resp, false)
}

func TestGetObjectSummaryApiHTTPErr(t *testing.T) {
	client, mock := newHTTPClientMock(t)
	mockresp := &http.Response{
		Body:          NewEmptyBody(),
		ContentLength: 0,
	}
	mock.EXPECT().Do(gomock.Any()).Return(mockresp, errors.New("dummy"))

	var foo *ObjectSummary
	summary, err := client.GetObjectSummary("mybucket", "")
	assertEquals(t, "Pass through HTTP error check.", err.Error(), "dummy")
	assertEquals(t, "Pass through HTTP error check.", summary, foo)
}

func TestGetObjectMetadataApiHTTPErr(t *testing.T) {
	client, mock := newHTTPClientMock(t)
	mockresp := &http.Response{Body: NewEmptyBody()}
	mock.EXPECT().Do(gomock.Any()).Return(mockresp, errors.New("dummy"))

	_, err := client.GetObjectMetadata("mybucket", "")
	assertEquals(t, "Pass through HTTP error check.", err.Error(), "dummy")
}

func TestGetObjectMetadataApiHTTPNotFoundErr(t *testing.T) {
	client, mock := newHTTPClientMock(t)
	mockresp := &http.Response{Body: NewEmptyBody()}
	mockresp.StatusCode = 404
	mock.EXPECT().Do(gomock.Any()).Return(mockresp, nil)

	var foo *Object
	object, err := client.GetObjectMetadata("mybucket", "")
	assertEquals(t, "Pass through HTTP StatusCode check.", err, nil)
	assertEquals(t, "Pass through HTTP StatusCode check.", object, foo)
}

func TestDeleteObjectApiHTTPErr(t *testing.T) {
	client, mock := newHTTPClientMock(t)
	mockresp := &http.Response{Body: NewEmptyBody()}
	mock.EXPECT().Do(gomock.Any()).Return(mockresp, errors.New("dummy"))

	err := client.DeleteObject("mybucket", "")
	assertEquals(t, "Pass through HTTP error check.", err.Error(), "dummy")
}

func TestDeleteMultipleObjectsApiHTTPErr(t *testing.T) {
	client, mock := newHTTPClientMock(t)
	mockresp := &http.Response{
		Body:          NewEmptyBody(),
		StatusCode:    500,
		ContentLength: 0,
	}
	mock.EXPECT().Do(gomock.Any()).Return(mockresp, errors.New("dummy"))

	keys := []string{"foo", "bar"}
	_, err := client.DeleteMultipleObjects("mybucket", keys, true)
	assertEquals(t, "Pass through HTTP error check.", err.Error(), "dummy")
}

func TestInitiateMultipartUploadApiHTTPErr(t *testing.T) {
	client, mock := newHTTPClientMock(t)
	mockresp := &http.Response{
		Body:          NewEmptyBody(),
		StatusCode:    500,
		ContentLength: 0,
	}
	mock.EXPECT().Do(gomock.Any()).Return(mockresp, errors.New("dummy"))

	_, err := client.InitiateMultipartUpload("mybucket", "", nil)
	assertEquals(t, "Pass through HTTP error check.", err.Error(), "dummy")
}

func TestAbortMultipartUploadApiHTTPErr(t *testing.T) {
	client, mock := newHTTPClientMock(t)
	mockresp := &http.Response{Body: NewEmptyBody()}
	mock.EXPECT().Do(gomock.Any()).Return(mockresp, errors.New("dummy"))

	upload := &MultipartUpload{}
	err := client.AbortMultipartUpload(upload)
	assertEquals(t, "Pass through HTTP error check.", err.Error(), "dummy")
}

func TestCompleteMultipartUploadApiHTTPErr(t *testing.T) {
	client, mock := newHTTPClientMock(t)
	mockresp := &http.Response{
		Body:       NewEmptyBody(),
		StatusCode: 500,
	}
	mock.EXPECT().Do(gomock.Any()).Return(mockresp, errors.New("dummy"))

	upload := &MultipartUpload{}
	var parts []*Part
	_, err := client.CompleteMultipartUpload(upload, parts)
	assertEquals(t, "Pass through HTTP error check.", err.Error(), "dummy")
}

func TestUploadPartApiHTTPErr(t *testing.T) {
	client, mock := newHTTPClientMock(t)
	mockresp := &http.Response{Body: NewEmptyBody()}
	mock.EXPECT().Do(gomock.Any()).Return(mockresp, errors.New("dummy"))

	f, err := os.OpenFile("test_file/test.txt", 0, 0644)
	assertEquals(t, "Can not Open test File.", err, nil)

	upload := &MultipartUpload{}
	_, err = client.UploadPart(upload, 1, f)
	assertEquals(t, "Pass through HTTP error check.", err.Error(), "dummy")
}

func TestUploadPartAtApiHTTPErr(t *testing.T) {
	client, mock := newHTTPClientMock(t)
	mockresp := &http.Response{Body: NewEmptyBody()}
	mock.EXPECT().Do(gomock.Any()).Return(mockresp, errors.New("dummy"))

	f, openerr := os.OpenFile("test_file/test.txt", 0, 0644)
	assertEquals(t, "Can not Open test File.", openerr, nil)

	upload := &MultipartUpload{}
	_, err := client.UploadPartAt(upload, 1, f, 1, 1)
	assertEquals(t, "Pass through HTTP error check.", err.Error(), "dummy")
}

func TestListBucketsApiErrWithoutID(t *testing.T) {
	e := newAnonymousMockEnvironment()
	_client, _ := NewStorageClient(&e)
	client := _client.(*DefaultStorageClient)

	_, err := client.ListBuckets()
	assertEquals(t, "Pass through AccessKeyID check.", err.Error(), "please check your access_key_id and secret_access_key and try again")
}

func TestPutBucketApiStatusCodeErr(t *testing.T) {
	client, mock := newHTTPClientMock(t)
	mockresp := &http.Response{Body: NewEmptyBody()}
	mockresp.StatusCode = 300
	mock.EXPECT().Do(gomock.Any()).Return(mockresp, nil)

	err := client.PutBucket("mybucket")
	expected := ErrorResponse{
		ErrorCode: 300,
	}
	assertEquals(t, "Pass through HTTP StatusCode check.", err, expected)
}

func TestDoesBucketExistApiNotFoundErr(t *testing.T) {
	client, mock := newHTTPClientMock(t)
	mockresp := &http.Response{
		Body:       NewEmptyBody(),
		StatusCode: 404,
	}
	mock.EXPECT().Do(gomock.Any()).Return(mockresp, nil)

	resp, err := client.DoesBucketExist("mybucket")
	assertEquals(t, "Pass through 404 error check.", resp, false)
	assertEquals(t, "Pass through HTTP error check.", err, nil)
}

func TestDoesBucketExistApiStatusCodeErr(t *testing.T) {
	client, mock := newHTTPClientMock(t)
	mockresp := &http.Response{Body: NewEmptyBody()}
	mockresp.StatusCode = 300
	mock.EXPECT().Do(gomock.Any()).Return(mockresp, nil)

	resp, _ := client.DoesBucketExist("mybucket")
	assertEquals(t, "Should return nil at normal end.", resp, false)
}

func TestGetBucketPolicyApiNotFoundErr(t *testing.T) {
	client, mock := newHTTPClientMock(t)
	mockresp := &http.Response{Body: NewEmptyBody()}
	mockresp.StatusCode = 404
	mock.EXPECT().Do(gomock.Any()).Return(mockresp, nil)

	resp, _ := client.GetBucketPolicy("mybucket")
	assertEquals(t, "Should return nil at no policy.", resp, nil)
}

func TestGetBucketPolicyApiStatusCodeErr(t *testing.T) {
	client, mock := newHTTPClientMock(t)
	mockresp := &http.Response{Body: NewEmptyBody()}
	mockresp.StatusCode = 299
	mock.EXPECT().Do(gomock.Any()).Return(mockresp, nil)

	resp, err := client.GetBucketPolicy("mybucket")
	assertEquals(t, "Pass through HTTP StatusCode check.", err.Error(), "invalid response")
	assertEquals(t, "Pass through HTTP StatusCode check.", resp, nil)
}

func TestPutBucketPolicyApiStatusCodeErr(t *testing.T) {
	client, mock := newHTTPClientMock(t)
	mockresp := &http.Response{Body: NewEmptyBody()}
	mockresp.StatusCode = 200
	mock.EXPECT().Do(gomock.Any()).Return(mockresp, nil)

	err := client.PutBucketPolicy("mybucket", nil)
	assertEquals(t, "Policy is not include in Response Body.", err.Error(), "invalid response")
}

func TestDeleteBucketPolicyApiStatusCodeErr(t *testing.T) {
	client, mock := newHTTPClientMock(t)
	mockresp := &http.Response{Body: NewEmptyBody()}
	mockresp.StatusCode = 299
	mock.EXPECT().Do(gomock.Any()).Return(mockresp, nil)

	err := client.DeleteBucketPolicy("mybucket")
	assertEquals(t, "Pass through HTTP StatusCode check.", err.Error(), "invalid response")
}

func TestListObjectsApiURLParseErr(t *testing.T) {
	client, mock := newHTTPClientMock(t)
	mockresp := &http.Response{Body: NewEmptyBody()}
	mock.EXPECT().Do(gomock.Any()).Return(mockresp, nil)
	_, err := client.ListObjects("%mybucket", "", "", "", 1)
	assertEquals(t, "Pass through URL Parse error check.", err.Error(), "parse https://storage-dag.iijgio.com/%mybucket?max-keys=1: invalid URL escape \"%my\"")
}

func TestPutObjectApiErr(t *testing.T) {
	client, mock := newHTTPClientMock(t)
	mockresp := &http.Response{Body: NewEmptyBody()}
	mock.EXPECT().Do(gomock.Any()).Return(mockresp, nil)

	err := client.PutObject("", "", nil, nil)
	assertEquals(t, "Pass through f.Stat error check.", err.Error(), "invalid argument")
}

func TestPutObjectAtApiStatusCodeErr(t *testing.T) {
	client, mock := newHTTPClientMock(t)
	mockresp := &http.Response{
		Body:          NewEmptyBody(),
		StatusCode:    299,
		ContentLength: 0,
	}
	mock.EXPECT().Do(gomock.Any()).Return(mockresp, nil)

	f, err := os.OpenFile("test_file/test.txt", 0, 0644)
	assertEquals(t, "Can not Open test File.", err, nil)
	stat, _ := f.Stat()
	size := stat.Size()

	err = client.PutObjectAt("mybucket", "", f, 0, size, nil)
	assertEquals(t, "Pass through HTTP StatusCode check.", err.Error(), "invalid response")
}

func TestGetObjectApiStatusCodeErr(t *testing.T) {
	client, mock := newHTTPClientMock(t)
	mockresp := &http.Response{
		Body:          NewEmptyBody(),
		ContentLength: 0,
		StatusCode:    299,
	}
	mock.EXPECT().Do(gomock.Any()).Return(mockresp, nil)

	_, err := client.GetObject("mybucket", "")
	assertEquals(t, "Pass through HTTP StatusCode check.", err.Error(), "invalid response")
}

func TestDoesObjectExistApiStatusCodeErr(t *testing.T) {
	client, mock := newHTTPClientMock(t)
	mockresp := &http.Response{
		Body:       NewEmptyBody(),
		StatusCode: 299,
	}
	mock.EXPECT().Do(gomock.Any()).Return(mockresp, errors.New("dummy"))

	resp, err := client.DoesObjectExist("mybucket", "")
	assertEquals(t, "Pass through HTTP StatusCode check.", err.Error(), "dummy")
	assertEquals(t, "Pass through HTTP StatusCode check.", resp, false)
}

func TestGetObjectSummaryApiStatusCodeErr(t *testing.T) {
	client, mock := newHTTPClientMock(t)
	mockresp := &http.Response{Body: NewEmptyBody()}
	mockresp.StatusCode = 299
	mock.EXPECT().Do(gomock.Any()).Return(mockresp, nil)

	summary, err := client.GetObjectSummary("mybucket", "")
	var foo *ObjectSummary
	assertEquals(t, "Pass through HTTP StatusCode check.", err, nil)
	assertEquals(t, "Pass through HTTP StatusCode check.", summary, foo)
}

func TestAbortMultipartUploadApiStatusCodeErr(t *testing.T) {
	client, mock := newHTTPClientMock(t)
	mockresp := &http.Response{Body: NewEmptyBody()}
	mockresp.StatusCode = 200
	mock.EXPECT().Do(gomock.Any()).Return(mockresp, nil)

	upload := &MultipartUpload{}
	err := client.AbortMultipartUpload(upload)
	assertEquals(t, "Should return StatusCode 204 at normal end.", err, nil)
}

type DummyResponseBody struct {
	data io.Reader
}

func (e *DummyResponseBody) Read(p []byte) (n int, err error) {
	return e.data.Read(p)
}

func (e *DummyResponseBody) Close() error {
	return nil
}

func NewEmptyBody() *DummyResponseBody {
	return &DummyResponseBody{data: strings.NewReader("")}
}

func NewBodyWithString(data string) *DummyResponseBody {
	return &DummyResponseBody{data: strings.NewReader(data)}
}
