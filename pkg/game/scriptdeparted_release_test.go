package game

import (
	"bytes"
	"slices"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// Mission 141 is won by the dragon's death after its flier body has left the
// world, across a SAV round trip taken between the removal and the next pass.

const (
	dragonMission     = 141
	dragonMapUnit     = 106
	dragonTooth       = 3621
	dragonReward      = 1500000
	dragonMageX       = 35
	dragonMageY       = 68
	dragonTicksCap    = 4000
	dragonScriptCycle = 16
)

func dragonStep(t *testing.T, f *FrontEnd, app *ui.App, seen *[]ui.NoticeKind) {
	t.Helper()
	if _, kind, up := f.LiveNotice(); up {
		*seen = append(*seen, kind)
		if kind == ui.NoticeSuccess {
			return
		}
		if err := app.HeadlessActivate("notice"); err != nil {
			t.Fatal(err)
		}
	}
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
}

// dragonFell deals the diagnostic blow to -1 health; the fall, dwell,
// teardown and decay after it are the ordinary ticks'.
func dragonFell(t *testing.T, f *FrontEnd, id sim.EntityID) {
	t.Helper()
	for _, e := range f.live.world.Entities() {
		if e.ID == id {
			f.LiveDamage(uint32(id), e.HP+1)
			return
		}
	}
	t.Fatalf("entity %d is not in the world", id)
}

func dragonOpen(t *testing.T) (*FrontEnd, *ui.App) {
	t.Helper()
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	f.Carried = f.ChargenParty(ui.ChargenResult{Name: "Dragon witness", Choices: []int{1, 0, 3}, Stats: []int{31, 27, 24, 29}})
	f.arriveInTown()
	for _, n := range f.Campaign.Value().Main {
		if n < dragonMission/10*10 {
			f.Town.Won(n)
		}
	}
	taken := false
	for _, building := range []TownBuilding{TownTavern, TownShop, TownSchool} {
		for _, offer := range f.Town.Offers(building) {
			if offer.Mission == dragonMission && !taken {
				if got, ok := f.Town.Take(building, offer.Index); !ok || got != dragonMission {
					t.Fatal("mission offer refused", got, ok)
				}
				f.TownScreen().(*townScreen).markWorldSelected(dragonMission)
				taken = true
			}
		}
	}
	if !taken {
		t.Fatalf("mission %d has no offer", dragonMission)
	}
	app := f.App("dragon death")
	if err := app.OpenMission(f.MissionOpenerWith(dragonMission, f.NextParty())); err != nil {
		t.Fatal(err)
	}
	return f, app
}

// dragonReload writes a SAV through the save dialog, loads it on a cold front
// end through the main-menu LOAD, and requires the World bytes unchanged.
func dragonReload(t *testing.T, f *FrontEnd) (*FrontEnd, *ui.App) {
	t.Helper()
	store := SaveStore{Dir: t.TempDir()}
	before, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := f.SaveDialogSeams(store, OriginalStore{}).Prepare(ui.SaveRequest{
		Directory: store.Dir, Name: "Dragon checkpoint", Format: ui.SaveSAV, OnMap: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if paths, err := prepared.Commit(false); err != nil || len(paths) != 1 {
		t.Fatal("publish checkpoint", paths, err)
	}
	cold := releaseFront(t)
	cold.SetDeterministicFrames(true)
	app := cold.App("dragon cold LOAD")
	cold.ConfigureSaveSeams(app, store, OriginalStore{}, nil)
	if err := app.HeadlessActivate("load game"); err != nil {
		t.Fatal(err)
	}
	rows := app.HeadlessRows()
	if len(rows) != 1 {
		t.Fatal("cold LOAD rows", rows)
	}
	if err := app.HeadlessActivate(rows[0].Text); err != nil || app.Screen() == ui.ScreenLoad {
		t.Fatal("cold LOAD refused the SAV", err, app.HeadlessMessage())
	}
	written, _, err := cold.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before.World, written.World) {
		var left, right sim.World
		if left.UnmarshalBinary(before.World) == nil && right.UnmarshalBinary(written.World) == nil {
			currentMenuWorldDiagnostics(t, &left, &right)
		}
		t.Fatal("the loaded World differs from the saved one")
	}
	return cold, app
}

func TestReleaseMissionDragonDeathWinsAfterItsBodyLeaves(t *testing.T) {
	f, app := dragonOpen(t)
	refs := mapload.ScriptUnits(f.live.mission.state.Map, f.live.mission.party)
	unit := func(mapUnit uint16) sim.EntityID {
		id, ok := refs[mapUnit]
		if !ok {
			t.Fatalf("mission %d binds no map unit %d", dragonMission, mapUnit)
		}
		return id
	}
	dragon, hero := unit(dragonMapUnit), f.live.mission.ids[0]
	var seen []ui.NoticeKind
	// The hero starts beside the Mage; T4 must wait for his walk back.
	f.live.pending = append(f.live.pending, sim.MoveTo(hero, sim.CellPoint{X: dragonMageX + 8, Y: dragonMageY - 4}))
	for _, supplier := range []uint16{75, 57, 30} {
		dragonFell(t, f, unit(supplier))
	}
	onMap := func() bool {
		for _, e := range f.live.world.Entities() {
			if e.ID == dragon {
				return !e.OffMap
			}
		}
		return false
	}
	for i := 0; i < dragonTicksCap && !(f.live.world.ScriptLatched(5) && onMap()); i++ {
		dragonStep(t, f, app, &seen)
	}
	if !onMap() {
		t.Fatal("the suppliers' deaths never returned the dragon to the map")
	}
	dragonFell(t, f, dragon)
	for i := 0; i < dragonTicksCap && onMap(); i++ {
		dragonStep(t, f, app, &seen)
	}
	if slices.ContainsFunc(f.live.world.Entities(), func(e sim.Entity) bool { return e.ID == dragon }) {
		t.Fatal("the dragon's body never left the world")
	}
	if f.live.world.ScriptLatched(0) {
		t.Fatal("T0 latched before the SAV round trip")
	}
	f, app = dragonReload(t, f)
	seen = seen[:0]
	w := f.live.world
	for i := 0; i < dragonTicksCap && !w.ScriptLatched(0); i++ {
		dragonStep(t, f, app, &seen)
	}
	holds := func() bool {
		carried, _ := w.Carried(hero)
		return slices.Contains(carried, dragonTooth)
	}
	for i := 0; i < 2*dragonScriptCycle && w.ScriptLatched(0) && !(holds() && slices.Contains(seen, ui.NoticeDialogue)); i++ {
		dragonStep(t, f, app, &seen)
	}
	if w.ScriptLatched(4) {
		t.Fatal("T4 latched before the hero walked back to the Mage")
	}
	if !w.ScriptLatched(0) || !slices.Contains(seen, ui.NoticeDialogue) || !holds() {
		t.Fatalf("after LOAD: T0 %v, notices %v, tooth held %v", w.ScriptLatched(0), seen, holds())
	}
	purse := w.Purse(sim.SelfSlot)
	f.live.pending = append(f.live.pending, sim.MoveTo(hero, sim.CellPoint{X: dragonMageX + 1, Y: dragonMageY}))
	for i := 0; i < dragonTicksCap && w.Outcome() == sim.OutcomeUndecided; i++ {
		dragonStep(t, f, app, &seen)
	}
	if !w.ScriptLatched(4) || w.Outcome() != sim.OutcomeWon {
		t.Fatalf("T4 %v, outcome %v; want the mission won", w.ScriptLatched(4), w.Outcome())
	}
	if holds() {
		t.Error("T4 won without taking the tooth")
	}
	if got := w.Purse(sim.SelfSlot) - purse; got != dragonReward {
		t.Errorf("T4 paid %d, want %d", got, dragonReward)
	}
}
