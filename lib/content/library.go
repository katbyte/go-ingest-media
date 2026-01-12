package content

import (
	"fmt"
	"path/filepath"

	"github.com/katbyte/go-ingest-media/lib/ktio"
)

type LibraryType int

const (
	LibraryTypeUnknown LibraryType = iota
	LibraryTypeSeries
	LibraryTypeMovies
	LibraryTypeStandup
)

// Library represents a single library location (either source or destination)
type Library struct {
	Path          string // full absolute path
	Type          LibraryType
	LetterFolders bool
}

// LibraryMapping joins a source library to a destination library for processing
type LibraryMapping struct {
	Source *Library
	Dest   *Library
}

// FolderRenames returns the folder renames for this mapping's library type
func (m LibraryMapping) FolderRenames() []FolderMapping {
	return folderRenames[m.Source.Type]
}

// Libraries - all known library locations (using pointers for direct access)
var Libraries = map[string]*Library{
	// Torrent (sorted) libraries - source (/mnt/ztmp/torrents/sorted/...)
	"torrent-anime-movies": {Path: "/mnt/ztmp/torrents/sorted/m.anime", Type: LibraryTypeMovies},
	"torrent-movies":       {Path: "/mnt/ztmp/torrents/sorted/m.movies", Type: LibraryTypeMovies},
	"torrent-documentary":  {Path: "/mnt/ztmp/torrents/sorted/m.docu", Type: LibraryTypeMovies},
	"torrent-standup":      {Path: "/mnt/ztmp/torrents/sorted/m.standup", Type: LibraryTypeStandup},
	"torrent-anime-series": {Path: "/mnt/ztmp/torrents/sorted/s.anime", Type: LibraryTypeSeries},
	"torrent-tv":           {Path: "/mnt/ztmp/torrents/sorted/s.tv", Type: LibraryTypeSeries},
	"torrent-docuseries":   {Path: "/mnt/ztmp/torrents/sorted/s.docu", Type: LibraryTypeSeries},

	// Video libraries - destination (/mnt/video/...)
	"video-anime-movies": {Path: "/mnt/video/anime/movies", Type: LibraryTypeMovies},
	"video-movies":       {Path: "/mnt/video/movies", Type: LibraryTypeMovies, LetterFolders: true},
	"video-documentary":  {Path: "/mnt/video/docu/documentary", Type: LibraryTypeMovies},
	"video-standup":      {Path: "/mnt/video/standup", Type: LibraryTypeStandup},
	"video-anime-series": {Path: "/mnt/video/anime/series", Type: LibraryTypeSeries, LetterFolders: true},
	"video-tv":           {Path: "/mnt/video/tv", Type: LibraryTypeSeries, LetterFolders: true},
	"video-docuseries":   {Path: "/mnt/video/docu/docuseries", Type: LibraryTypeSeries},
}

// LibraryMappingSortedTorrentsImport - mappings from source to destination for importing sorted torrents
var LibraryMappingSortedTorrentsImport = map[string]LibraryMapping{
	"anime-movies": {Source: Libraries["torrent-anime-movies"], Dest: Libraries["video-anime-movies"]},
	"movies":       {Source: Libraries["torrent-movies"], Dest: Libraries["video-movies"]},
	"documentary":  {Source: Libraries["torrent-documentary"], Dest: Libraries["video-documentary"]},
	"standup":      {Source: Libraries["torrent-standup"], Dest: Libraries["video-standup"]},
	"anime-series": {Source: Libraries["torrent-anime-series"], Dest: Libraries["video-anime-series"]},
	"tv":           {Source: Libraries["torrent-tv"], Dest: Libraries["video-tv"]},
	"docuseries":   {Source: Libraries["torrent-docuseries"], Dest: Libraries["video-docuseries"]},
}

// Contents scans this library and returns all content items
func (l *Library) Contents(onContentError func(folder string, err error)) ([]ContentInterface, error) {
	folders, err := ktio.ListFolders(l.Path)
	if err != nil {
		return nil, fmt.Errorf("error listing content folders: %w", err)
	}

	var contents []ContentInterface

	if !l.LetterFolders {
		for _, f := range folders {
			var c ContentInterface

			switch l.Type {
			case LibraryTypeMovies, LibraryTypeStandup:
				c, err = MovieFor(l, f)
			case LibraryTypeSeries:
				c, err = SeriesFor(l, f)
			case LibraryTypeUnknown:
				fallthrough
			default:
				return nil, fmt.Errorf("unknown library type: %d", l.Type)
			}
			if err != nil {
				onContentError(f, err)
				continue
			}

			contents = append(contents, c)
		}

		return contents, nil
	}

	// letter folders - folders already contains full paths
	for _, letterFolder := range folders {
		subFolders, err := ktio.ListFolders(letterFolder)
		if err != nil {
			onContentError(filepath.Base(letterFolder), err)
			continue
		}

		for _, lf := range subFolders {
			var c ContentInterface

			switch l.Type {
			case LibraryTypeMovies, LibraryTypeStandup:
				c, err = MovieFor(l, lf)
			case LibraryTypeSeries:
				c, err = SeriesFor(l, lf)
			case LibraryTypeUnknown:
				fallthrough
			default:
				return nil, fmt.Errorf("unknown library type: %d", l.Type)
			}
			if err != nil {
				onContentError(filepath.Base(lf), err)
				continue
			}

			contents = append(contents, c)
		}
	}

	return contents, nil
}

// Movies returns all movies from this library
func (l *Library) Movies(onContentError func(folder string, err error)) ([]Movie, error) {
	contents, err := l.Contents(onContentError)
	if err != nil {
		return nil, fmt.Errorf("error getting movies: %w", err)
	}

	movies := make([]Movie, 0, len(contents))

	for _, c := range contents {
		m := *c.(*Movie)
		movies = append(movies, m)
	}

	return movies, nil
}

// Series returns all series from this library
func (l *Library) Series(onContentError func(folder string, err error)) ([]Series, error) {
	contents, err := l.Contents(onContentError)
	if err != nil {
		return nil, fmt.Errorf("error getting series: %w", err)
	}

	series := make([]Series, 0, len(contents))

	for _, c := range contents {
		s := *c.(*Series)
		series = append(series, s)
	}

	return series, nil
}
