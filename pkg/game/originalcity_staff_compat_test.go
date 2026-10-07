package game

import (
	"encoding/binary"
	"reflect"
	"strings"
	"testing"

	"againrom/internal/cityfixture"
	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
)

func staffCityFixture(t *testing.T, weaponName string) (*FrontEnd, Snapshot, data.Weapon) {
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
	model.Objects[2].Unit.Equipment[0] = 4
	model.Objects = append(model.Objects,
		sav.CityObjectData{Class: "Weapon", Item: &sav.CityItemData{Token: token(0x4567), Fields: fields, Derived: make([]byte, 47), Effects: []uint16{5}}},
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
	if weaponName == "Wood Staff" {
		weapon, err = data.ResolveWeapon("Wood Staff {castSpell=Fire_Arrow:10}", table.Shapes, table.Materials, table.Weapons)
		if err != nil {
			t.Fatal(err)
		}
	}
	return f, s, weapon
}

func TestHistoricalCityStaffBaselineKeepsBothExactEncodings(t *testing.T) {
	for _, weaponName := range []string{"Wood Staff", "Bone Staff"} {
		for _, policy := range []string{"historical-current-party", "historical-old-party", "emitted-current-source", "new-current-source"} {
			t.Run(weaponName+"/"+policy, func(t *testing.T) {
				f, s, oldWeapon := staffCityFixture(t, weaponName)
				if policy != "new-current-source" {
					s.OriginalCity.Version = 7
				} else if s.OriginalCity.Version != 8 {
					t.Fatal("new import did not identify its current-effect baseline policy")
				}
				if policy == "historical-current-party" || policy == "historical-old-party" {
					for i := range s.OriginalCity.Bindings {
						p := &s.OriginalCity.Bindings[i].Baseline
						if p.StartingHero {
							w, h := oldWeapon, oldWeapon
							p.Weapon, p.OriginalHuman.Weapon = &w, &h
						}
					}
				}
				if policy == "historical-old-party" {
					w, h := oldWeapon, oldWeapon
					s.Party[0].Weapon, s.Party[0].OriginalHuman.Weapon = &w, &h
				}
				want := mapload.CloneParty(s.Party)
				for cut := 0; cut < 2; cut++ {
					raw, err := EncodeSave(s, "historical staff")
					if err != nil {
						t.Fatal(err)
					}
					decoded, _, err := DecodeSave(raw)
					if err != nil {
						t.Fatal(err)
					}
					before := decoded.OriginalCity
					fresh := &FrontEnd{InstallResources: f.InstallResources}
					if _, town, err := fresh.Restore(decoded); err != nil || !town {
						t.Fatal("historical LOAD", town, err)
					}
					s, _, err = fresh.Snapshot(false)
					if err != nil || !reflect.DeepEqual(s.Party, want) || !reflect.DeepEqual(s.OriginalCity, before) {
						t.Fatalf("LOAD changed party=%t baseline=%t: %v", !reflect.DeepEqual(s.Party, want), !reflect.DeepEqual(s.OriginalCity, before), err)
					}
				}
			})
		}
	}
}

func TestHistoricalCityStaffBaselineRejectsPartialAndForgedEncodings(t *testing.T) {
	for _, name := range []string{"weapon-only", "human-only", "other-weapon-field", "other-human-field", "item-effect", "wrong-power", "current-version"} {
		t.Run(name, func(t *testing.T) {
			f, s, oldWeapon := staffCityFixture(t, "Bone Staff")
			s.OriginalCity.Version = 7
			for i := range s.OriginalCity.Bindings {
				p := &s.OriginalCity.Bindings[i].Baseline
				if !p.StartingHero {
					continue
				}
				w, h := oldWeapon, oldWeapon
				if name != "human-only" {
					p.Weapon = &w
				}
				if name != "weapon-only" {
					p.OriginalHuman.Weapon = &h
				}
				switch name {
				case "other-weapon-field":
					p.Weapon.DamageBase++
				case "other-human-field":
					p.OriginalHuman.State.Modifier.Attack.Tail[0]++
				case "item-effect":
					p.WornItems[0].Effects[0].Operand++
				case "wrong-power":
					p.Weapon.SpellPower, p.OriginalHuman.Weapon.SpellPower = 11, 11
				case "current-version":
					s.OriginalCity.Version = 8
				}
			}
			before, town := mapload.CloneParty(f.Carried), f.Town
			if _, _, err := f.Restore(s); err == nil || !strings.Contains(err.Error(), "differs from semantic source/import context") {
				t.Fatal("forged historical baseline did not fail equality", err)
			}
			if !reflect.DeepEqual(f.Carried, before) || f.Town != town {
				t.Fatal("refused baseline changed the live session")
			}
		})
	}
}
