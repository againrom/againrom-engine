package data

import (
	"testing"

	"againrom/pkg/rules"
)

func derivedStats(d Derived) [4]int32 { return [4]int32{d.Body, d.Reaction, d.Mind, d.Spirit} }

// The second game's class-and-sex caps are the ceiling for earned gains; the
// first game keeps 50 (R2-ENGINE-290).
func TestSecondGameStatCapBasePerClassAndSex(t *testing.T) {
	second := rules.Rules{}.WithClassStatCaps()
	for _, c := range []struct {
		mage, female bool
		want         [4]int32
	}{
		{false, false, [4]int32{52, 50, 48, 46}},
		{false, true, [4]int32{50, 52, 46, 48}},
		{true, false, [4]int32{48, 46, 52, 50}},
		{true, true, [4]int32{46, 48, 50, 52}},
	} {
		if got := second.StatCapBase(c.mage, c.female); got != c.want {
			t.Errorf("mage %v female %v: %v, want %v", c.mage, c.female, got, c.want)
		}
		if got := (rules.Rules{}).StatCapBase(c.mage, c.female); got != [4]int32{50, 50, 50, 50} {
			t.Errorf("first game: %v, want 50s", got)
		}
	}
}

// A base above the cap that a SAV or map holds plays as stored up to the
// ceiling; an item's bonus adds above the cap; the first game keeps 50.
func TestSecondGameStatAboveCapPlays(t *testing.T) {
	second := rules.Rules{}.WithClassStatCaps()
	profile := Profile{Fighter: true}
	over := Hero{Body: 90, Reaction: 60, Mind: 120, Spirit: 30000}
	if got := derivedStats(over.Recompute(profile, Loadout{Rules: second})); got != [4]int32{90, 60, 100, 100} {
		t.Errorf("native above cap: %v", got)
	}
	if got := derivedStats(over.Recompute(profile, Loadout{})); got != [4]int32{50, 50, 50, 50} {
		t.Errorf("first game: %v", got)
	}
	atCap := Hero{Body: 52, Reaction: 50, Mind: 48, Spirit: 46}
	item := Loadout{Rules: second, Mod: EquipMod{Body: 5, Mind: 7}}
	if got := derivedStats(atCap.Recompute(profile, item)); got != [4]int32{57, 50, 55, 46} {
		t.Errorf("item above cap: %v", got)
	}
	state := capState(0)
	state.Body, state.Reaction, state.Mind, state.Spirit = 90, 60, 120, 30000
	out, err := state.WithRules(second).Derive()
	if err != nil {
		t.Fatal(err)
	}
	if got := [4]int32{int32(out.Body), int32(out.Reaction), int32(out.Mind), int32(out.Spirit)}; got != [4]int32{90, 60, 100, 100} {
		t.Errorf("stored above cap: %v", got)
	}
}
