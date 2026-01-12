package content

import (
	"fmt"
	"path/filepath"

	c "github.com/gookit/color"
	"github.com/katbyte/go-ingest-media/lib/ktio"
)

// Movie adds to the content type (folder) by adding video details

type Movie struct {
	Content
	Videos []VideoFile
}

// MovieFor creates a Movie from a folder in the given library
func MovieFor(lib *Library, folder string) (*Movie, error) {
	content, err := ContentFor(lib, folder)
	if err != nil {
		return nil, err
	}

	m := Movie{
		Content: *content,
	}

	return &m, nil
}

// LoadVideos loads video info for this movie
func (m *Movie) LoadVideos() error {
	var err error
	m.Videos, err = VideosInPath(m.Path())
	if err != nil {
		return fmt.Errorf("error loading videos: %w", err)
	}
	return nil
}

// MoveFilesTo moves video and other files to the given destination path
func (m *Movie) MoveFilesTo(destPath string, confirm bool, indent int) error {
	// move video files
	for _, v := range m.Videos {
		if err := ktio.RunCommand(indent, confirm, "mv", "-v", v.Path, destPath+"/"); err != nil {
			return fmt.Errorf("error moving video: %w", err)
		}
	}

	// move all other files in the source folder
	srcContents, err := ktio.ListFilesAndFolders(m.Path())
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
		if err := ktio.RunCommand(indent, confirm, "mv", "-v", contentPath, destPath+"/"); err != nil {
			return fmt.Errorf("error moving file or folder: %w", err)
		}
	}

	// delete source folder if empty
	if err := ktio.DeleteIfEmptyOrOnlyNfo(m.Path(), confirm, indent); err != nil {
		return fmt.Errorf("error deleting source folder: %w", err)
	}

	return nil
}

// DeleteVideos deletes all video files for this movie
func (m *Movie) DeleteVideos(confirm bool, indent int) {
	for _, v := range m.Videos {
		if err := ktio.RunCommand(indent, confirm, "rm", "-v", v.Path); err != nil {
			c.Printf("   <red>ERROR:</> deleting video: %s\n", err)
		}
	}
}
