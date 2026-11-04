package sim

import "testing"

// registryOrder reads the order byte of the Group that holds entity i in the
// World's Group registry, the record every first-game World decides from.
func registryOrder(w *World, i int) (order uint8, ok bool) {
	g := w.savedGroupFor(w.entities[i].ID)
	if g == nil {
		return 0, false
	}
	return g.AI[0x20], true
}

// regGroup is the registry Group of owner with the selector, the record a
// native group record (owner, group) became: AI +0x20 the order, +0x0a/+0x0b
// the commanded cell, +0x15 the roam counter, +0x38 the notice base.
func regGroup(t *testing.T, w *World, owner, selector uint32) *SavedGroup {
	t.Helper()
	for i := range w.savedGroups.Groups {
		if g := &w.savedGroups.Groups[i]; g.Owner.Owner == owner && g.Selector == selector {
			return g
		}
	}
	t.Fatalf("no registry Group (owner %d, selector %d)", owner, selector)
	return nil
}

// regState reads the registry order and notice base of (owner, selector).
func regState(w *World, owner, selector uint32) (order, base uint8, ok bool) {
	if g := regFind(w, owner, selector); g != nil {
		return g.AI[0x20], g.AI[0x38], true
	}
	return 0, 0, false
}

// regCommandedCell is groupCommandedCell over the decision record.
func regCommandedCell(w *World, owner, selector uint32) (x, y int32, ok bool) {
	if g := regFind(w, owner, selector); g != nil {
		return int32(g.AI[10]), int32(g.AI[11]), true
	}
	return 0, 0, false
}

func regFind(w *World, owner, selector uint32) *SavedGroup {
	for i := range w.savedGroups.Groups {
		if g := &w.savedGroups.Groups[i]; g.Owner.Owner == owner && g.Selector == selector {
			return g
		}
	}
	return nil
}

// regDecide runs the registry decision for (owner, selector).
func regDecide(t *testing.T, w *World, owner, selector uint32, members []int) {
	t.Helper()
	g := regGroup(t, w, owner, selector)
	order := w.savedRoute(g)
	if order == orderRoam {
		living, _ := w.savedLiving(g)
		w.savedRoam(g, living)
		order = orderSwarm2
	}
	w.savedDecision(g, order, nil)
}

// regWithdrawalPass runs the withdrawal stage alone: on a registry World the
// dispatch's tail over active Groups, then the patrol withdrawal pass.
func regWithdrawalPass(w *World, obs *withdrawalObs) {
	for gi := 0; gi < len(w.savedGroups.Groups); gi++ {
		if g := &w.savedGroups.Groups[gi]; g.AI[0x45] != 0 || w.safeMode {
			w.savedWithdrawalTail(g, obs)
		}
	}
	w.withdrawalPassObserved(obs)
}

// regActorPass runs the per-actor layer alone: on a registry World the
// order-0 Groups' members through the actor executor, as the dispatch does,
// then actorPass for patrollers with native order storage.
func regActorPass(w *World) {
	for gi := 0; gi < len(w.savedGroups.Groups); gi++ {
		if g := &w.savedGroups.Groups[gi]; g.AI[0x20] == orderNone && (g.AI[0x45] != 0 || w.safeMode) {
			w.walkSavedMembers(g, func(i int) {
				if !w.nativePatrol(i) {
					w.savedActorDispatch(i)
				}
			})
		}
	}
	w.actorPass()
}

// registryFormGrowth is the byte count a World's Group registry adds to its
// form beyond the constructed registry the form omits. The activity gate
// rewrites +0x45 from party coverage each tick (AI-ACTIVITY-324), so a stepped
// World's registry usually differs from the constructed one and is written.
func registryFormGrowth(w *World) int {
	c := *w
	c.savedGroups = c.constructedGroups(false)
	return len(w.encode()) - len(c.encode())
}
