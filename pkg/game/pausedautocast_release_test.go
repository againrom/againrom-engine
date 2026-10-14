package game

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// pausedAutocastApp opens the second-mage mission paused, with the mage's book
// on screen, and returns the spell whose cell the test toggles.
func pausedAutocastApp(t *testing.T) (*FrontEnd, *ui.App, sim.EntityID, uint32) {
	t.Helper()
	f, a := spellbookMissionApp(t)
	live := f.live
	if !live.stopped {
		if err := a.HeadlessKey("0"); err != nil {
			t.Fatal(err)
		}
	}
	if !live.stopped {
		t.Fatal("fixture: the mission did not pause")
	}
	const mage = sim.EntityID(0)
	for i := range live.fog.visible {
		live.fog.visible[i], live.fog.explored[i] = 1, 1
	}
	spellbookShow(t, a, live, mage)
	caster, _ := live.entity(mage)
	for id := uint32(1); id < 32; id++ {
		if caster.KnownSpells&(1<<id) == 0 {
			continue
		}
		if _, _, err := a.HeadlessSpellPoint(id); err == nil {
			return f, a, mage, id
		}
	}
	t.Fatal("fixture: no visible book cell")
	return nil, nil, 0, 0
}

func pausedAutocastFrame(t *testing.T, a *ui.App, spell uint32) bool {
	t.Helper()
	if err := a.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	on, err := a.HeadlessSpellAutocast(spell)
	if err != nil {
		t.Fatal(err)
	}
	return on
}

func pausedAutocastRightClick(t *testing.T, a *ui.App, spell uint32) {
	t.Helper()
	x, y, err := a.HeadlessSpellPoint(spell)
	if err != nil {
		t.Fatal(err)
	}
	for _, edge := range []string{"right-press", "right-release"} {
		if err := a.HeadlessPointer(edge, x, y); err != nil {
			t.Fatal(err)
		}
	}
}

func TestReleasePausedAutocastShowsAtOnceAndHoldsAfterResume(t *testing.T) {
	f, a, mage, spell := pausedAutocastApp(t)
	live := f.live
	before, _ := live.entity(mage)
	if before.AutoSpell != 0 || pausedAutocastFrame(t, a, spell) {
		t.Fatal("fixture: the mage already autocasts")
	}
	hash := live.world.Hash()
	// The right-click: on, off, on again, each shown on the next frame.
	for i, want := range []bool{true, false, true} {
		pausedAutocastRightClick(t, a, spell)
		if got := pausedAutocastFrame(t, a, spell); got != want {
			t.Fatalf("right-click %d: the book shows autocast %v, want %v", i+1, got, want)
		}
	}
	// The key: a selected spell, then Ctrl+A clears and sets again.
	x, y, err := a.HeadlessSpellPoint(spell)
	if err != nil {
		t.Fatal(err)
	}
	for _, edge := range []string{"press", "release"} {
		if err := a.HeadlessPointer(edge, x, y); err != nil {
			t.Fatal(err)
		}
	}
	for i, want := range []bool{false, true, false, true} {
		if err := a.HeadlessKey("autocast"); err != nil {
			t.Fatal(err)
		}
		if got := pausedAutocastFrame(t, a, spell); got != want {
			t.Fatalf("Ctrl+A %d: the book shows autocast %v, want %v", i+1, got, want)
		}
	}
	if live.world.Hash() != hash || !live.stopped {
		t.Fatal("the paused changes stepped or altered the World")
	}
	if now, _ := live.entity(mage); now.AutoSpell != 0 {
		t.Fatal("the paused changes wrote the World before the next tick")
	}
	// A SAVE while paused carries the setting, and a LOAD shows it at once.
	s, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(s, "Paused autocast")
	if err != nil {
		t.Fatal(err)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	actions, err := readCurrentActions(&doc)
	if err != nil || actions == nil || actions.Pending == nil {
		t.Fatal("the SAV carries no pending queue", err)
	}
	carried := false
	for _, c := range actions.Pending.Commands {
		if c.Command.Kind == sim.KindAutocast && c.Command.X == int32(spell) {
			carried = true
		}
	}
	if !carried {
		t.Fatalf("a SAV made while paused lost the autocast change: %+v", actions.Pending.Commands)
	}
	lf, la := pausedAutocastLoad(t, raw, mage, spell)
	pausedAutocastResume(t, f, a, mage, spell)
	pausedAutocastResume(t, lf, la, mage, spell)
}

// pausedAutocastResume resumes a paused game and requires the first ticks to
// have applied the setting the frame showed.
func pausedAutocastResume(t *testing.T, f *FrontEnd, a *ui.App, mage sim.EntityID, spell uint32) {
	t.Helper()
	live := f.live
	if err := a.HeadlessKey("0"); err != nil {
		t.Fatal(err)
	}
	for range 4 {
		if err := a.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	if live.world.Tick() == 0 {
		t.Fatal("the game did not resume")
	}
	after, _ := live.entity(mage)
	if after.AutoSpell != uint16(spell) || !pausedAutocastFrame(t, a, spell) || len(live.pending) != 0 {
		t.Fatalf("after resume the mage autocasts %d, want %d; queue %v", after.AutoSpell, spell, live.pending)
	}
}

func pausedAutocastLoad(t *testing.T, raw []byte, mage sim.EntityID, spell uint32) (*FrontEnd, *ui.App) {
	t.Helper()
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	f.Options = OptionsStore{Path: filepath.Join(t.TempDir(), "options.txt")}
	store := SaveStore{Dir: t.TempDir()}
	if err := os.WriteFile(filepath.Join(store.Dir, "paused.sav"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	app := f.App("paused autocast LOAD")
	t.Cleanup(app.StopAudio)
	app.Layout(1024, 768)
	f.ConfigureSaveSeams(app, store, OriginalStore{}, func() time.Time { return time.Unix(100, 0) })
	if err := app.HeadlessKey("load"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessActivate("@first"); err != nil || app.Screen() != ui.ScreenMap {
		t.Fatal("LOAD of the paused SAV", err, app.Screen())
	}
	live := f.live
	if !live.stopped {
		t.Fatal("the loaded game is not paused")
	}
	for i := range live.fog.visible {
		live.fog.visible[i], live.fog.explored[i] = 1, 1
	}
	spellbookShow(t, app, live, mage)
	if e, _ := live.entity(mage); e.AutoSpell != 0 {
		t.Fatal("the LOAD wrote the pending setting into the World")
	}
	if !pausedAutocastFrame(t, app, spell) {
		t.Fatal("after LOAD the book does not show the saved autocast setting")
	}
	return f, app
}

// TestReleaseSecondGamePausedAutocastShowsAtOnce is the same rule in the
// second game's mission screen, which builds the same book and sink.
func TestReleaseSecondGamePausedAutocastShowsAtOnce(t *testing.T) {
	f, app := secondCheatMission(t, true, false)
	live := f.live
	if !live.stopped {
		t.Fatal("fixture: the mission is not paused")
	}
	secondCheatUnlock(t, f, app)
	secondCheatLines(t, f, app, "#modify self +spell 6")
	live.stopped = true
	caster := releaseEntity(t, live, live.mission.ids[0])
	if caster.KnownSpells == 0 {
		t.Fatal("fixture: the mage hero knows no spell")
	}
	for i := range live.fog.visible {
		live.fog.visible[i], live.fog.explored[i] = 1, 1
	}
	spellbookShow(t, app, live, caster.ID)
	spell := uint32(0)
	for id := uint32(1); id < 32 && spell == 0; id++ {
		if _, _, err := app.HeadlessSpellPoint(id); err == nil && caster.KnownSpells&(1<<id) != 0 {
			spell = id
		}
	}
	if spell == 0 {
		t.Fatal("fixture: no visible book cell")
	}
	if pausedAutocastFrame(t, app, spell) {
		t.Fatal("fixture: the caster already autocasts")
	}
	for i, want := range []bool{true, false, true} {
		pausedAutocastRightClick(t, app, spell)
		if got := pausedAutocastFrame(t, app, spell); got != want {
			t.Fatalf("right-click %d: the book shows autocast %v, want %v", i+1, got, want)
		}
	}
	live.stopped = false
	for range 4 {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	after := releaseEntity(t, live, caster.ID)
	if live.world.Tick() == 0 || after.AutoSpell != uint16(spell) || len(live.pending) != 0 {
		t.Fatalf("after resume tick %d, autocast %d, want %d; queue %v", live.world.Tick(), after.AutoSpell, spell, live.pending)
	}
}
