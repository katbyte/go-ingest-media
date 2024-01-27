package cli

import (
	"fmt"
	"path"
	"sort"

	c "github.com/gookit/color"
	"github.com/katbyte/go-ingest-media/lib/content"
	"github.com/katbyte/go-ingest-media/lib/ktio"
	_ "github.com/mattn/go-sqlite3"
)

func ProcessMovies(l content.Library) error {
	f := GetFlags()

	movies, err := l.Movies(func(f string, err error) {
		c.Printf("  %s --> <red>ERROR:</>%s</>\n", path.Base(f), err)
	})
	if err != nil {
		return fmt.Errorf("error getting movies: %w", err)
	}

	sort.Slice(movies, func(i, j int) bool {
		return movies[i].Letter+"/"+movies[i].DstFolder < movies[j].Letter+"/"+movies[j].DstFolder
	})

	pathsToDelete := []string{}

	i := 0
	nMovies := len(movies)
	for _, m := range movies {
		i++

		// TODO
		// TODO
		// TODO use go channels to run multiple moves at once/queue them up in the background
		// TODO
		// TODO

		// if not exists just move folder nice and easy like
		if !m.DstExists() {
			c.Printf("<darkGray>%d/%d</> <white>%s</> --> <green>%s</>", i, nMovies, m.SrcFolder, m.DstFolder)
			m.MoveFolder(f.Confirm, 4)
			fmt.Println()
			continue
		}

		// exists so lets grab the video details
		c.Printf("<darkGray>%d/%d</>  <white>%s</> --> <yellow>%s</>\n", i, nMovies, m.SrcFolder, m.DstFolder)

		// load video details
		if err = m.LoadVideoInfo(); err != nil {
			c.Printf(" <red>ERROR:</>%s\n\n", err)
			continue
		}

		if len(m.DstVideos) > 1 {
			c.Printf(" <red>ERROR:</>%s\n", "destination has multiple video files\n\n")
			continue
		} else if len(m.DstVideos) == 0 {
			c.Printf("  <yellow>WARNING</> - destination has no video files\n")
			m.MoveFiles(false, 4)
			continue
		}

		// skip if the same and add delete command to rm collection
		same := m.SrcVideo.IsBasicallyTheSameTo(m.DstVideos[0])
		if same {
			c.Printf("  <green>SAME</> - adding to delete list\n\n\n")
			pathsToDelete = append(pathsToDelete, m.SrcPath())
			continue
		}

		if f.IgnoreExisting {
			c.Printf("  <magenta>EXISTING</> - skipping due to flag\n\n\n")
			continue
		}

		// output video comparison table
		RenderVideoComparisonTable(m.SrcVideo, m.DstVideos)

		c.Printf(" overwrite (y/a?) delete src (d?) skip (s?) quit (q?): ")
		s, err := ktio.GetSelection('a', 'y', 'd', 's', 'q')
		fmt.Println()
		fmt.Println()
		if err != nil {
			c.Printf(" <red>ERROR:</>%s\n", err)
			continue
		}

		switch s {
		case 'a':
			fallthrough
		case 'y':
			m.MoveFiles(false, 4)
			fmt.Println()
		case 'd':
			// c.Printf(" <darkGray>rm -rf '%s'...</>", m.SrcPath())

			// m.DeleteFolder() // this seems dangerous, should we even implement it?
			// maybe we output all rm statements at the end and let the user run them
			pathsToDelete = append(pathsToDelete, m.SrcPath())
			fmt.Println()
		case 's':
			continue
		case 'q':
			return fmt.Errorf("quitting")

		}
	}

	// print delete commands
	if len(pathsToDelete) > 0 {
		c.Printf("\n\n<red>%d items to DELETE:</>\n", len(pathsToDelete))
		for _, cmd := range pathsToDelete {
			c.Printf("%s\n", cmd)
		}

		c.Printf("<red>CONFIRM DELETE</> y/n: ")
		y, err := ktio.Confirm()
		fmt.Println()
		if err != nil {
			return err
		}

		if y {
			for _, path := range pathsToDelete {
				ktio.RunCommand(4, f.Confirm, "rm", "-rfv", path)
			}
		}
		fmt.Println()

	}
	return nil
}
