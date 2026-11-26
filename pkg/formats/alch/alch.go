// Package alch reads and writes the first game's network character file: the
// AlCh tag, the name the character list shows and the fields the writer
// stores in its order (HERO-213, MENU-168).
package alch

import (
	"bytes"
	"encoding/binary"
	"errors"
)

const (
	// Tag is the file's first dword, AlCh.
	Tag = 0x68436c41
	// NameStart and NameSize place the 32-byte name.
	NameStart, NameSize = 0x0c, 32
	// HeaderSize is the part every write holds: tag, 8 bytes, the name, 32
	// bytes, the flags, 4 bytes and two 28-byte blocks; a file whose flag bit
	// 0 is set ends there.
	HeaderSize = 0x8c
	// FlagShort is flag bit 0: the writer stops after the header.
	FlagShort = 1
)

// Header is what the character list reads: the name's bytes in the install's
// code page, cut at the first NUL.
type Header struct {
	Name []byte
}

// Character is a file's fields in the writer's order. Stats are the four
// creation statistics that open the first 28-byte block; the other spans
// keep their bytes, their meanings unread (HERO-213, Medium).
type Character struct {
	Store08 [8]byte
	// Name is the name's bytes, at most NameSize-1, with no NUL.
	Name    []byte
	Store30 [32]byte
	Flags   uint32
	Store74 uint32
	Stats   [4]uint32
	// Store90 is the rest of the first 28-byte block; its dword at 8 is file
	// +0x6c, the value the list compares (MENU-168).
	Store90 [12]byte
	StoreAC [28]byte
	// Long is every byte after the header, kept whole.
	Long []byte
}

var (
	ErrTag  = errors.New("alch: not a character file")
	ErrSize = errors.New("alch: shorter than the character header")
	ErrName = errors.New("alch: the name does not fit its 32-byte field")
)

// ReadHeader checks the tag and the header size and answers the name.
func ReadHeader(b []byte) (Header, error) {
	c, err := Read(b)
	return Header{Name: c.Name}, err
}

// Read checks the tag and the header size and answers the file's fields.
func Read(b []byte) (Character, error) {
	if len(b) < 4 || binary.LittleEndian.Uint32(b) != Tag {
		return Character{}, ErrTag
	}
	if len(b) < HeaderSize {
		return Character{}, ErrSize
	}
	var c Character
	copy(c.Store08[:], b[4:])
	name := b[NameStart : NameStart+NameSize]
	if i := bytes.IndexByte(name, 0); i >= 0 {
		name = name[:i]
	}
	c.Name = append([]byte(nil), name...)
	copy(c.Store30[:], b[0x2c:])
	c.Flags = binary.LittleEndian.Uint32(b[0x4c:])
	c.Store74 = binary.LittleEndian.Uint32(b[0x50:])
	for i := range c.Stats {
		c.Stats[i] = binary.LittleEndian.Uint32(b[0x54+4*i:])
	}
	copy(c.Store90[:], b[0x64:])
	copy(c.StoreAC[:], b[0x70:])
	if len(b) > HeaderSize {
		c.Long = append([]byte(nil), b[HeaderSize:]...)
	}
	return c, nil
}

// Write is the file of c: the header in the writer's order, then Long.
func Write(c Character) ([]byte, error) {
	if len(c.Name) >= NameSize || bytes.IndexByte(c.Name, 0) >= 0 {
		return nil, ErrName
	}
	b := binary.LittleEndian.AppendUint32(make([]byte, 0, HeaderSize+len(c.Long)), Tag)
	b = append(b, c.Store08[:]...)
	b = append(b, c.Name...)
	b = append(b, make([]byte, NameSize-len(c.Name))...)
	b = append(b, c.Store30[:]...)
	b = binary.LittleEndian.AppendUint32(b, c.Flags)
	b = binary.LittleEndian.AppendUint32(b, c.Store74)
	for _, s := range c.Stats {
		b = binary.LittleEndian.AppendUint32(b, s)
	}
	b = append(b, c.Store90[:]...)
	b = append(b, c.StoreAC[:]...)
	return append(b, c.Long...), nil
}
