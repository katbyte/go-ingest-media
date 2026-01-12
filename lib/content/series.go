package content

import (
	"fmt"
	"sync"

	c "github.com/gookit/color"
	"github.com/katbyte/go-ingest-media/lib/ktio"
)

// Series adds to the content type (folder) by adding video details for TV series

type Series struct {
	Content

	SrcSeasons map[int]Season
	DstSeasons map[int]Season

	ExtraFiles   []string
	SpecialFiles []string
}

// SeriesFor creates a Series from a source folder (used for single library scanning)
func SeriesFor(lib Library, folder string) (*Series, error) {
	s := Series{}

	// For single library scanning, we create a minimal content
	s.Content = Content{
		SrcFolder: folder,
		DstFolder: folder,
		Letter:    GetLetter(folder),
	}

	return &s, nil
}

// SeriesForMapping creates a Series for source->destination processing
func SeriesForMapping(mapping LibraryMapping, folder string) (*Series, error) {
	s := Series{}

	c, err := ContentFor(mapping, folder)
	if err != nil {
		return nil, err
	}
	s.Content = *c

	return &s, nil
}

// only called if destination folder exists
func (s *Series) LoadContentDetails() error {
	var wg sync.WaitGroup
	var srcErr, dstErr error

	wg.Add(2)

	// Load source seasons
	go func() {
		defer wg.Done()
		s.SrcSeasons, srcErr = GetSeasons(s.SrcPath())
	}()

	// Load destination seasons
	go func() {
		defer wg.Done()
		s.DstSeasons, dstErr = GetSeasons(s.DstPath())
	}()

	wg.Wait()

	// Log errors as warnings but continue
	if srcErr != nil {
		c.Printf("    <yellow>WARNING:</> error loading source seasons: %s\n", srcErr)
	}
	if dstErr != nil {
		c.Printf("    <yellow>WARNING:</> error loading destination seasons: %s\n", dstErr)
	}

	var err error
	if exists := ktio.PathExists(s.SrcPath() + "/extras"); exists {
		s.ExtraFiles, err = ktio.ListFiles(s.SrcPath() + "/extras")
		if err != nil {
			return fmt.Errorf("error listing extra files: %w", err)
		}
	}

	if exists := ktio.PathExists(s.SrcPath() + "/specials"); exists {
		s.SpecialFiles, err = ktio.ListFiles(s.SrcPath() + "/specials")
		if err != nil {
			return fmt.Errorf("error listing special files: %w", err)
		}
	}

	return nil
}
