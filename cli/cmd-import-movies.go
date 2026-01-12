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

func ProcessMovies(id string, mapping content.LibraryMapping) error {
	f := GetFlags()

	srcLib := mapping.Source
	dstLib := mapping.Dest

	// Get all movies from source library
	movies, err := srcLib.Movies(func(folder string, err error) {
		c.Printf("  %s --> <red>ERROR:</></>: %s\n", path.Base(folder), err)
	})
	if err != nil {
		return fmt.Errorf("error loading movies: %w", err)
	}

	sort.Slice(movies, func(i, j int) bool {
		return movies[i].Letter+"/"+movies[i].Folder < movies[j].Letter+"/"+movies[j].Folder
	})

	srcPathsToDelete := []string{}

	i := 0
	nMovies := len(movies)
	for _, m := range movies {
		i++
		fmt.Println()

		// Compute destination path (with potential rename)
		destPath, err := m.DestPathInWithRename(dstLib, srcLib.Type)
		if err != nil {
			c.Printf("<darkGray>%d/%d</> <white>%s</> --> <red>ERROR:</> computing dest path: %s\n", i, nMovies, m.Folder, err)
			continue
		}

		// if destination doesn't exist, just move folder
		if !ktio.PathExists(destPath) {
			c.Printf("<darkGray>%d/%d</> <white>%s</> --> <green>%s</>", i, nMovies, m.Folder, path.Base(destPath))
			if err := m.MoveFolder(destPath, f.Confirm, 4); err != nil {
				c.Printf(" <red>ERROR:</> moving folder: %s\n", err)
			}
			continue
		}

		// exists so lets grab the video details
		c.Printf("<darkGray>%d/%d</>  <white>%s</> --> <yellow>%s</>\n", i, nMovies, m.Folder, path.Base(destPath))

		// load source videos
		if err = m.LoadVideos(); err != nil {
			c.Printf(" <red>ERROR:</> loading source videos: %s\n\n", err)
			continue
		}

		// load destination videos
		dstVideos, err := content.VideosInPath(destPath)
		if err != nil {
			c.Printf(" <red>ERROR:</> loading dest videos: %s\n\n", err)
			continue
		}

		// if no source videos, delete nfo files and folder if empty
		if len(m.Videos) == 0 {
			c.Printf("  <yellow>WARNING</> - no source videos\n")

			if err := ktio.DeleteIfEmptyOrOnlyNfo(m.Path(), f.Confirm, 4); err != nil {
				c.Printf("   <red>ERROR:</> deleting source folder: %s\n", err)
				continue
			}

			if ktio.PathExists(m.Path()) {
				c.Printf("    <red>ERROR:</> source folder still exists, skipping\n")
			}
			continue
		}

		// if multiple source videos, ask which one to keep
		if len(m.Videos) > 1 {
			c.Printf("  <lightMagenta>WARNING - multiple source videos - WARNING </>\n")
			headers := []string{}
			for i := range m.Videos {
				headers = append(headers, fmt.Sprintf("Source %d", i+1))
			}

			RenderVideoComparisonTable(2, headers, m.Videos)
			c.Printf(" pick source to keep (1-%d) skip (s) quit (q): ", len(m.Videos))

			options := []rune{'s', 'q'}
			for k := 1; k <= len(m.Videos) && k <= 9; k++ {
				options = append(options, rune('0'+k))
			}
			s, err := ktio.GetSelection(options...)
			fmt.Println()
			if err != nil {
				c.Printf(" <red>ERROR:</>%s\n", err)
				continue
			}

			if s == 'q' {
				return errors.New("quitting")
			}
			if s == 's' {
				continue
			}

			keepIdx := int(s-'0') - 1
			keptVideo := m.Videos[keepIdx]

			for idx, v := range m.Videos {
				if idx != keepIdx {
					if err := ktio.RunCommand(4, f.Confirm, "rm", "-v", v.Path); err != nil {
						c.Printf("   <red>ERROR:</> deleting source video: %s\n", err)
					}
				}
			}
			m.Videos = []content.VideoFile{keptVideo}
		}

		if len(dstVideos) == 0 {
			c.Printf("  <yellow>WARNING</> - destination has no video files\n")
			if err := m.MoveFilesTo(destPath, f.Confirm, 4); err != nil {
				c.Printf("   <red>ERROR:</> moving files: %s\n", err)
			}
			continue
		}

		// handle single source video
		srcVideo := m.Videos[0]

		// skip if the same and add delete command to rm collection
		isSame := false
		for _, dstVideo := range dstVideos {
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
		headers := []string{"Source"}
		if len(dstVideos) == 1 {
			headers = append(headers, "Destination")
		} else {
			for i := range dstVideos {
				headers = append(headers, fmt.Sprintf("Dest %d", i+1))
			}
		}
		RenderVideoComparisonTable(2, headers, append([]content.VideoFile{srcVideo}, dstVideos...))
		c.Printf(" overwrite (y/a?) delete src (d?) skip (s?) pick dest (1-%d) quit (q?): ", len(dstVideos))
		options := []rune{'a', 'y', 'd', 's', 'q'}
		for k := 1; k <= len(dstVideos) && k <= 9; k++ {
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
			// delete destination video files first
			for _, v := range dstVideos {
				if err := ktio.RunCommand(4, f.Confirm, "rm", "-v", v.Path); err != nil {
					c.Printf("   <red>ERROR:</> deleting destination video: %s\n", err)
				}
			}
			// move source files to destination
			if err := m.MoveFilesTo(destPath, f.Confirm, 4); err != nil {
				c.Printf("   <red>ERROR:</> moving files: %s\n", err)
			}
		case 's':
		case '1', '2', '3', '4', '5', '6', '7', '8', '9':
			keepIdx := int(s-'0') - 1

			// delete destination video files except the selected one
			for idx, v := range dstVideos {
				if idx != keepIdx {
					if err := ktio.RunCommand(4, f.Confirm, "rm", "-v", v.Path); err != nil {
						c.Printf("   <red>ERROR:</> deleting destination video: %s\n", err)
					}
				}
			}
			fallthrough // now delete the source
		case 'd':
			srcPathsToDelete = append(srcPathsToDelete, m.Path())
			continue
		case 'q':
			return errors.New("quitting")
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
				if err := ktio.RunCommand(4, f.Confirm, "rm", "-rfv", path); err != nil {
					c.Printf("   <red>ERROR:</> deleting source folder: %s\n", err)
				}
			}
		}
	}
	return nil
}
