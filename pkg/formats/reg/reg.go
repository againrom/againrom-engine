package reg

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
)

// Layout constants for the ROM1 .reg registry (little-endian throughout). See
// the package doc and docs/0011-reg-registry/spec.md for the byte-level format:
//
//	header (0x18) | node table (nodeCount x 0x20) | heapSize (u32) | heap
const (
	signature  = 0x31415926 // header[0x00], shared with the .res container
	headerSize = 0x18       // the header is also the root directory's record
	nodeSize   = 0x20       // one node record, its 16-byte name field included
	nameLen    = 16         // node[0x10:0x20]

	// maxDepth is the stack backstop, not the cycle guard: the single-parent
	// set below is what makes a cyclic or self-referencing range terminate.
	// The root is depth 0 and a child of the root is depth 1, so 32 levels of
	// on-disk nodes below the root parse and a node that would sit at depth 33
	// is rejected.
	maxDepth = 32
)

// Bits of a node's kind word. The word is a bitfield, not an enumeration: the
// value type is kind&typeMask and every other bit is a flag that is never on its
// own grounds for rejection.
const (
	flagDir       = 0x01       // the node is a directory; its type bits are 0
	flagSorted    = 0x10       // this node's child list is name-sorted
	flagTruncated = 0x10000000 // the name was longer than 15 chars and was cut
	typeMask      = 0x0e       // the value type
)

// ValueType is a node's value type, the kind word masked with 0x0E. It is
// meaningless for a directory, whose type bits are zero.
type ValueType uint32

// The value types this parser recognises. Every other value — type 8 included —
// is undecoded and rejected rather than guessed at.
const (
	TypeString   ValueType = 0 // heap offset in data, byte length in size
	TypeInt      ValueType = 2 // the signed value itself in data
	TypeFloat    ValueType = 4 // binary64: data is the low word, size the high
	TypeIntArray ValueType = 6 // heap offset in data, byte length in size

	// TypeFloatArray is decoded — heap offset in data, byte length in size,
	// size/8 little-endian doubles — but deliberately not implemented: no
	// shipped registry holds one, so Parse rejects a type-10 node as
	// unsupported rather than shipping a decoder nothing exercises.
	TypeFloatArray ValueType = 10
)

// Node is one decoded registry node. Exactly one of the value fields carries
// meaning, selected by Type, and only when Dir is false.
//
// Name and Str hold the stream's bytes verbatim. The format defines no character
// encoding and this package applies none, so neither is guaranteed to be valid
// UTF-8; whatever displays one chooses and states its own convention.
type Node struct {
	Name string // raw bytes, cut at the first NUL of the 16-byte name field
	Kind uint32 // the raw kind word, verbatim, flag bits included
	Dir  bool   // Kind&0x01: this node is a directory
	Type ValueType

	// Children is non-nil for every directory, including one with no children,
	// and nil for every value node, so Dir and Children != nil always agree.
	// Children are in node-table order.
	Children []*Node

	Str   string  // TypeString
	Int   int32   // TypeInt
	Float float64 // TypeFloat

	// Ints is the parser's own slice, not a copy: a caller reading the tree
	// directly must not mutate it. It is empty-but-non-nil for a zero-length
	// array.
	Ints []int32 // TypeIntArray

	// index is this node's position in the on-disk node table: its record's
	// first byte is at headerSize+nodeSize*index. It is meaningless for Root,
	// which has no on-disk record of its own. SetInt and SetIntArray are the
	// only readers; nothing here exposes it, so a caller cannot mistake it
	// for a stable cross-file identity.
	index int

	// rawD, rawSz are the node record's own two words verbatim: the value
	// itself for TypeInt, the two halves of a TypeFloat, or a heap [rawD,
	// rawD+rawSz) span for TypeString/TypeIntArray. SetIntArray uses them to
	// find and resize a value's own heap bytes without re-deriving them from
	// the decoded fields above.
	rawD, rawSz uint32
}

// Sorted reports the kind word's bit 4: this node's child list is name-sorted.
func (n *Node) Sorted() bool { return n.Kind&flagSorted != 0 }

// NameTruncated reports the kind word's bit 28: the name was longer than 15
// characters and the writer cut it.
func (n *Node) NameTruncated() bool { return n.Kind&flagTruncated != 0 }

// Reg is a parsed registry: one synthetic root directory over the node table.
type Reg struct {
	// Root is the root directory, and is never nil. It has no on-disk node and
	// no name — the header is its record — so Root.Name is "" and Root.Dir is
	// true by construction rather than derived from a kind word.
	//
	// Root.Kind carries the header's rootFlags verbatim and unvalidated: the
	// format exempts that word from every rejection ground, so nothing here has
	// checked it. Root.Type is always 0 and is never derived from it.
	Root *Node

	// NodeCount is the header's node count, i.e. the length of the node table.
	// It counts every on-disk node at every depth, orphans included, and so may
	// exceed the number of nodes reachable from Root.
	NodeCount int
}

// Parse decodes a .reg registry from data and returns its tree.
//
// Parse retains no reference to data: names, string values and array elements
// are copied out during the parse, so a caller may reuse or modify the slice
// afterwards. A malformed stream yields a nil *Reg and a non-nil error — never a
// partial tree and never a panic.
func Parse(data []byte) (*Reg, error) {
	// --- Framing -----------------------------------------------------------
	if len(data) < headerSize {
		return nil, fmt.Errorf("reg: stream too small: %d bytes", len(data))
	}
	if sig := binary.LittleEndian.Uint32(data[0x00:0x04]); sig != signature {
		return nil, fmt.Errorf("reg: bad signature %#08x", sig)
	}
	rootFirst := binary.LittleEndian.Uint32(data[0x04:0x08])
	rootCount := binary.LittleEndian.Uint32(data[0x08:0x0c])
	rootFlags := binary.LittleEndian.Uint32(data[0x0c:0x10])
	nodeCount := binary.LittleEndian.Uint32(data[0x10:0x14])
	// data[0x14:0x18], the header's reserved word, is read and ignored.

	// Every length here is uint64: nodeCount and heapSize are u32 read from the
	// stream, and int is 32 bits on a 32-bit build, so int arithmetic could wrap
	// where these cannot.
	total := uint64(len(data))
	heapOrigin := uint64(headerSize) + uint64(nodeSize)*uint64(nodeCount)
	if heapOrigin > total {
		return nil, fmt.Errorf("reg: node table of %d nodes needs %d bytes, stream has %d",
			nodeCount, heapOrigin, total)
	}
	if heapOrigin+4 > total {
		return nil, fmt.Errorf("reg: heapSize word at %d past the end of a %d-byte stream", heapOrigin, total)
	}
	heapSize := uint64(binary.LittleEndian.Uint32(data[int(heapOrigin) : int(heapOrigin)+4]))
	heapStart := heapOrigin + 4
	heapEnd := heapStart + heapSize
	if heapEnd > total {
		return nil, fmt.Errorf("reg: heap [%d, %d) past the end of a %d-byte stream", heapStart, heapEnd, total)
	}
	// Bytes past the heap end are tolerated, as the sibling .res reader
	// tolerates its own unread header words.
	heap := data[int(heapStart):int(heapEnd)]

	// The root's own child range is a directory range and is checked by the
	// same rule as every other one. It is easy to miss because the root has no
	// node record, and missing it is a panic rather than an error: the walk
	// below would slice the node table out of range.
	if uint64(rootFirst)+uint64(rootCount) > uint64(nodeCount) {
		return nil, fmt.Errorf("reg: root child range [%d, %d) exceeds node count %d",
			rootFirst, uint64(rootFirst)+uint64(rootCount), nodeCount)
	}

	// The table fits in the stream, so nodeCount <= len(data)/nodeSize and the
	// conversion to int cannot overflow even on a 32-bit build.
	count := int(nodeCount)

	// --- Table pass --------------------------------------------------------
	// Every record is validated and decoded, in index order, orphans included:
	// "an unreferenced node is ignored" is a statement about reachability, not
	// a validation exemption, so a malformed record nothing points at is still
	// a malformed record.
	nodes := make([]Node, count)
	first := make([]uint32, count)
	span := make([]uint32, count)
	for i := 0; i < count; i++ {
		rec := data[headerSize+nodeSize*i : headerSize+nodeSize*i+nodeSize]
		// rec[0x00:0x04], the node's reserved word, is read and ignored.
		d := binary.LittleEndian.Uint32(rec[0x04:0x08])
		sz := binary.LittleEndian.Uint32(rec[0x08:0x0c])
		kind := binary.LittleEndian.Uint32(rec[0x0c:0x10])

		n := &nodes[i]
		n.Name = cutNUL(rec[0x10 : 0x10+nameLen])
		n.Kind = kind
		n.Dir = kind&flagDir != 0
		n.Type = ValueType(kind & typeMask)
		n.index, n.rawD, n.rawSz = i, d, sz

		// Bit 0 decides directory-versus-value first.
		if n.Dir {
			if kind&typeMask != 0 {
				return nil, fmt.Errorf("reg: node %d: directory kind %#x claims value type %d",
					i, kind, kind&typeMask)
			}
			if uint64(d)+uint64(sz) > uint64(nodeCount) {
				return nil, fmt.Errorf("reg: node %d child range [%d, %d) exceeds node count %d",
					i, d, uint64(d)+uint64(sz), nodeCount)
			}
			first[i], span[i] = d, sz
			n.Children = make([]*Node, 0, sz)
			continue
		}

		// A value's type is settled before any type-specific check: a size%4
		// rule is meaningless on a type that is not an int32 array.
		switch n.Type {
		case TypeString:
			if uint64(d)+uint64(sz) > heapSize {
				return nil, heapErr(i, d, sz, heapSize)
			}
			n.Str = cutNUL(heap[int(d) : int(d)+int(sz)])
		case TypeInt:
			n.Int = int32(d)
		case TypeFloat:
			// The eight bytes at node offset 0x04 read as one little-endian
			// binary64: data is the low word, size the high word. No heap.
			n.Float = math.Float64frombits(uint64(sz)<<32 | uint64(d))
		case TypeIntArray:
			if sz%4 != 0 {
				return nil, fmt.Errorf("reg: node %d: int32 array size %d is not a multiple of 4", i, sz)
			}
			if uint64(d)+uint64(sz) > heapSize {
				return nil, heapErr(i, d, sz, heapSize)
			}
			ints := make([]int32, sz/4)
			for k := range ints {
				ints[k] = int32(binary.LittleEndian.Uint32(heap[int(d)+4*k:]))
			}
			n.Ints = ints
		case TypeFloatArray:
			// Decoded, deliberately not implemented: rejected by name rather
			// than silently skipped.
			return nil, fmt.Errorf("reg: node %d: unsupported value type %d", i, uint32(n.Type))
		default:
			return nil, fmt.Errorf("reg: node %d: unrecognised value type %d", i, uint32(n.Type))
		}
	}

	// --- Tree walk ---------------------------------------------------------
	// seen is the single-parent set: entering an index twice is the rejection,
	// and it is what bounds total work to O(nodeCount). A cycle, a
	// self-referencing range and an acyclic shared subtree are all that one
	// rejection, and none of them can loop.
	seen := make([]bool, count)
	var visit func(i, depth int) error
	visit = func(i, depth int) error {
		if depth > maxDepth {
			return fmt.Errorf("reg: node %d nests deeper than the depth guard %d", i, maxDepth)
		}
		if seen[i] {
			return fmt.Errorf("reg: node %d referenced more than once (non-tree registry)", i)
		}
		seen[i] = true
		n := &nodes[i]
		if !n.Dir {
			return nil
		}
		for c := int(first[i]); c < int(first[i])+int(span[i]); c++ {
			if err := visit(c, depth+1); err != nil {
				return err
			}
			n.Children = append(n.Children, &nodes[c])
		}
		return nil
	}

	root := &Node{Kind: rootFlags, Dir: true, Children: make([]*Node, 0, rootCount)}
	for c := int(rootFirst); c < int(rootFirst)+int(rootCount); c++ {
		if err := visit(c, 1); err != nil {
			return nil, err
		}
		root.Children = append(root.Children, &nodes[c])
	}
	return &Reg{Root: root, NodeCount: count}, nil
}

// Size returns how many bytes of data the registry's own framing claims:
// header, node table, the heap-length word and the heap.
//
// IT IS THE FRAMED EXTENT AND NOT len(data). The two differ whenever a registry
// is embedded in a larger stream — the original save's uncompressed tail is one
// such stream, and a further region follows the registry there — and a consumer
// that took the whole slice as the registry would read that region as heap.
// Parse itself tolerates the trailing bytes; this is how a caller learns where
// they begin.
//
// It validates only what it has to read to answer: the signature, and that the
// node table and the heap-length word are inside data. The extent it returns may
// exceed len(data), in which case it reports that rather than a number: a stream
// whose declared heap does not fit is not a stream a split can be taken from.
// Every other rejection ground is Parse's.
func Size(data []byte) (int, error) {
	if len(data) < headerSize {
		return 0, fmt.Errorf("reg: stream too small: %d bytes", len(data))
	}
	if sig := binary.LittleEndian.Uint32(data[0x00:0x04]); sig != signature {
		return 0, fmt.Errorf("reg: bad signature %#08x", sig)
	}
	nodeCount := uint64(binary.LittleEndian.Uint32(data[0x10:0x14]))
	total := uint64(len(data))
	heapOrigin := uint64(headerSize) + uint64(nodeSize)*nodeCount
	if heapOrigin+4 > total {
		return 0, fmt.Errorf("reg: node table of %d nodes plus the heap-length word needs %d bytes, stream has %d",
			nodeCount, heapOrigin+4, total)
	}
	heapSize := uint64(binary.LittleEndian.Uint32(data[int(heapOrigin) : int(heapOrigin)+4]))
	end := heapOrigin + 4 + heapSize
	if end > total {
		return 0, fmt.Errorf("reg: framing declares %d bytes, stream has %d", end, total)
	}
	return int(end), nil
}

// heapErr reports a type-0 or type-6 reference that leaves the heap window.
func heapErr(i int, off, size uint32, heapSize uint64) error {
	return fmt.Errorf("reg: node %d heap reference [%d, %d) outside a %d-byte heap",
		i, off, uint64(off)+uint64(size), heapSize)
}

// cutNUL returns the bytes before the first NUL as a string, or all of them if
// there is none — the cutting rule both a name and a string value follow.
// Neither an unterminated name nor an unterminated string value is a rejection:
// the caller has already bounded the slice, so taking it whole cannot overrun.
// The conversion copies, which is what keeps Parse from retaining data.
func cutNUL(b []byte) string {
	if i := bytes.IndexByte(b, 0x00); i >= 0 {
		b = b[:i]
	}
	return string(b)
}
