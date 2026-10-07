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
				n = max(n, int64(e.TurnRemaining)+int64(c.Remaining)+int64(castRecoveryTicks(*e)))
			} else if c.Phase == bookRelaxing {
				n = max(n, int64(c.Remaining))
			}
		}
		if k, ok := w.scrollIndex(e.ID); ok && w.scrollCasts[k].Started {
			n = max(n, int64(e.TurnRemaining)+int64(w.scrollCasts[k].Remaining)+int64(castRecoveryTicks(*e)))
		}
		if m := w.motionFor(e.ID); m != nil && m.Current && m.Active {
			total := int64(binary.LittleEndian.Uint16(m.Mover[0xaa:]))
			elapsed := int64(binary.LittleEndian.Uint16(m.Mover[0xac:]))
			n = max(n, total-elapsed)
		}
		e.startAction(w.tick, n)
	}
}

// The book/scroll runner waits for its admitted turn before consuming wind-up.
// The run end uses the action's base wind-up/recovery, not later recovery RNG.
func (w *World) startSpellAction(i int) {
	e := &w.entities[i]
	e.startAction(w.tick, int64(e.TurnRemaining)+int64(castWindupTicks(*e))+int64(castRecoveryTicks(*e)))
}
