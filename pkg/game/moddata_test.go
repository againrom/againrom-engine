package game

import (
	"slices"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mod"
)

// The apply step and the base dump cover exactly the declared kinds: every
// kind has an apply step, the dump writes exactly the kinds that declare a
// dump, each under the kind's file name.
func TestModDataStepsCoverTheDeclaredKinds(t *testing.T) {
	var names, dumped []string
	for _, k := range mod.DataKinds() {
		names = append(names, k.Name)
		if k.Dump {
			dumped = append(dumped, k.Name)
		}
	}
	for name := range modDataAppliers {
		if !slices.Contains(names, name) {
			t.Errorf("an apply step for %q, which is not a declared kind", name)
		}
	}
	for _, name := range names {
		if modDataAppliers[name] == nil {
			t.Errorf("kind %q has no apply step", name)
		}
	}
	for name := range baseDumpLayers {
		if !slices.Contains(dumped, name) {
			t.Errorf("a base dump of %q, which declares none", name)
		}
	}
	for _, name := range dumped {
		if baseDumpLayers[name] == nil {
			t.Errorf("kind %q declares a dump the game does not write", name)
		}
	}
}

// The one apply step merges the companions of two mods: the later one wins
// when load-after orders it, the step names the override, and without the
// order the step refuses.
func TestSetModDataMergesCompanionLayers(t *testing.T) {
	later := joinOf(30, 22, 5)
	later.Mod, later.Line = "n", 7
	later.Fields = []mod.PatchField{{Name: "talk", Line: 9}}
	d := mod.Data{Companions: mod.CompanionData{Joins: []mod.CompanionJoin{joinOf(30, 22, 22), later}, Order: mod.Precedence{"n": {"m"}}}}
	f := joinFixtureFront(t)
	report, err := f.SetModData(d)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{`companion 22 of chapter 30 talk: mod "n" (data/companions.toml:9) overrides mod "m"`}; !slices.Equal(report.Overrides, want) {
		t.Fatalf("overrides %q, want %q", report.Overrides, want)
	}
	if got := f.Table.Mods.Companions.ForTalk(30, mod.BuildingTavern, 5); len(got) != 1 || got[0].Key != "k" {
		t.Fatalf("merged %+v", f.Table.Mods.Companions)
	}
	d.Companions.Order = nil
	_, err = joinFixtureFront(t).SetModData(d)
	if err == nil || !strings.Contains(err.Error(), `mod "n": data/companions.toml:7: companion 22 of chapter 30 is added by mod "m" and named by mod "n", and neither loads after the other`) {
		t.Fatalf("unordered: %v", err)
	}
}

// Two mods that choose a body for one weapon row: with load-after the later
// choice is drawn and named as an override; without an order the merge
// refuses; one mod naming one row twice is refused as before.
func TestWeaponBodyLayersFollowLoadAfter(t *testing.T) {
	src, units, weapons, list := bodyInstall(t)
	first, err := mod.ParseWeaponBodies("a", mod.WeaponBodiesFile, []byte("[[weapon]]\nweapon = \"Sword\"\nbody = \"clubman\"\n"))
	if err != nil {
		t.Fatal(err)
	}
	second, err := mod.ParseWeaponBodies("b", mod.WeaponBodiesFile, []byte("[[weapon]]\nrow = 2\nbody = \"unarmed\"\n"))
	if err != nil {
		t.Fatal(err)
	}
	d := mod.BodyData{Weapons: append(first, second...), Order: mod.Precedence{"b": {"a"}}}
	got, _, overrides, err := layerModBodies(src, units, weapons, list, d)
	if err != nil {
		t.Fatal(err)
	}
	var e data.Equipment
	e.SetCode(1, data.ItemCode(2))
	if b, _ := data.HeroBodyFor(got, e); b != data.BodyUnarmed {
		t.Fatalf("row 2 draws %q", b)
	}
	if len(overrides) != 1 || overrides[0].String() != `weapon row 2 body: mod "b" (data/wpnbodies.toml:3) overrides mod "a"` {
		t.Fatalf("overrides %v", overrides)
	}
	d.Order = nil
	_, _, _, err = layerModBodies(src, units, weapons, list, d)
	if err == nil || err.Error() != `mod "b": data/wpnbodies.toml:1: weapon row 2 body is set by mod "a" and by mod "b", and neither loads after the other: add load-after to one of them` {
		t.Fatalf("unordered: %v", err)
	}
}
