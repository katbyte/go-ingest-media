package cli

import (
	"errors"
	"fmt"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"

	c "github.com/gookit/color"
	"github.com/katbyte/go-ingest-media/lib/content"
	"github.com/katbyte/go-ingest-media/lib/ktio"
	_ "github.com/mattn/go-sqlite3"
)

// clearLine overwrites the current line with spaces
func clearLine() {
	fmt.Printf("\r%s\r", strings.Repeat(" ", 120))
}

// queryAI shells out to agy to ask about a title and returns the response
func queryAI(name string, isSeries bool) (string, error) {
	kind := "movie"
	if isSeries {
		kind = "series"
	}

	prompt := fmt.Sprintf("Why is the %s '%s' classified as a documentary? Why might it not be considered one? Please answer concisely in a single short paragraph.", kind, name)

	cmd := exec.Command("agy", "-p", prompt) //nolint:gosec
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("error running agy: %w\n%s", err, string(output))
	}

	return strings.TrimSpace(string(output)), nil
}

// ExtractDocuMovies scans the movies library for documentaries via NFO files
// and moves them to the documentary torrent-sorted import folder (m.docu)
func ExtractDocuMovies(movieLib, docuImportLib *content.Library) error {
	f := GetFlags()

	// Get all movies from the library
	movies, err := movieLib.Movies(func(folder string, err error) {
		c.Printf("  %s --> <red>ERROR:</> %s\n", path.Base(folder), err)
	})
	if err != nil {
		return fmt.Errorf("error loading movies: %w", err)
	}

	sort.Slice(movies, func(i, j int) bool {
		return movies[i].Letter+"/"+movies[i].Folder < movies[j].Letter+"/"+movies[j].Folder
	})

	found := 0
	moved := 0
	total := len(movies)

	for i, m := range movies {
		// Show progress for current folder
		c.Printf("\r<darkGray>%d/%d scanning %s/%s</> ", i+1, total, m.Letter, m.Folder)

		// Find NFO file in movie folder
		nfoPath, err := content.FindNfoFile(m.Path())
		if err != nil {
			clearLine()
			c.Printf("<darkGray>%d/%d</> <white>%s</> --> <red>ERROR:</> finding nfo: %s\n", i+1, total, m.Folder, err)
			continue
		}
		if nfoPath == "" {
			continue // no nfo file
		}

		// Parse NFO
		nfo, err := content.ReadNfo(nfoPath)
		if err != nil {
			clearLine()
			c.Printf("<darkGray>%d/%d</> <white>%s</> --> <red>ERROR:</> reading nfo: %s\n", i+1, total, m.Folder, err)
			continue
		}

		if !nfo.IsDocumentary() {
			continue
		}

		found++
		clearLine()

		// Print documentary info
		c.Printf("<cyan>%d</>/<darkGray>%d</> <white>%s</> <darkGray>%s</>\n", i+1, total, m.Folder, m.Path())
		if url := nfo.TmdbURL(false); url != "" {
			c.Printf("  <darkGray>tmdb:   </> %s\n", url)
		}
		c.Printf("  <darkGray>genres: </> %s\n", strings.Join(nfo.Genres, ", "))
		if nfo.Tagline != "" {
			c.Printf("  <darkGray>tagline:</> %s\n", nfo.Tagline)
		}
		if nfo.Outline != "" {
			c.Printf("  <darkGray>outline:</> %s\n", nfo.Outline)
		}
		if nfo.Plot != "" {
			c.Printf("  <darkGray>plot:   </> %s\n", nfo.Plot)
		}

		destPath := filepath.Join(docuImportLib.Path, m.Folder)
		c.Printf("  --> <green>%s</>\n", destPath)

		// Selection loop (re-asks after AI query)
		for {
			c.Printf("  [m]ove/[a]ccept | [s]kip | [q]uery ai | e[x]it: ")
			selection, err := ktio.GetSelection('m', 'a', 's', 'q', 'x')
			fmt.Println()
			if err != nil {
				c.Printf("  <red>ERROR:</> %s\n", err)
				break
			}

			switch selection {
			case 'm', 'a':
				if err := m.MoveFolder(destPath, f.Prompt, 4); err != nil {
					c.Printf("  <red>ERROR:</> moving folder: %s\n", err)
				} else {
					moved++
				}
			case 's':
				c.Printf("  <darkGray>skipping... removing documentary from genres...</>")
				if err := content.RemoveDocumentaryGenre(nfoPath); err != nil {
					c.Printf(" <red>ERROR:</> %s\n", err)
				} else {
					c.Printf(" <darkGray>done</>\n")
				}
			case 'q':
				c.Printf("  <darkGray>querying AI...</>\n")
				result, err := queryAI(m.Folder, false)
				if err != nil {
					c.Printf("  <red>ERROR:</> %s\n", err)
				} else {
					c.Printf("  <lightYellow>AI:</> %s\n", result)
				}
				continue // re-ask
			case 'x':
				return errors.New("quitting")
			}
			break
		}
		fmt.Println()
	}

	clearLine()
	c.Printf("<yellow>Found %d documentaries, moved %d</> out of %d movies\n", found, moved, total)

	return nil
}

// ExtractDocuSeries scans the TV library for docuseries via NFO files
// and moves them to the docuseries torrent-sorted import folder (s.docu)
func ExtractDocuSeries(tvLib, docuImportLib *content.Library) error {
	f := GetFlags()

	// Get all series from the library
	seriesList, err := tvLib.Series(func(folder string, err error) {
		c.Printf("  %s --> <red>ERROR:</> %s\n", path.Base(folder), err)
	})
	if err != nil {
		return fmt.Errorf("error loading series: %w", err)
	}

	sort.Slice(seriesList, func(i, j int) bool {
		return seriesList[i].Letter+"/"+seriesList[i].Folder < seriesList[j].Letter+"/"+seriesList[j].Folder
	})

	found := 0
	moved := 0
	total := len(seriesList)

	for i, s := range seriesList {
		// Show progress for current folder
		c.Printf("\r<darkGray>%d/%d scanning %s/%s</> ", i+1, total, s.Letter, s.Folder)

		// Find NFO file in series folder
		nfoPath, err := content.FindNfoFile(s.Path())
		if err != nil {
			clearLine()
			c.Printf("<darkGray>%d/%d</> <white>%s</> --> <red>ERROR:</> finding nfo: %s\n", i+1, total, s.Folder, err)
			continue
		}
		if nfoPath == "" {
			continue // no nfo file
		}

		// Parse NFO
		nfo, err := content.ReadNfo(nfoPath)
		if err != nil {
			clearLine()
			c.Printf("<darkGray>%d/%d</> <white>%s</> --> <red>ERROR:</> reading nfo: %s\n", i+1, total, s.Folder, err)
			continue
		}

		if !nfo.IsDocumentary() {
			continue
		}

		found++
		clearLine()

		// Print documentary info
		c.Printf("<cyan>%d</>/<darkGray>%d</> <white>%s</> <darkGray>%s</>\n", i+1, total, s.Folder, s.Path())
		if url := nfo.TmdbURL(true); url != "" {
			c.Printf("  <darkGray>tmdb:   </> %s\n", url)
		}
		c.Printf("  <darkGray>genres: </> %s\n", strings.Join(nfo.Genres, ", "))
		if nfo.Tagline != "" {
			c.Printf("  <darkGray>tagline:</> %s\n", nfo.Tagline)
		}
		if nfo.Outline != "" {
			c.Printf("  <darkGray>outline:</> %s\n", nfo.Outline)
		}
		if nfo.Plot != "" {
			c.Printf("  <darkGray>plot:   </> %s\n", nfo.Plot)
		}

		destPath := filepath.Join(docuImportLib.Path, s.Folder)
		c.Printf("  --> <green>%s</>\n", destPath)

		// Selection loop (re-asks after AI query)
		for {
			c.Printf("  [m]ove/[a]ccept | [s]kip | [q]uery ai | e[x]it: ")
			selection, err := ktio.GetSelection('m', 'a', 's', 'q', 'x')
			fmt.Println()
			if err != nil {
				c.Printf("  <red>ERROR:</> %s\n", err)
				break
			}

			switch selection {
			case 'm', 'a':
				if err := s.MoveFolder(destPath, f.Prompt, 4); err != nil {
					c.Printf("  <red>ERROR:</> moving folder: %s\n", err)
				} else {
					moved++
				}
			case 's':
				c.Printf("  <darkGray>skipping... removing documentary from genres...</>")
				if err := content.RemoveDocumentaryGenre(nfoPath); err != nil {
					c.Printf(" <red>ERROR:</> %s\n", err)
				} else {
					c.Printf(" <darkGray>done</>\n")
				}
			case 'q':
				c.Printf("  <darkGray>querying AI...</>\n")
				result, err := queryAI(s.Folder, true)
				if err != nil {
					c.Printf("  <red>ERROR:</> %s\n", err)
				} else {
					c.Printf("  <lightYellow>AI:</> %s\n", result)
				}
				continue // re-ask
			case 'x':
				return errors.New("quitting")
			}
			break
		}
		fmt.Println()
	}

	clearLine()
	c.Printf("<yellow>Found %d docuseries, moved %d</> out of %d series\n", found, moved, total)

	return nil
}
