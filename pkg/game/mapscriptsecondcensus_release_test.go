//go:build release

package game

import (
	"bytes"
	"fmt"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/ui"
)

// The census of the second game's triggers through the editor's own loader
// and write path: per root the triggers, each check and action by operation
// with its maps and nodes, the nodes the trigger window names in words, the
// nodes and operand slots it refuses to write, and the trigger fields it
// refuses, then the triggers it copies or opens in the wizard and the nodes
// it deletes. A write of a slot's own value is the probe: the host refuses it
// where the window keeps the slot read-only. It logs the table; it fails
// only when a map does not open.
func TestReleaseSecondMapTriggerCensus(t *testing.T) {
	x := installedInspector(t)
	sources, _ := installedSecondMaps(t, x)
	type opKey struct {
		condition bool
		opcode    uint32
	}
	type opCount struct {
		maps, nodes, named, refused int
		slots                       map[uint32]int // type code -> slots the write refuses
		types                       map[string]int // the stored type vector -> nodes
	}
	ops := map[opKey]*opCount{}
	var maps, triggers, triggerRefused, nameRefused, nodes, named, refused, slotsAll, slotsRefused int
	var copyable, wizardOpen, deletable, inTrigger int
	copyWhy, deleteWhy := map[string]int{}, map[string]int{}
	for _, source := range sources {
		d, err := x.Open(source)
		if err != nil || d.Script == nil {
			t.Fatalf("%s: open %v", source, err)
		}
		host, ok := d.Edits.(*mapInspectionEdits)
		if !ok {
			t.Fatalf("%s: no edit host", source)
		}
		maps++
		sc := d.Script
		triggers += len(sc.Triggers)
		for i, tr := range sc.Triggers {
			if _, err := host.SetFields(ui.KindTrigger, i, []ui.InspectionFieldValue{{Key: "once", Value: int64(tr.Once)}}); err != nil {
				triggerRefused++
			}
			if _, err := host.SetFields(ui.KindTrigger, i, []ui.InspectionFieldValue{{Key: "name", Text: true, TextValue: tr.Name}}); err != nil {
				nameRefused++
			}
			// The editor copies through the seam it offers; the copy is undone.
			if c, ok := ui.Seam[ui.InspectionTriggerCopy](d.Edits); ok {
				if _, err := c.CopyTrigger(i); err == nil {
					copyable++
					host.Undo()
				} else {
					copyWhy[err.Error()]++
				}
			}
			// The wizard opens where its seam is offered and Apply of the rows
			// as read writes the trigger back byte-equal.
			if w, ok := ui.Seam[ui.InspectionTriggerWizard](d.Edits); ok {
				plan := ui.WizardTrigger{Index: i, Name: tr.Name, Once: tr.Once}
				for k := range tr.Left {
					if tr.Left[k] != 0 || tr.Right[k] != 0 {
						plan.Pairs = append(plan.Pairs, ui.WizardPair{Left: ui.WizardNode{ID: tr.Left[k], Keep: true}, Right: ui.WizardNode{ID: tr.Right[k], Keep: true}, Cmp: tr.Cmp[k], Group: tr.GroupOf(k)})
					}
				}
				for _, id := range tr.Acts {
					if id != 0 {
						plan.Actions = append(plan.Actions, ui.WizardNode{ID: id, Keep: true})
					}
				}
				held := host.model.Bytes()
				if _, err := w.WriteTrigger(plan); err == nil && bytes.Equal(held, host.model.Bytes()) {
					wizardOpen++
				} else if err == nil {
					host.Undo()
				}
			}
		}
		seen := map[opKey]bool{}
		for _, family := range []struct {
			kind      ui.RecordKind
			condition bool
			nodes     []ui.InspectionScriptNode
			catalog   []ui.InspectionScriptOperation
		}{{ui.KindCondition, true, sc.Conditions, sc.Catalog.Checks}, {ui.KindAction, false, sc.Actions, sc.Catalog.Actions}} {
			for i, n := range family.nodes {
				k := opKey{family.condition, n.Opcode}
				c := ops[k]
				if c == nil {
					c = &opCount{slots: map[uint32]int{}, types: map[string]int{}}
					ops[k] = c
				}
				if !seen[k] {
					seen[k], c.maps = true, c.maps+1
				}
				c.nodes++
				nodes++
				c.types[fmt.Sprint(n.Types)]++
				if slices.ContainsFunc(family.catalog, func(op ui.InspectionScriptOperation) bool { return op.Opcode == n.Opcode && op.Named }) {
					c.named++
					named++
				}
				hit := false
				for s, typ := range n.Types {
					if typ == 0 {
						continue
					}
					slotsAll++
					key := fmt.Sprintf("value%d", s)
					if _, err := host.SetFields(family.kind, i, []ui.InspectionFieldValue{{Key: key, Value: int64(n.Values[s])}}); err != nil {
						c.slots[typ]++
						slotsRefused++
						hit = true
					}
				}
				if hit {
					c.refused++
					refused++
				}
				// A node no trigger names is deleted; one a trigger names
				// goes with its row in the wizard.
				switch _, err := host.Delete(family.kind, i); {
				case err == nil:
					deletable++
					host.Undo()
				case err.Error() == fmt.Sprintf(x.word("editor.err.noderef"), n.ID):
					inTrigger++
				default:
					deleteWhy[err.Error()]++
				}
			}
		}
	}
	if maps == 0 {
		t.Fatal("no maps")
	}
	keys := slices.SortedFunc(func(yield func(opKey) bool) {
		for k := range ops {
			if !yield(k) {
				return
			}
		}
	}, func(a, b opKey) int {
		if a.condition != b.condition {
			if a.condition {
				return -1
			}
			return 1
		}
		return int(a.opcode) - int(b.opcode)
	})
	for _, k := range keys {
		c := ops[k]
		family := "action"
		if k.condition {
			family = "check"
		}
		var slots, types []string
		for _, typ := range slices.Sorted(func(yield func(uint32) bool) {
			for t := range c.slots {
				if !yield(t) {
					return
				}
			}
		}) {
			slots = append(slots, fmt.Sprintf("type%d:%d", typ, c.slots[typ]))
		}
		for _, v := range slices.Sorted(func(yield func(string) bool) {
			for v := range c.types {
				if !yield(v) {
					return
				}
			}
		}) {
			types = append(types, fmt.Sprintf("%s x%d", v, c.types[v]))
		}
		t.Logf("CENSUS %s %s %d: maps %d, nodes %d, named %d, refused %d, refused slots [%s], vectors %s", installLanguage(x), family, k.opcode, c.maps, c.nodes, c.named, c.refused, strings.Join(slots, " "), strings.Join(types, "; "))
	}
	t.Logf("CENSUS %s total: %d maps, %d triggers (%d refuse a field write, %d refuse a name), %d nodes, %d named, %d refuse a slot write, %d of %d typed slots refused",
		installLanguage(x), maps, triggers, triggerRefused, nameRefused, nodes, named, refused, slotsRefused, slotsAll)
	t.Logf("CENSUS %s editing: %d of %d triggers copy, %d open in the wizard and apply byte-equal; %d nodes delete, %d named by a trigger, %d refused otherwise",
		installLanguage(x), copyable, triggers, wizardOpen, deletable, inTrigger, nodes-deletable-inTrigger)
	for _, r := range []struct {
		what string
		why  map[string]int
	}{{"copy", copyWhy}, {"delete", deleteWhy}} {
		var keys []string
		for why := range r.why {
			keys = append(keys, why)
		}
		slices.Sort(keys)
		for _, why := range keys {
			t.Logf("CENSUS %s %s refused x%d: %s", installLanguage(x), r.what, r.why[why], why)
		}
	}
}
