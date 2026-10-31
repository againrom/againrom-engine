package game

import (
	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// humanVoiceBanks are the eight human voice banks of the sound archive
// (ANIM-094), indexed by sex, male then female, and by the selector's arm.
var humanVoiceBanks = [2][4]string{
	{"m_mage", "mf_hero", "mf_merc", "m_peasant"},
	{"f_mage", "ff_hero", "ff_merc", "f_peasant"},
}

// The selector's arms in its own order (HERO-APPEAR-055): the mage pair
// first, then the hero pair, then a shown weapon, then none.
const (
	voiceArmMage = iota
	voiceArmHero
	voiceArmWeapon
	voiceArmNone
)

// voiceBank is the bank e's wounds and fall play from (ANIM-094, ANIM-096):
// the first game's voice in both games, as the second game's wound voice
// (R2-ENGINE-721) is not built. It is "" for an actor its drawn class's Sound
// array voices.
func (mw *mapWorld) voiceBank(e sim.Entity) string {
	return mw.unitVoice(e, &firstGameVoices, false).name
}

// unitVoice is one unit's voice under one game's rules.
type unitVoice struct {
	// name is the voice's directory in the sound archive under the rules'
	// dir; "" for a unit that never speaks.
	name string
	// plain is a person's voice of the arm no weapon shows.
	plain bool
	// tier is the speaker chooser's tier: 0 a hero-shaped person, 1 an armed
	// person or a hire the rules admit, 2 an unarmed person, -1 none. Without
	// tiered an unarmed mage's tier stands for both person tiers.
	tier int
}

// unitVoice is the one selector of a unit's voice and speaker tier (ANIM-094,
// VIDEO-068, R2-ENGINE-718, R2-ENGINE-719). Only a person composed as a
// figure is hero-shaped or a person (ANIM-096, UNIT-PICT-035, R2-ENGINE-680's
// face set). A hero-range drawn type reads sex and the mage axis off the id; a
// type below 0x1a reads the mage axis off ids 0x17 and 0x18 and the sex off
// the figure's directory (HERO-APPEAR-051). An identity word the rules name
// picks its own voice by sex and mage; otherwise the arm chooses among the
// rules' voices: the mage pair, the hero pair where the rules have one, a
// shown weapon, then none. The weapon is read only where the voice or, when
// tiered, the tier needs it.
func (mw *mapWorld) unitVoice(e sim.Entity, r *unitVoiceRules, tiered bool) unitVoice {
	fig, person := mw.figures[e.ID]
	t := e.TypeID
	if mw.mission != nil {
		t = drawnTypeID(e, mw.mission.drawnHeroes)
	}
	female, mage, armed, tier := false, false, false, -1
	switch {
	case person && data.FigureIsHero(t):
		female, mage, tier = (t-0x21)&1 != 0, (t-0x21)&2 != 0, 0
		armed = !r.heroArm && !mage && mw.showsWeapon(e.ID)
	case person && data.ComposesFigure(t):
		female, mage = fig.Dir.Female(), t == 0x17 || t == 0x18
		armed = (tiered || !mage) && mw.showsWeapon(e.ID)
		tier = 2
		if armed {
			tier = 1
		}
	}
	word, identified := uint16(0), false
	if r.named != nil || r.hires[1] != 0 {
		word, identified = mw.identityWord(e.ID)
	}
	if tier < 0 && identified && word >= r.hires[0] && word <= r.hires[1] {
		tier = 1
	}
	if tier < 0 {
		return unitVoice{tier: -1}
	}
	sex, by := 0, 0
	if female {
		sex, by = 1, 1
	}
	if mage {
		by += 2
	}
	if names, ok := r.named[word]; identified && ok {
		return unitVoice{name: names[by], tier: tier}
	}
	arm := voiceArmNone
	switch {
	case mage:
		arm = voiceArmMage
	case tier == 0 && r.heroArm:
		arm = voiceArmHero
	case armed:
		arm = voiceArmWeapon
	}
	return unitVoice{name: r.voices[sex][arm], plain: arm == voiceArmNone, tier: tier}
}

// identityWord is the unit's identity word, the second game's unit +0x1dc
// (R2-ENGINE-718): a party member's party identity (R2-ENGINE-105), else the
// NPC word of its placement (DIV-2381).
func (mw *mapWorld) identityWord(id sim.EntityID) (uint16, bool) {
	if mw.mission == nil {
		return 0, false
	}
	if member := mw.missionPartyMember(id); member != nil {
		if word, ok := mapload.PartyIdentity(*member, mw.mission.table); ok {
			return uint16(word), true
		}
	}
	word, ok := mw.mission.npcKeys[id]
	return word, ok
}

// showsWeapon reports whether id's figure shows a weapon in its first visible
// slot, which chooses the mercenary bank over the peasant bank
// (HERO-APPEAR-055). For a party or roster member that is the doll's own
// slot 1, which shows the starting weapon until it materializes.
func (mw *mapWorld) showsWeapon(id sim.EntityID) bool {
	eq := mw.equipmentOf(id)
	member := mw.missionPartyMember(id)
	if member == nil && mw.mission != nil && mw.mission.state != nil {
		if placed, ok := mw.mission.state.Start.Roster[id]; ok {
			member = &placed
		}
	}
	if member != nil {
		eq, _ = missionDollEquipment(mw.world, id, *member)
	}
	occupied, _ := eq.Occupied(1)
	return occupied
}
