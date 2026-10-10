package base

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

	// NewGameInTown: a new game opened without generation starts in the
	// campaign town with the default hero rather than on the first mission.
	NewGameInTown bool

	// TownDifficulty: a save taken in the town carries the session's own
	// difficulty in its session record.
	TownDifficulty bool

	// CompanionObjectiveMission is the mission whose persistent companion the
	// compiled script must keep alive; zero for none.
	CompanionObjectiveMission int

	// Cheats: the chat cheat line, the debug letters and the cheat item and
	// actor names exist.
	Cheats bool

	// FreshPlayers: a fresh mission builds its players by the engine's own
	// construction policy.
	FreshPlayers bool

	// The second game's file layouts: maps, scripts, the definition table,
	// the per-mission text files, the main menu art, the unit placement keys
	// and the spell arm table.
	SecondMaps        bool
	SecondScripts     bool
	SecondTable       bool
	SecondMissionText bool
	SecondMenu        bool
	SecondUnitKeys    bool
	SecondSpellArms   bool

	// CutsceneArchive names the archive every cutscene plays from; empty
	// plays each from the archive its caller names.
	CutsceneArchive string

	// StartupCutscenes: the logo and introduction movies play at start.
	StartupCutscenes bool

	// OriginalRandom: evidence for this game's own random number generator
	// and its draw forms exists, so the original generator can be switched
	// on. Without it the launch switch runs the default mode.
	OriginalRandom bool
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
	Game:                      GameROM1,
	Campaign:                  CampaignChapters,
	Town:                      "rom1",
	CompanionObjectiveMission: 40,
	Cheats:                    true,
	FreshPlayers:              true,
	StartupCutscenes:          true,
	OriginalRandom:            true,
}

var secondEdition = Edition{
	Game:              GameROM2,
	SaveTag:           GameROM2,
	Campaign:          CampaignDestinations,
	NewGameInTown:     true,
	TownDifficulty:    true,
	SecondMaps:        true,
	SecondScripts:     true,
	SecondTable:       true,
	SecondMissionText: true,
	SecondMenu:        true,
	SecondUnitKeys:    true,
	SecondSpellArms:   true,
	CutsceneArchive:   "video",
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
