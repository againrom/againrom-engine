package ui

import (
	"slices"
	"testing"

	"againrom/pkg/scriptprofile"
)

// secondCatalog is the second game's catalog as the host builds it from the
// operation rows: each offered row, its group commands and its slots.
func secondCatalog() InspectionScriptCatalog {
	family := func(condition bool) []InspectionScriptOperation {
		var out []InspectionScriptOperation
		for _, op := range scriptprofile.OpRows(scriptprofile.OpsSecond, condition) {
			if op.Offered {
				out = append(out, InspectionScriptOperation{Opcode: op.ID, Named: op.Decoded})
			}
		}
		return out
	}
	group, _ := scriptprofile.OpRow(scriptprofile.OpsSecond, false, scriptGroupCommand)
	return InspectionScriptCatalog{Checks: family(true), Actions: family(false), GroupCommands: group.Commands,
		Signature: func(condition bool, opcode, command uint32) ([10]InspectionScriptSlot, bool) {
			var out [10]InspectionScriptSlot
			op, ok := scriptprofile.OpRow(scriptprofile.OpsSecond, condition, opcode)
			if !ok {
				return out, false
			}
			slots, ok := op.Slots(command)
			for s, slot := range slots {
				out[s] = InspectionScriptSlot{Type: slot.Type, Role: slot.Role}
			}
			return out, ok
		}}
}

// On a second-game map the trigger wizard offers a ready row for each
// operation row whose slots and effect a claim reads: check 2 with a box,
// check 21 with a structure, action 6 under every group command, actions 21
// and 24; action 25, whose terrain effect is unread, and the compiler
// parameter are not offered. A stored action the rows read shows as its row,
// a new row is added, not refused, and Apply writes it as the second game's
// operation with its slots.
func TestSecondGameTriggerWizardOffersTheOperationRows(t *testing.T) {
	e, host := wizardChoiceEditor(t)
	e.Layout(1920, 1080)
	e.doc.Script.Limits = scriptprofile.Second
	e.doc.Script.Catalog = secondCatalog()
	v, ok := e.scriptPopup()
	if !ok || v.trigger == nil {
		t.Fatal("no trigger popup")
	}
	e.loadWizard(v)
	items := func(condition bool) map[uint32]wizItem {
		out := map[uint32]wizItem{}
		for _, it := range e.wizItems(condition) {
			if !it.variant {
				out[it.opcode] = it
			}
		}
		return out
	}
	checks, actions := items(true), items(false)
	for _, id := range []uint32{2, 21} {
		if _, ok := checks[id]; !ok {
			t.Errorf("no ready row for check %d", id)
		}
	}
	for _, id := range []uint32{6, 21, 24, 25} {
		if _, ok := actions[id]; !ok {
			t.Errorf("no ready row for action %d", id)
		}
	}
	for _, id := range []uint32{0x10002} {
		if _, ok := actions[id]; ok {
			t.Errorf("action %d has a ready row", id)
		}
	}
	if p := checks[2].params; len(p) < 2 || p[1].kind != wpBox || checks[21].params[0].kind != wpStructure {
		t.Errorf("check 2 parameters %+v, check 21 %+v", p, checks[21].params)
	}
	for _, c := range e.doc.Script.Catalog.GroupCommands {
		if e.commandItem(false, slices.IndexFunc(e.wizItems(false), func(it wizItem) bool { return it.opcode == scriptGroupCommand }), c) < 0 {
			t.Errorf("group command %d has no ready row", c)
		}
	}
	w := e.obj.wiz
	if r := w.rows; len(r) != 4 || r[2].item < 0 || e.wizItems(false)[r[2].item].labelWord != "editor.second.action.4" || r[3].item != -1 {
		t.Fatalf("the stored rows read %+v", r)
	}
	e.wizardRowButton(actAddAct, 0, false)
	if e.obj.note != "" || len(w.rows) != 5 {
		t.Fatalf("a new action row: note %q, %d rows", e.obj.note, len(w.rows))
	}
	cast := slices.IndexFunc(e.wizItems(false), func(it wizItem) bool { return it.opcode == 24 })
	wizardChoose(t, e, "a2.item", int64(cast))
	e.obj.open = -1
	e.applyWizard(v)
	if len(host.plans) != 1 {
		t.Fatalf("%d writes, note %q", len(host.plans), e.obj.note)
	}
	p := host.plans[0]
	if len(p.Actions) != 3 || !p.Actions[0].Keep || !p.Actions[1].Keep {
		t.Fatalf("the write is %+v", p)
	}
	if n := p.Actions[2]; n.Keep || n.Opcode != 24 || n.Types != [10]uint32{5, 6, 4, 1, 1} || n.Values != [10]uint32{32, 32, 10001, 1, 1} {
		t.Fatalf("the new action is written as %+v", n)
	}
}
