package game

import (
	"encoding/binary"
	"reflect"
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
)

func constructedTown(n int) sav.CityData {
	actors := make([]uint16, n)
	for i := range actors {
		actors[i] = uint16(i + 2)
	}
	return sav.CityData{Players: []uint16{1}, Objects: []sav.CityObjectData{
		{Class: "Player", Player: &sav.CityPlayerData{Groups: []sav.CityGroupData{{Raw80: make([]byte, 80), F44: nativeCityPlayerIdentity, Actors: actors}}}},
	}}
}

func hireGroupOf(actors ...uint16) sav.CityGroupData {
	g := cityHireGroup()
	g.Actors = actors
	return g
}

func TestLoadedCityGroupsFollowMembersAndLeaveJoinersToTheFirstGroup(t *testing.T) {
	unit := func(identity uint32) sav.CityObjectData {
		token := make([]byte, 37)
		binary.LittleEndian.PutUint32(token[29:], identity)
		return sav.CityObjectData{Class: "Human", Unit: &sav.CityUnitData{Token: token}}
	}
	fixed := make([]byte, 51)
	binary.LittleEndian.PutUint32(fixed[47:], 0x2c1ed70)
	raw := func(b byte) []byte { r := make([]byte, 80); r[0] = b; return r }
	loaded := &SnapshotOriginalCity{
		Document: sav.CityData{Players: []uint16{1}, Objects: []sav.CityObjectData{
			{Class: "Player", Player: &sav.CityPlayerData{Fixed: fixed, Groups: []sav.CityGroupData{
				{Raw80: raw(1), F1c: 7, F44: 0x2c1ed70, Actors: []uint16{3}},
				{Raw80: raw(2), F40: 0x5555, F44: 0x2c1ed70, Actors: []uint16{2}},
				{Raw80: raw(3), F44: 0x2c1ed70, Actors: []uint16{4, 5}},
				{Raw80: raw(4), F44: 0x2c1ed70, Actors: []uint16{4}},
			}}},
			unit(0xa0), unit(0xb0), unit(0xc0), unit(0xd0),
		}},
		Bindings: []SnapshotCityBinding{{Identity: 0xa0, PartyID: "a"}, {Identity: 0xb0, PartyID: "b"}, {Identity: 0xc0, PartyID: "gone"}, {Identity: 0xd0, PartyID: "old"}},
	}
	// "old" was hired before the load and stays in its loaded group; the
	// type-5 and type-3 squads were hired after it, in that order.
	party := []mapload.PartyMember{{ID: "a"}, {ID: "b"}, {ID: "joiner"}, {ID: "old", MercenaryType: 4},
		{ID: "m5", MercenaryType: 5}, {ID: "m3a", MercenaryType: 3}, {ID: "m3b", MercenaryType: 3}}
	written := constructedTown(len(party))
	order := writeCityGroups(&written, party, cityGroupsFromLoaded(party, loaded))
	if !slices.Equal(order, []int{1, 2, 0, 3, 4, 5, 6}) {
		t.Fatalf("group order %v", order)
	}
	want := []sav.CityGroupData{
		{Raw80: raw(1), F1c: 7, F44: nativeCityPlayerIdentity, Actors: []uint16{3, 4}},
		{Raw80: raw(2), F44: nativeCityPlayerIdentity, Actors: []uint16{2}},
		{Raw80: raw(3), F44: nativeCityPlayerIdentity, Actors: []uint16{5}},
		hireGroupOf(6),
		hireGroupOf(7, 8),
	}
	if got := written.Objects[0].Player.Groups; !reflect.DeepEqual(got, want) {
		t.Fatalf("written groups %+v, want %+v", got, want)
	}
}

func TestCityGroupsWithoutALoadedTownPutEachHireInItsOwnGroup(t *testing.T) {
	party := []mapload.PartyMember{{ID: "hero"}, {ID: "m5", MercenaryType: 5}, {ID: "m3a", MercenaryType: 3}, {ID: "m3b", MercenaryType: 3}, {ID: "joiner"}}
	written := constructedTown(len(party))
	order := writeCityGroups(&written, party, nil)
	if !slices.Equal(order, []int{0, 4, 1, 2, 3}) {
		t.Fatalf("group order %v", order)
	}
	want := []sav.CityGroupData{
		{Raw80: make([]byte, 80), F44: nativeCityPlayerIdentity, Actors: []uint16{2, 6}},
		hireGroupOf(3),
		hireGroupOf(4, 5),
	}
	if got := written.Objects[0].Player.Groups; !reflect.DeepEqual(got, want) {
		t.Fatalf("written groups %+v, want %+v", got, want)
	}

	plain := []mapload.PartyMember{{ID: "hero"}, {ID: "joiner"}}
	unhired := constructedTown(len(plain))
	if order := writeCityGroups(&unhired, plain, nil); !slices.Equal(order, []int{0, 1}) || !reflect.DeepEqual(unhired, constructedTown(len(plain))) {
		t.Fatalf("a town with no loaded document and no hire changed its one group: %v", order)
	}
}
