package cli

import (
	"fmt"
	"path"
	"sort"

	c "github.com/gookit/color"
	"github.com/katbyte/go-ingest-media/lib/content"
	_ "github.com/mattn/go-sqlite3"
)

func DocuDupsMovies(docuLibrary, movieLibrary content.Library) error {
	// f := GetFlags()make

	docus, err := docuLibrary.MoviesSource(func(f string, err error) {
		c.Printf("  %s --> <red>ERROR:</>%s</>\n", path.Base(f), err)
	})
	if err != nil {
		return fmt.Errorf("error getting docus: %w", err)
	}
	sort.Slice(docus, func(i, j int) bool {
		return docus[i].Letter+"/"+docus[i].DstFolder < docus[j].Letter+"/"+docus[j].DstFolder
	})

	movies, err := movieLibrary.MoviesSource(func(f string, err error) {
		c.Printf("  %s --> <red>ERROR:</>%s</>\n", path.Base(f), err)
	})
	if err != nil {
		return fmt.Errorf("error getting movies: %w", err)
	}
	sort.Slice(movies, func(i, j int) bool {
		return movies[i].Letter+"/"+movies[i].DstFolder < movies[j].Letter+"/"+movies[j].DstFolder
	})

	for _, d := range docus {
		for _, m := range movies {
			if d.DstFolder == m.DstFolder {
				// found a match,
				fmt.Println("Match found: ", d.DstFolder)
			}
		}
	}

	return nil
}
