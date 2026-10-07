package sav

import "testing"

func TestSetMapNameOddDeltaPadIsWrittenNotCarried(t *testing.T) {
	f := open(t, standard())

	// Reach a state with a pad byte at the very end of Body, regardless of
	// standard()'s own starting parity: if it does not already have one, one
	// odd-delta edit creates a fresh (zero) one; if it already has one, it
	// is already the state this test wants.
	doc, _, err := f.exactDocument()
	if err != nil {
		t.Fatal(err)
	}
	if doc.end%2 == 0 {
		if err := f.SetMapName(f.Head.MapName + "Q"); err != nil {
			t.Fatal(err)
		}
		doc, _, err = f.exactDocument()
		if err != nil {
			t.Fatal(err)
		}
	}
	if doc.end%2 == 0 {
		t.Fatalf("setup: body's natural end is still even (%d) after a forced odd-delta edit", doc.end)
	}
	if len(f.Body) == 0 || f.Body[len(f.Body)-1] != 0 {
		t.Fatalf("setup: fresh pad byte is %#x, want the fixture's own literal 0", f.Body[len(f.Body)-1])
	}

	// Overwrite the pad byte with a non-zero sentinel, simulating one of the
	// 46 of 136 preserved saves whose real alignment byte is non-zero
	// (F-2's own observed values include 0x04, 0x46, 0x5a, 0xde, 0x82, 0xd8,
	// 0x24). exactDocument's own length check does not inspect this byte's
	// value, so the body remains valid to edit.
	const sentinel = 0xab
	f.Body[len(f.Body)-1] = sentinel
	beforeName := f.Head.MapName
	wantMission, wantDiff, wantPlayers := f.Head.Mission, f.Head.Difficulty, len(f.Players)

	// There: an odd-delta edit that must REMOVE the pad. hadPad is true
	// (just established above), so the setter drops the last body byte --
	// the sentinel -- rather than carrying its value forward.
	thereName := beforeName + "R"
	if len(thereName)-len(beforeName) == 0 || (len(thereName)-len(beforeName))%2 == 0 {
		t.Fatalf("setup: delta %d is not odd", len(thereName)-len(beforeName))
	}
	if err := f.SetMapName(thereName); err != nil {
		t.Fatal(err)
	}
	doc, _, err = f.exactDocument()
	if err != nil {
		t.Fatal(err)
	}
	if doc.end%2 != 0 {
		t.Fatalf("after dropping the pad, the natural end is still odd (%d)", doc.end)
	}
	if len(f.Body) != doc.end {
		t.Fatalf("after dropping the pad, Body is %d bytes, want exactly the natural end %d (no pad)", len(f.Body), doc.end)
	}

	// Back: an odd-delta edit that must CREATE a pad again. hadPad is now
	// false, so the setter appends a FRESH literal 0 -- not the sentinel
	// the body carried before the "there" edit.
	if err := f.SetMapName(beforeName); err != nil {
		t.Fatal(err)
	}
	doc, _, err = f.exactDocument()
	if err != nil {
		t.Fatal(err)
	}
	if doc.end%2 == 0 {
		t.Fatalf("after re-creating the pad, the natural end is even (%d)", doc.end)
	}
	if len(f.Body) != doc.end+1 {
		t.Fatalf("after re-creating the pad, Body is %d bytes, want the natural end %d plus one pad byte", len(f.Body), doc.end)
	}
	got := f.Body[len(f.Body)-1]
	if got != 0 {
		t.Fatalf("recreated pad byte is %#x, want 0", got)
	}
	if got == sentinel {
		t.Fatal("the there-and-back pair restored the sentinel: it should have been overwritten with a fresh 0, not carried")
	}

	// Everything else is unchanged: the pad byte is the ONLY difference a
	// there-and-back pair of odd-delta edits leaves on a save whose own
	// alignment byte was non-zero.
	if f.Head.MapName != beforeName {
		t.Fatalf("MapName = %q, want %q restored", f.Head.MapName, beforeName)
	}
	if f.Head.Mission != wantMission || f.Head.Difficulty != wantDiff || len(f.Players) != wantPlayers {
		t.Fatalf("head/players moved: %+v", f.Head)
	}
}
