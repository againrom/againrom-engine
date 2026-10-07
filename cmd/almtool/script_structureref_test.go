package main

// The `script` verb's rendering of a Target_Structure reference, both halves
// (1033 follow-up W-1).
//
// No shipped map reaches the arm this file's second test proves: all 16
// authored check-21 nodes on both preserved roots resolve. The fixture is
// therefore synthetic, built from the documented type-4 and type-7 byte
// layouts (pkg/formats/alm's alm.go and script.go) rather than from a game
// file, on the same terms as roundtrip_test.go's minimalALM and
// roster_test.go's rebuildALM, which this file reuses.

import (
	"encoding/binary"
	"strings"
	"testing"
)

// structRefALM builds a map carrying one placed structure (type-4 record,
// Field12 = structRefPlaced) and one authored trigger gating on two check
// opcode 21 (ScriptCheckStructField) conditions: one whose Target_Structure
// names the placed structure and resolves, the other naming a value nothing
// placed. Neither reads a unit parameter at all, which is the point — the
// defect this fixture proves rendered a structure-only node in unit
// vocabulary.
const (
	structRefPlaced   = 42
	structRefDangling = 999
)

func structRefALM(t *testing.T) []byte {
	t.Helper()

	// type-4: one 20-byte object record, Kind 0 (no extension), Field12 the
	// structure's own +0x12 word that ScriptStructures keys by.
	obj := make([]byte, 20)
	binary.LittleEndian.PutUint16(obj[0x12:0x14], structRefPlaced)

	// type-7: entryCount(0 actions) + condCount(2) + two 796-byte condition
	// nodes + trigCount(1) + one 184-byte trigger.
	node := func(id uint32, structVal uint32) []byte {
		rec := make([]byte, 796)
		binary.LittleEndian.PutUint32(rec[0x040:0x044], 21) // opcode: ScriptCheckStructField
		binary.LittleEndian.PutUint32(rec[0x044:0x048], id)
		binary.LittleEndian.PutUint32(rec[0x04c:0x050], structVal) // Value[0]
		binary.LittleEndian.PutUint32(rec[0x074:0x078], 9)         // Type[0] = typeStructure
		return rec
	}
	trigger := make([]byte, 184)
	binary.LittleEndian.PutUint32(trigger[0x80:0x84], 1) // Left[0] = condition id 1
	binary.LittleEndian.PutUint32(trigger[0x84:0x88], 2) // Right[0] = condition id 2
	binary.LittleEndian.PutUint32(trigger[0xa8:0xac], 0) // Cmp[0] = ==

	var p7 []byte
	u32 := func(v uint32) {
		var w [4]byte
		binary.LittleEndian.PutUint32(w[:], v)
		p7 = append(p7, w[:]...)
	}
	u32(0) // entryCount: no actions
	u32(2) // condition count
	p7 = append(p7, node(1, structRefPlaced)...)
	p7 = append(p7, node(2, structRefDangling)...)
	u32(1) // trigger count
	p7 = append(p7, trigger...)

	return rebuildALM(t, map[int][]byte{4: obj, 7: p7}, map[int]uint32{0x20: 1})
}

// TestScriptVerbNamesAResolvedStructureReferenceAsAStructure is the resolved
// half. Before this fix, refNote (then unitNote) read c.HasUnit alone, so a
// check-21 node with no unit parameter always printed "names no unit"
// whatever its own structure reference did.
func TestScriptVerbNamesAResolvedStructureReferenceAsAStructure(t *testing.T) {
	path := writeALM(t, structRefALM(t))
	out, err := captureStdout(t, func() error { return cmdScript(path) })
	if err != nil {
		t.Fatalf("cmdScript: %v", err)
	}
	if strings.Contains(out, "names no unit") {
		t.Errorf("output still reads the resolved structure reference as an absent unit:\n%s", out)
	}
	if !strings.Contains(out, "structure resolved") {
		t.Errorf("output does not say the structure reference resolved:\n%s", out)
	}
}

// TestScriptVerbNamesAnUnresolvedStructureReferenceAsAStructure is the
// unresolved half. Before this fix, ScriptUnresolved carried no reference
// kind, so bindParams' typeStructure arm and its typeUnit arm fed one list
// that both listings rendered as a Target_Unit miss: the binder section named
// "unit 999" and the per-check listing gave a hero-band note for a value that
// was never in unit space at all.
func TestScriptVerbNamesAnUnresolvedStructureReferenceAsAStructure(t *testing.T) {
	path := writeALM(t, structRefALM(t))
	out, err := captureStdout(t, func() error { return cmdScript(path) })
	if err != nil {
		t.Fatalf("cmdScript: %v", err)
	}

	wantBinder := "condition node id=2 opcode=21 names structure 999 (a structure this map does not place)"
	if !strings.Contains(out, wantBinder) {
		t.Errorf("binder section does not read %q:\n%s", wantBinder, out)
	}
	if strings.Contains(out, "names unit 999") {
		t.Errorf("binder section still names the structure miss as a unit:\n%s", out)
	}

	wantCheck := "structure 999 is not in this map's structure table"
	if !strings.Contains(out, wantCheck) {
		t.Errorf("per-check listing does not read %q:\n%s", wantCheck, out)
	}
	for _, unitWord := range []string{"map unit 999", "hero ordinal", "name-table id"} {
		if strings.Contains(out, unitWord) {
			t.Errorf("per-check listing still uses unit vocabulary (%q) for a structure miss:\n%s", unitWord, out)
		}
	}
}
