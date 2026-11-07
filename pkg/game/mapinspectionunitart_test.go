package game

import (
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
)

// An edit of a unit's look redraws him, a move or an appended unit leaves the
// others as they were, and each answer is the one a map built from scratch
// gives. The fixture's type 3 sheet is the axe and its type 0x18 sheet the
// club.
func TestEditorMapViewRedrawsAPersonAfterAnEditOfHisLook(t *testing.T) {
	s := peopleIdentityEditor(t, "en")
	x := s.inspector
	x.units.Classes[3].Name, x.units.Classes[0x18].Name = "Axe", "Club"
	build := func(unit0 synth.ALMUnit, more ...synth.ALMUnit) map[uint32]string {
		raw := synth.ALM(synth.ALMOptions{Width: 40, Height: 40, Units: append([]synth.ALMUnit{unit0,
			{X: 12 << 8, Y: 10 << 8, ClassID: 3, DefID: 5001, UnitID: 2}}, more...)})
		model, err := mapedit.New(raw)
		if err != nil {
			t.Fatal(err)
		}
		doc, err := x.inspectModel(model, "art.alm", nil)
		if err != nil {
			t.Fatal(err)
		}
		out := map[uint32]string{}
		for i := range 2 + len(more) {
			if e, ok := doc.Viewer.EntityOf(uint32(i)); ok && e.Art != nil {
				out[uint32(i)] = e.Art.Name
			}
		}
		return out
	}
	club := synth.ALMUnit{X: 10 << 8, Y: 10 << 8, ClassID: 3, DefID: 5014, UnitID: 1}
	axe := synth.ALMUnit{X: 10 << 8, Y: 10 << 8, ClassID: 3, DefID: 5001, UnitID: 1}
	moved := club
	moved.X = 20 << 8
	appended := synth.ALMUnit{X: 16 << 8, Y: 10 << 8, ClassID: 3, DefID: 5014, UnitID: 3}
	for _, c := range []struct {
		name string
		got  map[uint32]string
		want []string
	}{
		{"first look", build(club), []string{"Club", "Axe"}},
		{"look edited to the axe row", build(axe), []string{"Axe", "Axe"}},
		{"look edited back", build(club), []string{"Club", "Axe"}},
		{"moved only", build(moved), []string{"Club", "Axe"}},
		{"a unit appended", build(club, appended), []string{"Club", "Axe", "Club"}},
		{"after the appended one", build(axe), []string{"Axe", "Axe"}},
	} {
		for i, w := range c.want {
			if c.got[uint32(i)] != w {
				t.Errorf("%s: unit %d drawn %q, want %q", c.name, i, c.got[uint32(i)], w)
			}
		}
	}
}

// The map view draws a placed person with the sheet the game draws him with:
// the sheet of his Humans row's type when the definition id names a row, not
// the sheet of his class key.
func TestEditorMapViewDrawsAPlacedPersonAsTheGameDoes(t *testing.T) {
	s := peopleIdentityEditor(t, "en")
	x := s.inspector
	axe, club := x.units.Classes[3], x.units.Classes[0x18]
	axe.Name, club.Name = "Axe", "Club"
	cases := []struct {
		name string
		unit synth.ALMUnit
		want string
	}{
		{"class key axe, row club", synth.ALMUnit{X: 10 << 8, Y: 10 << 8, ClassID: 3, DefID: 5014, UnitID: 1}, "Club"},
		{"class key club, row axe", synth.ALMUnit{X: 12 << 8, Y: 10 << 8, ClassID: 0x18, DefID: 5001, UnitID: 2}, "Axe"},
		{"class key axe, row axe", synth.ALMUnit{X: 14 << 8, Y: 10 << 8, ClassID: 3, DefID: 5001, UnitID: 3}, "Axe"},
	}
	var units []synth.ALMUnit
	for _, c := range cases {
		units = append(units, c.unit)
	}
	raw := synth.ALM(synth.ALMOptions{Width: 40, Height: 40, Units: units})
	doc, err := x.inspect(raw, "art.alm")
	if err != nil {
		t.Fatal(err)
	}
	m, err := alm.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	for i, c := range cases {
		game, _ := missionArt(t, s, m.Units[i])
		if game == nil || game.Name != c.want {
			t.Fatalf("%s: the game draws %v, want %s", c.name, game, c.want)
		}
		e, ok := doc.Viewer.EntityOf(uint32(i))
		if !ok {
			t.Fatalf("%s: the map view holds no unit %d", c.name, i)
		}
		if e.Art != game {
			drawn := "none"
			if e.Art != nil {
				drawn = e.Art.Name
			}
			t.Errorf("%s: the map view draws sheet %s, the game draws %s", c.name, drawn, game.Name)
		}
	}
}
