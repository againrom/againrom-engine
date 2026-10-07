package game

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// announced is the message lines an announcement posts: each text in white
// for 3000 ms (MISSION-MSGPOST-058).
func announced(texts ...string) []ui.MessageLine {
	var out []ui.MessageLine
	for _, s := range texts {
		out = append(out, ui.MessageLine{Text: s, Ink: ui.MessageWhite, Life: 3 * time.Second})
	}
	return out
}

// pickupLineRows orders hero 7, standing at (3,3) with stock in his pack, to
// pick up the sack at (4,3), ticks until the sack is gone and answers the
// lines the take posted. The fixture viewer holds the authored words, the EN
// root's own main.txt lines 85..89.
func pickupLineRows(t *testing.T, stock []uint16, gold uint32, items []uint16) []ui.MessageLine {
	t.Helper()
	return pickupLineWorld(t, stock, gold, items).view.MessageLines()
}

func pickupLineWorld(t *testing.T, stock []uint16, gold uint32, items []uint16) *mapWorld {
	t.Helper()
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: 7, X: 3, Y: 3, Owner: sim.SelfSlot, HP: 20, MaxHP: 20, Speed: 16, Facing: 64, DesiredFacing: 64}}, nil, sim.Relations{},
		[]sim.Sack{{X: 4, Y: 3, Gold: gold, Items: items}}, []sim.Stock{{ID: 7, Items: stock}})
	if err != nil {
		t.Fatal(err)
	}
	mw := grabWorld(t, w, 7, missionSource{})
	mw.invParty.table = eqDefsTable(t)
	mw.grab(7, 4, 3, true)
	for n := 0; n < 64 && len(w.Sacks()) != 0; n++ {
		mw.tick()
	}
	if len(w.Sacks()) != 0 {
		t.Fatal("the pickup order did not take the sack")
	}
	return mw
}

// A sack of gold and four Items posts five white rows, the gold row first.
func TestSackPickupRowsPostTheGoldRowFirst(t *testing.T) {
	codes := []uint16{eqSwordCode, eqNoSlotCode, eqSwordCode + 1, eqNoSlotCode + 1}
	lines := pickupLineWorld(t, nil, 272, codes).view.MessageLines()
	if len(lines) != 5 || lines[0].Text != "Picked up 272 gold" {
		t.Fatalf("the take posted %+v, want five rows with the gold row first", lines)
	}
	for i, ln := range lines {
		if i > 0 && !strings.HasPrefix(ln.Text, "Picked up ") {
			t.Fatalf("row %d reads %q, want an Item row", i, ln.Text)
		}
		if ln.Ink != ui.MessageWhite || ln.Life != 3*time.Second {
			t.Fatalf("row %d %q: ink %d, life %v; want white, 3s", i, ln.Text, ln.Ink, ln.Life)
		}
	}
}

// A pickup order onto a sack posts one line per carried Item it reached:
// main.txt global string 85, one space and the name, and for an Item of two
// ` (`, string 86, the count and string 87, `)` (ITEM-PICKTEXT-145,
// DIV-1425); the gold line comes first.
func TestSackPickupRowsStateThePickedUpLine(t *testing.T) {
	rows := pickupLineRows(t, nil, 500, []uint16{eqSwordCode, eqNoSlotCode, eqSwordCode})
	want := announced(
		"Picked up 500 gold",
		"Picked up Sword (now 2 pieces)",
		"Picked up "+data.ItemCode(eqNoSlotCode).Name(),
	)
	if !reflect.DeepEqual(rows, want) {
		t.Fatalf("the pickup order posted %+v, want %+v", rows, want)
	}
}

// A unit merged into a carried Item states that Item's count after the
// merge, not the one unit moved: the merge keeps the destination Item with
// the summed count, and the pickup flag the line answers is on it
// (ITEM-MERGE-129, ITEM-GROUNDMOVE-130). Carried Items the take left alone
// post nothing.
func TestSackPickupMergedIntoACarriedItemStatesItsNewCount(t *testing.T) {
	rows := pickupLineRows(t, []uint16{eqNoSlotCode, eqSwordCode}, 0, []uint16{eqSwordCode})
	want := announced("Picked up Sword (now 2 pieces)")
	if !reflect.DeepEqual(rows, want) {
		t.Fatalf("one Sword picked up beside one carried posted %+v, want %+v", rows, want)
	}
}

// Items a LOAD publishes at session entry post the same line, one row per
// flagged carried Item, each stating its own count (SAV-1114, SAV-1115).
func TestSessionEntryPickupRowsStateThePickedUpLine(t *testing.T) {
	w, err := sim.NewLootWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: 7, X: 3, Y: 3, Owner: sim.SelfSlot, HP: 20, MaxHP: 20}}, nil, sim.Relations{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	m := worldFixtureMap()
	ms := &Mission{Number: 1, Map: m, World: w,
		Party:          []mapload.PartyMember{{PlayerCharacter: true, StartingHero: true}},
		Start:          mapload.Start{IDs: []sim.EntityID{7}},
		pendingPickups: []sim.ItemStack{sim.PlainStack(eqSwordCode, 1), sim.PlainStack(eqSwordCode, 3)}}
	mw := openMission(ms, eqDefsTable(t), nil, worldFixtureViewer(t, m), missionSource{}, nil, nil)
	want := announced("Picked up Sword", "Picked up Sword (now 3 pieces)")
	if rows := mw.view.MessageLines(); !reflect.DeepEqual(rows, want) {
		t.Fatalf("session entry posted %+v, want %+v", rows, want)
	}
}
