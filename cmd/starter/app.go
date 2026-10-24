package main

import (
	"errors"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"

	basepkg "againrom/pkg/base"
	"againrom/pkg/game"
	"againrom/pkg/ini"
	"againrom/pkg/launch"
	"againrom/pkg/locale"
	"againrom/pkg/mod"
	"againrom/pkg/pathdialog"
	"againrom/pkg/ui"
)

// Window size in pixels. panelH is the height of the mod settings panel that
// sits between the mod list and the action buttons.
const (
	winW   = 760
	panelH = 140
	winH   = 620 + panelH
)

// Row capacities of the two scrolling lists and of the settings panel.
const (
	baseRows    = 5
	modRows     = 6
	settingRows = 5
)

// panelTop is the y of the settings panel's heading.
const panelTop = 398

type fieldID int

const (
	fNone fieldID = iota
	fAddBase
	fModsDir
	fVolume
	fSaves
	fMission
	fSkill
	fShopStockMultiplier
	fExtra
	fAgainrom
	fSetting0 // fSetting0..fSetting0+settingRows-1 edit the panel's integer settings
	_
	_
	_
	_
	fieldCount
)

type actionKind int

const (
	aNone actionKind = iota
	aSelectBase
	aToggleMod
	aAddBase
	aRemoveBase
	aRescan
	aSound
	aVideo
	aMovies
	aMarkers
	aChicken
	aPicker
	aCloseOnPlay
	aAcceptUnmarked
	aPlay
	aCheck
	aSave
	aFocus
	aSettingCycle // flip or step the panel's bool or choice setting, i is its row
	aSettingReset
	aMapEditor
	aChooseBase
	aSettingsOpen
	aSettingsClose
	aWindowed
)

// viewKind is which of the starter's two views the window shows.
type viewKind int

const (
	viewMain viewKind = iota
	viewSettings
)

type action struct {
	kind actionKind
	i    int // row index for list rows, fieldID for aFocus
}

type itemKind int

const (
	iText itemKind = iota
	iHeading
	iButton
	iToggle
	iField
	iBaseRow
	iModRow
)

type tone int

const (
	toneNormal tone = iota
	toneDim
	toneGood
	toneBad
)

// item is one thing the window shows. The same list drives drawing and hit
// testing, so what is drawn is what a click reaches.
type item struct {
	kind    itemKind
	rect    image.Rectangle
	text    string
	tone    tone
	checked bool // toggle, row mark
	focused bool // field
	act     action
}

// modRow is one line of the mod list.
type modRow struct {
	id      string
	label   string
	problem string // non-empty: cannot be ticked
	enabled bool
}

// deps are the effects the window cannot own: running programs, the clipboard
// and reading an install.
type deps struct {
	// run starts exe with args, waits for it and returns its combined output.
	run func(exe string, args []string) (string, error)
	// start launches exe with args and returns without waiting.
	start func(exe string, args []string) error
	paste func() string
	// inspect looks at a base root.
	inspect func(root string) game.InstallInfo
	// async runs f off the interface thread.
	async      func(f func())
	pathDialog pathdialog.Func
}

// app is the starter's state. It has no window code: Update and Draw in main.go
// feed it input and show the image render makes.
type app struct {
	deps deps

	selfPath string // this executable
	goos     string
	iniPath  string

	file     *ini.File
	problems []ini.Problem
	s        settings

	infos     map[string]game.InstallInfo // by base path
	scan      []mod.Entry
	scanErr   error
	selMod    string                   // mod whose settings the panel shows
	declared  map[string]declaredSetts // settings.toml of each scanned mod
	addText   string
	focus     fieldID
	view      viewKind
	scroll    [2]int // base list, mod list
	status    string
	statusBad bool
	output    []string

	starterVersion string
	gameVersion    string // what "againrom -version" printed, or "unknown"

	results             chan result
	folderDialogResults chan folderDialogResult
	dirty               bool
	closing             bool
}

type result struct {
	kind   string // "check", "version" or "editor"
	output string
	err    error
	launch []string // "editor": the program and arguments to start once the check passed
}

// newApp reads the settings file at iniPath (a missing file is the defaults)
// and looks at every base and the mods directory.
func newApp(iniPath, selfPath, goos, starterVersion string, d deps) *app {
	a := &app{
		deps: d, selfPath: selfPath, goos: goos, iniPath: iniPath,
		infos:          map[string]game.InstallInfo{},
		starterVersion: starterVersion,
		gameVersion:    versionUnknown,
		results:        make(chan result, 8),
		dirty:          true,
	}
	// The first problem is reported once the bases are inspected, so it is
	// worded in the language of the selected base.
	var early func()
	data, err := os.ReadFile(iniPath)
	switch {
	case err == nil:
	case errors.Is(err, os.ErrNotExist):
		data = nil
	default:
		early = func() { a.fail(cReadSettingsFailed, iniPath, err) }
	}
	a.file, a.problems = ini.Parse(data)
	a.s = loadSettings(a.file)
	if len(a.problems) > 0 {
		p := a.problems[0]
		early = func() { a.fail(cSettingsLine, iniPath, p, len(a.problems)) }
	}
	if early == nil {
		if err := probeWritable(filepath.Dir(iniPath)); err != nil {
			early = func() { a.fail(cFolderNotWritable, err) }
		}
	}
	for _, b := range a.s.Bases {
		a.inspectBase(b.Path)
	}
	if _, ok := a.s.basePath(a.s.LastBase); !ok && len(a.s.Bases) > 0 {
		a.s.LastBase = a.s.Bases[0].Key
	}
	if early != nil {
		early()
	}
	a.rescan()
	a.probeVersion()
	return a
}

func probeWritable(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".starter-write-test-*")
	if err != nil {
		return err
	}
	name := f.Name()
	f.Close()
	return os.Remove(name)
}

func (a *app) ok(id captionID, args ...any) {
	a.status, a.statusBad, a.dirty = a.t(id, args...), false, true
}

func (a *app) fail(id captionID, args ...any) {
	a.status, a.statusBad, a.dirty = a.t(id, args...), true, true
}

func (a *app) inspectBase(path string) {
	if _, ok := a.infos[path]; !ok {
		a.infos[path] = a.deps.inspect(path)
	}
}

func (a *app) selfDir() string { return filepath.Dir(a.selfPath) }

func (a *app) rescan() {
	a.scan, a.scanErr = mod.Scan(a.s.modsDir(a.selfDir()))
	a.declared = map[string]declaredSetts{}
	a.dirty = true
}

// declaredSetts is what reading a mod's settings.toml gave.
type declaredSetts struct {
	list []mod.Setting
	err  error
}

// settingsOf returns the settings the mod id declares, read once per scan.
func (a *app) settingsOf(id string) ([]mod.Setting, error) {
	if d, ok := a.declared[id]; ok {
		return d.list, d.err
	}
	var d declaredSetts
	found := false
	for _, e := range a.scan {
		if e.Folder == id && e.Err == nil {
			d.list, d.err = mod.LoadSettings(e.Dir)
			found = true
		}
	}
	if !found {
		d.err = captionErr(cModNoFolder, id)
	}
	a.declared[id] = d
	return d.list, d.err
}

// panelMod is the mod whose settings the panel shows: the selected one while it
// is listed, else the first enabled mod.
func (a *app) panelMod() (string, bool) {
	for _, r := range a.modRows() {
		if r.id == a.selMod {
			return r.id, true
		}
	}
	if len(a.s.Enabled) > 0 {
		return a.s.Enabled[0], true
	}
	return "", false
}

// lang is the language of the selected base, as a label language: the locale
// code of its language entry, else the reference language, including when the
// base is unknown.
func (a *app) lang() string {
	if root, ok := a.selectedRoot(); ok {
		if l, ok := locale.ByEntry(a.infos[root].Language); ok {
			return l.Code
		}
	}
	return locale.Fallback
}

// settingText is the text a setting shows: the stored value, else its default.
func (a *app) settingText(id string, st mod.Setting) string {
	if v, ok := a.s.modValue(id, st.Key); ok {
		return v
	}
	return st.Default.String()
}

// modSettings are the -mod-setting values for the enabled mods, in load order.
// A stored value for a key the mod no longer declares is left out. A value the
// mod's declaration refuses is an error naming the mod and the setting.
func (a *app) modSettings() ([]string, error) {
	var args []string
	for _, id := range a.s.Enabled {
		decl, err := a.settingsOf(id)
		if err != nil {
			return nil, captionErr(cModError, id, a.errText(err))
		}
		given := map[string]string{}
		for _, st := range decl {
			if v, ok := a.s.modValue(id, st.Key); ok && strings.TrimSpace(v) != "" {
				given[st.Key] = v
			}
		}
		if _, err := mod.ResolveSettings(id, decl, given); err != nil {
			return nil, err
		}
		for _, st := range decl {
			if v, ok := given[st.Key]; ok {
				args = append(args, mod.SettingFlag{Mod: id, Key: st.Key, Value: strings.TrimSpace(v)}.String())
			}
		}
	}
	return args, nil
}

func (a *app) selectedRoot() (string, bool) { return a.s.basePath(a.s.LastBase) }

// field returns the text a field edits.
func (a *app) field(id fieldID) *string {
	switch id {
	case fAddBase:
		return &a.addText
	case fModsDir:
		return &a.s.ModsDir
	case fAgainrom:
		return &a.s.Againrom
	}
	for _, v := range optionViews {
		if v.edit == id && id != fNone {
			return v.value(&a.s).(*string)
		}
	}
	if id >= fSetting0 && id < fSetting0+settingRows {
		return a.settingField(int(id - fSetting0))
	}
	return nil
}

// settingField returns the text of the panel's integer setting in row i, or nil
// when that row is not an integer setting.
func (a *app) settingField(i int) *string {
	id, ok := a.panelMod()
	if !ok {
		return nil
	}
	decl, err := a.settingsOf(id)
	if err != nil || i >= len(decl) || i >= settingRows || decl[i].Kind != mod.KindInt {
		return nil
	}
	return a.s.modValuePtr(id, decl[i].Key, decl[i].Default.String())
}

// modRows lists the enabled mods in load order, then the others by name.
func (a *app) modRows() []modRow {
	byID := map[string]mod.Entry{}
	for _, e := range a.scan {
		byID[e.Folder] = e
	}
	describe := func(e mod.Entry) modRow {
		if e.Err != nil {
			return modRow{id: e.Folder, label: e.Folder + ": " + e.Err.Error(), problem: e.Err.Error()}
		}
		m := e.Manifest
		return modRow{id: e.Folder, label: fmt.Sprintf("%s %s  %s  (%s)", m.ID, m.Version, m.Title, strings.Join(m.AppliesTo, ","))}
	}
	var rows []modRow
	listed := map[string]bool{}
	for _, id := range a.s.Enabled {
		listed[id] = true
		var r modRow
		if e, ok := byID[id]; ok {
			r = describe(e)
		} else {
			r = modRow{id: id, label: a.t(cModFolderMissing, id), problem: a.t(cProblemMissing)}
		}
		r.enabled = true
		rows = append(rows, r)
	}
	for _, e := range a.scan {
		if !listed[e.Folder] {
			rows = append(rows, describe(e))
		}
	}
	return rows
}

// layout is the window's content.
func (a *app) layout() []item {
	var it []item
	rect := func(x, y, w, h int) image.Rectangle { return image.Rect(x, y, x+w, y+h) }
	textW := func(x, y, w int, s string, t tone) {
		it = append(it, item{kind: iText, rect: rect(x, y, w, 16), text: s, tone: t})
	}
	text := func(x, y int, s string, t tone) { textW(x, y, winW-x-10, s, t) }
	const lw = winW - 20 // width of the main view's lists and notes
	heading := func(x, y int, s string) { it = append(it, item{kind: iHeading, rect: rect(x, y, 0, 16), text: s}) }
	button := func(r image.Rectangle, s string, act action) {
		it = append(it, item{kind: iButton, rect: r, text: s, act: act})
	}
	field := func(r image.Rectangle, id fieldID) {
		it = append(it, item{kind: iField, rect: r, text: *a.field(id), focused: a.focus == id, act: action{aFocus, int(id)}})
	}

	gameVersion := a.gameVersion
	if gameVersion == versionUnknown {
		gameVersion = a.t(cUnknown)
	}
	text(10, 6, a.t(cTitle, a.starterVersion, shortRevision()), toneNormal)
	text(330, 6, a.t(cLaunches, gameVersion), toneDim)
	text(10, 24, a.t(cSettingsFile, a.iniPath), toneDim)

	if a.view == viewSettings {
		// Parameters.
		heading(10, 48, a.t(cHeadParams))
		row := func(i int, label string) int { y := 66 + 24*i; textW(10, y+3, 300, label, toneNormal); return y }
		ctl := func(y int) image.Rectangle { return rect(320, y, winW-330, 22) }
		for i, v := range optionViews {
			if i > 0 && optionViews[i-1].row < advancedRow && v.row > advancedRow {
				heading(10, 238, a.t(cHeadAdvanced))
			}
			st, _ := launch.ByKey(v.key)
			y := row(v.row, a.labelText(st, v))
			switch st.Kind {
			case launch.Switch:
				on := *v.value(&a.s).(*bool)
				it = append(it, item{kind: iToggle, rect: ctl(y), text: a.onOff(on), checked: on, act: action{kind: v.act}})
			case launch.Override, launch.Choice:
				button(ctl(y), a.valueText(st, v, a.s.option(st)), action{kind: v.act})
			default:
				field(ctl(y), v.edit)
			}
		}
		heading(10, 404, a.t(cHeadStarter))
		y := row(15, a.t(cCloseAfterPlay))
		it = append(it, item{kind: iToggle, rect: ctl(y), text: a.onOff(a.s.CloseOnPlay), checked: a.s.CloseOnPlay, act: action{kind: aCloseOnPlay}})
		y = row(16, a.t(cGameProgram))
		field(ctl(y), fAgainrom)
		y = row(17, a.t(cLoadUnmarked))
		it = append(it, item{kind: iToggle, rect: ctl(y), text: a.onOff(a.s.AcceptUnmarked), checked: a.s.AcceptUnmarked, act: action{kind: aAcceptUnmarked}})
	} else {
		// Base games.
		heading(10, 48, a.t(cHeadBase))
		bases := a.s.Bases
		top := clampScroll(&a.scroll[0], len(bases), baseRows)
		for k := 0; k < baseRows && top+k < len(bases); k++ {
			b := bases[top+k]
			info := a.infos[b.Path]
			r := item{kind: iBaseRow, rect: rect(10, 66+18*k, lw, 18), checked: strings.EqualFold(b.Key, a.s.LastBase), act: action{aSelectBase, top + k}}
			if info.Valid() {
				r.tone = toneGood
				r.text = a.t(cBaseOK, b.Key, b.Path, a.baseLabel(info))
			} else {
				r.tone = toneBad
				r.text = a.t(cBaseBad, b.Key, b.Path, strings.Join(info.Missing, ", "))
			}
			it = append(it, r)
		}
		if len(bases) == 0 {
			textW(10, 66, lw, a.t(cNoBase), toneDim)
		}
		text(10, 162, a.t(cAddPath), toneDim)
		field(rect(76, 158, 100, 22), fAddBase)
		button(rect(182, 158, 64, 22), a.t(cBrowse), action{kind: aChooseBase})
		button(rect(252, 158, 50, 22), a.t(cAdd), action{kind: aAddBase})
		button(rect(308, 158, 67, 22), a.t(cRemove), action{kind: aRemoveBase})

		// Mods.
		heading(10, 190, a.t(cHeadMods))
		text(10, 214, a.t(cFolder), toneDim)
		field(rect(76, 210, 224, 22), fModsDir)
		button(rect(306, 210, 69, 22), a.t(cRescan), action{kind: aRescan})
		if a.s.ModsDir == "" {
			textW(76, 234, lw-66, a.t(cModsDefault, mod.DefaultDir(a.selfDir())), toneDim)
		}
		rows := a.modRows()
		top = clampScroll(&a.scroll[1], len(rows), modRows)
		for k := 0; k < modRows && top+k < len(rows); k++ {
			m := rows[top+k]
			label := m.label
			tn := toneNormal
			if m.enabled {
				n := 0
				for _, e := range a.s.Enabled {
					n++
					if e == m.id {
						break
					}
				}
				label = fmt.Sprintf("%d. %s", n, label)
			}
			if m.problem != "" {
				tn = toneBad
			}
			it = append(it, item{kind: iModRow, rect: rect(10, 250+18*k, lw, 18), text: label, tone: tn, checked: m.enabled, act: action{aToggleMod, top + k}})
		}
		if a.scanErr != nil {
			textW(10, 382, lw, a.t(cModsReadError, a.scanErr.Error()), toneBad)
		} else if len(rows) == 0 {
			textW(10, 250, lw, a.t(cNoMods), toneDim)
		}
		textW(10, 362, lw, a.t(cTickHint), toneDim)

		// Mod settings.
		heading(10, panelTop, a.t(cHeadModSettings))
		if id, ok := a.panelMod(); !ok {
			textW(10, panelTop+22, lw, a.t(cSelectMod), toneDim)
		} else {
			state, tn := a.t(cModNotEnabled), toneDim
			if a.s.modEnabled(id) {
				state, tn = a.t(cModEnabled), toneNormal
			}
			textW(120, panelTop, 300, a.t(cModState, id, state), tn)
			decl, err := a.settingsOf(id)
			switch {
			case err != nil:
				textW(10, panelTop+22, lw, a.t(cSettingsToml, a.errText(err)), toneBad)
			case len(decl) == 0:
				textW(10, panelTop+22, lw, a.t(cNoModSettings), toneDim)
			default:
				button(rect(winW-85, panelTop-3, 75, 22), a.t(cDefaults), action{kind: aSettingReset})
				for i := 0; i < settingRows && i < len(decl); i++ {
					st, y := decl[i], panelTop+24+24*i
					label := st.Label(a.lang())
					if st.Kind == mod.KindInt {
						label = fmt.Sprintf("%s (%d..%d)", label, st.Min, st.Max)
					}
					textW(10, y+3, 310, label, toneNormal)
					ctl, val := rect(330, y, winW-340, 22), a.settingText(id, st)
					switch st.Kind {
					case mod.KindInt:
						it = append(it, item{kind: iField, rect: ctl, text: val, focused: a.focus == fSetting0+fieldID(i), act: action{aFocus, int(fSetting0) + i}})
					case mod.KindBool:
						v, perr := st.Parse(val)
						it = append(it, item{kind: iToggle, rect: ctl, text: a.onOff(perr == nil && v.Bool), checked: perr == nil && v.Bool, act: action{aSettingCycle, i}})
					default:
						button(ctl, val, action{aSettingCycle, i})
					}
				}
				if len(decl) > settingRows {
					textW(10, panelTop+24+24*settingRows, lw, a.t(cMoreSettings, len(decl)-settingRows), toneDim)
				}
			}
		}
	}
	// Actions.
	by := 410 + panelH
	button(rect(10, by, 100, 28), a.t(cPlay), action{kind: aPlay})
	button(rect(118, by, 100, 28), a.t(cCheck), action{kind: aCheck})
	button(rect(226, by, 140, 28), a.t(cSaveSettings), action{kind: aSave})
	button(rect(374, by, 130, 28), a.t(cMapEditor), action{kind: aMapEditor})
	if a.view == viewSettings {
		button(rect(512, by, 110, 28), a.t(cClose), action{kind: aSettingsClose})
	} else {
		button(rect(512, by, 110, 28), a.t(cSettings), action{kind: aSettingsOpen})
	}
	if cmd, err := a.command(); err != nil {
		text(10, by+36, a.errText(err), toneBad)
	} else {
		text(10, by+36, cmd, toneDim)
	}
	st := toneGood
	if a.statusBad {
		st = toneBad
	}
	if a.status != "" {
		text(10, by+56, a.status, st)
	}
	for k, line := range a.output {
		if k == outputLines {
			break
		}
		text(14, by+80+18*k, line, toneNormal)
	}
	return it
}

// outputLines is how many lines of Check output the window keeps in view.
const outputLines = 6

// versionUnknown is what gameVersion holds until the game program has told its
// version.
const versionUnknown = "unknown"

// onOff is the caption of a switch.
func (a *app) onOff(b bool) string {
	if b {
		return a.t(cOn)
	}
	return a.t(cOff)
}

// valueText is the value s of the row st as its button shows it: the
// option's caption in the caption language, a value without one as it is.
func (a *app) valueText(st launch.Setting, v optionView, s string) string {
	if i, ok := st.Index(s); ok && i < len(v.values) && v.values[i] != "" {
		return a.t(v.values[i])
	}
	return s
}

func clampScroll(offset *int, n, capacity int) int {
	if max := n - capacity; *offset > max {
		*offset = max
	}
	if *offset < 0 {
		*offset = 0
	}
	return *offset
}

// command is the line Play would run.
func (a *app) command() (string, error) {
	root, _ := a.selectedRoot()
	args, err := a.gameArgs(root, false)
	if err != nil {
		return "", err
	}
	return filepath.Base(a.s.gameExecutable(a.selfPath, a.goos)) + " " + quoteArgs(args), nil
}

func quoteArgs(args []string) string {
	out := make([]string, len(args))
	for i, s := range args {
		if s == "" || strings.ContainsAny(s, " \t\"") {
			s = `"` + s + `"`
		}
		out[i] = s
	}
	return strings.Join(out, " ")
}

// click handles a left click at p.
func (a *app) click(p image.Point) {
	if a.folderDialogResults != nil {
		return
	}
	items := a.layout()
	prev := a.focus
	act := action{}
	for i := len(items) - 1; i >= 0; i-- {
		if it := items[i]; it.act.kind != aNone && p.In(it.rect) {
			act = it.act
			break
		}
	}
	if act.kind != aChooseBase {
		a.focus = fNone
	}
	if act.kind != aNone {
		a.apply(act)
	}
	if prev != a.focus {
		a.dirty = true
	}
}

func (a *app) apply(act action) {
	if a.folderDialogResults != nil {
		return
	}
	if act.kind == aChooseBase {
		a.chooseBase()
		return
	}
	a.dirty = true
	switch act.kind {
	case aFocus:
		a.focus = fieldID(act.i)
	case aSelectBase:
		if act.i < len(a.s.Bases) {
			b := a.s.Bases[act.i]
			a.s.LastBase = b.Key
			if info := a.infos[b.Path]; info.Valid() {
				a.ok(cText, a.baseNote(info))
			}
		}
	case aToggleMod:
		rows := a.modRows()
		if act.i < len(rows) {
			a.selMod = rows[act.i].id
			if r := rows[act.i]; r.enabled || r.problem == "" {
				a.s.toggleMod(r.id)
			} else {
				a.fail(cModCannotEnable, r.id, r.problem)
			}
		}
	case aSettingCycle:
		a.cycleSetting(act.i)
	case aSettingReset:
		if id, ok := a.panelMod(); ok {
			a.s.resetMod(id)
		}
	case aAddBase:
		a.addBase()
	case aRemoveBase:
		if _, ok := a.selectedRoot(); ok {
			a.s.removeBase(a.s.LastBase)
			a.s.LastBase = ""
			if len(a.s.Bases) > 0 {
				a.s.LastBase = a.s.Bases[0].Key
			}
		}
	case aRescan:
		a.rescan()
		a.ok(cModsFound, len(a.scan), a.s.modsDir(a.selfDir()))
	case aSound, aVideo, aMovies, aMarkers, aChicken, aWindowed, aPicker:
		a.stepOption(act.kind)
	case aAcceptUnmarked:
		a.s.AcceptUnmarked = !a.s.AcceptUnmarked
	case aCloseOnPlay:
		a.s.CloseOnPlay = !a.s.CloseOnPlay
	case aPlay:
		a.play()
	case aCheck:
		a.check()
	case aSave:
		a.save()
	case aMapEditor:
		a.openMapEditor()
	case aSettingsOpen:
		a.setView(viewSettings)
	case aSettingsClose:
		a.setView(viewMain)
	}
}

// setView shows v and drops the focus, which may sit on a field v does not
// show.
func (a *app) setView(v viewKind) {
	a.view, a.focus, a.dirty = v, fNone, true
}

// fieldShown reports whether the current view shows the field id.
func (a *app) fieldShown(id fieldID) bool {
	switch id {
	case fAgainrom:
		return a.view == viewSettings
	}
	for _, v := range optionViews {
		if v.edit == id && id != fNone {
			return a.view == viewSettings
		}
	}
	return a.view == viewMain
}

// cycleSetting flips the bool setting in panel row i, or steps the choice
// setting to its next choice.
func (a *app) cycleSetting(i int) {
	id, ok := a.panelMod()
	if !ok {
		return
	}
	decl, err := a.settingsOf(id)
	if err != nil || i >= len(decl) || i >= settingRows {
		return
	}
	st := decl[i]
	cur := a.settingText(id, st)
	next := ""
	switch st.Kind {
	case mod.KindBool:
		v, perr := st.Parse(cur)
		next = launch.FormatBool(perr != nil || !v.Bool)
	case mod.KindChoice:
		next = st.Choices[0]
		for k, c := range st.Choices {
			if c == cur && k+1 < len(st.Choices) {
				next = st.Choices[k+1]
			}
		}
	default:
		return
	}
	*a.s.modValuePtr(id, st.Key, st.Default.String()) = next
}

func (a *app) addBase() {
	path := strings.Trim(strings.TrimSpace(a.addText), `"`)
	if path == "" {
		a.fail(cTypeFolder)
		return
	}
	if st, err := os.Stat(path); err != nil || !st.IsDir() {
		a.fail(cNotFolder, path)
		return
	}
	key := a.s.addBase(path)
	a.inspectBase(path)
	a.s.LastBase = key
	a.addText = ""
	if info := a.infos[path]; info.Valid() {
		a.ok(cAdded, key, a.baseNote(info))
	} else {
		a.fail(cAddedNotInstall, key, strings.Join(info.Missing, ", "))
	}
}

// types appends typed text to the focused field.
func (a *app) typed(s string) {
	if a.folderDialogResults != nil {
		return
	}
	f := a.field(a.focus)
	if f == nil {
		return
	}
	before := *f
	*f = ui.EditText(ui.TextInputState{Text: *f}, ui.TextInputKeys{Typed: s},
		ui.TextInputRules{AppendOnly: true, SplitLineBreaks: true}).Text
	a.dirty = a.dirty || *f != before
}

// backspace removes the focused field's last character.
func (a *app) backspace() {
	if a.folderDialogResults != nil {
		return
	}
	if f := a.field(a.focus); f != nil && *f != "" {
		*f = ui.EditText(ui.TextInputState{Text: *f}, ui.TextInputKeys{Backspace: true},
			ui.TextInputRules{AppendOnly: true, SplitLineBreaks: true}).Text
		a.dirty = true
	}
}

// enter confirms the focused field.
func (a *app) enter() {
	if a.folderDialogResults != nil {
		return
	}
	switch a.focus {
	case fAddBase:
		a.addBase()
	case fModsDir:
		a.rescan()
	case fAgainrom:
		a.probeVersion()
	}
	a.focus = fNone
	a.dirty = true
}

// escape drops the focus and leaves the settings view.
func (a *app) escape() {
	if a.folderDialogResults == nil {
		a.focus, a.dirty = fNone, true
		a.view = viewMain
	}
}

// tab moves focus to the next field.
func (a *app) tab() {
	if a.folderDialogResults != nil {
		return
	}
	for n := 0; n < int(fieldCount); n++ {
		a.focus = a.focus%(fieldCount-1) + 1
		if a.field(a.focus) != nil && a.fieldShown(a.focus) {
			break
		}
	}
	a.dirty = true
}

// pasteText pastes the first line of the clipboard into the focused field.
func (a *app) pasteText() {
	if a.folderDialogResults != nil || a.focus == fNone || a.deps.paste == nil {
		return
	}
	s := strings.TrimSpace(a.deps.paste())
	if i := strings.IndexAny(s, "\r\n"); i >= 0 {
		s = s[:i]
	}
	a.typed(strings.Trim(s, `"`))
}

// wheel scrolls the list under p by lines.
func (a *app) wheel(p image.Point, lines int) {
	if a.folderDialogResults != nil {
		return
	}
	if a.view != viewMain {
		return
	}
	switch {
	case p.In(image.Rect(10, 66, winW-10, 156)):
		a.scroll[0] += lines
	case p.In(image.Rect(10, 250, winW-10, 360)):
		a.scroll[1] += lines
	default:
		return
	}
	a.dirty = true
}

func (a *app) save() {
	a.s.store(a.file)
	err := os.MkdirAll(filepath.Dir(a.iniPath), 0o755)
	if err == nil {
		err = os.WriteFile(a.iniPath, a.file.Bytes(), 0o644)
	}
	if err != nil {
		a.fail(cSaveFailed, err)
		return
	}
	a.ok(cSaved, a.iniPath)
	a.probeVersion()
}

// launchPlan resolves the executable and arguments for a launch or a check and
// reports why they cannot be had.
func (a *app) launchPlan(check bool) (string, []string, error) {
	root, ok := a.selectedRoot()
	if !ok {
		return "", nil, captionErr(cNoBaseSelected)
	}
	if info := a.infos[root]; !info.Valid() {
		return "", nil, captionErr(cBaseNotInstall, a.s.LastBase, strings.Join(info.Missing, ", "))
	}
	for _, m := range a.modRows() {
		if m.enabled && m.problem != "" {
			return "", nil, captionErr(cModUnusable, m.id, m.problem)
		}
	}
	args, err := a.gameArgs(root, check)
	if err != nil {
		return "", nil, err
	}
	exe := a.s.gameExecutable(a.selfPath, a.goos)
	if !fileExists(exe) {
		return "", nil, captionErr(cGameNotFound, exe)
	}
	return exe, args, nil
}

// gameArgs is the game's argument list for root.
func (a *app) gameArgs(root string, check bool) ([]string, error) {
	modSettings, err := a.modSettings()
	if err != nil {
		return nil, err
	}
	g := gameLaunch{Root: root, ModsDir: a.s.modsDir(a.selfDir()), ModSettings: modSettings, Check: check}
	if id := a.infos[root].Base.ID(); basepkg.Nameable(id) {
		g.Base = id
	}
	return a.s.launchArgs(g)
}

func (a *app) play() {
	exe, args, err := a.launchPlan(false)
	if err != nil {
		a.fail(cCannotPlay, a.errText(err))
		return
	}
	if err := a.deps.start(exe, args); err != nil {
		a.fail(cCannotStart, exe, err)
		return
	}
	a.ok(cStarted, filepath.Base(exe))
	if a.s.CloseOnPlay {
		a.closing = true
	}
}

func (a *app) check() {
	exe, args, err := a.launchPlan(true)
	if err != nil {
		a.fail(cCannotCheck, a.errText(err))
		return
	}
	a.ok(cRunningCheck, filepath.Base(exe))
	a.output = nil
	a.deps.async(func() {
		out, err := a.deps.run(exe, args)
		a.results <- result{kind: "check", output: out, err: err}
	})
}

// mapEditorPlan resolves the map editor beside this program and its arguments
// for the selected base, and reports why they cannot be had.
func (a *app) mapEditorPlan() (string, []string, error) {
	root, ok := a.selectedRoot()
	if !ok {
		return "", nil, captionErr(cNoBaseSelected)
	}
	if info := a.infos[root]; !info.Valid() {
		return "", nil, captionErr(cBaseNotInstall, a.s.LastBase, strings.Join(info.Missing, ", "))
	}
	exe := editorExecutable(a.selfPath, a.goos)
	if !fileExists(exe) {
		return "", nil, captionErr(cEditorNotFound, exe)
	}
	return exe, []string{"-assets", root}, nil
}

// openMapEditor first runs the map editor's map list on the selected base,
// which loads the base as the window would and exits; poll starts the editor
// the way Play starts the game when the list passed, and otherwise shows the
// editor's own error output, which a window started without a console loses.
func (a *app) openMapEditor() {
	exe, args, err := a.mapEditorPlan()
	if err != nil {
		a.fail(cCannotOpenEditor, a.errText(err))
		return
	}
	a.ok(cCheckingEditor, filepath.Base(exe), a.s.LastBase)
	a.output = nil
	a.deps.async(func() {
		out, err := a.deps.run(exe, append(append([]string{}, args...), "-list"))
		a.results <- result{kind: "editor", output: out, err: err, launch: append([]string{exe}, args...)}
	})
}

// startMapEditor takes in the map list check of openMapEditor.
func (a *app) startMapEditor(r result) {
	exe := r.launch[0]
	if r.err != nil {
		a.output = outputOf(r.output)
		a.fail(cEditorCannotOpen, r.err)
		return
	}
	if err := a.deps.start(exe, r.launch[1:]); err != nil {
		a.fail(cCannotStart, exe, err)
		return
	}
	a.ok(cStarted, filepath.Base(exe))
}

// probeVersion asks the game program for its version line.
func (a *app) probeVersion() {
	exe := a.s.gameExecutable(a.selfPath, a.goos)
	if !fileExists(exe) {
		a.setGameVersion("")
		return
	}
	a.deps.async(func() {
		args, _ := launch.Version.Args(launch.FormatBool(true))
		out, err := a.deps.run(exe, args)
		a.results <- result{kind: "version", output: out, err: err}
	})
}

func (a *app) setGameVersion(line string) {
	v := versionUnknown
	if f := strings.Fields(line); len(f) >= 2 && f[0] == "againrom" {
		v = strings.Join(f[1:], " ")
	}
	if v != a.gameVersion {
		a.gameVersion, a.dirty = v, true
	}
}

// poll takes in the results of finished background work.
func (a *app) poll() {
	for {
		select {
		case r := <-a.results:
			switch r.kind {
			case "version":
				if r.err != nil {
					r.output = ""
				}
				a.setGameVersion(strings.TrimSpace(r.output))
			case "check":
				a.output = outputOf(r.output)
				if r.err != nil {
					a.fail(cCheckFailed, r.err)
				} else {
					a.ok(cCheckPassed)
				}
				a.dirty = true
			case "editor":
				a.startMapEditor(r)
			}
		default:
			return
		}
	}
}

// outputOf is a program's output as the window's lines.
func outputOf(s string) []string {
	return strings.Split(strings.TrimRight(strings.ReplaceAll(s, "\r\n", "\n"), "\n"), "\n")
}

// takeDirty reports whether the image needs redrawing, and clears the flag.
func (a *app) takeDirty() bool {
	d := a.dirty
	a.dirty = false
	return d
}

// baseLabel is what a base row says of an install: its profile id, whether the
// build is one the profile knows, and whether the profile states limits. An
// install with no profile says its language.
func (a *app) baseLabel(info game.InstallInfo) string {
	m := info.Base
	if !m.Profile.Known() {
		return info.Language
	}
	label := m.Profile.ID
	if !m.Exact {
		label += a.t(cUnrecognised)
	}
	if hasLimits(m.Profile) {
		label += a.t(cHasLimits)
	}
	return label
}

func hasLimits(p basepkg.Profile) bool {
	l := p.Limits
	return l.NoCharacterGeneration || l.OriginalSaveRefusal != "" || len(l.Notes) != 0
}

// baseNote is the status line a selected or added base gets: the profile and
// each limit the profile states.
func (a *app) baseNote(info game.InstallInfo) string {
	m := info.Base
	if !m.Profile.Known() {
		return a.t(cInstallOf, info.Language)
	}
	parts := []string{m.String()}
	l := m.Profile.Limits
	if l.NoCharacterGeneration {
		parts = append(parts, a.t(cNoCharGen, m.Profile.Mission()))
	}
	if l.OriginalSaveRefusal != "" {
		parts = append(parts, a.t(cSavesRefused))
	}
	parts = append(parts, l.Notes...)
	return strings.Join(parts, "; ")
}
