// Package a2c reads the second game's character file: the container magic,
// section A's header and body, its descramble and its checksum, and the
// identity and name section A holds (R2-SESSION-022, R2-ENGINE-1840).
package a2c

import (
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
	// MaxBody bounds the section A length a reader accepts.
	MaxBody = 0x10000
)

// Identity is what section A gives the character list: the identity pair the
// file name repeats and the name's bytes in the install's code page.
type Identity struct {
	ID0, ID1 uint32
	Name     []byte
}

var (
	ErrMagic    = errors.New("a2c: not a character file")
	ErrSection  = errors.New("a2c: section A header missing or out of bounds")
	ErrChecksum = errors.New("a2c: section A checksum mismatch")
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
	name := body[NameStart:NameEnd]
	for i, c := range name {
		if c == 0 {
			name = name[:i]
			break
		}
	}
	return Identity{
		ID0:  binary.LittleEndian.Uint32(body),
		ID1:  binary.LittleEndian.Uint32(body[4:]),
		Name: append([]byte(nil), name...),
	}, nil
}

// Descramble answers the plain bytes of a stored section body: byte i is
// XORed with bits 16..23 of a register that starts as key | key<<16, shifts
// left one bit per byte and has key ORed back in after every 16th byte.
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
