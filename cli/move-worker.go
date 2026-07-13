package cli

import (
	"fmt"
	"os/exec"
	"strings"

	c "github.com/gookit/color"
	"github.com/katbyte/go-ingest-media/lib/ktio"
)

// moveAction represents a queued move request
type moveAction struct {
	srcPath  string
	destPath string
	folder   string
}

// moveResult holds the output of a background move operation
type moveResult struct {
	folder string
	output string
	err    error
}

// printMoveResult displays the output of a completed move
func printMoveResult(result moveResult) {
	if result.err != nil {
		c.Printf("  <red>ERROR:</> moving %s: %s\n", result.folder, result.err)
	}
	if result.output != "" {
		for _, line := range strings.Split(strings.TrimSpace(result.output), "\n") {
			if line != "" {
				fmt.Printf("    %s\n", line)
			}
		}
	}
}

// flushMoveResults prints any completed move results without blocking
func flushMoveResults(ch <-chan moveResult, pending *int, sb *ktio.StatusBar) {
	for {
		select {
		case result := <-ch:
			*pending--
			printMoveResult(result)
			if *pending == 0 {
				sb.UpdateMove("")
			}
		default:
			return
		}
	}
}

// drainMoveResults blocks until all pending moves are complete and prints their results
func drainMoveResults(ch <-chan moveResult, pending *int, sb *ktio.StatusBar) {
	for *pending > 0 {
		result := <-ch
		*pending--
		printMoveResult(result)
	}
	sb.UpdateMove("")
}

// startMoveWorker starts a background goroutine that processes moves sequentially from a queue
func startMoveWorker(queue <-chan moveAction, results chan<- moveResult, sb *ktio.StatusBar) {
	go func() {
		for action := range queue {
			queued := len(queue) + 1 // +1 for current
			sb.UpdateMove(c.Sprintf("<yellow>moving (%d) %s...</>", queued, action.folder))

			cmd := exec.Command("mv", "-v", action.srcPath, action.destPath) //nolint:gosec
			output, cmdErr := cmd.CombinedOutput()

			if cmdErr != nil {
				sb.UpdateMove(c.Sprintf("<red>ERROR moving %s</>", action.folder))
			} else {
				sb.UpdateMove(c.Sprintf("<green>moved %s ✓</>", action.folder))
			}

			results <- moveResult{folder: action.folder, output: string(output), err: cmdErr}
		}
		close(results)
	}()
}
