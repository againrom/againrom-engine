package game

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"againrom/pkg/mod"
	"againrom/pkg/ui"
	"againrom/pkg/words"
)

// screensAddKey and screensActionKey are the Screens page's adders.
const (
	screensAddKey    = "newscreen"
	screensActionKey = "newaction"
)

// screenFiles are the screens file and the strings files its texts are in.
func screenFiles() []string { return append([]string{mod.ScreensFile}, textFiles()...) }

// screenStructure is the page's rowText: every text key reads as a text, so
// the page holds a screen whose text is missing or too long; the gate and the
// Texts page name that.
func (s *mapInspectionEdits) screenStructure(*editorModDocument) (func(key string) (string, bool), error) {
	return func(string) (string, bool) { return "x", true }, nil
}

// screenKinds are the kinds of a screen, in the order of their choice values.
var screenKinds = []string{mod.ScreenInfo, mod.ScreenList, mod.ScreenTable}

// screenActions are the actions of an action entry, in choice order.
var screenActions = []string{mod.ActionAbandon, mod.ActionRestart}

// screenPlaces are the menus a screen opens from, in choice order.
var screenPlaces = []string{mod.PlaceMain, mod.PlaceGame, mod.PlaceBoth}

// screenPlace is the index in screenPlaces of where s opens from.
func screenPlace(s mod.Screen) int {
	switch {
	case s.Main && s.Game:
		return 2
	case s.Game:
		return 1
	}
	return 0
}

// setScreenPlace makes s open from the menu screenPlaces[n] names.
func setScreenPlace(s *mod.Screen, n int) { s.Main, s.Game = n != 1, n != 0 }

// screensGate refuses a save while a screen of the project is one the game
// cannot show, naming the screen: a text key with no text, a text too long
// for its place, or more entries than a menu has slots.
func (s *mapInspectionEdits) screensGate() error {
	x := s.inspector
	d := s.workspace.Project
	if d == nil {
		return nil
	}
	src, ok := d.Files[mod.ScreensFile]
	if !ok {
		return nil
	}
	held, err := s.heldRows(d, s.screenStructure, mod.ScreensFile)
	if err != nil {
		return fmt.Errorf(x.word("editor.screens.err.save"), mod.ScreensFile, err)
	}
	for _, l := range textLanguages() {
		look, err := d.textLookup(l)
		if err != nil {
			return err
		}
		_, err = mod.ParseScreens(x.mod.id, mod.ScreensFile, src, look, l)
		if err == nil {
			continue
		}
		name, msg := mod.ScreensFile, err.Error()
		var item *mod.ItemFileError
		if errors.As(err, &item) {
			name, msg = mod.ScreensFile, item.Msg
			for _, sc := range held.Screens.Screens {
				if sc.Line <= item.Line {
					name = sc.Key
				}
			}
		}
		return fmt.Errorf(x.word("editor.screens.err.save"), name, fmt.Sprintf(x.word("editor.screens.err.lang"), languageLabel(l), msg))
	}
	return nil
}

// screenKeyChoices are the choices of a text key of the strings files sets
// hold: none (when offered), each key by its text, then cur when it is none
// of them. A value is the key's place in keys, from 1.
func (x *MapInspector) screenKeyChoices(keys []string, look func(string) (string, bool), cur string, none bool) []ui.InspectionChoice {
	var out []ui.InspectionChoice
	if none {
		out = append(out, ui.InspectionChoice{Value: 0, Label: x.word("editor.screens.confirm.none")})
	}
	label := func(k string) string {
		text, _ := look(k)
		if utf8.RuneCountInString(text) > 40 {
			text = string([]rune(text)[:40]) + "..."
		}
		return x.inspectionUTF8(k + ": " + text)
	}
	for i, k := range keys {
		out = append(out, ui.InspectionChoice{Value: int64(i + 1), Label: label(k)})
	}
	if cur != "" && !slices.Contains(keys, cur) {
		out = append(out, ui.InspectionChoice{Value: int64(len(keys) + 1), Label: x.inspectionUTF8(cur + ": " + x.word("editor.screens.notext"))})
	}
	return out
}

// screensFields is W's Screens page: the adders, then each screen's key,
// kind, menu, texts and removal.
func (s *mapInspectionEdits) screensFields() []ui.InspectionField {
	x := s.inspector
	if !s.modPageHolds() {
		return nil
	}
	project := s.workspace.Project
	if project == nil {
		project = x.mod.document()
	}
	held, err := s.heldRows(project, s.screenStructure, mod.ScreensFile)
	if err != nil {
		return []ui.InspectionField{x.pageLine("error", "", err.Error())}
	}
	strs, err := s.heldRows(project, nil, textFiles()...)
	if err != nil {
		return []ui.InspectionField{x.pageLine("error", "", err.Error())}
	}
	look, err := project.textLookup(x.pageLanguage())
	if err != nil {
		return []ui.InspectionField{x.pageLine("error", "", err.Error())}
	}
	keys := textKeys(strs.Strings)
	out := []ui.InspectionField{
		{Key: screensAddKey, Label: x.word("editor.screens.add"), Meaning: x.word("editor.screens.add.help"), Action: true, Column: 1},
		{Key: screensActionKey, Label: x.word("editor.screens.addaction"), Meaning: x.word("editor.screens.addaction.help"), Action: true, Column: 1}}
	pick := func(key, label, hint, cur string, none bool) ui.InspectionField {
		v := int64(0)
		if i := slices.Index(keys, cur); i >= 0 {
			v = int64(i + 1)
		} else if cur != "" {
			v = int64(len(keys) + 1)
		}
		return ui.InspectionField{Key: key, Label: label, Meaning: hint, Value: v, Choices: x.screenKeyChoices(keys, look, cur, none)}
	}
	names := func(ids ...string) []ui.InspectionChoice {
		out := make([]ui.InspectionChoice, len(ids))
		for i, w := range ids {
			out[i] = ui.InspectionChoice{Value: int64(i), Label: x.word(w)}
		}
		return out
	}
	remove := func(key, label string) ui.InspectionField {
		return ui.InspectionField{Key: key, Label: label, Meaning: x.word("editor.screens.removeline.help"), Action: true, Column: 1}
	}
	for i, sc := range held.Screens.Screens {
		key := "screen." + strconv.Itoa(i) + "."
		start := len(out)
		keyField := workspaceText(key+"key", x.word("editor.screens.key"), sc.Key)
		keyField.Meaning = x.word("editor.screens.key.help")
		out = append(out, keyField)
		if mod.IsAction(sc.Kind) {
			out = append(out, ui.InspectionField{Key: key + "action", Label: x.word("editor.screens.action"), Meaning: x.word("editor.screens.action.help"),
				Value: int64(slices.Index(screenActions, sc.Kind)), Choices: names(screenActionWords...)},
				pick(key+"menu", x.word("editor.screens.menu"), x.word("editor.screens.menu.help"), sc.MenuKey, false),
				pick(key+"confirm", x.word("editor.screens.confirm"), x.word("editor.screens.confirm.help"), sc.ConfirmKey, true))
		} else {
			out = append(out, ui.InspectionField{Key: key + "kind", Label: x.word("editor.screens.kind"), Meaning: x.word("editor.screens.kind.help"),
				Value: int64(slices.Index(screenKinds, sc.Kind)), Choices: names(screenKindWords...)},
				ui.InspectionField{Key: key + "place", Label: x.word("editor.screens.place"), Meaning: x.word("editor.screens.place.help"),
					Value: int64(screenPlace(sc)), Choices: names(screenPlaceWords...)},
				pick(key+"title", x.word("editor.screens.title"), x.word("editor.screens.title.help"), sc.TitleKey, false),
				pick(key+"menu", x.word("editor.screens.menu"), x.word("editor.screens.menu.help"), sc.MenuKey, false))
			switch sc.Kind {
			case mod.ScreenInfo:
				for j, k := range sc.TextKeys {
					n := strconv.Itoa(j + 1)
					out = append(out, pick(key+"text."+n, fmt.Sprintf(x.word("editor.screens.paragraph"), j+1), x.word("editor.screens.body.help"), k, false),
						remove(key+"text."+n+".remove", fmt.Sprintf(x.word("editor.screens.removeparagraph"), j+1)))
				}
			case mod.ScreenList:
				for j, k := range sc.ItemKeys {
					n := strconv.Itoa(j + 1)
					out = append(out, pick(key+"items."+n, fmt.Sprintf(x.word("editor.screens.line"), j+1), x.word("editor.screens.body.help"), k, false),
						remove(key+"items."+n+".remove", fmt.Sprintf(x.word("editor.screens.removeline"), j+1)))
				}
			case mod.ScreenTable:
				for j, r := range sc.RowKeys {
					n := strconv.Itoa(j + 1)
					out = append(out, pick(key+"rows."+n+".0", fmt.Sprintf(x.word("editor.screens.rowlabel"), j+1), x.word("editor.screens.body.help"), r[0], false),
						pick(key+"rows."+n+".1", fmt.Sprintf(x.word("editor.screens.rowvalue"), j+1), x.word("editor.screens.body.help"), r[1], false),
						remove(key+"rows."+n+".remove", fmt.Sprintf(x.word("editor.screens.removerow"), j+1)))
				}
			}
			out = append(out, ui.InspectionField{Key: key + "addline", Label: x.word("editor.screens.addline." + sc.Kind), Meaning: x.word("editor.screens.addline.help"), Action: true, Column: 1})
		}
		out = append(out, ui.InspectionField{Key: key + "remove", Label: x.word("editor.screens.remove"), Meaning: x.word("editor.screens.remove.help"), Action: true, Column: 1})
		for k := start; k < len(out); k++ {
			out[k].Group = sc.Key
		}
	}
	return out
}

// setScreensField makes one edit of the Screens page a draft step.
func (s *mapInspectionEdits) setScreensField(key, value string) (*ui.InspectionDocument, error) {
	if err := s.prepareWorkspace(); err != nil {
		return nil, err
	}
	x := s.inspector
	if err := x.modWritable(); err != nil {
		return nil, err
	}
	value = strings.TrimSpace(value)
	gone := fmt.Errorf(x.word("editor.screens.err.gone"), key)
	return s.editKind(func(k kindFiles) error {
		return k.editRows(s.screenStructure, screenFiles(), func(d *mod.Data) error {
			switch key {
			case screensAddKey, screensActionKey:
				return s.addScreen(d, key == screensActionKey)
			}
			kind, rest, _ := strings.Cut(key, ".")
			index, field, _ := strings.Cut(rest, ".")
			i, err := strconv.Atoi(index)
			if err != nil || i < 0 || kind != "screen" || i >= len(d.Screens.Screens) {
				return gone
			}
			return s.editScreen(d, i, field, value, gone)
		})
	})
}

// screenValues are the text a new text key holds in each language.
func screenValues(what string) map[string]string {
	values := map[string]string{}
	for _, l := range textLanguages() {
		values[l] = words.For(l).Text(screenNewWords[what])
	}
	return values
}

// addScreen appends an information screen of a key none holds, or a menu
// action no screen holds, with new texts of its own in each language.
func (s *mapInspectionEdits) addScreen(d *mod.Data, action bool) error {
	x := s.inspector
	list := d.Screens.Screens
	taken := func(k string) bool { return slices.ContainsFunc(list, func(o mod.Screen) bool { return o.Key == k }) }
	if action {
		kind := ""
		for _, a := range screenActions {
			if kind == "" && !slices.ContainsFunc(list, func(o mod.Screen) bool { return o.Kind == a }) {
				kind = a
			}
		}
		if kind == "" {
			return errors.New(x.word("editor.screens.err.noaction"))
		}
		key := mod.RecordKey(kind, taken)
		d.Screens.Screens = append(list, mod.Screen{Key: key, Kind: kind, Game: true,
			MenuKey: newModText(d.Strings, "screen."+key+".menu", screenValues("action"))})
		return nil
	}
	main, game := d.Screens.Count()
	sc := mod.Screen{Key: mod.RecordKey("screen", taken), Kind: mod.ScreenInfo}
	switch {
	case main < mod.MaxMainMenuScreens:
		sc.Main = true
	case game < mod.MaxGameMenuScreens:
		sc.Game = true
	default:
		return fmt.Errorf(x.word("editor.screens.err.slots"), mod.MaxMainMenuScreens, mod.MaxGameMenuScreens)
	}
	sc.TitleKey = newModText(d.Strings, "screen."+sc.Key+".title", screenValues("title"))
	sc.MenuKey = newModText(d.Strings, "screen."+sc.Key+".menu", screenValues("menu"))
	sc.TextKeys = []string{newModText(d.Strings, "screen."+sc.Key+".text", screenValues("text"))}
	d.Screens.Screens = append(list, sc)
	return nil
}

// editScreen makes the edit of field of screen i. A text field's value is a
// place in the project's text keys, from 1, 0 for none.
func (s *mapInspectionEdits) editScreen(d *mod.Data, i int, field, value string, gone error) error {
	x := s.inspector
	sc := &d.Screens.Screens[i]
	if field == "remove" {
		d.Screens.Screens = slices.Delete(d.Screens.Screens, i, i+1)
		return nil
	}
	if field == "key" {
		sc.Key = value
		return nil
	}
	n, numErr := strconv.Atoi(value)
	keys := textKeys(d.Strings)
	// textKey is the key the choice value n names, and whether the value
	// names the key shown for a key the strings lack, which changes nothing.
	textKey := func() (key string, same bool, err error) {
		switch {
		case numErr != nil || n < 0 || n > len(keys)+1:
			return "", false, gone
		case n == 0:
			return "", false, nil
		case n == len(keys)+1:
			return "", true, nil
		}
		return keys[n-1], false, nil
	}
	switch field {
	case "kind":
		if numErr != nil || n < 0 || n >= len(screenKinds) || mod.IsAction(sc.Kind) {
			return gone
		}
		changeScreenKind(sc, screenKinds[n])
	case "action":
		if numErr != nil || n < 0 || n >= len(screenActions) || !mod.IsAction(sc.Kind) {
			return gone
		}
		sc.Kind = screenActions[n]
	case "place":
		if numErr != nil || n < 0 || n >= len(screenPlaces) || mod.IsAction(sc.Kind) {
			return gone
		}
		setScreenPlace(sc, n)
	case "title", "menu", "confirm":
		k, same, err := textKey()
		if err != nil {
			return err
		}
		switch {
		case same:
		case field == "title":
			sc.TitleKey = k
		case field == "menu":
			sc.MenuKey = k
		default:
			sc.ConfirmKey = k
		}
		if k == "" && !same && field != "confirm" {
			return errors.New(x.word("editor.screens.err.needtext"))
		}
	case "addline":
		return s.addScreenLine(d, sc)
	default:
		return s.editScreenBody(sc, field, textKey, gone)
	}
	return nil
}

// changeScreenKind makes sc a screen of kind, its body the keys it held, a
// table row naming each key as its label and its value.
func changeScreenKind(sc *mod.Screen, kind string) {
	var keys []string
	switch sc.Kind {
	case mod.ScreenInfo:
		keys = sc.TextKeys
	case mod.ScreenList:
		keys = sc.ItemKeys
	default:
		for _, r := range sc.RowKeys {
			keys = append(keys, r[0])
		}
	}
	sc.Kind, sc.TextKeys, sc.ItemKeys, sc.RowKeys = kind, nil, nil, nil
	switch kind {
	case mod.ScreenInfo:
		sc.TextKeys = keys
	case mod.ScreenList:
		sc.ItemKeys = keys
	default:
		for _, k := range keys {
			sc.RowKeys = append(sc.RowKeys, [2]string{k, k})
		}
	}
}

// addScreenLine appends a paragraph, a line or a row with new texts.
func (s *mapInspectionEdits) addScreenLine(d *mod.Data, sc *mod.Screen) error {
	text := func(what string) string {
		return newModText(d.Strings, "screen."+sc.Key+"."+what, screenValues(what))
	}
	switch sc.Kind {
	case mod.ScreenInfo:
		sc.TextKeys = append(slices.Clone(sc.TextKeys), text("text"))
	case mod.ScreenList:
		sc.ItemKeys = append(slices.Clone(sc.ItemKeys), text("text"))
	case mod.ScreenTable:
		sc.RowKeys = append(slices.Clone(sc.RowKeys), [2]string{text("label"), text("value")})
	default:
		return errors.New(s.inspector.word("editor.screens.err.noline"))
	}
	return nil
}

// editScreenBody makes the edit of a body entry field names: text.N,
// items.N or rows.N.0 and rows.N.1, each with a .remove of the entry.
func (s *mapInspectionEdits) editScreenBody(sc *mod.Screen, field string, textKey func() (string, bool, error), gone error) error {
	x := s.inspector
	parts := strings.Split(field, ".")
	if len(parts) < 2 {
		return gone
	}
	j, err := strconv.Atoi(parts[1])
	if err != nil || j < 1 {
		return gone
	}
	j--
	removing := parts[len(parts)-1] == "remove"
	var keys *[]string
	switch {
	case parts[0] == "text" && sc.Kind == mod.ScreenInfo:
		keys = &sc.TextKeys
	case parts[0] == "items" && sc.Kind == mod.ScreenList:
		keys = &sc.ItemKeys
	case parts[0] == "rows" && sc.Kind == mod.ScreenTable:
		if j >= len(sc.RowKeys) {
			return gone
		}
		sc.RowKeys = slices.Clone(sc.RowKeys)
		if removing {
			if len(sc.RowKeys) == 1 {
				return errors.New(x.word("editor.screens.err.last"))
			}
			sc.RowKeys = slices.Delete(sc.RowKeys, j, j+1)
			return nil
		}
		k, same, err := textKey()
		if err != nil || k == "" && !same {
			return cmpErr(err, x.word("editor.screens.err.needtext"))
		}
		if !same && len(parts) == 3 && (parts[2] == "0" || parts[2] == "1") {
			sc.RowKeys[j][int(parts[2][0]-'0')] = k
			return nil
		}
		if same {
			return nil
		}
		return gone
	default:
		return gone
	}
	if j >= len(*keys) {
		return gone
	}
	list := slices.Clone(*keys)
	if removing {
		if len(list) == 1 {
			return errors.New(x.word("editor.screens.err.last"))
		}
		*keys = slices.Delete(list, j, j+1)
		return nil
	}
	k, same, err := textKey()
	if err != nil || k == "" && !same {
		return cmpErr(err, x.word("editor.screens.err.needtext"))
	}
	if !same {
		list[j] = k
		*keys = list
	}
	return nil
}

// cmpErr is err when it is one, else the refusal with text.
func cmpErr(err error, text string) error {
	if err != nil {
		return err
	}
	return errors.New(text)
}
