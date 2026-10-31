package res

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"sort"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"

	"againrom/pkg/namefold"
)

// MaxNameBytes is the longest node name: the 16-byte name field holds the
// name and its NUL, and the original's writer and lookup bound a name at 15
// characters (RES-NODE-007, RES-LOOKUP-023).
const MaxNameBytes = 15

// WriteEntry is one node Write stores: Path is a slash path of names; a file
// carries Data, a directory sets Dir. The directories a path passes through
// are made without an entry of their own.
type WriteEntry struct {
	Path string
	Dir  bool
	Data []byte
}

// NameError is a path whose name the format cannot hold.
type NameError struct {
	Path   string
	Reason string
}

func (e *NameError) Error() string { return fmt.Sprintf("%s: %s", e.Path, e.Reason) }

// EncodeName is name as a node stores it. The archive's index decodes a
// name as CP866 (decodeName), so a name is written in CP866: ASCII as
// itself and each Cyrillic letter as one byte. A name that is empty, "." or
// "..", holds a control character or a character CP866 has no byte for, or
// is longer than MaxNameBytes bytes once encoded, is refused.
func EncodeName(name string) ([]byte, error) {
	switch {
	case name == "" || name == "." || name == "..":
		return nil, fmt.Errorf("%q is not a name a node can carry", name)
	case !utf8.ValidString(name):
		return nil, fmt.Errorf("the name is not UTF-8 text")
	case strings.ContainsAny(name, `/\`):
		return nil, fmt.Errorf("a name holds no path separator")
	}
	out := make([]byte, 0, len(name))
	for _, r := range name {
		if r < 0x20 || r == 0x7f {
			return nil, fmt.Errorf("the name holds the control character %U", r)
		}
		b, ok := charmap.CodePage866.EncodeRune(r)
		if !ok {
			return nil, fmt.Errorf("the name holds %q, which the archive's code page (CP866) has no byte for", r)
		}
		out = append(out, b)
	}
	if len(out) > MaxNameBytes {
		return nil, fmt.Errorf("the name %q is %d bytes; a node name is at most %d", name, len(out), MaxNameBytes)
	}
	return out, nil
}

// writeNode is one node being laid out.
type writeNode struct {
	path     string
	name     []byte
	dir      bool
	data     []byte
	children []*writeNode
}

// Write builds a tail-registry archive holding entries, in the format's
// write sequence: the header, the payloads, then the node array. The
// top-level nodes come first in the array and every directory's children
// are one contiguous range, assigned breadth first. Within a directory the
// children stand in the byte order of their ASCII-folded names; the sorted
// flag stays clear, so a lookup scans the children and folds ASCII case
// (RES-041). A name the format cannot hold, two names one folded name
// names, a file below a file, and an archive past the format's 32-bit
// offsets are refused with a NameError or an error naming the path.
func Write(entries []WriteEntry) ([]byte, error) {
	root := &writeNode{dir: true}
	for _, e := range entries {
		parts := strings.Split(e.Path, "/")
		at := root
		for i, part := range parts {
			p := strings.Join(parts[:i+1], "/")
			name, err := EncodeName(part)
			if err != nil {
				return nil, &NameError{Path: p, Reason: err.Error()}
			}
			last := i == len(parts)-1
			var found *writeNode
			for _, c := range at.children {
				if bytes.Equal(namefold.ArchiveBytes(c.name), namefold.ArchiveBytes(name)) {
					found = c
				}
			}
			switch {
			case found != nil && found.path != p:
				return nil, &NameError{Path: p, Reason: fmt.Sprintf("names the same node as %s: names compare without ASCII case", found.path)}
			case found != nil && (last && !e.Dir || !found.dir):
				return nil, &NameError{Path: p, Reason: "is given twice, or as a file and a folder"}
			case found == nil:
				found = &writeNode{path: p, name: name, dir: !last || e.Dir}
				if last && !e.Dir {
					found.data = e.Data
				}
				at.children = append(at.children, found)
			}
			at = found
		}
	}
	order := []*writeNode{}
	first := map[*writeNode]int{}
	queue := []*writeNode{root}
	for len(queue) > 0 {
		n := queue[0]
		queue = queue[1:]
		sort.Slice(n.children, func(i, j int) bool {
			return bytes.Compare(namefold.ArchiveBytes(n.children[i].name), namefold.ArchiveBytes(n.children[j].name)) < 0
		})
		first[n] = len(order)
		order = append(order, n.children...)
		for _, c := range n.children {
			if c.dir {
				queue = append(queue, c)
			}
		}
	}
	var size int64 = headerSize + int64(len(order))*nodeSize
	for _, n := range order {
		size += int64(len(n.data))
	}
	if size > math.MaxUint32 {
		return nil, fmt.Errorf("res: the archive would be %d bytes; the format's offsets reach %d", size, int64(math.MaxUint32))
	}
	out := make([]byte, headerSize, size)
	at := map[*writeNode]uint32{}
	for _, n := range order {
		if !n.dir {
			at[n] = uint32(len(out))
			out = append(out, n.data...)
		}
	}
	le := binary.LittleEndian
	le.PutUint32(out[0x00:], signature)
	le.PutUint32(out[0x04:], uint32(first[root]))
	le.PutUint32(out[0x08:], uint32(len(root.children)))
	le.PutUint32(out[0x0C:], typeDir)
	le.PutUint32(out[0x10:], uint32(len(out)))
	le.PutUint32(out[0x14:], uint32(len(order)))
	for _, n := range order {
		rec := make([]byte, nodeSize)
		if n.dir {
			le.PutUint32(rec[0x04:], uint32(first[n]))
			le.PutUint32(rec[0x08:], uint32(len(n.children)))
			le.PutUint32(rec[0x0C:], typeDir)
		} else {
			le.PutUint32(rec[0x04:], at[n])
			le.PutUint32(rec[0x08:], uint32(len(n.data)))
			le.PutUint32(rec[0x0C:], typeFile)
		}
		copy(rec[0x10:], n.name)
		out = append(out, rec...)
	}
	return out, nil
}
