package res

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"golang.org/x/text/encoding/charmap"
)

// Layout constants for the ROM1 .res container (little-endian). See the package
// doc and docs/0001-res-archive/spec.md for the byte-level format.
const (
	signature  = 0x31415926 // header[0x00] magic ("&YA1", little-endian)
	headerSize = 0x18       // 24-byte header; data payloads occupy [0x18, regOffset)
	nodeSize   = 0x20       // 32-byte registry node
	typeFile   = 0          // node[0x0C] == 0
	typeDir    = 1          // node[0x0C] == 1
	maxDepth   = 64         // fixed depth guard against a cyclic/deep registry
)

// Entry is one indexed file in an archive. Directories are structure, not
// entries, and are never returned. Path is the normalized key (see Open);
// Offset and Size delimit the payload within the archive's data region.
type Entry struct {
	Path   string
	Offset int64
	Size   int64
}

// Archive is a read-only index over an opened .res archive. It is safe for
// concurrent reads. The zero value is not usable; construct one with Open or
// OpenBytes.
type Archive struct {
	data    []byte
	entries []Entry        // file entries in registry (node) order
	index   map[string]int // normalized path -> position in entries
}

// Open reads and indexes the .res archive at path. The whole file is read into
// memory; payloads are served by positioned indexing (see ReadFile).
func Open(path string) (*Archive, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return OpenBytes(b)
}

// OpenBytes indexes an in-memory .res archive. The byte slice is retained for
// payload reads; callers must not mutate it while the Archive is in use.
// A malformed archive yields a non-nil error and a nil Archive, never a panic.
func OpenBytes(b []byte) (*Archive, error) {
	// Header + registry geometry. The registry is exactly the header's
	// nodeCount records starting at regOffset: its size is that word and never
	// the archive's length, and bytes past the records are not part of the
	// archive. The one thing the length decides is whether those records are
	// present — checked here, before the count is narrowed to int, so a
	// crafted 0x14 can neither drive a read past EOF nor overflow int.
	if len(b) < headerSize {
		return nil, fmt.Errorf("res: archive too small: %d bytes", len(b))
	}
	if sig := binary.LittleEndian.Uint32(b[0x00:0x04]); sig != signature {
		return nil, fmt.Errorf("res: bad signature %#08x", sig)
	}
	total := int64(len(b))
	regOffset := int64(binary.LittleEndian.Uint32(b[0x10:0x14]))
	// int64 throughout: the widest header count, 0xFFFFFFFF records of 32 bytes,
	// is under 2^37, so neither the product nor the sum below can wrap.
	headerNodeCount := int64(binary.LittleEndian.Uint32(b[0x14:0x18]))
	if regOffset < headerSize || regOffset > total {
		return nil, fmt.Errorf("res: regOffset %d outside [%d, %d]", regOffset, headerSize, total)
	}
	nodeBytes := headerNodeCount * nodeSize
	if regOffset+nodeBytes > total {
		return nil, fmt.Errorf("res: registry of %d nodes at %d ends at %d, past the archive's %d bytes",
			headerNodeCount, regOffset, regOffset+nodeBytes, total)
	}
	nodeCount := int(headerNodeCount)

	nodes, err := parseNodes(b, regOffset, nodeCount)
	if err != nil {
		return nil, err
	}
	entries, index, err := indexNodes(nodes)
	if err != nil {
		return nil, err
	}
	return &Archive{data: b, entries: entries, index: index}, nil
}

// indexNodes validates the registry tree and builds its file index. Both the
// in-memory reader and the disk-indexed reader use this one walk.
func indexNodes(nodes []node) ([]Entry, map[string]int, error) {
	nodeCount := len(nodes)
	// Roots are nodes never referenced as a child of any directory (the header
	// root count is a cross-check we deliberately do not consume).
	referenced := make([]bool, nodeCount)
	for i := range nodes {
		if nodes[i].typ == typeDir {
			for c := int(nodes[i].off); c < int(nodes[i].off)+int(nodes[i].size); c++ {
				referenced[c] = true // range already validated <= nodeCount
			}
		}
	}

	// Bounded, total tree walk: a visited-set rejects any node reached twice (a
	// cycle or a shared subtree, bounding total work to O(nodeCount)); the depth
	// guard bounds recursion; a final reachability check rejects unreachable
	// nodes and disjoint cycles.
	paths := make([]string, nodeCount)
	visited := make([]bool, nodeCount)
	visitedCount := 0
	var walk func(i int, prefix string, depth int) error
	walk = func(i int, prefix string, depth int) error {
		if depth > maxDepth {
			return fmt.Errorf("res: registry nesting exceeds depth guard %d", maxDepth)
		}
		if visited[i] {
			return fmt.Errorf("res: node %d referenced more than once (non-tree registry)", i)
		}
		visited[i] = true
		visitedCount++
		full := nodes[i].name
		if prefix != "" {
			full = prefix + "/" + nodes[i].name
		}
		if nodes[i].typ == typeDir {
			for c := int(nodes[i].off); c < int(nodes[i].off)+int(nodes[i].size); c++ {
				if err := walk(c, full, depth+1); err != nil {
					return err
				}
			}
			return nil
		}
		paths[i] = normalize(full)
		return nil
	}
	for i := 0; i < nodeCount; i++ {
		if !referenced[i] {
			if err := walk(i, "", 0); err != nil {
				return nil, nil, err
			}
		}
	}
	if visitedCount != nodeCount {
		return nil, nil, fmt.Errorf("res: %d of %d nodes unreachable from any root", nodeCount-visitedCount, nodeCount)
	}

	// Build the file index in registry (node) order.
	entries := make([]Entry, 0, nodeCount)
	index := make(map[string]int)
	for i := range nodes {
		if nodes[i].typ != typeFile {
			continue
		}
		e := Entry{Path: paths[i], Offset: int64(nodes[i].off), Size: int64(nodes[i].size)}
		if _, ok := index[e.Path]; !ok {
			index[e.Path] = len(entries)
		}
		entries = append(entries, e)
	}
	return entries, index, nil
}

// node is a decoded 32-byte registry record. The opaque word at 0x00 is read and
// discarded (spec R-1). For a file, off/size are the payload byte range; for a
// directory, off is the first child's node index and size the child count.
type node struct {
	off  uint32
	size uint32
	typ  uint32
	name string
}

// parseNodes decodes and validates every registry record. File payload ranges
// must lie within the data region [headerSize, regOffset); directory child
// ranges must lie within [0, nodeCount); the type must be file or directory.
// All range arithmetic is done in 64-bit to defeat 32-bit overflow.
func parseNodes(b []byte, regOffset int64, nodeCount int) ([]node, error) {
	return parseNodeRecords(b[regOffset:regOffset+int64(nodeCount)*nodeSize], regOffset, int64(len(b)), nodeCount)
}

func parseNodeRecords(records []byte, regOffset, total int64, nodeCount int) ([]node, error) {
	nodes := make([]node, nodeCount)
	for i := 0; i < nodeCount; i++ {
		rec := records[int64(i)*nodeSize : int64(i)*nodeSize+nodeSize]
		off := binary.LittleEndian.Uint32(rec[0x04:0x08])
		size := binary.LittleEndian.Uint32(rec[0x08:0x0C])
		typ := binary.LittleEndian.Uint32(rec[0x0C:0x10])
		switch typ {
		case typeFile:
			if int64(off) < headerSize || int64(off)+int64(size) > regOffset || int64(off)+int64(size) > total {
				return nil, fmt.Errorf("res: node %d payload [%d, %d) outside data region [%d, %d)",
					i, off, int64(off)+int64(size), headerSize, regOffset)
			}
		case typeDir:
			if int64(off)+int64(size) > int64(nodeCount) {
				return nil, fmt.Errorf("res: node %d child range [%d, %d) exceeds node count %d",
					i, off, int64(off)+int64(size), nodeCount)
			}
		default:
			return nil, fmt.Errorf("res: node %d has unknown type %d", i, typ)
		}
		nodes[i] = node{off: off, size: size, typ: typ, name: decodeName(rec[0x10:0x20])}
	}
	return nodes, nil
}

// Entries returns the archive's file entries in registry (node) order. The
// returned slice is a fresh copy the caller may retain or modify freely.
func (a *Archive) Entries() []Entry {
	out := make([]Entry, len(a.entries))
	copy(out, a.entries)
	return out
}

// ReadFile returns a copy of the bytes of the entry at name. Lookup is
// case-insensitive and accepts '/' or '\' separators. A path with no matching
// entry yields an error satisfying errors.Is(err, fs.ErrNotExist).
func (a *Archive) ReadFile(name string) ([]byte, error) {
	i, ok := a.index[normalize(name)]
	if !ok {
		return nil, &fs.PathError{Op: "readfile", Path: name, Err: fs.ErrNotExist}
	}
	e := a.entries[i]
	out := make([]byte, e.Size)
	copy(out, a.data[e.Offset:e.Offset+e.Size])
	return out, nil
}

// decodeName decodes a 16-byte node name: bytes up to the first NUL, interpreted
// as CP866 (the 0xCD padding lives past the NUL and is dropped). CP866 is a
// total single-byte map, so decoding cannot fail.
func decodeName(raw []byte) string {
	if i := bytes.IndexByte(raw, 0x00); i >= 0 {
		raw = raw[:i]
	}
	return DecodeCP866(raw)
}

// DecodeCP866 is this archive's code page turned into a Go string.
//
// IT IS EXPORTED FOR CONSUMERS THAT REPORT AN ARCHIVE'S CONTENT rather than draw
// it: a tier above this one that has to put an entry's words into JSON, or any
// other UTF-8 sink, needs the same map the names are read with, and the x/text
// grant that supplies it belongs to this tier alone.
//
// THE DRAW PATH MUST NOT USE IT. A renderer built for the game's own byte string
// receives the bytes unconverted, and converting first would corrupt every
// non-ASCII byte on the way there.
//
// CP866 is a total single-byte map, so decoding cannot fail; the error arm
// returns the bytes unchanged rather than inventing a replacement.
func DecodeCP866(raw []byte) string {
	s, err := charmap.CodePage866.NewDecoder().Bytes(raw)
	if err != nil {
		return string(raw)
	}
	return string(s)
}

// foldASCII maps 'A'-'Z' to 'a'-'z' and leaves every other byte, ASCII or
// not, unchanged.
//
// This is written by hand over bytes rather than delegated to
// strings.ToLower/EqualFold. Entry names decode as CP866 (see decodeName),
// so a byte >= 0x80 becomes a Cyrillic (or other) rune, and a Unicode fold
// would collapse pairs the original engine keeps distinct; strings.EqualFold
// has that same defect plus two more (the Kelvin sign U+212A folds to 'k',
// the long s U+017F to 's'). The contract folds ASCII A-Z alone and nothing
// else. See pkg/formats/reg/lookup.go for the identical idiom, applied there
// to a byte-wise comparison instead of a folded string.
func foldASCII(b byte) byte {
	if b >= 'A' && b <= 'Z' {
		return b - 'A' + 'a'
	}
	return b
}

// normalize maps a lookup path or an archive path to the canonical index key:
// '\' becomes '/', leading/trailing '/' are trimmed, and case is folded byte
// by byte via foldASCII — only 'A'-'Z' changes; every other byte, including
// every byte of a multi-byte UTF-8 sequence decoded from a high CP866 byte,
// compares as itself.
func normalize(p string) string {
	p = strings.ReplaceAll(p, "\\", "/")
	p = strings.Trim(p, "/")
	b := []byte(p)
	for i, c := range b {
		b[i] = foldASCII(c)
	}
	return string(b)
}
