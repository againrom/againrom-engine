package game

import (
	"fmt"
	"path/filepath"
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

// gameOptionKeys are the profile keys of ui's option table, in GameOption
// order.
var gameOptionKeys = func() (keys [len(ui.GameOptionValues{})]string) {
	for _, o := range ui.GameOptions() {
		keys[o] = o.Key()
	}
	return keys
}()

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

func (f *FrontEnd) gameOptionValues(onMap bool) ui.GameOptionValues {
	values := ui.GameOptionValues{1, 1, 1, 1, f.wimpyMode, boolOption(f.showPathfinding)}
	f.graphicsValues(&values)
	values[ui.GameOptionAutoHealing] = 1
	column := f.optionColumn()
	values[ui.GameOptionAutoCast] = ui.GameOptionAutoCast.Default(column)
	if !onMap || f.live == nil {
		stored, present, _ := f.Options.gameOptions()
		for i, ok := range present {
			if ok && (i < int(ui.GameOptionSmoothing) || i == int(ui.GameOptionAutoHealing) ||
				i == int(ui.GameOptionAutoCast) && ui.GameOptionAutoCast.Offered(column, stored[i])) {
				values[i] = stored[i]
			}
		}
		return values
	}
	mw := f.live
	values[ui.GameOptionDayNight] = boolOption(mw.view.TimeFlow())
	values[ui.GameOptionHealth] = boolOption(mw.view.HealthBarsShown())
	damage, _ := mw.view.DamageNumerals()
	values[ui.GameOptionDamage] = boolOption(damage)
	values[ui.GameOptionFormation] = mw.formationSetting()
	values[ui.GameOptionPathfinding] = boolOption(mw.view.PathfindingShown())
	values[ui.GameOptionAutoHealing] = mw.autoHealingSetting()
	values[ui.GameOptionAutoCast] = mw.autoCastSetting(sim.SelfSlot).LevelNumber()
	return values
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

func (f *FrontEnd) setGameOption(onMap bool, o ui.GameOption, value int) error {
	if !o.Offered(f.optionColumn(), value) {
		return fmt.Errorf("invalid game option %d value %d", o, value)
	}
	if o >= ui.GameOptionSmoothing && o <= ui.GameOptionAnimation || o == ui.GameOptionLootDots {
		return f.setGraphicsOption(onMap, o, value)
	}
	if o == ui.GameOptionAutoHealing && onMap && f.live != nil {
		if err := f.live.checkAutoHealing(value); err != nil {
			return err
		}
	}
	if f.Options.Path != "" {
		if err := f.Options.setGameOption(o, value); err != nil {
			return err
		}
	}
	if o == ui.GameOptionPathfinding {
		f.showPathfinding = value != 0
		if onMap && f.live != nil {
			f.live.view.SetPathfinding(f.showPathfinding)
		}
		return nil
	}
	if onMap && f.live != nil {
		f.ensureGameOptionApplication(f.live)
	}
	if o == ui.GameOptionRetreat {
		f.wimpyMode = value
	}
	if onMap && f.live != nil {
		f.live.applyGameOption(o, value)
	}
	return nil
}

func (mw *mapWorld) applyGameOption(o ui.GameOption, value int) {
	on := value != 0
	switch o {
	case ui.GameOptionDayNight:
		mw.view.SetTimeFlow(on)
	case ui.GameOptionHealth:
		if mw.view.HealthBarsShown() != on {
			mw.view.ToggleShowHealth()
		}
	case ui.GameOptionDamage:
		if current, _ := mw.view.DamageNumerals(); current != on {
			mw.view.ToggleDamageNumerals()
		}
	case ui.GameOptionFormation:
		mw.enqueueFormation(int32(value))
	case ui.GameOptionRetreat:
		mw.enqueueRetreat(value)
	case ui.GameOptionAutoHealing:
		mw.pending = append(mw.pending, sim.SetPlayerParameter(sim.SelfSlot, sim.PlayerParameterAutoHealing, int32(value)))
	case ui.GameOptionAutoCast:
		// The level the world already reads queues nothing.
		if mw.autoCastSetting(sim.SelfSlot).LevelNumber() != value {
			mw.setAutoCastLevel(sim.AutoCastLevel(value))
		}
	case ui.GameOptionPathfinding:
		mw.view.SetPathfinding(on)
	}
}

// Only fresh map doors call this. Restore/import must retain both application
// flags and queued commands from their checkpoint, regardless of the profile.
func (p *PersistenceContext) applyFreshGameOptions(mw *mapWorld, incoming []mapload.PartyMember) error {
	values, present, err := p.Options.gameOptions()
	if err != nil {
		return fmt.Errorf("read game options: %w", err)
	}
	// LOAD can have restored a different session label. Resolve the fresh
	// profile before creating the application record: NewGameOpener adopts
	// that record at commit, so a stale Wimpy value here would win again.
	p.wimpyMode = wimpyModeOff
	if present[ui.GameOptionRetreat] {
		p.wimpyMode = values[ui.GameOptionRetreat]
	}
	p.ensureGameOptionApplication(mw)
	// Opening another mission is not starting another campaign. A carried
	// source party already has its Player percentage and stored mana floors;
	// an absent profile key must not replace them with the new-game default.
	var inherited uint32
	hasInherited := false
	if !present[ui.GameOptionAutoHealing] {
		inherited, hasInherited, err = carriedAutoHealing(incoming)
		if err != nil {
			return err
		}
	}
	if hasInherited {
		mw.world.ImportAutoHealing(sim.SelfSlot, inherited)
	} else {
		healing := 1
		if present[ui.GameOptionAutoHealing] {
			healing = values[ui.GameOptionAutoHealing]
		}
		if !mw.world.SetAutoHealing(sim.SelfSlot, int32(healing)) {
			return fmt.Errorf("fresh party autohealing could not be initialized")
		}
	}
	for i, ok := range present {
		if ok && i != int(ui.GameOptionAutoHealing) {
			mw.applyGameOption(ui.GameOption(i), values[i])
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
	value := 2
	if mode == 0 {
		value = 0
	} else if mode == 2 {
		value = 1
	}
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

func (f *FrontEnd) wireGameOptions(a *ui.App) {
	c := ui.GameOptionControls{Column: f.optionColumn(), Read: f.gameOptionValues, Write: f.setGameOption,
		Words: ui.GameOptionWords{Title: "Game Options", Tips: "Tips", OK: "OK", Cancel: "Cancel", Speed: "Game Speed",
			Labels: [len(ui.GameOptionValues{})]string{"Day/Night changes", "Show Health", "Show flying damage", "Formation Mode", "Retreat Mode", "Show pathfinding", "Smoothing", "", "", "", "AutoHealing", "Loot dots on minimap"}}}
	c.Words.Choices[ui.GameOptionFormation] = [4]string{"Off", "Auto", "On"}
	c.Words.Choices[ui.GameOptionRetreat] = [4]string{"Never", "Low Health", "Medium Health"}
	c.Words.Choices[ui.GameOptionAutoHealing] = [4]string{"No", "Standard", "Often"}
	if f.Archives != nil {
		t := LoadTextTable(f.Archives.Containers, DialogsTextPath, f.textCode())
		for i, p := range map[int]*string{150: &c.Words.Title, 156: &c.Words.Tips, 0: &c.Words.OK, 1: &c.Words.Cancel, 50: &c.Words.Speed,
			52: &c.Words.Labels[ui.GameOptionSmoothing], 54: &c.Words.Labels[0], 56: &c.Words.Labels[1], 78: &c.Words.Labels[2],
			58: &c.Words.Labels[3], 63: &c.Words.Labels[4],
			60: &c.Words.Choices[ui.GameOptionFormation][0], 61: &c.Words.Choices[ui.GameOptionFormation][1], 62: &c.Words.Choices[ui.GameOptionFormation][2],
			65: &c.Words.Choices[ui.GameOptionRetreat][0], 66: &c.Words.Choices[ui.GameOptionRetreat][1], 67: &c.Words.Choices[ui.GameOptionRetreat][2],
			159: &c.Words.Labels[ui.GameOptionAutoHealing], 160: &c.Words.Choices[ui.GameOptionAutoHealing][0], 161: &c.Words.Choices[ui.GameOptionAutoHealing][1], 162: &c.Words.Choices[ui.GameOptionAutoHealing][2]} {
			if s, ok := t.At(i); ok {
				*p = s
			}
		}
		// The second game's dialogs lines name the automatic-casting row and
		// its three levels (R2-ASSET-085); the first game's root has none and
		// the page supplies the engine's words (DIV-3011).
		if lines := filesOf(f.Archives.Game()).autoCastCaptions; len(lines) == 4 {
			for i, p := range [...]*string{&c.Words.Labels[ui.GameOptionAutoCast], &c.Words.Choices[ui.GameOptionAutoCast][1],
				&c.Words.Choices[ui.GameOptionAutoCast][2], &c.Words.Choices[ui.GameOptionAutoCast][3]} {
				if s, ok := t.At(lines[i]); ok {
					*p = s
				}
			}
			// The label line ends in its own colon; the row adds one.
			c.Words.Labels[ui.GameOptionAutoCast] = strings.TrimSuffix(strings.TrimSpace(c.Words.Labels[ui.GameOptionAutoCast]), ":")
		}
		if patch, err := OpenContainers(filepath.Join(f.Archives.Root, patchArchive)); err == nil {
			table := LoadTextTable(patch, patchTextPath, f.textCode())
			for i, o := range [...]ui.GameOption{ui.GameOptionShadows, ui.GameOptionLighting, ui.GameOptionAnimation} {
				if s, ok := table.At(patchGraphicsRow + i); ok {
					c.Words.Labels[o] = s
				}
			}
		}
		set, _ := loadScreen(f.Archives.artSource(), ScreensDescription(f.profile()), "game-options")
		for i := range c.Checks {
			c.Checks[i] = set.Picture(fmt.Sprintf("check/%d", i))
			c.Radios[i] = set.Picture(fmt.Sprintf("radio/%d", i))
		}
	}
	a.SetGameOptionControls(c)
}
