package synth

import (
	"encoding/binary"
	"sort"
	"strings"
	"unicode/utf16"
)

// ---------------------------------------------------------------------------
// Windows executables carrying resources
// ---------------------------------------------------------------------------

// PEResource is one resource of a synthetic executable.
type PEResource struct {
	Type     uint16 // the resource type: 3 for an icon, 14 for an icon group
	ID       uint16 // the numeric name, used when Name is empty
	Name     string // a string name; named resources list before numeric ones
	Language uint16
	Data     []byte
}

const (
	peDOSHeaderLen = 0x40
	peFileAlign    = 0x200
	peSectionVA    = 0x3000
	peResourceDir  = 2 // the data-directory slot of the resource table
	peHighBit      = 1 << 31
)

// PE assembles a 32-bit Windows executable whose one section, .rsrc, holds the
// resources under a type, name, language directory. The section sits at a
// virtual address that differs from its file offset, so a reader that confuses
// the two reads the wrong bytes. No code and no import table exist: the image is
// a container for resources and nothing runs it.
func PE(resources []PEResource) []byte {
	section := peSection(resources, peSectionVA)
	raw := (len(section) + peFileAlign - 1) / peFileAlign * peFileAlign
	out := make([]byte, peFileAlign+raw)
	le := binary.LittleEndian

	copy(out, "MZ")
	le.PutUint32(out[0x3c:], peDOSHeaderLen)
	copy(out[peDOSHeaderLen:], "PE\x00\x00")

	coff := peDOSHeaderLen + 4
	le.PutUint16(out[coff:], 0x14c)     // Machine: i386
	le.PutUint16(out[coff+2:], 1)       // NumberOfSections
	le.PutUint16(out[coff+16:], 224)    // SizeOfOptionalHeader
	le.PutUint16(out[coff+18:], 0x0102) // executable, 32-bit
	opt := coff + 20
	le.PutUint16(out[opt:], 0x10b)       // PE32
	le.PutUint32(out[opt+28:], 0x400000) // ImageBase
	le.PutUint32(out[opt+32:], 0x1000)   // SectionAlignment
	le.PutUint32(out[opt+36:], peFileAlign)
	le.PutUint16(out[opt+40:], 4)
	le.PutUint16(out[opt+48:], 4)
	le.PutUint32(out[opt+56:], peSectionVA+uint32((len(section)+0xfff)/0x1000*0x1000))
	le.PutUint32(out[opt+60:], peFileAlign)
	le.PutUint16(out[opt+68:], 2) // Subsystem: Windows GUI
	le.PutUint32(out[opt+92:], 16)
	dir := opt + 96 + peResourceDir*8
	le.PutUint32(out[dir:], peSectionVA)
	le.PutUint32(out[dir+4:], uint32(len(section)))

	head := opt + 224
	copy(out[head:], ".rsrc")
	le.PutUint32(out[head+8:], uint32(len(section)))
	le.PutUint32(out[head+12:], peSectionVA)
	le.PutUint32(out[head+16:], uint32(raw))
	le.PutUint32(out[head+20:], peFileAlign)
	le.PutUint32(out[head+36:], 0x40000040) // initialised data, readable
	copy(out[peFileAlign:], section)
	return out
}

// peNode is one entry of the directory tree peSection lays out.
type peNode struct {
	named bool
	name  string
	id    uint16
	kids  []*peNode
	res   *PEResource // set on a language entry, which holds data
}

func (n *peNode) child(named bool, name string, id uint16) *peNode {
	for _, k := range n.kids {
		if k.res == nil && k.named == named && k.name == name && k.id == id {
			return k
		}
	}
	k := &peNode{named: named, name: name, id: id}
	n.kids = append(n.kids, k)
	return k
}

func (n *peNode) sortKids() {
	sort.SliceStable(n.kids, func(i, j int) bool {
		a, b := n.kids[i], n.kids[j]
		if a.named != b.named {
			return a.named
		}
		if a.named {
			return strings.ToUpper(a.name) < strings.ToUpper(b.name)
		}
		return a.id < b.id
	})
	for _, k := range n.kids {
		k.sortKids()
	}
}

// peWriter appends to the resource section, aligning every block it hands out.
type peWriter struct {
	buf []byte
	va  uint32
}

func (w *peWriter) alloc(n, align int) int {
	for len(w.buf)%align != 0 {
		w.buf = append(w.buf, 0)
	}
	off := len(w.buf)
	w.buf = append(w.buf, make([]byte, n)...)
	return off
}

func (w *peWriter) text(s string) int {
	units := utf16.Encode([]rune(s))
	off := w.alloc(2+2*len(units), 2)
	binary.LittleEndian.PutUint16(w.buf[off:], uint16(len(units)))
	for i, u := range units {
		binary.LittleEndian.PutUint16(w.buf[off+2+2*i:], u)
	}
	return off
}

func (w *peWriter) data(r *PEResource) int {
	entry := w.alloc(16, 4)
	blob := w.alloc(len(r.Data), 4)
	copy(w.buf[blob:], r.Data)
	binary.LittleEndian.PutUint32(w.buf[entry:], w.va+uint32(blob))
	binary.LittleEndian.PutUint32(w.buf[entry+4:], uint32(len(r.Data)))
	return entry
}

// dir writes the directory n names and every block below it, and returns the
// directory's offset. Entries are written after the blocks they point at exist,
// so no offset is patched later.
func (w *peWriter) dir(n *peNode) int {
	le := binary.LittleEndian
	named := 0
	for _, k := range n.kids {
		if k.named {
			named++
		}
	}
	off := w.alloc(16+8*len(n.kids), 4)
	le.PutUint16(w.buf[off+12:], uint16(named))
	le.PutUint16(w.buf[off+14:], uint16(len(n.kids)-named))
	for i, k := range n.kids {
		key := uint32(k.id)
		if k.named {
			key = peHighBit | uint32(w.text(k.name))
		}
		var next uint32
		if k.res != nil {
			next = uint32(w.data(k.res))
		} else {
			next = peHighBit | uint32(w.dir(k))
		}
		le.PutUint32(w.buf[off+16+8*i:], key)
		le.PutUint32(w.buf[off+16+8*i+4:], next)
	}
	return off
}

func peSection(resources []PEResource, va uint32) []byte {
	root := &peNode{}
	for i := range resources {
		r := &resources[i]
		byType := root.child(false, "", r.Type)
		byName := byType.child(r.Name != "", r.Name, r.ID)
		byName.kids = append(byName.kids, &peNode{id: r.Language, res: r})
	}
	root.sortKids()
	w := &peWriter{va: va}
	w.dir(root)
	return w.buf
}
