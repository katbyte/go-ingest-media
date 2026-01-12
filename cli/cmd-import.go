package cli

import (
	"fmt"
	"path/filepath"
	"strconv"

	c "github.com/gookit/color"
	"github.com/katbyte/go-ingest-media/lib/content"
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/cobra"
)

func ImportDownloadedContent(cmd *cobra.Command, args []string) error {
	f := GetFlags()

	// Initialise library paths
	content.InitLibraries(f.BaseSrcPath, f.BaseDstPath)

	for id, mapping := range content.GetLibraryMappings() {
		src := mapping.Source
		dst := mapping.Dest

		srcFolder := filepath.Base(src.Path)
		dstFolder := filepath.Base(dst.Path)

		c.Printf("%s/<white>%s</> --> %s/<lightBlue>%s</> ", f.BaseSrcPath, srcFolder, f.BaseDstPath, dstFolder)
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
