package game

import (
	"image"
	"path/filepath"
	"testing"
	"time"

	"againrom/pkg/formats/sav"
	"againrom/pkg/ui"
)

func TestReleaseTownPointerPopupEntryAndColdLoad(t *testing.T) {
	r := newTownFiguresRig(t)
	r.f.Options = OptionsStore{Path: filepath.Join(t.TempDir(), "options.txt")}
	r.f.SetTipsOff(false)
	r.s.loadTip(roomSquare)
	middle := func(rect image.Rectangle) image.Point { return rect.Min.Add(rect.Max).Div(2) }
	pointer := func(edge string, p image.Point) {
		t.Helper()
		if err := r.app.HeadlessPointer(edge, p.X, p.Y); err != nil {
			t.Fatal(err)
		}
	}
	tip := func() ui.TipPanelView {
		switch r.s.room {
		case roomSquare:
			return r.s.TownSquareView().Tip
		case roomShop:
			return r.s.ShopScreen().TipPanel
		default:
			return r.s.TownSurface().Tip
		}
	}
	checkPopup := func() {
		t.Helper()
		v := tip()
		if !v.Showing() {
			t.Fatalf("installed room %d lacks popup", r.s.room)
		}
		close := middle(ui.TipPanelCloseRect(v.Rect))
		pointer("release", close)
		if !tip().Showing() {
			t.Fatal("unarmed installed Close release dismissed popup")
		}
		checkbox := middle(ui.TipPanelToggleRect(v.Rect))
		pointer("press", checkbox)
		shown, err := r.f.Options.TipsMode()
		if err != nil || shown || !r.f.TipsOff() || !tip().Showing() {
			t.Fatal("checkbox down failed persistent option or deleted constructed popup", shown, err)
		}
		pointer("release", image.Pt(2, 479))
		pointer("press", checkbox)
		pointer("release", checkbox)
		if r.f.TipsOff() {
			t.Fatal("second checkbox down did not restore option")
		}
		pointer("press", close)
		pointer("release", image.Pt(2, 479))
		if !tip().Showing() {
			t.Fatal("outside Close release deleted popup")
		}
		pointer("press", close)
		pointer("release", close)
		if tip().Showing() {
			t.Fatal("registered installed Close did not dismiss popup")
		}
	}
	checkPopup()
	r.f.SetTipsOff(true)
	r.enter("Tavern", roomTavern)
	r.endConversation(true)
	if tip().Showing() {
		t.Fatal("TipsOff constructed a popup on actual room entry")
	}
	r.leave(roomTavern, true)
	if tip().Showing() {
		t.Fatal("TipsOff constructed square popup on actual return")
	}
	r.f.SetTipsOff(false)
	r.enter("Tavern", roomTavern)
	r.endConversation(true)
	r.leave(roomTavern, true)
	if !tip().Showing() {
		t.Fatal("restored global option failed square reconstruction")
	}
	for _, room := range []struct {
		door string
		kind townRoom
	}{{"Tavern", roomTavern}, {"Shop", roomShop}, {"School", roomSchool}} {
		r.enter(room.door, room.kind)
		r.endConversation(true)
		checkPopup()
		r.leave(room.kind, false)
		if !r.s.TownSquareView().Tip.Showing() || r.s.exterior.guardStep != 0 || r.s.exterior.frame.Guard != 7 {
			t.Fatal("room Exit did not recreate square popup and idle guard")
		}
		r.enter(room.door, room.kind)
		r.endConversation(true)
		if !tip().Showing() {
			t.Fatal("ordinary room return retained dismissal")
		}
		r.leave(room.kind, true)
	}
	r.s.CloseTip()
	if err := r.app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if err := r.app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if !r.s.TownSquareView().Tip.Showing() {
		t.Fatal("menu resume retained old popup dismissal")
	}
	r.s.CloseTip()
	store := SaveStore{Dir: t.TempDir()}
	r.f.ConfigureSaveSeams(r.app, store, OriginalStore{}, nil)
	if err := r.app.HeadlessKey("f2"); err != nil {
		t.Fatal(err)
	}
	if err := r.app.HeadlessSaveAction("save"); err != nil {
		t.Fatal(err)
	}
	entries, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("SAVE wrote %d entries", len(entries))
	}
	raw, err := store.Read(entries[0].Name)
	if err != nil {
		t.Fatal("SAVE lacks SAV output", err)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil || doc.Head.Mission != 0 || doc.World != nil {
		t.Fatal("SAVE is not a town SAV", err)
	}
	if err := r.app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if r.app.Screen() != ui.ScreenTown {
		t.Fatal("SAVE dialog did not return to town")
	}
	oldScreen, lastPaint := r.s, r.s.townPaintLast
	r.now = r.now.Add(67 * time.Millisecond)
	if err := r.app.HeadlessKey("f3"); err != nil {
		t.Fatal(err)
	}
	if err := r.app.HeadlessKey("enter"); err != nil {
		t.Fatal(err)
	}
	if r.f.TownScreen() != oldScreen || r.s.townPaintLast != lastPaint || !r.s.TownSquareView().Tip.Showing() {
		t.Fatalf("same-process SAV LOAD screen=%s same=%v last=%v before=%v now=%v popup=%v tipsOff=%v", r.app.Screen(), r.f.TownScreen() == oldScreen, r.s.townPaintLast, lastPaint, r.now, r.s.TownSquareView().Tip.Showing(), r.f.TipsOff())
	}
	other := releaseFront(t)
	other.Options = r.f.Options
	other.LoadOptions()
	other.TownAnimationNow = func() time.Time { return r.now }
	other.TownAnimationRandom = func(int) int { return 0 }
	cold := other.App("town pointer cold load")
	cold.Layout(640, 480)
	other.ConfigureSaveSeams(cold, store, OriginalStore{}, nil)
	if err := cold.HeadlessKey("load"); err != nil {
		t.Fatal(err)
	}
	if err := cold.HeadlessKey("enter"); err != nil {
		t.Fatal(err)
	}
	s := other.TownScreen().(*townScreen)
	if cold.Screen() != ui.ScreenTown || !s.TownSquareView().Tip.Showing() || s.exterior.guardStep != 0 || s.townPaintLast.IsZero() {
		t.Fatal("cold town LOAD lacks reconstructed popup/synchronous first paint")
	}
	r.f, r.app, r.s = other, cold, s
	checkPopup()
	r.enter("Tavern", roomTavern)
	r.endConversation(true)
	r.s.CloseTip()
	v := r.s.TownSurface()
	reachedOccupied := 0
	for i, cell := range v.Cells {
		if !cell.Portrait || cell.Selected {
			continue
		}
		p := image.Pt(176+(i%6)*48+24, 480-(i/6+1)*64+32)
		pointer("press", p)
		if r.s.TownSurface().Cells[i].Key != cell.Key || !r.s.TownSurface().Cells[i].Selected {
			t.Fatal("cold LOAD next reached occupied down did not select stable candidate")
		}
		pointer("release", image.Pt(2, 2))
		reachedOccupied++
		break
	}
	if reachedOccupied != 1 {
		t.Fatalf("reached %d nonselected occupied installed cells; want 1", reachedOccupied)
	}
	before := r.s.tavernSelection
	pointer("press", image.Pt(200, 100))
	pointer("release", image.Pt(200, 100))
	if r.s.tavernSelection != before {
		t.Fatal("installed candle miss selected roster")
	}
	// This loss control supplies zero roster occupancy over installed room art.
	r.f.Town = NewTown(Campaign{})
	r.s.tavernSelection = tavernCandidateKey{}
	if cells := r.s.TownSurface().Cells; len(cells) != 0 {
		t.Fatalf("zero population has %d cells", len(cells))
	}
	for _, p := range []image.Point{image.Pt(300, 300), image.Pt(430, 300), image.Pt(200, 100)} {
		pointer("press", p)
		pointer("release", p)
		if r.s.tavernSelection != (tavernCandidateKey{}) {
			t.Fatal("zero-occupancy figure changed selection")
		}
	}
	t.Logf("installed popup routes, three Exit/Escape returns, menu resume, SAV/same-process and cold LOAD, occupied down=%d and zero-population figure loss control verified", reachedOccupied)
}

func TestReleaseTownPopupDismissalSurvivesDialogue(t *testing.T) {
	for _, tipsOff := range []bool{false, true} {
		for _, route := range []string{"tavern", "gate"} {
			name := route
			if tipsOff {
				name += "-tips-off"
			}
			t.Run(name, func(t *testing.T) {
				r := newTownFiguresRig(t)
				t.Cleanup(r.app.StopAudio)
				r.f.Options = OptionsStore{Path: filepath.Join(t.TempDir(), "options.txt")}
				r.f.SetTipsOff(tipsOff)
				r.s.loadTip(roomSquare)
				popup := func() ui.TipPanelView {
					if route == "gate" {
						return r.s.TownSquareView().Tip
					}
					return r.s.TownSurface().Tip
				}
				closeDialogue := func() {
					t.Helper()
					if _, shown := r.s.TownDialogue(); !shown || r.s.room != roomTalk {
						t.Fatal("installed App route did not open dialogue")
					}
					for n := 0; r.s.room == roomTalk && n < 64; n++ {
						if err := r.app.HeadlessKey("enter"); err != nil {
							t.Fatal(err)
						}
					}
					if r.s.room == roomTalk {
						t.Fatal("Enter did not close installed dialogue")
					}
				}
				if route == "tavern" {
					r.enter("Tavern", roomTavern)
				}
				if popup().Showing() == tipsOff {
					t.Fatal("room construction does not match global option")
				}
				if !tipsOff {
					v := popup()
					button := ui.TipPanelCloseRect(v.Rect)
					r.click(button.Min.Add(button.Max).Div(2))
					if popup().Showing() {
						t.Fatal("owned Close did not dismiss installed popup")
					}
				}
				revision := r.s.tipRevision
				if route == "tavern" {
					if len(r.s.TownSurface().Cells) == 0 {
						t.Fatal("installed tavern has no talk candidate")
					}
					r.click(image.Pt(200, 450))
					if !r.s.TownSurface().Cells[0].Selected {
						t.Fatal("App down failed to select installed talk candidate")
					}
					button := ui.TownSurfaceButtonRect(ui.TownSurfaceTavern, tavernButtonTalk)
					r.click(button.Min.Add(button.Max).Div(2))
				} else {
					r.setQuest(false)
					r.click(squareGatePoint(t, r.f))
					if !r.s.AtTownSquare() || r.s.AtWorldMap() {
						t.Fatal("gate line left square before dialogue close")
					}
				}
				closeDialogue()
				if popup().Showing() {
					t.Errorf("dismissed %s popup recreated after dialogue close without room exit", route)
				}
				if !tipsOff && r.s.tipRevision != revision {
					t.Errorf("dialogue close reconstructed popup revision %d -> %d", revision, r.s.tipRevision)
				}
				if route == "tavern" {
					if r.s.room != roomTavern {
						t.Fatal("dialogue close left tavern")
					}
					r.leave(roomTavern, false)
					r.enter("Tavern", roomTavern)
				} else {
					if r.s.room != roomSquare {
						t.Fatal("gate close left square")
					}
					r.enter("Tavern", roomTavern)
					r.leave(roomTavern, false)
				}
				if popup().Showing() == tipsOff {
					t.Fatal("actual room leave/re-enter did not reconstruct according to global option")
				}
				t.Logf("%s TipsOff=%v: owned popup Close, installed dialogue Enter-close, actual leave/re-entry", route, tipsOff)
			})
		}
	}
}
