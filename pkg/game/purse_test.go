package game

import (
	"image"
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestEnqueueGoldDropReservesThePreviewAndClampsToIt(t *testing.T) {
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: 7, X: 3, Y: 3, Owner: sim.SelfSlot}}, nil, sim.Relations{}, nil,
		[]sim.Stock{{ID: 7, Items: []uint16{packCode1}}})
	if err != nil {
		t.Fatal(err)
	}
	mw := grabWorld(t, w, 7, missionSource{})
	w.SetPurse(sim.SelfSlot, 2500)
	mw.refreshPack()
	if inventoryGold(mw.invSubject) != 2500 {
		t.Fatalf("shown purse %d, want 2500", inventoryGold(mw.invSubject))
	}

	mw.enqueueGoldDrop(ui.GoldDrop{Amount: 700, AtSubject: true})
	want := sim.DropGold(sim.SelfSlot, 700, sim.CellPoint{X: 3, Y: 3})
	if len(mw.pending) != 1 || mw.pending[0] != want {
		t.Fatalf("pending %+v, want %+v", mw.pending, want)
	}
	mw.refreshPack()
	if w.Purse(sim.SelfSlot) != 2500 || inventoryGold(mw.invSubject) != 1800 {
		t.Fatalf("server purse %d shown %d, want 2500 and 1800", w.Purse(sim.SelfSlot), inventoryGold(mw.invSubject))
	}

	mw.enqueueGoldDrop(ui.GoldDrop{Amount: 1 << 31, X: 4, Y: 4})
	if len(mw.pending) != 2 || mw.pending[1] != sim.DropGold(sim.SelfSlot, 1800, sim.CellPoint{X: 4, Y: 4}) {
		t.Fatalf("an oversized request was not clamped to the previewed 1800: %+v", mw.pending)
	}
	mw.enqueueGoldDrop(ui.GoldDrop{Amount: 5, AtSubject: true})
	if len(mw.pending) != 2 {
		t.Fatalf("a request against an empty preview queued %+v", mw.pending)
	}
	mw.refreshPack()
	if inventoryGold(mw.invSubject) != 0 {
		t.Fatalf("shown purse %d with every coin reserved", inventoryGold(mw.invSubject))
	}
	if w.Purse(sim.SelfSlot) != 2500 || len(w.Sacks()) != 0 {
		t.Fatal("queued requests changed the simulation before a tick")
	}

	mw.tick()
	if w.Purse(sim.SelfSlot) != 0 || len(mw.pending) != 0 {
		t.Fatalf("after the tick purse %d pending %d", w.Purse(sim.SelfSlot), len(mw.pending))
	}
	gold := uint32(0)
	for _, s := range w.Sacks() {
		gold += s.Gold
	}
	if gold != 2500 {
		t.Fatalf("ground gold %d, want 2500 (%+v)", gold, w.Sacks())
	}
}

func TestEnqueueGoldDropIgnoresASubjectWithoutAPurse(t *testing.T) {
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: 7, X: 3, Y: 3, Owner: sim.SelfSlot}}, nil, sim.Relations{}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	mw := grabWorld(t, w, 7, missionSource{})
	w.SetPurse(sim.SelfSlot, 100)
	mw.invParty.primary = false
	mw.enqueueGoldDrop(ui.GoldDrop{Amount: 50, AtSubject: true})
	mw.invParty.primary, mw.invSubjectSet = true, false
	mw.enqueueGoldDrop(ui.GoldDrop{Amount: 50, AtSubject: true})
	if len(mw.pending) != 0 {
		t.Fatalf("pending %+v", mw.pending)
	}
}

// The pending queue admits the gold command and keeps its full-width amount,
// its roster slot and its cell through the current SAV projection, without an
// actor binding and without losing the endpoint filter's other commands.
func TestPendingGoldCommandSurvivesValidationAndProjection(t *testing.T) {
	drop := sim.DropGold(sim.SelfSlot, 4000000000, sim.CellPoint{X: 12, Y: 22})
	q := &SnapshotPendingQueue{Commands: []sim.Command{drop}, Ignored: []bool{false}}
	if err := validateSnapshotPending(q); err != nil {
		t.Fatal(err)
	}
	cur, err := projectCurrentPending(SnapshotResidue{PendingQueue: q}, map[sim.EntityID]uint16{}, map[sim.EntityID]uint16{})
	if err != nil {
		t.Fatal(err)
	}
	if len(cur.Commands) != 1 || cur.Commands[0].Issuer != nil || cur.Commands[0].Target != nil || cur.Commands[0].Command != drop {
		t.Fatalf("projection %+v", cur.Commands)
	}
	bound := *cur
	bound.Commands = []currentPendingCommand{{Command: drop, Issuer: &currentActionBinding{ID: 1}}}
	if validateCurrentPending(&bound) == nil {
		t.Fatal("an actor binding on a player command was accepted")
	}
	q.Commands[0].Kind = 0x24
	if validateSnapshotPending(q) == nil {
		t.Fatal("an unknown kind was accepted")
	}
}

func purseSavedMoney(t *testing.T, doc sav.DocumentData, slot uint32) uint32 {
	t.Helper()
	var money uint32
	found := 0
	for _, record := range doc.Objects {
		if record.Class == "Player" && actorProjectionValue(t, record, "Slot") == slot {
			money, found = actorProjectionValue(t, record, "Money"), found+1
		}
	}
	if found != 1 {
		t.Fatalf("%d Player records for slot %d", found, slot)
	}
	return money
}

func purseSavedSacks(doc sav.DocumentData, t *testing.T) (gold, value []uint32) {
	t.Helper()
	for _, record := range doc.Objects {
		if record.Class == "Sack" {
			gold = append(gold, actorProjectionValue(t, record, "S3C"))
			value = append(value, actorProjectionValue(t, record, "T1C"))
		}
	}
	return gold, value
}

func purseFixture(t *testing.T) (*FrontEnd, *ui.App) {
	t.Helper()
	f := currentPoolFixtureFront(t, 91, 92)
	f.SetDeterministicFrames(true)
	app := f.App("purse gold")
	if err := app.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	app.Layout(1024, 768)
	if !f.live.invParty.primary {
		t.Fatal("setup: the mission subject has no purse")
	}
	// The synthetic map admits no party cell, so the roster actor is placed on
	// the map in a rebuilt world.
	actors := f.live.world.Entities()
	for i := range actors {
		if actors[i].Owner == sim.SelfSlot {
			actors[i].X, actors[i].Y, actors[i].OffMap = 10, 10, false
		}
	}
	rebuilt, err := sim.NewWorld(1, f.live.world.Bounds(), sim.ModeCanonical, nil, actors)
	if err != nil {
		t.Fatal(err)
	}
	f.live.world, f.live.mission.state.World = rebuilt, rebuilt
	f.live.world.SetPurse(sim.SelfSlot, 2500)
	for i := range f.live.fog.visible {
		f.live.fog.visible[i], f.live.fog.explored[i] = 1, 1
	}
	f.live.push()
	subject, _ := f.live.entity(f.live.mission.ids[0])
	f.live.view.Camera().CenterOn(float64(subject.X*32), float64(subject.Y*32))
	if err := app.HeadlessSelectEntity(uint32(f.live.mission.ids[0])); err != nil {
		t.Fatal(err)
	}
	if !f.live.view.SaveApplication().InventoryOpen {
		if err := app.HeadlessKey("i"); err != nil {
			t.Fatal(err)
		}
	}
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	return f, app
}

func purseCell(t *testing.T, f *FrontEnd) int {
	t.Helper()
	for i, p := range f.live.invSubject.PackPurse {
		if p {
			return i
		}
	}
	t.Fatalf("the pack shows no purse: %+v", f.live.invSubject.PackPurse)
	return 0
}

func purseGround(t *testing.T, w *sim.World) (gold uint32, n int) {
	t.Helper()
	for _, s := range w.Sacks() {
		gold, n = gold+s.Gold, n+1
	}
	return gold, n
}

// The whole route, in the order the player meets it: the purse cell is dragged
// to the ground while the game is actively paused, the request waits in the
// pending queue with the server purse untouched, a SAVE keeps exactly that
// state, a cold LOAD restores it, the first resumed tick drops the gold once,
// a SAVE and cold LOAD keep the ground sack, and the ordinary pickup returns
// every coin and retires the sack.
func TestPurseDragDropsGoldThroughPauseSaveColdLoadAndPickup(t *testing.T) {
	f, app := purseFixture(t)
	w := f.live.world
	subject := f.live.invSubject.ID
	cell := purseCell(t, f)

	if err := app.HeadlessKey("0"); err != nil || !f.live.stopped {
		t.Fatalf("pause: %v stopped=%v", err, f.live.stopped)
	}
	px, py, err := app.HeadlessPackCellPoint(cell)
	if err != nil {
		t.Fatal(err)
	}
	gx, gy, err := app.HeadlessGroundPoint()
	if err != nil {
		t.Fatal(err)
	}
	for _, step := range []struct {
		action string
		x, y   int
	}{{"press", px, py}, {"move", px + 24, py}, {"release", gx, gy}} {
		if err := app.HeadlessPointer(step.action, step.x, step.y); err != nil {
			t.Fatal(err)
		}
	}
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}

	// Paused: one pending request, the server purse and the ground untouched,
	// the preview showing the reservation.
	if len(f.live.pending) != 1 || f.live.pending[0].Kind != sim.KindPlayerDropGold || f.live.pending[0].Group != 1000 || f.live.pending[0].Player != sim.SelfSlot {
		t.Fatalf("pending after the drag: %+v", f.live.pending)
	}
	request := f.live.pending[0]
	if g, n := purseGround(t, w); w.Purse(sim.SelfSlot) != 2500 || n != 0 || g != 0 {
		t.Fatalf("paused: purse %d ground %d/%d", w.Purse(sim.SelfSlot), g, n)
	}
	if inventoryGold(f.live.invSubject) != 1500 {
		t.Fatalf("preview %d, want 1500", inventoryGold(f.live.invSubject))
	}

	// SAVE while the request waits.
	raw, doc, actions := saveCurrentEffect(t, f)
	if got := purseSavedMoney(t, doc, sim.SelfSlot); got != 2500 {
		t.Fatalf("SAV Money %d, want the unspent 2500", got)
	}
	if gold, _ := purseSavedSacks(doc, t); len(gold) != 0 {
		t.Fatalf("SAV holds ground gold %v before the request ran", gold)
	}
	if actions.Pending == nil || len(actions.Pending.Commands) != 1 || actions.Pending.Commands[0].Command != request || actions.Pending.Commands[0].Issuer != nil {
		t.Fatalf("SAV pending %+v, want the one request", actions.Pending)
	}

	// Cold LOAD: the same state, then the first resumed tick spends it once.
	cold := openCurrentEffectSave(t, f, raw)
	if cold.live.world.Purse(sim.SelfSlot) != 2500 || len(cold.live.pending) != 1 || cold.live.pending[0] != request {
		t.Fatalf("cold LOAD purse %d pending %+v", cold.live.world.Purse(sim.SelfSlot), cold.live.pending)
	}
	if g, n := purseGround(t, cold.live.world); n != 0 || g != 0 {
		t.Fatalf("cold LOAD holds ground gold %d/%d", g, n)
	}
	cold.live.tick()
	w.SetPurse(sim.SelfSlot, w.Purse(sim.SelfSlot)) // the live world resumes below
	f.live.stopped = false
	f.live.tick()
	for name, mw := range map[string]*mapWorld{"live": f.live, "cold": cold.live} {
		if mw.world.Purse(sim.SelfSlot) != 1500 || len(mw.pending) != 0 {
			t.Fatalf("%s first tick: purse %d pending %d", name, mw.world.Purse(sim.SelfSlot), len(mw.pending))
		}
		if g, n := purseGround(t, mw.world); g != 1000 || n != 1 {
			t.Fatalf("%s first tick: ground %d/%d, want 1000 in one sack", name, g, n)
		}
	}
	if f.live.world.Hash() != cold.live.world.Hash() {
		t.Fatal("live and cold worlds differ after the first tick")
	}
	f.live.tick()
	cold.live.tick()
	if g, n := purseGround(t, cold.live.world); cold.live.world.Purse(sim.SelfSlot) != 1500 || g != 1000 || n != 1 {
		t.Fatalf("the second tick repeated the drop: purse %d ground %d/%d", cold.live.world.Purse(sim.SelfSlot), g, n)
	}
	sack := f.live.world.Sacks()[0]
	actor, _ := f.live.world.Entity(sim.EntityID(subject))
	if sack.X != actor.X || sack.Y != actor.Y {
		t.Fatalf("the sack at (%d,%d) is not on the first actor's cell (%d,%d); the request cell was (%d,%d)", sack.X, sack.Y, actor.X, actor.Y, request.X, request.Y)
	}

	// SAVE after the drop, cold LOAD, next pickup.
	raw2, doc2, _ := saveCurrentEffect(t, f)
	if got := purseSavedMoney(t, doc2, sim.SelfSlot); got != 1500 {
		t.Fatalf("SAV Money after the drop %d, want 1500", got)
	}
	gold, value := purseSavedSacks(doc2, t)
	if len(gold) != 1 || gold[0] != 1000 || value[0] != 1000 {
		t.Fatalf("SAV Sack S3C %v T1C %v, want one sack of 1000", gold, value)
	}
	cold2 := openCurrentEffectSave(t, f, raw2)
	if g, n := purseGround(t, cold2.live.world); cold2.live.world.Purse(sim.SelfSlot) != 1500 || g != 1000 || n != 1 || len(cold2.live.pending) != 0 {
		t.Fatalf("cold LOAD after the drop: purse %d ground %d/%d pending %d", cold2.live.world.Purse(sim.SelfSlot), g, n, len(cold2.live.pending))
	}
	assertCurrentWorldEqual(t, f.live.world, cold2.live.world, "after the drop")
	cold2.live.takeSackFor(sim.EntityID(subject))
	if cold2.live.world.Purse(sim.SelfSlot) != 2500 {
		t.Fatalf("the pickup returned the purse to %d, want 2500", cold2.live.world.Purse(sim.SelfSlot))
	}
	if _, n := purseGround(t, cold2.live.world); n != 0 {
		t.Fatalf("the pickup left %d sacks", n)
	}
	cold2.live.takeSackFor(sim.EntityID(subject))
	if cold2.live.world.Purse(sim.SelfSlot) != 2500 {
		t.Fatal("a repeated pickup changed the purse")
	}
	raw3, doc3, _ := saveCurrentEffect(t, cold2)
	if got := purseSavedMoney(t, doc3, sim.SelfSlot); got != 2500 {
		t.Fatalf("SAV Money after the pickup %d, want 2500", got)
	}
	if gold, _ := purseSavedSacks(doc3, t); len(gold) != 0 {
		t.Fatalf("SAV keeps retired ground gold %v", gold)
	}
	cold3 := openCurrentEffectSave(t, f, raw3)
	assertCurrentWorldEqual(t, cold2.live.world, cold3.live.world, "after the pickup")
	for tick := range 20 {
		cold2.live.tick()
		cold3.live.tick()
		if cold2.live.world.Hash() != cold3.live.world.Hash() {
			t.Fatalf("continuation differs %d ticks after the pickup", tick)
		}
	}
	if cold3.live.world.Purse(sim.SelfSlot) != 2500 {
		t.Fatalf("cold purse %d after 20 ticks", cold3.live.world.Purse(sim.SelfSlot))
	}

	// Loss control: a changed Money scalar in the SAV is the restored purse,
	// so the equality above compares restored state and not a copied value.
	lossy := alterSAV(t, raw2, func(d *sav.DocumentData) bool {
		for i := range d.Objects {
			if d.Objects[i].Class != "Player" || actorProjectionValue(t, d.Objects[i], "Slot") != sim.SelfSlot {
				continue
			}
			for k := range d.Objects[i].Values {
				if d.Objects[i].Values[k].Name == "Money" {
					d.Objects[i].Values[k].Value ^= 0x5c073f4d
					return true
				}
			}
		}
		return false
	})
	lost := openCurrentEffectSave(t, f, lossy)
	if lost.live.world.Purse(sim.SelfSlot) == 1500 {
		t.Fatal("an altered Money scalar still restored the original purse")
	}
}

func pursePointer(t *testing.T, app *ui.App, x, y int, edges ...string) {
	t.Helper()
	for _, edge := range edges {
		if err := app.HeadlessPointer(edge, x, y); err != nil {
			t.Fatal(err)
		}
	}
}

// The editor route: a double-click on the purse opens the editor, typed digits
// replace the prefilled 0, Enter queues one request at the subject's cell
// without touching the simulation, and the next tick applies it once.
func TestPurseEditorQueuesOneRequestAppliedOnce(t *testing.T) {
	f, app := purseFixture(t)
	w := f.live.world
	x, y, err := app.HeadlessPackCellPoint(purseCell(t, f))
	if err != nil {
		t.Fatal(err)
	}
	pursePointer(t, app, x, y, "press", "release", "press", "release")
	if state, open := app.HeadlessGold(); !open || !state.Open || state.Text != "0" {
		t.Fatalf("editor %+v, %v", state, open)
	}
	for _, key := range []string{"delete"} {
		if err := app.HeadlessKey(key); err != nil {
			t.Fatal(err)
		}
	}
	if err := app.HeadlessType("700", false); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessKey("enter"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	if _, open := app.HeadlessGold(); open {
		t.Fatal("the editor stayed open after Enter")
	}
	if len(f.live.pending) != 0 || w.Purse(sim.SelfSlot) != 1800 {
		// The running frame applied the request in the same step.
		t.Fatalf("pending %d purse %d, want the one request applied once", len(f.live.pending), w.Purse(sim.SelfSlot))
	}
	if g, n := purseGround(t, w); g != 700 || n != 1 {
		t.Fatalf("ground %d/%d, want 700 in one sack", g, n)
	}
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	if g, _ := purseGround(t, w); g != 700 || w.Purse(sim.SelfSlot) != 1800 {
		t.Fatalf("a later frame changed the result: purse %d ground %d", w.Purse(sim.SelfSlot), g)
	}
}

// An unsubmitted editor or held share is not part of a SAVE: the save dialog's
// function key cancels it, and the later frames queue nothing.
func TestPurseDraftIsCancelledByTheSaveKeyAndNothingDropsAfterwards(t *testing.T) {
	f, app := purseFixture(t)
	x, y, err := app.HeadlessPackCellPoint(purseCell(t, f))
	if err != nil {
		t.Fatal(err)
	}
	pursePointer(t, app, x, y, "press", "release", "press", "release")
	if _, open := app.HeadlessGold(); !open {
		t.Fatal("setup: the editor did not open")
	}
	if err := app.HeadlessKey("f2"); err != nil {
		t.Fatal(err)
	}
	if _, open := app.HeadlessGold(); open {
		t.Fatal("the save key left the editor open")
	}
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	if len(f.live.pending) != 0 || f.live.world.Purse(sim.SelfSlot) != 2500 {
		t.Fatalf("pending %d purse %d after the cancelled draft", len(f.live.pending), f.live.world.Purse(sim.SelfSlot))
	}
	if _, n := purseGround(t, f.live.world); n != 0 {
		t.Fatalf("%d sacks on the ground after the cancelled draft", n)
	}
}

// A pack rebuilt after the purse's position moved marks only the purse cell,
// and a pack without gold marks none, so an item cell is never taken for the
// purse.
func TestAppendInventoryGoldMarksOnlyTheCurrentPurseCell(t *testing.T) {
	subject := ui.InventorySubject{
		Pack: make([]*image.RGBA, 2), PackCount: []uint32{1, 1}, PackStars: []bool{false, false},
		PackPurse: []bool{false, true},
	}
	appendInventoryGold(&subject, 100, nil)
	if !slices.Equal(subject.PackPurse, []bool{false, false, true}) {
		t.Fatalf("PackPurse %v, want only the appended cell marked", subject.PackPurse)
	}
	subject.Pack, subject.PackCount, subject.PackStars = subject.Pack[:2], subject.PackCount[:2], subject.PackStars[:2]
	appendInventoryGold(&subject, 0, nil)
	if len(subject.PackPurse) != 0 {
		t.Fatalf("PackPurse %v for a pack without gold", subject.PackPurse)
	}
}
