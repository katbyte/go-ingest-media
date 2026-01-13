package cli

import (
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"

	c "github.com/gookit/color"
	"github.com/katbyte/go-ingest-media/lib/content"
	"github.com/katbyte/go-ingest-media/lib/ktio"
	_ "github.com/mattn/go-sqlite3"
)

// printSeriesPaths prints source --> dest series paths with color formatting:
// shared path parts in gray, differing parts in white, and folder names colored
func printSeriesPaths(srcPath, destPath, srcColor, destColor string) {
	srcParts := strings.Split(srcPath, "/")
	destParts := strings.Split(destPath, "/")

	// Find common prefix length
	commonLen := 0
	for i := 0; i < len(srcParts) && i < len(destParts); i++ {
		if srcParts[i] == destParts[i] {
			commonLen = i + 1
		} else {
			break
		}
	}

	commonPath := strings.Join(srcParts[:commonLen], "/")
	srcDiff := strings.Join(srcParts[commonLen:], "/")
	destDiff := strings.Join(destParts[commonLen:], "/")

	srcFolder := path.Base(srcPath)
	destFolder := path.Base(destPath)

	// Remove folder from diff for separate coloring
	srcDiffNoFolder := strings.TrimSuffix(srcDiff, srcFolder)
	destDiffNoFolder := strings.TrimSuffix(destDiff, destFolder)

	// Print: common/srcDiff/srcFolder --> common/destDiff/destFolder
	c.Printf("<gray>%s/</>", commonPath)
	if srcDiffNoFolder != "" {
		c.Printf("<white>%s</>", srcDiffNoFolder)
	}
	c.Printf("<%s>%s</>", srcColor, srcFolder)
	c.Printf(" <gray>--></> ")
	c.Printf("<gray>%s/</>", commonPath)
	if destDiffNoFolder != "" {
		c.Printf("<white>%s</>", destDiffNoFolder)
	}
	c.Printf("<%s>%s</>\n", destColor, destFolder)
}

// FindAndCombineDocuSeries scans the docuseries library and checks if each series
// also exists in the TV library. For duplicates, it compares seasons/episodes and
// processes episode-by-episode to determine which to keep.
func FindAndCombineDocuSeries(docuseriesLibrary, tvLibrary *content.Library) error {
	// Get docuseries using Series() helper
	docuSeriesList, err := docuseriesLibrary.Series(func(folder string, err error) {
		c.Printf("  %s --> <red>ERROR:</>: %s\n", path.Base(folder), err)
	})
	if err != nil {
		return fmt.Errorf("error loading docuseries: %w", err)
	}

	// Build map of docuseries names -> Series objects
	docuMap := make(map[string]*content.Series)
	for i := range docuSeriesList {
		docuMap[docuSeriesList[i].Folder] = &docuSeriesList[i]
	}

	// Get TV series using Series() helper
	tvSeriesList, err := tvLibrary.Series(func(folder string, err error) {
		c.Printf("  %s --> <red>ERROR:</>: %s\n", path.Base(folder), err)
	})
	if err != nil {
		return fmt.Errorf("error loading tv series: %w", err)
	}

	// Build map of TV series names -> Series objects
	tvMap := make(map[string]*content.Series)
	for i := range tvSeriesList {
		tvMap[tvSeriesList[i].Folder] = &tvSeriesList[i]
	}

	// Get sorted list of docuseries names for consistent ordering
	docuNames := make([]string, 0, len(docuMap))
	for name := range docuMap {
		docuNames = append(docuNames, name)
	}
	sort.Strings(docuNames)

	matchCount := 0
	for i, docuName := range docuNames {
		tvEntry, existsInTV := tvMap[docuName]
		if !existsInTV {
			continue
		}

		matchCount++
		docuEntry := docuMap[docuName]

		c.Printf("\n<yellow>%d/%d</> ", i+1, len(docuNames))
		printSeriesPaths(tvEntry.Path(), docuEntry.Path(), "cyan", "magenta")

		// Load seasons for both
		if err := docuEntry.LoadSeasons(); err != nil {
			c.Printf("  <red>ERROR:</> loading docuseries seasons: %s\n", err)
			continue
		}

		if err := tvEntry.LoadSeasons(); err != nil {
			c.Printf("  <red>ERROR:</> loading tv seasons: %s\n", err)
			continue
		}

		// Compare seasons
		docuSeasonCount := len(docuEntry.Seasons)
		tvSeasonCount := len(tvEntry.Seasons)

		c.Printf("  <darkGray>Docuseries: %d seasons, TV: %d seasons</>\n", docuSeasonCount, tvSeasonCount)

		// Process episode by episode
		if processSeriesEpisodes(docuEntry, tvEntry) {
			return errors.New("quitting")
		}

		// Clean up empty TV folders
		cleanupEmptySeriesFolders(tvEntry)
	}

	if matchCount == 0 {
		c.Printf("\n<green>No duplicates found between docuseries and TV libraries.</>\n")
	} else {
		c.Printf("\n<yellow>Processed %d duplicates.</>\n", matchCount)
	}

	return nil
}

// processSeriesEpisodes goes through episodes one by one and handles duplicates
// Returns true if user chose to quit
func processSeriesEpisodes(docu, tv *content.Series) bool {
	f := GetFlags()

	// Get all unique season numbers from both
	allSeasons := make(map[int]bool)
	for s := range docu.Seasons {
		allSeasons[s] = true
	}
	for s := range tv.Seasons {
		allSeasons[s] = true
	}

	seasonNums := make([]int, 0, len(allSeasons))
	for s := range allSeasons {
		seasonNums = append(seasonNums, s)
	}
	sort.Ints(seasonNums)

	for _, seasonNum := range seasonNums {
		docuSeason, docuExists := docu.Seasons[seasonNum]
		tvSeason, tvExists := tv.Seasons[seasonNum]

		// Season only in TV - move entire season folder to docuseries
		if !docuExists && tvExists {
			c.Printf("    season <magenta>%d</> - only in TV (%d eps) - moving to docuseries\n", seasonNum, len(tvSeason.Episodes))
			destPath := docu.Path() + "/"
			if err := ktio.RunCommand(6, f.Confirm, "mv", "-v", tvSeason.Path, destPath); err != nil {
				c.Printf("      <red>ERROR:</> moving TV season: %s\n", err)
			}
			continue
		}

		// Season only in docuseries - nothing to do
		if docuExists && !tvExists {
			continue
		}

		c.Printf("    season <yellow>%d</>: <darkGray>docu=%d eps, tv=%d eps</>\n",
			seasonNum, len(docuSeason.Episodes), len(tvSeason.Episodes))

		// Get all unique episode numbers from both
		allEps := make(map[int]bool)
		for e := range docuSeason.Episodes {
			allEps[e] = true
		}
		for e := range tvSeason.Episodes {
			allEps[e] = true
		}

		epNums := make([]int, 0, len(allEps))
		for e := range allEps {
			epNums = append(epNums, e)
		}
		sort.Ints(epNums)

		for _, epNum := range epNums {
			docuEp, docuEpExists := docuSeason.Episodes[epNum]
			tvEp, tvEpExists := tvSeason.Episodes[epNum]

			// Skip episodes only in docuseries (nothing to do)
			if !tvEpExists {
				continue
			}

			// Episode only in TV - move to docuseries
			if !docuEpExists {
				c.Printf("      S%02dE%02d: <magenta>TV only</> - moving to docuseries\n", seasonNum, epNum)
				for _, v := range tvEp.Videos {
					destPath := docuSeason.Path + "/"
					if err := ktio.RunCommand(8, f.Confirm, "mv", "-v", v.Path, destPath); err != nil {
						c.Printf("        <red>ERROR:</> moving TV video: %s\n", err)
					}
				}
				continue
			}

			// Both exist - compare videos
			if len(docuEp.Videos) == 0 && len(tvEp.Videos) == 0 {
				c.Printf("      S%02dE%02d: <yellow>no videos in either</>\n", seasonNum, epNum)
				continue
			}

			// Check if same
			if len(docuEp.Videos) == 1 && len(tvEp.Videos) == 1 {
				if docuEp.Videos[0].IsBasicallyTheSameTo(tvEp.Videos[0]) {
					c.Printf("      S%02dE%02d: <green>SAME</> - deleting TV version\n", seasonNum, epNum)
					for _, v := range tvEp.Videos {
						if err := ktio.RunCommand(8, f.Confirm, "rm", "-v", v.Path); err != nil {
							c.Printf("        <red>ERROR:</> deleting TV video: %s\n", err)
						}
					}
					continue
				}
			}

			// Different - show comparison and ask
			c.Printf("      S%02dE%02d: <yellow>different</>\n", seasonNum, epNum)

			headers := []string{}
			videos := []content.VideoFile{}

			for i := range docuEp.Videos {
				headers = append(headers, fmt.Sprintf("Docu %d", i+1))
				videos = append(videos, docuEp.Videos[i])
			}
			for i := range tvEp.Videos {
				headers = append(headers, fmt.Sprintf("TV %d", i+1))
				videos = append(videos, tvEp.Videos[i])
			}

			if len(videos) > 0 {
				RenderVideoComparisonTable(8, headers, videos)
			}

			// Ask what to do
			c.Printf("        keep [d]ocu | keep [t]v | [s]kip | [q]uit: ")
			selection, err := ktio.GetSelection('d', 't', 's', 'q')
			fmt.Println()
			if err != nil {
				c.Printf("        <red>ERROR:</> %s\n", err)
				continue
			}

			switch selection {
			case 'd':
				// Delete TV version
				for _, v := range tvEp.Videos {
					if err := ktio.RunCommand(8, f.Confirm, "rm", "-v", v.Path); err != nil {
						c.Printf("        <red>ERROR:</> deleting TV video: %s\n", err)
					}
				}
			case 't':
				// Delete docuseries version and move TV to docu
				for _, v := range docuEp.Videos {
					if err := ktio.RunCommand(8, f.Confirm, "rm", "-v", v.Path); err != nil {
						c.Printf("        <red>ERROR:</> deleting docu video: %s\n", err)
					}
				}
				for _, v := range tvEp.Videos {
					destPath := docuSeason.Path + "/"
					if err := ktio.RunCommand(8, f.Confirm, "mv", "-v", v.Path, destPath); err != nil {
						c.Printf("        <red>ERROR:</> moving TV video: %s\n", err)
					}
				}
			case 's':
				// skip
			case 'q':
				return true
			}
		}
	}

	return false
}

// cleanupEmptySeriesFolders cleans up empty season and series folders in TV
func cleanupEmptySeriesFolders(tv *content.Series) {
	f := GetFlags()

	// Skip if series folder no longer exists
	if !ktio.PathExists(tv.Path()) {
		return
	}

	// Clean up empty season folders
	for _, season := range tv.Seasons {
		if !ktio.PathExists(season.Path) {
			continue
		}

		if err := ktio.DeleteIfEmptyOrOnlyNfo(season.Path, f.Confirm, 6); err != nil {
			c.Printf("      <red>ERROR:</> cleaning up TV season: %s\n", err)
		}
	}

	// Clean up empty series folder
	if err := ktio.DeleteIfEmptyOrOnlyNfo(tv.Path(), f.Confirm, 4); err != nil {
		c.Printf("    <red>ERROR:</> cleaning up TV series folder: %s\n", err)
	}
}
