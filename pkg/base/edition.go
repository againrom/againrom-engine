package base

import "fmt"

// Edition is what the two games differ in, stated as data. A profile selects
// one through its game, once, when the base is detected; code outside this
// package reads the fields and never compares a game. Behaviour that is not a
// value sits behind the game package's campaign service, which is picked from
// Campaign.
type Edition struct {
	// Game is the game the edition belongs to.
	Game Game

	// SaveTag is the game name the engine's session record in a save carries.
	// The first game's saves predate the tag and carry none.
	SaveTag Game

	// Campaign is the campaign model the game package picks its campaign
	// service from.
	Campaign Campaign

	// Town names the town description the square is composed from; empty
	// when the game's town is not a composed square.
	Town string

	// Generator names the character generator description; empty when the
	// game has none.
	Generator string

	// Music names the music description: the archive, the scene lists and
	// the rules the music controller plays them by.
	Music string

	// Rooms names the town description whose rooms the first game's tavern,
	// shop and school pages and their scene art are read from;
	// empty when the game has no such pages. The second game's towns name
	// their rooms in their own descriptions.
	Rooms string

	// Screens names the screen art description: the art manifest of every
	// screen no town or generator description names.
	Screens                   string
	CreatureResistanceHeading bool

	// CastNeedsSight makes the mission map's cast cursor stand only over a
	// cell one of whose four corners is in current sight; elsewhere an armed
	// cast shows the move cursor and a click moves.
	CastNeedsSight bool

	// Tips names the tip table: every tip popup the game shows, its text,
	// rectangle, raising event, once-only rule and pointer owner.
	Tips string

	// Spells names the spell presentation set: each effect identifier's
	// life, art, light and sound rules.
	Spells string

	// WorldMap names the world-map description: the campaign map's art,
	// modes, pick rule, travel and text as data.
	WorldMap string

	// Options names the Game Options column a stepping option row reads its
	// offered values and its new-game value from.
	Options string

	// CompanionObjectiveMission is the mission whose persistent companion the
	// compiled script must keep alive; zero for none.
	CompanionObjectiveMission int

	// TextCodePage is the code page the install writes a language's text
	// files in, from the language's font and Windows code pages. The second
	// game's are in the Windows one and its loaders convert them to the
	// font's (R2-ENGINE-052, R2-ENGINE-093).
	TextCodePage func(font, windows int) int

	// Cutscenes maps each speed route a player may ask for, video4 or
	// video8, to the archive its cutscenes play from.
	Cutscenes map[string]CutsceneArchive

	// DefaultCutscenes is the speed route a session plays from when the
	// player asked for none; a route Cutscenes does not map plays none.
	DefaultCutscenes string

	// StartupCutscenes are the movies that play before the menu, in order.
	StartupCutscenes []string

	// OriginalGenerator names the original random number generator this
	// game's evidence establishes, with its start-up, reseeds and draw forms;
	// empty when no evidence exists, and the launch switch for the original
	// generator then runs the default mode.
	OriginalGenerator string

	// ModFamily are the words a mod's applies-to list names every base of
	// the game by, beside each base's own id.
	ModFamily []string
}

// CutsceneArchive is one archive cutscenes play from.
type CutsceneArchive struct {
	// Path is the archive file under the install root, its components
	// separated by slashes.
	Path string
	// Name is the directory the archive addresses its members under.
	Name string
	// Logos is the speed route whose archive the logos/ movies play from;
	// a route Cutscenes does not map plays them from this archive.
	Logos string
}

// Campaign names a campaign model.
type Campaign int

const (
	// CampaignChapters is the first game's campaign: numbered chapters over
	// one town, opened from the world map.
	CampaignChapters Campaign = iota
	// CampaignDestinations is the second game's campaign: a scenario bank
	// that names the towns and missions the party may travel to.
	CampaignDestinations
)

var firstEdition = Edition{
	TextCodePage:              FontText,
	Game:                      GameROM1,
	Campaign:                  CampaignChapters,
	Town:                      "rom1",
	Generator:                 "rom1",
	Music:                     "rom1",
	Rooms:                     "rom1",
	Screens:                   "rom1",
	WorldMap:                  "rom1",
	CreatureResistanceHeading: true,
	CastNeedsSight:            true,
	Tips:                      "rom1",
	Options:                   "rom1",
	Spells:                    "rom1",
	CompanionObjectiveMission: 40,
	Cutscenes: map[string]CutsceneArchive{
		"video4": {Path: "Allods/video4.res", Name: "video4"},
		"video8": {Path: "Allods/video8.res", Name: "video8", Logos: "video4"},
	},
	StartupCutscenes:  append([]string{"logos/buka.smk", "logos/nival.smk", "logos/1c.smk"}, numberedCutscenes("intro")...),
	OriginalGenerator: "msvc",
	ModFamily:         []string{"rom1"},
}

var secondEdition = Edition{
	Game:         GameROM2,
	SaveTag:      GameROM2,
	Campaign:     CampaignDestinations,
	Generator:    "rom2",
	Music:        "rom2",
	Screens:      "rom2",
	WorldMap:     "rom2",
	Tips:         "rom2",
	Options:      "rom2",
	Spells:       "rom2",
	TextCodePage: WindowsText,
	Cutscenes: map[string]CutsceneArchive{
		"video4": {Path: "video.res", Name: "video"},
		"video8": {Path: "video.res", Name: "video"},
	},
	DefaultCutscenes: "video4",
}

// AppliesTo are the names a base answers to in a mod's applies-to list: its
// id, then the family words of the edition of the profile carrying that id. An
// id no profile carries answers to itself alone.
func AppliesTo(id string) []string {
	names := []string{id}
	if p, ok := Find(id); ok {
		names = append(names, p.Edition().ModFamily...)
	}
	return names
}

// numberedCutscenes are the 99 numbered movies of a directory.
func numberedCutscenes(directory string) []string {
	names := make([]string, 99)
	for n := range names {
		names[n] = fmt.Sprintf("%s/%02d.smk", directory, n+1)
	}
	return names
}

// Edition is the game's edition. The empty game and any game this package
// does not name are the first game, as Profile.GameOf reads them.
func (g Game) Edition() Edition {
	if g == GameROM2 {
		return secondEdition
	}
	return firstEdition
}

// Edition is the profile's edition.
func (p Profile) Edition() Edition { return p.GameOf().Edition() }

func (p Profile) CreatureResistanceHeading() bool { return p.Edition().CreatureResistanceHeading }

func (p Profile) CastNeedsSight() bool { return p.Edition().CastNeedsSight }

// Known reports whether g is a game this package names; the empty game is the
// first game.
func (g Game) Known() bool { return g == "" || g == GameROM1 || g == GameROM2 }

// Normal is g with the empty game read as the first game.
func (g Game) Normal() Game {
	if g == "" {
		return GameROM1
	}
	return g
}

// SameGame reports whether a and b name one game, the empty game being the
// first.
func SameGame(a, b Game) bool { return a.Normal() == b.Normal() }

// FontText is the font's code page: the text is written as the font draws it.
func FontText(font, _ int) int { return font }

// WindowsText is the Windows code page, the font's when the language names
// none.
func WindowsText(font, windows int) int {
	if windows == 0 {
		return font
	}
	return windows
}
