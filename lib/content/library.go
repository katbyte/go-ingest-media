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

type Library struct {
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
	{SrcFolder: "m.anime", DstFolder: "anime/movies", Type: LibraryTypeMovies, LetterFolders: false},
	{SrcFolder: "m.movies", DstFolder: "movies", Type: LibraryTypeMovies, LetterFolders: true},
	{SrcFolder: "m.docu", DstFolder: "docu/documentary", Type: LibraryTypeMovies},
	{SrcFolder: "m.standup", DstFolder: "standup", Type: LibraryTypeStandup},
	{SrcFolder: "s.anime", DstFolder: "anime/series", Type: LibraryTypeSeries, LetterFolders: true},
	{SrcFolder: "s.tv", DstFolder: "tv", Type: LibraryTypeSeries, LetterFolders: true},
	{SrcFolder: "s.docu", DstFolder: "docu/docuseries", Type: LibraryTypeSeries},
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

func (l Library) Contents(onContentError func(folder string, err error)) ([]ContentInterface, error) {
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

func (l Library) Movies(onContentError func(folder string, err error)) ([]Movie, error) {
	var movies []Movie

	contents, err := l.Contents(onContentError)
	if err != nil {
		return nil, fmt.Errorf("error getting movies: %w", err)
	}

	for _, c := range contents {
		m := *c.(*Movie)
		movies = append(movies, m)
	}

	return movies, nil
}

func (l Library) Series(onContentError func(folder string, err error)) ([]Series, error) {
	var movies []Series

	contents, err := l.Contents(onContentError)
	if err != nil {
		return nil, fmt.Errorf("error getting movies: %w", err)
	}

	for _, c := range contents {
		m := *c.(*Series)
		movies = append(movies, m)
	}

	return movies, nil
}
