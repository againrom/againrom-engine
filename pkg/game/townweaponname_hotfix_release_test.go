package game

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
)

func TestReleaseTownWeaponNamesSurviveColdSAVLoad(t *testing.T) {
	_, source := groundCorpusFile(t, "2026-08-27/EXP-0261-owner-runs/game0005.sav", "9060508e3103aecb394f48ac206d45c388e86af24c1015013a294f7d2c03325f")
	f := releaseFront(t)
	if _, town, err := f.RestoreOriginal(source); err != nil || !town {
		t.Fatalf("load source town: town=%v err=%v", town, err)
	}
	// The hashed source's 0x0109 code is shape 0, material 0, weapon 9.
	// Read those installed cells directly instead of using the SAV producer.
	if f.Table.Shapes.EntryName(0) != "Common" || f.Table.Materials.EntryName(0) != "Iron" || f.Table.Weapons.EntryName(9) != "Mace" {
		t.Fatal("source mace definition changed", f.Table.Shapes.EntryName(0), f.Table.Materials.EntryName(0), f.Table.Weapons.EntryName(9))
	}
	want := map[string]data.Weapon{}
	for _, member := range f.Carried {
		if member.StartingHero || member.Weapon == nil || mapload.MemberItemEquipment(member, f.Table)[0].Code != 0x0109 {
			continue
		}
		if member.Weapon.Code != 0x0109 || member.Weapon.Name != "Common Iron Mace" {
			t.Fatalf("source hired weapon %q: %+v", member.ID, member.Weapon)
		}
		if name := partyPanelSubject(member, f.Table).Char.Weapon; name != member.Weapon.Name {
			t.Fatalf("source hired %q panel name %q, want %q", member.ID, name, member.Weapon.Name)
		}
		want[member.ID] = *member.Weapon
	}
	if len(want) != 3 {
		t.Fatalf("source has %d hired Iron Maces, want 3", len(want))
	}
	for cycle := 0; cycle < 2; cycle++ {
		f = currentTownReload(t, currentTownSave(t, f))
		found := 0
		for _, member := range f.Carried {
			before, ok := want[member.ID]
			if !ok {
				continue
			}
			found++
			if member.Weapon == nil || *member.Weapon != before || mapload.MemberItemEquipment(member, f.Table)[0].Code != 0x0109 {
				t.Fatalf("cycle %d hired %q changed weapon: before %+v after %+v", cycle+1, member.ID, before, member.Weapon)
			}
			if name := partyPanelSubject(member, f.Table).Char.Weapon; name != before.Name {
				t.Fatalf("cycle %d hired %q panel name %q, want %q", cycle+1, member.ID, name, before.Name)
			}
		}
		if found != len(want) {
			t.Fatalf("cycle %d kept %d hired weapons, want %d", cycle+1, found, len(want))
		}
	}
}
