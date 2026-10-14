package game

import (
	"os"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// TestReleasePlainMapItemPickedUpAfterAnEquipKeepsItsOwnSavedRecord removes and
// re-wears the hero's weapon in mission 50 entered from the city, picks up a
// plain map item, SAVEs and cold-LOADs. The equip must leave the constructor
// source of a later pickup alone: the pack cell stays the map item, and its
// Item record keeps the constructor's shape, material and magic capacity and
// the map item's own Identity.
func TestReleasePlainMapItemPickedUpAfterAnEquipKeepsItsOwnSavedRecord(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	prepareAcceptedCampaignMission(t, f, 50)
	if err := f.App("plain").OpenMission(f.MissionOpenerWith(50, f.NextParty())); err != nil {
		t.Fatalf("open mission 50: %v", err)
	}
	hero := equipmentReturnHero(t, f)
	w := f.live.world
	f.live.switchInventorySubject(uint32(hero))
	f.live.enqueueUnequip(0)
	for range 4 {
		f.live.tick()
	}
	stacks, _ := w.CarriedStacks(hero)
	worn := false
	for i, st := range stacks {
		if st.Code == 0x110f {
			f.live.enqueueEquip(i)
			worn = true
			break
		}
	}
	if !worn {
		t.Fatal("the removed weapon 0x110f is not in the pack")
	}
	for range 4 {
		f.live.tick()
	}
	const code = 0x2934
	var at *sim.CellPoint
	for _, sk := range w.Sacks() {
		if sk.X == 72 && sk.Y == 78 {
			for _, it := range sk.ItemInstances {
				if it.Code == code && it.ObjectID == 0 && !it.WeightPresent && it.SourceEquipment.Class == 0 {
					at = &sim.CellPoint{X: sk.X, Y: sk.Y}
				}
			}
		}
	}
	if at == nil {
		t.Fatalf("no plain %#x lies at (72,78)", code)
	}
	packJoinTake(t, f, hero, *at)
	path := saveCorpseMission(t, f, t.TempDir())
	cold := loadAreaContinuation(t, path)
	coldStacks, _ := cold.live.world.CarriedStacks(equipmentReturnHero(t, cold))
	found := false
	for _, st := range coldStacks {
		if st.Code != code {
			continue
		}
		found = true
		it := st.Instance()
		if it.ObjectID != 0 || it.WeightPresent || it.SourceEquipment.Class != 0 {
			t.Errorf("cold pack cell %#x = obj %d weight present %v class %d, want the map item (0, false, 0)", it.Code, it.ObjectID, it.WeightPresent, it.SourceEquipment.Class)
		}
	}
	if !found {
		t.Fatalf("the cold pack holds no %#x", code)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	records := 0
	for i := range doc.Objects {
		r := &doc.Objects[i]
		if c, err := savedStructureValue(r, "F40"); err != nil || c != code || !savedItemClass(r.Class) {
			continue
		}
		records++
		for name, want := range map[string]uint32{"F45": 1, "F46": 2, "F48": 2, "Identity": 0x1000048} {
			if got, err := savedStructureValue(r, name); err != nil || got != want {
				t.Errorf("record %d %s = %d (%v), want %d", i+1, name, got, err, want)
			}
		}
	}
	if records != 1 {
		t.Fatalf("the save holds %d Item records of %#x, want 1", records, code)
	}
}
