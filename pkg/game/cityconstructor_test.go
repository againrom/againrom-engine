package game

import (
	"bytes"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
)

func TestCurrentCityMovementMissingDefinitionUsesDefaults(t *testing.T) {
	for _, tc := range []struct {
		name  string
		row   byte
		table *mapload.Table
	}{
		{name: "no table"},
		{name: "no collection", table: &mapload.Table{}},
		{name: "no selected row", table: &mapload.Table{Humans: dbCollection{{}, {name: "actor"}}}},
		{name: "outside collection", row: 7, table: &mapload.Table{Humans: dbCollection{{}, {name: "actor"}}}},
		{name: "absent parameters", row: 1, table: &mapload.Table{Humans: dbCollection{{}, {name: "actor"}}}},
		{name: "short parameters", row: 1, table: &mapload.Table{Humans: dbCollection{{}, {name: "actor", params: []int32{30, 20}}}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			u := sav.CityUnitData{Token: make([]byte, 37), Scalar1: make([]byte, 19), Raw154: make([]byte, 180)}
			u.Token[16] = tc.row
			before := append([]byte(nil), u.Token...)
			if err := nativeCityInitializeHumanMovement(&u, tc.table); err != nil {
				t.Fatal(err)
			}
			if u.Scalar1[0] != 1 || u.Scalar1[1] != 1 || u.Raw154[5] != 0x41 || !reflect.DeepEqual(before, u.Token) {
				t.Fatal("missing definition changed constructor movement or definition identity")
			}
		})
	}
	params := make([]int32, 23)
	for i := range params {
		params[i] = -1
	}
	params[21], params[22] = 3, 3
	table := &mapload.Table{Humans: dbCollection{{}, {name: "partial", params: params}}}
	u := sav.CityUnitData{Token: make([]byte, 37), Scalar1: make([]byte, 19), Raw154: make([]byte, 180)}
	u.Token[16] = 1
	if err := nativeCityInitializeHumanMovement(&u, table); err != nil || u.Scalar1[0] != 3 || u.Scalar1[1] != 3 || u.Raw154[5] != 0x82 {
		t.Fatal("present columns in partial row replaced by defaults", err)
	}
	params[22] = 0
	if err := nativeCityInitializeHumanMovement(&u, table); err == nil {
		t.Fatal("explicit invalid domain treated as absent")
	}
}

func TestCurrentCityGraftPreservesExplicitMovement(t *testing.T) {
	for _, tuple := range [][3]byte{{0, 0, 0}, {7, 2, 0x19}} {
		source, current := cityGraftFixture(t, false), cityGraftFixture(t, false)
		old := cityGraftNamed(t, &source, "Leader")
		index := cityGraftNamed(t, &current, "Leader")
		mustSetValue(&source.Objects[old-1], "U49", uint32(tuple[0]))
		mustSetValue(&source.Objects[old-1], "U4A", uint32(tuple[1]))
		cityGraftRaw(t, &source.Objects[old-1], "U154")[5] = tuple[2]
		mustSetValue(&current.Objects[index-1], "U49", 1)
		mustSetValue(&current.Objects[index-1], "U4A", 1)
		cityGraftRaw(t, &current.Objects[index-1], "U154")[5] = 0x41
		for cycle := 0; cycle < 2; cycle++ {
			mergeCityActorResidue(&current.Objects[index-1], &source.Objects[old-1], currentCityActorGraft{})
			wire, err := sav.EncodeDocumentData(current)
			if err != nil {
				t.Fatal(err)
			}
			current, err = sav.DecodeDocumentData(wire)
			if err != nil {
				t.Fatal(err)
			}
			index = cityGraftNamed(t, &current, "Leader")
			r := &current.Objects[index-1]
			if cityGraftValue(t, r, "U49") != uint32(tuple[0]) || cityGraftValue(t, r, "U4A") != uint32(tuple[1]) || cityGraftRaw(t, r, "U154")[5] != tuple[2] {
				t.Fatal("constructor replaced explicitly saved movement", cycle, tuple)
			}
		}
	}
}

func TestCurrentCityOpaqueHumanTailsFollowFieldAuthority(t *testing.T) {
	for _, mode := range []string{"live load", "retired member", "retained document", "constructor"} {
		t.Run(mode, func(t *testing.T) {
			f := currentTrainingCity(t)
			s, _, err := f.Snapshot(false)
			if err != nil {
				t.Fatal(err)
			}
			var member *mapload.PartyMember
			for i := range s.Party {
				if s.Party[i].ID == "hero" {
					member = &s.Party[i]
				}
			}
			if member == nil || member.OriginalHuman == nil || member.Carry.LiveLoad == nil {
				t.Fatal("fixture lacks distinct current and retained Human carriers")
			}
			want := [3][2]byte{{0xab, 0xcd}, {0x12, 0x34}, {0xde, 0xad}}
			switch mode {
			case "live load":
				want = [3][2]byte{{0x91, 0x92}, {0x93, 0x94}, {0x95, 0x96}}
				source := &member.Carry.LiveLoad.Inventory.Source
				copy(source.Attack[22:24], want[0][:])
				copy(source.Base[22:24], want[1][:])
				copy(source.Modifier[40:42], want[2][:])
			case "retired member":
				member.Carry.LiveLoad = nil
				member.OriginalHuman.Retired = true
				want = [3][2]byte{{0x81, 0x82}, {0x83, 0x84}, {0x85, 0x86}}
				h := &member.OriginalHuman.State
				h.Attack.Tail, h.Base.Tail, h.Modifier.Attack.Tail = want[0], want[1], want[2]
				h.Attack.ToHit, h.Health, h.Experience = 60000, 60000, 60000
			case "retained document", "constructor":
				member.Carry.LiveLoad, member.OriginalHuman = nil, nil
				if mode == "constructor" {
					s.OriginalCity = nil
					want = [3][2]byte{}
				}
			}
			d, hp, mp := mapload.PartyDisplayWithTable(*member, f.Table)
			record := func(doc *sav.DocumentData) *sav.DocumentRecordData {
				t.Helper()
				a, err := readCurrentActions(doc)
				if err != nil || a == nil {
					t.Fatal("current city lacks explicit actor bindings", err)
				}
				for _, p := range a.Party {
					if string(p.ID) != "hero" {
						continue
					}
					for _, b := range a.Bindings {
						if b.ID == p.Entity && !b.Structure && !b.Missing && b.Object != 0 && int(b.Object) <= len(doc.Objects) {
							return &doc.Objects[b.Object-1]
						}
					}
				}
				t.Fatal("current hero has no ordinary Human record")
				return nil
			}
			check := func(doc *sav.DocumentData, tails [3][2]byte) {
				t.Helper()
				r := record(doc)
				for i, name := range []string{"UA6", "U114", "UD4"} {
					offset := 22
					if name == "UD4" {
						offset = 40
					}
					if got := cityGraftRaw(t, r, name)[offset : offset+2]; !bytes.Equal(got, tails[i][:]) {
						t.Fatalf("%s tail = %x, want %x", name, got, tails[i])
					}
				}
			}
			roundtrip := func(tails [3][2]byte) sav.DocumentData {
				t.Helper()
				before := mapload.CloneParty(s.Party)
				raw, err := f.ExportOriginalSave(s, "current opaque Human tails")
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(before, s.Party) {
					t.Fatal("SAVE changed the input party")
				}
				doc, err := sav.DecodeDocumentData(raw)
				if err != nil {
					t.Fatal(err)
				}
				check(&doc, tails)
				cold := &FrontEnd{InstallResources: f.InstallResources, RuntimeServices: f.RuntimeServices}
				if _, town, err := cold.RestoreOriginal(raw); err != nil || !town {
					t.Fatal("cold city LOAD", town, err)
				}
				got, gotHP, gotMP := mapload.PartyDisplayWithTable(trainingPartyMember(t, cold, "hero"), cold.Table)
				if !reflect.DeepEqual(d, got) || hp != gotHP || mp != gotMP {
					t.Fatal("opaque tails displaced current Human operands", d, got, hp, gotHP, mp, gotMP)
				}
				f = cold
				s, _, err = f.Snapshot(false)
				if err != nil {
					t.Fatal(err)
				}
				return doc
			}
			roundtrip(want)
			doc := roundtrip(want)
			for _, editedTails := range [][3][2]byte{{}, {{0x41, 0x42}, {0x43, 0x44}, {0x45, 0x46}}} {
				leaf, _, err := sav.NativeActions(doc.State)
				if err != nil {
					t.Fatal(err)
				}
				r := record(&doc)
				for i, name := range []string{"UA6", "U114", "UD4"} {
					offset := 22
					if name == "UD4" {
						offset = 40
					}
					copy(cityGraftRaw(t, r, name)[offset:offset+2], editedTails[i][:])
				}
				raw, err := sav.EncodeDocumentData(doc)
				if err != nil {
					t.Fatal(err)
				}
				unchanged, _, err := sav.NativeActions(doc.State)
				if err != nil || !bytes.Equal(leaf, unchanged) {
					t.Fatal("ordinary tail edit changed private policy", err)
				}
				cold := &FrontEnd{InstallResources: f.InstallResources, RuntimeServices: f.RuntimeServices}
				if _, town, err := cold.RestoreOriginal(raw); err != nil || !town {
					t.Fatal("ordinary tail edit LOAD", town, err)
				}
				f = cold
				s, _, err = f.Snapshot(false)
				if err != nil {
					t.Fatal(err)
				}
				roundtrip(editedTails)
				doc = roundtrip(editedTails)
			}
		})
	}
}
