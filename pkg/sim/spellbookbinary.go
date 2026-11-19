package sim

import "encoding/binary"

// One presence state followed by 28 fixed (range, Defensive, cost) slots.
// Membership remains the existing KnownSpells word. No archive key enters sim.
const spellbookRecordLen = 1 + 28*4

func encodeSpellbook(b []byte, book Spellbook) {
	b[0] = byte(book.State)
	for i, slot := range book.Slots {
		o := 1 + 4*i
		b[o], b[o+1] = slot.Range, slot.Defensive
		binary.LittleEndian.PutUint16(b[o+2:], slot.ManaCost)
	}
}

func decodeSpellbook(b []byte, known uint32) (Spellbook, error) {
	book := Spellbook{State: BookState(b[0])}
	for i := range book.Slots {
		o := 1 + 4*i
		book.Slots[i] = BookSpell{b[o], b[o+1], binary.LittleEndian.Uint16(b[o+2:])}
	}
	// A native mode that names only ids a book holds is an older form of an
	// ordinary book: its spell past the slots has no saved parameters.
	if book.hasNativeSpells() && known & ^bookIDMask == 0 {
		if book.State == BookNativePresent {
			book.State = BookPresent
		} else {
			book.State = BookAbsent
		}
	}
	return book, book.Validate(known)
}
