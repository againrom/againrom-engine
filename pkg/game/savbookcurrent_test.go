package game

import (
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func projectCurrentBooks(state *SnapshotSAVDocument, world *sim.World) error {
	keys, err := sav.ReserveDocumentKeys(*state.Document, 65536)
	if err != nil {
		return err
	}
	b := generatedDocumentBuilder{doc: *state.Document, reservedKeys: keys}
	before := savedDocumentIncoming(state.Document)
	if err := projectCurrentBookGraph(&b, state, world, true); err != nil {
		return err
	}
	after := savedDocumentIncoming(&b.doc)
	var retired []uint16
	for id := range before {
		if after[id] == 0 && b.doc.Objects[id-1].Class == "Spell" {
			retired = append(retired, id)
		}
	}
	doc, permutation, err := sav.RetireDocumentData(b.doc, retired)
	if err != nil {
		return err
	}
	state.Document = &doc
	return remapSavedSackDocument(state, permutation)
}

func TestCurrentLegacyBookSnapshotAndOrdinaryRanges(t *testing.T) {
	for _, ordinary := range []bool{false, true} {
		doc, binding, _ := actorProjectionFixture(t, "Human")
		e := sim.Entity{ID: binding.EntityID, X: 3, Y: 4, HP: 10, MaxHP: 10, Mind: 10,
			KnownSpells: 1 << 2, Skill: [6]int32{0, 110}}
		world, err := sim.NewSpelledWorld(1, sim.Bounds{Width: 16, Height: 16}, sim.ModeCanonical, nil,
			[]sim.Entity{e}, nil, []sim.SpellRule{{ID: 2, School: 1, MaxRange: 10, ManaCost: 4}})
		if err != nil {
			t.Fatal(err)
		}
		b := generatedDocumentBuilder{doc: doc}
		state := &SnapshotSAVDocument{Document: &b.doc, Actors: []SnapshotSAVActor{binding}}
		before := world.Hash()
		if err := projectCurrentBookGraph(&b, state, world, ordinary); err != nil {
			t.Fatal(err)
		}
		refs, _ := savedObjectRefs(&b.doc.Objects[binding.ObjectIndex-1], "Spells")
		want := uint32(13)
		if ordinary {
			want = 12
		}
		if len(refs) != 2 || refs[1] == 0 || savedRecordValueForTest(t, b.doc.Objects[refs[1]-1], "S09") != want {
			t.Fatal("wrong legacy range representation", ordinary, refs, want)
		}
		if world.Hash() != before {
			t.Fatal("range projection changed current World")
		}
	}
}

func TestCurrentBookProjectionRetainsUnchangedWeaponAlias(t *testing.T) {
	doc, binding, w := actorProjectionFixture(t, "Human")
	old, _ := savedObjectRefs(&doc.Objects[binding.ObjectIndex-1], "Spells")
	value, err := savedSpellRecord(&doc.Objects[old[0]-1])
	if err != nil {
		t.Fatal(err)
	}
	keys, err := sav.ReserveDocumentKeys(doc, 65536)
	if err != nil {
		t.Fatal(err)
	}
	b := generatedDocumentBuilder{doc: doc, reservedKeys: keys}
	weapon, err := b.item(sim.ItemInstance{Code: 0x1101, WeightPresent: true, SourceEquipment: sim.SourceEquipment{Class: sim.SourceWeapon}}, 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	mustSetRefs(&b.doc.Objects[weapon-1], "WeaponSpell", []uint16{old[0]})
	mustSetRefs(&b.doc.Objects[binding.ObjectIndex-1], "HeldWeapon", []uint16{weapon})
	book := sim.Spellbook{State: sim.BookPresent}
	book.Slots[0] = sim.BookSpell{Range: value.Value.Range, Defensive: value.Value.Defensive, ManaCost: value.Value.ManaCost}
	if err := w.ImportOriginalActorSpellbooks([]sim.OriginalActorSpellbook{{ID: binding.EntityID, KnownSpells: 1 << 1, Book: book}}); err != nil {
		t.Fatal(err)
	}
	state := &SnapshotSAVDocument{Document: &b.doc, Actors: []SnapshotSAVActor{binding}}
	want := w.Hash()
	for cycle := range 2 {
		if err := projectCurrentBooks(state, w); err != nil {
			t.Fatal(err)
		}
		actor := &state.Document.Objects[state.Actors[0].ObjectIndex-1]
		bookRefs, _ := savedObjectRefs(actor, "Spells")
		weaponRefs, _ := savedObjectRefs(actor, "HeldWeapon")
		spellRefs, _ := savedObjectRefs(&state.Document.Objects[weaponRefs[0]-1], "WeaponSpell")
		if len(bookRefs) != 1 || bookRefs[0] != spellRefs[0] {
			t.Fatalf("cycle %d split an unchanged book/weapon Spell: book %v weapon %v", cycle, bookRefs, spellRefs)
		}
		if got, err := savedSpellRecord(&state.Document.Objects[spellRefs[0]-1]); err != nil || got != value {
			t.Fatal("shared ordinary Spell changed", got, err)
		}
		if w.Hash() != want {
			t.Fatal("SAVE changed current world")
		}
	}
	book.Slots[0].Range++
	if err := w.ImportOriginalActorSpellbooks([]sim.OriginalActorSpellbook{{ID: binding.EntityID, KnownSpells: 1 << 1, Book: book}}); err != nil {
		t.Fatal(err)
	}
	if err := projectCurrentBooks(state, w); err != nil {
		t.Fatal(err)
	}
	actor := &state.Document.Objects[state.Actors[0].ObjectIndex-1]
	bookRefs, _ := savedObjectRefs(actor, "Spells")
	weaponRefs, _ := savedObjectRefs(actor, "HeldWeapon")
	spellRefs, _ := savedObjectRefs(&state.Document.Objects[weaponRefs[0]-1], "WeaponSpell")
	if bookRefs[0] == spellRefs[0] {
		t.Fatal("changed book rewrote its weapon's Spell")
	}
	if got, err := savedSpellRecord(&state.Document.Objects[spellRefs[0]-1]); err != nil || got != value {
		t.Fatal("changed book changed weapon parameters", got, err)
	}
}

func TestCurrentBookProjectionUpdatesSlotsWithoutChangingSharedSpell(t *testing.T) {
	for _, shared := range []bool{false, true} {
		doc, b, w := actorProjectionFixture(t, "Human")
		s := &SnapshotSAVDocument{Document: &doc, Actors: []SnapshotSAVActor{b}}
		old, _ := savedObjectRefs(&doc.Objects[b.ObjectIndex-1], "Spells")
		oldKey, _ := savedStructureValue(&doc.Objects[old[0]-1], "This")
		if shared {
			// Another actor owns the same old Spell. Its parameters must remain
			// untouched when this actor changes its own book instance.
			found := false
			for i := range doc.Objects {
				if i != int(b.ObjectIndex)-1 && (doc.Objects[i].Class == "Human" || doc.Objects[i].Class == "Unit") {
					r := &doc.Objects[i]
					savedObjectSetValue(r, "HasSpellbook", 1)
					savedObjectSetValue(r, "SpellsHeader", 0)
					savedObjectSetRefs(r, "Spells", []uint16{old[0]}, true)
					mustSetCount(r, "Spells", 2)
					found = true
					break
				}
			}
			if !found {
				t.Fatal("shared book fixture has no second actor")
			}
		}
		book := sim.Spellbook{State: sim.BookPresent}
		book.Slots[0] = sim.BookSpell{Range: 17, Defensive: 1, ManaCost: 123}
		book.Slots[6] = sim.BookSpell{Range: 5, ManaCost: 321}
		if err := w.ImportOriginalActorSpellbooks([]sim.OriginalActorSpellbook{{ID: b.EntityID, KnownSpells: 1<<1 | 1<<7, Book: book}}); err != nil {
			t.Fatal(err)
		}
		hash := w.Hash()
		if err := projectCurrentBooks(s, w); err != nil {
			t.Fatal(err)
		}
		if w.Hash() != hash {
			t.Fatal("book projection mutated current state")
		}
		r := &s.Document.Objects[s.Actors[0].ObjectIndex-1]
		slots, _ := savedObjectRefs(r, "Spells")
		if len(slots) != 7 || slots[0] == 0 || slots[6] == 0 || slots[2] != 0 {
			t.Fatalf("current book positions%v", slots)
		}
		for _, q := range []struct {
			slot            int
			rangeByte, cost uint32
		}{{0, 17, 123}, {6, 5, 321}} {
			spell := s.Document.Objects[slots[q.slot]-1]
			if savedRecordValueForTest(t, spell, "S08") != uint32(q.slot+1) || savedRecordValueForTest(t, spell, "S09") != q.rangeByte || savedRecordValueForTest(t, spell, "S0C") != q.cost {
				t.Fatal("current Spell values lost")
			}
		}
		for _, spell := range s.Document.Objects {
			if spell.Class == "Spell" {
				key, _ := savedStructureValue(&spell, "This")
				if shared && key == oldKey && savedRecordValueForTest(t, spell, "S09") != 9 {
					t.Fatal("changed shared Spell")
				}
			}
		}
		// Present-empty and absent are distinct source states, and neither may
		// keep stale positional entries after a second current projection.
		for _, presence := range []sim.BookState{sim.BookPresent, sim.BookAbsent} {
			if err := w.ImportOriginalActorSpellbooks([]sim.OriginalActorSpellbook{{ID: b.EntityID, Book: sim.Spellbook{State: presence}}}); err != nil {
				t.Fatal(err)
			}
			if err := projectCurrentBooks(s, w); err != nil {
				t.Fatal(err)
			}
			r = &s.Document.Objects[s.Actors[0].ObjectIndex-1]
			slots, _ = savedObjectRefs(r, "Spells")
			want := uint32(0)
			if presence == sim.BookPresent {
				want = 1
			}
			if savedRecordValueForTest(t, *r, "HasSpellbook") != want || len(slots) != 0 {
				t.Fatal("empty/absent book reused donor entries")
			}
		}
	}
}
