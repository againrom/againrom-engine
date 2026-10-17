package sim

import (
	"fmt"
	"testing"
)

// experienceRecipient is one attacker population the award gate is asked
// about. The type words and GainsXP values are the ones the map and party
// loaders give each population on the shipped campaign (HERO-JOIN-120 for the
// four Hero npcs, PARTY-M20-030 for mission 20's transferred Humans).
type experienceRecipient struct {
	name   string
	typeID int32
	gains  bool
	owner  uint32
	paid   bool
}

func experienceRecipients() []experienceRecipient {
	return []experienceRecipient{
		{name: "player hero", typeID: HeroTypeID(false, false), gains: true, owner: SelfSlot, paid: true},
		{name: "player mage heroine", typeID: HeroTypeID(true, true), gains: true, owner: SelfSlot, paid: true},
		// Mission 40's npc25 before it joins: a Hero npc owned by player 2.
		{name: "Hero npc before joining", typeID: 0x21, gains: true, owner: 2, paid: true},
		{name: "enemy person", typeID: 0x0a, gains: false, owner: 2, paid: false},
		{name: "neutral person", typeID: 0x17, gains: false, owner: 6, paid: false},
		// Mission 20's Sarindar after the transfer: the player's, not a hero.
		{name: "transferred person", typeID: 0x17, gains: false, owner: SelfSlot, paid: false},
		{name: "hired mercenary", typeID: 0x08, gains: false, owner: SelfSlot, paid: false},
		{name: "monster", typeID: 0x41, gains: false, owner: 3, paid: false},
	}
}

// experienceDuel lands one deterministic blow of damage on a target owned by
// a slot the attacker does not share, and returns the attacker's six slots
// and whether the target fell.
func experienceDuel(t *testing.T, r experienceRecipient, damage int32) ([skillSlots]int32, bool) {
	t.Helper()
	a := cbFighter(1, 0, 0, 1, 0)
	a.Owner, a.GainsXP, a.Mind, a.XPSlot, a.TypeID = r.owner, r.gains, 60, 3, r.typeID
	a.DamageBase = damage
	v := cbEnt(2, 1, 0)
	v.Owner, v.XPValue = 4, 10
	if r.owner == 4 {
		v.Owner = 5
	}
	w := cbWorld(t, 200, a, v)
	Step(w, []Command{cbOrder(1, 2)})
	target := cbAt(t, w, 2)
	return cbAt(t, w, 1).SkillXP, target.Dead()
}

// Only the player-character band is paid for a blow and for a kill
// (HERO-KILL-027, HERO-SKILLGATE-074, HERO-ITEMSKILL-096). Ownership by the
// player decides nothing: a Hero npc still on its own side is paid, and a
// person or a mercenary standing on the player's side is not.
func TestOnlyThePlayerCharacterBandGainsForABlowAndAKill(t *testing.T) {
	for _, r := range experienceRecipients() {
		for _, blow := range []struct {
			name   string
			damage int32
			kills  bool
		}{{"blow", 10, false}, {"kill", 105, true}} {
			t.Run(fmt.Sprintf("%s %s", r.name, blow.name), func(t *testing.T) {
				xp, dead := experienceDuel(t, r, blow.damage)
				if dead != blow.kills {
					t.Fatalf("target dead %v, want %v", dead, blow.kills)
				}
				if got := xp[3] > 0; got != r.paid {
					t.Fatalf("slot 3 holds %d: paid %v, want %v", xp[3], got, r.paid)
				}
				for slot, v := range xp {
					if slot != 3 && v != 0 {
						t.Fatalf("slot %d holds %d, want 0", slot, v)
					}
				}
			})
		}
	}
}

// The loss control: the type word alone moves a Hero npc out of the paid set,
// and a gaining class flag does not buy a person outside the band an award.
func TestTheTypeWordAloneDecidesTheAward(t *testing.T) {
	brian := experienceRecipient{typeID: 0x21, gains: true, owner: 2}
	if xp, _ := experienceDuel(t, brian, 10); xp[3] <= 0 {
		t.Fatalf("control: the band npc was not paid: %v", xp)
	}
	brian.typeID = 0x0a
	if xp, _ := experienceDuel(t, brian, 10); xp != zeroXP {
		t.Fatalf("a gaining flag outside the band was paid: %v", xp)
	}
	brian.typeID = PersistHigh
	if xp, _ := experienceDuel(t, brian, 10); xp != zeroXP {
		t.Fatalf("the first type word above the band was paid: %v", xp)
	}
}
