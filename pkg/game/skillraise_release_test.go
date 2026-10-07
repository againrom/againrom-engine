package game

import (
	"reflect"
	"strconv"
	"strings"
	"testing"
	"unicode"

	"golang.org/x/text/encoding/charmap"

	"againrom/pkg/data"
	"againrom/pkg/render/text"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// A generated character in mission 20 fights the nearest hostile through the
// player's orders until one award raises a skill: a Blade fighter by attack
// orders, a Fire mage by move and cast orders. The one posted row must be the
// installed main.txt line for that class and slot, read here straight from
// the container, then ": " and the new level, and every byte must draw an
// inked glyph of the map font under the install's own language selector.
func TestReleaseSkillRaiseNoticeIsInstalledText(t *testing.T) {
	for _, mage := range []bool{false, true} {
		t.Run(map[bool]string{false: "fighter", true: "mage"}[mage], func(t *testing.T) {
			f := releaseFront(t)
			f.SetDeterministicFrames(true)
			raw, err := f.Archives.Containers.ReadFile(MainTextPath)
			if err != nil {
				t.Fatal(err)
			}
			lines := strings.Split(string(raw), "\r\n")
			if len(lines) < 141 {
				t.Fatalf("main.txt holds %d lines, want the raise lines 130..139", len(lines))
			}
			font := f.Font.Value()
			language, err := f.Archives.Containers.ReadFile(LanguagePath)
			if err != nil || font == nil || len(language) == 0 || int(language[len(language)-1])-'0' != font.Selector {
				t.Fatal("map font or its install language selector is absent", err)
			}
			for i := range f.Words.SkillRaised {
				if f.Words.SkillRaised[i] != lines[130+i] || lines[130+i] == "" {
					t.Fatalf("SkillRaised[%d] = %q, want installed main.txt[%d] %q", i, f.Words.SkillRaised[i], 130+i, lines[130+i])
				}
				requireInkedNoticeGlyphs(t, font, lines[130+i])
			}
			before, after, rows := raiseSkillByOrders(t, f, mage)
			slot := int32(-1)
			for s := range after.Skill {
				if after.Skill[s] != before.Skill[s] {
					if slot >= 0 || after.Skill[s] != before.Skill[s]+1 || after.SkillXP[s] <= before.SkillXP[s] {
						t.Fatalf("skills %v -> %v, experience %v -> %v: want one raise by one award", before.Skill, after.Skill, before.SkillXP, after.SkillXP)
					}
					slot = int32(s)
				}
			}
			if slot < 1 || !mage && slot != data.SkillBlade {
				t.Fatalf("raised slot %d", slot)
			}
			index := 130 + int(slot) - 1
			if mage {
				index += 5
			}
			want := announced(lines[index] + ": " + strconv.Itoa(int(after.Skill[slot])))
			if !reflect.DeepEqual(rows, want) {
				t.Fatalf("the raise posted %+v, want one line %q", rows, want[0].Text)
			}
			requireInkedNoticeGlyphs(t, font, rows[0].Text)
			shown := rows[0].Text
			if font.Selector == text.SelectorConverting {
				if shown, err = charmap.CodePage866.NewDecoder().String(rows[0].Text); err != nil {
					t.Fatal(err)
				}
				for _, r := range strings.TrimSuffix(shown, ": "+strconv.Itoa(int(after.Skill[slot]))) {
					if r != ' ' && !unicode.Is(unicode.Cyrillic, r) {
						t.Fatalf("RU notice %q holds non-Cyrillic %q", shown, r)
					}
				}
			} else if lines[index] != ui.AuthoredWords().SkillRaised[index-130] {
				t.Fatalf("EN main.txt[%d] %q differs from the authored fallback", index, lines[index])
			}
			t.Logf("selector %d, tick %d: slot %d %d -> %d, main.txt[%d], notice %q", font.Selector, f.live.world.Tick(),
				slot, before.Skill[slot], after.Skill[slot], index, shown)
		})
	}
}

// raiseSkillByOrders opens mission 20 with one generated Blade fighter or Fire
// mage and gives the player's orders against the nearest hostile until the
// message line holds a line. It returns the character before and after, and
// the lines.
func raiseSkillByOrders(t *testing.T, f *FrontEnd, mage bool) (sim.Entity, sim.Entity, []ui.MessageLine) {
	t.Helper()
	class := 0
	if mage {
		class = 1
	}
	party := f.ChargenParty(ui.ChargenResult{Name: "Raise", Choices: []int{0, class, 0}, Stats: []int{31, 27, 24, 29}})
	app := f.App("skill raise notice")
	if err := app.OpenMission(f.MissionOpenerWith(20, party)); err != nil {
		t.Fatal(err)
	}
	app.Layout(messageWitnessW, messageWitnessH)
	id := f.live.mission.ids[0]
	hero := generatedCampaignEntity(t, f, id)
	var target sim.Entity
	distance := int32(1 << 30)
	for _, e := range f.live.world.Entities() {
		dx, dy := e.X-hero.X, e.Y-hero.Y
		if e.Domain == hero.Domain && f.live.world.Relations().Hostile(hero.Owner, e.Owner) && dx*dx+dy*dy < distance {
			distance, target = dx*dx+dy*dy, e
		}
	}
	var spell uint32
	for id := uint32(1); mage && id < 32 && spell == 0; id++ {
		if rule, ok := f.live.world.Spell(id); ok && hero.KnownSpells&(1<<id) != 0 && rule.Damaging && rule.TargetsUnit {
			spell = id
		}
	}
	if target.ID == 0 || (hero.MaxMana > 0) != mage || mage && spell == 0 {
		t.Fatalf("mission 20 lacks a hostile target, or the character is not the chosen class: %+v", hero)
	}
	f.live.tick()
	if mage {
		f.live.enqueue(uint32(id), int(target.X), int(target.Y))
	} else {
		f.live.strike(uint32(id), uint32(target.ID))
	}
	var rows []ui.MessageLine
	for n := 0; n < 2400 && len(rows) == 0; n++ {
		if mage {
			f.live.castAt(uint32(id), uint32(target.ID), spell)
		} else {
			now, victim := generatedCampaignEntity(t, f, id), generatedCampaignEntity(t, f, target.ID)
			if dx, dy := victim.X-now.X, victim.Y-now.Y; dx*dx+dy*dy < 8 {
				f.live.strike(uint32(id), uint32(target.ID))
			}
		}
		f.live.tick()
		rows = f.live.view.MessageLines()
	}
	return hero, generatedCampaignEntity(t, f, id), rows
}

// requireInkedNoticeGlyphs fails unless every non-space byte of s selects a
// record of f, not the space fallback, that paints at least one pixel.
func requireInkedNoticeGlyphs(t *testing.T, f *text.Font, s string) {
	t.Helper()
	for i := 0; i < len(s); i++ {
		if s[i] == ' ' {
			continue
		}
		record := int(text.Convert(s[i], f.Selector)) - text.FirstChar
		g := f.GlyphFor(s[i])
		inked := false
		for _, p := range g.Pixels {
			inked = inked || p.Painted
		}
		if record < 1 || record >= len(f.Glyphs) || !inked {
			t.Fatalf("byte %#x of %q selects record %d with ink %v", s[i], s, record, inked)
		}
	}
}
