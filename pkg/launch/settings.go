package launch

import (
	"fmt"
	"strconv"
	"strings"
)

// Kind is the shape of a launch setting: how the starter keeps its value and
// which arguments the value passes to the game.
type Kind uint8

const (
	// Switch is "true" or "false". It passes its flag only when the value
	// differs from Default: -flag when on, -flag=false when off.
	Switch Kind = iota
	// Override is one of Options. The first passes nothing; every other sets
	// the game's bool flag to its On.
	Override
	// Choice is one of Options. The first passes nothing; every other passes
	// its own bool flag.
	Choice
	// Number is a decimal whole number from Min to Max, or from Min up when Max
	// is 0. A blank value means Default; the Default value passes nothing.
	Number
	// Text passes its flag with the value when the value is not blank.
	Text
	// Arguments are free arguments, split by Split.
	Arguments
	// Verbatim passes its flag with the value as given unless it is empty.
	Verbatim
)

// Read is how the game reads a Number flag.
type Read uint8

const (
	// ReadInt takes the flag package's integer syntax; Bounds refuses a value
	// outside Min..Max once every flag is parsed.
	ReadInt Read = iota
	// ReadWhole takes what Whole takes and refuses while the flag is parsed.
	ReadWhole
	// ReadAtStart takes the flag package's integer syntax; the game refuses a
	// number that names nothing when it starts.
	ReadAtStart
)

// Option is one value of an Override or Choice setting.
type Option struct {
	Value string // the value's text in the settings file
	Flag  string // Choice: the game flag the value passes
	On    bool   // Override: what the value sets the game flag to
	Usage string // Choice: the help of the value's flag
}

// Setting is one row of the launch table: a value the starter passes to the
// game and the game reads. The starter's captions for a row are keyed by Key.
type Setting struct {
	Key     string // key in the settings file
	Kind    Kind
	Usage   string // the flag's help; Help fills its %s
	Repeat  bool   // any number of values, each passed alone
	Needs   string // the key of a row that, blank, leaves this one out
	Default string // value of a missing or unrecognised key
	Options []Option
	// Min and Max bound a Number; Read is how the game reads it.
	Min, Max int
	Read     Read
	// GameDefault is the game flag's value when the flag is not given; empty
	// means Default.
	GameDefault string
	Flag        string // the game flag; a Choice's options name their own
	// Excludes is the key of a Switch that may not be on while this setting
	// is not blank.
	Excludes string
}

// The upper bounds of the bounded Number settings.
const (
	MaxVolume              = 100
	MaxShopStockMultiplier = 1000
)

// The values of the Override and Choice rows.
const (
	SoundDefault = "default"
	SoundOn      = "on"
	SoundOff     = "off"
	VideoNormal  = "normal"
	Video4x      = "4x"
	Video8x      = "8x"
)

// The rows not kept in [options].
var (
	Assets = Setting{Key: "assets", Kind: Verbatim, Flag: "assets",
		Usage: "path to the game asset root (overrides AGAINROM_ASSETS, which overrides the install found at the working directory or beside this binary)"}
	Base = Setting{Key: "base", Kind: Verbatim, Flag: "base",
		Usage: "refuse to run unless the asset root is detected as this base profile (one of: %s); -check names the detected profile either way"}
	Mods = Setting{Key: "mods", Kind: Verbatim, Flag: "mods",
		Usage: "comma-separated ids of the mods to load; each is a folder with a mod.toml or a packed file mod-<id>.res in the mods directory (the load order follows requires and load-after, not this list)"}
	ModsDir = Setting{Key: "mods-dir", Kind: Verbatim, Flag: "mods-dir", Needs: "mods",
		Usage: "directory holding the mod folders and packed mods (default: mods beside this binary)"}
	ModSetting = Setting{Key: "mod-setting", Kind: Verbatim, Repeat: true, Flag: "mod-setting", Needs: "mods",
		Usage: "a mod setting as <mod>.<key>=<value>; repeat the flag for several (a setting not given keeps the mod's default)"}
	ModsAcceptUnmarked = Setting{Key: "mods-accept-unmarked", Kind: Switch, Default: "false", Flag: "mods-accept-unmarked", Needs: "mods",
		Usage: "let a saved game that carries no mod mark start under the active mods (without it such a save is refused)"}
	Check = Setting{Key: "check", Kind: Switch, Default: "false", Flag: "check",
		Usage: "load the assets and the map list, print a summary, and exit without a window"}
	Version = Setting{Key: "version", Kind: Switch, Default: "false", Flag: "version",
		Usage: "print the program name, version and source revision, and exit"}
)

// The rows kept in [options].
var (
	Sound = Setting{Key: "sound", Kind: Override, Default: SoundDefault, GameDefault: "true", Flag: "sound",
		Usage:   "play sound and music (overrides the saved preference; on by default)",
		Options: []Option{{Value: SoundDefault}, {Value: SoundOn, On: true}, {Value: SoundOff}}}
	Volume = Setting{Key: "volume", Kind: Number, Min: 0, Max: MaxVolume, Read: ReadInt, GameDefault: strconv.Itoa(MaxVolume),
		Flag: "volume", Usage: fmt.Sprintf("sound and music master volume, %d-%d (overrides the saved preference)", 0, MaxVolume)}
	Movies = Setting{Key: "movies", Kind: Switch, Default: "true", Flag: "movies",
		Usage: "play installed startup, new-game and mission movies"}
	Video = Setting{Key: "video", Kind: Choice, Default: VideoNormal,
		Options: []Option{{Value: VideoNormal}, {Value: Video4x, Flag: "4x", Usage: "select VIDEO4 mission movies (overrides -8x)"},
			{Value: Video8x, Flag: "8x", Usage: "select VIDEO8 mission movies"}}}
	Markers = Setting{Key: "markers", Kind: Switch, Default: "false", Flag: "markers",
		Usage: "diagnostic: draw a marker on each placed structure, unit and static object (off by default; -markers turns all three on)"}
	Chicken = Setting{Key: "chicken", Kind: Switch, Default: "false", Flag: "chicken",
		Usage: "set the ROM1 #Chicken state at every mission start"}
	Window = Setting{Key: "window", Kind: Switch, Default: "false", Flag: "window",
		Usage: "open in a decorated window instead of over the whole screen; Alt+Enter switches between the two"}
	Saves = Setting{Key: "saves", Kind: Text, Flag: "saves",
		Usage: "directory for saved games (default: saves/ beside the binary; inside an install, Againrom/saves or the user configuration profile)"}
	Mission = Setting{Key: "mission", Kind: Number, Min: 1, Read: ReadAtStart, Flag: "mission", Excludes: "picker",
		Usage: "override the startup campaign mission (default: the first mission of the install's campaign registry)"}
	// -picker makes NEW GAME open the picker instead of generation; a normal
	// launch never reaches the picker.
	Picker = Setting{Key: "picker", Kind: Switch, Default: "false", Flag: "picker",
		Usage: "debug: make NEW GAME open the former map picker (campaign and loose maps) instead of character generation"}
	Skill = Setting{Key: "skill", Kind: Text, Flag: "skill",
		Usage: "train the party's hero in this skill instead of the shipped default (one of: %s)"}
	ShopStockMultiplier = Setting{Key: "shop-stock-multiplier", Kind: Number, Default: "1", Min: 1, Max: MaxShopStockMultiplier,
		Read: ReadWhole, Flag: "shop-stock-multiplier",
		Usage: fmt.Sprintf("roll every ROM1 shop shelf a whole number of times from %d to %d", 1, MaxShopStockMultiplier)}
	Extra = Setting{Key: "extra", Kind: Arguments}
)

// Settings are the rows kept in [options], in the order of the file.
var Settings = []Setting{Sound, Volume, Movies, Video, Markers, Chicken, Window, Saves, Mission, Picker, Skill, ShopStockMultiplier, Extra}

// Table is every row, in the order the game's arguments pass them.
var Table = append(append([]Setting{Assets, Base, Mods, ModsDir, ModSetting, ModsAcceptUnmarked}, Settings...), Check, Version)

// ByKey returns the row with key.
func ByKey(key string) (Setting, bool) {
	for _, s := range Table {
		if s.Key == key {
			return s, true
		}
	}
	return Setting{}, false
}

// InBounds reports whether n lies from Min to Max.
func (s Setting) InBounds(n int) bool { return n >= s.Min && (s.Max == 0 || n <= s.Max) }

// Whole reads a decimal whole number within the bounds.
func (s Setting) Whole(v string) (int, error) {
	n, err := strconv.Atoi(v)
	if err != nil || !s.InBounds(n) {
		if s.Max == 0 {
			return 0, fmt.Errorf("-%s must be a whole number from %d", s.Flag, s.Min)
		}
		return 0, fmt.Errorf("-%s must be a whole number from %d to %d", s.Flag, s.Min, s.Max)
	}
	return n, nil
}

// Bounds refuses a ReadInt value outside the bounds.
func (s Setting) Bounds(n int) error {
	if !s.InBounds(n) {
		return fmt.Errorf("-%s must be between %d and %d", s.Flag, s.Min, s.Max)
	}
	return nil
}

// Help is the help text with list, joined by commas, in place of its %s.
func (s Setting) Help(list ...string) string {
	if len(list) == 0 {
		return s.Usage
	}
	return fmt.Sprintf(s.Usage, strings.Join(list, ", "))
}

// GameBool is a Switch's or Override's game flag value when not given.
func (s Setting) GameBool() bool { return s.gameDefault() == "true" }

// GameInt is a Number's game flag value when not given.
func (s Setting) GameInt() int {
	n, _ := strconv.Atoi(s.gameDefault())
	return n
}

func (s Setting) gameDefault() string {
	if s.GameDefault != "" {
		return s.GameDefault
	}
	return s.Default
}

// Load is the value the starter keeps for the settings file's text raw.
func (s Setting) Load(raw string) string {
	switch s.Kind {
	case Switch:
		return FormatBool(ParseBool(raw, s.Default == "true"))
	case Override, Choice:
		if _, ok := s.Index(strings.ToLower(raw)); ok {
			return strings.ToLower(raw)
		}
		return s.Default
	case Number:
		if raw == "" {
			return s.Default
		}
	}
	return raw
}

// Next is the Override or Choice value after v, the first after the last.
func (s Setting) Next(v string) string {
	i, ok := s.Index(v)
	if !ok {
		return s.Default
	}
	return s.Options[(i+1)%len(s.Options)].Value
}

// Index is the position of the value v in Options.
func (s Setting) Index(v string) (int, bool) {
	for i, o := range s.Options {
		if o.Value == v {
			return i, true
		}
	}
	return 0, false
}

// Refusal is a value the game's arguments cannot carry.
type Refusal struct {
	Setting Setting
	Value   string
	// Clash is set when the value is not blank while Setting.Excludes is on.
	Clash bool
}

func (r *Refusal) Error() string {
	if r.Clash {
		return fmt.Sprintf("%s and %s cannot both be set", r.Setting.Excludes, r.Setting.Key)
	}
	return fmt.Sprintf("%s %q is refused", r.Setting.Key, r.Value)
}

// Args is the game's arguments for the value v.
func (s Setting) Args(v string) ([]string, error) {
	switch s.Kind {
	case Switch:
		on := ParseBool(v, false)
		if on == (s.Default == "true") {
			return nil, nil
		}
		if on {
			return []string{"-" + s.Flag}, nil
		}
		return []string{"-" + s.Flag + "=false"}, nil
	case Override:
		if i, ok := s.Index(v); ok && i > 0 {
			return []string{"-" + s.Flag + "=" + strconv.FormatBool(s.Options[i].On)}, nil
		}
	case Choice:
		if i, ok := s.Index(v); ok && i > 0 {
			return []string{"-" + s.Options[i].Flag}, nil
		}
	case Number:
		t := strings.TrimSpace(v)
		if t == "" {
			if s.Default == "" {
				return nil, nil
			}
			t = s.Default
		}
		n, err := s.Whole(t)
		if err != nil {
			return nil, &Refusal{Setting: s, Value: t}
		}
		if s.Default != "" && strconv.Itoa(n) == s.Default {
			return nil, nil
		}
		return []string{"-" + s.Flag, strconv.Itoa(n)}, nil
	case Text:
		if t := strings.TrimSpace(v); t != "" {
			return []string{"-" + s.Flag, t}, nil
		}
	case Arguments:
		args, ok := Split(v)
		if !ok {
			return nil, &Refusal{Setting: s, Value: v}
		}
		return args, nil
	case Verbatim:
		if v != "" {
			return []string{"-" + s.Flag, v}, nil
		}
	}
	return nil, nil
}

// Build is the arguments for each row of Table with its values: the first,
// or each for a row that repeats. The first refused value stops it.
func Build(values func(Setting) []string) ([]string, error) {
	first := func(s Setting) string {
		if vs := values(s); len(vs) > 0 {
			return vs[0]
		}
		return ""
	}
	var args []string
	for _, s := range Table {
		if n, ok := ByKey(s.Needs); ok && strings.TrimSpace(first(n)) == "" {
			continue
		}
		vs := values(s)
		if !s.Repeat {
			vs = []string{first(s)}
		}
		for _, v := range vs {
			if s.Excludes != "" && strings.TrimSpace(v) != "" {
				if x, ok := ByKey(s.Excludes); ok && ParseBool(first(x), false) {
					return nil, &Refusal{Setting: s, Value: v, Clash: true}
				}
			}
			more, err := s.Args(v)
			if err != nil {
				return nil, err
			}
			args = append(args, more...)
		}
	}
	return args, nil
}

// One is Build's values from one value per row.
func One(value func(Setting) string) func(Setting) []string {
	return func(s Setting) []string { return []string{value(s)} }
}

// ParseBool reads the settings file's spellings of a switch, else def.
func ParseBool(v string, def bool) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	}
	return def
}

// FormatBool is the settings file's spelling of b.
func FormatBool(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// Split splits a command-line fragment on blanks. Double quotes group words
// and are removed; there are no escapes. It is false for an unmatched quote.
func Split(line string) ([]string, bool) {
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
		return nil, false
	}
	if have {
		args = append(args, cur.String())
	}
	return args, true
}
