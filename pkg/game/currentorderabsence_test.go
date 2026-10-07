package game

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func currentBookOrderFront(t *testing.T, raw []byte) *FrontEnd {
	t.Helper()
	f := withCurrentMenuDefinitions(t, nonPartyBookFront1105(t))
	open, town, err := f.RestoreOriginal(raw)
	if err != nil || town {
		t.Fatal("book order LOAD", town, err)
	}
	if err := f.App("current book order").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	return f
}

func currentBookOrderSource(t *testing.T) (*FrontEnd, sim.EntityID) {
	t.Helper()
	a := actorBookFixture(91, &poolFixtureSpell{id: 1, rangeByte: 7, cost: 3})
	b := actorBookFixture(92, &poolFixtureSpell{id: 1, rangeByte: 9, defensive: 2, cost: 65535})
	a.cell, b.cell, b.human = 0x0c0c, 0x0c0f, true
	f := currentBookOrderFront(t, savedContainer(poolFixtureBody([]*poolFixturePlayer{{groups: [][]*poolFixtureActor{{a}}}, {groups: [][]*poolFixtureActor{{b}}}}, nil)))
	caster, target := poolEntity(t, f.live.world, 91), poolEntity(t, f.live.world, 92)
	sim.Step(f.live.world, []sim.Command{sim.Cast(caster.ID, target.ID, 1)})
	return f, caster.ID
}

func currentBookOrderValue(t *testing.T, w *sim.World, caster sim.EntityID) uint32 {
	t.Helper()
	_, orders, _ := w.SavedGroups()
	for _, order := range orders {
		if order.Entity == caster {
			return binary.LittleEndian.Uint32(order.Raw[0x30:])
		}
	}
	t.Fatal("current caster has no Order")
	return 0
}

func TestCurrentOrderSpellAbsenceYieldsToOrdinaryFields(t *testing.T) {
	for _, edit := range []string{"none", "pointer", "spell-key", "slot"} {
		t.Run(edit, func(t *testing.T) {
			f, caster := currentBookOrderSource(t)
			if currentBookOrderValue(t, f.live.world, caster) != 0 {
				t.Fatal("control requires a book without a native Spell pointer")
			}
			doc, a := currentRootSAVDocument(t, f)
			target := poolEntity(t, f.live.world, 92).ID
			var actor, other uint16
			for _, row := range a.Bindings {
				if !row.Structure && !row.Missing {
					if row.ID == caster {
						actor = row.Object
					} else if row.ID == target {
						other = row.Object
					}
				}
			}
			refs, _ := savedObjectRefs(&doc.Objects[actor-1], "Spells")
			wire, err := savedStructureValue(&doc.Objects[refs[0]-1], "This")
			if err != nil || wire == 0 {
				t.Fatal("ordinary Spell key", wire, err)
			}
			leaf, _, _ := sav.NativeActions(doc.State)
			rawOrder, _ := savedMotionRaw(&doc.Objects[actor-1], "U158", 148)
			want := wire
			switch edit {
			case "none":
				want = 0
			case "pointer":
				want = wire ^ 0x5f00
				binary.LittleEndian.PutUint32(rawOrder[0x30:], want)
			case "spell-key":
				mustSetValue(&doc.Objects[refs[0]-1], "This", wire^0x7f00)
			case "slot":
				otherRefs, _ := savedObjectRefs(&doc.Objects[other-1], "Spells")
				savedObjectSetRefs(&doc.Objects[actor-1], "Spells", otherRefs, false)
				savedObjectSetRefs(&doc.Objects[other-1], "Spells", refs, false)
			}
			unchanged, _, _ := sav.NativeActions(doc.State)
			if !bytes.Equal(leaf, unchanged) {
				t.Fatal("ordinary edit changed the absence leaf")
			}
			doc, _, err = sav.ReindexDocumentData(doc)
			if err != nil {
				t.Fatal(err)
			}
			raw, err := sav.EncodeDocumentData(doc)
			if err != nil {
				t.Fatal(err)
			}
			cold := currentBookOrderFront(t, raw)
			if got := currentBookOrderValue(t, cold.live.world, caster); got != want {
				t.Fatalf("Order Spell key %x, want %x after %s", got, want, edit)
			}
			if edit == "none" {
				if f.live.world.Hash() != cold.live.world.Hash() {
					t.Fatal("absence LOAD changed exact World")
				}
				for range 2 {
					next, _ := currentRootSAVDocument(t, cold)
					encoded, err := sav.EncodeDocumentData(next)
					if err != nil {
						t.Fatal(err)
					}
					fresh := currentBookOrderFront(t, encoded)
					if cold.live.world.Hash() != fresh.live.world.Hash() {
						t.Fatal("second cycle changed exact current World")
					}
					cold = fresh
				}
			}
		})
	}
}

func TestCurrentOrderSpellAbsenceMalformedIsAtomic(t *testing.T) {
	f, _ := currentBookOrderSource(t)
	doc, a := currentRootSAVDocument(t, f)
	found := false
	for i := range a.Actions.Actors {
		if o := a.Actions.Actors[i].Order; o != nil && o.AbsentSpellKey != nil {
			*o.AbsentSpellKey = 0
			found = true
		}
	}
	if !found {
		t.Fatal("missing anchored native absence")
	}
	leaf, _ := json.Marshal(a)
	if err := sav.SetNativeActions(&doc.State, leaf); err != nil {
		t.Fatal(err)
	}
	raw, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	hash := f.live.world.Hash()
	if _, _, err := f.RestoreOriginal(raw); err == nil || f.live.world.Hash() != hash {
		t.Fatal("malformed absence accepted or changed the live World", err)
	}
}
