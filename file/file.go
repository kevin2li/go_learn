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
	fmt.Printf("file content is :\n%s\n", string(content))
	return content, nil
}

func ListDir(path string) ([]string, error) {
	entry_list, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	res := make([]string, 0, len(entry_list))
	for _, entry := range entry_list {
		fmt.Printf("%s\n", entry.Name())
		res = append(res, entry.Name())
	}
	return res, nil
}