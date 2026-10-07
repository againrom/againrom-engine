package game

import (
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// Simulation producers already committed the complete source sheet. This is
// presentation only, not a delayed derived-state repair.
func (mw *mapWorld) refreshSourceCharacter(e sim.Entity) {
	h := mapload.SourceHumanState(e.SourceNow(), e.ActorLoad.Accumulator)
	if ch, ok := mw.chars[e.ID]; ok {
		ch.Body, ch.Reaction, ch.Mind, ch.Spirit = int(int16(h.Body)), int(int16(h.Reaction)), int(int16(h.Mind)), int(int16(h.Spirit))
		ch.Experience, ch.Sight, ch.Sight256 = int(int32(h.Experience)), int(h.Sight>>8), h.Sight
		for i := range ch.Skills {
			ch.Skills[i] = int(e.Skill[i])
		}
		for i := range ch.Protection {
			ch.Protection[i], ch.Resistance[i] = int(e.Protection[i]), int(e.Resistance[i])
		}
		mw.chars[e.ID] = ch
	}
}
