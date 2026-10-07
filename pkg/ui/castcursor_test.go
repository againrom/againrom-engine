package ui

import (
	"image"
	"slices"
	"testing"
)

const (
	castTarget         = 1  // unit-target
	castArea           = 2  // area
	castSelf           = 18 // self
	castSacrifice      = 4  // self, ordered at a cell
	castPlainSacrifice = 5  // loss control: the same entry without the self flag
)

func castKindBook() []SpellEntry {
	return []SpellEntry{
		{ID: castTarget, Name: "Fire Arrow"},
		{ID: castArea, Name: "Fire Ball", PointTarget: true},
		{ID: castSelf, Name: "Shield", SelfOnly: true},
		{ID: castSacrifice, Name: "Fire Sacrifice", PointTarget: true, SelfOnly: true},
		{ID: castPlainSacrifice, Name: "Plain Area", PointTarget: true},
	}
}

func castCursorApp(t *testing.T) (*App, *Viewer) {
	t.Helper()
	a, v := panelCasterApp(t)
	v.SetSpellbook(panelCasterID, castKindBook())
	return a, v
}

func hoverFrame(t *testing.T, a *App, v *Viewer, p image.Point) {
	t.Helper()
	x, y, err := v.frameToWindow(p, "hover")
	if err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessPointer("hover", x, y); err != nil {
		t.Fatal(err)
	}
}

func shownCursor(v *Viewer) string {
	if v.attackShown() {
		return "attack"
	}
	return v.mapCursorName()
}

func cursorAtCell(t *testing.T, a *App, v *Viewer, col, row int) string {
	t.Helper()
	x, y := cellPoint(v, col, row)
	hoverFrame(t, a, v, image.Pt(x, y))
	return shownCursor(v)
}

func chooseSpell(t *testing.T, v *Viewer, id uint32) {
	t.Helper()
	book := v.spellbook
	for i, e := range book {
		if e.ID == id {
			x, y := sbEntryPoint(t, v, book, i)
			sbClick(v, x, y)
			if v.selectedSpell != id {
				t.Fatalf("setup: spell %d chosen, want %d", v.selectedSpell, id)
			}
			return
		}
	}
	t.Fatalf("setup: spell %d is not in the book", id)
}

var castCells = struct{ caster, other, ground [2]int }{[2]int{3, 3}, [2]int{7, 3}, [2]int{10, 8}}

func TestCursorByKindWithTheBookOpenByHand(t *testing.T) {
	cases := []struct {
		name                  string
		spell                 uint32
		caster, other, ground string
	}{
		{"target", castTarget, "cast", "cast", "move"},
		{"area", castArea, "cast", "cast", "cast"},
		{"self", castSelf, "cast", "move", "move"},
		{"self at a cell", castSacrifice, "cast", "move", "move"},
		{"loss control: area without the self flag", castPlainSacrifice, "cast", "cast", "cast"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			a, v := castCursorApp(t)
			pressPanelKey(t, a, "space")
			chooseSpell(t, v, c.spell)
			if !v.spellModeLive() || v.castOnce {
				t.Fatalf("setup: live %v once %v", v.spellModeLive(), v.castOnce)
			}
			for _, p := range []struct {
				where string
				cell  [2]int
				want  string
			}{{"caster", castCells.caster, c.caster}, {"other unit", castCells.other, c.other}, {"ground", castCells.ground, c.ground}} {
				got := cursorAtCell(t, a, v, p.cell[0], p.cell[1])
				if (got == "cast") != (p.want == "cast") {
					t.Fatalf("%s over %s: cursor %q, want cast=%v", c.name, p.where, got, p.want == "cast")
				}
				x, y := cellPoint(v, p.cell[0], p.cell[1])
				if click := v.gestureCursorAt(x, y); click != got {
					t.Fatalf("%s over %s: click is made under %q, drawn %q", c.name, p.where, click, got)
				}
			}
		})
	}
}

func TestSelfSpellCastsOnlyOnTheCasterAndTargetSpellOnlyOnAUnit(t *testing.T) {
	a, v := castCursorApp(t)
	pressPanelKey(t, a, "space")

	chooseSpell(t, v, castSelf)
	x, y := cellPoint(v, castCells.caster[0], castCells.caster[1])
	ords, ok := tapAt(v, x, y)
	if !ok || len(ords) != 1 || ords[0].kind != orderKindCast || ords[0].victim != panelCasterID || ords[0].cell {
		t.Fatalf("self spell on the caster: %+v", ords)
	}
	x, y = cellPoint(v, castCells.other[0], castCells.other[1])
	if ords, _ := tapAt(v, x, y); len(ords) != 0 || !slices.Equal(v.sel, selection{panelVictimID}) {
		t.Fatalf("self spell on another unit did not fall to the ordinary select: orders %+v sel %v", ords, v.sel)
	}
	v.sel = selection{panelCasterID}

	chooseSpell(t, v, castTarget)
	x, y = cellPoint(v, castCells.ground[0], castCells.ground[1])
	if ords, _ := tapAt(v, x, y); len(ords) != 1 || ords[0].kind != orderKindMove {
		t.Fatalf("target spell on open ground: %+v, want the ordinary move", ords)
	}
	x, y = cellPoint(v, castCells.other[0], castCells.other[1])
	if ords, _ := tapAt(v, x, y); len(ords) != 1 || ords[0].kind != orderKindCast || ords[0].victim != panelVictimID {
		t.Fatalf("target spell on a unit: %+v", ords)
	}

	chooseSpell(t, v, castArea)
	x, y = cellPoint(v, castCells.ground[0], castCells.ground[1])
	if ords, _ := tapAt(v, x, y); len(ords) != 1 || ords[0].kind != orderKindCast || !ords[0].cell || ords[0].x != castCells.ground[0] {
		t.Fatalf("area spell on open ground: %+v", ords)
	}
}

func TestSelfSpellOrderedAtACellCastsOnlyOnTheCaster(t *testing.T) {
	a, v := castCursorApp(t)
	pressPanelKey(t, a, "space")
	chooseSpell(t, v, castSacrifice)
	x, y := cellPoint(v, castCells.caster[0], castCells.caster[1])
	ords, ok := tapAt(v, x, y)
	if !ok || len(ords) != 1 || ords[0].kind != orderKindCast || !ords[0].cell || ords[0].spell != castSacrifice {
		t.Fatalf("self spell at a cell on the caster: %+v", ords)
	}
	for _, c := range [][2]int{castCells.other, castCells.ground} {
		x, y = cellPoint(v, c[0], c[1])
		if ords, _ := tapAt(v, x, y); len(ords) == 1 && ords[0].kind == orderKindCast {
			t.Fatalf("self spell at a cell cast away from the caster at %v: %+v", c, ords)
		}
		v.sel = selection{panelCasterID}
	}
}

func TestAreaCursorStandsOverUnseenGroundAndTargetCursorNeedsASeenUnit(t *testing.T) {
	a, v := castCursorApp(t)
	v.SetLocalOwner(1)
	v.SetEntities([]MapEntity{
		{ID: panelCasterID, Owner: 1, Cell: image.Pt(3, 3), Life: LifeAlive, CastCapable: true},
		{ID: panelVictimID, Owner: 2, Hostile: true, Cell: image.Pt(7, 3), Life: LifeAlive},
	})
	v.SetFog(make([]byte, 60*60), 60, 60)
	pressPanelKey(t, a, "space")

	chooseSpell(t, v, castArea)
	if got := cursorAtCell(t, a, v, castCells.ground[0], castCells.ground[1]); got != "cast" {
		t.Fatalf("area spell over an unseen cell: %q, want cast", got)
	}
	chooseSpell(t, v, castTarget)
	if got := cursorAtCell(t, a, v, castCells.other[0], castCells.other[1]); got == "cast" {
		t.Fatalf("target spell over a unit the party cannot see: %q", got)
	}
	plane := make([]byte, 60*60)
	for i := range plane {
		plane[i] = FogVisible
	}
	v.SetFog(plane, 60, 60)
	if got := cursorAtCell(t, a, v, castCells.other[0], castCells.other[1]); got != "cast" {
		t.Fatalf("target spell over a seen hostile: %q, want cast", got)
	}
}

func TestChosenSpellIsNeverAMagicCursorWithTheBookClosedAndNoHook(t *testing.T) {
	for _, spell := range []uint32{castTarget, castArea, castSelf} {
		a, v := castCursorApp(t)
		pressPanelKey(t, a, "space")
		chooseSpell(t, v, spell)
		pressPanelKey(t, a, "space")
		if v.hudShown(hudPanelBook) || v.spellModeLive() || v.selectedSpell != spell {
			t.Fatalf("spell %d setup: book %v live %v spell %d", spell, v.hudShown(hudPanelBook), v.spellModeLive(), v.selectedSpell)
		}
		for _, cell := range [][2]int{castCells.caster, castCells.other, castCells.ground} {
			if got := cursorAtCell(t, a, v, cell[0], cell[1]); got == "cast" {
				t.Fatalf("spell %d, book closed, cell %v: the cursor is the magic cursor", spell, cell)
			}
		}
		pressPanelKey(t, a, "c")
		pressPanelKey(t, a, "book")
		if !v.spellModeLive() {
			t.Fatalf("spell %d: control hook not armed", spell)
		}
		var any bool
		for _, cell := range [][2]int{castCells.caster, castCells.other, castCells.ground} {
			any = any || cursorAtCell(t, a, v, cell[0], cell[1]) == "cast"
		}
		if !any {
			t.Fatalf("spell %d: the armed hook shows no magic cursor anywhere", spell)
		}
	}
}

func TestCastHookKeepsItsCursorByKindAfterTheBookCloses(t *testing.T) {
	cases := []struct {
		name   string
		spell  uint32
		cell   [2]int
		castOn bool
	}{
		{"target on a unit", castTarget, castCells.other, true},
		{"target on ground", castTarget, castCells.ground, false},
		{"area on ground", castArea, castCells.ground, true},
		{"self on the caster", castSelf, castCells.caster, true},
		{"self on ground", castSelf, castCells.ground, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			a, v := castCursorApp(t)
			pressPanelKey(t, a, "space")
			chooseSpell(t, v, c.spell)
			pressPanelKey(t, a, "space")
			pressPanelKey(t, a, "c")
			if !v.hudShown(hudPanelBook) || !v.castOnce {
				t.Fatalf("C: book %v once %v", v.hudShown(hudPanelBook), v.castOnce)
			}
			pressPanelKey(t, a, "book")
			if v.hudShown(hudPanelBook) || !v.castOnce || !v.spellModeLive() {
				t.Fatalf("closing the book: book %v once %v live %v", v.hudShown(hudPanelBook), v.castOnce, v.spellModeLive())
			}
			if got := cursorAtCell(t, a, v, c.cell[0], c.cell[1]); (got == "cast") != c.castOn {
				t.Fatalf("book closed after C, cursor %q, want cast=%v", got, c.castOn)
			}
			if !c.castOn {
				x, y := cellPoint(v, c.cell[0], c.cell[1])
				tapAt(v, x, y)
				if !v.castOnce || !v.spellModeLive() {
					t.Fatalf("an ordinary click spent the hook: once %v live %v", v.castOnce, v.spellModeLive())
				}
			}
		})
	}
}

func TestCastHookIsSpentByACastAndByAMiscast(t *testing.T) {
	cast := func(t *testing.T, a *App, v *Viewer) {
		t.Helper()
		pressPanelKey(t, a, "space")
		chooseSpell(t, v, castTarget)
		pressPanelKey(t, a, "space")
		pressPanelKey(t, a, "c")
		pressPanelKey(t, a, "book")
	}
	spent := func(t *testing.T, a *App, v *Viewer, what string) {
		t.Helper()
		if v.castOnce || v.spellModeLive() || v.hudShown(hudPanelBook) || v.selectedSpell != castTarget {
			t.Fatalf("%s: once %v live %v book %v spell %d, want spent, book closed, selection kept",
				what, v.castOnce, v.spellModeLive(), v.hudShown(hudPanelBook), v.selectedSpell)
		}
		if got := cursorAtCell(t, a, v, castCells.other[0], castCells.other[1]); got == "cast" {
			t.Fatalf("%s: the magic cursor stayed", what)
		}
	}

	a, v := castCursorApp(t)
	cast(t, a, v)
	x, y := cellPoint(v, castCells.other[0], castCells.other[1])
	if ords, _ := tapAt(v, x, y); len(ords) != 1 || ords[0].kind != orderKindCast {
		t.Fatalf("cast: %+v", ords)
	}
	spent(t, a, v, "after a cast")

	a, v = castCursorApp(t)
	cast(t, a, v)
	v.SetEntities([]MapEntity{
		{ID: panelCasterID, Cell: image.Pt(3, 3), Life: LifeAlive, CastCapable: true, SpellStateKnown: true},
		{ID: panelVictimID, Cell: image.Pt(7, 3), Life: LifeAlive},
	})
	x, y = cellPoint(v, castCells.other[0], castCells.other[1])
	if got := v.gestureCursorAt(x, y); got != "cast" {
		t.Fatalf("miscast setup: cursor %q", got)
	}
	if ords, _ := tapAt(v, x, y); len(ords) != 0 {
		t.Fatalf("miscast issued %+v", ords)
	}
	spent(t, a, v, "after a miscast")
}

func TestRightClickCancelClosesTheBookWhenCastIsArmedWhoeverOpenedIt(t *testing.T) {
	a, v := castCursorApp(t)
	pressPanelKey(t, a, "space")
	chooseSpell(t, v, castTarget)
	x, y := cellPoint(v, castCells.ground[0], castCells.ground[1])
	rightUpAt(v, x, y)
	if v.hudShown(hudPanelBook) || v.spellArmed || v.selectedSpell != 0 || !slices.Equal(v.sel, selection{panelCasterID}) {
		t.Fatalf("cancel of a hand-opened book with a spell chosen: book %v armed %v spell %d sel %v",
			v.hudShown(hudPanelBook), v.spellArmed, v.selectedSpell, v.sel)
	}
	if _, book := panelsOpen(v); book != v.characterPaneView(false).BookOpen {
		t.Fatal("book icon disagrees with the book")
	}

	a, v = castCursorApp(t)
	pressPanelKey(t, a, "space")
	rightUpAt(v, x, y)
	if !v.hudShown(hudPanelBook) || len(v.sel) != 0 {
		t.Fatalf("cancel with no spell chosen: book %v sel %v, want open and deselected", v.hudShown(hudPanelBook), v.sel)
	}
}
