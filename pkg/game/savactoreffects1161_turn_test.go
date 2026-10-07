package game

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// The sole review's counterexample uses ordinary Haste24 and Slow28 at
// magnitude7, with the shorter Haste expiring during a north-to-east turn.
// This synthetic SAV adds exact exclusive children to an existing complete
// actor fixture; expected rates/counts never call the production derive.
func actor1161TurnSource(t *testing.T) []byte {
	t.Helper()
	doc, err := sav.DecodeDocumentData(unit1158AppFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	var refs []uint16
	for i, spec := range []struct{ id, operand uint32 }{{24, 7 | 3<<16}, {28, 65529 | 800<<16}} {
		effect := literalSavedEffectRecord(0xabc000 + uint32(i))
		for name, value := range map[string]uint32{"E0C": spec.id, "E3C": 17, "E3D": 1, "E40": spec.operand} {
			savedObjectSetValue(&effect, name, value)
		}
		doc.Objects = append(doc.Objects, effect)
		refs = append(refs, uint16(len(doc.Objects)))
	}
	found := 0
	for i := range doc.Objects {
		a := &doc.Objects[i]
		if a.Class != "Human" {
			continue
		}
		found++
		savedObjectSetRefs(a, "Effects", refs, true)
		for name, value := range map[string]uint32{
			"Body": 10, "Reaction": 7, "Mind": 10, "Spirit": 10, "Speed": 7,
			"U8E": 0, "U90": 0, "Capacity": 101, "Health": 20, "HealthMax": 20,
			"HealthRegen": 100, "Mana": 0, "ManaMax": 0, "ManaRegen": 50,
			"U144": 1<<24 | 1<<28, "T0C": 33, "UA2": 0, "UA3": 0,
		} {
			if err := savedActorSetValue(a, name, value); err != nil {
				t.Fatal(err)
			}
		}
		for j := range a.Raw {
			r := &a.Raw[j]
			switch r.Name {
			case "UA6", "UBE", "U114", "UD4", "H1CC", "U154":
				clear(r.Bytes)
				if r.Name == "U154" {
					r.Bytes[10] = 7
				}
			}
		}
	}
	if found != 1 {
		t.Fatal("turn fixture Human population", found)
	}
	doc, _, err = sav.ReindexDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func actor1161TurnState(t *testing.T, f *FrontEnd, ticks int, wantSpeed int32, wantTurn uint8) sim.Entity {
	t.Helper()
	var subject sim.Entity
	found := false
	for _, e := range f.live.world.Entities() {
		if e.MapUnitID == 92 {
			subject, found = e, true
		}
	}
	if !found || subject.ActorLoad.Source.Class != 2 || subject.Speed != wantSpeed || subject.RotationSpeed != wantSpeed || subject.ActorLoad.Source.MoverSpeed != uint8(wantSpeed) || subject.TurnRemaining != wantTurn {
		t.Fatalf("tick%d source-derived speed/turn=%d/%d/%d %d/%d want%d remaining%d", ticks, subject.Speed, subject.RotationSpeed, subject.ActorLoad.Source.MoverSpeed, subject.TurnRemaining, subject.TurnTotal, wantSpeed, wantTurn)
	}
	if ticks >= 3 && (subject.Facing != 64 || subject.DesiredFacing != 64 || subject.TurnTotal != 0) {
		t.Fatal("zero-rate compatibility did not settle the requested facing", subject.Facing, subject.DesiredFacing, subject.TurnTotal)
	}
	active := f.live.world.ActiveEffects()
	if ticks < 3 {
		if len(active) != 2 || active[0].Spell != 24 || active[0].Remaining != uint16(3-ticks) || active[1].Spell != 28 || active[1].Remaining != uint16(800-ticks) {
			t.Fatal("pre-expiry nominal timers differ", active)
		}
	} else if len(active) != 1 || active[0].Spell != 28 || active[0].Magnitude != -7 || active[0].Remaining != uint16(800-ticks) {
		t.Fatal("expiry lost or rewrote the remaining Slow", active)
	}
	snapshot, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatalf("tick%d ordinary Snapshot: %v", ticks, err)
	}
	if snapshot.SavedDocument.ActorEffects.Unavailable != "" {
		t.Fatal("current effect projection unavailable", snapshot.SavedDocument.ActorEffects.Unavailable)
	}
	if err := validateSavedActorEffectsWorld(snapshot.SavedDocument, f.live.world); err != nil {
		t.Fatal("current Document effect state", err)
	}
	// An independent current-document assertion supplements native hashes and
	// the production binding validator: the settled facing/rate and surviving
	// Slow operand must be authored, and the expired Haste child must disappear.
	for _, binding := range snapshot.SavedDocument.Actors {
		if binding.EntityID != subject.ID {
			continue
		}
		actor := &snapshot.SavedDocument.Document.Objects[binding.ObjectIndex-1]
		speed, speedErr := savedStructureValue(actor, "Speed")
		mover, moverErr := savedMotionRaw(actor, "U154", 180)
		if speedErr != nil || moverErr != nil || speed != uint32(wantSpeed) || mover[0] != subject.Facing || mover[10] != uint8(wantSpeed) {
			t.Fatal("current Document speed/facing/rate differs", speed, speedErr, moverErr)
		}
		if ticks >= 3 {
			refs, ok := savedObjectRefs(actor, "Effects")
			mask, maskErr := savedStructureValue(actor, "U144")
			modifier, modifierErr := savedMotionRaw(actor, "UD4", 64)
			if !ok || len(refs) != 1 || maskErr != nil || mask&(1<<24|1<<28) != 1<<28 || modifierErr != nil || int16(binary.LittleEndian.Uint16(modifier[4:])) != -7 {
				t.Fatal("current Document retained expired Haste or lost Slow/mask/modifier", refs, mask)
			}
			child := &snapshot.SavedDocument.Document.Objects[refs[0]-1]
			id, _ := savedStructureValue(child, "E0C")
			operand, _ := savedStructureValue(child, "E40")
			if id != 28 || operand != 65529|uint32(800-ticks)<<16 {
				t.Fatal("current Document Slow ID/operand differs", id, operand)
			}
		}
	}
	return subject
}

func TestActorEffects1161ExpiryToZeroDuringTurnAppSave(t *testing.T) {
	f := withCurrentMenuDefinitions(t, unit1158FixtureFront(t), 33)
	f.SetDeterministicFrames(true)
	app := f.App("Source-derived effect expiry during turn")
	app.Layout(1024, 768)
	path := filepath.Join(t.TempDir(), "turn.sav")
	if err := os.WriteFile(path, actor1161TurnSource(t), 0600); err != nil {
		t.Fatal(err)
	}
	store := SaveStore{Dir: t.TempDir()}
	save, list, load := f.SaveSeams(store, OriginalStore{Dir: filepath.Dir(path)}, nil)
	app.SetSaveSeams(save, list, load)
	groundAppLoad(t, app, list, "turn.sav")
	subject := actor1161TurnState(t, f, 0, 7, 0)
	sim.Step(f.live.world, []sim.Command{{Kind: sim.KindMoveTo, Entity: subject.ID, X: subject.X + 3, Y: subject.Y}})
	actor1161TurnState(t, f, 1, 7, 10)
	sim.Step(f.live.world, nil)
	actor1161TurnState(t, f, 2, 7, 9)
	mover1160Menu(t, app)
	if err := app.HeadlessGameMenuAction("save"); err != nil {
		t.Fatal("pre-expiry menu SAVE", err)
	}
	entries, err := store.List()
	if err != nil || len(entries) != 1 {
		t.Fatal(entries, err, app.HeadlessMessage())
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	fresh := withCurrentMenuDefinitions(t, unit1158FixtureFront(t), 33)
	fresh.SetDeterministicFrames(true)
	app2 := fresh.App("Fresh native zero-rate expiry")
	app2.Layout(1024, 768)
	save, list, load = fresh.SaveSeams(store, OriginalStore{}, nil)
	app2.SetSaveSeams(save, list, load)
	groundAppLoad(t, app2, list, entries[0].Name)
	actor1161TurnState(t, fresh, 2, 7, 9)
	if f.live.world.Hash() != fresh.live.world.Hash() {
		currentMenuWorldDiagnostics(t, f.live.world, fresh.live.world)
		t.Fatal("pre-expiry ordinary SAVE changed current world")
	}
	for tick := 3; tick <= 5; tick++ {
		sim.Step(f.live.world, nil)
		sim.Step(fresh.live.world, nil)
		actor1161TurnState(t, f, tick, 0, 0)
		actor1161TurnState(t, fresh, tick, 0, 0)
		if f.live.world.Hash() != fresh.live.world.Hash() {
			currentMenuWorldDiagnostics(t, f.live.world, fresh.live.world)
			t.Fatal("source-free expiry continuation differs", tick)
		}
		if tick == 3 {
			mover1160Menu(t, app2)
			if err := app2.HeadlessGameMenuAction("save"); err != nil {
				t.Fatal("post-expiry ordinary menu SAVE", err)
			}
			entries, err = store.List()
			if err != nil {
				t.Fatal(err)
			}
			fresh = withCurrentMenuDefinitions(t, unit1158FixtureFront(t), 33)
			fresh.SetDeterministicFrames(true)
			app2 = fresh.App("Fresh post-expiry native LOAD")
			app2.Layout(1024, 768)
			save, list, load = fresh.SaveSeams(store, OriginalStore{}, nil)
			app2.SetSaveSeams(save, list, load)
			groundAppLoad(t, app2, list, entries[0].Name)
			actor1161TurnState(t, fresh, 3, 0, 0)
		}
	}
	t.Log("real BindSourceDerive: Haste24+7 expiry at tick3 leaves Slow28-7, rates7->0, turn9/10->settled east; ordinary menu SAVE before and after expiry, removed SAV source, fresh native LOAD and continued ticks pass")
}
