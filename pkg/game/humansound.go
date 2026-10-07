package game

import (
	"againrom/pkg/data"
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

// voiceBank is the one voice bank every voice request of e plays from: its
// wounds and fall, and its selection and command replies, all of which the
// original serves from one selector (ANIM-094, ANIM-096). It is "" for an
// actor its drawn class's Sound array voices. Only a human composed as a
// figure has a bank (ANIM-096, UNIT-PICT-035). A hero-range type id sets the
// hero arm and reads sex and the mage axis off the id itself; a type id below
// 0x1a reads the mage axis off ids 0x17 and 0x18 and the sex off the bit that
// also picks the figure's directory (HERO-APPEAR-051), so the sex the figure
// is drawn with. Equipment and the drawn class choose nothing but the weapon
// arm, so a person no weapon shows takes the peasant pair.
func (mw *mapWorld) voiceBank(e sim.Entity) string {
	fig, human := mw.figures[e.ID]
	if !human {
		return ""
	}
	t, female, arm := e.TypeID, fig.Dir.Female(), voiceArmNone
	switch {
	case data.FigureIsHero(t):
		female, arm = (t-0x21)&1 != 0, voiceArmHero
		if (t-0x21)&2 != 0 {
			arm = voiceArmMage
		}
	case data.ComposesFigure(t):
		if t == 0x17 || t == 0x18 {
			arm = voiceArmMage
		} else if mw.showsWeapon(e.ID) {
			arm = voiceArmWeapon
		}
	default:
		return ""
	}
	sex := 0
	if female {
		sex = 1
	}
	return humanVoiceBanks[sex][arm]
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

// voiceTier is the speaker chooser's tier of e (VIDEO-068): 0 for a
// hero-shaped drawable, 1 for an armed human, 2 for an unarmed human and -1
// for an actor with no voice bank, which the chooser drops.
func (mw *mapWorld) voiceTier(e sim.Entity) int {
	if mw.voiceBank(e) == "" {
		return -1
	}
	switch {
	case data.FigureIsHero(e.TypeID):
		return 0
	case mw.showsWeapon(e.ID):
		return 1
	}
	return 2
}
