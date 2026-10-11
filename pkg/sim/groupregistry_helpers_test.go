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

// regState is groupState over the World's decision record: the registry Group
// of (owner, selector) when the World holds a registry, else the native record.
// It reads the order, the notice base and whether the pair has a Group.
func regState(w *World, owner, selector uint32) (order, base uint8, ok bool) {
	if w.savedGroups == nil {
		return w.groupState(owner, selector)
	}
	if g := regFind(w, owner, selector); g != nil {
		return g.AI[0x20], g.AI[0x38], true
	}
	return 0, 0, false
}

// regCommandedCell is groupCommandedCell over the decision record.
func regCommandedCell(w *World, owner, selector uint32) (x, y int32, ok bool) {
	if w.savedGroups == nil {
		return w.groupCommandedCell(owner, selector)
	}
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

// regDecide runs one Group decision for (owner, selector) on the World's
// decision record: the registry Group's dispatch when the World holds a
// registry, else the native decide over members.
func regDecide(t *testing.T, w *World, owner, selector uint32, members []int) {
	t.Helper()
	if w.savedGroups == nil {
		w.decide(aiGroup{owner: owner, group: selector, members: members})
		return
	}
	g := regGroup(t, w, owner, selector)
	order := g.AI[0x20]
	if order == orderRoam {
		living, _ := w.savedLiving(g)
		w.savedRoam(g, living)
		order = orderSwarm2
	}
	w.savedDecision(g, order, nil)
}

// regWithdrawalPass runs the withdrawal stage alone: on a registry World the
// dispatch's tail over every active Group's members, then the native pass,
// which still takes a native patroller.
func regWithdrawalPass(w *World, obs *withdrawalObs) {
	if w.savedGroups != nil {
		for gi := 0; gi < len(w.savedGroups.Groups); gi++ {
			if g := &w.savedGroups.Groups[gi]; g.AI[0x45] != 0 || w.safeMode {
				w.savedWithdrawalTail(g, obs)
			}
		}
	}
	w.withdrawalPassObserved(obs)
}

// nativeGroupArm drops the World's registry so the native group and
// per-actor arms decide, as in a second-game World that loaded no registry.
// A test of those arms' own structure uses it; a first-game World decides
// from its registry.
func nativeGroupArm(w *World) *World {
	w.savedGroups = nil
	return w
}

// regActorPass runs the per-actor layer alone: on a registry World the
// order-0 Groups' members through the actor executor, as the dispatch does,
// then the native actor pass, which still takes a native patroller.
func regActorPass(w *World) {
	if w.savedGroups != nil {
		for gi := 0; gi < len(w.savedGroups.Groups); gi++ {
			if g := &w.savedGroups.Groups[gi]; g.AI[0x20] == orderNone && (g.AI[0x45] != 0 || w.safeMode) {
				w.walkSavedMembers(g, func(i int) {
					if !w.nativePatrol(i) {
						w.savedActorDispatch(i)
					}
				})
			}
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
