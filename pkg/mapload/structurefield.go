package mapload

import (
	"againrom/pkg/formats/alm"
	"againrom/pkg/sim"
)

// StructureFieldRef is one compiled script node that reads or writes the
// structure script field `+0x42`, beside the value that field is SEEDED to at
// tick 0 for the structure the node names.
//
// Check opcode 21 reads the field and instant opcode 26 writes it
// (`TRIG-CHECK-053`). The seed itself is Structures' business
// (`ALM-CLS-053`); this type joins the two so a caller can answer "does any
// node on this map reach the seed, and what does it read" without holding
// either rule.
//
// It carries a structure reference as a plain uint32 rather than a
// sim.StructureID, so a reporting tool can print it without reaching the
// simulation tier. The number is the map's own placement index, which is the
// number Structures orders its list by.
//
// Placed separates a reference resolving to a structure this map placed from
// one that does not. HasRef separates a node naming a structure from one whose
// reference the binder could not resolve at all. Value is meaningful only when
// Placed is true.
// Node is an index into the compiled array the node belongs to -- the check
// array when Writes is false, the instant array when it is true -- so a check
// and an instant can carry the same Node and name different nodes. It is the
// same numbering cmd/almtool's script verb prints.
type StructureFieldRef struct {
	Node   int
	Writes bool
	Ref    uint32
	HasRef bool
	Placed bool
	Value  uint16
}

// StructureFieldRefs is every such node on one map, checks first and then
// instants, each in compiled-array order.
//
// It joins placement state to actual script references. The same aggregate
// health count can occur with different trigger dependencies. Building's
// authored-zero override (ALM-128) is included through Structures.
//
// The script is compiled with NO PARTY, on cmd/almtool's own terms. That leaves
// hero-band unit references unresolved and leaves every structure reference
// unaffected, because a structure reference resolves against the map's own
// placements alone.
//
// The COMPARISON a check node's register feeds is not reported. It belongs to
// the trigger that reads the register rather than to the node, and cmd/almtool's
// script verb lists it.
//
// An empty result and a compile failure are separated by the error: a map that
// authors no such node is a fact, and a map whose script did not compile is not.
func StructureFieldRefs(m *alm.Map, t *Table) ([]StructureFieldRef, error) {
	s, _, err := CompileScript(m, ScriptRefs{Structures: ScriptStructures(m)})
	if err != nil {
		return nil, err
	}
	if s == nil {
		return nil, nil
	}
	return structureFieldRefs(s, Structures(m, t)), nil
}

// structureFieldRefs is the selection and the join, with the compile and the
// seed already done. It is separate so a test can install a compiled script and
// a structure list directly, rather than having to synthesize a map carrying a
// type-7 record to reach two loops.
func structureFieldRefs(s *sim.Script, structs []sim.Structure) []StructureFieldRef {
	out := []StructureFieldRef{}
	add := func(node int, writes bool, id uint32, has bool) {
		r := StructureFieldRef{Node: node, Writes: writes, Ref: id, HasRef: has}
		if has && int(id) < len(structs) {
			r.Placed = true
			r.Value = structs[id].Field42
		}
		out = append(out, r)
	}
	for i, c := range s.Checks() {
		if c.Op == sim.ScriptCheckStructField {
			add(i, false, uint32(c.Structure), c.HasStructure)
		}
	}
	for i, in := range s.Instants() {
		if in.Op == sim.ScriptInstantStructField {
			add(i, true, uint32(in.Structure), in.HasStructure)
		}
	}
	return out
}
