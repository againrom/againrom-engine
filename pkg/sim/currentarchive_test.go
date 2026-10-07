package sim

import (
	"bytes"
	"testing"
)

func TestCurrentArchiveCoordinatesRelocateEveryTypedHolder(t *testing.T) {
	for _, family := range []string{"actor and Groups", "structure", "terminal weapon"} {
		t.Run(family, func(t *testing.T) {
			var w *World
			switch family {
			case "actor and Groups":
				w = sourceBindingWorld1111(t)
				if err := w.ImportSavedGroups([]SavedGroup{{ID: 1, Selector: 41,
					Reference: SavedGroupReference{Key: 0x10000002, Archive: 2, Class: 1, Owner: 9},
					Owner:     SavedGroupReference{Key: 0x10000002, Archive: 2, Class: 1, Owner: 9},
					Members:   []SavedGroupMember{{}, {Archive: 4, Entity: 7, Bound: true}}, Words: []uint16{1, 2, 3}}}, nil); err != nil {
					t.Fatal(err)
				}
			case "structure":
				w = sourceCopyWorld(t)
			case "terminal weapon":
				w = deadWorld(t)
				d := deadInput(1, 5, -10007)
				d.Source.HeldWeapon = deadWeaponSentinel()
				if err := w.ImportOriginalDeadActors([]OriginalDeadActor{d}); err != nil {
					t.Fatal(err)
				}
			}
			before := mustMarshal(t, w)
			for cycle := 0; cycle < 2; cycle++ {
				seen := map[uint16]bool{}
				var rows, inverse []CurrentArchiveCoordinate
				w.currentArchiveFields(func(index *uint16) {
					if *index != 0 && !seen[*index] {
						rows = append(rows, CurrentArchiveCoordinate{*index, *index + 100})
						inverse = append(inverse, CurrentArchiveCoordinate{*index + 100, *index})
						seen[*index] = true
					}
				})
				if len(rows) == 0 {
					t.Fatal("fixture lacks typed archive coordinates")
				}
				if err := w.RestoreCurrentArchiveCoordinates(rows, nil); err != nil {
					t.Fatal(err)
				}
				if bytes.Equal(before, mustMarshal(t, w)) {
					t.Fatal("coordinate mutation was not hashed")
				}
				if err := w.RestoreCurrentArchiveCoordinates(inverse, nil); err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(before, mustMarshal(t, w)) {
					t.Fatal("inverse coordinate mapping lost values or aliasing", cycle)
				}
			}
		})
	}
}

func TestCurrentArchiveCoordinatesAreAtomicAndDoNotAliasUnmovedHolders(t *testing.T) {
	w := sourceBindingWorld1111(t)
	if err := w.ImportSavedGroups([]SavedGroup{{ID: 1, Owner: SavedGroupReference{Class: 1, Archive: 2, Key: 0x10000002, Owner: 9}, Members: []SavedGroupMember{{Archive: 4, Entity: 7, Bound: true}}}}, nil); err != nil {
		t.Fatal(err)
	}
	before := mustMarshal(t, w)
	for _, rows := range [][]CurrentArchiveCoordinate{{{4, 2}}, {{4, 8}, {2, 8}}, {{4, 8}, {4, 9}}, {{0, 8}}} {
		if err := w.RestoreCurrentArchiveCoordinates(rows, nil); err == nil {
			t.Fatal("malformed coordinate mapping accepted", rows)
		}
		if !bytes.Equal(before, mustMarshal(t, w)) {
			t.Fatal("failed mapping changed the World")
		}
	}
	if err := w.RestoreCurrentArchiveCoordinates(nil, map[EntityID]CurrentSourceGroup{99: {Index: 3}}); err == nil || !bytes.Equal(before, mustMarshal(t, w)) {
		t.Fatal("missing Group actor was adopted", err)
	}
}
