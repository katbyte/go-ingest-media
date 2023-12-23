package cli

import (
	"fmt"

	c "github.com/gookit/color"
	"github.com/katbyte/go-ingest-media/lib/content"
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func ValidateParams(params []string) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		for _, p := range params {
			if viper.GetString(p) == "" {
				return fmt.Errorf(p + " parameter can't be empty")
			}
		}

		return nil
	}
}

// TODO
// check if movie exists in documentatry folder?
// or find a way to blah, OR just let emby figure it out and then movie it

func Make(cmdName string) (*cobra.Command, error) {

	root := &cobra.Command{
		Use:           cmdName + " [command]",
		Short:         cmdName + "go-ingest-media media into my specific folder structure",
		Long:          `A CLI tool to intelligently go-ingest-media media into my specific folder structure taking into account existing media and video format/quality.`,
		SilenceErrors: true,
		PreRunE:       ValidateParams([]string{"src", "dst"}),
		RunE: func(cmd *cobra.Command, args []string) error {
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
		},
	}

	if err := configureFlags(root); err != nil {
		return nil, fmt.Errorf("unable to configure flags: %w", err)
	}

	return root, nil
}
