package reg_test

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"slices"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/reg"
)

// ---------------------------------------------------------------------------
// Hand-stated expectations
// ---------------------------------------------------------------------------

// tree is one node of an expected tree, written out by hand. Only the value
// field typ selects is read.
type tree struct {
	name string
	kind uint32
	dir  bool
	typ  reg.ValueType
	str  string
	i32  int32
	f64  float64
	ints []int32
	kids []tree
}

func checkChildren(t *testing.T, path string, got []*reg.Node, want []tree) {
	t.Helper()
	if len(got) != len(want) {
		names := make([]string, len(got))
		for i, n := range got {
			names[i] = n.Name
		}
		t.Fatalf("%s has %d children %q, want %d", path, len(got), names, len(want))
	}
	for i := range want {
		checkNode(t, path+"/"+want[i].name, got[i], want[i])
	}
}

func checkNode(t *testing.T, path string, got *reg.Node, want tree) {
	t.Helper()
	if got.Name != want.name {
		t.Errorf("%s: name = %q, want %q", path, got.Name, want.name)
	}
	if got.Kind != want.kind {
		t.Errorf("%s: kind = %#x, want %#x", path, got.Kind, want.kind)
	}
	if got.Dir != want.dir {
		t.Errorf("%s: Dir = %v, want %v", path, got.Dir, want.dir)
	}
	if want.dir {
		// Dir and Children != nil always agree, so a walker needs one test.
		if got.Children == nil {
			t.Errorf("%s: directory has nil Children", path)
		}
		checkChildren(t, path, got.Children, want.kids)
		return
	}
	if got.Type != want.typ {
		t.Errorf("%s: type = %d, want %d", path, got.Type, want.typ)
		return
	}
	if got.Children != nil {
		t.Errorf("%s: value node carries Children %v, want nil", path, got.Children)
	}
	switch want.typ {
	case reg.TypeString:
		if got.Str != want.str {
			t.Errorf("%s: Str = %q, want %q", path, got.Str, want.str)
		}
	case reg.TypeInt:
		if got.Int != want.i32 {
			t.Errorf("%s: Int = %d, want %d", path, got.Int, want.i32)
		}
	case reg.TypeFloat:
		if math.Float64bits(got.Float) != math.Float64bits(want.f64) {
			t.Errorf("%s: Float = %v (%#016x), want %v (%#016x)", path,
				got.Float, math.Float64bits(got.Float), want.f64, math.Float64bits(want.f64))
		}
	case reg.TypeIntArray:
		if !slices.Equal(got.Ints, want.ints) {
			t.Errorf("%s: Ints = %v, want %v", path, got.Ints, want.ints)
		}
		if got.Ints == nil {
			t.Errorf("%s: Ints is nil, want empty-but-non-nil", path)
		}
	}
}

// ---------------------------------------------------------------------------
// SC-1 — the whole tree, the two heapless fixtures, and the hand-laid witness
// ---------------------------------------------------------------------------

// fiveKinds is a registry using all five implemented node kinds — string,
// directory, int32, float64, int32 array — with a directory nested inside a
// directory and one directory carrying kind 17 (a directory whose child list is
// name-sorted).
func fiveKinds() []synth.RegNode {
	return []synth.RegNode{
		{Name: "Global", Kind: 0x11, Children: []synth.RegNode{
			{Name: "Title", Kind: 0x00, Str: "rage"},
			{Name: "UnitCount", Kind: 0x02, Int: 34},
			{Name: "Inner", Kind: 0x01, Children: []synth.RegNode{
				{Name: "Depth", Kind: 0x02, Int: -1},
			}},
		}},
		{Name: "Unit0", Kind: 0x01, Children: []synth.RegNode{
			{Name: "Scale", Kind: 0x04, Float: 0.1},
			{Name: "Frames", Kind: 0x06, Ints: []int32{1, -2, 3}},
		}},
	}
}

// fiveKindsTree is fiveKinds' expected parse, stated by hand.
func fiveKindsTree() []tree {
	return []tree{
		{name: "Global", kind: 0x11, dir: true, kids: []tree{
			{name: "Title", kind: 0x00, typ: reg.TypeString, str: "rage"},
			{name: "UnitCount", kind: 0x02, typ: reg.TypeInt, i32: 34},
			{name: "Inner", kind: 0x01, dir: true, kids: []tree{
				{name: "Depth", kind: 0x02, typ: reg.TypeInt, i32: -1},
			}},
		}},
		{name: "Unit0", kind: 0x01, dir: true, kids: []tree{
			{name: "Scale", kind: 0x04, typ: reg.TypeFloat, f64: 0.1},
			{name: "Frames", kind: 0x06, typ: reg.TypeIntArray, ints: []int32{1, -2, 3}},
		}},
	}
}

func TestParseTree(t *testing.T) {
	t.Run("all five implemented kinds, nested", func(t *testing.T) {
		r := mustParse(t, synth.Reg(17, fiveKinds()))

		if r.NodeCount != 8 {
			t.Errorf("NodeCount = %d, want 8 (2 root children + 3 + 2 + 1)", r.NodeCount)
		}
		if r.Root == nil {
			t.Fatal("Root is nil")
		}
		if r.Root.Name != "" {
			t.Errorf("Root.Name = %q, want \"\": the root has no on-disk node and no name", r.Root.Name)
		}
		if !r.Root.Dir {
			t.Error("Root.Dir = false, want true by construction")
		}
		if r.Root.Kind != 17 {
			t.Errorf("Root.Kind = %d, want the header's rootFlags 17 verbatim", r.Root.Kind)
		}
		if r.Root.Type != 0 {
			t.Errorf("Root.Type = %d, want 0: it is never derived from rootFlags", r.Root.Type)
		}
		checkChildren(t, "", r.Root.Children, fiveKindsTree())

		// kind == 17 is a directory whose children are name-sorted, exactly as
		// the root's own flags word reads. It is not a rejection.
		global := r.Root.Children[0]
		if !global.Dir || !global.Sorted() {
			t.Errorf("Global: Dir = %v, Sorted() = %v, want true and true for kind 17", global.Dir, global.Sorted())
		}
		if global.NameTruncated() {
			t.Error("Global: NameTruncated() = true, want false (bit 28 is clear)")
		}
	})

	// AC-9 is two fixtures: they exercise different code and neither is a
	// rejection.
	t.Run("AC-9 an empty registry", func(t *testing.T) {
		r := mustParse(t, synth.Reg(17, nil))
		if r.NodeCount != 0 {
			t.Errorf("NodeCount = %d, want 0", r.NodeCount)
		}
		if r.Root.Children == nil {
			t.Error("Root.Children is nil, want empty-but-non-nil")
		}
		if len(r.Root.Children) != 0 {
			t.Errorf("Root has %d children, want 0", len(r.Root.Children))
		}
	})

	t.Run("AC-9 heapless but populated", func(t *testing.T) {
		stream := synth.Reg(17, []synth.RegNode{
			{Name: "Count", Kind: 0x02, Int: 7},
			{Name: "Scale", Kind: 0x04, Float: 1.5},
		})
		// The fixture is only the case it claims to be if its heap really is
		// empty: heapSize sits at 0x18 + 0x20*nodeCount.
		if hs := binary.LittleEndian.Uint32(stream[0x18+0x20*2:]); hs != 0 {
			t.Fatalf("fixture heapSize = %d, want 0 (no string or array value)", hs)
		}
		r := mustParse(t, stream)
		checkChildren(t, "", r.Root.Children, []tree{
			{name: "Count", kind: 0x02, typ: reg.TypeInt, i32: 7},
			{name: "Scale", kind: 0x04, typ: reg.TypeFloat, f64: 1.5},
		})
	})

	t.Run("the hand-laid stream", func(t *testing.T) {
		// The IEEE-754 claim the hex below makes about node 3, checked against
		// the standard library rather than against the parser under test.
		if bits := math.Float64bits(0.1); bits != 0x3fb999999999999a {
			t.Fatalf("math.Float64bits(0.1) = %#016x; node 3's bytes encode 0x3fb999999999999a", bits)
		}
		stream := hexStream(t, handLaid)
		if len(stream) != 233 {
			t.Fatalf("the hand-laid stream decodes to %d bytes, but its own layout adds up to 233"+
				" (0x18 header + 6*0x20 records + 4 heapSize + 13 heap)", len(stream))
		}

		r := mustParse(t, stream)
		if r.NodeCount != 6 {
			t.Errorf("NodeCount = %d, want 6", r.NodeCount)
		}
		if r.Root.Kind != 0x11 {
			t.Errorf("Root.Kind = %#x, want 0x11", r.Root.Kind)
		}
		checkChildren(t, "", r.Root.Children, []tree{
			{name: "Fade", kind: 0x11, dir: true, kids: []tree{
				{name: "Title", kind: 0x00, typ: reg.TypeString, str: "trio"},
				{name: "Gain", kind: 0x04, typ: reg.TypeFloat, f64: 0.1},
				{name: "Sub", kind: 0x01, dir: true, kids: []tree{
					{name: "Depth", kind: 0x02, typ: reg.TypeInt, i32: -7},
				}},
			}},
			{name: "Steps", kind: 0x06, typ: reg.TypeIntArray, ints: []int32{7, -1}},
		})
	})
}

// handLaid is a .reg stream written out byte by byte from spec.md's Format
// definition tables and produced by no builder. It is the instrument outside the
// model: a fixture builder and a parser that share a misread offset agree with
// each other and pass every test between them, so one stream here has to have
// been laid by hand and its expected tree read off the layout tables rather than
// off a parser run.
//
// It carries, deliberately, the four things a thinner witness would miss — two
// heap-backed values (a misplaced heapOrigin is otherwise invisible), nodes at
// six different table indices (a wrong node stride is otherwise invisible), two
// directories with child ranges (a wrong first/count slot is otherwise
// invisible), and a type-4 node (the data/size word order).
//
// The double is 0.1, whose binary64 pattern 0x3fb999999999999a has a non-zero
// mantissa low word, so a swapped pair of words cannot pass. The array's heap
// offset is 5, deliberately unaligned: the format constrains an array's size to
// a multiple of 4 and says nothing about its offset.
const handLaid = `
	26 59 41 31   // 0x00 signature = 0x31415926
	00 00 00 00   // 0x04 rootFirst = 0
	02 00 00 00   // 0x08 rootCount = 2
	11 00 00 00   // 0x0c rootFlags = 17 (directory, children name-sorted)
	06 00 00 00   // 0x10 nodeCount = 6
	00 00 00 00   // 0x14 reserved

	// node 0 @ 0x18 - "Fade", directory + sorted, children [2, 5)
	00 00 00 00   // reserved
	02 00 00 00   // data = first child 2
	03 00 00 00   // size = 3 children
	11 00 00 00   // kind = 0x11
	46 61 64 65 00 00 00 00 00 00 00 00 00 00 00 00   // "Fade"

	// node 1 @ 0x38 - "Steps", type 6 int32[2], heap [5, 13)
	00 00 00 00
	05 00 00 00   // data = heap offset 5
	08 00 00 00   // size = 8 bytes = two int32
	06 00 00 00   // kind = 0x06
	53 74 65 70 73 00 00 00 00 00 00 00 00 00 00 00   // "Steps"

	// node 2 @ 0x58 - "Title", type 0 string, heap [0, 5)
	00 00 00 00
	00 00 00 00   // data = heap offset 0
	05 00 00 00   // size = 5, the trailing NUL included
	00 00 00 00   // kind = 0x00
	54 69 74 6c 65 00 00 00 00 00 00 00 00 00 00 00   // "Title"

	// node 3 @ 0x78 - "Gain", type 4 float64 = 0.1
	00 00 00 00
	9a 99 99 99   // data = low  word 0x9999999a
	99 99 b9 3f   // size = high word 0x3fb99999
	04 00 00 00   // kind = 0x04
	47 61 69 6e 00 00 00 00 00 00 00 00 00 00 00 00   // "Gain"

	// node 4 @ 0x98 - "Sub", directory, children [5, 6)
	00 00 00 00
	05 00 00 00   // data = first child 5
	01 00 00 00   // size = 1 child
	01 00 00 00   // kind = 0x01
	53 75 62 00 00 00 00 00 00 00 00 00 00 00 00 00   // "Sub"

	// node 5 @ 0xb8 - "Depth", type 2 int32 = -7
	00 00 00 00
	f9 ff ff ff   // data = -7
	00 00 00 00   // size ignored by the type
	02 00 00 00   // kind = 0x02
	44 65 70 74 68 00 00 00 00 00 00 00 00 00 00 00   // "Depth"

	0d 00 00 00   // 0xd8 heapSize = 13

	74 72 69 6f 00   // 0xdc heap [0, 5)   "trio\0"
	07 00 00 00      //      heap [5, 9)   7
	ff ff ff ff      //      heap [9, 13)  -1
`

// ---------------------------------------------------------------------------
// SC-2 — the double
// ---------------------------------------------------------------------------

func TestParseFloat(t *testing.T) {
	// pi: 0x400921fb54442d18. The low word 0x54442d18 is non-zero, so a parser
	// that swapped data and size, or read only the high half, cannot pass.
	bits := uint64(0x400921fb54442d18)
	if math.Float64bits(math.Pi) != bits {
		t.Fatalf("math.Float64bits(math.Pi) = %#016x, want %#016x", math.Float64bits(math.Pi), bits)
	}
	stream := synth.RegRaw(0, 1, 17, 1, []synth.RegRawNode{
		{Name: []byte("Scale"), Kind: 0x04, Data: uint32(bits), Size: uint32(bits >> 32)},
	}, nil)
	if lo := binary.LittleEndian.Uint32(stream[0x18+0x04:]); lo == 0 {
		t.Fatal("the fixture's mantissa low word is zero; it cannot witness the word order")
	}

	r := mustParse(t, stream)
	got := r.Root.Children[0]
	if math.Float64bits(got.Float) != bits {
		t.Errorf("Float = %v (%#016x), want %v (%#016x)", got.Float, math.Float64bits(got.Float), math.Pi, bits)
	}

	// The pair the developer-run criterion will read off a cutscene registry.
	r = mustParse(t, synth.Reg(17, []synth.RegNode{
		{Name: "startfade", Kind: 0x04, Float: 0.0},
		{Name: "endfade", Kind: 0x04, Float: 1.0},
	}))
	for i, want := range []float64{0.0, 1.0} {
		if got := r.Root.Children[i].Float; math.Float64bits(got) != math.Float64bits(want) {
			t.Errorf("%s = %v, want %v", r.Root.Children[i].Name, got, want)
		}
	}
}

// ---------------------------------------------------------------------------
// SC-3 — framing rejections
// ---------------------------------------------------------------------------

func TestParseRejectsFraming(t *testing.T) {
	good := synth.Reg(17, fiveKinds())

	badSig := slices.Clone(good)
	binary.LittleEndian.PutUint32(badSig[0x00:], 0x31415927)

	for _, c := range []struct {
		name   string
		stream []byte
	}{
		{"empty", nil},
		{"shorter than the header", good[:0x10]},
		{"truncated mid node table", good[:0x18+0x20]},
		{"truncated before the heapSize word", good[:len(good)-20]},
		{"bad signature", badSig},
		// The header claims four nodes and one record follows it.
		{"node table shorter than nodeCount", synth.RegRaw(0, 1, 17, 4,
			[]synth.RegRawNode{{Name: []byte("a"), Kind: 0x02, Data: 5}}, nil)},
	} {
		t.Run(c.name, func(t *testing.T) {
			r, err := reg.Parse(c.stream)
			if err == nil {
				t.Fatalf("Parse succeeded, want an error")
			}
			if r != nil {
				t.Errorf("Parse returned a tree beside its error: %+v", r)
			}
		})
	}

	// The boundary on the other side of the same length check: the spec
	// tolerates bytes past the heap end.
	t.Run("trailing bytes past the heap end", func(t *testing.T) {
		stream := append(slices.Clone(good), 0xde, 0xad, 0xbe, 0xef)
		r := mustParse(t, stream)
		checkChildren(t, "", r.Root.Children, fiveKindsTree())
	})
}

// ---------------------------------------------------------------------------
// SC-4 — range, cycle and depth rejections
// ---------------------------------------------------------------------------

func TestParseRejectsRanges(t *testing.T) {
	dir := func(first, count uint32) synth.RegRawNode {
		return synth.RegRawNode{Name: []byte("d"), Kind: 0x01, Data: first, Size: count}
	}
	val := synth.RegRawNode{Name: []byte("v"), Kind: 0x02, Data: 1}

	for _, c := range []struct {
		name   string
		stream []byte
		want   string // a substring the message must carry, "" for any error
	}{
		{"a directory range past nodeCount",
			synth.RegRaw(0, 1, 17, 2, []synth.RegRawNode{dir(1, 9), val}, nil), "node 0"},
		// The root's range has no node record of its own. A parser that forgot
		// to check it would slice the node table out of range — a panic, not an
		// error.
		{"the header's own root range past nodeCount",
			synth.RegRaw(0, 5, 17, 2, []synth.RegRawNode{val, val}, nil), "root child range"},
		{"the header's own rootFirst past nodeCount",
			synth.RegRaw(7, 1, 17, 2, []synth.RegRawNode{val, val}, nil), "root child range"},
		// first + count wraps to 1 in 32-bit arithmetic, which is inside a
		// 2-node table; in uint64 it is 0x100000001 and is not.
		{"a node range whose first + count overflows 32 bits",
			synth.RegRaw(0, 1, 17, 2, []synth.RegRawNode{dir(0xffffffff, 2), val}, nil), "node 0"},
		{"a root range whose first + count overflows 32 bits",
			synth.RegRaw(0xffffffff, 2, 17, 2, []synth.RegRawNode{val, val}, nil), "root child range"},
		// Precedence: the single-parent set is the cycle guard, so it is what
		// fires here — the walk re-enters an index it has already marked, at a
		// depth far below the cap. The depth guard is only the stack backstop
		// and must not be what terminates a cycle.
		{"a self-referencing range",
			synth.RegRaw(0, 1, 17, 1, []synth.RegRawNode{dir(0, 1)}, nil),
			"node 0 referenced more than once"},
		{"a two-node cycle",
			synth.RegRaw(0, 1, 17, 2, []synth.RegRawNode{dir(1, 1), dir(0, 1)}, nil),
			"node 0 referenced more than once"},
	} {
		t.Run(c.name, func(t *testing.T) {
			r, err := reg.Parse(c.stream)
			if err == nil {
				t.Fatalf("Parse succeeded, want an error")
			}
			if r != nil {
				t.Errorf("Parse returned a tree beside its error: %+v", r)
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Errorf("error %q does not mention %q", err, c.want)
			}
		})
	}

	// The depth boundary, asserted at both sides. regChain(n) nests n nodes
	// below the root, so the deepest sits at depth n.
	t.Run("32 levels below the root parse", func(t *testing.T) {
		r := mustParse(t, regChain(32))
		if got := depth(r.Root); got != 32 {
			t.Errorf("the parsed chain is %d deep, want 32 — the fixture is not the boundary it claims", got)
		}
	})
	t.Run("33 levels are rejected", func(t *testing.T) {
		// 33 *distinct* nodes, so the depth guard is the only rule that can
		// fire — the other side of the precedence the cycle cases pin.
		r, err := reg.Parse(regChain(33))
		if err == nil {
			t.Fatalf("Parse succeeded, want a depth-guard error")
		}
		if r != nil {
			t.Errorf("Parse returned a tree beside its error")
		}
		if !strings.Contains(err.Error(), "depth guard") {
			t.Errorf("error %q is not the depth guard's", err)
		}
	})
}

// regChain builds a registry of n nodes in a single chain below the root: node i
// is a directory holding node i+1, and the last node is an int32. The root's one
// child is node 0, so node i sits at depth i+1 and the deepest node at depth n.
func regChain(n int) []byte {
	nodes := make([]synth.RegRawNode, n)
	for i := range nodes {
		name := []byte(fmt.Sprintf("n%d", i))
		if i == n-1 {
			nodes[i] = synth.RegRawNode{Name: name, Kind: 0x02, Data: uint32(i)}
			continue
		}
		nodes[i] = synth.RegRawNode{Name: name, Kind: 0x01, Data: uint32(i + 1), Size: 1}
	}
	return synth.RegRaw(0, 1, 17, uint32(n), nodes, nil)
}

// depth reports how many node levels sit below n. The root is depth 0, so a
// chain of 32 on-disk nodes below it returns 32.
func depth(n *reg.Node) int {
	best := 0
	for _, c := range n.Children {
		if d := depth(c) + 1; d > best {
			best = d
		}
	}
	return best
}

// ---------------------------------------------------------------------------
// SC-5 — value-type rejections
// ---------------------------------------------------------------------------

// regDefect builds a four-node registry whose first three nodes are well-formed
// int32 values and whose node 3 is the record under test. Index 3 is chosen
// because it is neither 8 nor 10 nor a substring of either, so an assertion that
// reads a type number and a node index out of a message cannot pass on the
// wrong one.
func regDefect(bad synth.RegRawNode, heap []byte) []byte {
	nodes := []synth.RegRawNode{
		{Name: []byte("a"), Kind: 0x02, Data: 1},
		{Name: []byte("b"), Kind: 0x02, Data: 2},
		{Name: []byte("c"), Kind: 0x02, Data: 3},
		bad,
	}
	return synth.RegRaw(0, 4, 17, 4, nodes, heap)
}

func TestParseRejectsValues(t *testing.T) {
	heap := []byte{0x01, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00} // 8 bytes

	for _, c := range []struct {
		name   string
		bad    synth.RegRawNode
		heap   []byte
		want   []string
		reject string
	}{
		{
			name: "type 6 with a size that is not a multiple of 4",
			bad:  synth.RegRawNode{Name: []byte("arr"), Kind: 0x06, Data: 0, Size: 6},
			heap: heap,
			want: []string{"node 3"},
		},
		{
			name: "a type-0 heap overrun",
			bad:  synth.RegRawNode{Name: []byte("str"), Kind: 0x00, Data: 4, Size: 8},
			heap: heap,
			want: []string{"node 3"},
		},
		{
			name: "a type-6 heap overrun",
			bad:  synth.RegRawNode{Name: []byte("arr"), Kind: 0x06, Data: 4, Size: 8},
			heap: heap,
			want: []string{"node 3"},
		},
		{
			// Decoded but deliberately not implemented, so it is named
			// unsupported rather than unrecognised: the words are contract.
			name: "a type-10 node",
			bad:  synth.RegRawNode{Name: []byte("floats"), Kind: 0x0a, Data: 0, Size: 8},
			heap: heap,
			want: []string{"unsupported", "10", "node 3"},
		},
		{
			name: "a type-8 node",
			bad:  synth.RegRawNode{Name: []byte("opaque"), Kind: 0x08},
			want: []string{"unrecognised", "8", "node 3"},
		},
		{
			// A directory that claims a value type: bit 0 over non-zero type
			// bits, which the spec assigns no meaning to.
			name: "bit 0 set over non-zero type bits",
			bad:  synth.RegRawNode{Name: []byte("both"), Kind: 0x03},
			want: []string{"node 3"},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			r, err := reg.Parse(regDefect(c.bad, c.heap))
			if err == nil {
				t.Fatalf("Parse succeeded, want an error")
			}
			if r != nil {
				t.Errorf("Parse returned a tree beside its error: %+v", r)
			}
			for _, w := range c.want {
				if !strings.Contains(err.Error(), w) {
					t.Errorf("error %q does not carry %q", err, w)
				}
			}
		})
	}

	// The two words are different on purpose and must not be interchanged: type
	// 10 is decoded and unimplemented, type 8 is undecoded.
	_, err := reg.Parse(regDefect(synth.RegRawNode{Kind: 0x0a}, nil))
	if err == nil || strings.Contains(err.Error(), "unrecognised") {
		t.Errorf("a type-10 node reported %v, want the word \"unsupported\"", err)
	}
	_, err = reg.Parse(regDefect(synth.RegRawNode{Kind: 0x08}, nil))
	if err == nil || strings.Contains(err.Error(), "unsupported") {
		t.Errorf("a type-8 node reported %v, want the word \"unrecognised\"", err)
	}
}

// ---------------------------------------------------------------------------
// SC-6 — byte fidelity
// ---------------------------------------------------------------------------

func TestParseBytesVerbatim(t *testing.T) {
	// Written as hex, never as literal non-ASCII text. Node 0's name fills all
	// 16 bytes with no NUL — a stream the format's own writer could not have
	// produced, which a reader must nevertheless take whole rather than run
	// into the next record.
	name0 := []byte{0x80, 0x81, 0x90, 0xa0, 0xb5, 0xc0, 0xd0, 0xe0, 0xf0, 0xfe, 0xff, 0x41, 0x5a, 0x7e, 0x7f, 0xcd}
	name1 := []byte{0x4b, 0xc0, 0xff}
	// "A", 0x80, 0xff, 0xfe, "z", NUL, then a byte past the terminator that the
	// value must not reach.
	heapBytes := []byte{0x41, 0x80, 0xff, 0xfe, 0x7a, 0x00, 0xcd}
	wantStr := []byte{0x41, 0x80, 0xff, 0xfe, 0x7a}

	if len(name0) != 16 {
		t.Fatalf("the 16-byte name fixture is %d bytes", len(name0))
	}
	if bytes.IndexByte(name0, 0x00) >= 0 {
		t.Fatal("the 16-byte name fixture carries a NUL; it cannot witness the no-terminator case")
	}

	r := mustParse(t, synth.RegRaw(0, 2, 17, 2, []synth.RegRawNode{
		{Name: name0, Kind: 0x00, Data: 0, Size: uint32(len(heapBytes))},
		{Name: name1, Kind: 0x02, Data: 42},
	}, heapBytes))

	n0, n1 := r.Root.Children[0], r.Root.Children[1]
	if got := []byte(n0.Name); !bytes.Equal(got, name0) {
		t.Errorf("node 0 name = %x, want %x unchanged byte for byte", got, name0)
	}
	if got := []byte(n0.Str); !bytes.Equal(got, wantStr) {
		t.Errorf("node 0 value = %x, want %x (cut at the first NUL, no mapping applied)", got, wantStr)
	}
	if got := []byte(n1.Name); !bytes.Equal(got, name1) {
		t.Errorf("node 1 name = %x, want %x", got, name1)
	}
	if n1.Int != 42 {
		t.Errorf("node 1 value = %d, want 42 — node 0's name ran into the next record", n1.Int)
	}
}

// ---------------------------------------------------------------------------
// SC-8 — reachability, orphans and shared nodes
// ---------------------------------------------------------------------------

func TestReachability(t *testing.T) {
	t.Run("every node is reached exactly once", func(t *testing.T) {
		r := mustParse(t, synth.Reg(17, fiveKinds()))
		walked, unique := walk(r.Root)
		if walked != r.NodeCount {
			t.Errorf("the walk yielded %d nodes, want NodeCount = %d", walked, r.NodeCount)
		}
		if unique != r.NodeCount {
			t.Errorf("the walk yielded %d distinct nodes out of %d, so one was reached twice", unique, walked)
		}
	})

	// A three-node tree plus one node no range references. The orphan is
	// ignored — absent from the tree and not an error — but its own record is
	// validated exactly like any other.
	orphaned := func(orphan synth.RegRawNode) []byte {
		return synth.RegRaw(0, 1, 17, 4, []synth.RegRawNode{
			{Name: []byte("dir"), Kind: 0x01, Data: 1, Size: 2},
			{Name: []byte("a"), Kind: 0x02, Data: 1},
			{Name: []byte("b"), Kind: 0x02, Data: 2},
			orphan,
		}, nil)
	}

	t.Run("a well-formed orphan is ignored", func(t *testing.T) {
		r := mustParse(t, orphaned(synth.RegRawNode{Name: []byte("orphan"), Kind: 0x02, Data: 9}))
		if r.NodeCount != 4 {
			t.Fatalf("NodeCount = %d, want 4 (the table's length, orphan included)", r.NodeCount)
		}
		walked, _ := walk(r.Root)
		if walked != r.NodeCount-1 {
			t.Errorf("the walk yielded %d nodes, want NodeCount-1 = %d", walked, r.NodeCount-1)
		}
		for _, n := range flatten(r.Root) {
			if n.Name == "orphan" {
				t.Error("the orphan is present in the tree, want it absent")
			}
		}
	})

	t.Run("an orphan of an unrecognised type is still rejected", func(t *testing.T) {
		// "Ignored" is about reachability, not a validation exemption.
		r, err := reg.Parse(orphaned(synth.RegRawNode{Name: []byte("orphan"), Kind: 0x08}))
		if err == nil {
			t.Fatalf("Parse succeeded, want an error naming the type-8 orphan")
		}
		if r != nil {
			t.Errorf("Parse returned a tree beside its error")
		}
		if !strings.Contains(err.Error(), "node 3") || !strings.Contains(err.Error(), "unrecognised") {
			t.Errorf("error %q does not name the offending orphan and its type", err)
		}
	})

	t.Run("a doubly-referenced node is rejected", func(t *testing.T) {
		// Two distinct, non-nested directories both list node 2 — the acyclic
		// case, distinct from a cycle.
		stream := synth.RegRaw(0, 2, 17, 3, []synth.RegRawNode{
			{Name: []byte("left"), Kind: 0x01, Data: 2, Size: 1},
			{Name: []byte("right"), Kind: 0x01, Data: 2, Size: 1},
			{Name: []byte("shared"), Kind: 0x02, Data: 5},
		}, nil)
		r, err := reg.Parse(stream)
		if err == nil {
			t.Fatalf("Parse succeeded, want an error")
		}
		if r != nil {
			t.Errorf("Parse returned a tree beside its error")
		}
		if !strings.Contains(err.Error(), "node 2 referenced more than once") {
			t.Errorf("error %q does not name the doubly-referenced node index 2", err)
		}
	})
}

// walk counts the nodes below n and the number of distinct pointers among them.
func walk(n *reg.Node) (total, distinct int) {
	seen := map[*reg.Node]bool{}
	for _, c := range flatten(n) {
		total++
		seen[c] = true
	}
	return total, len(seen)
}

func flatten(n *reg.Node) []*reg.Node {
	var out []*reg.Node
	for _, c := range n.Children {
		out = append(out, c)
		out = append(out, flatten(c)...)
	}
	return out
}

func FuzzParse(f *testing.F) {
	f.Add(synth.Reg(17, fiveKinds()))
	f.Add(hexStream(f, handLaid))
	f.Add([]byte(nil))
	f.Add(synth.Reg(17, fiveKinds())[:0x18])                                // header only
	f.Add(synth.RegRaw(0, 1, 17, 4, []synth.RegRawNode{{Kind: 0x02}}, nil)) // truncated table
	f.Add(synth.RegRaw(0, 1, 17, 1, []synth.RegRawNode{{Kind: 0x08}}, nil)) // type 8
	f.Add(synth.RegRaw(0, 1, 17, 2, []synth.RegRawNode{{Kind: 0x01, Data: 1, Size: 1},
		{Kind: 0x01, Data: 0, Size: 1}}, nil)) // a cycle

	f.Fuzz(func(t *testing.T, data []byte) {
		// Exactly one of the two shapes, and nothing more: the tree is not
		// walked here, because a walk over a tree a broken parser produced need
		// not terminate and a fuzz target that hangs reports nothing at all.
		r, err := reg.Parse(data)
		switch {
		case err != nil && r != nil:
			t.Fatalf("Parse returned both a tree and the error %v", err)
		case err == nil && r == nil:
			t.Fatal("Parse returned neither a tree nor an error")
		}
	})
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func mustParse(t *testing.T, stream []byte) *reg.Reg {
	t.Helper()
	r, err := reg.Parse(stream)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if r == nil {
		t.Fatal("Parse returned a nil *Reg and a nil error")
	}
	return r
}

// hexStream decodes an annotated hex fixture: everything from "//" to the end of
// a line is a comment and all other whitespace is insignificant.
func hexStream(tb testing.TB, s string) []byte {
	tb.Helper()
	var b strings.Builder
	for _, line := range strings.Split(s, "\n") {
		if i := strings.Index(line, "//"); i >= 0 {
			line = line[:i]
		}
		for _, f := range strings.Fields(line) {
			b.WriteString(f)
		}
	}
	out, err := hex.DecodeString(b.String())
	if err != nil {
		tb.Fatalf("the hand-laid stream is not valid hex: %v", err)
	}
	return out
}

// ---------------------------------------------------------------------------
// Size — the framed extent
// ---------------------------------------------------------------------------

// TestSizeReportsTheFramedExtent: Size answers where the registry ENDS, which
// is what a consumer splitting an embedded registry out of a larger stream
// needs and what Parse does not report.
//
// The stream that motivates it is the original save's uncompressed tail, where
// 268 bytes of a different structure follow the registry. Parse tolerates them;
// a reader that took len(data) as the registry would read them as heap.
func TestSizeReportsTheFramedExtent(t *testing.T) {
	heap := []byte{0x01, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00}
	body := synth.RegRaw(0, 1, 0, 1, []synth.RegRawNode{
		{Name: []byte("n"), Kind: 0x06, Data: 0, Size: uint32(len(heap))},
	}, heap)
	want := len(body)
	if n, err := reg.Size(body); err != nil || n != want {
		t.Fatalf("Size(exact) = %d, %v; want %d", n, err, want)
	}
	trailed := append(append([]byte(nil), body...), make([]byte, 268)...)
	if n, err := reg.Size(trailed); err != nil || n != want {
		t.Fatalf("Size(trailed) = %d, %v; want %d — the trailing region was counted as registry",
			n, err, want)
	}
	// Parse and Size agree on the same stream: one reads the tree out of it,
	// the other says how much of it the tree occupies.
	r, err := reg.Parse(trailed)
	if err != nil {
		t.Fatalf("Parse(trailed): %v", err)
	}
	if r.NodeCount != 1 {
		t.Fatalf("NodeCount %d, want 1", r.NodeCount)
	}
}

// TestSizeRefusesWhatItCannotSplit: each of these is a stream no split can be
// taken from, and each is refused rather than answered with a number a caller
// would slice on.
func TestSizeRefusesWhatItCannotSplit(t *testing.T) {
	good := synth.RegRaw(0, 0, 0, 0, nil, nil)
	badSig := append([]byte(nil), good...)
	binary.LittleEndian.PutUint32(badSig[0x00:], 0xdeadbeef)
	// A header claiming 99 nodes over a table holding none: the node table
	// alone leaves the stream.
	shortTable := synth.RegRaw(0, 0, 0, 99, nil, nil)
	// A heap length larger than the bytes that follow it.
	bigHeap := append([]byte(nil), good...)
	binary.LittleEndian.PutUint32(bigHeap[synth.RegHeaderSize:], 9999)

	for _, c := range []struct {
		name string
		data []byte
	}{
		{"shorter than a header", good[:synth.RegHeaderSize-1]},
		{"bad signature", badSig},
		{"node table past the end", shortTable},
		{"no room for the heap-length word", good[:synth.RegHeaderSize]},
		{"heap past the end", bigHeap},
	} {
		t.Run(c.name, func(t *testing.T) {
			if n, err := reg.Size(c.data); err == nil {
				t.Fatalf("Size = %d, want an error", n)
			}
		})
	}
}
