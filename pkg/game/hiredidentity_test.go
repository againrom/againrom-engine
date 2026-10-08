package game

import (
	"encoding/binary"
	"fmt"
	"slices"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
)

func TestHiredActorNativeCityOrdinaryIdentity(t *testing.T) {
	f, _ := cityRosterHiredFixture(t, []int{12, 13}, []int{3, 4})
	for _, member := range f.Carried {
		if !member.Hired() {
			continue
		}
		unit := nativeCityUnitData(0x20, 0x10, member, f.Carried[0], f.Table, [sav.UnitStatWords]uint16{}, [sav.CharacterSkillSlots]uint16{})
		row := data.FindHumanByName(f.Table.Humans, member.Name)
		if got := binary.LittleEndian.Uint32(unit.Scalar2[47:51]); got != uint32(member.MercenaryType) {
			t.Errorf("%s U148=%d, want hired type %d", member.Name, got, member.MercenaryType)
		}
		if unit.Name != "" || int(unit.Token[16]) != row || binary.LittleEndian.Uint16(unit.Token[17:19]) != uint16(member.Class) || binary.LittleEndian.Uint16(unit.Token[23:25]) != 2 {
			t.Errorf("%s ordinary actor name=%q row=%d type=%d publication=%d", member.Name, unit.Name, unit.Token[16], binary.LittleEndian.Uint16(unit.Token[17:19]), binary.LittleEndian.Uint16(unit.Token[23:25]))
		}
	}
}

func TestHiredActorSourceCityOrdinaryIdentity(t *testing.T) {
	f, newFront := cityRosterHiredFixture(t, []int{12, 13}, []int{3, 4})
	for cycle := range 2 {
		s, label, err := f.Snapshot(false)
		if err != nil {
			t.Fatal(err)
		}
		raw, err := f.ExportCurrentSave(s, label)
		if err != nil {
			t.Fatal(err)
		}
		hiredActorWire(t, raw, f.Table, 1)
		ordinary := cityProjectionLoad(t, hiredActorWithoutSupplement(t, raw), newFront)
		hiredActorCounts(t, ordinary.Carried, 3, 4)
		cold := cityProjectionLoad(t, raw, newFront)
		cityRosterSame(t, f, cold)
		f = cold
		t.Logf("city cycle %d keeps native 3/4 identity and ordinary-only import", cycle)
	}
	screen := f.TownScreen().(*townScreen)
	for _, typ := range []int{12, 13} {
		for _, hired := range []bool{false, true} {
			if _, ok := screen.toggleMercenary(typ); !ok || f.Town.MercenaryHired(typ) != hired {
				t.Fatal("cancel/rehire", typ, hired)
			}
		}
	}
	hiredActorCounts(t, f.Carried, 3, 4)
}

func hiredActorWire(t *testing.T, raw []byte, table *mapload.Table, level int) {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	counts := map[int]int{}
	for _, record := range doc.Objects {
		if record.Class != "Human" {
			continue
		}
		values := map[string]uint32{}
		for _, value := range record.Values {
			values[value.Name] = value.Value
		}
		for _, typ := range []int{12, 13} {
			row := data.FindHumanByName(table.Humans, fmt.Sprintf("NPC%02d_%d", typ, level))
			if values["T0C"] != uint32(row) {
				continue
			}
			counts[typ]++
			name := ""
			for _, text := range record.Texts {
				if text.Name == "Name" {
					name = text.Value
				}
			}
			def, err := data.NewHumanDef(table.Humans.EntryName(row), table.Humans.EntryParams(row))
			if err != nil {
				t.Fatal(err)
			}
			if values["U148"] != uint32(typ) || name != "" || values["T0E"] != uint32(def.TypeID) || values["T18"] != 2 {
				t.Errorf("type %d ordinary actor U148=%d Name=%q TypeID=%d publication=%d", typ, values["U148"], name, values["T0E"], values["T18"])
			}
		}
	}
	if counts[12] != 3 || counts[13] != 4 {
		t.Fatal("ordinary hired population", counts)
	}
}

func hiredActorWithoutSupplement(t *testing.T, raw []byte) []byte {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	doc.State.ValueRecords = slices.DeleteFunc(doc.State.ValueRecords, func(row sav.CityStateRecordData) bool { return row.Path == sav.NativeActionsPath })
	raw, err = sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func hiredActorCounts(t *testing.T, party []mapload.PartyMember, first, second int) {
	t.Helper()
	counts := map[uint8]int{}
	for _, member := range party {
		counts[member.MercenaryType]++
	}
	if counts[12] != first || counts[13] != second {
		t.Fatal("current hired population", counts)
	}
}

func TestHiredActorOrdinaryMarkerAndLegacyNameBoundary(t *testing.T) {
	f, _ := cityRosterHiredFixture(t, []int{12, 13}, []int{3, 4})
	row := uint8(data.FindHumanByName(f.Table.Humans, "NPC12_1"))
	for _, test := range []struct {
		name     string
		backing  uint32
		ordinary string
		row      uint8
		want     bool
	}{
		{"anonymous", 12, "", row, true},
		{"low byte", 0x1000000c, "", row, true},
		{"named", 12, "Veteran", row, true},
		{"zero is not hired", 0, "", row, false},
		{"invalid type", 16, "", row, false},
		{"non NPC definition", 12, "", 5, false},
		{"legacy exact name", 0, "NPC12_1", row, true},
		{"legacy wrong row", 0, "NPC13_1", row, false},
		{"legacy prefix", 0, "NPC12_1 veteran", row, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			c := sav.Character{Class: "Human", DefRow: test.row, DisplayBacking: test.backing, Name: test.ordinary}
			got, hired := originalMercenaryType(c, f.Table)
			if hired != test.want || hired && got != 12 {
				t.Fatal("ordinary hire identity", got, hired)
			}
		})
	}
}

func TestHiredActorRenamedCityKeepsDefinitionAndHire(t *testing.T) {
	f, newFront := cityRosterHiredFixture(t, []int{12, 13}, []int{3, 4})
	rows := map[string]uint8{}
	for i := range f.Carried {
		member := &f.Carried[i]
		if !member.Hired() {
			continue
		}
		rows[member.ID] = member.DefinitionRow
		member.Name = fmt.Sprintf("Veteran %d", i)
	}
	for cycle := 0; cycle < 2; cycle++ {
		raw := currentTownSave(t, f)
		cold := cityProjectionLoad(t, raw, newFront)
		cityRosterSame(t, f, cold)
		for _, member := range cold.Carried {
			if member.Hired() && member.DefinitionRow != rows[member.ID] {
				t.Fatal("renamed hire changed template", member.ID, member.DefinitionRow, rows[member.ID])
			}
		}
		f = cold
	}
}

func TestHiredActorCityResidueTakesNoHireBits(t *testing.T) {
	current := sav.DocumentRecordData{Class: "Human", Values: []sav.DocumentValueData{{Name: "U148", Value: 13}}}
	source := sav.DocumentRecordData{Class: "Human", Values: []sav.DocumentValueData{{Name: "U148", Value: 0x12345600}}}
	mergeCityActorResidue(&current, &source, currentCityActorGraft{}, unknownRecordSpans())
	if current.Values[0].Value != 13 {
		t.Fatalf("hire word=%x, want the constructed 13", current.Values[0].Value)
	}
}

func TestHiredActorNamesProjectWithoutManifest(t *testing.T) {
	f, _ := cityRosterHiredFixture(t, []int{12, 13}, []int{3, 4})
	row := uint8(data.FindHumanByName(f.Table.Humans, "NPC12_1"))
	for _, test := range []struct {
		name string
		typ  uint8
		want string
	}{
		{"NPC12_1", 12, ""},
		{"Veteran", 12, "Veteran"},
		{"NPC12_1", 0, "NPC12_1"},
	} {
		member := mapload.PartyMember{Name: test.name, MercenaryType: test.typ, DefinitionRow: row}
		doc := sav.DocumentData{Objects: []sav.DocumentRecordData{{Class: "Human", Texts: []sav.DocumentTextData{{Name: "Name", Value: test.name}}}}}
		a := currentActionData{Party: []currentPartyMember{{Entity: 1, Member: &member}}, Bindings: []currentActionBinding{{ID: 1, Object: 1}}}
		projectCurrentHiredNames(&doc, &a, f.Table)
		if got := doc.Objects[0].Texts[0].Value; got != test.want || member.Name != test.name {
			t.Fatal("current hire name projection", got, test.want, member.Name)
		}
	}
}
