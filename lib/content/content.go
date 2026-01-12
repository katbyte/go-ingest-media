package content

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/katbyte/go-ingest-media/lib/ktio"
)

// Content represents "the folder" for a movie or series
// It knows about both source and destination paths via the mapping
type Content struct {
	Mapping   LibraryMapping
	SrcFolder string
	DstFolder string
	Letter    string
	Year      int
}

type ContentInterface interface {
	// Methods if any
}

// ContentFor creates a Content from a source folder path using the given mapping
func ContentFor(mapping LibraryMapping, path string) (*Content, error) {
	f := filepath.Base(path)

	// Validate that source and destination library types match
	if mapping.Source.Type != mapping.Dest.Type {
		return nil, fmt.Errorf("library type mismatch: source=%d, dest=%d", mapping.Source.Type, mapping.Dest.Type)
	}

	// check for trailing whitespace
	if f != strings.TrimSpace(f) {
		return nil, fmt.Errorf("folder name has leading/trailing whitespace: %q", f)
	}

	c := Content{
		Mapping:   mapping,
		SrcFolder: f,
		Letter:    GetLetter(f),
	}

	// lookup potential alt folder
	altFolder, err := AltFolderFor(mapping.Source.Type, f)
	if err != nil {
		return nil, err
	}
	if altFolder != nil {
		c.DstFolder = *altFolder
	} else {
		c.DstFolder = f
	}

	// get year
	regex := regexp.MustCompile(`\((?P<year>\d{4})\)$`)
	matches := regex.FindStringSubmatch(c.SrcFolder)
	if len(matches) == 0 {
		return nil, fmt.Errorf("no year found in folder name: '%q'", c.SrcFolder)
	} else if len(matches) > 2 {
		return nil, fmt.Errorf("more than one year found in folder name: '%q'", c.SrcFolder)
	}

	// this must be a valid year
	c.Year, _ = strconv.Atoi(matches[1])

	return &c, nil
}

func (c Content) SrcPath() string {
	return filepath.Join(c.Mapping.Source.Path, c.SrcFolder)
}

func (c Content) DstPath() string {
	dest := c.Mapping.Dest
	if dest.LetterFolders {
		return filepath.Join(dest.Path, c.Letter, c.DstFolder)
	}
	return filepath.Join(dest.Path, c.DstFolder)
}

func (c Content) DstExists() bool {
	return ktio.PathExists(c.DstPath())
}

func (c Content) MoveFolder(confirm bool, indent int) error {
	return ktio.RunCommand(indent, confirm, "mv", "-v", c.SrcPath(), c.DstPath())
}
