package main

import (
	"eploader/nc"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/service/s3"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Region       string `yaml:"Region,omitempty"`
	AccessKeyID  string `yaml:"AccessKeyID,omitempty"`
	SecretAccess string `yaml:"SecretKey,omitempty"`
	EndPoint     string `yaml:"EndPoint,omitempty"`
	Bucket       string `yaml:"Bucket,omitempty"`
	ObjectRoot   string `yaml:"ObjectRoot,omitempty"`
	DownloadPath string `yaml:"DownloadPath,omitempty"`
}

func main() {
	if len(os.Args) != 2 {
		for _, arg := range os.Args {
			fmt.Println(arg)
		}
		panic("Usage: ncuploader <downloadConfig.yaml>")
	}

	downloadConfig, err := os.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}

	var config Config
	if err := yaml.Unmarshal(downloadConfig, &config); err != nil {
		panic(err)
	}

	// config.DownloadPath 검증
	if _, err := os.Stat(config.DownloadPath); os.IsNotExist(err) {
		// 없다면 실행 위치로 변경
		config.DownloadPath, err = os.Getwd()
	}

	client := nc.NewClient(
		config.Region,
		config.AccessKeyID,
		config.SecretAccess,
		config.EndPoint,
		config.Bucket,
		config.ObjectRoot,
		config.DownloadPath,
	)

	if client == nil {
		panic("client is nil")
	}

	s3Objects, err := client.GetObjects()
	if err != nil {
		panic(err)
	}

	// S3 객체를 맵에 저장하여 빠르게 접근 가능하게 함
	s3ObjectMap := make(map[string]*s3.Object)

	for _, resp := range s3Objects {
		for _, i := range resp.Contents {
			prefixKey := strings.TrimPrefix(*i.Key, config.ObjectRoot)
			if prefixKey[0] == '/' {
				prefixKey = prefixKey[1:]
			}
			s3ObjectMap[prefixKey] = i
		}
	}

	for _, i := range s3ObjectMap {
		prefixKey := strings.TrimPrefix(*i.Key, config.ObjectRoot)
		if prefixKey[0] == '/' {
			prefixKey = prefixKey[1:]
		}

		outputPath := filepath.Join(config.DownloadPath, prefixKey)

		if localFileHash, err := nc.GetFileMD5Hash(outputPath); err == nil {
			relPath, err := filepath.Rel(config.DownloadPath, outputPath)
			relPath = strings.Replace(relPath, "\\", "/", -1)
			if err != nil {
				panic(err)
			}

			// S3 객체와 비교
			if s3Obj, exists := s3ObjectMap[relPath]; exists {
				s3FileHash := *s3Obj.ETag // ETag is usually the MD5 hash in quotes
				if strings.Trim(s3FileHash, `"`) == localFileHash {
					continue // 파일이 같으므로 업로드 건너뛰기
				}
			}
		}

		if err := os.MkdirAll(filepath.Dir(outputPath), os.ModePerm); err != nil {
			panic(err)
		}

		err := client.DownloadFile(*i.Key, outputPath)
		if err != nil {
			panic(err)
		}
		fmt.Println("Downloaded", *i.Key, "to", outputPath)
	}

	fmt.Println("Download completed")
}
