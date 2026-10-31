package game

import (
	"bytes"
	"image"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// scriptMissionNumber is the installed campaign mission whose win is the death
// of one unit.
const scriptMissionNumber = 131

// scriptWinTarget is the unit id the installed map's first win arm reads.
func scriptWinTarget(t *testing.T, m *alm.Map) uint32 {
	t.Helper()
	sc, err := m.Script()
	if err != nil {
		t.Fatal(err)
	}
	for _, g := range scriptGoalRows(sc) {
		if g.arm == armWin && g.param == scriptUnitParam && g.value < scriptHeroBand {
			return g.value
		}
	}
	t.Fatal("the installed map has no win arm that reads a unit")
	return 0
}

// unscriptedUnit is the index of the first interior unit whose id no other
// unit shares and that no script node names, directly or by its group.
func unscriptedUnit(t *testing.T, m *alm.Map) (int, uint16) {
	t.Helper()
	sc, err := m.Script()
	if err != nil {
		t.Fatal(err)
	}
	named := map[uint32]bool{}
	groups := map[uint32]bool{}
	for _, nodes := range [][]alm.ScriptNode{sc.Actions, sc.Conditions} {
		for _, n := range nodes {
			for p, tag := range n.Type {
				switch tag {
				case scriptUnitParam:
					named[n.Value[p]] = true
				case scriptGroupParam:
					groups[n.Value[p]] = true
				}
			}
		}
	}
	count := map[uint16]int{}
	for _, u := range m.Units {
		count[u.UnitID]++
	}
	for i, u := range m.Units {
		x, y := int(u.X>>8), int(u.Y>>8)
		if u.UnitID != 0 && count[u.UnitID] == 1 && !named[uint32(u.UnitID)] && !groups[u.GroupID] && u.ClassID >= 0x1a &&
			x >= 12 && y >= 12 && x < int(m.Width)-12 && y < int(m.Height)-12 {
			return i, u.UnitID
		}
	}
	t.Fatal("no interior unit is free of the script")
	return 0, 0
}

// freeDropCell is a ground cell well inside the map, away from every unit and
// from the map's own drop cell.
func freeDropCell(t *testing.T, m *alm.Map) image.Point {
	t.Helper()
	plane := mapload.Passability(m)
	old := mapload.DropCells(m)
	for y := 14; y < int(m.Height)-14; y++ {
		for x := 14; x < int(m.Width)-14; x++ {
			if sim.DomainGround.BlockedBy(plane[y*m.Width+x]) {
				continue
			}
			near := false
			for _, u := range m.Units {
				dx, dy := x-int(u.X>>8), y-int(u.Y>>8)
				near = near || dx*dx+dy*dy < 100
			}
			for _, d := range old {
				dx, dy := x-int(d.X), y-int(d.Y)
				near = near || dx*dx+dy*dy < 100
			}
			if !near {
				return image.Pt(x, y)
			}
		}
	}
	t.Fatal("no free ground cell for the drop")
	return image.Point{}
}

// startScriptMission opens campaign mission number through the production
// route, in a front end that serves the mod dir (none when dir is empty).
func startScriptMission(t *testing.T, dir string, number int) (*FrontEnd, *ui.App) {
	t.Helper()
	var f *FrontEnd
	if dir == "" {
		f = guestFront(t, "")
	} else {
		f = guestFront(t, dir, replaceModID)
	}
	party := f.ChargenParty(ui.ChargenResult{Name: "Script", Choices: []int{0, 0, 0}, Stats: []int{31, 27, 24, 29}})
	app := f.App("script mission")
	if err := app.OpenMission(f.MissionOpenerWith(number, party)); err != nil {
		t.Fatal(err)
	}
	return f, app
}

// runScript steps the mission for frames, closing every notice, and reports the
// outcome it ends on.
func runScript(t *testing.T, f *FrontEnd, app *ui.App, frames int) sim.Outcome {
	t.Helper()
	for i := 0; i < frames; i++ {
		if _, kind, up := f.LiveNotice(); up {
			if kind == ui.NoticeSuccess {
				break
			}
			if err := app.HeadlessActivate("notice"); err != nil {
				t.Fatal(err)
			}
		}
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
		if out := f.live.world.Outcome(); out != sim.OutcomeUndecided {
			return out
		}
	}
	return f.live.world.Outcome()
}

func killScriptUnit(t *testing.T, f *FrontEnd, id uint32) {
	t.Helper()
	st := f.live.mission.state
	ref, ok := mapload.ScriptUnits(st.Map, st.Party)[uint16(id)]
	if !ok {
		t.Fatalf("the mission does not play unit %d", id)
	}
	if err := f.live.world.HeadlessKill(ref); err != nil {
		t.Fatal(err)
	}
}

// editScriptMap opens the installed map in the editor with the mod, applies
// edit to its script host and saves into the mod through the window's Save.
func editScriptMap(t *testing.T, x *MapInspector, edit func(ui.InspectionScript, *alm.Map)) {
	t.Helper()
	e := x.Editor()
	if err := e.Open("scenario/" + strconv.Itoa(scriptMissionNumber) + ".alm"); err != nil {
		t.Fatal(err)
	}
	e.Layout(1280, 800)
	m, err := alm.Open(e.Document().SourceBytes)
	if err != nil {
		t.Fatal(err)
	}
	edit(e.Document().Edits.(ui.InspectionScript), m)
	if err := e.Key("saveas"); err != nil || e.Dirty() {
		t.Fatalf("Save into the mod: %v", err)
	}
}

// TestReleaseMapEditorMovesTheDropCellAndRetargetsTheWinArm proves on the
// lawful install, through the editor's script host, Save into a mod and the
// ordinary mission start: the party lands at the moved drop cell, the old
// win target no longer wins and the new one does. The same mission without the
// mod is the loss control.
func TestReleaseMapEditorMovesTheDropCellAndRetargetsTheWinArm(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	dir := replaceModNumber(t, scriptMissionNumber)
	install := os.Getenv("AGAINROM_ASSETS")
	x, err := NewMapInspector(install)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := x.archives.Containers.ReadFile("scenario/" + strconv.Itoa(scriptMissionNumber) + ".alm")
	if err != nil {
		t.Fatal(err)
	}
	installed, err := alm.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	oldTarget := scriptWinTarget(t, installed)
	newIndex, newTarget := unscriptedUnit(t, installed)
	cell := freeDropCell(t, installed)
	if err := x.SetMod(filepath.Join(dir, replaceModID)); err != nil {
		t.Fatal(err)
	}

	// Loss control: the installed mission lands at its own drop cell, and its
	// own target wins it.
	plain, plainApp := startScriptMission(t, "", scriptMissionNumber)
	if got := plain.live.mission.state.Start.Drop; got != mapload.DropCells(installed)[0] || plain.live.mission.state.Start.Fallback {
		t.Fatalf("loss control: the party landed at %v", got)
	}
	killScriptUnit(t, plain, oldTarget)
	if out := runScript(t, plain, plainApp, 600); out != sim.OutcomeWon {
		t.Fatalf("loss control: killing unit %d ends in %v", oldTarget, out)
	}

	editScriptMap(t, x, func(s ui.InspectionScript, _ *alm.Map) {
		goals := s.Goals()
		arm := -1
		for i, g := range goals {
			if g.Arm == armWin && g.Pick == "Unit" && arm < 0 {
				arm = i
			}
		}
		if arm < 0 {
			t.Fatal("no win arm")
		}
		if d, err := s.SetDropCell(cell); err != nil || d == nil {
			t.Fatalf("move the drop cell: %v", err)
		}
		if d, err := s.SetGoalTarget(arm, "Unit", newIndex); err != nil || d == nil {
			t.Fatalf("retarget the win arm: %v", err)
		}
	})

	// The mod's mission: the party lands at the moved cell.
	f, app := startScriptMission(t, dir, scriptMissionNumber)
	st := f.live.mission.state
	if st.Start.Drop != (mapload.Cell{X: int32(cell.X), Y: int32(cell.Y)}) || st.Start.Fallback || st.Start.Cells[0] != st.Start.Drop {
		t.Fatalf("the party landed at %v (hero %v), want %v", st.Start.Drop, st.Start.Cells[0], cell)
	}
	// The old target no longer wins.
	killScriptUnit(t, f, oldTarget)
	if out := runScript(t, f, app, 600); out != sim.OutcomeUndecided {
		t.Fatalf("killing the old target %d ends in %v", oldTarget, out)
	}
	// The new target does.
	killScriptUnit(t, f, uint32(newTarget))
	if out := runScript(t, f, app, 600); out != sim.OutcomeWon {
		t.Fatalf("killing the new target %d ends in %v", newTarget, out)
	}
}

// TestReleaseMapEditorSetsAProtectUnitCheck proves a protect check the editor
// adds: the protected unit's death loses the mission, and the same death
// without the check does not.
func TestReleaseMapEditorSetsAProtectUnitCheck(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	dir := replaceModNumber(t, scriptMissionNumber)
	install := os.Getenv("AGAINROM_ASSETS")
	x, err := NewMapInspector(install)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := x.archives.Containers.ReadFile("scenario/" + strconv.Itoa(scriptMissionNumber) + ".alm")
	if err != nil {
		t.Fatal(err)
	}
	installed, err := alm.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	index, id := unscriptedUnit(t, installed)
	if err := x.SetMod(filepath.Join(dir, replaceModID)); err != nil {
		t.Fatal(err)
	}

	plain, plainApp := startScriptMission(t, "", scriptMissionNumber)
	killScriptUnit(t, plain, uint32(id))
	if out := runScript(t, plain, plainApp, 600); out != sim.OutcomeUndecided {
		t.Fatalf("loss control: the unit's death ends the installed mission in %v", out)
	}

	editScriptMap(t, x, func(s ui.InspectionScript, _ *alm.Map) {
		goals := s.Goals()
		if d, err := s.SetGoalTarget(len(goals)-1, "Unit", index); err != nil || d == nil {
			t.Fatalf("add the protect check: %v", err)
		}
	})
	f, app := startScriptMission(t, dir, scriptMissionNumber)
	if out := runScript(t, f, app, 200); out != sim.OutcomeUndecided {
		t.Fatalf("the protect check ends the mission in %v before the unit dies", out)
	}
	killScriptUnit(t, f, uint32(id))
	if out := runScript(t, f, app, 600); out != sim.OutcomeLost {
		t.Fatalf("the death of protected unit %d ends in %v", id, out)
	}
}

// TestReleaseEveryInstalledMapListsItsScriptArmsAndWritesItsBytes opens every
// first-game map: the script panel lists its arms, every arm resolves its
// target to a unit, structure or group on the map or to the hero band, a map
// with a win trigger has a win arm, and opening and saving writes the
// installed bytes.
func TestReleaseEveryInstalledMapListsItsScriptArmsAndWritesItsBytes(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	out := t.TempDir()
	arms := map[string]int{}
	for n, entry := range x.Maps {
		key := "loose/" + entry.Source
		if entry.FromArchive {
			key = "scenario/" + entry.Source
		}
		d, err := x.Open(key)
		if err != nil {
			t.Fatalf("%s: %v", key, err)
		}
		m, err := alm.Open(d.SourceBytes)
		if err != nil {
			t.Fatal(err)
		}
		sc, err := m.Script()
		if err != nil {
			t.Fatalf("%s: %v", key, err)
		}
		rows := scriptGoalRows(sc)
		goals := d.Edits.(ui.InspectionScript).Goals()
		if len(goals) != len(rows) || goals[len(goals)-1].Index != -1 {
			t.Fatalf("%s: %d arms listed for %d rows", key, len(goals), len(rows))
		}
		for _, g := range rows {
			if g.node < 0 {
				continue
			}
			arms[g.arm]++
			if text, _, _ := describeTarget(m, ui.HeroLabels{Format: "party role %d"}, g.param, g.value); strings.Contains(text, "not on the map") {
				t.Logf("%s: %s arm names %s", key, g.arm, text)
			}
		}
		// A map that has a win arm row offers no row that adds one.
		for _, g := range rows {
			if g.node < 0 && g.add != addProtect && hasArmRow(rows, g.arm) {
				t.Fatalf("%s: the panel offers to add a %s arm to a map that has one", key, g.arm)
			}
		}
		target := filepath.Join(out, "map"+strconv.Itoa(n)+".alm")
		saved, err := d.Edits.SaveAs(target)
		if err != nil {
			t.Fatalf("%s: Save As of the unedited map: %v", key, err)
		}
		got, err := os.ReadFile(saved)
		if err != nil || !bytes.Equal(got, d.SourceBytes) {
			t.Fatalf("%s: the unedited save differs from the installed bytes: %v", key, err)
		}
	}
	t.Logf("%d maps: win %d, lose %d, protect %d target parameters", len(x.Maps), arms[armWin], arms[armLose], arms[armProtect])
	if arms[armWin] == 0 || arms[armLose] == 0 || arms[armProtect] == 0 {
		t.Fatalf("an arm kind is missing: %v", arms)
	}
}

// hasArmRow reports whether rows hold a row of arm that edits an existing node.
func hasArmRow(rows []goalRow, arm string) bool {
	for _, g := range rows {
		if g.arm == arm && g.node >= 0 {
			return true
		}
	}
	return false
}
