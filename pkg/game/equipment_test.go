package game

import (
	"image"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/sim"
)

func TestRefreshEquipmentMovesTheTrackerWhenAPieceLandsOutsideSlotOne(t *testing.T) {
	table := gaTable(t)
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: 7, X: 3, Y: 3, HP: 10, MaxHP: 10}}, nil, sim.Relations{}, nil,
		[]sim.Stock{{ID: 7, Items: []uint16{gaBootsCode}}})
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	mw := equipMission(t, w, 7, eqHero(), nil, table)

	if mw.invFigureEquipment != (data.Equipment{}) {
		t.Fatalf("setup: invFigureEquipment = %+v, want the zero value at mission open", mw.invFigureEquipment)
	}

	mw.enqueueEquip(0) // the boots, the pack's only item
	mw.tick()          // applies the equip; gaBootsCode's own row lands it in slot 12

	if mw.invFigureEquipment != (data.Equipment{}) {
		t.Fatalf("invFigureEquipment moved to %+v from a bare tick, before refreshEquipment ever ran — "+
			"the two trackers are no longer on separate seams", mw.invFigureEquipment)
	}

	mw.refreshEquipment()

	var want data.Equipment
	want.SetCode(12, data.ItemCode(gaBootsCode))
	if mw.invFigureEquipment != want {
		t.Errorf("invFigureEquipment = %+v, want %+v (the boots, in slot 12) — the tracker did not follow "+
			"the piece into a slot other than the first", mw.invFigureEquipment, want)
	}
}

// TestRefreshEquipmentRecomposesNothingWhenTheEquipmentHasNotChanged is the
// guard's OTHER arm — eq == mw.invFigureEquipment (world.go) — and it is
// what makes the assertion above mean something: without it, a
// refreshEquipment that recomposed on every call regardless of the compare
// would pass the first test by coincidence.
//
// A SENTINEL, NOT A NIL FIGURE, is the witness: composeInventorySubject
// against this file's empty missionSource{} (world_test.go) always decodes
// to a nil Figure, so a Figure that is ALREADY nil could not tell "recomposed
// to nil" from "never touched". Planting a distinguishable pointer between
// the two calls and checking it survives a second call over UNCHANGED
// equipment is what discriminates the two.
func TestRefreshEquipmentRecomposesNothingWhenTheEquipmentHasNotChanged(t *testing.T) {
	table := gaTable(t)
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: 7, X: 3, Y: 3, HP: 10, MaxHP: 10}}, nil, sim.Relations{}, nil,
		[]sim.Stock{{ID: 7, Items: []uint16{gaBootsCode}}})
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	mw := equipMission(t, w, 7, eqHero(), nil, table)

	mw.enqueueEquip(0)
	mw.tick()
	mw.refreshEquipment() // the first call: equipment moved, so this one recomposes

	var want data.Equipment
	want.SetCode(12, data.ItemCode(gaBootsCode))
	if mw.invFigureEquipment != want {
		t.Fatalf("setup: invFigureEquipment = %+v, want %+v — the first recompose did not run, "+
			"so the guard below would not be discriminating anything", mw.invFigureEquipment, want)
	}

	sentinel := &image.RGBA{} // a pointer composeInventorySubject could never itself produce here
	mw.invSubject.Figure = sentinel

	mw.refreshEquipment() // the second call: nothing in the world's equipment moved in between

	if mw.invFigureEquipment != want {
		t.Errorf("invFigureEquipment = %+v, want %+v (unchanged) — moved on a call where the equipment did not",
			mw.invFigureEquipment, want)
	}
	if mw.invSubject.Figure != sentinel {
		t.Errorf("Figure was overwritten by a call over unchanged equipment — the guard's other arm did not " +
			"hold, composeInventorySubject ran when eq == invFigureEquipment")
	}
}

// TestRefreshEquipmentDrawsEveryOccupiedSlotsIconIncludingTheTwelfth is
// TestRefreshEquipmentDrawsOnlySlotOnesIconEvenWhenSlotTwelveIsWorn's own
// successor — RENAMED AND RE-PINNED (docs/0136-armour-counts T5: spec
// FR-10b, FR-10c), because the fact it asserted STOPPED BEING TRUE. That
// test pinned a disclosed limit: composeInventorySubject (inventory.go)
// painted and composed slot 1 alone, so a piece worn in slot 12 moved the
// entity's own numbers (rearm_test.go, equip_test.go) but never reached the
// window. T5 lifted that limit — composeInventorySubject now loops over
// every occupied slot, base first, in ascending order — so the old
// assertion (Slots[11] stays nil even though slot 12 is truly occupied) now
// describes a defect this build no longer has. This test keeps the SAME
// setup — the same code worn in both slot 1 and slot 12 at once, by the
// same direct sim.Command the old test used — and asserts the new contract
// instead: BOTH slots now carry their own icon.
//
// THE ARCHIVE IS SUPPLIED DIRECTLY ON mw.mission.src, a same-package field of
// missionNotices (world.go), because equipMission (equip_test.go) always
// opens with an EMPTY missionSource{} — enough for every other test in this
// file and in rearm_test.go, which only read the entity's own numbers back,
// but not for this one, which has to witness an icon actually composing
// rather than merely being attempted. inventory_test.go's own fixtures
// (invBaseSheet, invLayerSheet, invIconStream, partyFigureDir,
// partyFigureFace) are reused rather than re-invented, same package.
func TestRefreshEquipmentDrawsEveryOccupiedSlotsIconIncludingTheTwelfth(t *testing.T) {
	table := gaTable(t)
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: 7, X: 3, Y: 3, HP: 10, MaxHP: 10}}, nil, sim.Relations{}, nil,
		[]sim.Stock{{ID: 7, Items: []uint16{gaBootsCode, gaBootsCode}}})
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	mw := equipMission(t, w, 7, eqHero(), nil, table)

	code := data.ItemCode(gaBootsCode)
	mw.mission.src = missionSource{
		graphicsPrefix + data.ItemFigureBasePath(partyFigureDir, partyFigureFace): invBaseSheet(),
		graphicsPrefix + data.ItemFigureLayerPath(partyFigureDir, code):           invLayerSheet(),
		graphicsPrefix + data.ItemIconPath(code):                                  invIconStream(),
	}

	mw.pending = append(mw.pending,
		sim.Command{Kind: sim.KindEquip, Entity: 7, X: 0, Y: 1},  // the first boots, forced into slot 1
		sim.Command{Kind: sim.KindEquip, Entity: 7, X: 0, Y: 12}, // the second, now at index 0, into slot 12
	)
	mw.tick()
	mw.refreshEquipment()

	eq := mw.currentEquipment()
	if occ, ok := eq.Occupied(1); !ok || !occ {
		t.Fatalf("setup: slot 1 is not occupied — this test cannot witness the contrast")
	}
	if occ, ok := eq.Occupied(12); !ok || !occ {
		t.Fatalf("setup: slot 12 is not occupied — this test cannot witness anything")
	}

	if mw.invSubject.Slots[0] == nil {
		t.Error("Slots[0] is nil though slot 1's own icon is in the archive")
	}
	if mw.invSubject.Slots[11] == nil {
		t.Error("Slots[11] is nil, want the boots' own icon — FR-10c: the twelfth slot's icon composes " +
			"from its own slot's code exactly as the first slot's does, and the limit the retired test " +
			"pinned is lifted")
	}
}
