package sim

import "encoding/binary"

// ActionClock records the simulation run's expected end, never a rendered frame
// or wall time. Unknown clocks (old/native or original LOAD) begin on the first
// executed tick, after any already represented action interval. DIV-1016.
type ActionClock struct {
	Known bool
	End   uint32
}

func (e *Entity) startAction(now uint64, duration int64) {
	if e.HealthRegenPeriod == 0 && e.ManaRegenPeriod == 0 {
		return
	}
	e.ActionClock = ActionClock{Known: true, End: uint32(now) + uint32(duration)}
}

// SAV-REGENORDER-531: subtract as wrapping dwords, then compare signed.
func (e *Entity) regenerationRate(now uint64) int32 {
	if e.ActionClock.Known && int32(uint32(now)-e.ActionClock.End) > 80 {
		return 3
	}
	return 1
}

func (w *World) initializeActionClocks() {
	for i := range w.entities {
		e := &w.entities[i]
		if !e.Alive() || e.ActionClock.Known {
			continue
		}
		n := max(int64(e.Transit), int64(e.TurnRemaining), int64(e.CastWait))
		switch e.AttackPhase {
		case AttackCharging, AttackCasting:
			n = max(n, int64(e.AttackCountdown)+relaxTicks(*e))
		case AttackRelaxing:
			n = max(n, int64(e.AttackCountdown))
		}
		if k, ok := w.bookCastIndex(e.ID); ok {
			c := w.bookCasts[k]
			if c.Phase == bookCharging {
				n = max(n, int64(e.TurnRemaining)+int64(c.Remaining)+w.castRecoveryTicks(i, originEstimate, SpellRule{}))
			} else if c.Phase == bookRelaxing {
				n = max(n, int64(c.Remaining))
			}
		}
		if k, ok := w.scrollIndex(e.ID); ok && w.scrollCasts[k].Started {
			n = max(n, int64(e.TurnRemaining)+int64(w.scrollCasts[k].Remaining)+w.castRecoveryTicks(i, originEstimate, SpellRule{}))
		}
		if m := w.motionFor(e.ID); m != nil && m.Current && m.Active {
			total := int64(binary.LittleEndian.Uint16(m.Mover[0xaa:]))
			elapsed := int64(binary.LittleEndian.Uint16(m.Mover[0xac:]))
			n = max(n, total-elapsed)
		}
		e.startAction(w.tick, n)
	}
}

// startSpellAction stores a book or scroll cast's activity stamp at moment m,
// where the game's castStamp row stamps one.
func (w *World) startSpellAction(i int, m castMoment) {
	if d, ok := castStamps[w.game.castStamp](w, i, m); ok {
		w.entities[i].startAction(w.tick, d)
	}
}

// castMoment is a point of a cast's run that may stamp its activity.
type castMoment uint8

const (
	// castBookAdmitted: a book order is admitted and its turn set.
	castBookAdmitted castMoment = iota
	// castScrollStarted: a scroll cast starts and its turn is set.
	castScrollStarted
	// castBookPaid: a book cast pays its cost.
	castBookPaid
	// castScrollWindup: a scroll cast's wind-up runs its first tick, its
	// reader facing the aim.
	castScrollWindup
)

// castStampRule is when a cast stamps its activity and how far ahead.
type castStampRule uint8

const (
	// castStampsRunEnd: the first game's book/scroll runner waits for its
	// admitted turn before consuming wind-up, so the stamp is the run end:
	// turn, base wind-up and base recovery, not later recovery RNG.
	castStampsRunEnd castStampRule = iota
	// castStampsRoutine: the second game's cast routine runs once the caster
	// faces the aim, and its sender stores the bare session counter
	// (R2-ENGINE-543, R2-ENGINE-926, R2-ENGINE-1565): a book cast on the
	// tick it pays, a scroll cast on its wind-up's first tick, the turn's end.
	castStampsRoutine
)

var castStamps = [...]func(*World, int, castMoment) (int64, bool){
	castStampsRunEnd: func(w *World, i int, m castMoment) (int64, bool) {
		e := w.entities[i]
		return int64(e.TurnRemaining) + int64(castWindupTicks(e)) + w.castRecoveryTicks(i, originEstimate, SpellRule{}), m == castBookAdmitted || m == castScrollStarted
	},
	castStampsRoutine: func(w *World, i int, m castMoment) (int64, bool) {
		return 0, m == castBookPaid || m == castScrollWindup
	},
}
