package game

import (
	"strings"
	"testing"

	"againrom/pkg/data"
)

// damageRowMod writes the mod folder id holding one weapon row over the Pike
// with the given damage lines.
func damageRowMod(t *testing.T, dir, id, damage string) {
	t.Helper()
	itemRowsMod(t, dir, id, map[string]string{
		"data/items.toml":      "[[weapon]]\nkey = \"hard_pike\"\nname = \"item.hard_pike\"\nbase = \"Pike\"\n" + damage,
		"text/en/strings.toml": "\"item.hard_pike\" = \"Hard Pike\"\n",
		"text/ru/strings.toml": "\"item.hard_pike\" = \"Hard Pike\"\n",
	})
}

// TestReleaseModWeaponRowRefusesDamageThatWouldWrapAByte: a row with damage-min
// above damage-max, or a scaled damage past 255, is refused at load.
func TestReleaseModWeaponRowRefusesDamageThatWouldWrapAByte(t *testing.T) {
	dir := t.TempDir()
	damageRowMod(t, dir, "min-only", "damage-min = 100\n")
	damageRowMod(t, dir, "reversed", "damage-min = 80\ndamage-max = 40\n")
	damageRowMod(t, dir, "past-a-byte", "damage-min = 250\ndamage-max = 255\n")
	for _, c := range []struct {
		id   string
		want []string
	}{
		{"min-only", []string{`mod "min-only"`, "data/items.toml:5", `weapon row "hard_pike"`, "damage-min is 100 and damage-max is 56"}},
		{"reversed", []string{`mod "reversed"`, "data/items.toml:6", `weapon row "hard_pike"`, "damage-min is 80 and damage-max is 40"}},
		{"past-a-byte", []string{`mod "past-a-byte"`, `weapon row "hard_pike"`, "damage-min 250 and damage-max 255 scale to", "within 0..255"}},
	} {
		_, err := modRowsFrontErr(t, dir, false, c.id)
		if err == nil {
			t.Fatalf("%s loaded", c.id)
		}
		for _, w := range c.want {
			if !strings.Contains(err.Error(), w) {
				t.Fatalf("%s: %v; want %q in it", c.id, err, w)
			}
		}
	}
}

// TestReleaseModWeaponRowInOrderDamageHitsInItsStatedRange: an in-order row loads
// and hits within its damage-max; the Pike hits as before.
func TestReleaseModWeaponRowInOrderDamageHitsInItsStatedRange(t *testing.T) {
	dir := t.TempDir()
	damageRowMod(t, dir, "in-order", "damage-min = 40\ndamage-max = 80\n")
	f := modRowsFront(t, dir, false, "in-order")
	plain := releaseFront(t)
	row := weaponRowNamed(t, f.Table, "hard_pike")
	code := weaponPairCode(t, f.Table, row, "Common", "Iron")
	pike := weaponPairCode(t, plain.Table, weaponRowNamed(t, plain.Table, "Pike"), "Common", "Iron")
	for _, c := range []struct {
		name  string
		front *FrontEnd
		code  uint16
		max   int32
	}{{"mod row", f, code, 80}, {"shipped Pike", plain, pike, 56}} {
		w, err := data.WeaponFromCode(data.ItemCode(c.code), c.front.Table.Shapes, c.front.Table.Materials, c.front.Table.Weapons)
		if err != nil {
			t.Fatal(err)
		}
		mw, hero := heroSwingMission(t, c.front, c.code)
		for range 3 {
			mw.tick()
		}
		e, _ := mw.entity(hero)
		if e.DamageBase != w.DamageBase || e.DamageSpread != w.DamageSpread {
			t.Fatalf("%s: hero base %d spread %d, item base %d spread %d", c.name, e.DamageBase, e.DamageSpread, w.DamageBase, w.DamageSpread)
		}
		if e.DamageBase <= 0 || e.DamageBase+e.DamageSpread > c.max {
			t.Fatalf("%s: blows %d..%d, stated damage-max %d", c.name, e.DamageBase, e.DamageBase+e.DamageSpread, c.max)
		}
	}
}
