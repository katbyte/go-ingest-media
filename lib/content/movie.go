package content

import (
	"fmt"

	"github.com/katbyte/go-ingest-media/lib/ktio"
)

// adds to the content type (folder) by adding singular video details as 1 movie has 1 video file

type Movie struct {
	Content

	// video details
	SrcVideo  VideoFile
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
	vs, err := VideosInPath(m.SrcPath())
	if err != nil {
		return fmt.Errorf("error loading source video: %w", err)
	}

	if len(vs) != 1 {
		return fmt.Errorf("expected 1 src video file, found %d", len(vs))
	}
	m.SrcVideo = vs[0]

	m.DstVideos, err = VideosInPath(m.DstPath())
	if err != nil {
		return fmt.Errorf("error loading destination video info: %w", err)
	}

	return nil
}

func (m *Movie) MoveFiles(confirm bool, indent int) error {

	// delete video files in destination
	files, err := ktio.ListFiles(m.DstPath())
	if err != nil {
		return fmt.Errorf("error listing destination files: %w", err)
	}

	for _, f := range files {
		if IsVideoFile(f) {
			err := ktio.RunCommand(indent, confirm, "rm", "-v", f)
			if err != nil {
				return err
			}
		}
	}

	// move each file/folder in source path
	files, err = ktio.ListFiles(m.SrcPath())
	if err != nil {
		return fmt.Errorf("error listing destination files: %w", err)
	}

	// move all video files (do we really care about anything else?)
	for _, f := range files {
		if IsVideoFile(f) {
			err := ktio.RunCommand(indent, confirm, "mv", "-v", f, m.DstPath()+"/")
			if err != nil {
				return err
			}
		}
	}

	// delete source folder via rmdir
	return ktio.RunCommand(indent, confirm, "rmdir", "-v", m.SrcPath())
}
