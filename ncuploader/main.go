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
	Region       string   `yaml:"Region,omitempty"`
	AccessKeyID  string   `yaml:"AccessKeyID,omitempty"`
	SecretAccess string   `yaml:"SecretKey,omitempty"`
	EndPoint     string   `yaml:"EndPoint,omitempty"`
	Bucket       string   `yaml:"Bucket,omitempty"`
	ObjectRoot   string   `yaml:"ObjectRoot,omitempty"`
	UploadPaths  []string `yaml:"UploadPaths,omitempty"`
	ExcludeTags  []string `yaml:"ExcludeTags,omitempty"`
}

func main() {
	if len(os.Args) != 2 {
		for _, arg := range os.Args {
			fmt.Println(arg)
		}
		panic("Usage: ncuploader <uploadList.yaml>")
	}

	uploadList, err := os.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}

	var config Config
	if err := yaml.Unmarshal(uploadList, &config); err != nil {
		panic(err)
	}

	client := nc.NewClient(
		config.Region,
		config.AccessKeyID,
		config.SecretAccess,
		config.EndPoint,
		config.Bucket,
		config.ObjectRoot,
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
	for _, item := range s3Objects.Contents {
		s3ObjectMap[*item.Key] = item
	}

	requests := make([]*nc.FileRequest, 0, 1000)
	fail := false

	// 로컬 파일 목록 생성
	localFiles := make(map[string]struct{})

	// 제외할 디렉터리를 맵으로 변환
	excludeDirs := make(map[string]bool)
	for _, tag := range config.ExcludeTags {
		excludeDirs[tag] = true
	}

	for _, p := range config.UploadPaths {
		err := filepath.Walk(
			p,
			func(path string, info os.FileInfo, err error) error {
				if err != nil {
					fmt.Println("Error walking path:", path)
					return err
				}

				relPath, err := filepath.Rel(p, path)
				if err != nil {
					return err
				}

				if info.IsDir() {
					// 경로에 제외 대상 디렉터리가 포함되어 있는지 확인
					if excludeDirs[filepath.Base(relPath)] {
						return filepath.SkipDir
					}
					return nil
				}

				localFiles[relPath] = struct{}{}

				// C:/Temp로 복사
				// relPath2, err := filepath.Rel("D:/UnrealEngine", path)
				// if err != nil {
				// 	return err
				// }

				// destPath := filepath.Join("C:/Temp", relPath2)
				// if err := os.MkdirAll(filepath.Dir(destPath), os.ModePerm); err != nil {
				// 	return err
				// }

				// srcFile, err := os.Open(path)
				// if err != nil {
				// 	return err
				// }

				// defer srcFile.Close()

				// destFile, err := os.Create(destPath)
				// if err != nil {
				// 	return err
				// }

				// defer destFile.Close()

				// if _, err := io.Copy(destFile, srcFile); err != nil {
				// 	return err
				// }

				// 로컬 파일 해시 계산
				localFileHash, err := nc.GetFileMD5Hash(path)
				if err != nil {
					return err
				}

				// S3 객체와 비교
				if s3Obj, exists := s3ObjectMap[relPath]; exists {
					s3FileHash := *s3Obj.ETag // ETag is usually the MD5 hash in quotes
					if strings.Trim(s3FileHash, `"`) == localFileHash {
						return nil // 파일이 같으므로 업로드 건너뛰기
					}
				}

				req, err := client.GetUploadObjects(path)
				if err != nil {
					return err
				}
				requests = append(requests, req)

				return nil
			},
		)

		if err != nil {
			println(err.Error())
			fail = true
			break
		}
	}

	// 존재하지 않는 S3 객체 삭제
	for key, _ := range s3ObjectMap {
		if _, exists := localFiles[key]; !exists {
			err := client.UploadDeleteObject(key)
			if err != nil {
				fmt.Printf("Failed to delete %s: %v\n", key, err)
			} else {
				fmt.Printf("Deleted %s\n", key)
			}
		}
	}

	if fail {
		for _, req := range requests {
			req.Abort()
		}

		println("Upload failed")
		return
	}

	if len(requests) > 0 {
		client.UploadObjects(requests)
	}

	fmt.Println("Upload complete")
}
