package cli

import (
	"errors"
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

	movies, err := l.MoviesSource(func(f string, err error) {
		c.Printf("  %s --> <red>ERROR:</>%s</>\n", path.Base(f), err)
	})
	if err != nil {
		return fmt.Errorf("error getting movies: %w", err)
	}

	sort.Slice(movies, func(i, j int) bool {
		return movies[i].Letter+"/"+movies[i].DstFolder < movies[j].Letter+"/"+movies[j].DstFolder
	})

	srcPathsToDelete := []string{}

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

		if len(m.DstVideos) == 0 {
			c.Printf("  <yellow>WARNING</> - destination has no video files\n")

			// Move all source video files
			for _, srcVideo := range m.SrcVideos {
				ktio.RunCommand(4, f.Confirm, "mv", "-v", srcVideo.Path, m.DstPath()+"/")
			}

			// Move all other files in the source folder
			srcFiles, err := ktio.ListFiles(m.SrcPath())
			if err != nil {
				c.Printf(" <red>ERROR:</> error listing source files: %s\n", err)
			}
			for _, file := range srcFiles {
				if !content.IsVideoFile(file) {
					_ = ktio.RunCommand(4, f.Confirm, "mv", "-v", file, m.DstPath()+"/")
				}
			}

			// delete source folder if empty
			empty, err := ktio.FolderEmpty(m.SrcPath())
			if err != nil {
				c.Printf(" <red>ERROR:</> checking if empty%s\n", err)
			}
			if empty {
				_ = ktio.RunCommand(4, f.Confirm, "rmdir", "-v", m.SrcPath())
			}

			continue
		}

		// handle multiple source videos by simply looping over them
		srcCount := 0
		for _, srcVideo := range m.SrcVideos {
			srcCount++
			// skip if the same and add delete command to rm collection
			isSame := false
			for _, dstVideo := range m.DstVideos {
				if srcVideo.IsBasicallyTheSameTo(dstVideo) {
					isSame = true
					break
				}
			}

			if isSame {
				c.Printf("  <green>SAME</> - adding to delete list\n\n\n")
				srcPathsToDelete = append(srcPathsToDelete, srcVideo.Path)
				continue
			}

			if f.IgnoreExisting {
				c.Printf("  <magenta>EXISTING</> - skipping due to flag\n\n\n")
				continue
			}

			// output video comparison table
			if len(m.SrcVideos) > 1 {
				RenderVideoComparisonTable(2, srcVideo, m.DstVideos, srcCount)
			} else {
				RenderVideoComparisonTable(2, srcVideo, m.DstVideos, -1)
			}
			c.Printf(" overwrite (y/a?) delete src (d?) skip (s?) pick dest (1-%d) quit (q?): ", len(m.DstVideos))
			options := []rune{'a', 'y', 'd', 's', 'q'}
			for k := 1; k <= len(m.DstVideos) && k <= 9; k++ {
				options = append(options, rune('0'+k))
			}
			s, err := ktio.GetSelection(options...)
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
				// delete destination video files and move new one
				_ = m.MoveFiles(srcVideo, f.Confirm, 4)
				fmt.Println()
				continue
			case 's':
				fmt.Println()
				continue
			case '1', '2', '3', '4', '5', '6', '7', '8', '9':
				keepIdx := int(s-'0') - 1

				// delete destination video files and move new one
				// add all but the selected one to delete list and update destination struct incase there is another source
				var newDst []content.VideoFile
				for idx, v := range m.DstVideos {
					if idx == keepIdx {
						newDst = append(newDst, v)
					} else {
						c.Printf("    <darkGray>rm -v \"%s\"</>\n", v.Path)
						_ = ktio.RunCommand(4, f.Confirm, "rm", "-v", v.Path)
					}
				}
				m.DstVideos = newDst
				fallthrough // no "delete" the source
			case 'd':
				// c.Printf(" <darkGray>rm -rf '%s'...</>", m.SrcPath())

				// m.DeleteFolder() // this seems dangerous, should we even implement it?
				// maybe we output all rm statements at the end and let the user run them
				srcPathsToDelete = append(srcPathsToDelete, srcVideo.Path)
				fmt.Println()
				continue
			case 'q':
				return errors.New("quitting")
			}
		}
	}

	// print delete commands
	if len(srcPathsToDelete) > 0 {
		c.Printf("\n\n<red>%d items to DELETE:</>\n", len(srcPathsToDelete))
		for _, cmd := range srcPathsToDelete {
			c.Printf("%s\n", cmd)
		}

		c.Printf("<red>CONFIRM DELETE</> y/n: ")
		y, err := ktio.Confirm()
		fmt.Println()
		if err != nil {
			return err
		}

		if y {
			for _, path := range srcPathsToDelete {
				_ = ktio.RunCommand(4, f.Confirm, "rm", "-rfv", path)
			}
		}
		fmt.Println()

	}
	return nil
}
