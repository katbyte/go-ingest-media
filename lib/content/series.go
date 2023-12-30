package content

import (
	"fmt"
)

// adds to the content type (folder) by adding singular video details as 1 movie has 1 video file

type Series struct {
	Content

	SrcSeasons map[int]Season
	DstSeasons map[int]Season
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
func (s *Series) LoadSeasons() error {

	var err error
	s.SrcSeasons, err = GetSeasons(s.SrcPath())
	if err != nil {
		return fmt.Errorf("error loading source seasons: %w", err)
	}

	s.DstSeasons, err = GetSeasons(s.DstPath())
	if err != nil {
		return fmt.Errorf("error loading source seasons: %w", err)
	}

	return nil
}
