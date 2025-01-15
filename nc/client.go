package nc

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type FileRequest struct {
	file *os.File
	key  string
	*request.Request
}

func NewFileRequest(file *os.File, key string, req *request.Request) *FileRequest {
	return &FileRequest{
		file:    file,
		key:     key,
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
	uploadRoot string
}

func NewClient(region, accessKeyID, secretKey, ep, bucket, objectRoot, uploadRoot string) *Client {
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
		uploadRoot: uploadRoot,
	}
}

func (c *Client) GetObjects() ([]*s3.ListObjectsV2Output, error) {
	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(c.bucket),
		Prefix: aws.String(c.objectRoot),
	}

	result := []*s3.ListObjectsV2Output{}

	for {
		resp, err := c.ListObjectsV2(input)
		if err != nil {
			return nil, err
		}

		result = append(result, resp)

		if resp.IsTruncated != nil && *resp.IsTruncated {
			input.ContinuationToken = resp.NextContinuationToken
		} else {
			break
		}
	}

	return result, nil
}

func (c *Client) GetUploadObjects(filePath string) (*FileRequest, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}

	s3Key, err := filepath.Rel(c.uploadRoot, filePath)
	if err != nil {
		file.Close()
		return nil, err
	}
	s3Key = filepath.Join(c.objectRoot, s3Key)
	s3Key = strings.Replace(s3Key, "\\", "/", -1)

	input := &s3.PutObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(s3Key),
		Body:   file,
	}

	req, _ := c.PutObjectRequest(input)
	return NewFileRequest(file, s3Key, req), nil
}

func (c *Client) UploadObjects(requests []*FileRequest) {
	for _, req := range requests {
		if err := req.Send(); err != nil {
			println(err.Error())
		} else {
			println("Uploaded: ", req.key)
		}
		req.file.Close()
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
	outFile, err := os.Create(output)
	if err != nil {
		return err
	}
	defer outFile.Close()

	input := &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	}

	resp, err := c.GetObject(input)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 버퍼를 사용한 파일 쓰기
	writer := bufio.NewWriter(outFile)
	if _, err := io.Copy(writer, resp.Body); err != nil {
		return err
	}

	return writer.Flush()
}
