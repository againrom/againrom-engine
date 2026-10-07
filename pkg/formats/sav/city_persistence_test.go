package sav_test

import (
	"bytes"
	"encoding/binary"
	"strings"
	"testing"

	"againrom/internal/cityfixture"
	"againrom/pkg/formats/sav"
)

func TestCityDataDetachedGraphAliasesAndRemintedRebuild(t *testing.T) {
	d := cityfixture.City(true)
	p, err := sav.CityFromData(d)
	if err != nil {
		t.Fatal(err)
	}
	d.Objects[1].Unit.Scalar2[0] = 99
	d.State.Values["/SpellBook/Shortcuts"].Bytes[0] = 99
	d.Campaign.Parallel[0][0] = 99
	copy := p.Data()
	if copy.Objects[1].Unit.Scalar2[0] != 31 || cityStateValue(t, copy, "/SpellBook/Shortcuts").Bytes[0] != 0xff || copy.Campaign.Parallel[0][0] != 0 {
		t.Fatal("input aliases retained")
	}
	copy.Objects[1].Unit.Name = "not the character"
	cityStateValue(t, copy, "/SpellBook/Shortcuts").Bytes[0] = 88
	copy.State.DirectoryRecords[0].Kind = 99
	if p.Data().Objects[1].Unit.Name != "Companion" {
		t.Fatal("Data shares storage")
	}
	if cityStateValue(t, p.Data(), "/SpellBook/Shortcuts").Bytes[0] != 0xff || p.Data().State.DirectoryRecords[0].Kind != 1 {
		t.Fatal("state records share storage")
	}
	first, err := cityfixture.Original(true)
	if err != nil {
		t.Fatal(err)
	}
	second, err := cityfixture.Original(true)
	if err != nil || !bytes.Equal(first, second) {
		t.Fatalf("nondeterministic rebuild: %v", err)
	}
	f, err := sav.Open(first)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := f.CityProvenance()
	if err != nil {
		t.Fatal(err)
	}
	rebuilt := restored.Data()
	if len(rebuilt.Objects) != 4 {
		t.Fatalf("object count %d", len(rebuilt.Objects))
	}
	var companion, leader *sav.CityUnitData
	for _, o := range rebuilt.Objects {
		if o.Unit != nil {
			if o.Unit.Name == "Leader" {
				leader = o.Unit
			} else {
				companion = o.Unit
			}
		}
	}
	if companion == nil || leader == nil || companion.Reference68 != leader.Reference68 || rebuilt.Objects[companion.Reference74-1].Unit != leader || rebuilt.Objects[leader.Reference78-1].Unit != companion {
		t.Fatal("cycle or shared alias changed")
	}
	if binary.LittleEndian.Uint32(leader.Token[29:]) == 0x3456 || binary.LittleEndian.Uint16(leader.Scalar2) != 41 {
		t.Fatal("identity not reminted or value changed")
	}
}

func TestCityDataRejectsMalformedWholeModels(t *testing.T) {
	tests := map[string]func(*sav.CityData){
		"future":       func(d *sav.CityData) { d.Version = sav.CityDataVersion + 1 },
		"file version": func(d *sav.CityData) { d.FileVersion = 0 },
		"mission":      func(d *sav.CityData) { d.Head[11] = 10 },
		"missing body": func(d *sav.CityData) { d.Objects[1].Unit = nil },
		"extra body":   func(d *sav.CityData) { d.Objects[1].Spell = &sav.CitySpellData{} },
		"foreign ref":  func(d *sav.CityData) { d.Objects[1].Unit.Reference74 = 99 },
		"unreachable": func(d *sav.CityData) {
			d.Objects = append(d.Objects, sav.CityObjectData{Class: "Diary", Diary: &sav.CityDiaryData{}})
		},
		"field length":      func(d *sav.CityData) { d.Objects[1].Unit.RawD4 = nil },
		"equipment count":   func(d *sav.CityData) { d.Objects[1].Unit.Equipment = nil },
		"spell count":       func(d *sav.CityData) { d.Objects[1].Unit.SpellbookFlag = 1; d.Objects[1].Unit.SpellbookCount = 3 },
		"inactive field":    func(d *sav.CityData) { d.Objects[1].Unit.Spells = []uint16{2} },
		"identity relation": func(d *sav.CityData) { binary.LittleEndian.PutUint32(d.Objects[0].Player.Fixed[43:], 0xdead) },
		"parallel count":    func(d *sav.CityData) { d.Campaign.Parallel[0] = nil },
		"state shape":       func(d *sav.CityData) { delete(d.State.Values, "/View/X") },
		"state alignment": func(d *sav.CityData) {
			d.State.Values["/Objects/Selection"] = sav.CityStateValueData{Kind: 6, Bytes: []byte{1}}
		},
		"objects bound":  func(d *sav.CityData) { d.Objects = make([]sav.CityObjectData, 4097) },
		"elements bound": func(d *sav.CityData) { d.Objects[1].Unit.Words15c = make([]uint16, 1<<16) },
		"bytes bound":    func(d *sav.CityData) { d.MapName = strings.Repeat("x", (4<<20)+1) },
		"depth": func(d *sav.CityData) {
			for i := 0; i < 70; i++ {
				d.Objects = append(d.Objects, sav.CityObjectData{Class: "Weapon", Item: &sav.CityItemData{WeaponExtra: uint16(len(d.Objects) + 2)}})
			}
			d.Objects[1].Unit.Reference74 = 4
		},
	}
	for name, change := range tests {
		t.Run(name, func(t *testing.T) {
			d := cityfixture.City(false)
			change(&d)
			if _, err := sav.CityFromData(d); err == nil {
				t.Fatal("accepted malformed DTO")
			}
		})
	}
}

func cityStateValue(t *testing.T, d sav.CityData, path string) sav.CityStateValueData {
	t.Helper()
	for _, record := range d.State.ValueRecords {
		if record.Path == path {
			return record.Value
		}
	}
	t.Fatalf("missing state value %s", path)
	return sav.CityStateValueData{}
}

func TestCityDataOrderedStateRejectsAmbiguityAndBounds(t *testing.T) {
	for name, change := range map[string]func(*sav.CityData){
		"directory order": func(d *sav.CityData) {
			d.State.DirectoryRecords[0], d.State.DirectoryRecords[1] = d.State.DirectoryRecords[1], d.State.DirectoryRecords[0]
		},
		"value order": func(d *sav.CityData) {
			d.State.ValueRecords[0], d.State.ValueRecords[1] = d.State.ValueRecords[1], d.State.ValueRecords[0]
		},
		"duplicate directory":      func(d *sav.CityData) { d.State.DirectoryRecords[1] = d.State.DirectoryRecords[0] },
		"duplicate value":          func(d *sav.CityData) { d.State.ValueRecords[1] = d.State.ValueRecords[0] },
		"legacy map ambiguity":     func(d *sav.CityData) { d.State.Directories = map[string]uint32{"/Character": 1} },
		"legacy version ambiguity": func(d *sav.CityData) { d.Version = 1 },
		"record bound":             func(d *sav.CityData) { d.State.ValueRecords = make([]sav.CityStateRecordData, (1<<16)+1) },
	} {
		t.Run(name, func(t *testing.T) {
			p, err := sav.CityFromData(cityfixture.City(false))
			if err != nil {
				t.Fatal(err)
			}
			d := p.Data()
			change(&d)
			if _, err := sav.CityFromData(d); err == nil {
				t.Fatal("ambiguous state accepted")
			}
		})
	}
}

func TestCityDataSourcePartyBoundsExpandedAliases(t *testing.T) {
	d := cityfixture.City(true)
	// Reuse the single item twice as a full stack. The compact graph is small,
	// but importing both into independent loadouts must be rejected up front.
	d.Objects[1].Unit.Reference74, d.Objects[2].Unit.Reference78 = 0, 0
	d.Objects[1].Unit.Container = []uint16{4, 4}
	binary.LittleEndian.PutUint16(d.Objects[3].Item.Fields[2:], 65535)
	p, err := sav.CityFromData(d)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.SourceParty(); err == nil || !strings.Contains(err.Error(), "expansion") {
		t.Fatalf("unbounded source projection: %v", err)
	}
}
