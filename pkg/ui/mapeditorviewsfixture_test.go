package ui

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"againrom/pkg/formats/textinput"
	"againrom/pkg/render/text"
	"againrom/pkg/words"
)

// wideHost is the views fixture with records shaped like an installed map's:
// a human with gear, the twelve equipment slots (two of them read-only
// lines), a bag, every stat and protection group and the read-only sheets;
// a structure; and a sack of three items whose plain words are the longest
// the tables hold.
type wideHost struct {
	*viewsHost
}

// wideSlots are the equipment slots' labels, slot 1 first.
var wideSlots = []string{"Weapon", "Shield", "Slot 3 (unused)", "Ring", "Amulet", "Head", "Body armour", "Outer armour", "Bracers", "Gloves", "Legs", "Boots"}

func (h *wideHost) Fields(kind RecordKind, index int) []InspectionField {
	field := func(key, label, section, group string) InspectionField {
		return InspectionField{Key: key, Label: label, Section: section, Group: group, Value: h.value(kind, key), Choices: viewsChoices(key)}
	}
	switch kind {
	case KindUnit:
		stat := func(key, label, group string) InspectionField {
			f := field(key, label, "stats", group)
			f.Inherit, f.Inherited = true, "10"
			return f
		}
		identity := field("identity", "Who it is", "identity", "")
		identity.Choices = viewsIdentities
		out := []InspectionField{
			identity,
			field("owner", "Owner slot", "identity", ""),
			field("group", "Group", "identity", ""),
			field("face", "Face", "identity", ""),
			field("sex", "Sex", "identity", ""),
			{Key: "speaker", Label: "In a dialogue", ReadOnly: true, Section: "identity", Lines: []InspectionLine{{Text: "In a dialogue"}, {Text: "A mission-start dialogue does not show it."}}},
			{Key: "keys", Label: "Map keys", ReadOnly: true, Section: "identity", Lines: []InspectionLine{{Text: "Map keys"}, {Text: "Class key: 300"}, {Text: "Secondary key: 0"}}},
			field("gear", "Gear set", "equipment", "Equipment"),
		}
		for n, label := range wideSlots {
			key := fmt.Sprintf("wear%d", n+1)
			f := field(key, label, "equipment", "Equipment")
			f.Choices = []InspectionChoice{{-1, "Inherit"}, {0, "Empty"}, {0x101, "0x0101: Sword"}}
			if n+1 == 3 || n+1 == 11 {
				f.ReadOnly, f.Lines = true, []InspectionLine{{Text: label + ": Empty"}}
			}
			out = append(out, f)
			if n == 0 {
				for _, part := range []string{"material", "tier", "recipe"} {
					child := field("wear"+part+"1", label+" / "+part, "equipment", "Equipment")
					child.Parent = key
					out = append(out, child)
				}
			}
		}
		out = append(out,
			field("carry1", "Carried item 1", "inventory", "Added items"),
			field("carry2", "Carried item 2", "inventory", "Added items"),
			field("carryadd", "Add carried item", "inventory", "Added items"),
			InspectionField{Key: "holdings", ReadOnly: true, Section: "inventory", Group: "After the map", Lines: []InspectionLine{{Text: "Corpse: Sword"}, {Text: "Bag: Mace"}}},
			stat("hp", "Health", "Health and mana"),
			stat("mana", "Mana", "Health and mana"),
			stat("body", "Body", "Attributes"),
			stat("mind", "Mind", "Attributes"),
			stat("spirit", "Spirit", "Attributes"),
			stat("reaction", "Reaction", "Attributes"),
			stat("defence", "Defence", "Defence"),
			stat("absorption", "Absorption", "Defence"),
		)
		for _, s := range []string{"blade", "axe", "bludgeon", "pike", "shooting"} {
			out = append(out, stat("skill."+s, strings.ToUpper(s[:1])+s[1:], "Weapon skills"))
		}
		for _, s := range []string{"fire", "water", "air", "earth", "astral"} {
			out = append(out, stat("protection."+s, strings.ToUpper(s[:1])+s[1:]+" protection", "Protections"))
		}
		var sheet []InspectionLine
		for _, l := range []string{"Attack 12", "Defence 8", "Speed 4", "Scan range 9"} {
			sheet = append(sheet, InspectionLine{Text: l})
		}
		return append(out,
			InspectionField{Key: "resolved", ReadOnly: true, Section: "stats", Lines: sheet},
			InspectionField{Key: "overrides", ReadOnly: true, Section: "stats", Lines: []InspectionLine{{Text: "No stat override"}}})
	case KindSack:
		out := []InspectionField{field("gold", "Gold", "", "")}
		for n := 1; n <= 3; n++ {
			out = append(out, field(fmt.Sprintf("item%d", n), fmt.Sprintf("Item %d code (0 removes)", n), "", ""))
		}
		out = append(out, field("additem", "Add item code", "", ""))
		for n := 1; n <= 3; n++ {
			out = append(out, field(fmt.Sprintf("use%d", n), fmt.Sprintf("Item %d native dispatch", n), "", ""),
				field(fmt.Sprintf("recipe%d", n), fmt.Sprintf("Item %d recipe (0 = plain)", n), "", ""))
		}
		return out
	}
	return h.viewsHost.Fields(kind, index)
}

func (h *wideHost) SetField(kind RecordKind, index int, key string, value int64) (*InspectionDocument, error) {
	return h.SetFields(kind, index, []InspectionFieldValue{{Key: key, Value: value}})
}

func (h *wideHost) SetFields(kind RecordKind, index int, values []InspectionFieldValue) (*InspectionDocument, error) {
	if _, err := h.viewsHost.SetFields(kind, index, values); err != nil {
		return nil, err
	}
	return wideDocument(h), nil
}

func (h *wideHost) UnitSettings(index int, values []InspectionFieldValue) (MapEntity, []InspectionField, *InspectionUnitPreview, error) {
	entity, _, preview, err := h.viewsHost.UnitSettings(index, values)
	fields := h.Fields(KindUnit, index)
	for _, v := range values {
		for i := range fields {
			if fields[i].Key == v.Key {
				fields[i].Value = v.Value
			}
		}
	}
	return entity, fields, preview, err
}

func wideDocument(h *wideHost) *InspectionDocument {
	d := viewsDocument(h.viewsHost)
	d.Edits = h
	return d
}

// wordsFont is a font as wide as the installed editor fonts' letters or
// wider: every glyph ten pixels with its spacing, sixteen high. The
// installed EN and RU body fonts are sixteen high and their editor words
// average under ten pixels a letter.
func wordsFont(lang string) *text.Font {
	f := &text.Font{Spacing: 2, Glyphs: make([]text.Glyph, 224)}
	if lang == "ru" {
		f.Selector = text.SelectorConverting
	}
	for k := range f.Glyphs {
		g := text.Glyph{Width: 8, Height: 16, Pixels: make([]text.Pixel, 8*16), Advance: 8}
		if k > 0 {
			g.Pixels[(k%16)*8+k%8] = text.Pixel{Level: text.MaxLevel, Painted: true}
		}
		f.Glyphs[k] = g
	}
	return f
}

// cardFont is a card font of the installed card's proportions: ten pixels
// high, six a letter, so the card's lines fit its 242 pixels.
func cardFont() *text.Font {
	f := &text.Font{Spacing: 1, Glyphs: make([]text.Glyph, 224)}
	for k := range f.Glyphs {
		g := text.Glyph{Width: 5, Height: 10, Pixels: make([]text.Pixel, 5*10), Advance: 5}
		if k > 0 {
			g.Pixels[(k%10)*5+k%5] = text.Pixel{Level: text.MaxLevel, Painted: true}
		}
		f.Glyphs[k] = g
	}
	return f
}

// wideWidths are the right panel's widths the tests set: its least, its
// default and its greatest, and 0 for the panel minimized, the window
// floating.
var wideWidths = []int{360, 520, 800, 0}

// wideEditor is the editor over a wideHost in language lang at 1920x1080,
// in the installed fonts' size, after the profile with the right panel
// restored at width (0 minimizes it) and the view named view loads. set
// changes the host's values first.
func wideEditor(t *testing.T, lang string, width int, view string, set map[string]int64) *MapEditor {
	t.Helper()
	inner, _ := viewsEditor(t, lang)
	h := &wideHost{viewsHost: inner.doc.Edits.(*viewsHost)}
	for n := 1; n <= 12; n++ {
		if n != 3 && n != 11 {
			h.values[fmt.Sprintf("Unit/wear%d", n)] = 0x101
		}
	}
	h.values["Sack/item3"] = 0x101
	for k, v := range set {
		h.values[k] = v
	}
	e := NewMapEditor(wordsFont(lang), []EditorMap{{Key: "views", Label: "Views"}}, func(string) (*InspectionDocument, error) { return wideDocument(h), nil })
	e.SetWords(words.For(lang))
	e.SetHostTextEncoder(func(s string) string {
		encoded, _ := textinput.EncodeText(s, e.font.Selector, textinput.Substitute)
		return encoded
	})
	right := `"minimized":true,"width":520`
	if width > 0 {
		right = fmt.Sprintf(`"minimized":false,"width":%d`, width)
	}
	e.SetCardFont(cardFont())
	e.SetPanelProfile([]byte(fmt.Sprintf(`{"left":{"minimized":false,"width":260},"right":{%s,"view":%q}}`, right, view)), nil, nil)
	e.Layout(1920, 1080) // Owner item 5: supported editor window.
	if err := e.Open("views"); err != nil {
		t.Fatal(err)
	}
	return e
}

// wideSections are the window's states a map maker steps through for a
// record: the person window's four tabs, or the one window.
func wideSections(e *MapEditor) []string {
	if _, ok := e.ObjectSectionRect("identity"); ok {
		return []string{"identity", "equipment", "inventory", "stats"}
	}
	return []string{""}
}

// openSection presses the person window's tab of section, none for "".
func openSection(t *testing.T, e *MapEditor, section string) {
	t.Helper()
	if section == "" {
		return
	}
	r, ok := e.ObjectSectionRect(section)
	if !ok {
		t.Fatalf("no tab %s", section)
	}
	wideNow = wideNow.Add(time.Second)
	editorClickAt(e, center(r), wideNow)
}

// wideNow is the time of the next tab press, a second after the last.
var wideNow = time.Unix(1, 0)

// technicalReachText is, for each record and person window tab in the
// technical view at each panel width, the button row's lines and the
// fields the window shows before any scroll.
func technicalReachText(t *testing.T) string {
	var b strings.Builder
	for _, lang := range []string{"en", "ru"} {
		for _, width := range wideWidths {
			for _, record := range []int{viewsUnit, viewsStructure, viewsSack} {
				e := wideEditor(t, lang, width, "technical", nil)
				e.Select(record, false)
				e.objectSpec()
				for _, section := range wideSections(e) {
					openSection(t, e, section)
					e.objectTop = 0
					s, _, _ := e.objectSpec()
					l := layoutPopup(s, e.font, e.bodyFont())
					lines := map[int]bool{}
					for _, r := range l.Buttons {
						lines[r.Min.Y] = true
					}
					var shown []string
					for _, rr := range l.Rows {
						if i, ok := e.obj.rowField[rr.Index]; ok {
							shown = append(shown, e.obj.fields[i].Key)
						}
					}
					line := fmt.Sprintf("%s panel %d %s %s: %d button lines, rows %d..%d, fields %s", lang, width, e.doc.Records[record].Kind, section,
						len(lines), s.Top, s.Top+len(l.Rows), strings.Join(shown, " "))
					b.WriteString(strings.TrimRight(line, " ") + "\n")
				}
			}
		}
	}
	return b.String()
}

const technicalReachGolden = "testdata/mapeditor_technical_reach.golden"

// The technical view keeps the rows the window showed before the plain
// view: at each panel width and floating, in both languages, its button
// row stays on its lines and the window shows the same fields before a
// scroll, recorded from that editor. MAPEDITOR_VIEWS_WRITE=1 rewrites the
// record.
func TestMapEditorTechnicalViewKeepsTheRows(t *testing.T) {
	got := technicalReachText(t)
	if os.Getenv("MAPEDITOR_VIEWS_WRITE") != "" {
		if err := os.WriteFile(technicalReachGolden, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	raw, err := os.ReadFile(technicalReachGolden)
	if err != nil {
		t.Fatal(err)
	}
	if want := strings.ReplaceAll(string(raw), "\r\n", "\n"); got != want {
		t.Fatalf("technical windows' rows differ from %s:\n%s", technicalReachGolden, got)
	}
}
