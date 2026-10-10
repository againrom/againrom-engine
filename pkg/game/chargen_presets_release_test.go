package game

import (
	"image"
	"slices"
	"testing"

	"againrom/pkg/formats/reg"
	"againrom/pkg/ui"
)

// TestReleaseChargenHeroesStartOnArchetypeSections: each pre-create picture's
// Forward loads npc.reg's archetype section for its sex and class — Body,
// Reaction, Mind, Spirit and the one-based Skill (HERO-STAT-001,
// HERO-CHARGEN-082, HERO-CHARGEN-083) — and the pool shows what the spread
// leaves of the budget. The registry is read here independently of the
// definition loader.
func TestReleaseChargenHeroesStartOnArchetypeSections(t *testing.T) {
	f := releaseFront(t)
	b, err := f.Archives.Containers.ReadFile(NPCRegistry)
	if err != nil {
		t.Fatal(err)
	}
	r, err := reg.Parse(b)
	if err != nil {
		t.Fatal(err)
	}
	setup := f.ChargenSetup()
	heroes := f.generator().PreCreate.Heroes
	if len(heroes) != 4 {
		t.Fatalf("%d heroes, want 4", len(heroes))
	}
	for i, h := range heroes {
		section := [2][2]string{{"MaleFighter", "MaleMage"}, {"FemaleFighter", "FemaleMage"}}[h.Sex][h.Class]
		want := make([]int, 0, 4)
		for _, k := range []string{"Body", "Reaction", "Mind", "Spirit"} {
			v, ok := r.GetInt(section, k)
			if !ok {
				t.Fatalf("%s has no %s", section, k)
			}
			want = append(want, int(v))
		}
		skill, ok := r.GetInt(section, "Skill")
		if !ok {
			t.Fatalf("%s has no Skill", section)
		}
		c := ui.NewChargen(setup)
		c.SelectPreChoice(i)
		c.Forward()
		res, ok := c.Result()
		if !ok {
			t.Fatalf("hero %d (%s): no legal result after Forward", i, section)
		}
		for k := range want {
			if res.Stats[k] != want[k] {
				t.Errorf("hero %d (%s): stats %v, want %v", i, section, res.Stats, want)
				break
			}
		}
		if res.Choices[0] != h.Sex || res.Choices[1] != h.Class {
			t.Errorf("hero %d (%s): sex/class %v, want %d/%d", i, section, res.Choices, h.Sex, h.Class)
		}
		if res.Choices[2] != int(skill)-1 {
			t.Errorf("hero %d (%s): skill index %d, want %d (Skill=%d)", i, section, res.Choices[2], skill-1, skill)
		}
		if c.Remaining() != 0 {
			t.Errorf("hero %d (%s): pool %d, want 0", i, section, c.Remaining())
		}
		t.Logf("hero %d %s: %v skill %d pool %d", i, section, res.Stats, res.Choices[2], c.Remaining())
	}
	// The owner's original EN screenshot of Danath on a fresh generator.
	c := ui.NewChargen(setup)
	c.Forward()
	if res, _ := c.Result(); len(res.Stats) != 4 || res.Stats[0] != 41 || res.Stats[1] != 35 || res.Stats[2] != 20 || res.Stats[3] != 15 || res.Choices[2] != 0 {
		t.Errorf("Danath enters with %v skill %v, want 41/35/20/15 and Blade", res.Stats, res.Choices)
	}
}

// TestReleaseChargenCardRowsShareTheTownCardPens: the detailed page's card is
// the town card builder's card, moved by the description's card offset. Every
// row stands on the card style's pens and value ends (DIV-191): Weight and XP
// share one pen, Sight and Speed another, values flush right.
func TestReleaseChargenCardRowsShareTheTownCardPens(t *testing.T) {
	f := releaseFront(t)
	c := ui.NewChargen(f.ChargenSetup())
	c.Forward()
	v := c.CardView()
	gen := v.CardReport()
	town := ui.TownCharacterView{Statistics: true, CardFont: f.tipFont(), StatsPane: f.characterPanes().Stats,
		HasSubject: true, Subject: v.Subject}.CardReport()
	if len(gen) == 0 || len(gen) != len(town) {
		t.Fatalf("generator card has %d rows, town card %d", len(gen), len(town))
	}
	for i := range gen {
		g, w := gen[i], town[i]
		if g.At != w.At || g.ValueX != w.ValueX || g.Value != w.Value || g.RightX != w.RightX || g.RightValueX != w.RightValueX {
			t.Errorf("row %d %q: generator at %v value %q+%d right %d+%d, town at %v value %q+%d right %d+%d",
				i, g.Label, g.At, g.Value, g.ValueX, g.RightX, g.RightValueX, w.At, w.Value, w.ValueX, w.RightX, w.RightValueX)
		}
	}
	// Rows by their place on the card: name, four attribute rows, damage,
	// attack, headings, five skill rows, weight, XP, sight, speed. The XP row's
	// glyph band meets the lower ornament, and the builder's writable-face
	// clamp moves it one pixel in on each side, on the town card as here.
	if len(gen) != 17 {
		t.Fatalf("generator card has %d rows, want 17", len(gen))
	}
	font := f.ChargenAssets.Presentation.Font
	for _, row := range []struct{ i, pen, end int }{
		{1, 6, 80}, {2, 6, 80}, {3, 6, 80}, {4, 6, 80}, {8, 6, 70}, {12, 6, 70},
		{13, 16, 127}, {14, 17, 126}, {15, 40, 106}, {16, 40, 106},
	} {
		g := gen[row.i]
		w, _ := font.Measure(g.Value)
		if g.At.X != row.pen || g.At.X+g.ValueX+w != row.end {
			t.Errorf("row %q: pen %d value end %d, want %d and %d", g.Label, g.At.X, g.At.X+g.ValueX+w, row.pen, row.end)
		}
	}
	if off := v.CardRect().Min.Sub(v.PaneRect.Min); off.X <= 0 {
		t.Errorf("card canvas offset %v, want the description's card offset", off)
	}
	// The page paints exactly what the town builder paints for this view.
	page := ui.ComposeChargenFrame(c)
	ref := image.NewRGBA(page.Bounds())
	ui.DrawCharacterPaneBody(ref, v)
	bad := 0
	for y := v.PaneRect.Min.Y; y < v.PaneRect.Max.Y; y++ {
		for x := v.PaneRect.Min.X; x < v.PaneRect.Max.X; x++ {
			if page.RGBAAt(x, y) != ref.RGBAAt(x, y) {
				bad++
			}
		}
	}
	if bad != 0 {
		t.Errorf("%d card pixel(s) differ from the town builder's card", bad)
	}
}

// TestReleaseChargenResetIsTheOriginalsAndRestoreReturnsThePreset: on every
// hero portrait, Reset gives pool 100 and every statistic 25 (MENU-139,
// HERO-CHARGEN-082) after any preset or spend and keeps the chosen skill;
// Restore gives the portrait's preset after any spend or a Reset.
func TestReleaseChargenResetIsTheOriginalsAndRestoreReturnsThePreset(t *testing.T) {
	f := releaseFront(t)
	setup := f.ChargenSetup()
	if got := len(f.generator().Detail.Commands); got != 4 {
		t.Fatalf("%d detailed commands, want Accept, Reset, Back and Restore", got)
	}
	for i := range f.generator().PreCreate.Heroes {
		c := ui.NewChargen(setup)
		c.SelectPreChoice(i)
		c.Forward()
		preset, ok := c.Result()
		if !ok {
			t.Fatalf("hero %d: no result after Forward", i)
		}
		presetStats := append([]int(nil), preset.Stats...)
		presetPool := c.Remaining()
		startSkill := preset.Choices[2]
		if startSkill != setup.PresetSkills[i]-1 {
			t.Fatalf("hero %d: opens with skill %d, PresetSkills says %d", i, startSkill, setup.PresetSkills[i]-1)
		}
		skill := (startSkill + 1) % 5
		c.SelectSkill(skill)

		c.Reset()
		r, _ := c.Result()
		if want := []int{25, 25, 25, 25}; !slices.Equal(r.Stats, want) || c.Remaining() != 100 || r.Choices[2] != skill {
			t.Errorf("hero %d: Reset gave %v, pool %d, skill %d; want 25 each, pool 100, skill %d", i, r.Stats, c.Remaining(), r.Choices[2], skill)
		}
		c.Restore()
		r, _ = c.Result()
		if !slices.Equal(r.Stats, presetStats) || c.Remaining() != presetPool || r.Choices[2] != startSkill {
			t.Errorf("hero %d: Restore after Reset gave %v, pool %d, skill %d; want %v, pool %d, skill %d", i, r.Stats, c.Remaining(), r.Choices[2], presetStats, presetPool, startSkill)
		}
		c.SelectSkill(skill)
		c.AdjustStat(0, -1)
		c.AdjustStat(1, -1)
		c.Restore()
		r, _ = c.Result()
		if !slices.Equal(r.Stats, presetStats) || c.Remaining() != presetPool || r.Choices[2] != startSkill {
			t.Errorf("hero %d: Restore after a spend gave %v, pool %d, skill %d; want %v, pool %d, skill %d", i, r.Stats, c.Remaining(), r.Choices[2], presetStats, presetPool, startSkill)
		}
		c.SelectSkill(skill)
		c.AdjustStat(0, 1)
		c.Reset()
		r, _ = c.Result()
		if want := []int{25, 25, 25, 25}; !slices.Equal(r.Stats, want) || c.Remaining() != 100 || r.Choices[2] != skill {
			t.Errorf("hero %d: Reset after a spend gave %v, pool %d, skill %d; want skill %d kept", i, r.Stats, c.Remaining(), r.Choices[2], skill)
		}
	}
}
