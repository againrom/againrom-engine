package game

import (
	"fmt"
	"strconv"
	"strings"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// The install's patch archive holds the Shadows, Dynamic lighting and Object
// animations captions in three consecutive rows (TEXT-099).
const (
	patchArchive     = "patch.res"
	patchTextPath    = "patch/patch.txt"
	patchGraphicsRow = 52
)

// newGameAutoHealing is the autohealing choice a new party starts at,
// Standard.
const newGameAutoHealing = 1

// gameOptionKeys are the profile keys of ui's option table, in GameOption
// order.
var gameOptionKeys = func() (keys [len(ui.GameOptionValues{})]string) {
	for _, o := range ui.GameOptions() {
		keys[o] = o.Key()
	}
	return keys
}()

// gameOptionRow is one Game Options row's game behaviour. ui's
// gameOptionTable holds the same row's page side and profile key; this table
// holds what the game does with the value. A nil function or field is the
// row's explicit absence of that behaviour. Every function the options use
// walks these rows; none decides by the option's identity.
type gameOptionRow struct {
	// initial is the value a new profile starts at, per Game Options column.
	initial func(column string) int
	// graphics is the renderer flag a graphics option is held, stored and
	// drawn as. A graphics row is written with its group (setGraphicsOption)
	// and the live map takes the whole group.
	graphics *graphicsFlag
	// session reads the copy of the value the session holds; remember writes
	// it on a set and at LoadOptions. A row with neither shows its initial
	// value until the profile or the live map names one.
	session  func(f *FrontEnd) int
	remember func(f *FrontEnd, value int)
	// adopt reports whether the page shows a stored value off the map.
	adopt func(column string, value int) bool
	// read is the live map's value, commands waiting for the next tick
	// included.
	read func(mw *mapWorld) int
	// check refuses a value the live map cannot take, before the store.
	check func(mw *mapWorld, value int) error
	// recorded: a live change opens the map's application record first.
	recorded bool
	// apply sets the value on the live map.
	apply func(mw *mapWorld, value int)
	// enter runs at a fresh map door before the application record exists,
	// with the stored value and whether the profile holds one.
	enter func(p *PersistenceContext, value int, present bool)
	// open replaces apply at a fresh map door.
	open func(mw *mapWorld, incoming []mapload.PartyMember, value int, present bool) error
	// queue is the Player parameter a waiting command carries; a SAVE keeps
	// such a command as a pending option.
	queue *gameOptionQueue
	// captions are the row's words on the page.
	captions gameOptionCaptions
}

// gameOptionQueue is a queued option's Player parameter and the option value
// its command value holds, if it holds one.
type gameOptionQueue struct {
	parameter sim.PlayerParameter
	value     func(y int32) (int, bool)
}

// gameOptionCaptions are a row's English words and the install text lines
// that replace them. lines are the dialogs lines of the label and then of
// each choice, negative for none; perGame names them from the game's files
// instead. patch is the patch text row of the label, 0 for none. trimColon
// drops the colon a label line carries; the page adds its own.
type gameOptionCaptions struct {
	label     string
	choices   []string
	lines     []int
	perGame   func(files *gameFiles) []int
	patch     int
	trimColon bool
}

// graphicsFlag is a graphics option's renderer flag; hides marks a flag that
// is true when the option is 0.
type graphicsFlag struct {
	field func(g *ui.GraphicsOptions) *bool
	hides bool
}

func (g *graphicsFlag) value(o ui.GraphicsOptions) int { return boolOption(*g.field(&o) != g.hides) }

func (g *graphicsFlag) set(o *ui.GraphicsOptions, value int) { *g.field(o) = (value != 0) != g.hides }

// fixedValue is an initial value the same in every column.
func fixedValue(v int) func(string) int { return func(string) int { return v } }

// adoptAny adopts every stored value the store accepts.
func adoptAny(string, int) bool { return true }

// queuedParameter is a queued option whose command value is the option value.
func queuedParameter(p sim.PlayerParameter) *gameOptionQueue {
	return &gameOptionQueue{p, func(y int32) (int, bool) { return int(y), true }}
}

// gameOptionRows holds every option's game behaviour in GameOption order.
var gameOptionRows = [len(ui.GameOptionValues{})]gameOptionRow{
	ui.GameOptionDayNight: {initial: fixedValue(1), adopt: adoptAny, recorded: true,
		read:     func(mw *mapWorld) int { return boolOption(mw.view.TimeFlow()) },
		apply:    func(mw *mapWorld, v int) { mw.view.SetTimeFlow(v != 0) },
		captions: gameOptionCaptions{label: "Day/Night changes", lines: []int{54}}},
	ui.GameOptionHealth: {initial: fixedValue(1), adopt: adoptAny, recorded: true,
		read: func(mw *mapWorld) int { return boolOption(mw.view.HealthBarsShown()) },
		apply: func(mw *mapWorld, v int) {
			if mw.view.HealthBarsShown() != (v != 0) {
				mw.view.ToggleShowHealth()
			}
		},
		captions: gameOptionCaptions{label: "Show Health", lines: []int{56}}},
	ui.GameOptionDamage: {initial: fixedValue(1), adopt: adoptAny, recorded: true,
		read: func(mw *mapWorld) int {
			on, _ := mw.view.DamageNumerals()
			return boolOption(on)
		},
		apply: func(mw *mapWorld, v int) {
			if on, _ := mw.view.DamageNumerals(); on != (v != 0) {
				mw.view.ToggleDamageNumerals()
			}
		},
		captions: gameOptionCaptions{label: "Show flying damage", lines: []int{78}}},
	ui.GameOptionFormation: {initial: fixedValue(1), adopt: adoptAny, recorded: true,
		read:  func(mw *mapWorld) int { return mw.formationSetting() },
		apply: func(mw *mapWorld, v int) { mw.enqueueFormation(int32(v)) },
		queue: queuedParameter(sim.PlayerParameterFormation),
		captions: gameOptionCaptions{label: "Formation Mode", choices: []string{"Off", "Auto", "On"},
			lines: []int{58, 60, 61, 62}}},
	// The retreat label is the session's: the live map's Withdraw values
	// stay the save's until an ordinary parameter-3 command runs.
	ui.GameOptionRetreat: {initial: fixedValue(wimpyModeOff), adopt: adoptAny, recorded: true,
		session:  func(f *FrontEnd) int { return f.wimpyMode },
		remember: func(f *FrontEnd, v int) { f.wimpyMode = v },
		apply:    func(mw *mapWorld, v int) { mw.enqueueRetreat(v) },
		// LOAD can have restored a different session label. The fresh
		// profile's label is resolved before the application record exists:
		// NewGameOpener adopts that record at commit, so a stale Wimpy value
		// here would win again.
		enter: func(p *PersistenceContext, v int, present bool) {
			p.wimpyMode = wimpyModeOff
			if present {
				p.wimpyMode = v
			}
		},
		queue: queuedParameter(sim.PlayerParameterRetreat),
		captions: gameOptionCaptions{label: "Retreat Mode", choices: []string{"Never", "Low Health", "Medium Health"},
			lines: []int{63, 65, 66, 67}}},
	// Pathfinding is a display of the session: it is no part of the map's
	// application record.
	ui.GameOptionPathfinding: {initial: fixedValue(0), adopt: adoptAny,
		session:  func(f *FrontEnd) int { return boolOption(f.showPathfinding) },
		remember: func(f *FrontEnd, v int) { f.showPathfinding = v != 0 },
		read:     func(mw *mapWorld) int { return boolOption(mw.view.PathfindingShown()) },
		apply:    func(mw *mapWorld, v int) { mw.view.SetPathfinding(v != 0) },
		captions: gameOptionCaptions{label: "Show pathfinding"}},
	ui.GameOptionSmoothing: {initial: fixedValue(1),
		graphics: &graphicsFlag{field: func(g *ui.GraphicsOptions) *bool { return &g.Smoothing }},
		captions: gameOptionCaptions{label: "Smoothing", lines: []int{52}}},
	ui.GameOptionShadows: {initial: fixedValue(1),
		graphics: &graphicsFlag{field: func(g *ui.GraphicsOptions) *bool { return &g.HideShadows }, hides: true},
		captions: gameOptionCaptions{patch: patchGraphicsRow}},
	ui.GameOptionLighting: {initial: fixedValue(1),
		graphics: &graphicsFlag{field: func(g *ui.GraphicsOptions) *bool { return &g.DisableLighting }, hides: true},
		captions: gameOptionCaptions{patch: patchGraphicsRow + 1}},
	ui.GameOptionAnimation: {initial: fixedValue(1),
		graphics: &graphicsFlag{field: func(g *ui.GraphicsOptions) *bool { return &g.StaticObjects }, hides: true},
		captions: gameOptionCaptions{patch: patchGraphicsRow + 2}},
	ui.GameOptionAutoHealing: {initial: fixedValue(newGameAutoHealing), adopt: adoptAny, recorded: true,
		read:  func(mw *mapWorld) int { return mw.autoHealingSetting() },
		check: func(mw *mapWorld, v int) error { return mw.checkAutoHealing(v) },
		apply: func(mw *mapWorld, v int) {
			mw.pending = append(mw.pending, sim.SetPlayerParameter(sim.SelfSlot, sim.PlayerParameterAutoHealing, int32(v)))
		},
		open:  openAutoHealing,
		queue: queuedParameter(sim.PlayerParameterAutoHealing),
		captions: gameOptionCaptions{label: "AutoHealing", choices: []string{"No", "Standard", "Often"},
			lines: []int{159, 160, 161, 162}}},
	// Loot dots stay off until the player asks: the original's minimap marks
	// no sack (MISSION-063, DIV-2954).
	ui.GameOptionLootDots: {initial: fixedValue(0),
		graphics: &graphicsFlag{field: func(g *ui.GraphicsOptions) *bool { return &g.HideLootDots }, hides: true},
		captions: gameOptionCaptions{label: "Loot dots on minimap"}},
	// The automatic-casting level; the edition's column offers the values
	// and names the new-game level.
	ui.GameOptionAutoCast: {initial: ui.GameOptionAutoCast.Default, adopt: ui.GameOptionAutoCast.Offered, recorded: true,
		read: func(mw *mapWorld) int { return mw.autoCastSetting(sim.SelfSlot).LevelNumber() },
		apply: func(mw *mapWorld, v int) {
			// The level the world already reads queues nothing.
			if mw.autoCastSetting(sim.SelfSlot).LevelNumber() != v {
				mw.setAutoCastLevel(sim.AutoCastLevel(v))
			}
		},
		queue: &gameOptionQueue{sim.PlayerParameterAutoCast, func(y int32) (int, bool) {
			if y < 0 || y > 0xff {
				return 0, false
			}
			return sim.AutoCastSetting(y).LevelNumber(), true
		}},
		// The second game's dialogs lines name the row and its three levels
		// (R2-ASSET-085); the first game's root has none and the page
		// supplies the engine's words (DIV-3011).
		captions: gameOptionCaptions{trimColon: true, perGame: func(files *gameFiles) []int {
			if c := files.autoCastCaptions; len(c) == 4 {
				return []int{c[0], -1, c[1], c[2], c[3]}
			}
			return nil
		}}},
}

// Opening another mission is not starting another campaign. A carried source
// party already has its Player percentage and stored mana floors; an absent
// profile key must not replace them with the new-game default.
func openAutoHealing(mw *mapWorld, incoming []mapload.PartyMember, value int, present bool) error {
	if !present {
		inherited, carried, err := carriedAutoHealing(incoming)
		if err != nil {
			return err
		}
		if carried {
			mw.world.ImportAutoHealing(sim.SelfSlot, inherited)
			return nil
		}
		value = newGameAutoHealing
	}
	if !mw.world.SetAutoHealing(sim.SelfSlot, int32(value)) {
		return fmt.Errorf("fresh party autohealing could not be initialized")
	}
	return nil
}

// Missing or malformed values are unspecified: an old profile must not reset
// the map's existing defaults merely because it predates a new control.
func (s OptionsStore) gameOptions() (ui.GameOptionValues, [len(gameOptionKeys)]bool, error) {
	var values ui.GameOptionValues
	var present [len(gameOptionKeys)]bool
	m, err := s.readAll()
	if err != nil {
		return values, present, err
	}
	for i, key := range gameOptionKeys {
		v, err := strconv.Atoi(m[key])
		if err == nil && v >= 0 && v < ui.GameOption(i).Choices() {
			values[i], present[i] = v, true
		}
	}
	return values, present, nil
}

func (s OptionsStore) setGameOption(o ui.GameOption, value int) error {
	if value < 0 || value >= o.Choices() {
		return fmt.Errorf("invalid game option %d value %d", o, value)
	}
	m, err := s.readAll()
	if err != nil {
		return err
	}
	m[gameOptionKeys[o]] = strconv.Itoa(value)
	return s.writeAll(m)
}

// sessionValue is the row's value without the profile or the live map.
func (r *gameOptionRow) sessionValue(f *FrontEnd, column string) int {
	switch {
	case r.graphics != nil:
		return r.graphics.value(f.graphics)
	case r.session != nil:
		return r.session(f)
	}
	return r.initial(column)
}

func (f *FrontEnd) gameOptionValues(onMap bool) ui.GameOptionValues {
	column := f.optionColumn()
	var values ui.GameOptionValues
	for o := range gameOptionRows {
		values[o] = gameOptionRows[o].sessionValue(f, column)
	}
	if !onMap || f.live == nil {
		stored, present, _ := f.Options.gameOptions()
		for o, r := range gameOptionRows {
			if present[o] && r.adopt != nil && r.adopt(column, stored[o]) {
				values[o] = stored[o]
			}
		}
		return values
	}
	for o, r := range gameOptionRows {
		if r.read != nil {
			values[o] = r.read(f.live)
		}
	}
	return values
}

// loadGameOptions is LoadOptions' half of the rows: each graphics flag and
// each session copy from the profile, or the row's initial value.
func (f *FrontEnd) loadGameOptions() {
	column := f.optionColumn()
	stored, present, _ := f.Options.gameOptions()
	var graphics ui.GameOptionValues
	for o, r := range gameOptionRows {
		v := r.initial(column)
		if present[o] {
			v = stored[o]
		}
		graphics[o] = v
		if r.remember != nil {
			r.remember(f, v)
		}
	}
	f.graphics = graphicsValues(graphics.Constrained())
}

// optionColumn is the edition's Game Options column: the offers and the
// new-game values of the table's stepping rows.
func (f *FrontEnd) optionColumn() string { return f.Base().Profile.Edition().Options }

func boolOption(on bool) int {
	if on {
		return 1
	}
	return 0
}

// setGameOption writes one option: the live map's refusal, the store, the
// application record, the session copy and the live map, in that order.
func (f *FrontEnd) setGameOption(onMap bool, o ui.GameOption, value int) error {
	if !o.Offered(f.optionColumn(), value) {
		return fmt.Errorf("invalid game option %d value %d", o, value)
	}
	r := &gameOptionRows[o]
	if r.graphics != nil {
		return f.setGraphicsOption(onMap, o, value)
	}
	live := f.live
	if !onMap {
		live = nil
	}
	if live != nil && r.check != nil {
		if err := r.check(live, value); err != nil {
			return err
		}
	}
	if f.Options.Path != "" {
		if err := f.Options.setGameOption(o, value); err != nil {
			return err
		}
	}
	if live != nil && r.recorded {
		f.ensureGameOptionApplication(live)
	}
	if r.remember != nil {
		r.remember(f, value)
	}
	if live != nil && r.apply != nil {
		r.apply(live, value)
	}
	return nil
}

func (mw *mapWorld) applyGameOption(o ui.GameOption, value int) {
	if int(o) < len(gameOptionRows) && gameOptionRows[o].apply != nil {
		gameOptionRows[o].apply(mw, value)
	}
}

// Only fresh map doors call this. Restore/import must retain both application
// flags and queued commands from their checkpoint, regardless of the profile.
func (p *PersistenceContext) applyFreshGameOptions(mw *mapWorld, incoming []mapload.PartyMember) error {
	values, present, err := p.Options.gameOptions()
	if err != nil {
		return fmt.Errorf("read game options: %w", err)
	}
	for o, r := range gameOptionRows {
		if r.enter != nil {
			r.enter(p, values[o], present[o])
		}
	}
	p.ensureGameOptionApplication(mw)
	for o, r := range gameOptionRows {
		if r.open != nil {
			if err := r.open(mw, incoming, values[o], present[o]); err != nil {
				return err
			}
		}
	}
	for o, r := range gameOptionRows {
		if present[o] && r.open == nil && r.apply != nil {
			r.apply(mw, values[o])
		}
	}
	return nil
}

func (p *PersistenceContext) ensureGameOptionApplication(mw *mapWorld) {
	if mw.applicationState != nil {
		return
	}
	view := localOnlyApplicationView(mw.view.SaveApplication())
	mw.applicationState = &SnapshotApplicationState{LocalOnly: true, Version: applicationStateVersion,
		View: view, Baseline: view, WimpyBaseline: p.wimpyMode,
		Original: OriginalStateData{Wimpy: int32(p.wimpyMode), Formation: int32(mw.formationSetting())}}
}

func (mw *mapWorld) formationSetting() int {
	mode, _ := mw.world.CommandFormationMode(sim.SelfSlot)
	value := int(sim.FormationSettingOfMode(mode))
	for _, c := range mw.pending {
		if c.Kind == sim.KindPlayerParameter && c.Player == sim.SelfSlot && sim.PlayerParameter(c.X) == sim.PlayerParameterFormation {
			value = int(c.Y)
			if value < 0 || value > 2 {
				value = 1
			}
		}
	}
	return value
}

// gameOptionWords are the page's captions: each row's English words, then
// the install's dialogs and patch lines the rows name.
func (f *FrontEnd) gameOptionWords() ui.GameOptionWords {
	w := ui.GameOptionWords{Title: "Game Options", Tips: "Tips", OK: "OK", Cancel: "Cancel", Speed: "Game Speed"}
	for o, r := range gameOptionRows {
		w.Labels[o] = r.captions.label
		copy(w.Choices[o][:], r.captions.choices)
	}
	if f.Archives == nil {
		return w
	}
	t := LoadTextTable(f.Archives.Containers, DialogsTextPath, f.textCode())
	for i, p := range map[int]*string{150: &w.Title, 156: &w.Tips, 0: &w.OK, 1: &w.Cancel, 50: &w.Speed} {
		if s, ok := t.At(i); ok {
			*p = s
		}
	}
	files := filesOf(f.Archives.Game())
	patched := false
	for o, r := range gameOptionRows {
		c := r.captions
		lines := c.lines
		if c.perGame != nil {
			lines = c.perGame(files)
		}
		for i, line := range lines {
			s, ok := t.At(line)
			switch {
			case !ok:
			case i == 0:
				w.Labels[o] = s
			default:
				w.Choices[o][i-1] = s
			}
		}
		if c.trimColon && len(lines) != 0 {
			w.Labels[o] = strings.TrimSuffix(strings.TrimSpace(w.Labels[o]), ":")
		}
		patched = patched || c.patch != 0
	}
	if !patched {
		return w
	}
	if patch, err := openInstallContainers(f.Archives, patchArchive); err == nil {
		table := LoadTextTable(patch, patchTextPath, f.textCode())
		for o, r := range gameOptionRows {
			if r.captions.patch == 0 {
				continue
			}
			if s, ok := table.At(r.captions.patch); ok {
				w.Labels[o] = s
			}
		}
	}
	return w
}

func (f *FrontEnd) wireGameOptions(a *ui.App) {
	c := ui.GameOptionControls{Column: f.optionColumn(), Read: f.gameOptionValues, Write: f.setGameOption, Words: f.gameOptionWords()}
	if f.Archives != nil {
		set, _ := loadScreen(f.Archives.artSource(), ScreensDescription(f.profile()), "game-options")
		for i := range c.Checks {
			c.Checks[i] = set.Picture(fmt.Sprintf("check/%d", i))
			c.Radios[i] = set.Picture(fmt.Sprintf("radio/%d", i))
		}
	}
	a.SetGameOptionControls(c)
}
