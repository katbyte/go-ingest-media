package cli

import (
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"sort"
	"strings"

	c "github.com/gookit/color"
	"github.com/katbyte/go-ingest-media/lib/content"
	"github.com/katbyte/go-ingest-media/lib/ktio"
)

func FindAndCombineAnime(animeLib, stdLib *content.Library, libType content.LibraryType) error {
	f := GetFlags()

	// Load Anime Folders
	var animeItems []content.Content
	var stdItems []content.Content
	var err error

	if libType == content.LibraryTypeSeries {
		var s1, s2 []content.Series
		s1, err = animeLib.Series(func(folder string, err error) {
			c.Printf("  %s --> <red>ERROR:</>: %s\n", path.Base(folder), err)
		})
		for _, s := range s1 {
			animeItems = append(animeItems, s.Content)
		}
		s2, err = stdLib.Series(func(folder string, err error) {
			c.Printf("  %s --> <red>ERROR:</>: %s\n", path.Base(folder), err)
		})
		for _, s := range s2 {
			stdItems = append(stdItems, s.Content)
		}
	} else {
		var m1, m2 []content.Movie
		m1, err = animeLib.Movies(func(folder string, err error) {
			c.Printf("  %s --> <red>ERROR:</>: %s\n", path.Base(folder), err)
		})
		for _, m := range m1 {
			animeItems = append(animeItems, m.Content)
		}
		m2, err = stdLib.Movies(func(folder string, err error) {
			c.Printf("  %s --> <red>ERROR:</>: %s\n", path.Base(folder), err)
		})
		for _, m := range m2 {
			stdItems = append(stdItems, m.Content)
		}
	}

	if err != nil {
		return err
	}

	// Build map for standard items by their normalized folder name
	stdMap := make(map[string]content.Content)
	for _, item := range stdItems {
		// Use DestPathInWithRename to get the normalized target name it would have
		normPath, _ := item.DestPathInWithRename(stdLib, libType)
		normFolder := path.Base(normPath)
		stdMap[strings.ToLower(normFolder)] = item
		// Also add raw folder name just in case
		stdMap[strings.ToLower(item.Folder)] = item
	}

	// Sort anime items
	sort.Slice(animeItems, func(i, j int) bool {
		return animeItems[i].Folder < animeItems[j].Folder
	})

	var dups []struct {
		anime content.Content
		std   content.Content
	}

	for _, item := range animeItems {
		normPath, _ := item.DestPathInWithRename(animeLib, libType)
		normFolder := path.Base(normPath)

		if matched, ok := stdMap[strings.ToLower(normFolder)]; ok {
			dups = append(dups, struct{ anime, std content.Content }{item, matched})
		} else if matched, ok := stdMap[strings.ToLower(item.Folder)]; ok {
			dups = append(dups, struct{ anime, std content.Content }{item, matched})
		}
	}

	if len(dups) == 0 {
		c.Printf("  <green>No local duplicates found.</>\n")
		return nil
	}

	c.Printf("\n<yellow>Found %d duplicates.</> Starting review...\n\n", len(dups))

	for i, dup := range dups {
		c.Printf("<yellow>[%d/%d]</> <white>%s</>\n", i+1, len(dups), dup.anime.Folder)
		c.Printf("  <cyan>A:</> %s\n", dup.std.Path())
		c.Printf("  <magenta>B:</> %s\n", dup.anime.Path())

		infoA := ScanFolder(dup.std.Path())
		infoB := ScanFolder(dup.anime.Path())

		RenderFolderComparison(4, infoA, infoB, "A (Standard)", "B (Anime)")

		for {
			c.Printf("  keep [a] | keep [b] | [c]ompare videos | [s]kip | e[x]it: ")
			selection, err := ktio.GetSelection('a', 'b', 'c', 's', 'x')
			fmt.Println()
			if err != nil {
				c.Printf("  <red>ERROR:</> %s\n", err)
				break
			}

			if selection == 'x' {
				return errors.New("quitting")
			}
			if selection == 's' {
				c.Printf("  <darkGray>Skipping...</>\n\n")
				break
			}

			if selection == 'c' {
				stdVideos, errA := content.VideosInPath(dup.std.Path())
				if errA != nil {
					c.Printf("   <red>Error loading videos A:</> %v\n", errA)
					continue
				}
				animeVideos, errB := content.VideosInPath(dup.anime.Path())
				if errB != nil {
					c.Printf("   <red>Error loading videos B:</> %v\n", errB)
					continue
				}

				headers := []string{}
				videos := []content.VideoFile{}

				for i := range stdVideos {
					headers = append(headers, fmt.Sprintf("A-%d", i+1))
					videos = append(videos, stdVideos[i])
				}
				for i := range animeVideos {
					headers = append(headers, fmt.Sprintf("B-%d", i+1))
					videos = append(videos, animeVideos[i])
				}

				if len(videos) > 0 {
					RenderVideoComparisonTable(4, headers, videos)
				} else {
					c.Printf("   <red>No videos found to compare.</>\n")
				}
				continue
			}

			if selection == 'a' {
				c.Printf("  <cyan>Keeping standard version, moving to anime library...</>\n")
				if err := dup.anime.DeleteFolder(f.Prompt, 4); err != nil {
					c.Printf("  <red>ERROR:</> deleting anime folder: %s\n", err)
					break
				}
				destPath := filepath.Join(animeLib.Path, dup.anime.Folder)
				if err := ktio.RunCommand(4, f.Prompt, "mv", "-v", dup.std.Path(), destPath); err != nil {
					c.Printf("  <red>ERROR:</> moving standard folder: %s\n", err)
				}
				break
			}

			if selection == 'b' {
				c.Printf("  <magenta>Keeping anime version, deleting standard folder...</>\n")
				if err := dup.std.DeleteFolder(f.Prompt, 4); err != nil {
					c.Printf("  <red>ERROR:</> deleting standard folder: %s\n", err)
				}
				break
			}
		}
		fmt.Println()
	}

	return nil
}
