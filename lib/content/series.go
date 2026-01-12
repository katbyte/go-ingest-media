package content

import (
	"fmt"

	c "github.com/gookit/color"
	"github.com/katbyte/go-ingest-media/lib/ktio"
)

// Series adds to the content type (folder) by adding video details for TV series

type Series struct {
	Content

	Seasons map[int]Season

	// For import flow - destination seasons loaded separately
	DstSeasons map[int]Season

	ExtraFiles   []string
	SpecialFiles []string
}

// SeriesFor creates a Series from a folder in the given library
func SeriesFor(lib *Library, folder string) (*Series, error) {
	content, err := ContentFor(lib, folder)
	if err != nil {
		return nil, err
	}

	s := Series{
		Content: *content,
	}

	return &s, nil
}

// LoadSeasons loads season info for this series
func (s *Series) LoadSeasons() error {
	var err error
	s.Seasons, err = GetSeasons(s.Path())
	if err != nil {
		c.Printf("    <yellow>WARNING:</> error loading seasons: %s\n", err)
	}

	// Load extras and specials
	if exists := ktio.PathExists(s.Path() + "/extras"); exists {
		s.ExtraFiles, err = ktio.ListFiles(s.Path() + "/extras")
		if err != nil {
			return fmt.Errorf("error listing extra files: %w", err)
		}
	}

	if exists := ktio.PathExists(s.Path() + "/specials"); exists {
		s.SpecialFiles, err = ktio.ListFiles(s.Path() + "/specials")
		if err != nil {
			return fmt.Errorf("error listing special files: %w", err)
		}
	}

	return nil
}

// LoadDestSeasons loads season info from a destination path (for import comparison)
func (s *Series) LoadDestSeasons(destPath string) error {
	var err error
	s.DstSeasons, err = GetSeasons(destPath)
	if err != nil {
		c.Printf("    <yellow>WARNING:</> error loading destination seasons: %s\n", err)
	}
	return nil
}
