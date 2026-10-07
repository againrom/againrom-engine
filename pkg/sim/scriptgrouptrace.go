package sim

// ScriptGroupState is the claim-facing projection of one runtime group record:
// the identity a script names, its current order and the cell carried by the
// three command forms that own one.
//
// It is observer data only. The private record remains the sole state; this
// value is a copy returned to a developer witness and enters neither the world,
// its byte form nor its digest.
type ScriptGroupState struct {
	Owner                  uint32
	Group                  uint32
	Order                  uint8
	CommandedX, CommandedY int32
}

// ObserveScriptGroups returns every runtime group whose raw group id is group,
// in canonical group order. A raw id may occur once per owner, so the result is
// a slice rather than an assumed singleton.
func ObserveScriptGroups(w *World, group uint32) []ScriptGroupState {
	if w == nil {
		return nil
	}
	var out []ScriptGroupState
	if w.savedGroups != nil {
		for _, g := range w.savedGroups.Groups {
			if g.Selector != group {
				continue
			}
			owner := g.Owner.Owner
			if p, ok := w.savedPlayerByID(g.OwnerID); ok {
				owner = p.Slot
			}
			out = append(out, ScriptGroupState{Owner: owner, Group: g.Selector, Order: g.AI[0x20],
				CommandedX: int32(g.AI[10]), CommandedY: int32(g.AI[11])})
		}
		return out
	}
	for _, i := range w.groupsNamed(group) {
		g := w.groups[i]
		out = append(out, ScriptGroupState{
			Owner: g.owner, Group: g.group, Order: uint8(g.order),
			CommandedX: g.commandedX, CommandedY: g.commandedY,
		})
	}
	return out
}
