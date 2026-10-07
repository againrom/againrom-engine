package ui

import (
	"image"
	"testing"
)

func TestQuickSpellPopulationCapabilityAndCastFiltering(t *testing.T) {
	_, v, slots := quickSpellApp(t)
	entities := []MapEntity{
		{ID: 5, Owner: 1, Life: LifeAlive, Cell: image.Pt(3, 3), SpellStateKnown: true, KnownSpells: 1 << 16},
		{ID: 6, Owner: 1, Life: LifeAlive, SpellStateKnown: true, KnownSpells: 1 << 1, CastCapable: true},
		{ID: 7, Owner: 1, Life: LifeAlive, SpellStateKnown: true, KnownSpells: 1 << 16, CastCapable: true},
		{ID: 9, Owner: 2, Life: LifeAlive, Cell: image.Pt(7, 3)},
	}
	v.SetEntities(entities)
	v.sel = selection{5, 6, 7}
	v.SetSpellbook(5, []SpellEntry{{ID: 16}})
	v.Camera().X, v.Camera().Y = 0, 0
	v.Camera().Clamp()
	*slots = [4]uint32{16}
	v.toggleHudPanel(hudPanelBook)
	v.quickSpell(0, false, -1, -1)
	if !v.spellArmed || !v.commandCastActive() {
		t.Fatal("later eligible selected object did not supply Cast capability")
	}
	x, y := cellPoint(v, 7, 3)
	orders, ok := tapAt(v, x, y)
	if !ok || len(orders) != 2 || orders[0].entity != 5 || orders[1].entity != 7 {
		t.Fatalf("book producer did not filter each selected object: %+v", orders)
	}
	for _, order := range orders {
		if order.kind != orderKindCast || order.spell != 16 || order.victim != 9 {
			t.Fatalf("wrong real-ID cast %+v", order)
		}
	}
	// The cast now keeps its mode. Start the ownership admission probe from
	// an explicitly cancelled mode, rather than relying on a cast to clear it.
	v.cancelMapCommand()
	entities[0].Owner = 2
	v.SetEntities(entities)
	v.quickSpell(0, false, -1, -1)
	if v.spellArmed {
		t.Fatal("a later owned caster bypassed primary ownership")
	}
	entities[0].Owner = 1
	entities[1].CastCapable, entities[2].CastCapable = false, false
	entities[0].MaxMana = 999
	v.SetEntities(entities)
	v.quickSpell(0, false, -1, -1)
	if v.spellArmed {
		t.Fatal("a known production projection substituted mana for capability")
	}
	var many []MapEntity
	for i := range 300 {
		many = append(many, MapEntity{ID: uint32(i + 1), SpellStateKnown: true, KnownSpells: 1 << 16})
	}
	if got := bookCasters(many, 16); len(got) != 253 || got[252].ID != 253 {
		t.Fatal("book producer population does not stop at253")
	}
	if len(bookCasters(many, 0)) != 0 || len(bookCasters(many, 31)) != 0 {
		t.Fatal("empty or unavailable current generated a caster")
	}
}

func TestQuickSpellDoesNotReplaceItemOperand(t *testing.T) {
	_, v, slots := quickSpellApp(t)
	v.SetEntities([]MapEntity{{ID: 5, Owner: 1, Life: LifeAlive, Cell: image.Pt(3, 3), MaxMana: 100}, {ID: 9, Owner: 2, Life: LifeAlive, Cell: image.Pt(7, 3)}})
	v.Camera().X, v.Camera().Y = 0, 0
	v.Camera().Clamp()
	called := false
	v.SetItemCastSink(func(owner uint32, index int, key string, victim uint32, x, y int, cell bool) {
		called = owner == 5 && index == 7 && key == "item-key" && victim == 9 && !cell
	})
	if !v.ArmItemCast(5, 7, 6, "item-key") {
		t.Fatal("item fixture did not arm")
	}
	*slots = [4]uint32{89}
	v.toggleHudPanel(hudPanelBook)
	v.quickSpell(0, false, -1, -1)
	if v.itemCast == nil || v.selectedSpell != 89 {
		t.Fatal("quick selection destroyed the independent item context")
	}
	x, y := cellPoint(v, 7, 3)
	orders, _ := tapAt(v, x, y)
	if !called || len(orders) != 0 {
		t.Fatal("item position was translated as a book ordinal or book order emitted")
	}
}

func TestQuickSpellFixedCatalogFitsMinimumAndOrdinaryFrames(t *testing.T) {
	a, v, _ := quickSpellApp(t)
	entries := make([]SpellEntry, 24)
	for i := range entries {
		entries[i] = SpellEntry{ID: uint32(i + 1), Unavailable: true}
	}
	for _, area := range []image.Point{{640, 480}, {1024, 768}} {
		a.Layout(area.X, area.Y)
		v.SetSpellbookCatalog(5, entries)
		bar, cols, ok := v.spellbookBar()
		cells := bookCellRects(bar, cols)
		if !ok || cols != 12 || len(cells) != 24 {
			t.Fatalf("%v catalog missing cells", area)
		}
		for i, cell := range cells {
			if !cell.In(bar) {
				t.Fatalf("%v cell%d clipped by book", area, i)
			}
			if index, ok := v.spellbookEntryAt(cell.Min.X+2, cell.Min.Y+2); !ok || index != i {
				t.Fatalf("%v unavailable cell%d cannot be hovered", area, i)
			}
			x, y, err := a.HeadlessSpellPoint(entries[i].ID)
			if err != nil {
				t.Fatal(err)
			}
			fx, fy := v.windowToFrame(x, y)
			if index, ok := v.spellbookEntryAt(fx, fy); !ok || index != i {
				t.Fatalf("%v downscaled cell%d oracle resolves the wrong cell", area, i)
			}
		}
	}
}
