package game

import (
	"bytes"
	"image"
	"maps"
	"slices"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

// peopleRow is one Humans row of the people fixture: its name, type id,
// face and gender column (1 a woman).
type peopleRow struct {
	name               string
	typeID, face, male int32
}

// peopleRows are the people fixture's Humans rows from row 1: a woman and a
// man fighter (the man the named character record 8 reaches), two PC_ rows,
// two mercenary types of four levels each, a woman and a man mage and rider,
// and a type that composes no figure.
var peopleRows = []peopleRow{
	{"F_Woman", 3, 5, 1}, {"Courier", 3, 7, 0}, {"PC_Hero", 3, 1, 0}, {"PC_Mage", 0x18, 1, 1},
	{"NPC03_1", 0x18, 2, 0}, {"NPC10_1", 3, 2, 1}, {"NPC03_2", 0x18, 2, 0}, {"NPC10_2", 3, 2, 1},
	{"NPC03_3", 0x18, 2, 0}, {"NPC10_3", 3, 2, 1}, {"NPC03_4", 0x18, 2, 0}, {"NPC10_4", 3, 2, 1},
	{"M_Woman", 0x17, 3, 1}, {"M_Man", 0x18, 3, 0}, {"H_Woman", 0x13, 2, 1}, {"H_Man", 0x15, 2, 0},
	{"Odd", 0x1c, 1, 0},
}

// peopleClasses are the fixture's class sheets by type id, each a frame of
// its own size so that two classes never draw one picture.
var peopleClasses = map[int32]int{3: 3, 0x17: 4, 0x18: 5, 0x13: 6, 0x15: 7, 0x1c: 2}

// peopleIdentityEditor is unitIdentityEditor with the people fixture's
// Humans rows, each keyed by its own server id (Courier by 4321, which npc
// record 8 names), and its class sheets, in lang.
func peopleIdentityEditor(t *testing.T, lang string) *mapInspectionEdits {
	t.Helper()
	s, _ := unitIdentityEditor(t)
	x := s.inspector
	humans := dbCollection{{}}
	for i, r := range peopleRows {
		p := figureRow(r.typeID, r.face, r.male)
		p[24] = int32(5001 + i)
		if r.name == "Courier" {
			p[24] = 4321
		}
		humans = append(humans, dbEntry{name: r.name, params: p})
	}
	x.table.Humans = humans
	for typeID, side := range peopleClasses {
		c := worldFixtureArt(16, 16, 8, 16, side, side, 1)
		c.Name = "Class " + string(rune('A'+typeID%26))
		x.units.Classes[typeID] = c
	}
	x.units.Classes[0x1c].Name = "Odd class"
	x.recordOptions, x.identities, x.recordPictures = nil, nil, nil
	if lang == "ru" {
		x.archives.Base.Profile.Language = "russian"
	}
	return s
}

func personValue(row int) int64 { return identityValue(identityPerson, row) }

// The people block holds story heroes, player characters, one cell per
// mercenary type and the other rows by the class sheet the mission draws,
// men first, in both languages.
func TestEditorIdentityPeopleGroupByRoleAndDrawnClass(t *testing.T) {
	heads := map[string][]string{
		"en": {"Story heroes", "Player characters", "Mercenaries", "Others: fighters", "Others: mages", "Others: riders", "Others: Odd class"},
		"ru": {"Сюжетные герои", "Персонажи игрока", "Наемники", "Прочие: бойцы", "Прочие: маги", "Прочие: всадники", "Прочие: Odd class"},
	}
	want := [][]int64{{identityValue(identityNamed, 8)}, {personValue(3), personValue(4)}, {personValue(5), personValue(6)},
		{personValue(2), personValue(1)}, {personValue(14), personValue(13)}, {personValue(16), personValue(15)}, {personValue(17)}}
	for _, lang := range []string{"en", "ru"} {
		t.Run(lang, func(t *testing.T) {
			s := peopleIdentityEditor(t, lang)
			x := s.inspector
			step := s.NewRecordStep(ui.InspectionNewRecord{Kind: ui.KindUnit}, "identity")
			people := x.inspectionUTF8(x.word("editor.unit.groups.people"))
			var got []ui.InspectionChoiceGroup
			for _, g := range step.Groups {
				if g.Block == people {
					got = append(got, g)
				}
			}
			if len(got) != len(want) {
				t.Fatalf("people block has %d groups %+v, want %d", len(got), got, len(want))
			}
			for i, g := range got {
				if g.Head != x.inspectionUTF8(heads[lang][i]) || !slices.Equal(g.Values, want[i]) {
					t.Errorf("group %d = %q %v, want %q %v", i, g.Head, g.Values, heads[lang][i], want[i])
				}
			}
			for row := 7; row <= 12; row++ {
				if slices.ContainsFunc(step.Choices, func(c ui.InspectionChoice) bool { return c.Value == personValue(row) }) {
					t.Errorf("mercenary row %d of a level above 1 has a cell of its own", row)
				}
			}
		})
	}
}

// missionArt is the class the mission draws placement u with when it starts
// u alone on the editor's map, and the tier it draws it at.
func missionArt(t *testing.T, s *mapInspectionEdits, u alm.Unit) (*terrain.UnitClass, int) {
	t.Helper()
	x := s.inspector
	m, err := s.model.Map()
	if err != nil {
		t.Fatal(err)
	}
	u.Owner, u.UnitID = 1, 1
	u.X, u.Y = cellWords(image.Pt(20, 20), newUnitFraction, newUnitFraction)
	m.Units, m.AuthoredUnits = []alm.Unit{u}, nil
	w, st, err := mapload.StartCampaignMission(m, x.table, openDifficulty, nil, s.mission())
	if err != nil {
		t.Fatal(err)
	}
	ms := &Mission{Number: s.mission(), Map: m, World: w, Start: st}
	src := x.archives.artSource()
	list, _ := ReadBodyList(src)
	canonicalizePartyAppearance(ms, x.units, src, list)
	return missionAppearanceArt(ms, x.units, list, src, placedClientHeroes(ms, x.table))[0], entityTiers(m, x.table)[0]
}

// Every person cell, a named character's too, is the sprite the mission
// draws the same placement with: the class sheet of its row's type, never a
// portrait or a bare paper doll.
func TestEditorIdentityPeopleCellsDrawTheMissionSprite(t *testing.T) {
	s := peopleIdentityEditor(t, "en")
	x := s.inspector
	portrait := s.npcPortraits([]ui.InspectionChoice{{Value: 8}})[8]
	values := []int64{identityValue(identityNamed, 8)}
	for row := 1; row <= len(peopleRows); row++ {
		values = append(values, personValue(row))
	}
	seen := map[int32][]byte{}
	for _, v := range values {
		id, ok := s.identityOf(v)
		if !ok {
			continue
		}
		u, ok := mapload.Place(alm.Unit{}, id, x.table, s.mission())
		if !ok {
			t.Fatalf("identity %#x cannot be placed", v)
		}
		pic := s.NewRecordPreview(ui.KindUnit, "identity", v)
		c, tier := missionArt(t, s, u)
		want := s.unitPreview(c, tier)
		if pic == nil || want == nil || pic.Bounds() != want.Bounds() || !bytes.Equal(pic.Pix, want.Pix) {
			t.Errorf("identity %#x: cell differs from the mission's sprite", v)
			continue
		}
		r := mapload.Resolve(u, x.table)
		typeID := peopleRows[r.Index-1].typeID
		if c != x.units.Classes[typeID] {
			t.Errorf("identity %#x is drawn with another sheet than its type %#x", v, typeID)
		}
		if portrait != nil && bytes.Equal(pic.Pix, portrait.Pix) {
			t.Errorf("identity %#x shows the dialogue portrait", v)
		}
		if dir, face, ok := mapload.PlacedFigure(u, x.table); ok {
			if doll, _ := composeDoll(x.archives.artSource(), 0, data.Equipment{}, nil, dir, face, false); doll.Figure != nil && bytes.Equal(doll.Figure.Pix, pic.Pix) {
				t.Errorf("identity %#x shows the bare paper doll", v)
			}
		}
		if prev, ok := seen[typeID]; ok && !bytes.Equal(prev, pic.Pix) {
			t.Errorf("two rows of type %#x draw two pictures", typeID)
		}
		seen[typeID] = pic.Pix
	}
	if len(seen) != len(peopleClasses) {
		t.Fatalf("cells drew %d class sheets, want %d", len(seen), len(peopleClasses))
	}
}

// A mercenary cell places its type at level 1; the unit popup's level field
// switches the record to the type's row of that level as one edit that undo
// and redo travel, the other fields keep their values and the identity
// field shows the type's one cell at every level.
func TestEditorUnitMercenaryLevelSwitchesTheRowWithUndo(t *testing.T) {
	s := peopleIdentityEditor(t, "en")
	x := s.inspector
	if _, err := s.AddUnitIdentity(image.Pt(20, 20), personValue(5)); err != nil {
		t.Fatal(err)
	}
	m, _ := s.model.Map()
	index := len(m.AuthoredUnits) - 1
	row := func() int {
		m, _ := s.model.Map()
		id, _, _ := mapload.IdentifyAt(m.AuthoredUnits[index], x.table, s.mission())
		return id.Row
	}
	settled := func() map[string]int64 {
		out := map[string]int64{}
		for _, f := range s.Fields(ui.KindUnit, index) {
			if !f.ReadOnly && f.Key != "level" {
				out[f.Key] = f.Value
			}
		}
		return out
	}
	level := func() ui.InspectionField {
		fields := s.Fields(ui.KindUnit, index)
		return fields[fieldIndex(t, fields, "level")]
	}
	if row() != 5 {
		t.Fatalf("the mercenary cell placed row %d, want NPC03_1 (row 5)", row())
	}
	f := level()
	if f.Value != 1 || len(f.Choices) != 4 || f.Choices[0].Label != "1" || f.Choices[3].Value != 4 || f.Label != "Level" {
		t.Fatalf("level field = %+v, want Level 1 of 1..4", f)
	}
	before, raw := settled(), append([]byte(nil), s.model.Bytes()...)
	if _, err := s.SetField(ui.KindUnit, index, "level", 3); err != nil {
		t.Fatal(err)
	}
	after := append([]byte(nil), s.model.Bytes()...)
	if row() != 9 || level().Value != 3 {
		t.Fatalf("level 3 keyed row %d at level %d, want NPC03_3 (row 9)", row(), level().Value)
	}
	if got := settled(); !maps.Equal(got, before) {
		t.Fatalf("the level changed other fields: %v, was %v", got, before)
	}
	if _, err := s.SetField(ui.KindUnit, index, "identity", personValue(5)); err != nil || !bytes.Equal(after, s.model.Bytes()) {
		t.Fatal("choosing the type's own cell changed the level", err)
	}
	if _, err := s.Undo(); err != nil || !bytes.Equal(raw, s.model.Bytes()) || row() != 5 {
		t.Fatal("undo did not restore level 1", err)
	}
	if _, err := s.Redo(); err != nil || !bytes.Equal(after, s.model.Bytes()) || row() != 9 {
		t.Fatal("redo did not restore level 3", err)
	}
	if _, err := s.SetField(ui.KindUnit, index, "level", 5); err == nil {
		t.Fatal("a level the type has no row for was accepted")
	}
	fields := s.Fields(ui.KindUnit, 0)
	if slices.ContainsFunc(fields, func(f ui.InspectionField) bool { return f.Key == "level" }) {
		t.Fatal("a unit that is no mercenary has a level field")
	}
}
