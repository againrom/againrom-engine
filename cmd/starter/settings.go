package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"

	"againrom/pkg/ini"
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

// Three-state and choice values kept in the settings file.
const (
	soundDefault = "default"
	soundOn      = "on"
	soundOff     = "off"
	videoNormal  = "normal"
	video4x      = "4x"
	video8x      = "8x"
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

	Sound   string // soundDefault, soundOn or soundOff
	Volume  string // empty means the saved preference; else 0-100
	Movies  bool
	Video   string // videoNormal, video4x or video8x
	Markers bool
	Saves   string
	Mission string // empty means the game's default
	Picker  bool
	Skill   string
	Extra   string // free arguments, split like a command line
}

func defaultSettings() settings {
	return settings{Sound: soundDefault, Movies: true, Video: videoNormal}
}

// load reads settings from f; a missing or unrecognised value keeps its default.
func loadSettings(f *ini.File) settings {
	s := defaultSettings()
	get := func(sec, key string) string { v, _ := f.Get(sec, key); return v }
	s.Againrom = get(secStarter, "againrom")
	s.CloseOnPlay = parseBool(get(secStarter, "close-on-play"), false)
	s.LastBase = get(secStarter, "last-base")
	for _, k := range f.Keys(secBases) {
		if p := get(secBases, k); p != "" {
			s.Bases = append(s.Bases, base{k, p})
		}
	}
	s.ModsDir = get(secMods, "dir")
	s.Enabled = splitList(get(secMods, "enabled"))
	s.AcceptUnmarked = parseBool(get(secMods, "accept-unmarked"), false)
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
	switch v := strings.ToLower(get(secOptions, "sound")); v {
	case soundOn, soundOff:
		s.Sound = v
	}
	s.Volume = get(secOptions, "volume")
	s.Movies = parseBool(get(secOptions, "movies"), true)
	switch v := strings.ToLower(get(secOptions, "video")); v {
	case video4x, video8x:
		s.Video = v
	}
	s.Markers = parseBool(get(secOptions, "markers"), false)
	s.Saves = get(secOptions, "saves")
	s.Mission = get(secOptions, "mission")
	s.Picker = parseBool(get(secOptions, "picker"), false)
	s.Skill = get(secOptions, "skill")
	s.Extra = get(secOptions, "extra")
	return s
}

// store writes s into f, leaving every key it does not own untouched. A base
// that was removed is deleted from [bases].
func (s settings) store(f *ini.File) {
	f.Set(secStarter, "againrom", s.Againrom)
	f.Set(secStarter, "close-on-play", formatBool(s.CloseOnPlay))
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
	f.Set(secMods, "accept-unmarked", formatBool(s.AcceptUnmarked))
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
	f.Set(secOptions, "sound", s.Sound)
	f.Set(secOptions, "volume", s.Volume)
	f.Set(secOptions, "movies", formatBool(s.Movies))
	f.Set(secOptions, "video", s.Video)
	f.Set(secOptions, "markers", formatBool(s.Markers))
	f.Set(secOptions, "saves", s.Saves)
	f.Set(secOptions, "mission", s.Mission)
	f.Set(secOptions, "picker", formatBool(s.Picker))
	f.Set(secOptions, "skill", s.Skill)
	f.Set(secOptions, "extra", s.Extra)
}

func parseBool(v string, def bool) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	}
	return def
}

func formatBool(b bool) string {
	if b {
		return "true"
	}
	return "false"
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

// launchArgs is the argument list that starts the game for root. modsDir is the
// directory -mods-dir names; it is passed only when mods are enabled, and
// modArgs (the -mod-setting arguments) follow it. Options left at their
// defaults add no argument, so the game's saved preferences keep applying.
func (s settings) launchArgs(root, modsDir string, modArgs ...string) ([]string, error) {
	if strings.TrimSpace(root) == "" {
		return nil, errors.New("no base game selected")
	}
	args := []string{"-assets", root}
	if len(s.Enabled) > 0 {
		args = append(args, "-mods", strings.Join(s.Enabled, ","), "-mods-dir", modsDir)
		args = append(args, modArgs...)
		if s.AcceptUnmarked {
			args = append(args, "-mods-accept-unmarked")
		}
	}
	switch s.Sound {
	case soundOn:
		args = append(args, "-sound=true")
	case soundOff:
		args = append(args, "-sound=false")
	}
	if v := strings.TrimSpace(s.Volume); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 || n > 100 {
			return nil, fmt.Errorf("volume %q is not a number from 0 to 100", v)
		}
		args = append(args, "-volume", strconv.Itoa(n))
	}
	if !s.Movies {
		args = append(args, "-movies=false")
	}
	switch s.Video {
	case video4x:
		args = append(args, "-4x")
	case video8x:
		args = append(args, "-8x")
	}
	if s.Markers {
		args = append(args, "-markers")
	}
	if v := strings.TrimSpace(s.Saves); v != "" {
		args = append(args, "-saves", v)
	}
	mission := strings.TrimSpace(s.Mission)
	if s.Picker && mission != "" {
		return nil, errors.New("picker and mission cannot both be set")
	}
	if mission != "" {
		n, err := strconv.Atoi(mission)
		if err != nil || n < 1 {
			return nil, fmt.Errorf("mission %q is not a positive number", mission)
		}
		args = append(args, "-mission", strconv.Itoa(n))
	}
	if s.Picker {
		args = append(args, "-picker")
	}
	if v := strings.TrimSpace(s.Skill); v != "" {
		args = append(args, "-skill", v)
	}
	extra, err := splitArgs(s.Extra)
	if err != nil {
		return nil, err
	}
	return append(args, extra...), nil
}

// splitArgs splits a command-line fragment on blanks. Double quotes group
// words and are removed; there are no escapes.
func splitArgs(line string) ([]string, error) {
	var args []string
	var cur strings.Builder
	inQuote, have := false, false
	for _, r := range line {
		switch {
		case r == '"':
			inQuote, have = !inQuote, true
		case (r == ' ' || r == '\t') && !inQuote:
			if have {
				args = append(args, cur.String())
				cur.Reset()
				have = false
			}
		default:
			cur.WriteRune(r)
			have = true
		}
	}
	if inQuote {
		return nil, errors.New("extra arguments have an unmatched quote")
	}
	if have {
		args = append(args, cur.String())
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
	name := "againrom"
	if goos == "windows" {
		name += ".exe"
	}
	return filepath.Join(filepath.Dir(self), name)
}

func fileExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}
