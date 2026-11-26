// Package a2c reads and writes the second game's character file: the
// container magic and its sections, each with a header, a scramble and a
// checksum (R2-SESSION-022, R2-SESSION-512, R2-SESSION-513). ReadIdentity
// is the character list's read of section A alone (R2-ENGINE-1840).
package a2c

import (
	"bytes"
	"encoding/binary"
	"errors"
)

const (
	// Magic is the file's first dword.
	Magic = 0x04507989
	// TagA is section A's tag; A is the first section after the magic.
	TagA = 0xAAAAAAAA
	// HeaderSize is a section header: tag, length, zero, key, checksum.
	HeaderSize = 16
	// NameStart and NameEnd bound section A's NUL-terminated name.
	NameStart, NameEnd = 0x0c, 0x2c
	// MaxBody bounds the section length, and B's decoded length, a reader
	// accepts.
	MaxBody = 0x10000
)

// The other section tags (R2-SESSION-022).
const (
	TagB = 0x55555555
	TagC = 0x40A40A40
	TagD = 0xDE0DE0DE
	TagE = 0x41392521
	TagF = 0x3A5A3A5A
)

const (
	// ASize is section A's length (R2-SESSION-513).
	ASize = 0x34
	// ESize is section E's decoded field bytes (R2-SESSION-517).
	ESize = 0x34
	// KillsSize is section B's decoded length (R2-SESSION-022).
	KillsSize = 2560
	// HotkeysSize is section C with nine entries of no bytes (R2-SESSION-515).
	HotkeysSize = 72
	// ListKeyAny is an A +8 value the list replaces by the application's
	// own, so the list keeps the file under any value (R2-ENGINE-1840).
	ListKeyAny = 0x529c0291
)

// StartRows are the unit rows section A +0x2c bits 6 and 7 pick, in bit
// order (R2-SESSION-514).
var StartRows = [4]string{"Start_MF", "Start_MM", "Start_FF", "Start_FM"}

// Identity is what section A gives the character list: the identity pair the
// file name repeats and the name's bytes in the install's code page.
type Identity struct {
	ID0, ID1 uint32
	Name     []byte
}

// A is section A's fields (R2-SESSION-514).
type A struct {
	ID0, ID1 uint32
	// ListKey is A +8, the value the list compares.
	ListKey uint32
	// Name is the name's bytes, at most 31, with no NUL.
	Name []byte
	// Unit is A +0x2c; bits 6 and 7 pick the start unit row.
	Unit byte
	// Rest is A +0x2d..+0x33, kept whole.
	Rest [7]byte
}

// E is section E's decoded fields (R2-SESSION-517; names Medium).
type E struct {
	Player [4]uint32
	Money  uint32
	// Base is the four base statistics: Body, Reaction, Mind, Spirit.
	Base    [4]uint8
	Spells  uint32
	Unit1C  uint32
	SkillXP [5]uint32
}

// Character is a file's sections. B is held decoded; C, D and F are their
// plain bodies. A nil slice is a section the file lacks; A and E are
// required, as the client loader requires them (R2-ENGINE-1852).
type Character struct {
	A       A
	Kills   []byte
	Hotkeys []byte
	Worn    []byte
	E       E
	Bag     []byte
}

var (
	ErrMagic    = errors.New("a2c: not a character file")
	ErrSection  = errors.New("a2c: section A header missing or out of bounds")
	ErrChecksum = errors.New("a2c: section A checksum mismatch")
	ErrBody     = errors.New("a2c: a section is out of bounds or fails its checksum")
	ErrTag      = errors.New("a2c: unknown section tag")
	ErrMissing  = errors.New("a2c: section A or E missing")
	ErrName     = errors.New("a2c: the name does not fit section A")
)

// ReadIdentity checks the magic, section A's tag and length against the
// file's size, descrambles A and checks its checksum, then answers its
// identity and name.
func ReadIdentity(b []byte) (Identity, error) {
	if len(b) < 4 || binary.LittleEndian.Uint32(b) != Magic {
		return Identity{}, ErrMagic
	}
	h := b[4:]
	if len(h) < HeaderSize || binary.LittleEndian.Uint32(h) != TagA {
		return Identity{}, ErrSection
	}
	n := binary.LittleEndian.Uint32(h[4:])
	if n < NameEnd || n > MaxBody || uint64(n) > uint64(len(h)-HeaderSize) {
		return Identity{}, ErrSection
	}
	key := binary.LittleEndian.Uint16(h[10:])
	body := Descramble(h[HeaderSize:HeaderSize+int(n)], key)
	if Checksum(body) != binary.LittleEndian.Uint32(h[12:]) {
		return Identity{}, ErrChecksum
	}
	return Identity{
		ID0:  binary.LittleEndian.Uint32(body),
		ID1:  binary.LittleEndian.Uint32(body[4:]),
		Name: cutName(body[NameStart:NameEnd]),
	}, nil
}

func cutName(field []byte) []byte {
	if i := bytes.IndexByte(field, 0); i >= 0 {
		field = field[:i]
	}
	return append([]byte(nil), field...)
}

// Read walks every section in any order, as the reader does: an unknown tag
// or a failed checksum refuses the file, and a repeated tag replaces the
// earlier section (R2-SESSION-512). Every length is checked against the file.
func Read(b []byte) (Character, error) {
	var c Character
	if len(b) < 4 || binary.LittleEndian.Uint32(b) != Magic {
		return c, ErrMagic
	}
	var haveA, haveE bool
	for at := 4; at < len(b); {
		if len(b)-at < HeaderSize {
			return c, ErrBody
		}
		h := b[at:]
		tag, n := binary.LittleEndian.Uint32(h), binary.LittleEndian.Uint32(h[4:])
		key, sum := binary.LittleEndian.Uint16(h[10:]), binary.LittleEndian.Uint32(h[12:])
		at += HeaderSize
		if tag == TagE {
			e, used, err := readE(b[at:], key, sum)
			if err != nil {
				return c, err
			}
			c.E, haveE = e, true
			at += used
			continue
		}
		if n > MaxBody || uint64(n) > uint64(len(b)-at) {
			return c, ErrBody
		}
		plain := Descramble(b[at:at+int(n)], key)
		at += int(n)
		if Checksum(plain) != sum {
			return c, ErrBody
		}
		switch tag {
		case TagA:
			if len(plain) != ASize {
				return c, ErrBody
			}
			c.A = readA(plain)
			haveA = true
		case TagB:
			kills, err := runDecode(plain)
			if err != nil {
				return c, err
			}
			c.Kills = kills
		case TagC:
			c.Hotkeys = plain
		case TagD:
			c.Worn = plain
		case TagF:
			c.Bag = plain
		default:
			return c, ErrTag
		}
	}
	if !haveA || !haveE {
		return c, ErrMissing
	}
	return c, nil
}

func readA(p []byte) A {
	a := A{
		ID0:     binary.LittleEndian.Uint32(p),
		ID1:     binary.LittleEndian.Uint32(p[4:]),
		ListKey: binary.LittleEndian.Uint32(p[8:]),
		Name:    cutName(p[NameStart:NameEnd]),
		Unit:    p[NameEnd],
	}
	copy(a.Rest[:], p[NameEnd+1:])
	return a
}

// Write is the file of c with the sections in the writer's order A, B, C, D,
// E, F, each present one under its own key (R2-SESSION-513). A key is the
// low 15 bits of the section's checksum and every E pad byte is zero, where
// the writer draws both at random (DIV-5241).
func Write(c Character) ([]byte, error) {
	a, err := writeA(c.A)
	if err != nil {
		return nil, err
	}
	out := binary.LittleEndian.AppendUint32(nil, Magic)
	out = appendSection(out, TagA, a)
	if c.Kills != nil {
		if len(c.Kills) > MaxBody {
			return nil, ErrBody
		}
		out = appendSection(out, TagB, runEncode(c.Kills))
	}
	for _, s := range []struct {
		tag  uint32
		body []byte
	}{{TagC, c.Hotkeys}, {TagD, c.Worn}} {
		if s.body != nil {
			out = appendSection(out, s.tag, s.body)
		}
	}
	out = appendE(out, c.E)
	if c.Bag != nil {
		out = appendSection(out, TagF, c.Bag)
	}
	return out, nil
}

func writeA(a A) ([]byte, error) {
	if len(a.Name) >= NameEnd-NameStart || bytes.IndexByte(a.Name, 0) >= 0 {
		return nil, ErrName
	}
	p := make([]byte, ASize)
	binary.LittleEndian.PutUint32(p, a.ID0)
	binary.LittleEndian.PutUint32(p[4:], a.ID1)
	binary.LittleEndian.PutUint32(p[8:], a.ListKey)
	copy(p[NameStart:], a.Name)
	p[NameEnd] = a.Unit
	copy(p[NameEnd+1:], a.Rest[:])
	return p, nil
}

// sectionKey is the key a plain body is stored under.
func sectionKey(plain []byte) uint16 { return uint16(Checksum(plain) & 0x7fff) }

func appendSection(out []byte, tag uint32, plain []byte) []byte {
	key := sectionKey(plain)
	out = binary.LittleEndian.AppendUint32(out, tag)
	out = binary.LittleEndian.AppendUint32(out, uint32(len(plain)))
	out = binary.LittleEndian.AppendUint16(out, 0)
	out = binary.LittleEndian.AppendUint16(out, key)
	out = binary.LittleEndian.AppendUint32(out, Checksum(plain))
	return append(out, Descramble(plain, key)...)
}

// Descramble answers the plain bytes of a stored section body: byte i is
// XORed with bits 16..23 of a register that starts as key | key<<16, shifts
// left one bit per byte and has key ORed back in after every 16th byte. The
// XOR makes it its own inverse, so it also scrambles a plain body.
func Descramble(stored []byte, key uint16) []byte {
	out := make([]byte, len(stored))
	r := uint32(key) | uint32(key)<<16
	for i, c := range stored {
		out[i] = c ^ byte(r>>16)
		r <<= 1
		if (i+1)%16 == 0 {
			r |= uint32(key)
		}
	}
	return out
}

// Checksum is c = c*2 + byte over the plain bytes, 32-bit.
func Checksum(plain []byte) uint32 {
	var c uint32
	for _, b := range plain {
		c = c*2 + uint32(b)
	}
	return c
}

// runDecode is section B's run code: u32 decoded length, then packets; an
// opcode below 0x80 copies that many bytes, any other repeats the next byte
// opcode & 0x7f times (R2-SESSION-022). The decoded length must match.
func runDecode(p []byte) ([]byte, error) {
	if len(p) < 4 {
		return nil, ErrBody
	}
	n := binary.LittleEndian.Uint32(p)
	if n > MaxBody {
		return nil, ErrBody
	}
	out := make([]byte, 0, n)
	for i := 4; i < len(p); {
		op := int(p[i])
		i++
		if op < 0x80 {
			if len(p)-i < op || len(out)+op > int(n) {
				return nil, ErrBody
			}
			out = append(out, p[i:i+op]...)
			i += op
			continue
		}
		if i >= len(p) || len(out)+(op&0x7f) > int(n) {
			return nil, ErrBody
		}
		out = append(out, bytes.Repeat(p[i:i+1], op&0x7f)...)
		i++
	}
	if len(out) != int(n) {
		return nil, ErrBody
	}
	return out, nil
}

// runEncode is runDecode's input for plain: a byte repeated twice or more is
// one repeat packet of up to 127, every other stretch a literal packet of up
// to 127. The writer's own split is unread (DIV-5241).
func runEncode(plain []byte) []byte {
	out := binary.LittleEndian.AppendUint32(nil, uint32(len(plain)))
	run := func(i int) int {
		n := 1
		for i+n < len(plain) && plain[i+n] == plain[i] && n < 0x7f {
			n++
		}
		return n
	}
	for i := 0; i < len(plain); {
		if n := run(i); n >= 2 {
			out = append(out, 0x80|byte(n), plain[i])
			i += n
			continue
		}
		j := i + 1
		for j < len(plain) && j-i < 0x7f && run(j) < 2 {
			j++
		}
		out = append(out, byte(j-i))
		out = append(out, plain[i:j]...)
		i = j
	}
	return out
}
