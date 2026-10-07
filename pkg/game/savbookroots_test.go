package game

import (
	"bytes"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func TestCurrentBookOnlyRootSurvivesGraphRetirementAndColdLoads(t *testing.T) {
	for _, edit := range []bool{false, true} {
		t.Run(map[bool]string{false: "unchanged", true: "ordinary-values"}[edit], func(t *testing.T) {
			f, _ := seededCityMission(t)
			w := f.live.world
			registry := w.SavedObjects()
			actor := registry.BookRoots[0].Entity
			var current sim.Entity
			for _, e := range w.Entities() {
				if e.ID == actor {
					current = e
				}
			}
			current.KnownSpells |= 1 << 7
			current.Book.Slots[6] = sim.BookSpell{Range: 29, Defensive: 1, ManaCost: 456}
			if err := w.ImportOriginalActorSpellbooks([]sim.OriginalActorSpellbook{{ID: actor, KnownSpells: current.KnownSpells, Book: current.Book}}); err != nil {
				t.Fatal(err)
			}
			id := registry.NextID
			registry.NextID++
			registry.BookRoots[0].Slots[6] = id
			registry.Spells = append(registry.Spells, sim.SavedSpellObject{ID: id, Origin: sim.SavedObjectOrigin{Kind: sim.SavedObjectGenerated}, Value: sim.SourceItemSpell{Present: true, ID: 7, Range: 29, Defensive: 1, ManaCost: 456}})
			identities := map[sim.SavedObjectID]sim.SavedObjectID{}
			for _, item := range registry.Items {
				identities[item.ID] = item.ID
				if item.Spell == id {
					t.Fatal("book-only witness has an Item edge")
				}
			}
			if err := w.ReplaceCurrentObjects(registry, identities, nil); err != nil {
				t.Fatal(err)
			}
			wantRange, wantMana := uint32(29), uint32(456)
			for cycle := range 2 {
				before := f.live.world.Hash()
				snapshot, label, err := f.Snapshot(true)
				if err != nil {
					t.Fatal(err)
				}
				raw, err := f.ExportCurrentSave(snapshot, label)
				if err != nil {
					t.Fatal("book-only current SAVE", cycle, err)
				}
				if f.live.world.Hash() != before {
					t.Fatal("SAVE changed current World")
				}
				doc, err := sav.DecodeDocumentData(raw)
				if err != nil {
					t.Fatal(err)
				}
				a, err := readCurrentActions(&doc)
				if err != nil || a == nil {
					t.Fatal("current policy", err)
				}
				var object, owner uint16
				for _, row := range a.Ownership {
					if row.ID == id {
						object = row.Object
					}
				}
				for _, row := range a.Bindings {
					if row.ID == actor && !row.Structure && !row.Missing {
						owner = row.Object
					}
				}
				if object == 0 || owner == 0 {
					t.Fatal("book-only Spell retired before owner attachment", id)
				}
				refs, _ := savedObjectRefs(&doc.Objects[owner-1], "Spells")
				if len(refs) <= 6 || refs[6] != object {
					t.Fatal("current slot does not name its exact Spell", refs)
				}
				if edit && cycle == 0 {
					wantRange, wantMana = 73, 54321
					leaf, present, err := sav.NativeActions(doc.State)
					if err != nil || !present {
						t.Fatal(err)
					}
					savedObjectSetValue(&doc.Objects[object-1], "S09", wantRange)
					savedObjectSetValue(&doc.Objects[object-1], "S0C", wantMana)
					raw, err = sav.EncodeDocumentData(doc)
					after, retained, leafErr := sav.NativeActions(doc.State)
					if err != nil || leafErr != nil || !retained || !bytes.Equal(leaf, after) {
						t.Fatal("ordinary edit changed policy", err, leafErr)
					}
				}
				cold := cellStateFront(t)
				cold.Campaign, cold.Table, cold.Humans = f.Campaign, f.Table, f.Humans
				open, town, err := cold.RestoreOriginal(raw)
				if err == nil && !town {
					err = cold.App("book-only current LOAD").OpenMission(open)
				}
				if err != nil || town {
					t.Fatal("book-only cold LOAD", cycle, town, err)
				}
				if !edit || cycle != 0 {
					if cold.live.world.Hash() != before {
						t.Fatalf("book-only cycle %d full World %x -> %x", cycle, before, cold.live.world.Hash())
					}
					for range 20 {
						f.live.tick()
						cold.live.tick()
						if f.live.world.Hash() != cold.live.world.Hash() {
							t.Fatal("book-only next tick differs")
						}
					}
				}
				got := cold.live.world.SavedObjects()
				if got.BookRoots[0].Slots[6] != id {
					t.Fatal("cold LOAD changed native Spell identity")
				}
				for _, e := range cold.live.world.Entities() {
					if e.ID == actor && (uint32(e.Book.Slots[6].Range) != wantRange || uint32(e.Book.Slots[6].ManaCost) != wantMana) {
						t.Fatal("ordinary Book values were overwritten", e.Book.Slots[6])
					}
				}
				f = cold
			}
		})
	}
}

func TestOriginalBookRootsAccountForSharedSpellReferences(t *testing.T) {
	f := itemObjectsOpen(t, true, func(doc *sav.DocumentData) {
		var spell uint16
		for i, row := range doc.Objects {
			if row.Class == "Spell" {
				spell = uint16(i + 1)
			}
		}
		if spell == 0 {
			t.Fatal("literal Spell is absent")
		}
		for i := range doc.Objects {
			r := &doc.Objects[i]
			if r.Class != "Unit" && r.Class != "Human" && r.Class != "Humanoid" {
				continue
			}
			savedObjectSetValue(r, "HasSpellbook", 1)
			savedObjectSetValue(r, "SpellsHeader", 0)
			savedObjectSetValue(r, "U144", 1<<7)
			savedObjectSetRefs(r, "Spells", []uint16{0, 0, 0, 0, 0, 0, spell}, true)
			mustSetCount(r, "Spells", 8)
		}
	})
	w, state := f.live.world, f.live.mission.state.savedDocument
	r := w.SavedObjects()
	if len(r.BookRoots) != 3 || len(r.Spells) != 1 || r.Spells[0].ExternalReferences != 0 {
		t.Fatal("modeled book/weapon edges were counted as external", r.BookRoots, r.Spells)
	}
	for _, root := range r.BookRoots {
		if root.Slots[6] != r.Spells[0].ID {
			t.Fatal("literal shared Spell was split")
		}
	}
	if err := validateSavedItemBindingWorld(state, w); err != nil {
		t.Fatal("exact original Book/weapon graph", err)
	}
	before := w.Hash()
	copy, err := cloneSavedDocument(state)
	if err != nil {
		t.Fatal(err)
	}
	for _, actor := range copy.Actors {
		refs, present := savedObjectRefs(&copy.Document.Objects[actor.ObjectIndex-1], "Spells")
		if present && len(refs) > 6 && refs[6] != 0 {
			refs[6] = 0
			break
		}
	}
	if err := validateSavedItemBindingWorld(copy, w); err == nil {
		t.Fatal("changed ordinary Book edge was hidden by external-reference accounting")
	}
	if w.Hash() != before || !reflect.DeepEqual(w.SavedObjects(), r) {
		t.Fatal("validation changed current registry")
	}
}
