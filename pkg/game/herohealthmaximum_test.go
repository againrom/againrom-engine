package game_test

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/game"
	"againrom/pkg/mapload"
)

// TestMissionPartyHealthMaximumComesFromTheDerivation is this story's own
// regression. Before it, a party built through MissionParty's Table-less
// fallback (chargenProfile's "no base row resolves" arm, pkg/game/hero.go)
// carried HealthColumn false, so mapload.PartySpawn minted the hero at the
// provisional mapload.SpawnHP (100) regardless of his authored Body and
// trained skill -- the owner's own witnessed defect (docs/1224, docs/1225).
// HealthColumn now reads true on this exact arm, so the same party's health
// maximum is HERO-HP-072's own graph, over the same inputs Hero.Recompute
// already used for every party member carrying a resolved base row: Body 43
// doubled for the fighter's clear class bit, folded with the trained Blade
// skill's own experience term.
func TestMissionPartyHealthMaximumComesFromTheDerivation(t *testing.T) {
	party := game.MissionParty(nil, nil, nil)
	if len(party) != 1 {
		t.Fatalf("party of %d, want 1", len(party))
	}
	m := party[0]
	if !m.Profile.HealthColumn {
		t.Fatalf("Profile.HealthColumn = false, want true")
	}
	d, health, _ := mapload.PartySpawn(m)
	if health == mapload.SpawnHP {
		t.Fatalf("health = %d, the provisional constant -- want the derivation's own maximum", health)
	}
	if health != d.HealthMax || health <= 0 {
		t.Fatalf("health = %d, d.HealthMax = %d -- want the recompute's own positive maximum", health, d.HealthMax)
	}
	// DIV-1390 names the exact gap: an original resave of this identical
	// fixture (same Body, same trained skill, replayed through no chargen
	// screen of its own) reads a health maximum of 137, eight under this
	// number. This assertion pins what Hero.Recompute -- implemented from
	// HERO-HP-072's own text -- actually produces today, not the ground
	// number the resave shows.
	if d.HealthMax != 145 {
		t.Fatalf("HealthMax = %d, want 145 (DIV-1390 records the gap against a 137 ground resave)", d.HealthMax)
	}
	if m.Hero.Body != 43 || m.Hero.Skill[data.SkillBlade] != 10 {
		t.Fatalf("fixture drifted: Body=%d Skill[Blade]=%d", m.Hero.Body, m.Hero.Skill[data.SkillBlade])
	}
}
