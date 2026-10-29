package game

import (
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// The SAVE writes 0 for a Token Reference of an Item-class or Effect record
// that its loaded record held without a key of a record at or before it in
// the loaded archive order. The written order does not decide it, a Reference
// play changed is written as it is, and other classes keep theirs.
func TestTokenReferenceKeepsOnlyAnEarlierLoadedKey(t *testing.T) {
	record := func(class string, values ...sav.DocumentValueData) sav.DocumentRecordData {
		return sav.DocumentRecordData{Class: class, Values: values}
	}
	v := func(name string, value uint32) sav.DocumentValueData {
		return sav.DocumentValueData{Name: name, Value: value}
	}
	loaded := sav.DocumentData{Objects: []sav.DocumentRecordData{
		record("Player", v("This", 0x10)),
		record("Human", v("Identity", 0x20), v("Reference", 0x99)),
		record("Armor", v("Identity", 0x21), v("Reference", 0x20)),
		record("Weapon", v("Identity", 0x22), v("Reference", 0x30)),
		record("Effect", v("Identity", 0x23), v("Reference", 0x10)),
		record("Item", v("Identity", 0x24), v("Reference", 0x99)),
		record("Item", v("Identity", 0x28), v("Reference", 0x99)),
		record("Shield", v("Identity", 0x25), v("Reference", 0x25)),
		record("Effect_DirectDamage", v("Identity", 0x26), v("Reference", 0x31)),
		record("Sack", v("Identity", 0x27), v("Reference", 0x99)),
		record("Human", v("Identity", 0x30), v("Reference", 0x10)),
		record("Human", v("Identity", 0x31), v("Reference", 0x10)),
	}}
	written := sav.DocumentData{Objects: []sav.DocumentRecordData{
		record("Player", v("This", 0x10)),
		record("Human", v("Identity", 0x30), v("Reference", 0x10)),
		record("Human", v("Identity", 0x20), v("Reference", 0x99)),
		record("Armor", v("Identity", 0x21), v("Reference", 0x20)),
		record("Weapon", v("Identity", 0x22), v("Reference", 0x30)),
		record("Effect", v("Identity", 0x23), v("Reference", 0x10)),
		record("Item", v("Identity", 0x24), v("Reference", 0x10)),
		record("Item", v("Identity", 0x28), v("Reference", 0x99)),
		record("Shield", v("Identity", 0x25), v("Reference", 0x25)),
		record("Effect_DirectDamage", v("Identity", 0x26), v("Reference", 0x31)),
		record("Sack", v("Identity", 0x27), v("Reference", 0x99)),
		record("Weapon", v("Identity", 0x29), v("Reference", 0x77)),
	}}
	ours := written
	ours.Objects = slices.Clone(written.Objects)
	for i := range ours.Objects {
		ours.Objects[i].Values = slices.Clone(written.Objects[i].Values)
	}
	finalizeLoadedTokenReferences(&written, loadedTokenReferenceMisses(&loaded))
	for i, want := range []uint32{0, 0x10, 0x99, 0x20, 0, 0x10, 0x10, 0, 0x25, 0, 0x99, 0x77} {
		if i == 0 {
			continue
		}
		if got, err := savedStructureValue(&written.Objects[i], "Reference"); err != nil || got != want {
			t.Errorf("%s #%d Reference %#x (%v), want %#x", written.Objects[i].Class, i+1, got, err, want)
		}
	}
	// A loaded file with Againrom state keeps every Reference (DIV-4145).
	if err := sav.SetNativeActions(&loaded.State, []byte("{}")); err != nil {
		t.Fatal(err)
	}
	if misses := loadedTokenReferenceMisses(&loaded); misses != nil {
		t.Errorf("Againrom file misses %v, want none", misses)
	}
	finalizeLoadedTokenReferences(&ours, loadedTokenReferenceMisses(&loaded))
	if got, _ := savedStructureValue(&ours.Objects[4], "Reference"); got != 0x30 {
		t.Errorf("Againrom file Weapon Reference %#x, want the loaded 0x30", got)
	}
}

// A dead record whose body the World holds as no entity writes the load
// word U90 of its loaded record; a body the World holds as an entity keeps
// the word its writer built.
func TestDeadLoadWrittenFromTheLoadedRecord(t *testing.T) {
	loadedHuman := mustNewRecord("Human")
	mustSetValue(&loadedHuman, "Identity", 0x0f6a3530)
	mustSetValue(&loadedHuman, "U90", 22)
	loaded := sav.DocumentData{Objects: []sav.DocumentRecordData{loadedHuman}, DeadActors: []uint16{1}}
	b := &generatedDocumentBuilder{loaded: &loaded, nextKey: 0x51000000}
	source := sim.OriginalDeadSource{Identity: 0x0f6a3530, Class: 2}

	absent := mustNewRecord("Human")
	if err := b.retainDeadLoad(&absent, source, false); err != nil {
		t.Fatal(err)
	}
	if got, err := savedStructureValue(&absent, "U90"); err != nil || got != 22 {
		t.Fatalf("absent body U90 %d (%v), want the loaded 22", got, err)
	}
	held := mustNewRecord("Human")
	want, _ := savedStructureValue(&held, "U90")
	if err := b.retainDeadLoad(&held, source, true); err != nil {
		t.Fatal(err)
	}
	if got, _ := savedStructureValue(&held, "U90"); got != want {
		t.Fatalf("entity body U90 %d, want the built %d", got, want)
	}
	other := mustNewRecord("Human")
	if err := b.retainDeadLoad(&other, sim.OriginalDeadSource{Identity: 0x0f6a3540, Class: 2}, false); err != nil {
		t.Fatal(err)
	}
	if got, _ := savedStructureValue(&other, "U90"); got != want {
		t.Fatalf("body with no loaded record U90 %d, want the built %d", got, want)
	}
}
