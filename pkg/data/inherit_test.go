package data

import (
	"slices"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/reg"
)

// The inheritance guards, over synthetic registries: internal/synth writes the
// .reg byte stream, pkg/formats/reg parses it back, and the loader only ever
// sees a tree it could have got from a real one. Nothing here reads an install.
//
// Every fixture class carries File = 0 against a one-entry [Files], so none of
// them leaves a sprite reference for a later stage to trip over.

func parseReg(t *testing.T, children []synth.RegNode) *reg.Reg {
	t.Helper()
	r, err := reg.Parse(synth.Reg(0x11, children))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return r
}

func regDir(name string, children ...synth.RegNode) synth.RegNode {
	return synth.RegNode{Name: name, Kind: 0x01, Children: children}
}

func regInt(name string, v int32) synth.RegNode {
	return synth.RegNode{Name: name, Kind: 0x02, Int: v}
}

func regStr(name, s string) synth.RegNode {
	return synth.RegNode{Name: name, Kind: 0x00, Str: s}
}

// regInts with no elements is the zero-length int array. It is a different node
// from regStr(name, ""), the editor's "none" marker, and the point of the array
// guard is that the two are the same length and so behave alike.
func regInts(name string, v ...int32) synth.RegNode {
	return synth.RegNode{Name: name, Kind: 0x06, Ints: v}
}

func unitsReg(t *testing.T, count int32, sections ...synth.RegNode) *reg.Reg {
	t.Helper()
	return parseReg(t, append([]synth.RegNode{
		regDir("Global", regInt("UnitCount", count), regInt("FileCount", 1)),
		regDir("Files", regStr("File0", "unit")),
	}, sections...))
}

func objectsReg(t *testing.T, count int32, sections ...synth.RegNode) *reg.Reg {
	t.Helper()
	return parseReg(t, append([]synth.RegNode{
		regDir("Global", regInt("ObjectCount", count), regInt("FileCount", 1)),
		regDir("Files", regStr("File0", "object")),
	}, sections...))
}

func loadUnits(t *testing.T, r *reg.Reg) *UnitClasses {
	t.Helper()
	cs, err := LoadUnitClasses(r)
	if err != nil {
		t.Fatalf("LoadUnitClasses: %v", err)
	}
	return cs
}

func unitByID(t *testing.T, cs *UnitClasses, id int32) *UnitClass {
	t.Helper()
	c, ok := cs.ByID(id)
	if !ok {
		t.Fatalf("ByID(%d) missed", id)
	}
	return c
}

// SC-1 (AC-1). A scalar is inherited iff the child's own section does not
// contain the key, so a written 0 overrides the parent and stays 0. A loader
// that inherits on a zero loaded field returns 4 for both children.
func TestScalarOverriddenByAWrittenZero(t *testing.T) {
	cs := loadUnits(t, unitsReg(t, 3,
		regDir("Unit0", regInt("ID", 1), regInt("File", 0), regInt("AttackDelay", 4)),
		regDir("Unit1", regInt("ID", 2), regInt("File", 0), regInt("Parent", 1),
			regInt("AttackDelay", 0)),
		regDir("Unit2", regInt("ID", 3), regInt("File", 0), regInt("Parent", 1)),
	))

	for _, tc := range []struct {
		id, want int32
		why      string
	}{
		{1, 4, "the parent's own value"},
		{2, 0, "the child writes 0, which overrides the parent's 4"},
		{3, 4, "the child omits the key, so it inherits"},
	} {
		if got := unitByID(t, cs, tc.id).AttackDelay; got != tc.want {
			t.Errorf("ID %d: AttackDelay = %d, want %d — %s", tc.id, got, tc.want, tc.why)
		}
	}
}

// SC-2 (AC-2). An array is inherited iff the child's own read has length 0 —
// absent, the empty-string sentinel, and a zero-length array alike. There is no
// representation of "clear this array" in the format, so an explicitly empty
// value falls through the guard and the parent's array is taken.
func TestArrayInheritsOnLengthAndNothingClearsIt(t *testing.T) {
	seven := []int32{11, 12, 13, 14, 15, 16, 17}
	three := []int32{21, 22, 23}

	cs := loadUnits(t, unitsReg(t, 5,
		regDir("Unit0", regInt("ID", 1), regInt("File", 0),
			regInts("AttackAnimTime", seven...), regInts("AttackAnimFrame", seven...)),
		// The key omitted.
		regDir("Unit1", regInt("ID", 2), regInt("File", 0), regInt("Parent", 1)),
		// The editor's "none" marker: an array key stored as an empty string.
		regDir("Unit2", regInt("ID", 3), regInt("File", 0), regInt("Parent", 1),
			regStr("AttackAnimTime", "")),
		// A zero-length int array.
		regDir("Unit3", regInt("ID", 4), regInt("File", 0), regInt("Parent", 1),
			regInts("AttackAnimTime")),
		// Three elements of its own, its partner written at the same length.
		regDir("Unit4", regInt("ID", 5), regInt("File", 0), regInt("Parent", 1),
			regInts("AttackAnimTime", three...), regInts("AttackAnimFrame", three...)),
	))

	for _, tc := range []struct {
		id   int32
		want []int32
		why  string
	}{
		{2, seven, "the key is absent"},
		{3, seven, "an empty string is length 0, and nothing clears an array"},
		{4, seven, "a zero-length array is length 0 too"},
		{5, three, "its own three elements stand"},
	} {
		if got := unitByID(t, cs, tc.id).AttackAnimTime; !slices.Equal(got, tc.want) {
			t.Errorf("ID %d: AttackAnimTime = %v, want %v — %s", tc.id, got, tc.want, tc.why)
		}
	}

	// An inherited array is copied, not aliased: one parent's array reaches
	// every child that takes it, and Node.Ints is the parser's own memory.
	taken := unitByID(t, cs, 3).AttackAnimTime
	if len(taken) == 0 {
		return // already reported above; nothing to alias
	}
	taken[0] = -1
	if got := unitByID(t, cs, 4).AttackAnimTime[0]; got == -1 {
		t.Error("two children share one inherited array — it must be copied when taken")
	}
	if got := unitByID(t, cs, 1).AttackAnimTime[0]; got == -1 {
		t.Error("a child's inherited array aliases the parent's node")
	}
}

// SC-3 (AC-3). A scalar chains through ancestors because it reads the parent's
// RESOLVED value; an array takes exactly one hop because it reads the parent's
// OWN section. An array resolved from what the parent inherited would chain too,
// and the grandchild would come back with the grandparent's array.
func TestScalarsChainWhileArraysTakeOneHop(t *testing.T) {
	seven := []int32{31, 32, 33, 34, 35, 36, 37}

	cs := loadUnits(t, unitsReg(t, 3,
		regDir("Unit0", regInt("ID", 1), regInt("File", 0), regInt("AttackDelay", 4),
			regInts("MoveAnimTime", seven...), regInts("MoveAnimFrame", seven...)),
		regDir("Unit1", regInt("ID", 2), regInt("File", 0), regInt("Parent", 1)),
		regDir("Unit2", regInt("ID", 3), regInt("File", 0), regInt("Parent", 2)),
	))

	// One hop: the parent takes the grandparent's array and its scalar.
	parent := unitByID(t, cs, 2)
	if got := parent.AttackDelay; got != 4 {
		t.Errorf("parent: AttackDelay = %d, want 4", got)
	}
	if got := parent.MoveAnimTime; !slices.Equal(got, seven) {
		t.Errorf("parent: MoveAnimTime = %v, want %v — one hop reaches it", got, seven)
	}

	// Two hops: the scalar arrives, the array does not.
	child := unitByID(t, cs, 3)
	if got := child.AttackDelay; got != 4 {
		t.Errorf("child: AttackDelay = %d, want 4 — scalars chain through ancestors", got)
	}
	if got := child.MoveAnimTime; got != nil {
		t.Errorf("child: MoveAnimTime = %v, want nil — an array takes exactly one hop, "+
			"and its parent's own section does not set the key", got)
	}
	if got := child.MoveAnimFrame; got != nil {
		t.Errorf("child: MoveAnimFrame = %v, want nil", got)
	}
}

// SC-4 (AC-4). Presence of the Parent key decides, never its value: the objects
// ID domain is 0-based, so Parent = 0 is a real reference to the class whose ID
// is 0. An absent Parent is what means no inheritance.
func TestParentZeroIsARealReference(t *testing.T) {
	r := objectsReg(t, 3,
		regDir("Object0", regInt("ID", 0), regInt("File", 0), regInt("Width", 5)),
		regDir("Object1", regInt("ID", 1), regInt("File", 0), regInt("Parent", 0)),
		regDir("Object2", regInt("ID", 2), regInt("File", 0)),
	)
	cs, err := LoadObjectClasses(r)
	if err != nil {
		t.Fatalf("LoadObjectClasses: %v", err)
	}

	byID := func(id int32) *ObjectClass {
		t.Helper()
		c, ok := cs.ByID(id)
		if !ok {
			t.Fatalf("ByID(%d) missed", id)
		}
		return c
	}

	if got := byID(1).Width; got != 5 {
		t.Errorf("Parent = 0: Width = %d, want 5 — 0 names the class whose ID is 0", got)
	}
	if got := byID(1).Parent; got != 0 {
		t.Errorf("Parent = 0: Parent = %d, want 0 verbatim", got)
	}
	// An absent Parent inherits nothing, so Width is set by no section on this
	// class's chain and reads its absent-everywhere default rather than the
	// parent's 5. -1 is that default, not a second reading of "no parent":
	// nothing here has looked at a Parent value at all (defaults_test.go).
	if got := byID(2).Width; got != -1 {
		t.Errorf("no Parent key: Width = %d, want -1 — an absent Parent inherits nothing", got)
	}
}
