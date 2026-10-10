package game

import (
	"fmt"
	"reflect"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// skillRaiseFixture opens a one-hero mission: a sword fighter whose Blade
// stands at level 10 with exactly S(10) experience, beside a victim that
// survives every blow. The first landed blow therefore raises Blade to 11
// (HERO-SKILLUP-073). The hand-built world carries its own combat block,
// which a mission opened from a map receives from its loader.
func skillRaiseFixture(t *testing.T) (*mapWorld, sim.EntityID, sim.EntityID) {
	t.Helper()
	const hero, victim = sim.EntityID(7), sim.EntityID(8)
	table := eqDefsTable(t)
	h := eqBladeHero()
	reward := h.Reward()
	a := sim.Entity{ID: hero, X: 3, Y: 3, HP: 100, MaxHP: 100, Reach: 1,
		Owner: 2, GainsXP: true, TypeID: sim.HumanTypeID, Mind: reward.Mind, SkillXP: reward.SkillXP,
		XPSlot: uint8(data.SkillBlade), DamageBase: 5, AlwaysHits: true, AttackCharge: 1, AttackRelax: 1}
	a.Skill[data.SkillBlade] = h.Skill[data.SkillBlade]
	v := sim.Entity{ID: victim, X: 4, Y: 3, HP: 1_000_000, MaxHP: 1_000_000,
		DyingTime: 200, Owner: 3, XPValue: 4}
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH},
		sim.ModeCanonical, sim.Terrain{}, []sim.Entity{a, v}, nil, sim.Relations{}, nil, nil)
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	return equipMission(t, w, hero, h, eqSword(t, table), table), hero, victim
}

// strikeUntilNotice runs one tick with no order, then gives the player's
// attack order before every tick, as a player holding an enemy under attack
// does, and returns the message line's lines once the first one is posted.
func strikeUntilNotice(t *testing.T, mw *mapWorld, hero, victim sim.EntityID) []ui.MessageLine {
	t.Helper()
	mw.tick()
	for i := 0; i < 4000; i++ {
		mw.strike(uint32(hero), uint32(victim))
		mw.tick()
		if rows := mw.view.MessageLines(); len(rows) > 0 {
			return rows
		}
	}
	e, _ := mw.entity(hero)
	t.Fatalf("no notice after 4000 ticks: Blade %d with %d experience", e.Skill[data.SkillBlade], e.SkillXP[data.SkillBlade])
	return nil
}

func TestSkillRaiseNoticeStatesTheRaiseAndTheNewLevel(t *testing.T) {
	mw, hero, victim := skillRaiseFixture(t)
	rows := strikeUntilNotice(t, mw, hero, victim)
	if e, _ := mw.entity(hero); e.Skill[data.SkillBlade] != 11 {
		t.Fatalf("the notice came with Blade at %d, want the raise to 11", e.Skill[data.SkillBlade])
	}
	want := announced("Blade skill improved: 11")
	if !reflect.DeepEqual(rows, want) {
		t.Fatalf("the raise posted %+v, want %+v", rows, want)
	}
}

// The notice states the viewer's resolved install line byte for byte. The
// fixture line stands for a Russian one and is written as CP866 bytes.
func TestSkillRaiseNoticeUsesTheViewerInstallLine(t *testing.T) {
	mw, hero, victim := skillRaiseFixture(t)
	words := mw.view.Words()
	words.SkillRaised[0] = "\x8c\xa5\xe7 \x8d\xa0\xa2\xeb\xaa"
	words.SkillRaised[5] = "school line"
	mw.view.SetWords(words)
	rows := strikeUntilNotice(t, mw, hero, victim)
	want := announced("\x8c\xa5\xe7 \x8d\xa0\xa2\xeb\xaa: 11")
	if !reflect.DeepEqual(rows, want) {
		t.Fatalf("the raise posted %+v, want one line %q", rows, want[0].Text)
	}
}

// Every slot 1..5 takes its own line of the resolved word set, in both
// classes, and slot 0 has none.
func TestSkillRiseRowsTakeTheResolvedLineForEverySlot(t *testing.T) {
	words := ui.AuthoredWords()
	for i := range words.SkillRaised {
		words.SkillRaised[i] = fmt.Sprintf("line %d", i)
	}
	for _, mage := range []bool{false, true} {
		var last, current [data.SkillSlots]int32
		for slot := range current {
			current[slot] = int32(20 + slot)
		}
		got := skillRiseRows(&words, mage, last, current)
		if len(got) != 5 {
			t.Fatalf("mage=%v: %+v, want five rows and none for slot 0", mage, got)
		}
		for i, row := range got {
			line := i
			if mage {
				line += 5
			}
			if want := fmt.Sprintf("line %d: %d", line, 21+i); row != want {
				t.Errorf("mage=%v slot %d = %q, want %q", mage, i+1, row, want)
			}
		}
	}
}

// Distinct neighbours at 129 and 140 make a one-off index fail here.
func TestSkillRaisedWordsUseMainSlots130To139(t *testing.T) {
	lines := map[int]string{129: "before", 140: "after"}
	for i := 0; i < 10; i++ {
		lines[130+i] = fmt.Sprintf("raised %d", i)
	}
	got := LoadInstallWords(installFixture{MainTextPath: textFile(142, lines)}, TextCode{}).Words().SkillRaised
	for i, s := range got {
		if want := fmt.Sprintf("raised %d", i); s != want {
			t.Errorf("SkillRaised[%d] = %q, want main.txt[%d] %q", i, s, 130+i, want)
		}
	}
	if absent := LoadInstallWords(installFixture{}, TextCode{}).Words().SkillRaised; absent != ui.AuthoredWords().SkillRaised {
		t.Fatalf("an install without main.txt resolved %q", absent)
	}
}

// A raise in the first tick after a mission opens posts its notice like any
// later raise: the baseline is the level the mission opened with, not the
// level the first tick left behind.
func TestSkillRaiseInTheFirstTickPostsItsNotice(t *testing.T) {
	mw, hero, victim := skillRaiseFixture(t)
	mw.strike(uint32(hero), uint32(victim))
	mw.tick()
	if e, _ := mw.entity(hero); e.Skill[data.SkillBlade] != 11 {
		t.Fatalf("Blade is %d after the first tick, want the raise to 11", e.Skill[data.SkillBlade])
	}
	want := announced("Blade skill improved: 11")
	if rows := mw.view.MessageLines(); !reflect.DeepEqual(rows, want) {
		t.Fatalf("the first-tick raise posted %+v, want %+v", rows, want)
	}
	mw.strike(uint32(hero), uint32(victim))
	mw.tick()
	if rows := mw.view.MessageLines(); !reflect.DeepEqual(rows, want) {
		t.Errorf("the tick after the raise changed the notice rows to %+v, want one notice only", rows)
	}
}

// A mission that opens already holding raised levels posts nothing for them
// on its first tick, which is the state a restored save opens in.
func TestSkillRaiseOpeningLevelsPostNothingInTheFirstTick(t *testing.T) {
	mw, hero, _ := skillRaiseFixture(t)
	if got := mw.skillPosted[hero][data.SkillBlade]; got != 10 {
		t.Fatalf("baseline Blade = %d at open, want the opening level 10", got)
	}
	mw.tick()
	if rows := mw.view.MessageLines(); len(rows) != 0 {
		t.Errorf("an idle first tick posted %+v, want nothing", rows)
	}
}
