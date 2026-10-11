package game

// The spellbook popup states the spell record's own values (TEXT-096) at the
// power the actor's cast takes, on the installed Data.bin spell rows and the
// installed main.txt labels, on either root. The expectations are the claims'
// formulas applied to the installed columns here, not the simulation's own
// projection. The original's record level reads 100 for a skill and Mind sum
// below 30 (TEXT-097); the owner's popup states the cast instead (DIV-1882).
// This is a release witness: it skips without AGAINROM_ASSETS.

import (
	"bytes"
	"fmt"
	"image"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestReleaseSpellbookPopupStatesTheCastAtTheActorsPower(t *testing.T) {
	f := releaseFront(t)
	mainTable := LoadTextTable(f.Archives.Containers, MainTextPath, f.textCode())
	if mainTable == nil {
		t.Fatal("install carries no main.txt")
	}
	label := func(i int) string {
		s, ok := mainTable.At(i)
		if !ok {
			t.Fatalf("main.txt carries no row %d", i)
		}
		return s
	}
	rules := mapload.SpellRules(f.Table)
	if len(rules) < 28 {
		t.Fatalf("install names %d spells, want 28", len(rules))
	}
	// The record's byte +9, +0xe and +0xf at a power, from the row's columns
	// (TEXT-096): range plus power/30 when non-zero (power/3 for id 26); the
	// damage columns times power/30 + 1, the spread above the minimum.
	recordRange := func(r sim.SpellRule, power int) int {
		base := int(r.MaxRange)
		switch {
		case r.ID == 26:
			base += power / 3
		case base != 0:
			base += power / 30
		}
		return base & 0xff
	}
	recordDamage := func(r sim.SpellRule, power int) (lo, hi int) {
		factor := float64(power)/30.0 + 1.0
		if r.DamageMin > 0 {
			lo = int(float64(r.DamageMin)*factor) & 0xff
		}
		if r.DamageMax > 0 {
			hi = int(float64(r.DamageMax)*factor-float64(lo)) & 0xff
		}
		return lo, lo + hi
	}
	damageLine := func(lo, hi int) string {
		if hi == 0 {
			return ""
		}
		return label(spellLabelDamage) + ": " + spellPair(int64(lo), int64(hi), "%d", "-")
	}
	rangeLine := func(rg int) string {
		if rg == 0 {
			return ""
		}
		return label(spellLabelRange) + ": " + fmt.Sprint(rg)
	}
	// Every skill and Mind sum from 0 to 290: below 30 the cast takes power
	// 0, from 30 to 285 the sum less 30, above that the ceiling 255. From 30
	// up the expected lines are also the former popup's, which filled the
	// record at that same power. Loss control: below 30 the former popup
	// filled it at 100, and its lines differ.
	var damageIDs []uint16
	formerDiffers := map[uint16]bool{}
	for sum := 0; sum <= 290; sum++ {
		power := min(max(sum-30, 0), 255)
		for _, r := range rules {
			e := sim.Entity{Mind: int32(sum)}
			if r.School < 6 {
				e.Mind, e.Skill[r.School] = 30, int32(sum-30)
			}
			c := sim.SpellCharacteristicsFor(sim.Rules{}, e, r)
			if c.Power != int32(power) {
				t.Fatalf("skill plus Mind %d, spell %d: cast power %d, want %d", sum, r.ID, c.Power, power)
			}
			lo, hi := recordDamage(r, power)
			if castLo, castHi := sim.SpellDamagePair(sim.Rules{}, r, c.Power); castLo != int64(lo) || castHi != int64(hi) {
				t.Fatalf("skill plus Mind %d, spell %d: the cast's pair %d-%d, the record %d-%d", sum, r.ID, castLo, castHi, lo, hi)
			}
			lines := spellInfoLines(r, c, "x", &f.Words)
			joined := "\n" + strings.Join(lines, "\n") + "\n"
			if sum == 30 && r.DamageMin > 0 {
				damageIDs = append(damageIDs, r.ID)
			}
			for _, pair := range [][2]string{{damageLine(lo, hi), label(spellLabelDamage) + ":"}, {rangeLine(recordRange(r, power)), label(spellLabelRange) + ":"}} {
				got := ""
				for _, l := range lines {
					if strings.HasPrefix(l, pair[1]) {
						got = l
					}
				}
				if got != pair[0] {
					t.Errorf("skill plus Mind %d, spell %d: line %q, want %q in %q", sum, r.ID, got, pair[0], joined)
				}
			}
			if sum < 30 {
				flo, fhi := recordDamage(r, 100)
				if damageLine(flo, fhi) != damageLine(lo, hi) || rangeLine(recordRange(r, 100)) != rangeLine(recordRange(r, power)) {
					formerDiffers[r.ID] = true
				}
			}
		}
	}
	if fmt.Sprint(damageIDs) != "[1 2 3 6 9 11 13 14 21]" {
		t.Errorf("installed rows with a damage minimum = %v, want TEXT-096's [1 2 3 6 9 11 13 14 21]", damageIDs)
	}
	if len(formerDiffers) == 0 {
		t.Error("loss control: below 30 the former record level 100 states every row as the cast does")
	}
	t.Logf("%d installed rows x 291 sums; below 30 the former popup differed on %d rows", len(rules), len(formerDiffers))

	// The same through the mission hover: Heal, a row with damage columns and
	// a restorative arm, on an actor whose skill plus Mind is below thirty.
	t.Run("mission-hover", func(t *testing.T) {
		f := releaseFront(t)
		f.SetDeterministicFrames(true)
		hero := data.Hero{Body: 60, Reaction: 60, Mind: 10, Spirit: 100}
		party := []mapload.PartyMember{{ID: "hero", PlayerCharacter: true, StartingHero: true, Mage: true, Class: 0x18,
			Profile: data.Profile{HealthColumn: true, ManaColumn: true}, Hero: hero,
			KnownSpells: 1 << 6,
			Saved:       &mapload.Saved{Cell: mapload.Cell{X: 29, Y: 50}, HP: 100, MaxHP: 100, Mana: 1000, MaxMana: 1000}}}
		a := f.App("spellbook record hover")
		a.Layout(1024, 768)
		a.SetTooltipDelayPreference(0, nil)
		a.SetTooltipFont(f.Font.Value())
		if err := a.OpenMission(f.MissionOpenerWith(10, party)); err != nil {
			t.Fatal(err)
		}
		if err := a.HeadlessKey("0"); err != nil {
			t.Fatal(err)
		}
		id := f.live.mission.ids[0]
		inspectionCentre(f.live, 29, 50)
		if err := a.HeadlessSelectEntity(uint32(id)); err != nil {
			t.Fatal(err)
		}
		if err := a.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
		if _, _, err := a.HeadlessSpellPoint(6); err != nil {
			if err := a.HeadlessKey("book"); err != nil {
				t.Fatal(err)
			}
		}
		x, y, err := a.HeadlessSpellPoint(6)
		if err != nil {
			t.Fatal(err)
		}
		if err := a.HeadlessPointer("hover", x, y); err != nil {
			t.Fatal(err)
		}
		state, pic := a.HeadlessTooltip()
		if state.Target != "spell/6" || !state.Visible || pic == nil {
			t.Fatalf("heal hover = %+v", state)
		}
		entity, ok := f.live.entity(id)
		if !ok {
			t.Fatal("selected caster absent")
		}
		var heal sim.SpellRule
		for _, r := range f.live.world.Spells() {
			if r.ID == 6 {
				heal = r
			}
		}
		if heal.ID != 6 || heal.DamageMax <= 0 {
			t.Fatalf("installed heal row = %+v", heal)
		}
		// The actor's skill plus Mind is below 30: the cast takes power 0,
		// where the original's byte fold reads 100 (TEXT-097).
		if sum := int(entity.Skill[heal.School]) + int(entity.Mind); sum >= 30 {
			t.Fatalf("fixture actor's skill plus Mind is %d, want below 30", sum)
		}
		const power = 0
		lo, hi := recordDamage(heal, power)
		if flo, fhi := recordDamage(heal, 100); flo == lo && fhi == hi {
			t.Fatalf("loss control: Heal states %d-%d at both levels", lo, hi)
		}
		bookTable := LoadTextTable(f.Archives.Containers, SpellBookNamesTextPath, f.textCode())
		row, _ := bookTable.At(12)
		want := []string{strings.SplitN(row, "#", 2)[0],
			fmt.Sprintf("%s: %d", label(spellLabelManaCost), heal.ManaCost),
			fmt.Sprintf("%s: %s", label(spellLabelDamage), spellPair(int64(lo), int64(hi), "%d", "-")),
			fmt.Sprintf("%s: %d", label(spellLabelRange), recordRange(heal, power))}
		wantPic, _, ok := ui.ComposeTooltipHint(want, f.Font.Value(), image.Point{}, image.Rect(0, 0, 1024, 768), f.HoverBall())
		if !ok || !bytes.Equal(pic.Pix, wantPic.Pix) || pic.Bounds() != wantPic.Bounds() {
			t.Fatalf("heal hover did not paint %q", want)
		}
	})
}

// TestReleaseSecondGameSpellbookPopupStatesTheCastPair holds the popup's
// damage pair on every installed second-game row to the second game's cast
// arithmetic, column times (power + 30) / 30 in integers with the spread
// above the base, for every skill and Mind sum from 0 to 290. The popup is the
// one composer for both games (spellbookOf). Loss control: the former popup's
// binary64 byte fold states a different pair on some installed row and power.
func TestReleaseSecondGameSpellbookPopupStatesTheCastPair(t *testing.T) {
	archives := secondGameRoot(t)
	defs, err := LoadDefinitionsFor(archives.Containers, archives.Game())
	if err != nil {
		t.Fatal(err)
	}
	rules := mapload.SpellRules(defs.Table)
	if len(rules) != 29 {
		t.Fatalf("SpellRules = %d rows, want 29", len(rules))
	}
	formerDiffers := 0
	for sum := 0; sum <= 290; sum++ {
		power := int64(min(max(sum-30, 0), 255))
		for _, r := range rules {
			e := sim.Entity{Mind: int32(sum)}
			if r.School < 6 {
				e.Mind, e.Skill[r.School] = 30, int32(sum-30)
			}
			c := sim.SpellCharacteristicsFor(sim.Rules{}, e, r)
			base := int64(r.DamageMin) * (power + 30) / 30
			top := max(int64(r.DamageMax)*(power+30)/30, base)
			if int64(c.Power) != power || c.DamageMin != base || c.DamageMax != top {
				t.Fatalf("skill plus Mind %d, spell %d: power %d pair %d-%d, want power %d pair %d-%d", sum, r.ID, c.Power, c.DamageMin, c.DamageMax, power, base, top)
			}
			if lo, hi := sim.SpellDamagePair(sim.Rules{}, r, c.Power); lo != c.DamageMin || hi != c.DamageMax {
				t.Fatalf("skill plus Mind %d, spell %d: the cast's pair %d-%d, the popup %d-%d", sum, r.ID, lo, hi, c.DamageMin, c.DamageMax)
			}
			if sum < 30 || sum > 285 {
				continue
			}
			product := func(column int32) int {
				return int(float64(float64(column) * (float64(power)/30.0 + 1.0)))
			}
			var lo, spread int
			if r.DamageMin > 0 {
				lo = int(uint8(product(r.DamageMin)))
			}
			if r.DamageMax > 0 {
				spread = int(uint8(product(r.DamageMax) - lo))
			}
			if int64(lo) != base || int64(lo+spread) != top {
				formerDiffers++
			}
		}
	}
	if formerDiffers == 0 {
		t.Error("loss control: the former byte fold states every installed row and power as the cast does")
	}
	t.Logf("29 installed rows x 291 sums; at sums 30..285 the former popup differed at %d row and power pairs", formerDiffers)
}

// TestReleaseSpellbookPopupStatesTheCastPairUnderADamageFactorTable gives every
// installed first-game row one damage factor table, 30 + 10 * power, and holds
// the popup's pair to the cast's arithmetic for every skill and Mind sum from
// 0 to 290: base column * factor / 30 and top the same on the maximum column,
// whole. Loss control: the former popup folded byte-column factor products to
// bytes and states a different pair where a product passes 255.
func TestReleaseSpellbookPopupStatesTheCastPairUnderADamageFactorTable(t *testing.T) {
	f := releaseFront(t)
	rules := mapload.SpellRules(f.Table)
	if len(rules) != 28 {
		t.Fatalf("install names %d spells, want 28", len(rules))
	}
	factor := make([]int32, 256)
	for p := range factor {
		factor[p] = 30 + 10*int32(p)
	}
	var set sim.SpellFormulaSet
	set.Global[sim.FormulaDamage] = factor
	tabled, err := sim.Rules{}.WithSpellFormulas(set)
	if err != nil {
		t.Fatal(err)
	}
	formerDiffers := 0
	for sum := 0; sum <= 290; sum++ {
		power := min(max(sum-30, 0), 255)
		for _, r := range rules {
			e := sim.Entity{Mind: int32(sum)}
			if r.School < 6 {
				e.Mind, e.Skill[r.School] = 30, int32(sum-30)
			}
			c := sim.SpellCharacteristicsFor(tabled, e, r)
			fp := int64(factor[power])
			base := int64(r.DamageMin) * fp / 30
			top := max(int64(r.DamageMax)*fp/30, base)
			if int(c.Power) != power || c.DamageMin != base || c.DamageMax != top {
				t.Fatalf("skill plus Mind %d, spell %d: power %d pair %d-%d, want power %d pair %d-%d", sum, r.ID, c.Power, c.DamageMin, c.DamageMax, power, base, top)
			}
			if lo, hi := sim.SpellDamagePair(tabled, r, c.Power); lo != c.DamageMin || hi != c.DamageMax {
				t.Fatalf("skill plus Mind %d, spell %d: the cast's pair %d-%d, the popup %d-%d", sum, r.ID, lo, hi, c.DamageMin, c.DamageMax)
			}
			lo := int64(0)
			if r.DamageMin > 0 {
				lo = int64(uint8(base))
			}
			spread := int64(0)
			if r.DamageMax > 0 {
				spread = int64(uint8(int64(r.DamageMax)*fp/30 - lo))
			}
			if lo != base || lo+spread != top {
				formerDiffers++
			}
		}
	}
	if formerDiffers == 0 {
		t.Error("loss control: the former byte fold states every row and power as the cast does")
	}
	t.Logf("28 installed rows x 291 sums under one factor table; the former popup differed at %d row and sum pairs", formerDiffers)
}
