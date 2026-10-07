package game

import (
	"encoding/binary"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func TestCurrentTerminalLootRelocationKeepsSAVOwnersAndNextTicks(t *testing.T) {
	f, _ := itemMutationOpen(t, 1, false)
	f.live.tick()
	actor := newGroupActors(t, f.live.world)[newGroupA]
	cell := uint16(actor.Y)<<8 | uint16(actor.X)
	doc, _ := currentRootSAVDocument(t, f)
	found := false
	for i := range doc.World.Blocks {
		b := &doc.World.Blocks[i]
		if b.Cell == cell {
			b.Static, b.Dyn = b.Static|1, b.Dyn|0x21
			found = true
		}
	}
	if !found {
		doc.World.Blocks = append(doc.World.Blocks, sav.BlockRecord{Cell: cell, Static: 0x21, Dyn: 0x21})
	}
	front := func(t *testing.T) *FrontEnd { return itemMutationFront(t, 1) }
	f = loadCurrentRootSAV(t, doc, front)
	planes, _ := f.live.world.SavedCellPlanes()
	if planes == nil || planes.Static[cell]&1 == 0 || planes.Dynamic[cell]&1 == 0 {
		t.Fatal("independent blocked death cell did not reach the game")
	}
	var weapon sim.SavedObjectID
	for _, item := range f.live.world.SavedObjects().Items {
		if item.Token.Identity == itemMutationWeapon {
			weapon = item.ID
		}
	}
	if weapon == 0 {
		t.Fatal("fixture has no original weapon identity")
	}
	f.live.pending = append(f.live.pending, sim.TerminalKill(actor.ID))
	var loot sim.Sack
	for tick := 0; tick < 100 && loot.ObjectID == 0; tick++ {
		f.live.tick()
		for _, sack := range f.live.world.Sacks() {
			for _, item := range sack.ItemInstances {
				if item.ObjectID == weapon {
					loot = sack
				}
			}
		}
	}
	if loot.ObjectID == 0 || loot.X == actor.X && loot.Y == actor.Y {
		t.Fatalf("terminal loot did not leave the blocked death cell: %+v", loot)
	}
	planes, _ = f.live.world.SavedCellPlanes()
	key := uint16(loot.Y)<<8 | uint16(loot.X)
	if planes.Static[key]&1 != 0 || planes.Dynamic[key]&1 != 0 {
		t.Fatal("relocated terminal sack still blocks ground")
	}
	if body, ok := f.live.entity(actor.ID); ok && (body.X != actor.X || body.Y != actor.Y) {
		t.Fatal("terminal loot relocation moved the body")
	}
	for _, row := range f.live.world.SavedObjects().Sacks {
		if row.ID == loot.ObjectID && binary.LittleEndian.Uint16(row.Token.Position[2:]) != key {
			t.Fatal("current Sack token retains the blocked cell")
		}
	}
	for cycle := 0; cycle < 2; cycle++ {
		doc, _ = currentRootSAVDocument(t, f)
		var ordinaryMatches int
		for _, root := range doc.World.Sacks {
			record := &doc.Objects[root-1]
			token, gold, _, err := savedSackRecord(record)
			if err != nil {
				t.Fatal(err)
			}
			if int32(token.Position[2]) != loot.X || int32(token.Position[3]) != loot.Y {
				continue
			}
			ordinaryMatches++
			if gold != loot.Gold {
				t.Fatal("ordinary Sack record lost its gold")
			}
			refs, _ := savedObjectRefs(record, "Contents")
			var ordinaryWeapon bool
			for _, ref := range refs {
				identity, _ := savedStructureValue(&doc.Objects[ref-1], "Identity")
				ordinaryWeapon = ordinaryWeapon || identity == itemMutationWeapon
			}
			if !ordinaryWeapon {
				t.Fatal("ordinary Sack record lost the original weapon")
			}
		}
		if ordinaryMatches != 1 {
			t.Fatal("ordinary SAV did not encode exactly one relocated Sack")
		}
		cold := loadCurrentRootSAV(t, doc, front)
		for tick := 0; tick < 17; tick++ {
			if f.live.world.Hash() != cold.live.world.Hash() || !reflect.DeepEqual(f.live.world.Sacks(), cold.live.world.Sacks()) {
				t.Fatalf("cycle %d tick %d changed relocated sack continuation", cycle, tick)
			}
			if tick == 16 {
				break
			}
			command := []sim.Command{sim.TerminalKill(actor.ID)}
			sim.Step(f.live.world, command)
			sim.Step(cold.live.world, command)
		}
		var matches int
		for _, sack := range cold.live.world.Sacks() {
			if sack.ObjectID == loot.ObjectID {
				matches++
				if !reflect.DeepEqual(sack, loot) {
					t.Fatal("resaving or repeating the death changed loot")
				}
			}
		}
		if matches != 1 {
			t.Fatal("resaving duplicated or lost the relocated sack")
		}
		f = cold
	}
}
