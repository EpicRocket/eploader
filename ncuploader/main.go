package main

import (
	"eploader/nc"
	"fmt"
	"os"
	"path/filepath"

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

	//s3Objects := client.GetObjects()
	requests := make([]*nc.FileRequest, 0, 1000)
	fail := false

	for _, p := range config.UploadPaths {
		err := filepath.Walk(
			p,
			func(path string, info os.FileInfo, err error) error {
				if err != nil {
					fmt.Println("Error walking path:", path)
					return err
				}

				if info.IsDir() {
					return nil
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
