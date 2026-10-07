package sav

import "testing"

func TestSessionStateCarriesRegistersAndRawRegions(t *testing.T) {
	f := open(t, standard())
	if err := f.SetTriggerResult(0, -1); err != nil {
		t.Fatal(err)
	}
	if err := f.SetTriggerResult(99, 12345); err != nil {
		t.Fatal(err)
	}
	var head [48]byte
	head[10] = 0x42
	if err := f.SetRawHead(head); err != nil {
		t.Fatal(err)
	}
	var mid [400]byte
	mid[200] = 0x99
	if err := f.SetRawMid(mid); err != nil {
		t.Fatal(err)
	}
	state, err := f.SessionState()
	if err != nil {
		t.Fatal(err)
	}
	if state.Registers[0] != -1 || state.Registers[99] != 12345 {
		t.Fatalf("registers[0]/[99] = %d/%d, want -1/12345", state.Registers[0], state.Registers[99])
	}
	for i, v := range state.Registers {
		if i == 0 || i == 99 {
			continue
		}
		if v != 0 {
			t.Fatalf("register %d = %d, want 0 (only 0 and 99 were set)", i, v)
		}
	}
	if state.RawHead != head {
		t.Fatalf("RawHead = %v, want %v", state.RawHead, head)
	}
	if state.RawMid != mid {
		t.Fatalf("RawMid = %v, want %v", state.RawMid, mid)
	}
}

// TestSessionBlockLayoutAccountsForEveryByte locks the six accessor
// populations plus the three still-unpromoted gaps SAV-SESS-031 itself names
// (the two bytes and one dword before the win/lose pair, and the dword
// between them research has not named) to exactly sessionLen, so a change to
// one width fails here instead of silently opening or closing a gap.
func TestSessionBlockLayoutAccountsForEveryByte(t *testing.T) {
	const (
		diplomacyLen = diplomacySide * diplomacySide
		prefixGap    = sessDiplomacy - (sessRawMid + rawMidLen) // +0xa9bc..+0xa9c4
		tailGap      = sessWon - (sessDiplomacy + diplomacyLen) // u8 +0xa48, u8 +0xa49, u32 +0xa4c
		middleGap    = sessLose - (sessWon + 4)                 // the unnamed +0xb3b0 dword
		counterLen   = 4 + 4                                    // won, lost
	)
	if prefixGap != 8 || tailGap != 6 || middleGap != 4 {
		t.Fatalf("gap widths = %d/%d/%d, want 8/6/4", prefixGap, tailGap, middleGap)
	}
	sum := triggerResultCount*4 + triggerLatchCount + rawHeadLen + rawMidLen +
		prefixGap + diplomacyLen + tailGap + middleGap + counterLen
	if sum != sessionLen {
		t.Fatalf("accounted session bytes = %d, want %d", sum, sessionLen)
	}
}
