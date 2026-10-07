package ui

// The mission map's own KEY surface: the two selection keys, the ten group
// digits, the panel's Swarm cell, and the internal panel pair operation.
//
// Everything here is driven through the App's own frame or through the viewer's
// own state -- no window, no engine -- so each case runs the statement a played
// frame runs and not a copy of it.

import (
	"image"
	"slices"
	"testing"
)

// mkOnMap is atOnMap's fixture with a local participant established, which is
// what makes ownership observable at all: with localOwner zero every entity is
// owned by the reading of an unestablished participant, and a filter on owners
// cannot be told from no filter.
func mkOnMap(t *testing.T) (*App, *Viewer) {
	t.Helper()
	a, v, _ := atOnMap(t)
	v.SetLocalOwner(1)
	v.SetEntities([]MapEntity{
		{ID: 4, Cell: image.Pt(3, 3), Life: LifeAlive, Owner: 1},
		{ID: 7, Cell: image.Pt(4, 3), Life: LifeAlive, Owner: 2},
		{ID: 9, Cell: image.Pt(5, 3), Life: LifeAlive, Owner: 1},
		{ID: 12, Cell: image.Pt(6, 3), Life: LifeDead, Owner: 1},
	})
	return a, v
}

// TestTheArmyKeySelectsEveryOwnedLiveUnit is `AI-KEY-125`'s E row and
// `AI-SELECT-122`'s "E selects every owned exact-name CUnit": every owned unit
// and no other owner's, replacing whatever stood before.
func TestTheArmyKeySelectsEveryOwnedLiveUnit(t *testing.T) {
	_, v := mkOnMap(t)

	// A prior selection that is neither a subset nor a superset of the answer,
	// so "replaces" is asserted against something the answer must both add to
	// and remove from.
	v.sel = selection{7}
	v.selectAllOwnedUnits()

	if want := (selection{4, 9}); !slices.Equal(v.sel, want) {
		t.Fatalf("selection = %+v, want %+v — owned and alive, ascending", v.sel, want)
	}

	// NON-VACUITY: with no local participant established the same call takes
	// every live entity, so the case above is reading the ownership filter and
	// not an empty snapshot.
	v.SetLocalOwner(0)
	v.selectAllOwnedUnits()
	if want := (selection{4, 7, 9}); !slices.Equal(v.sel, want) {
		t.Fatalf("with no participant established: selection = %+v, want %+v", v.sel, want)
	}
}

// TestTheArmyKeyReachesTheViewerThroughItsOwnFrame is the same key read through
// the App's own dispatch rather than through the method: E, and nothing else on
// the frame, leaves the whole owned army selected.
func TestTheArmyKeyReachesTheViewerThroughItsOwnFrame(t *testing.T) {
	a, v := mkOnMap(t)
	wornBefore := v.hudShown(hudPanelWorn)

	a.step(appInput{SelectAll: true}, atAt)

	if want := (selection{4, 9}); !slices.Equal(v.sel, want) {
		t.Fatalf("the E frame left selection %+v, want %+v", v.sel, want)
	}
	if v.hudShown(hudPanelWorn) != wornBefore {
		t.Errorf("the E frame also flipped the worn-set switch — the letter was not vacated")
	}
}

// TestTheGroupDigitsAssignRecallAugmentAndCentre is `AI-KEY-125`'s three digit
// rows: Ctrl assigns the owned selection, plain recalls it, Shift augments
// rather than replacing, Alt recalls and centres, and Ctrl wins over Alt.
func TestTheGroupDigitsAssignRecallAugmentAndCentre(t *testing.T) {
	t.Run("Ctrl assigns the OWNED members only", func(t *testing.T) {
		_, v := mkOnMap(t)
		v.sel = selection{4, 7, 9} // 7 is another owner's
		v.groupKey(1, true, false, false)
		if want := (selection{4, 9}); !slices.Equal(v.groups[1], want) {
			t.Fatalf("group 1 = %+v, want %+v — a foreign object is not carried into a group", v.groups[1], want)
		}
		if want := (selection{4, 7, 9}); !slices.Equal(v.sel, want) {
			t.Errorf("the assignment moved the selection to %+v, want %+v", v.sel, want)
		}
	})

	t.Run("plain recalls, replacing", func(t *testing.T) {
		_, v := mkOnMap(t)
		v.groups[2] = selection{9}
		v.sel = selection{4}
		v.groupKey(2, false, false, false)
		if want := (selection{9}); !slices.Equal(v.sel, want) {
			t.Fatalf("selection = %+v, want %+v", v.sel, want)
		}
	})

	t.Run("Shift augments and keeps the set ascending", func(t *testing.T) {
		_, v := mkOnMap(t)
		v.groups[2] = selection{4}
		v.sel = selection{9}
		v.groupKey(2, false, false, true)
		if want := (selection{4, 9}); !slices.Equal(v.sel, want) {
			t.Fatalf("selection = %+v, want %+v", v.sel, want)
		}
	})

	t.Run("Shift over a member already selected adds no duplicate", func(t *testing.T) {
		_, v := mkOnMap(t)
		v.groups[2] = selection{4, 9}
		v.sel = selection{9}
		v.groupKey(2, false, false, true)
		if want := (selection{4, 9}); !slices.Equal(v.sel, want) {
			t.Fatalf("selection = %+v, want %+v", v.sel, want)
		}
	})

	t.Run("a group whose members are all gone preserves the selection", func(t *testing.T) {
		_, v := mkOnMap(t)
		v.groups[3] = selection{999} // never in the snapshot
		v.sel = selection{4}
		v.groupKey(3, false, false, false)
		if want := (selection{4}); !slices.Equal(v.sel, want) {
			t.Fatalf("selection = %+v, want %+v — a selection form with no qualifier preserves", v.sel, want)
		}
	})

	t.Run("Alt recalls and centres, plain recalls and does not", func(t *testing.T) {
		_, v := mkOnMap(t)
		// A member far from the origin. The camera clamps to the map, so a
		// member near cell (0,0) would leave the camera at the origin whichever
		// arm ran and the case would read the clamp rather than the centring.
		v.SetEntities(append(v.entities, MapEntity{ID: 40, Cell: image.Pt(40, 40), Life: LifeAlive, Owner: 1}))
		v.groups[4] = selection{40}

		v.Camera().X, v.Camera().Y = 0, 0
		v.Camera().Clamp()
		plainX, plainY := v.Camera().X, v.Camera().Y
		v.groupKey(4, false, false, false)
		if v.Camera().X != plainX || v.Camera().Y != plainY {
			t.Fatalf("a plain recall moved the camera to (%v,%v) from (%v,%v)", v.Camera().X, v.Camera().Y, plainX, plainY)
		}

		v.groupKey(4, false, true, false)
		if v.Camera().X == plainX && v.Camera().Y == plainY {
			t.Fatalf("an Alt recall left the camera at (%v,%v) — it must centre on the member mean", plainX, plainY)
		}
		if want := (selection{40}); !slices.Equal(v.sel, want) {
			t.Errorf("the Alt recall left selection %+v, want %+v", v.sel, want)
		}
	})

	t.Run("Ctrl wins over Alt", func(t *testing.T) {
		_, v := mkOnMap(t)
		v.groups[5] = selection{9}
		v.sel = selection{4}
		v.Camera().X, v.Camera().Y = 0, 0
		v.Camera().Clamp()
		before := [2]float64{v.Camera().X, v.Camera().Y}

		v.groupKey(5, true, true, false)

		if want := (selection{4}); !slices.Equal(v.sel, want) {
			t.Fatalf("Ctrl+Alt recalled instead of assigning: selection = %+v, want %+v", v.sel, want)
		}
		if want := (selection{4}); !slices.Equal(v.groups[5], want) {
			t.Fatalf("Ctrl+Alt left group 5 = %+v, want %+v", v.groups[5], want)
		}
		if v.Camera().X != before[0] || v.Camera().Y != before[1] {
			t.Errorf("Ctrl+Alt centred the camera; the assignment arm centres nothing")
		}
	})
}

// TestTheGroupDigitReachesTheViewerThroughItsOwnFrame is the digit row read
// through the App's own dispatch: a frame carrying GroupKey and the Ctrl latch
// assigns, and a later plain frame recalls.
//
// THE ZERO VALUE IS THE CASE THAT MATTERS. Group 0 is a real group, so an
// `appInput{}` with no key pressed must not reach the group handler at all;
// that is what the neutral frame in the middle asserts.
func TestTheGroupDigitReachesTheViewerThroughItsOwnFrame(t *testing.T) {
	a, v := mkOnMap(t)
	v.sel = selection{4, 9}

	assign := appInput{GroupKey: true, GroupDigit: 0}
	assign.Viewer.Ctrl = true
	a.step(assign, atAt)
	if want := (selection{4, 9}); !slices.Equal(v.groups[0], want) {
		t.Fatalf("group 0 = %+v, want %+v", v.groups[0], want)
	}

	v.sel = selection{7}
	a.step(appInput{}, atAt)
	if want := (selection{7}); !slices.Equal(v.sel, want) {
		t.Fatalf("a neutral frame recalled group 0: selection = %+v, want %+v", v.sel, want)
	}

	a.step(appInput{GroupKey: true, GroupDigit: 0}, atAt)
	if want := (selection{4, 9}); !slices.Equal(v.sel, want) {
		t.Fatalf("the recall frame left selection %+v, want %+v", v.sel, want)
	}
}

func TestPanelPairKeepsMixedState(t *testing.T) {
	set := func(v *Viewer, pack, book bool) {
		if v.hudShown(hudPanelPack) != pack {
			v.toggleHudPanel(hudPanelPack)
		}
		if v.hudShown(hudPanelBook) != book {
			v.toggleHudPanel(hudPanelBook)
		}
	}

	for _, tc := range []struct {
		name                     string
		pack, book               bool
		wantPack, wantBook       bool
		wantWhy                  string
		dollBefore, dollExpected bool
	}{
		{name: "both closed opens both", pack: false, book: false, wantPack: true, wantBook: true},
		{name: "both open closes both", pack: true, book: true, wantPack: false, wantBook: false},
		{name: "pack open alone closes the pack and leaves the book closed",
			pack: true, book: false, wantPack: false, wantBook: false,
			wantWhy: "a pair of toggles would have opened the book"},
		{name: "book open alone closes the book and leaves the pack closed",
			pack: false, book: true, wantPack: false, wantBook: false,
			wantWhy: "a pair of toggles would have opened the pack"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, v := mkOnMap(t)
			set(v, tc.pack, tc.book)
			doll := v.hudShown(hudPanelDoll)

			a.step(appInput{Panels: true}, atAt)

			if got := v.hudShown(hudPanelPack); got != tc.wantPack {
				t.Errorf("pack = %v, want %v — %s", got, tc.wantPack, tc.wantWhy)
			}
			if got := v.hudShown(hudPanelBook); got != tc.wantBook {
				t.Errorf("book = %v, want %v — %s", got, tc.wantBook, tc.wantWhy)
			}
			if v.hudShown(hudPanelDoll) != doll {
				t.Errorf("Space also flipped the paperdoll switch; it names two surfaces and not four")
			}
		})
	}
}

// TestThePanelSwarmCellArmsTheSwarmOrder is `AI-PANEL-123` cell 5: armed
// mode 6, click opcode `0x1a`.
func TestThePanelSwarmCellArmsTheSwarmOrder(t *testing.T) {
	if commandPanelCellSkipped(commandCellSwarm) {
		t.Fatal("the Swarm cell is still in commandPanelSkipMask")
	}
	for _, cell := range []commandPanelCell{commandCellDefend, commandCellRetreat} {
		if commandPanelCellSkipped(cell) {
			t.Errorf("implemented player order cell %d is disabled", cell)
		}
	}

	_, v := mkOnMap(t)
	v.sel = selection{4}
	v.pressCommandPanelCell(commandCellSwarm, true)

	if v.aimedOrder() != commandSwarm {
		t.Fatalf("the Swarm cell armed %v, want commandSwarm", v.aimedOrder())
	}
	if cell, ok := commandPanelSelected(v); !ok || cell != commandCellSwarm {
		t.Errorf("the overlay shows cell %v (ok=%v), want the Swarm cell", cell, ok)
	}
}

// DIV-287
func TestThePanelCastCellNeedsAManaPool(t *testing.T) {
	_, v := mkOnMap(t)
	// Entity 4 (mkOnMap) carries no MaxMana, so its zero value is the
	// no-mana case; entity 9 is given one for the mana case below. The first
	// leaves the overlay cleared; the second arms Cast mode as the C key does
	// (MENU-055), with the standing spell untouched.
	v.selectedSpell = 3
	v.spellArmed = true

	t.Run("no mana pool: the overlay clears, matching a disabled cell", func(t *testing.T) {
		v.sel = selection{4}
		v.cmdOverlayHidden = false
		v.pressCommandPanelCell(commandCellCast, true)

		if cell, ok := commandPanelSelected(v); ok {
			t.Errorf("the overlay shows cell %v (ok=%v), want none — the primary has no mana pool", cell, ok)
		}
	})

	t.Run("a mana pool: the overlay shows Cast selected, as before this clause", func(t *testing.T) {
		v.SetEntities([]MapEntity{
			{ID: 4, Cell: image.Pt(3, 3), Life: LifeAlive, Owner: 1, Mana: 5, MaxMana: 20},
		})
		v.sel = selection{4}
		v.cmdOverlayHidden = true
		v.pressCommandPanelCell(commandCellCast, true)

		if cell, ok := commandPanelSelected(v); !ok || cell != commandCellCast {
			t.Errorf("the overlay shows cell %v (ok=%v), want the Cast cell", cell, ok)
		}
	})

	t.Run("a press with no standing spell is a no-op, with one it arms Cast", func(t *testing.T) {
		v.cancelMapCommand()
		v.sel = selection{4}
		v.pressCommandPanelCell(commandCellCast, true)
		if v.missionMode() == modeCast {
			t.Errorf("mode %d with no spell, want no Cast", v.missionMode())
		}
		v.selectedSpell = 3
		v.pressCommandPanelCell(commandCellCast, true)
		if v.missionMode() != modeCast || v.selectedSpell != 3 {
			t.Errorf("mode %d with spell %d, want Cast armed with the standing spell", v.missionMode(), v.selectedSpell)
		}
	})
}
