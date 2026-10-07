package game

import (
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func TestCurrentManaReserveKeepsDistinctBasisAndOrdinaryOwnerEdit(t *testing.T) {
	for _, edited := range []bool{false, true} {
		f := currentArchiveFixture(t)
		var actor sim.Entity
		for _, e := range f.live.world.Entities() {
			if e.ActorLoad.Source.HasOwner {
				actor = e
				break
			}
		}
		if !actor.ActorLoad.Source.HasOwner || !f.live.world.ImportAutoHealing(actor.Owner, 77) {
			t.Fatal("fixture lacks current actor owner")
		}
		wanted := actor.ActorLoad.Source.ManaReservePercent
		for cycle := 0; cycle < 2; cycle++ {
			s, _, err := f.Snapshot(true)
			if err != nil {
				t.Fatal(err)
			}
			raw, err := f.ExportCurrentSave(s, "current owner mana reserve")
			if err != nil {
				t.Fatal(err)
			}
			doc, err := sav.DecodeDocumentData(raw)
			if err != nil {
				t.Fatal(err)
			}
			a, err := readCurrentActions(&doc)
			if err != nil {
				t.Fatal(err)
			}
			if cycle == 0 && a.Values[actor.ID].ManaReserve == nil {
				t.Fatal("distinct current actor basis lacks its exact owner anchor")
			}
			if edited && cycle == 0 {
				key := a.Values[actor.ID].ManaReserve.OwnerKey
				for i := range doc.Objects {
					r := &doc.Objects[i]
					if r.Class == "Player" {
						current, _ := savedStructureValue(r, "This")
						if current == key {
							mustSetValue(r, "F58", 83)
						}
					}
				}
				wanted = 83
			}
			raw, err = sav.EncodeDocumentData(doc)
			if err != nil {
				t.Fatal(err)
			}
			f = openCurrentArchive(t, raw)
			found := false
			for _, e := range f.live.world.Entities() {
				if e.ID == actor.ID {
					found = true
					if e.ActorLoad.Source.ManaReservePercent != wanted {
						t.Fatal("ordinary owner edit or native basis lost", edited, cycle, e.ActorLoad.Source.ManaReservePercent, wanted)
					}
				}
			}
			if !found {
				t.Fatal("actor identity lost")
			}
		}
	}
}
