package cli

import (
	"fmt"
	"path"
	"sort"
	"strconv"
	"strings"

	c "github.com/gookit/color"
	"github.com/katbyte/go-ingest-media/lib/content"
	"github.com/katbyte/go-ingest-media/lib/ktio"
	_ "github.com/mattn/go-sqlite3"
)

func ProcessSeries(l content.Library) error {
	f := GetFlags()

	series, err := l.Series(func(f string, err error) {
		c.Printf("  %s --> <red>ERROR:</>%s</>\n", path.Base(f), err)
	})
	if err != nil {
		return fmt.Errorf("error getting series: %w", err)
	}

	sort.Slice(series, func(i, j int) bool {
		return series[i].Letter+"/"+series[i].DstFolder < series[j].Letter+"/"+series[j].DstFolder
	})

	// pathsToDelete := []string{}

	i := 0
	nMovies := len(series)
	for _, s := range series {
		i++

		// TODO
		// TODO
		// TODO use go channels to run multiple moves at once/queue them up in the background
		// TODO
		// TODO ALSO maybe do the easy movies first then do the prompting ones after

		// if not exists just move folder nice and easy like
		if !s.DstExists() {
			c.Printf("<darkGray>%d/%d</> <white>%s</> --> <green>%s</>", i, nMovies, s.SrcFolder, s.DstFolder)
			s.MoveFolder(f.Confirm, 4)
			fmt.Println()
			continue
		}

		// exists so lets grab the video details
		c.Printf("<darkGray>%d/%d</>  <white>%s</> --> <yellow>%s</>\n", i, nMovies, s.SrcFolder, s.DstFolder)
		// load video details
		if err = s.LoadSeasons(); err != nil {
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

		// todo there must be a better way to do indentation...

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

			c.Printf("%s   season <yellow>%d</>:\n", intentStr, seasonNum)

			var srcEpisodeNumbers []int
			for k := range ss.Episodes {
				srcEpisodeNumbers = append(srcEpisodeNumbers, k)
			}

			// Sort the srcSeasonNumbers slice
			sort.Ints(srcEpisodeNumbers)

			// for each episode in src season
			for _, episodeNum := range srcEpisodeNumbers {
				se := ss.Episodes[episodeNum]

				// see if there is a dst episode
				_, exists := ds.Episodes[episodeNum]
				if !exists {
					// move episode files
					c.Printf("%s     <green>%dx%d</> --> ", intentStr, seasonNum, episodeNum)
					se.MoveFiles(f.Confirm, indent+10, ds.Path+"/")
					continue
				}

				if f.IgnoreExisting {
					c.Printf("  <magenta>EXISTING</> - skipping due to flag\n\n\n")
					continue
				}

				c.Printf("%s     <yellow>%dx%d</> --> ", intentStr, seasonNum, episodeNum)
				// move video files
				// todo
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

		// if empty series folder then delete it
		empty, err := ktio.FolderEmpty(s.SrcPath())
		if err != nil {
			c.Printf(" <red>ERROR:</> checking if empty%s\n", err)
			continue
		}
		if empty {
			c.Printf("%s   <green>EMPTY</> - removing directory: ", intentStr)
			ktio.RunCommand(indent+4, f.Confirm, "rmdir", "-v", s.SrcPath())
		}
		fmt.Println()

		// for each episode in each season

	}
	return nil
}
