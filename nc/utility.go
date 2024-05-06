package nc

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"strings"
)

func ExistsFolder(path string) bool {
	if info, err := os.Stat(path); err != nil {
		return false
	} else {
		return info.IsDir()
	}
}

func CalculateFileHash(path string) string {
	file, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		panic(err)
	}
	return fmt.Sprintf("%x", hasher.Sum(nil))
}

func ExtractPath(fullPath, rootKeyword string) string {
	startPos := strings.Index(fullPath, rootKeyword)
	if startPos == -1 {
		return ""
	}
	return fullPath[startPos:]
}
