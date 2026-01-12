package content

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/katbyte/go-ingest-media/lib/ktio"
)

// Content represents a folder for a movie or series in a single library
type Content struct {
	Library *Library
	Folder  string // folder name (not full path)
	Letter  string
	Year    int
}

type ContentInterface interface {
	// Methods if any
}

// ContentFor creates a Content from a folder name in the given library
func ContentFor(lib *Library, folder string) (*Content, error) {
	f := filepath.Base(folder)

	// check for trailing whitespace
	if f != strings.TrimSpace(f) {
		return nil, fmt.Errorf("folder name has leading/trailing whitespace: %q", f)
	}

	c := Content{
		Library: lib,
		Folder:  f,
		Letter:  GetLetter(f),
	}

	// get year - look for (YYYY) anywhere in the folder name
	regex := regexp.MustCompile(`\((\d{4})\)`)
	allMatches := regex.FindAllStringSubmatch(c.Folder, -1)
	if len(allMatches) > 1 {
		return nil, fmt.Errorf("multiple years found in folder name: %q", c.Folder)
	}
	if len(allMatches) == 1 {
		c.Year, _ = strconv.Atoi(allMatches[0][1])
	}
	// Year is optional - will be 0 if not found

	return &c, nil
}

// Path returns the full path to this content folder
func (c Content) Path() string {
	if c.Library.LetterFolders {
		return filepath.Join(c.Library.Path, c.Letter, c.Folder)
	}
	return filepath.Join(c.Library.Path, c.Folder)
}

// Exists returns true if this content folder exists
func (c Content) Exists() bool {
	return ktio.PathExists(c.Path())
}

// MoveFolder moves this content folder to the given destination path
func (c Content) MoveFolder(destPath string, confirm bool, indent int) error {
	return ktio.RunCommand(indent, confirm, "mv", "-v", c.Path(), destPath)
}

// DeleteFolder deletes this content folder
func (c Content) DeleteFolder(confirm bool, indent int) error {
	return ktio.RunCommand(indent, confirm, "rm", "-rfv", c.Path())
}

// DestPathIn returns what the path would be in the given destination library
func (c Content) DestPathIn(destLib *Library) string {
	if destLib.LetterFolders {
		return filepath.Join(destLib.Path, c.Letter, c.Folder)
	}
	return filepath.Join(destLib.Path, c.Folder)
}

// DestPathInWithRename returns the destination path with potential folder rename
func (c Content) DestPathInWithRename(destLib *Library, libType LibraryType) (string, error) {
	destFolder := c.Folder

	// lookup potential alt folder name
	altFolder, err := AltFolderFor(libType, c.Folder)
	if err != nil {
		return "", err
	}
	if altFolder != nil {
		destFolder = *altFolder
	}

	if destLib.LetterFolders {
		return filepath.Join(destLib.Path, c.Letter, destFolder), nil
	}
	return filepath.Join(destLib.Path, destFolder), nil
}

// ExistsIn returns true if this content exists in the given library
func (c Content) ExistsIn(lib *Library) bool {
	return ktio.PathExists(c.DestPathIn(lib))
}
