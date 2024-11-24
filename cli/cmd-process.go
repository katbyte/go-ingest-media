package cli

import (
	"fmt"

	c "github.com/gookit/color"
	"github.com/katbyte/go-ingest-media/lib/content"
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/cobra"
)

func ProcessLibraries(cmd *cobra.Command, args []string) error {
	f := GetFlags()

	for _, l := range content.GetLibraries(f.BaseSrcPath, f.BaseDstPath) {

		c.Printf("%s/<white>%s</> --> %s/<lightBlue>%s</> ", f.BaseSrcPath, l.SrcFolder, f.BaseDstPath, l.DstFolder)
		if l.Type == content.LibraryTypeMovies {
			c.Printf("<cyan>(movies)</> ")
		} else {
			c.Printf("<magenta>(series)</> ")
		}

		if l.LetterFolders {
			c.Printf("<lightGreen>(letter)</> ")
		}
		fmt.Println()

		if l.Type == content.LibraryTypeMovies || l.Type == content.LibraryTypeStandup { // standup is the same except a slighty different alt folder
			err := ProcessMovies(l)
			if err != nil {
				return err
			}
		} else if l.Type == content.LibraryTypeSeries {
			err := ProcessSeries(l)
			if err != nil {
				return err
			}
		} else {
			panic("unknown library type: " + string(l.Type))
		}
	}

	return nil
}
