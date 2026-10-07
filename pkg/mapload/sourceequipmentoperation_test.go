package mapload

import (
	"reflect"
	"testing"

	"againrom/pkg/sim"
)

func TestSourceTownEquipmentOperationPublishesOnlySuccessfulOutput(t *testing.T) {
	item := sim.ItemInstance{Code: 0x701, WeightPresent: true, Weight: 2, SourceEquipment: sim.SourceEquipment{Class: sim.SourceArmor, DefinitionRow: 1, OwnKind: 7}}
	w := sourceWorld1110(t, sourceLiteral1110(), []sim.ItemInstance{item})
	p := PartyMember{ID: "hero", Carry: &Carry{Items: []uint16{item.Code}, ItemInstances: []sim.ItemInstance{item}, OrderedStacks: []sim.ItemStack{sim.StackItem(item, 1)}, LiveLoad: w.Entities()[0].CurrentActorLoad()}}
	before := CloneParty([]PartyMember{p})[0]
	r := sim.SourceEquipmentReceipt{Events: []sim.SourceEquipmentEvent{{Ticket: 91, Count: 4}}, Reservation: 92}
	prior := r
	prior.Events = append([]sim.SourceEquipmentEvent(nil), r.Events...)
	topology := sim.SourceEquipmentTopology{}
	operation := sim.SourceEquipmentOperation{Topology: &topology, Receipt: &r}
	if next, _, ok := SourceTownEquipment(p, nil, 0, 7, sim.ItemInstance{}, false, operation); ok || !reflect.DeepEqual(p, before) || !reflect.DeepEqual(next, before) || !reflect.DeepEqual(r, prior) {
		t.Fatal("failed town operation changed input or published receipt")
	}
	topology.Pack = []uint64{7}
	next, _, ok := SourceTownEquipment(p, nil, 0, 7, sim.ItemInstance{}, false, operation)
	if !ok || !reflect.DeepEqual(p, before) || len(r.Events) != 2 || r.Events[0].Count != 1 || r.Events[1].Place.Kind != sim.SourceEquipmentWorn || r.Reservation != 0 || len(next.Carry.OrderedStacks) != 0 || next.Carry.EquippedItems[6].Code != item.Code || next.Carry.EquippedItems[6].ObjectID != 0 {
		t.Fatal("successful town operation lost current values or explicit output", r)
	}
}
