package content

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var yearRegEx = regexp.MustCompile(`\(\d{4}\)`)

func strPtr(s string) *string { return &s }

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
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Battlestar Galactica")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Below Deck")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Beverly Hills Cop")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Blade")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Bourne")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Candyman")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Captain America")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Children of the Corn")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^The Chronicles of Narnia")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^City Hunter")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Critters")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Cube")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Death Wish")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Dead Space")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Deathstalker")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^The Descent")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Die Hard")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Family Guy Presents")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Futurama")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Friday the 13th")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Gamera")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Garfield")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile(`^G\.I\. Joe`)},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Get Smart")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Ginger Snaps")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^The Godfather")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Guardians of the Galaxy")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Halloween ")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Hellraiser")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Highlander")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Hitman")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Howling")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Jurassic Park")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Jaws")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Jay and Silent Bob")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Jackass")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Jurassic World")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Kung Fu Panda")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Mega Shark")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Mission Impossible")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Monster High")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Monty Python")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Mortal Kombat")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^National Treasure")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^NCIS")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Night at the Museum")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^A Nightmare on Elm Street")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^One Piece")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Paranormal Activity")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Perry Mason")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Police Academy")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Prince of Persia")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^The Princess Diaries")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Puss in Boots")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^The Purge")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Resident Evil")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Return of the Living Dead")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^RoboCop")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Rurouni Kenshin")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Saw")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Scooby-Doo")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^South Park")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Spider-Man")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Teenage Mutant Ninja Turtles")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^The Adventures of Young Indiana Jones")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^The Conjuring")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^The Fast and the Furious")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^The Hunger Games")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^The Lion King")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^The Matrix")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^The Terminator")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Tinker Bell")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Tom and Jerry")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Transformers")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Trinity Seven")},
		{Type: MappingTypeMoveYearRegex, FindRegex: regexp.MustCompile("^Wallace")},

		// Exact renames for franchises where the series title isn't at the start
		{Type: MappingTypeReplace, FindStr: strPtr("Beneath the Planet of the Apes (1970)"), ReplaceStr: strPtr("Planet of the Apes (1970) - Beneath the Planet of the Apes")},
		{Type: MappingTypeReplace, FindStr: strPtr("Escape from the Planet of the Apes (1971)"), ReplaceStr: strPtr("Planet of the Apes (1971) - Escape from the Planet of the Apes")},
		{Type: MappingTypeReplace, FindStr: strPtr("Conquest of the Planet of the Apes (1972)"), ReplaceStr: strPtr("Planet of the Apes (1972) - Conquest of the Planet of the Apes")},
		{Type: MappingTypeReplace, FindStr: strPtr("Battle for the Planet of the Apes (1973)"), ReplaceStr: strPtr("Planet of the Apes (1973) - Battle for the Planet of the Apes")},
		{Type: MappingTypeReplace, FindStr: strPtr("The Chronicles of Riddick (2004)"), ReplaceStr: strPtr("Riddick (2004) - The Chronicles of Riddick")},
		{Type: MappingTypeReplace, FindStr: strPtr("Live Free or Die Hard (2007)"), ReplaceStr: strPtr("Die Hard (2007) - Live Free or Die Hard")},
		{Type: MappingTypeReplace, FindStr: strPtr("A Good Day to Die Hard (2013)"), ReplaceStr: strPtr("Die Hard (2013) - A Good Day to Die Hard")},
		{Type: MappingTypeReplace, FindStr: strPtr("A Grand Day Out (1990)"), ReplaceStr: strPtr("Wallace And Gromit (1990) - A Grand Day Out")},
		{Type: MappingTypeReplace, FindStr: strPtr("The Wrong Trousers (1993)"), ReplaceStr: strPtr("Wallace And Gromit (1993) - The Wrong Trousers")},
		{Type: MappingTypeReplace, FindStr: strPtr("A Close Shave (1995)"), ReplaceStr: strPtr("Wallace And Gromit (1995) - A Close Shave")},
		{Type: MappingTypeReplace, FindStr: strPtr("A Matter of Loaf and Death (2008)"), ReplaceStr: strPtr("Wallace And Gromit (2008) - A Matter of Loaf and Death")},
		{Type: MappingTypeReplace, FindStr: strPtr("And Now for Something Completely Different (1971)"), ReplaceStr: strPtr("Monty Python (1971) - And Now for Something Completely Different")},
		{Type: MappingTypeReplace, FindStr: strPtr("Jabberwocky (1977)"), ReplaceStr: strPtr("Monty Python (1977) - Jabberwocky")},
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
		case MappingTypeReplace:
			if m.FindStr != nil && *m.FindStr == folder {
				return m.ReplaceStr, nil
			}
		case UnknownMapping:
			fallthrough
		default:
			return nil, fmt.Errorf("unknown mapping type: %d", m.Type)
		}
	}

	// if we get here no direct mapping was found, so handle special library mappings (standup)
	if libType == LibraryTypeStandup {
		// for standup we take the year from the end of the folder and then move it to before the first -
		// ie "Jim Jefferies - Freedumb (2016)" -> "Jim Jefferies (2016) - Freedumb"
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
