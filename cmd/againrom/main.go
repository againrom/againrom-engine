// Command againrom is the game entry point.
//
// Usage:
//
//	againrom -assets <dir> [-check] [-markers] [-picker | -mission <n>] [-skill <name>]
//
// It opens the original 640x480 brooch main menu, composited from the bitmaps in
// main.res, and nothing else opens first (owner reversal: a normal launch must
// show the menu, not a screen the player never asked for). The menu's NEW GAME
// starts character generation for campaign mission 10 directly and, on a legal
// confirm, enters that mission. -picker changes what NEW GAME opens instead: the
// former map picker, listing campaign and loose maps, in place of generation —
// it is a debug option and does not change what a launch with no flags shows.
// Esc unwinds one screen at a time and, at the menu, exits.
//
// The window covers the screen. macOS uses native fullscreen; other desktops
// use a borderless window the size of the monitor without changing video mode.
// Esc at the main menu and the brooch's EXIT button close the game.
//
// The taskbar and the task switcher show the installed game's icon: the first
// icon group of the install's own executable, read at start like every other
// asset and never compiled in. An install whose icon cannot be read still
// starts, with the platform's default icon and one line on the error stream.
//
// -check does the whole startup headlessly: it opens the required archives,
// validates the menu assets, builds the map list and reports what NEW GAME (or
// an explicit -mission) would open, without opening a window. With -picker it
// reports the former one-line install summary instead, because the picker
// route opens nothing on its own.
//
// -mission is a direct entry for development: GIVEN EXPLICITLY, it bypasses the
// main menu and opens character generation for that campaign mission number
// immediately, then that mission's map screen on a legal confirm — the same
// route NEW GAME's own default takes, entered without a click. Left at its
// default of 10, it only NAMES the mission NEW GAME opens; it does not by
// itself open anything, so a bare launch still shows the menu. -picker is
// mutually exclusive with an explicit -mission and changes what NEW GAME opens
// instead of bypassing the menu.
//
// -markers draws a diagnostic cross on every placed structure, every placed
// unit and every static object of the map on screen. One question, three
// glyphs: the game asks whether the instrument is on, not which part of it.
// What the cross is still good for is unchanged: the art and the cross are
// derived independently, so a sprite standing away from the cell marked for
// it is a placement bug you can see rather than one you have to measure.
//
// Character generation is not a flag any more (owner). STARTING a campaign
// opens it: -mission N shows a sex, a class, a skill and the four statistics
// of the spread before that mission's map screen, and so does a mission row
// chosen from the map list NEW GAME opens. What the player asks for when he
// starts a campaign is a character to play it with, and there was no reason
// for that to depend on remembering a flag.
//
// CONTINUING a campaign does not open it. A mission won hands its successor the
// character it was won with, and a generation screen between the two would throw
// that character away; nothing on that path reaches the screen (pkg/ui's own
// ChargenGate carries why it cannot). -mission N is a fresh start because a
// process that begins inside a mission has nothing to have carried.
//
// -chargen is still ACCEPTED and now DOES NOTHING. It is kept rather than
// deleted because deleting it turns an existing invocation into a hard parse
// error, which is a worse way to learn the flag is gone than a help line saying
// so. It no longer requires -mission either, because it no longer requires
// anything.
//
// -skill trains the party's hero in a named skill instead of the shipped
// default (blade), by NAME rather than by number — one of blade, axe,
// bludgen, pike or shooting, case-insensitive. It is a front-end setting
// applied before the install loads, so it changes which weapon character
// generation hands the hero and, through that weapon, his reach, his cadence
// and, for a melee skill, his damage, to-hit and defence. Without it every
// existing path behaves exactly as it does today. An unrecognised name is
// reported and exits non-zero, the same as any other malformed flag.
//
// The asset root comes from -assets, or AGAINROM_ASSETS, or an install found
// at the working directory or beside this binary, and is never compiled in.
package main

import (
	"errors"
	"flag"
	"fmt"
	"image"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"againrom/internal/buildinfo"
	"againrom/pkg/audio"
	"againrom/pkg/base"
	"againrom/pkg/data"
	"againrom/pkg/game"
	"againrom/pkg/mod"
	"againrom/pkg/modrt"
	"againrom/pkg/ui"
)

const usage = "usage: againrom [-assets <dir>] [-check | --headless <scenario.json>] [-markers] [-chicken] [-seed <n>] [-original-random] [-picker | -mission <n>] [-skill <name>] [-chargen] [-base <id>] [-sound] [-volume <n>] [-saves <dir>] [-mods <id,id,...>] [-mods-dir <dir>] [-mod-setting <mod>.<key>=<value>]... [-mods-accept-unmarked]\n       with no -assets and no AGAINROM_ASSETS, the working directory and then this binary's own folder are searched for an install"

// markersUsage is the -markers flag's own help text. Named rather than
// written inline so the wording has ONE home, mirroring cmd/mapview's flag
// help: parse discards the flag set's output, so this string reaches no user
// and no test, and a second copy of it could drift with nothing anywhere to
// notice.
const markersUsage = "diagnostic: draw a marker on each placed structure, unit and static object (off by default; -markers turns all three on)"

const defaultMarkers = false

// missionUsage is the -mission flag's own help text. Named beside
// markersUsage and for its reason: parse discards the flag set's output, so
// a second copy of this string could drift with nothing anywhere to notice.
const missionUsage = "override the startup campaign mission (default: the first mission of the install's campaign registry)"

// pickerUsage names the debug map-picker route (owner reversal: a normal
// launch opens the main menu and NEW GAME there must not reach the picker
// without this flag). It changes what NEW GAME itself opens rather than
// retaining a startup route — the main menu is the same screen either way,
// and -picker is mutually exclusive with an explicit -mission.
const pickerUsage = "debug: make NEW GAME open the former map picker (campaign and loose maps) instead of character generation"

// chargenUsage is the -chargen flag's own help text, named beside markersUsage
// and missionUsage and for their exact reason: parse discards the flag set's
// output, so a second copy of this string could drift from this one with
// nothing anywhere to notice.
//
// THE FLAG IS INERT AS OF 0140 and this string is the only place it is
// mentioned outside parse: generation runs whenever a campaign is started now,
// which -mission always is, so there is nothing left for it to turn on. Nothing
// in run reads its value — it is not even a field of options any more — so
// "inert" is a property of the code and not a promise this comment makes.
const chargenUsage = "accepted and ignored: character generation now runs whenever a campaign is started"

// startMission is the campaign mission NEW GAME opens character generation
// for: -mission when given, otherwise front.NewGameMission, read from the
// installed campaign registry. The map picker is reachable only through
// -picker, and only as what NEW GAME opens instead of generation.
func (o options) startMission(front *game.FrontEnd) int {
	if o.missionSet {
		return o.mission
	}
	return front.NewGameMission()
}

// skillNone is the resolved -skill value that means the flag was not given:
// data.SkillGeneral, which is ALREADY not a slot any hero can be trained in
// (StartingWeaponName's own refusal) — so "leave it alone" and "a slot
// that trains nothing" need no second sentinel to tell apart.
const skillNone = data.SkillGeneral

// skillUsage is the -skill flag's own help text, named beside markersUsage
// and missionUsage and for their reason: parse discards the flag set's
// output, so a second copy of this string could drift with nothing anywhere
// to notice. It is built from data.SkillName over skillSlot's own legal
// range rather than a second, hand-written list of skill spellings, so the
// values this flag accepts and the values printed here cannot part company.
var skillUsage = buildSkillUsage()

func buildSkillUsage() string {
	var names []string
	for slot := int32(data.SkillGeneral + 1); slot < data.SkillSlots; slot++ {
		names = append(names, strings.ToLower(data.SkillName(slot)))
	}
	return "train the party's hero in this skill instead of the shipped default (one of: " +
		strings.Join(names, ", ") + ")"
}

const (
	defaultSound  = true
	defaultVolume = audio.MasterUnit
)

// soundUsage and volumeUsage are -sound's and -volume's own help text, named
// beside markersUsage, missionUsage and skillUsage and for their shared
// reason: parse discards the flag set's output, so a second copy of either
// string could drift with nothing anywhere to notice.
const (
	soundUsage  = "play sound and music (overrides the saved preference; on by default)"
	volumeUsage = "sound and music master volume, 0-100 (overrides the saved preference)"
	savesUsage  = "directory for saved games (default: saves/ beside the binary; inside an install, Againrom/saves or the user configuration profile)"
)

// baseUsage is -base's own help text. The ids are the known profiles', so the
// flag and the profile list cannot part company.
var baseUsage = "refuse to run unless the asset root is detected as this base profile (one of: " +
	strings.Join(base.IDs(), ", ") + "); -check names the detected profile either way"

// The mod flags' own help text. Mods are resolved, ordered and run before the
// front end is built; -check lists them.
const (
	modsUsage               = "comma-separated ids of the mods to load; each is a folder with a mod.toml in the mods directory (the load order follows requires and load-after, not this list)"
	modsDirUsage            = "directory holding the mod folders (default: mods beside this binary)"
	modSettingUsage         = "a mod setting as <mod>.<key>=<value>; repeat the flag for several (a setting not given keeps the mod's default)"
	modsAcceptUnmarkedUsage = "let a saved game that carries no mod mark start under the active mods (without it such a save is refused)"
)

// resolveMods loads the mods o names. A missing or malformed mod is an error
// naming it. Without -mods it returns nothing and reads no directory.
func resolveMods(o options, executable string) ([]mod.Entry, error) {
	if len(o.mods) == 0 {
		return nil, nil
	}
	dir := o.modsDir
	if dir == "" {
		if executable == "" {
			return nil, errors.New("no -mods-dir and the executable's folder is unknown")
		}
		dir = mod.DefaultDir(filepath.Dir(executable))
	}
	return mod.Resolve(dir, o.mods)
}

// activateMods orders and runs the mods o names for the base of the install at
// root and returns what they set. Without -mods it runs nothing and returns the
// original game's rules.
func activateMods(o options, mods []mod.Entry, root string) (modrt.Result, error) {
	base := game.BaseID(game.InspectInstall(root))
	return modrt.Load(mods, base, o.modSettings, modrt.Options{})
}

// modLines is the -check report of the active mods: one line for the set, one
// per mod in load order with its settings, and the rules the mods set.
func modLines(res modrt.Result) []string {
	lines := []string{fmt.Sprintf("againrom: %d mod(s) active on %s, mod-set %s", len(res.Ordered), res.Set.Base, res.Set.Digest())}
	for i, m := range res.Ordered {
		entry := res.Set.Mods[i]
		var settings []string
		for _, v := range entry.Settings {
			settings = append(settings, v.Key+"="+v.Value.String())
		}
		lines = append(lines, fmt.Sprintf("againrom: mod %d: %s %s %q applies-to=%s settings=%s dir=%s", i+1,
			m.Manifest.ID, m.Manifest.Version, m.Manifest.Title, strings.Join(m.Manifest.AppliesTo, ","),
			strings.Join(settings, ","), m.Dir))
	}
	lines = append(lines, fmt.Sprintf("againrom: rules skill_cap=%d", res.Rules.SkillCap()))
	if !res.Items.Empty() {
		lines = append(lines, fmt.Sprintf("againrom: items added=%d changed=%d", len(res.Items.Rows), len(res.Items.Changes)))
	}
	if !res.Characters.Empty() {
		lines = append(lines, fmt.Sprintf("againrom: characters edited=%d", len(res.Characters.Rows)))
	}
	if !res.Screens.Empty() {
		main, gameMenu := res.Screens.Count()
		lines = append(lines, fmt.Sprintf("againrom: screens=%d main-menu=%d game-menu=%d", len(res.Screens.Screens), main, gameMenu))
	}
	if !res.Companions.Empty() {
		lines = append(lines, fmt.Sprintf("againrom: companion joins=%d", len(res.Companions.Joins)))
	}
	if !res.Spells.Empty() {
		lines = append(lines, fmt.Sprintf("againrom: spells edited=%d global=%d", len(res.Spells.Rows), len(res.Spells.Global.Globals())))
	}
	return lines
}

func skillSlot(name string) (int32, bool) {
	name = strings.ToLower(name)
	for slot := int32(data.SkillGeneral + 1); slot < data.SkillSlots; slot++ {
		if strings.ToLower(data.SkillName(slot)) == name {
			return slot, true
		}
	}
	return 0, false
}

func main() {
	os.Exit(run(os.Args[1:], os.Getenv, os.Stdout, os.Stderr))
}

// desktop is the window system a normal launch ends in: the call that gives the
// window its icon and the call that opens the window and runs the game in it. A
// test replaces both to watch a launch without opening a window.
var desktop = struct {
	setIcon func([]image.Image)
	run     func(*ui.App) error
}{ui.SetWindowIcon, (*ui.App).Run}

// installWindowIcon hands the installed game's icon to the window. An install
// whose icon cannot be read still starts: the reason is reported and the window
// keeps the platform's default icon.
func installWindowIcon(root string, stderr io.Writer) {
	images, err := game.WindowIcon(root)
	if err != nil {
		fmt.Fprintf(stderr, "againrom: window icon unavailable: %v; the window keeps the default icon\n", err)
		return
	}
	desktop.setIcon(images)
}

// launchStamp identifies the source tree a normal window was built from. A
// `go run` process is otherwise visually indistinguishable from a stale binary,
// which made the owner's live acceptance report impossible to attribute. The
// VCS values are Go build metadata; no generated version file can go stale.
func launchStamp() string { return buildinfo.Revision() }

// runMissionStage is the whole mission-stage headless route: open the archives
// under the resolved root, start the mission the scenario names, and drive it.
//
// The root is the one the caller already resolved, which is the flag or
// AGAINROM_ASSETS or neither. It is deliberately NOT o.assets: the flag is
// empty whenever the root came from the environment, which is how the game
// is normally launched, and reading the flag here would give a mission
// scenario a different install from the one every other mode uses.
// runSyntheticStage drives a scenario over the world the scenario itself
// authors. It opens nothing and needs no asset root.
func runSyntheticStage(scenario game.HeadlessScenario, stdout, stderr io.Writer) error {
	play, err := scenario.World.Build()
	if err != nil {
		return err
	}
	fmt.Fprintf(stderr, "againrom: synthetic world  %dx%d, %d unit(s), party of %d\n",
		scenario.World.Width, scenario.World.Height, len(scenario.World.Units), len(play.Party))
	return game.RunPlayScenario(play, scenario, stdout, stderr)
}

func runMissionStage(root string, scenario game.HeadlessScenario, stdout, stderr io.Writer) error {
	archives, err := game.OpenArchives(root)
	if err != nil {
		return err
	}
	play, ms, err := game.StartScenarioMission(archives, scenario)
	if err != nil {
		return err
	}
	fmt.Fprintf(stderr, "againrom: mission %d  %s  %d unit(s), party of %d, %d script arm(s) this build cannot run\n",
		ms.Number, ms.Address, len(ms.World.Entities())-len(play.Party), len(play.Party),
		len(ms.World.Script().Unsupported()))
	if ms.RaiseErr != nil {
		fmt.Fprintln(stderr, "againrom: this mission's script did not decode:", ms.RaiseErr)
	}
	return game.RunPlayScenario(play, scenario, stdout, stderr)
}

func launchSource() string {
	dir, err := os.Getwd()
	if err != nil {
		return "unknown"
	}
	if abs, err := filepath.Abs(dir); err == nil {
		return abs
	}
	return dir
}

// options is this command's parsed command line.
type options struct {
	assets  string
	check   bool
	markers bool
	chicken bool
	// seed is -seed, the fixed session seed, when seedSet; originalRandom is
	// -original-random, the original's random number generator.
	seed           uint64
	seedSet        bool
	originalRandom bool
	picker         bool
	mission        int

	// missionSet is whether -mission was given ON THE COMMAND LINE, distinct
	// from mission's VALUE: a bare launch leaves mission at zero and
	// startMission reads the installed campaign instead; only this field
	// tells a bare launch from an explicit number. run() reads it to decide whether -mission
	// bypasses the main menu as a direct entry, or a bare launch instead
	// leaves the number for NEW GAME to open when it is pressed.
	missionSet bool

	// skill is -skill's value, already resolved to its slot by skillSlot —
	// skillNone when the flag was not given. Resolving it here rather than in
	// frontEnd means a bad name is reported at parse time, the same moment
	// every other malformed flag is.
	skill int32

	sound               bool
	volume              int
	soundSet, volumeSet bool

	// saves is the explicit override; an empty value selects the launch profile.
	saves string

	// headless names a versioned declarative scenario. Unlike -check it drives
	// the production App/controller/simulation pipeline and omits only the
	// window, renderer and physical-input adapters.
	headless      string
	cutsceneCheck string
	mediaCheck    bool
	soundCheck    string
	fidelityCheck string
	movies        bool
	// noMusic is -nomusic: no music archive is opened, so every music
	// request is silent.
	noMusic       bool
	fourX, eightX bool

	// mods names the mods to load, in order; modsDir is the directory holding
	// their folders, empty for the profile default.
	mods    []string
	modsDir string

	// modSettings are the -mod-setting values in the order given;
	// modsAcceptUnmarked is -mods-accept-unmarked.
	modSettings        []mod.SettingFlag
	modsAcceptUnmarked bool

	// base is -base: the profile id the asset root must be detected as, empty
	// for any.
	base string

	// version asks for the version line and nothing else.
	version bool
}

// parse reads the command line. Split out of run so a test can read the
// flags and their defaults — the marker default in particular, which is
// contract and not a convenience — without opening a window.
func parse(args []string) (options, error) {
	var o options
	fs := flag.NewFlagSet("againrom", flag.ContinueOnError)
	fs.SetOutput(io.Discard) // the error is reported once, by run
	fs.StringVar(&o.assets, "assets", "", "path to the game asset root (overrides AGAINROM_ASSETS, which overrides the install found at the working directory or beside this binary)")
	fs.BoolVar(&o.check, "check", false, "load the assets and the map list, print a summary, and exit without a window")
	fs.BoolVar(&o.markers, "markers", defaultMarkers, markersUsage)
	fs.BoolVar(&o.chicken, "chicken", false, "set the ROM1 #Chicken state at every mission start")
	fs.Func("seed", "fix the session seed every new game draws from (default: the clock at each new game)", func(v string) error {
		n, err := strconv.ParseUint(v, 0, 64)
		if err != nil {
			return fmt.Errorf("-seed must be an unsigned integer: %w", err)
		}
		o.seed, o.seedSet = n, true
		return nil
	})
	fs.BoolVar(&o.originalRandom, "original-random", false, "draw every placed random consumer from the original game's random number generator")
	fs.BoolVar(&o.picker, "picker", false, pickerUsage)
	fs.IntVar(&o.mission, "mission", 0, missionUsage)
	// -chargen IS BOUND TO A THROWAWAY (0140). The flag must still PARSE — an
	// invocation that carries it has to keep working — but nothing may read it,
	// because generation is unconditional now and a value nobody reads is the
	// only shape of "inert" a later reader cannot mistake for a switch that
	// happens to be unused this month.
	var ignoredChargen bool
	fs.BoolVar(&ignoredChargen, "chargen", false, chargenUsage)
	fs.BoolVar(&o.sound, "sound", defaultSound, soundUsage)
	fs.IntVar(&o.volume, "volume", defaultVolume, volumeUsage)
	// The flag itself is a NAME, empty by default meaning "leave the
	// trained skill alone" — o.skill starts at skillNone and stays there
	// unless a name is given, exactly that default's own resolved form.
	var skill string
	fs.StringVar(&skill, "skill", "", skillUsage)
	fs.StringVar(&o.saves, "saves", "", savesUsage)
	fs.StringVar(&o.base, "base", "", baseUsage)
	fs.BoolVar(&o.version, "version", false, "print the program name, version and source revision, and exit")
	var mods string
	fs.StringVar(&mods, "mods", "", modsUsage)
	fs.StringVar(&o.modsDir, "mods-dir", "", modsDirUsage)
	fs.Func("mod-setting", modSettingUsage, func(v string) error {
		f, err := mod.ParseSettingFlag(v)
		if err != nil {
			return err
		}
		o.modSettings = append(o.modSettings, f)
		return nil
	})
	fs.BoolVar(&o.modsAcceptUnmarked, "mods-accept-unmarked", false, modsAcceptUnmarkedUsage)
	fs.StringVar(&o.headless, "headless", "", "run a versioned JSON scenario through the production controller without opening a window")
	fs.BoolVar(&o.movies, "movies", true, "play installed startup, new-game and mission movies")
	fs.BoolVar(&o.noMusic, "nomusic", false, "play no music: every music request is silent")
	fs.BoolVar(&o.fourX, "4x", false, "select VIDEO4 mission movies (overrides -8x)")
	fs.BoolVar(&o.eightX, "8x", false, "select VIDEO8 mission movies")
	fs.StringVar(&o.cutsceneCheck, "cutscene-check", "", "read-only native/App witness; write frames only to this existing private directory")
	fs.BoolVar(&o.mediaCheck, "media-check", false, "check movie events and town speech without writing files or loading the native decoder")
	fs.StringVar(&o.soundCheck, "sound-options-check", "", "check installed sound controls and active players; write only to this existing private profile directory")
	fs.StringVar(&o.fidelityCheck, "owner-fidelity-check", "", "check installed HUD, pathfinding preference and LOAD; write only to this existing private profile directory")
	if err := fs.Parse(args); err != nil {
		return o, err
	}
	fs.Visit(func(f *flag.Flag) {
		o.missionSet = o.missionSet || f.Name == "mission"
		o.soundSet = o.soundSet || f.Name == "sound"
		o.volumeSet = o.volumeSet || f.Name == "volume"
	})
	if o.volume < 0 || o.volume > audio.MasterUnit {
		return o, fmt.Errorf("-volume must be between 0 and %d", audio.MasterUnit)
	}
	if strings.TrimSpace(mods) == "" && (len(o.modSettings) != 0 || o.modsAcceptUnmarked) {
		return o, errors.New("-mod-setting and -mods-accept-unmarked need -mods")
	}
	if o.picker && o.missionSet {
		return o, errors.New("-picker and -mission are mutually exclusive")
	}
	if o.base != "" {
		if !base.Nameable(o.base) {
			return o, fmt.Errorf("unknown -base %q; one of: %s", o.base, strings.Join(base.IDs(), ", "))
		}
	}
	if o.check && o.headless != "" {
		return o, errors.New("-check and -headless are mutually exclusive")
	}
	if (o.cutsceneCheck != "" || o.mediaCheck) && (o.check || o.headless != "" || !o.movies) {
		return o, errors.New("media checks require movies and exclude -check/-headless")
	}
	if o.cutsceneCheck != "" && o.mediaCheck {
		return o, errors.New("choose either -media-check or -cutscene-check")
	}
	if o.soundCheck != "" && (o.saves == "" || o.check || o.headless != "" || o.mediaCheck || o.cutsceneCheck != "") {
		return o, errors.New("-sound-options-check requires an explicit -saves directory and excludes other check modes")
	}
	if o.fidelityCheck != "" && (o.saves == "" || o.check || o.headless != "" || o.mediaCheck || o.cutsceneCheck != "" || o.soundCheck != "") {
		return o, errors.New("-owner-fidelity-check requires an explicit -saves directory and excludes other check modes")
	}
	if strings.TrimSpace(mods) != "" {
		for _, id := range strings.Split(mods, ",") {
			o.mods = append(o.mods, strings.TrimSpace(id))
		}
	}
	o.skill = skillNone
	if skill != "" {
		slot, ok := skillSlot(skill)
		if !ok {
			return o, fmt.Errorf("unknown -skill %q; %s", skill, skillUsage)
		}
		o.skill = slot
	}
	return o, nil
}

// loadSources selects one profile for saves, options and the hall. Original
// saves remain readable from the resolved asset root if profile selection fails.
func loadSources(root string, o options, executable, userConfigDir string) (game.RuntimeProfile, game.OriginalStore, error) {
	orig := game.OriginalStore{Dir: root}
	profile, err := game.ResolveRuntimeProfile(o.saves, executable, userConfigDir, root)
	return profile, orig, err
}

// frontEnd loads an install and applies the options that configure the
// front-end itself.
//
// Split out of run for the same reason parse is: this is where -markers becomes
// the marker selection every map the session opens is loaded with, and a test
// can drive it against a synthetic install where run() would try to open a
// window.
func frontEnd(root string, o options, preferences game.OptionsStore) (*game.FrontEnd, error) {
	// -skill, BEFORE game.NewFrontEnd: NewFrontEnd calls LoadDefinitions, which
	// resolves the party's starting weapon against whatever game.PartySkillSlot
	// answers AT THAT MOMENT (table.go's own resolveStartingWeapon). Setting it
	// after NewFrontEnd has returned would reach a hero already built and
	// change nothing.
	if o.skill != skillNone {
		if err := game.SetPartySkill(o.skill); err != nil {
			return nil, err
		}
	}
	// Resolve local preferences before opening any audio device. Each explicit
	// flag overrides only its own value, including true and 100. Launching
	// alone never writes those overrides back to the preference file.
	sound := startupSoundOptions(preferences, o)
	game.SetSoundOptions(sound)
	channels, _ := preferences.SoundChannelVolumes()
	game.SetSoundChannelVolumes(channels)
	front, err := game.NewFrontEnd(root)
	if err != nil {
		return nil, err
	}
	// One flag, all three overlays: the game asks whether the instrument is on,
	// not which part of it (DD33). The independent toggles are the developer
	// viewer's surface.
	//
	// The static-object cross rides this flag rather than one of its own because
	// it is the SAME question — and the layer's art is not on this switch at all:
	// trees and stones are the map's content, drawn unconditionally by the load
	// path, so -markers=false leaves them on screen and takes the crosses off.
	front.Markers = game.Markers{Objects: o.markers, Units: o.markers, Statics: o.markers}
	front.SetChickenAtMissionStart(o.chicken)
	front.SetRandomLaunch(o.seed, o.seedSet, o.originalRandom)
	front.Options = preferences
	front.LoadOptions()
	if o.noMusic {
		// DIV-2860: the original's -nomusic clears music availability.
		front.MusicBank = nil
	}
	return front, nil
}

func startupSoundOptions(store game.OptionsStore, o options) game.SoundOptions {
	sound, _ := store.SoundOptions(game.SoundOptions{Enabled: o.sound, Volume: o.volume})
	if o.soundSet {
		sound.Enabled = o.sound
	}
	if o.volumeSet {
		sound.Volume = o.volume
	}
	if o.headless != "" {
		sound.Enabled = false
	}
	return sound
}

// armNewGameDoor decides what generation for o.startMission is wired to, once app
// and front are built and that mission is already known to name a real mission
// (run's own MissionMap check at the call site). It is split out of run so a
// test can drive App.HeadlessActivate("new game") against the exact wiring a
// window gets, without opening one — see cmd/againrom's own
// TestNewGameOpensGenerationDirectly.
//
// o.missionSet, NOT o.mission's value, decides between the two arms: the flag
// value alone does not say whether the flag was given, so the number cannot tell
// a bare launch from an explicit "-mission 10". SET means -mission is a direct
// entry for development and bypasses the menu, arming generation before the
// first window frame exactly as normal startup always did before the owner's
// reversal (docs/DIVERGENCES.md DIV-523). UNSET means the menu is what a bare
// launch shows, and NEW GAME there arms the same setup instead, through
// SetNewGameChargen.
//
// begin IS ONE CLOSURE SHARED BY BOTH ARMS. Whichever door reaches it, a
// confirmed spread does what front.NewGameBegin(o.startMission(front)) does
// for the install's game: the first game opens that mission, the second game
// commits its new campaign and shows its first town.
func armNewGameDoor(app *ui.App, front *game.FrontEnd, o options) error {
	// A base that ships no generation art opens its first mission from NEW GAME
	// (front.App installed that); there is no generation for -mission to open.
	if p := front.Base().Profile; p.Limits.NoCharacterGeneration {
		if o.missionSet {
			return fmt.Errorf("-mission opens character generation, which base %s does not ship; start New Game from the menu, it opens mission %d", p.ID, p.Mission())
		}
		return nil
	}
	begin := front.NewGameBegin(o.startMission(front))
	if o.missionSet {
		return app.OpenChargen(ui.NewChargen(front.ChargenSetup()), begin)
	}
	app.SetNewGameChargen(func() *ui.ChargenEntry {
		return &ui.ChargenEntry{Model: ui.NewChargen(front.ChargenSetup()), Begin: begin}
	})
	return nil
}

func chargenSummary(front *game.FrontEnd) string {
	setup := front.ChargenSetup()
	names := make([]string, 0, len(setup.Choices)+len(setup.Stats))
	for _, c := range setup.Choices {
		names = append(names, c.Name)
	}
	for _, s := range setup.Stats {
		names = append(names, s.Name)
	}
	return fmt.Sprintf("againrom: chargen offers %s; budget %d", strings.Join(names, ", "), setup.Budget)
}

// processDirs names the directories to search for an install when nobody
// configured an asset root: the working directory, then the directory holding
// the executable. game.DiscoverAssetRoot takes them in that order.
//
// THE WORKING DIRECTORY IS FIRST BECAUSE IT IS WHAT THE ORIGINAL USES.
// RES-DIR-037 is High for the code: rom.exe's resource manager is constructed
// before main, calls GetCurrentDirectoryA and AddDirectory(cwd), and AddArchive
// then indexes dir[0] alone with no loop, so an archive not sitting beside the
// working directory never opens whatever else is registered. The working
// directory IS the original's install lookup, and ordering the executable's own
// directory ahead of it would put an authored choice in front of a decoded fact.
//
// The executable's directory is second, and it is ours rather than the
// original's. It answers the case the working directory cannot — a shortcut that
// starts elsewhere, or a binary invoked by absolute path — and it is an addition
// layered after the original's behaviour, never in front of it. `go run` reaches
// neither: its executable lives in a build cache, and its working directory is
// the source tree.
//
// Neither candidate is compiled in; both are read from the running process, so
// boundary 3 holds.
//
// THIS IS NOT ALL THE ORIGINAL DOES. The same claim records that startup also
// registers INSTALLDIR from HKLM\SOFTWARE\1C\Allods, the stripped GetTempPathA
// result, and the CD registry value plus "Allods". None of those is implemented
// here, and the gap is DIV-430 rather than a silence. The claim's own second
// half is Medium — which directories a given launch actually holds was not
// observed at runtime — so what is decoded is the mechanism, not the population.
//
// An error from either call drops that candidate rather than failing the run.
// Nothing here is required: the whole function only ever supplies a fallback, and
// a fallback that refuses to be computed would turn a missing root into a
// different error than the one the caller is about to print.
func processDirs(executable string) []string {
	dirs := make([]string, 0, 2)
	if wd, err := os.Getwd(); err == nil {
		dirs = append(dirs, wd)
	}
	if executable != "" {
		dirs = append(dirs, filepath.Dir(executable))
	}
	return dirs
}

// run is the whole command, with its inputs and outputs injected so that both
// the exit status and the environment fallback are testable without mutating the
// process.
//
// Every startup failure is reported on the error stream and exits non-zero, in
// the windowed and the headless mode alike. That includes an install missing an
// archive the menu itself never reads: the application should say so before the
// user picks a map, not after.
func run(args []string, getenv func(string) string, stdout, stderr io.Writer) int {
	executable, _ := os.Executable()
	userConfigDir, _ := os.UserConfigDir()
	return runWithProfilePaths(args, getenv, stdout, stderr, executable, userConfigDir)
}

func runWithProfilePaths(args []string, getenv func(string) string, stdout, stderr io.Writer, executable, userConfigDir string) int {
	o, err := parse(args)
	if err != nil {
		fmt.Fprintf(stderr, "againrom: %v\n%s\n", err, usage)
		return 2
	}
	if o.version {
		fmt.Fprintln(stdout, versionLine())
		return 0
	}
	mods, err := resolveMods(o, executable)
	if err != nil {
		fmt.Fprintln(stderr, "againrom:", err)
		return 2
	}

	// THE SCENARIO IS READ BEFORE THE ASSET ROOT IS REQUIRED. A synthetic
	// scenario states its own world and reads no game file at all, so demanding
	// a root for it would make the one mode that needs no install the one mode
	// that refuses to run without one.
	var scenario game.HeadlessScenario
	if o.headless != "" {
		scenario, err = game.ReadHeadlessScenario(o.headless)
		if err != nil {
			fmt.Fprintln(stderr, "againrom: headless scenario:", err)
			return 2
		}
		// A scenario has no sound device adapter. This is the same supported
		// silent state as -sound=false, selected before NewFrontEnd opens audio.
		o.sound = false
		if scenario.AssetSource() == game.AssetsSynthetic {
			if err := runSyntheticStage(scenario, stdout, stderr); err != nil {
				fmt.Fprintln(stderr, "againrom: headless:", err)
				return 1
			}
			return 0
		}
	}

	// Flag over environment over unset, decided in the one place that already
	// owns that precedence.
	root := game.ResolveAssetRoot(o.assets, getenv("AGAINROM_ASSETS"))

	// A COPY OF THIS BINARY SITTING IN THE GAME FOLDER CONFIGURES ITSELF.
	// Discovery is the LAST step of the precedence, so -assets and
	// AGAINROM_ASSETS both still win and no existing invocation changes.
	//
	// It reports the root it picked. A configured root is already printed in the
	// launch line below, but the mission stage returns before that line and a
	// discovered root is the one case where nobody typed the path — a run that
	// silently chose between two installs on the same machine would be
	// unattributable.
	if root == "" {
		if root = game.DiscoverAssetRoot(processDirs(executable)...); root != "" {
			fmt.Fprintf(stderr, "againrom: no -assets and no AGAINROM_ASSETS; using the install found at %s\n", root)
		}
	}
	if root == "" {
		fmt.Fprintf(stderr, "againrom: no asset root configured; pass -assets or set AGAINROM_ASSETS, or run this binary from inside the game folder\n%s\n", usage)
		return 2
	}

	// THE BASE IS DETECTED BEFORE ANYTHING IS BUILT FROM THE INSTALL. -base is a
	// demand on the result. A root that merely lacks archives keeps the message
	// the front end gives for it; a known build missing a file of its profile is
	// refused here, naming the file.
	match, baseErr := game.DetectBase(root)
	var partial *base.PartialError
	switch {
	case baseErr == nil:
		if err := base.Require(o.base, match); err != nil {
			fmt.Fprintln(stderr, "againrom:", err)
			return 2
		}
	case o.base != "" || errors.As(baseErr, &partial):
		fmt.Fprintln(stderr, "againrom:", baseErr)
		return 2
	}

	// THE MODS RUN BEFORE ANYTHING IS BUILT FROM THE INSTALL: a script error, a
	// refused setting or a refused order stops the launch here, naming the mod.
	var modRun modrt.Result
	if len(mods) > 0 {
		if o.headless != "" && scenario.StageName() == game.StageMission {
			fmt.Fprintln(stderr, "againrom: -mods does not apply to a mission-stage scenario")
			return 2
		}
		if modRun, err = activateMods(o, mods, root); err != nil {
			fmt.Fprintln(stderr, "againrom:", err)
			return 2
		}
	}

	// THE MISSION STAGE RETURNS BEFORE THE FRONT END IS BUILT. Such a scenario
	// drives one mission's simulation and reaches no menu, no font, no sprite
	// sheet and no save directory, so building the whole front end for it would
	// load assets nothing in the run reads and would make a play scenario fail
	// on an install whose menu graphics are the only thing wrong with it.
	if o.headless != "" && scenario.StageName() == game.StageMission {
		if err := runMissionStage(root, scenario, stdout, stderr); err != nil {
			fmt.Fprintln(stderr, "againrom: headless:", err)
			return 1
		}
		return 0
	}

	profileOptions := o
	if o.headless != "" && scenario.Saves != "" {
		profileOptions.saves = scenario.Saves
	}
	var profile game.RuntimeProfile
	orig := game.OriginalStore{Dir: root}
	if profileOptions.saves != "" || !o.check && !o.mediaCheck && o.cutsceneCheck == "" {
		profile, orig, err = loadSources(root, profileOptions, executable, userConfigDir)
		if err != nil {
			fmt.Fprintln(stderr, "againrom: profile unavailable:", err)
		}
	}
	front, err := frontEnd(root, o, profile.Options)
	if err != nil {
		fmt.Fprintln(stderr, "againrom:", err)
		return 1
	}
	if len(mods) > 0 {
		if err := front.SetMods(modRun.Rules, modRun.Set, o.modsAcceptUnmarked); err != nil {
			fmt.Fprintln(stderr, "againrom:", err)
			return 1
		}
		if err := front.SetModItems(modRun.Items); err != nil {
			fmt.Fprintln(stderr, "againrom:", err)
			return 2
		}
		if err := front.SetModCharacters(modRun.Characters); err != nil {
			fmt.Fprintln(stderr, "againrom:", err)
			return 2
		}
		if err := front.SetModCompanions(modRun.Companions); err != nil {
			fmt.Fprintln(stderr, "againrom:", err)
			return 2
		}
		if err := front.SetModSpells(modRun.Spells); err != nil {
			fmt.Fprintln(stderr, "againrom:", err)
			return 2
		}
	}
	if o.fidelityCheck != "" {
		fmt.Fprintf(stdout, "againrom: owner fidelity witness build=%s assets=%s\n", launchStamp(), root)
		if err := front.WitnessOwnerFidelity(root, o.fidelityCheck, stdout); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	}
	if o.soundCheck != "" {
		fmt.Fprintf(stdout, "againrom: sound options witness build=%s assets=%s\n", launchStamp(), root)
		if err := front.WitnessSoundOptions(root, o.soundCheck, stdout); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	}
	if o.mediaCheck {
		front.Cutscenes = front.OpenCutscenes(game.CutsceneArchive(o.fourX, o.eightX))
		fmt.Fprintf(stdout, "againrom: media witness build=%s assets=%s files=read-only\n", launchStamp(), root)
		if err := front.WitnessMedia(stdout); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	}
	if o.cutsceneCheck != "" {
		if executable == "" {
			fmt.Fprintln(stderr, "againrom: executable path unavailable")
			return 1
		}
		helper := filepath.Join(filepath.Dir(executable), "cutscenehelper.exe")
		front.Cutscenes = front.OpenCutscenes(game.CutsceneArchive(o.fourX, o.eightX))
		fmt.Fprintf(stdout, "againrom: cutscene witness build=%s helper=%s assets=%s\n", launchStamp(), helper, root)
		if err := front.WitnessCutscene(root, helper, o.cutsceneCheck, stdout); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	}

	if o.check {
		fmt.Fprintln(stdout, front.CheckLine())
		for _, l := range front.BaseLines() {
			fmt.Fprintln(stdout, l)
		}
		if len(mods) > 0 {
			for _, l := range modLines(modRun) {
				fmt.Fprintln(stdout, l)
			}
		}
		// THE CHECK MODE ACCEPTS -mission TOO and reports the same outcome without
		// opening a window. All three startup failures are settled before anything
		// a window touches, so the headless run and the windowed one agree about
		// whether a mission can be started.
		if !o.picker {
			line, err := front.MissionLine(o.startMission(front))
			if err != nil {
				fmt.Fprintln(stderr, "againrom:", err)
				return 1
			}
			fmt.Fprintln(stdout, line)
			// THE ADVANCE LINE COMES RIGHT AFTER THE MISSION LINE, on the same
			// "-check -mission N" run and about the SAME mission: MissionLine answers
			// what mission N itself is, this answers what winning it would do, and it
			// runs the real decision — starting mission N and, where a successor
			// comes back, starting that mission too — rather than reporting a guess
			// (front.AdvanceLine's own doc).
			advance, err := front.AdvanceLine(o.startMission(front))
			if err != nil {
				fmt.Fprintln(stderr, "againrom:", err)
				return 1
			}
			fmt.Fprintln(stdout, advance)
			if !front.Base().Profile.Limits.NoCharacterGeneration {
				fmt.Fprintln(stdout, chargenSummary(front))
			}
		}
		return 0
	}

	if o.headless != "" {
		front.SetDeterministicFrames(true)
		app := front.App("againrom-headless")
		defer app.StopAudio()
		defer app.FlushBackground()
		if scenario.OriginalSaves != "" {
			orig = game.OriginalStore{Dir: scenario.OriginalSaves}
		}
		if profile.Saves.Dir != "" {
			fmt.Fprintln(stderr, "againrom: headless saves in", profile.Saves.Dir)
		}
		front.ConfigureSaveSeams(app, profile.Saves, orig, nil)
		if err := game.RunHeadlessScenario(front, app, scenario, stdout, stderr); err != nil {
			fmt.Fprintln(stderr, "againrom: headless:", err)
			return 1
		}
		return 0
	}

	// A normal GUI run states the exact checkout and build revision before the
	// window opens, and repeats their compact form in the OS window title. This
	// is intentionally after the headless/check exits: their output formats are
	// already machine-readable contracts, while this line diagnoses which live
	// executable the owner is actually accepting.
	for _, l := range front.BaseLines() {
		fmt.Fprintln(stderr, l)
	}
	source, stamp := launchSource(), launchStamp()
	fmt.Fprintf(stderr, "againrom: launch source=%s build=%s assets=%s profile=%s\n", source, stamp, root, profile.Directory)
	if len(mods) > 0 {
		fmt.Fprintf(stderr, "againrom: %d mod(s) active, mod-set %s\n", len(modRun.Ordered), modRun.Set.Digest())
	}
	if o.movies {
		front.Cutscenes = front.OpenCutscenes(game.CutsceneArchive(o.fourX, o.eightX))
	}
	app := front.App(fmt.Sprintf("againrom [%s %s]", filepath.Base(source), stamp))
	app.SetMenuLabel(menuLabel())
	if err := app.SetModScreens(game.ModScreens(modRun.Screens)); err != nil {
		fmt.Fprintln(stderr, "againrom:", err)
		return 2
	}
	if profile.Saves.Dir != "" {
		fmt.Fprintln(stderr, "againrom: saves in", profile.Saves.Dir)
	}
	front.ConfigureSaveSeams(app, profile.Saves, orig, nil)
	// NEW GAME IS THE GENERATION DOOR FOR MISSION N, NOT NORMAL STARTUP (owner
	// reversal; docs/DIVERGENCES.md DIV-523 carries the ruling this replaces).
	// The window always opens at the main menu now — app.Run below draws
	// ScreenMenu's own first frame — and what pressing NEW GAME there does is
	// decided once, here, before that ever happens. -picker alone skips this
	// block and leaves NEW GAME at its own zero value, the former map-picker
	// route front.App already wired above through SetChargenGate.
	//
	// -mission N, GIVEN EXPLICITLY (o.missionSet, not o.mission's value: the
	// flag's value alone does not say it was given, so it cannot tell a bare
	// launch from "-mission 10"), is the one case that still bypasses the
	// menu — a direct entry for development, arming generation before the
	// first window frame exactly as normal startup always did before the
	// reversal. Left implicit, the same setup instead becomes what NEW GAME
	// itself arms, through SetNewGameChargen, so the menu is what a bare
	// launch actually shows.
	//
	// THIS DOOR IS A FRESH START EITHER WAY, which is why both arms take the
	// same screen NEW GAME does: a process that begins inside a mission has no
	// won mission behind it and nothing to have carried out of one. The
	// campaign's OWN transitions, which do have something to carry, are
	// elsewhere entirely — front.App's gate does not see them and neither
	// does this statement.
	//
	// THE CONFIRM CALLBACK OPENS NOTHING ITSELF. It hands back a MapOpener,
	// exactly as MissionOpener does, and the same flow.enter every other entry
	// path uses is what eventually enters it — which is also why a mission whose
	// MAP will not decode is reported on the generation screen's own message
	// line rather than before the window opens. That is the price of putting a
	// screen in front of it, and it is the same price a picker row has always
	// paid for a map that will not decode.
	//
	// A NUMBER THAT NAMES NO MISSION AT ALL still never opens a window (0140).
	// game.MissionMap decides that WITHOUT reading an archive — it refuses a
	// number that is not positive and composes an address for every other — so
	// the half of the old refusal that can be settled before anything is opened
	// still is, and only the half that needs the map itself moved onto the
	// screen. It is asked here rather than inside begin because a command line
	// that could never have worked should not first cost the player a spread
	// or, on the menu-first route, a click.
	if o.picker {
		app.SetNewGameDirect(nil)
	} else {
		if _, ok := game.MissionMap(o.startMission(front)); !ok {
			fmt.Fprintf(stderr, "againrom: %d names no mission\n%s\n", o.startMission(front), usage)
			return 2
		}
		if err := armNewGameDoor(app, front, o); err != nil {
			fmt.Fprintln(stderr, "againrom:", err)
			return 1
		}
	}

	installWindowIcon(root, stderr)
	if err := desktop.run(app); err != nil {
		fmt.Fprintln(stderr, "againrom:", err)
		return 1
	}
	return 0
}
