package game

// The spellbook popup states the spell record's own values at the actor's
// power (TEXT-096, TEXT-097), on the installed Data.bin spell rows and the
// installed main.txt labels, on either root. The expectations are the claims'
// formulas applied to the installed columns here, not the simulation's own
// projection. This is a release witness: it skips without AGAINROM_ASSETS.

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

func TestReleaseSpellbookPopupStatesTheRecordAtTheActorsPower(t *testing.T) {
	f := releaseFront(t)
	mainTable := LoadTextTable(f.Archives.Containers, MainTextPath)
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
	var damageIDs []uint16
	for _, tc := range []struct {
		name        string
		skill, mind int32
		power       int
	}{
		{"skill plus mind below thirty", 4, 10, 100},
		{"power zero", 0, 30, 0},
		{"power fifty", 30, 50, 50},
		{"skill plus mind of 286", 186, 100, 255},
	} {
		for _, r := range rules {
			e := sim.Entity{Mind: tc.mind}
			if r.School < 8 {
				e.Skill[r.School] = tc.skill
			}
			lines := spellInfoLines(r, sim.SpellCharacteristicsFor(sim.Rules{}, e, r), "x", &f.Words)
			joined := "\n" + strings.Join(lines, "\n") + "\n"
			var wantDamage, wantRange string
			if lo, hi := recordDamage(r, tc.power); hi != 0 {
				wantDamage = label(spellLabelDamage) + ": " + spellPair(int64(lo), int64(hi), "%d", "-")
				if tc.name == "power zero" && r.DamageMin > 0 {
					damageIDs = append(damageIDs, r.ID)
				}
			}
			if rg := recordRange(r, tc.power); rg != 0 {
				wantRange = label(spellLabelRange) + ": " + fmt.Sprint(rg)
			}
			for _, pair := range [][2]string{{wantDamage, label(spellLabelDamage) + ":"}, {wantRange, label(spellLabelRange) + ":"}} {
				got := ""
				for _, l := range lines {
					if strings.HasPrefix(l, pair[1]) {
						got = l
					}
				}
				if got != pair[0] {
					t.Errorf("%s, spell %d: line %q, want %q in %q", tc.name, r.ID, got, pair[0], joined)
				}
			}
		}
	}
	if fmt.Sprint(damageIDs) != "[1 2 3 6 9 11 13 14 21]" {
		t.Errorf("installed rows with a damage minimum = %v, want TEXT-096's [1 2 3 6 9 11 13 14 21]", damageIDs)
	}

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
		// The actor's own skill and Mind, summed in a byte as the fold does.
		sum := int(entity.Skill[heal.School]) + int(entity.Mind) - 30
		power := sum & 0xff
		if power > 100 {
			power = 100
		}
		if power < 100 {
			t.Fatalf("fixture actor reads record power %d, want the wrapped 100", power)
		}
		lo, hi := recordDamage(heal, power)
		bookTable := LoadTextTable(f.Archives.Containers, SpellBookNamesTextPath)
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
