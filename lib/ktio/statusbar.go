package ktio

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"golang.org/x/sys/unix"
)

// StatusBar manages fixed status lines at the bottom of the terminal
// using ANSI scroll regions. All normal output scrolls above the status lines.
type StatusBar struct {
	mu     sync.Mutex
	height int
	width  int
}

// NewStatusBar creates a status bar with a divider + 2 status lines at the bottom.
// It sets a scroll region that excludes the bottom 3 lines.
func NewStatusBar() *StatusBar {
	ws, err := unix.IoctlGetWinsize(int(os.Stdout.Fd()), unix.TIOCGWINSZ)
	if err != nil {
		return &StatusBar{height: 24, width: 80}
	}

	sb := &StatusBar{
		height: int(ws.Row),
		width:  int(ws.Col),
	}

	// Set scroll region to exclude bottom 3 lines (divider + move + scan)
	fmt.Printf("\033[1;%dr", sb.height-3)
	// Move cursor to end of scroll region
	fmt.Printf("\033[%d;1H", sb.height-3)
	// Render divider and empty status lines
	fmt.Printf("\033[s")
	fmt.Printf("\033[%d;1H\033[K%s", sb.height-2, strings.Repeat("═", sb.width))
	fmt.Printf("\033[%d;1H\033[K", sb.height-1)
	fmt.Printf("\033[%d;1H\033[K", sb.height)
	fmt.Printf("\033[u")

	return sb
}

// UpdateScan updates the scan status line (bottom line of terminal)
func (s *StatusBar) UpdateScan(text string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(text) > s.width-2 {
		text = text[:s.width-5] + "..."
	}
	// Save cursor, move to bottom line, clear, write, restore cursor
	fmt.Printf("\033[s\033[%d;1H\033[K%s\033[u", s.height, text)
}

// UpdateMove updates the move status line (second from bottom)
func (s *StatusBar) UpdateMove(text string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(text) > s.width-2 {
		text = text[:s.width-5] + "..."
	}
	fmt.Printf("\033[s\033[%d;1H\033[K%s\033[u", s.height-1, text)
}

// Close resets the terminal scroll region and cleans up status lines
func (s *StatusBar) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Reset scroll region to full terminal
	fmt.Printf("\033[r")
	// Clear status lines
	fmt.Printf("\033[%d;1H\033[K\033[%d;1H\033[K\033[%d;1H\033[K", s.height-2, s.height-1, s.height)
	// Move cursor to after content area
	fmt.Printf("\033[%d;1H", s.height-3)
}
