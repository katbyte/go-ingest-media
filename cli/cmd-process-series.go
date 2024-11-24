package cli

import (
	"fmt"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"

	c "github.com/gookit/color"
	"github.com/katbyte/go-ingest-media/lib/content"
	"github.com/katbyte/go-ingest-media/lib/ktio"
)

func ProcessSeries(l content.Library) error {
	f := GetFlags()

	series, err := l.SeriesSource(func(f string, err error) {
		c.Printf("  %s --> <red>ERROR:</>%s</>\n", path.Base(f), err)
	})
	if err != nil {
		return fmt.Errorf("error getting series: %w", err)
	}

	sort.Slice(series, func(i, j int) bool {
		return series[i].Letter+"/"+series[i].DstFolder < series[j].Letter+"/"+series[j].DstFolder
	})

	pathsToDelete := []string{}

	i := 0
	nMovies := len(series)
	for _, s := range series {
		i++

		// if not exists just move folder nice and easy like
		if !s.DstExists() {
			c.Printf("<darkGray>%d/%d</> <white>%s</> --> <green>%s</>", i, nMovies, s.SrcFolder, s.DstFolder)
			s.MoveFolder(f.Confirm, 4)
			fmt.Println()
			continue
		}

		// exists so lets grab the video details
		c.Printf("<darkGray>%d/%d</>  <white>%s</> --> <yellow>%s</>\n", i, nMovies, s.SrcFolder, s.DstFolder)
		// load video details, we do this after confirming the dst exists
		if err = s.LoadContentDetails(); err != nil {
			c.Printf(" <red>ERROR:</>%s\n\n", err)
			continue
		}

		// calculate indent from "num/total"
		indent := len(strconv.Itoa(nMovies)) + 1 + len(strconv.Itoa(i)) + 1
		intentStr := strings.Repeat(" ", indent)

		var srcSeasonNumbers []int
		for k := range s.SrcSeasons {
			srcSeasonNumbers = append(srcSeasonNumbers, k)
		}

		// Sort the srcSeasonNumbers slice
		sort.Ints(srcSeasonNumbers)

		// for each src season
		for _, seasonNum := range srcSeasonNumbers {
			ss := s.SrcSeasons[seasonNum]

			// see if there is a dst season
			ds, exists := s.DstSeasons[ss.Number]
			if !exists {
				c.Printf("%s   season <green>%d</> --> ", intentStr, seasonNum)
				ss.MoveFolder(f.Confirm, indent+4, s.DstPath()+"/")
				continue
			}

			c.Printf("%s   season <yellow>%d</>: <darkGray>%d episodes</>\n", intentStr, seasonNum, len(ss.Episodes))

			var srcEpisodeNumbers []int
			for k := range ss.Episodes {
				srcEpisodeNumbers = append(srcEpisodeNumbers, k)
			}

			// Sort the srcSeasonNumbers slice
			sort.Ints(srcEpisodeNumbers)

			// for each episode in src season
			moveAll := false
			deleteAll := false
			skipAll := false
			for _, episodeNum := range srcEpisodeNumbers {
				se := ss.Episodes[episodeNum]

				// see if there is a dst episode
				de, exists := ds.Episodes[episodeNum]
				if !exists {
					// move episode files
					c.Printf("%s     <green>%dx%d</> --> ", intentStr, seasonNum, episodeNum)
					se.MoveFiles(f.Confirm, indent+10, ds.Path+"/")
					continue
				}

				if len(se.Videos) > 1 {
					c.Printf("%s     <red>%dx%d</> --> <red>ERROR</> - multiple source video files\n", intentStr, seasonNum, episodeNum)
					continue
				}

				if len(de.Videos) > 1 {
					c.Printf("%s     <red>%dx%d</> --> <red>ERROR</> - multiple destination video files\n", intentStr, seasonNum, episodeNum)
					continue
				}

				if len(se.Videos) == 0 {
					c.Printf("%s     <yellow>%dx%d</> --> source has no video file, copying other files except nfo\n", intentStr, seasonNum, episodeNum)

					// for each source file move it unless it is a nfo file
					for _, file := range se.OtherFiles {
						if strings.HasSuffix(file, ".nfo") {
							c.Printf("%s           --> nfo, skipping\n", intentStr)
						} else {
							c.Printf("%s           --> <white>%s</>", intentStr, path.Base(file))
							// ask for confirmation
							c.Printf(" move (y/n)? ")
							if yes, err := ktio.Confirm(); err != nil {
								c.Printf(" <red>ERROR:</>%s\n", err)
								continue
							} else if yes {
								ktio.RunCommand(indent+10, f.Confirm, "mv", "-v", file, ds.Path+"/")
							} else {
								// add to deletes
								fmt.Println()
								pathsToDelete = append(pathsToDelete, file)
							}
						}
					}

					continue
				}

				if len(de.Videos) == 0 {
					c.Printf("%s     <red>%dx%d</> --> <red>ERROR</> - dst has no video file\n", intentStr, seasonNum, episodeNum)
					continue
				}

				// skip if the same and add delete command to rm collection
				same := se.Videos[0].IsBasicallyTheSameTo(de.Videos[0])
				if same {
					c.Printf("%s     <green>%dx%d</> --> SAME - adding to delete list\n", intentStr, seasonNum, episodeNum)
					pathsToDelete = append(pathsToDelete, se.Videos[0].Path)
					continue
				}

				if f.IgnoreExisting {
					c.Printf("%s     <magenta>%dx%d</> --> skipping due to flag\n", intentStr, seasonNum, episodeNum)
					continue
				}

				c.Printf("%s     <yellow>%dx%d</> --> <darkGray>%s</>\n", intentStr, seasonNum, episodeNum, ds.Path)

				// output video comparison table
				RenderVideoComparisonTable(se.Videos[0], de.Videos, 15)

				s := 'q'
				if moveAll {
					s = 'A'
				} else if deleteAll {
					s = 'D'
				} else if skipAll {
					s = 'S'
				} else {
					c.Printf(" overwrite (y/a/A (all)?) delete src (d/D (all)?) skip (s/S?) quit (q?): ")
					s2, err := ktio.GetSelection('a', 'y', 'd', 's', 'q', 'A', 'D', 'S')
					if err != nil {
						c.Printf(" <red>ERROR:</>%s\n", err)
						continue
					}
					s = s2
				}

				switch s {
				case 'A':
					moveAll = true
					fallthrough
				case 'a':
					fallthrough
				case 'y':

					// delete de files
					fmt.Println()
					de.DeleteVideoFiles()

					se.MoveFiles(false, 4, ds.Path+"/")
				case 'D':
					deleteAll = true
					fallthrough
				case 'd':
					// c.Printf(" <darkGray>rm -rf '%s'...</>", m.SrcPath())

					// m.DeleteFolder() // this seems dangerous, should we even implement it?
					// maybe we output all rm statements at the end and let the user run them
					pathsToDelete = append(pathsToDelete, se.Videos[0].Path)
					fmt.Println()
				case 'S':
					skipAll = true
					continue
				case 's':
					continue
				case 'q':
					return fmt.Errorf("quitting")

				}
				fmt.Println()
			}

			// if empty season remove it
			empty, err := ktio.FolderEmpty(ss.Path)
			if err != nil {
				c.Printf(" <red>ERROR:</> checking if empty%s\n", err)
				continue
			}
			if empty {
				c.Printf("%s     <green>EMPTY</> - removing directory: ", intentStr)
				ktio.RunCommand(indent+6, f.Confirm, "rmdir", "-v", ss.Path)
			}
			fmt.Println()
		}

		if len(s.SpecialFiles) > 0 {
			c.Printf("%s   <magenta>%d special files</> \n", intentStr, len(s.SpecialFiles))
			ProcessSpecialFiles(indent, s, "specials", s.SpecialFiles, &pathsToDelete)
		}

		if len(s.ExtraFiles) > 0 {
			c.Printf("%s   <magenta>%d extra files</> \n", intentStr, len(s.ExtraFiles))
			ProcessSpecialFiles(indent, s, "extras", s.ExtraFiles, &pathsToDelete)
		}

		// if empty series folder then delete it
		empty, err := ktio.FolderEmpty(s.SrcPath())
		if err != nil {
			c.Printf(" <red>ERROR:</> checking if empty%s\n", err)
			continue
		}
		if empty {
			c.Printf("%s   <green>EMPTY</> - removing directory: ", intentStr)
			ktio.RunCommand(indent+4, f.Confirm, "rmdir", "-v", s.SrcPath())
			fmt.Println()
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
	}

	// post deletes re-check all series folders for emptiness
	c.Printf("\n\nChecking series and season folders for empties...\n")
	for _, s := range series {
		var srcSeasonNumbers []int
		for k := range s.SrcSeasons {
			srcSeasonNumbers = append(srcSeasonNumbers, k)
		}

		// Sort the srcSeasonNumbers slice
		sort.Ints(srcSeasonNumbers)

		// for each src season
		for _, seasonNum := range srcSeasonNumbers {
			ss := s.SrcSeasons[seasonNum]

			// if empty season remove it
			empty, err := ktio.FolderEmpty(ss.Path)
			if err != nil {
				c.Printf(" <red>ERROR:</> checking if empty%s\n", err)
				continue
			}
			if empty {
				ktio.RunCommand(4, f.Confirm, "rmdir", "-v", ss.Path)
			}

		}

		empty, err := ktio.FolderEmpty(s.SrcPath())
		if err != nil {
			c.Printf(" <red>ERROR:</> checking if empty%s\n", err)
			continue
		}
		if empty {
			ktio.RunCommand(4, f.Confirm, "rmdir", "-v", s.SrcPath())
			fmt.Println()
		}
	}

	return nil
}

func ProcessSpecialFiles(indent int, s content.Series, folder string, files []string, pathsToDelete *[]string) error {
	f := GetFlags()

	dstPath := path.Join(s.DstPath(), folder)
	if !ktio.PathExists(dstPath) {
		if err := os.MkdirAll(dstPath, 0755); err != nil {
			return fmt.Errorf("error creating specials directory: %w", err)
		}
	}

	for _, file := range files {
		c.Printf("%s       --> <white>%s</> move (y/n)? ", strings.Repeat(" ", indent), path.Base(file))
		if yes, err := ktio.Confirm(); err != nil {
			return fmt.Errorf("confirmation error: %w", err)
		} else if yes {
			if err := ktio.RunCommand(indent+6, f.Confirm, "mv", "-v", file, dstPath+"/"); err != nil {
				return fmt.Errorf("error moving file: %w", err)
			}
		}
	}

	empty, err := ktio.FolderEmpty(dstPath)
	if err != nil {
		return fmt.Errorf("error checking if specials directory is empty: %w", err)
	}
	if empty {
		c.Printf("%s   <green>EMPTY</> - removing directory: ", strings.Repeat(" ", indent))
		ktio.RunCommand(indent+4, f.Confirm, "rmdir", "-v", dstPath)
		fmt.Println()
	}

	return nil
}
