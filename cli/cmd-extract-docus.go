package cli

import (
	"errors"
	"fmt"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"

	c "github.com/gookit/color"
	"github.com/katbyte/go-ingest-media/lib/content"
	"github.com/katbyte/go-ingest-media/lib/ktio"
	_ "github.com/mattn/go-sqlite3"
)

// docuItem represents a documentary found by the background scanner
type docuItem struct {
	index   int
	total   int
	folder  string
	path    string
	letter  string
	nfoPath string
	nfo     *content.NfoFile
}

// queryAI shells out to agy to ask about a title and returns the response
func queryAI(name string, isSeries bool) (string, error) {
	kind := "movie"
	if isSeries {
		kind = "series"
	}

	prompt := fmt.Sprintf("Why is the %s '%s' classified as a documentary? Why might it not be considered one? Please answer each question on its own line with a blank inbetween in 1-2 sentences, then on the 3rd line make your own judgement call.", kind, name)

	cmd := exec.Command("agy", "-p", prompt) //nolint:gosec
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("error running agy: %w\n%s", err, string(output))
	}

	return strings.TrimSpace(string(output)), nil
}

// scanMoviesForDocus scans the movie library in the background, sending found documentaries to a channel
func scanMoviesForDocus(lib *content.Library, sb *ktio.StatusBar, logChan chan<- string) (<-chan docuItem, int, error) {
	movies, err := lib.Movies(func(folder string, err error) {
		logChan <- c.Sprintf("  %s --> <red>ERROR:</> %s", path.Base(folder), err)
	})
	if err != nil {
		return nil, 0, fmt.Errorf("error loading movies: %w", err)
	}

	sort.Slice(movies, func(i, j int) bool {
		return movies[i].Letter+"/"+movies[i].Folder < movies[j].Letter+"/"+movies[j].Folder
	})

	total := len(movies)
	ch := make(chan docuItem, 25)

	go func() {
		defer close(ch)

		workCh := make(chan int, 100)
		resultCh := make(chan docuItem, 25)

		var foundCount atomic.Int32

		// Start 10 scan workers
		var wg sync.WaitGroup
		for w := 0; w < 25; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i := range workCh {
					nfoPath, err := content.FindNfoFile(movies[i].Path())
					if err != nil {
						logChan <- c.Sprintf("<darkGray>%d/%d</> <white>%s</> --> <red>ERROR:</> finding nfo: %s", i+1, total, movies[i].Folder, err)
						continue
					}
					if nfoPath == "" {
						continue
					}

					nfo, err := content.ReadNfo(nfoPath)
					if err != nil {
						logChan <- c.Sprintf("<darkGray>%d/%d</> <white>%s</> --> <red>ERROR:</> reading nfo: %s", i+1, total, movies[i].Folder, err)
						continue
					}

					if !nfo.IsDocumentary() {
						continue
					}

					foundCount.Add(1)
					resultCh <- docuItem{
						index:   i,
						total:   total,
						folder:  movies[i].Folder,
						path:    movies[i].Path(),
						letter:  movies[i].Letter,
						nfoPath: nfoPath,
						nfo:     nfo,
					}
				}
			}()
		}

		// Feeder goroutine - dispatches work and updates scan status
		go func() {
			for i := range movies {
				waiting := len(ch)
				sb.UpdateScan(c.Sprintf("<darkGray>scanning</> <cyan>%d</>/<darkGray>%d (found %d/waiting %d)</> <darkGray>%s/%s</>", i+1, total, foundCount.Load(), waiting, movies[i].Letter, movies[i].Folder))
				workCh <- i
			}
			close(workCh)
			wg.Wait()
			close(resultCh)
		}()

		// Forward results to output channel
		for item := range resultCh {
			ch <- item
		}

		sb.UpdateScan(c.Sprintf("<green>scan complete</> <darkGray>(%d movies scanned)</>", total))
	}()

	return ch, total, nil
}

// scanSeriesForDocus scans the TV library in the background, sending found docuseries to a channel
func scanSeriesForDocus(lib *content.Library, sb *ktio.StatusBar, logChan chan<- string) (<-chan docuItem, int, error) {
	seriesList, err := lib.Series(func(folder string, err error) {
		logChan <- c.Sprintf("  %s --> <red>ERROR:</> %s", path.Base(folder), err)
	})
	if err != nil {
		return nil, 0, fmt.Errorf("error loading series: %w", err)
	}

	sort.Slice(seriesList, func(i, j int) bool {
		return seriesList[i].Letter+"/"+seriesList[i].Folder < seriesList[j].Letter+"/"+seriesList[j].Folder
	})

	total := len(seriesList)
	ch := make(chan docuItem, 25)

	go func() {
		defer close(ch)

		workCh := make(chan int, 100)
		resultCh := make(chan docuItem, 25)

		var foundCount atomic.Int32

		// Start 10 scan workers
		var wg sync.WaitGroup
		for w := 0; w < 25; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i := range workCh {
					nfoPath, err := content.FindNfoFile(seriesList[i].Path())
					if err != nil {
						logChan <- c.Sprintf("<darkGray>%d/%d</> <white>%s</> --> <red>ERROR:</> finding nfo: %s", i+1, total, seriesList[i].Folder, err)
						continue
					}
					if nfoPath == "" {
						continue
					}

					nfo, err := content.ReadNfo(nfoPath)
					if err != nil {
						logChan <- c.Sprintf("<darkGray>%d/%d</> <white>%s</> --> <red>ERROR:</> reading nfo: %s", i+1, total, seriesList[i].Folder, err)
						continue
					}

					if !nfo.IsDocumentary() {
						continue
					}

					foundCount.Add(1)
					resultCh <- docuItem{
						index:   i,
						total:   total,
						folder:  seriesList[i].Folder,
						path:    seriesList[i].Path(),
						letter:  seriesList[i].Letter,
						nfoPath: nfoPath,
						nfo:     nfo,
					}
				}
			}()
		}

		// Feeder goroutine - dispatches work and updates scan status
		go func() {
			for i := range seriesList {
				waiting := len(ch)
				sb.UpdateScan(c.Sprintf("<darkGray>scanning</> <cyan>%d</>/<darkGray>%d (found %d/waiting %d)</> <darkGray>%s/%s</>", i+1, total, foundCount.Load(), waiting, seriesList[i].Letter, seriesList[i].Folder))
				workCh <- i
			}
			close(workCh)
			wg.Wait()
			close(resultCh)
		}()

		// Forward results to output channel
		for item := range resultCh {
			ch <- item
		}

		sb.UpdateScan(c.Sprintf("<green>scan complete</> <darkGray>(%d series scanned)</>", total))
	}()

	return ch, total, nil
}

// processDocuItems is the main interactive loop for presenting documentaries to the user
func processDocuItems(docuChan <-chan docuItem, destLib *content.Library, isSeries bool, sb *ktio.StatusBar, logChan <-chan string) (found, moved int, err error) {
	moveQueueChan := make(chan moveAction, 100)
	moveResultChan := make(chan moveResult, 100)
	pendingMoves := 0

	// Start the move worker
	startMoveWorker(moveQueueChan, moveResultChan, sb)

	for item := range docuChan {
		found++

		// Flush any buffered log messages from scanner
		for {
			select {
			case msg := <-logChan:
				fmt.Println(msg)
			default:
				goto doneFlushingLogs
			}
		}
	doneFlushingLogs:

		// Flush any completed move results (prints mv output from previous moves)
		flushMoveResults(moveResultChan, &pendingMoves, sb)

		// Print documentary info
		fmt.Println()
		c.Printf("<cyan>%d</>/<darkGray>%d</> <white>%s</> <darkGray>%s</>\n", item.index+1, item.total, item.folder, item.path)
		if url := item.nfo.TmdbURL(isSeries); url != "" {
			c.Printf("  <darkGray>tmdb:   </> %s\n", url)
		}
		c.Printf("  <darkGray>genres: </> %s\n", strings.Join(item.nfo.Genres, ", "))
		if item.nfo.Tagline != "" {
			c.Printf("  <darkGray>tagline:</> %s\n", item.nfo.Tagline)
		}
		if item.nfo.Outline != "" {
			c.Printf("  <darkGray>outline:</> %s\n", item.nfo.Outline)
		}
		if item.nfo.Plot != "" {
			c.Printf("  <darkGray>plot:   </> %s\n", item.nfo.Plot)
		}

		destPath := filepath.Join(destLib.Path, item.folder)
		c.Printf("  --> <green>%s</>\n", destPath)

		// Selection loop (re-asks after AI query)
		decided := false
		for !decided {
			c.Printf("  [m]ove/[a]ccept | [s]kip | [q]uery ai | e[x]it: ")
			selection, selErr := ktio.GetSelection('m', 'a', 's', 'q', 'x')
			fmt.Println()
			if selErr != nil {
				c.Printf("  <red>ERROR:</> %s\n", selErr)
				decided = true
				continue
			}

			switch selection {
			case 'm', 'a':
				// Queue the move (non-blocking) and continue immediately
				pendingMoves++
				moved++
				moveQueueChan <- moveAction{
					srcPath:  item.path,
					destPath: destPath,
					folder:   item.folder,
				}
				sb.UpdateMove(c.Sprintf("<yellow>queued (%d) %s</>", pendingMoves, item.folder))
				decided = true

			case 's':
				c.Printf("  <darkGray>skipping... removing documentary from genres...</>")
				if rmErr := content.RemoveDocumentaryGenre(item.nfoPath); rmErr != nil {
					c.Printf(" <red>ERROR:</> %s\n", rmErr)
				} else {
					c.Printf(" <darkGray>done</>\n")
				}
				decided = true

			case 'q':
				c.Printf("  <darkGray>querying AI...</>\n")
				result, aiErr := queryAI(item.folder, isSeries)
				if aiErr != nil {
					c.Printf("  <red>ERROR:</> %s\n", aiErr)
				} else {
					c.Printf("  <lightYellow>AI:</> %s\n", result)
				}
				// don't set decided - re-ask

			case 'x':
				close(moveQueueChan)
				drainMoveResults(moveResultChan, &pendingMoves, sb)
				return found, moved, errors.New("quitting")
			}
		}
		fmt.Println()
	}

	// Close the queue and wait for all pending moves to finish
	close(moveQueueChan)
	drainMoveResults(moveResultChan, &pendingMoves, sb)

	return found, moved, nil
}

// ExtractDocuMovies scans the movies library for documentaries via NFO files
// and moves them to the documentary torrent-sorted import folder (m.docu)
func ExtractDocuMovies(movieLib, docuImportLib *content.Library, sb *ktio.StatusBar) error {
	logChan := make(chan string, 100)
	docuChan, total, err := scanMoviesForDocus(movieLib, sb, logChan)
	if err != nil {
		return err
	}

	found, moved, err := processDocuItems(docuChan, docuImportLib, false, sb, logChan)
	if err != nil {
		return err
	}

	c.Printf("<yellow>Found %d documentaries, moved %d</> out of %d movies\n", found, moved, total)
	return nil
}

// ExtractDocuSeries scans the TV library for docuseries via NFO files
// and moves them to the docuseries torrent-sorted import folder (s.docu)
func ExtractDocuSeries(tvLib, docuImportLib *content.Library, sb *ktio.StatusBar) error {
	logChan := make(chan string, 100)
	docuChan, total, err := scanSeriesForDocus(tvLib, sb, logChan)
	if err != nil {
		return err
	}

	found, moved, err := processDocuItems(docuChan, docuImportLib, true, sb, logChan)
	if err != nil {
		return err
	}

	c.Printf("<yellow>Found %d docuseries, moved %d</> out of %d series\n", found, moved, total)
	return nil
}
