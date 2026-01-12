package content

import (
	"fmt"
	"path/filepath"
	"strings"

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
	Path          string // full absolute path, set by InitLibraries
	SubPath       string // relative folder path (e.g., "m.movies" or "movies")
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
	// Torrent (sorted) libraries - source
	"torrent-anime-movies": {SubPath: "m.anime", Type: LibraryTypeMovies},
	"torrent-movies":       {SubPath: "m.movies", Type: LibraryTypeMovies},
	"torrent-documentary":  {SubPath: "m.docu", Type: LibraryTypeMovies},
	"torrent-standup":      {SubPath: "m.standup", Type: LibraryTypeStandup},
	"torrent-anime-series": {SubPath: "s.anime", Type: LibraryTypeSeries},
	"torrent-tv":           {SubPath: "s.tv", Type: LibraryTypeSeries},
	"torrent-docuseries":   {SubPath: "s.docu", Type: LibraryTypeSeries},

	// Video libraries - destination
	"video-anime-movies": {SubPath: "anime/movies", Type: LibraryTypeMovies},
	"video-movies":       {SubPath: "movies", Type: LibraryTypeMovies, LetterFolders: true},
	"video-documentary":  {SubPath: "docu/documentary", Type: LibraryTypeMovies},
	"video-standup":      {SubPath: "standup", Type: LibraryTypeStandup},
	"video-anime-series": {SubPath: "anime/series", Type: LibraryTypeSeries, LetterFolders: true},
	"video-tv":           {SubPath: "tv", Type: LibraryTypeSeries, LetterFolders: true},
	"video-docuseries":   {SubPath: "docu/docuseries", Type: LibraryTypeSeries},
}

// LibraryMappings - mappings from source to destination (using direct pointers)
var LibraryMappings = map[string]LibraryMapping{
	"anime-movies": {Source: Libraries["torrent-anime-movies"], Dest: Libraries["video-anime-movies"]},
	"movies":       {Source: Libraries["torrent-movies"], Dest: Libraries["video-movies"]},
	"documentary":  {Source: Libraries["torrent-documentary"], Dest: Libraries["video-documentary"]},
	"standup":      {Source: Libraries["torrent-standup"], Dest: Libraries["video-standup"]},
	"anime-series": {Source: Libraries["torrent-anime-series"], Dest: Libraries["video-anime-series"]},
	"tv":           {Source: Libraries["torrent-tv"], Dest: Libraries["video-tv"]},
	"docuseries":   {Source: Libraries["torrent-docuseries"], Dest: Libraries["video-docuseries"]},
}

// Library folder name mappings (library key -> relative folder path)
// SubPath information is now stored directly in the Library struct; the separate libraryFolders map is no longer needed.

// InitLibraries initialises library paths with the given base paths
func InitLibraries(srcBasePath, dstBasePath string) {
	for key, lib := range Libraries {
		if lib.SubPath == "" {
			continue
		}
		if strings.HasPrefix(key, "torrent") {
			lib.Path = filepath.Join(srcBasePath, lib.SubPath)
		} else {
			lib.Path = filepath.Join(dstBasePath, lib.SubPath)
		}
		// No need to reassign - we're modifying the pointer directly
	}
}

// GetLibraryMappings returns all library mappings (for CLI iteration)
func GetLibraryMappings() map[string]LibraryMapping {
	return LibraryMappings
}

// GetLibraryMapping returns a specific library mapping by key
func GetLibraryMapping(key string) (LibraryMapping, bool) {
	m, ok := LibraryMappings[key]
	return m, ok
}

// Contents scans this library and returns all content items
func (l Library) Contents(onContentError func(folder string, err error)) ([]ContentInterface, error) {
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

	// letter folders
	for _, f := range folders {
		letterFolders, err := ktio.ListFolders(filepath.Join(l.Path, f))
		if err != nil {
			onContentError(f, err)
			continue
		}

		for _, lf := range letterFolders {
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
				onContentError(lf, err)
				continue
			}

			contents = append(contents, c)
		}
	}

	return contents, nil
}

// Movies returns all movies from this library
func (l Library) Movies(onContentError func(folder string, err error)) ([]Movie, error) {
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
func (l Library) Series(onContentError func(folder string, err error)) ([]Series, error) {
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
