package game

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestReleaseQuickSpellsCustomOriginalTownSAVRoundTrip(t *testing.T) {
	f := releaseFront(t)
	path, source := groundCorpusFile(t, "2026-08-15/game0010.sav", "89cfca4c14e2b0bafd1fe28911badf246e213d83398874699947008739e8c5d4")
	store := SaveStore{Dir: t.TempDir()}
	a := f.App("1119 original-town SAV witness")
	original := OriginalStore{Dir: filepath.Dir(path)}
	f.ConfigureSaveSeams(a, store, original, nil)
	if err := headlessOpenLoad(a); err != nil {
		t.Fatal(err)
	}
	label := ""
	for _, entry := range original.List() {
		if entry.Name == filepath.Base(path) {
			label = entry.Label
		}
	}
	if label == "" {
		t.Fatal("source city missing from load menu")
	}
	if err := a.HeadlessActivate(label); err != nil || a.Screen() != ui.ScreenTown {
		t.Fatalf("town load screen%s err%v", a.Screen(), err)
	}
	// ID 65535 is a current custom binding with no cell in the original
	// 24-entry map: the legacy field projects it unbound, the current SAV
	// extension retains it. Againrom policy, not a ROM1 claim.
	want := [4]uint32{65535, 1, 6, 19}
	wantLegacy := [4]uint32{0, 1, 6, 19}
	f.quickSpells = want
	ordinarySave := func(app *ui.App, name string) {
		t.Helper()
		if err := app.HeadlessKey("escape"); err != nil {
			t.Fatal(err)
		}
		if err := app.HeadlessGameMenuAction("save"); err != nil {
			t.Fatal(err)
		}
		if app.Screen() != ui.ScreenSave {
			t.Fatalf("ordinary SAVE did not open its dialog: screen %s", app.Screen())
		}
		if err := app.HeadlessSaveEdit(store.Dir, name, ui.SaveSAV); err != nil {
			t.Fatal(err)
		}
		if err := app.HeadlessSaveAction("save"); err != nil {
			t.Fatal(err)
		}
	}
	assertSAV := func(name string) {
		t.Helper()
		raw, err := store.Read(name)
		if err != nil {
			t.Fatal(err)
		}
		legacy, err := originalQuickSpellsMustRead(raw)
		if err != nil || legacy != wantLegacy {
			t.Fatalf("legacy shortcut projection = %v, want %v (err %v)", legacy, wantLegacy, err)
		}
		doc, err := sav.DecodeDocumentData(raw)
		if err != nil {
			t.Fatal(err)
		}
		actions, err := readCurrentActions(&doc)
		if err != nil || actions == nil || actions.Session == nil || len(actions.Session.QuickSpells) != 1 || actions.Session.QuickSpells[0] != (currentQuickSpell{Slot: 0, ID: 65535}) {
			t.Fatalf("current SAV custom shortcut extension = %+v (err %v)", actions, err)
		}
	}
	ordinarySave(a, "custom shortcuts")
	rows, err := store.List()
	if err != nil || len(rows) != 1 || filepath.Ext(rows[0].Name) != ".sav" {
		t.Fatalf("ordinary SAVE rows%v err%v", rows, err)
	}
	assertSAV(rows[0].Name)

	g := releaseFront(t)
	b := g.App("1119 original-town cold SAV LOAD")
	g.ConfigureSaveSeams(b, store, OriginalStore{}, nil)
	if err := b.HeadlessKey("load"); err != nil {
		t.Fatal(err)
	}
	if err := b.HeadlessKey("enter"); err != nil {
		t.Fatal(err)
	}
	if b.Screen() != ui.ScreenTown || g.quickSpells != want || g.originalCity == nil {
		t.Fatalf("fresh source-backed SAV LOAD screen%s slots%v provenance%v", b.Screen(), g.quickSpells, g.originalCity != nil)
	}
	ordinarySave(b, "custom shortcuts again")
	rows, err = store.List()
	if err != nil || len(rows) != 2 {
		t.Fatalf("second SAVE rows%v err%v", rows, err)
	}
	for _, row := range rows {
		if filepath.Ext(row.Name) != ".sav" {
			t.Fatalf("ordinary SAVE wrote non-SAV row: %s", row.Name)
		}
		assertSAV(row.Name)
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(source, after) {
		t.Fatal("source install/corpus bytes changed")
	}
	t.Logf("ordinary SAV stores representable shortcuts%v and retains custom ID 65535 in the current session extension; cold LOAD and next SAVE preserve all four; source bytes unchanged", want)
}

// Installed mission/spells/art and real App key/target/menu/store paths. The
// single mage's known spells and mana are controlled input, not a claim about
// campaign acquisition or an original-game populated-slot runtime witness.
func TestReleaseQuickSpellsSaveFreshLoadAndCast(t *testing.T) {
	t.Run("paused-all-unavailable-book-viewport", quickSpellsPausedViewportWitness)
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	hero := data.Hero{Body: 60, Reaction: 60, Mind: 100, Spirit: 100}
	hero.Skill[1] = 100
	party := []mapload.PartyMember{{ID: "hero", PlayerCharacter: true, StartingHero: true, Mage: true, Class: 0x18,
		Profile: data.Profile{HealthColumn: true, ManaColumn: true}, Hero: hero,
		KnownSpells: 1<<1 | 1<<16 | 1<<6 | 1<<19,
		Saved:       &mapload.Saved{Cell: mapload.Cell{X: 29, Y: 50}, HP: 100, MaxHP: 100, Mana: 1000, MaxMana: 1000, HealthRegenPeriod: 100, ManaRegenPeriod: 50}}}
	store := SaveStore{Dir: t.TempDir()}
	a := f.App("1119 quick spell witness")
	a.Layout(1024, 768)
	f.ConfigureSaveSeams(a, store, OriginalStore{}, nil)
	if err := a.OpenMission(f.MissionOpenerWith(10, party)); err != nil {
		t.Fatal(err)
	}
	missionAutosaveRow(t, f, store, 10)
	key := func(a *ui.App, key string) {
		t.Helper()
		if err := a.HeadlessKey(key); err != nil {
			t.Fatal(err)
		}
	}
	key(a, "0")
	id := f.live.mission.ids[0]
	inspectionCentre(f.live, 29, 50)
	if err := a.HeadlessSelectEntity(uint32(id)); err != nil {
		t.Fatal(err)
	}
	pausedHash, pausedTick := f.live.world.Hash(), f.live.world.Tick()
	if err := a.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	if f.live.world.Hash() != pausedHash || f.live.world.Tick() != pausedTick {
		t.Fatal("paused spellbook refresh advanced simulation")
	}
	if _, _, err := a.HeadlessSpellPoint(16); err != nil {
		key(a, "book")
	}
	// An installed but unlearned cell remains bindable. Its populated shortcut
	// changes current in either visibility state, without Cast or any order.
	ux, uy, err := a.HeadlessSpellPoint(2)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessPointer("hover", ux, uy); err != nil {
		t.Fatal(err)
	}
	key(a, "ctrl-f8")
	key(a, "f8")
	for _, visibility := range []string{"open", "closed"} {
		if visibility == "closed" {
			key(a, "book")
			key(a, "f8")
		}
		if _, current, armed := f.live.view.QuickSpellState(); current != 2 || armed || len(f.live.pending) != 0 {
			t.Fatalf("%s unavailable changed more than current: current%d armed%v pending%v", visibility, current, armed, f.live.pending)
		}
	}
	gx, gy, err := a.HeadlessGroundPoint()
	if err != nil {
		t.Fatal(err)
	}
	for _, edge := range []string{"right-press", "right-release"} {
		if err := a.HeadlessPointer(edge, gx, gy); err != nil {
			t.Fatal(err)
		}
	}
	key(a, "book")
	want := [4]uint32{16, 1, 6, 19}
	for slot, spell := range want {
		x, y, err := a.HeadlessSpellPoint(spell)
		if err != nil {
			t.Fatal(err)
		}
		if err := a.HeadlessPointer("hover", x, y); err != nil {
			t.Fatal(err)
		}
		key(a, "ctrl-f"+string(byte('5'+slot)))
	}
	if got, current, armed := f.live.view.QuickSpellState(); got != want || current != 0 || armed {
		t.Fatalf("assignment=%v current%d armed%v", got, current, armed)
	}
	key(a, "f5")
	if _, current, armed := f.live.view.QuickSpellState(); current != 16 || armed {
		t.Fatal("open-book invoke must select only")
	}
	key(a, "book")
	key(a, "f5")
	if _, current, armed := f.live.view.QuickSpellState(); current != 16 || !armed {
		e, _ := f.live.entity(id)
		t.Fatalf("closed F5 current%d armed%v type%d known%#x mana%d", current, armed, e.TypeID, e.KnownSpells, e.MaxMana)
	}
	queueCast := func(a *ui.App, f *FrontEnd) {
		t.Helper()
		before := len(f.live.pending)
		x, y, err := a.HeadlessEntityPoint(uint32(id))
		if err != nil {
			t.Fatal(err)
		}
		for _, edge := range []string{"press", "release"} {
			if err := a.HeadlessPointer(edge, x, y); err != nil {
				t.Fatal(err)
			}
		}
		if got := f.live.pending; len(got) != before+1 || got[len(got)-1].Kind != sim.KindCast || got[len(got)-1].Y != 16 || got[len(got)-1].X != int32(id) {
			t.Fatalf("quick targeting queued %+v", got)
		}
	}
	queueCast(a, f)
	before, _ := f.live.entity(id)
	spent := false
	for i := 0; i < 200; i++ {
		f.live.tick()
		after, _ := f.live.entity(id)
		if after.Mana < before.Mana {
			spent = true
			break
		}
	}
	if !spent {
		t.Fatalf("targeted quick spell never spent mana: refusal=%s", f.live.world.BookSpellRefusal(id, id, 16))
	}
	for i := 0; a.HeadlessNoticeOpen() && i < 16; i++ {
		key(a, "enter")
	}
	key(a, "escape")
	if err := a.HeadlessGameMenuAction("save"); err != nil {
		t.Fatal(err)
	}
	if a.Screen() != ui.ScreenSave {
		t.Fatalf("ordinary SAVE did not open its dialog: screen %s notice %v", a.Screen(), a.HeadlessNoticeOpen())
	}
	if err := a.HeadlessSaveEdit(store.Dir, "Quick spells", ui.SaveSAV); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessSaveAction("save"); err != nil {
		t.Fatal(err)
	}
	rows, err := store.List()
	var manual SaveEntry
	for _, row := range rows {
		if row.Name == "Quick spells.sav" {
			manual = row
		}
	}
	if err != nil || len(rows) != 2 || !IsOriginal(manual.Name) {
		t.Fatalf("menu SAVE rows%v err%v", rows, err)
	}

	g := releaseFront(t)
	g.SetDeterministicFrames(true)
	b := g.App("1119 fresh load witness")
	b.Layout(1024, 768)
	save, list, load := g.SaveSeams(store, OriginalStore{}, nil)
	b.SetSaveSeams(save, list, load)
	groundAppLoad(t, b, list, manual.Name)
	if b.Screen() != ui.ScreenMap || g.quickSpells != want {
		t.Fatalf("fresh LOAD screen%s slots%v", b.Screen(), g.quickSpells)
	}
	if _, current, armed := g.live.view.QuickSpellState(); current != 16 || !armed {
		t.Fatalf("LOAD lost the saved cast cursor: current %d armed %v", current, armed)
	}
	releasePauseMission(t, g, b)
	inspectionCentre(g.live, 29, 50)
	if err := b.HeadlessSelectEntity(uint32(id)); err != nil {
		t.Fatal(err)
	}
	if err := b.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	if _, _, err := b.HeadlessSpellPoint(16); err == nil {
		key(b, "book")
	}
	key(b, "f5")
	queueCast(b, g)
	if g.quickSpells != want {
		t.Fatal("next action changed bindings")
	}
	completed := g.live
	g.FinishMission(10, completed.mission.party, completed.world, completed.mission.ids)
	g.arriveInTown()
	if g.quickSpells != want {
		t.Fatal("production mission finish/town arrival lost bindings")
	}
	if err := b.OpenMission(g.MissionOpener(20)); err != nil {
		t.Fatal(err)
	}
	if slots, current, armed := g.live.view.QuickSpellState(); slots != want || current != 0 || armed {
		t.Fatal("next mission viewer did not retain only session bindings")
	}
	t.Logf("App hover+Ctrl-F5..F8=%v; open selects, closed arms; targeted cast spent mana; menu SAVE/fresh LOAD retained all four; next F5 targeted spell16", want)
}

// pausedBookBarPx is the map viewport an open, all-unavailable spell book gives
// up to MAGIC-ICON-024's 85-pixel atlas, flush against the inventory below.
const pausedBookBarPx = 85

// The old 1087 fixture implicitly depended on a stale book owner while paused.
// Refreshing the selected fighter's all-unavailable book must realize the open
// panel, not move the camera or advance the world to rescue an obscured target.
func quickSpellsPausedViewportWitness(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	a := f.App("1119 paused book viewport")
	s, err := ReadHeadlessScenario("../../scenarios/1087-player-defend.json")
	if err != nil {
		t.Fatal(err)
	}
	captures := make(map[string]HeadlessState)
	for _, step := range s.Steps[:5] {
		if err := runHeadlessStep(f, a, step, captures); err != nil {
			t.Fatal(err)
		}
	}
	v := f.live.view
	id := f.live.mission.ids[0]
	e, _ := f.live.entity(id)
	if e.KnownSpells != 0 || !f.live.stopped || v.Camera().ViewH != 593 {
		t.Fatalf("fixture: known%#x stopped%v height%d", e.KnownSpells, f.live.stopped, v.Camera().ViewH)
	}
	// Put the hero under the band the open book reserves: an obscured target
	// stays obscured, and opening the book neither moves the camera nor
	// advances the world.
	// Close the book to place the hero in its band, then open it.
	if err := a.HeadlessKey("book"); err != nil {
		t.Fatal(err)
	}
	v.Camera().Pan(0, -pausedBookBarPx)
	if _, _, err := a.HeadlessEntityPoint(uint32(id)); err != nil {
		t.Fatalf("fixture: hero must be pickable before the book opens: %v", err)
	}
	hash, tick := f.live.world.Hash(), f.live.world.Tick()
	x, y := v.Camera().X, v.Camera().Y
	if err := a.HeadlessKey("book"); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	if v.Camera().ViewH != 593 {
		t.Fatalf("paused all-unavailable book did not reserve its bar: %d", v.Camera().ViewH)
	}
	if _, _, err := a.HeadlessSpellPoint(1); err != nil {
		t.Fatalf("all-unavailable catalog is not present: %v", err)
	}
	if _, py, err := a.HeadlessEntityPoint(uint32(id)); err == nil && py < v.ViewportSize().Y {
		t.Fatal("fixture hero should be behind the open book")
	}
	if err := a.HeadlessKey("book"); err != nil {
		t.Fatal(err)
	}
	if v.Camera().ViewH != 678 {
		t.Fatalf("closing book did not restore viewport: %d", v.Camera().ViewH)
	}
	if err := a.HeadlessSelectEntity(uint32(id)); err != nil {
		t.Fatalf("ordinary close did not restore hero hit: %v", err)
	}
	if v.Camera().X != x || v.Camera().Y != y || f.live.world.Hash() != hash || f.live.world.Tick() != tick {
		t.Fatal("paused book visibility changed camera origin or simulation")
	}
	t.Log("paused all-unavailable book reserves97px; ordinary close restores the hero hit; camera origin, tick and hash unchanged")
}
