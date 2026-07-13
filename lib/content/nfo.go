package content

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/katbyte/go-ingest-media/lib/ktio"
)

// NfoFile represents the relevant fields from an Emby/Jellyfin NFO XML file
type NfoFile struct {
	Title   string   `xml:"title"`
	Year    string   `xml:"year"`
	Genres  []string `xml:"genre"`
	Plot    string   `xml:"plot"`
	Outline string   `xml:"outline"`
	Tagline string   `xml:"tagline"`
	TmdbId  string   `xml:"tmdbid"`
}

// ReadNfo reads and parses an NFO XML file
func ReadNfo(filePath string) (*NfoFile, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading nfo file: %w", err)
	}

	var nfo NfoFile
	if err := xml.Unmarshal(data, &nfo); err != nil {
		return nil, fmt.Errorf("error parsing nfo xml: %w", err)
	}

	return &nfo, nil
}

// FindNfoFile finds the first .nfo file in a directory (non-recursive)
// It first tries the expected filename (folder_name.nfo) for speed, then falls back to listing
func FindNfoFile(dirPath string) (string, error) {
	// Fast path: try expected filename (folder_name.nfo) first - avoids ReadDir over NFS
	expected := filepath.Join(dirPath, filepath.Base(dirPath)+".nfo")
	if _, err := os.Stat(expected); err == nil {
		return expected, nil
	}

	// Slow path: list all files and find any .nfo
	files, err := ktio.ListFiles(dirPath)
	if err != nil {
		return "", fmt.Errorf("error listing files: %w", err)
	}

	for _, f := range files {
		if strings.ToLower(filepath.Ext(f)) == ".nfo" {
			return f, nil
		}
	}

	return "", nil // no nfo file found (not an error)
}

// IsDocumentary returns true if any genre contains "documentary" (case-insensitive)
func (n *NfoFile) IsDocumentary() bool {
	for _, genre := range n.Genres {
		if strings.Contains(strings.ToLower(genre), "documentary") {
			return true
		}
	}
	return false
}

// TmdbURL returns the TMDB URL for this content
func (n *NfoFile) TmdbURL(isSeries bool) string {
	if n.TmdbId == "" {
		return ""
	}

	if isSeries {
		return "https://www.themoviedb.org/tv/" + n.TmdbId
	}
	return "https://www.themoviedb.org/movie/" + n.TmdbId
}

// RemoveDocumentaryGenre removes documentary genre tags from an NFO file on disk
// by filtering out matching <genre> lines, preserving all other content
func RemoveDocumentaryGenre(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("error reading nfo file: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	var filtered []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(strings.ToLower(line))
		// skip lines like <genre>documentary</genre>
		if strings.HasPrefix(trimmed, "<genre>") && strings.Contains(trimmed, "documentary") && strings.HasSuffix(trimmed, "</genre>") {
			continue
		}
		filtered = append(filtered, line)
	}

	return os.WriteFile(filePath, []byte(strings.Join(filtered, "\n")), 0o644)
}
