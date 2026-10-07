package game

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func tacticalCurrentFixture(t *testing.T) (*FrontEnd, Snapshot, *sim.World) {
	t.Helper()
	f, snapshot, base := partialCurrentGraph(t)
	var rel sim.Relations
	rel.Set(sim.SelfSlot, 3, 1)
	rel.Set(3, sim.SelfSlot, 2)
	self := base.Entities()[0]
	self.ID, self.Owner, self.X, self.Y = 0, sim.SelfSlot, 20, 20
	self.Reach, self.ScanRange, self.Facing = 4, 8, 64
	self.DamageBase, self.AttackCharge, self.AttackRelax = 1, 10, 4
	first, last := self, self
	first.ID, first.Owner, first.X, first.Y = 41, 3, 20, 18
	last.ID, last.Owner, last.X, last.Y = 42, 3, 20, 22
	w, err := sim.NewRelatedWorld(11, base.Bounds(), sim.ModeCanonical, sim.Terrain{}, []sim.Entity{self, first, last}, nil, rel)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.RestoreActorTraversal([]sim.EntityID{0, 42, 41}); err != nil {
		t.Fatal(err)
	}
	sim.Step(w, []sim.Command{sim.GroupRetreat(0, sim.SelfSlot, 1)})
	actions := w.Actions()
	for i := range actions.Actors {
		if actions.Actors[i].Entity == 0 {
			a := &actions.Actors[i]
			a.HasAttackTarget, a.AttackTarget, a.AttackPhase, a.AttackCountdown = true, 41, sim.AttackCharging, 1
			a.Retreat = &sim.RetreatContinuation{Known: true, Pending: true, X: 17, Y: 20, Progress: 1, Counter: 2, Complete: true}
		}
	}
	if err := w.RestoreActions(actions, nil); err != nil {
		t.Fatal(err)
	}
	snapshot.World, err = w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	snapshot.SavedDocument = nil
	return f, snapshot, w
}

func coldTacticalCurrent(t *testing.T, f *FrontEnd, raw []byte) *FrontEnd {
	t.Helper()
	cold := cellStateFront(t)
	cold.Table, cold.Campaign = f.Table, f.Campaign
	open, town, err := cold.RestoreOriginal(raw)
	if err != nil || town {
		t.Fatal(town, err)
	}
	if err := cold.App("tactical continuation").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	return cold
}

func requireTacticalActorBytes(t *testing.T, doc *sav.DocumentData, actions *currentActionData, id sim.EntityID, x, y int32) {
	t.Helper()
	for _, binding := range actions.Bindings {
		if binding.ID != id || binding.Structure || binding.Missing {
			continue
		}
		actor := &doc.Objects[binding.Object-1]
		order, err := savedMotionRaw(actor, "U158", 148)
		if err != nil || order[8] != 1 || order[9] != 1 || binary.LittleEndian.Uint16(order[2:]) != uint16(x)|uint16(y)<<8 {
			t.Fatalf("pending/progress actor bytes: %x, %v", order, err)
		}
		for _, field := range []struct {
			name string
			want uint32
		}{{"U50", 0x16}, {"U54", 3}, {"U58", 5}} {
			raw, err := savedMotionRaw(actor, field.name, 4)
			if err != nil || binary.LittleEndian.Uint32(raw) != field.want {
				t.Fatalf("%s = %x, want %d: %v", field.name, raw, field.want, err)
			}
		}
		complete, err := savedStructureValue(actor, "U136")
		if err != nil || complete != 1 {
			t.Fatal("completion byte", complete, err)
		}
		return
	}
	t.Fatal("SAV actor binding absent")
}

func requireCompletionTacticalActorBytes(t *testing.T, doc *sav.DocumentData, actions *currentActionData, id sim.EntityID, phase sim.AttackPhase) {
	t.Helper()
	var target sim.EntityID
	found := false
	for _, actor := range actions.Actions.Actors {
		if actor.Entity == id {
			if !actor.HasAttackTarget || actor.AttackPhase != phase || actor.Retreat == nil || actor.Retreat.Progress != 0 || !actor.Retreat.Pending {
				t.Fatal("completion-clear supplement lost physical carrier or pending", actor)
			}
			target, found = actor.AttackTarget, true
		}
	}
	if !found {
		t.Fatal("completion-clear actor absent")
	}
	key := uint32(0)
	for _, binding := range actions.Bindings {
		if binding.ID == target && !binding.Structure && !binding.Missing {
			key, _ = savedStructureValue(&doc.Objects[binding.Object-1], "Identity")
		}
	}
	for _, binding := range actions.Bindings {
		if binding.ID != id || binding.Structure || binding.Missing {
			continue
		}
		actor := &doc.Objects[binding.Object-1]
		order, err := savedMotionRaw(actor, "U158", 148)
		if err != nil || order[8] != 1 || order[9] != 0 {
			t.Fatal("completion-clear pending/progress wire", order, err)
		}
		physicalPhase, complete := uint32(5), uint32(0)
		if phase == sim.AttackBoundaryOne {
			physicalPhase, complete = 0, 1
		}
		for _, field := range []struct {
			name string
			want uint32
		}{{"U50", 0x16}, {"U54", 3}, {"U58", physicalPhase}} {
			raw, err := savedMotionRaw(actor, field.name, 4)
			if err != nil || binary.LittleEndian.Uint32(raw) != field.want {
				t.Fatal("completion-clear physical wire", field.name, raw, field.want, err)
			}
		}
		got, _ := savedStructureValue(actor, "U5C")
		completed, _ := savedStructureValue(actor, "U136")
		if key == 0 || got != key || completed != complete {
			t.Fatal("completion-clear physical endpoint/completion wire", got, key, completed, complete)
		}
		return
	}
	t.Fatal("completion-clear SAV binding absent")
}

func TestCurrentTacticalSAVAfterCompletionClearIsWritable(t *testing.T) {
	for _, phase := range []sim.AttackPhase{sim.AttackCharging, sim.AttackBoundaryOne} {
		f, snapshot, w := tacticalCurrentFixture(t)
		actions := w.Actions()
		for i := range actions.Actors {
			if actions.Actors[i].Entity == 0 {
				actions.Actors[i].AttackPhase = phase
				if phase == sim.AttackBoundaryOne {
					actions.Actors[i].AttackCountdown = 0
				}
			}
		}
		if err := w.RestoreActions(actions, nil); err != nil {
			t.Fatal(err)
		}
		sim.Step(w, nil)
		var err error
		snapshot.World, err = w.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		raw, err := f.ExportCurrentSave(snapshot, "Retreat completion boundary")
		if err != nil {
			t.Fatal("current completion-clear SAVE refused", err)
		}
		doc, err := sav.DecodeDocumentData(raw)
		if err != nil {
			t.Fatal(err)
		}
		current, err := readCurrentActions(&doc)
		if err != nil || current == nil {
			t.Fatal(err)
		}
		requireCompletionTacticalActorBytes(t, &doc, current, 0, phase)
		cold := coldTacticalCurrent(t, f, raw)
		if cold.live.world.Hash() != w.Hash() {
			t.Fatal("completion-clear cold SAV LOAD changed state")
		}
		for i := range current.Actions.Actors {
			current.Actions.Actors[i].Retreat = nil
		}
		leaf, err := json.Marshal(current)
		if err != nil {
			t.Fatal(err)
		}
		if err := sav.SetNativeActions(&doc.State, leaf); err != nil {
			t.Fatal(err)
		}
		loss, err := sav.EncodeDocumentData(doc)
		if err != nil || bytes.Equal(raw, loss) {
			t.Fatal("completion-clear loss changed no SAV bytes", err)
		}
		lost := coldTacticalCurrent(t, f, loss)
		if lost.live.world.Hash() == w.Hash() {
			t.Fatal("completion-clear loss changed no decoded state")
		}
		for tick := range 2 {
			sim.Step(w, nil)
			sim.Step(cold.live.world, nil)
			sim.Step(lost.live.world, nil)
			if cold.live.world.Hash() != w.Hash() {
				t.Fatal("completion-clear SAV changed next production action", tick)
			}
			if tick == 0 {
				mine, other := w.Entities()[0], lost.live.world.Entities()[0]
				if mine.HasTarget == other.HasTarget && mine.TargetX == other.TargetX && mine.TargetY == other.TargetY && mine.HasAttackTarget == other.HasAttackTarget && mine.AttackPhase == other.AttackPhase && mine.AttackCountdown == other.AttackCountdown && mine.X == other.X && mine.Y == other.Y {
					t.Fatal("completion-clear loss changed no first production action")
				}
			}
		}
	}
}

func TestCurrentTacticalSAVColdLoadAndNextAction(t *testing.T) {
	f, snapshot, w := tacticalCurrentFixture(t)
	raw, err := f.ExportCurrentSave(snapshot, "tactical continuation")
	if err != nil {
		t.Fatal(err)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	actions, err := readCurrentActions(&doc)
	if err != nil || actions == nil || actions.Actions.ActorTraversal == nil {
		t.Fatal("missing current traversal", err)
	}
	if !reflect.DeepEqual(*actions.Actions.ActorTraversal, []sim.EntityID{0, 42, 41}) {
		t.Fatal("SAV producer changed traversal")
	}
	var held *sim.RetreatContinuation
	for _, a := range actions.Actions.Actors {
		if a.Entity == 0 {
			held = a.Retreat
		}
	}
	if held == nil || !held.Pending || held.Progress != 1 || held.X != 17 {
		t.Fatal("SAV producer lost pending/progress", held)
	}
	requireTacticalActorBytes(t, &doc, actions, 0, 17, 20)
	cold := coldTacticalCurrent(t, f, raw)
	if cold.live.world.Hash() != w.Hash() {
		t.Fatalf("cold tactical hash %x, want %x", cold.live.world.Hash(), w.Hash())
	}
	for range 2 {
		sim.Step(w, nil)
		sim.Step(cold.live.world, nil)
		if cold.live.world.Hash() != w.Hash() {
			t.Fatal("cold LOAD changed next production action")
		}
	}
}

func TestCurrentTacticalSAVSupplementLossChangesNextAction(t *testing.T) {
	f, snapshot, w := tacticalCurrentFixture(t)
	raw, err := f.ExportCurrentSave(snapshot, "tactical loss control")
	if err != nil {
		t.Fatal(err)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	actions, err := readCurrentActions(&doc)
	if err != nil {
		t.Fatal(err)
	}
	actions.Actions.ActorTraversal = nil
	for i := range actions.Actions.Actors {
		actions.Actions.Actors[i].Retreat = nil
	}
	leaf, err := json.Marshal(actions)
	if err != nil {
		t.Fatal(err)
	}
	if err := sav.SetNativeActions(&doc.State, leaf); err != nil {
		t.Fatal(err)
	}
	loss, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(raw, loss) {
		t.Fatal("loss control changed no SAV bytes")
	}
	cold := coldTacticalCurrent(t, f, loss)
	if cold.live.world.Hash() == w.Hash() {
		t.Fatal("loss control changed no decoded state")
	}
	sim.Step(w, nil)
	sim.Step(cold.live.world, nil)
	if cold.live.world.Hash() == w.Hash() {
		t.Fatal("loss control changed no next action")
	}
}

func TestCurrentTacticalSAVCastDoesNotOverwritePending(t *testing.T) {
	f, snapshot, w := tacticalCurrentFixture(t)
	actions := w.Actions()
	for i := range actions.Actors {
		if actions.Actors[i].Entity == 0 {
			actions.Actors[i].Retreat.Progress = 2
			actions.Actors[i].Retreat.Complete = false
		}
	}
	actions.Books = []sim.BookContinuation{{Caster: 0, Spell: 1, AtCell: true, X: 22, Y: 20, Remaining: 1, Phase: 1, Paid: true}}
	if err := w.RestoreActions(actions, nil); err != nil {
		t.Fatal(err)
	}
	var err error
	snapshot.World, err = w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(snapshot, "retained cast pending flee")
	if err != nil {
		t.Fatal(err)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	current, err := readCurrentActions(&doc)
	if err != nil || current == nil {
		t.Fatal("current cast actions absent", err)
	}
	for _, binding := range current.Bindings {
		if binding.ID == 0 && !binding.Structure && !binding.Missing {
			actor := &doc.Objects[binding.Object-1]
			order, err := savedMotionRaw(actor, "U158", 148)
			if err != nil || order[8] != 1 || order[9] != 2 || binary.LittleEndian.Uint16(order[2:]) != 0x1411 || order[0x5c] != 0 {
				t.Fatal("cast writer replaced pending/progress or operands", order, err)
			}
			action, err := savedMotionRaw(actor, "U54", 4)
			if err != nil || binary.LittleEndian.Uint32(action) != 0xe {
				t.Fatal("Retreat replaced retained cast action", action, err)
			}
			cold := coldTacticalCurrent(t, f, raw)
			if cold.live.world.Hash() != w.Hash() {
				t.Fatal("cold LOAD lost retained cast or pending flee")
			}
			return
		}
	}
	t.Fatal("retained cast actor absent")
}
