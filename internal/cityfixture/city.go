// Package cityfixture constructs synthetic semantic inputs, never game bytes.
// These values exercise structural routing; they are not ROM1-safe defaults.
package cityfixture

import (
	"encoding/binary"

	"againrom/pkg/formats/sav"
)

func City(graph bool) sav.CityData {
	token := func(id uint32, row byte) []byte {
		b := make([]byte, 37)
		b[16] = row
		binary.LittleEndian.PutUint32(b[29:], id)
		binary.LittleEndian.PutUint32(b[33:], 0x1234)
		return b
	}
	human := func(name string, id uint32, row byte, strength uint16) *sav.CityUnitData {
		u := &sav.CityUnitData{Token: token(id, row), Name: name, RawA6: make([]byte, 24), RawBE: make([]byte, 22), Raw114: make([]byte, 24), RawD4: make([]byte, 64), Raw154: make([]byte, 180), Raw158: make([]byte, 148), Scalar1: make([]byte, 19), Scalar2: make([]byte, 55), ScalarTail: make([]byte, 17), XP: make([]byte, 24), Equipment: make([]uint16, 13), ContainerFlag: 1}
		binary.LittleEndian.PutUint16(u.Scalar2, strength)
		return u
	}
	p := &sav.CityPlayerData{Name: "Synthetic Player", Fixed: make([]byte, 51), Raw32: make([]byte, 32),
		Groups: []sav.CityGroupData{{Raw80: make([]byte, 80), Actors: []uint16{2, 3}, F44: 0x1234}}}
	binary.LittleEndian.PutUint16(p.Fixed, 1)
	binary.LittleEndian.PutUint32(p.Fixed[2:], 1)
	binary.LittleEndian.PutUint32(p.Fixed[21:], 123^0x5c073f4d)
	binary.LittleEndian.PutUint32(p.Fixed[43:], 0x3456)
	binary.LittleEndian.PutUint32(p.Fixed[47:], 0x1234)
	// Exercise the published version-1 map representation as an independent
	// input too; production Data/EncodeSave canonicalize it to ordered version 2.
	d := sav.CityData{Version: 1, FileVersion: sav.MinVersion, MapName: "10.alm", PlayerList: 1, Players: []uint16{1}, Marker: 0xbadface1, TrailerState: make([]byte, 400),
		Objects: []sav.CityObjectData{{Class: "Player", Player: p}, {Class: "Human", Unit: human("Companion", 0x2345, 29, 31)}, {Class: "Human", Unit: human("Leader", 0x3456, 28, 41)}}}
	d.Head[12] = 2
	d.Campaign.Base.DWords[0] = 10
	d.Campaign.Parallel = [2][]uint16{make([]uint16, 15), make([]uint16, 15)}
	d.Campaign.DWords = make([]uint32, 15)
	d.Campaign.Scalars[0] = 10
	d.State = sav.CityStateData{RootKind: 17, Directories: map[string]uint32{}, Values: map[string]sav.CityStateValueData{}}
	for _, dir := range []string{"Character", "CurrentState", "GameOptions", "Inventory", "Objects", "SpellBook", "View"} {
		d.State.Directories["/"+dir] = 1
	}
	for _, path := range []string{"CurrentState/InBattle", "GameOptions/FlyingHP", "GameOptions/Formation", "GameOptions/ShowHP", "GameOptions/ShowTimeFlow", "GameOptions/Speed", "GameOptions/Wimpy", "Inventory/IsOpen", "SpellBook/IsOpen", "SpellBook/Pressed", "View/X", "View/Y"} {
		d.State.Values["/"+path] = sav.CityStateValueData{Kind: 2}
	}
	d.State.Values["/Character/Name"] = sav.CityStateValueData{Bytes: []byte("Leader\x00")}
	d.State.Values["/Objects/Selection"] = sav.CityStateValueData{Kind: 6}
	// Four empty shortcuts, not four copies of the populated index zero.
	// AI-QUICKSAVE-281 establishes -1 as the empty sentinel.
	shortcuts := make([]byte, 16)
	for i := range shortcuts {
		shortcuts[i] = 0xff
	}
	d.State.Values["/SpellBook/Shortcuts"] = sav.CityStateValueData{Kind: 6, Bytes: shortcuts}
	if graph {
		// Two Human backreferences form a cycle; the Item is a shared alias.
		d.Objects[1].Unit.Reference74 = 3
		d.Objects[2].Unit.Reference78 = 2
		d.Objects[1].Unit.Reference68 = 4
		d.Objects[2].Unit.Reference68 = 4
		d.Objects = append(d.Objects, sav.CityObjectData{Class: "Item", Item: &sav.CityItemData{Token: token(0x4567, 0), Fields: make([]byte, 12)}})
	}
	return d
}

func Original(graph bool) ([]byte, error) {
	p, err := sav.CityFromData(City(graph))
	if err != nil {
		return nil, err
	}
	update := sav.CityUpdate{Label: []byte("synthetic city"), Money: 123}
	for _, c := range p.Roster() {
		update.Characters = append(update.Characters, sav.CityCharacterUpdate{Identity: c.Identity, Name: c.Name, Stats: c.Stats, SkillLevels: c.SkillLevels, SkillXP: c.SkillXP, Experience: c.Experience})
	}
	return p.Marshal(update)
}
