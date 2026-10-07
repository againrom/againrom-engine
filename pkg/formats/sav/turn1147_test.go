package sav

import "testing"

func TestTurnInProgressUsesOnlyTheCompleteDword1147(t *testing.T) {
	for _, offset := range []int{0x9f, 0xa0, 0xa1, 0xa2, 0xa3, 0xa4} {
		var a ActorRecord
		a.Mover[offset] = 1
		if got, want := a.TurnInProgress(), offset >= 0xa0 && offset < 0xa4; got != want {
			t.Fatalf("nonzero byte %x: turning=%v, want %v", offset, got, want)
		}
	}
}
