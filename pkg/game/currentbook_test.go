package game

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func TestCurrentCityLegacyBookKeepsNativeExtraMembership(t *testing.T) {
	payload, newFront := spellbookCitySource(t)
	f := newFront()
	if _, town, err := f.RestoreOriginal(payload); err != nil || !town {
		t.Fatal(err)
	}
	f.Carried[0].Book = sim.Spellbook{}
	f.Carried[0].KnownSpells = 2 | 1 | 1<<29 | 1<<31
	f.Carried[0].SpellbookRestored, f.Carried[0].SpellbookPresent = false, false
	want := mapload.CloneParty(f.Carried)
	s, label, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(s, label)
	if err != nil {
		t.Fatal(err)
	}
	assertCityBookSAV(t, raw, newFront, want)
}

func TestCurrentNativeOnlyBookKeepsAbsentOrdinaryBook(t *testing.T) {
	payload, newFront := spellbookCitySource(t)
	f := newFront()
	if _, town, err := f.RestoreOriginal(payload); err != nil || !town {
		t.Fatal(err)
	}
	extra := uint32(1 | 1<<29 | 1<<31)
	f.Carried[0].Book = sim.Spellbook{State: sim.BookNativeAbsent}
	f.Carried[0].KnownSpells = extra
	f.Carried[0].SpellbookRestored, f.Carried[0].SpellbookPresent = true, false
	for cycle := 0; cycle < 2; cycle++ {
		s, label, err := f.Snapshot(false)
		if err != nil {
			t.Fatal(err)
		}
		raw, err := f.ExportCurrentSave(s, label)
		if err != nil {
			t.Fatal(err)
		}
		doc, err := sav.DecodeDocumentData(raw)
		if err != nil {
			t.Fatal(err)
		}
		a, err := readCurrentActions(&doc)
		if err != nil || len(a.Bindings) == 0 {
			t.Fatal(err)
		}
		characters, err := sav.ReadDocumentCharacters(doc, []uint16{a.Bindings[0].Object})
		if err != nil || len(characters) != 1 || characters[0].Character.HasSpellbook || characters[0].Character.KnownSpells() != 0 {
			t.Fatal("native-only membership invented an ordinary book", err, characters)
		}
		f = newFront()
		if _, town, err := f.RestoreOriginal(raw); err != nil || !town {
			t.Fatal(err)
		}
		got := f.Carried[0]
		if got.Book != (sim.Spellbook{State: sim.BookNativeAbsent}) || got.KnownSpells != extra || !got.SpellbookRestored || got.SpellbookPresent {
			t.Fatal("absent ordinary book lost native-only membership", cycle, got.Book, got.KnownSpells)
		}
	}
}

func TestCurrentCityBookPolicyIsAtomicAndMissingAnchorKeepsOrdinaryBook(t *testing.T) {
	payload, newFront := spellbookCitySource(t)
	f := newFront()
	if _, town, err := f.RestoreOriginal(payload); err != nil || !town {
		t.Fatal(err)
	}
	f.Carried[0].Book, f.Carried[0].KnownSpells = sim.Spellbook{}, 2
	f.Carried[0].SpellbookRestored, f.Carried[0].SpellbookPresent = false, false
	s, label, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(s, label)
	if err != nil {
		t.Fatal(err)
	}
	for _, fault := range []string{"zero anchor", "present with anchor", "extra ordinary bit", "old missing anchor"} {
		doc, err := sav.DecodeDocumentData(raw)
		if err != nil {
			t.Fatal(err)
		}
		a, err := readCurrentActions(&doc)
		if err != nil || a.Party[0].City.LegacyBookWire == nil {
			t.Fatal("missing fixture book anchor", err)
		}
		switch fault {
		case "zero anchor":
			a.Party[0].City.LegacyBookWire = new([32]byte)
		case "present with anchor":
			a.Party[0].City.BookMode = uint8(sim.BookPresent)
		case "extra ordinary bit":
			a.Party[0].City.ExtraSpells = 2
		case "old missing anchor":
			a.Party[0].City.LegacyBookWire = nil
		}
		leaf, err := json.Marshal(a)
		if err != nil {
			t.Fatal(err)
		}
		if err := sav.SetNativeActions(&doc.State, leaf); err != nil {
			t.Fatal(err)
		}
		candidate, err := sav.EncodeDocumentData(doc)
		if err != nil {
			t.Fatal(err)
		}
		party, town := mapload.CloneParty(f.Carried), f.Town
		_, inTown, err := f.RestoreOriginal(candidate)
		if fault == "old missing anchor" {
			if err != nil || !inTown || f.Carried[0].Book.State != sim.BookPresent || f.Carried[0].KnownSpells != 2 || !f.Carried[0].SpellbookPresent {
				t.Fatal("unanchored metadata erased ordinary city book", err, f.Carried[0].Book)
			}
		} else if err == nil || f.Town != town || !reflect.DeepEqual(f.Carried, party) {
			t.Fatal("malformed book policy changed current city", fault, err)
		}
	}
}

func TestCurrentTemplateLegacyBookOrdinaryLiteralsWin(t *testing.T) {
	for _, extra := range []uint32{0, 1 | 1<<31} {
		table := eqDefsTable(t)
		member := mapload.PartyMember{ID: "native", Name: "unplaced", FigureFace: 3, FigureDir: "fighter", Hero: eqHero(),
			Profile: data.Profile{Fighter: true, HealthColumn: true}, Class: 41, StartingHero: true,
			Weapon: eqSword(t, table), KnownSpells: 2 | extra}
		w := emptyTemplateWorld(t)
		before := w.Hash()
		for cycle := 0; cycle < 2; cycle++ {
			p, err := captureCurrentPartyTemplate(90, member, member, table)
			if err != nil {
				t.Fatal(err)
			}
			policy, err := json.Marshal(p.Template.Policy)
			if err != nil {
				t.Fatal(err)
			}
			if cycle == 0 {
				r := &p.Template.Records.Objects[p.Template.Records.Actor-1]
				refs, present := savedObjectRefs(r, "Spells")
				if !present || len(refs) == 0 || refs[0] == 0 {
					t.Fatal("template lacks ordinary spell 1")
				}
				spell := refs[0]
				mustSetRefs(r, "Spells", []uint16{0, 0, spell})
				mustSetCount(r, "Spells", 4)
				for _, v := range []sav.DocumentValueData{{Name: "S08", Value: 3}, {Name: "S09", Value: 73}, {Name: "S0A", Value: 0x83}, {Name: "S0C", Value: 54321}} {
					mustSetValue(&p.Template.Records.Objects[spell-1], v.Name, v.Value)
				}
			}
			unchanged, _ := json.Marshal(p.Template.Policy)
			if !bytes.Equal(policy, unchanged) {
				t.Fatal("ordinary template edit changed mode policy")
			}
			raw, err := json.Marshal(p)
			if err != nil {
				t.Fatal(err)
			}
			var cold currentPartyMember
			if err := json.Unmarshal(raw, &cold); err != nil {
				t.Fatal(err)
			}
			member, err = cold.restoreFromCurrent(w, table)
			wantMode := sim.BookPresent
			if extra != 0 {
				wantMode = sim.BookNativePresent
			}
			if err != nil || member.Book.State != wantMode || member.KnownSpells != 8|extra || member.Book.Slots[2] != (sim.BookSpell{Range: 73, Defensive: 0x83, ManaCost: 54321}) || !member.SpellbookRestored || !member.SpellbookPresent {
				t.Fatal("template ordinary book lost to legacy mode", extra, cycle, err, member.Book, member.KnownSpells)
			}
		}
		if w.Hash() != before || len(w.Entities()) != 0 {
			t.Fatal("template book reconstruction admitted an actor")
		}
	}
}

func TestCurrentMissionLegacyBookOrdinaryLiteralsWin(t *testing.T) {
	for _, extra := range []uint32{0, 1 | 1<<29 | 1<<31} {
		f, s, source := currentPlayerSlotFixture(t, 1)
		entities := source.Entities()
		entities[0].KnownSpells = 2 | extra
		w, err := sim.NewWorld(31, source.Bounds(), sim.ModeCanonical, nil, entities)
		if err != nil {
			t.Fatal(err)
		}
		s.World, err = w.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		raw, err := f.ExportCurrentSave(s, "legacy mission book")
		if err != nil {
			t.Fatal(err)
		}
		doc, err := sav.DecodeDocumentData(raw)
		if err != nil {
			t.Fatal(err)
		}
		a, err := readCurrentActions(&doc)
		if err != nil || a.Values[0].LegacyBookWire == nil {
			t.Fatal("mission legacy book has no emitted value anchor", err)
		}
		var object uint16
		for _, binding := range a.Bindings {
			if binding.ID == 0 && !binding.Structure && !binding.Missing {
				object = binding.Object
			}
		}
		if object == 0 {
			t.Fatal("mission book has no exact actor")
		}
		leaf, _, _ := sav.NativeActions(doc.State)
		refs, present := savedObjectRefs(&doc.Objects[object-1], "Spells")
		if !present || len(refs) == 0 || refs[0] == 0 {
			t.Fatal("mission book has no ordinary spell 1")
		}
		for _, v := range []sav.DocumentValueData{{Name: "S09", Value: 73}, {Name: "S0A", Value: 0x83}, {Name: "S0C", Value: 54321}} {
			mustSetValue(&doc.Objects[refs[0]-1], v.Name, v.Value)
		}
		raw, err = sav.EncodeDocumentData(doc)
		if err != nil {
			t.Fatal(err)
		}
		back, _ := sav.DecodeDocumentData(raw)
		unchanged, _, _ := sav.NativeActions(back.State)
		if !bytes.Equal(leaf, unchanged) {
			t.Fatal("mission ordinary book edit changed supplement")
		}
		for cycle := 0; cycle < 2; cycle++ {
			cold := cellStateFront(t)
			cold.Campaign, cold.Table = f.Campaign, f.Table
			open, town, err := cold.RestoreOriginal(raw)
			if err == nil && !town {
				err = cold.App("mission literal book").OpenMission(open)
			}
			if err != nil || town {
				t.Fatal(extra, cycle, err)
			}
			got := cold.live.world.Entities()[0]
			wantMode := sim.BookPresent
			if extra != 0 {
				wantMode = sim.BookNativePresent
			}
			if got.ID != 0 || got.KnownSpells != 2|extra || got.Book.State != wantMode || got.Book.Slots[0] != (sim.BookSpell{Range: 73, Defensive: 0x83, ManaCost: 54321}) {
				t.Fatal("mission ordinary values lost to legacy policy", extra, cycle, got.ID, got.Book, got.KnownSpells)
			}
			if cycle == 0 {
				next, label, err := cold.Snapshot(true)
				if err != nil {
					t.Fatal(err)
				}
				raw, err = cold.ExportCurrentSave(next, label)
				if err != nil {
					t.Fatal(err)
				}
			}
		}
	}
}
