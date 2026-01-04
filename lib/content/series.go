package content

import (
	"fmt"

	"github.com/katbyte/go-ingest-media/lib/ktio"
)

// adds to the content type (folder) by adding singular video details as 1 movie has 1 video file

type Series struct {
	Content

	SrcSeasons map[int]Season
	DstSeasons map[int]Season

	ExtraFiles   []string
	SpecialFiles []string
}

func (l Library) SeriesFor(folder string) (*Series, error) {
	s := Series{}

	c, err := l.ContentFor(folder)
	if err != nil {
		return nil, err
	}
	s.Content = *c

	return &s, nil
}

// only called if destination folder exists
func (s *Series) LoadContentDetails() error {
	var err error
	s.SrcSeasons, err = GetSeasons(s.SrcPath())
	if err != nil {
		return fmt.Errorf("error loading source seasons: %w", err)
	}

	s.DstSeasons, err = GetSeasons(s.DstPath())
	if err != nil {
		return fmt.Errorf("error loading destination seasons: %w", err)
	}

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
