package cli

import (
	"fmt"
	"path"
	"sort"

	c "github.com/gookit/color"
	"github.com/katbyte/go-ingest-media/lib/content"
	"github.com/katbyte/go-ingest-media/lib/ktio"
	_ "github.com/mattn/go-sqlite3"
)

func DocuDupsMovies(docuLibrary, movieLibrary *content.Library) error {
	// Get documentary folders
	docuFolders, err := ktio.ListFolders(docuLibrary.Path)
	if err != nil {
		return fmt.Errorf("error listing docu folders: %w", err)
	}

	docus := make([]string, 0, len(docuFolders))
	for _, folder := range docuFolders {
		docus = append(docus, path.Base(folder))
	}
	sort.Strings(docus)

	// Get movie folders
	movieFolders, err := ktio.ListFolders(movieLibrary.Path)
	if err != nil {
		return fmt.Errorf("error listing movie folders: %w", err)
	}

	// For letter folders, need to go one level deeper
	movies := []string{}
	if movieLibrary.LetterFolders {
		for _, letterFolder := range movieFolders {
			subFolders, err := ktio.ListFolders(letterFolder)
			if err != nil {
				c.Printf("  %s --> <red>ERROR:</>%s</>\n", path.Base(letterFolder), err)
				continue
			}
			for _, folder := range subFolders {
				movies = append(movies, path.Base(folder))
			}
		}
	} else {
		for _, folder := range movieFolders {
			movies = append(movies, path.Base(folder))
		}
	}
	sort.Strings(movies)

	for _, d := range docus {
		for _, m := range movies {
			if d == m {
				// found a match
				fmt.Println("Match found: ", d)
			}
		}
	}

	return nil
}
