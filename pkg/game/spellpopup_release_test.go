package game

// The mission/shop spellbook popup's own composition (R1, TEXT-HOVERTEXT-052):
// a spell's NAME comes from the installed spells.txt row, not the internal
// binary identifier data.LoadSpells reads, and each bound caption comes from
// the installed main.txt row the claim's getter names — never a literal, on
// either root, so a mismatched RU install fails this test exactly as an
// English regression would. This is the release chain's own witness
// (pipeline/check-release-tests.sh); it skips without AGAINROM_ASSETS.

import (
	"bytes"
	"fmt"
	"image"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestReleaseSpellbookPopupComposesFromSpellsTxtAndInstalledLabels(t *testing.T) {
	f := releaseFront(t)
	mainTable := LoadTextTable(f.Archives.Containers, MainTextPath, f.textCode())
	bookTable := LoadTextTable(f.Archives.Containers, SpellBookNamesTextPath, f.textCode())
	if mainTable == nil || bookTable == nil {
		t.Fatal("install carries no main.txt or spells.txt to read independently")
	}
	names := spellNamesFrom(f.Table, f.Words.SpellBookNames)
	rules := mapload.SpellRules(f.Table)
	if len(rules) == 0 {
		t.Fatal("install names no spells at all")
	}

	if bookTable.Lines() != 24 {
		t.Fatalf("spells.txt has %d cells, want 24", bookTable.Lines())
	}
	cellIDs := [...]uint16{1, 2, 3, 4, 5, 23, 24, 16, 15, 14, 13, 12, 6, 7, 8, 9, 10, 25, 26, 22, 21, 20, 19, 18}
	for cell, id := range cellIDs {
		row, ok := bookTable.At(cell)
		if !ok {
			t.Fatalf("spells.txt cell %d has no name", cell)
		}
		want := strings.SplitN(row, "#", 2)[0]
		if got := names[id]; got != want {
			t.Errorf("cell %d spell %d name = %q, want %q", cell, id, got, want)
		}
	}
	if names[23] != strings.SplitN(bookTable.lines[5], "#", 2)[0] || names[6] != strings.SplitN(bookTable.lines[12], "#", 2)[0] {
		t.Error("Bless or Heal shifted away from its book cell")
	}

	label := func(i int) string {
		s, ok := mainTable.At(i)
		if !ok {
			t.Fatalf("main.txt carries no row %d for a caption this popup binds", i)
		}
		return s
	}
	manaLabel := label(spellLabelManaCost)
	rangeLabel := label(spellLabelRange)
	damageLabel := label(spellLabelDamage)
	durationLabel := label(spellLabelDuration)

	rule := rules[0]
	rule.Damaging = true
	rule.DamageMin, rule.DamageMax = 2, 5
	c := sim.SpellCharacteristics{ManaCost: 7, Range: 4, Duration: 3}
	lines := spellInfoLines(rule, c, names[rule.ID], &f.Words)
	want := []string{names[rule.ID], manaLabel + ": 7", damageLabel + ": 2-5", rangeLabel + ": 4", durationLabel + ": " + fmt.Sprintf("%5.1f", 3*0.0625)}
	if strings.Join(lines, "|") != strings.Join(want, "|") {
		t.Fatalf("popup lines = %v, want %v — installed labels, the claim join and order", lines, want)
	}

	// A rule that neither damages nor carries a duration states neither line
	// rather than composing one with an empty value.
	rule.Damaging = false
	rule.DamageMin, rule.DamageMax = 0, 0
	c.Duration = 0
	lines = spellInfoLines(rule, c, names[rule.ID], &f.Words)
	want = []string{names[rule.ID], manaLabel + ": 7", rangeLabel + ": 4"}
	if strings.Join(lines, "|") != strings.Join(want, "|") {
		t.Fatalf("non-damaging, zero-duration popup lines = %v, want %v", lines, want)
	}

	t.Run("mission-hover-and-cast", func(t *testing.T) {
		f := releaseFront(t)
		f.SetDeterministicFrames(true)
		hero := data.Hero{Body: 60, Reaction: 60, Mind: 100, Spirit: 100}
		party := []mapload.PartyMember{{ID: "hero", PlayerCharacter: true, StartingHero: true, Mage: true, Class: 0x18,
			Profile: data.Profile{HealthColumn: true, ManaColumn: true}, Hero: hero,
			KnownSpells: 1 << 23,
			Saved:       &mapload.Saved{Cell: mapload.Cell{X: 29, Y: 50}, HP: 100, MaxHP: 100, Mana: 1000, MaxMana: 1000}}}
		a := f.App("spellbook hover")
		a.Layout(1024, 768)
		a.SetTooltipDelayPreference(0, nil)
		a.SetTooltipFont(f.Font.Value())
		if err := a.OpenMission(f.MissionOpenerWith(10, party)); err != nil {
			t.Fatal(err)
		}
		if err := a.HeadlessKey("0"); err != nil {
			t.Fatal(err)
		}
		id := f.live.mission.ids[0]
		inspectionCentre(f.live, 29, 50)
		if err := a.HeadlessSelectEntity(uint32(id)); err != nil {
			t.Fatal(err)
		}
		if err := a.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
		if _, _, err := a.HeadlessSpellPoint(23); err != nil {
			if err := a.HeadlessKey("book"); err != nil {
				t.Fatal(err)
			}
		}
		x, y, err := a.HeadlessSpellPoint(23)
		if err != nil {
			t.Fatal(err)
		}
		if err := a.HeadlessPointer("hover", x, y); err != nil {
			t.Fatal(err)
		}
		state, pic := a.HeadlessTooltip()
		if state.Target != "spell/23" || !state.Visible || pic == nil {
			t.Fatalf("mission cell 5 hover = %+v", state)
		}
		entity, ok := f.live.entity(id)
		if !ok {
			t.Fatal("selected caster absent")
		}
		var bless sim.SpellRule
		for _, rule := range f.live.world.Spells() {
			if rule.ID == 23 {
				bless = rule
				break
			}
		}
		if bless.ID != 23 {
			t.Fatal("installed rules have no spell 23")
		}
		row, _ := bookTable.At(5)
		expected := spellInfoLines(bless, sim.SpellCharacteristicsFor(sim.Rules{}, entity, bless), strings.SplitN(row, "#", 2)[0], &f.Words)
		wantPic, _, ok := ui.ComposeTooltipHint(expected, f.Font.Value(), image.Point{}, image.Rect(0, 0, 1024, 768), f.HoverBall())
		if !ok || !bytes.Equal(pic.Pix, wantPic.Pix) || pic.Bounds() != wantPic.Bounds() {
			t.Fatal("mission hover did not paint cell 5's installed name")
		}
		for _, edge := range []string{"press", "release"} {
			if err := a.HeadlessPointer(edge, x, y); err != nil {
				t.Fatal(err)
			}
		}
		if _, current, _ := f.live.view.QuickSpellState(); current != 23 {
			t.Fatalf("cell 5 selected spell %d, want ID 23", current)
		}
		x, y, err = a.HeadlessEntityPoint(uint32(id))
		if err != nil {
			t.Fatal(err)
		}
		for _, edge := range []string{"press", "release"} {
			if err := a.HeadlessPointer(edge, x, y); err != nil {
				t.Fatal(err)
			}
		}
		if pending := f.live.pending; len(pending) != 1 || pending[0].Kind != sim.KindCast || pending[0].Y != 23 {
			t.Fatalf("cell 5 cast = %+v, want spell ID 23", pending)
		}
		indices, err := quickSpellsToOriginalIndices([4]uint32{23, 6})
		if err != nil || indices[0] != 5 || indices[1] != 12 {
			t.Fatalf("quick slot indices = %v, %v", indices, err)
		}
		slots, err := quickSpellsFromOriginalIndices(indices[:])
		if err != nil || slots != ([4]uint32{23, 6}) {
			t.Fatalf("quick slot IDs = %v, %v", slots, err)
		}
	})

	t.Run("town-hover", func(t *testing.T) {
		f := releaseFront(t)
		hero := data.Hero{Body: 60, Reaction: 60, Mind: 100, Spirit: 100}
		f.Carried = []mapload.PartyMember{{ID: "hero", PlayerCharacter: true, StartingHero: true, Mage: true, Class: 0x18,
			Profile: data.Profile{HealthColumn: true, ManaColumn: true}, Hero: hero, KnownSpells: 1 << 6}}
		f.Town = NewTown(f.Campaign.Value())
		f.Shop = NewShop(5000)
		f.Shop.Generate(f.Table, 59)
		s := f.TownScreen().(*townScreen)
		s.room = roomShop
		s.ShopClick(ui.ShopControl{Kind: ui.ShopControlBook})
		view := s.ShopScreen()
		if !view.Book || !view.SpellCatalog || len(view.Spells) != 24 || view.Spells[12].ID != 6 {
			t.Fatalf("town book cell 12 = %+v", view.Spells)
		}
		p := image.Pt(24, 365)
		lines, ok := ui.ShopHoverLines(view, p)
		row, _ := bookTable.At(12)
		if !ok || len(lines) == 0 || lines[0] != strings.SplitN(row, "#", 2)[0] {
			t.Fatalf("town cell 12 hover = %q, want spells.txt row 12", lines)
		}
		a := f.App("town spellbook hover")
		a.Layout(640, 480)
		a.SetTooltipDelayPreference(0, nil)
		a.SetTooltipFont(f.Font.Value())
		a.SetTown(s)
		a.SetSaveSeams(nil, func() []ui.SaveEntry { return []ui.SaveEntry{{Name: "town", Label: "town"}} }, func(string) (ui.MapOpener, bool, error) { return nil, true, nil })
		if err := a.HeadlessKey("load"); err != nil {
			t.Fatal(err)
		}
		if err := a.HeadlessActivate("town"); err != nil {
			t.Fatal(err)
		}
		if err := a.HeadlessPointer("hover", p.X, p.Y); err != nil {
			t.Fatal(err)
		}
		state, pic := a.HeadlessTooltip()
		if !state.Visible || pic == nil || !strings.HasPrefix(state.Target, "shop-item/") {
			t.Fatalf("town hover route = %+v", state)
		}
		wantPic, _, ok := ui.ComposeTooltipHint(lines, f.Font.Value(), p, image.Rect(0, 0, 640, 480), f.HoverBall())
		if !ok || !bytes.Equal(pic.Pix, wantPic.Pix) || pic.Bounds() != wantPic.Bounds() {
			t.Fatal("town hover did not paint cell 12's installed name")
		}
	})
}
