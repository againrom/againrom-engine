package ui

import (
	"fmt"
	"hash/fnv"
	"image"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"againrom/internal/witness"
)

// plainEditor is viewsEditor after the editor profile profile loads, with
// the record selected; it returns each profile the editor wrote.
func plainEditor(t *testing.T, lang string, profile []byte, record int) (*MapEditor, *viewsHost, *[][]byte) {
	t.Helper()
	e, h := viewsEditor(t, lang)
	saved := &[][]byte{}
	readErr := error(nil)
	if profile == nil {
		readErr = os.ErrNotExist
	}
	e.SetPanelProfile(profile, readErr, func(b []byte) error {
		*saved = append(*saved, append([]byte(nil), b...))
		return nil
	})
	e.Layout(1280, 800)
	e.Select(record, false)
	return e, h, saved
}

// plainRowsText is the shown window's rows, one per line: the row's kind,
// its fold mark, label and value.
func plainRowsText(e *MapEditor) []string {
	s, _, _ := e.objectSpec()
	var out []string
	for _, r := range s.Rows {
		out = append(out, strings.TrimSpace(fmt.Sprintf("%d %s | %s", r.Kind, markedLabel(r), r.Value)))
	}
	return out
}

// markedLabel is a row's label after its fold mark, as PopupLines writes it.
func markedLabel(r popupRow) string {
	if r.Fold == FoldNone {
		return r.Label
	}
	return rowFoldMarks[r.Fold] + " " + r.Label
}

// viewsNow is the time of the next click; each click is a second later,
// so no two make a double click.
var viewsNow = time.Unix(1, 0)

// viewsClick clicks the middle of r.
func viewsClick(t *testing.T, e *MapEditor, r image.Rectangle, ok bool, what string) {
	t.Helper()
	if !ok {
		t.Fatalf("%s is not shown", what)
	}
	viewsNow = viewsNow.Add(time.Second)
	editorClickAt(e, image.Pt((r.Min.X+r.Max.X)/2, (r.Min.Y+r.Max.Y)/2), viewsNow)
}

// clickRow presses the window's row whose label is label.
func clickRow(t *testing.T, e *MapEditor, label string) {
	t.Helper()
	s, _, _ := e.objectSpec()
	n := slices.IndexFunc(s.Rows, func(r popupRow) bool { return markedLabel(r) == label })
	r, ok := e.PopupRowRect(n)
	viewsClick(t, e, r, n >= 0 && ok, fmt.Sprintf("row %q among %q", label, plainRowsText(e)))
}

// clickSwitch presses the view switch.
func clickSwitch(t *testing.T, e *MapEditor) {
	t.Helper()
	r, ok := e.ViewSwitchRect()
	viewsClick(t, e, r, ok, "the view switch")
}

// inheritNote is the note under each group of inheriting stats.
const inheritNote = "An empty field inherits. 0 cannot be set: the map stores 0 as inherited."

// structureRows are the fixture structure's plain rows: the main block and
// the script action naming it, then tail.
func structureRows(tail ...string) []string {
	return append([]string{
		"4 Position | 20,20  2x2",
		"3 Type | 51: Hall",
		"3 Owner | Self",
		"2 Script target | 3",
		"2 Width | 0",
		"2 Height | 0",
		"0 Used by |",
		"5 Trigger 1  Raid: Action 1  Burn the hall |",
	}, tail...)
}

// sackRows are the fixture sack's main block, each item by its name, then
// tail.
func sackRows(tail ...string) []string {
	return append([]string{
		"4 Position | 14,12  1x1",
		"2 Gold | 30",
		"2 Item 1 | Sword",
		"2 Item 2 | Mace",
		"2 Add an item | 0",
	}, tail...)
}

// (b) Each kind's plain view: its rows in the table's order and blocks,
// under the plain words, from the fixture's fields. The structure's and the
// sack's folded blocks are shown closed, then open.
func TestMapEditorPlainViewRowsOfEachKind(t *testing.T) {
	for _, c := range []struct {
		name   string
		record int
		open   []string
		want   []string
	}{
		{"unit", viewsUnit, nil, unitRows()},
		{"structure", viewsStructure, nil, structureRows("1 + Placement word |", "1 + Data |")},
		{"structure open", viewsStructure, []string{"+ Placement word", "+ Data"}, structureRows(
			"1 - Placement word |",
			"2 Word | 5",
			"0 Hex 0x0005; bits set: 0, 2 |",
			"0 Not 0: the building starts with its type's health. |",
			"1 - Data |",
			"0 Hall |",
		)},
		{"sack", viewsSack, nil, sackRows("1 + Details |", "1 + Data |")},
		{"sack open", viewsSack, []string{"+ Details"}, sackRows(
			"1 - Details |",
			"0 Item 1 id | 257",
			"2 Item 1 native dispatch | 0",
			"2 Item 1 recipe and enchantment | 1",
			"0 Item 2 id | 258",
			"2 Item 2 native dispatch | 0",
			"2 Item 2 recipe and enchantment | 0",
			"1 + Data |",
		)},
	} {
		t.Run(c.name, func(t *testing.T) {
			e, _, _ := plainEditor(t, "en", minimizedPanels, c.record)
			for _, label := range c.open {
				clickRow(t, e, label)
			}
			if got := plainRowsText(e); !slices.Equal(got, c.want) {
				t.Errorf("rows\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(c.want, "\n"))
			}
		})
	}
}

// unitRows are the fixture unit's basic rows: the name, the player, the
// group, the kind and the tier, then a row on the card for each value the
// card draws, in the card's order: the field's input where the fixture has
// the field, else a line stating the value, then a line of words for a stat
// whose card value differs from the stored one.
func unitRows() []string {
	return []string{
		"4 Position | 10,10  1x1",
		"0 Name | Guard",
		"3 Player | Self",
		"2 Group | 2",
		"3 Kind | 300: Fighter",
		"2 Tier | 1",
		"0  | Guard",
		"2 Body | 12",
		"0 AGILITY | 0",
		"2 Health | 40",
		"2 Mind |",
		"0 SPIRIT | 0",
		"2 Mana |",
		"0 BLADE | 0",
		"2 Fire protection |",
		"0 AXE | 0",
		"0 WATER | 0",
		"0 BLUDGEON | 0",
		"0 AIR | 0",
		"0 PIKE | 0",
		"0 EARTH | 0",
		"0 SHOOTING | 0",
		"0 ASTRAL | 0",
		"0 WEIGHT | 0.0",
		"0 XP | 1200",
		"0 SIGHT | 0.0",
		"0 BODY: Derived by the game from the unit; in game: 10 |",
	}
}

// plainShapes places every field shape of the fixture: the block of the
// plain view whose row edits it. A shape this table lacks fails until the
// plain table and this table place it.
var plainShapes = map[RecordKind]map[string]string{
	// A unit's field the basic view leaves out maps to "": it stands in the
	// detailed view only. The fixture's skill.fire is no skill the card
	// draws.
	KindUnit: {
		"owner": "main", "class": "main", "tier": "main", "group": "main",
		"hp": "card", "mana": "card", "body": "card", "mind": "card", "protection.*": "card",
		"face": "", "sex": "", "gear": "", "wear#": "", "wearmaterial#": "", "wearrecipe#": "",
		"sack": "", "holdings": "", "skill.*": "", "resolved": "", "definition": "",
	},
	KindStructure: {
		"kind": "main", "owner": "main", "identity": "main", "width": "main", "height": "main",
		"placement": "placement",
	},
	KindSack: {
		"gold": "main", "item#": "main", "additem": "main", "use#": "details", "recipe#": "details",
	},
}

// fieldShape is key with its first number as "#" and anything after a dot
// as "*".
func fieldShape(key string) string {
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

// (c) Every field of the fixture's unit, structure and sack is placed in a
// block of the plain view, and every field is reachable in the technical
// view: its row stands in the window in the section that holds it. A field
// of the unit's sack is the sack's own window's.
func TestMapEditorEveryFieldHasAPlace(t *testing.T) {
	for _, record := range []int{viewsUnit, viewsStructure, viewsSack} {
		e, _, _ := plainEditor(t, "en", minimizedPanels, record)
		kind := e.doc.Records[record].Kind
		blocks := e.PlainFieldBlocks()
		for _, f := range e.ObjectFields() {
			if sackField(e.doc.Records[record], f) {
				if _, shown := blocks[f.Key]; shown {
					t.Errorf("%s %s: the unit's window edits its sack", kind, f.Key)
				}
				continue
			}
			want, listed := plainShapes[kind][fieldShape(f.Key)]
			if got := blocks[f.Key]; !listed || got != want {
				t.Errorf("%s %s: plain block %q, the table %q (listed %v)", kind, f.Key, got, want, listed)
			}
			// A unit's field in no block of the basic view has no row there.
			if i := slices.IndexFunc(e.ObjectFields(), func(g InspectionField) bool { return g.Key == f.Key }); kind == KindUnit && (want == "") != (e.fieldRow(i) < 0) {
				t.Errorf("%s %s: block %q, basic view row %d", kind, f.Key, want, e.fieldRow(i))
			}
		}
		clickSwitch(t, e)
		if e.PlainView() {
			t.Fatal("the switch kept the plain view")
		}
		for i, f := range e.ObjectFields() {
			if sackField(e.doc.Records[record], f) {
				continue
			}
			if tab, ok := e.ObjectSectionRect(f.Section); ok {
				viewsClick(t, e, tab, ok, "section "+f.Section)
			}
			e.obj.detail = f.Parent
			_, ok := e.FieldRect(i)
			if f.ReadOnly {
				ok = len(f.Lines) > 0 && slices.Contains(e.PopupLines(), f.Lines[0].Text)
			}
			if !ok {
				t.Errorf("%s %s has no row in the technical view", kind, f.Key)
			}
			e.obj.detail = ""
		}
	}
}

// (d) The switch changes no data: a value typed in the plain view stays in
// the draft across the switch, and Apply in either view writes the one
// value, as the window wrote it before.
func TestMapEditorBothViewsWriteTheSameEdit(t *testing.T) {
	var writes [][]string
	for _, switchBefore := range []bool{false, true} {
		e, h, _ := plainEditor(t, "en", minimizedPanels, viewsStructure)
		fields := e.ObjectFields()
		i := slices.IndexFunc(fields, func(f InspectionField) bool { return f.Key == "width" })
		r, ok := e.FieldRect(i)
		viewsClick(t, e, r, ok, "the plain width row")
		for range 4 {
			_ = e.Key("backspace")
		}
		e.TextInput("3")
		if switchBefore {
			clickSwitch(t, e)
			clickSwitch(t, e)
			clickSwitch(t, e)
			if e.PlainView() || e.obj.buf[i] != "3" || len(h.writes) != 0 {
				t.Fatalf("the switch changed the draft: plain %v, width %q, writes %q", e.PlainView(), e.obj.buf[i], h.writes)
			}
		}
		r, ok = e.PopupButtonRect(0)
		viewsClick(t, e, r, ok, "Apply")
		writes = append(writes, h.writes)
	}
	if want := []string{"Structure width=3"}; !slices.Equal(writes[0], want) || !slices.Equal(writes[1], want) {
		t.Fatalf("plain Apply wrote %q, technical Apply %q, want %q", writes[0], writes[1], want)
	}
}

// (f) The view is kept in the editor profile beside the side panels. A
// missing, broken or older profile opens the plain view; an editor without
// a profile shows the technical view, the window as it was.
func TestMapEditorViewChoiceRoundTrip(t *testing.T) {
	for _, c := range []struct {
		name    string
		profile []byte
		plain   bool
	}{
		{"missing", nil, true},
		{"broken", []byte(`{"right":`), true},
		{"older", minimizedPanels, true},
		{"technical", []byte(`{"right":{"minimized":true,"view":"technical","width":520}}`), false},
		{"plain", []byte(`{"right":{"minimized":true,"view":"plain","width":520}}`), true},
		{"unknown word", []byte(`{"right":{"minimized":true,"view":"tabs","width":520}}`), true},
	} {
		e, _, _ := plainEditor(t, "en", c.profile, viewsUnit)
		if e.PlainView() != c.plain {
			t.Errorf("%s: plain %v, want %v", c.name, e.PlainView(), c.plain)
		}
	}
	bare, _ := viewsEditor(t, "en")
	bare.Select(viewsUnit, false)
	if bare.PlainView() {
		t.Fatal("an editor without a profile shows the plain view")
	}
	e, _, saved := plainEditor(t, "en", nil, viewsUnit)
	clickSwitch(t, e)
	if len(*saved) == 0 || !strings.Contains(string((*saved)[len(*saved)-1]), `"view":"technical"`) {
		t.Fatalf("the switch saved %q", *saved)
	}
	last := (*saved)[len(*saved)-1]
	f, _, again := plainEditor(t, "en", last, viewsUnit)
	if f.PlainView() || f.panels.state != e.panels.state {
		t.Fatalf("read back plain %v panels %v from %q", f.PlainView(), f.panels.state, last)
	}
	clickSwitch(t, f)
	if !f.PlainView() || len(*again) == 0 || !strings.Contains(string((*again)[len(*again)-1]), `"view":"plain"`) {
		t.Fatalf("the switch back: plain %v, saved %q", f.PlainView(), *again)
	}
}

// The switch names the shown view Basic or Detailed, in English and
// Russian, at 1920x1080; a profile that kept the technical view's word
// opens the detailed view.
func TestMapEditorViewSwitchNamesBasicAndDetailed(t *testing.T) {
	for _, c := range []struct{ lang, basic, detailed string }{
		{"en", "View: Basic", "View: Detailed"},
		{"ru", "Вид: Обычный", "Вид: Детальный"},
	} {
		for _, record := range []int{viewsUnit, viewsStructure, viewsSack} {
			e, _, _ := plainEditor(t, c.lang, minimizedPanels, record)
			e.Layout(1920, 1080)
			caption := func() string {
				s, _, _ := e.objectSpec()
				return s.TitleButton
			}
			if got := caption(); got != e.editorHostText(c.basic) || !e.PlainView() {
				t.Errorf("%s record %d: switch %q plain %v, want %q", c.lang, record, got, e.PlainView(), c.basic)
			}
			clickSwitch(t, e)
			if got := caption(); got != e.editorHostText(c.detailed) || e.PlainView() {
				t.Errorf("%s record %d: switched %q plain %v, want %q", c.lang, record, got, e.PlainView(), c.detailed)
			}
		}
		old, _, _ := plainEditor(t, c.lang, []byte(`{"right":{"minimized":true,"view":"technical","width":520}}`), viewsUnit)
		old.Layout(1920, 1080)
		if s, _, _ := old.objectSpec(); s.TitleButton != old.editorHostText(c.detailed) || old.PlainView() {
			t.Errorf("%s: the technical profile opens %q plain %v, want %q", c.lang, s.TitleButton, old.PlainView(), c.detailed)
		}
	}
}

// (g) A sack's items show the name the host's item-name builder gives and
// the icon the host reads for the code.
func TestMapEditorPlainSackShowsItemNamesAndIcons(t *testing.T) {
	e, h, _ := plainEditor(t, "en", minimizedPanels, viewsSack)
	s, _, _ := e.objectSpec()
	for _, c := range []struct {
		label, name string
		code        int64
	}{{"Item 1", "Sword", 0x101}, {"Item 2", "Mace", 0x102}} {
		n := slices.IndexFunc(s.Rows, func(r popupRow) bool { return r.Label == c.label })
		if n < 0 {
			t.Fatalf("no %s row", c.label)
		}
		name, icon := h.ItemLook(c.code)
		row := s.Rows[n]
		if row.Value != name || name != c.name || len(row.Icons) != 1 || row.Icons[0] == nil || row.Icons[0].RGBAAt(0, 0) != icon.RGBAAt(0, 0) {
			t.Errorf("%s: value %q icons %d, want %q and its icon", c.label, row.Value, len(row.Icons), c.name)
		}
	}
	clickSwitch(t, e)
	if s, _, _ := e.objectSpec(); slices.ContainsFunc(s.Rows, func(r popupRow) bool { return len(r.Icons) > 0 }) {
		t.Fatal("the technical view shows item icons")
	}
}

const plainViewsGolden = "testdata/mapeditor_plain_views.golden"

// The plain view of a unit, a structure and a sack at 1920x1080 in English
// and Russian, in the right panel and floating, and the structure with its
// placement block open. MAPEDITOR_VIEWS_WRITE=1 rewrites the record;
// AGAINROM_VIEWS_PNG names a folder for the pictures.
func TestMapEditorPlainViewGolden(t *testing.T) {
	var out strings.Builder
	for _, lang := range []string{"en", "ru"} {
		for _, place := range []string{"panel", "floating"} {
			profile := []byte(nil)
			if place == "floating" {
				profile = minimizedPanels
			}
			for _, c := range []struct {
				name   string
				record int
				open   bool
			}{{"unit", viewsUnit, false}, {"structure", viewsStructure, false}, {"structure-placement", viewsStructure, true}, {"sack", viewsSack, false}} {
				e, _, _ := plainEditor(t, lang, profile, c.record)
				e.Layout(1920, 1080) // Owner item 5: supported editor window.
				if c.open {
					e.toggleBlock("placement")
				}
				pic := e.ChromeImage()
				h := fnv.New64a()
				h.Write(pic.Pix)
				name := fmt.Sprintf("%s-%s-%s-1920x1080", c.name, lang, place)
				front, _ := e.PopupRect()
				s, _, _ := e.objectSpec()
				fmt.Fprintf(&out, "== %s %016x window=%v\n%s", name, h.Sum64(), front, rowsText(s))
				if dir := os.Getenv("AGAINROM_VIEWS_PNG"); dir != "" {
					witness.PNG(t, filepath.Join(dir, name+".png"), pic)
				}
			}
		}
	}
	got := out.String()
	if os.Getenv("MAPEDITOR_VIEWS_WRITE") != "" {
		if err := os.WriteFile(plainViewsGolden, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	raw, err := os.ReadFile(plainViewsGolden)
	if err != nil {
		t.Fatal(err)
	}
	if want := strings.ReplaceAll(string(raw), "\r\n", "\n"); got != want {
		t.Fatalf("plain views differ from %s:\n%s", plainViewsGolden, got)
	}
}
