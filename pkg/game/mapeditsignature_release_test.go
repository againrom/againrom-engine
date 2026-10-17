package game

import (
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
)

// TestReleaseMapEditorSignatureCensus compares, over every script node of
// every installed map, the stored slot types with the signature row the
// trigger editor writes. The known exceptions are counted, not skipped
// silently: a TRUE/FALSE constant condition stores Const where the row
// stores Int, and the node labelled PalNearChief stores another schema.
func TestReleaseMapEditorSignatureCensus(t *testing.T) {
	x := installedInspector(t)
	keys, maps := installedMapBytes(t, x)
	var nodes, checked, constants, anomalies, noRow int
	for i, b := range maps {
		m, err := alm.Open(b)
		if err != nil {
			t.Fatal(keys[i], err)
		}
		sc, err := m.Script()
		if err != nil {
			continue
		}
		for _, family := range []struct {
			condition bool
			nodes     []alm.ScriptNode
		}{{true, sc.Conditions}, {false, sc.Actions}} {
			for n, node := range family.nodes {
				nodes++
				command := uint32(0)
				if !family.condition && node.Opcode == mapload.ScriptGroupCommand {
					command = node.Value[0]
				}
				sig, ok := mapload.ScriptSignature(family.condition, node.Opcode, command)
				if !ok {
					noRow++
					continue
				}
				var want [10]uint32
				for s, slot := range sig {
					want[s] = slot.Type
				}
				if want == node.Type {
					checked++
					continue
				}
				switch {
				case family.condition && node.Opcode == mapload.ScriptConstant && node.Type[0] == mapload.ScriptConst:
					constants++
				case node.Label == "PalNearChief":
					anomalies++
				default:
					t.Errorf("%s %v node %d %q opcode %d command %d: stored %v, signature %v", keys[i], family.condition, n, node.Label, node.Opcode, command, node.Type, want)
				}
			}
		}
	}
	t.Logf("%d nodes: %d equal their row, %d TRUE/FALSE constants, %d PalNearChief, %d with no row", nodes, checked, constants, anomalies, noRow)
	if checked == 0 {
		t.Fatal("no installed node has a signature row")
	}
}
