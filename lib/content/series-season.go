package content

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	c "github.com/gookit/color"
	"github.com/katbyte/go-ingest-media/lib/ktio"
)

// adds to the content type (folder) by adding singular video details as 1 movie has 1 video file

type Season struct {
	Number int
	Year   int
	Path   string

	Episodes map[int]*Episode
}

type Episode struct {
	Season         int
	Number         int
	EpisodeNumbers []int // list of episodes this file represents

	Videos []VideoFile

	OtherFiles []string // we want to know about these, so we can move them all, we don't care about dest files?
}

func GetSeasons(path string) (map[int]Season, error) {
	// for each folder in source path
	srcFolders, err := ktio.ListFolders(path)
	if err != nil {
		return nil, fmt.Errorf("error listing source folders: %w", err)
	}

	var wg sync.WaitGroup
	errorChan := make(chan error, len(srcFolders))
	doneChan := make(chan bool)

	seasons := make(map[int]Season)
	var mutex sync.Mutex // Mutex to synchronise access to the seasons map

	for _, f := range srcFolders {
		wg.Add(1)

		go func(f string) {
			defer wg.Done()
			s := Season{
				Path: f,
			}

			// Updated regex to match folders in the format "SeriesSource Name - s##", "SeriesSource Name - s## (####)", or "SeriesSource Name - s## ()"
			isSeason, err := regexp.MatchString(`.* - s(\d+)(?: \((\d*)\))?`, f)
			if err != nil {
				errorChan <- fmt.Errorf("error matching season folder: %w", err)
				return
			}
			if !isSeason {
				c.Printf("    <darkGray>SKIP:</> folder doesn't match season format: %s\n", filepath.Base(f))
				return // Skip folders not matching the format
			}

			// get season number and year (if present) from folder name
			re := regexp.MustCompile(`.* - s(\d+)(?: \((\d*)\))?`)
			matches := re.FindStringSubmatch(f)
			s.Number, _ = strconv.Atoi(matches[1]) // Convert season number to int

			if len(matches) > 2 && matches[2] != "" {
				s.Year, _ = strconv.Atoi(matches[2]) // Convert year to int, if present
			}

			// Get the episodes in a season
			if err := s.LoadEpisodes(); err != nil {
				errorChan <- fmt.Errorf("error loading episodes: %w", err)
				return
			}

			// Lock the mutex to prevent concurrent writes to the seasons map
			mutex.Lock()

			// warn if season already exists (don't error - just use the first one found)
			if existing, ok := seasons[s.Number]; ok {
				c.Printf("    <yellow>WARNING:</> season %d already exists (%s vs %s), using first\n", s.Number, existing.Path, s.Path)
			} else {
				seasons[s.Number] = s
			}
			mutex.Unlock()
		}(f)
	}
	go func() {
		wg.Wait()
		close(doneChan)
	}()

	for {
		select {
		case err := <-errorChan:
			return nil, err
		case <-doneChan:
			return seasons, nil
		}
	}
}

func (s *Season) LoadEpisodes() error {
	// for each file in season path
	files, err := ktio.ListFiles(s.Path)
	if err != nil {
		return fmt.Errorf("error listing source files: %w", err)
	}

	s.Episodes = make(map[int]*Episode) // Initialise the Episodes map

	// match 01x01, 01x01-02, 01x01+02
	episodeRegex := regexp.MustCompile(`.* - (\d+)x(\d+)(?:([-+])(\d+))? - .*`)

	for _, file := range files {
		// Check if the file name matches the episode format
		if matches := episodeRegex.FindStringSubmatch(file); matches != nil {
			episodeNumber, err := strconv.Atoi(matches[2]) // Convert episode number to int
			if err != nil {
				return fmt.Errorf("error parsing episode number: %w", err)
			}

			// check for multi-episode
			episodeNumbers := []int{episodeNumber}
			if len(matches) > 4 && matches[4] != "" {
				separator := matches[3]
				endEpisodeNumber, err := strconv.Atoi(matches[4])
				if err != nil {
					return fmt.Errorf("error parsing end episode number: %w", err)
				}

				if endEpisodeNumber > episodeNumber {
					if separator == "-" {
						// range: add all intermediate episodes
						for i := episodeNumber + 1; i <= endEpisodeNumber; i++ {
							episodeNumbers = append(episodeNumbers, i)
						}
					} else {
						// plus: add just the end episode (assuming "01+03" means 1 and 3)
						episodeNumbers = append(episodeNumbers, endEpisodeNumber)
					}
				}
			}

			// find or create episode
			var episode *Episode

			// check if any of the episode numbers already exist
			for _, num := range episodeNumbers {
				if existing, exists := s.Episodes[num]; exists {
					episode = existing
					break
				}
			}

			if episode == nil {
				episode = &Episode{
					Season:         s.Number,
					Number:         episodeNumber,
					EpisodeNumbers: episodeNumbers,
					OtherFiles:     []string{},
					Videos:         []VideoFile{},
				}
			}

			// Determine if the file is a video file or another type (like subtitles)
			if IsVideoFile(file) {
				v, err := VideoFor(file)
				if err != nil {
					return fmt.Errorf("error loading source video: %w", err)
				}
				
				episode.Videos = append(episode.Videos, *v)
			} else {
				episode.OtherFiles = append(episode.OtherFiles, file)
			}

			// map all episode numbers to this episode instance
			for _, num := range episodeNumbers {
				s.Episodes[num] = episode
			}
		}
	}

	return nil
}

func (s *Season) MoveFolder(confirm bool, indent int, dstPath string) error {
	return ktio.RunCommand(indent, confirm, "mv", "-v", s.Path, dstPath)
}

func (e *Episode) MoveFiles(confirm bool, indent int, dstPath string) error {
	// ensure there is only 1 source video file
	if len(e.Videos) != 1 {
		return fmt.Errorf("expected 1 src video file, found %d", len(e.Videos))
	}

	// movie video file
	// movie video file
	if err := ktio.RunCommand(indent, confirm, "mv", "-v", e.Videos[0].Path, dstPath); err != nil {
		c.Printf("   <red>ERROR:</> moving video: %s\n", err)
	}

	return e.MoveExtras(confirm, indent, dstPath)
}

func (e *Episode) MoveExtras(confirm bool, indent int, dstPath string) error {
	// move all other files
	for _, file := range e.OtherFiles {
		// skip nfo files
		if filepath.Ext(file) == ".nfo" {
			continue
		}

		// calculate indent from "seasonXepisode -->"
		epNumStr := strconv.Itoa(e.Number)
		if len(e.EpisodeNumbers) > 1 {
			// if multi episode, assume last is the end
			epNumStr = fmt.Sprintf("%d-%d", e.EpisodeNumbers[0], e.EpisodeNumbers[len(e.EpisodeNumbers)-1])
		}
		fmt.Printf("%s --> ", strings.Repeat(" ", indent-len(strconv.Itoa(e.Season))+1+len(epNumStr)))
		if err := ktio.RunCommand(indent, confirm, "mv", "-v", file, dstPath); err != nil {
			c.Printf("   <red>ERROR:</> moving other file: %s\n", err)
		}
	}

	return nil
}

func (e *Episode) DeleteVideoFiles() {
	for _, v := range e.Videos {
		if err := ktio.RunCommand(0, false, "rm", "-v", v.Path); err != nil {
			c.Printf("   <red>ERROR:</> deleting destination video: %s\n", err)
		}
	}
}
