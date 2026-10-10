package game

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
)

// heroExportedHealth opens mission with party, exports the current save
// through the real ExportCurrentSave path and returns the party's own hero
// Human record's live health pair, read the same way cmd/savtool's party walk
// reads it.
func heroExportedHealth(t *testing.T, f *FrontEnd, party []mapload.PartyMember, mission int, label string) (health, healthMax uint16) {
	t.Helper()
	c := heroExportedCharacter(t, f, party, mission, label)
	return c.Stat(sav.StatHealth), c.Stat(sav.StatHealthMax)
}

// heroExportedCharacter is heroExportedHealth's own setup, returning the
// full sav.Character record so a caller can read any exported field, not
// only the health pair.
func heroExportedCharacter(t *testing.T, f *FrontEnd, party []mapload.PartyMember, mission int, label string) sav.Character {
	t.Helper()
	app := f.App(label)
	if err := app.OpenMission(f.MissionOpenerWith(mission, party)); err != nil {
		t.Fatal(err)
	}
	snapshot, snapLabel, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	snapshot.NativeMissionTerrain = true
	raw, err := f.ExportCurrentSave(snapshot, snapLabel)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	chars, _, err := doc.PartyWalk()
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range chars {
		if c.Hero {
			return c
		}
	}
	t.Fatal("no hero record in the human participant's own subtree")
	return sav.Character{}
}

// TestReleaseGeneratedHeroAggregateExperienceMatchesItsSkillSlots is this
// story's own sole-review correction for its returning finding: the SAV
// writer wrote actor+0x130 (HERO-XP-077's running six-slot experience aggregate,
// the operand the original's own health and mana maximum formulas read,
// HERO-HP-005/HERO-GENERAL-091) as 0 for every engine-minted Human, beside a
// nonzero six-slot SkillXP sum. currentActorSource now builds that aggregate
// from the live SkillXP slots for a Human or Humanoid record with no source
// basis yet, the same sum famestate.go's campaign score already uses
// (DIV-1267). Checked for the Table-less recipe the owner kit uses
// (MissionParty(nil, nil, nil)) and the ordinary production recipes a real
// installed Table drives, fighter and mage. Reverting currentActorSource's
// fix fails every case here; TestMissionPartyHealthMaximumComesFromTheDerivation
// and the ground-truth test above stay silent about it because neither reads
// U130.
func TestReleaseGeneratedHeroAggregateExperienceMatchesItsSkillSlots(t *testing.T) {
	recipes := map[string]func(table *mapload.Table) []mapload.PartyMember{
		"table-less":         func(*mapload.Table) []mapload.PartyMember { return MissionParty(nil, data.BodyList{}, nil) },
		"production-fighter": func(table *mapload.Table) []mapload.PartyMember { return MissionParty(nil, data.BodyList{}, table) },
		"production-mage": func(table *mapload.Table) []mapload.PartyMember {
			return MissionPartyAs(true, nil, data.BodyList{}, table)
		},
	}
	for name, build := range recipes {
		t.Run(name, func(t *testing.T) {
			f := releaseFront(t)
			c := heroExportedCharacter(t, f, build(f.Table), 10, "aggregate experience "+name)
			var want uint32
			for _, xp := range c.SkillXP {
				want += xp
			}
			if c.Experience != want {
				t.Fatalf("exported aggregate experience (U130) = %d, want %d (the sum of its six skill-XP slots, HERO-XP-077)", c.Experience, want)
			}
		})
	}
}

// TestReleaseGeneratedHeroHealthMaximumFollowsTheDerivation asserts this
// story's own fix reaches the exported document, not only mapload's own
// return values: a party opened with no chargen screen at all -- the exact
// MissionParty(nil, nil, nil) recipe the prior owner kit used, and the
// recipe behind the owner's own witnessed constant 100 -- writes the
// SAME positive maximum mapload.PartySpawn derives onto both the current and
// the maximum health fields of the exported hero record, for missions 10 and
// 20 alike. It does not assert a literal number: Hero.Recompute's own value
// for this fixture is asserted once, in
// TestMissionPartyHealthMaximumComesFromTheDerivation, and DIV-1390 records
// that number's own remaining gap against an original resave.
func TestReleaseGeneratedHeroHealthMaximumFollowsTheDerivation(t *testing.T) {
	for _, mission := range []int{10, 20} {
		t.Run(missionSubtestName(mission), func(t *testing.T) {
			f := releaseFront(t)
			party := MissionParty(nil, data.BodyList{}, nil)
			d, wantHealth, _ := mapload.PartySpawn(party[0])
			if wantHealth != d.HealthMax || wantHealth == mapload.SpawnHP {
				t.Fatalf("setup: PartySpawn health=%d HealthMax=%d SpawnHP=%d", wantHealth, d.HealthMax, mapload.SpawnHP)
			}
			health, healthMax := heroExportedHealth(t, f, party, mission, "generated hero health maximum")
			if int32(healthMax) != wantHealth {
				t.Errorf("exported health maximum = %d, want %d (mapload.PartySpawn's own derived maximum)", healthMax, wantHealth)
			}
			if int32(health) != wantHealth {
				t.Errorf("exported current health = %d, want %d (current equals maximum at creation, HERO-HP-072)", health, wantHealth)
			}
		})
	}
}

// TestReleaseHeroHealthMaximumGroundTruthResave is the aggregate-0 witness
// the review's correction retargets it as (DIV-1390, closed): this file is
// the original EN client's own resave, after play (gameversions/saves), of
// the prior owner kit's game9250.sav -- byte-identical to what
// MissionParty(nil, nil, nil) still generates in body 43/reaction 26/mind
// 15/spirit 15 and the Blade skill trained at level 10, six-slot skill-XP
// sum 1593. It also carries aggregate experience (U130) 0: the value our SAV
// writer stored before this correction, for every engine-minted Human. The
// original's own health derive reads that aggregate (HERO-HP-005,
// HERO-XP-077, HERO-GENERAL-091), so it recomputes 137 from a zero operand
// here -- eight under TestMissionPartyHealthMaximumComesFromTheDerivation's
// 145, which is the same formula over the correct aggregate, 1593. This is
// not a second, lower ground truth for the formula: it is what the formula
// gives when handed the wrong operand, pinned so a future regression that
// reintroduces the zero has a fixture to fail against.
func TestReleaseHeroHealthMaximumGroundTruthResave(t *testing.T) {
	_, payload := groundCorpusFile(t, "2026-09-24/exp-engine-lineage/game0000-original-resave9250.sav",
		"a1bc3c65ba5b384dc0802449cba6d45f5a167eecd0ed3a1e707e34a9e0da071d")
	f, err := sav.Open(payload)
	if err != nil {
		t.Fatal(err)
	}
	chars, _, err := f.PartyWalk()
	if err != nil {
		t.Fatal(err)
	}
	if len(chars) != 1 || !chars[0].Hero {
		t.Fatalf("resave party = %d character(s), want exactly one hero", len(chars))
	}
	c := chars[0]
	if c.Stat(sav.StatBody) != 43 || c.Stat(sav.StatReaction) != 26 ||
		c.Stat(sav.StatMind) != 15 || c.Stat(sav.StatSpirit) != 15 {
		t.Fatalf("resave stats = %d/%d/%d/%d, want 43/26/15/15",
			c.Stat(sav.StatBody), c.Stat(sav.StatReaction), c.Stat(sav.StatMind), c.Stat(sav.StatSpirit))
	}
	wantSkill := [6]uint16{0, 10, 0, 0, 0, 0}
	if c.SkillLevels != wantSkill {
		t.Fatalf("resave skills = %v, want %v", c.SkillLevels, wantSkill)
	}
	wantXP := [6]uint32{0, 1593, 0, 0, 0, 0}
	if c.SkillXP != wantXP {
		t.Fatalf("resave skill XP = %v, want %v", c.SkillXP, wantXP)
	}
	if c.Experience != 0 {
		t.Fatalf("resave aggregate experience (U130) = %d, want 0 (this fixture is the zero-aggregate witness, not the corrected-writer case)", c.Experience)
	}
	if got := c.Stat(sav.StatHealthMax); got != 137 {
		t.Fatalf("resave health maximum = %d, want 137 (the derive over aggregate 0, DIV-1390)", got)
	}
}
