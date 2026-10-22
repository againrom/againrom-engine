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
	// Arguments are free arguments, split by Split and passed last.
	Arguments
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
}

// Setting is one row of the launch settings table: a value the starter keeps
// in its [options] section and passes to the game on its command line. The
// starter's captions for a row are keyed by Key in the starter.
type Setting struct {
	Key     string // key in the settings file
	Kind    Kind
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

// The rows of the table.
var (
	Sound = Setting{Key: "sound", Kind: Override, Default: SoundDefault, GameDefault: "true", Flag: "sound",
		Options: []Option{{Value: SoundDefault}, {Value: SoundOn, On: true}, {Value: SoundOff}}}
	Volume = Setting{Key: "volume", Kind: Number, Min: 0, Max: MaxVolume, Read: ReadInt, GameDefault: strconv.Itoa(MaxVolume),
		Flag: "volume"}
	Movies = Setting{Key: "movies", Kind: Switch, Default: "true", Flag: "movies"}
	Video  = Setting{Key: "video", Kind: Choice, Default: VideoNormal,
		Options: []Option{{Value: VideoNormal}, {Value: Video4x, Flag: "4x"}, {Value: Video8x, Flag: "8x"}}}
	Markers             = Setting{Key: "markers", Kind: Switch, Default: "false", Flag: "markers"}
	Chicken             = Setting{Key: "chicken", Kind: Switch, Default: "false", Flag: "chicken"}
	Window              = Setting{Key: "window", Kind: Switch, Default: "false", Flag: "window"}
	Saves               = Setting{Key: "saves", Kind: Text, Flag: "saves"}
	Mission             = Setting{Key: "mission", Kind: Number, Min: 1, Read: ReadAtStart, Flag: "mission", Excludes: "picker"}
	Picker              = Setting{Key: "picker", Kind: Switch, Default: "false", Flag: "picker"}
	Skill               = Setting{Key: "skill", Kind: Text, Flag: "skill"}
	ShopStockMultiplier = Setting{Key: "shop-stock-multiplier", Kind: Number, Default: "1", Min: 1, Max: MaxShopStockMultiplier,
		Read: ReadWhole, Flag: "shop-stock-multiplier"}
	Extra = Setting{Key: "extra", Kind: Arguments}
)

// Settings is the launch settings table, in the order the settings file keeps
// the keys and the game's arguments pass them.
var Settings = []Setting{Sound, Volume, Movies, Video, Markers, Chicken, Window, Saves, Mission, Picker, Skill, ShopStockMultiplier, Extra}

// ByKey returns the row with key.
func ByKey(key string) (Setting, bool) {
	for _, s := range Settings {
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
	}
	return nil, nil
}

// Build is the game's arguments for every row, in table order, with the
// values value gives. The first refused value stops it.
func Build(value func(Setting) string) ([]string, error) {
	var args []string
	for _, s := range Settings {
		v := value(s)
		if s.Excludes != "" && strings.TrimSpace(v) != "" {
			if x, ok := ByKey(s.Excludes); ok && ParseBool(value(x), false) {
				return nil, &Refusal{Setting: s, Value: v, Clash: true}
			}
		}
		more, err := s.Args(v)
		if err != nil {
			return nil, err
		}
		args = append(args, more...)
	}
	return args, nil
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
