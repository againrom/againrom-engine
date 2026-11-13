package main

import (
	"errors"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	basepkg "againrom/pkg/base"
	"againrom/pkg/hostfile"
	"againrom/pkg/ini"
	"againrom/pkg/installinfo"
	"againrom/pkg/launch"
	"againrom/pkg/locale"
	"againrom/pkg/mod"
	"againrom/pkg/pathdialog"
	"againrom/pkg/ui/widgetkit"
)

// Window size in pixels. panelH is the height of the mod settings panel, which
// sits between the mod list and the action buttons. The window fits a screen
// 768 pixels high; the mission list takes its rows from the Check output's.
const (
	winW   = 760
	panelH = 140
	winH   = 620 + panelH
)

// Rows in view of the four scrolling lists: the bases, the mods, the selected
// mod's settings and its missions.
const (
	baseRows    = 5
	modRows     = 6
	settingRows = 5
	missionRows = 3
)

// panelTop is the y of the settings panel's heading, actionsTop that of the
// action buttons, with the command line and the status line under them, and
// missionsTop that of the mission list's heading, under the status line.
const (
	panelTop    = 398
	actionsTop  = 410 + panelH
	statusTop   = actionsTop + 56
	missionsTop = statusTop + 26
)

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
	fSetting0 // fSetting0+k edits the integer setting in the panel's row k in view
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
	aSettingCycle // flip or step the panel's bool or choice setting, i is its declaration index
	aSettingDown  // lower the panel's integer setting by one within its bounds, i as above
	aSettingUp    // raise it by one within its bounds
	aSettingReset
	aMapEditor
	aChooseBase
	aSettingsOpen
	aSettingsClose
	aWindowed
	aSelectMission // choose the mission in row i of the panel mod's list
	aStartMission  // start the game on the chosen mission
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
	refusal string // non-empty: the game refuses the mod on the selected base
	order   int    // place in the load order from 1; 0 when not ordered
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
	inspect func(root string) installinfo.Info
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

	infos    map[string]installinfo.Info // by base path
	scan     []mod.Entry
	scanErr  error
	selMod   string                   // mod whose settings the panel shows
	declared map[string]declaredSetts // settings.toml of each scanned mod
	// missions holds what each scanned mod declares, by id and language;
	// selMission is the number of the mission Start names, 0 for none, and
	// selMissionMod the mod it is a mission of.
	missions      map[string]declaredMissions
	selMission    int
	selMissionMod string
	// focused is whether the window had the focus at the last frame.
	focused   bool
	addText   string
	focus     fieldID
	view      viewKind
	scroll    [listCount]int // offset of each scrolling list
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
		infos:          map[string]installinfo.Info{},
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
	a.s.LastBase = launch.SelectedKey(a.s.Bases, a.s.LastBase)
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
	a.missions = map[string]declaredMissions{}
	a.dirty = true
}

// setFocused takes in whether the window has the focus. Regaining it reads
// the mods folder again, so a mod published meanwhile is listed; the ticks,
// the order and the selections are settings and stay.
func (a *app) setFocused(now bool) {
	if now && !a.focused {
		a.rescan()
	}
	a.focused = now
}

// declaredMissions is what reading a mod's missions gave.
type declaredMissions struct {
	list []mod.Mission
	err  error
}

// missionsOf returns the missions the mod id declares, worded in the language
// of the selected base and read once per scan by the mod package's one
// mission description.
func (a *app) missionsOf(id string) ([]mod.Mission, error) {
	lang := a.lang()
	key := id + "\x00" + lang
	if d, ok := a.missions[key]; ok {
		return d.list, d.err
	}
	d := declaredMissions{err: captionErr(cModNoFolder, id)}
	for _, e := range a.scan {
		if e.Folder == id && e.Err == nil {
			var data mod.MissionData
			data, _, d.err = mod.DescribeMissions(e.Dir, e.Manifest.ID, lang)
			d.list = data.Listed()
		}
	}
	a.missions[key] = d
	return d.list, d.err
}

// panelMissions are the missions the mission list shows: those of the panel's
// mod while it is enabled.
func (a *app) panelMissions() ([]mod.Mission, bool) {
	id, ok := a.panelMod()
	if !ok || !a.s.modEnabled(id) {
		return nil, false
	}
	list, err := a.missionsOf(id)
	return list, err == nil
}

// missionLabel is a mission as its row of the mission list shows it: the
// number and the title the mod's strings give it in the starter's language.
func missionLabel(m mod.Mission) string {
	return strings.TrimSpace(fmt.Sprintf("%d. %s", m.Number, m.Title))
}

// chosenMission is the number of the mission Start names, while the mission
// list still shows it as a mission of the mod it was chosen in.
func (a *app) chosenMission() (int, bool) {
	if id, ok := a.panelMod(); !ok || id != a.selMissionMod {
		return 0, false
	}
	list, _ := a.panelMissions()
	for _, m := range list {
		if m.Number == a.selMission {
			return m.Number, true
		}
	}
	return 0, false
}

// missionsShown reports whether the window shows the mission list: on the
// main view while the panel has a mod.
func (a *app) missionsShown() bool {
	_, ok := a.panelMod()
	return ok && a.view == viewMain
}

// outputBox is the frame of the Check output and the lines it holds. The
// mission list, while shown, stands above it and leaves it two lines.
func (a *app) outputBox() (image.Rectangle, int) {
	if a.missionsShown() {
		return image.Rect(10, scrollLists[listMissions].area.Max.Y+4, winW-10, winH-6), 2
	}
	return image.Rect(10, statusTop+22, winW-10, winH-6), outputLines
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
// is listed, else the first enabled mod of the list.
func (a *app) panelMod() (string, bool) {
	rows := a.modRows()
	for _, r := range rows {
		if r.id == a.selMod {
			return r.id, true
		}
	}
	if len(rows) > 0 && rows[0].enabled {
		return rows[0].id, true
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

// panelSettings is the panel's mod and its declared settings, when the panel
// shows any.
func (a *app) panelSettings() (string, []mod.Setting, bool) {
	id, ok := a.panelMod()
	if !ok {
		return "", nil, false
	}
	decl, err := a.settingsOf(id)
	return id, decl, err == nil && len(decl) > 0
}

// settingInView is the setting in the panel's row k in view, with its mod.
func (a *app) settingInView(k int) (string, mod.Setting, bool) {
	id, decl, ok := a.panelSettings()
	if !ok {
		return "", mod.Setting{}, false
	}
	top, shown := a.window(listSettings, len(decl))
	if k < 0 || k >= shown {
		return "", mod.Setting{}, false
	}
	return id, decl[top+k], true
}

// settingField returns the text of the integer setting in the panel's row k in
// view for editing, or nil when that row is not an integer setting.
func (a *app) settingField(k int) *string {
	id, st, ok := a.settingInView(k)
	if !ok || st.Kind != mod.KindInt {
		return nil
	}
	return a.s.modValuePtr(id, st.Key, st.Default.String())
}

// hasField reports whether field id exists, without storing the default of a
// setting row as settingField does.
func (a *app) hasField(id fieldID) bool {
	if id >= fSetting0 && id < fSetting0+settingRows {
		_, st, ok := a.settingInView(int(id - fSetting0))
		return ok && st.Kind == mod.KindInt
	}
	return a.field(id) != nil
}

// modBase is the base the game orders mods for on the selected base, and
// whether a base is selected.
func (a *app) modBase() (string, bool) {
	root, ok := a.selectedRoot()
	if !ok {
		return "", false
	}
	return a.infos[root].BaseID(), true
}

// modRows lists the enabled mods in the load order mod.Order gives them on the
// selected base, each numbered and with the refusals that stand on it; then
// the enabled mods that cannot be read, by name; then the others by name, each
// with the refusals mod.Fit raises for it on the base. Without a selected base
// nothing is ordered or refused.
func (a *app) modRows() []modRow {
	byID := map[string]mod.Entry{}
	for _, e := range a.scan {
		byID[e.Folder] = e
	}
	baseID, haveBase := a.modBase()
	var unusable []string
	for _, id := range slices.Sorted(slices.Values(a.s.Enabled)) {
		if e, ok := byID[id]; !ok || e.Err != nil {
			unusable = append(unusable, id)
		}
	}
	describe := func(e mod.Entry, refused []mod.Refusal) modRow {
		if e.Err != nil {
			return modRow{id: e.Folder, label: e.Folder + ": " + e.Err.Error(), problem: e.Err.Error()}
		}
		m := e.Manifest
		r := modRow{id: e.Folder, label: fmt.Sprintf("%s %s  %s  (%s)", m.ID, m.Version, m.Title, strings.Join(m.AppliesTo, ","))}
		if len(refused) > 0 {
			var why []string
			for _, f := range refused {
				why = append(why, a.refusalText(e.Folder, f, unusable))
			}
			r.refusal = strings.Join(why, "; ")
			r.label = fmt.Sprintf("%s %s: %s  %s  (%s)", m.ID, m.Version, r.refusal, m.Title, strings.Join(m.AppliesTo, ","))
		}
		return r
	}
	var usable []mod.Entry
	for _, id := range slices.Sorted(slices.Values(a.s.Enabled)) {
		if e, ok := byID[id]; ok && e.Err == nil {
			usable = append(usable, e)
		}
	}
	order := mod.Ordering{Mods: usable}
	if haveBase {
		order = mod.Order(usable, baseID)
	}
	var rows []modRow
	for i, e := range order.Mods {
		r := describe(e, order.Of(e.Folder))
		r.enabled = true
		if haveBase {
			r.order = i + 1
		}
		rows = append(rows, r)
	}
	for _, id := range unusable {
		r := modRow{id: id, label: a.t(cModFolderMissing, id), problem: a.t(cProblemMissing)}
		if e, ok := byID[id]; ok {
			r = describe(e, nil)
		}
		r.enabled = true
		rows = append(rows, r)
	}
	for _, e := range a.scan {
		if a.s.modEnabled(e.Folder) {
			continue
		}
		var refused []mod.Refusal
		if haveBase && e.Err == nil {
			refused = mod.Fit(e.Manifest, baseID)
		}
		rows = append(rows, describe(e, refused))
	}
	return rows
}

// refusalText is a refusal as the row of mod id shows it, in the language of
// the selected base, with the mods and values it names as data. unusable lists
// the enabled mods that cannot be read or are missing: a requirement on one of
// them is worded as such, not as a mod that is not enabled.
func (a *app) refusalText(id string, r mod.Refusal, unusable []string) string {
	switch r.Kind {
	case mod.RefusedAPI:
		return a.t(cRefusedAPI, r.API, mod.SupportedAPI)
	case mod.RefusedBase:
		return a.t(cRefusedBase, r.Base)
	case mod.RefusedConflict:
		return a.t(cRefusedConflict, r.Other)
	case mod.RefusedRequirementMissing:
		if slices.Contains(unusable, r.Other) {
			return a.t(cRefusedUnusable, r.Other)
		}
		return a.t(cRefusedMissing, r.Other)
	case mod.RefusedRequirementVersion:
		return a.t(cRefusedVersion, r.Requirement, r.Version)
	case mod.RefusedCycle:
		var others []string
		for _, c := range r.Cycle {
			if c != id {
				others = append(others, c)
			}
		}
		return a.t(cRefusedCycle, strings.Join(others, ", "))
	}
	return a.t(cRefusedUnreadable, r.Requirement)
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
		top, shown := a.window(listBases, len(bases))
		for k := 0; k < shown; k++ {
			b := bases[top+k]
			info := a.infos[b.Path]
			r := item{kind: iBaseRow, rect: scrollLists[listBases].row(k), checked: strings.EqualFold(b.Key, a.s.LastBase), act: action{aSelectBase, top + k}}
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
			textW(76, 234, lw-66, a.t(cModsDefault, launch.ModsFolder("", a.selfDir())), toneDim)
		}
		rows := a.modRows()
		top, shown = a.window(listMods, len(rows))
		for k := 0; k < shown; k++ {
			m := rows[top+k]
			label := m.label
			tn := toneNormal
			if m.order > 0 {
				label = fmt.Sprintf("%d. %s", m.order, label)
			}
			if m.problem != "" || m.refusal != "" {
				tn = toneBad
			}
			it = append(it, item{kind: iModRow, rect: scrollLists[listMods].row(k), text: label, tone: tn, checked: m.enabled, act: action{aToggleMod, top + k}})
		}
		if a.scanErr != nil {
			textW(10, 382, lw, a.t(cModsReadError, a.scanErr.Error()), toneBad)
		} else if len(rows) == 0 {
			textW(10, 250, lw, a.t(cNoMods), toneDim)
		}
		hint := cTickHintPlain
		if _, ok := a.modBase(); ok {
			hint = cTickHint
		}
		textW(10, 362, lw, a.t(hint), toneDim)

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
				top, shown := a.window(listSettings, len(decl))
				if shown < len(decl) {
					textW(430, panelTop, winW-525, a.t(cSettingsInView, top+1, top+shown, len(decl)), toneDim)
				}
				for k := 0; k < shown; k++ {
					i := top + k
					st, y := decl[i], scrollLists[listSettings].row(k).Min.Y
					label := st.Label(a.lang())
					if st.Kind == mod.KindInt {
						label = fmt.Sprintf("%s (%d..%d)", label, st.Min, st.Max)
					}
					textW(10, y+3, 310, label, toneNormal)
					ctl, val := rect(330, y, 204, 22), a.settingText(id, st)
					v, perr := st.Parse(val)
					switch st.Kind {
					case mod.KindInt:
						it = append(it, item{kind: iField, rect: rect(330, y, 150, 22), text: val, focused: a.focus == fSetting0+fieldID(k), act: action{aFocus, int(fSetting0) + k}})
						button(rect(486, y, 22, 22), "-", action{aSettingDown, i})
						button(rect(512, y, 22, 22), "+", action{aSettingUp, i})
					case mod.KindBool:
						it = append(it, item{kind: iToggle, rect: ctl, text: a.onOff(perr == nil && v.Bool), checked: perr == nil && v.Bool, act: action{aSettingCycle, i}})
					default:
						button(ctl, val, action{aSettingCycle, i})
					}
					// An empty value is the default; any other the loader refuses
					// shows the loader's own words.
					if perr != nil && strings.TrimSpace(val) != "" {
						textW(542, y+3, winW-552, perr.Error(), toneBad)
					}
				}
			}
		}
	}
	// Actions.
	by := actionsTop
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
		text(10, statusTop, a.status, st)
	}
	// Missions of the panel's mod, under the status line.
	if id, _ := a.panelMod(); a.missionsShown() {
		lw := winW - 20
		heading(10, missionsTop, a.t(cHeadMissions))
		if !a.s.modEnabled(id) {
			textW(10, missionsTop+22, lw, a.t(cMissionsDisabled), toneDim)
		} else if list, err := a.missionsOf(id); err != nil {
			textW(10, missionsTop+22, lw, a.t(cMissionsToml, a.errText(err)), toneBad)
		} else if len(list) == 0 {
			textW(10, missionsTop+22, lw, a.t(cNoMissions), toneDim)
		} else {
			textW(120, missionsTop, 200, id, toneNormal)
			button(rect(winW-150, missionsTop-3, 140, 22), a.t(cStartMission), action{kind: aStartMission})
			top, shown := a.window(listMissions, len(list))
			if shown < len(list) {
				textW(330, missionsTop, winW-490, a.t(cSettingsInView, top+1, top+shown, len(list)), toneDim)
			}
			for k := 0; k < shown; k++ {
				m := list[top+k]
				it = append(it, item{kind: iBaseRow, rect: scrollLists[listMissions].row(k), text: missionLabel(m),
					checked: m.Number == a.selMission && id == a.selMissionMod, act: action{aSelectMission, top + k}})
			}
		}
	}
	box, lines := a.outputBox()
	for k, line := range a.output {
		if k == lines {
			break
		}
		text(14, box.Min.Y+2+18*k, line, toneNormal)
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

// scrollList is one scrolling list of the main view: rows of rowH pixels down
// from the top of area, at most rows of them in view, moved one row a wheel
// notch over area. Its offset is the app's scroll entry of the same index.
type scrollList struct {
	area       image.Rectangle
	rowH, rows int
}

// The scrolling lists, as indices of scrollLists and app.scroll.
const (
	listBases = iota
	listMods
	listSettings
	listMissions
	listCount
)

var scrollLists = [listCount]scrollList{
	listBases:    {area: image.Rect(10, 66, winW-10, 156), rowH: 18, rows: baseRows},
	listMods:     {area: image.Rect(10, 250, winW-10, 360), rowH: 18, rows: modRows},
	listSettings: {area: image.Rect(10, panelTop+24, winW-10, panelTop+24+24*settingRows), rowH: 24, rows: settingRows},
	listMissions: {area: image.Rect(10, missionsTop+22, winW-10, missionsTop+22+18*missionRows), rowH: 18, rows: missionRows},
}

// row is the rectangle of the list's row k in view.
func (l scrollList) row(k int) image.Rectangle {
	y := l.area.Min.Y + l.rowH*k
	return image.Rect(l.area.Min.X, y, l.area.Max.X, y+l.rowH)
}

// window clamps list's offset to n rows and returns the first row in view and
// how many rows are in view.
func (a *app) window(list, n int) (top, shown int) {
	rows := scrollLists[list].rows
	top = clampScroll(&a.scroll[list], n, rows)
	return top, min(rows, n-top)
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
	args, err := a.gameArgs(root, false, "")
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
			if rows[act.i].id != a.selMod {
				a.scroll[listSettings], a.scroll[listMissions] = 0, 0
				a.selMission, a.selMissionMod = 0, ""
			}
			a.selMod = rows[act.i].id
			if r := rows[act.i]; r.enabled || r.problem == "" {
				a.s.toggleMod(r.id)
			} else {
				a.fail(cModCannotEnable, r.id, r.problem)
			}
		}
	case aSettingCycle:
		a.cycleSetting(act.i)
	case aSettingDown:
		a.stepSetting(act.i, -1)
	case aSettingUp:
		a.stepSetting(act.i, 1)
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
	case aSelectMission:
		if list, _ := a.panelMissions(); act.i < len(list) {
			a.selMission = list[act.i].Number
			a.selMissionMod, _ = a.panelMod()
		}
	case aStartMission:
		if n, ok := a.chosenMission(); ok {
			a.start(strconv.Itoa(n))
		} else {
			a.fail(cCannotPlay, a.t(cSelectMission))
		}
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

// cycleSetting flips the panel's bool setting i, an index into its mod's
// declarations, or steps the choice setting to its next choice.
func (a *app) cycleSetting(i int) {
	id, decl, ok := a.panelSettings()
	if !ok || i < 0 || i >= len(decl) {
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

// stepSetting moves the panel's integer setting i by one in the direction of
// by and keeps it within its bounds; a value that is not an integer steps from
// the default.
func (a *app) stepSetting(i, by int) {
	id, decl, ok := a.panelSettings()
	if !ok || i < 0 || i >= len(decl) || decl[i].Kind != mod.KindInt {
		return
	}
	st := decl[i]
	n, err := strconv.ParseInt(strings.TrimSpace(a.settingText(id, st)), 10, 64)
	if err != nil {
		n = st.Default.Int
	}
	switch {
	case by < 0 && n > st.Min:
		n--
	case by > 0 && n < st.Max:
		n++
	}
	n = max(st.Min, min(st.Max, n))
	*a.s.modValuePtr(id, st.Key, st.Default.String()) = strconv.FormatInt(n, 10)
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
	if a.folderDialogResults != nil || s == "" {
		return
	}
	f := a.field(a.focus)
	if f == nil {
		return
	}
	before := *f
	*f = widgetkit.EditText(widgetkit.TextInputState{Text: *f}, widgetkit.TextInputKeys{Typed: s},
		widgetkit.TextInputRules{AppendOnly: true, SplitLineBreaks: true}).Text
	a.dirty = a.dirty || *f != before
}

// backspace removes the focused field's last character.
func (a *app) backspace() {
	if a.folderDialogResults != nil {
		return
	}
	if f := a.field(a.focus); f != nil && *f != "" {
		*f = widgetkit.EditText(widgetkit.TextInputState{Text: *f}, widgetkit.TextInputKeys{Backspace: true},
			widgetkit.TextInputRules{AppendOnly: true, SplitLineBreaks: true}).Text
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
		if a.hasField(a.focus) && a.fieldShown(a.focus) {
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
	for i, l := range scrollLists {
		if p.In(l.area) {
			a.scroll[i] += lines
			a.dirty = true
			// A setting field is a row in view; scrolled, it would edit
			// another setting.
			if i == listSettings && a.focus >= fSetting0 && a.focus < fSetting0+settingRows {
				a.focus = fNone
			}
			return
		}
	}
}

func (a *app) save() {
	if err := a.writeSettings(); err != nil {
		a.fail(cSaveFailed, err)
		return
	}
	a.ok(cSaved, a.iniPath)
	a.probeVersion()
}

// writeSettings stores the settings in starter.ini, the one writer of that
// file: Save settings calls it, and so does a start of the game, so the game
// and the next starter read the list of mods the player just played with.
func (a *app) writeSettings() error {
	a.s.store(a.file)
	// starter.ini is written in place, its folders made and kept.
	_, err := hostfile.Commit(hostfile.Set{
		Files: []hostfile.File{{
			Path: a.iniPath, Data: a.file.Bytes(), Perm: 0o644, Staging: hostfile.Direct, Overwrite: hostfile.Replace,
		}},
		Dirs: hostfile.Parents, DirPerm: 0o755,
	})
	return err
}

// launchPlan resolves the executable and arguments for a launch or a check and
// reports why they cannot be had. mission, when not blank, is the mission the
// launch names in place of the mission setting.
func (a *app) launchPlan(check bool, mission string) (string, []string, error) {
	root, ok := a.selectedRoot()
	if !ok {
		return "", nil, captionErr(cNoBaseSelected)
	}
	if info := a.infos[root]; !info.Valid() {
		return "", nil, captionErr(cBaseNotInstall, a.s.LastBase, strings.Join(info.Missing, ", "))
	}
	rows := a.modRows()
	for _, m := range rows {
		if m.enabled && m.problem != "" {
			return "", nil, captionErr(cModUnusable, m.id, m.problem)
		}
	}
	for _, m := range rows {
		if m.enabled && m.refusal != "" {
			return "", nil, captionErr(cModUnusable, m.id, m.refusal)
		}
	}
	args, err := a.gameArgs(root, check, mission)
	if err != nil {
		return "", nil, err
	}
	exe := a.s.gameExecutable(a.selfPath, a.goos)
	if !fileExists(exe) {
		return "", nil, captionErr(cGameNotFound, exe)
	}
	return exe, args, nil
}

// gameArgs is the game's argument list for root; mission as launchPlan takes it.
func (a *app) gameArgs(root string, check bool, mission string) ([]string, error) {
	modSettings, err := a.modSettings()
	if err != nil {
		return nil, err
	}
	g := gameLaunch{Root: root, ModsDir: a.s.modsDir(a.selfDir()), Mission: mission, ModSettings: modSettings, Check: check}
	if id := a.infos[root].Base.ID(); basepkg.Nameable(id) {
		g.Base = id
	}
	return a.s.launchArgs(g)
}

// play starts the game as the settings say.
func (a *app) play() { a.start("") }

// start starts the game: on the mission when one is named, else as the
// settings say. The settings are written first, so the enabled list the game
// was started with is the one starter.ini holds; a failed write is reported
// and does not hold the game back.
func (a *app) start(mission string) {
	exe, args, err := a.launchPlan(false, mission)
	if err != nil {
		a.fail(cCannotPlay, a.errText(err))
		return
	}
	written := a.writeSettings()
	if err := a.deps.start(exe, args); err != nil {
		a.fail(cCannotStart, exe, err)
		return
	}
	a.ok(cStarted, filepath.Base(exe))
	if written != nil {
		a.fail(cSaveFailed, written)
	}
	if a.s.CloseOnPlay {
		a.closing = true
	}
}

func (a *app) check() {
	exe, args, err := a.launchPlan(true, "")
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

// mapEditorPlan resolves the map editor beside this program, its arguments
// for the selected base and the arguments of its map list check, and reports
// why they cannot be had.
func (a *app) mapEditorPlan() (exe string, start, check []string, err error) {
	root, ok := a.selectedRoot()
	if !ok {
		return "", nil, nil, captionErr(cNoBaseSelected)
	}
	if info := a.infos[root]; !info.Valid() {
		return "", nil, nil, captionErr(cBaseNotInstall, a.s.LastBase, strings.Join(info.Missing, ", "))
	}
	exe = editorExecutable(a.selfPath, a.goos)
	if !fileExists(exe) {
		return "", nil, nil, captionErr(cEditorNotFound, exe)
	}
	modsDir := a.s.modsDir(a.selfDir())
	if start, err = editorArgs(root, modsDir, false); err == nil {
		check, err = editorArgs(root, modsDir, true)
	}
	if err != nil {
		return "", nil, nil, refusalErr(err)
	}
	return exe, start, check, nil
}

// editorArgs is the map editor's arguments for the asset root and the mods
// folder its mod window publishes into, with the map list when list is set,
// built from the rows of launch.Editor.
func editorArgs(root, modsDir string, list bool) ([]string, error) {
	return launch.Editor.Build(launch.One(func(st launch.Setting) string {
		switch st.Key {
		case launch.EditorMode.Key:
			return launch.FormatBool(true)
		case launch.EditorAssets.Key:
			return root
		case launch.EditorModsDir.Key:
			return modsDir
		case launch.EditorList.Key:
			return launch.FormatBool(list)
		}
		return st.Load("")
	}))
}

// openMapEditor first runs the map editor's map list on the selected base,
// which loads the base as the window would and exits; poll starts the editor
// the way Play starts the game when the list passed, and otherwise shows the
// editor's own error output, which a window started without a console loses.
func (a *app) openMapEditor() {
	exe, args, check, err := a.mapEditorPlan()
	if err != nil {
		a.fail(cCannotOpenEditor, a.errText(err))
		return
	}
	a.ok(cCheckingEditor, filepath.Base(exe), a.s.LastBase)
	a.output = nil
	a.deps.async(func() {
		out, err := a.deps.run(exe, check)
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
func (a *app) baseLabel(info installinfo.Info) string {
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
func (a *app) baseNote(info installinfo.Info) string {
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
