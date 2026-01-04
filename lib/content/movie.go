package content

import (
	"fmt"

	c "github.com/gookit/color"
	"github.com/katbyte/go-ingest-media/lib/ktio"
)

// adds to the content type (folder) by adding singular video details as 1 movie has 1 video file

type Movie struct {
	Content

	// video details
	SrcVideos []VideoFile
	DstVideos []VideoFile
}

func (l Library) MovieFor(folder string) (*Movie, error) {
	m := Movie{}

	c, err := l.ContentFor(folder)
	if err != nil {
		return nil, err
	}
	m.Content = *c

	return &m, nil
}

func (m *Movie) LoadVideoInfo() error {
	var err error

	m.SrcVideos, err = VideosInPath(m.SrcPath())
	if err != nil {
		return fmt.Errorf("error loading source video: %w", err)
	}

	m.DstVideos, err = VideosInPath(m.DstPath())
	if err != nil {
		return fmt.Errorf("error loading destination video info: %w", err)
	}

	return nil
}

func (m *Movie) MoveFiles(srcVideo VideoFile, confirm bool, indent int) error {
	// delete destination video files
	for _, v := range m.DstVideos {
		_ = ktio.RunCommand(indent, confirm, "rm", "-v", v.Path)
	}

	// move source video file
	err := ktio.RunCommand(indent, confirm, "mv", "-v", srcVideo.Path, m.DstPath()+"/")
	if err != nil {
		return err
	}

	// delete source folder if empty
	empty, err := ktio.FolderEmpty(m.SrcPath())
	if err != nil {
		c.Printf(" <red>ERROR:</> checking if empty%s\n", err)
	}
	if empty {
		_ = ktio.RunCommand(indent, confirm, "rmdir", "-v", m.SrcPath())
	}

	return nil
}
