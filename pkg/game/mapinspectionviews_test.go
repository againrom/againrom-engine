package game

import (
	"bytes"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
	"againrom/pkg/render/text"
	"againrom/pkg/ui"
)

// viewsMap is the person-window fixture with a ground sack of a health
// potion and a sword, whose potion icon the install holds. It returns the
// host and the map bytes.
func viewsMap(t *testing.T) (*mapInspectionEdits, []byte) {
	t.Helper()
	f, s, _ := nativeInventoryEditor(t)
	sack, err := s.model.Place(mapedit.KindLoot, s.model.GroundSack(20*256+128, 20*256+128, 0x0e08))
	if err != nil {
		t.Fatal(err)
	}
	if err := s.model.SetField(mapedit.KindLoot, sack, "additem", 0x101); err != nil {
		t.Fatal(err)
	}
	// The unit carries an item, so it has a stock its sack row opens.
	if _, err := s.SetField("Unit", 0, "carryadd", 0x102); err != nil {
		t.Fatal(err)
	}
	f.Archives.Containers.Overlay(graphicsPrefix+data.ItemIconPath(0x0e08), func() ([]byte, error) { return packIconStream(0xd0, 0x30, 0x10), nil })
	s.unitViews, s.inspector.recordOptions = nil, nil
	return s, s.model.Bytes()
}

// viewsCardFont is a card font of 224 records, each 5 by 10 pixels.
func viewsCardFont() *text.Font {
	f := &text.Font{Spacing: 1, Glyphs: make([]text.Glyph, 224)}
	for k := range f.Glyphs {
		f.Glyphs[k] = text.Glyph{Width: 5, Height: 10, Pixels: make([]text.Pixel, 5*10), Advance: 5}
	}
	return f
}

// viewsWindow opens the map in an editor whose profile chose the view,
// plain or technical, with the record of kind selected.
func viewsWindow(t *testing.T, s *mapInspectionEdits, plain bool, kind ui.RecordKind) *ui.MapEditor {
	t.Helper()
	doc, err := s.project(s.model)
	if err != nil {
		t.Fatal(err)
	}
	e := ui.NewMapEditor(nil, nil, func(string) (*ui.InspectionDocument, error) { return doc, nil })
	// The unit's basic view places its inputs where the card's font draws
	// each stat.
	e.SetCardFont(viewsCardFont())
	e.Layout(1280, 800)
	view := "technical"
	if plain {
		view = "plain"
	}
	e.SetPanelProfile([]byte(`{"right":{"minimized":true,"view":"`+view+`","width":520}}`), nil, nil)
	if err := e.Open("views.alm"); err != nil {
		t.Fatal(err)
	}
	at := slices.IndexFunc(doc.Records, func(r ui.InspectionRecord) bool { return r.Kind == kind })
	if at < 0 {
		t.Fatalf("no %s record", kind)
	}
	e.Select(at, false)
	if e.PlainView() != plain {
		t.Fatalf("%s window plain %v, want %v", kind, e.PlainView(), plain)
	}
	return e
}

// openBlocks opens every closed block of the plain window.
func openBlocks(t *testing.T, e *ui.MapEditor, c *editorClicker) {
	t.Helper()
	for range 8 {
		n := slices.IndexFunc(e.PopupLines(), func(line string) bool { return strings.HasPrefix(line, "+ ") })
		if n < 0 {
			return
		}
		e.ScrollDetails(-1000)
		r, ok := e.PopupRowRect(n)
		for step := 0; !ok && step < 80; step++ {
			e.ScrollDetails(1)
			r, ok = e.PopupRowRect(n)
		}
		c.click(e, r, ok, e.PopupLines()[n])
	}
	t.Fatal("a block does not open")
}

// showField brings field key's row into the window: its section, its parent
// slot's detail, the scroll that shows it. It reports whether the row
// shows; a read-only field shows as its first line, or as the row its link
// opens.
func showField(t *testing.T, e *ui.MapEditor, c *editorClicker, key string) bool {
	t.Helper()
	fields := e.ObjectFields()
	i := fieldIndex(t, fields, key)
	f := fields[i]
	if f.Parent != "" {
		if !showField(t, e, c, f.Parent) {
			return false
		}
		r, ok := e.FieldRect(fieldIndex(t, e.ObjectFields(), f.Parent))
		c.click(e, r, ok, f.Parent)
		_ = e.Key("escape")
	}
	if tab, ok := e.ObjectSectionRect(f.Section); ok {
		c.click(e, tab, true, "section "+f.Section)
	}
	// An editor with the host's encoder holds its rows in the font's code
	// page; a fixture editor without one holds the label as it stands.
	label := f.Label
	if host, ok := e.Document().Edits.(*mapInspectionEdits); ok {
		label = host.inspector.inspectionUTF8(label)
	}
	e.ScrollDetails(-1000)
	for range 80 {
		if f.ReadOnly {
			if len(f.Lines) > 0 && slices.Contains(e.PopupLines(), f.Lines[0].Text) {
				return true
			}
			if f.Link.Kind != "" && slices.ContainsFunc(e.PopupLines(), func(line string) bool { return strings.Contains(line, label) || strings.Contains(line, f.Label) }) {
				return true
			}
		} else if _, ok := e.FieldRect(i); ok {
			return true
		}
		e.ScrollDetails(1)
	}
	return false
}

// gameFieldShapes places every field shape the game's host gives the
// fixture's unit, structure and sack: the block of the plain view whose row
// edits it, "card" for an input on the unit's card, "" for a field only the
// detailed view shows. A shape the host adds fails here until the plain
// table and this table place it.
var gameFieldShapes = map[ui.RecordKind]map[string]string{
	ui.KindUnit: {
		"identity": "main", "name": "main", "owner": "main", "group": "main",
		"hp": "card", "mana": "card", "body": "card", "mind": "card", "spirit": "card", "reaction": "card",
		"defence": "card", "absorption": "card", "skill.*": "card", "protection.*": "card",
		"speed": "card", "sight": "card",
		"face": "", "sex": "",
		"gear": "", "wear#": "", "wearmaterial#": "", "weartier#": "", "wearrecipe#": "",
		"carryadd": "", "carry#": "", "carryrecipe#": "", "carryrecipe#.*": "", "resolved": "", "overrides": "",
	},
	ui.KindStructure: {
		"kind": "main", "owner": "main", "identity": "main", "width": "main", "height": "main",
		"placement": "placement",
	},
	ui.KindSack: {
		"gold": "main", "item#": "main", "additem": "main", "use#": "details", "recipe#": "details", "recipe#.*": "details",
	},
	ui.KindStock: {
		"gold": "main", "item#": "main", "additem": "main", "use#": "details", "recipe#": "details", "recipe#.*": "details",
	},
}

// unitHiddenField reports that f is one of the unit's fields neither view
// shows: its facing, its dialogue speaker and its map keys.
func unitHiddenField(kind ui.RecordKind, f ui.InspectionField) bool {
	return kind == ui.KindUnit && slices.Contains([]string{"facing", "speaker", "keys"}, f.Key)
}

// viewsShape is key with its first number as "#" and anything after a dot
// as "*".
func viewsShape(key string) string {
	if head, _, ok := strings.Cut(key, "."); ok {
		key = head + ".*"
	}
	start := strings.IndexAny(key, "0123456789")
	if start < 0 {
		return key
	}
	end := start
	for end < len(key) && key[end] >= '0' && key[end] <= '9' {
		end++
	}
	return key[:start] + "#" + key[end:]
}

// (c) Every field the game's host gives a unit, a structure, a sack and a
// unit's sack has its block in the plain view and shows there once the
// blocks open, or is a unit field the basic view leaves to the detailed
// view, and shows in the technical view in the section that holds it; a
// unit's facing, speaker and map keys show in neither unit view.
func TestEditorViewsPlaceEveryHostField(t *testing.T) {
	for _, kind := range []ui.RecordKind{ui.KindUnit, ui.KindStructure, ui.KindSack, ui.KindStock} {
		s, _ := viewsMap(t)
		c := &editorClicker{t: t}
		plain := viewsWindow(t, s, true, kind)
		blocks := plain.PlainFieldBlocks()
		openBlocks(t, plain, c)
		fields := plain.ObjectFields()
		if len(fields) == 0 {
			t.Fatalf("%s has no fields", kind)
		}
		for _, f := range fields {
			if unitHiddenField(kind, f) {
				if _, shown := blocks[f.Key]; shown || showField(t, plain, c, f.Key) {
					t.Errorf("%s %s: the unit's window edits its sack", kind, f.Key)
				}
				continue
			}
			want, listed := gameFieldShapes[kind][viewsShape(f.Key)]
			if got := blocks[f.Key]; !listed || got != want {
				t.Errorf("%s %s: plain block %q, the table %q (listed %v)", kind, f.Key, got, want, listed)
			}
			if want == "" {
				// The detailed view's own field: the basic view leaves it out.
				if showField(t, plain, c, f.Key) {
					t.Errorf("%s %s shows in the basic view", kind, f.Key)
				}
				continue
			}
			if !showField(t, plain, c, f.Key) {
				t.Errorf("%s %s does not show in the plain view", kind, f.Key)
			}
		}
		technical := viewsWindow(t, s, false, kind)
		for _, f := range technical.ObjectFields() {
			if shown := showField(t, technical, c, f.Key); shown == unitHiddenField(kind, f) {
				t.Errorf("%s %s shows %v in the technical view", kind, f.Key, shown)
			}
		}
	}
}

// typeField types value into field key's box and presses Apply.
func typeField(t *testing.T, e *ui.MapEditor, c *editorClicker, key string, value int64) {
	t.Helper()
	if !showField(t, e, c, key) {
		t.Fatalf("%s does not show", key)
	}
	r, ok := e.FieldRect(fieldIndex(t, e.ObjectFields(), key))
	c.click(e, r, ok, key)
	for range 16 {
		_ = e.Key("end")
		_ = e.Key("backspace")
	}
	e.TextInput(strconv.FormatInt(value, 10))
	r, ok = e.PopupButtonRect(0)
	c.click(e, r, ok, "Apply")
}

// chooseField opens field key's drop-down, chooses the row of the choice
// value as a map maker does, and presses Apply.
func chooseField(t *testing.T, e *ui.MapEditor, c *editorClicker, key string, value int64) {
	t.Helper()
	if !showField(t, e, c, key) {
		t.Fatalf("%s does not show", key)
	}
	n := fieldIndex(t, e.ObjectFields(), key)
	f := e.ObjectFields()[n]
	at := slices.IndexFunc(f.Choices, func(choice ui.InspectionChoice) bool { return choice.Value == value })
	if at < 0 {
		t.Fatalf("%s offers no choice %d", key, value)
	}
	if r, ok := e.FieldChoiceRect(n, at); ok {
		c.click(e, r, true, key)
		r, ok = e.PopupButtonRect(0)
		c.click(e, r, ok, "Apply")
		return
	}
	r, ok := e.FieldRect(n)
	c.click(e, r, ok, key)
	for _, group := range f.Groups {
		if !slices.Contains(group.Values, value) {
			continue
		}
		for _, label := range []string{group.Block + " >", group.Head + " >"} {
			row := slices.Index(e.DropdownChoices(), label)
			if row < 0 {
				t.Fatalf("%s's list has no group %q: %q", key, label, e.DropdownChoices())
			}
			e.ScrollDropdown(-len(f.Choices))
			r, ok = e.DropdownRowRect(row)
			for step := 0; !ok && step < len(f.Choices)+10; step++ {
				e.ScrollDropdown(1)
				r, ok = e.DropdownRowRect(row)
			}
			c.click(e, r, ok, label)
		}
		break
	}
	row := slices.Index(e.DropdownChoices(), f.Choices[at].Label)
	if row < 0 {
		t.Fatalf("%s's open list has no row %q: %q", key, f.Choices[at].Label, e.DropdownChoices())
	}
	e.ScrollDropdown(-len(f.Choices))
	r, ok = e.DropdownRowRect(row)
	for step := 0; !ok && step < len(f.Choices)+10; step++ {
		e.ScrollDropdown(1)
		r, ok = e.DropdownRowRect(row)
	}
	c.click(e, r, ok, fmt.Sprintf("%s's row %d", key, row))
	r, ok = e.PopupButtonRect(0)
	c.click(e, r, ok, "Apply")
}

// (d) The same edit typed in the plain view, in the technical view, and
// written straight through the host's field writer, which the window used
// before the plain view, gives the same map bytes; switching the view
// before Apply changes none of them. A unit's stat is typed in its input on
// the card.
func TestEditorViewsWriteTheSameBytes(t *testing.T) {
	for _, edit := range []struct {
		kind  ui.RecordKind
		key   string
		value int64
	}{{ui.KindUnit, "hp", 50}, {ui.KindUnit, "body", 31}, {ui.KindUnit, "protection.fire", 23}, {ui.KindStructure, "placement", 7}, {ui.KindSack, "gold", 77}} {
		var got [][]byte
		for _, mode := range []string{"plain", "technical", "switched"} {
			s, raw := viewsMap(t)
			c := &editorClicker{t: t}
			e := viewsWindow(t, s, mode != "technical", edit.kind)
			if mode == "plain" {
				openBlocks(t, e, c)
				if block := e.PlainFieldBlocks()[edit.key]; edit.kind == ui.KindUnit && block != "card" {
					t.Fatalf("the unit's %s stands in block %q, not on the card", edit.key, block)
				}
			}
			if mode == "switched" {
				r, ok := e.ViewSwitchRect()
				c.click(e, r, ok, "view switch")
				if e.PlainView() || !bytes.Equal(raw, s.model.Bytes()) {
					t.Fatalf("the switch wrote the map or kept the plain view")
				}
			}
			typeField(t, e, c, edit.key, edit.value)
			if bytes.Equal(raw, s.model.Bytes()) {
				t.Fatalf("%s %s %s: Apply wrote nothing; %q", mode, edit.kind, edit.key, e.PopupLines())
			}
			got = append(got, s.model.Bytes())
		}
		s, _ := viewsMap(t)
		doc, err := s.project(s.model)
		if err != nil {
			t.Fatal(err)
		}
		index := doc.Records[slices.IndexFunc(doc.Records, func(r ui.InspectionRecord) bool { return r.Kind == edit.kind })].Index
		if _, err := s.SetFields(edit.kind, index, []ui.InspectionFieldValue{{Key: edit.key, Value: edit.value, Step: 1}}); err != nil {
			t.Fatal(err)
		}
		for n, b := range got {
			if !bytes.Equal(b, s.model.Bytes()) {
				t.Errorf("%s %s: view %d wrote other bytes than the field writer", edit.kind, edit.key, n)
			}
		}
	}
}

// (e) A section and a trailer no field names, and every byte outside the
// edited field, keep their bytes across the view switch and an edit of
// another field.
func TestEditorViewsKeepUnknownBytes(t *testing.T) {
	b := inspectionFixtureBytes(t)
	_, e, _ := openTestMapEditor(t, b)
	e.SetPanelProfile(nil, os.ErrNotExist, nil)
	e.Select(0, false)
	if !e.PlainView() {
		t.Fatal("the unit window opened technical")
	}
	c := &editorClicker{t: t}
	host := e.Document().Edits.(*mapInspectionEdits)
	for range 3 {
		r, ok := e.ViewSwitchRect()
		c.click(e, r, ok, "view switch")
	}
	if !bytes.Equal(b, host.model.Bytes()) {
		t.Fatal("the view switch wrote the map")
	}
	typeField(t, e, c, "group", 4)
	after := host.model.Bytes()
	for _, unknown := range []string{"unknown physical payload", "uncounted trailer"} {
		if bytes.Count(after, []byte(unknown)) != 1 {
			t.Errorf("the edit lost %q", unknown)
		}
	}
	if len(after) != len(b) {
		t.Fatalf("the edit resized the map: %d bytes, was %d", len(after), len(b))
	}
	var changed []int
	for i := range b {
		if b[i] != after[i] {
			changed = append(changed, i)
		}
	}
	if len(changed) == 0 || changed[len(changed)-1]-changed[0] >= 4 {
		t.Fatalf("the edit changed bytes %v, more than the group field", changed)
	}
}

// (g) A sack item's name comes from the item-name builder and its icon from
// the inventory's icon reader; the plain view shows the name.
func TestEditorViewsNameAndPictureSackItems(t *testing.T) {
	s, _ := viewsMap(t)
	name, icon := s.ItemLook(0x0e08)
	want := mapload.ItemDisplayName(s.inspector.table, mapload.ItemSubject{Code: 0x0e08, Form: mapload.ItemNameEditorList})
	read, err := loadItemIcon(s.inspector.archives.artSource(), graphicsPrefix+data.ItemIconPath(0x0e08))
	if err != nil || name != want || !strings.Contains(name, "Health potion") || icon == nil || !bytes.Equal(icon.Pix, read.Pix) {
		t.Fatalf("ItemLook = %q %v, want %q and the potion icon (%v)", name, icon, want, err)
	}
	if again, second := s.ItemLook(0x0e08); again != name || second != icon {
		t.Fatal("ItemLook read the icon twice")
	}
	if _, none := s.ItemLook(0x0303); none != nil {
		t.Fatal("an item the install has no icon for has one")
	}
	e := viewsWindow(t, s, true, ui.KindSack)
	lines := e.PopupLines()
	if !slices.Contains(lines, "Item 1 "+want) {
		t.Fatalf("the plain sack does not name its potion: %q", lines)
	}
}
