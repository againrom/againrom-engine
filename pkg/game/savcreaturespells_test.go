package game

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func TestCurrentCreatureSlotsReplaceEveryLoadedWord(t *testing.T) {
	for field := 0; field < 6; field++ {
		for _, value := range []uint32{0, 19} {
			t.Run(fmt.Sprintf("word%d/value%d", field, value), func(t *testing.T) {
				f := castOrderFront(t)
				raw, _, _ := saveCurrentEffect(t, f)
				f = openCurrentEffectSave(t, f, raw)
				w := f.live.world
				var slots [sim.CreatureSpellSlots]sim.CreatureSpell
				if field < 3 {
					slots[field].ID = value
				} else {
					slots[field-3].Threshold = value
				}
				e := w.Entities()[len(w.Entities())-1]
				if err := w.ImportOriginalActorSpellbooks([]sim.OriginalActorSpellbook{{ID: e.ID,
					Book: sim.Spellbook{State: sim.BookAbsent}, HasCreatureSpells: true, CreatureSpells: slots}}); err != nil {
					t.Fatal(err)
				}
				loaded := bytes.Repeat([]byte{0x5a}, 148)
				clear(loaded[0x78:0x90])
				binary.LittleEndian.PutUint32(loaded[0x78+field*4:], 0x5a5a5a5a)
				doc := sav.DocumentData{Objects: []sav.DocumentRecordData{{Class: "Unit",
					Raw: []sav.DocumentRawData{{Name: "U158", Bytes: append([]byte(nil), loaded...)}}}}}
				state := &SnapshotSAVDocument{Actors: []SnapshotSAVActor{{EntityID: e.ID, ObjectIndex: 1}}}
				before := w.Hash()
				if err := projectCurrentCreatureSpells(&doc, state, w); err != nil {
					t.Fatal(err)
				}
				got := doc.Objects[0].Raw[0].Bytes
				want := append([]byte(nil), loaded...)
				clear(want[0x78:0x90])
				binary.LittleEndian.PutUint32(want[0x78+field*4:], value)
				if !bytes.Equal(got, want) {
					t.Fatalf("current slots = %x, want %x; only word %d may be nonzero", got[0x78:0x90], want[0x78:0x90], field)
				}
				if w.Hash() != before {
					t.Fatal("writer changed the World")
				}
			})
		}
	}
}

func TestCurrentCreatureSlotWordsSurviveSaveAndColdLoad(t *testing.T) {
	for field := 0; field < 6; field++ {
		t.Run(fmt.Sprintf("word%d", field), func(t *testing.T) {
			f := castOrderFront(t)
			raw, _, _ := saveCurrentEffect(t, f)
			f = openCurrentEffectSave(t, f, raw)
			w := f.live.world
			e := w.Entities()[len(w.Entities())-1]
			var slots [sim.CreatureSpellSlots]sim.CreatureSpell
			if field < 3 {
				slots[field].ID = 3
			} else {
				slots[field-3].Threshold = 197
			}
			setCurrentCreatureSlotWords(t, w, e, slots)
			raw, doc, actions := saveCurrentEffect(t, f)
			order := savedRecordRawForTest(t, castOrderRecord(t, doc, actions, e.ID), "U158")
			var want [24]byte
			value := uint32(3)
			if field >= 3 {
				value = 197
			}
			binary.LittleEndian.PutUint32(want[field*4:], value)
			if !bytes.Equal(order[0x78:0x90], want[:]) {
				t.Fatalf("saved slot words %x, want %x", order[0x78:0x90], want)
			}
			cold := openCurrentEffectSave(t, f, raw)
			assertCurrentWorldEqual(t, w, cold.live.world, "changed creature slots cold LOAD")
			_, again, next := saveCurrentEffect(t, cold)
			if got := savedRecordRawForTest(t, castOrderRecord(t, again, next, e.ID), "U158"); !bytes.Equal(got[0x78:0x90], want[:]) {
				t.Fatal("second SAVE lost changed slots")
			}
			for tick := 0; tick < 3; tick++ {
				sim.Step(w, nil)
				sim.Step(cold.live.world, nil)
				assertCurrentWorldEqual(t, w, cold.live.world, "changed creature slots next tick")
			}
		})
	}
}

func setCurrentCreatureSlotWords(t *testing.T, w *sim.World, actor sim.Entity, slots [sim.CreatureSpellSlots]sim.CreatureSpell) {
	t.Helper()
	book := actor.Book
	if book.State != sim.BookPresent && book.State != sim.BookAbsent {
		book.State = sim.BookAbsent
		if actor.Book.WirePresent(actor.KnownSpells) {
			book.State = sim.BookPresent
		}
	}
	if err := w.ImportOriginalActorSpellbooks([]sim.OriginalActorSpellbook{{ID: actor.ID,
		Book: book, KnownSpells: actor.KnownSpells, HasCreatureSpells: true, CreatureSpells: slots}}); err != nil {
		t.Fatal(err)
	}
	groups, orders, present := w.SavedGroups()
	if !present {
		return
	}
	for i := range orders {
		if orders[i].Entity != actor.ID {
			continue
		}
		for k, slot := range slots {
			binary.LittleEndian.PutUint32(orders[i].Raw[0x78+4*k:], slot.ID)
			binary.LittleEndian.PutUint32(orders[i].Raw[0x84+4*k:], slot.Threshold)
		}
	}
	if err := w.ImportSavedGroups(groups, orders); err != nil {
		t.Fatal(err)
	}
}
