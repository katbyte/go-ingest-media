package cli

import (
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"sort"

	c "github.com/gookit/color"
	"github.com/katbyte/go-ingest-media/lib/content"
	"github.com/katbyte/go-ingest-media/lib/ktio"
	_ "github.com/mattn/go-sqlite3"
)

// FindAndCombineDocu scans the documentary library and checks if each documentary
// also exists in the movies library. For duplicates, it compares videos and asks
// the user to choose which to keep, or to move the movie copy to the documentary folder.
func FindAndCombineDocu(docuLibrary, movieLibrary *content.Library) error {
	f := GetFlags()

	// Get documentaries using Movies() helper
	docuMovies, err := docuLibrary.Movies(func(folder string, err error) {
		c.Printf("  %s --> <red>ERROR:</>: %s\n", path.Base(folder), err)
	})
	if err != nil {
		return fmt.Errorf("error loading documentaries: %w", err)
	}

	// Build map of documentary names -> Movie objects
	docuMap := make(map[string]*content.Movie)
	for i := range docuMovies {
		docuMap[docuMovies[i].Folder] = &docuMovies[i]
	}

	// Get movies using Movies() helper
	movieList, err := movieLibrary.Movies(func(folder string, err error) {
		c.Printf("  %s --> <red>ERROR:</>: %s\n", path.Base(folder), err)
	})
	if err != nil {
		return fmt.Errorf("error loading movies: %w", err)
	}

	// Build map of movie names -> Movie objects
	movieMap := make(map[string]*content.Movie)
	for i := range movieList {
		movieMap[movieList[i].Folder] = &movieList[i]
	}

	// Get sorted list of documentary names for consistent ordering
	docuNames := make([]string, 0, len(docuMap))
	for name := range docuMap {
		docuNames = append(docuNames, name)
	}
	sort.Strings(docuNames)

	matchCount := 0
	for i, docuName := range docuNames {
		movieEntry, existsInMovies := movieMap[docuName]
		if !existsInMovies {
			continue
		}

		matchCount++
		docuEntry := docuMap[docuName]

		c.Printf("\n<yellow>%d/%d</> <white>%s</>\n", i+1, len(docuNames), docuName)
		c.Printf("  <cyan>DOCU:</> %s\n", docuEntry.Path())
		c.Printf("  <magenta>MOVIE:</> %s\n", movieEntry.Path())

		// Load video info for both
		if err := docuEntry.LoadVideos(); err != nil {
			c.Printf("  <red>ERROR:</> loading docu videos: %s\n", err)
			continue
		}

		if err := movieEntry.LoadVideos(); err != nil {
			c.Printf("  <red>ERROR:</> loading movie videos: %s\n", err)
			continue
		}

		if len(docuEntry.Videos) == 0 && len(movieEntry.Videos) == 0 {
			c.Printf("  <yellow>WARNING:</> no videos in either folder\n")
			continue
		}

		// Check if videos are the same - if so, auto-keep documentary
		if len(docuEntry.Videos) == 1 && len(movieEntry.Videos) == 1 {
			if docuEntry.Videos[0].IsBasicallyTheSameTo(movieEntry.Videos[0]) {
				c.Printf("  <green>SAME</> - keeping documentary, deleting movie copy\n")
				if err := movieEntry.DeleteFolder(f.Confirm, 4); err != nil {
					c.Printf("  <red>ERROR:</> deleting movie folder: %s\n", err)
				}
				continue
			}
		}

		// Display comparison table
		if len(docuEntry.Videos) > 0 || len(movieEntry.Videos) > 0 {
			headers := []string{}
			videos := []content.VideoFile{}

			for i := range docuEntry.Videos {
				headers = append(headers, fmt.Sprintf("Docu %d", i+1))
				videos = append(videos, docuEntry.Videos[i])
			}
			for i := range movieEntry.Videos {
				headers = append(headers, fmt.Sprintf("Movie %d", i+1))
				videos = append(videos, movieEntry.Videos[i])
			}

			if len(videos) > 0 {
				RenderVideoComparisonTable(4, headers, videos)
			}
		}

		// Ask user what to do
		c.Printf("  Actions: keep <cyan>[d]ocu</> | keep <magenta>[m]ovie</> | [s]kip | [q]uit: ")
		selection, err := ktio.GetSelection('d', 'm', 's', 'q')
		fmt.Println()
		if err != nil {
			c.Printf("  <red>ERROR:</> %s\n", err)
			continue
		}

		switch selection {
		case 'd':
			// Keep documentary, delete movie copy
			c.Printf("  <cyan>Keeping documentary, deleting movie copy...</>\n")
			if err := movieEntry.DeleteFolder(f.Confirm, 4); err != nil {
				c.Printf("  <red>ERROR:</> deleting movie folder: %s\n", err)
			}

		case 'm':
			// Keep movie, delete existing docu and move movie to documentary folder
			c.Printf("  <magenta>Deleting existing documentary...</>\n")
			if err := docuEntry.DeleteFolder(f.Confirm, 4); err != nil {
				c.Printf("  <red>ERROR:</> deleting docu folder: %s\n", err)
				continue
			}
			// Move movie to documentary folder
			destPath := filepath.Join(docuLibrary.Path, docuName)
			c.Printf("  <magenta>Moving movie to documentary folder...</>\n")
			if err := ktio.RunCommand(4, f.Confirm, "mv", "-v", movieEntry.Path(), destPath); err != nil {
				c.Printf("  <red>ERROR:</> moving movie folder: %s\n", err)
			}

		case 's':
			c.Printf("  <darkGray>Skipping...</>\n")
			continue

		case 'q':
			return errors.New("quitting")
		}
	}

	if matchCount == 0 {
		c.Printf("\n<green>No duplicates found between documentary and movies libraries.</>\n")
	} else {
		c.Printf("\n<yellow>Processed %d duplicates.</>\n", matchCount)
	}

	return nil
}
