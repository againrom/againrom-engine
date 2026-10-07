package sim

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
)

// BookValueAnchor binds absent table-backed mode to the ordinary values that
// supplied its representation. Object addresses and unrelated actor fields do
// not participate, so graph relocation cannot change this policy.
func BookValueAnchor(book Spellbook, known uint32) [32]byte {
	var wire [5 + 28*4]byte
	wire[0] = byte(book.State)
	binary.LittleEndian.PutUint32(wire[1:], known)
	for i, slot := range book.Slots {
		at := 5 + i*4
		wire[at], wire[at+1] = slot.Range, slot.Defensive
		binary.LittleEndian.PutUint16(wire[at+2:], slot.ManaCost)
	}
	return sha256.Sum256(wire[:])
}

// RestoreBookMode keeps ordinary presence, membership and instance parameters.
// Only a matching representation may recover absent native legacy mode. Older
// unanchored metadata has no authority to erase an ordinary book.
func RestoreBookMode(book Spellbook, known uint32, mode BookState, anchor *[32]byte, extra uint32) (Spellbook, uint32, error) {
	if mode > BookNativeAbsent || extra&0x1ffffffe != 0 || mode != BookLegacy && anchor != nil || anchor != nil && *anchor == ([32]byte{}) {
		return Spellbook{}, 0, fmt.Errorf("invalid current book mode policy")
	}
	if mode == BookLegacy && anchor != nil && BookValueAnchor(book, known) == *anchor {
		book = Spellbook{}
	}
	known |= extra
	if book.State != BookLegacy && known & ^uint32(0x1ffffffe) != 0 {
		if book.HasInstances() {
			book.State = BookNativePresent
		} else {
			book.State = BookNativeAbsent
		}
	}
	if err := book.Validate(known); err != nil {
		return Spellbook{}, 0, err
	}
	return book, known, nil
}
