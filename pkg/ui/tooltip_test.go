package ui

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestTooltipCommandDelayAndSelectedSpellTextInvalidation(t *testing.T) {
	a, _ := acApp(t, acBook(0))
	v := a.flow.viewer
	v.SetFont(messageFont())
	v.words.Command[commandCellCast] = "CAST"
	v.SetEntities([]MapEntity{panelEntity(sbUnitID, "Mage", 50, 100, 4, 4)})
	v.sel = selection{sbUnitID}
	v.selectedSpell = v.spellbook[0].ID
	bar, ok := v.commandPanelBar()
	if !ok {
		t.Fatal("no command panel")
	}
	p := commandCellRects(bar)[commandCellCast].Min.Add(image.Pt(3, 3))
	in := appInput{CursorX: p.X, CursorY: p.Y, Viewer: Input{CursorX: p.X, CursorY: p.Y}}
	start := time.Unix(100, 0)
	a.step(in, start)
	a.step(in, start.Add(500*time.Millisecond))
	if _, _, ok := v.tooltipPresent(); !ok || v.tooltipTarget().kind != tooltipCommand {
		t.Fatal("command did not use shared clock", v.tooltipTarget())
	}
	v.spellbook[0].Name = "Renamed spell"
	if _, _, ok := v.tooltipPresent(); ok {
		t.Fatal("command retained old spell name's wait")
	}
}

func TestTooltipSchoolUsesBothClassesAndSemanticMaskOrder(t *testing.T) {
	codes := [2][5]uint8{{0xff, 0x9e, 0xd2, 0x87, 0x37}, {0x87, 0x37, 0xff, 0xd2, 0x9e}}
	for class := range codes {
		for slot, code := range codes[class] {
			panel := SchoolPanelRect(class)
			art := &TownSchoolArt{}
			art.Masks[class] = solidMask(panel.Dx(), panel.Dy(), code)
			cells := make([]TownSurfaceCell, 10)
			for i := range cells {
				cells[i].Enabled = true
			}
			s := &fakeSchoolAnimator{}
			s.view = TownSurfaceView{Kind: TownSurfaceSchool, SchoolArt: art, SchoolClass: class, Cells: cells, Font: messageFont()}
			a := newTestApp(t, nil, nil)
			a.Layout(640, 480)
			a.SetTown(s)
			a.flow.showTown("")
			index := 171 + class*5 + slot
			a.flow.words.Hover[index] = fmt.Sprintf("skill %d", index)
			a.tooltipPoint = panel.Min.Add(image.Pt(3, 3))
			if got := a.tooltipTarget(); got.key() == "" || got.lines[0] != a.flow.words.Hover[index] {
				t.Fatalf("class%d slot%d: wrong tooltip %+v", class, slot, got)
			}
		}
	}
}

func TestTooltipShopStockSelectorsUseTheirInstalledTextSlots(t *testing.T) {
	w := AuthoredWords()
	v := ShopScreenView{Font: messageFont(), Live: [4]bool{true, true, true, true}, Chosen: 0}
	for i, r := range shopShelfPickRects {
		w.Hover[62+i] = fmt.Sprintf("stock %d", i)
		tip := shopTooltip(v, w, r.Min.Add(image.Pt(4, 4)))
		if tip.key() == "" || tip.lines[0] != w.Hover[62+i] {
			t.Fatal(i, tip)
		}
	}
}

type tooltipShopFixture struct {
	view     ShopScreenView
	dialogue bool
}

func (*tooltipShopFixture) Header() string                               { return "shop" }
func (*tooltipShopFixture) Rows() []TownRow                              { return nil }
func (*tooltipShopFixture) Footer() []string                             { return nil }
func (*tooltipShopFixture) Choose(int) TownAction                        { return TownAction{} }
func (*tooltipShopFixture) Back() bool                                   { return false }
func (*tooltipShopFixture) AtTownShop() bool                             { return true }
func (s *tooltipShopFixture) ShopScreen() ShopScreenView                 { return s.view }
func (*tooltipShopFixture) ShopClick(ShopControl) TownAction             { return TownAction{} }
func (*tooltipShopFixture) ShopScroll(ShopWheelRegion, int)              {}
func (*tooltipShopFixture) ShopDrag(ShopControl, ShopControl) TownAction { return TownAction{} }
func (*tooltipShopFixture) ShopSuppressDoll(int)                         {}
func (s *tooltipShopFixture) TownDialogue() (*image.RGBA, bool)          { return nil, s.dialogue }
func (*tooltipShopFixture) TownDialogueButton() (image.Rectangle, bool) {
	return image.Rectangle{}, false
}
func (*tooltipShopFixture) AdvanceTownDialogue() TownAction { return TownAction{} }

func TestTooltipAppShopPaintWaitsAndInvalidatesOnDialogue(t *testing.T) {
	a := NewApp("tooltip", nil, nil, nil)
	a.Layout(640, 480)
	shop := &tooltipShopFixture{view: ShopScreenView{Font: messageFont(), Live: [4]bool{true, true, true, true}, Chosen: 0}}
	shop.view.Shelf[0] = ShopCell{Back: 1, Info: []string{"Sword", "Damage 5"}}
	a.flow.town, a.flow.screen = shop, ScreenTown
	a.flow.townList = NewPicker(nil)
	p := ShopShelfCellRect(0).Min.Add(image.Pt(10, 10))
	in := appInput{CursorX: p.X, CursorY: p.Y}
	start := time.Unix(100, 0)
	a.step(in, start)
	before, _, err := a.HeadlessFrame()
	if err != nil {
		t.Fatal(err)
	}
	if a.tooltip.key == "" {
		t.Fatal("fixture has no tooltip target")
	}
	a.step(in, start.Add(499*time.Millisecond))
	early, _, err := a.HeadlessFrame()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before.Pix, early.Pix) {
		t.Fatal("tooltip painted before delay")
	}
	a.step(in, start.Add(500*time.Millisecond))
	after, _, err := a.HeadlessFrame()
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(before.Pix, after.Pix) {
		t.Fatal("production CPU composition did not paint the delayed tooltip")
	}
	shop.dialogue = true
	if _, _, ok := a.tooltipPresent(after.Bounds()); ok {
		t.Fatal("stale hover over a dialogue")
	}
	a.step(in, start.Add(time.Second))
	shop.dialogue = false
	a.step(in, start.Add(2*time.Second))
	if _, _, ok := a.tooltipPresent(after.Bounds()); ok {
		t.Fatal("closing dialogue skipped delay")
	}
	a.step(in, start.Add(2500*time.Millisecond))
	if _, _, ok := a.tooltipPresent(after.Bounds()); !ok {
		t.Fatal("return did not reveal again")
	}
	a.Layout(1280, 720)
	if a.tooltip.visible(a.tooltip.key) {
		t.Fatal("resize retained elapsed time")
	}
}

func TestTooltipSpellSharesTheAppController(t *testing.T) {
	a, _ := acApp(t, acBook(0))
	a.SetTooltipDelayPreference(300, nil)
	v := a.flow.viewer
	v.SetFont(messageFont())
	x, y := sbEntryPoint(t, v, v.spellbook, 0)
	in := appInput{CursorX: x, CursorY: y, Viewer: Input{CursorX: x, CursorY: y}}
	start := time.Unix(100, 0)
	a.step(in, start)
	if v.tooltip != &a.tooltip {
		t.Fatal("viewer did not share the application timer")
	}
	a.step(in, start.Add(299*time.Millisecond))
	if _, _, ok := v.tooltipPresent(); ok {
		t.Fatal("spell appeared early")
	}
	a.step(in, start.Add(300*time.Millisecond))
	if _, _, ok := v.tooltipPresent(); !ok {
		t.Fatalf("spell did not appear: screen=%v controller=%+v target=%+v cursor=%d,%d", a.flow.screen, a.tooltip, v.tooltipTarget(), v.cursorX, v.cursorY)
	}
	if v.tooltipTarget().kind != tooltipSpell {
		t.Fatal("fixture did not hit a spell")
	}
	// An unchanged pointer over an updated spell must not reuse the prior text.
	v.spellbook[0].Info = []string{"Changed spell"}
	if _, _, ok := v.tooltipPresent(); ok {
		t.Fatal("old elapsed time exposed changed content")
	}
	a.step(in, start.Add(time.Second))
	if _, _, ok := v.tooltipPresent(); ok {
		t.Fatal("new spell text skipped its wait")
	}
	a.SetTooltipDelayPreference(0, nil)
	a.step(in, start.Add(2*time.Second))
	if _, _, ok := v.tooltipPresent(); !ok {
		t.Fatal("zero delay did not reach the spell")
	}
	a.step(appInput{CursorX: x, CursorY: y, Unfocused: true}, start.Add(3*time.Second))
	if _, _, ok := v.tooltipPresent(); ok {
		t.Fatal("tooltip survived focus loss")
	}
}

func TestTooltipDelayEveryChoiceAndBoundary(t *testing.T) {
	start := time.Unix(100, 0)
	for _, ms := range []int{0, 100, 200, 300, 400, 500} {
		h := tooltipController{delay: ms}
		p := image.Pt(10, 20)
		h.observe(start, p, "one", false)
		if ms > 0 {
			h.observe(start.Add(time.Duration(ms-1)*time.Millisecond), p, "one", false)
			if h.visible("one") {
				t.Fatalf("%dms appeared early", ms)
			}
		}
		h.observe(start.Add(time.Duration(ms)*time.Millisecond), p, "one", false)
		if !h.visible("one") {
			t.Fatalf("%dms did not appear at its threshold", ms)
		}
		h.observe(start.Add(time.Duration(ms)*time.Millisecond+25*time.Second), p, "one", false)
		if h.visible("one") {
			t.Fatalf("%dms did not expire", ms)
		}
	}
}

func TestTooltipMovementTargetBlockingAndClockReset(t *testing.T) {
	start, p := time.Unix(100, 0), image.Pt(10, 20)
	h := tooltipController{delay: 500}
	h.observe(start, p, "item", false)
	h.observe(start.Add(time.Second), p, "item", false)
	if !h.visible("item") {
		t.Fatal("setup did not reveal")
	}
	for _, change := range []struct {
		key     string
		point   image.Point
		blocked bool
	}{
		{"item", p.Add(image.Pt(1, 0)), false}, // same widget, another pixel
		{"spell", p, false},                    // target changes without moving
		{"spell-new-value", p, false},          // content changes without moving
		{"spell-new-value", p, true},           // modal, focus or drag
		{"", p, false},                         // no target
	} {
		h.observe(start.Add(2*time.Second), change.point, change.key, change.blocked)
		if h.visible(change.key) {
			t.Fatalf("stale tooltip after %+v", change)
		}
	}
	h.observe(start.Add(3*time.Second), p, "item", false)
	h.observe(start.Add(3500*time.Millisecond), p, "item", false)
	if !h.visible("item") {
		t.Fatal("return did not start a fresh wait")
	}
	h.observe(start, p, "item", false)
	if h.visible("item") {
		t.Fatal("clock reversal kept old elapsed time")
	}
	h.setDelay(0)
	if h.visible("item") {
		t.Fatal("setting change kept previous target")
	}
	h.observe(start, p, "item", false)
	if !h.visible("item") {
		t.Fatal("zero delay was treated as disabled")
	}
}

func TestTooltipMenuCyclesSixChoicesAndKeepsLiveValueOnWriteFailure(t *testing.T) {
	a := newTestApp(t, appRows(1), okLoader(t))
	var stored []int
	a.SetTooltipDelayPreference(500, func(ms int) error {
		stored = append(stored, ms)
		if ms == 200 {
			return errors.New("read only")
		}
		return nil
	})
	a.flow.openGameMenu(ScreenTown)
	if err := a.HeadlessGameMenuAction("game-options"); err != nil {
		t.Fatal(err)
	}
	for _, want := range []int{0, 100, 200, 300, 400, 500} {
		if err := a.HeadlessGameMenuAction("tooltip-delay"); err != nil {
			t.Fatal(err)
		}
		if got := a.TooltipDelay(); got != want {
			t.Fatalf("got %d, want %d", got, want)
		}
		if want == 200 && !strings.Contains(a.flow.msg, "not saved") {
			t.Fatal("write failure was hidden")
		}
	}
	if !reflect.DeepEqual(stored, []int{0, 100, 200, 300, 400, 500}) {
		t.Fatal(stored)
	}
}

func TestTooltipItemUsesTheProductionDelayAndModalGate(t *testing.T) {
	s := InventorySubject{ID: 7}
	s.Slots[0] = solidPic(4, 4, color.RGBA{A: 255})
	s.SlotInfo[0] = []string{"Sword", "Damage 5"}
	v := itemPopupViewer(t, s)
	v.SetTooltipDelay(500)
	box, ok := v.wornBox()
	if !ok {
		t.Fatal("no worn box")
	}
	p := wornSlotRects()[0].Add(box.Min).Min.Add(image.Pt(2, 2))
	v.cursorX, v.cursorY, v.hasCursor = p.X, p.Y, true
	start := time.Unix(100, 0)
	in := Input{CursorX: p.X, CursorY: p.Y}
	v.updateTooltip(in, start)
	if _, _, ok := v.tooltipPresent(); ok {
		t.Fatal("item appeared immediately")
	}
	v.updateTooltip(in, start.Add(500*time.Millisecond))
	if _, _, ok := v.tooltipPresent(); !ok {
		t.Fatal("item did not appear")
	}
	v.menuUp = true
	if _, _, ok := v.tooltipPresent(); ok {
		t.Fatal("item leaked over menu before next update")
	}
	v.updateTooltip(in, start.Add(time.Second))
	v.menuUp = false
	v.updateTooltip(in, start.Add(2*time.Second))
	if _, _, ok := v.tooltipPresent(); ok {
		t.Fatal("closing menu reused old wait")
	}
}

func TestTooltipCardHitsPaintedFieldsAndKeepsDisclosure(t *testing.T) {
	font := panelFont()
	w := AuthoredWords()
	w.Hover[155], w.Hover[156] = "BODY#EXPLANATION", "REACTION"
	l := PanelLayout{Size: image.Pt(160, 100), Pad: image.Pt(4, 4), Rows: []PanelRow{
		{Field: PanelFieldBody, Label: "BODY", Right: &PanelCell{Field: PanelFieldReaction, Label: "REACTION"}},
	}}
	s := PanelSubject{ID: 9, Char: UnitCharacter{Known: true}, DetailSet: true, DetailLevel: 10}
	v := TownCharacterView{Subject: s, HasSubject: true, Statistics: true, Font: font, CardLayout: &l, PaneRect: image.Rect(0, 0, 160, 100)}
	report := CharacterPanelReport(l, font, s)
	if len(report) != 1 {
		t.Fatal(report)
	}
	r := report[0]
	left := characterStatsTooltip(v, w, r.At)
	if !reflect.DeepEqual(left.lines, []string{"BODY", "EXPLANATION"}) {
		t.Fatal(left)
	}
	right := characterStatsTooltip(v, w, r.At.Add(image.Pt(r.RightX, 0)))
	if !reflect.DeepEqual(right.lines, []string{"REACTION"}) {
		t.Fatal(right)
	}
	v.Subject.DetailLevel = 0
	if got := characterStatsTooltip(v, w, r.At); got.key() != "" {
		t.Fatal("hidden stat got a tooltip", got)
	}
}

// TEXT-HOVERTEXT-052: the card helper's monster spell list joins the
// installed heading with every known spell's name, over the same SPELLCASTER
// cell the shipped card already draws — and only for a creature, since no
// claim binds this list to a person's own row.
func TestTooltipMonsterSpellListJoinsHeadingAndKnownSpellNames(t *testing.T) {
	font := panelFont()
	w := AuthoredWords()
	w.Hover[192] = "Spells: "
	w.ItemSpellNames[3] = "Wall of Fire"
	w.ItemSpellNames[6] = "Heal"
	l := PanelLayout{Size: image.Pt(160, 100), Pad: image.Pt(4, 4), Rows: []PanelRow{
		{Field: PanelFieldWeight, Label: "WEIGHT", Right: &PanelCell{Field: PanelFieldSpellcaster, Label: "SPELLCASTER"}},
	}}
	s := PanelSubject{ID: 4, WeightKnown: true,
		Char:          UnitCharacter{Band: CharacterBandCreature},
		OriginalPanel: OriginalPanelActor{Known: true, XPValue: 1},
		DetailSet:     true, DetailLevel: 7,
		KnownSpells: 1<<3 | 1<<6,
	}
	v := TownCharacterView{Subject: s, HasSubject: true, Statistics: true, Font: font, CardLayout: &l, PaneRect: image.Rect(0, 0, 160, 100)}
	report := CharacterPanelReport(l, font, s)
	if len(report) != 1 {
		t.Fatal(report)
	}
	r := report[0]
	at := r.At.Add(image.Pt(r.RightX, 0))
	got := characterStatsTooltip(v, w, at)
	if want := []string{"Spells: Wall of Fire, Heal"}; !reflect.DeepEqual(got.lines, want) {
		t.Fatalf("creature spellcaster hint = %v, want %v", got.lines, want)
	}
	// A person spellcaster's own row keeps its existing absence: the claim
	// binds this composed list to a monster, not to a hero's spellbook,
	// which the mission spell bar already states.
	v.Subject.Char.Band = CharacterBandPerson
	if got := characterStatsTooltip(v, w, at); got.key() != "" {
		t.Fatalf("person spellcaster row got a monster hint: %v", got)
	}
	// A creature with no known spell keeps the row's plain absent hint too.
	v.Subject.Char.Band = CharacterBandCreature
	v.Subject.KnownSpells = 0
	if got := characterStatsTooltip(v, w, at); got.key() != "" {
		t.Fatalf("creature with no known spell got a hint: %v", got)
	}
}

func TestTooltipWrappingKeepsAuthoredLinesAndFitsFrame(t *testing.T) {
	font := messageFont()
	target := tooltipTarget{tooltipText, "text", []string{"ONE#TWO", strings.Repeat("wide ", 40)}, font}
	pic, at, ok := tooltipPicture(target, image.Pt(99, 119), image.Rect(0, 0, 100, 240), nil)
	if !ok || pic.Bounds().Dx() > 100 || at.X < 0 || at.Y < 0 || at.X+pic.Bounds().Dx() > 100 {
		t.Fatal("tooltip escaped frame", at, pic.Bounds())
	}
	if got := tooltipLines("ONE#TWO"); !reflect.DeepEqual(got, []string{"ONE", "TWO"}) {
		t.Fatal(got)
	}
}

func TestTooltipKindsShareCardFontAndBoundedWidth(t *testing.T) {
	font := townShellRosterTestFont()
	for i := range font.Glyphs {
		font.Glyphs[i].Height = 15
	}
	original := font.Height()
	cardFont := townShellRosterTestFont()
	controller := tooltipController{baseFont: cardFont}
	for _, kind := range []uint8{tooltipText, tooltipItem, tooltipSpell, tooltipCommand} {
		for _, frameWidth := range []int{240, 640, 1920} {
			target := tooltipTarget{kind, "long", []string{"TITLE#" + strings.Repeat("word ", 70)}, font}
			pic, at, ok := controller.picture(target, image.Pt(frameWidth-1, 479), image.Rect(0, 0, frameWidth, 480))
			if !ok || pic == nil || pic.Bounds().Dx() > min(320, frameWidth) || at.X < 0 || at.X+pic.Bounds().Dx() > frameWidth {
				t.Fatalf("kind%d width%d escaped its column: %v", kind, frameWidth, at)
			}
			if pic.Bounds().Dy() <= 2*hoverPitch+hoverExtraH {
				t.Fatal("long tooltip did not wrap downward")
			}
		}
	}
	if controller.baseFont != cardFont || font.Height() != original {
		t.Fatal("tooltip replaced the native card font or changed the label font")
	}
}

func TestTooltipAnchorsBottomLeftAtCursorAndStaysInsideEdges(t *testing.T) {
	font := townShellRosterTestFont()
	frame := image.Rect(0, 0, 640, 480)
	for _, kind := range []uint8{tooltipText, tooltipItem, tooltipSpell, tooltipCommand} {
		for _, lines := range [][]string{{"NAME"}, {"NAME", "DAMAGE", "MAGIC"}, {strings.Repeat("long name ", 30)}} {
			target := tooltipTarget{kind, "anchor", lines, font}
			pic, at, ok := tooltipPicture(target, image.Pt(200, 350), frame, nil)
			if !ok || at.X != 200 || at.Y+pic.Bounds().Dy() != 350 {
				t.Fatalf("kind%d: lower-left corner moved away from pointer: %v", kind, at)
			}
			for _, p := range []image.Point{{0, 0}, {639, 0}, {0, 479}, {639, 479}} {
				pic, at, ok := tooltipPicture(target, p, frame, nil)
				if !ok || !pic.Bounds().Add(at).In(frame) {
					t.Fatalf("kind%d cursor%v: popup escaped frame at %v", kind, p, at)
				}
				if p.Y == 0 && at.Y != 0 || p.X == 639 && at.X+pic.Bounds().Dx() != 640 {
					t.Fatalf("kind%d cursor%v: edge correction is wrong: %v", kind, p, at)
				}
			}
		}
	}
}
