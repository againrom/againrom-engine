package synth

import (
	"encoding/binary"
	"fmt"
	"math"
	"sort"
	"strconv"
)

// ---------------------------------------------------------------------------
// .reg registries (see docs/0011-reg-registry/spec.md, "Format definition")
// ---------------------------------------------------------------------------

// The .reg stream is
//
//	header (0x18) | node table (nodeCount x 0x20) | heapSize (u32) | heap
//
// with every integer little-endian. The header is
//
//	0x00 u32 signature (0x31415926)   0x0c u32 rootFlags
//	0x04 u32 rootFirst                0x10 u32 nodeCount
//	0x08 u32 rootCount                0x14 u32 reserved (zero)
//
// and each node record is
//
//	0x00 u32 reserved (zero)          0x0c u32 kind
//	0x04 u32 data                     0x10 char[16] name
//	0x08 u32 size
//
// The root directory has no node record of its own: the header is its record,
// which is why rootFirst/rootCount/rootFlags are parameters here rather than a
// node the caller builds.
const (
	regSignature = 0x31415926

	// RegHeaderSize is the fixed header length; the node table starts here.
	RegHeaderSize = 0x18
	// RegNodeSize is one node record, name field included.
	RegNodeSize = 0x20

	regNameLen = 16
)

// RegNodeOffset returns the byte offset of node i's 32-byte record. The node
// table follows the header with no padding, so this is the only address
// arithmetic in the builders — and the offset a test patches a field at.
func RegNodeOffset(i int) int { return RegHeaderSize + RegNodeSize*i }

// RegRawNode is one node record written verbatim: Name is laid into the 16-byte
// name field (NUL-padded, cut at 16), Data/Size/Kind are written as given.
// Nothing is validated.
//
// A Name of exactly 16 bytes therefore lands with no NUL — the stream the
// format's own writer could not produce, which a reader must nevertheless
// tolerate — and the record's reserved word is written as zero, as that writer
// zeroes it.
type RegRawNode struct {
	Name       []byte
	Data, Size uint32
	Kind       uint32
}

// RegRaw lays out header | node table | heapSize | heap with no validation of
// any kind: the escape hatch for fixtures a well-formed tree cannot express — an
// orphan, a cycle, a range past the table, an undecodable value type, a heap
// reference that overruns.
//
// nodeCount is written to the header from its own parameter, independently of
// len(nodes): passing a count larger than the table produces a stream that is
// short by design, which is the "node table shorter than nodeCount" fixture.
// The heapSize word follows the records actually written, i.e. it sits at
// RegNodeOffset(len(nodes)) — the format's 0x18 + 0x20*nodeCount exactly when
// the two agree. The stream ends at the end of the heap.
func RegRaw(rootFirst, rootCount, rootFlags, nodeCount uint32, nodes []RegRawNode, heap []byte) []byte {
	out := make([]byte, RegHeaderSize+RegNodeSize*len(nodes)+4+len(heap))

	binary.LittleEndian.PutUint32(out[0x00:], regSignature)
	binary.LittleEndian.PutUint32(out[0x04:], rootFirst)
	binary.LittleEndian.PutUint32(out[0x08:], rootCount)
	binary.LittleEndian.PutUint32(out[0x0c:], rootFlags)
	binary.LittleEndian.PutUint32(out[0x10:], nodeCount)
	// 0x14, the header's reserved word, stays zero.

	for i, n := range nodes {
		rec := out[RegNodeOffset(i) : RegNodeOffset(i)+RegNodeSize]
		// rec[0x00], the node's reserved word, stays zero.
		binary.LittleEndian.PutUint32(rec[0x04:], n.Data)
		binary.LittleEndian.PutUint32(rec[0x08:], n.Size)
		binary.LittleEndian.PutUint32(rec[0x0c:], n.Kind)
		copy(rec[0x10:0x10+regNameLen], n.Name)
	}

	tableEnd := RegNodeOffset(len(nodes))
	binary.LittleEndian.PutUint32(out[tableEnd:], uint32(len(heap)))
	copy(out[tableEnd+4:], heap)
	return out
}

// RegNode is a node in a well-formed tree. Only the field the Kind's type bits
// select is read; a record word the type ignores is written as 0.
//
// Kind is the raw kind word: bit 0 marks a directory (and then Children is the
// field that is read), bit 4 marks the child list as name-sorted, and bits
// 1..3 carry the value type — 0 string, 2 int32, 4 float64, 6 int32 array.
type RegNode struct {
	Name     string
	Kind     uint32
	Children []RegNode
	Str      string
	Int      int32
	Float    float64
	Ints     []int32
}

// Reg assembles a well-formed registry from a tree, assigning node indices
// breadth-first (so each directory's children are a contiguous range) and
// packing heap items in node-index order with no gap. rootFlags is written
// verbatim.
//
// A string value goes to the heap with a trailing NUL and its size counts that
// NUL, as the format's own writer stores strlen(s)+1. A float64 goes into the
// record itself: data is the low half of its binary64 bit pattern and size the
// high half. Reg is implemented on top of RegRaw, so exactly one function in
// this package knows the byte layout.
//
// A Kind whose type bits name a value type Reg cannot build — an undecoded one,
// or the float64 array this project decodes but does not implement — panics:
// those fixtures belong to RegRaw, which validates nothing.
func Reg(rootFlags uint32, children []RegNode) []byte {
	// Breadth-first index assignment: the root's own children occupy indices
	// [0, len(children)), and each directory's children are appended as one
	// contiguous block recorded in first[i].
	order := append([]RegNode(nil), children...)
	first := make([]uint32, 0, len(children))
	for i := 0; i < len(order); i++ {
		first = append(first, uint32(len(order)))
		if order[i].Kind&0x01 != 0 {
			order = append(order, order[i].Children...)
		}
	}

	var heap []byte
	raw := make([]RegRawNode, len(order))
	for i, n := range order {
		rec := RegRawNode{Name: []byte(n.Name), Kind: n.Kind}
		if n.Kind&0x01 != 0 {
			rec.Data = first[i]
			rec.Size = uint32(len(n.Children))
		} else {
			switch n.Kind & 0x0e {
			case 0: // string: heap offset and byte length, the NUL included
				rec.Data = uint32(len(heap))
				rec.Size = uint32(len(n.Str) + 1)
				heap = append(heap, n.Str...)
				heap = append(heap, 0x00)
			case 2: // int32: the value itself, size ignored
				rec.Data = uint32(n.Int)
			case 4: // float64: the eight bytes at record offset 0x04
				bits := math.Float64bits(n.Float)
				rec.Data = uint32(bits)
				rec.Size = uint32(bits >> 32)
			case 6: // int32 array: heap offset and byte length
				rec.Data = uint32(len(heap))
				rec.Size = uint32(4 * len(n.Ints))
				for _, v := range n.Ints {
					heap = binary.LittleEndian.AppendUint32(heap, uint32(v))
				}
			default:
				panic(fmt.Sprintf("synth: Reg cannot build node %q of kind %#x; use RegRaw", n.Name, n.Kind))
			}
		}
		raw[i] = rec
	}

	return RegRaw(0, uint32(len(children)), rootFlags, uint32(len(order)), raw, heap)
}

// ---------------------------------------------------------------------------
// objects/objects.reg and units/units.reg
// (see docs/0016-data-classes/spec.md, "Source contract")
// ---------------------------------------------------------------------------

// The two [Files]-table registries share one shape: a flat directory of
// sections — [Global] carrying the class count and FileCount, [Files] carrying
// the extensionless, backslash-separated sprite paths at
// File0..File<FileCount-1>, and one class section per class, dense over
// [0, count). Only the count key's name and the section prefix differ:
// ObjectCount/[ObjectN] against UnitCount/[UnitN].
const (
	// regKindStr, regKindInt and regKindDir are the kind words this builder
	// writes: a value's type bits, and bit 0 for a directory.
	regKindStr = 0x00
	regKindDir = 0x01
	regKindInt = 0x02

	// regRootSorted is the root's own kind word in every shipped registry: a
	// directory (bit 0) whose child list is name-sorted (bit 4).
	regRootSorted = 0x11
)

// classReg assembles one [Files]-table registry — the shape ObjectsReg and
// UnitsReg share: [Global] with the two counts the classes and the file table
// imply, [Files] holding the given paths, and one class section per element of
// classes carrying exactly the keys that element supplies.
//
// It is deliberately MINIMAL. It writes the three structural facts a loader
// cannot work without — the section names, the dense numbering and the two
// counts — and asserts nothing about the keys: a caller supplies whichever ID,
// File, Width, Height, Parent or unknown key its case needs, and a case that
// needs a key missing simply omits it. The counts are DERIVED rather than
// parameters, so this builder cannot produce a registry whose [Global]
// disagrees with its own body; the fixtures that need that disagreement are
// RegRaw's, or 0016's own, and not a sprite layer's.
//
// A path in files is stored as given — backslash-separated and extensionless is
// the shipped convention, and translating one here would hide the translation
// the consumer performs. An empty string at some index is a legal thing to
// write and is how the "[Files] entry is empty" fault is built.
//
// The root's children are written NAME-SORTED, which is the on-disk order of
// the shipped registries and not section order: Files, Global, Object0,
// Object1, Object10, Object11, ..., Object2, ... The root's kind word says so
// (bit 4), so a builder that emitted section order would be writing a flag that
// lies about the stream it is in. Inside a section the keys stay in the order
// the caller gave them, since no shipped registry marks a section's child list
// sorted. The rule lives HERE, once, so the two builders over it cannot
// disagree about it.
func classReg(countKey, prefix string, files []string, classes ...[]RegNode) []byte {
	global := []RegNode{
		{Name: countKey, Kind: regKindInt, Int: int32(len(classes))},
		{Name: "FileCount", Kind: regKindInt, Int: int32(len(files))},
	}
	table := make([]RegNode, 0, len(files))
	for i, f := range files {
		table = append(table, RegNode{Name: "File" + strconv.Itoa(i), Kind: regKindStr, Str: f})
	}

	children := []RegNode{
		{Name: "Global", Kind: regKindDir, Children: global},
		{Name: "Files", Kind: regKindDir, Children: table},
	}
	for i, keys := range classes {
		children = append(children, RegNode{
			Name:     prefix + strconv.Itoa(i),
			Kind:     regKindDir,
			Children: keys,
		})
	}
	sort.Slice(children, func(i, j int) bool { return children[i].Name < children[j].Name })

	return Reg(regRootSorted, children)
}

// ObjectsReg assembles a synthetic objects/objects.reg: [Global] with
// ObjectCount and FileCount, [Files], and one [ObjectN] section per element of
// classes. See classReg for the shared shape and its conventions.
func ObjectsReg(files []string, classes ...[]RegNode) []byte {
	return classReg("ObjectCount", "Object", files, classes...)
}

// UnitsReg assembles a synthetic units/units.reg — ObjectsReg's sibling,
// writing UnitCount and [UnitN] sections over the same shared shape.
func UnitsReg(files []string, classes ...[]RegNode) []byte {
	return classReg("UnitCount", "Unit", files, classes...)
}

// ProjectilesReg assembles a synthetic graphics/projectiles/projectiles.reg:
// [Global] with Count alone, and one [ProjectileN] section per element of rows.
//
// IT WRITES NO [Files] TABLE, because the shipped registry has none: this
// registry's File key is a PATH STRING and not an index, unlike units.reg and
// objects.reg. That difference is the reason it is not classReg's third caller.
//
// count is written verbatim rather than derived from len(rows), so a fixture can
// state a Count that overshoots its own sections — which is a state the loader
// has a rule for and a fixture must be able to reach.
func ProjectilesReg(count int32, rows ...[]RegNode) []byte {
	children := []RegNode{{Name: "Global", Kind: regKindDir, Children: []RegNode{
		{Name: "Count", Kind: regKindInt, Int: count},
	}}}
	for i, keys := range rows {
		children = append(children, RegNode{
			Name:     "Projectile" + strconv.Itoa(i),
			Kind:     regKindDir,
			Children: keys,
		})
	}
	sort.Slice(children, func(i, j int) bool { return children[i].Name < children[j].Name })
	return Reg(regRootSorted, children)
}

// NPCReg assembles a synthetic scenario npc.reg: one `[npc<n>]` section per
// entry of defs, each carrying that entry's DataBinID.
//
// It writes NO [Global] section and no count key, because the shipped registry
// has neither: the per-NPC sections are found by name, not enumerated by a
// count, which is what lets a fixture hold two of them without declaring a
// hundred and five.
//
// Sections come out in ascending subscript order so one map yields one byte
// stream, and a fixture asserting on bytes is not asserting on Go's map
// iteration.
func NPCReg(defs map[int32]int32) []byte {
	ids := make([]int, 0, len(defs))
	for id := range defs {
		ids = append(ids, int(id))
	}
	sort.Ints(ids)
	sections := make([]RegNode, 0, len(ids))
	for _, id := range ids {
		sections = append(sections, RegNode{
			Name: "npc" + strconv.Itoa(id), Kind: 0x01,
			Children: []RegNode{{Name: "DataBinID", Kind: 0x02, Int: defs[int32(id)]}},
		})
	}
	return Reg(0x11, sections)
}
