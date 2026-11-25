package game

import (
	"bytes"
	"fmt"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/scriptprofile"
	"againrom/pkg/ui"
)

// secondNodeTextsAt is where a node record's ten 64-byte text fields begin
// (R2-ENGINE-1681).
const secondNodeTextsAt = 0x9c

// secondTextedNodes is the fixture second-game map with one new check and
// one new action whose ten text fields hold "check N" and "action N", each
// followed by a NUL and the residue byte 0x7e, as installed fields carry
// bytes after their terminator (R2-ASSET-364). The check's operation is
// one no claim decodes. It answers the map and each node's record offset.
func secondTextedNodes(t *testing.T) (b []byte, check, action int) {
	t.Helper()
	b = bytes.Clone(secondNewNodes(t).model.Bytes())
	check, action = bytes.Index(b, []byte("Compiler parameter")), bytes.Index(b, []byte("Send event"))
	if check < 0 || action < 0 {
		t.Fatal("the new nodes are not in the map")
	}
	for _, n := range []struct {
		at   int
		name string
	}{{check, "check"}, {action, "action"}} {
		for i := range alm.ScriptParams {
			field := b[n.at+secondNodeTextsAt+alm.ScriptLabelLen*i:][:alm.ScriptLabelLen]
			copy(field, fmt.Sprintf("%s %d\x00\x7e", n.name, i+1))
		}
	}
	op := uint32(1000)
	for mapload.ScriptOperationDecoded(scriptprofile.Second.Ops, true, op) {
		op++
	}
	copy(b[check+alm.ScriptNodeOpcode:], []byte{byte(op), byte(op >> 8), byte(op >> 16), byte(op >> 24)})
	return b, check, action
}

// An edited text field, and the name of a node whose operation is not
// decoded, save into their own bytes, filled to their width; every other
// byte of the file stays. An operation word edit on that node is refused.
func TestSecondMapNodeTextEditSavesIntoItsField(t *testing.T) {
	b, check, action := secondTextedNodes(t)
	d, err := secondTestInspector(t).inspect(b, "second.alm")
	if err != nil {
		t.Fatal(err)
	}
	host := d.Edits.(*mapInspectionEdits)
	if _, err := host.SetFields(ui.KindAction, 0, []ui.InspectionFieldValue{{Key: "text3", Text: true, TextValue: "Новый текст"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := host.SetFields(ui.KindCondition, 0, []ui.InspectionFieldValue{{Key: "name", Text: true, TextValue: "Renamed"}, {Key: "text9", Text: true, TextValue: "last"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := host.SetFields(ui.KindCondition, 0, []ui.InspectionFieldValue{{Key: "value0", Value: 5}}); err == nil {
		t.Fatal("a parameter word of an undecoded node was written")
	}
	want := bytes.Clone(b)
	put := func(at int, text []byte) {
		clear(want[at : at+alm.ScriptLabelLen])
		copy(want[at:], text)
	}
	put(action+secondNodeTextsAt+alm.ScriptLabelLen*3, []byte{0xcd, 0xee, 0xe2, 0xfb, 0xe9, ' ', 0xf2, 0xe5, 0xea, 0xf1, 0xf2})
	put(check, []byte("Renamed"))
	put(check+secondNodeTextsAt+alm.ScriptLabelLen*9, []byte("last"))
	if got := host.model.Bytes(); !bytes.Equal(got, want) {
		for i := range min(len(got), len(want)) {
			if got[i] != want[i] {
				t.Fatalf("byte %#x is %#x, want %#x (lengths %d, %d)", i, got[i], want[i], len(got), len(want))
			}
		}
		t.Fatalf("lengths %d, want %d", len(got), len(want))
	}
}
