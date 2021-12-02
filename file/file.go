package file

import (
	"fmt"
	"os"
)

func ReadFile(path string) ([]byte, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	fmt.Printf("file content is :%s\n", string(content))
	return content, nil
}

