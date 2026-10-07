package sim

import "fmt"

// OriginalActorAction is an admission batch, not another persisted graph.
// The ordinary actor programme owns its run deadline, attack phase/countdown,
// target and surviving kill attribution. EntityID references have already
// passed the document's exact key map at the game boundary.
type OriginalActorAction struct {
	Entity                 EntityID
	HasTarget              bool
	Target                 EntityID
	Phase                  AttackPhase
	Countdown              int32
	PendingAttackTarget    EntityID
	HasPendingAttackTarget bool
	HasCredit              bool
	Credit                 EntityID
	CreditSpell            int8
	ActorState             uint8
	PostX, PostY           int32
}

func (w *World) ImportOriginalActorActions(actions []OriginalActorAction) error {
	if w == nil {
		return fmt.Errorf("sim: actor actions require a world")
	}
	staged := *w
	staged.entities = append([]Entity(nil), w.entities...)
	seen := map[EntityID]bool{}
	for _, a := range actions {
		i := indexOfEntity(staged.entities, a.Entity)
		if i < 0 || seen[a.Entity] {
			return fmt.Errorf("sim: actor action has absent or duplicate entity %d", a.Entity)
		}
		seen[a.Entity] = true
		e := &staged.entities[i]
		state := a.ActorState
		if escortState(state) {
			// A raw escort state becomes native state only with its repaired
			// target. Unresolved source orders stay pending in SavedGroups.
			o := staged.savedOrder(a.Entity)
			if !e.Alive() || o == nil || o.State != uint32(state) || !o.EscortBound || !savedEscortBindingValid(*o, staged.entities, staged.originalDead) {
				state = 0
			} else {
				e.EscortTarget, e.HasEscortTarget, e.EscortRange = o.EscortTarget, true, o.Raw[0x70]
			}
		}
		if !e.Alive() && livingOnlyState(state) {
			// A body holds no order a living actor holds: the state is
			// residue, dropped as the World constructor drops it (DIV-1433).
			state = 0
		}
		if state != 0 {
			e.ActorState, e.PostX, e.PostY = state, a.PostX, a.PostY
		}
		if a.HasTarget {
			// HERO-DYINGTICK-145: a pursuer whose blow lands while it is still
			// charging keeps that order frozen through its own dying window,
			// because the per-tick dying branch never reaches the order
			// machine that would otherwise clear it. A restored batch is
			// therefore admitted for a still-dying entity on the same terms as
			// a living one; only a torn-down or otherwise absent entity is
			// refused here.
			if a.Target == a.Entity || indexOfEntity(staged.entities, a.Target) < 0 || (!e.Alive() && !e.Dying()) {
				return fmt.Errorf("sim: actor %d has invalid restored attack target", a.Entity)
			}
			e.AttackTarget, e.HasAttackTarget, e.AttackTargetKind = a.Target, true, AttackTargetUnit
			e.AcquirePursuit = false
			e.AttackPhase, e.AttackCountdown = a.Phase, a.Countdown
			if err := attackFault(*e); err != nil {
				return fmt.Errorf("sim: actor %d restored attack: %w", a.Entity, err)
			}
		}
		if a.HasCredit {
			if indexOfEntity(staged.entities, a.Credit) < 0 {
				return fmt.Errorf("sim: actor %d has invalid restored credit", a.Entity)
			}
			e.KillCreditSource, e.HasKillCredit = a.Credit, true
		}
		if a.HasPendingAttackTarget {
			e.PendingAttackTarget, e.PendingAttackTargetKind, e.HasPendingAttackTarget = a.PendingAttackTarget, AttackTargetUnit, true
		}
		e.KillCreditSpell = a.CreditSpell
	}
	if err := staged.pendingAttackFault(); err != nil {
		return err
	}
	w.entities = staged.entities
	return nil
}
