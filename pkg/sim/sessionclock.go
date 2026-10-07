package sim

// SessionClock is the independent pair serialized by original world saves
// (SAV-HEAD-025). Neither field is inferred from the other. Their wire type is
// unsigned; the ordinary wrapper's phase tests interpret SubTick as signed32.
type SessionClock struct {
	SubTick  uint32
	FullTick uint32
}

// phase is the C signed remainder used by SESS-TICK-004/006. In particular,
// masking the low four bits would incorrectly schedule work for negative
// signed subticks. Callers inspect the clock at the actual dispatch boundary;
// they must not capture one phase and reuse it after incrementing SubTick.
func (c SessionClock) phase() int32 { return int32(c.SubTick) % 16 }

func (c SessionClock) scriptDue() bool { return c.phase() == 6 }
func (c SessionClock) poolsDue() bool  { return c.phase() == 12 }
func (c SessionClock) reportDue() bool { return c.phase() == 15 }

// These are distinct mutations because the subtick increments before commands
// and actor dispatch, while the fulltick increments after reporting returns.
func (c *SessionClock) advanceSubTick()  { c.SubTick++ }
func (c *SessionClock) advanceFullTick() { c.FullTick++ }

// healthDue supplies only the fulltick filter, not positive-health, deficit,
// period or callback gates (SAV-REGENORDER-531). Dead-list and still-live decay
// have separate filters (HERO-DEATH-026 and HERO-DECAY-069).
func (c SessionClock) healthDue() bool    { return int32(c.FullTick)%4 == 0 }
func (c SessionClock) liveDecayDue() bool { return c.FullTick&3 == 0 }
func (c SessionClock) deadDecayDue() bool { return c.FullTick&1 == 0 }

// SessionClock returns the independent original-session pair when present.
// False identifies the legacy native clock policy; it does not fabricate an
// absent historical FullTick from Tick(). The returned value owns no state.
func (w *World) SessionClock() (SessionClock, bool) {
	if !w.hasSessionClock {
		return SessionClock{}, false
	}
	return SessionClock{SubTick: uint32(w.tick), FullTick: w.fullTick}, true
}

// SESS-TICK-006: the two periodic arms inspect the entry SubTick. Only then
// does the ordinary body increment it, before commands and actor dispatch.
// The internal ordering of this build's script/group/actor AI remains its
// existing bounded policy; this wrapper does not decode missing callbacks.
func (w *World) beginSessionTick(tr *ScriptTrace, obs *castObs, withdrawals *withdrawalObs) {
	c, _ := w.SessionClock()
	if tr != nil {
		tr.Tick = uint64(c.SubTick)
	}
	if c.scriptDue() {
		if tr != nil {
			tr.Pass = true
		}
		w.scriptPassObserved(tr, obs)
		w.engagementPassObserved(obs)
		w.actorPass()
		w.withdrawalPassObserved(withdrawals)
	}
	if c.poolsDue() {
		w.regenerateActors(c.healthDue())
		w.decaySessionActors(c)
	}
	c.advanceSubTick()
	w.tick = uint64(c.SubTick)
}

// The report tests SubTick again after the body. Thus entry14 reports and
// entry15 does not. FullTick changes only after that report, wrapping alone.
func (w *World) endSessionTick(tr *ScriptTrace) {
	c, _ := w.SessionClock()
	if c.reportDue() {
		if tr != nil {
			tr.Report = true
		}
		w.scriptReport()
		c.advanceFullTick()
		w.fullTick = c.FullTick
	}
	if tr != nil {
		tr.Won, tr.Lost, tr.Outcome = w.won, w.lost, w.outcome
	}
}

// The represented Fallen/Bones partition is this build's lifecycle boundary,
// not a claim that every incoming original actor state is reconstructed.
// HERO-DECAY-069 gives the still-live negative-health filter; the dead list
// uses HERO-DEATH-026's separate low-bit filter. Dwell ages only in the body.
func (w *World) decaySessionActors(c SessionClock) {
	w.decayHeldDead(c.deadDecayDue())
	var gone []EntityID
	for i := range w.entities {
		e := &w.entities[i]
		beforeHP, beforeStage := e.HP, e.Decay
		if e.Decay < DecayBones {
			hp := e.HP
			if e.CurrentProfileBasis == ProfileOriginalCurrent {
				hp = int32(int16(hp))
			}
			if c.liveDecayDue() && hp < 0 {
				wasTargetable := e.OrdinaryTargetable()
				decaySessionHealth(e)
				// Keep native lifecycle invariants when a retained unsigned
				// word is first consumed as negative, or decrement wraps it
				// positive. The original notification callback is Unknown.
				if e.Decay == DecayNone || e.HP > 0 {
					w.clearFelled(i)
				}
				e = &w.entities[i]
				if wasTargetable && !e.OrdinaryTargetable() {
					w.clearInvalidTargetReferences(e.ID)
				}
			}
			continue
		}
		if c.deadDecayDue() {
			decaySessionHealth(e)
		}
		if stage := decayStageFor(e.HP); stage > decayLast {
			gone = append(gone, e.ID)
		} else if stage > e.Decay {
			e.Decay = stage
		}
		w.syncOriginalDeadState(*e, beforeHP, beforeStage)
	}
	if len(gone) != 0 {
		w.remove(gone)
	}
}

func decaySessionHealth(e *Entity) {
	if e.CurrentProfileBasis == ProfileOriginalCurrent {
		e.setCurrentHealth(int32(int16(uint16(e.HP) - 1)))
	} else if e.HP < 0 && e.HP > minHP {
		// Native-domain arithmetic is retained; importing clocks does not
		// authorize converting a native actor's health representation.
		e.setCurrentHealth(e.HP - 1)
	}
}
