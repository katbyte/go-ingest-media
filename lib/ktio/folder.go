package ktio

import (
	"fmt"
	"os"
	"path/filepath"
)

func ListFolders(ipath string) ([]string, error) {
	paths := []string{}

	files, err := os.ReadDir(ipath)
	if err != nil {
		return nil, fmt.Errorf("error reading the path %v: %v\n", ipath, err)
	}

	for _, file := range files {
		if file.IsDir() {
			fullPath := filepath.Join(ipath, file.Name())
			paths = append(paths, fullPath)
		}
	}

	return paths, nil
}

func ListFiles(ipath string) ([]string, error) {
	var paths []string

	files, err := os.ReadDir(ipath)
	if err != nil {
		return nil, fmt.Errorf("error reading the path %v: %v\n", ipath, err)
	}

	for _, file := range files {
		if !file.IsDir() {
			fullPath := filepath.Join(ipath, file.Name())
			paths = append(paths, fullPath)
		}
	}

	return paths, nil
}

func PathExists(path string) bool {
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

	// MoveFolder the folder
	err := os.Rename(source, destination)
	if err != nil {
		return fmt.Errorf("error while moving folder: %w", err)
	}

	return nil
}

func FolderEmpty(dirPath string) (bool, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return false, err
	}

	return len(entries) == 0, nil
}
