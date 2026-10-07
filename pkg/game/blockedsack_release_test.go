package game

import (
	"fmt"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func TestReleaseDropOntoGroundBlockedCellSAVRoundTrip(t *testing.T) {
	_, raw := groundCorpusFile(t, "2026-08-24/game0021.sav", "7acf1d56d3a98e388a527ae386f225036cc01551273480a4fafc52fa8fcf817c")
	f := releaseFront(t)
	app, _ := openOriginalSAVApp(t, f, raw, "blocked-drop.sav")
	planes, ok := f.live.world.SavedCellPlanes()
	if !ok {
		t.Fatal("no current cell planes")
	}
	hero := f.live.mission.ids[0]
	walk := func(at sim.CellPoint) {
		f.live.pending = append(f.live.pending, sim.MoveTo(hero, at))
		for range 600 {
			f.live.tick()
			if e, _ := f.live.entity(hero); e.X == at.X && e.Y == at.Y && !e.HasTarget && sackPickupArrived(f.live.world, e) {
				return
			}
		}
		t.Fatal("hero did not reach", at)
	}
	stand, tree, tree2, open := sim.CellPoint{X: 22, Y: 62}, sim.CellPoint{X: 22, Y: 61}, sim.CellPoint{X: 21, Y: 61}, sim.CellPoint{X: 23, Y: 63}
	for _, c := range []sim.CellPoint{stand, tree, tree2, open} {
		blocked := planes.Dynamic[c.Y*256+c.X]&1 != 0
		if blocked != (c == tree || c == tree2) || groundAt(f.live.world.Sacks(), c.X, c.Y) != nil {
			t.Fatalf("fixture cell %v: Dynamic %#x", c, planes.Dynamic[c.Y*256+c.X])
		}
	}
	const weapon = 4358
	walk(sim.CellPoint{X: 20, Y: 65})
	f.live.takeSackFor(hero)
	walk(stand)
	carried := func(code uint16) (sim.ItemSlot, bool) {
		pack, _ := f.live.world.CarriedStacks(hero)
		for i, st := range pack {
			if st.Code == code || code == 0 && st.Code != weapon && data.ItemCode(st.Code) != data.QuestDocumentCode {
				return sim.ItemSlot(i), true
			}
		}
		return 0, false
	}
	drop := func(c sim.Command) {
		f.live.pending = append(f.live.pending, c)
		f.live.tick()
	}
	slot, ok := carried(weapon)
	if !ok {
		t.Fatal("the hero did not pick up the Weapon from the Sack at 20,65")
	}
	drop(sim.DropCarried(hero, slot, tree))
	worn, _ := f.live.world.EquippedItems(hero)
	var wornCode uint16
	for i, item := range worn {
		if !item.Empty() {
			wornCode = item.Code
			drop(sim.DropWorn(hero, sim.EquipSlot(i+1), tree2))
			break
		}
	}
	if wornCode == 0 {
		t.Fatal("the hero wears nothing")
	}
	if slot, ok := carried(0); ok {
		drop(sim.DropCarried(hero, slot, open))
	} else {
		worn, _ = f.live.world.EquippedItems(hero)
		for i, item := range worn {
			if !item.Empty() {
				drop(sim.DropWorn(hero, sim.EquipSlot(i+1), open))
				break
			}
		}
	}
	at := groundAt(f.live.world.Sacks(), stand.X, stand.Y)
	if at == nil || len(at.Items) != 2 || at.Items[0] != weapon || at.Items[1] != wornCode {
		t.Fatalf("the dropper's cell holds %s, want the Weapon and the worn %04x", blockedSackSeam(f, stand), wornCode)
	}
	if blockedSackSeam(f, tree) != "none" || blockedSackSeam(f, tree2) != "none" || blockedSackSeam(f, open) == "none" {
		t.Fatalf("trees %s | %s, open cell %s", blockedSackSeam(f, tree), blockedSackSeam(f, tree2), blockedSackSeam(f, open))
	}
	cells := []sim.CellPoint{stand, open, tree, tree2}
	seam := func(f *FrontEnd) string {
		out := ""
		for _, c := range cells {
			out += fmt.Sprintf("%v:%s; ", c, blockedSackSeam(f, c))
		}
		return out
	}
	want := seam(f)
	store, name, written := menuSAVE(t, f, app, OriginalStore{})
	doc, err := sav.DecodeDocumentData(written)
	if err != nil {
		t.Fatal(err)
	}
	var sackKey uint32
	for _, c := range doc.World.Cells {
		switch c.Cell {
		case uint16(tree.Y)<<8 | uint16(tree.X), uint16(tree2.Y)<<8 | uint16(tree2.X):
			if c.Sack != 0 {
				t.Fatalf("written tree cell node holds a Sack: %+v", c)
			}
		case uint16(stand.Y)<<8 | uint16(stand.X):
			sackKey = c.Sack
		}
	}
	if sackKey == 0 {
		t.Fatal("no written cell node for the dropper's Sack")
	}
	fresh := loadLocalLegacySave(t, store, name)
	for tick := 0; ; tick++ {
		if got := seam(fresh); got != seam(f) || got != want {
			t.Fatalf("tick%d restored Sacks differ:\n%s\n%s", tick, seam(f), got)
		}
		if tick == 16 {
			break
		}
		f.live.tick()
		fresh.live.tick()
	}
	lost := loadAlteredSAV(t, written, func(doc *sav.DocumentData) bool {
		for i := range doc.Objects {
			if id, _ := savedStructureValue(&doc.Objects[i], "Identity"); id == sackKey && doc.Objects[i].Class == "Sack" {
				refs, _ := savedObjectRefs(&doc.Objects[i], "Contents")
				if len(refs) != 0 {
					return savedStructureSetValue(&doc.Objects[refs[0]-1], "F42", 2) == nil
				}
			}
		}
		return false
	})
	if seam(lost) == want {
		t.Fatal("loss control: a SAV with the Sack's item count altered still matches")
	}
	t.Logf("drops onto tree cells %v %v land on %v, one onto %v stays there; menu SAVE %s, LOAD and 16 ticks keep %s", tree, tree2, stand, open, name, want)
}

func blockedSackSeam(f *FrontEnd, at sim.CellPoint) string {
	s := groundAt(f.live.world.Sacks(), at.X, at.Y)
	if s == nil {
		return "none"
	}
	out := fmt.Sprintf("gold%d items%v", s.Gold, s.Items)
	for _, item := range s.ItemInstances {
		out += fmt.Sprintf(" %04x/%d/%t", item.Code, item.Weight, item.WeightPresent)
	}
	return out
}
