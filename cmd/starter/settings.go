package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"

	"againrom/pkg/ini"
	"againrom/pkg/launch"
	"againrom/pkg/mod"
)

// Section and key names of the settings file.
const (
	secStarter = "starter"
	secBases   = "bases"
	secMods    = "mods"
	secOptions = "options"

	// secModPrefix starts the name of a mod's own section: [mod.<id>] holds the
	// values of the mod's settings.
	secModPrefix = "mod."
)

// modSetting is the text of one mod setting as the starter keeps it: the value
// the player typed or chose, which the game checks against the mod's declaration.
type modSetting struct {
	Mod, Key, Value string
}

// base is one named game root.
type base struct {
	Key  string
	Path string
}

// settings is everything the starter keeps between runs.
type settings struct {
	Againrom       string // path of the game executable; empty means beside this program
	CloseOnPlay    bool
	LastBase       string // key of the selected base
	Bases          []base
	ModsDir        string   // empty means the profile default
	Enabled        []string // mod ids in load order
	AcceptUnmarked bool     // pass -mods-accept-unmarked: load saves written without a mod mark
	ModSettings    []modSetting

	// The rows of launch.Settings, kept in [options]; optionViews binds each
	// to its field.
	Sound               string
	Volume              string // empty means the saved preference
	Movies              bool
	Video               string
	Markers             bool
	Chicken             bool
	Saves               string
	Mission             string // empty means the game's default
	Picker              bool
	Skill               string
	ShopStockMultiplier string
	Extra               string // free arguments, split like a command line
	// Windowed opens the game in a decorated window rather than over the
	// whole screen.
	Windowed bool
}

func defaultSettings() settings {
	var s settings
	for _, st := range launch.Settings {
		s.setOption(st, st.Load(""))
	}
	return s
}

// load reads settings from f; a missing or unrecognised value keeps its default.
func loadSettings(f *ini.File) settings {
	s := defaultSettings()
	get := func(sec, key string) string { v, _ := f.Get(sec, key); return v }
	s.Againrom = get(secStarter, "againrom")
	s.CloseOnPlay = launch.ParseBool(get(secStarter, "close-on-play"), false)
	s.LastBase = get(secStarter, "last-base")
	for _, k := range f.Keys(secBases) {
		if p := get(secBases, k); p != "" {
			s.Bases = append(s.Bases, base{k, p})
		}
	}
	s.ModsDir = get(secMods, "dir")
	s.Enabled = splitList(get(secMods, "enabled"))
	s.AcceptUnmarked = launch.ParseBool(get(secMods, "accept-unmarked"), false)
	for _, sec := range f.Sections() {
		id, ok := strings.CutPrefix(sec, secModPrefix)
		if !ok || id == "" {
			continue
		}
		for _, k := range f.Keys(sec) {
			v, _ := f.Get(sec, k)
			s.ModSettings = append(s.ModSettings, modSetting{id, k, v})
		}
	}
	for _, st := range launch.Settings {
		s.setOption(st, st.Load(get(secOptions, st.Key)))
	}
	return s
}

// store writes s into f, leaving every key it does not own untouched. A base
// that was removed is deleted from [bases].
func (s settings) store(f *ini.File) {
	f.Set(secStarter, "againrom", s.Againrom)
	f.Set(secStarter, "close-on-play", launch.FormatBool(s.CloseOnPlay))
	f.Set(secStarter, "last-base", s.LastBase)
	keep := map[string]bool{}
	for _, b := range s.Bases {
		keep[strings.ToLower(b.Key)] = true
		f.Set(secBases, b.Key, b.Path)
	}
	for _, k := range f.Keys(secBases) {
		if !keep[strings.ToLower(k)] {
			f.Delete(secBases, k)
		}
	}
	f.Set(secMods, "dir", s.ModsDir)
	f.Set(secMods, "enabled", strings.Join(s.Enabled, ","))
	f.Set(secMods, "accept-unmarked", launch.FormatBool(s.AcceptUnmarked))
	kept := map[string]bool{}
	for _, m := range s.ModSettings {
		kept[strings.ToLower(secModPrefix+m.Mod+"\x00"+m.Key)] = true
		f.Set(secModPrefix+m.Mod, m.Key, m.Value)
	}
	for _, sec := range f.Sections() {
		if id, ok := strings.CutPrefix(sec, secModPrefix); ok && id != "" {
			for _, k := range f.Keys(sec) {
				if !kept[strings.ToLower(sec+"\x00"+k)] {
					f.Delete(sec, k)
				}
			}
		}
	}
	for _, st := range launch.Settings {
		f.Set(secOptions, st.Key, s.option(st))
	}
}

func splitList(v string) []string {
	var out []string
	for _, p := range strings.Split(v, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// basePath returns the path of the base with key.
func (s settings) basePath(key string) (string, bool) {
	for _, b := range s.Bases {
		if strings.EqualFold(b.Key, key) {
			return b.Path, true
		}
	}
	return "", false
}

// addBase records path under a key made from its last folder name, adding a
// numeric suffix when the key is taken. An existing entry with the same path is
// reused. It returns the key.
func (s *settings) addBase(path string) string {
	for _, b := range s.Bases {
		if strings.EqualFold(b.Path, path) {
			return b.Key
		}
	}
	key := baseKey(path)
	unique := key
	for n := 2; ; n++ {
		if _, taken := s.basePath(unique); !taken {
			break
		}
		unique = key + "-" + strconv.Itoa(n)
	}
	s.Bases = append(s.Bases, base{unique, path})
	return unique
}

// baseKey turns the last folder of path into an ini key.
func baseKey(path string) string {
	name := strings.ToLower(filepath.Base(filepath.Clean(path)))
	var b strings.Builder
	for _, r := range name {
		if r < unicode.MaxASCII && (unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_') {
			b.WriteRune(r)
		} else if r == '.' || r == ' ' {
			b.WriteByte('-')
		}
	}
	if b.Len() == 0 {
		return "base"
	}
	return b.String()
}

// removeBase deletes the base with key.
func (s *settings) removeBase(key string) {
	var kept []base
	for _, b := range s.Bases {
		if !strings.EqualFold(b.Key, key) {
			kept = append(kept, b)
		}
	}
	s.Bases = kept
}

// toggleMod enables id at the end of the load order, or disables it.
func (s *settings) toggleMod(id string) {
	for i, e := range s.Enabled {
		if e == id {
			s.Enabled = append(s.Enabled[:i:i], s.Enabled[i+1:]...)
			return
		}
	}
	s.Enabled = append(s.Enabled, id)
}

// modValue returns the stored text of a mod setting, and whether there is one.
func (s settings) modValue(id, key string) (string, bool) {
	for _, m := range s.ModSettings {
		if m.Mod == id && m.Key == key {
			return m.Value, true
		}
	}
	return "", false
}

// modValuePtr returns the stored text of a mod setting for editing, adding an
// entry that holds def when there is none.
func (s *settings) modValuePtr(id, key, def string) *string {
	for i := range s.ModSettings {
		if s.ModSettings[i].Mod == id && s.ModSettings[i].Key == key {
			return &s.ModSettings[i].Value
		}
	}
	s.ModSettings = append(s.ModSettings, modSetting{id, key, def})
	return &s.ModSettings[len(s.ModSettings)-1].Value
}

// resetMod forgets the stored settings of a mod, so its defaults apply.
func (s *settings) resetMod(id string) {
	var kept []modSetting
	for _, m := range s.ModSettings {
		if m.Mod != id {
			kept = append(kept, m)
		}
	}
	s.ModSettings = kept
}

func (s settings) modEnabled(id string) bool {
	for _, e := range s.Enabled {
		if e == id {
			return true
		}
	}
	return false
}

// gameLaunch is what one start of the game takes besides the settings: Base
// is the detected profile id or empty, ModSettings mod.SettingFlag texts.
type gameLaunch struct {
	Root, Base, ModsDir string
	ModSettings         []string
	Check               bool
}

// launchArgs is the argument list that starts the game for g, built from the
// rows of launch.Table. Options left at their defaults add no argument, so the
// game's saved preferences keep applying.
func (s settings) launchArgs(g gameLaunch) ([]string, error) {
	if strings.TrimSpace(g.Root) == "" {
		return nil, captionErr(cNoBaseSelected)
	}
	args, err := launch.Build(func(st launch.Setting) []string {
		switch st.Key {
		case launch.Assets.Key:
			return []string{g.Root}
		case launch.Base.Key:
			return []string{g.Base}
		case launch.Mods.Key:
			return []string{strings.Join(s.Enabled, ",")}
		case launch.ModsDir.Key:
			return []string{g.ModsDir}
		case launch.ModSetting.Key:
			return g.ModSettings
		case launch.ModsAcceptUnmarked.Key:
			return []string{launch.FormatBool(s.AcceptUnmarked)}
		case launch.Check.Key:
			return []string{launch.FormatBool(g.Check)}
		case launch.Version.Key:
			return nil
		}
		return []string{s.option(st)}
	})
	if err != nil {
		return nil, refusalErr(err)
	}
	return args, nil
}

// defaultINIPath is the settings file beside the starter executable.
func defaultINIPath(selfDir string) string {
	return filepath.Join(selfDir, "starter.ini")
}

// modsDir is the directory mods are read from: the configured one, else the
// mods folder beside the starter executable.
func (s settings) modsDir(selfDir string) string {
	if d := strings.TrimSpace(s.ModsDir); d != "" {
		return d
	}
	return mod.DefaultDir(selfDir)
}

// gameExecutable is the program Play starts: the configured path, else the
// game beside this program.
func (s settings) gameExecutable(self, goos string) string {
	if p := strings.TrimSpace(s.Againrom); p != "" {
		return p
	}
	return launch.Beside(self, "againrom", goos)
}

// editorExecutable is the map editor the starter opens: the one beside this
// program.
func editorExecutable(self, goos string) string {
	return launch.Beside(self, "mapedit", goos)
}

func fileExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}
