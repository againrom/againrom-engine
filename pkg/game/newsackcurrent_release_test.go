package game

import (
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestReleaseEngineSAVNewGroundSackSurvivesSaveLoad(t *testing.T) {
	f := releaseFront(t)
	party := f.ChargenParty(ui.ChargenResult{Name: "Ground save", Choices: []int{1, 0, 3}, Stats: []int{31, 27, 24, 29}})
	app := f.App("new ground Sack from engine SAV")
	if err := app.OpenMission(f.MissionOpenerWith(20, party)); err != nil {
		t.Fatal(err)
	}
	app.Layout(1024, 768)
	store := SaveStore{Dir: t.TempDir()}
	save, _, _ := f.SaveSeams(store, OriginalStore{}, nil)
	firstName, err := save(true)
	if err != nil {
		t.Fatal("first engine SAV", err)
	}
	raw, err := store.Read(firstName)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	loaded, _ := loadSAVWindow(t, store, firstName)
	if _, present := loaded.live.world.SavedCellPlanes(); present {
		t.Fatal("engine-written SAV unexpectedly restored a current cell-plane carrier")
	}
	listed := make(map[uint16]bool, len(doc.World.Cells))
	for _, cell := range doc.World.Cells {
		listed[cell.Cell] = true
	}
	w := loaded.live.world
	bounds := w.Bounds()
	terrain := w.CurrentPolicy().Terrain
	var target sim.CellPoint
	var drop sim.Command
	var item uint16
	found := false
	for _, id := range loaded.live.mission.ids {
		actor, ok := loaded.live.entity(id)
		if !ok {
			continue
		}
		pack, _ := w.CarriedStacks(id)
		worn, _ := w.EquippedItems(id)
		for dy := int32(-2); dy <= 2 && !found; dy++ {
			for dx := int32(-2); dx <= 2; dx++ {
				x, y := actor.X+dx, actor.Y+dy
				if x < 0 || y < 0 || x >= bounds.Width || y >= bounds.Height || x >= 256 || y >= 256 {
					continue
				}
				cell := uint16(x) | uint16(y)<<8
				if listed[cell] || terrain.Block[int(y*bounds.Width+x)]&1 != 0 || groundAt(w.Sacks(), x, y) != nil {
					continue
				}
				target = sim.CellPoint{X: x, Y: y}
				if len(pack) != 0 {
					item, drop = pack[0].Code, sim.DropCarried(id, 0, target)
				} else {
					for slot, value := range worn {
						if !value.Empty() {
							item, drop = value.Code, sim.DropWorn(id, sim.EquipSlot(slot+1), target)
							break
						}
					}
				}
				found = item != 0
				if found {
					break
				}
			}
		}
		if found {
			break
		}
	}
	if !found {
		t.Fatal("engine SAV has no equipped or carried item with an open unlisted nearby cell")
	}
	loaded.live.pending = append(loaded.live.pending, drop)
	loaded.live.tick()
	ground := groundAt(w.Sacks(), target.X, target.Y)
	if ground == nil || len(ground.Items) != 1 || ground.Items[0] != item {
		t.Fatalf("ordinary drop did not create the expected Sack at %+v: %+v", target, ground)
	}
	secondSave, _, _ := loaded.SaveSeams(store, OriginalStore{}, nil)
	secondName, err := secondSave(true)
	if err != nil {
		t.Fatal("SAVE after a drop in an engine-written SAV", err)
	}
	written, err := store.Read(secondName)
	if err != nil {
		t.Fatal(err)
	}
	writtenDoc, err := sav.DecodeDocumentData(written)
	if err != nil {
		t.Fatal(err)
	}
	targetCell := uint16(target.X) | uint16(target.Y)<<8
	at := int(target.Y*bounds.Width + target.X)
	var sourceStatic byte
	for _, block := range doc.World.Blocks {
		if block.Cell == targetCell {
			sourceStatic = block.Static
			break
		}
	}
	static := sourceStatic&^7 | terrain.Block[at]&3 | (terrain.Block[at]&8)>>1
	cellSeen, blockSeen := false, false
	for _, cell := range writtenDoc.World.Cells {
		if cell.Cell == targetCell {
			cellSeen = true
			if cell.Cost != terrain.Cost[at] || cell.Static != static || cell.Sack == 0 {
				t.Fatalf("new Sack Cell is not current: %+v, cost %d static %#x", cell, terrain.Cost[at], static)
			}
		}
	}
	for _, block := range writtenDoc.World.Blocks {
		if block.Cell == targetCell {
			blockSeen = true
			if block.Static != static|0x20 || block.Dyn != static|0x20 {
				t.Fatalf("new Sack Block lacks recomputed planes: %+v, static %#x", block, static)
			}
		}
	}
	if !cellSeen || !blockSeen {
		t.Fatal("new Sack has no Cell or Block record", cellSeen, blockSeen)
	}
	result, _ := loadSAVWindow(t, store, secondName)
	restored := groundAt(result.live.world.Sacks(), target.X, target.Y)
	if restored == nil || len(restored.Items) != 1 || restored.Items[0] != item || restored.Gold != ground.Gold {
		t.Fatalf("LOAD lost the new Sack at %+v: got %+v, want %+v", target, restored, ground)
	}
}
