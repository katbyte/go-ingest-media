package cli

import (
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"

	c "github.com/gookit/color"
	"github.com/katbyte/go-ingest-media/lib/content"
	"github.com/katbyte/go-ingest-media/lib/ktio"
)

type wrongItem struct {
	actualPath     string
	folderName     string
	actualLetter   string
	expectedLetter string
}

func findMislettered(lib *content.Library, sb *ktio.StatusBar) []wrongItem {
	if !lib.LetterFolders {
		return nil // Doesn't use letter folders
	}

	folders, err := ktio.ListFolders(lib.Path)
	if err != nil {
		sb.UpdateScan(c.Sprintf("<red>ERROR listing %s: %v</>", lib.Path, err))
		return nil
	}

	var letterFolders []string
	for _, f := range folders {
		if len(filepath.Base(f)) <= 1 {
			letterFolders = append(letterFolders, f)
		}
	}

	total := len(letterFolders)
	var foundCount atomic.Int32

	workCh := make(chan string, len(letterFolders))
	resultCh := make(chan wrongItem, 25)

	var wg sync.WaitGroup
	for w := 0; w < 10; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for letterFolder := range workCh {
				letter := filepath.Base(letterFolder)

				subFolders, err := ktio.ListFolders(letterFolder)
				if err != nil {
					continue // skip on error
				}

				for _, lf := range subFolders {
					folderName := filepath.Base(lf)
					expectedLetter := content.GetLetter(folderName)

					if expectedLetter != letter {
						foundCount.Add(1)
						resultCh <- wrongItem{
							actualPath:     lf,
							folderName:     folderName,
							actualLetter:   letter,
							expectedLetter: expectedLetter,
						}
					}
				}
			}
		}()
	}

	// Feeder goroutine
	go func() {
		for i, lf := range letterFolders {
			sb.UpdateScan(c.Sprintf("<darkGray>scanning</> <cyan>%d</>/<darkGray>%d (found %d)</> <darkGray>%s</>", i+1, total, foundCount.Load(), filepath.Base(lf)))
			workCh <- lf
		}
		close(workCh)
		wg.Wait()
		close(resultCh)
	}()

	var items []wrongItem
	for item := range resultCh {
		items = append(items, item)
	}

	sb.UpdateScan(c.Sprintf("<green>scan complete</> <darkGray>(found %d misplaced folders)</>", len(items)))
	return items
}

func FixLettering(sourceLib, destLib *content.Library, sb *ktio.StatusBar) error {
	c.Printf("<white>%s</> (scanning for wrong letter folders)\n", sourceLib.Path)

	items := findMislettered(sourceLib, sb)
	totalItems := len(items)

	if totalItems == 0 {
		c.Println("  <green>All folders in correct letter directories ✓</>")
		return nil
	}

	moveQueueChan := make(chan moveAction, 100)
	moveResultChan := make(chan moveResult, 100)
	var pendingMoves int

	startMoveWorker(moveQueueChan, moveResultChan, sb)

	for i, item := range items {
		flushMoveResults(moveResultChan, &pendingMoves, sb)

		c.Printf("<darkGray>[%d/%d]</> <yellow>%s</> is in <red>%s</> but should be in <green>%s</>\n", i+1, totalItems, item.folderName, item.actualLetter, item.expectedLetter)

		destPath := filepath.Join(destLib.Path, item.folderName)
		c.Printf("    moving to <lightBlue>%s</>\n", destPath)

		c.Printf("    [m]ove/[a]ccept | [s]kip | [q]uit: ")
		selection, err := ktio.GetSelection('m', 'a', 's', 'q')
		fmt.Println()
		if err != nil {
			c.Printf("    <red>ERROR:</> %s\n", err)
			continue
		}

		switch selection {
		case 'm', 'a':
			pendingMoves++
			moveQueueChan <- moveAction{
				srcPath:  item.actualPath,
				destPath: destPath,
				folder:   item.folderName,
			}
			sb.UpdateMove(c.Sprintf("<yellow>queued (%d) %s</>", pendingMoves, item.folderName))
		case 's':
			c.Println("    <darkGray>skipped</>")
		case 'q':
			close(moveQueueChan)
			drainMoveResults(moveResultChan, &pendingMoves, sb)
			return nil
		}
	}

	close(moveQueueChan)
	drainMoveResults(moveResultChan, &pendingMoves, sb)

	return nil
}
