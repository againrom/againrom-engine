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

	"againrom/internal/witness"
)

// cardHost is the wide fixture whose card states the unit's values as
// typed: a person for class 300, a creature for any other class, its
// health, mana, attributes, defence, skills and protections from the
// fields, as the game's host resolves a unit's settings.
type cardHost struct {
	*wideHost
}

func (h *cardHost) UnitSettings(index int, values []InspectionFieldValue) (MapEntity, []InspectionField, *InspectionUnitPreview, error) {
	entity, fields, preview, err := h.wideHost.UnitSettings(index, values)
	value := func(key string, fallback int) int {
		for _, f := range fields {
			if f.Key == key && f.Value > 0 {
				return int(f.Value)
			}
		}
		return fallback
	}
	band := CharacterBandCreature
	if value("identity", 0) == 300 {
		band = CharacterBandPerson
	}
	entity.Name = map[CharacterBand]string{CharacterBandPerson: "Guard", CharacterBandCreature: "Wolf"}[band]
	entity.Char = UnitCharacter{Known: true, Band: band, Experience: 1200, Sight: 6,
		Body: value("body", 10), Reaction: value("reaction", 11), Mind: value("mind", 12), Spirit: value("spirit", 13)}
	for k, key := range []string{"blade", "axe", "bludgeon", "pike", "shooting"} {
		entity.Char.Skills[k+1] = value("skill."+key, 20+k)
	}
	for k, key := range []string{"fire", "water", "air", "earth", "astral"} {
		entity.Char.Protection[k] = value("protection."+key, 30+k)
	}
	entity.Combat = UnitCombat{Known: true, DamageBase: 6, DamageSpread: 4, ToHit: 40, Defence: value("defence", 15), Absorption: value("absorption", 7)}
	entity.HP, entity.MaxHP = value("hp", 40), 50
	entity.Mana, entity.MaxMana = value("mana", 20), 30
	return entity, fields, preview, err
}

// cardEditor is the editor over a cardHost in language lang at 1920x1080,
// the unit of class class selected in the basic view, its window floating
// or in the right panel 520 wide.
func cardEditor(t *testing.T, lang string, class int64, panel bool) (*MapEditor, *cardHost) {
	t.Helper()
	width := 0
	if panel {
		width = 520
	}
	e := wideEditor(t, lang, width, "plain", map[string]int64{"Unit/identity": class})
	h := &cardHost{wideHost: e.doc.Edits.(*wideHost)}
	e.open = func(string) (*InspectionDocument, error) {
		d := wideDocument(h.wideHost)
		d.Edits = h
		return d, nil
	}
	if err := e.Open("views"); err != nil {
		t.Fatal(err)
	}
	e.Layout(1920, 1080)
	e.Select(viewsUnit, false)
	return e, h
}

// cardKeys are the unit's fields the basic view edits on the card, by the
// card's field.
var cardKeys = map[PanelField]string{
	PanelFieldHealth: "hp", PanelFieldManaCard: "mana",
	PanelFieldBody: "body", PanelFieldReaction: "reaction", PanelFieldMind: "mind", PanelFieldSpirit: "spirit",
	PanelFieldAbsorption: "absorption", PanelFieldDefence: "defence",
	PanelFieldSkillBlade: "skill.blade", PanelFieldSkillAxe: "skill.axe", PanelFieldSkillBludgeon: "skill.bludgeon",
	PanelFieldSkillPike: "skill.pike", PanelFieldSkillShooting: "skill.shooting",
	PanelFieldProtFire: "protection.fire", PanelFieldProtWater: "protection.water", PanelFieldProtAir: "protection.air",
	PanelFieldProtEarth: "protection.earth", PanelFieldProtAstral: "protection.astral",
}

// drawnCell is one cell the card draws with a value: its field, its words,
// and the box of its value in the card's own pixels.
type drawnCell struct {
	field        PanelField
	label, value string
	box          image.Rectangle
}

// drawnCells are the cells the window's card draws, read from the card's
// line report.
func drawnCells(e *MapEditor, s popupSpec) []drawnCell {
	font := s.CardFont
	l := CompactPanelLayout(nil)
	var out []drawnCell
	add := func(f PanelField, label, value string, pen, y int) {
		if value != "" {
			out = append(out, drawnCell{f, label, value, image.Rect(pen, y, pen+font.Advance(value), y+font.Height()+l.Gap)})
		}
	}
	for _, r := range CharacterPanelReport(l, font, *s.Card) {
		add(r.Field, r.Label, r.Value, r.At.X+r.Indent+r.ValueX, r.At.Y)
		if r.Right {
			add(r.RightField, r.RightLabel, r.RightValue, r.At.X+r.RightX+r.RightValueX, r.At.Y)
		}
	}
	return out
}

// A unit's basic view is its card drawn twice its size under the name,
// the player, the group, who the unit is and its dialogue line: each stat
// the card draws is the field's input at the place the card draws it, and
// each other value the card draws a line whose tip says the map does not
// store it. Equipment, the bag, the look and the map file fields are not
// shown. For a person and a creature, in English and Russian, floating and
// in the right panel, at 1920x1080.
func TestMapEditorUnitBasicViewIsTheCardWithItsInputs(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		for _, panel := range []bool{false, true} {
			for _, class := range []int64{300, 301} {
				name := fmt.Sprintf("%s class %d panel %v", lang, class, panel)
				e, _ := cardEditor(t, lang, class, panel)
				s, _, ok := e.objectSpec()
				if !ok || s.Card == nil || s.CardScale != 2 || !s.CardBelow || !e.PlainView() {
					t.Fatalf("%s: window %v card %v scale %d below %v", name, ok, s.Card != nil, s.CardScale, s.CardBelow)
				}
				if want := map[int64]CharacterBand{300: CharacterBandPerson, 301: CharacterBandCreature}[class]; s.Card.Char.Band != want {
					t.Fatalf("%s: band %v", name, s.Card.Char.Band)
				}
				if dir := os.Getenv("AGAINROM_VIEWS_PNG"); dir != "" {
					witness.PNG(t, filepath.Join(dir, fmt.Sprintf("unit-basic-%s-%d-panel-%v.png", lang, class, panel)), e.ChromeImage())
				}
				l := layoutPopup(s, e.font, e.bodyFont())
				if l.Card.Size() != image.Pt(2*compactPanelW, 2*compactPanelH) || !l.Card.In(l.Rect) {
					t.Fatalf("%s: card %v in window %v", name, l.Card, l.Rect)
				}
				var main []string
				keys := map[string]bool{}
				speaker := false
				for _, rr := range l.Rows {
					row := s.Rows[rr.Index]
					if i, ok := e.obj.rowField[rr.Index]; ok {
						keys[e.obj.fields[i].Key] = true
					}
					if !row.placed() {
						if rr.Rect.Max.Y > l.Card.Min.Y {
							t.Errorf("%s: row %q stands over the card", name, row.Label)
						}
						if i, ok := e.obj.rowField[rr.Index]; ok {
							main = append(main, e.obj.fields[i].Key)
						}
						speaker = speaker || strings.HasPrefix(row.Label, "A mission-start dialogue")
					}
				}
				if want := []string{"owner", "group", "identity"}; !slices.Equal(main, want) {
					t.Errorf("%s: rows above the card edit %q, want %q", name, main, want)
				}
				if !speaker {
					t.Errorf("%s: no line above the card says whether a dialogue shows the unit", name)
				}
				for _, key := range []string{"face", "sex", "gear", "wear1", "carry1", "keys"} {
					if keys[key] {
						t.Errorf("%s: the basic view shows %s", name, key)
					}
				}
				placed := map[image.Rectangle]popupRowRect{}
				for _, rr := range l.Rows {
					if s.Rows[rr.Index].placed() {
						placed[rr.Rect] = rr
					}
				}
				inputs := 0
				for _, c := range drawnCells(e, s) {
					// A pair such as health's keeps its max part on the card:
					// the input stands on the first number only.
					if number, _, pair := strings.Cut(c.value, "/"); pair {
						c.box.Max.X = c.box.Min.X + s.CardFont.Advance(number)
					}
					value := image.Rectangle{Min: c.box.Min.Mul(2), Max: c.box.Max.Mul(2)}.Add(l.Card.Min)
					var at popupRowRect
					found := false
					for r, rr := range placed {
						if r.Max == value.Max && r.Min.Y == value.Min.Y && r.Min.X <= value.Min.X {
							at, found = rr, true
						}
					}
					if !found {
						t.Errorf("%s: no row on the card's %q %q at %v", name, c.label, c.value, value)
						continue
					}
					row := s.Rows[at.Index]
					i, edits := e.obj.rowField[at.Index]
					if key, ok := cardKeys[c.field]; ok {
						inputs++
						if !edits || e.obj.fields[i].Key != key || row.Kind != popupEdit || at.Control != at.Rect || !strings.Contains(row.Tip, e.editorHostText(e.obj.fields[i].Label)) {
							t.Errorf("%s: %q is not the %s input: %+v", name, c.label, key, row)
						}
						if at.Rect.Dx() < e.bodyFont().Advance("0000")+2*editTextX && at.Rect.Min.X > l.Card.Min.X+2*c.box.Min.X-e.bodyFont().Advance("0000") {
							t.Errorf("%s: the %s input %v holds no four digits", name, key, at.Rect)
						}
						continue
					}
					if edits || row.Kind != popupText || !at.Control.Empty() || !strings.HasPrefix(row.Tip, strings.TrimSpace(c.label+" "+c.value)+":") {
						t.Errorf("%s: %q %q is not a stated value: %+v", name, c.label, c.value, row)
					}
					fixed := strings.TrimPrefix(e.word("editor.plain.fixed"), "%s")
					if !strings.HasSuffix(row.Tip, fixed) {
						t.Errorf("%s: %q tip %q", name, c.label, row.Tip)
					}
				}
				if inputs < 16 {
					t.Errorf("%s: %d inputs on the card", name, inputs)
				}
			}
		}
	}
}

// cardFieldIndex is the index of the unit's field key.
func cardFieldIndex(t *testing.T, e *MapEditor, key string) int {
	t.Helper()
	i := slices.IndexFunc(e.ObjectFields(), func(f InspectionField) bool { return f.Key == key })
	if i < 0 {
		t.Fatalf("no field %s", key)
	}
	return i
}

// typeCardField clicks the field key's input and types value over it.
func typeCardField(t *testing.T, e *MapEditor, key, value string) {
	t.Helper()
	r, ok := e.FieldRect(cardFieldIndex(t, e, key))
	viewsClick(t, e, r, ok, "the "+key+" input")
	for range 6 {
		_ = e.Key("backspace")
	}
	e.TextInput(value)
}

// A value typed on the card redraws the card as the game would show it:
// the card states the typed body, its pixels change, and the input stays
// at the card's place for it.
func TestMapEditorUnitBasicViewTypedValueRedrawsTheCard(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		e, _ := cardEditor(t, lang, 300, false)
		cardPixels := func() (uint64, image.Rectangle) {
			s, _, _ := e.objectSpec()
			l := layoutPopup(s, e.font, e.bodyFont())
			pic := e.ChromeImage()
			h := fnv.New64a()
			for y := l.Card.Min.Y; y < l.Card.Max.Y; y++ {
				h.Write(pic.Pix[pic.PixOffset(l.Card.Min.X, y):pic.PixOffset(l.Card.Max.X, y)])
			}
			return h.Sum64(), l.Card
		}
		before, card := cardPixels()
		typeCardField(t, e, "body", "77")
		s, _, _ := e.objectSpec()
		if s.Card.Char.Body != 77 {
			t.Fatalf("%s: the card states body %d", lang, s.Card.Char.Body)
		}
		after, again := cardPixels()
		if after == before || again != card {
			t.Fatalf("%s: card pixels %x then %x, card %v then %v", lang, before, after, card, again)
		}
		r, _ := e.FieldRect(cardFieldIndex(t, e, "body"))
		for _, c := range drawnCells(e, s) {
			if c.field == PanelFieldBody && (r.Max != c.box.Max.Mul(2).Add(card.Min) || c.value != "77") {
				t.Fatalf("%s: the body input %v, the card's value %q at %v", lang, r, c.value, c.box)
			}
		}
	}
}

// Over a value the map does not store the hover line says so whole.
func TestMapEditorUnitBasicViewTipsAFixedValue(t *testing.T) {
	e, _ := cardEditor(t, "en", 300, false)
	s, _, _ := e.objectSpec()
	l := layoutPopup(s, e.font, e.bodyFont())
	for _, rr := range l.Rows {
		row := s.Rows[rr.Index]
		if !row.placed() || row.Label != "XP" {
			continue
		}
		tip, ok := popupTipAt(e.popupBounds(), e.font, e.bodyFont(), s, l, center(rr.Rect), true)
		got := strings.Join(strings.Fields(strings.Join(tip.Lines, " ")), " ")
		if !ok || got != "XP 1200: the map does not store this value; the unit's kind, tier, NPC record and equipment give it." {
			t.Fatalf("tip %v %q", ok, got)
		}
		return
	}
	t.Fatal("no XP line on the card")
}
