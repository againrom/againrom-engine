package mod

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// SettingsName is the optional file that declares a mod's knobs.
const SettingsName = "settings.toml"

var settingKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,31}$`)

// ValidSettingKey reports whether key can name a setting.
func ValidSettingKey(key string) bool { return settingKeyPattern.MatchString(key) }

// Kind is the type of a setting.
type Kind int

// The setting types.
const (
	KindInt Kind = iota
	KindBool
	KindChoice
)

func (k Kind) String() string { return [...]string{"int", "bool", "choice"}[k] }

// Value is a setting's value: one of an integer, a boolean or a string.
type Value struct {
	Kind Kind
	Int  int64
	Bool bool
	Str  string
}

// String is the canonical text of the value, the form -mod-setting reads back.
func (v Value) String() string {
	switch v.Kind {
	case KindInt:
		return strconv.FormatInt(v.Int, 10)
	case KindBool:
		return strconv.FormatBool(v.Bool)
	}
	return v.Str
}

// Setting is one declared knob of a mod.
type Setting struct {
	Key      string
	LabelEN  string
	LabelRU  string
	Kind     Kind
	Default  Value
	Min, Max int64    // KindInt only
	Choices  []string // KindChoice only
}

// Label is the setting's label in lang ("en" or "ru"); any other language, and
// a language the mod gave no label for, read the English label.
func (s Setting) Label(lang string) string {
	if lang == "ru" && s.LabelRU != "" {
		return s.LabelRU
	}
	if s.LabelEN != "" {
		return s.LabelEN
	}
	return s.Key
}

// Parse reads text as a value of the setting. The error says what is allowed.
func (s Setting) Parse(text string) (Value, error) {
	switch s.Kind {
	case KindInt:
		n, err := strconv.ParseInt(strings.TrimSpace(text), 10, 64)
		if err != nil {
			return Value{}, fmt.Errorf("%q is not an integer", text)
		}
		if n < s.Min || n > s.Max {
			return Value{}, fmt.Errorf("%d is outside %d..%d", n, s.Min, s.Max)
		}
		return Value{Kind: KindInt, Int: n}, nil
	case KindBool:
		switch strings.ToLower(strings.TrimSpace(text)) {
		case "true", "on", "yes", "1":
			return Value{Kind: KindBool, Bool: true}, nil
		case "false", "off", "no", "0":
			return Value{Kind: KindBool, Bool: false}, nil
		}
		return Value{}, fmt.Errorf("%q is not true or false", text)
	}
	for _, c := range s.Choices {
		if c == text {
			return Value{Kind: KindChoice, Str: c}, nil
		}
	}
	return Value{}, fmt.Errorf("%q is not one of %s", text, strings.Join(s.Choices, ", "))
}

// LoadSettings reads settings.toml of the mod folder dir. A mod without the
// file declares no settings.
func LoadSettings(dir string) ([]Setting, error) {
	data, err := os.ReadFile(filepath.Join(dir, SettingsName))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	out, err := ParseSettings(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", filepath.Join(dir, SettingsName), err)
	}
	return out, nil
}

// ParseSettings reads a settings.toml. The error names the line and the key.
func ParseSettings(data []byte) ([]Setting, error) {
	tables, err := parseTOML(data)
	if err != nil {
		return nil, err
	}
	if len(tables[0].pairs) != 0 {
		return nil, fmt.Errorf("line %d: a setting is a [table]; %s is outside one", tables[0].pairs[0].line, tables[0].pairs[0].key)
	}
	var out []Setting
	for _, t := range tables[1:] {
		s, err := settingFrom(t)
		if err != nil {
			return nil, fmt.Errorf("setting %s (line %d): %w", t.name, t.line, err)
		}
		out = append(out, s)
	}
	return out, nil
}

func settingFrom(t tomlTable) (Setting, error) {
	if !ValidSettingKey(t.name) {
		return Setting{}, errors.New("the key must be 1 to 32 of a-z, 0-9 and '_', starting with a letter")
	}
	s := Setting{Key: t.name}
	var typ string
	var def, lo, hi *tomlPair
	var choices []string
	for i := range t.pairs {
		p := &t.pairs[i]
		switch p.key {
		case "label":
			switch p.val.kind {
			case tomlString:
				s.LabelEN = p.val.str
			case tomlInline:
				for _, q := range p.val.tbl {
					if q.val.kind != tomlString {
						return Setting{}, fmt.Errorf("line %d: label.%s must be a string", p.line, q.key)
					}
					switch q.key {
					case "en":
						s.LabelEN = q.val.str
					case "ru":
						s.LabelRU = q.val.str
					default:
						return Setting{}, fmt.Errorf("line %d: label has no language %q (use en and ru)", p.line, q.key)
					}
				}
			default:
				return Setting{}, fmt.Errorf("line %d: label must be a string or { en = \"...\", ru = \"...\" }", p.line)
			}
		case "type":
			if p.val.kind != tomlString {
				return Setting{}, fmt.Errorf("line %d: type must be a string", p.line)
			}
			typ = p.val.str
		case "default":
			def = p
		case "min":
			lo = p
		case "max":
			hi = p
		case "choices":
			if p.val.kind != tomlArray {
				return Setting{}, fmt.Errorf("line %d: choices must be an array of strings", p.line)
			}
			for _, c := range p.val.list {
				if c.kind != tomlString {
					return Setting{}, fmt.Errorf("line %d: choices must be an array of strings", p.line)
				}
				choices = append(choices, c.str)
			}
		default:
			return Setting{}, fmt.Errorf("line %d: unknown key %q", p.line, p.key)
		}
	}
	if def == nil {
		return Setting{}, errors.New("default is missing")
	}
	switch typ {
	case "int":
		s.Kind = KindInt
		if def.val.kind != tomlInt {
			return Setting{}, fmt.Errorf("line %d: default must be %s", def.line, tomlInt)
		}
		if lo == nil || hi == nil {
			return Setting{}, errors.New("an int setting needs min and max")
		}
		if lo.val.kind != tomlInt || hi.val.kind != tomlInt {
			return Setting{}, fmt.Errorf("line %d: min and max must be integers", lo.line)
		}
		s.Min, s.Max = lo.val.num, hi.val.num
		if s.Min > s.Max {
			return Setting{}, fmt.Errorf("line %d: min %d is above max %d", lo.line, s.Min, s.Max)
		}
		if def.val.num < s.Min || def.val.num > s.Max {
			return Setting{}, fmt.Errorf("line %d: default %d is outside %d..%d", def.line, def.val.num, s.Min, s.Max)
		}
		s.Default = Value{Kind: KindInt, Int: def.val.num}
	case "bool":
		s.Kind = KindBool
		if def.val.kind != tomlBool {
			return Setting{}, fmt.Errorf("line %d: default must be %s", def.line, tomlBool)
		}
		if lo != nil || hi != nil || len(choices) != 0 {
			return Setting{}, errors.New("a bool setting takes no min, max or choices")
		}
		s.Default = Value{Kind: KindBool, Bool: def.val.flag}
	case "choice":
		s.Kind = KindChoice
		if def.val.kind != tomlString {
			return Setting{}, fmt.Errorf("line %d: default must be %s", def.line, tomlString)
		}
		if len(choices) == 0 {
			return Setting{}, errors.New("a choice setting needs choices")
		}
		if lo != nil || hi != nil {
			return Setting{}, errors.New("a choice setting takes no min or max")
		}
		s.Choices = choices
		if _, err := s.Parse(def.val.str); err != nil {
			return Setting{}, fmt.Errorf("line %d: default: %w", def.line, err)
		}
		s.Default = Value{Kind: KindChoice, Str: def.val.str}
	case "":
		return Setting{}, errors.New("type is missing (int, bool or choice)")
	default:
		return Setting{}, fmt.Errorf("type %q is not int, bool or choice", typ)
	}
	return s, nil
}

// SettingValue is one resolved setting of a mod.
type SettingValue struct {
	Key   string
	Value Value
}

// ResolveSettings applies given (key to text) over the declared defaults, in
// declaration order. A key the mod does not declare, or a value its setting
// refuses, is an error naming the mod and the key.
func ResolveSettings(modID string, decl []Setting, given map[string]string) ([]SettingValue, error) {
	known := map[string]bool{}
	out := make([]SettingValue, 0, len(decl))
	for _, s := range decl {
		known[s.Key] = true
		v := s.Default
		if text, ok := given[s.Key]; ok {
			var err error
			if v, err = s.Parse(text); err != nil {
				return nil, fmt.Errorf("mod %q setting %s: %w", modID, s.Key, err)
			}
		}
		out = append(out, SettingValue{Key: s.Key, Value: v})
	}
	for key := range given {
		if !known[key] {
			return nil, fmt.Errorf("mod %q has no setting %q", modID, key)
		}
	}
	return out, nil
}

// SettingFlag is one parsed -mod-setting argument.
type SettingFlag struct{ Mod, Key, Value string }

// ParseSettingFlag reads "<mod id>.<key>=<value>". The key is the text after
// the last '.' before the '=', since a mod id may itself contain dots and a
// key may not.
func ParseSettingFlag(arg string) (SettingFlag, error) {
	eq := strings.IndexByte(arg, '=')
	if eq < 0 {
		return SettingFlag{}, fmt.Errorf("-mod-setting %q: want <mod>.<key>=<value>", arg)
	}
	name := arg[:eq]
	dot := strings.LastIndexByte(name, '.')
	if dot <= 0 || dot == len(name)-1 {
		return SettingFlag{}, fmt.Errorf("-mod-setting %q: want <mod>.<key>=<value>", arg)
	}
	f := SettingFlag{Mod: name[:dot], Key: name[dot+1:], Value: arg[eq+1:]}
	if !ValidID(f.Mod) {
		return SettingFlag{}, fmt.Errorf("-mod-setting %q: %q is not a valid mod id", arg, f.Mod)
	}
	if !ValidSettingKey(f.Key) {
		return SettingFlag{}, fmt.Errorf("-mod-setting %q: %q is not a valid setting key", arg, f.Key)
	}
	return f, nil
}
