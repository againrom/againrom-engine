package game

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// The two stages a scenario can be written against.
//
// A stage is which driver the file's steps are dispatched to, and the two are
// disjoint on purpose. StageFrontEnd is the production controller: menus, saves,
// the party sheet — everything a player reaches with the keyboard. StageMission
// is the simulation of one campaign mission with no controller above it: units,
// orders, ticks and the mission's own verdict.
//
// THEY ARE NOT MIXED IN ONE FILE, and that is a decision rather than an
// omission. A front-end scenario reaches a live world only where the
// controller put one there, and which world that is depends on the menu path
// the file walked; a mission scenario names its world outright. Allowing
// both in one file would make every play step's meaning depend on the menu
// history above it.
const (
	// StageFrontEnd drives *FrontEnd and *ui.App. It is the default, so every
	// version-1 file is a front-end scenario without saying so.
	StageFrontEnd = "frontend"
	// StageMission drives one campaign mission's *sim.World directly.
	StageMission = "mission"
)

// HeadlessScenarioVersion is the highest scenario version this build reads.
//
// A LOWER VERSION KEEPS EXACTLY THE VOCABULARY IT SHIPPED WITH. A version-1
// file is refused the moment it names a stage, a mission or a version-2
// command, and a version-2 file is refused the moment it names a version-3
// command, so a file written before this story cannot change meaning and
// cannot silently acquire behaviour its author never wrote. New work is
// written at the current version.
//
// VERSION 3 IS 0163: create_character on the front-end stage, and wait_until
// reaching that stage for the first time.
//
// VERSION 4 IS 1005 ROUND 2: the pointer step on the front-end stage, and the
// two assertions a pointer gesture over an inventory is judged by —
// assert_inventory and assert_shop. Before it no scenario could press, move or
// release anything, so no inventory gesture could be driven at all.
//
// VERSION 5 IS 1020: the abort_game step on the front-end stage, shaped like
// save — it escapes to the game menu if not already there and chooses the
// ABORT GAME row by its production action, never by its displayed label,
// which is the localized dialogs.txt word MenuAbort. Before it the only way
// to drive that row was open_menu "game" then activate by the row's own
// displayed text, which passes on an EN install and fails on RU.
//
// VERSION 6 IS 1060: the mission stage's direct kill, player-wide kill,
// teleport, remote pick-up and heal actions. Each action changes ordinary
// simulation state and then advances one production script tick; none can set
// a latch or outcome.
// Version 7 adds observation-only transition assertions and a frontend wait
// for a choosable list control. It adds no simulation action to that stage.
// Version 8 adds quick-spell observations and a book-cell pointer address.
// These reach ordinary App input; no simulation or binding setter is added.
const HeadlessScenarioVersion = 9

// HeadlessScenario is the versioned declarative input accepted by againrom's
// production no-window runner. Paths are resolved relative to the scenario.
type HeadlessScenario struct {
	Version int `json:"version"`

	// Stage is StageFrontEnd or StageMission; empty means StageFrontEnd.
	Stage string `json:"stage,omitempty"`

	// Assets is where the run's bytes come from: AssetsInstall or
	// AssetsSynthetic. Empty means AssetsInstall.
	Assets string `json:"assets,omitempty"`

	// Mission, Difficulty and Mage are the mission stage's own inputs: which
	// campaign mission to start, at which difficulty, and whether the party's
	// hero opens as a caster. They are refused on the front-end stage, and on
	// the mission stage they are refused for a synthetic scenario, which builds
	// its world from World instead.
	Mission    int    `json:"mission,omitempty"`
	Difficulty string `json:"difficulty,omitempty"`
	Mage       bool   `json:"mage,omitempty"`

	// World is the authored world a synthetic mission scenario runs on. It is
	// composed here rather than sampled out of an archive: a subset cut from a
	// real archive would be game data in the repository (golden rule 1).
	World *HeadlessWorldSpec `json:"world,omitempty"`

	// Window is the window size the front-end stage runs at, in pixels (1005
	// round 2). It reaches the production controller through App.Layout, the
	// same door the engine drives when a real window is created or resized.
	//
	// A FILE THAT DRIVES A POINTER HAS TO STATE ONE. Every HUD box on the map
	// screen is placed against the window and REFUSED where there is no room:
	// at the headless default of 1280x960 a selected hero's own unit panel is
	// 373 pixels tall, which leaves the doll box below its floor and undrawn,
	// so a scenario pressing a doll slot there is pressing something the
	// running build is not showing. Naming the size is what makes the gesture
	// reproducible rather than dependent on a default nobody chose.
	Window *HeadlessWindow `json:"window,omitempty"`

	OriginalSaves string         `json:"original_saves,omitempty"`
	Saves         string         `json:"saves,omitempty"`
	Steps         []HeadlessStep `json:"steps"`
}

// HeadlessWindow is a window size in pixels.
type HeadlessWindow struct {
	W int `json:"w"`
	H int `json:"h"`
}

// HeadlessStep is one controller or simulation action. Validation below makes
// each command's legal parameter set explicit and rejects unused fields as
// likely mistakes.
type HeadlessStep struct {
	Command   string `json:"command"`
	Target    string `json:"target,omitempty"`
	Key       string `json:"key,omitempty"`
	Ticks     int    `json:"ticks,omitempty"`
	Name      string `json:"name,omitempty"`
	ID        string `json:"id,omitempty"`
	Directory string `json:"directory,omitempty"`

	// Unit is a mission-stage unit reference: uNN for the identifier the map's
	// script uses, pN for the party's Nth member, eNN for a raw entity id.
	// Order, X, Y and Target carry the order it is given.
	Unit  string `json:"unit,omitempty"`
	Order string `json:"order,omitempty"`
	X     *int32 `json:"x,omitempty"`
	Y     *int32 `json:"y,omitempty"`

	// Spell is the spell id a cast or an autocast order names. It is a POINTER
	// so that a step naming 0 is distinguishable from a step naming none: 0 is
	// what clears an autocast, and a form with no way to say it could not
	// express the toggle's other half.
	Spell *uint32 `json:"spell,omitempty"`

	// Player is the script player slot affected by a player-sized direct
	// action. It is a pointer because slot zero is a real value.
	Player *uint32 `json:"player,omitempty"`

	// Character is the spread create_character asks the production generation
	// screen for.
	Character *HeadlessCharacter `json:"character,omitempty"`

	// Action and At are the pointer step's own pair (1005 round 2): which edge
	// of a gesture this is, and the surface it lands on.
	Action string         `json:"action,omitempty"`
	At     *HeadlessPoint `json:"at,omitempty"`

	Member    *HeadlessMemberAssertion    `json:"member,omitempty"`
	State     *HeadlessStateAssertion     `json:"state,omitempty"`
	Until     *HeadlessUntil              `json:"until,omitempty"`
	Expect    *HeadlessUnitAssertion      `json:"expect,omitempty"`
	World     *HeadlessWorldAssertion     `json:"world,omitempty"`
	Inventory *HeadlessInventoryAssertion `json:"inventory,omitempty"`
	Shop      *HeadlessShopAssertion      `json:"shop,omitempty"`
}

// HeadlessCharacter is one generated character, named the way the generation
// screen names its own rows.
//
// EVERY FIELD IS OPTIONAL and a field not named keeps whatever the generator
// opened with. That is not a convenience: the screen opens on a complete, legal
// character, so a scenario that cares about the class alone should not have to
// restate a spread it has no opinion about, and restating one would make the
// file assert the default's own numbers by accident.
//
// Sex, Class and Skill are OPTION LABELS the screen shows, matched case- and
// space-insensitively. Stats maps a statistic's own name to the value it is to
// be bought to. Neither set of strings is authored here; both come from
// ChargenSetup, so a scenario is naming what a player reads on the screen.
type HeadlessCharacter struct {
	Difficulty string         `json:"difficulty,omitempty"`
	Name       string         `json:"name,omitempty"`
	Sex        string         `json:"sex,omitempty"`
	Class      string         `json:"class,omitempty"`
	Skill      string         `json:"skill,omitempty"`
	Stats      map[string]int `json:"stats,omitempty"`
}

func (c *HeadlessCharacter) validate() error {
	if c == nil {
		return errors.New("create_character requires a character")
	}
	if c.Name == "" && c.Sex == "" && c.Class == "" && c.Skill == "" && c.Difficulty == "" && len(c.Stats) == 0 {
		return errors.New("create_character: the character states nothing")
	}
	if _, err := headlessDifficulty(c.Difficulty); err != nil {
		return err
	}
	for name, value := range c.Stats {
		if strings.TrimSpace(name) == "" {
			return errors.New("create_character: a statistic has no name")
		}
		if value < 0 {
			return fmt.Errorf("create_character: statistic %q asks for %d", name, value)
		}
	}
	return nil
}

type HeadlessMemberAssertion struct {
	ID         string `json:"id"`
	XP         *int64 `json:"xp,omitempty"`
	Defense    *int32 `json:"defense,omitempty"`
	Absorption *int32 `json:"absorption,omitempty"`
	// Load and Capacity are the carried weight and what it is measured against.
	// They are pointers on the file's own convention: a scenario that does not
	// name one asserts nothing about it, and zero is a real load.
	Load            *int32           `json:"load,omitempty"`
	Capacity        *int32           `json:"capacity,omitempty"`
	Temporary       *bool            `json:"temporary,omitempty"`
	Skills          map[string]int32 `json:"skills,omitempty"`
	SameCharacterAs string           `json:"same_character_as,omitempty"`
}

type HeadlessStateAssertion struct {
	QuickSpells  *[4]uint32 `json:"quick_spells,omitempty"`
	CurrentSpell *uint32    `json:"current_spell,omitempty"`
	SpellArmed   *bool      `json:"spell_armed,omitempty"`
	Difficulty   *int       `json:"difficulty,omitempty"`
	Screen       string     `json:"screen,omitempty"`
	Mission      *int       `json:"mission,omitempty"`
	Notice       string     `json:"notice,omitempty"`
	TownPlace    string     `json:"town_place,omitempty"`
	Purse        *uint32    `json:"purse,omitempty"`
	Documents    *uint32    `json:"documents,omitempty"`
	// Collected is the CAMPAIGN DOCUMENT COLLECTION's own size, which is a
	// different quantity from Documents above it: Documents counts the ACCESS
	// ITEM in the primary's pack, and this counts the (value, kind) elements
	// the campaign has granted. A scenario asserting one says nothing about the
	// other.
	Collected   *int                      `json:"collected_documents,omitempty"`
	MemberCount *int                      `json:"member_count,omitempty"`
	Members     []HeadlessMemberAssertion `json:"members,omitempty"`

	// Document is the open documents panel's own position. A scenario naming
	// it asserts that the panel is showing THAT element and THAT page, which
	// is what tells an arrow that pages from one that does nothing.
	Document *HeadlessDocumentAssertion `json:"document,omitempty"`
}

// HeadlessDocumentAssertion is what a scenario asserts about the open documents
// panel. Every field is a pointer on this file's own convention: a field the
// scenario does not name asserts nothing about it.
//
// PAGES IS LANGUAGE-DEPENDENT and a scenario meant to run on both shipped
// roots must not name it. The panel wraps the install's own text to the
// sheet's width with the install's own font, so the same element fills four
// pages on the English root and five on the Russian one.
type HeadlessDocumentAssertion struct {
	Element  *int  `json:"element,omitempty"`
	Elements *int  `json:"elements,omitempty"`
	Page     *int  `json:"page,omitempty"`
	Pages    *int  `json:"pages,omitempty"`
	Picture  *bool `json:"picture,omitempty"`
}

// HeadlessState is one deterministic observation. XP is scalar per member;
// per-slot residues are included separately so an array can never obscure who
// owns the total.
type HeadlessState struct {
	Tooltip        ui.HeadlessTooltipState  `json:"tooltip"`
	ScorchedGround int                      `json:"scorched_ground,omitempty"`
	BurnedObjects  int                      `json:"burned_objects,omitempty"`
	QuickSpells    [4]uint32                `json:"quick_spells"`
	CurrentSpell   uint32                   `json:"current_spell,omitempty"`
	SpellArmed     bool                     `json:"spell_armed,omitempty"`
	Difficulty     int                      `json:"difficulty"`
	Screen         string                   `json:"screen"`
	Mission        int                      `json:"mission,omitempty"`
	Notice         string                   `json:"notice,omitempty"`
	TownPlace      string                   `json:"town_place,omitempty"`
	Purse          uint32                   `json:"purse"`
	Documents      uint32                   `json:"documents"`
	Collected      int                      `json:"collected_documents"`
	Members        []HeadlessMemberSnapshot `json:"members"`

	// Chargen is the generation screen, present only while it is showing. It is
	// a pointer so that "absent" and "showing nothing" are different values in
	// the emitted JSON.
	Chargen *ui.HeadlessChargen `json:"chargen,omitempty"`

	// Inventory is the map screen's own doll, worn array, container and the
	// ground's sacks, present only while a mission with an inventory subject
	// is running (1005 round 2). Shop is the town's own shop room, present
	// only while one is open. Both are pointers on Chargen's own reasoning.
	Inventory *HeadlessInventoryState `json:"inventory,omitempty"`
	Shop      *HeadlessShopState      `json:"shop,omitempty"`

	// Document is the campaign documents panel, present only while it is
	// showing, on Chargen's own reasoning.
	Document   *ui.HeadlessDocuments  `json:"document,omitempty"`
	SaveDialog *ui.HeadlessSaveDialog `json:"save_dialog,omitempty"`
}

type HeadlessMemberSnapshot struct {
	ID         string                  `json:"id"`
	Name       string                  `json:"name"`
	Membership string                  `json:"membership"`
	Temporary  bool                    `json:"temporary"`
	Entity     uint32                  `json:"entity,omitempty"`
	Appearance HeadlessAppearance      `json:"appearance"`
	Weapon     *HeadlessWeaponSnapshot `json:"weapon,omitempty"`
	Equipment  [sim.EquipSlots]uint16  `json:"equipment"`
	Worn       []HeadlessWornSnapshot  `json:"worn"`
	XP         int64                   `json:"xp"`
	Defense    int32                   `json:"defense"`
	Absorption int32                   `json:"absorption"`
	Load       int32                   `json:"load"`
	Capacity   int32                   `json:"capacity"`
	SkillXP    [data.SkillSlots]int32  `json:"skill_xp"`
	Skills     []HeadlessSkillSnapshot `json:"skills"`
}

type HeadlessWeaponSnapshot struct {
	Code    uint16 `json:"code"`
	Name    string `json:"name"`
	Defense int32  `json:"defense"`
}

type HeadlessWornSnapshot struct {
	Slot       int      `json:"slot"`
	Code       uint16   `json:"code"`
	Name       string   `json:"name"`
	Info       []string `json:"info"`
	Defense    int32    `json:"defense"`
	Absorption int32    `json:"absorption"`
}

type HeadlessAppearance struct {
	Body      string `json:"body"`
	BodyDir   string `json:"body_dir"`
	FigureDir string `json:"figure_dir"`
	Face      int    `json:"face"`
	Mage      bool   `json:"mage"`
}

type HeadlessSkillSnapshot struct {
	Slot  int    `json:"slot"`
	Name  string `json:"name"`
	Level int32  `json:"level"`
	XP    int32  `json:"xp"`
}

// HeadlessEvent is one step's machine-readable record. State is the front-end
// stage's observation and World the mission stage's; a run emits one or the
// other, never both, because a step belongs to exactly one stage.
type HeadlessEvent struct {
	Step    int                 `json:"step"`
	Command string              `json:"command"`
	Name    string              `json:"name,omitempty"`
	State   *HeadlessState      `json:"state,omitempty"`
	World   *HeadlessWorldState `json:"world,omitempty"`
}

// ReadHeadlessScenario strictly decodes and validates a scenario file.
func ReadHeadlessScenario(path string) (HeadlessScenario, error) {
	f, err := os.Open(path)
	if err != nil {
		return HeadlessScenario{}, err
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	dec.DisallowUnknownFields()
	var s HeadlessScenario
	if err := dec.Decode(&s); err != nil {
		return HeadlessScenario{}, fmt.Errorf("decode scenario: %w", err)
	}
	var extra any
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return HeadlessScenario{}, errors.New("decode scenario: trailing JSON value")
		}
		return HeadlessScenario{}, fmt.Errorf("decode scenario: %w", err)
	}
	if err := s.Validate(); err != nil {
		return HeadlessScenario{}, err
	}
	base, err := filepath.Abs(filepath.Dir(path))
	if err != nil {
		return HeadlessScenario{}, err
	}
	if s.OriginalSaves != "" && !filepath.IsAbs(s.OriginalSaves) {
		s.OriginalSaves = filepath.Clean(filepath.Join(base, s.OriginalSaves))
	}
	if s.Saves != "" && !filepath.IsAbs(s.Saves) {
		s.Saves = filepath.Clean(filepath.Join(base, s.Saves))
	}
	return s, nil
}

// headlessCommandSpec is one command's grammar: which stage runs it, the first
// scenario version that has it, which parameters it accepts and which of those
// it requires.
//
// IT IS A TABLE AND NOT A SWITCH so that the parameter set, the stage and the
// version are stated once each. The predecessor spelled the accept-list twice —
// once as a switch arm and once as a list of field names — and a parameter added
// to one and not the other is accepted by a validator that reports it as
// unexpected.
type headlessCommandSpec struct {
	// stages is which stages run this command and, for each, the scenario
	// version that stage got it in.
	//
	// IT IS A VERSION PER STAGE AND NOT ONE VERSION PER COMMAND. wait_until
	// reached the mission stage in version 2 and the front-end stage in
	// version 3, and a single number could say only one of those: either a
	// version-2 front-end file would silently acquire a command its author
	// never wrote, or a version-2 mission file would lose one it did.
	stages   map[string]int
	params   []string
	required []string
}

func frontEndAt(v int) map[string]int { return map[string]int{StageFrontEnd: v} }
func missionAt(v int) map[string]int  { return map[string]int{StageMission: v} }

var headlessCommands = map[string]headlessCommandSpec{
	"key":            {stages: frontEndAt(1), params: []string{"key"}, required: []string{"key"}},
	"activate":       {stages: frontEndAt(1), params: []string{"target"}, required: []string{"target"}},
	"select_control": {stages: frontEndAt(1), params: []string{"target"}, required: []string{"target"}},
	"open_menu":      {stages: frontEndAt(1), params: []string{"target"}, required: []string{"target"}},
	"capture":        {stages: frontEndAt(1), params: []string{"name"}, required: []string{"name"}},
	"select_member":  {stages: frontEndAt(1), params: []string{"id"}, required: []string{"id"}},
	"save":           {stages: frontEndAt(1)},
	"save_open":      {stages: frontEndAt(9)},
	"save_edit":      {stages: frontEndAt(9), params: []string{"directory", "name", "target"}},
	"save_action":    {stages: frontEndAt(9), params: []string{"target"}, required: []string{"target"}},
	"save_select":    {stages: frontEndAt(9), params: []string{"target"}, required: []string{"target"}},
	"load":           {stages: frontEndAt(1), params: []string{"target"}, required: []string{"target"}},
	"abort_game":     {stages: frontEndAt(5)},
	"assert_member":  {stages: frontEndAt(1), params: []string{"member"}, required: []string{"member"}},
	"assert_state":   {stages: frontEndAt(1), params: []string{"state"}, required: []string{"state"}},

	// wait_ticks is the one command both stages have had from the start. On
	// the front-end stage it steps the controller, on the mission stage the
	// world; either way it advances exactly the named number of fixed ticks
	// and reads no clock.
	"wait_ticks": {stages: map[string]int{StageFrontEnd: 1, StageMission: 1},
		params: []string{"ticks"}, required: []string{"ticks"}},

	"order":        {stages: missionAt(2), params: []string{"unit", "order", "target", "x", "y"}, required: []string{"unit", "order"}},
	"assert_unit":  {stages: missionAt(2), params: []string{"unit", "expect"}, required: []string{"unit", "expect"}},
	"assert_world": {stages: missionAt(2), params: []string{"world"}, required: []string{"world"}},
	"report":       {stages: missionAt(2), params: []string{"name"}, required: []string{"name"}},
	"kill":         {stages: missionAt(6), params: []string{"unit"}, required: []string{"unit"}},
	"kill_player":  {stages: missionAt(6), params: []string{"player"}, required: []string{"player"}},
	"place":        {stages: missionAt(6), params: []string{"unit", "x", "y"}, required: []string{"unit", "x", "y"}},
	"pick_item":    {stages: missionAt(6), params: []string{"unit", "x", "y"}, required: []string{"unit", "x", "y"}},
	"heal":         {stages: missionAt(6), params: []string{"unit"}, required: []string{"unit"}},

	// wait_until steps whichever driver the stage names until its condition
	// holds. The condition forms are disjoint per stage and Validate refuses
	// the other stage's form.
	"wait_until": {stages: map[string]int{StageMission: 2, StageFrontEnd: 3},
		params: []string{"until", "ticks"}, required: []string{"until"}},

	// create_character drives the production generation screen.
	"create_character": {stages: frontEndAt(3), params: []string{"character"}, required: []string{"character"}},

	// pointer dispatches one edge of a mouse gesture at a surface the
	// production hit test resolves (1005 round 2). Three of them make a drag
	// and two make a tap, which is the distinction TapSlop decides and a
	// single "drag" command could not express.
	"pointer":          {stages: frontEndAt(4), params: []string{"action", "at"}, required: []string{"action", "at"}},
	"assert_inventory": {stages: frontEndAt(4), params: []string{"inventory"}, required: []string{"inventory"}},
	"assert_shop":      {stages: frontEndAt(4), params: []string{"shop"}, required: []string{"shop"}},
}

// commandStages names the stages a command runs on, in a stable order, for a
// refusal that has to name them. A command on one stage reads "stage %q", which
// is the wording the refusal has always had; a command on both never reaches
// this sentence, because there is no third stage to be refused from.
func commandStages(spec headlessCommandSpec) string {
	out := make([]string, 0, len(spec.stages))
	for _, stage := range []string{StageFrontEnd, StageMission} {
		if _, ok := spec.stages[stage]; ok {
			out = append(out, fmt.Sprintf("%q", stage))
		}
	}
	if len(out) == 1 {
		return "stage " + out[0]
	}
	return "stages " + strings.Join(out, " and ")
}

// StageName is the stage this scenario runs on, with the default applied.
func (s HeadlessScenario) StageName() string {
	stage := strings.ToLower(strings.TrimSpace(s.Stage))
	if stage == "" {
		return StageFrontEnd
	}
	return stage
}

func (s HeadlessScenario) Validate() error {
	if s.Version < 1 || s.Version > HeadlessScenarioVersion {
		return fmt.Errorf("scenario version %d: this build reads versions 1 to %d",
			s.Version, HeadlessScenarioVersion)
	}
	if s.Version < 2 && (s.Stage != "" || s.Assets != "" || s.Mission != 0 ||
		s.Difficulty != "" || s.Mage || s.World != nil) {
		return fmt.Errorf("version %d scenario: stage, assets, mission, difficulty, mage and world "+
			"were introduced in version 2", s.Version)
	}
	if s.Window != nil && s.Version < 4 {
		return fmt.Errorf("version %d scenario: window was introduced in version 4", s.Version)
	}
	if s.Window != nil && (s.Window.W <= 0 || s.Window.H <= 0) {
		return fmt.Errorf("window %dx%d: both sides must be positive", s.Window.W, s.Window.H)
	}
	stage := s.StageName()
	switch stage {
	case StageFrontEnd:
		if s.Mission != 0 || s.Difficulty != "" || s.Mage || s.World != nil {
			return fmt.Errorf("stage %q: mission, difficulty, mage and world belong to stage %q",
				stage, StageMission)
		}
		// The front-end stage builds the whole production application, which
		// reads an installed root by construction. There is no synthetic
		// provider for it and claiming one would be a lie about what ran.
		if s.AssetSource() != AssetsInstall {
			return fmt.Errorf("stage %q runs on assets %q only", stage, AssetsInstall)
		}
	case StageMission:
		switch s.AssetSource() {
		case AssetsSynthetic:
			// A synthetic scenario states its own world and names no
			// mission: there is no archive for a mission number to index.
			if s.Mission != 0 || s.Difficulty != "" || s.Mage {
				return fmt.Errorf("assets %q: mission, difficulty and mage name an installed map",
					AssetsSynthetic)
			}
			if err := s.World.validate(); err != nil {
				return err
			}
		case AssetsInstall:
			if s.World != nil {
				return fmt.Errorf("assets %q: world is the authored world of an %q scenario",
					AssetsInstall, AssetsSynthetic)
			}
			if s.Mission <= 0 {
				return fmt.Errorf("stage %q requires a positive mission number", stage)
			}
		default:
			return fmt.Errorf("unknown assets %q: want %q or %q", s.Assets, AssetsInstall, AssetsSynthetic)
		}
		// The two save directories are the front-end stage's inputs. A
		// mission scenario reads and writes no save at all, so naming one is
		// a mistake about which stage the file is on rather than a harmless
		// spare field.
		if s.OriginalSaves != "" || s.Saves != "" {
			return fmt.Errorf("stage %q: original_saves and saves belong to stage %q", stage, StageFrontEnd)
		}
		if s.Window != nil {
			return fmt.Errorf("stage %q: window is the front-end stage's own, since only it has a screen", stage)
		}
		if _, err := headlessDifficulty(s.Difficulty); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown stage %q", s.Stage)
	}
	if len(s.Steps) == 0 {
		return errors.New("scenario has no steps")
	}
	for i, step := range s.Steps {
		if err := step.validate(s.Version, stage); err != nil {
			return fmt.Errorf("step %d (%q): %w", i+1, step.Command, err)
		}
	}
	return nil
}

// present is which parameters this step actually carries, by the names the
// command table uses.
func (s HeadlessStep) present() map[string]bool {
	set := map[string]bool{}
	add := func(name string, carried bool) {
		if carried {
			set[name] = true
		}
	}
	add("target", s.Target != "")
	add("key", s.Key != "")
	add("ticks", s.Ticks != 0)
	add("name", s.Name != "")
	add("id", s.ID != "")
	add("directory", s.Directory != "")
	add("unit", s.Unit != "")
	add("order", s.Order != "")
	add("x", s.X != nil)
	add("y", s.Y != nil)
	add("player", s.Player != nil)
	add("character", s.Character != nil)
	add("member", s.Member != nil)
	add("state", s.State != nil)
	add("until", s.Until != nil)
	add("expect", s.Expect != nil)
	add("world", s.World != nil)
	add("action", s.Action != "")
	add("at", s.At != nil)
	add("inventory", s.Inventory != nil)
	add("shop", s.Shop != nil)
	return set
}

func (s HeadlessStep) validate(version int, stage string) error {
	command := strings.ToLower(strings.TrimSpace(s.Command))
	if command == "" {
		return errors.New("missing command")
	}
	spec, known := headlessCommands[command]
	if !known {
		return fmt.Errorf("unknown command %q", s.Command)
	}
	got, runs := spec.stages[stage]
	if !runs {
		return fmt.Errorf("command belongs to %s; this file is on stage %q",
			commandStages(spec), stage)
	}
	if version < got {
		return fmt.Errorf("command reached stage %q in scenario version %d; this file declares version %d",
			stage, got, version)
	}
	legal := make(map[string]bool, len(spec.params))
	for _, name := range spec.params {
		legal[name] = true
	}
	carried := s.present()
	var unexpected []string
	for name := range carried {
		if !legal[name] {
			unexpected = append(unexpected, name)
		}
	}
	if len(unexpected) != 0 {
		sort.Strings(unexpected)
		return fmt.Errorf("unexpected parameter(s): %s", strings.Join(unexpected, ", "))
	}
	var missing []string
	for _, name := range spec.required {
		if !carried[name] {
			missing = append(missing, name)
		}
	}
	if len(missing) != 0 {
		return fmt.Errorf("missing parameter(s): %s", strings.Join(missing, ", "))
	}
	if s.Ticks < 0 {
		return errors.New("ticks must be positive")
	}
	switch command {
	case "assert_member":
		if s.Member.ID == "" {
			return errors.New("assert_member requires member.id")
		}
	case "create_character":
		return s.Character.validate()
	case "order":
		return s.validateOrder()
	case "wait_until":
		if s.Until.Control != "" && version < 7 {
			return errors.New("until.control requires scenario version 7")
		}
		return s.Until.validate(stage)
	case "assert_state":
		if version < 8 && (s.State.QuickSpells != nil || s.State.CurrentSpell != nil || s.State.SpellArmed != nil) {
			return errors.New("quick-spell state assertions require scenario version 8")
		}
		if version < 7 && (s.State.Mission != nil || s.State.Notice != "" || s.State.TownPlace != "") {
			return errors.New("state.mission, state.notice and state.town_place require scenario version 7")
		}
	case "assert_unit":
		return s.Expect.validate()
	case "assert_world":
		return s.World.validate()
	case "pointer":
		if version < 8 && strings.EqualFold(strings.TrimSpace(s.Action), "hover") {
			return errors.New("pointer hover requires scenario version 8")
		}
		if version < 8 && s.At.Spell != nil {
			return errors.New("at.spell requires scenario version 8")
		}
		return s.At.validate()
	case "assert_inventory":
		return s.Inventory.validate()
	case "assert_shop":
		return s.Shop.validate()
	}
	return nil
}

// HeadlessSnapshot reads the live party at its ownership boundary. It never
// seeds, copies into, or otherwise mutates a member.
func (f *FrontEnd) HeadlessSnapshot(screen ui.Screen) HeadlessState {
	state := HeadlessState{Screen: screen.String()}
	state.Difficulty = int(f.Difficulty)
	if state.Difficulty == 0 {
		state.Difficulty = 2
	}
	party := f.Carried
	var ids []sim.EntityID
	if screen == ui.ScreenMap && f.live != nil {
		state.ScorchedGround, state.BurnedObjects = f.live.view.ScorchedScenery()
		state.Mission = f.liveMission
		if _, kind, open := f.LiveNotice(); open {
			switch kind {
			case ui.NoticeDialogue:
				state.Notice = "dialogue"
			case ui.NoticeSuccess:
				state.Notice = "victory"
			case ui.NoticeFailure:
				state.Notice = "defeat"
			default:
				state.Notice = "outcome"
			}
		} else {
			state.Notice = "none"
		}
		party = f.liveParty
		if f.live.mission != nil {
			party = f.live.mission.party
			ids = f.live.mission.ids
		}
		state.Purse = f.live.world.Purse(sim.SelfSlot)
	} else if f.Town != nil {
		state.Purse = uint32(f.Town.Gold())
		if screen == ui.ScreenTown && f.townUI != nil {
			switch {
			case f.townUI.AtWorldMap():
				state.TownPlace = "world_map"
			case f.townUI.AtTownSquare():
				state.TownPlace = "square"
			}
		}
	}
	party = mapload.CloneParty(party)
	state.Members = make([]HeadlessMemberSnapshot, 0, len(party))
	for i, member := range party {
		m := headlessMemberFromParty(member, f.Table)
		derived, _, _ := mapload.PartySpawnWithTable(member, f.Table)
		m.Defense, m.Absorption = derived.Combat.Defence, derived.Combat.Absorption
		// OUT OF A MISSION THE LOAD COMES FROM THE SAME LAW A MISSION USES.
		// mapload.PartyLoad resolves the member's own worn and carried codes
		// through the table and folds them with sim.CarriedLoad, which is the one
		// statement of that law in this tree. The capacity is the same recompute
		// that already answered his defence.
		m.Load, m.Capacity = mapload.PartyLoad(member, f.Table), derived.Capacity
		if member.Carry != nil && member.Carry.LiveLoad != nil {
			m.Capacity = member.Carry.LiveLoad.Capacity
		}
		if i < len(ids) && f.live != nil {
			m.Entity = uint32(ids[i])
			if e, ok := headlessEntity(f.live.world, ids[i]); ok {
				m.XP, m.SkillXP = scalarXP(e.SkillXP), e.SkillXP
				m.Defense, m.Absorption = e.Defence, e.Absorption
				// INSIDE ONE, THE ENTITY'S OWN FIELDS WIN, on the same reading the two
				// lines above take: the live actor is what the player is looking at, and
				// his load moved every time he picked something up.
				m.Load, m.Capacity = e.Load, e.Capacity
				m.Skills = headlessSkills(member.Mage, e.Skill, e.SkillXP)
			}
			if eq, ok := f.live.world.Equipped(ids[i]); ok {
				m.Equipment = eq
			}
		}
		id, hasLiveID := sim.EntityID(0), i < len(ids)
		if hasLiveID {
			id = ids[i]
		}
		resolveDamage := func(code data.ItemCode) (weaponDamageInterval, bool) {
			if hasLiveID && f.live != nil {
				if equipped, ok := f.live.world.Equipped(id); ok && equipped[0] == uint16(code) {
					return liveWeaponSpellDamage(f.live.world, id, code)
				}
			}
			return storedWeaponSpellDamage(code, mapload.MemberWeapon(member, f.Table), member.Mage, f.Table)
		}
		m.Worn = headlessWorn(m.Equipment, f.Table, resolveDamage)
		state.Members = append(state.Members, m)
	}
	if f.Town != nil {
		state.Collected = len(f.Town.Documents())
	}
	primary := -1
	for i, member := range party {
		if member.StartingHero {
			primary = i
			break
		}
	}
	if primary < 0 && len(party) > 0 {
		primary = 0
	}
	if primary >= 0 {
		var items []uint16
		if screen == ui.ScreenMap && primary < len(ids) && f.live != nil {
			items, _ = f.live.world.Carried(ids[primary])
		} else if party[primary].Carry != nil {
			items = party[primary].Carry.Items
		} else {
			items = party[primary].Carried
		}
		for _, code := range items {
			if code == uint16(data.QuestDocumentCode) {
				state.Documents++
			}
		}
	}
	return state
}

func headlessWorn(slots [sim.EquipSlots]uint16, table *mapload.Table,
	resolveDamage weaponDamageResolver) []HeadlessWornSnapshot {
	out := make([]HeadlessWornSnapshot, 0, sim.EquipSlots)
	for i, raw := range slots {
		if raw == 0 {
			continue
		}
		code := data.ItemCode(raw)
		piece := HeadlessWornSnapshot{
			Slot: i + 1, Code: raw, Name: itemName(code, table),
			Info: itemInfoLinesWithWeaponDamage(code, table, resolveDamage),
		}
		if table != nil && table.Shapes != nil && table.Materials != nil {
			armourFound := false
			if table.Armors != nil {
				if armour, err := data.ArmorFromCode(code, table.Shapes, table.Materials, table.Armors); err == nil {
					piece.Defense, piece.Absorption = armour.Defence, armour.Absorption
					armourFound = true
				}
			}
			if !armourFound && table.Weapons != nil {
				weapon, err := data.WeaponFromCode(code, table.Shapes, table.Materials, table.Weapons)
				if err == nil {
					piece.Defense = weapon.Defence
				}
			}
		}
		out = append(out, piece)
	}
	return out
}

func headlessAppSnapshot(front *FrontEnd, app *ui.App) HeadlessState {
	state := front.HeadlessSnapshot(app.HeadlessGameplayScreen())
	state.Tooltip, _ = app.HeadlessTooltip()
	state.QuickSpells = front.quickSpells
	if app.HeadlessGameplayScreen() == ui.ScreenMap && front.live != nil && front.live.view != nil {
		_, state.CurrentSpell, state.SpellArmed = front.live.view.QuickSpellState()
	}
	state.Screen = app.Screen().String()
	// The generation screen's own state, present only while it is showing. It
	// is attached here rather than inside HeadlessSnapshot because it is a
	// property of the controller and not of the front end: the front end holds
	// no generation state at all between the gate that mints a model and the
	// party that comes back out.
	if chargen, ok := app.HeadlessChargenState(); ok {
		state.Chargen = &chargen
	}
	// THE INVENTORY AND THE SHOP RIDE ON EVERY EVENT (1005 round 2), on the
	// generation screen's own terms above: a failing assertion is read out of
	// the event stream, and a state carried only by the step that asserts it
	// cannot be compared against the step before.
	state.Inventory, state.Shop = front.headlessInventory(), front.headlessShop()
	if doc, ok := app.HeadlessDocumentState(); ok {
		state.Document = &doc
	}
	if dialog, ok := app.HeadlessSaveState(); ok {
		state.SaveDialog = &dialog
	}
	return state
}

func headlessEntity(w *sim.World, id sim.EntityID) (sim.Entity, bool) {
	if w == nil {
		return sim.Entity{}, false
	}
	for _, e := range w.Entities() {
		if e.ID == id {
			return e, true
		}
	}
	return sim.Entity{}, false
}

func headlessMemberFromParty(p mapload.PartyMember, table *mapload.Table) HeadlessMemberSnapshot {
	m := HeadlessMemberSnapshot{
		ID: p.ID, Name: p.Name, Temporary: p.Temporary,
		Appearance: HeadlessAppearance{Body: p.Body, BodyDir: p.BodyDir,
			FigureDir: p.FigureDir, Face: p.FigureFace, Mage: p.Mage},
		Equipment: partyEquipmentCodes(p),
	}
	if weapon := mapload.MemberWeapon(p, table); weapon != nil {
		m.Weapon = &HeadlessWeaponSnapshot{Code: uint16(weapon.Code), Name: weapon.Name, Defense: weapon.Defence}
	}
	if p.Temporary {
		m.Membership = "temporary"
	} else if p.StartingHero {
		m.Membership = "primary"
	} else {
		m.Membership = "permanent"
	}
	xp := [data.SkillSlots]int32{}
	if p.Carry != nil {
		xp = p.Carry.SkillXP
	} else {
		for i, level := range p.Hero.Skill {
			xp[i] = data.SkillXPFor(level)
		}
	}
	m.XP, m.SkillXP = scalarXP(xp), xp
	m.Skills = headlessSkills(p.Mage, p.Hero.Skill, xp)
	return m
}

func partyEquipmentCodes(p mapload.PartyMember) [sim.EquipSlots]uint16 {
	items := mapload.MemberItemEquipment(p, nil)
	var codes [sim.EquipSlots]uint16
	for i, item := range items {
		codes[i] = item.Code
	}
	return codes
}

func scalarXP(xp [data.SkillSlots]int32) int64 {
	var total int64
	for _, n := range xp {
		total += int64(n)
	}
	return total
}

func headlessSkills(mage bool, levels, xp [data.SkillSlots]int32) []HeadlessSkillSnapshot {
	out := make([]HeadlessSkillSnapshot, data.SkillSlots)
	for i := range out {
		// This is deliberately the live panel's resolver, not a second
		// headless-only class mapping. A scenario therefore cannot pass while
		// the GUI calls a mage's slot 2 Axe.
		name := ui.CharacterSkillName(mage, i)
		out[i] = HeadlessSkillSnapshot{Slot: i, Name: name, Level: levels[i], XP: xp[i]}
	}
	return out
}

// validateFrontEnd admits the front-end stage's own until forms and refuses
// the mission stage's, which name a world this stage has no direct handle on.
func (u *HeadlessUntil) validateFrontEnd() error {
	if u.Unit != "" || u.Within != nil || u.Dead != nil || u.Outcome != "" {
		return fmt.Errorf("until.unit, until.within, until.dead and until.outcome belong to stage %q",
			StageMission)
	}
	forms := 0
	if u.Screen != "" {
		forms++
		if _, err := headlessScreenNamed(u.Screen); err != nil {
			return err
		}
	}
	if u.Notice != nil {
		forms++
	}
	if u.Control != "" {
		if strings.TrimSpace(u.Control) == "" {
			return errors.New("until.control requires a nonempty label")
		}
		forms++
	}
	if forms != 1 {
		return errors.New("until names exactly one of screen, notice or control")
	}
	return nil
}

// headlessScreenNamed resolves a screen name to the production Screen value.
// The names are ui.Screen's own String output, so a scenario, a trace line and
// an event spell a screen the same way.
func headlessScreenNamed(name string) (ui.Screen, error) {
	want := strings.ToLower(strings.TrimSpace(name))
	for _, s := range []ui.Screen{ui.ScreenMenu, ui.ScreenPicker, ui.ScreenMap,
		ui.ScreenChargen, ui.ScreenTown, ui.ScreenGameMenu, ui.ScreenLoad,
		ui.ScreenDocuments, ui.ScreenEnding} {
		if s.String() == want {
			return s, nil
		}
	}
	return 0, fmt.Errorf("unknown screen %q", name)
}

// frontEndHolds reports whether a front-end condition is satisfied right now.
func (u *HeadlessUntil) frontEndHolds(app *ui.App) (bool, error) {
	if u.Control != "" {
		for _, row := range app.HeadlessRows() {
			text, target := strings.TrimSpace(row.Text), strings.TrimSpace(u.Control)
			if row.Choosable && (strings.EqualFold(text, target) ||
				(len(text) > len(target) && strings.EqualFold(text[:len(target)], target) &&
					strings.ContainsRune(" \t\r\n", rune(text[len(target)])))) {
				return true, nil
			}
		}
		return false, nil
	}
	if u.Notice != nil {
		return app.HeadlessNoticeOpen() == *u.Notice, nil
	}
	want, err := headlessScreenNamed(u.Screen)
	if err != nil {
		return false, err
	}
	return app.Screen() == want, nil
}

// headlessWaitFrontEnd steps the production application until the condition
// holds.
//
// IT PRESSES NOTHING. Stepping is the whole of it, so a notice it is waiting
// for is still open when it returns and the scenario's next step is the one
// that dismisses it. A wait that dismissed what it found would make the step
// after it depend on how the wait happened to end.
//
// THE CONDITION IS TESTED BEFORE THE FIRST STEP, so a condition that already
// holds costs no tick and no scenario has to know whether it does.
func headlessWaitFrontEnd(app *ui.App, step HeadlessStep) (string, error) {
	limit := step.Ticks
	if limit == 0 {
		limit = headlessDefaultWait
	}
	for n := 0; ; n++ {
		ok, err := step.Until.frontEndHolds(app)
		if err != nil {
			return "", err
		}
		if ok {
			return fmt.Sprintf("%s after %d tick(s)", step.Until, n), nil
		}
		if n >= limit {
			return "", fmt.Errorf("%s did not hold within %d tick(s); screen is %s",
				step.Until, limit, app.Screen())
		}
		if err := app.HeadlessStep(); err != nil {
			return "", fmt.Errorf("tick %d/%d: %w", n+1, limit, err)
		}
	}
}

// RunHeadlessScenario executes a validated scenario through the supplied
// production App. JSON events go to machine and a compact trace goes to human.
func RunHeadlessScenario(front *FrontEnd, app *ui.App, scenario HeadlessScenario, machine, human io.Writer) error {
	if front == nil || app == nil {
		return errors.New("headless run: nil front end or application")
	}
	if err := scenario.Validate(); err != nil {
		return err
	}
	if scenario.StageName() != StageFrontEnd {
		return fmt.Errorf("front-end run: scenario is on stage %q", scenario.StageName())
	}
	// THE WINDOW IS SIZED BEFORE THE FIRST STEP, through the production
	// Layout the engine itself drives, so every hit test below is asked about
	// the layout this file names (1005 round 2).
	if scenario.Window != nil {
		app.Layout(scenario.Window.W, scenario.Window.H)
	}
	captures := make(map[string]HeadlessState)
	enc := json.NewEncoder(machine)
	for i, step := range scenario.Steps {
		command := strings.ToLower(strings.TrimSpace(step.Command))
		if err := runHeadlessStep(front, app, step, captures); err != nil {
			return fmt.Errorf("step %d (%s): %w", i+1, step.Command, err)
		}
		state := headlessAppSnapshot(front, app)
		event := HeadlessEvent{Step: i + 1, Command: command, State: &state}
		if command == "capture" {
			event.Name = step.Name
		}
		if err := enc.Encode(event); err != nil {
			return fmt.Errorf("step %d output: %w", i+1, err)
		}
		fmt.Fprintf(human, "[%d] %-14s screen=%s members=%d purse=%d docs=%d\n",
			i+1, command, state.Screen, len(state.Members), state.Purse, state.Documents)
		for _, member := range state.Members {
			fmt.Fprintf(human, "    %s %-12s entity=%d xp=%d defense=%d absorption=%d load=%d capacity=%d skills=%s\n",
				member.ID, member.Name, member.Entity, member.XP, member.Defense,
				member.Absorption, member.Load, member.Capacity, headlessSkillLine(member.Skills))
		}
	}
	return nil
}

func runHeadlessStep(front *FrontEnd, app *ui.App, step HeadlessStep, captures map[string]HeadlessState) error {
	switch strings.ToLower(strings.TrimSpace(step.Command)) {
	case "key":
		return app.HeadlessKey(step.Key)
	case "activate", "select_control":
		return app.HeadlessActivate(step.Target)
	case "open_menu":
		switch strings.ToLower(strings.TrimSpace(step.Target)) {
		case "game":
			if app.Screen() != ui.ScreenMap && app.Screen() != ui.ScreenTown {
				return fmt.Errorf("game menu cannot open from %s", app.Screen())
			}
			if err := app.HeadlessKey("escape"); err != nil {
				return err
			}
			if app.Screen() != ui.ScreenGameMenu {
				return fmt.Errorf("game menu did not open; screen is %s", app.Screen())
			}
			return nil
		case "load":
			return headlessOpenLoad(app)
		default:
			return fmt.Errorf("unknown menu %q", step.Target)
		}
	case "wait_ticks":
		if app.Screen() != ui.ScreenMap {
			return fmt.Errorf("wait_ticks requires map screen, got %s", app.Screen())
		}
		for n := 0; n < step.Ticks; n++ {
			if err := app.HeadlessStep(); err != nil {
				return fmt.Errorf("tick %d/%d: %w", n+1, step.Ticks, err)
			}
		}
		return nil
	case "capture":
		if _, exists := captures[step.Name]; exists {
			return fmt.Errorf("capture %q already exists", step.Name)
		}
		captures[step.Name] = headlessAppSnapshot(front, app)
		return nil
	case "select_member":
		state := headlessAppSnapshot(front, app)
		member, err := findHeadlessMember(state, step.ID)
		if err != nil {
			return err
		}
		if member.Entity == 0 {
			return fmt.Errorf("member %q has no live entity", step.ID)
		}
		return app.HeadlessSelectEntity(member.Entity)
	case "save_edit":
		return app.HeadlessSaveEdit(step.Directory, step.Name, ui.SaveFormat(step.Target))
	case "save_action":
		return app.HeadlessSaveAction(step.Target)
	case "save_select":
		index, err := strconv.Atoi(step.Target)
		if err != nil {
			return fmt.Errorf("save_select requires a row index: %w", err)
		}
		return app.HeadlessSaveSelect(index)
	case "save", "save_open":
		if app.Screen() == ui.ScreenMap || app.Screen() == ui.ScreenTown {
			if err := app.HeadlessKey("escape"); err != nil {
				return err
			}
		}
		if app.Screen() != ui.ScreenGameMenu {
			return fmt.Errorf("save requires game menu, got %s", app.Screen())
		}
		if err := app.HeadlessGameMenuAction("save"); err != nil {
			return err
		}
		if strings.EqualFold(strings.TrimSpace(step.Command), "save_open") {
			if app.Screen() != ui.ScreenSave {
				return fmt.Errorf("save dialog is not configured")
			}
			return nil
		}
		if app.Screen() == ui.ScreenSave {
			// Historical scenarios' convenience command uses the actual dialog
			// and explicitly chooses SAV, the only format SAVE accepts. New
			// scenarios drive each visible step.
			name := fmt.Sprintf("scenario-save-%d", time.Now().UnixNano())
			if err := app.HeadlessSaveEdit("", name, ui.SaveSAV); err != nil {
				return err
			}
			if err := app.HeadlessSaveAction("save"); err != nil {
				return err
			}
		}
		if message := app.HeadlessMessage(); message != front.Words.SaveAcknowledgement {
			if message == "" {
				message = "no success status"
			}
			return fmt.Errorf("save failed: %s", message)
		}
		return nil
	case "abort_game":
		if app.Screen() == ui.ScreenMap || app.Screen() == ui.ScreenTown {
			if err := app.HeadlessKey("escape"); err != nil {
				return err
			}
		}
		if app.Screen() != ui.ScreenGameMenu {
			return fmt.Errorf("abort_game requires game menu, got %s", app.Screen())
		}
		if err := app.HeadlessGameMenuAction("abort"); err != nil {
			return err
		}
		if app.Screen() != ui.ScreenGameMenu {
			return fmt.Errorf("abort_game did not raise its confirmation; screen is %s", app.Screen())
		}
		if err := app.HeadlessGameMenuAction("confirm-abort"); err != nil {
			return err
		}
		if app.Screen() != ui.ScreenMenu {
			return fmt.Errorf("abort_game did not return to the main menu; screen is %s", app.Screen())
		}
		return nil
	case "load":
		if err := headlessOpenLoad(app); err != nil {
			return err
		}
		if err := app.HeadlessActivate(step.Target); err != nil {
			return err
		}
		if app.Screen() != ui.ScreenMap && app.Screen() != ui.ScreenTown && app.Screen() != ui.ScreenEnding {
			message := app.HeadlessMessage()
			if message == "" {
				message = "no refusal status"
			}
			return fmt.Errorf("load %q did not enter a game; screen is %s: %s", step.Target, app.Screen(), message)
		}
		return nil
	case "create_character":
		return headlessCreateCharacter(app, *step.Character)
	case "wait_until":
		_, err := headlessWaitFrontEnd(app, step)
		return err
	case "assert_member":
		return assertHeadlessMember(headlessAppSnapshot(front, app), *step.Member, captures)
	case "assert_state":
		return assertHeadlessState(headlessAppSnapshot(front, app), *step.State, captures)
	case "pointer":
		return headlessPointer(front, app, step)
	case "assert_inventory":
		return assertHeadlessInventory(front.headlessInventory(), *step.Inventory)
	case "assert_shop":
		return assertHeadlessShop(front.headlessShop(), *step.Shop)
	}
	return fmt.Errorf("unknown command %q", step.Command)
}

func headlessOpenLoad(app *ui.App) error {
	var err error
	switch app.Screen() {
	case ui.ScreenMenu:
		err = app.HeadlessKey("load")
	case ui.ScreenMap, ui.ScreenTown:
		if err = app.HeadlessKey("escape"); err != nil {
			return err
		}
		err = app.HeadlessGameMenuAction("load")
	case ui.ScreenGameMenu:
		err = app.HeadlessGameMenuAction("load")
	case ui.ScreenLoad:
		return nil
	default:
		return fmt.Errorf("load menu cannot open from %s", app.Screen())
	}
	if err != nil {
		return err
	}
	if app.Screen() != ui.ScreenLoad {
		return fmt.Errorf("load menu did not open; screen is %s", app.Screen())
	}
	return nil
}

func findHeadlessMember(state HeadlessState, id string) (HeadlessMemberSnapshot, error) {
	for _, member := range state.Members {
		if member.ID == id {
			return member, nil
		}
	}
	return HeadlessMemberSnapshot{}, fmt.Errorf("member %q is not present", id)
}

func assertHeadlessState(got HeadlessState, want HeadlessStateAssertion, captures map[string]HeadlessState) error {
	if want.QuickSpells != nil && got.QuickSpells != *want.QuickSpells {
		return fmt.Errorf("quick spells = %v, want %v", got.QuickSpells, *want.QuickSpells)
	}
	if want.CurrentSpell != nil && got.CurrentSpell != *want.CurrentSpell {
		return fmt.Errorf("current spell = %d, want %d", got.CurrentSpell, *want.CurrentSpell)
	}
	if want.SpellArmed != nil && got.SpellArmed != *want.SpellArmed {
		return fmt.Errorf("spell armed = %v, want %v", got.SpellArmed, *want.SpellArmed)
	}
	if want.Difficulty != nil && got.Difficulty != *want.Difficulty {
		return fmt.Errorf("difficulty = %d, want %d", got.Difficulty, *want.Difficulty)
	}
	if want.Screen != "" && !strings.EqualFold(got.Screen, want.Screen) {
		return fmt.Errorf("screen = %q, want %q", got.Screen, want.Screen)
	}
	if want.Mission != nil && got.Mission != *want.Mission {
		return fmt.Errorf("mission = %d, want %d", got.Mission, *want.Mission)
	}
	if want.Notice != "" && got.Notice != want.Notice {
		return fmt.Errorf("notice = %q, want %q", got.Notice, want.Notice)
	}
	if want.TownPlace != "" && got.TownPlace != want.TownPlace {
		return fmt.Errorf("town_place = %q, want %q", got.TownPlace, want.TownPlace)
	}
	if want.Purse != nil && got.Purse != *want.Purse {
		return fmt.Errorf("purse = %d, want %d", got.Purse, *want.Purse)
	}
	if want.Collected != nil && got.Collected != *want.Collected {
		return fmt.Errorf("collected_documents = %d, want %d", got.Collected, *want.Collected)
	}
	if want.Documents != nil && got.Documents != *want.Documents {
		return fmt.Errorf("documents = %d, want %d", got.Documents, *want.Documents)
	}
	if want.Document != nil {
		if got.Document == nil {
			return errors.New("document: no documents panel is open")
		}
		if err := assertHeadlessDocument(*got.Document, *want.Document); err != nil {
			return err
		}
	}
	if want.MemberCount != nil && len(got.Members) != *want.MemberCount {
		return fmt.Errorf("member count = %d, want %d", len(got.Members), *want.MemberCount)
	}
	for _, member := range want.Members {
		if err := assertHeadlessMember(got, member, captures); err != nil {
			return err
		}
	}
	return nil
}

func assertHeadlessDocument(got ui.HeadlessDocuments, want HeadlessDocumentAssertion) error {
	if want.Element != nil && got.Element != *want.Element {
		return fmt.Errorf("document element = %d, want %d", got.Element, *want.Element)
	}
	if want.Elements != nil && got.Elements != *want.Elements {
		return fmt.Errorf("document elements = %d, want %d", got.Elements, *want.Elements)
	}
	if want.Page != nil && got.Page != *want.Page {
		return fmt.Errorf("document page = %d, want %d", got.Page, *want.Page)
	}
	if want.Pages != nil && got.Pages != *want.Pages {
		return fmt.Errorf("document pages = %d, want %d", got.Pages, *want.Pages)
	}
	if want.Picture != nil && got.Picture != *want.Picture {
		return fmt.Errorf("document picture = %v, want %v", got.Picture, *want.Picture)
	}
	return nil
}

func assertHeadlessMember(got HeadlessState, want HeadlessMemberAssertion, captures map[string]HeadlessState) error {
	member, err := findHeadlessMember(got, want.ID)
	if err != nil {
		return err
	}
	if want.XP != nil && member.XP != *want.XP {
		return fmt.Errorf("member %q xp = %d, want %d", want.ID, member.XP, *want.XP)
	}
	if want.Defense != nil && member.Defense != *want.Defense {
		return fmt.Errorf("member %q defense = %d, want %d", want.ID, member.Defense, *want.Defense)
	}
	if want.Absorption != nil && member.Absorption != *want.Absorption {
		return fmt.Errorf("member %q absorption = %d, want %d", want.ID, member.Absorption, *want.Absorption)
	}
	if want.Load != nil && member.Load != *want.Load {
		return fmt.Errorf("member %q load = %d, want %d", want.ID, member.Load, *want.Load)
	}
	if want.Capacity != nil && member.Capacity != *want.Capacity {
		return fmt.Errorf("member %q capacity = %d, want %d", want.ID, member.Capacity, *want.Capacity)
	}
	if want.Temporary != nil && member.Temporary != *want.Temporary {
		return fmt.Errorf("member %q temporary = %v, want %v", want.ID, member.Temporary, *want.Temporary)
	}
	for name, level := range want.Skills {
		found := false
		for _, skill := range member.Skills {
			if strings.EqualFold(skill.Name, name) {
				found = true
				if skill.Level != level {
					return fmt.Errorf("member %q skill %q = %d, want %d", want.ID, name, skill.Level, level)
				}
				break
			}
		}
		if !found {
			return fmt.Errorf("member %q has no skill named %q", want.ID, name)
		}
	}
	if want.SameCharacterAs != "" {
		capture, ok := captures[want.SameCharacterAs]
		if !ok {
			return fmt.Errorf("capture %q does not exist", want.SameCharacterAs)
		}
		before, err := findHeadlessMember(capture, want.ID)
		if err != nil {
			return fmt.Errorf("capture %q: %w", want.SameCharacterAs, err)
		}
		before.Entity, member.Entity = 0, 0
		if !reflect.DeepEqual(before, member) {
			return fmt.Errorf("member %q differs from capture %q\n got: %#v\nwant: %#v", want.ID, want.SameCharacterAs, member, before)
		}
	}
	return nil
}

func headlessSkillLine(skills []HeadlessSkillSnapshot) string {
	parts := make([]string, 0, len(skills))
	for _, skill := range skills {
		if skill.Level != 0 || skill.XP != 0 {
			parts = append(parts, fmt.Sprintf("%s=%d(%d)", skill.Name, skill.Level, skill.XP))
		}
	}
	if len(parts) == 0 {
		return "none"
	}
	return strings.Join(parts, ",")
}
