package game

import (
	"crypto/sha256"
	"fmt"
	"os"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func TestReleaseTerminalLootGroundCurrentSAV(t *testing.T) {
	path := os.Getenv("AGAINROM_TERMINAL_LOOT_SAV")
	if path == "" {
		t.Skip("set AGAINROM_TERMINAL_LOOT_SAV to the owner blocked-loot source")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	const sourceSHA = "94105d209cc27eba5086d78bab4c883315824e8bab8d5059eb53c8c522f157ab"
	if fmt.Sprintf("%x", sha256.Sum256(raw)) != sourceSHA {
		t.Fatal("blocked-loot source SHA differs")
	}
	f := releaseFront(t)
	app, _ := openOriginalSAVApp(t, f, raw, "blocked-loot-source.sav")
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil || doc.World == nil {
		t.Fatal("source lacks ordinary world blocks", err)
	}
	masks := make(map[uint16]sav.BlockRecord, len(doc.World.Blocks))
	for _, block := range doc.World.Blocks {
		masks[block.Cell] = block
	}
	var target sim.Entity
	for _, e := range f.live.world.Entities() {
		if !e.Alive() || e.OffMap || e.Domain != sim.DomainAir {
			continue
		}
		key := uint16(e.Y)<<8 | uint16(e.X)
		block := masks[key]
		t.Logf("live flyer id=%d type=%d cell=%d,%d masks=%02x/%02x gold=%d:%d+%d", e.ID, e.TypeID, e.X, e.Y, block.Static, block.Dyn, e.GoldChance, e.TreasureMin, e.TreasureMax)
		if target.ID == 0 && (block.Static&1 != 0 || block.Dyn&1 != 0) && e.GoldChance > 0 && (e.TreasureMin > 0 || e.TreasureMax > 0) {
			target = e
		}
	}
	if target.ID == 0 {
		t.Fatal("source has no live gold-bearing flyer over blocked ground")
	}
	before := f.live.world.Sacks()
	f.LiveDamage(uint32(target.ID), target.HP+10)
	var loot sim.Sack
	for tick := 0; tick < 96 && loot.Gold == 0; tick++ {
		f.live.tick()
		for _, sack := range f.live.world.Sacks() {
			old := groundAt(before, sack.X, sack.Y)
			if old == nil && sack.Gold != 0 {
				loot = sack
			}
		}
	}
	if loot.Gold == 0 || loot.X == target.X && loot.Y == target.Y {
		t.Fatal("new terminal gold sack did not leave blocked corpse cell")
	}
	key := uint16(loot.Y)<<8 | uint16(loot.X)
	for _, old := range before {
		current := groundAt(f.live.world.Sacks(), old.X, old.Y)
		if current == nil || !reflect.DeepEqual(*current, old) {
			t.Fatal("terminal relocation rewrote an existing sack")
		}
	}
	if body, present := f.live.entity(target.ID); present && (body.X != target.X || body.Y != target.Y) {
		t.Fatal("relocation moved corpse position")
	}
	store, name, saved := menuSAVE(t, f, app, OriginalStore{})
	output, err := sav.DecodeDocumentData(saved)
	if err != nil {
		t.Fatal(err)
	}
	for _, block := range output.World.Blocks {
		if block.Cell == key && (block.Static&1 != 0 || block.Dyn&1 != 0) {
			t.Fatal("new terminal sack is still on blocked ground")
		}
	}
	cold := loadLocalLegacySave(t, store, name)
	for tick := 0; tick < 5; tick++ {
		if cold.live.world.Hash() != f.live.world.Hash() {
			logCurrentCarrierDiff(t, f.live.world, cold.live.world)
			t.Fatalf("cold SAV tick %d differs", tick)
		}
		current := groundAt(cold.live.world.Sacks(), loot.X, loot.Y)
		if current == nil || !reflect.DeepEqual(*current, loot) {
			t.Fatal("cold SAV lost relocated loot")
		}
		if tick != 4 {
			f.live.tick()
			cold.live.tick()
		}
	}
	t.Logf("source=%s actor=%d origin=(%d,%d) new-sack=(%d,%d) gold=%d old-sacks=%d; ordinary menu SAV, cold LOAD and 4 next ticks agree", sourceSHA, target.ID, target.X, target.Y, loot.X, loot.Y, loot.Gold, len(before))
}
