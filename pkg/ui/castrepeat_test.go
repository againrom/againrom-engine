package ui

import (
	"image"
	"testing"
)

type castClick struct {
	cast         int
	armed        bool
	selected     bool
	itemCastLive bool
}

func TestBookScrollStaysPickedAcrossCastsLikeASpell(t *testing.T) {
	v, shots := bookScrollFighter(t)
	x, y := bookCellCentre(t, v, 4)
	sbClick(v, x, y)
	fx, fy := cellPoint(v, 3, 3)
	var got []castClick
	for i := 0; i < 2; i++ {
		before := len(*shots)
		tapAt(v, fx, fy)
		got = append(got, castClick{len(*shots) - before, v.itemCast != nil, v.bookLitSpell() == 24, v.itemCast != nil})
	}
	for i, c := range got {
		if c.cast != 1 || !c.armed || !c.selected {
			t.Fatalf("scroll click %d: %+v (all %+v)", i+1, c, got)
		}
	}
	t.Logf("scroll clicks: %+v", got)
}

func TestBookSpellStaysPickedAcrossCasts(t *testing.T) {
	a, v := castCursorApp(t)
	pressPanelKey(t, a, "space")
	chooseSpell(t, v, castTarget)
	fx, fy := cellPoint(v, castCells.caster[0], castCells.caster[1])
	var got []castClick
	for i := 0; i < 2; i++ {
		ords, _ := tapAt(v, fx, fy)
		got = append(got, castClick{len(ords), v.spellModeLive(), v.selectedSpell == castTarget, false})
	}
	for i, c := range got {
		if c.cast != 1 || !c.armed || !c.selected {
			t.Fatalf("spell click %d: %+v", i+1, c)
		}
	}
	t.Logf("spell clicks: %+v", got)
}
func TestBookScrollArmEndsWhenTheStackIsSpent(t *testing.T) {
	v, shots := bookScrollFighter(t)
	x, y := bookCellCentre(t, v, 4)
	sbClick(v, x, y)
	fx, fy := cellPoint(v, 3, 3)
	tapAt(v, fx, fy)
	if v.itemCast == nil {
		t.Fatal("the arm ended while a scroll was still listed")
	}
	v.SetSpellbookCatalog(panelCasterID, make([]SpellEntry, 24))
	if v.itemCast != nil || v.bookLitSpell() != 0 {
		t.Fatalf("the arm stayed after the last scroll: %+v", v.itemCast)
	}
	n := len(*shots)
	tapAt(v, fx, fy)
	if len(*shots) != n {
		t.Fatal("a click after the stack was spent issued a cast")
	}
}
func TestBookScrollArmKeptOnMissEndedByRightClickOneShotWhenClosed(t *testing.T) {
	v, shots := bookScrollFighter(t)
	x, y := bookCellCentre(t, v, 4)
	sbClick(v, x, y)
	ox, oy := cellPoint(v, 0, 0)
	tapAt(v, ox, oy)
	if v.itemCast == nil {
		t.Fatal("a click with no valid target dropped the arm")
	}
	fx, fy := cellPoint(v, 3, 3)
	rightUpAt(v, fx, fy)
	if v.itemCast != nil || len(v.sel) == 0 {
		t.Fatalf("right click: arm %+v sel %v", v.itemCast, v.sel)
	}
	n := len(*shots)
	tapAt(v, fx, fy)
	if len(*shots) != n {
		t.Fatal("a click after the right click cast")
	}

	v, shots = bookScrollFighter(t)
	x, y = bookCellCentre(t, v, 4)
	sbClick(v, x, y)
	v.toggleHudPanel(hudPanelBook)
	tapAt(v, fx, fy)
	if len(*shots) != 1 || v.itemCast != nil {
		t.Fatalf("closed book: shots %d arm %+v, want one cast and the arm spent", len(*shots), v.itemCast)
	}
}
func enemyClickOrders(t *testing.T, scroll bool, victimCol int) (ords []order, sink int) {
	t.Helper()
	var v *Viewer
	if scroll {
		var shots *[]bookScrollShot
		v, shots = bookScrollFighter(t)
		x, y := bookCellCentre(t, v, 4)
		sbClick(v, x, y)
		defer func() { sink = len(*shots) }()
	} else {
		var a *App
		a, v = castCursorApp(t)
		pressPanelKey(t, a, "space")
		chooseSpell(t, v, castTarget)
	}
	v.SetEntities([]MapEntity{
		{ID: panelCasterID, Owner: 1, Cell: image.Pt(3, 3), Life: LifeAlive, SpellStateKnown: true, CastCapable: true},
		{ID: panelVictimID, Owner: 2, Cell: image.Pt(victimCol, 3), Life: LifeAlive},
	})
	x, y := cellPoint(v, victimCol, 3)
	ords, _ = tapAt(v, x, y)
	return ords, 0
}

func TestScrollClickOnAnEnemySendsOnlyWhatASpellClickSends(t *testing.T) {
	for _, col := range []int{9, 4} {
		sp, _ := enemyClickOrders(t, false, col)
		sc, _ := enemyClickOrders(t, true, col)
		_, n := enemyClickOrders(t, true, col)
		t.Logf("enemy at col %d: spell orders %+v, scroll orders %+v, scroll sink calls %d", col, sp, sc, n)
		for _, o := range sc {
			if o.kind == orderKindMove {
				t.Fatalf("enemy at col %d: the scroll click sent a move %+v", col, o)
			}
		}
	}
}
