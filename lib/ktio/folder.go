package ktio

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func ListFolders(ipath string) ([]string, error) {
	paths := []string{}

	files, err := os.ReadDir(ipath)
	if err != nil {
		return nil, fmt.Errorf("error reading the path %v: %w\n", ipath, err)
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
		return nil, fmt.Errorf("error reading the path %v: %w\n", ipath, err)
	}

	for _, file := range files {
		if !file.IsDir() {
			fullPath := filepath.Join(ipath, file.Name())
			paths = append(paths, fullPath)
		}
	}

	return paths, nil
}

func ListFilesAndFolders(ipath string) ([]string, error) {
	var paths []string

	files, err := os.ReadDir(ipath)
	if err != nil {
		return nil, fmt.Errorf("error reading the path %v: %w\n", ipath, err)
	}

	for _, file := range files {
		fullPath := filepath.Join(ipath, file.Name())
		paths = append(paths, fullPath)
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
		return errors.New("destination folder already exists")
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

func DeleteIfEmpty(path string, confirm bool, indent int) error {
	empty, err := FolderEmpty(path)
	if err != nil {
		return fmt.Errorf("error checking if folder is empty: %w", err)
	}

	if empty {
		if err := RunCommand(indent, confirm, "rmdir", "-v", path); err != nil {
			return fmt.Errorf("error deleting empty folder: %w", err)
		}
	}

	return nil
}

func DeleteIfEmptyOrOnlyNfo(path string, confirm bool, indent int) error {
	srcContents, err := ListFilesAndFolders(path)
	if err != nil {
		return fmt.Errorf("error listing source content: %w", err)
	}

	// delete all nfo files
	for _, contentPath := range srcContents {
		if filepath.Ext(contentPath) == ".nfo" {
			if err := RunCommand(indent, confirm, "rm", "-v", contentPath); err != nil {
				return fmt.Errorf("error deleting nfo file: %w", err)
			}
		}
	}

	// delete source folder if empty
	if err := DeleteIfEmpty(path, confirm, indent); err != nil {
		return fmt.Errorf("error deleting source folder: %w", err)
	}

	return nil
}
