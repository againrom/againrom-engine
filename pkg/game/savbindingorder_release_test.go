package game

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// bindingOrderRows decodes the action supplement of a SAV: its actor bindings
// in file order and the IDs of its terminal actors in ascending order.
func bindingOrderRows(t *testing.T, raw []byte) (rows []currentActionBinding, terminal []sim.EntityID) {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	a, err := readCurrentActions(&doc)
	if err != nil || a == nil {
		t.Fatal("action supplement", err)
	}
	for id, v := range a.Values {
		if v.CurrentTerminal != nil {
			terminal = append(terminal, id)
		}
	}
	slices.Sort(terminal)
	return a.Bindings, terminal
}

// bindingOrderOwnerSave loads the owner SAV at path, which holds terminal
// actors: bodies that decayed out of play and are bound only through the
// action supplement's current values. F2 writes the mission twice under one
// name and the two files match byte for byte. Their terminal actors' bindings
// stand in ascending entity ID. The first file loaded cold restores the same
// terminal actors, and F2 from that session binds them in the same order.
func bindingOrderOwnerSave(t *testing.T, path, sha string, terminalActors int) {
	t.Helper()
	input, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprintf("%x", sha256.Sum256(input)); got != sha {
		t.Fatalf("owner save changed: %s", got)
	}
	if _, held := bindingOrderRows(t, input); len(held) != terminalActors {
		t.Fatalf("owner save holds %d terminal actors, want %d", len(held), terminalActors)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, filepath.Base(path)), input, 0o600); err != nil {
		t.Fatal(err)
	}
	f, app := castOrderSession(t, dir)
	loaded := f.live.world.Hash()
	savedDir, first := autoGetF2Save(t, f, app, "one byte sequence")
	_, second := autoGetF2Save(t, f, app, "one byte sequence")
	if f.live.world.Hash() != loaded {
		t.Fatal("the mission advanced between the two SAVEs")
	}
	rows, terminal := bindingOrderRows(t, first)
	if !bytes.Equal(first, second) {
		other, _ := bindingOrderRows(t, second)
		at := 0
		for at < len(rows) && at < len(other) && rows[at] == other[at] {
			at++
		}
		t.Fatalf("two F2 SAVEs of one state differ (%d and %d bytes); the bindings share their first %d of %d and %d rows",
			len(first), len(second), at, len(rows), len(other))
	}
	var bound []sim.EntityID
	for _, row := range rows {
		if _, held := slices.BinarySearch(terminal, row.ID); held && !row.Structure {
			bound = append(bound, row.ID)
		}
	}
	if len(terminal) != terminalActors || !slices.Equal(bound, terminal) {
		t.Fatalf("terminal bindings %v, want %v in ascending ID", bound, terminal)
	}
	g, app2 := castOrderSession(t, savedDir)
	var cold []sim.EntityID
	for _, row := range g.live.world.CurrentTerminalActors() {
		cold = append(cold, row.ID)
	}
	slices.Sort(cold)
	if !slices.Equal(cold, terminal) {
		t.Fatalf("cold LOAD restored terminal actors %v, saved %v", cold, terminal)
	}
	_, third := autoGetF2Save(t, g, app2, "one byte sequence")
	if again, _ := bindingOrderRows(t, third); !slices.Equal(again, rows) {
		t.Fatalf("bindings after the cold LOAD differ from the saved %d rows", len(rows))
	}
	t.Logf("%d bindings, %d terminal actors %v; two F2 files of %d bytes equal", len(rows), len(terminal), terminal, len(first))
}

// TestReleaseOwnerSavesWriteOneByteSequence runs the owner's SAVs that hold 8,
// 9 and 10 terminal actors through bindingOrderOwnerSave.
func TestReleaseOwnerSavesWriteOneByteSequence(t *testing.T) {
	t.Run("cast crash game0001", func(t *testing.T) {
		path := os.Getenv("AGAINROM_CAST_CRASH_SAV")
		if path == "" {
			t.Skip("no AGAINROM_CAST_CRASH_SAV: the owner's crashing game0001 is an owner input")
		}
		bindingOrderOwnerSave(t, path, "ad3a71eac52191582a0bb1ad27592a3bc7029c422ba9af337558f1476fb3cf02", 8)
	})
	t.Run("mission 20 end game0007", func(t *testing.T) {
		path := os.Getenv("AGAINROM_MISSION20_END_SAV")
		if path == "" {
			t.Skip("no AGAINROM_MISSION20_END_SAV: the owner's mission20end save is an owner input")
		}
		bindingOrderOwnerSave(t, path, "987482f272dd2d765b99daf05dfbf4a34c2983824174784389e3d65356d5b467", 9)
	})
	t.Run("mission 140 engine-savealive", func(t *testing.T) {
		path := os.Getenv("AGAINROM_ROOD_BAD_SAV")
		if path == "" {
			t.Skip("no AGAINROM_ROOD_BAD_SAV: the owner's mission-140 engine SAV is an owner input")
		}
		bindingOrderOwnerSave(t, path, "5b3eb30c1ed3f32be5ab847ecc95e5a04040d6befda85ac40fc40a1e985e1058", 10)
	})
}
