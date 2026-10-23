package game

import (
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func TestSavEquipmentSlotLayoutTable(t *testing.T) {
	type place struct {
		field   string
		ordinal int
	}
	want := [sim.EquipSlots]place{{"HeldWeapon", 0}, {"HeldShield", 0}}
	for slot := 2; slot < sim.EquipSlots; slot++ {
		want[slot] = place{"Worn", slot}
	}
	for _, class := range []struct {
		name  string
		carry bool
	}{{"Unit", false}, {"Humanoid", true}, {"Human", true}} {
		if got := savClassCarriesWorn(class.name); got != class.carry {
			t.Fatalf("%s carries Worn = %v, want %v", class.name, got, class.carry)
		}
		for slot := range want {
			field, ordinal := savEquipmentSlotRef(slot)
			if (place{field, ordinal}) != want[slot] {
				t.Fatalf("%s slot %d at %s[%d], want %v", class.name, slot, field, ordinal, want[slot])
			}
			if field == "Worn" && !class.carry {
				continue
			}
			if back, ok := savEquipmentFieldSlot(field, ordinal); !ok || back != slot {
				t.Fatalf("%s %s[%d] reads back as slot %d %v, want %d", class.name, field, ordinal, back, ok, slot)
			}
		}
	}
	if _, ok := savEquipmentFieldSlot("Inventory", 0); ok {
		t.Fatal("Inventory maps to an equipment slot")
	}
	if want := [3]string{"HeldWeapon", "HeldShield", "Worn"}; savEquipmentFields != want {
		t.Fatalf("fields %v, want %v", savEquipmentFields, want)
	}
}

func TestSavEquipmentSitePolicyTable(t *testing.T) {
	readers := map[string]savEquipmentPolicy{
		"city topology":    {Worn: savWornByField, Held: savHeldExact, Low: savLowExclusive, High: savWornExactLength},
		"worn mask":        {Worn: savWornByField, Held: savHeldFirst, Low: savLowFillsEmpty, High: savHighIgnored},
		"passive body":     {Worn: savWornByClass, Held: savHeldSingle, Low: savLowIgnored, High: savHighIgnored},
		"saved item graph": {Worn: savWornByField, Held: savHeldExact, Low: savLowRefused, High: savHighRefused},
		"city party":       {Worn: savWornByField, Held: savHeldFirst, Low: savLowFillsZero, High: savHighIgnored},
	}
	got := map[string]savEquipmentPolicy{
		"city topology": cityTopologyEquipment, "worn mask": wornMaskEquipment, "passive body": passiveBodyEquipment,
		"saved item graph": savedItemEquipment, "city party": cityPartyEquipment,
	}
	if !reflect.DeepEqual(got, readers) {
		t.Fatalf("reader policies %+v, want %+v", got, readers)
	}
	writers := map[string]savWornRule{"city projection": savWornByField, "terminal body": savWornByClass,
		"current items": savWornByClass, "saved items": savWornByClass, "item binding check": savWornByClass}
	gotWriters := map[string]savWornRule{"city projection": cityProjectionWorn, "terminal body": terminalBodyWorn,
		"current items": currentItemsWorn, "saved items": savedItemsWorn, "item binding check": itemBindingCheckWorn}
	if !reflect.DeepEqual(gotWriters, writers) {
		t.Fatalf("writer rules %v, want %v", gotWriters, writers)
	}
}

func equipmentRecord(class string, weapon, shield []uint16, worn []uint16) *sav.DocumentRecordData {
	r := &sav.DocumentRecordData{Class: class}
	if weapon != nil {
		r.RefSlots = append(r.RefSlots, sav.DocumentRefsData{Name: "HeldWeapon", Objects: weapon})
	}
	if shield != nil {
		r.RefSlots = append(r.RefSlots, sav.DocumentRefsData{Name: "HeldShield", Objects: shield})
	}
	if worn != nil {
		r.RefSlots = append(r.RefSlots, sav.DocumentRefsData{Name: "Worn", Objects: worn})
	}
	return r
}

func wornRun(n int, at map[int]uint16) []uint16 {
	out := make([]uint16, n)
	for i, ref := range at {
		out[i] = ref
	}
	return out
}

// Each record shape reads per site policy: the refs of slots 0, 1 and 2, or
// the fault the policy raises.
func TestReadSavEquipmentPolicies(t *testing.T) {
	type outcome struct {
		refs  [3]uint16
		fault savEquipmentFault
	}
	policies := []savEquipmentPolicy{cityTopologyEquipment, wornMaskEquipment, passiveBodyEquipment, savedItemEquipment, cityPartyEquipment}
	cases := []struct {
		name   string
		record *sav.DocumentRecordData
		want   [5]outcome
	}{
		{"plain", equipmentRecord("Humanoid", []uint16{5}, []uint16{0}, wornRun(12, map[int]uint16{2: 7})),
			[5]outcome{{refs: [3]uint16{5, 0, 7}}, {refs: [3]uint16{5, 0, 7}}, {refs: [3]uint16{5, 0, 7}}, {refs: [3]uint16{5, 0, 7}}, {refs: [3]uint16{5, 0, 7}}}},
		{"held beside Worn 0", equipmentRecord("Humanoid", []uint16{5}, []uint16{0}, wornRun(12, map[int]uint16{0: 9})),
			[5]outcome{{fault: savOverlapFault}, {refs: [3]uint16{5}}, {refs: [3]uint16{5}}, {fault: savOverlapFault}, {refs: [3]uint16{5}}}},
		{"Worn 0 under an empty held field", equipmentRecord("Humanoid", []uint16{0}, []uint16{0}, wornRun(12, map[int]uint16{0: 9})),
			[5]outcome{{refs: [3]uint16{9}}, {refs: [3]uint16{9}}, {refs: [3]uint16{}}, {fault: savOverlapFault}, {refs: [3]uint16{9}}}},
		{"Worn 0 without a held field", equipmentRecord("Humanoid", nil, []uint16{0}, wornRun(12, map[int]uint16{0: 9})),
			[5]outcome{{fault: savHeldShapeFault}, {refs: [3]uint16{9}}, {refs: [3]uint16{}}, {fault: savHeldShapeFault}, {refs: [3]uint16{}}}},
		{"Unit with a Worn field", equipmentRecord("Unit", []uint16{5}, []uint16{6}, wornRun(12, map[int]uint16{2: 7})),
			[5]outcome{{refs: [3]uint16{5, 6, 7}}, {refs: [3]uint16{5, 6, 7}}, {refs: [3]uint16{5, 6}}, {refs: [3]uint16{5, 6, 7}}, {refs: [3]uint16{5, 6, 7}}}},
		{"Worn past the last slot", equipmentRecord("Humanoid", []uint16{5}, []uint16{0}, wornRun(13, map[int]uint16{2: 7, 12: 4})),
			[5]outcome{{fault: savWornShapeFault}, {refs: [3]uint16{5, 0, 7}}, {refs: [3]uint16{5, 0, 7}}, {fault: savOverlapFault}, {refs: [3]uint16{5, 0, 7}}}},
		{"two refs in a held field", equipmentRecord("Humanoid", []uint16{5, 8}, []uint16{0}, wornRun(12, nil)),
			[5]outcome{{fault: savHeldShapeFault}, {refs: [3]uint16{5}}, {refs: [3]uint16{}}, {fault: savHeldShapeFault}, {refs: [3]uint16{5}}}},
	}
	for _, c := range cases {
		for i, p := range policies {
			v, err := readSavEquipment(c.record, p, savedObjectRefsGetter(c.record))
			var got outcome
			if err != nil {
				got.fault = err.(*savEquipmentError).Fault
			} else {
				copy(got.refs[:], v.Refs[:3])
			}
			if got != c.want[i] {
				t.Errorf("%s under policy %+v: got %+v, want %+v", c.name, p, got, c.want[i])
			}
		}
	}
}

func TestReadSavEquipmentLocated(t *testing.T) {
	r := equipmentRecord("Humanoid", nil, []uint16{0}, wornRun(4, map[int]uint16{3: 7}))
	v, err := readSavEquipment(r, cityPartyEquipment, savedObjectRefsGetter(r))
	if err != nil {
		t.Fatal(err)
	}
	want := [sim.EquipSlots]bool{false, true, true, true}
	if v.Located != want {
		t.Fatalf("located %v, want %v", v.Located, want)
	}
}

func TestWriteSavEquipment(t *testing.T) {
	var refs [sim.EquipSlots]uint16
	for slot := range refs {
		refs[slot] = uint16(slot + 1)
	}
	for _, carries := range []bool{false, true} {
		got := map[string][]uint16{}
		var order []string
		writeSavEquipment(refs, carries, func(field string, refs []uint16) {
			got[field], order = refs, append(order, field)
		})
		want := map[string][]uint16{"HeldWeapon": {1}, "HeldShield": {2}}
		wantOrder := []string{"HeldWeapon", "HeldShield"}
		if carries {
			want["Worn"] = []uint16{0, 0, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}
			wantOrder = append(wantOrder, "Worn")
		}
		if !reflect.DeepEqual(got, want) || !reflect.DeepEqual(order, wantOrder) {
			t.Fatalf("carries %v: wrote %v in order %v, want %v in order %v", carries, got, order, want, wantOrder)
		}
	}
	if refs[0] != 1 || refs[1] != 2 {
		t.Fatal("writer changed its caller's refs")
	}
}
