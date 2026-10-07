package sim

// AI-FORM-037 and MOVE-GATE-035 define the formation mode byte. Commands
// remap 0/1/2 to 0/2/1; instant7 writes its raw parameter narrowed to one byte.
// Saved Players bind commands by signed +04, triggers by unique +08, and
// Group movement by the explicit owner identity (AI-FORMCMD-315,
// AI-FORMTRIGGER-316, AI-FORMOWNER-314). Old native worlds without that
// carrier retain the historical roster-slot array below.

// formationDefault is the mode a player has until something writes one: 2, the
// value the decoded player constructor writes into its settings block and the
// only nonzero byte that block is constructed with.
//
// It is not a policy choice. Under mode 2 the group-move distribution is gated
// on the spread test alone, which is exactly what this package did for every
// group before this story, so a world nothing has written a mode into behaves
// as it always did.
const formationDefault uint8 = 2

// resetFormations puts every roster slot at the default. It is the
// constructor's own act and the one place the default is applied (D-2).
func (w *World) resetFormations() {
	for i := range w.formations {
		w.formations[i] = formationDefault
	}
}

// setFormationMode is the raw instant7 write. Missing or colliding saved +08
// targets write nothing; the original temporary-map collision law is Unknown.
// Only the absent-carrier compatibility arm interprets player as a roster slot.
func (w *World) setFormationMode(player uint32, mode int32) {
	if w.hasSavedFormations() {
		if p := w.triggerFormation(player); p != nil {
			p.Mode = uint8(mode)
		}
		return
	}
	if player >= relationSlots {
		return
	}
	w.formations[player] = uint8(mode)
}

// FormationMode reads the legacy roster-slot carrier. Exact saved Group owners
// use savedGroupFormation; the client uses CommandFormationMode.
func (w *World) FormationMode(player uint32) uint8 {
	if player >= relationSlots {
		return formationDefault
	}
	return w.formations[player]
}

// groupInFormation is `MOVE-GATE-035`'s three behaviours, for the group
// whose members are given and whose owner is owner.
//
// Mode 0 is NEVER in formation and the spread test is not run at all. Mode 2 is
// in formation exactly when the spread test passes. Any OTHER nonzero mode is in
// formation unconditionally, again without running the spread test — 254 of the
// 256 values reach that arm, which is why it is written as the fall-through and
// not as an enumeration.
//
// It is one function rather than three lines inside issueGroupDestination so
// that the mode's three arms sit together and are read together. The spread test
// itself is unchanged and is still the only thing that decides a mode-2 group.
func (w *World) groupInFormation(owner uint32, members []int, cx, cy int32) bool {
	return w.modeInFormation(w.FormationMode(owner), members, cx, cy)
}

func (w *World) modeInFormation(mode uint8, members []int, cx, cy int32) bool {
	switch mode {
	case 0:
		return false
	case formationDefault:
		return inFormation(w.entities, members, cx, cy)
	}
	return true
}
