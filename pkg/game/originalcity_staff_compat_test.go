package game

import (
	"encoding/binary"
	"reflect"
	"testing"

	"againrom/internal/cityfixture"
	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
)

func staffCityFixture(t *testing.T, weaponName string) (*FrontEnd, Snapshot) {
	t.Helper()
	params := make([]int32, data.MinHumanRow)
	params[16], params[17], params[18] = 0x17, 4, 1
	rows := make(dbCollection, 30)
	rows[28], rows[29] = dbEntry{name: "PC_Leader", params: params}, dbEntry{name: "PC_Companion", params: params}
	table := &mapload.Table{Humans: rows, Shapes: emptyScale{}, Materials: emptyScale{},
		Weapons: dbCollection{{}, {name: "Wood Staff", params: chargenWeaponParams(data.SkillBlade)}, {name: "Bone Staff", params: chargenWeaponParams(data.SkillBlade)}},
		Spells:  dbCollection{{}, {name: "Fire Arrow"}, {name: "Stone Curse"}}}
	weapon, err := data.ResolveWeapon(weaponName, table.Shapes, table.Materials, table.Weapons)
	if err != nil {
		t.Fatal(err)
	}
	model := cityfixture.City(false)
	for _, obj := range model.Objects {
		if u := obj.Unit; u != nil {
			u.SpellbookFlag, u.SpellbookCount, u.Spells = 1, 27, make([]uint16, 26)
			u.Scalar1[3] = 4
		}
	}
	token := func(key uint32) []byte {
		b := make([]byte, 37)
		binary.LittleEndian.PutUint32(b[29:], key)
		return b
	}
	fields, effect := make([]byte, 12), []byte{41, 0, 2, 0, 0xfe, 0xff, 0}
	binary.LittleEndian.PutUint16(fields, uint16(weapon.Code))
	binary.LittleEndian.PutUint16(fields[2:], 1)
	fields[4] = 2
	derived := make([]byte, 47)
	derived[14], derived[15], derived[46] = 10, 10, 1
	model.Objects[2].Unit.Equipment[0] = 4
	model.Objects = append(model.Objects,
		sav.CityObjectData{Class: "Weapon", Item: &sav.CityItemData{Token: token(0x4567), Fields: fields, Derived: derived, Effects: []uint16{5}}},
		sav.CityObjectData{Class: "Effect", Effect: &sav.CityEffectData{Token: token(0x5678), Fields: effect}})
	document, err := sav.CityFromData(model)
	if err != nil {
		t.Fatal(err)
	}
	update := sav.CityUpdate{Money: 123}
	for _, c := range document.Roster() {
		update.Characters = append(update.Characters, originalCityBaselineUpdate(c))
	}
	raw, err := document.Marshal(update)
	if err != nil {
		t.Fatal(err)
	}
	f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(saveCampaign(), nil), Table: table}}
	if _, town, err := f.RestoreOriginal(raw); err != nil || !town {
		t.Fatal("source import", town, err)
	}
	s, _, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	if s.Party[0].Weapon.SpellName != "Stone_Curse" || s.Party[0].Weapon.SpellPower != -2 || s.Party[0].OriginalHuman == nil {
		t.Fatalf("source fixture lost current signed effect: weapon=%+v human=%t", s.Party[0].Weapon, s.Party[0].OriginalHuman != nil)
	}
	current := mapload.MemberWeapon(s.Party[0], f.Table)
	if current == nil || current.DamageBase != 10 || current.DamageSpread != 10 || current.Range != 1 || current.SpellPower != -2 {
		t.Fatalf("source concrete staff disagrees with independent fixture: %+v", current)
	}
	return f, s
}

func TestCurrentCitySignedStaffSurvivesTwoSAVCycles(t *testing.T) {
	for _, weaponName := range []string{"Wood Staff", "Bone Staff"} {
		t.Run(weaponName, func(t *testing.T) {
			f, s := staffCityFixture(t, weaponName)
			bindCurrentCityFixtureRow(t, f)
			want := mapload.CloneParty(s.Party)
			for cut := 0; cut < 2; cut++ {
				before := mapload.CloneParty(f.Carried)
				raw, err := f.ExportCurrentSave(s, "current signed staff")
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(f.Carried, before) {
					t.Fatal("SAVE changed current staff source")
				}
				fresh := &FrontEnd{InstallResources: f.InstallResources}
				if _, town, err := fresh.RestoreOriginal(raw); err != nil || !town {
					t.Fatal("current staff SAV LOAD", town, err)
				}
				if !reflect.DeepEqual(fresh.Carried, want) {
					currentItemFieldDiagnostics(t, "Party", reflect.ValueOf(want), reflect.ValueOf(fresh.Carried))
					t.Fatal("current staff SAV changed party")
				}
				f = fresh
				s, _, err = f.Snapshot(false)
				if err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}
