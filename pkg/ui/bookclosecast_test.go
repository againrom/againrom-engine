package ui

import (
	"image"
	"testing"
)

const bookCloseTeleport = 26

func bookCloseApp(t *testing.T) (*App, *Viewer) {
	t.Helper()
	a, v := castCursorApp(t)
	v.SetLocalOwner(1)
	v.SetEntities([]MapEntity{
		{ID: panelCasterID, Cell: image.Pt(3, 3), Life: LifeAlive, CastCapable: true, Owner: 1},
		{ID: panelVictimID, Cell: image.Pt(7, 3), Life: LifeAlive, Owner: 2, Hostile: true},
	})
	v.SetSpellbook(panelCasterID, append(castKindBook(), SpellEntry{ID: bookCloseTeleport, Name: "Teleport", PointTarget: true}))
	pressPanelKey(t, a, "space")
	chooseSpell(t, v, bookCloseTeleport)
	return a, v
}

func clickEnemy(t *testing.T, a *App, v *Viewer) (string, []order) {
	t.Helper()
	cursor := cursorAtCell(t, a, v, castCells.other[0], castCells.other[1])
	x, y := cellPoint(v, castCells.other[0], castCells.other[1])
	ords, _ := tapAt(v, x, y)
	return cursor, ords
}

func wantPlainAttack(t *testing.T, how, cursor string, ords []order) {
	t.Helper()
	if cursor != "attack" {
		t.Fatalf("%s: cursor over the enemy %q, want attack", how, cursor)
	}
	if len(ords) != 1 || ords[0].kind != orderKindAttack || ords[0].victim != panelVictimID || ords[0].spell != 0 || ords[0].cell {
		t.Fatalf("%s: click on the enemy %+v, want one plain attack with no spell", how, ords)
	}
}

func TestClosedBookClickOnEnemyAttacksWithoutCasting(t *testing.T) {
	closes := []struct {
		name  string
		close func(t *testing.T, a *App, v *Viewer)
	}{
		{"space", func(t *testing.T, a *App, _ *Viewer) { pressPanelKey(t, a, "space") }},
		{"book key", func(t *testing.T, a *App, _ *Viewer) { pressPanelKey(t, a, "b") }},
		{"pane book corner", func(t *testing.T, _ *App, v *Viewer) {
			pane, ok := v.characterPaneRect()
			if !ok {
				t.Fatal("setup: no character pane")
			}
			at := CharacterPaneCornerRect(pane, CharacterPaneBook).Min.Add(image.Pt(1, 1))
			if !v.pressCharacterPaneCorner(at.X, at.Y) {
				t.Fatal("setup: the book corner was not pressed")
			}
		}},
	}
	for _, c := range closes {
		t.Run(c.name, func(t *testing.T) {
			a, v := bookCloseApp(t)
			c.close(t, a, v)
			if v.hudShown(hudPanelBook) {
				t.Fatal("setup: the book is still open")
			}
			cursor, ords := clickEnemy(t, a, v)
			wantPlainAttack(t, c.name, cursor, ords)
		})
	}
}

func TestOpenBookClickOnEnemyStillCasts(t *testing.T) {
	a, v := bookCloseApp(t)
	cursor, ords := clickEnemy(t, a, v)
	if cursor != "cast" {
		t.Fatalf("open book: cursor %q, want cast", cursor)
	}
	if len(ords) != 1 || ords[0].kind != orderKindCast || ords[0].spell != bookCloseTeleport {
		t.Fatalf("open book: click %+v, want the Teleport cast", ords)
	}
	if !v.spellModeLive() {
		t.Fatal("open book: the cast spent the book-chosen spell")
	}
}

func TestArmedAttackOverAChosenSpellDoesNotCast(t *testing.T) {
	t.Run("armed", func(t *testing.T) {
		a, v := bookCloseApp(t)
		v.pressCommandPanelCell(commandCellAttack, true)
		cursor, ords := clickEnemy(t, a, v)
		wantPlainAttack(t, "armed attack", cursor, ords)
	})
	t.Run("armed then cancelled", func(t *testing.T) {
		a, v := bookCloseApp(t)
		v.pressCommandPanelCell(commandCellAttack, true)
		v.pressCommandPanelCell(commandCellAttack, true)
		cursor, ords := clickEnemy(t, a, v)
		wantPlainAttack(t, "cancelled attack", cursor, ords)
	})
}
