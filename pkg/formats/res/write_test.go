package res_test

import (
	"bytes"
	"encoding/binary"
	"errors"
	"strings"
	"testing"

	"againrom/pkg/formats/res"
)

// A one-file archive is pinned byte for byte against the documented layout:
// the 24-byte header whose first 16 bytes are the root's node shape (first
// child 0, one child, directory type with the sorted flag clear), the
// payload at 0x18, then one 32-byte node with its NUL-terminated name.
func TestWriteLaysOutTheDocumentedFormat(t *testing.T) {
	got, err := res.Write([]res.WriteEntry{{Path: "a.txt", Data: []byte("hi")}})
	if err != nil {
		t.Fatal(err)
	}
	want := []byte{
		0x26, 0x59, 0x41, 0x31, // magic 0x31415926
		0x00, 0x00, 0x00, 0x00, // root: first child
		0x01, 0x00, 0x00, 0x00, // root: child count
		0x01, 0x00, 0x00, 0x00, // root: directory, children unsorted
		0x1A, 0x00, 0x00, 0x00, // registry offset 0x1A
		0x01, 0x00, 0x00, 0x00, // one node
		'h', 'i',
		0x00, 0x00, 0x00, 0x00, // reserved
		0x18, 0x00, 0x00, 0x00, // payload offset
		0x02, 0x00, 0x00, 0x00, // payload size
		0x00, 0x00, 0x00, 0x00, // file
		'a', '.', 't', 'x', 't', 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("archive\n% x\nwant\n% x", got, want)
	}
}

// What Write stores, the reader reads back: every file at its path, the
// names with their case, empty folders, each folder's children in one
// contiguous range, and no folder flagged sorted.
func TestWriteReadsBackThroughTheReader(t *testing.T) {
	in := []res.WriteEntry{
		{Path: "mod.toml", Data: []byte("id = \"x\"\n")},
		{Path: "files/common/graphics/Menu.bmp", Data: []byte("BM..")},
		{Path: "files/ru", Dir: true},
		{Path: "data/items.toml", Data: []byte("[[item]]\n")},
		{Path: "text/карта.txt", Data: []byte("map")},
		{Path: "empty.txt"},
	}
	b, err := res.Write(in)
	if err != nil {
		t.Fatal(err)
	}
	a, err := res.OpenBytes(b)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range in {
		if e.Dir {
			continue
		}
		got, err := a.ReadFile(e.Path)
		if err != nil || !bytes.Equal(got, e.Data) {
			t.Errorf("%s: %q, %v", e.Path, got, err)
		}
	}
	nodes, roots := a.Tree()
	paths := map[string]res.Node{}
	var walk func(prefix string, kids []int)
	walk = func(prefix string, kids []int) {
		for k, i := range kids {
			if k > 0 && i != kids[k-1]+1 {
				t.Errorf("%s: children %v are not one range", prefix, kids)
			}
			n := nodes[i]
			paths[prefix+n.Name] = n
			if n.Dir {
				walk(prefix+n.Name+"/", n.Children)
			}
		}
	}
	walk("", roots)
	for _, p := range []string{"files/common/graphics/Menu.bmp", "text/карта.txt", "empty.txt"} {
		if n, ok := paths[p]; !ok || n.Dir {
			t.Errorf("%s: not a file node of the tree", p)
		}
	}
	if n, ok := paths["files/ru"]; !ok || !n.Dir || len(n.Children) != 0 {
		t.Errorf("files/ru: %+v %v, want an empty folder", n, ok)
	}
	if roots[0] != 0 || int(binary.LittleEndian.Uint32(b[0x08:])) != len(roots) {
		t.Errorf("roots %v, header count %d", roots, binary.LittleEndian.Uint32(b[0x08:]))
	}
	reg := int(binary.LittleEndian.Uint32(b[0x10:]))
	for i := 0; i < len(nodes); i++ {
		if kind := binary.LittleEndian.Uint32(b[reg+32*i+0x0C:]); kind&0x10 != 0 {
			t.Errorf("node %d carries the sorted flag", i)
		}
	}
	// A Cyrillic letter is one CP866 byte in the name field.
	at := bytes.Index(b[reg:], []byte{0xAA, 0xA0, 0xE0, 0xE2, 0xA0, '.', 't', 'x', 't', 0})
	if at < 0 {
		t.Error("карта.txt is not stored in CP866")
	}
}

// A name over 15 bytes, a character CP866 has no byte for, a control
// character, two names that differ only by ASCII case and a file given as a
// folder's name are refused, each naming the path.
func TestWriteRefusesWhatTheFormatCannotHold(t *testing.T) {
	long := strings.Repeat("a", res.MaxNameBytes+1)
	for _, c := range []struct {
		name    string
		entries []res.WriteEntry
		path    string
		reason  string
	}{
		{"long file", []res.WriteEntry{{Path: "data/" + long}}, "data/" + long, "is 16 bytes; a node name is at most 15"},
		{"long folder", []res.WriteEntry{{Path: long + "/a.txt"}}, long, "is 16 bytes"},
		{"Cyrillic over the bound", []res.WriteEntry{{Path: "ааааааааааааааа.txt"}}, "ааааааааааааааа.txt", "is 19 bytes"},
		{"no CP866 byte", []res.WriteEntry{{Path: "text/日本.txt"}}, "text/日本.txt", "has no byte for"},
		{"control", []res.WriteEntry{{Path: "a\tb"}}, "a\tb", "control character"},
		{"case twin", []res.WriteEntry{{Path: "Data/a.txt"}, {Path: "data/b.txt"}}, "data", "names the same node as Data"},
		{"file and folder", []res.WriteEntry{{Path: "a"}, {Path: "a/b"}}, "a", "as a file and a folder"},
		{"dot dot", []res.WriteEntry{{Path: "../a"}}, "..", "not a name"},
	} {
		_, err := res.Write(c.entries)
		var ne *res.NameError
		if !errors.As(err, &ne) || ne.Path != c.path || !strings.Contains(ne.Reason, c.reason) {
			t.Errorf("%s: %v, want a refusal of %q that says %q", c.name, err, c.path, c.reason)
		}
	}
	if _, err := res.EncodeName(strings.Repeat("я", res.MaxNameBytes)); err != nil {
		t.Errorf("15 Cyrillic letters are 15 bytes: %v", err)
	}
}
