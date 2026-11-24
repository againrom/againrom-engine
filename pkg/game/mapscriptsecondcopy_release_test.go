//go:build release

package game

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/ui"
)

// secondTally is the type-0 word +0x28 of a second-game map whose type-0
// record comes first, and the sum of its three type-7 counts (R2-ASSET-255).
func secondTally(t *testing.T, data []byte) (word, sum uint32) {
	t.Helper()
	sc := secondScriptOf(t, data)
	return binary.LittleEndian.Uint32(data[2*alm.RecordHeaderSize+0x28:]), uint32(len(sc.Actions) + len(sc.Conditions) + len(sc.Triggers))
}

func savedSecond(t *testing.T, host *mapInspectionEdits, name string) []byte {
	t.Helper()
	target, err := host.SaveAs(filepath.Join(t.TempDir(), name))
	if err != nil {
		t.Fatal(err)
	}
	saved, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	return saved
}

// A trigger copy carries its nodes' records verbatim under new ids, the
// wizard writes a trigger's kept rows and drops the node a removed row
// named, and both keep the type-0 word +0x28 at the sum of the type-7
// counts. A new trigger or node is refused in words.
func TestReleaseSecondMapTriggerCopyAndWizardKeepTheScriptTally(t *testing.T) {
	x := installedInspector(t)
	sources, raws := installedSecondMaps(t, x)
	refusal := x.word("editor.second.newrecord")
	copied, dropped, refused := 0, 0, 0
	for i, source := range sources {
		if copied >= 3 && dropped >= 3 {
			break
		}
		if w, sum := secondTally(t, raws[i]); w != sum {
			t.Fatalf("%s: installed word +0x28 %d, counts %d", source, w, sum)
		}
		sc := secondScriptOf(t, raws[i])
		d, err := x.Open(source)
		if err != nil {
			t.Fatal(err)
		}
		host := d.Edits.(*mapInspectionEdits)
		if _, err := host.WriteTrigger(ui.WizardTrigger{Index: -1, Name: "New", Once: 1}); err == nil || err.Error() != refusal {
			t.Fatalf("%s: a new trigger is not refused in words: %v", source, err)
		}
		if len(sc.Triggers) > 0 {
			tr := sc.Triggers[0]
			plan := ui.WizardTrigger{Index: 0, Name: tr.Name, Once: tr.Once, Actions: []ui.WizardNode{{Opcode: 2, Types: [10]uint32{1}}}}
			if _, err := host.WriteTrigger(plan); err == nil || !strings.Contains(err.Error(), refusal) {
				t.Fatalf("%s: a new node is not refused in words: %v", source, err)
			}
			refused++
		}
		for ti := range sc.Triggers {
			if copied >= 3 {
				break
			}
			before, err := mapedit.New(host.model.Bytes())
			if err != nil {
				t.Fatal(err)
			}
			if _, err := host.CopyTrigger(ti); err != nil {
				continue
			}
			saved := savedSecond(t, host, "copied.alm")
			if w, sum := secondTally(t, saved); w != sum || sum <= uint32(len(sc.Actions)+len(sc.Conditions)+len(sc.Triggers)) {
				t.Fatalf("%s: after a copy the word +0x28 is %d, the counts %d", source, w, sum)
			}
			after, err := mapedit.New(saved)
			if err != nil {
				t.Fatal(err)
			}
			for _, k := range []mapedit.Kind{mapedit.KindAction, mapedit.KindCondition} {
				for n := before.Count(k); n < after.Count(k); n++ {
					rec, _ := after.Record(k, n)
					found := false
					for j := 0; j < before.Count(k) && !found; j++ {
						src, _ := before.Record(k, j)
						src = slices.Clone(src)
						copy(src[alm.ScriptNodeID:alm.ScriptNodeID+4], rec[alm.ScriptNodeID:alm.ScriptNodeID+4])
						found = bytes.Equal(src, rec)
					}
					if !found {
						t.Fatalf("%s: copied node %d of kind %v is no source record under a new id", source, n, k)
					}
				}
			}
			host.Undo()
			copied++
			break
		}
		// The last action of a trigger that no other slot names goes with
		// its row.
		refs := map[uint32]int{}
		for _, tr := range sc.Triggers {
			for _, id := range tr.Acts {
				refs[id]++
			}
		}
		for ti, tr := range sc.Triggers {
			acts := slices.DeleteFunc(slices.Clone(tr.Acts), func(id uint32) bool { return id == 0 })
			if dropped >= 3 || len(acts) < 2 || refs[acts[len(acts)-1]] != 1 {
				continue
			}
			plan := ui.WizardTrigger{Index: ti, Name: tr.Name, Once: tr.Once}
			for k := range tr.Left {
				if tr.Left[k] != 0 || tr.Right[k] != 0 {
					plan.Pairs = append(plan.Pairs, ui.WizardPair{Left: ui.WizardNode{ID: tr.Left[k], Keep: true}, Right: ui.WizardNode{ID: tr.Right[k], Keep: true}, Cmp: tr.Cmp[k]})
				}
			}
			for _, id := range acts[:len(acts)-1] {
				plan.Actions = append(plan.Actions, ui.WizardNode{ID: id, Keep: true})
			}
			if _, err := host.WriteTrigger(plan); err != nil {
				t.Fatalf("%s: trigger %d kept rows: %v", source, ti, err)
			}
			saved := savedSecond(t, host, "dropped.alm")
			got := secondScriptOf(t, saved)
			if w, sum := secondTally(t, saved); w != sum || len(got.Actions) != len(sc.Actions)-1 || slices.Contains(got.Triggers[ti].Acts, acts[len(acts)-1]) {
				t.Fatalf("%s: trigger %d: word +0x28 %d, counts %d, %d actions of %d", source, ti, w, sum, len(got.Actions), len(sc.Actions))
			}
			if !slices.EqualFunc(got.Conditions, sc.Conditions, func(a, b alm.ScriptNode) bool { return a.ID == b.ID && a.Label == b.Label }) {
				t.Fatalf("%s: the checks changed", source)
			}
			dropped++
			break
		}
	}
	if copied == 0 || dropped == 0 || refused == 0 {
		t.Fatalf("copied %d, dropped %d, refused %d", copied, dropped, refused)
	}
	t.Logf("PASS: %d copies, %d removed rows, %d new-node refusals (%s)", copied, dropped, refused, installLanguage(x))
}
