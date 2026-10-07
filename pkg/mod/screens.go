package mod

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// ScreensFile is the data file that declares screens. A mod loads it from its
// entry script with game.data.add.
const ScreensFile = "data/screens.toml"

// Screen kinds, as the "kind" key spells them. Each is a page the engine draws
// from data; a mod cannot ship screen code.
const (
	ScreenInfo  = "info"
	ScreenList  = "list"
	ScreenTable = "table"
)

// Entry kinds that run an action instead of opening a page, as the "action" key
// of an [[action]] table spells them. Such an entry is only in the in-game menu
// over a mission.
const (
	// ActionAbandon leaves the mission without victory and returns to the town
	// it was entered from.
	ActionAbandon = "abandon"
	// ActionRestart reloads the mission from its entry point.
	ActionRestart = "restart"
)

// IsAction reports whether a screen kind is an action entry rather than a page.
func IsAction(kind string) bool { return kind == ActionAbandon || kind == ActionRestart }

// Screen placements, as the "place" key spells them: the menu that carries the
// entry which opens the screen.
const (
	PlaceMain = "main"
	PlaceGame = "game"
	PlaceBoth = "both"
)

// The menus have a fixed number of slots for entries mods register. The main
// menu draws its entries beside the brooch; the in-game menu appends rows below
// the shipped ones.
const (
	MaxMainMenuScreens = 3
	MaxGameMenuScreens = 2
)

// Bounds of a screen's text.
const (
	maxScreenTitleRunes = 40
	maxScreenMenuRunes  = 28
	maxActionMenuRunes  = 40
	maxScreenTextRunes  = 1200
	maxScreenLines      = 60
)

// Screen is one screen a mod declares, with its text resolved in the language
// the game runs in.
type Screen struct {
	Mod  string
	File string
	Line int

	Key  string
	Kind string
	// Title heads the screen; MenuLabel is the text of the menu entry that opens
	// it. For an action entry Title is the label of the confirming row.
	Title     string
	MenuLabel string
	// Main and Game say which menus carry the entry.
	Main bool
	Game bool

	// Paragraphs is the body of an info screen, Items the lines of a list and
	// Rows the label and value of each row of a table.
	Paragraphs []string
	Items      []string
	Rows       [][2]string
}

// ScreenData is the screens the mods of a set declare, in load order.
type ScreenData struct {
	Screens []Screen
}

// Empty reports that no mod declares a screen.
func (d ScreenData) Empty() bool { return len(d.Screens) == 0 }

// Count returns how many screens put an entry in the main menu and in the game
// menu.
func (d ScreenData) Count() (main, game int) {
	for _, s := range d.Screens {
		if s.Main {
			main++
		}
		if s.Game {
			game++
		}
	}
	return main, game
}

// CheckSlots refuses a set of screens that has more entries than a menu has
// slots, naming the screen that does not fit.
func (d ScreenData) CheckSlots() error {
	main, game := 0, 0
	actions := map[string]string{}
	for _, s := range d.Screens {
		if IsAction(s.Kind) {
			if first, dup := actions[s.Kind]; dup {
				return &ItemFileError{File: s.File, Line: s.Line, Msg: fmt.Sprintf(
					"action %q is already added by mod %q", s.Kind, first)}
			}
			actions[s.Kind] = s.Mod
		}
		if s.Main {
			if main++; main > MaxMainMenuScreens {
				return &ItemFileError{File: s.File, Line: s.Line, Msg: fmt.Sprintf(
					"screen %q does not fit: the main menu has %d slots for mod screens", s.Key, MaxMainMenuScreens)}
			}
		}
		if s.Game {
			if game++; game > MaxGameMenuScreens {
				return &ItemFileError{File: s.File, Line: s.Line, Msg: fmt.Sprintf(
					"screen %q does not fit: the game menu has %d slots for mod screens", s.Key, MaxGameMenuScreens)}
			}
		}
	}
	return nil
}

// ParseScreens reads data/screens.toml of the mod id. text resolves a text key
// to its string in the game's language and reports whether the key exists; lang
// names that language in a refusal. A refusal names the file and the line.
func ParseScreens(id, file string, data []byte, text func(key string) (string, bool), lang string) (ScreenData, error) {
	tables, err := parseTOMLWith(data, tomlOptions{arrays: true})
	if err != nil {
		return ScreenData{}, fileError(file, err)
	}
	bad := func(line int, format string, args ...any) error {
		return &ItemFileError{File: file, Line: line, Msg: fmt.Sprintf(format, args...)}
	}
	if len(tables[0].pairs) != 0 {
		p := tables[0].pairs[0]
		return ScreenData{}, bad(p.line, "%s is outside a [[screen]] table", p.key)
	}
	var out ScreenData
	keys := map[string]int{}
	for _, t := range tables[1:] {
		if !t.array || t.name != "screen" && t.name != "action" {
			return ScreenData{}, bad(t.line, "unknown table %q (this file holds [[screen]] and [[action]])", t.name)
		}
		var s Screen
		var err error
		if t.name == "action" {
			s, err = actionFrom(id, file, t, text, lang)
		} else {
			s, err = screenFrom(id, file, t, text, lang)
		}
		if err != nil {
			return ScreenData{}, err
		}
		if first, dup := keys[s.Key]; dup {
			return ScreenData{}, bad(t.line, "screen key %q is already used at line %d", s.Key, first)
		}
		keys[s.Key] = t.line
		out.Screens = append(out.Screens, s)
	}
	if err := out.CheckSlots(); err != nil {
		return ScreenData{}, err
	}
	return out, nil
}

func screenFrom(id, file string, t tomlTable, text func(string) (string, bool), lang string) (Screen, error) {
	bad := func(line int, format string, args ...any) error {
		return &ItemFileError{File: file, Line: line, Msg: fmt.Sprintf(format, args...)}
	}
	s := Screen{Mod: id, File: file, Line: t.line}
	have := map[string]int{}
	var titleKey, menuKey string
	var paragraphKeys, itemKeys []string
	var rowKeys [][2]string
	for _, p := range t.pairs {
		have[p.key] = p.line
		str := func() (string, error) {
			if p.val.kind != tomlString {
				return "", bad(p.line, "%s must be a string, not %s", p.key, p.val.kind)
			}
			return p.val.str, nil
		}
		keyList := func() ([]string, error) {
			if p.val.kind != tomlArray {
				return nil, bad(p.line, "%s must be an array of text keys, not %s", p.key, p.val.kind)
			}
			if len(p.val.list) == 0 {
				return nil, bad(p.line, "%s is empty", p.key)
			}
			if len(p.val.list) > maxScreenLines {
				return nil, bad(p.line, "%s has %d entries; a screen holds at most %d", p.key, len(p.val.list), maxScreenLines)
			}
			out := make([]string, len(p.val.list))
			for i, v := range p.val.list {
				if v.kind != tomlString {
					return nil, bad(p.line, "%s entry %d must be a text key, not %s", p.key, i+1, v.kind)
				}
				out[i] = v.str
			}
			return out, nil
		}
		var err error
		switch p.key {
		case "key":
			if s.Key, err = str(); err == nil && !itemKeyPattern.MatchString(s.Key) {
				err = bad(p.line, "key %q must be 1 to 32 of a-z, 0-9 and '_', starting with a letter", s.Key)
			}
		case "kind":
			if s.Kind, err = str(); err == nil {
				switch s.Kind {
				case ScreenInfo, ScreenList, ScreenTable:
				default:
					err = bad(p.line, "unknown screen kind %q (kinds: %s, %s, %s)", s.Kind, ScreenInfo, ScreenList, ScreenTable)
				}
			}
		case "title":
			titleKey, err = str()
		case "menu":
			menuKey, err = str()
		case "place":
			var place string
			if place, err = str(); err == nil {
				switch place {
				case PlaceMain:
					s.Main = true
				case PlaceGame:
					s.Game = true
				case PlaceBoth:
					s.Main, s.Game = true, true
				default:
					err = bad(p.line, "place is %q; use %s, %s or %s", place, PlaceMain, PlaceGame, PlaceBoth)
				}
			}
		case "text":
			paragraphKeys, err = keyList()
		case "items":
			itemKeys, err = keyList()
		case "rows":
			if p.val.kind != tomlArray {
				err = bad(p.line, "rows must be an array of [label key, value key] pairs, not %s", p.val.kind)
				break
			}
			if len(p.val.list) == 0 {
				err = bad(p.line, "rows is empty")
				break
			}
			if len(p.val.list) > maxScreenLines {
				err = bad(p.line, "rows has %d entries; a screen holds at most %d", len(p.val.list), maxScreenLines)
				break
			}
			for i, v := range p.val.list {
				if v.kind != tomlArray || len(v.list) != 2 || v.list[0].kind != tomlString || v.list[1].kind != tomlString {
					err = bad(p.line, "rows entry %d must be a pair [\"label key\", \"value key\"]", i+1)
					break
				}
				rowKeys = append(rowKeys, [2]string{v.list[0].str, v.list[1].str})
			}
		default:
			err = bad(p.line, "unknown key %q in [[screen]]", p.key)
		}
		if err != nil {
			return Screen{}, err
		}
	}
	for _, need := range []string{"key", "kind", "title", "menu", "place"} {
		if _, ok := have[need]; !ok {
			return Screen{}, bad(t.line, "[[screen]] has no %s", need)
		}
	}
	body := map[string]struct {
		kind string
		set  bool
	}{
		"text":  {ScreenInfo, paragraphKeys != nil},
		"items": {ScreenList, itemKeys != nil},
		"rows":  {ScreenTable, rowKeys != nil},
	}
	for _, k := range []string{"text", "items", "rows"} {
		if b := body[k]; b.kind != s.Kind && b.set {
			return Screen{}, bad(have[k], "%s belongs to kind %q, not to kind %q", k, b.kind, s.Kind)
		}
	}
	for _, k := range []string{"text", "items", "rows"} {
		if b := body[k]; b.kind == s.Kind && !b.set {
			return Screen{}, bad(t.line, "kind %q needs %s", s.Kind, k)
		}
	}
	resolve := func(key string, line int, what string, maxRunes int) (string, error) {
		v, ok := text(key)
		if !ok {
			return "", bad(line, "text key %q is not in %s (nor in %s)", key, StringsFile(lang), StringsFile("en"))
		}
		if n := utf8.RuneCountInString(strings.TrimSpace(v)); n == 0 || n > maxRunes {
			return "", bad(line, "the text of %q has %d characters; %s has 1 to %d", key, n, what, maxRunes)
		}
		return v, nil
	}
	var err error
	if s.Title, err = resolve(titleKey, have["title"], "a screen title", maxScreenTitleRunes); err != nil {
		return Screen{}, err
	}
	if s.MenuLabel, err = resolve(menuKey, have["menu"], "a menu entry", maxScreenMenuRunes); err != nil {
		return Screen{}, err
	}
	for _, k := range paragraphKeys {
		v, err := resolve(k, have["text"], "a paragraph", maxScreenTextRunes)
		if err != nil {
			return Screen{}, err
		}
		s.Paragraphs = append(s.Paragraphs, v)
	}
	for _, k := range itemKeys {
		v, err := resolve(k, have["items"], "a list line", maxScreenTextRunes)
		if err != nil {
			return Screen{}, err
		}
		s.Items = append(s.Items, v)
	}
	for _, r := range rowKeys {
		label, err := resolve(r[0], have["rows"], "a row label", maxScreenTitleRunes)
		if err != nil {
			return Screen{}, err
		}
		value, err := resolve(r[1], have["rows"], "a row value", maxScreenTitleRunes)
		if err != nil {
			return Screen{}, err
		}
		s.Rows = append(s.Rows, [2]string{label, value})
	}
	return s, nil
}

// actionFrom reads one [[action]] table: a menu entry that runs an action of
// the game. It takes `key`, `action`, `menu` (the entry's label) and an optional
// `confirm` (the label of the confirming row, the entry's label by default).
// Every text is a key of the mod's strings file.
func actionFrom(id, file string, t tomlTable, text func(string) (string, bool), lang string) (Screen, error) {
	bad := func(line int, format string, args ...any) error {
		return &ItemFileError{File: file, Line: line, Msg: fmt.Sprintf(format, args...)}
	}
	s := Screen{Mod: id, File: file, Line: t.line, Game: true}
	have := map[string]int{}
	var menuKey, confirmKey string
	for _, p := range t.pairs {
		have[p.key] = p.line
		if p.val.kind != tomlString {
			return Screen{}, bad(p.line, "%s must be a string, not %s", p.key, p.val.kind)
		}
		switch p.key {
		case "key":
			if s.Key = p.val.str; !itemKeyPattern.MatchString(s.Key) {
				return Screen{}, bad(p.line, "key %q must be 1 to 32 of a-z, 0-9 and '_', starting with a letter", s.Key)
			}
		case "action":
			switch p.val.str {
			case ActionAbandon, ActionRestart:
				s.Kind = p.val.str
			default:
				return Screen{}, bad(p.line, "unknown action %q (actions: %s, %s)", p.val.str, ActionAbandon, ActionRestart)
			}
		case "menu":
			menuKey = p.val.str
		case "confirm":
			confirmKey = p.val.str
		case "place":
			return Screen{}, bad(p.line, "an [[action]] is always in the in-game menu over a mission; it takes no place")
		default:
			return Screen{}, bad(p.line, "unknown key %q in [[action]]", p.key)
		}
	}
	for _, need := range []string{"key", "action", "menu"} {
		if _, ok := have[need]; !ok {
			return Screen{}, bad(t.line, "[[action]] has no %s", need)
		}
	}
	resolve := func(key string, line int, what string) (string, error) {
		v, ok := text(key)
		if !ok {
			return "", bad(line, "text key %q is not in %s (nor in %s)", key, StringsFile(lang), StringsFile("en"))
		}
		if n := utf8.RuneCountInString(strings.TrimSpace(v)); n == 0 || n > maxActionMenuRunes {
			return "", bad(line, "the text of %q has %d characters; %s has 1 to %d", key, n, what, maxActionMenuRunes)
		}
		return v, nil
	}
	var err error
	if s.MenuLabel, err = resolve(menuKey, have["menu"], "a menu entry"); err != nil {
		return Screen{}, err
	}
	s.Title = s.MenuLabel
	if confirmKey != "" {
		if s.Title, err = resolve(confirmKey, have["confirm"], "a confirming row"); err != nil {
			return Screen{}, err
		}
	}
	return s, nil
}
