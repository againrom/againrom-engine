package mod

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"againrom/pkg/locale"
)

// ItemMaterial allows or forbids one material for one tier of an armour,
// shield or weapon row: one bit of the row's five material masks
// (DAT-MATMASK-020). Item names the row as the item tables spell it, or the
// key of a weapon row a mod adds; Tier and Material are the names of the
// shape and material rows.
type ItemMaterial struct {
	Mod  string
	File string
	Line int
	// Dir is the mod's folder or RES file, set by the loader that read the file.
	Dir string

	Item     string
	Tier     string
	Material string
	Allow    bool
	// AllowLine is the line that sets allow.
	AllowLine int
	// Sprite is the PNG of the pair's inventory picture, relative to the mod
	// folder; empty when the install or the added row's base gives one.
	Sprite     string
	SpriteLine int
	// NameKey is the text key of the pair's shown name and Name its text; both
	// empty when the pair shows the name the item tables give it.
	NameKey  string
	Name     string
	NameLine int
}

// WeaponParam is one parameter of a weapon row a mod may set: its key, its
// column in the row and the bounds of its value.
type WeaponParam struct {
	Key      string
	Column   int
	Min, Max int64
}

// WeaponParams lists the weapon row parameters a mod may set. Each column's
// meaning is a pinned claim: price SHOP-103, weight ITEM-SCALE-017, the
// damage pair ITEM-DMGCOL-018, to-hit and defence ITEM-SCALE-017, range
// ITEM-221. Every other column keeps the base row's value.
var WeaponParams = []WeaponParam{
	{Key: "price", Column: 2, Min: 0, Max: maxItemPrice},
	{Key: "weight", Column: 3, Min: 0, Max: maxItemWeight},
	{Key: "damage-min", Column: 6, Min: 0, Max: 255},
	{Key: "damage-max", Column: 7, Min: 0, Max: 255},
	{Key: "to-hit", Column: 8, Min: 0, Max: 255},
	{Key: "defence", Column: 9, Min: 0, Max: 255},
	{Key: "range", Column: 11, Min: 1, Max: 255},
}

// WeaponParamValue is one parameter a weapon row sets, with its line.
type WeaponParamValue struct {
	WeaponParam
	Value int32
	Line  int
}

// WeaponRow is a weapon row a mod adds: a copy of the shipped row Base names,
// with the parameters it sets.
type WeaponRow struct {
	Mod  string
	File string
	Line int
	// Dir is the mod's folder or RES file, set by the loader that read the file.
	Dir string

	Key      string
	NameKey  string
	Name     string
	NameLine int
	Base     string
	BaseLine int
	Params   []WeaponParamValue
	// Sprite is the PNG of the row's inventory picture for every material and
	// tier; empty when each pair shows the base row's picture of that pair.
	Sprite     string
	SpriteLine int
}

func weaponParamKeys() string {
	keys := make([]string, len(WeaponParams))
	for i, p := range WeaponParams {
		keys[i] = p.Key
	}
	return strings.Join(keys, ", ")
}

func itemMaterialFrom(id, file string, t tomlTable, text func(string) (string, bool), lang string) (ItemMaterial, error) {
	bad := func(line int, format string, args ...any) error {
		return &ItemFileError{File: file, Line: line, Msg: fmt.Sprintf(format, args...)}
	}
	m := ItemMaterial{Mod: id, File: file, Line: t.line}
	have := map[string]bool{}
	for _, p := range t.pairs {
		have[p.key] = true
		str := func() (string, error) {
			if p.val.kind != tomlString || strings.TrimSpace(p.val.str) == "" {
				return "", bad(p.line, "%s must be a non-empty string", p.key)
			}
			return p.val.str, nil
		}
		var err error
		switch p.key {
		case "item":
			m.Item, err = str()
		case "tier":
			m.Tier, err = str()
		case "material":
			m.Material, err = str()
		case "allow":
			if p.val.kind != tomlBool {
				err = bad(p.line, "allow must be true or false, not %s", p.val.kind)
			}
			m.Allow, m.AllowLine = p.val.flag, p.line
		case "sprite":
			m.SpriteLine = p.line
			if m.Sprite, err = str(); err == nil {
				err = checkModPath(file, p.line, "sprite", m.Sprite, ".png")
			}
		case "name":
			m.NameLine = p.line
			m.NameKey, err = str()
		default:
			err = bad(p.line, "unknown key %q in [[material]] (keys: item, tier, material, allow, sprite, name)", p.key)
		}
		if err != nil {
			return ItemMaterial{}, err
		}
	}
	for _, need := range []string{"item", "tier", "material", "allow"} {
		if !have[need] {
			return ItemMaterial{}, bad(t.line, "[[material]] has no %s", need)
		}
	}
	if !m.Allow && (m.Sprite != "" || m.NameKey != "") {
		return ItemMaterial{}, bad(m.AllowLine, "a forbidden material takes no sprite and no name")
	}
	if m.NameKey != "" {
		name, err := itemText(file, m.NameLine, m.NameKey, text, lang)
		if err != nil {
			return ItemMaterial{}, err
		}
		m.Name = name
	}
	return m, nil
}

func weaponRowFrom(id, file string, t tomlTable, text func(string) (string, bool), lang string) (WeaponRow, error) {
	bad := func(line int, format string, args ...any) error {
		return &ItemFileError{File: file, Line: line, Msg: fmt.Sprintf(format, args...)}
	}
	w := WeaponRow{Mod: id, File: file, Line: t.line}
	have := map[string]bool{}
	for _, p := range t.pairs {
		have[p.key] = true
		str := func() (string, error) {
			if p.val.kind != tomlString || strings.TrimSpace(p.val.str) == "" {
				return "", bad(p.line, "%s must be a non-empty string", p.key)
			}
			return p.val.str, nil
		}
		var err error
		switch p.key {
		case "key":
			if w.Key, err = str(); err == nil && !itemKeyPattern.MatchString(w.Key) {
				err = bad(p.line, "key %q must be 1 to 32 of a-z, 0-9 and '_', starting with a letter", w.Key)
			}
		case "name":
			w.NameLine = p.line
			w.NameKey, err = str()
		case "base":
			w.BaseLine = p.line
			w.Base, err = str()
		case "sprite":
			w.SpriteLine = p.line
			if w.Sprite, err = str(); err == nil {
				err = checkModPath(file, p.line, "sprite", w.Sprite, ".png")
			}
		default:
			i := weaponParamIndex(p.key)
			if i < 0 {
				err = bad(p.line, "%q is not a weapon parameter a mod may set (keys: key, name, base, sprite, %s); every other parameter keeps the base row's value", p.key, weaponParamKeys())
				break
			}
			wp := WeaponParams[i]
			if p.val.kind != tomlInt {
				err = bad(p.line, "%s must be an integer, not %s", p.key, p.val.kind)
				break
			}
			if p.val.num < wp.Min || p.val.num > wp.Max {
				err = bad(p.line, "%s is %d, outside %d..%d", p.key, p.val.num, wp.Min, wp.Max)
				break
			}
			w.Params = append(w.Params, WeaponParamValue{WeaponParam: wp, Value: int32(p.val.num), Line: p.line})
		}
		if err != nil {
			return WeaponRow{}, err
		}
	}
	for _, need := range []string{"key", "name", "base"} {
		if !have[need] {
			return WeaponRow{}, bad(t.line, "[[weapon]] has no %s", need)
		}
	}
	name, err := itemText(file, w.NameLine, w.NameKey, text, lang)
	if err != nil {
		return WeaponRow{}, err
	}
	w.Name = name
	return w, nil
}

func weaponParamIndex(key string) int {
	for i, p := range WeaponParams {
		if p.Key == key {
			return i
		}
	}
	return -1
}

// itemText is the shown name a text key gives, refused when the key is absent
// or its text is not 1 to maxItemNameRunes characters.
func itemText(file string, line int, key string, text func(string) (string, bool), lang string) (string, error) {
	name, ok := text(key)
	if !ok {
		return "", &ItemFileError{File: file, Line: line, Msg: fmt.Sprintf("text key %q is not in %s (nor in %s)", key, StringsFile(lang), StringsFile(locale.Fallback))}
	}
	if n := utf8.RuneCountInString(name); n == 0 || n > maxItemNameRunes {
		return "", &ItemFileError{File: file, Line: line, Msg: fmt.Sprintf("the text of %q has %d characters; an item name has 1 to %d", key, n, maxItemNameRunes)}
	}
	return name, nil
}
