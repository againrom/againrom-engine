package sav

import (
	"encoding/binary"
	"testing"
)

func TestNewCityStateDataBuildsAValidCityAcceptedByCityFromData(t *testing.T) {
	const identity, owner = uint32(0x20), uint32(0x10)
	fixed := make([]byte, 51)
	binary.LittleEndian.PutUint16(fixed[0:2], 1)
	binary.LittleEndian.PutUint32(fixed[2:6], 1)
	binary.LittleEndian.PutUint32(fixed[43:47], identity)
	binary.LittleEndian.PutUint32(fixed[47:51], owner)
	unit := CityUnitData{
		Token: cityTestToken(identity, owner, 0), RawA6: make([]byte, 24), RawBE: make([]byte, 22),
		Raw114: make([]byte, 24), RawD4: make([]byte, 64), Raw154: make([]byte, 180), Raw158: make([]byte, 148),
		Scalar1: make([]byte, 19), Name: "Hero", Scalar2: make([]byte, 55),
		ScalarTail: make([]byte, 17), XP: make([]byte, 24), Equipment: make([]uint16, 13),
	}
	data := CityData{
		Version: CityDataVersion, FileVersion: MinVersion, Head: [13]uint32{12: 2},
		PlayerList: 1, Players: []uint16{1}, Marker: GeneratedCityMarker,
		TrailerState: make([]byte, generatedCityTrailerLen),
		Objects: []CityObjectData{
			{Class: "Player", Player: &CityPlayerData{
				Name: "Player", Fixed: fixed, Raw32: make([]byte, 32),
				Groups: []CityGroupData{{Raw80: make([]byte, 80), F44: owner, Actors: []uint16{2}}},
			}},
			{Class: "Human", Unit: &unit},
		},
		State: NewCityStateData("Hero"),
	}
	provenance, err := CityFromData(data)
	if err != nil {
		t.Fatalf("CityFromData: %v", err)
	}
	roster := provenance.Roster()
	if len(roster) != 1 || roster[0].Name != "Hero" || !roster[0].Hero {
		t.Fatalf("roster = %#v, want one hero named Hero", roster)
	}
	update := CityUpdate{Label: []byte("native"), Money: 555, Characters: cityTestUpdates(roster)}
	raw, err := provenance.Marshal(update)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	file, err := Open(raw)
	if err != nil {
		t.Fatalf("Open(generated): %v", err)
	}
	if string(file.Label) != "native" || len(file.Players) != 1 || file.Players[0].Money != 555 {
		t.Fatalf("label=%q players=%#v", file.Label, file.Players)
	}
	reopened, err := file.CityProvenance()
	if err != nil {
		t.Fatalf("CityProvenance(generated): %v", err)
	}
	if got := reopened.Roster(); len(got) != 1 || got[0].Name != "Hero" || !got[0].Hero {
		t.Fatalf("reopened roster = %#v", got)
	}
}

// TestNewCityStateDataRejectsNoStartingHeroAtTheDTOBoundary pins the other
// side of the same boundary: CityFromData requires the Player's hero
// identity (Fixed[43:47]) to match exactly one Unit's own identity, the same
// "exactly one starting hero" invariant nativeCityData
// (pkg/game/nativecity.go) enforces before it ever builds a CityData at all.
// A hero identity that resolves to zero roster members is rejected here, not
// silently accepted as a heroless roster.
func TestNewCityStateDataRejectsNoStartingHeroAtTheDTOBoundary(t *testing.T) {
	const identity, owner = uint32(0x20), uint32(0x10)
	fixed := make([]byte, 51)
	binary.LittleEndian.PutUint16(fixed[0:2], 1)
	binary.LittleEndian.PutUint32(fixed[2:6], 1)
	binary.LittleEndian.PutUint32(fixed[43:47], 0xffffffff) // matches no Unit
	binary.LittleEndian.PutUint32(fixed[47:51], owner)
	unit := CityUnitData{
		Token: cityTestToken(identity, owner, 0), RawA6: make([]byte, 24), RawBE: make([]byte, 22),
		Raw114: make([]byte, 24), RawD4: make([]byte, 64), Raw154: make([]byte, 180), Raw158: make([]byte, 148),
		Scalar1: make([]byte, 19), Name: "Hero", Scalar2: make([]byte, 55),
		ScalarTail: make([]byte, 17), XP: make([]byte, 24), Equipment: make([]uint16, 13),
	}
	data := CityData{
		Version: CityDataVersion, FileVersion: MinVersion, Head: [13]uint32{12: 2},
		PlayerList: 1, Players: []uint16{1}, Marker: GeneratedCityMarker,
		TrailerState: make([]byte, generatedCityTrailerLen),
		Objects: []CityObjectData{
			{Class: "Player", Player: &CityPlayerData{
				Name: "Player", Fixed: fixed, Raw32: make([]byte, 32),
				Groups: []CityGroupData{{Raw80: make([]byte, 80), F44: owner, Actors: []uint16{2}}},
			}},
			{Class: "Human", Unit: &unit},
		},
		State: NewCityStateData("Hero"),
	}
	if _, err := CityFromData(data); err == nil {
		t.Fatal("CityFromData accepted a hero identity matching no roster member")
	}
}
