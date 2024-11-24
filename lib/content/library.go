package content

import (
	"fmt"
	"github.com/katbyte/go-ingest-media/lib/ktio"
	"path/filepath"
)

type LibraryType int

const (
	LibraryTypeUnknown LibraryType = iota
	LibraryTypeSeries
	LibraryTypeMovies
	LibraryTypeStandup
)

type Library struct {
	ID            string
	SrcFolder     string
	DstFolder     string
	SrcPath       string
	DstPath       string
	Type          LibraryType
	LetterFolders bool

	Mappings []Mapping
}

// filebot -> t/s & t/m, manual move to m.anime s.tv etc
var libraries = []Library{
	{ID: "anime-movies", SrcFolder: "m.anime", DstFolder: "anime/movies", Type: LibraryTypeMovies, LetterFolders: false},
	{ID: "movies", SrcFolder: "m.movies", DstFolder: "movies", Type: LibraryTypeMovies, LetterFolders: true},
	{ID: "documentary", SrcFolder: "m.docu", DstFolder: "docu/documentary", Type: LibraryTypeMovies},
	{ID: "standup", SrcFolder: "m.standup", DstFolder: "standup", Type: LibraryTypeStandup},
	{ID: "anime-series", SrcFolder: "s.anime", DstFolder: "anime/series", Type: LibraryTypeSeries, LetterFolders: true},
	{ID: "tv", SrcFolder: "s.tv", DstFolder: "tv", Type: LibraryTypeSeries, LetterFolders: true},
	{ID: "docuseries", SrcFolder: "s.docu", DstFolder: "docu/docuseries", Type: LibraryTypeSeries},
}

func GetLibraries(srcPath, dstPath string) []Library {
	updated := make([]Library, len(libraries))

	for i, l := range libraries {
		l2 := l
		l2.SrcPath = filepath.Join(srcPath, l2.SrcFolder)
		l2.DstPath = filepath.Join(dstPath, l2.DstFolder)
		l2.Mappings = l2.mappings()
		updated[i] = l2
	}

	return updated
}

func GetLibrariesMap(srcPath, dstPath string) map[string]Library {
	libs := GetLibraries(srcPath, dstPath)
	m := make(map[string]Library)

	for _, l := range libs {
		m[l.ID] = l
	}

	return m
}

func (l Library) ContentsSource(onContentError func(folder string, err error)) ([]ContentInterface, error) {
	var contents []ContentInterface

	folders, err := ktio.ListFolders(l.SrcPath)
	if err != nil {
		return nil, fmt.Errorf("error listing content folders: %w", err)
	}

	for _, f := range folders {
		var c ContentInterface

		if l.Type == LibraryTypeMovies || l.Type == LibraryTypeStandup { // standup is the same for now except a slighty different alt folder
			c, err = l.MovieFor(f)
		} else if l.Type == LibraryTypeSeries {
			c, err = l.SeriesFor(f)
		} else {
			return nil, fmt.Errorf("unknown library type: %s", l.Type)
		}
		if err != nil {
			onContentError(f, err)
			continue
		}

		contents = append(contents, c)
	}

	return contents, nil
}

func (l Library) MoviesSource(onContentError func(folder string, err error)) ([]Movie, error) {
	var movies []Movie

	contents, err := l.ContentsSource(onContentError)
	if err != nil {
		return nil, fmt.Errorf("error getting movies: %w", err)
	}

	for _, c := range contents {
		m := *c.(*Movie)
		movies = append(movies, m)
	}

	return movies, nil
}

func (l Library) SeriesSource(onContentError func(folder string, err error)) ([]Series, error) {
	var movies []Series

	contents, err := l.ContentsSource(onContentError)
	if err != nil {
		return nil, fmt.Errorf("error getting movies: %w", err)
	}

	for _, c := range contents {
		m := *c.(*Series)
		movies = append(movies, m)
	}

	return movies, nil
}

func (l Library) ContentsDestination(onContentError func(folder string, err error)) ([]ContentInterface, error) {
	var contents []ContentInterface

	folders, err := ktio.ListFolders(l.DstPath)
	if err != nil {
		return nil, fmt.Errorf("error listing content folders: %w", err)
	}

	if !l.LetterFolders {
		for _, f := range folders {
			var c ContentInterface

			if l.Type == LibraryTypeMovies || l.Type == LibraryTypeStandup { // standup is the same for now except a slighty different alt folder
				c, err = l.MovieFor(f)
			} else if l.Type == LibraryTypeSeries {
				c, err = l.SeriesFor(f)
			} else {
				return nil, fmt.Errorf("unknown library type: %s", l.Type)
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
		letterFolders, err := ktio.ListFolders(filepath.Join(l.DstPath, f))
		if err != nil {
			onContentError(f, err)
			continue
		}

		for _, lf := range letterFolders {
			var c ContentInterface

			if l.Type == LibraryTypeMovies || l.Type == LibraryTypeStandup { // standup is the same for now except a slighty different alt folder
				c, err = l.MovieFor(lf)
			} else if l.Type == LibraryTypeSeries {
				c, err = l.SeriesFor(lf)
			} else {
				return nil, fmt.Errorf("unknown library type: %s", l.Type)
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
