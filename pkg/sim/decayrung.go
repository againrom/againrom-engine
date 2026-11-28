package sim

// The one corpse decay ladder. Each pass picks its bodies and its clock and
// hands the rest to settleDecay as data.

// decayDue reports whether the native clock's walk takes one health at tick.
func decayDue(tick uint64) bool { return tick%decayCycle == decayPhase }

// decayTeardown names the steps a body makes when it leaves the ladder.
type decayTeardown struct {
	Kill    bool // record the kill a downed body leaves unseen
	Release bool // free the footprint of a body not yet torn down
	Detach  bool // leave the saved group
	Rank    bool // join the dead list
	RankNew bool // join only a body that holds no rank yet
	Expire  bool // expire the attachments once the pass is closed
}

// decayCall is one body's place on the ladder and what its caller differs in.
type decayCall struct {
	Index    int // entity index, or -1 for a held record
	HP       int32
	Stage    DecayStage
	BeforeHP int32
	Settled  bool // the rung is read already; only the teardown remains
	Fresh    bool // the body has not made its teardown yet
	OnGone   decayTeardown
	OnLeft   decayTeardown // a body whose corpse is finished
}

type decayOutcome struct {
	Stage  DecayStage
	Gone   bool
	Expire bool
}

// settleDecay reads the rung health names, raises the stage to it and makes the
// teardown. A held record has no entity, so its caller writes the outcome back.
func (w *World) settleDecay(c decayCall) decayOutcome {
	out := decayOutcome{Stage: c.Stage}
	if !c.Settled {
		s := w.game.decay.stage(c.HP)
		if s > decayLast {
			out.Gone = true
			out.Expire = c.OnGone.Expire
			w.decayTeardown(c, c.OnGone)
			return out
		}
		out.Stage = max(out.Stage, s)
		if c.Index >= 0 {
			w.entities[c.Index].Decay = out.Stage
		}
	}
	if c.Index >= 0 && c.Fresh && w.entities[c.Index].CorpseRemoved() {
		out.Expire = c.OnLeft.Expire
		w.decayTeardown(c, c.OnLeft)
	}
	return out
}

func (w *World) decayTeardown(c decayCall, t decayTeardown) {
	i := c.Index
	if i < 0 {
		return
	}
	if t.Kill && LifeNonNegative.Holds(int64(c.BeforeHP), 0, false) {
		w.recordDiaryKill(i)
	}
	if t.Release && c.Fresh {
		w.releaseAtTeardown(i)
	}
	if t.Detach {
		w.detachSavedMember(w.entities[i].ID)
	}
	if t.Rank && (!t.RankNew || w.entities[i].deadRank == 0) {
		w.joinDeadList(i)
	}
}

// closeDecay ends a pass: expiry may copy the world, so it runs after the walk.
func (w *World) closeDecay(gone, expiring []EntityID) {
	for _, id := range expiring {
		w.expireBodyAttachments(id)
	}
	if len(gone) > 0 {
		w.remove(gone)
	}
}
