package folders

import (
	"fmt"
	"os"
	"path/filepath"
)

func List(ipath string) ([]string, error) {
	paths := []string{}
	err := filepath.Walk(ipath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("error with path %v: %v\n", path, err)
		}

		// skip current dir
		if path == ipath {
			return nil
		}

		if info.IsDir() {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error walking the path %v: %v\n", ipath, err)
	}
	return paths, nil
}

func Exists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}

	return info.IsDir()
}

func Move(source, destination string) error {
	// Check if destination already exists
	if _, err := os.Stat(destination); !os.IsNotExist(err) {
		return fmt.Errorf("destination folder already exists")
	}

	// Move the folder
	err := os.Rename(source, destination)
	if err != nil {
		return fmt.Errorf("error while moving folder: %w", err)
	}

	return nil
}
