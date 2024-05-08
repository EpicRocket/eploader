package nc

import (
	"errors"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type FileRequest struct {
	file *os.File
	*request.Request
}

func NewFileRequest(file *os.File, req *request.Request) *FileRequest {
	return &FileRequest{
		file:    file,
		Request: req,
	}
}

func (f *FileRequest) Send() error {
	defer f.file.Close()
	return f.Request.Send()
}

func (f *FileRequest) Abort() {
	defer f.file.Close()
}

type Client struct {
	*s3.S3
	ep         string
	bucket     string
	objectRoot string
}

func NewClient(region, accessKeyID, secretKey, ep, bucket, objectRoot string) *Client {
	creds := credentials.NewStaticCredentials(accessKeyID, secretKey, "")

	sess, err := session.NewSession(&aws.Config{
		Region:           aws.String(region),
		Endpoint:         aws.String(ep),
		Credentials:      creds,
		S3ForcePathStyle: aws.Bool(true),
	})

	if err != nil {
		println(err.Error())
		return nil
	}

	return &Client{
		S3:         s3.New(sess),
		ep:         ep,
		bucket:     bucket,
		objectRoot: objectRoot,
	}
}

func (c *Client) GetObjects() (*s3.ListObjectsV2Output, error) {
	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(c.bucket),
		Prefix: aws.String(c.objectRoot),
	}

	resp, err := c.ListObjectsV2(input)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (c *Client) GetUploadObjects(filePath string) (*FileRequest, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}

	s3Key := ExtractPath(filePath, c.objectRoot)
	if s3Key == "" {
		file.Close()
		return nil, errors.New("invalid file path")
	}
	s3Key = strings.Replace(s3Key, "\\", "/", -1)

	println("Upload Request:", s3Key)

	input := &s3.PutObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(s3Key),
		Body:   file,
	}

	req, _ := c.PutObjectRequest(input)
	return NewFileRequest(file, req), nil
}

func (c *Client) UploadObjects(requests []*FileRequest) {
	for _, req := range requests {
		if err := req.Send(); err != nil {
			println(err.Error())
		}
	}
}

func (c *Client) UploadDeleteObject(key string) error {
	input := &s3.DeleteObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	}

	_, err := c.DeleteObject(input)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) DownloadFile(key, output string) error {
	outFile, err := os.OpenFile(output, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return err
	}

	// S3에서 파일 스트림 다운로드
	input := &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	}

	resp, err := c.GetObject(input)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 로컬 파일로 복사
	if _, err := outFile.ReadFrom(resp.Body); err != nil {
		return err
	}
	defer outFile.Close()

	return nil
}
