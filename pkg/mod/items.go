package mod

import (
	"againrom/pkg/locale"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

// ItemsFile is the data file that adds items and edits existing ones. A mod
// loads it from its entry script with game.data.load.
const ItemsFile = "data/items.toml"

// StringsFile returns the text file of a language, relative to the mod folder.
func StringsFile(lang string) string { return "text/" + lang + "/strings.toml" }

var itemKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,31}$`)

// ItemSlots names the equipment slots a mod item can occupy and the slot
// number each carries in an item code. The shield slot is the class of the
// shields; every other slot is a place an armour is worn. Slot 1 is the weapon
// class, which a mod cannot add to yet, and slots 3 and 11 hold no armour.
var ItemSlots = map[string]int{
	"shield": 2, "ring": 4, "amulet": 5, "head": 6, "body": 7,
	"outer": 8, "bracers": 9, "gloves": 10, "boots": 12,
}

// ItemSlotNames lists the slot names in slot-number order.
func ItemSlotNames() []string {
	names := make([]string, 0, len(ItemSlots))
	for n := range ItemSlots {
		names = append(names, n)
	}
	sort.Slice(names, func(i, j int) bool { return ItemSlots[names[i]] < ItemSlots[names[j]] })
	return names
}

// AnchorSlots names the equipment slots a layer can be placed against and the
// slot number each has in the twelve-slot record. A layer is drawn just before
// (under) or just after (over) the item worn in its anchor slot, whether or not
// that slot is occupied.
var AnchorSlots = map[string]int{
	"weapon": 1, "shield": 2, "ring": 4, "amulet": 5, "head": 6, "body": 7,
	"outer": 8, "bracers": 9, "gloves": 10, "legs": 11, "boots": 12,
}

// AnchorSlotNames lists the anchor names in slot-number order.
func AnchorSlotNames() []string {
	names := make([]string, 0, len(AnchorSlots))
	for n := range AnchorSlots {
		names = append(names, n)
	}
	sort.Slice(names, func(i, j int) bool { return AnchorSlots[names[i]] < AnchorSlots[names[j]] })
	return names
}

// Layer placements, as the "layer" key spells them. LayerMain is the ordinary
// item that occupies its slot; the other two make the item a clothing layer.
const (
	LayerMain  = "main"
	LayerUnder = "under"
	LayerOver  = "over"
)

// Item suitability, as the "for" key spells it: who may use the item.
const (
	SuitFighter = 1
	SuitMage    = 2
	SuitAny     = SuitFighter | SuitMage
)

// Bounds of the numbers an item row carries.
const (
	maxItemDefence    = 250
	maxItemAbsorption = 50
	maxItemWeight     = 30000
	maxItemPrice      = 100_000_000
	maxItemNameRunes  = 40
)

// ItemRow is one item a mod adds. Its numbers are those of the item when it is
// made of iron and of common quality, which every other material and quality
// scales; the engine writes them into the item table row it allocates.
type ItemRow struct {
	Mod  string
	File string
	Line int
	// Dir is the mod folder, set by the loader that read the file.
	Dir string

	Key     string
	NameKey string
	// Name is the text of NameKey in the language the game runs in.
	Name       string
	NameLine   int
	Slot       string
	SlotNo     int
	Defence    int32
	Absorption int32
	Weight     int32
	Price      int32
	// Sprite is the PNG of the item's inventory picture, relative to the mod
	// folder; empty when the item shows its stand-in's picture. SpriteLine is
	// the line that gives it.
	Sprite     string
	SpriteLine int
	// StandIn names the original item a save holds in this item's place;
	// StandInLine is the line that gives it.
	StandIn     string
	StandInLine int
	Suit        int32
	// Stock puts one unit on the shelf of every town shop.
	Stock bool

	// Layer is LayerUnder or LayerOver for a clothing layer and empty for an
	// ordinary item. A layer is worn beside the item in its anchor slot, never
	// in a slot of its own. Anchor is the anchor slot's name and AnchorNo its
	// number; both are set for a layer, defaulting to the item's own slot.
	Layer     string
	LayerLine int
	Anchor    string
	AnchorNo  int
	// Figure names the original armour item whose worn picture the item is
	// drawn with; empty draws the stand-in's. FigureLine is the line that
	// gives it.
	Figure     string
	FigureLine int
}

// ItemChange edits fields of an item the game already has. A nil field is left
// as it is.
type ItemChange struct {
	Mod  string
	File string
	Line int

	// Item is the name of the row in the armour, shield or weapon table.
	Item       string
	Defence    *int32
	Absorption *int32
	Weight     *int32
	Price      *int32
}

// ItemData is what the mods of a set add and edit, in load order.
type ItemData struct {
	Rows    []ItemRow
	Changes []ItemChange
}

// Empty reports that no mod adds or edits an item.
func (d ItemData) Empty() bool { return len(d.Rows) == 0 && len(d.Changes) == 0 }

// ItemFileError is a refusal of a data file, naming the file and the line.
type ItemFileError struct {
	File string
	Line int
	Msg  string
}

func (e *ItemFileError) Error() string {
	if e.Line > 0 {
		return fmt.Sprintf("%s:%d: %s", e.File, e.Line, e.Msg)
	}
	return fmt.Sprintf("%s: %s", e.File, e.Msg)
}

// Strings is the text of one language: a text key to its string.
type Strings map[string]string

// ParseStrings reads text/<lang>/strings.toml: one `"key" = "text"` pair per
// line, keys written as a string or bare.
func ParseStrings(file string, data []byte) (Strings, error) {
	tables, err := parseTOMLWith(data, tomlOptions{quotedKeys: true})
	if err != nil {
		return nil, fileError(file, err)
	}
	if len(tables) > 1 {
		return nil, &ItemFileError{File: file, Line: tables[1].line, Msg: "a strings file holds key = \"text\" lines and no table"}
	}
	out := Strings{}
	for _, p := range tables[0].pairs {
		if p.val.kind != tomlString {
			return nil, &ItemFileError{File: file, Line: p.line, Msg: fmt.Sprintf("%q must be a string", p.key)}
		}
		out[p.key] = p.val.str
	}
	return out, nil
}

// fileError turns a parser error of the form "line N: message" into one that
// names the file.
func fileError(file string, err error) error {
	msg := err.Error()
	var line int
	if n, _ := fmt.Sscanf(msg, "line %d:", &line); n == 1 {
		msg = strings.TrimSpace(msg[strings.Index(msg, ":")+1:])
		return &ItemFileError{File: file, Line: line, Msg: msg}
	}
	return &ItemFileError{File: file, Msg: msg}
}

// LoadStrings reads the strings of lang from the mod folder dir. A mod without
// the file has no text in that language.
func LoadStrings(dir, lang string) (Strings, error) {
	file := StringsFile(lang)
	if _, err := os.Lstat(filepath.Join(dir, filepath.FromSlash(file))); errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	data, _, err := ReadInside(dir, file)
	if err != nil {
		return nil, &ItemFileError{File: file, Msg: err.Error()}
	}
	return ParseStrings(file, data)
}

// TextLookup reads the strings a game running in lang uses from the mod folder
// dir and returns the lookup ParseItems takes: the language's own string, else
// the English one.
func TextLookup(dir, lang string) (func(key string) (string, bool), error) {
	return Lookup(lang, func(l string) (Strings, error) { return LoadStrings(dir, l) })
}

// Lookup is the one text lookup over strings tables in the text/<lang> layout,
// for a mod's folder and for the engine's own words alike. load returns the
// table of one language, nil when there is none; a key reads the language's
// own string, else the English one.
func Lookup(lang string, load func(lang string) (Strings, error)) (func(key string) (string, bool), error) {
	var sets []Strings
	for _, l := range languagesFor(lang) {
		s, err := load(l)
		if err != nil {
			return nil, err
		}
		sets = append(sets, s)
	}
	return func(key string) (string, bool) {
		for _, s := range sets {
			if v, ok := s[key]; ok {
				return v, true
			}
		}
		return "", false
	}, nil
}

func languagesFor(lang string) []string {
	if lang == "" || lang == locale.Fallback {
		return []string{locale.Fallback}
	}
	return []string{lang, locale.Fallback}
}

// ParseItems reads data/items.toml of the mod id. text resolves a text key to
// its string in the game's language and reports whether the key exists; lang
// names that language in a refusal. A refusal names the file and the line.
func ParseItems(id, file string, data []byte, text func(key string) (string, bool), lang string) (ItemData, error) {
	tables, err := parseTOMLWith(data, tomlOptions{arrays: true})
	if err != nil {
		return ItemData{}, fileError(file, err)
	}
	bad := func(line int, format string, args ...any) error {
		return &ItemFileError{File: file, Line: line, Msg: fmt.Sprintf(format, args...)}
	}
	if len(tables[0].pairs) != 0 {
		p := tables[0].pairs[0]
		return ItemData{}, bad(p.line, "%s is outside an [[item]] or [[change]] table", p.key)
	}
	var out ItemData
	keys := map[string]int{}
	for _, t := range tables[1:] {
		switch {
		case t.name == "item" && t.array:
			row, err := itemRowFrom(id, file, t, text, lang)
			if err != nil {
				return ItemData{}, err
			}
			if first, dup := keys[row.Key]; dup {
				return ItemData{}, bad(t.line, "item key %q is already used at line %d", row.Key, first)
			}
			keys[row.Key] = t.line
			out.Rows = append(out.Rows, row)
		case t.name == "change" && t.array:
			c, err := itemChangeFrom(id, file, t)
			if err != nil {
				return ItemData{}, err
			}
			out.Changes = append(out.Changes, c)
		default:
			return ItemData{}, bad(t.line, "unknown table %q (this file holds [[item]] and [[change]])", t.name)
		}
	}
	return out, nil
}

func itemRowFrom(id, file string, t tomlTable, text func(string) (string, bool), lang string) (ItemRow, error) {
	bad := func(line int, format string, args ...any) error {
		return &ItemFileError{File: file, Line: line, Msg: fmt.Sprintf(format, args...)}
	}
	r := ItemRow{Mod: id, File: file, Line: t.line, Suit: SuitAny}
	var have = map[string]int{}
	nameLine := t.line
	for _, p := range t.pairs {
		have[p.key] = p.line
		str := func() (string, error) {
			if p.val.kind != tomlString {
				return "", bad(p.line, "%s must be a string, not %s", p.key, p.val.kind)
			}
			return p.val.str, nil
		}
		num := func(max int64) (int32, error) {
			if p.val.kind != tomlInt {
				return 0, bad(p.line, "%s must be an integer, not %s", p.key, p.val.kind)
			}
			if p.val.num < 0 || p.val.num > max {
				return 0, bad(p.line, "%s is %d, outside 0..%d", p.key, p.val.num, max)
			}
			return int32(p.val.num), nil
		}
		var err error
		switch p.key {
		case "key":
			if r.Key, err = str(); err == nil && !itemKeyPattern.MatchString(r.Key) {
				err = bad(p.line, "key %q must be 1 to 32 of a-z, 0-9 and '_', starting with a letter", r.Key)
			}
		case "name":
			nameLine = p.line
			r.NameKey, err = str()
		case "slot":
			if r.Slot, err = str(); err == nil {
				n, ok := ItemSlots[r.Slot]
				if !ok {
					if r.Slot == "weapon" {
						err = bad(p.line, "slot \"weapon\": a mod cannot add a weapon yet (slots: %s)", strings.Join(ItemSlotNames(), ", "))
					} else {
						err = bad(p.line, "unknown slot %q (slots: %s)", r.Slot, strings.Join(ItemSlotNames(), ", "))
					}
				}
				r.SlotNo = n
			}
		case "defence":
			r.Defence, err = num(maxItemDefence)
		case "absorption":
			r.Absorption, err = num(maxItemAbsorption)
		case "weight":
			r.Weight, err = num(maxItemWeight)
		case "price":
			r.Price, err = num(maxItemPrice)
		case "sprite":
			r.SpriteLine = p.line
			if r.Sprite, err = str(); err == nil {
				err = checkModPath(file, p.line, "sprite", r.Sprite, ".png")
			}
		case "stand-in":
			r.StandInLine = p.line
			if r.StandIn, err = str(); err == nil && strings.TrimSpace(r.StandIn) == "" {
				err = bad(p.line, "stand-in is empty")
			}
		case "for":
			var who string
			if who, err = str(); err == nil {
				switch who {
				case "any":
					r.Suit = SuitAny
				case "fighter":
					r.Suit = SuitFighter
				case "mage":
					r.Suit = SuitMage
				default:
					err = bad(p.line, "for is %q; use any, fighter or mage", who)
				}
			}
		case "layer":
			r.LayerLine = p.line
			var placement string
			if placement, err = str(); err == nil {
				switch placement {
				case LayerMain:
					r.Layer = ""
				case LayerUnder, LayerOver:
					r.Layer = placement
				default:
					err = bad(p.line, "layer is %q; use under, main or over", placement)
				}
			}
		case "anchor":
			if r.Anchor, err = str(); err == nil {
				n, ok := AnchorSlots[r.Anchor]
				if !ok {
					err = bad(p.line, "unknown anchor %q (anchors: %s)", r.Anchor, strings.Join(AnchorSlotNames(), ", "))
				}
				r.AnchorNo = n
			}
		case "figure":
			r.FigureLine = p.line
			if r.Figure, err = str(); err == nil && strings.TrimSpace(r.Figure) == "" {
				err = bad(p.line, "figure is empty")
			}
		case "shop":
			if p.val.kind != tomlBool {
				err = bad(p.line, "shop must be true or false, not %s", p.val.kind)
			}
			r.Stock = p.val.flag
		default:
			err = bad(p.line, "unknown key %q in [[item]]", p.key)
		}
		if err != nil {
			return ItemRow{}, err
		}
	}
	for _, need := range []string{"key", "name", "slot", "stand-in"} {
		if _, ok := have[need]; !ok {
			return ItemRow{}, bad(t.line, "[[item]] has no %s", need)
		}
	}
	if line, ok := have["anchor"]; ok && r.Layer == "" {
		return ItemRow{}, bad(line, "anchor needs layer = \"under\" or \"over\"")
	}
	if r.Layer != "" && r.Anchor == "" {
		r.Anchor, r.AnchorNo = r.Slot, ItemSlots[r.Slot]
	}
	name, ok := text(r.NameKey)
	if !ok {
		return ItemRow{}, bad(nameLine, "text key %q is not in %s (nor in %s)", r.NameKey, StringsFile(lang), StringsFile("en"))
	}
	if n := utf8.RuneCountInString(name); n == 0 || n > maxItemNameRunes {
		return ItemRow{}, bad(nameLine, "the text of %q has %d characters; an item name has 1 to %d", r.NameKey, n, maxItemNameRunes)
	}
	r.Name, r.NameLine = name, nameLine
	return r, nil
}

func itemChangeFrom(id, file string, t tomlTable) (ItemChange, error) {
	bad := func(line int, format string, args ...any) error {
		return &ItemFileError{File: file, Line: line, Msg: fmt.Sprintf(format, args...)}
	}
	c := ItemChange{Mod: id, File: file, Line: t.line}
	for _, p := range t.pairs {
		var dst **int32
		var max int64
		switch p.key {
		case "item":
			if p.val.kind != tomlString || p.val.str == "" {
				return ItemChange{}, bad(p.line, "item must be the name of a row, as a string")
			}
			c.Item = p.val.str
			continue
		case "defence":
			dst, max = &c.Defence, maxItemDefence
		case "absorption":
			dst, max = &c.Absorption, maxItemAbsorption
		case "weight":
			dst, max = &c.Weight, maxItemWeight
		case "price":
			dst, max = &c.Price, maxItemPrice
		default:
			return ItemChange{}, bad(p.line, "unknown key %q in [[change]]", p.key)
		}
		if p.val.kind != tomlInt {
			return ItemChange{}, bad(p.line, "%s must be an integer, not %s", p.key, p.val.kind)
		}
		if p.val.num < 0 || p.val.num > max {
			return ItemChange{}, bad(p.line, "%s is %d, outside 0..%d", p.key, p.val.num, max)
		}
		v := int32(p.val.num)
		*dst = &v
	}
	if c.Item == "" {
		return ItemChange{}, bad(t.line, "[[change]] has no item")
	}
	if c.Defence == nil && c.Absorption == nil && c.Weight == nil && c.Price == nil {
		return ItemChange{}, bad(t.line, "[[change]] of %q changes nothing", c.Item)
	}
	return c, nil
}

// checkModPath refuses a path that is not a relative slash path inside the mod
// folder with the extension ext.
func checkModPath(file string, line int, what, p, ext string) error {
	bad := func(format string, args ...any) error {
		return &ItemFileError{File: file, Line: line, Msg: fmt.Sprintf(format, args...)}
	}
	if p == "" || strings.ContainsAny(p, "\\:\x00") || strings.HasPrefix(p, "/") {
		return bad("%s %q is not a path inside the mod folder", what, p)
	}
	clean := path.Clean(p)
	if clean != p || clean == ".." || strings.HasPrefix(clean, "../") {
		return bad("%s %q is not a clean path inside the mod folder", what, p)
	}
	if !strings.EqualFold(path.Ext(p), ext) {
		return bad("%s %q is not a %s file", what, p, ext)
	}
	return nil
}
