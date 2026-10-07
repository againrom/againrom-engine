package game

import (
	"bytes"
	"testing"

	"againrom/pkg/formats/sav"
)

func creatureDocumentRecord(t *testing.T, doc sav.DocumentData, mapID uint32) *sav.DocumentRecordData {
	t.Helper()
	var found *sav.DocumentRecordData
	for i := range doc.Objects {
		r := &doc.Objects[i]
		if r.Class != "Unit" {
			continue
		}
		id, err := savedStructureValue(r, "T08")
		if err == nil && id == mapID {
			if found != nil {
				t.Fatalf("map unit %d has repeated Unit records", mapID)
			}
			found = r
		}
	}
	if found == nil {
		t.Fatalf("map unit %d has no Unit record", mapID)
	}
	return found
}

func requireCreatureCombatDocument(t *testing.T, doc sav.DocumentData) {
	t.Helper()
	for _, mapID := range []uint32{1098, 1099} {
		r := creatureDocumentRecord(t, doc, mapID)
		flags, err := savedStructureValue(r, "U4C")
		if err != nil || flags != 0x12 {
			t.Fatalf("unit %d U4C=%#x err=%v, want 0x12", mapID, flags, err)
		}
		weapons, _ := savedObjectRefs(r, "HeldWeapon")
		if len(weapons) != 1 || weapons[0] == 0 || int(weapons[0]) > len(doc.Objects) || doc.Objects[weapons[0]-1].Class != "Weapon" {
			t.Fatalf("unit %d HeldWeapon=%v, want a live Weapon object", mapID, weapons)
		}
		weaponCode, err := savedStructureValue(&doc.Objects[weapons[0]-1], "F40")
		if err != nil || weaponCode != 0xf118 {
			t.Fatalf("unit %d held weapon code=%#x err=%v, want Sonic Beam", mapID, weaponCode, err)
		}
		var attack []byte
		for _, raw := range r.Raw {
			if raw.Name == "UA6" {
				attack = raw.Bytes
			}
		}
		wantAttack := []byte{0x45, 0x01, 0x40, 0x01, 0x1e, 0, 0x1e, 0, 0x1e, 0, 0x1e, 0, 0x1e, 0, 0x04, 0x0c, 0x05, 0, 0, 0, 0, 0, 0, 0}
		if !bytes.Equal(attack, wantAttack) {
			t.Fatalf("unit %d UA6=%x, want %x", mapID, attack, wantAttack)
		}
		spells, _ := savedObjectRefs(r, "Spells")
		if len(spells) < 17 || spells[16] == 0 || int(spells[16]) > len(doc.Objects) {
			t.Fatalf("unit %d Spell 16 reference=%v", mapID, spells)
		}
		spell := &doc.Objects[spells[16]-1]
		id, err := savedStructureValue(spell, "S08")
		if spell.Class != "Spell" || err != nil || id != 17 {
			t.Fatalf("unit %d book slot 16 = %s/%d err=%v", mapID, spell.Class, id, err)
		}
		for field, want := range map[string]uint32{"S09": 6, "S0A": 0, "S0C": 5} {
			value, err := savedStructureValue(spell, field)
			if err != nil || value != want {
				t.Fatalf("unit %d book slot 16 %s=%d err=%v, want %d", mapID, field, value, err, want)
			}
		}
	}
}

func TestReleaseCreatureInnateCombatSAV(t *testing.T) {
	f := releaseFront(t)
	if err := f.App("creature combat").OpenMission(f.MissionOpener(140)); err != nil {
		t.Fatal(err)
	}
	first := saveRoodMission(t, f)
	firstDoc, err := sav.DecodeDocumentData(first)
	if err != nil {
		t.Fatal(err)
	}
	requireCreatureCombatDocument(t, firstDoc)
	cold := loadRoodMission(t, first)
	for range 32 {
		cold.live.tick()
	}
	second := saveRoodMission(t, cold)
	secondDoc, err := sav.DecodeDocumentData(second)
	if err != nil {
		t.Fatal(err)
	}
	requireCreatureCombatDocument(t, secondDoc)
}
