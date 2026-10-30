package game

import (
	"encoding/binary"
	"reflect"
	"slices"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func townActorFixture() *sav.CityUnitData {
	u := &sav.CityUnitData{Scalar1: make([]byte, 19), Scalar2: make([]byte, 55), Raw154: make([]byte, 180), Raw158: make([]byte, 148)}
	u.Raw154[5], u.Raw154[10] = 0x41, 9
	return u
}

// A town-built actor writes the Unit constructor's +0x50, +0x136 and +0x138;
// a kept actor writes its carrier, keeping the Human's current turn rate.
func TestTownActorWritesCarrierOrConstructor(t *testing.T) {
	built := townActorFixture()
	if err := writeTownActor(built, nil); err != nil {
		t.Fatal(err)
	}
	if got := binary.LittleEndian.Uint32(built.Scalar1[4:8]); got != 0x0b {
		t.Fatalf("town-built U50 = %#x, want 0xb", got)
	}
	if built.Scalar2[41] != 1 || binary.LittleEndian.Uint32(built.Scalar2[42:46]) != 0 || built.Raw154[5] != 0x41 {
		t.Fatalf("town-built U136/U138/mover = %d/%d/%#x", built.Scalar2[41], binary.LittleEndian.Uint32(built.Scalar2[42:46]), built.Raw154[5])
	}

	k := &mapload.KeptActor{State: 0, Done: 1, Deadline: 577, Timer: 3, MoverPresent: true,
		StaticRoute: []uint16{0x1010}, DynamicRoute: []uint16{0x1111, 0x1212}, OrderPresent: true, OrderRing: []uint16{7}}
	k.Mover[0x80], k.Mover[10], k.Mover[5] = 0x22, 77, 0x44
	k.Order[8] = 1
	kept := townActorFixture()
	if err := writeTownActor(kept, k); err != nil {
		t.Fatal(err)
	}
	if binary.LittleEndian.Uint32(kept.Scalar1[4:8]) != 0 || kept.Scalar1[18] != 3 || kept.Scalar2[41] != 1 || binary.LittleEndian.Uint32(kept.Scalar2[42:46]) != 577 {
		t.Fatalf("kept scalars U50/U6C/U136/U138 = %d/%d/%d/%d", binary.LittleEndian.Uint32(kept.Scalar1[4:8]), kept.Scalar1[18], kept.Scalar2[41], binary.LittleEndian.Uint32(kept.Scalar2[42:46]))
	}
	if kept.Raw154[0x80] != 0x22 || kept.Raw154[5] != 0x44 || kept.Raw154[10] != 9 {
		t.Fatalf("kept mover +80/+5/+10 = %#x/%#x/%d", kept.Raw154[0x80], kept.Raw154[5], kept.Raw154[10])
	}
	if !slices.Equal(kept.Words15c, k.StaticRoute) || !slices.Equal(kept.Words178, k.DynamicRoute) || kept.Raw158[8] != 1 || !slices.Equal(kept.Words158, k.OrderRing) {
		t.Fatal("kept lists or Order not written")
	}
}

// A town LOAD fills the carrier with exactly the record's fields, so a SAVE
// right after writes them back.
func TestTownActorLoadThenWriteKeepsFields(t *testing.T) {
	mover, order := make([]byte, 180), make([]byte, 148)
	mover[0x80], mover[10], order[3] = 5, 9, 6
	c := sav.Character{Town: sav.TownActorState{State: 0, Done: 1, Deadline: 623, Timer: 0, Mover: mover, Order: order,
		StaticRoute: []uint16{1}, OrderRing: []uint16{4, 5}}}
	u := townActorFixture()
	if err := writeTownActor(u, keptFromCharacter(c)); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(u.Raw154, mover) || !slices.Equal(u.Raw158, order) || binary.LittleEndian.Uint32(u.Scalar2[42:46]) != 623 || u.Scalar2[41] != 1 {
		t.Fatal("loaded fields not written back")
	}
	if !slices.Equal(u.Words15c, []uint16{1}) || len(u.Words178) != 0 || !slices.Equal(u.Words158, []uint16{4, 5}) {
		t.Fatal("loaded lists not written back")
	}
}

// The writer walks the city's actor records in party order and refuses a
// count that does not match.
// A record holding only what the town writer writes for a town-built actor
// loads as no carrier; a zero Order and a constructor Mover load as absent,
// so the carrier crosses a SAVE and LOAD unchanged.
func TestTownActorLoadKeepsTheCarrierShape(t *testing.T) {
	u := townActorFixture()
	if err := writeTownActor(u, nil); err != nil {
		t.Fatal(err)
	}
	built := sav.Character{Town: sav.TownActorState{State: townBuiltState, Done: townBuiltDone, Deadline: townBuiltDeadline,
		Mover: u.Raw154, Order: u.Raw158, StaticRoute: []uint16{}, DynamicRoute: []uint16{}}}
	if k := keptFromCharacter(built); k != nil {
		t.Fatalf("a town-built record loads a carrier %+v", k)
	}
	kept := &mapload.KeptActor{State: cullActorState, Done: cullActorDone, Deadline: 29}
	u = townActorFixture()
	if err := writeTownActor(u, kept); err != nil {
		t.Fatal(err)
	}
	c := sav.Character{Town: sav.TownActorState{State: kept.State, Done: kept.Done, Deadline: kept.Deadline, Mover: u.Raw154, Order: u.Raw158}}
	got := keptFromCharacter(c)
	if got == nil || !reflect.DeepEqual(*got, *kept) {
		t.Fatalf("kept carrier %+v loads as %+v", kept, got)
	}
}

// Only the first game carries the cull's actor state to town; a town LOAD
// fills the carrier from the actor record in both games.
func TestOnlyTheFirstGameKeepsTownActorState(t *testing.T) {
	if !(firstCampaignRules{}).keepsTownActorState() || (secondCampaignRules{}).keepsTownActorState() {
		t.Fatal("town actor state switch: want first game true, second game false")
	}
	chars := []sav.Character{{Town: sav.TownActorState{State: cullActorState, Done: cullActorDone, Deadline: 623}}}
	party := []mapload.PartyMember{{ID: "hero"}}
	keepLoadedTownActors(party, chars)
	if party[0].Kept == nil || party[0].Kept.Deadline != 623 {
		t.Fatalf("LOAD carrier = %+v, want deadline 623", party[0].Kept)
	}
}

func TestTownActorsFollowPartyOrder(t *testing.T) {
	city := sav.CityData{Objects: []sav.CityObjectData{{Class: "Player", Player: &sav.CityPlayerData{Fixed: make([]byte, 51)}},
		{Class: "Human", Unit: townActorFixture()}, {Class: "Unit", Unit: townActorFixture()}}}
	party := []mapload.PartyMember{{ID: "a", Kept: &mapload.KeptActor{Done: 1, Deadline: 40}}, {ID: "b"}}
	if err := writeTownActors(&city, party); err != nil {
		t.Fatal(err)
	}
	if binary.LittleEndian.Uint32(city.Objects[1].Unit.Scalar2[42:46]) != 40 || binary.LittleEndian.Uint32(city.Objects[2].Unit.Scalar1[4:8]) != 0x0b {
		t.Fatal("actor records not written in party order")
	}
	if err := writeTownActors(&city, party[:1]); err == nil {
		t.Fatal("a party shorter than the actor records was accepted")
	}
	if err := writeTownPlayerColor(&city, 5); err != nil || city.Objects[0].Player.Fixed[14] != 5 {
		t.Fatalf("Player F44 = %d, %v", city.Objects[0].Player.Fixed[14], err)
	}
}

// The Player +0x44 a mission leaves is its map group's colour plus one.
func TestMissionPlayerColor(t *testing.T) {
	m := &alm.Map{Groups: []alm.Group{{Color: 4}}}
	if missionPlayerColor(m, sim.SelfSlot) != 5 || missionPlayerColor(m, 2) != 1 || missionPlayerColor(nil, 1) != 1 {
		t.Fatal("mission Player colour")
	}
}

// The run end a kept actor carries is its current one, else the one an
// original LOAD read.
func TestCurrentActionRunEnd(t *testing.T) {
	e := sim.Entity{ActionClock: sim.ActionClock{Known: true, End: 9}}
	if currentActionRunEnd(e) != 9 {
		t.Fatal("known run end")
	}
	e = sim.Entity{}
	if currentActionRunEnd(e) != 0 {
		t.Fatal("unknown run end without native basis")
	}
}
