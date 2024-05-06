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

func (c *Client) GetObjects() map[string]string {
	result, err := c.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket: aws.String(c.bucket),
	})
	if err != nil {
		return nil
	}

	s3Objects := make(map[string]string)
	for _, obj := range result.Contents {
		s3Objects[*obj.Key] = *obj.ETag
	}
	return s3Objects
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
