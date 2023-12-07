package cli

import (
	"fmt"
	"path"
	"sort"

	c "github.com/gookit/color"
	"github.com/katbyte/go-injest-media/lib/content"
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

func Make(cmdName string) (*cobra.Command, error) {

	root := &cobra.Command{
		Use:           cmdName + " [command]",
		Short:         cmdName + "injest media into my specific folder structure",
		Long:          `A CLI tool to intelligently injest media into my specific folder structure taking into account existing media and video format/quality.`,
		SilenceErrors: true,
		PreRunE:       ValidateParams([]string{"base-in", "base-out"}),
		RunE: func(cmd *cobra.Command, args []string) error {
			f := GetFlags()

			for _, l := range content.GetLibraries(f.BaseSrcPath, f.BaseDstPath) {

				c.Printf("%s/<white>%s</> --> %s/<lightBlue>%s</> ", f.BaseSrcPath, l.SrcFolder, f.BaseDstPath, l.DstFolder)
				if l.Type == content.Movies {
					c.Printf("<cyan>(movies)</> ")
				} else {
					c.Printf("<magenta>(series)</> ")
				}

				if l.LetterFolders {
					c.Printf("<lightGreen>(letter)</> ")
				}
				fmt.Println()

				if l.Type == content.Movies {
					movies, err := l.Movies(func(f string, err error) {
						c.Printf("  %s --> <red>ERROR:</>%s</>\n", path.Base(f), err)
					})
					if err != nil {
						return fmt.Errorf("error getting movies: %w", err)
					}

					sort.Slice(movies, func(i, j int) bool {
						return movies[i].Letter+"/"+movies[i].DstFolder < movies[j].Letter+"/"+movies[j].DstFolder
					})

					for _, m := range movies {

						// if not exists just move folde
						if !m.DstExists() {
							c.Printf("  <white>%s</> --> <green>%s</>", m.SrcFolder, m.DstFolder)
							c.Printf(" <darkGray>mv '%s' '%s'...</>\n", m.SrcPath(), m.DstPath())
							m.Move(4)
							continue
						}

						c.Printf("  <white>%s</> --> <yellow>%s</>\n", m.SrcFolder, m.DstFolder)
						continue
					}
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
