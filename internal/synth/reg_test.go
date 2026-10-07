package synth_test

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/data"
	"againrom/pkg/formats/reg"
)

// The .reg layout, restated here from docs/0011-reg-registry/spec.md's Format
// definition tables. Every expectation in this file is computed from these
// constants rather than from the builder under test, so a builder that agreed
// with itself about a wrong offset still fails.
//
//	header:  0x00 signature | 0x04 rootFirst | 0x08 rootCount |
//	         0x0c rootFlags | 0x10 nodeCount | 0x14 reserved
//	node:    0x00 reserved  | 0x04 data      | 0x08 size      |
//	         0x0c kind      | 0x10 name[16]
//	then a u32 heapSize at 0x18 + 0x20*nodeCount, then heapSize heap bytes.
const (
	regSignature = 0x31415926

	regHdrSize  = 0x18
	regNodeSize = 0x20
	regNameLen  = 16

	offSignature   = 0x00
	offRootFirst   = 0x04
	offRootCount   = 0x08
	offRootFlags   = 0x0c
	offNodeCount   = 0x10
	offHdrReserved = 0x14

	offNodeReserved = 0x00
	offData         = 0x04
	offSize         = 0x08
	offKind         = 0x0c
	offName         = 0x10
)

// regNodeAt and regHeapOrigin are the spec's own address arithmetic, written
// out independently of synth.RegNodeOffset (which TestRegBuilders pins against
// them).
func regNodeAt(i int) int           { return regHdrSize + regNodeSize*i }
func regHeapOrigin(nodes int) int   { return regHdrSize + regNodeSize*nodes }
func le32(b []byte, off int) uint32 { return binary.LittleEndian.Uint32(b[off:]) }

// regName reads node i's 16-byte name field whole, with no NUL cutting: the
// field is 16 bytes of the node's own record whatever it holds.
func regName(b []byte, i int) []byte {
	return b[regNodeAt(i)+offName : regNodeAt(i)+offName+regNameLen]
}

// TestRegBuilders pins the .reg builders at the offsets the format contract
// names. It runs before any parser exists, which is the point: a builder and a
// parser written against the same misread offset agree perfectly and pass every
// test between them, so the builder is fixed against the spec first.
func TestRegBuilders(t *testing.T) {
	t.Run("the layout constants are the spec's", func(t *testing.T) {
		if synth.RegHeaderSize != regHdrSize {
			t.Errorf("RegHeaderSize = %#x, want %#x", synth.RegHeaderSize, regHdrSize)
		}
		if synth.RegNodeSize != regNodeSize {
			t.Errorf("RegNodeSize = %#x, want %#x", synth.RegNodeSize, regNodeSize)
		}
		for i := 0; i < 5; i++ {
			if got, want := synth.RegNodeOffset(i), regNodeAt(i); got != want {
				t.Errorf("RegNodeOffset(%d) = %#x, want %#x", i, got, want)
			}
		}
	})

	t.Run("RegRaw writes header, table, heapSize and heap", func(t *testing.T) {
		// Distinct values everywhere, so a swapped pair of fields cannot pass.
		nodes := []synth.RegRawNode{
			{Name: []byte("alpha"), Data: 0x11223344, Size: 0x55667788, Kind: 0x0000000d},
			{Name: []byte("beta"), Data: 0x99aabbcc, Size: 0xddeeff00, Kind: 0x10000006},
			{Name: []byte("gamma"), Data: 0x00000001, Size: 0xfffffffe, Kind: 0x00000011},
		}
		heap := []byte{0xde, 0xad, 0xbe, 0xef, 0x00, 0x7f}
		got := synth.RegRaw(7, 2, 17, 3, nodes, heap)

		if want := regHdrSize + regNodeSize*len(nodes) + 4 + len(heap); len(got) != want {
			t.Fatalf("stream is %d bytes, want %d (header + %d nodes + heapSize + heap)", len(got), want, len(nodes))
		}
		for _, c := range []struct {
			off  int
			name string
			want uint32
		}{
			{offSignature, "signature", regSignature},
			{offRootFirst, "rootFirst", 7},
			{offRootCount, "rootCount", 2},
			{offRootFlags, "rootFlags", 17},
			{offNodeCount, "nodeCount", 3},
			{offHdrReserved, "reserved", 0},
		} {
			if v := le32(got, c.off); v != c.want {
				t.Errorf("header %s at %#04x = %#08x, want %#08x", c.name, c.off, v, c.want)
			}
		}

		for i, n := range nodes {
			base := regNodeAt(i)
			if v := le32(got, base+offNodeReserved); v != 0 {
				t.Errorf("node %d reserved = %#08x, want 0", i, v)
			}
			if v := le32(got, base+offData); v != n.Data {
				t.Errorf("node %d data at %#04x = %#08x, want %#08x", i, base+offData, v, n.Data)
			}
			if v := le32(got, base+offSize); v != n.Size {
				t.Errorf("node %d size at %#04x = %#08x, want %#08x", i, base+offSize, v, n.Size)
			}
			if v := le32(got, base+offKind); v != n.Kind {
				t.Errorf("node %d kind at %#04x = %#08x, want %#08x", i, base+offKind, v, n.Kind)
			}
			want := make([]byte, regNameLen)
			copy(want, n.Name)
			if name := regName(got, i); !bytes.Equal(name, want) {
				t.Errorf("node %d name at %#04x = %x, want %x", i, base+offName, name, want)
			}
		}

		origin := regHeapOrigin(len(nodes))
		if v := le32(got, origin); v != uint32(len(heap)) {
			t.Errorf("heapSize at %#04x = %d, want %d", origin, v, len(heap))
		}
		if h := got[origin+4:]; !bytes.Equal(h, heap) {
			t.Errorf("heap at %#04x = %x, want %x", origin+4, h, heap)
		}
	})

	t.Run("RegRaw honours a nodeCount larger than the table it writes", func(t *testing.T) {
		// The header claims four nodes; one record is written. The stream is
		// short by design — the "node table shorter than nodeCount" fixture.
		got := synth.RegRaw(0, 1, 17, 4, []synth.RegRawNode{{Kind: 0x02, Data: 5}}, nil)
		if v := le32(got, offNodeCount); v != 4 {
			t.Errorf("nodeCount = %d, want 4", v)
		}
		if want := regHdrSize + regNodeSize*1 + 4; len(got) != want {
			t.Errorf("stream is %d bytes, want %d", len(got), want)
		}
		if len(got) >= regHeapOrigin(4) {
			t.Errorf("stream is %d bytes, want fewer than the %d the header's nodeCount implies", len(got), regHeapOrigin(4))
		}
	})

	t.Run("RegRaw writes a 16-byte name with no NUL and cuts a longer one", func(t *testing.T) {
		full := []byte("0123456789abcdef") // exactly 16 bytes
		over := []byte("0123456789abcdefghij")
		got := synth.RegRaw(0, 2, 17, 2, []synth.RegRawNode{{Name: full}, {Name: over}}, nil)

		if name := regName(got, 0); !bytes.Equal(name, full) {
			t.Errorf("16-byte name = %x, want %x", name, full)
		} else if bytes.IndexByte(name, 0x00) >= 0 {
			t.Errorf("16-byte name %x carries a NUL; the field must fill with no terminator", name)
		}
		if name, want := regName(got, 1), over[:regNameLen]; !bytes.Equal(name, want) {
			t.Errorf("over-long name = %x, want it cut to %x", name, want)
		}
	})

	t.Run("Reg assigns indices breadth-first over a two-level tree", func(t *testing.T) {
		got := synth.Reg(17, regTree())

		if v := le32(got, offNodeCount); v != 7 {
			t.Fatalf("nodeCount = %d, want 7 (2 root children + 3 + 2 grandchildren)", v)
		}
		if v := le32(got, offRootFirst); v != 0 {
			t.Errorf("rootFirst = %d, want 0 (the root's children are the first indices)", v)
		}
		if v := le32(got, offRootCount); v != 2 {
			t.Errorf("rootCount = %d, want 2", v)
		}
		if v := le32(got, offRootFlags); v != 17 {
			t.Errorf("rootFlags = %d, want 17 (written verbatim)", v)
		}

		// Breadth-first: the root's children first, then each directory's
		// children in the order their parents appear.
		want := []struct {
			name string
			kind uint32
		}{
			{"Global", 0x11}, {"Unit0", 0x01},
			{"Name", 0x00}, {"Count", 0x02}, {"Scale", 0x04},
			{"Desc", 0x00}, {"Frames", 0x06},
		}
		for i, w := range want {
			wantName := make([]byte, regNameLen)
			copy(wantName, w.name)
			if name := regName(got, i); !bytes.Equal(name, wantName) {
				t.Errorf("node %d name = %q, want %q", i, trimNUL(name), w.name)
			}
			if k := le32(got, regNodeAt(i)+offKind); k != w.kind {
				t.Errorf("node %d (%s) kind = %#x, want %#x", i, w.name, k, w.kind)
			}
		}

		// Each directory's children are a contiguous [first, first+count) range.
		for _, d := range []struct {
			i            int
			first, count uint32
			name         string
		}{
			{0, 2, 3, "Global"},
			{1, 5, 2, "Unit0"},
		} {
			if v := le32(got, regNodeAt(d.i)+offData); v != d.first {
				t.Errorf("%s first child = %d, want %d", d.name, v, d.first)
			}
			if v := le32(got, regNodeAt(d.i)+offSize); v != d.count {
				t.Errorf("%s child count = %d, want %d", d.name, v, d.count)
			}
		}

		// The two string values are packed into the heap in node-index order
		// with no gap: "alpha\0" at 0, "beta\0" at 6, then the int32 array.
		origin := regHeapOrigin(7)
		heapSize := le32(got, origin)
		if want := len("alpha") + 1 + len("beta") + 1 + 4*3; int(heapSize) != want {
			t.Fatalf("heapSize at %#04x = %d, want %d", origin, heapSize, want)
		}
		heap := got[origin+4 : origin+4+int(heapSize)]
		for _, s := range []struct {
			i         int
			off, size uint32
			want      string
		}{
			{2, 0, 6, "alpha\x00"},
			{5, 6, 5, "beta\x00"},
		} {
			if v := le32(got, regNodeAt(s.i)+offData); v != s.off {
				t.Errorf("node %d heap offset = %d, want %d (packed with no gap)", s.i, v, s.off)
			}
			if v := le32(got, regNodeAt(s.i)+offSize); v != s.size {
				t.Errorf("node %d size = %d, want %d (the trailing NUL counts)", s.i, v, s.size)
			}
			if got := string(heap[s.off : s.off+s.size]); got != s.want {
				t.Errorf("node %d heap bytes = %q, want %q", s.i, got, s.want)
			}
		}
		if v := le32(got, regNodeAt(6)+offData); v != 11 {
			t.Errorf("Frames heap offset = %d, want 11", v)
		}
		if v := le32(got, regNodeAt(6)+offSize); v != 12 {
			t.Errorf("Frames size = %d, want 12 (three int32)", v)
		}
		for k, w := range []int32{1, -2, 3} {
			if v := int32(binary.LittleEndian.Uint32(heap[11+4*k:])); v != w {
				t.Errorf("Frames[%d] = %d, want %d", k, v, w)
			}
		}

		// int32: the value itself in data, size ignored and written as zero.
		if v := int32(le32(got, regNodeAt(3)+offData)); v != 34 {
			t.Errorf("Count data = %d, want 34", v)
		}
		if v := le32(got, regNodeAt(3)+offSize); v != 0 {
			t.Errorf("Count size = %d, want 0 (the type ignores it)", v)
		}

		// float64: data is the low half of the bit pattern, size the high half.
		lo, hi := le32(got, regNodeAt(4)+offData), le32(got, regNodeAt(4)+offSize)
		if bits := uint64(hi)<<32 | uint64(lo); bits != math.Float64bits(math.Pi) {
			t.Errorf("Scale bits = %#016x (data %#08x, size %#08x), want %#016x with data the low word",
				bits, lo, hi, math.Float64bits(math.Pi))
		}
	})

	t.Run("Reg(17, nil) is a 28-byte empty registry", func(t *testing.T) {
		got := synth.Reg(17, nil)
		if len(got) != 28 {
			t.Fatalf("stream is %d bytes, want 28 (a 0x18 header and a zero heapSize word)", len(got))
		}
		if v := le32(got, offNodeCount); v != 0 {
			t.Errorf("nodeCount = %d, want 0", v)
		}
		if v := le32(got, regHeapOrigin(0)); v != 0 {
			t.Errorf("heapSize at %#04x = %d, want 0", regHeapOrigin(0), v)
		}
	})
}

// regTree is the two-level fixture TestRegBuilders and TestRegHandLaidStream
// share: two directories under the root, one of them flagged name-sorted, five
// values between them covering every kind Reg can build.
func regTree() []synth.RegNode {
	return []synth.RegNode{
		{Name: "Global", Kind: 0x11, Children: []synth.RegNode{
			{Name: "Name", Kind: 0x00, Str: "alpha"},
			{Name: "Count", Kind: 0x02, Int: 34},
			{Name: "Scale", Kind: 0x04, Float: math.Pi},
		}},
		{Name: "Unit0", Kind: 0x01, Children: []synth.RegNode{
			{Name: "Desc", Kind: 0x00, Str: "beta"},
			{Name: "Frames", Kind: 0x06, Ints: []int32{1, -2, 3}},
		}},
	}
}

// handLaidReg is the same registry regTree describes, written out byte by byte
// from spec.md's layout tables and produced by no builder. It is the instrument
// outside the model: a builder and a parser that share a misread offset agree
// with each other and pass every test between them, so one stream in this
// package has to have been laid by hand.
//
// It carries, deliberately, the four things a thinner witness would miss — a
// heap-backed value (a misplaced heapOrigin is otherwise invisible), nodes at
// several different table indices (a wrong node stride is otherwise invisible),
// a directory with a child range (a wrong first/count slot is otherwise
// invisible), and a type-4 node (the data/size word order).
//
// The double is pi, whose binary64 bit pattern is 0x400921fb54442d18 — a
// non-zero mantissa low word, so a swapped pair of words cannot pass.
const handLaidReg = `
	26 59 41 31   // 0x00 signature 0x31415926
	00 00 00 00   // 0x04 rootFirst = 0
	02 00 00 00   // 0x08 rootCount = 2
	11 00 00 00   // 0x0c rootFlags = 17 (directory, children name-sorted)
	07 00 00 00   // 0x10 nodeCount = 7
	00 00 00 00   // 0x14 reserved

	// node 0 @ 0x18 - "Global", directory + sorted, children [2, 5)
	00 00 00 00   // reserved
	02 00 00 00   // data = first child 2
	03 00 00 00   // size = 3 children
	11 00 00 00   // kind = 0x11
	47 6c 6f 62 61 6c 00 00 00 00 00 00 00 00 00 00   // "Global"

	// node 1 @ 0x38 - "Unit0", directory, children [5, 7)
	00 00 00 00
	05 00 00 00   // data = first child 5
	02 00 00 00   // size = 2 children
	01 00 00 00   // kind = 0x01
	55 6e 69 74 30 00 00 00 00 00 00 00 00 00 00 00   // "Unit0"

	// node 2 @ 0x58 - "Name", type 0 string, heap [0, 6)
	00 00 00 00
	00 00 00 00   // data = heap offset 0
	06 00 00 00   // size = 6, the trailing NUL included
	00 00 00 00   // kind = 0x00
	4e 61 6d 65 00 00 00 00 00 00 00 00 00 00 00 00   // "Name"

	// node 3 @ 0x78 - "Count", type 2 int32 = 34
	00 00 00 00
	22 00 00 00   // data = 34
	00 00 00 00   // size ignored
	02 00 00 00   // kind = 0x02
	43 6f 75 6e 74 00 00 00 00 00 00 00 00 00 00 00   // "Count"

	// node 4 @ 0x98 - "Scale", type 4 float64 = pi
	00 00 00 00
	18 2d 44 54   // data = low  word 0x54442d18
	fb 21 09 40   // size = high word 0x400921fb
	04 00 00 00   // kind = 0x04
	53 63 61 6c 65 00 00 00 00 00 00 00 00 00 00 00   // "Scale"

	// node 5 @ 0xb8 - "Desc", type 0 string, heap [6, 11)
	00 00 00 00
	06 00 00 00   // data = heap offset 6
	05 00 00 00   // size = 5
	00 00 00 00   // kind = 0x00
	44 65 73 63 00 00 00 00 00 00 00 00 00 00 00 00   // "Desc"

	// node 6 @ 0xd8 - "Frames", type 6 int32[3], heap [11, 23)
	00 00 00 00
	0b 00 00 00   // data = heap offset 11
	0c 00 00 00   // size = 12 bytes
	06 00 00 00   // kind = 0x06
	46 72 61 6d 65 73 00 00 00 00 00 00 00 00 00 00   // "Frames"

	17 00 00 00   // 0xf8 heapSize = 23

	61 6c 70 68 61 00               // 0xfc heap [0, 6)   "alpha\0"
	62 65 74 61 00                  //      heap [6, 11)  "beta\0"
	01 00 00 00                     //      heap [11, 15) 1
	fe ff ff ff                     //      heap [15, 19) -2
	03 00 00 00                     //      heap [19, 23) 3
`

// TestRegHandLaidStream is the anti-tautology check T2 exists for: the builders'
// output for regTree must equal, byte for byte, a stream laid out by hand from
// the spec's tables. If the two ever disagree, one of them is wrong against the
// format and the disagreement is the finding — it is never reconciled by moving
// whichever is easier to move.
func TestRegHandLaidStream(t *testing.T) {
	want := regHex(t, handLaidReg)

	// 0x18 header + 7 * 0x20 node records + a 4-byte heapSize + 23 heap bytes,
	// added up by hand from the layout above.
	if len(want) != 275 {
		t.Fatalf("the hand-laid stream decodes to %d bytes, but its own layout adds up to 275", len(want))
	}
	if math.Float64bits(math.Pi) != 0x400921fb54442d18 {
		t.Fatalf("math.Pi is %#016x; the hand-laid bytes at node 4 encode 0x400921fb54442d18",
			math.Float64bits(math.Pi))
	}

	got := synth.Reg(17, regTree())
	if !bytes.Equal(got, want) {
		t.Fatalf("synth.Reg disagrees with the hand-laid stream: %s", firstDiff(got, want))
	}
}

// firstDiff names where two streams part company, so a disagreement points at
// the field that differs rather than at a wall of hex.
func firstDiff(got, want []byte) string {
	n := min(len(got), len(want))
	for i := 0; i < n; i++ {
		if got[i] != want[i] {
			return fmt.Sprintf("first difference at offset %#04x: built %x, hand-laid %x",
				i, got[i:min(i+8, len(got))], want[i:min(i+8, len(want))])
		}
	}
	return fmt.Sprintf("streams agree over %d bytes but are %d and %d bytes long", n, len(got), len(want))
}

// regHex decodes an annotated hex fixture: everything from "//" to the end of a
// line is a comment and all other whitespace is insignificant.
func regHex(t *testing.T, s string) []byte {
	t.Helper()
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
		t.Fatalf("the hand-laid stream is not valid hex: %v", err)
	}
	return out
}

func trimNUL(b []byte) string {
	if i := bytes.IndexByte(b, 0x00); i >= 0 {
		return string(b[:i])
	}
	return string(b)
}

// loadUnitsReg parses a built stream and resolves it as units/units.reg.
func loadUnitsReg(t *testing.T, stream []byte) *data.UnitClasses {
	t.Helper()
	r, err := reg.Parse(stream)
	if err != nil {
		t.Fatalf("reg.Parse: %v", err)
	}
	cs, err := data.LoadUnitClasses(r)
	if err != nil {
		t.Fatalf("data.LoadUnitClasses: %v", err)
	}
	return cs
}

// TestUnitsRegBuilder — 0022 T3: ObjectsReg's sibling writes UnitCount and
// [UnitN] sections a units-registry load resolves. The round-trip exercises
// the one resolution the object registry cannot: units.reg's File INHERITS
// through Parent, where objects.reg's is noInherit — which is exactly what
// the unit-bundle fixtures lean on.
func TestUnitsRegBuilder(t *testing.T) {
	stream := synth.UnitsReg([]string{`warrior\walk`, `archer\walk`},
		[]synth.RegNode{
			regObjInt("ID", 3), regObjInt("File", 0), regObjInt("Flip", 0),
			regObjInt("Width", 48), regObjInt("Height", 56),
			regObjInt("CenterX", 24), regObjInt("CenterY", 50),
		},
		[]synth.RegNode{regObjInt("ID", 7), regObjInt("File", 1), regObjInt("Flip", 1)},
		// No File of its own: it must arrive down the Parent chain.
		[]synth.RegNode{regObjInt("ID", 9), regObjInt("Parent", 3)},
	)

	t.Run("a registry round-trips into LoadUnitClasses, File inheriting through Parent", func(t *testing.T) {
		cs := loadUnitsReg(t, stream)
		if got := len(cs.All()); got != 3 {
			t.Fatalf("All() has %d classes, want 3 — [Global] UnitCount must follow the sections written", got)
		}

		for _, want := range []struct {
			id    int32
			flip  int32
			width int32
			path  string
		}{
			{3, 0, 48, "units/warrior/walk.256"},
			// Width is set nowhere on this class's chain, so it reads units.reg's
			// decoded absent-everywhere default, -1 since 0024 (data/keys.go).
			{7, 1, -1, "units/archer/walk.256"},
			// File AND the scalars chain from ID 3: the sprite path is the
			// parent's, the canvas width its resolved 48.
			{9, 0, 48, "units/warrior/walk.256"},
		} {
			c, ok := cs.ByID(want.id)
			if !ok {
				t.Errorf("ByID(%d) missed", want.id)
				continue
			}
			if c.Flip != want.flip || c.Width != want.width {
				t.Errorf("ID %d: Flip/Width = %d/%d, want %d/%d",
					want.id, c.Flip, c.Width, want.flip, want.width)
			}
			if got := c.SpritePath(); got != want.path {
				t.Errorf("ID %d: SpritePath() = %q, want %q", want.id, got, want.path)
			}
		}
	})

	t.Run("the sibling keeps the name-sorted root", func(t *testing.T) {
		// Sorted order puts Files BEFORE Global, where the assembly order is
		// Global first — so the very first child discriminates the two. The
		// deeper Object10-before-Object2 case is pinned on the shared assembly
		// by TestObjectsRegBuilder.
		want := []string{"Files", "Global", "Unit0", "Unit1", "Unit2"}
		for i, name := range want {
			if got := trimNUL(regName(stream, i)); got != name {
				t.Errorf("root child %d = %q, want %q", i, got, name)
			}
		}
		if flags := le32(stream, offRootFlags); flags != 0x11 {
			t.Errorf("rootFlags = %#x, want %#x — a directory whose child list is name-sorted", flags, 0x11)
		}
	})
}
