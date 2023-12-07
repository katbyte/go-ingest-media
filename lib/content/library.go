package content

import (
	"fmt"
	"path/filepath"

	"github.com/katbyte/go-injest-media/lib/folders"
)

type LibraryType int

const (
	Unknown LibraryType = iota
	Series
	Movies
)

type Library struct {
	SrcFolder     string
	DstFolder     string
	SrcPath       string
	DstPath       string
	Type          LibraryType
	LetterFolders bool
}

var libraries = []Library{
	{SrcFolder: "m.anime", DstFolder: "anime/movies", Type: Movies, LetterFolders: false},
	{SrcFolder: "m.movies", DstFolder: "movies", Type: Movies, LetterFolders: true},
	// {InPath: "s.anime", OutPath: "anime/series", Type: Series},
	// {InPath: "s.docu", OutPath: "docu/docuseries", Type: Series},
	// {InPath: "s.tv", OutPath: "tv", Type: Series},
}

func GetLibraries(srcPath, dstPath string) []Library {
	updated := make([]Library, len(libraries))

	for i, l := range libraries {
		l2 := l
		l2.SrcPath = filepath.Join(srcPath, l2.SrcFolder)
		l2.DstPath = filepath.Join(dstPath, l2.DstFolder)
		updated[i] = l2
	}

	return updated
}

func (l Library) Contents(onContentError func(folder string, err error)) ([]ContentInterface, error) {
	var contents []ContentInterface

	folders, err := folders.List(l.SrcPath)
	if err != nil {
		return nil, fmt.Errorf("error listing content folders: %w", err)
	}

	for _, f := range folders {
		var c ContentInterface

		if l.Type == Movies {
			c, err = l.MovieFor(f)
		} else if l.Type == Series {
			// c, err = l.SeriesFor(f) TODO
			return nil, fmt.Errorf("series: %s", f)
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
