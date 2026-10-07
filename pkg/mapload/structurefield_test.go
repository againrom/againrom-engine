package mapload

// The +0x42 READER LISTING (1033 B3, adversarial pass 1).
//
// The per-map summary of the seeded field reports a population and says nothing
// about whether any script node reaches it. A build seeding the wrong value and
// one seeding the right value print the same summary on every shipped map,
// because no shipped trigger holds either way. This listing is what separates
// them, so it is tested on what it selects and on the value it joins.
//
// It is a white-box test: the exported entry point needs a map carrying a
// decoded type-7 record, and the selection and the join do not.

import (
	"testing"

	"againrom/pkg/sim"
)

// sfScript compiles one script directly, with the arms and references spelt out
// rather than assembled from map bytes.
func sfScript(t *testing.T, checks []sim.ScriptCheck, instants []sim.ScriptInstant) *sim.Script {
	t.Helper()
	s, err := sim.NewScript(checks, instants, nil)
	if err != nil {
		t.Fatalf("NewScript: %v", err)
	}
	return s
}

// TestTheFieldListingSelectsTheTwoArmsThatTouchTheField is the selection. Only
// check opcode 21 and instant opcode 26 read or write +0x42, and a listing that
// took every node with a structure reference, or every node of the script,
// would report arms that never touch the field.
func TestTheFieldListingSelectsTheTwoArmsThatTouchTheField(t *testing.T) {
	t.Parallel()

	s := sfScript(t,
		[]sim.ScriptCheck{
			{Op: sim.ScriptCheckConstant, Register: 0, Args: [10]int32{7}},
			{Op: sim.ScriptCheckStructField, Register: 1, Structure: 2, HasStructure: true},
			{Op: sim.ScriptCheckAlive, Register: 2, Unit: 1, HasUnit: true},
			{Op: sim.ScriptCheckStructField, Register: 3, Structure: 0, HasStructure: true},
		},
		[]sim.ScriptInstant{
			{Op: sim.ScriptInstantMessage},
			{Op: sim.ScriptInstantStructField, Structure: 1, HasStructure: true},
		},
	)
	structs := []sim.Structure{
		{ID: 0, Field42: 300},
		{ID: 1, Field42: 30000},
		{ID: 2, Field42: 7},
	}

	got := structureFieldRefs(s, structs)
	want := []StructureFieldRef{
		{Node: 1, Writes: false, Ref: 2, HasRef: true, Placed: true, Value: 7},
		{Node: 3, Writes: false, Ref: 0, HasRef: true, Placed: true, Value: 300},
		{Node: 1, Writes: true, Ref: 1, HasRef: true, Placed: true, Value: 30000},
	}
	if len(got) != len(want) {
		t.Fatalf("listed %d node(s), want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("entry %d listed as %+v, want %+v", i, got[i], want[i])
		}
	}
}

// TestTheFieldListingReportsTheValueTheArmWouldRead is the join. Each node
// carries its own structure's seeded value and not another structure's, not a
// constant, and not its own node index.
//
// The three values are distinct, none is a Go zero value, and none equals a
// node index or a structure id in the fixture, so a listing joining on the
// wrong key fails here rather than agreeing by coincidence.
func TestTheFieldListingReportsTheValueTheArmWouldRead(t *testing.T) {
	t.Parallel()

	s := sfScript(t, []sim.ScriptCheck{
		{Op: sim.ScriptCheckStructField, Register: 0, Structure: 2, HasStructure: true},
		{Op: sim.ScriptCheckStructField, Register: 1, Structure: 1, HasStructure: true},
		{Op: sim.ScriptCheckStructField, Register: 2, Structure: 0, HasStructure: true},
	}, nil)
	structs := []sim.Structure{{ID: 0, Field42: 511}, {ID: 1, Field42: 4097}, {ID: 2, Field42: 60000}}

	got := structureFieldRefs(s, structs)
	for i, want := range []uint16{60000, 4097, 511} {
		if got[i].Value != want {
			t.Errorf("node %d reports value %d, want structure %d's own %d",
				got[i].Node, got[i].Value, got[i].Ref, want)
		}
	}
}

// TestTheFieldListingSeparatesAnUnresolvedReferenceFromAZeroValue is the
// honesty of the line. A node whose reference the binder could not resolve, and
// one naming a structure outside the placed list, each report zero for the
// value, and zero is also a legitimate seeded value. The two flags are what
// separate the three cases; without them the listing would print `= 0` for all
// three and a reader could not tell an unresolved reference from a structure
// the table gave nothing for.
func TestTheFieldListingSeparatesAnUnresolvedReferenceFromAZeroValue(t *testing.T) {
	t.Parallel()

	s := sfScript(t, []sim.ScriptCheck{
		{Op: sim.ScriptCheckStructField, Register: 0},                                   // no reference at all
		{Op: sim.ScriptCheckStructField, Register: 1, Structure: 9, HasStructure: true}, // past the list
		{Op: sim.ScriptCheckStructField, Register: 2, Structure: 0, HasStructure: true}, // placed, seeded 0
	}, nil)
	got := structureFieldRefs(s, []sim.Structure{{ID: 0, Field42: 0}})

	if len(got) != 3 {
		t.Fatalf("listed %d node(s), want 3: %+v", len(got), got)
	}
	if got[0].HasRef || got[0].Placed {
		t.Errorf("a node naming no structure listed as %+v", got[0])
	}
	if !got[1].HasRef || got[1].Placed {
		t.Errorf("a reference past the placed list listed as %+v", got[1])
	}
	if !got[2].HasRef || !got[2].Placed || got[2].Value != 0 {
		t.Errorf("a placed structure seeded to zero listed as %+v", got[2])
	}
}

// TestTheFieldListingIsEmptyWhereNoNodeTouchesTheField is the case every
// shipped campaign map but four is in. An empty list is a fact about the map;
// the exported entry point separates it from a compile failure through the
// error, and this covers the inner half.
func TestTheFieldListingIsEmptyWhereNoNodeTouchesTheField(t *testing.T) {
	t.Parallel()

	s := sfScript(t,
		[]sim.ScriptCheck{{Op: sim.ScriptCheckConstant, Register: 0, Args: [10]int32{1}}},
		[]sim.ScriptInstant{{Op: sim.ScriptInstantWin}},
	)
	if got := structureFieldRefs(s, []sim.Structure{{ID: 0, Field42: 5}}); len(got) != 0 {
		t.Errorf("listed %+v for a script touching the field nowhere", got)
	}
}
