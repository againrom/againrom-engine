package sim

import (
	"encoding/binary"
	"reflect"
	"testing"
)

func TestAuthoredPatrolKeepsNativePhasesWithGroupOwnership(t *testing.T) {
	actor := engFighter(1, 2, 5, 5)
	actor.Group = 1
	native := engWorld(t, engRel(t), actor)
	bound := engWorld(t, engRel(t), actor)
	g := SavedGroup{ID: 1, Selector: 1, Authored: true, Owner: SavedGroupReference{Class: 1, Owner: 2}, Members: []SavedGroupMember{{Entity: 1, Bound: true}}}
	g.AI[0x45] = 1
	if err := bound.ImportSavedGroups([]SavedGroup{g}, []SavedActorOrder{{Entity: 1, State: uint32(actorStateGuard), Authored: true}}); err != nil {
		t.Fatal(err)
	}
	order := ScriptInstant{Op: ScriptInstantGroupOrder, Group: 1, HasGroup: true, Args: [scriptParams]int32{subCommandPatrol, 8, 5}}
	native.runInstant(order)
	bound.runInstant(order)
	var cold *World
	for tick := 1; tick <= 257; tick++ {
		Step(native, nil)
		Step(bound, nil)
		if !reflect.DeepEqual(native.Entities(), bound.Entities()) || !reflect.DeepEqual(native.Route(1), bound.Route(1)) || native.RandomState() != bound.RandomState() {
			t.Fatalf("Group registration changed native patrol at tick%d\nnative=%+v\nbound=%+v", tick, native.Entities(), bound.Entities())
		}
		if bound.savedGroupFor(1) == nil || !bound.nativePatrol(0) {
			t.Fatal("patrol dropped Group/order ownership")
		}
		o, e := bound.savedOrder(1), bound.entities[0]
		if binary.LittleEndian.Uint16(o.Raw[2:]) != uint16(e.legX())|uint16(e.legY())<<8 {
			t.Fatal("native ring cursor lost its archive operands")
		}
		if cold != nil {
			Step(cold, nil)
			if cold.Hash() != bound.Hash() {
				t.Fatalf("cold patrol changed at tick%d", tick)
			}
		}
		if tick == 33 || tick == 129 {
			b, err := bound.MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}
			cold = &World{}
			if err = cold.UnmarshalBinary(b); err != nil {
				t.Fatal(err)
			}
		}
	}
	bound.cmdGroupRoam(1)
	if bound.nativePatrol(0) || bound.entities[0].ActorState != actorStatePatrol || bound.savedGroupFor(1) == nil {
		t.Fatal("current Roam order lost precedence or discarded the retained actor policy")
	}
}

func TestOriginalPatrolKeepsItsOrderBlockDispatch(t *testing.T) {
	w := savedGroupWorld(t)
	w.savedGroups.Groups[0].AI[0x20] = 0
	o := w.savedOrder(10)
	o.State, o.Authored, o.Patrol = uint32(actorStatePatrol), false, []uint16{0x0808, 0x0909, 0x0a0a}
	binary.LittleEndian.PutUint16(o.Raw[2:], 0x0808)
	w.entities[0].ActorState = actorStatePatrol
	before := w.Hash()
	w.actorPass()
	if w.Hash() != before || w.nativePatrol(0) {
		t.Fatal("original order selected the native patrol phase")
	}
	w.savedActorDispatch(0)
	if binary.LittleEndian.Uint16(o.Raw[2:]) != 0x0909 {
		t.Fatal("original three-point ring lost its own successor")
	}
	o.Authored = true
	w.entities[0].ActorState = actorStateGuard
	if w.nativePatrol(0) {
		t.Fatal("legacy authored order lost its retained block dispatch")
	}
}
