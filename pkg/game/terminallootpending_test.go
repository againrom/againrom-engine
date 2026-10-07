package game

import (
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func TestCurrentTerminalLootWithoutGroundKeepsPendingSAVDeath(t *testing.T) {
	f, _ := itemMutationOpen(t, 1, false)
	f.live.tick()
	actor := newGroupActors(t, f.live.world)[newGroupA]
	doc, _ := currentRootSAVDocument(t, f)
	bounds := f.live.world.Bounds()
	blocks := make(map[uint16]sav.BlockRecord, len(doc.World.Blocks))
	for _, block := range doc.World.Blocks {
		blocks[block.Cell] = block
	}
	doc.World.Blocks = nil
	for y := int32(0); y < bounds.Height; y++ {
		for x := int32(0); x < bounds.Width; x++ {
			key := uint16(y)<<8 | uint16(x)
			block := blocks[key]
			block.Cell, block.Static, block.Dyn = key, block.Static|1, block.Dyn|0x21
			doc.World.Blocks = append(doc.World.Blocks, block)
		}
	}
	front := func(t *testing.T) *FrontEnd { return itemMutationFront(t, 1) }
	f = loadCurrentRootSAV(t, doc, front)
	beforeItems, _ := f.live.world.EquippedItems(actor.ID)
	beforeSacks := f.live.world.Sacks()
	headlessDamage(t, f.live.world, actor.ID, actor.HP+10)
	for range 8 {
		sim.Step(f.live.world, nil)
	}
	body, ok := f.live.entity(actor.ID)
	if !ok || body.Decay != sim.DecayFallen || body.Dwell != 0 || !reflect.DeepEqual(f.live.world.Sacks(), beforeSacks) {
		t.Fatal("blocked map completed terminal teardown or altered existing sacks")
	}
	afterItems, _ := f.live.world.EquippedItems(actor.ID)
	if !reflect.DeepEqual(beforeItems, afterItems) {
		t.Fatal("pending terminal death moved equipment before a destination existed")
	}
	doc, _ = currentRootSAVDocument(t, f)
	cold := loadCurrentRootSAV(t, doc, front)
	for tick := 0; tick < 3; tick++ {
		if f.live.world.Hash() != cold.live.world.Hash() {
			t.Fatalf("pending terminal death changed on cold SAV tick %d", tick)
		}
		sim.Step(f.live.world, nil)
		sim.Step(cold.live.world, nil)
	}
	doc, _ = currentRootSAVDocument(t, cold)
	key := uint16(actor.Y-1)<<8 | uint16(actor.X-1)
	for i := range doc.World.Blocks {
		if doc.World.Blocks[i].Cell == key {
			doc.World.Blocks[i].Static &^= 1
			doc.World.Blocks[i].Dyn &^= 1
		}
	}
	cold = loadCurrentRootSAV(t, doc, front)
	sim.Step(cold.live.world, nil)
	loot := groundAt(cold.live.world.Sacks(), actor.X-1, actor.Y-1)
	if loot == nil || len(loot.ItemInstances) == 0 || loot.ItemInstances[0].ObjectID != beforeItems[0].ObjectID {
		t.Fatal("pending SAV death failed to deliver the retained original weapon")
	}
}
