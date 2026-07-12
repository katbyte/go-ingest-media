package cli

import (
	"errors"
	"fmt"

	c "github.com/gookit/color"
	"github.com/katbyte/go-ingest-media/lib/content"
	"github.com/katbyte/go-ingest-media/lib/ktio"
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func ValidateParams(params []string) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		for _, p := range params {
			if viper.GetString(p) == "" {
				return errors.New(p + " parameter can't be empty")
			}
		}

		return nil
	}
}

// TODO
// check if movie exists in documentatry folder?
// or find a way to blah, OR just let emby figure it out and then movie it

// NEW COMMAND - find all movies/series that are "documentary" type and move them to the documentary folder -
// check if already exists, use move logic like it was a new movie, if not prompt/ask with link to moviedb?

// NEW COMMAND - find all movies/series that are "standup" type and move them to the standup folder?
// parse comedy and then look up somehow

// NEW COMMAND - search through all folders and apply "library mappings" to them
// ie if there is a Batman movie check if it needs to be updated to conform to the new library mapps

func Make(cmdName string) (*cobra.Command, error) {
	// basic import
	root := &cobra.Command{
		Use:           cmdName + " [command]",
		Short:         cmdName + "move media from source paths into my specific folder structure",
		Long:          `A CLI tool to intelligently go-ingest-media media into my specific folder structure taking into account existing media and video format/quality.`,
		SilenceErrors: true,
		RunE:          ImportDownloadedContent,
	}

	// check fo duco duplicates between docu folders and movie/tv folders
	root.AddCommand(&cobra.Command{
		Use:           "docudups",
		Short:         cmdName + " check for duplicate documentaries/docuseries",
		Long:          `check for duplicate documentaries/docuseries between docu folders and movie/tv folders and then compare allowing deletion or move`,
		SilenceErrors: true,
		// PreRunE:       ValidateParams([]string{"cache"}),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Movie documentary duplicates
			docuLib := content.Libraries["video-documentary"]
			moviesLib := content.Libraries["video-movies"]

			c.Printf("%s <-- %s ", docuLib.Path, moviesLib.Path)
			fmt.Println()
			err := FindAndCombineDocu(docuLib, moviesLib)
			if err != nil {
				return err
			}

			// Series documentary duplicates
			docuseriesLib := content.Libraries["video-docuseries"]
			tvLib := content.Libraries["video-tv"]

			fmt.Println()
			c.Printf("%s <-- %s ", docuseriesLib.Path, tvLib.Path)
			fmt.Println()
			err = FindAndCombineDocuSeries(docuseriesLib, tvLib)
			if err != nil {
				return err
			}

			return nil
		},
	})

	// scan movie and tv libraries for documentaries via NFO files and move to import folders
	root.AddCommand(&cobra.Command{
		Use:           "docu-extract",
		Short:         cmdName + " scan libraries for documentaries and move to import folders",
		Long:          `Scan movie and TV libraries for content marked as documentary via NFO genre files and move them to the m.docu/s.docu torrent-sorted import folders`,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			sb := ktio.NewStatusBar()
			defer sb.Close()

			// Extract documentary movies: video-movies --> torrent-documentary (m.docu)
			moviesLib := content.Libraries["video-movies"]
			docuImportLib := content.Libraries["torrent-documentary"]

			c.Printf("<white>%s</> --> <lightBlue>%s</> ", moviesLib.Path, docuImportLib.Path)
			fmt.Println()
			if err := ExtractDocuMovies(moviesLib, docuImportLib, sb); err != nil {
				return err
			}

			// Extract documentary series: video-tv --> torrent-docuseries (s.docu)
			tvLib := content.Libraries["video-tv"]
			docuseriesImportLib := content.Libraries["torrent-docuseries"]

			fmt.Println()
			c.Printf("<white>%s</> --> <lightBlue>%s</> ", tvLib.Path, docuseriesImportLib.Path)
			fmt.Println()
			if err := ExtractDocuSeries(tvLib, docuseriesImportLib, sb); err != nil {
				return err
			}

			return nil
		},
	})

	if err := configureFlags(root); err != nil {
		return nil, fmt.Errorf("unable to configure flags: %w", err)
	}

	return root, nil
}
