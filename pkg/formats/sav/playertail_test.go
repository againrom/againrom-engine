package sav

import "testing"

// TestPlayerTailAccountsForEveryByte is the layout test: a distinctive
// 32-byte pattern (byte i holds value i) survives PlayerTailFromRecord
// unchanged at every one of the 32 indices, and Formation() reads exactly
// PlayerTailFormationByte (SAV-662: Player+0x30's own 0x1f) and no other.
// The other 31 indices have no named field (SAV-662, SAV-663: zero in every
// one of 410 preserved records, and no promoted claim gives any of them a
// value this package could apply), so Raw is their whole account.
func TestPlayerTailAccountsForEveryByte(t *testing.T) {
	rec := newRecord("Player", 0, 0)
	raw := make([]byte, PlayerTailLen)
	for i := range raw {
		raw[i] = byte(i)
	}
	rec.Raw["PRaw32"] = raw
	tail, err := PlayerTailFromRecord(rec)
	if err != nil {
		t.Fatalf("PlayerTailFromRecord: %v", err)
	}
	for i := 0; i < PlayerTailLen; i++ {
		if tail.Raw[i] != byte(i) {
			t.Errorf("byte %d = %#02x, want %#02x", i, tail.Raw[i], byte(i))
		}
	}
	if got, want := tail.Formation(), byte(PlayerTailFormationByte); got != want {
		t.Errorf("Formation() = %#02x, want byte %#02x (index %#x)", got, want, PlayerTailFormationByte)
	}
}

// TestPlayerTailFromRecordRefusesAMissingOrShortBlock never reads past a
// record that does not carry the full 32-byte span.
func TestPlayerTailFromRecordRefusesAMissingOrShortBlock(t *testing.T) {
	if _, err := PlayerTailFromRecord(nil); err == nil {
		t.Error("accepted a nil record")
	}
	if _, err := PlayerTailFromRecord(newRecord("Player", 0, 0)); err == nil {
		t.Error("accepted a Player record with no PRaw32 member")
	}
	short := newRecord("Player", 0, 0)
	short.Raw["PRaw32"] = make([]byte, 31)
	if _, err := PlayerTailFromRecord(short); err == nil {
		t.Error("accepted a 31-byte tail")
	}
}

// TestPlayerTailFromRecordReachesTheRealWalk proves the accessor reads the
// same walked field program_test.go's own fixture writes (32 bytes of
// 0x99), end to end through PartyWalk rather than a hand-built Record.
func TestPlayerTailFromRecordReachesTheRealWalk(t *testing.T) {
	f := walkOpen(t, walkFixture{mapName: "10.alm", mission: 10, chars: []wantChar{hero()}})
	_, rec, err := f.PartyWalk()
	if err != nil {
		t.Fatalf("PartyWalk: %v", err)
	}
	tail, err := PlayerTailFromRecord(rec)
	if err != nil {
		t.Fatalf("PlayerTailFromRecord: %v", err)
	}
	for i, b := range tail.Raw {
		if b != 0x99 {
			t.Fatalf("byte %d = %#02x, want the fixture's own 0x99 fill", i, b)
		}
	}
	if tail.Formation() != 0x99 {
		t.Errorf("Formation() = %#02x, want 0x99", tail.Formation())
	}
}
