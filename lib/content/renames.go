package content

import (
	"errors"
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

// FolderMapping handles folder name transformations (renamed from Mapping)
type FolderMapping struct {
	Type             MappingType
	FindStr          *string        // find exact match including year
	FindRegex        *regexp.Regexp // find anything matching regex
	FindPrefix       *string        // find anything matching prefix
	ReplaceStr       *string        // replace exactly with this string
	ReplaceYearRegex *regexp.Regexp // replace with this regex where (YYYY) is replaced with ie (1999)
}

// folderRenames - global folder rename mappings per library type
var folderRenames = map[LibraryType][]FolderMapping{
	LibraryTypeMovies: {
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Alvin and the Chipmunks")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^American Ninja")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^American Pie")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Amityville")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Asterix")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Batman")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Below Deck")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Beverly Hills Cop")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Blade")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Bourne")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Captain America")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^City Hunter")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Deathstalker")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Family Guy Presents")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Futurama")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Gamera")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Guardians of the Galaxy")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Halloween ")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Hellraiser")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Jurassic Park")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Jurassic World")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Kung Fu Panda")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Mega Shark")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Mission Impossible")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Monster High")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Mortal Kombat")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^NCIS")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Night at the Museum")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^One Piece")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Paranormal Activity")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Perry Mason")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Police Academy")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Puss in Boots")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Resident Evil")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Saw")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Scooby-Doo")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^South Park")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Spider-Man")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Teenage Mutant Ninja Turtles")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^The Adventures of Young Indiana Jones")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^The Fast and the Furious")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^The Hunger Games")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^The Lion King")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^The Matrix")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^The Terminator")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Tinker Bell")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Tom and Jerry")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Transformers")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Trinity Seven")},
	},
	LibraryTypeStandup: {},
	LibraryTypeSeries: {
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Batman")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Law & Order")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Mobile Suit Gundam")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Spider-Man")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Star Trek")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Star Wars")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Stargate")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Teenage Mutant Ninja Turtles")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Transformers")},
	},
}

// AltFolderFor returns an alternate folder name based on folder renames
func AltFolderFor(libType LibraryType, folder string) (*string, error) {
	maps := folderRenames[libType]

	// find matching mappings and then return the "alt" folder
	for _, m := range maps {
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
		case UnknownMapping, MappingTypeReplace:
			fallthrough
		default:
			return nil, fmt.Errorf("unknown mapping type: %d", m.Type)
		}
	}

	// if we get here no direct mapping was found, so handle special library mappings (standup)
	if libType == LibraryTypeStandup {
		// for standup we take the year from the end of the folder and then move it to before the first -
		// ie "Jim Jefferies - Freedumb (2016)" -> "Freedumb (2016) - Jim Jefferies"
		year := yearRegEx.FindString(folder)
		if year == "" {
			return nil, errors.New("no year found in folder name")
		}

		// remove year from folder name
		folderWithoutYear := strings.TrimSuffix(yearRegEx.ReplaceAllString(folder, ""), " ")

		// Split the folder name into two parts based on the first dash
		parts := strings.SplitN(folderWithoutYear, "-", 2)
		if len(parts) < 2 {
			return nil, errors.New("invalid folder name format")
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
