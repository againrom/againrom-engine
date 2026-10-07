package ui

import (
	"bytes"
	"image"
	"strings"
	"testing"
	"time"
)

func TestQuickSpellFeedbackDoesNotConflateCurrentArmOrSlot(t *testing.T) {
	_, v, slots := quickSpellApp(t)
	base, at, ok := v.spellbookPresent()
	if !ok {
		t.Fatal("no composed book")
	}
	*slots = [4]uint32{89}
	marked, sameAt, ok := v.spellbookPresent()
	if !ok || at != sameAt || bytes.Equal(base.Pix, marked.Pix) {
		t.Fatal("F5 binding mark is absent or moves book")
	}
	bar, cols, _ := v.spellbookBar()
	cell := bookCellRects(bar, cols)[1].Sub(bar.Min)
	for y := 0; y < base.Bounds().Dy(); y++ {
		for x := 0; x < base.Bounds().Dx(); x++ {
			if !image.Pt(x, y).In(cell) && base.RGBAAt(x, y) != marked.RGBAAt(x, y) {
				t.Fatal("binding mark changed a different cell")
			}
		}
	}
	// Construct the literal utility label independently from the binding
	// compositor. Swapping F5/F6 or omitting the label fails this comparison.
	w, h := v.font.Measure("F5")
	expected := image.NewRGBA(image.Rect(0, 0, w+2, h+2))
	for y := 0; y < expected.Bounds().Dy(); y++ {
		for x := 0; x < expected.Bounds().Dx(); x++ {
			expected.SetRGBA(x, y, spellbookFill)
		}
	}
	v.font.Draw(expected, "F5", 1, 1, spellbookText)
	for y := 0; y < expected.Bounds().Dy(); y++ {
		for x := 0; x < expected.Bounds().Dx(); x++ {
			if expected.RGBAAt(x, y) != marked.RGBAAt(cell.Max.X-w-3+x, cell.Max.Y-h-3+y) {
				t.Fatal("bound cell does not carry literal F5 bitmap")
			}
		}
	}
	v.quickSpell(0, false, -1, -1)
	selected, _, _ := v.spellbookPresent()
	if selected.RGBAAt(cell.Min.X, cell.Min.Y) != spellbookSelected || v.spellArmed {
		t.Fatal("current cell not marked independently of Cast")
	}
	if _, ok := commandPanelSelected(v); ok {
		t.Fatal("select-only key lit an armed command")
	}
	v.toggleHudPanel(hudPanelBook)
	v.quickSpell(0, false, -1, -1)
	if cell, ok := commandPanelSelected(v); !ok || cell != commandCellCast {
		t.Fatal("closed invocation does not mark Cast")
	}
	v.cancelMapCommand()
	*slots = [4]uint32{3}
	v.quickSpell(0, false, -1, -1)
	if v.selectedSpell != 3 || v.spellArmed || v.missionMode() != modeNone {
		t.Fatal("unavailable current was hidden or armed")
	}
	if _, ok := commandPanelSelected(v); ok {
		t.Fatal("unavailable current lit Cast")
	}
	v.words = AuthoredWords()
	panel, _ := v.commandPanelBar()
	cast := commandCellRects(panel)[commandCellCast]
	v.cursorX, v.cursorY = cast.Min.X+2, cast.Min.Y+2
	v.hasCursor = true
	tooltip, _, ok := v.commandHoverPresent()
	wantTooltip := composeItemPopup([]string{v.words.Command[commandCellCast] + ": Unavailable"}, v.font)
	if !ok || !bytes.Equal(tooltip.Pix, wantTooltip.Pix) {
		t.Fatal("Cast tooltip lost the current unavailable spell name")
	}
	v.selectedSpell = 999
	tooltip, _, ok = v.commandHoverPresent()
	wantTooltip = composeItemPopup([]string{v.words.Command[commandCellCast]}, v.font)
	if !ok || !bytes.Equal(tooltip.Pix, wantTooltip.Pix) {
		t.Fatal("unknown ID got a fabricated name")
	}
}

func TestQuickSpellCtrlClosedBookStillRequestsCapabilityGatedCast(t *testing.T) {
	_, v, slots := quickSpellApp(t)
	v.selectedSpell = 89
	v.toggleHudPanel(hudPanelBook)
	v.quickSpell(3, true, -1, -1)
	if slots[3] != 89 || v.selectedSpell != 89 || !v.spellArmed {
		t.Fatal("Ctrl assignment skipped the later closed-book arm")
	}
	v.cancelMapCommand()
	v.selectedSpell = 3
	v.quickSpell(2, true, -1, -1)
	if slots[2] != 3 || v.spellArmed {
		t.Fatal("Ctrl unavailable assignment armed")
	}
	v.cancelMapCommand()
	// Ctrl with no current/hover source leaves a populated binding intact,
	// but still reaches its closed-book availability gate (AI-QUICKASSIGN-278).
	// Cast mode must remain cancellable even when current is independently 0.
	v.quickSpell(3, true, -1, -1)
	if v.selectedSpell != 0 || !v.spellArmed {
		t.Fatal("source-less Ctrl changed current or skipped its later mode gate")
	}
	v.cancelMapCommand()
	if v.spellArmed || len(v.sel) != 1 {
		t.Fatal("cancel did not lower independent Cast mode before deselecting units")
	}
	// An actual blank catalog cell must not supply ID0 or acquire a current
	// border merely because the neutral selection also uses zero.
	v.SetSpellbook(sbUnitID, []SpellEntry{{}})
	v.selectedSpell = 0
	v.toggleHudPanel(hudPanelBook)
	x, y := sbEntryPoint(t, v, v.spellbook, 0)
	before := *slots
	v.quickSpell(0, true, x, y)
	if *slots != before {
		t.Fatal("blank cell changed a binding")
	}
	sbClick(v, x, y)
	if v.selectedSpell != 0 || v.spellArmed {
		t.Fatal("blank cell armed Cast")
	}
}

func quickSpellApp(t *testing.T) (*App, *Viewer, *[4]uint32) {
	t.Helper()
	a := spellbookTestApp(t)
	mapAtScaleOne(a)
	v := a.flow.viewer
	v.localOwner = 1
	v.SetEntities([]MapEntity{{ID: sbUnitID, Owner: 1, Life: LifeAlive, MaxMana: 100}})
	v.sel = selection{sbUnitID}
	v.SetSpellbook(sbUnitID, []SpellEntry{{ID: 17, Name: "First"}, {ID: 89, Name: "Second"}, {ID: 3, Name: "Unavailable", Unavailable: true}})
	slots := new([4]uint32)
	v.SetQuickSpells(slots)
	return a, v, slots
}

func TestQuickSpellPhysicalEdges(t *testing.T) {
	bindings := bindingSource(t)
	for _, key := range []string{"F5", "F6", "F7", "F8"} {
		want := "inpututil.IsKeyJustPressed(ebiten.Key" + key + ")"
		if !strings.Contains(bindings["QuickSpell"], want) {
			t.Errorf("missing %s edge: %s", key, bindings["QuickSpell"])
		}
		for field, expr := range bindings {
			if field != "QuickSpell" && strings.Contains(expr, "ebiten.Key"+key+")") {
				t.Errorf("duplicate physical key %s in %s", key, field)
			}
		}
	}
}

func TestQuickSpellAssignCurrentHoverDuplicateAndEmpty(t *testing.T) {
	a, v, slots := quickSpellApp(t)
	x, y := sbEntryPoint(t, v, v.spellbook, 1)
	a.step(appInput{QuickSpell: [4]bool{true}, Viewer: Input{Ctrl: true}, CursorX: x, CursorY: y}, time.Unix(10, 0))
	if *slots != [4]uint32{89} || v.selectedSpell != 0 || v.spellArmed {
		t.Fatalf("hover assignment=%v current=%d armed=%v", *slots, v.selectedSpell, v.spellArmed)
	}
	v.selectedSpell = 17
	v.quickSpell(1, true, x, y)
	if *slots != [4]uint32{89, 17} {
		t.Fatalf("current did not outrank hovered real ID: %v", *slots)
	}
	v.quickSpell(0, true, -1, -1)
	if *slots != [4]uint32{17} {
		t.Fatalf("duplicate did not move/replace rather than swap: %v", *slots)
	}
	v.selectedSpell = 0
	v.quickSpell(2, true, -1, -1)
	if *slots != [4]uint32{17} {
		t.Fatal("empty source changed slots")
	}
	x, y = sbEntryPoint(t, v, v.spellbook, 2)
	v.quickSpell(3, true, x, y)
	if slots[3] != 3 || v.selectedSpell != 0 || v.spellArmed {
		t.Fatal("unavailable hover was not independently bindable")
	}
}

func TestQuickSpellInvokeCurrentAndArmedAreSeparate(t *testing.T) {
	_, v, slots := quickSpellApp(t)
	*slots = [4]uint32{89, 3}
	v.quickSpell(0, false, -1, -1)
	if v.selectedSpell != 89 || v.missionMode() != modeNone {
		t.Fatal("open-book invocation must select only")
	}
	if _, selected := commandPanelSelected(v); selected {
		t.Fatal("select-only invocation lit armed command overlay")
	}
	v.toggleHudPanel(hudPanelBook)
	v.quickSpell(0, false, -1, -1)
	if v.selectedSpell != 89 || v.missionMode() != modeCast || v.hudShown(hudPanelBook) {
		t.Fatal("closed available invocation did not arm without opening book")
	}
	v.quickSpell(2, false, -1, -1)
	if v.selectedSpell != 89 || v.missionMode() != modeCast {
		t.Fatal("empty invocation changed current/mode")
	}
	v.quickSpell(1, false, -1, -1)
	if v.selectedSpell != 3 || v.missionMode() != modeCast {
		t.Fatal("unavailable invocation must change current, not request another mode")
	}
	v.cancelMapCommand()
	v.quickSpell(1, false, -1, -1)
	if v.selectedSpell != 3 || v.missionMode() != modeNone {
		t.Fatal("unavailable populated slot armed from neutral")
	}
	for _, unit := range []MapEntity{{ID: sbUnitID, Owner: 1, Life: LifeAlive}, {ID: sbUnitID, Owner: 2, Life: LifeAlive, MaxMana: 100}} {
		v.SetEntities([]MapEntity{unit})
		v.quickSpell(0, false, -1, -1)
		if v.selectedSpell != 89 || v.spellArmed {
			t.Fatal("Cast capability/ownership gate bypassed")
		}
	}
	v.SetSpellbook(sbOtherID, nil)
	if v.selectedSpell != 0 || v.spellArmed || *slots != [4]uint32{89, 3} {
		t.Fatal("selection change lost session slots or kept current mode")
	}
}

func TestQuickSpellMapFocusAndModalOwnership(t *testing.T) {
	for _, gate := range []string{"unfocused", "notice", "dismiss", "popup", "menu", "load", "town", "cutscene drain"} {
		t.Run(gate, func(t *testing.T) {
			a, v, slots := quickSpellApp(t)
			v.selectedSpell = 17
			in := appInput{QuickSpell: [4]bool{true}, Viewer: Input{Ctrl: true}}
			switch gate {
			case "unfocused":
				in.Unfocused = true
			case "notice", "dismiss":
				v.SetNotice("notice", NoticeDialogue)
				in.Enter = gate == "dismiss"
			case "popup":
				v.menuUp = true
			case "menu":
				a.flow.screen = ScreenGameMenu
			case "load":
				a.flow.screen = ScreenLoad
			case "town":
				a.flow.screen = ScreenTown
			case "cutscene drain":
				a.cutsceneDrain = true
			}
			a.step(in, time.Unix(11, 0))
			if *slots != [4]uint32{} {
				t.Fatalf("%s assigned %v", gate, *slots)
			}
		})
	}
}

func TestQuickSpellHeadlessAliasesAndScaledHover(t *testing.T) {
	a, v, slots := quickSpellApp(t)
	for i, key := range []string{"f5", "f6", "f7", "f8"} {
		v.selectedSpell = uint32(100 + i)
		if err := a.HeadlessKey("ctrl-" + key); err != nil {
			t.Fatal(err)
		}
		v.selectedSpell = 0
		if err := a.HeadlessKey(key); err != nil {
			t.Fatal(err)
		}
		if slots[i] != uint32(100+i) || v.selectedSpell != slots[i] || v.spellArmed {
			t.Fatalf("alias %s: slots%v current%d", key, *slots, v.selectedSpell)
		}
	}
	// Deliberately different window and frame rectangles; hit-test the picture,
	// not a compact index+1 or the unconverted window coordinates.
	v.selectedSpell = 0
	v.Layout(2048, 1600)
	x, y := sbEntryPoint(t, v, v.spellbook, 1)
	v.quickSpell(0, true, x*2, 32+y*2)
	if slots[0] != 89 {
		t.Fatalf("scaled hover bound %d, want real ID89", slots[0])
	}
}
