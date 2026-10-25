package game

import (
	"image"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// armsMissionNumber is the campaign mission the arms test mods replace.
const armsMissionNumber = 20

// startArmsMission opens mission 20 from the test mod through the production
// route, headless.
func startArmsMission(t *testing.T, root string) (*FrontEnd, *ui.App) {
	t.Helper()
	f := guestFront(t, root, newMapModID)
	app := f.App("map arms")
	app.Layout(1024, 768)
	party := f.ChargenParty(ui.ChargenResult{Name: "Arms", Choices: []int{0, 0, 0}, Stats: []int{31, 27, 24, 29}})
	if err := app.OpenMission(f.MissionOpenerWith(armsMissionNumber, party)); err != nil {
		t.Fatal(err)
	}
	return f, app
}

// goalRowOf is the index of the panel row that adds the arm described by text.
func goalRowOf(t *testing.T, s ui.InspectionScript, arm, text string) int {
	t.Helper()
	return goalIndex(t, s.Goals(), arm, text)
}

// newArmsMod builds, through the window's own operations, a new 32x32 map with
// a drop cell, an enemy unit (id 1, roster slot 2) and a unit of the party's
// side (id 2, roster slot 1), optionally gives it the win arm on the enemy and
// the lose arm on the party unit, and saves it into a fresh mod.
func newArmsMod(t *testing.T, withArms bool) string {
	t.Helper()
	root := newMapMod(t)
	x := installedInspector(t)
	if err := x.SetMod(filepath.Join(root, newMapModID)); err != nil {
		t.Fatal(err)
	}
	class := creatureClass(t, x)
	e := x.Editor()
	if err := e.Open(newMapSource(32, 32)); err != nil {
		t.Fatal(err)
	}
	e.Layout(1280, 800)
	s := e.Document().Edits.(*mapInspectionEdits)
	steps := []struct {
		name string
		do   func() (*ui.InspectionDocument, error)
	}{
		// A new map's drop cell is its centre, 16,16; the window moves it.
		{"drop cell", func() (*ui.InspectionDocument, error) { return s.SetDropCell(image.Pt(14, 16)) }},
		{"enemy", func() (*ui.InspectionDocument, error) { return s.AddUnit(image.Pt(22, 22), int64(class)) }},
		{"party unit", func() (*ui.InspectionDocument, error) { return s.AddUnit(image.Pt(12, 22), int64(class)) }},
		{"party side", func() (*ui.InspectionDocument, error) { return s.SetField("Unit", 1, "owner", partySlot) }},
	}
	for _, st := range steps {
		if d, err := st.do(); err != nil || d == nil {
			t.Fatalf("%s: %v", st.name, err)
		}
	}
	if withArms {
		if d, err := s.SetGoalTarget(goalRowOf(t, s, armWin, "death of a unit"), "Unit", 0); err != nil || d == nil {
			t.Fatalf("add the win arm: %v", err)
		}
		if d, err := s.SetGoalTarget(goalRowOf(t, s, armLose, "party unit"), "Unit", 1); err != nil || d == nil {
			t.Fatalf("add the lose arm: %v", err)
		}
	}
	if err := pastNotice(e, "saveas"); err != nil || e.Dirty() {
		t.Fatalf("Save into the mod: %v", err)
	}
	return root
}

// TestReleaseMapEditorNewMapIsWonAndLostByItsArms proves on the lawful install,
// through the editor's script host, Save into a mod and the ordinary mission
// start: on a new map the enemy's death wins once the win arm is added and the
// party unit's death loses once the lose arm is. The same map without the arms
// is neither won nor lost by those deaths.
func TestReleaseMapEditorNewMapIsWonAndLostByItsArms(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	armed, plain := newArmsMod(t, true), newArmsMod(t, false)
	const enemy, ally = 1, 2

	f, app := startArmsMission(t, armed)
	if out := runScript(t, f, app, 200); out != sim.OutcomeUndecided {
		t.Fatalf("the armed map ends in %v before anyone dies", out)
	}
	killScriptUnit(t, f, enemy)
	if out := runScript(t, f, app, 600); out != sim.OutcomeWon {
		t.Fatalf("killing the enemy ends the armed map in %v", out)
	}

	f, app = startArmsMission(t, armed)
	killScriptUnit(t, f, ally)
	if out := runScript(t, f, app, 600); out != sim.OutcomeLost {
		t.Fatalf("the death of the party unit ends the armed map in %v", out)
	}

	f, app = startArmsMission(t, plain)
	killScriptUnit(t, f, enemy)
	killScriptUnit(t, f, ally)
	if out := runScript(t, f, app, 600); out != sim.OutcomeUndecided {
		t.Fatalf("loss control: the same deaths end the map without arms in %v", out)
	}
}

// armsCopy is the loose installed map a copy test edits, with the records the
// two arms name.
type armsCopy struct {
	key        string
	bytes      []byte
	group      uint32
	members    []int // unit records of the target group
	other      []int // unit records of a second group
	structure  int   // structure record the arm names
	otherStruc int   // a second structure with a standing health
}

// findArmsCopy picks the first loose map whose structures and groups an arm can
// name: two structures with a unique id and health, and two groups of few
// units each.
func findArmsCopy(t *testing.T, x *MapInspector) armsCopy {
	t.Helper()
	var keys []string
	for _, entry := range x.Maps {
		if !entry.FromArchive {
			keys = append(keys, "loose/"+entry.Source)
		}
	}
	sort.Strings(keys)
	for _, key := range keys {
		d, err := x.Open(key)
		if err != nil {
			t.Fatal(err)
		}
		m, err := alm.Open(d.SourceBytes)
		if err != nil {
			t.Fatal(err)
		}
		c := armsCopy{key: key, bytes: d.SourceBytes, structure: -1, otherStruc: -1}
		built := mapload.Structures(m, x.table)
		ids := map[uint16]int{}
		for _, o := range m.Objects {
			ids[o.Field12]++
		}
		for i, o := range m.Objects {
			if o.Field12 == 0 || ids[o.Field12] != 1 || int16(built[i].Field42) <= 0 {
				continue
			}
			if c.structure < 0 {
				c.structure = i
			} else if c.otherStruc < 0 {
				c.otherStruc = i
			}
		}
		groups := map[uint32][]int{}
		for i, u := range m.Units {
			if u.GroupID != 0 {
				groups[u.GroupID] = append(groups[u.GroupID], i)
			}
		}
		var order []uint32
		for g := range groups {
			order = append(order, g)
		}
		sort.Slice(order, func(a, b int) bool { return order[a] < order[b] })
		for _, g := range order {
			owners := map[uint32]bool{}
			for _, i := range groups[g] {
				owners[m.Units[i].Owner] = true
			}
			if len(owners) != 1 || len(groups[g]) > 4 {
				continue
			}
			if c.members == nil {
				c.group, c.members = g, groups[g]
			} else if c.other == nil {
				c.other = groups[g]
			}
		}
		if c.structure >= 0 && c.otherStruc >= 0 && c.members != nil && c.other != nil {
			return c
		}
	}
	t.Fatal("no loose map holds two standing structures and two small groups")
	return armsCopy{}
}

// armsCopyMod writes the copy into a fresh mod as mission 20's map, opens it in
// the editor as a host file, lets add edit its script and saves into the mod
// through the window's Save.
func armsCopyMod(t *testing.T, c armsCopy, add func(s ui.InspectionScript)) string {
	t.Helper()
	root := newMapMod(t)
	modDir := filepath.Join(root, newMapModID)
	target := filepath.Join(modDir, "mission", "map.alm")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, c.bytes, 0o644); err != nil {
		t.Fatal(err)
	}
	x := installedInspector(t)
	if err := x.SetMod(modDir); err != nil {
		t.Fatal(err)
	}
	e := x.Editor()
	if err := e.Open(target); err != nil {
		t.Fatal(err)
	}
	e.Layout(1280, 800)
	add(e.Document().Edits.(ui.InspectionScript))
	if err := pastNotice(e, "saveas"); err != nil || e.Dirty() {
		t.Fatalf("Save into the mod: %v", err)
	}
	return root
}

func killRecords(t *testing.T, f *FrontEnd, records []int) {
	t.Helper()
	for _, i := range records {
		if err := f.live.world.HeadlessKill(sim.EntityID(i)); err != nil {
			t.Fatal(err)
		}
	}
}

// TestReleaseMapEditorGroupAndStructureArmsOnAnInstalledMapCopy proves the two
// arms the new-map test does not use, on a copy of a loose installed map (which
// has no win arm): a group wipes out and a structure's health falls to zero.
// Each run first spends the deaths or the wound of a different group or
// structure, which must not win.
func TestReleaseMapEditorGroupAndStructureArmsOnAnInstalledMapCopy(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	c := findArmsCopy(t, installedInspector(t))
	t.Logf("copy of %s: group %d of %d units, structures %d and %d", c.key, c.group, len(c.members), c.structure, c.otherStruc)

	groupRoot := armsCopyMod(t, c, func(s ui.InspectionScript) {
		if d, err := s.SetGoalTarget(goalRowOf(t, s, armWin, "group wiped out"), "Unit", c.members[0]); err != nil || d == nil {
			t.Fatalf("add the group arm: %v", err)
		}
	})
	f, app := startArmsMission(t, groupRoot)
	if out := runScript(t, f, app, 200); out != sim.OutcomeUndecided {
		t.Fatalf("the group arm ends the copy in %v before anyone dies", out)
	}
	killRecords(t, f, c.other)
	if out := runScript(t, f, app, 900); out != sim.OutcomeUndecided {
		t.Fatalf("wiping out another group ends the copy in %v", out)
	}
	killRecords(t, f, c.members)
	if out := runScript(t, f, app, 2400); out != sim.OutcomeWon {
		t.Fatalf("wiping out group %d ends the copy in %v", c.group, out)
	}

	structRoot := armsCopyMod(t, c, func(s ui.InspectionScript) {
		if d, err := s.SetGoalTarget(goalRowOf(t, s, armWin, "structure destroyed"), "Structure", c.structure); err != nil || d == nil {
			t.Fatalf("add the structure arm: %v", err)
		}
	})
	f, app = startArmsMission(t, structRoot)
	if out := runScript(t, f, app, 200); out != sim.OutcomeUndecided {
		t.Fatalf("the structure arm ends the copy in %v before anything falls", out)
	}
	felled := func(index int) {
		health := []sim.OriginalStructureHealth{{ID: sim.StructureID(index), Health: 0, MaxHealth: 1}}
		if err := f.live.world.ImportOriginalStructureHealth(health); err != nil {
			t.Fatal(err)
		}
	}
	felled(c.otherStruc)
	if out := runScript(t, f, app, 300); out != sim.OutcomeUndecided {
		t.Fatalf("destroying another structure ends the copy in %v", out)
	}
	felled(c.structure)
	if out := runScript(t, f, app, 600); out != sim.OutcomeWon {
		t.Fatalf("destroying structure %d ends the copy in %v", c.structure, out)
	}
}
