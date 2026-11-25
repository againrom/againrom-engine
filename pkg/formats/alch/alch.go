// Package alch reads the first game's network character file: the AlCh tag
// and the name the character list shows (HERO-213, MENU-168).
package alch

import (
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
)

// Header is what the character list reads: the name's bytes in the install's
// code page, cut at the first NUL.
type Header struct {
	Name []byte
}

var (
	ErrTag  = errors.New("alch: not a character file")
	ErrSize = errors.New("alch: shorter than the character header")
)

// ReadHeader checks the tag and the header size and answers the name.
func ReadHeader(b []byte) (Header, error) {
	if len(b) < 4 || binary.LittleEndian.Uint32(b) != Tag {
		return Header{}, ErrTag
	}
	if len(b) < HeaderSize {
		return Header{}, ErrSize
	}
	name := b[NameStart : NameStart+NameSize]
	for i, c := range name {
		if c == 0 {
			name = name[:i]
			break
		}
	}
	return Header{Name: append([]byte(nil), name...)}, nil
}
