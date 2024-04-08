package content

import (
	"fmt"
	"regexp"
	"strings"
)

var yearRegEx = regexp.MustCompile(`\(\d{4}\)`)

type MappingType int

const (
	UnknownMapping     MappingType = iota
	MappingTypeReplace             // replace find regex with replace string
	MappingTypeMoveYearRegex
)

// todo base interface and then implement for each type ??
// there has to be a better way to do this
type Mapping struct {
	Type             MappingType
	FindStr          *string        // find exact match including year
	FindRegex        *regexp.Regexp // find anything matching regex
	FindPrefix       *string        // find anything matching prefix
	ReplaceStr       *string        // replace exactly with this string
	ReplaceYearRegex *regexp.Regexp // replace with this regex where (YYYY) is replaced with ie (1999)
}

var mappings = map[LibraryType][]Mapping{
	LibraryTypeMovies: {
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^American Ninja")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^American Pie")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Amityville")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Asterix")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Batman")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Beverly Hills Cop")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Blade")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Bourne")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Below Deck")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Captain America")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Deathstalker")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Hellraiser")}, {Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Gamera")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Mega Shark")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Mission Impossible")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^One Piece")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^NCIS")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Perry Mason")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Police Academy")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Resident Evil")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Saw")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^South Park")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Spider-Man")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Trinity Seven")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Transformers")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Teenage Mutant Ninja Turtles")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^The Hunger Games")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Guardians of the Galaxy")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Halloween")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^The Matrix")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^The Terminator")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^The Fast and the Furious")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^The Adventures of Young Indiana Jones")},
	},
	LibraryTypeStandup: {},
	LibraryTypeSeries: {
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Batman")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Law & Order")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Spider-Man")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Star Trek")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Star Wars")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Transformers")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Teenage Mutant Ninja Turtles")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Mobile Suit Gundam")},
	},
}

// Ong-Bak 3 (2010) -> Ong Bak 3 (2010)

func (l Library) mappings() []Mapping {
	maps, ok := mappings[l.Type]
	if !ok {
		panic(fmt.Sprintf("no mappings for library type: %d", l.Type))
	}

	return maps
}

func (l Library) AltFolderFor(folder string) (*string, error) {
	// find matching mappings and then return the "alt" folder
	for _, m := range l.Mappings {
		switch m.Type {
		case MappingTypeMoveYearRegex:
			if m.FindRegex.MatchString(folder) {
				match := m.FindRegex.FindString(folder)
				// get year from folder name
				year := yearRegEx.FindString(folder)

				// remove year from folder name
				folderWithoutYear := strings.TrimSuffix(yearRegEx.ReplaceAllString(folder, ""), " ")

				// replace find regex match with year
				newFolder := m.FindRegex.ReplaceAllString(folderWithoutYear, fmt.Sprintf("%s %s", match, year))
				return &newFolder, nil
			}
		default:
			return nil, fmt.Errorf("unknown mapping type: %d", m.Type)
		}
	}

	// if we get here no direct mapping was found, so handle special library mappings (standup)
	// should this be part of a special library type for standup?
	if l.Type == LibraryTypeStandup {
		// for standup we take the year from the end of the folder and then move it to before the first -
		// ie "Jim Jefferies - Freedumb (2016)" -> "Freedumb (2016) - Jim Jefferies"
		year := yearRegEx.FindString(folder)
		if year == "" {
			return nil, fmt.Errorf("no year found in folder name")
		}

		// remove year from folder name
		folderWithoutYear := strings.TrimSuffix(yearRegEx.ReplaceAllString(folder, ""), " ")

		// Split the folder name into two parts based on the first dash
		parts := strings.SplitN(folderWithoutYear, "-", 2)
		if len(parts) < 2 {
			return nil, fmt.Errorf("invalid folder name format")
		}

		// Trim any leading or trailing spaces
		for i, part := range parts {
			parts[i] = strings.TrimSpace(part)
		}

		newFolderName := fmt.Sprintf("%s %s - %s", parts[0], year, parts[1])

		return &newFolderName, nil
	}

	return nil, nil
}
