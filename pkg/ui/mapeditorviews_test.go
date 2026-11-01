package ui

import (
	"errors"
	"fmt"
	"image"
	"os"
	"strings"
	"testing"

	"againrom/pkg/formats/textinput"
	"againrom/pkg/render/terrain"
	"againrom/pkg/words"
)

// viewsHost is an edit host of one unit, one structure and one sack whose
// fields carry the keys, labels, sections and choices the game's host gives
// those records. Its map bytes are the records' values as text.
type viewsHost struct {
	values map[string]int64 // by kind and key, "Unit/hp"
	writes []string
	looks  int
}

const (
	viewsUnit      = 0
	viewsStructure = 1
	viewsSack      = 2
)

func (h *viewsHost) Dirty() bool   { return len(h.writes) > 0 }
func (h *viewsHost) CanUndo() bool { return false }
func (h *viewsHost) CanRedo() bool { return false }
func (h *viewsHost) Move(RecordKind, int, image.Point) (*InspectionDocument, error) {
	return nil, nil
}
func (h *viewsHost) Clone(RecordKind, int, image.Point) (*InspectionDocument, error) {
	return nil, nil
}
func (h *viewsHost) Delete(RecordKind, int) (*InspectionDocument, error) { return nil, nil }
func (h *viewsHost) Undo() (*InspectionDocument, error)                  { return nil, nil }
func (h *viewsHost) Redo() (*InspectionDocument, error)                  { return nil, nil }
func (h *viewsHost) SaveAs(string) (string, error)                       { return "", errors.New("no writer") }

func (h *viewsHost) value(kind RecordKind, key string) int64 { return h.values[string(kind)+"/"+key] }

// viewsChoices are a field's choices by key shape.
func viewsChoices(key string) []InspectionChoice {
	switch {
	case key == "owner":
		return []InspectionChoice{{1, "Self"}, {2, "Enemy"}}
	case key == "class":
		return []InspectionChoice{{300, "300: Fighter"}, {301, "301: Archer"}}
	case key == "kind":
		return []InspectionChoice{{51, "51: Hall"}, {52, "52: Tower"}}
	case key == "wear1" || key == "wear2" || key == "carry1":
		return []InspectionChoice{{-1, "Inherit"}, {0, "Empty"}, {0x101, "0x0101: Sword"}, {0x102, "0x0102: Mace"}}
	}
	return nil
}

func (h *viewsHost) Fields(kind RecordKind, index int) []InspectionField {
	field := func(key, label, section string) InspectionField {
		return InspectionField{Key: key, Label: label, Section: section, Value: h.value(kind, key), Choices: viewsChoices(key)}
	}
	switch kind {
	case KindUnit:
		stat := func(key, label, group string) InspectionField {
			f := field(key, label, "stats")
			f.Inherit, f.Group, f.Inherited = true, group, "10"
			return f
		}
		wear := field("wear1", "Slot 1", "equipment")
		material := field("wearmaterial1", "Slot 1 material", "equipment")
		material.Parent = "wear1"
		recipe := field("wearrecipe1", "Slot 1 recipe", "equipment")
		recipe.Parent = "wear1"
		return []InspectionField{
			field("owner", "Owner slot", "identity"),
			field("class", "Class", "identity"),
			field("tier", "Tier", "identity"),
			field("definition", "Definition id", "identity"),
			field("group", "Group", "identity"),
			field("face", "Face", "identity"),
			field("sex", "Sex", "identity"),
			stat("hp", "Health", ""),
			stat("mana", "Mana", ""),
			stat("body", "Body", "Attributes"),
			stat("mind", "Mind", "Attributes"),
			stat("skill.fire", "Fire", "Skills"),
			stat("protection.fire", "Fire protection", "Protection"),
			{Key: "resolved", ReadOnly: true, Section: "stats", Lines: []InspectionLine{{Text: "Attack 12"}}},
			field("gear", "Gear set", "equipment"),
			wear, material, recipe,
			field("wear2", "Slot 2", "equipment"),
			field("carry1", "Carried item 1", "inventory"),
			field("carryadd", "Add carried item", "inventory"),
			{Key: "holdings", ReadOnly: true, Section: "inventory", Lines: []InspectionLine{{Text: "Corpse: Sword"}}},
		}
	case KindStructure:
		return []InspectionField{
			field("kind", "Kind (structures.reg ID)", ""),
			field("owner", "Owner slot", ""),
			field("identity", "Script target id", ""),
			field("placement", "Placement word (signed raw)", ""),
			field("width", "Footprint width (0 = table)", ""),
			field("height", "Footprint height (0 = table)", ""),
		}
	case KindSack:
		return []InspectionField{
			field("gold", "Gold", ""),
			field("item1", "Item 1 code (0 removes)", ""),
			field("item2", "Item 2 code (0 removes)", ""),
			field("additem", "Add item code", ""),
			field("use1", "Item 1 native dispatch", ""),
			field("recipe1", "Item 1 recipe (0 = plain)", ""),
			field("use2", "Item 2 native dispatch", ""),
			field("recipe2", "Item 2 recipe (0 = plain)", ""),
		}
	}
	return nil
}

func (h *viewsHost) SetField(kind RecordKind, index int, key string, value int64) (*InspectionDocument, error) {
	return h.SetFields(kind, index, []InspectionFieldValue{{Key: key, Value: value}})
}

func (h *viewsHost) SetFields(kind RecordKind, index int, values []InspectionFieldValue) (*InspectionDocument, error) {
	for _, v := range values {
		h.values[string(kind)+"/"+v.Key] = v.Value
		h.writes = append(h.writes, fmt.Sprintf("%s %s=%d", kind, v.Key, v.Value))
	}
	return viewsDocument(h), nil
}

// UnitSettings makes the unit's window the person window, as the game's
// host does.
func (h *viewsHost) UnitSettings(index int, values []InspectionFieldValue) (MapEntity, []InspectionField, *InspectionUnitPreview, error) {
	fields := h.Fields(KindUnit, index)
	for _, v := range values {
		for i := range fields {
			if fields[i].Key == v.Key {
				fields[i].Value = v.Value
			}
		}
	}
	pic := image.NewRGBA(image.Rect(0, 0, 2, 2))
	pic.Pix[3] = 255
	char := UnitCharacter{Known: true, Band: CharacterBandPerson, Body: 10, Experience: 1200}
	return MapEntity{ID: 1, Name: "Guard", Char: char, HP: 40, MaxHP: 50}, fields, &InspectionUnitPreview{Doll: pic, Face: pic}, nil
}

// ItemLook names Sword and Mace and gives each a one-colour icon.
func (h *viewsHost) ItemLook(code int64) (string, *image.RGBA) {
	h.looks++
	names := map[int64]string{0x101: "Sword", 0x102: "Mace"}
	name, ok := names[code]
	if !ok {
		return "", nil
	}
	icon := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for i := 0; i < len(icon.Pix); i += 4 {
		icon.Pix[i], icon.Pix[i+1], icon.Pix[i+2], icon.Pix[i+3] = byte(code), 0x80, 0x20, 255
	}
	return name, icon
}

func (h *viewsHost) bytes() string {
	var b strings.Builder
	for _, k := range []string{"Unit/owner", "Unit/hp", "Unit/wear1", "Structure/placement", "Sack/item1", "Sack/gold"} {
		fmt.Fprintf(&b, "%s=%d;", k, h.values[k])
	}
	return b.String()
}

func viewsDocument(h *viewsHost) *InspectionDocument {
	v, err := NewViewer("views", terrain.Grid{Width: 40, Height: 40, Tiles: make([]uint16, 1600)}, &terrain.Tileset{})
	if err != nil {
		panic(err)
	}
	d := &InspectionDocument{Viewer: v, Source: "views", Width: 40, Height: 40, SourceBytes: []byte(h.bytes()), Edits: h}
	d.Records = []InspectionRecord{
		{Kind: KindUnit, Index: 0, Section: 6, Cell: image.Pt(10, 10), Spatial: true, Size: image.Pt(1, 1), Label: "Guard", Player: 1},
		{Kind: KindStructure, Index: 0, Section: 4, Cell: image.Pt(20, 20), Spatial: true, Size: image.Pt(2, 2), Label: "Hall", Player: 1},
		{Kind: KindSack, Index: 0, Section: 8, Cell: image.Pt(14, 12), Spatial: true, Size: image.Pt(1, 1), Label: "Sack"},
		{Kind: KindAction, Index: 0, Section: 7},
		{Kind: KindTrigger, Index: 0, Section: 7},
	}
	d.Script = &InspectionScriptModel{
		Actions:  []InspectionScriptNode{{Record: 3, ID: 1, Opcode: 32, Label: "Burn the hall", Types: [10]uint32{scriptStructure}, Values: [10]uint32{3}}},
		Triggers: []InspectionTrigger{{Record: 4, Name: "Raid", Acts: [4]uint32{1}}},
		Targets:  map[uint32][]InspectionScriptTarget{scriptStructure: {{Value: 3, Record: viewsStructure, Uses: []InspectionScriptUse{{Node: 3, Triggers: []int{4}}}}}},
	}
	return d
}

// viewsEditor is the editor over a viewsHost in language lang at 1280x800,
// without a profile: the window as the base showed it.
func viewsEditor(t *testing.T, lang string) (*MapEditor, *viewsHost) {
	t.Helper()
	h := &viewsHost{values: map[string]int64{
		"Unit/owner": 1, "Unit/class": 300, "Unit/tier": 1, "Unit/definition": 1234, "Unit/group": 2, "Unit/face": 7, "Unit/sex": 0,
		"Unit/hp": 40, "Unit/mana": 0, "Unit/body": 12, "Unit/wear1": 0x101, "Unit/wear2": -1, "Unit/carry1": 0x102,
		"Structure/kind": 51, "Structure/owner": 1, "Structure/identity": 3, "Structure/placement": 5,
		"Sack/gold": 30, "Sack/item1": 0x101, "Sack/item2": 0x102, "Sack/recipe1": 1,
	}}
	e := NewMapEditor(panelFont(), []EditorMap{{Key: "views", Label: "Views"}}, func(string) (*InspectionDocument, error) { return viewsDocument(h), nil })
	e.SetWords(words.For(lang))
	e.SetHostTextEncoder(func(s string) string {
		encoded, _ := textinput.EncodeText(s, e.font.Selector, textinput.Substitute)
		return encoded
	})
	e.Layout(1280, 800)
	if err := e.Open("views"); err != nil {
		t.Fatal(err)
	}
	return e, h
}

// viewsWindowText is the shown object window as its spec states it: rows,
// tabs, columns and buttons, the view switch left out.
func viewsWindowText(e *MapEditor) string {
	s, _, ok := e.objectSpec()
	if !ok {
		return "no window\n"
	}
	var b strings.Builder
	b.WriteString(rowsText(s))
	fmt.Fprintf(&b, "tabs %q active %d columns %d\n", s.Tabs, s.ActiveTab, s.Columns)
	caption, _, switched := strings.Cut(e.word("editor.view.switch"), "%")
	var buttons []string
	for _, label := range s.Buttons {
		if switched && strings.HasPrefix(label, caption) {
			continue
		}
		buttons = append(buttons, label)
	}
	fmt.Fprintf(&b, "buttons %q\n", buttons)
	return b.String()
}

// technicalViewsText is each fixture record's technical window, the unit's
// in each of its four sections, in English and Russian.
func technicalViewsText(t *testing.T) string {
	var b strings.Builder
	for _, lang := range []string{"en", "ru"} {
		for _, record := range []int{viewsUnit, viewsStructure, viewsSack} {
			sections := 1
			if record == viewsUnit {
				sections = 4
			}
			for section := 0; section < sections; section++ {
				e, _ := viewsEditor(t, lang)
				e.Select(record, false)
				e.objectSpec()
				e.obj.section = section
				fmt.Fprintf(&b, "== %s %s section %d\n%s", lang, e.doc.Records[record].Kind, section, viewsWindowText(e))
			}
		}
	}
	return b.String()
}

const technicalViewsGolden = "testdata/mapeditor_technical_views.golden"

// The technical view of a unit, a structure and a sack is the window as it
// was before the plain view: each row, tab, column and word, recorded from
// that editor. MAPEDITOR_VIEWS_WRITE=1 rewrites the record.
func TestMapEditorTechnicalViewKeepsTheWindow(t *testing.T) {
	got := technicalViewsText(t)
	if os.Getenv("MAPEDITOR_VIEWS_WRITE") != "" {
		if err := os.WriteFile(technicalViewsGolden, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	raw, err := os.ReadFile(technicalViewsGolden)
	if err != nil {
		t.Fatal(err)
	}
	if want := strings.ReplaceAll(string(raw), "\r\n", "\n"); got != want {
		t.Fatalf("technical windows differ from %s:\n%s", technicalViewsGolden, got)
	}
}
