package content

import (
	"fmt"
	"path/filepath"

	c "github.com/gookit/color"
	"github.com/katbyte/go-ingest-media/lib/ktio"
)

// Movie adds to the content type (folder) by adding singular video details as 1 movie has 1 video file

type Movie struct {
	Content

	// video details
	SrcVideos []VideoFile
	DstVideos []VideoFile
}

// MovieFor creates a Movie from a source folder (used for single library scanning)
func MovieFor(lib Library, folder string) (*Movie, error) {
	m := Movie{}

	// For single library scanning, we create a dummy mapping
	// This is used when scanning a library independently (not for src->dst processing)
	m.Content = Content{
		SrcFolder: folder,
		DstFolder: folder,
		Letter:    GetLetter(folder),
	}

	return &m, nil
}

// MovieForMapping creates a Movie for source->destination processing
func MovieForMapping(mapping LibraryMapping, folder string) (*Movie, error) {
	m := Movie{}

	c, err := ContentFor(mapping, folder)
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

func (m *Movie) MoveFiles(confirm bool, indent int) error {
	// delete destination video files
	for _, v := range m.DstVideos {
		if err := ktio.RunCommand(indent, confirm, "rm", "-v", v.Path); err != nil {
			c.Printf("   <red>ERROR:</> deleting destination video: %s\n", err)
		}
	}

	// move source video files
	for _, v := range m.SrcVideos {
		if err := ktio.RunCommand(indent, confirm, "mv", "-v", v.Path, m.DstPath()+"/"); err != nil {
			return fmt.Errorf("error moving source video: %w", err)
		}
	}

	// move all other files in the source folder
	srcContents, err := ktio.ListFilesAndFolders(m.SrcPath())
	if err != nil {
		return fmt.Errorf("error listing source content: %w", err)
	}

	for _, contentPath := range srcContents {
		// skip nfo files
		if filepath.Ext(contentPath) == ".nfo" {
			continue
		}

		// skip if video file (already moved)
		if IsVideoFile(contentPath) {
			continue
		}

		// move file or folder
		if err := ktio.RunCommand(indent, confirm, "mv", "-v", contentPath, m.DstPath()+"/"); err != nil {
			return fmt.Errorf("error moving file or folder: %w", err)
		}
	}

	// delete source folder if empty
	if err := ktio.DeleteIfEmptyOrOnlyNfo(m.SrcPath(), confirm, indent); err != nil {
		return fmt.Errorf("error deleting source folder: %w", err)
	}

	return nil
}
