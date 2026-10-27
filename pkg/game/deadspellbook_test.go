package game

import (
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// A dead record whose body the World holds as no entity takes its book from
// the loaded record of the same Identity and class: the flag, the header
// word, the slot count and each slot's Spell operands. A loaded record with
// no book clears the book the writer built, and a body the World holds as an
// entity keeps the book the writer built.
func TestDeadBookWrittenFromTheLoadedRecord(t *testing.T) {
	spell := func(id, rng, defensive uint8, mana uint16, this uint32) sav.DocumentRecordData {
		return savedCurrentSpellRecord(sim.SavedSpellObject{Value: sim.SourceItemSpell{Present: true, ID: id, Range: rng, Defensive: defensive, ManaCost: mana}, This: this})
	}
	booked := mustNewRecord("Human", "HasSpellbook")
	mustSetValue(&booked, "Identity", 0x0f6a3530)
	savedObjectSetValue(&booked, "SpellsHeader", 20)
	savedObjectSetRefs(&booked, "Spells", []uint16{0, 3, 4}, true)
	mustSetCount(&booked, "Spells", 4)
	bare := mustNewRecord("Unit")
	mustSetValue(&bare, "Identity", 0x0f6a3540)
	loaded := sav.DocumentData{Objects: []sav.DocumentRecordData{booked, bare, spell(2, 12, 0, 30, 0x0e000010), spell(3, 8, 1, 30, 0x0e000020)}, DeadActors: []uint16{1, 2}}
	b := &generatedDocumentBuilder{loaded: &loaded, nextKey: 0x51000000}
	if err := b.reserveRetainedDeadEffectKeys(&sim.World{}); err != nil {
		t.Fatal(err)
	}

	record := mustNewRecord("Human")
	if err := b.retainDeadBook(&record, sim.OriginalDeadSource{Identity: 0x0f6a3530, Class: 2}, false); err != nil {
		t.Fatal(err)
	}
	has, _ := savedStructureValue(&record, "HasSpellbook")
	header, err := savedStructureValue(&record, "SpellsHeader")
	refs, _ := savedObjectRefs(&record, "Spells")
	if has != 1 || err != nil || header != 20 || mustCountValue(&record, "Spells") != 4 || len(refs) != 3 || refs[0] != 0 {
		t.Fatalf("written book flag %d header %d (%v) count %d refs %v, want the loaded book", has, header, err, mustCountValue(&record, "Spells"), refs)
	}
	for i, want := range []uint16{3, 4} {
		got, err := savedSpellRecord(&b.doc.Objects[refs[i+1]-1])
		source, _ := savedSpellRecord(&loaded.Objects[want-1])
		if err != nil || got.Value != source.Value {
			t.Fatalf("slot %d Spell %+v (%v), want %+v", i+1, got.Value, err, source.Value)
		}
	}

	cleared := mustNewRecord("Unit", "HasSpellbook")
	if err := b.retainDeadBook(&cleared, sim.OriginalDeadSource{Identity: 0x0f6a3540, Class: 1}, false); err != nil {
		t.Fatal(err)
	}
	has, _ = savedStructureValue(&cleared, "HasSpellbook")
	if _, err := savedStructureValue(&cleared, "SpellsHeader"); has != 0 || err == nil {
		t.Fatalf("a loaded record with no book wrote flag %d and a header", has)
	}
	if refs, found := savedObjectRefs(&cleared, "Spells"); found {
		t.Fatalf("a loaded record with no book wrote slots %v", refs)
	}

	held := mustNewRecord("Unit", "HasSpellbook")
	want := len(held.Values)
	if err := b.retainDeadBook(&held, sim.OriginalDeadSource{Identity: 0x0f6a3540, Class: 1}, true); err != nil {
		t.Fatal(err)
	}
	if has, _ := savedStructureValue(&held, "HasSpellbook"); has != 1 || len(held.Values) != want {
		t.Fatalf("an entity body's book changed: flag %d", has)
	}
}
