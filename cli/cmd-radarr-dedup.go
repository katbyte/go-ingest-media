package cli

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	c "github.com/gookit/color"
	"github.com/katbyte/go-ingest-media/lib/content"
	"github.com/katbyte/go-ingest-media/lib/ktio"
	"github.com/katbyte/go-ingest-media/lib/radarr"
)

var yearRegex = regexp.MustCompile(`\((\d{4})\)\s*$`)

// parseFolderName extracts title and year from a folder name like "Movie Title (2020)"
func parseFolderName(name string) (string, string) {
	matches := yearRegex.FindStringSubmatch(name)
	if len(matches) < 2 {
		return name, ""
	}

	title := strings.TrimSpace(yearRegex.ReplaceAllString(name, ""))
	return title, matches[1]
}

// dupItem represents a duplicate found by the radarr-dedup scan
type dupItem struct {
	unmappedName string        // display name of the unmapped folder
	unmappedPath string        // full path of the unmapped folder on disk
	matchedMovie *radarr.Movie // the existing movie in Radarr this matches to
	matchedPath  string        // full path of the existing movie on disk
}

// unmappedEntry is used for the concurrent scan phase
type unmappedEntry struct {
	name string
	path string // radarr-relative path
}

// DedupRadarr connects to Radarr and identifies folders on disk that are duplicates of existing movies
func DedupRadarr(radarrUrl, apiKey, basePath string, pathMaps []string) error {
	if radarrUrl == "" || apiKey == "" {
		return fmt.Errorf("radarr url and api key are required (--radarr-url / --radarr-api-key or RADARR_URL / RADARR_API_KEY)")
	}

	// Parse path maps - supports both repeated flags and comma-separated
	// e.g. --radarr-path-map "documentary=docu,anime=anime/movies"
	// or   --radarr-path-map "documentary=docu" --radarr-path-map "anime=anime/movies"
	type pathMapEntry struct{ from, to string }
	var pathMapPairs []pathMapEntry
	for _, pm := range pathMaps {
		for _, entry := range strings.Split(pm, ",") {
			entry = strings.TrimSpace(entry)
			parts := strings.SplitN(entry, "=", 2)
			if len(parts) == 2 {
				pathMapPairs = append(pathMapPairs, pathMapEntry{from: parts[0], to: parts[1]})
			}
		}
	}

	resolveLocalPath := func(radarrPath string) string {
		p := radarrPath
		if basePath != "" {
			p = filepath.Join(basePath, p)
		}
		for _, pm := range pathMapPairs {
			p = strings.ReplaceAll(p, "/"+pm.from+"/", "/"+pm.to+"/")
		}
		return p
	}

	client := radarr.NewClient(radarrUrl, apiKey)

	// Show a spinner while loading (single large API call)
	fmt.Printf("Loading movies from Radarr at %s", radarrUrl)
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				fmt.Print(".")
			}
		}
	}()

	movies, err := client.GetMovies()
	close(done)
	fmt.Println()
	if err != nil {
		return fmt.Errorf("failed to get movies: %w", err)
	}
	c.Printf("<green>Loaded %d movies from Radarr</>\n", len(movies))

	existingByTmdb := make(map[int]*radarr.Movie)
	for i := range movies {
		existingByTmdb[movies[i].TmdbId] = &movies[i]
	}

	// Step 2: Get root folders (with unmapped folders on disk)
	c.Printf("<darkGray>Loading root folders...</>\n")
	rootFolders, err := client.GetRootFolders()
	if err != nil {
		return fmt.Errorf("failed to get root folders: %w", err)
	}

	// Collect all unmapped folders into a flat list
	var allUnmapped []unmappedEntry
	for _, rf := range rootFolders {
		for _, uf := range rf.UnmappedFolders {
			allUnmapped = append(allUnmapped, unmappedEntry{name: uf.Name, path: uf.Path})
		}
	}
	c.Printf("<green>Found %d root folders with %d total unmapped folders</>\n", len(rootFolders), len(allUnmapped))

	// Step 3: Scan for duplicates using concurrent workers
	c.Printf("<darkGray>Scanning for duplicates...</>\n\n")

	const numWorkers = 8

	type workItem struct {
		index int
		entry unmappedEntry
	}

	workCh := make(chan workItem, len(allUnmapped))
	for i, uf := range allUnmapped {
		workCh <- workItem{index: i, entry: uf}
	}
	close(workCh)

	var mu sync.Mutex
	var dups []dupItem
	var processed atomic.Int32

	var wg sync.WaitGroup
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for work := range workCh {
				uf := work.entry
				cur := int(processed.Add(1))

				localPath := resolveLocalPath(uf.path)

				// First try a quick local match by folder name against existing movies
				title, year := parseFolderName(uf.name)
				var matched *radarr.Movie

				for i := range movies {
					m := &movies[i]
					mFolder := filepath.Base(m.Path)
					if strings.EqualFold(uf.name, mFolder) {
						matched = m
						break
					}
					if year != "" && m.Year > 0 && fmt.Sprintf("%d", m.Year) == year && strings.EqualFold(strings.TrimSpace(title), strings.TrimSpace(m.Title)) {
						matched = m
						break
					}
				}

				// If no local match, use Radarr's TMDB lookup
				if matched == nil {
					lookupResults, lookupErr := client.LookupMovie(uf.name)
					if lookupErr != nil {
						c.Printf("  <darkGray>[%d/%d]</> <red>ERROR looking up %s: %s</>\n", cur, len(allUnmapped), uf.name, lookupErr)
						return
					}

					if len(lookupResults) > 0 {
						topMatch := lookupResults[0]
						mu.Lock()
						existing, ok := existingByTmdb[topMatch.TmdbId]
						mu.Unlock()
						if ok {
							matched = existing
						}
					}
				}

				if matched != nil && matched.HasFile {
					matchedPath := resolveLocalPath(matched.Path)

					mu.Lock()
					dups = append(dups, dupItem{
						unmappedName: uf.name,
						unmappedPath: localPath,
						matchedMovie: matched,
						matchedPath:  matchedPath,
					})
					mu.Unlock()

					c.Printf("  <darkGray>[%d/%d]</> <yellow>found:</> %s\n", cur, len(allUnmapped), uf.name)
				} else {
					// Print progress for non-matches periodically
					if cur%50 == 0 || cur == len(allUnmapped) {
						c.Printf("  <darkGray>[%d/%d] scanning...</>\n", cur, len(allUnmapped))
					}
				}
			}
		}()
	}

	wg.Wait()

	if len(dups) == 0 {
		c.Printf("\n<green>No existing duplicates found.</>\n")
		return nil
	}

	c.Printf("\n<yellow>Found %d existing duplicates.</> Starting review...\n\n", len(dups))

	// Step 4: Interactive review loop
	f := GetFlags()
	var deleted int

	for i, dup := range dups {
		c.Printf("<yellow>[%d/%d]</> <white>%s</> (%d)\n", i+1, len(dups), dup.matchedMovie.Title, dup.matchedMovie.Year)
		c.Printf("  <cyan>A:</> %s\n", dup.unmappedPath)
		c.Printf("  <magenta>B:</> %s <darkGray>(Radarr managed)</>\n", dup.matchedPath)

		// Scan both folders for quick comparison
		infoA := ScanFolder(dup.unmappedPath)
		infoB := ScanFolder(dup.matchedPath)

		if !infoA.Exists {
			c.Printf("  <red>Side A does not exist on disk, skipping...</>\n\n")
			continue
		}
		if !infoB.Exists {
			c.Printf("  <red>Side B does not exist on disk, skipping...</>\n\n")
			continue
		}

		RenderFolderComparison(4, infoA, infoB, "A (unmapped)", "B (radarr)")

		// Interactive prompt loop (re-prompts after [c]ompare)
		for {
			c.Printf("  keep [a] | keep [b] | [c]ompare videos | [s]kip | e[x]it: ")
			selection, selErr := ktio.GetSelection('a', 'b', 'c', 's', 'x')
			fmt.Println()
			if selErr != nil {
				c.Printf("  <red>ERROR:</> %s\n", selErr)
				continue
			}

			switch selection {
			case 'a':
				// Keep A, delete B
				c.Printf("  <cyan>Keeping A, deleting B: %s...</>\n", dup.matchedPath)
				if err := ktio.RunCommand(4, f.Prompt, "rm", "-rfv", dup.matchedPath); err != nil {
					c.Printf("  <red>ERROR:</> %s\n", err)
				} else {
					deleted++
				}

			case 'b':
				// Keep B, delete A
				c.Printf("  <magenta>Keeping B, deleting A: %s...</>\n", dup.unmappedPath)
				if err := ktio.RunCommand(4, f.Prompt, "rm", "-rfv", dup.unmappedPath); err != nil {
					c.Printf("  <red>ERROR:</> %s\n", err)
				} else {
					deleted++
				}

			case 'c':
				c.Printf("  <darkGray>Loading video details (ffprobe)...</>\n")

				videosA, errA := content.VideosInPath(dup.unmappedPath)
				videosB, errB := content.VideosInPath(dup.matchedPath)

				if errA != nil {
					c.Printf("  <red>ERROR loading A videos:</> %s\n", errA)
				}
				if errB != nil {
					c.Printf("  <red>ERROR loading B videos:</> %s\n", errB)
				}

				if errA == nil && errB == nil && (len(videosA) > 0 || len(videosB) > 0) {
					headers := []string{}
					allVideos := []content.VideoFile{}

					for j := range videosA {
						headers = append(headers, fmt.Sprintf("A-%d", j+1))
						allVideos = append(allVideos, videosA[j])
					}
					for j := range videosB {
						headers = append(headers, fmt.Sprintf("B-%d", j+1))
						allVideos = append(allVideos, videosB[j])
					}

					RenderVideoComparisonTable(4, headers, allVideos)
					fmt.Println()
				} else if len(videosA) == 0 && len(videosB) == 0 {
					c.Printf("  <yellow>No video files found in either folder.</>\n")
				}

				continue // re-prompt after compare

			case 's':
				c.Printf("  <darkGray>skipped</>\n")

			case 'x':
				c.Printf("\n<green>Exited.</> Deleted %d folders.\n", deleted)
				return errors.New("exit")
			}

			break
		}

		fmt.Println()
	}

	c.Printf("<green>Done.</> Reviewed %d duplicates, deleted %d.\n", len(dups), deleted)
	return nil
}
