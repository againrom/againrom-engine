package mapload

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/herokind"
)

// The starting hero is the member marked so, else the first; its kind is
// its class and its figure's sex, the composed selector order.
func TestStartingHeroKindReadsTheMarkedMembersClassAndSex(t *testing.T) {
	for _, tc := range []struct {
		mage bool
		dir  data.FigureDir
		want herokind.Kind
	}{
		{false, data.FigureDirManFighter, herokind.MaleFighter},
		{false, data.FigureDirWomanFighter, herokind.FemaleFighter},
		{true, data.FigureDirManMage, herokind.MaleMage},
		{true, data.FigureDirWomanMage, herokind.FemaleMage},
	} {
		hero := PartyMember{ID: "hero", StartingHero: true, Mage: tc.mage, FigureDir: string(tc.dir)}
		guest := PartyMember{ID: "npc:22", Mage: !tc.mage, FigureDir: string(data.FigureDirWomanMage)}
		if got := StartingHeroKind(HeroHold{Party: []PartyMember{guest, hero}}); got != tc.want {
			t.Errorf("marked %v: %v", tc.want, got)
		}
		hero.StartingHero = false
		if got := StartingHeroKind(HeroHold{Party: []PartyMember{hero, guest}}); got != tc.want || StartingHero([]PartyMember{hero, guest}).ID != "hero" {
			t.Errorf("first %v: %v", tc.want, got)
		}
	}
	if got := StartingHeroKind(HeroHold{}); got != herokind.MaleFighter {
		t.Errorf("no party: %v", got)
	}
}
