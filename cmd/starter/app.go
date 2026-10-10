package main

import (
	"errors"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	basepkg "againrom/pkg/base"
	"againrom/pkg/game"
	"againrom/pkg/ini"
	"againrom/pkg/mod"
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
	async func(f func())
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
	scroll    [2]int // base list, mod list
	status    string
	statusBad bool
	output    []string

	starterVersion string
	gameVersion    string // what "againrom -version" printed, or "unknown"

	results chan result
	dirty   bool
	closing bool
}

type result struct {
	kind   string // "check" or "version"
	output string
	err    error
}

// newApp reads the settings file at iniPath (a missing file is the defaults)
// and looks at every base and the mods directory.
func newApp(iniPath, selfPath, goos, starterVersion string, d deps) *app {
	a := &app{
		deps: d, selfPath: selfPath, goos: goos, iniPath: iniPath,
		infos:          map[string]game.InstallInfo{},
		starterVersion: starterVersion,
		gameVersion:    "unknown",
		results:        make(chan result, 8),
		dirty:          true,
	}
	data, err := os.ReadFile(iniPath)
	switch {
	case err == nil:
	case errors.Is(err, os.ErrNotExist):
		data = nil
	default:
		a.fail("cannot read settings %s: %v", iniPath, err)
	}
	a.file, a.problems = ini.Parse(data)
	a.s = loadSettings(a.file)
	if len(a.problems) > 0 {
		p := a.problems[0]
		a.fail("settings %s: %s (%d such line(s); the lines are kept as they are)", iniPath, p, len(a.problems))
	}
	if a.status == "" {
		if err := probeWritable(filepath.Dir(iniPath)); err != nil {
			a.fail("settings folder is not writable, Save settings will fail: %v", err)
		}
	}
	for _, b := range a.s.Bases {
		a.inspectBase(b.Path)
	}
	if _, ok := a.s.basePath(a.s.LastBase); !ok && len(a.s.Bases) > 0 {
		a.s.LastBase = a.s.Bases[0].Key
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

func (a *app) ok(format string, args ...any) {
	a.status, a.statusBad, a.dirty = fmt.Sprintf(format, args...), false, true
}

func (a *app) fail(format string, args ...any) {
	a.status, a.statusBad, a.dirty = fmt.Sprintf(format, args...), true, true
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
		d.err = fmt.Errorf("mod %s has no usable folder", id)
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

// lang is the language of the selected base, as a label language: "ru" for a
// Russian install and "en" otherwise, including when the base is unknown.
func (a *app) lang() string {
	if root, ok := a.selectedRoot(); ok && a.infos[root].Language == "russian" {
		return "ru"
	}
	return "en"
}

// settingText is the text a setting shows: the stored value, else its default.
func (a *app) settingText(id string, st mod.Setting) string {
	if v, ok := a.s.modValue(id, st.Key); ok {
		return v
	}
	return st.Default.String()
}

// modArgs are the -mod-setting arguments for the enabled mods, in load order.
// A stored value for a key the mod no longer declares is left out. A value the
// mod's declaration refuses is an error naming the mod and the setting.
func (a *app) modArgs() ([]string, error) {
	var args []string
	for _, id := range a.s.Enabled {
		decl, err := a.settingsOf(id)
		if err != nil {
			return nil, fmt.Errorf("mod %s: %v", id, err)
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
				args = append(args, "-mod-setting", id+"."+st.Key+"="+strings.TrimSpace(v))
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
	case fVolume:
		return &a.s.Volume
	case fSaves:
		return &a.s.Saves
	case fMission:
		return &a.s.Mission
	case fSkill:
		return &a.s.Skill
	case fExtra:
		return &a.s.Extra
	case fAgainrom:
		return &a.s.Againrom
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
			r = modRow{id: id, label: id + ": folder not found in the mods directory", problem: "folder not found"}
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
	heading := func(x, y int, s string) { it = append(it, item{kind: iHeading, rect: rect(x, y, 0, 16), text: s}) }
	button := func(r image.Rectangle, s string, act action) {
		it = append(it, item{kind: iButton, rect: r, text: s, act: act})
	}
	field := func(r image.Rectangle, id fieldID) {
		it = append(it, item{kind: iField, rect: r, text: *a.field(id), focused: a.focus == id, act: action{aFocus, int(id)}})
	}

	title := "Starter " + a.starterVersion + " (" + shortRevision() + ")"
	text(10, 6, title, toneNormal)
	text(330, 6, "launches againrom "+a.gameVersion, toneDim)
	text(10, 24, "settings: "+a.iniPath, toneDim)

	// Base games.
	heading(10, 48, "Base game")
	bases := a.s.Bases
	top := clampScroll(&a.scroll[0], len(bases), baseRows)
	for k := 0; k < baseRows && top+k < len(bases); k++ {
		b := bases[top+k]
		info := a.infos[b.Path]
		r := item{kind: iBaseRow, rect: rect(10, 66+18*k, 365, 18), checked: strings.EqualFold(b.Key, a.s.LastBase), act: action{aSelectBase, top + k}}
		if info.Valid() {
			r.tone = toneGood
			r.text = b.Key + "  " + b.Path + "  OK " + baseLabel(info)
		} else {
			r.tone = toneBad
			r.text = b.Key + "  " + b.Path + "  not an install, missing " + strings.Join(info.Missing, ", ")
		}
		it = append(it, r)
	}
	if len(bases) == 0 {
		textW(10, 66, 365, "No base game yet. Type or paste a folder below.", toneDim)
	}
	text(10, 162, "Add path:", toneDim)
	field(rect(76, 158, 170, 22), fAddBase)
	button(rect(252, 158, 50, 22), "Add", action{kind: aAddBase})
	button(rect(308, 158, 67, 22), "Remove", action{kind: aRemoveBase})

	// Mods.
	heading(10, 190, "Mods")
	text(10, 214, "Folder:", toneDim)
	field(rect(76, 210, 224, 22), fModsDir)
	button(rect(306, 210, 69, 22), "Rescan", action{kind: aRescan})
	if a.s.ModsDir == "" {
		textW(76, 234, 299, "(empty: "+mod.DefaultDir(a.selfDir())+")", toneDim)
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
		it = append(it, item{kind: iModRow, rect: rect(10, 250+18*k, 365, 18), text: label, tone: tn, checked: m.enabled, act: action{aToggleMod, top + k}})
	}
	if a.scanErr != nil {
		textW(10, 382, 365, "cannot read the mods folder: "+a.scanErr.Error(), toneBad)
	} else if len(rows) == 0 {
		textW(10, 250, 365, "No mods found.", toneDim)
	}
	textW(10, 362, 365, "Tick to enable. Load order is the order shown.", toneDim)

	// Parameters.
	heading(395, 48, "Parameters")
	row := func(i int, label string) int { y := 66 + 24*i; textW(395, y+3, 135, label, toneNormal); return y }
	ctl := func(y int) image.Rectangle { return rect(535, y, 215, 22) }
	y := row(0, "Sound")
	button(ctl(y), a.s.Sound, action{kind: aSound})
	y = row(1, "Volume 0-100")
	field(ctl(y), fVolume)
	y = row(2, "Movies")
	it = append(it, item{kind: iToggle, rect: ctl(y), text: onOff(a.s.Movies), checked: a.s.Movies, act: action{kind: aMovies}})
	y = row(3, "Mission movies")
	button(ctl(y), a.s.Video, action{kind: aVideo})
	y = row(4, "Markers")
	it = append(it, item{kind: iToggle, rect: ctl(y), text: onOff(a.s.Markers), checked: a.s.Markers, act: action{kind: aMarkers}})
	y = row(5, "Saves folder")
	field(ctl(y), fSaves)
	heading(395, 214, "Advanced")
	y = row(7, "Mission number")
	field(ctl(y), fMission)
	y = row(8, "Map picker")
	it = append(it, item{kind: iToggle, rect: ctl(y), text: onOff(a.s.Picker), checked: a.s.Picker, act: action{kind: aPicker}})
	y = row(9, "#Chicken each mission")
	it = append(it, item{kind: iToggle, rect: ctl(y), text: onOff(a.s.Chicken), checked: a.s.Chicken, act: action{kind: aChicken}})
	y = row(10, "Skill")
	field(ctl(y), fSkill)
	y = row(11, "Extra arguments")
	field(ctl(y), fExtra)
	heading(395, 356, "Starter")
	y = row(13, "Close after Play")
	it = append(it, item{kind: iToggle, rect: ctl(y), text: onOff(a.s.CloseOnPlay), checked: a.s.CloseOnPlay, act: action{kind: aCloseOnPlay}})
	y = row(14, "Game program")
	field(ctl(y), fAgainrom)
	y = row(15, "Load unmarked saves")
	it = append(it, item{kind: iToggle, rect: ctl(y), text: onOff(a.s.AcceptUnmarked), checked: a.s.AcceptUnmarked, act: action{kind: aAcceptUnmarked}})

	// Mod settings.
	heading(10, panelTop, "Mod settings")
	if id, ok := a.panelMod(); !ok {
		textW(10, panelTop+22, 365, "Select a mod in the list to edit its settings.", toneDim)
	} else {
		state, tn := "not enabled", toneDim
		if a.s.modEnabled(id) {
			state, tn = "enabled", toneNormal
		}
		textW(120, panelTop, 170, id+" ("+state+")", tn)
		decl, err := a.settingsOf(id)
		switch {
		case err != nil:
			textW(10, panelTop+22, 365, "settings.toml: "+err.Error(), toneBad)
		case len(decl) == 0:
			textW(10, panelTop+22, 365, "This mod has no settings.", toneDim)
		default:
			button(rect(300, panelTop-3, 75, 22), "Defaults", action{kind: aSettingReset})
			for i := 0; i < settingRows && i < len(decl); i++ {
				st, y := decl[i], panelTop+24+24*i
				label := st.Label(a.lang())
				if st.Kind == mod.KindInt {
					label = fmt.Sprintf("%s (%d..%d)", label, st.Min, st.Max)
				}
				textW(10, y+3, 200, label, toneNormal)
				ctl, val := rect(215, y, 160, 22), a.settingText(id, st)
				switch st.Kind {
				case mod.KindInt:
					it = append(it, item{kind: iField, rect: ctl, text: val, focused: a.focus == fSetting0+fieldID(i), act: action{aFocus, int(fSetting0) + i}})
				case mod.KindBool:
					v, perr := st.Parse(val)
					it = append(it, item{kind: iToggle, rect: ctl, text: onOff(perr == nil && v.Bool), checked: perr == nil && v.Bool, act: action{aSettingCycle, i}})
				default:
					button(ctl, val, action{aSettingCycle, i})
				}
			}
			if len(decl) > settingRows {
				textW(10, panelTop+24+24*settingRows, 365, fmt.Sprintf("%d more setting(s) are not shown.", len(decl)-settingRows), toneDim)
			}
		}
	}

	// Actions.
	by := 410 + panelH
	button(rect(10, by, 100, 28), "Play", action{kind: aPlay})
	button(rect(118, by, 100, 28), "Check", action{kind: aCheck})
	button(rect(226, by, 140, 28), "Save settings", action{kind: aSave})
	if cmd, err := a.command(); err != nil {
		text(10, by+36, err.Error(), toneBad)
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

func onOff(b bool) string {
	if b {
		return "on"
	}
	return "off"
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
	modArgs, err := a.modArgs()
	if err != nil {
		return "", err
	}
	args, err := a.s.launchArgs(root, a.s.modsDir(a.selfDir()), modArgs...)
	if err != nil {
		return "", err
	}
	args = withBase(args, a.infos[root])
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
	items := a.layout()
	prev := a.focus
	a.focus = fNone
	for i := len(items) - 1; i >= 0; i-- {
		if it := items[i]; it.act.kind != aNone && p.In(it.rect) {
			a.apply(it.act)
			break
		}
	}
	if prev != a.focus {
		a.dirty = true
	}
}

func (a *app) apply(act action) {
	a.dirty = true
	switch act.kind {
	case aFocus:
		a.focus = fieldID(act.i)
	case aSelectBase:
		if act.i < len(a.s.Bases) {
			b := a.s.Bases[act.i]
			a.s.LastBase = b.Key
			if info := a.infos[b.Path]; info.Valid() {
				a.ok("%s", baseNote(info))
			}
		}
	case aToggleMod:
		rows := a.modRows()
		if act.i < len(rows) {
			a.selMod = rows[act.i].id
			if r := rows[act.i]; r.enabled || r.problem == "" {
				a.s.toggleMod(r.id)
			} else {
				a.fail("mod %s cannot be enabled: %s", r.id, r.problem)
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
		a.ok("%d mod folder(s) found in %s", len(a.scan), a.s.modsDir(a.selfDir()))
	case aSound:
		a.s.Sound = map[string]string{soundDefault: soundOn, soundOn: soundOff, soundOff: soundDefault}[a.s.Sound]
		if a.s.Sound == "" {
			a.s.Sound = soundDefault
		}
	case aVideo:
		a.s.Video = map[string]string{videoNormal: video4x, video4x: video8x, video8x: videoNormal}[a.s.Video]
		if a.s.Video == "" {
			a.s.Video = videoNormal
		}
	case aMovies:
		a.s.Movies = !a.s.Movies
	case aMarkers:
		a.s.Markers = !a.s.Markers
	case aChicken:
		a.s.Chicken = !a.s.Chicken
	case aPicker:
		a.s.Picker = !a.s.Picker
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
	}
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
		next = formatBool(perr != nil || !v.Bool)
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
		a.fail("type or paste the folder of a game install first")
		return
	}
	if st, err := os.Stat(path); err != nil || !st.IsDir() {
		a.fail("%s is not a folder", path)
		return
	}
	key := a.s.addBase(path)
	a.inspectBase(path)
	a.s.LastBase = key
	a.addText = ""
	if info := a.infos[path]; info.Valid() {
		a.ok("added %s: %s", key, baseNote(info))
	} else {
		a.fail("added %s, but it is not an install: missing %s", key, strings.Join(info.Missing, ", "))
	}
}

// types appends typed text to the focused field.
func (a *app) typed(s string) {
	f := a.field(a.focus)
	if f == nil {
		return
	}
	for _, r := range s {
		if !unicode.IsControl(r) {
			*f += string(r)
			a.dirty = true
		}
	}
}

// backspace removes the focused field's last character.
func (a *app) backspace() {
	if f := a.field(a.focus); f != nil && *f != "" {
		r := []rune(*f)
		*f = string(r[:len(r)-1])
		a.dirty = true
	}
}

// enter confirms the focused field.
func (a *app) enter() {
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

func (a *app) escape() { a.focus, a.dirty = fNone, true }

// tab moves focus to the next field.
func (a *app) tab() {
	for n := 0; n < int(fieldCount); n++ {
		a.focus = a.focus%(fieldCount-1) + 1
		if a.field(a.focus) != nil {
			break
		}
	}
	a.dirty = true
}

// pasteText pastes the first line of the clipboard into the focused field.
func (a *app) pasteText() {
	if a.focus == fNone || a.deps.paste == nil {
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
	switch {
	case p.In(image.Rect(10, 66, 375, 156)):
		a.scroll[0] += lines
	case p.In(image.Rect(10, 250, 375, 360)):
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
		a.fail("cannot save settings: %v", err)
		return
	}
	a.ok("settings saved to %s", a.iniPath)
	a.probeVersion()
}

// launchPlan resolves the executable and arguments for a launch or a check and
// reports why they cannot be had.
func (a *app) launchPlan(extra ...string) (string, []string, error) {
	root, ok := a.selectedRoot()
	if !ok {
		return "", nil, errors.New("no base game selected")
	}
	if info := a.infos[root]; !info.Valid() {
		return "", nil, fmt.Errorf("base %s is not an install: missing %s", a.s.LastBase, strings.Join(info.Missing, ", "))
	}
	for _, m := range a.modRows() {
		if m.enabled && m.problem != "" {
			return "", nil, fmt.Errorf("mod %s is enabled but cannot be used: %s", m.id, m.problem)
		}
	}
	modArgs, err := a.modArgs()
	if err != nil {
		return "", nil, err
	}
	args, err := a.s.launchArgs(root, a.s.modsDir(a.selfDir()), modArgs...)
	if err != nil {
		return "", nil, err
	}
	args = withBase(args, a.infos[root])
	exe := a.s.gameExecutable(a.selfPath, a.goos)
	if !fileExists(exe) {
		return "", nil, fmt.Errorf("game program not found: %s", exe)
	}
	return exe, append(args, extra...), nil
}

func (a *app) play() {
	exe, args, err := a.launchPlan()
	if err != nil {
		a.fail("cannot play: %v", err)
		return
	}
	if err := a.deps.start(exe, args); err != nil {
		a.fail("cannot start %s: %v", exe, err)
		return
	}
	a.ok("started %s", filepath.Base(exe))
	if a.s.CloseOnPlay {
		a.closing = true
	}
}

func (a *app) check() {
	exe, args, err := a.launchPlan("-check")
	if err != nil {
		a.fail("cannot check: %v", err)
		return
	}
	a.ok("running %s -check ...", filepath.Base(exe))
	a.output = nil
	a.deps.async(func() {
		out, err := a.deps.run(exe, args)
		a.results <- result{"check", out, err}
	})
}

// probeVersion asks the game program for its version line.
func (a *app) probeVersion() {
	exe := a.s.gameExecutable(a.selfPath, a.goos)
	if !fileExists(exe) {
		a.setGameVersion("")
		return
	}
	a.deps.async(func() {
		out, err := a.deps.run(exe, []string{"-version"})
		a.results <- result{"version", out, err}
	})
}

func (a *app) setGameVersion(line string) {
	v := "unknown"
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
				a.output = strings.Split(strings.TrimRight(strings.ReplaceAll(r.output, "\r\n", "\n"), "\n"), "\n")
				if r.err != nil {
					a.fail("check failed: %v", r.err)
				} else {
					a.ok("check passed")
				}
				a.dirty = true
			}
		default:
			return
		}
	}
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
func baseLabel(info game.InstallInfo) string {
	m := info.Base
	if !m.Profile.Known() {
		return info.Language
	}
	label := m.Profile.ID
	if !m.Exact {
		label += " (unrecognised build)"
	}
	if hasLimits(m.Profile) {
		label += ", has limits"
	}
	return label
}

func hasLimits(p basepkg.Profile) bool {
	l := p.Limits
	return l.NoCharacterGeneration || l.OriginalSaveRefusal != "" || len(l.Notes) != 0
}

// baseNote is the status line a selected or added base gets: the profile and
// each limit the profile states.
func baseNote(info game.InstallInfo) string {
	m := info.Base
	if !m.Profile.Known() {
		return info.Language + " install"
	}
	parts := []string{m.String()}
	l := m.Profile.Limits
	if l.NoCharacterGeneration {
		parts = append(parts, fmt.Sprintf("no character generation, new game opens mission %d", m.Profile.Mission()))
	}
	if l.OriginalSaveRefusal != "" {
		parts = append(parts, "original saves refused")
	}
	parts = append(parts, l.Notes...)
	return strings.Join(parts, "; ")
}

// withBase inserts "-base <profile id>" after the leading "-assets <root>" pair
// when the install was detected as a known profile, so the game refuses to run
// on a root that is no longer that base. An install with no named profile gets
// no argument.
func withBase(args []string, info game.InstallInfo) []string {
	id := info.Base.ID()
	if !basepkg.Nameable(id) || len(args) < 2 || args[0] != "-assets" {
		return args
	}
	out := append([]string{}, args[:2]...)
	out = append(out, "-base", id)
	return append(out, args[2:]...)
}
