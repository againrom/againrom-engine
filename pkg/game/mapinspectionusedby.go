package game

import "againrom/pkg/ui"

func inspectScriptUses(d *ui.InspectionDocument) {
	m := d.Script
	if m == nil {
		return
	}
	for _, family := range []struct {
		nodes     []ui.InspectionScriptNode
		condition bool
	}{{m.Conditions, true}, {m.Actions, false}} {
		for _, n := range family.nodes {
			var triggers []int
			if n.ID != 0 {
				for _, t := range m.Triggers {
					ids := t.Acts[:]
					if family.condition {
						ids = append(append([]uint32{}, t.Left[:]...), t.Right[:]...)
					}
					for _, id := range ids {
						if id == n.ID {
							triggers = append(triggers, t.Record)
							break
						}
					}
				}
			}
			for slot, value := range n.Values {
				kind := scriptParameterKind(family.condition, n.Opcode, n.Types, slot)
				if target := m.OpenTarget(kind, value); target != nil {
					target.Uses = append(target.Uses, ui.InspectionScriptUse{Node: n.Record, Slot: slot, Triggers: triggers})
				}
			}
		}
	}
}
