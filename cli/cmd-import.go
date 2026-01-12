package cli

import (
	"fmt"
	"sort"
	"strconv"

	c "github.com/gookit/color"
	"github.com/katbyte/go-ingest-media/lib/content"
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/cobra"
)

func ImportDownloadedContent(cmd *cobra.Command, args []string) error {
	// Sort keys for consistent ordering
	keys := make([]string, 0, len(content.LibraryMappingSortedTorrentsImport))
	for k := range content.LibraryMappingSortedTorrentsImport {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, id := range keys {
		mapping := content.LibraryMappingSortedTorrentsImport[id]
		src := mapping.Source
		dst := mapping.Dest

		c.Printf("<white>%s</> --> <lightBlue>%s</> ", src.Path, dst.Path)
		switch src.Type {
		case content.LibraryTypeMovies:
			c.Printf("<cyan>(movies)</> ")
		case content.LibraryTypeStandup:
			c.Printf("<cyan>(standup)</> ")
		case content.LibraryTypeSeries:
			c.Printf("<magenta>(series)</> ")
		case content.LibraryTypeUnknown:
			c.Printf("<red>(unknown)</> ")
		}

		if dst.LetterFolders {
			c.Printf("<lightGreen>(letter)</> ")
		}
		fmt.Println()

		switch src.Type {
		case content.LibraryTypeMovies, content.LibraryTypeStandup:
			err := ProcessMovies(id, mapping)
			if err != nil {
				return err
			}
		case content.LibraryTypeSeries:
			err := ProcessSeries(id, mapping)
			if err != nil {
				return err
			}
		case content.LibraryTypeUnknown:
			fallthrough
		default:
			panic("unknown library type: " + strconv.Itoa(int(src.Type)))
		}
	}

	return nil
}
