package cli

import (
	"errors"
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
			_ = s.MoveFolder(f.Confirm, 4)
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
				_ = ss.MoveFolder(f.Confirm, indent+4, s.DstPath()+"/")
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
					_ = se.MoveFiles(f.Confirm, indent+10, ds.Path+"/")
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
								if err := ktio.RunCommand(indent+10, f.Confirm, "mv", "-v", file, ds.Path+"/"); err != nil {
									c.Printf("          <red>ERROR:</> moving file: %s\n", err)
								}
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

				for _, srcVideo := range se.Videos {
					isSame := false
					for _, dstVideo := range de.Videos {
						if srcVideo.IsBasicallyTheSameTo(dstVideo) {
							isSame = true
							break
						}
					}

					if isSame {
						c.Printf("%s     <green>%dx%d</> --> SAME - adding to delete list\n", intentStr, seasonNum, episodeNum)
						pathsToDelete = append(pathsToDelete, srcVideo.Path)
						continue
					}

					if f.IgnoreExisting {
						c.Printf("%s     <magenta>%dx%d</> --> skipping due to flag\n", intentStr, seasonNum, episodeNum)
						continue
					}

					c.Printf("%s     <yellow>%dx%d</> --> <darkGray>%s</>\n", intentStr, seasonNum, episodeNum, ds.Path)

					// output video comparison table
					headers := []string{"Source"}
					for i := range de.Videos {
						headers = append(headers, fmt.Sprintf("Dest %d", i+1))
					}
					RenderVideoComparisonTable(2, headers, append([]content.VideoFile{srcVideo}, de.Videos...))

					var s rune
					if moveAll {
						s = 'A'
					} else if deleteAll {
						s = 'D'
					} else if skipAll {
						s = 'S'
					} else {
						c.Printf(" overwrite (y/a/A (all)?) delete src (d/D (all)?) skip (s/S?) quit (q?): ")
						s, err = ktio.GetSelection('a', 'y', 'd', 's', 'q', 'A', 'D', 'S')
						if err != nil {
							c.Printf(" <red>ERROR:</>%s\n", err)
							continue
						}
					}

					switch s {
					case 'A':
						moveAll = true
						fallthrough
					case 'a', 'y':
						// delete de files
						fmt.Println()
						for _, v := range de.Videos {
							if err := ktio.RunCommand(4, f.Confirm, "rm", "-v", v.Path); err != nil {
								c.Printf("    <red>ERROR:</> deleting destination video: %s\n", err)
							}
						}

						// move all se files
						if err := ktio.RunCommand(4, f.Confirm, "mv", "-v", srcVideo.Path, ds.Path+"/"); err != nil {
							c.Printf("    <red>ERROR:</> moving source video: %s\n", err)
						}
						for _, file := range se.OtherFiles {
							if err := ktio.RunCommand(4, f.Confirm, "mv", "-v", file, ds.Path+"/"); err != nil {
								c.Printf("    <red>ERROR:</> moving other file: %s\n", err)
							}
						}

					case 'D':
						deleteAll = true
						fallthrough
					case 'd':
						pathsToDelete = append(pathsToDelete, srcVideo.Path)
						fmt.Println()
					case 'S':
						skipAll = true
						continue
					case 's':
						continue
					case 'q':
						return errors.New("quitting")
					}
					fmt.Println()
				}
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
			_ = ProcessSpecialFiles(indent, s, "specials", s.SpecialFiles, &pathsToDelete)
		}

		if len(s.ExtraFiles) > 0 {
			c.Printf("%s   <magenta>%d extra files</> \n", intentStr, len(s.ExtraFiles))
			_ = ProcessSpecialFiles(indent, s, "extras", s.ExtraFiles, &pathsToDelete)
		}

		// if empty series folder then delete it
		empty, err := ktio.FolderEmpty(s.SrcPath())
		if err != nil {
			c.Printf(" <red>ERROR:</> checking if empty%s\n", err)
			continue
		}
		if empty {
			c.Printf("%s   <green>EMPTY</> - removing directory: ", intentStr)
			if err := ktio.RunCommand(indent+4, f.Confirm, "rmdir", "-v", s.SrcPath()); err != nil {
				c.Printf("    <red>ERROR:</> deleting source folder: %s\n", err)
			}
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
				if err := ktio.RunCommand(4, f.Confirm, "rm", "-rfv", path); err != nil {
					c.Printf("    <red>ERROR:</> deleting path: %s\n", err)
				}
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
				if err := ktio.RunCommand(4, f.Confirm, "rmdir", "-v", ss.Path); err != nil {
					c.Printf("    <red>ERROR:</> deleting season folder: %s\n", err)
				}
			}

		}

		empty, err := ktio.FolderEmpty(s.SrcPath())
		if err != nil {
			c.Printf(" <red>ERROR:</> checking if empty%s\n", err)
			continue
		}
		if empty {
			if err := ktio.RunCommand(4, f.Confirm, "rmdir", "-v", s.SrcPath()); err != nil {
				c.Printf("    <red>ERROR:</> deleting source folder: %s\n", err)
			}
			fmt.Println()
		}
	}

	return nil
}

func ProcessSpecialFiles(indent int, s content.Series, folder string, files []string, pathsToDelete *[]string) error {
	f := GetFlags()

	dstPath := path.Join(s.DstPath(), folder)
	if !ktio.PathExists(dstPath) {
		if err := os.MkdirAll(dstPath, 0o755); err != nil {
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
		if err := ktio.RunCommand(indent+4, f.Confirm, "rmdir", "-v", dstPath); err != nil {
			c.Printf("    <red>ERROR:</> deleting empty destination: %s\n", err)
		}
		fmt.Println()
	}

	return nil
}
