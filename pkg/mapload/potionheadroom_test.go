package mapload

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/rules"
)

// A potion raises a base statistic only up to the game's cap for the hero's
// class and sex; a base already above the cap leaves no headroom and breaks
// nothing.
func TestPotionHeadroomUsesClassAndSexCap(t *testing.T) {
	second := rules.Rules{}.WithClassStatCaps()
	hero := data.Hero{Body: 40, Reaction: 40, Mind: 40, Spirit: 60}
	for _, c := range []struct {
		profile data.Profile
		r       rules.Rules
		want    [4]int32
	}{
		{data.Profile{Fighter: true}, second, [4]int32{12, 10, 8, 0}},
		{data.Profile{Fighter: true, Female: true}, second, [4]int32{10, 12, 6, 0}},
		{data.Profile{}, second, [4]int32{8, 6, 12, 0}},
		{data.Profile{Female: true}, second, [4]int32{6, 8, 10, 0}},
		{data.Profile{Fighter: true}, rules.Rules{}, [4]int32{10, 10, 10, 0}},
	} {
		d := data.Derived{Body: 40, Reaction: 40, Mind: 40, Spirit: 60}
		if got := PotionHeadroom(hero, d, c.profile, c.r); got != c.want {
			t.Errorf("%+v: %v, want %v", c.profile, got, c.want)
		}
	}
}
