package game

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/sav"
	"againrom/pkg/ui"
)

// secondFortyPersons is mission 40's 35 persons by Humans row and record, with
// the world sprite and the paperdoll body the claims give each row
// (R2-ASSET-172, R2-ASSET-173, R2-ENGINE-680, R2-ENGINE-681).
var secondFortyPersons = []struct {
	row          string
	records      []int
	sprite, body string
}{
	{"2U_PeasantMale", []int{25, 26, 27, 28, 29, 35}, "units/humans/unarmed/sprites.256", "equipment/mfighter/46.256"},
	{"2U_PeasantFemale", []int{30, 31, 32, 33, 34}, "units/humans/unarmed/sprites.256", "equipment/ffighter/13.256"},
	{"2U_PeasantLeader", []int{9}, "units/humans/unarmed/sprites.256", "equipment/mfighter/44.256"},
	{"2F_PeasantGuard", []int{10, 11, 12, 13, 14, 15, 16, 17}, "units/humans/clubman/clubman.256", "equipment/mfighter/45.256"},
	{"2A_PeasantGuard", []int{18, 19, 20, 21, 22, 23, 24}, "units/humans/archer/archer.256", "equipment/ffighter/16.256"},
	{"Mage_Hero", []int{36}, "units/heroes/mage/sprites.256", "equipment/fmage/20.256"},
	{"2F_Brigand2", []int{0, 39, 40}, "units/humans/axeman_sh/axeman_sh.256", "equipment/mfighter/51.256"},
	{"2A_Brigand2", []int{3, 4}, "units/humans/archer/archer.256", "equipment/ffighter/17.256"},
	{"2M_Brigand2", []int{2}, "units/humans/mage_st/mage_st.256", "equipment/fmage/15.256"},
	{"2F_BrigandLeader2", []int{5}, "units/humans/axeman_2hd/axeman_2hd.256", "equipment/mfighter/52.256"},
}

// secondPersonLook is what one placed person draws with: the world sheet, the
// paperdoll body and the face byte his actor carries.
type secondPersonLook struct {
	Row          string
	Sprite, Body string
	FaceByte     uint8
}

// TestReleaseSecondMissionPersonsDrawTheirOwnLook opens missions 40 and 30 of
// the second game, reads each person's world sprite, paperdoll body and face
// byte, and compares them by record with the claims: mission 40 against the
// written table, mission 30 by the same rule from its own map and rows. A SAVE
// stores each face byte, a cold LOAD keeps every key, and a resave of the
// loaded mission and its LOAD keep them again.
func TestReleaseSecondMissionPersonsDrawTheirOwnLook(t *testing.T) {
	for _, mission := range []int{40, 30} {
		t.Run(fmt.Sprintf("mission %d", mission), func(t *testing.T) {
			f, app := secondPersonsMission(t, mission)
			m := f.live.mission.state.Map
			want := secondPersonsWanted(t, f, m, mission)
			fresh := secondPersonLooks(t, f, f.live, m)
			secondPersonsCompare(t, "fresh", want, fresh)
			for _, step := range []string{"loaded", "loaded again"} {
				hash := f.live.world.Hash()
				t.Logf("%s: saved World hash %#x", step, hash)
				out := t.TempDir()
				secondPersonsSaved(t, step, secondMissionNamedSave(t, f, app, out, "Persons"), want)
				f, app = secondMissionColdAt(t, out, secondSaveFile(t, out, "Persons"), mission)
				if f.live.world.Hash() != hash {
					t.Fatal(step, "differs from the saved World")
				}
				secondPersonsCompare(t, step, want, secondPersonLooks(t, f, f.live, m))
			}
		})
	}
}

// secondPersonsMission plays a new campaign into mission 30 after Iglez joins,
// or into mission 40 after the stage 40 inn talks.
func secondPersonsMission(t *testing.T, mission int) (*FrontEnd, *ui.App) {
	t.Helper()
	if mission == 30 {
		f, app, _ := secondIglezJoins(t)
		secondLaterChoose(t, app, "EXIT", "GATES")
		secondLaterEnter(t, f, app, 30)
		return f, app
	}
	f := secondGameFront(t)
	app := f.App("mission persons")
	app.Layout(1024, 768)
	secondStageFortyInn(t, f, app)
	secondStageFortyTalks(t, f, app)
	secondLaterChoose(t, app, "EXIT", "GATES")
	secondLaterEnter(t, f, app, 40)
	return f, app
}

// secondPersonsWanted is the look the claims give each person record of m, keyed
// by its map unit id. Mission 40 reads the table above; any other mission states
// the rule from its rows: the units.reg File of the row's type id, the hero
// mage's sheet for NPC words 21..23, and the body of the row's face with its
// gender in bit 7 (R2-ENGINE-680, R2-ENGINE-681).
func secondPersonsWanted(t *testing.T, f *FrontEnd, m *alm.Map, mission int) map[uint16]secondPersonLook {
	t.Helper()
	classes, err := loadUnitClasses(f.Archives.Containers)
	if err != nil {
		t.Fatal(err)
	}
	sprites := map[int32]string{}
	for _, c := range classes.All() {
		sprites[c.ID] = secondSheetKey(c.SpritePath())
	}
	rows := map[int32]int{}
	for i := 1; i < f.Table.Humans.Len(); i++ {
		if p := f.Table.Humans.EntryParams(i); len(p) > 24 {
			if _, dup := rows[p[24]]; !dup {
				rows[p[24]] = i
			}
		}
	}
	byRecord := map[int]string{}
	if mission == 40 {
		for _, g := range secondFortyPersons {
			for _, r := range g.records {
				byRecord[r] = g.row
			}
		}
	}
	out := map[uint16]secondPersonLook{}
	for i, u := range m.Units {
		if u.Flags&0x10 == 0 {
			continue
		}
		n, ok := rows[int32(u.ServerID)]
		if !ok {
			t.Fatalf("record %d: server id %d names no Humans row", i, u.ServerID)
		}
		p := f.Table.Humans.EntryParams(n)
		typeID, face, gender, mana := p[16], p[17], p[18], p[5]
		b := uint8(face) | uint8(gender&1)<<7
		look := secondPersonLook{Row: f.Table.Humans.EntryName(n), Sprite: sprites[typeID], FaceByte: b,
			Body: secondSheetKey(data.ItemFigureBasePath(data.FigureDirFor(typeID == 0x17 || typeID == 0x18, b&0x80 != 0), int(b&0x7f)))}
		word := u.ServerID
		if word > 10000 {
			word = word / 10 % 1000
		}
		if word >= 21 && word <= 23 {
			if mana <= 0 {
				t.Fatalf("record %d (%s): a hero fighter's sheet is not stated here", i, look.Row)
			}
			look.Sprite = "units/heroes/mage/sprites.256"
		}
		if mission == 40 {
			if byRecord[i] != look.Row {
				t.Fatalf("record %d is %s, R2-ASSET-172 gives %q", i, look.Row, byRecord[i])
			}
			for _, g := range secondFortyPersons {
				if g.row == look.Row && (g.sprite != look.Sprite || g.body != look.Body) {
					t.Fatalf("record %d (%s): rule gives %s and %s, the claims give %s and %s", i, look.Row, look.Sprite, look.Body, g.sprite, g.body)
				}
			}
		}
		if _, dup := out[u.UnitID]; dup || u.UnitID == 0 {
			t.Fatalf("record %d: map unit id %d is not unique", i, u.UnitID)
		}
		out[u.UnitID] = look
	}
	if mission == 40 && len(out) != 35 || mission == 30 && len(out) != 48 {
		t.Fatalf("mission %d has %d persons", mission, len(out))
	}
	return out
}

// secondPersonLooks reads each person entity of live: the class or body the
// renderer draws him with and the figure his inspection composes.
func secondPersonLooks(t *testing.T, f *FrontEnd, live *mapWorld, m *alm.Map) map[uint16]secondPersonLook {
	t.Helper()
	classes, err := loadUnitClasses(f.Archives.Containers)
	if err != nil {
		t.Fatal(err)
	}
	persons := map[uint16]bool{}
	for _, u := range m.Units {
		if u.Flags&0x10 != 0 {
			persons[u.UnitID] = true
		}
	}
	out := map[uint16]secondPersonLook{}
	for _, e := range live.world.Entities() {
		if !persons[e.MapUnitID] {
			continue
		}
		c := live.units.Classes[e.Class]
		if art := live.art[e.ID]; art != nil {
			c = art
		}
		var look secondPersonLook
		for _, uc := range classes.All() {
			if live.units.Classes[uc.ID] == c {
				look.Sprite = secondSheetKey(uc.SpritePath())
			}
		}
		for key, body := range live.units.Bodies {
			if body == c {
				dir, name, _ := strings.Cut(key, "/")
				look.Sprite = secondSheetKey(data.HeroSheetPath(dir, data.HeroBody(name)))
			}
		}
		fig, ok := live.figures[e.ID]
		if !ok || live.inspectionUnitPicture(uint32(e.ID)) == nil {
			t.Fatalf("entity %d (map unit %d) has no paperdoll", e.ID, e.MapUnitID)
		}
		look.Body = secondSheetKey(data.ItemFigureBasePath(fig.Dir, fig.Face))
		if r, ok := live.mission.state.Start.Roster[e.ID]; ok {
			look.Row = r.Name
		}
		out[e.MapUnitID] = look
	}
	return out
}

func secondPersonsCompare(t *testing.T, when string, want, got map[uint16]secondPersonLook) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("%s: %d person entities, want %d", when, len(got), len(want))
	}
	for unit, w := range want {
		g := got[unit]
		if g.Sprite != w.Sprite || g.Body != w.Body {
			t.Errorf("%s: map unit %d (%s) draws %s and %s, want %s and %s", when, unit, w.Row, g.Sprite, g.Body, w.Sprite, w.Body)
		}
	}
	if !t.Failed() {
		t.Logf("%s: %d persons match", when, len(want))
	}
}

// secondPersonsSaved reads the face byte SAVE stored for each person.
func secondPersonsSaved(t *testing.T, when string, raw []byte, looks map[uint16]secondPersonLook) {
	t.Helper()
	file, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	graph, err := file.ActorGraph()
	if err != nil {
		t.Fatal(err)
	}
	saved := map[uint16]sav.ActorRecord{}
	for _, r := range graph.Actors {
		saved[r.MapUnitID] = r
	}
	for unit, look := range looks {
		r, ok := saved[unit]
		if !ok || r.Face != look.FaceByte {
			t.Errorf("%s: map unit %d (%s): saved face byte %#x present %t, want %#x", when, unit, look.Row, r.Face, ok, look.FaceByte)
		}
	}
}

// secondSheetKey is an archive address from its units/ or equipment/ segment,
// lower case with forward separators.
func secondSheetKey(p string) string {
	p = strings.ToLower(filepath.ToSlash(strings.ReplaceAll(p, "\\", "/")))
	for _, seg := range []string{"units/", "equipment/"} {
		if i := strings.Index(p, seg); i >= 0 {
			return p[i:]
		}
	}
	return p
}
