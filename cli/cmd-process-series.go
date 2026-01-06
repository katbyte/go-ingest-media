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
			if err := s.MoveFolder(f.Confirm, 4); err != nil {
				c.Printf(" <red>ERROR:</>%s\n\n", err)
			}
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
				if err := ss.MoveFolder(f.Confirm, indent+4, s.DstPath()+"/"); err != nil {
					c.Printf(" <red>ERROR:</>%s\n\n", err)
				}
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

			processedEpisodes := make(map[*content.Episode]bool)

			for _, episodeNum := range srcEpisodeNumbers {
				se := ss.Episodes[episodeNum]

				// check if already processed
				if processedEpisodes[se] {
					continue
				}
				processedEpisodes[se] = true

				// handle multiple source videos for the episode
				if len(se.Videos) > 1 {
					c.Printf("%s     <lightMagenta>WARNING - multiple source videos - WARNING</>\n", intentStr)

					headers := []string{}
					for i := range se.Videos {
						headers = append(headers, fmt.Sprintf("Source %d", i+1))
					}
					RenderVideoComparisonTable(indent+6, headers, se.Videos)

					c.Printf("%s     pick source to keep (1-%d) skip (s) quit (q): ", intentStr, len(se.Videos))
					options := []rune{'s', 'q'}
					for k := 1; k <= len(se.Videos) && k <= 9; k++ {
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
					keptVideo := se.Videos[keepIdx]

					for idx, v := range se.Videos {
						if idx != keepIdx {
							if err := ktio.RunCommand(indent+6, f.Confirm, "rm", "-v", v.Path); err != nil {
								c.Printf("      <red>ERROR:</> deleting source video: %s\n", err)
							}
						}
					}
					se.Videos = []content.VideoFile{keptVideo}
					// update map
					ss.Episodes[episodeNum] = se
				}

				// see if there is a dst episode
				de, exists := ds.Episodes[episodeNum]
				if !exists {
					// move episode files
					c.Printf("%s     <green>%dx%d</> --> ", intentStr, seasonNum, episodeNum)
					if err := se.MoveFiles(f.Confirm, indent+10, ds.Path+"/"); err != nil {
						c.Printf("      <red>ERROR:</> moving files: %s\n", err)
					}
					continue
				}

				if len(se.Videos) == 0 {
					c.Printf("%s     <yellow>%dx%d</> --> source has no video file, copying other files except nfo\n", intentStr, seasonNum, episodeNum)

					// for each source file move it unless it is a nfo file
					for _, file := range se.OtherFiles {
						if strings.HasSuffix(file, ".nfo") {
							c.Printf("%s           --> nfo, deleting\n", intentStr)
							if err := ktio.RunCommand(indent+10, f.Confirm, "rm", "-v", file); err != nil {
								c.Printf("          <red>ERROR:</> deleting nfo: %s\n", err)
							}
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
					c.Printf("%s     <red>%dx%d</> --> <yellow>WARNING</> - dst has no video file, moving source\n", intentStr, seasonNum, episodeNum)
					if err := se.MoveFiles(f.Confirm, indent+10, ds.Path+"/"); err != nil {
						c.Printf("      <red>ERROR:</> moving files: %s\n", err)
					}
					continue
				}

				// we take the first source video, as we have already handled multiple source videos above
				srcVideo := se.Videos[0]
				isSame := false
				for _, dstVideo := range de.Videos {
					if srcVideo.IsBasicallyTheSameTo(dstVideo) {
						isSame = true
						break
					}
				}

				if isSame {
					c.Printf("%s     <green>%dx%d</> --> SAME - deleting source and syncing extras\n", intentStr, seasonNum, episodeNum)
					if err := ktio.RunCommand(indent+10, f.Confirm, "rm", "-v", srcVideo.Path); err != nil {
						c.Printf("      <red>ERROR:</> deleting source video: %s\n", err)
					}
					// move extras
					if err := se.MoveExtras(f.Confirm, indent+10, ds.Path+"/"); err != nil {
						c.Printf("      <red>ERROR:</> moving extras: %s\n", err)
					}
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
				switch {
				case moveAll:
					s = 'A'
				case deleteAll:
					s = 'D'
				case skipAll:
					s = 'S'
				default:
					c.Printf(" overwrite (y/a/A (all)?) delete src (d/D (all)?) pick dest (1-%d) skip (s/S?) quit (q?): ", len(de.Videos))
					options := []rune{'a', 'y', 'd', 's', 'q', 'A', 'D', 'S'}
					for k := 1; k <= len(de.Videos) && k <= 9; k++ {
						options = append(options, rune('0'+k))
					}
					s, err = ktio.GetSelection(options...)
					if err != nil {
						c.Printf(" <red>ERROR:</>%s\n", err)
						continue
					}
					fmt.Println()
				}

				switch s {
				case 'A':
					moveAll = true
					deleteAll = false
					skipAll = false
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
					if err := se.MoveFiles(f.Confirm, 4, ds.Path+"/"); err != nil {
						c.Printf("    <red>ERROR:</> moving files: %s\n", err)
					}

				case '1', '2', '3', '4', '5', '6', '7', '8', '9':
					keepIdx := int(s-'0') - 1

					// delete destination video files except the selected one
					// and update destination struct in case there is another source
					var newDst []content.VideoFile
					for idx, v := range de.Videos {
						if idx == keepIdx {
							newDst = append(newDst, v)
						} else {
							if err := ktio.RunCommand(4, f.Confirm, "rm", "-v", v.Path); err != nil {
								c.Printf("    <red>ERROR:</> deleting destination video: %s\n", err)
							}
						}
					}
					de.Videos = newDst
					// update map
					ds.Episodes[episodeNum] = de

					// delete the source video
					if err := ktio.RunCommand(4, f.Confirm, "rm", "-v", srcVideo.Path); err != nil {
						c.Printf("    <red>ERROR:</> deleting source video: %s\n", err)
					}

				case 'D':
					deleteAll = true
					moveAll = false
					skipAll = false
					fallthrough
				case 'd':
					if err := ktio.RunCommand(4, f.Confirm, "rm", "-v", srcVideo.Path); err != nil {
						c.Printf("    <red>ERROR:</> deleting source video: %s\n", err)
					}
				case 'S':
					skipAll = true
					moveAll = false
					deleteAll = false
					continue
				case 's':
					continue
				case 'q':
					return errors.New("quitting")
				}
				fmt.Println()
			}

			// if empty season (or only nfo files) remove it
			if err := ktio.DeleteIfEmptyOrOnlyNfo(ss.Path, f.Confirm, indent+6); err != nil {
				c.Printf("      <red>ERROR:</> cleaning up season folder: %s\n", err)
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

		// cleanup empty specials/extras first (and any remaining nfo files)
		for _, sub := range []string{"specials", "extras"} {
			subPath := fmt.Sprintf("%s/%s", s.SrcPath(), sub)
			if ktio.PathExists(subPath) {
				if err := ktio.DeleteIfEmptyOrOnlyNfo(subPath, f.Confirm, indent); err != nil {
					c.Printf(" <red>ERROR:</> deleting %s folder: %s\n", sub, err)
				}
			}
		}

		// if empty series folder (or only nfo files) then delete it
		if err := ktio.DeleteIfEmptyOrOnlyNfo(s.SrcPath(), f.Confirm, indent); err != nil {
			c.Printf(" <red>ERROR:</> deleting series folder: %s\n", err)
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
		if err := os.MkdirAll(dstPath, 0o750); err != nil {
			return fmt.Errorf("error creating specials directory: %w", err)
		}
	}

	moveAll := false
	for _, file := range files {
		shouldMove := moveAll
		if !moveAll {
			c.Printf("%s       --> <white>%s</> move (y/n/a)? ", strings.Repeat(" ", indent), path.Base(file))
			s, err := ktio.GetSelection('y', 'n', 'a')
			fmt.Println() // newline after selection
			if err != nil {
				return fmt.Errorf("selection error: %w", err)
			}
			switch s {
			case 'a':
				moveAll = true
				shouldMove = true
			case 'y':
				shouldMove = true
			}
		} else {
			c.Printf("%s       --> <white>%s</> moving...\n", strings.Repeat(" ", indent), path.Base(file))
		}

		if shouldMove {
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
