package game

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/sim"
)

// Every shipped Weapons row's information text lists the base pairs of the
// original's Weapon packer (ITEM-211): none without sutableFor bit 0 (a ranged row keeps its range line, DIV-1757), else the
// damage pair, to-hit and defence when positive, and the range byte when it
// exceeds 1 under the stats caption of tag 38. When AGAINROM_WEAPON_BASE_TABLE
// names a file the per-row table is written there.
func TestReleaseShippedWeaponsShowTheOriginalsBasePairs(t *testing.T) {
	f := releaseFront(t)
	table, words := f.Table, f.Words
	var report strings.Builder
	for row := 1; row < table.Weapons.Len(); row++ {
		p := table.Weapons.EntryParams(row)
		if len(p) <= data.SutableForColumn {
			continue
		}
		code := data.ComposeItemCode(15, 1, 0, row)
		w, err := data.WeaponFromCode(code, table.Shapes, table.Materials, table.Weapons)
		if err != nil {
			t.Fatalf("row %d: %v", row, err)
		}
		got := itemInfoLinesWithWeaponDamage(code, table, nil, words)
		inst := itemInstanceInfoLines(sim.PlainItem(uint16(code)), table, words)
		if strings.Join(got, "|") != strings.Join(inst, "|") {
			t.Errorf("row %d %q: code lines %q, instance lines %q", row, w.Name, got, inst)
		}
		bit0 := p[data.SutableForColumn]&1 != 0
		var want []string
		if bit0 {
			want = append(want, fmt.Sprintf("#%s: %d-%d", words.ItemStats[43], w.DamageBase, w.DamageBase+w.DamageSpread))
			if w.ToHit > 0 {
				want = append(want, fmt.Sprintf("#%s %+d", words.ItemStats[12], uint8(w.ToHit)))
			}
			if w.Defence > 0 {
				want = append(want, fmt.Sprintf("#%s %+d", words.ItemStats[15], uint8(w.Defence)))
			}
			if p[11] > 1 {
				want = append(want, fmt.Sprintf("#%s %d", words.ItemStats[38], uint8(p[11])))
			}
		}
		if !bit0 && (w.Ranged() || w.AttackType == data.SkillShoot) && p[11] > 1 {
			want = append(want, fmt.Sprintf("#%s %d", words.ItemStats[38], uint8(p[11])))
		}
		if len(got) != 1+len(want) || strings.Join(got[1:], "|") != strings.Join(want, "|") {
			t.Errorf("row %d %q sutableFor %d: lines %q, want base %q", row, w.Name, p[data.SutableForColumn], got, want)
		}
		before := []string{"damage", "to-hit", "defence"}
		if w.Ranged() || w.AttackType == data.SkillShoot {
			before = append(before, "range")
		}
		fmt.Fprintf(&report, "%d\t%s\t%d\t%s\t%s\n", row, w.Name, p[data.SutableForColumn], strings.Join(before, ","), strings.Join(got[1:], ","))
	}
	if path := os.Getenv("AGAINROM_WEAPON_BASE_TABLE"); path != "" {
		if err := os.WriteFile(path, []byte(report.String()), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
