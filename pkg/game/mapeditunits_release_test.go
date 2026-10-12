package game

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/modrt"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// unitPicks are the installed map's records the release test edits: S is
// cloned onto Cell, D deleted, C changed to Owner, Class, Def and HP. Each is
// an interior creature whose unit id no other record shares.
type unitPicks struct {
	S, D, C       int
	Cell          image.Point
	Owner         uint32
	Class         int16
	Def           uint32
	HP            int16
	SID, DID, CID uint16
	CloneID       uint16
}

func pickUnits(t *testing.T, m *alm.Map, table *mapload.Table) unitPicks {
	t.Helper()
	ids := map[uint16]int{}
	cells := map[image.Point]bool{}
	var highest uint16
	for _, u := range m.Units {
		ids[u.UnitID]++
		cells[image.Pt(int(u.X>>8), int(u.Y>>8))] = true
		highest = max(highest, u.UnitID)
	}
	var picks []int
	for i, u := range m.Units {
		x, y := int(u.X>>8), int(u.Y>>8)
		if u.ClassID >= 0x1a && u.UnitID != 0 && ids[u.UnitID] == 1 && x >= 10 && y >= 10 && x < int(m.Width)-10 && y < int(m.Height)-10 {
			picks = append(picks, i)
		}
	}
	if len(picks) < 3 || len(m.Groups) < 2 {
		t.Fatalf("the map offers %d interior creatures and %d sides", len(picks), len(m.Groups))
	}
	p := unitPicks{S: picks[0], D: picks[1], C: picks[2], CloneID: highest + 1}
	s, c := m.Units[p.S], m.Units[p.C]
	p.SID, p.DID, p.CID = s.UnitID, m.Units[p.D].UnitID, c.UnitID
	for dx := 1; ; dx++ {
		cell := image.Pt(int(s.X>>8)+dx, int(s.Y>>8))
		if !cells[cell] {
			p.Cell = cell
			break
		}
	}
	p.Owner = c.Owner%uint32(len(m.Groups)) + 1
	for _, u := range m.Units {
		probe := alm.Unit{ClassID: u.ClassID, ClassSubID: c.ClassSubID}
		if u.ClassID >= 0x1a && u.ClassID != c.ClassID && mapload.Resolve(probe, table).Index != data.NotFound {
			p.Class = u.ClassID
			break
		}
	}
	for i := 1; i < table.Humans.Len() && p.Def == 0; i++ {
		if v := table.Humans.EntryParams(i)[0x18]; v > 0 {
			p.Def = uint32(v)
		}
	}
	if p.Class == 0 || p.Def == 0 {
		t.Fatalf("no second class (%d) or definition id (%d) to set", p.Class, p.Def)
	}
	p.HP = 7
	return p
}

// saveEditedGuestMap opens the installed mission-10 map in the editor host,
// applies edit, and Save As writes it over the mod's assembled map path.
func saveEditedGuestMap(t *testing.T, x *MapInspector, dir string, edit func(*ui.MapEditor)) {
	t.Helper()
	lang := modrt.LanguageFor(BaseID(InspectInstall(os.Getenv("AGAINROM_ASSETS"))))
	target := filepath.Join(dir, guestModID, "mission", lang, "map.alm")
	if err := os.Remove(target); err != nil {
		t.Fatal(err)
	}
	e := x.Editor()
	if err := e.Open("scenario/10.alm"); err != nil {
		t.Fatal(err)
	}
	e.Layout(1280, 800)
	edit(e)
	_ = e.Key("saveas")
	e.TextInput(target)
	if err := e.Key("enter"); err != nil || e.Dirty() {
		t.Fatalf("Save As: %v", err)
	}
}

func guestWorld(t *testing.T, dir string) ([]sim.Entity, *alm.Map) {
	t.Helper()
	f := guestFront(t, dir, guestModID)
	app := f.App("edited guest mission")
	party := f.ChargenParty(ui.ChargenResult{Name: "Guest", Choices: []int{0, 0, 0}, Stats: []int{31, 27, 24, 29}})
	if err := app.OpenMission(f.MissionOpenerWith(guestMissionNo, party)); err != nil {
		t.Fatal(err)
	}
	return f.live.world.Entities(), f.live.mission.state.Map
}

func byMapUnitID(ents []sim.Entity, id uint16) []sim.Entity {
	var out []sim.Entity
	for _, e := range ents {
		if e.MapUnitID == id {
			out = append(out, e)
		}
	}
	return out
}

func TestReleaseMapEditorUnitAuthoringPlaysAsModMission(t *testing.T) {
	plain := guestMods(t, guestModID)
	edited := guestMods(t, guestModID)
	x, err := NewMapInspector(os.Getenv("AGAINROM_ASSETS"))
	if err != nil {
		t.Fatal(err)
	}
	raw, err := x.archives.Containers.ReadFile("scenario/10.alm")
	if err != nil {
		t.Fatal(err)
	}
	m, err := alm.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	p := pickUnits(t, m, x.table)
	t.Logf("picks %+v", p)

	saveEditedGuestMap(t, x, plain, func(*ui.MapEditor) {})
	before, _ := guestWorld(t, plain)
	for _, id := range []uint16{p.SID, p.DID, p.CID} {
		if len(byMapUnitID(before, id)) != 1 {
			t.Fatalf("loss control: unit id %d is not one entity in the unedited world", id)
		}
	}
	if len(byMapUnitID(before, p.CloneID)) != 0 {
		t.Fatal("loss control: the clone's id already plays")
	}
	src := byMapUnitID(before, p.SID)[0]

	saveEditedGuestMap(t, x, edited, func(e *ui.MapEditor) {
		docs := e.Document().Edits
		apply := func(d *ui.InspectionDocument, err error) {
			t.Helper()
			if err != nil || d == nil {
				t.Fatalf("edit refused: %v", err)
			}
		}
		apply(docs.CloneUnit(p.S, p.Cell))
		for _, f := range []struct {
			key string
			v   int64
		}{{"owner", int64(p.Owner)}, {"class", int64(p.Class)}, {"definition", int64(p.Def)}, {"hp", int64(p.HP)}} {
			apply(docs.SetUnitField(p.C, f.key, f.v))
		}
		apply(docs.DeleteUnit(p.D))
	})
	after, built := guestWorld(t, edited)
	clone := byMapUnitID(after, p.CloneID)
	if len(clone) != 1 || clone[0].X != int32(p.Cell.X) || clone[0].Y != int32(p.Cell.Y) || clone[0].Class != src.Class || clone[0].TypeID != src.TypeID || clone[0].MaxHP != src.MaxHP || clone[0].Group != src.Group {
		t.Fatalf("clone %+v from source class %d type %d at %v", clone, src.Class, src.TypeID, p.Cell)
	}
	if len(byMapUnitID(after, p.DID)) != 0 {
		t.Fatal("the deleted unit plays")
	}
	c := byMapUnitID(after, p.CID)
	if len(c) != 1 || c[0].Owner != p.Owner || c[0].Class != int32(p.Class) || c[0].HP != int32(p.HP) {
		t.Fatalf("changed unit %+v, want owner %d class %d hp %d", c, p.Owner, p.Class, p.HP)
	}
	found := false
	for _, u := range built.Units {
		if u.UnitID == p.CID {
			found = u.DefID == p.Def
		}
	}
	if !found {
		t.Fatalf("the built map does not carry definition id %d on unit %d", p.Def, p.CID)
	}
	if len(after) != len(before) {
		t.Fatalf("world holds %d entities, want %d (one clone, one delete)", len(after), len(before))
	}
}

// scriptedHighest picks the first installed map, by name order, whose highest
// unit id a Target_Unit parameter of its script names and whose next lower
// unit ids are contiguous, so that one above the highest id left after the
// delete would be the deleted id. It answers that record's index and a
// different record to clone.
func scriptedHighest(t *testing.T, x *MapInspector) (name string, raw []byte, del, src int) {
	t.Helper()
	for n := 1; n <= 400; n++ {
		name = fmt.Sprintf("scenario/%d.alm", n)
		b, err := x.archives.Containers.ReadFile(name)
		if err != nil {
			continue
		}
		m, err := alm.Open(b)
		if err != nil || len(m.Units) < 2 {
			continue
		}
		sc, err := m.Script()
		if err != nil {
			continue
		}
		del = 0
		for i, u := range m.Units {
			if u.UnitID >= m.Units[del].UnitID {
				del = i
			}
		}
		next := uint16(0) // the highest id left once that record is deleted
		for i, u := range m.Units {
			if i != del {
				next = max(next, u.UnitID)
			}
		}
		if next+1 != m.Units[del].UnitID {
			continue // a gap below it: the plain next id would not collide
		}
		for _, nodes := range [][]alm.ScriptNode{sc.Actions, sc.Conditions} {
			for _, nd := range nodes {
				for k := range nd.Type {
					if nd.Type[k] == 4 && nd.Value[k] == uint32(m.Units[del].UnitID) {
						return name, b, del, (del + 1) % len(m.Units)
					}
				}
			}
		}
	}
	t.Fatal("no installed map has its highest unit id named by a Target_Unit parameter")
	return
}

func TestReleaseMapEditorCloneNeverTakesADeletedScriptedUnitID(t *testing.T) {
	if os.Getenv("AGAINROM_ASSETS") == "" {
		t.Skip("AGAINROM_ASSETS names no install")
	}
	x, err := NewMapInspector(os.Getenv("AGAINROM_ASSETS"))
	if err != nil {
		t.Fatal(err)
	}
	name, raw, del, src := scriptedHighest(t, x)
	m, _ := alm.Open(raw)
	deleted := m.Units[del].UnitID
	if bound := mapload.ScriptUnits(m, nil)[deleted]; bound != sim.EntityID(del) {
		t.Fatalf("loss control: the unedited map binds id %d to entity %d, want %d", deleted, bound, del)
	}
	e := x.Editor()
	if err := e.Open(name); err != nil {
		t.Fatal(err)
	}
	e.Layout(1280, 800)
	edits := e.Document().Edits
	cell := image.Pt(int(m.Units[src].X>>8)+1, int(m.Units[src].Y>>8))
	if d, err := edits.DeleteUnit(del); err != nil || d == nil {
		t.Fatalf("delete refused: %v", err)
	}
	if d, err := edits.CloneUnit(src, cell); err != nil || d == nil {
		t.Fatalf("clone refused: %v", err)
	}
	out := filepath.Join(t.TempDir(), "edited.alm")
	_ = e.Key("saveas")
	e.TextInput(out)
	if err := e.Key("enter"); err != nil || e.Dirty() {
		t.Fatalf("Save As: %v", err)
	}
	saved, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	m2, err := alm.Open(saved)
	if err != nil || len(m2.Units) != len(m.Units) {
		t.Fatalf("edited map: %v, %d units, want %d", err, len(m2.Units), len(m.Units))
	}
	clone := m2.Units[len(m2.Units)-1]
	if clone.UnitID == deleted {
		t.Fatalf("%s: the clone took the deleted unit's id %d", name, deleted)
	}
	if id, ok := mapload.ScriptUnits(m2, nil)[deleted]; ok {
		t.Fatalf("%s: script id %d binds to entity %d after its unit was deleted", name, deleted, id)
	}
	t.Logf("%s: deleted record %d (id %d), cloned record %d as id %d", name, del, deleted, src, clone.UnitID)
}
