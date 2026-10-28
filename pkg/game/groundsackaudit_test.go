package game

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
)

// sameGroundSacksByCellForAudit compares a file's ground Sacks with the ones a
// SAVE wrote, joined by cell: a Sack LOAD keeps unbound gets a new Identity at
// each SAVE (DIV-2884). Each file Sack must be written once at its cell with
// its gold, items, container tail and Token value unchanged, and the SAVE
// writes no other Sack.
func sameGroundSacksByCellForAudit(file, saved []sav.GroundSack) (string, bool) {
	if len(saved) != len(file) {
		return fmt.Sprintf("SAVE wrote %d Sack(s) for the file's %d", len(saved), len(file)), false
	}
	fileCells := make(map[uint16]int, len(file))
	for _, s := range file {
		fileCells[s.Cell]++
	}
	written := make(map[uint16][]sav.GroundSack, len(saved))
	for _, s := range saved {
		written[s.Cell] = append(written[s.Cell], s)
	}
	for _, want := range file {
		got := written[want.Cell]
		if fileCells[want.Cell] != 1 || len(got) != 1 {
			return fmt.Sprintf("Sack %#x at cell %#x: the file holds %d and SAVE wrote %d there", want.Identity, want.Cell, fileCells[want.Cell], len(got)), false
		}
		g := got[0]
		if g.FineX != want.FineX || g.FineY != want.FineY || g.Gold != want.Gold || g.InsertIndex != want.InsertIndex || g.Accumulator != want.Accumulator || g.Token1C != want.Token1C {
			return fmt.Sprintf("Sack %#x at cell %#x: SAVE wrote fine %d,%d gold %d insert %d acc %d token %d, want %d,%d %d %d %d %d", want.Identity, want.Cell,
				g.FineX, g.FineY, g.Gold, g.InsertIndex, g.Accumulator, g.Token1C, want.FineX, want.FineY, want.Gold, want.InsertIndex, want.Accumulator, want.Token1C), false
		}
		if !reflect.DeepEqual(g.Items, want.Items) {
			return fmt.Sprintf("Sack %#x at cell %#x: SAVE wrote items %+v, want %+v", want.Identity, want.Cell, g.Items, want.Items), false
		}
	}
	return "", true
}

func TestSameGroundSacksByCellForAuditJoinsByCell(t *testing.T) {
	file := []sav.GroundSack{
		{Identity: 0x1000013, RuntimeID: 132, Cell: 0x6a14, Gold: 111, Items: []sav.Piece{{Class: "Weapon", Code: 33076, Price: 267, Weight: 3}}, Accumulator: 3, Token1C: 378},
		{Identity: 0x1000011, RuntimeID: 133, Cell: 0x2646, Gold: 862, Items: []sav.Piece{{Class: "Weapon", Code: 33045, Price: 700, Weight: 5}}, Accumulator: 5, Token1C: 1562},
	}
	clone := func() []sav.GroundSack {
		out := make([]sav.GroundSack, len(file))
		for i, s := range file {
			out[i] = s
			out[i].Items = append([]sav.Piece(nil), s.Items...)
		}
		return out
	}
	// A SAVE renews each unbound Sack's Identity and may write them in
	// another order; the join by cell still finds both.
	renewed := clone()
	renewed[0], renewed[1] = renewed[1], renewed[0]
	renewed[0].Identity, renewed[1].Identity = 0x51000320, 0x51000340
	if msg, ok := sameGroundSacksByCellForAudit(file, renewed); !ok {
		t.Fatalf("renewed identities in another order: %s", msg)
	}
	for name, edit := range map[string]func([]sav.GroundSack) []sav.GroundSack{
		"gold":        func(s []sav.GroundSack) []sav.GroundSack { s[0].Gold++; return s },
		"item":        func(s []sav.GroundSack) []sav.GroundSack { s[1].Items[0].Code++; return s },
		"lost item":   func(s []sav.GroundSack) []sav.GroundSack { s[1].Items = nil; return s },
		"insert":      func(s []sav.GroundSack) []sav.GroundSack { s[0].InsertIndex++; return s },
		"accumulator": func(s []sav.GroundSack) []sav.GroundSack { s[1].Accumulator--; return s },
		"token":       func(s []sav.GroundSack) []sav.GroundSack { s[0].Token1C++; return s },
		"fine":        func(s []sav.GroundSack) []sav.GroundSack { s[0].FineX++; return s },
		"moved":       func(s []sav.GroundSack) []sav.GroundSack { s[0].Cell++; return s },
		"lost Sack":   func(s []sav.GroundSack) []sav.GroundSack { return s[:1] },
		"extra Sack":  func(s []sav.GroundSack) []sav.GroundSack { return append(s, sav.GroundSack{Identity: 9, Cell: 0x0101}) },
		"same cell":   func(s []sav.GroundSack) []sav.GroundSack { s[1].Cell = s[0].Cell; return s },
	} {
		if msg, ok := sameGroundSacksByCellForAudit(file, edit(clone())); ok || !strings.Contains(msg, "SAVE wrote") {
			t.Errorf("%s: compared equal (%q)", name, msg)
		}
	}
	// Two file Sacks on one cell are never joined, even when the SAVE
	// writes the same count.
	shared := clone()
	shared[1].Cell = shared[0].Cell
	if _, ok := sameGroundSacksByCellForAudit(shared, append(clone()[:1], sav.GroundSack{Cell: 0x0101})); ok {
		t.Error("two file Sacks on one cell compared equal")
	}
}
