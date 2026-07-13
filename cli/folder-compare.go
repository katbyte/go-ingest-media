package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	c "github.com/gookit/color"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/katbyte/go-ingest-media/lib/content"
	"github.com/katbyte/go-ingest-media/lib/ktio"
)

// FolderInfo holds a quick summary of a movie folder's contents (no ffprobe)
type FolderInfo struct {
	Path      string
	ModTime   time.Time
	Exists    bool
	Videos    []FileEntry // video files (name, size, ext)
	Extras    []string    // files in Extras/Featurettes/etc subfolders
	Specials  []string    // files in Specials subfolder
	Trailers  []string    // files in Trailers subfolder
	Subs      []string    // .srt/.ass/.sub files
	HasNFO    bool
	NfoTitle  string      // title from NFO file
	NfoYear   string      // year from NFO file
	TotalSize int64
}

// FileEntry is a simple file summary (no ffprobe)
type FileEntry struct {
	Name string
	Ext  string
	Size int64
}

// Known "extras" subfolder names (case-insensitive match)
var extrasSubfolders = []string{
	"extras", "featurettes", "behind the scenes",
	"deleted scenes", "interviews", "scenes", "shorts", "other",
}

// ScanFolder quickly scans a movie folder and returns a FolderInfo summary
func ScanFolder(folderPath string) FolderInfo {
	info := FolderInfo{Path: folderPath}

	stat, err := os.Stat(folderPath)
	if err != nil {
		return info
	}
	info.Exists = true
	info.ModTime = stat.ModTime()

	entries, err := os.ReadDir(folderPath)
	if err != nil {
		return info
	}

	for _, entry := range entries {
		fullPath := filepath.Join(folderPath, entry.Name())
		nameL := strings.ToLower(entry.Name())

		if entry.IsDir() {
			// Check for known special subfolders
			switch {
			case nameL == "specials":
				info.Specials = listFilesIn(fullPath)
			case nameL == "trailers" || nameL == "trailer":
				info.Trailers = listFilesIn(fullPath)
			case isExtrasFolder(nameL):
				info.Extras = append(info.Extras, listFilesIn(fullPath)...)
			}
			continue
		}

		fi, err := entry.Info()
		if err != nil {
			continue
		}

		ext := strings.ToLower(filepath.Ext(entry.Name()))
		size := fi.Size()
		info.TotalSize += size

		switch {
		case content.IsVideoFile(fullPath):
			info.Videos = append(info.Videos, FileEntry{
				Name: entry.Name(),
				Ext:  ext,
				Size: size,
			})
		case ext == ".srt" || ext == ".ass" || ext == ".sub" || ext == ".ssa" || ext == ".idx":
			info.Subs = append(info.Subs, entry.Name())
		case ext == ".nfo":
			info.HasNFO = true
			nfo, nfoErr := content.ReadNfo(fullPath)
			if nfoErr == nil && nfo != nil {
				info.NfoTitle = nfo.Title
				info.NfoYear = nfo.Year
			}
		}
	}

	return info
}

func isExtrasFolder(name string) bool {
	for _, ef := range extrasSubfolders {
		if name == ef {
			return true
		}
	}
	return false
}

func listFilesIn(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names
}

func formatSize(bytes int64) string {
	gb := float64(bytes) / 1024 / 1024 / 1024
	if gb >= 1 {
		return fmt.Sprintf("%.2f GB", gb)
	}
	mb := float64(bytes) / 1024 / 1024
	return fmt.Sprintf("%.0f MB", mb)
}

func formatVideoSummary(videos []FileEntry) string {
	if len(videos) == 0 {
		return c.Sprintf("<red>none</>")
	}

	exts := make([]string, 0, len(videos))
	for _, v := range videos {
		exts = append(exts, v.Ext)
	}
	return fmt.Sprintf("%d (%s)", len(videos), strings.Join(exts, ", "))
}

func formatCheck(items []string) string {
	if len(items) == 0 {
		return c.Sprintf("<darkGray>—</>")
	}
	return c.Sprintf("<green>✓</> (%d files)", len(items))
}

func formatBool(b bool) string {
	if b {
		return c.Sprintf("<green>✓</>")
	}
	return c.Sprintf("<darkGray>—</>")
}

// RenderFolderComparison renders a side-by-side folder comparison table
func RenderFolderComparison(indent int, infoA, infoB FolderInfo, labelA, labelB string) {
	var buf bytes.Buffer
	t := table.NewWriter()
	t.SetOutputMirror(&buf)
	t.SetStyle(tablestyle)

	t.AppendHeader(table.Row{"", labelA, labelB})
	t.AppendSeparator()

	// Modification date
	modA := c.Sprintf("<darkGray>n/a</>")
	modB := c.Sprintf("<darkGray>n/a</>")
	if infoA.Exists {
		modA = infoA.ModTime.Format("2006-01-02 15:04")
	}
	if infoB.Exists {
		modB = infoB.ModTime.Format("2006-01-02 15:04")
	}
	// Highlight newer
	if infoA.Exists && infoB.Exists {
		if infoA.ModTime.After(infoB.ModTime) {
			modA = c.Sprintf("<lightGreen>%s</>", modA)
			modB = c.Sprintf("<lightRed>%s</>", modB)
		} else if infoB.ModTime.After(infoA.ModTime) {
			modA = c.Sprintf("<lightRed>%s</>", modA)
			modB = c.Sprintf("<lightGreen>%s</>", modB)
		} else {
			modA = c.Sprintf("<green>%s</>", modA)
			modB = c.Sprintf("<green>%s</>", modB)
		}
	}
	t.AppendRow(table.Row{c.Sprintf("<darkGray>Modified</>"), modA, modB})

	// Video files
	vidA := formatVideoSummary(infoA.Videos)
	vidB := formatVideoSummary(infoB.Videos)
	if len(infoA.Videos) > 0 && vidA == vidB {
		vidA = c.Sprintf("<green>%s</>", vidA)
		vidB = c.Sprintf("<green>%s</>", vidB)
	}
	t.AppendRow(table.Row{c.Sprintf("<darkGray>Videos</>"), vidA, vidB})

	// Total size
	sizeA := formatSize(infoA.TotalSize)
	sizeB := formatSize(infoB.TotalSize)
	if infoA.TotalSize > infoB.TotalSize {
		sizeA = c.Sprintf("<lightGreen>%s</>", sizeA)
		sizeB = c.Sprintf("<lightRed>%s</>", sizeB)
	} else if infoB.TotalSize > infoA.TotalSize {
		sizeA = c.Sprintf("<lightRed>%s</>", sizeA)
		sizeB = c.Sprintf("<lightGreen>%s</>", sizeB)
	} else {
		sizeA = c.Sprintf("<green>%s</>", sizeA)
		sizeB = c.Sprintf("<green>%s</>", sizeB)
	}
	t.AppendRow(table.Row{c.Sprintf("<darkGray>Size</>"), sizeA, sizeB})

	// Individual video files with sizes
	maxVideos := len(infoA.Videos)
	if len(infoB.Videos) > maxVideos {
		maxVideos = len(infoB.Videos)
	}
	for i := 0; i < maxVideos; i++ {
		var a, b string
		var sameBytes bool
		if i < len(infoA.Videos) && i < len(infoB.Videos) && infoA.Videos[i].Size == infoB.Videos[i].Size {
			sameBytes = true
		}
		if i < len(infoA.Videos) {
			a = fmt.Sprintf("%s %s", infoA.Videos[i].Ext, formatSize(infoA.Videos[i].Size))
			if sameBytes {
				a = c.Sprintf("<green>%s (same)</>", a)
			}
		}
		if i < len(infoB.Videos) {
			b = fmt.Sprintf("%s %s", infoB.Videos[i].Ext, formatSize(infoB.Videos[i].Size))
			if sameBytes {
				b = c.Sprintf("<green>%s (same)</>", b)
			}
		}
		t.AppendRow(table.Row{c.Sprintf("<darkGray>  File %d</>", i+1), a, b})
	}

	// Extras
	t.AppendRow(table.Row{c.Sprintf("<darkGray>Extras</>"), formatCheck(infoA.Extras), formatCheck(infoB.Extras)})

	// Specials
	t.AppendRow(table.Row{c.Sprintf("<darkGray>Specials</>"), formatCheck(infoA.Specials), formatCheck(infoB.Specials)})

	// Trailers
	t.AppendRow(table.Row{c.Sprintf("<darkGray>Trailers</>"), formatCheck(infoA.Trailers), formatCheck(infoB.Trailers)})

	// Subtitles
	subA := c.Sprintf("<darkGray>—</>")
	subB := c.Sprintf("<darkGray>—</>")
	if len(infoA.Subs) > 0 {
		subA = fmt.Sprintf("%d files", len(infoA.Subs))
	}
	if len(infoB.Subs) > 0 {
		subB = fmt.Sprintf("%d files", len(infoB.Subs))
	}
	if len(infoA.Subs) > 0 && len(infoA.Subs) == len(infoB.Subs) {
		subA = c.Sprintf("<green>%s</>", subA)
		subB = c.Sprintf("<green>%s</>", subB)
	}
	t.AppendRow(table.Row{c.Sprintf("<darkGray>Subtitles</>"), subA, subB})

	// NFO
	t.AppendRow(table.Row{c.Sprintf("<darkGray>NFO</>"), formatBool(infoA.HasNFO), formatBool(infoB.HasNFO)})

	// NFO Title
	nfoTitleA := c.Sprintf("<darkGray>—</>")
	nfoTitleB := c.Sprintf("<darkGray>—</>")
	if infoA.NfoTitle != "" {
		nfoTitleA = infoA.NfoTitle
	}
	if infoB.NfoTitle != "" {
		nfoTitleB = infoB.NfoTitle
	}
	if infoA.NfoTitle != "" && infoA.NfoTitle == infoB.NfoTitle {
		nfoTitleA = c.Sprintf("<green>%s</>", nfoTitleA)
		nfoTitleB = c.Sprintf("<green>%s</>", nfoTitleB)
	}
	t.AppendRow(table.Row{c.Sprintf("<darkGray>NFO Title</>"), nfoTitleA, nfoTitleB})

	// NFO Year
	nfoYearA := c.Sprintf("<darkGray>—</>")
	nfoYearB := c.Sprintf("<darkGray>—</>")
	if infoA.NfoYear != "" {
		nfoYearA = infoA.NfoYear
	}
	if infoB.NfoYear != "" {
		nfoYearB = infoB.NfoYear
	}
	if infoA.NfoYear != "" && infoA.NfoYear == infoB.NfoYear {
		nfoYearA = c.Sprintf("<green>%s</>", nfoYearA)
		nfoYearB = c.Sprintf("<green>%s</>", nfoYearB)
	}
	t.AppendRow(table.Row{c.Sprintf("<darkGray>NFO Year</>"), nfoYearA, nfoYearB})

	t.Render()

	output := buf.String()
	if len(output) > 0 {
		output = output[:len(output)-1]
	}
	_, _ = ktio.IndentWriter{W: os.Stdout, Indent: strings.Repeat(" ", indent)}.Write([]byte(output))
	fmt.Println()
}
