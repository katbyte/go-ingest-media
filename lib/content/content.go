package content

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/katbyte/go-ingest-media/lib/ktio"
)

// represents "the folder" for a movie or series
// centred around a source folder and a destination folder
type Content struct {
	Library   Library
	SrcFolder string
	DstFolder string
	Letter    string
	Year      int
}

type ContentInterface interface {
	// Methods if any
}

func (l Library) ContentFor(path string) (*Content, error) {
	f := filepath.Base(path)

	// check for trailing whitespace
	if f != strings.TrimSpace(f) {
		return nil, fmt.Errorf("folder name has leading/trailing whitespace: %q", f)
	}

	m := Content{
		Library:   l,
		SrcFolder: f,
		Letter:    GetLetter(f),
	}

	// lookup potential alt folder
	altFolder, err := l.AltFolderFor(f)
	if err != nil {
		return nil, err
	}
	if altFolder != nil {
		m.DstFolder = *altFolder
	} else {
		m.DstFolder = f
	}

	// get year
	regex := regexp.MustCompile(`\((?P<year>\d{4})\)$`)
	matches := regex.FindStringSubmatch(m.SrcFolder)
	if len(matches) == 0 {
		return nil, fmt.Errorf("no year found in folder name: '%q'", m.SrcFolder)
	} else if len(matches) > 2 {
		return nil, fmt.Errorf("more than one year found in folder name: '%q'", m.SrcFolder)
	}

	// this must be a valid year
	m.Year, _ = strconv.Atoi(matches[1])

	return &m, nil
}

func (c Content) SrcPath() string {
	return filepath.Join(c.Library.SrcPath, c.SrcFolder)
}

func (c Content) DstPath() string {
	if c.Library.LetterFolders {
		return filepath.Join(c.Library.DstPath, c.Letter, c.DstFolder)
	} else {
		return filepath.Join(c.Library.DstPath, c.DstFolder)
	}
}

func (c Content) DstExists() bool {
	return ktio.PathExists(c.DstPath())
}

func (c Content) MoveFolder(confirm bool, indent int) error {
	return ktio.RunCommand(indent, confirm, "mv", "-v", c.SrcPath(), c.DstPath())
}
