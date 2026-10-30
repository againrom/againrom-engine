package game

import (
	"strings"
	"testing"
)

// twoTownBoundaryFront is mercenaryBoundaryFront over the travel fixture's
// two towns: West, town 1, holds the two hired squads; East, town 2, has a
// pool of its own of each type West hired.
func twoTownBoundaryFront(t *testing.T) *FrontEnd {
	t.Helper()
	f := mercenaryBoundaryFront(t)
	tw := f.Town
	book := travelBook(t, travelCatalog)
	tw.camp.Towns = book
	e2, _ := book.catalog.Town(2)
	east := newTownState(e2, tw.camp)
	for _, typ := range []int{boundaryHiredType, boundarySecondType} {
		east.mercPool[typ], east.mercCapacity[typ] = 4, 4
	}
	tw.others = map[int]townState{2: east}
	tw.open = true
	return f
}

// A squad hired in West and taken on a mission from East goes home to West
// at the mission end: its hire ends there and West's pool becomes the live
// tally; East, which never offered the type while West held it, keeps its
// own pool. A second mission ended from West grows that pool by the base
// rule instead of wiping it.
func TestHireEndsInItsTownWhereverTheMissionEnds(t *testing.T) {
	f := twoTownBoundaryFront(t)
	tw := f.Town
	if !tw.travelTo(2) {
		t.Fatal("the party cannot travel to East")
	}
	if _, ok := tw.mercenaryHire(boundaryHiredType, 0); ok {
		t.Error("East hires a type West holds: the party carries two squads of one type")
	}
	ms, ids := mercenaryBoundaryMission(t, 10)
	f.FinishMission(10, ms.Party, ms.World, ids)
	tw = f.Town
	if tw.townID() != 2 || tw.MercenaryHired(boundaryHiredType) || tw.MercenaryPool(boundaryHiredType) != 4 {
		t.Errorf("East after the mission: town %d, hired %v, pool %d; want East not hired with its pool of 4",
			tw.townID(), tw.MercenaryHired(boundaryHiredType), tw.MercenaryPool(boundaryHiredType))
	}
	tw.open = true
	if !tw.travelTo(1) {
		t.Fatal("the party cannot travel back to West")
	}
	for _, c := range []struct{ typ, pool int }{
		{boundaryHiredType, 1}, {boundarySecondType, 1}, {boundaryUnhiredType, boundaryUnhiredStart + 1},
	} {
		if tw.MercenaryHired(c.typ) || tw.MercenaryPool(c.typ) != c.pool {
			t.Errorf("West type %d after the mission from East: hired %v pool %d, want not hired and pool %d",
				c.typ, tw.MercenaryHired(c.typ), tw.MercenaryPool(c.typ), c.pool)
		}
	}
	ms2, ids2 := mercenaryBoundaryMission(t, 20)
	f.FinishMission(20, ms2.Party[:1], ms2.World, ids2[:1])
	if tw = f.Town; tw.MercenaryHired(boundaryHiredType) || tw.MercenaryPool(boundaryHiredType) != 2 {
		t.Fatalf("West after a second mission from West: hired %v pool %d, want not hired and pool 2",
			tw.MercenaryHired(boundaryHiredType), tw.MercenaryPool(boundaryHiredType))
	}
}

// unlockCampaign is the travel fixture's campaign whose mission 30 unlocks
// mercenary type 4, over catalog raw.
func unlockCampaign(t *testing.T, raw string) Campaign {
	t.Helper()
	c := townCampaign(t)
	c.MercenaryCount = []int{4, 4, 4, 4}
	c.Towns = travelBook(t, raw)
	ch := c.Chapters[30]
	ch.EnableMercenary = []int{4}
	c.Chapters[30] = ch
	return c
}

// A mission's mercenary unlock won in West enables the type in East, in a
// town that opens after that very mission, and in a town a cold LOAD builds
// fresh.
func TestMercenaryUnlockReachesEveryTown(t *testing.T) {
	opensAfter := strings.Replace(travelCatalog, `"missions": [31]`, `"missions": [31], "opens-after": 30`, 1)
	for _, c := range []struct{ name, raw string }{{"open", travelCatalog}, {"opens-after", opensAfter}} {
		tw := NewTown(unlockCampaign(t, c.raw))
		tw.Arrive()
		if _, ok := tw.Won(30); !ok {
			t.Fatal("the win of mission 30 is refused")
		}
		if !tw.MercenaryEnabled(4) || !tw.travelTo(2) {
			t.Fatalf("%s: West enabled %v, travel to East refused", c.name, tw.MercenaryEnabled(4))
		}
		if !tw.MercenaryEnabled(4) {
			t.Errorf("%s: the unlock won in West is not enabled in East", c.name)
		}
	}

	camp := unlockCampaign(t, travelCatalog)
	tw := NewTown(camp)
	tw.Arrive()
	tw.Won(30)
	var s Snapshot
	snapshotTown(tw, &s, firstCampaignRules{})
	back := testRestoreTown(camp, s)
	if !back.travelTo(2) || !back.MercenaryEnabled(4) {
		t.Errorf("after a cold LOAD East enables type 4: %v", back.MercenaryEnabled(4))
	}
}

// testRestoreTown is restoreTown for a snapshot the test built valid: an
// error is the test's own fault.
func testRestoreTown(c Campaign, s Snapshot) *Town {
	t, err := restoreTown(c, s)
	if err != nil {
		panic(err)
	}
	return t
}

// A town record whose mercenary state the town cannot hold is refused: a
// pool below zero or above its capacity, a hire of a type the town has no
// squad of, and one type hired in two towns.
func TestTownRecordRefusesBadMercenaryState(t *testing.T) {
	camp := unlockCampaign(t, travelCatalog)
	west := currentTownState{ID: 1}
	west.Pool[0], west.Capacity[0] = 3, 4
	if err := NewTown(camp).restoreTowns(&currentTowns{Town: 2, Others: []currentTownState{west}}); err != nil {
		t.Fatalf("a valid record is refused: %v", err)
	}
	for name, bad := range map[string]func(o *currentTownState){
		"negative pool":       func(o *currentTownState) { o.Pool[0] = -1 },
		"pool above capacity": func(o *currentTownState) { o.Pool[0] = 5 },
		"hire of no squad":    func(o *currentTownState) { o.Hired[5] = true },
	} {
		o := west
		bad(&o)
		if err := NewTown(camp).restoreTowns(&currentTowns{Town: 2, Others: []currentTownState{o}}); err == nil {
			t.Errorf("%s: the record restores", name)
		}
	}
	both := NewTown(camp)
	both.mercHired[1] = true
	o := west
	o.Hired[0] = true
	if err := both.restoreTowns(&currentTowns{Town: 2, Others: []currentTownState{o}}); err == nil {
		t.Error("one type hired in two towns restores")
	}
}

// A chapter town that opens after a mission the campaign does not hold is
// refused with the campaign.
func TestCampaignRefusesATownOpeningAfterAnUnknownMission(t *testing.T) {
	if err := unlockCampaign(t, strings.Replace(travelCatalog, `"missions": [31]`, `"missions": [31], "opens-after": 30`, 1)).checkTowns(); err != nil {
		t.Fatalf("a town opening after mission 30: %v", err)
	}
	err := unlockCampaign(t, strings.Replace(travelCatalog, `"missions": [31]`, `"missions": [31], "opens-after": 99`, 1)).checkTowns()
	if err == nil || !strings.Contains(err.Error(), "mission 99") {
		t.Fatalf("a town opening after mission 99: %v", err)
	}
}
