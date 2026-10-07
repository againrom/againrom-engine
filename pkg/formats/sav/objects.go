package sav

import "fmt"

// Object is one instance the stream carries, decoded through its class's
// programme.
type Object struct {
	// Class is the class the stream introduced it as.
	Class string

	// Off is the body offset of the record's first byte — the head, for a
	// class that has one.
	Off int

	// Fields is the whole record as its programme read it.
	Fields Fields
}

// ClassRecord is one class introduction in the stream.
type ClassRecord struct {
	// Name is the class name, Schema its schema word, and Off the offset of
	// the FF FF that introduces it. First is where the first instance's own
	// bytes begin, immediately after the record.
	Name   string
	Schema uint16
	Off    int
	First  int
}

// maxClassName bounds a class-introduction name so that a coincidental FF FF
// cannot ask for an unbounded read. The longest name the image carries is
// fourteen characters.
const maxClassName = 32

// ClassRecords lists every class introduction in the stream, in stream order.
//
// It is a scan for the FF FF marker with the name validated as printable ASCII,
// which is what makes it a location rather than a guess: a coincidental FF FF is
// followed by a length and that many bytes that are almost never all letters.
//
// A NAME THIS PACKAGE HAS NO PROGRAMME FOR IS STILL A CLASS RECORD. The image
// carries twenty-eight serializable classes and only eleven have been read, so
// meeting an unknown one is the normal case and not an error — Lookup degrades
// and the caller sees a class it cannot walk, which it already handles.
func (f *File) ClassRecords() []ClassRecord {
	b := f.Body
	var out []ClassRecord
	for p := f.Head.End; p+6 <= len(b); p++ {
		if u16(b, p) != classIntro {
			continue
		}
		n := int(u16(b, p+4))
		if n == 0 || n > maxClassName || p+6+n > len(b) {
			continue
		}
		name := b[p+6 : p+6+n]
		ok := true
		for _, c := range name {
			if c < 'A' || (c > 'Z' && c < 'a') || c > 'z' {
				ok = false
				break
			}
		}
		if !ok {
			continue
		}
		out = append(out, ClassRecord{
			Name:   string(name),
			Schema: u16(b, p+2),
			Off:    p,
			First:  p + 6 + n,
		})
		p += 5 + n
	}
	return out
}

// instanceTag is the u16 that introduces a later instance of a class already
// introduced: the high bit set, and the class's index in the shared counter in
// the low bits.
const instanceTagBit = 0x8000

// Chain decodes every consecutive instance of a FIXED-LENGTH class, starting at
// its class record.
//
// This is a WALK and not a scan, and it is available for exactly one class
// today: a class whose record is a constant length, written consecutively, so
// that stepping its length lands on the next instance's tag. Building is that
// class — 77 bytes, no count, no string and no branch.
//
// The instance tag is LEARNED from the first step rather than computed. The
// index a class takes in the stream's shared counter depends on how many objects
// and classes preceded it, which is exactly what this package cannot count while
// most classes have no programme; but the tag is the same word at every step, so
// reading it once and then requiring it is a stronger check than computing it
// would be — it must hold at every instance or the chain stops.
//
// The chain ends on a 0x0000 word, which is the null arm of an object reference,
// and that terminator is what says the walk consumed the run exactly rather than
// stopping where it ran out of patience.
//
// IT RETURNS WHAT IT DECODED ALONGSIDE ITS ERROR. A run that breaks at its second
// instance still decoded its first, and that first record is evidence about the
// class's programme which a caller throwing the slice away would lose — it is how
// "this class's record length is wrong" is told apart from "this class is not
// written consecutively", which are different findings.
func (f *File) Chain(name string) ([]Object, error) {
	c := Lookup(name)
	size, ok := c.Fixed()
	if !ok {
		return nil, fmt.Errorf("sav: %s has no fixed record length, so it cannot be chained", name)
	}
	// A FIXED LENGTH IS NOT ENOUGH. `Effect` is exactly 44 bytes and its
	// instances are elements of another object's counted list, so stepping the
	// length lands back inside the enclosing record. Refusing here rather than
	// discovering it at the second instance is the difference between a class
	// this package knows is not chainable and one it tried and failed on.
	if !c.Consecutive {
		return nil, fmt.Errorf("sav: %s instances are not written consecutively, "+
			"so a chain cannot step from one to the next", name)
	}
	var rec *ClassRecord
	for _, r := range f.ClassRecords() {
		if r.Name == name {
			rec = &r
			break
		}
	}
	if rec == nil {
		return nil, fmt.Errorf("sav: this save introduces no %s", name)
	}
	b := f.Body
	var out []Object
	p := rec.First
	tag := uint16(0)
	for {
		if p+size > len(b) {
			return out, fmt.Errorf("sav: %s instance %d at %d overruns the stream",
				name, len(out), p)
		}
		fields, err := c.decode(b, p)
		if err != nil {
			return out, err
		}
		out = append(out, Object{Class: name, Off: p, Fields: fields})
		p += size
		if p+2 > len(b) {
			return out, fmt.Errorf("sav: the %s run ends with no terminator", name)
		}
		next := u16(b, p)
		if next == 0 {
			return out, nil
		}
		if tag == 0 {
			if next&instanceTagBit == 0 {
				return out, fmt.Errorf("sav: %s+%d is %#04x, not an instance tag", name, size, next)
			}
			tag = next
		}
		if next != tag {
			return out, fmt.Errorf("sav: %s instance %d is tagged %#04x, want %#04x",
				name, len(out), next, tag)
		}
		p += 2
	}
}

// Reencode rebuilds one object's record FROM ITS DECODED FORM and reports
// whether that reproduces the bytes it was read from.
//
// This is the check a byte-carrying round trip cannot make. Marshal proves that
// nothing was disturbed; this proves that what was read was UNDERSTOOD — a field
// read at the wrong width or into the wrong variable survives the first and
// fails here.
func (f *File) Reencode(o Object) (ok bool, at int, err error) {
	c := Lookup(o.Class)
	got, err := c.encode(o.Fields)
	if err != nil {
		return false, 0, err
	}
	want := f.Body[o.Fields.Start:o.Fields.End]
	if len(got) != len(want) {
		return false, 0, fmt.Errorf("sav: %s re-encodes to %d bytes, read %d",
			o.Class, len(got), len(want))
	}
	for i := range got {
		if got[i] != want[i] {
			return false, o.Fields.Start + i, nil
		}
	}
	return true, 0, nil
}
