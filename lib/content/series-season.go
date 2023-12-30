package content

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/katbyte/go-ingest-media/lib/ktio"
)

// adds to the content type (folder) by adding singular video details as 1 movie has 1 video file

type Season struct {
	Number int
	Year   int
	Path   string

	Episodes map[int]Episode
}

type Episode struct {
	Season int
	Number int

	Videos []VideoFile

	Files []string // we want to know about these, so we can move them all, we don't care about dest files?
}

func GetSeasons(path string) (map[int]Season, error) {

	// for each folder in source path
	srcFolders, err := ktio.ListFolders(path)
	if err != nil {
		return nil, fmt.Errorf("error listing source folders: %w", err)
	}

	seasons := make(map[int]Season)
	for _, f := range srcFolders {
		s := Season{
			Path: f,
		}

		// Updated regex to match folders in the format "Series Name - s##", "Series Name - s## (####)", or "Series Name - s## ()"
		isSeason, err := regexp.MatchString(`.* - s(\d+)(?: \((\d*)\))?`, f)
		if err != nil {
			return nil, fmt.Errorf("error parsing folder name: %w", err)
		}
		if !isSeason {
			continue // Skip folders not matching the format
		}

		// get season number and year (if present) from folder name
		re := regexp.MustCompile(`.* - s(\d+)(?: \((\d*)\))?`)
		matches := re.FindStringSubmatch(f)
		s.Number, _ = strconv.Atoi(matches[1]) // Convert season number to int

		if len(matches) > 2 && matches[2] != "" {
			s.Year, _ = strconv.Atoi(matches[2]) // Convert year to int, if present
		}

		// Get the episodes in a season
		s.LoadEpisodes()

		// error if season already exists
		if _, ok := seasons[s.Number]; ok {
			return nil, fmt.Errorf("season %d already exists", s.Number)
		}
		seasons[s.Number] = s
	}

	return seasons, nil
}

func (s *Season) LoadEpisodes() error {
	// for each file in season path
	files, err := ktio.ListFiles(s.Path)
	if err != nil {
		return fmt.Errorf("error listing source files: %w", err)
	}

	s.Episodes = make(map[int]Episode) // Initialize the Episodes map

	episodeRegex := regexp.MustCompile(`.* - (\d+)x(\d+) - .*`)

	for _, file := range files {

		// Check if the file name matches the episode format
		if matches := episodeRegex.FindStringSubmatch(file); matches != nil {
			episodeNumber, err := strconv.Atoi(matches[2]) // Convert episode number to int
			if err != nil {
				return fmt.Errorf("error parsing episode number: %w", err)
			}

			episode, exists := s.Episodes[episodeNumber]
			if !exists {
				episode = Episode{
					Season: s.Number,
					Number: episodeNumber,
					Files:  []string{},
					Videos: []VideoFile{},
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
				episode.Files = append(episode.Files, file)
			}

			s.Episodes[episodeNumber] = episode
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
	ktio.RunCommand(indent, confirm, "mv", "-v", e.Videos[0].Path, dstPath)

	// move all other files
	for _, file := range e.Files {
		// calculate indent from "seasonXepisode -->"
		fmt.Printf("%s --> ", strings.Repeat(" ", indent-len(strconv.Itoa(e.Season))+1+len(strconv.Itoa(e.Number))))
		ktio.RunCommand(indent, confirm, "mv", "-v", file, dstPath)
	}

	return nil
}
