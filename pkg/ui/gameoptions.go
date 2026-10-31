package ui

import (
	"fmt"
	"image"
	"slices"
	"strings"
)

// GameOption names a working setting shared by the menu and map shortcuts.
type GameOption uint8

const (
	GameOptionDayNight GameOption = iota
	GameOptionHealth
	GameOptionDamage
	GameOptionFormation
	GameOptionRetreat
	GameOptionPathfinding
	GameOptionSmoothing
	GameOptionShadows
	GameOptionLighting
	GameOptionAnimation
	GameOptionAutoHealing
	GameOptionLootDots
	GameOptionAutoCast
	gameOptionCount
)

type GameOptionValues [gameOptionCount]int

// gameOptionDefinition is one Game Options row: the profile key, the number
// of values (2 a checkbox, 3 a radio group), the rectangle from the frame's
// corner, the headless control name and the row's place in the page's focus
// order and in the OK commit order (MENU-074). A row with steps is a push
// button that steps through the values it offers in each game. The page, the
// commit, the shortcut cycle and the profile keys read this one table.
type gameOptionDefinition struct {
	key, name     string
	choices       int
	rect          image.Rectangle
	focus, commit int
	steps         *gameOptionSteps
	// needs are the options whose 0 forces this one to 0 on commit and in
	// the store (Animation 0 forces Lighting 0, MENU-074).
	needs []GameOption
	// word is the engine's word for the label: it replaces the controls'
	// label when wordWins, and stands in for an absent one otherwise.
	word     string
	wordWins bool
	// choiceWords stand in for absent choice captions, one per value.
	choiceWords []string
	// unknown: a negative value, a party with no reading, shows "--".
	unknown bool
}

// gameOptionSteps is a stepping row's data per Game Options column, the
// edition's Options name: the values it offers in order and the value a new
// game starts at. The empty column, a page without an edition, offers the
// first game's.
type gameOptionSteps struct {
	offers   map[string][]int
	defaults map[string]int
}

// gameOptionTable holds every row in GameOption order. A focus value between
// two rows' places an engine row that is not an option (the tooltip delay at
// 55, Tips at 75, the timed autosave pair at 125).
var gameOptionTable = [gameOptionCount]gameOptionDefinition{
	GameOptionDayNight:  {key: "ShowTimeFlow", name: "day-night", choices: 2, rect: image.Rect(24, 104, 234, 128), focus: 10, commit: 0},
	GameOptionHealth:    {key: "ShowAllHitPoints", name: "show-health", choices: 2, rect: image.Rect(240, 48, 456, 72), focus: 60, commit: 5},
	GameOptionDamage:    {key: "ShowFlyingHP", name: "flying-damage", choices: 2, rect: image.Rect(240, 76, 456, 100), focus: 70, commit: 6},
	GameOptionFormation: {key: "FormationMode", name: "formation", choices: 3, rect: image.Rect(24, 248, 216, 338), focus: 110, commit: 8},
	GameOptionRetreat:   {key: "WimpyMode", name: "retreat-mode", choices: 3, rect: image.Rect(240, 248, 408, 338), focus: 120, commit: 9},
	GameOptionPathfinding: {key: "ShowPathfinding", name: "pathfinding", choices: 2, rect: image.Rect(240, 224, 456, 248), focus: 100, commit: 7,
		word: "options.pathfinding", wordWins: true},
	GameOptionSmoothing: {key: "Smoothing", name: "smoothing", choices: 2, rect: image.Rect(24, 128, 234, 152), focus: 20, commit: 1},
	// The three graphics captions come from the install when it supplies
	// them; the engine's words stand in only for an absent one.
	GameOptionShadows: {key: "Shadows", name: "shadows", choices: 2, rect: image.Rect(24, 152, 234, 176), focus: 30, commit: 2,
		word: "options.shadows"},
	GameOptionLighting: {key: "Lighting", name: "lighting", choices: 2, rect: image.Rect(24, 176, 234, 200), focus: 40, commit: 3,
		needs: []GameOption{GameOptionAnimation}, word: "options.lighting"},
	GameOptionAnimation: {key: "Animation", name: "animation", choices: 2, rect: image.Rect(24, 200, 234, 224), focus: 50, commit: 4,
		word: "options.animation"},
	GameOptionAutoHealing: {key: "AutoCasting", name: "autohealing", choices: 3, rect: image.Rect(240, 132, 432, 224), focus: 90, commit: 10,
		unknown: true},
	GameOptionLootDots: {key: "LootDots", name: "loot-dots", choices: 2, rect: image.Rect(24, 388, 234, 412), focus: 130, commit: 11,
		word: "options.loot_dots", wordWins: true},
	// The automatic-casting level: Off, Minimum, Average, Maximum. The first
	// game offers all four and starts Off (DIV-3006); the second offers its
	// three levels and starts at Average, the constructor's (DIV-3007). The
	// row reads the install's captions when the game supplies them, the
	// engine's words otherwise (DIV-3011).
	GameOptionAutoCast: {key: "AutoCastMode", name: "autocast-mode", choices: 4, rect: image.Rect(240, 340, 456, 364), focus: 140, commit: 12,
		steps: &gameOptionSteps{map[string][]int{"": {0, 1, 2, 3}, "rom1": {0, 1, 2, 3}, "rom2": {1, 2, 3}}, map[string]int{"": 0, "rom1": 0, "rom2": 2}},
		word:  "options.autocast", choiceWords: []string{"options.autocast_off", "options.autocast_minimum", "options.autocast_average", "options.autocast_maximum"}},
}

// Constrained is v with each row's needs applied: a row whose needed option
// is 0 is 0.
func (v GameOptionValues) Constrained() GameOptionValues {
	for o, d := range gameOptionTable {
		for _, n := range d.needs {
			if v[n] == 0 {
				v[o] = 0
			}
		}
	}
	return v
}

// Offers lists the values the option offers in the given Game Options column.
func (o GameOption) Offers(column string) []int {
	if o >= gameOptionCount {
		return nil
	}
	if steps := gameOptionTable[o].steps; steps != nil {
		return steps.offers[column]
	}
	out := make([]int, gameOptionTable[o].choices)
	for i := range out {
		out[i] = i
	}
	return out
}

// Offered reports whether the option offers value in the given column.
func (o GameOption) Offered(column string, value int) bool {
	return slices.Contains(o.Offers(column), value)
}

// Default is the option's starting value in the given column, for a row whose
// table entry states one.
func (o GameOption) Default(column string) int {
	if o >= gameOptionCount || gameOptionTable[o].steps == nil {
		return 0
	}
	return gameOptionTable[o].steps.defaults[column]
}

// stepping reports whether the option is a push button stepping its offers.
func (o GameOption) stepping() bool {
	return o < gameOptionCount && gameOptionTable[o].steps != nil
}

// nextOffer is the value after v among the offers, wrapping; a value the
// game does not offer steps to the first.
func (o GameOption) nextOffer(column string, v int) int {
	offers := o.Offers(column)
	if len(offers) == 0 {
		return v
	}
	i := slices.Index(offers, v)
	return offers[(i+1)%len(offers)]
}

// gameOptionNamed is the option a headless control name names.
func gameOptionNamed(name string) (GameOption, bool) {
	name = strings.ToLower(strings.TrimSpace(name))
	for i, d := range gameOptionTable {
		if d.name == name {
			return GameOption(i), true
		}
	}
	return 0, false
}

// Key is the option's profile key.
func (o GameOption) Key() string {
	if o < gameOptionCount {
		return gameOptionTable[o].key
	}
	return ""
}

// GameOptions lists every option in GameOption order.
func GameOptions() []GameOption {
	out := make([]GameOption, gameOptionCount)
	for i := range out {
		out[i] = GameOption(i)
	}
	return out
}

func (o GameOption) Choices() int {
	if o < gameOptionCount {
		return gameOptionTable[o].choices
	}
	return 0
}

// gameOptionsBy lists the options ordered by the given rank.
func gameOptionsBy(rank func(gameOptionDefinition) int) []GameOption {
	out := GameOptions()
	slices.SortStableFunc(out, func(a, b GameOption) int { return rank(gameOptionTable[a]) - rank(gameOptionTable[b]) })
	return out
}

type GameOptionWords struct {
	Title, Tips, OK, Cancel, Speed string
	Labels                         [gameOptionCount]string
	// Choices are a row's captions per value, per option.
	Choices [gameOptionCount][4]string
}

// Captions are already encoded for the installed font. Write persists before
// applying a change; Read includes commands waiting for the next ordinary tick.
type GameOptionControls struct {
	// Column is the edition's Game Options column (base.Edition.Options).
	Column   string
	Autosave TimedAutosaveControls
	Read     func(onMap bool) GameOptionValues
	Write    func(onMap bool, option GameOption, value int) error
	Words    GameOptionWords
	Checks   [2]image.Image
	Radios   [2]image.Image
	// draft is the open page's local copy; nil while the page is closed.
	draft *gameOptionsDraft
}

func (a *App) SetGameOptionControls(c GameOptionControls) {
	if a != nil && a.flow != nil {
		f := a.flow
		for o, d := range gameOptionTable {
			if d.word != "" && (d.wordWins || c.Words.Labels[o] == "") {
				c.Words.Labels[o] = f.menuWord(d.word)
			}
			for i, word := range d.choiceWords {
				if c.Words.Choices[o][i] == "" {
					c.Words.Choices[o][i] = f.menuWord(word)
				}
			}
		}
		a.flow.gameOptions = c
	}
}

// GameOptionWords returns the page's captions as drawn, the install's or the
// engine's.
func (a *App) GameOptionWords() GameOptionWords {
	if a == nil || a.flow == nil {
		return GameOptionWords{}
	}
	return a.flow.gameOptions.Words
}

func optionAction(o GameOption) gameMenuAction { return gameMenuDayNight + gameMenuAction(o) }

func (f *flow) gameOptionsOnMap() bool {
	return f.screen == ScreenMap || f.screen == ScreenGameMenu && f.menuBack == ScreenMap
}

func (f *flow) readGameOptions() GameOptionValues { return f.gameOptions.Read(f.gameOptionsOnMap()) }

func (f *flow) setGameOption(o GameOption, value int) {
	if f.gameOptions.Write == nil || !o.Offered(f.gameOptions.Column, value) {
		return
	}
	err := f.gameOptions.Write(f.gameOptionsOnMap(), o, value)
	if f.screen == ScreenGameMenu {
		selection := f.menuList.Selection()
		f.rebuildGameMenu(gameMenuGameOptionsPage, selection)
	}
	if err != nil {
		f.msg = fmt.Sprintf("Setting not saved: %v", err)
		if f.screen == ScreenMap {
			f.showTextNotice(f.msg)
		}
	}
}

// False retains the standalone viewer's existing shortcut behavior when no
// preference seam is installed.
func (f *flow) cycleGameOption(o GameOption) bool {
	if f.gameOptions.Read == nil || f.gameOptions.Write == nil {
		return false
	}
	f.setGameOption(o, o.nextOffer(f.gameOptions.Column, f.readGameOptions()[o]))
	return true
}
