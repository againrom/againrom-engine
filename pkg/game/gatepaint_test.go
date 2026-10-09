package game

import (
	"image"
	"reflect"
	"testing"
	"time"

	"againrom/pkg/ui"
)

func gatePaintApp(t *testing.T) (*FrontEnd, *ui.App, *townScreen, *time.Time, *int) {
	t.Helper()
	f, _ := gateLineFront(t, true)
	now, rolls := time.Unix(100, 0), 0
	f.TownAnimationNow = func() time.Time { return now }
	f.TownAnimationRandom = func(n int) int { rolls++; return 0 }
	a, s := exteriorApp(t, f)
	t.Cleanup(a.StopAudio)
	return f, a, s, &now, &rolls
}

func gatePaintFrame(t *testing.T, a *ui.App) *image.RGBA {
	t.Helper()
	p, note, err := a.HeadlessFrame()
	if err != nil || p == nil || note != "" {
		t.Fatalf("frame: %q %v", note, err)
	}
	return p
}

func TestGateDialoguePaintAdmission(t *testing.T) {
	t.Run("ordinary-square-control", func(t *testing.T) {
		_, a, s, now, _ := gatePaintApp(t)
		exteriorPointer(t, a, 10)
		gatePaintFrame(t, a)
		*now = now.Add(68 * time.Millisecond)
		gatePaintFrame(t, a)
		if s.sqEpisode("tavern").Frame != 1 {
			t.Fatalf("ordinary due square paint: tavern=%d want 1", s.sqEpisode("tavern").Frame)
		}
		t.Log("ordinary due square paint advances tavern 0 to 1")
	})
	t.Run("npc35-due-square-paint", func(t *testing.T) {
		f, a, s, now, rolls := gatePaintApp(t)
		gatePaintFrame(t, a)
		pressGate(t, a, squareGatePoint(t, f))
		if !gateDialogueOpen(s) || a.Screen() != ui.ScreenTown {
			t.Fatal("npc35 not shown on town")
		}
		before, beforeRolls, beforeClock := s.sqFrames(), *rolls, s.sqPaintLast()
		*now = now.Add(68 * time.Millisecond)
		gatePaintFrame(t, a)
		t.Logf("npc35 shown: guard %d=>%d; hub random calls %d=>%d; square=%v", before.Guard, s.sqGuard().Frame, beforeRolls, *rolls, s.AtTownSquare())
		if !reflect.DeepEqual(s.townLines(), []string{"Nobody has given you work yet."}) || !gateDialogueOpen(s) || a.Screen() != ui.ScreenTown {
			t.Fatal("paint changed dialogue content or screen")
		}
		if s.sqFrames() != before || *rolls != beforeRolls {
			t.Errorf("npc35 admits the blocked town paint: frame %+v=>%+v random calls %d=>%d", before, s.sqFrames(), beforeRolls, *rolls)
		}
		if s.sqPaintLast() != beforeClock {
			t.Fatal("blocked paint changed the process timestamp")
		}
		exteriorPointer(t, a, 10)
		if s.sqSelector() != 2 || !s.sqEpisode("tavern").Enabled || s.sqFrames() != before {
			t.Fatal("shown dialogue blocked delivered pointer or pointer advanced the frame")
		}
		gatePaintFrame(t, a)
		if s.sqFrames() != before || *rolls != beforeRolls || s.sqPaintLast() != beforeClock {
			t.Fatal("pointer-triggered dialogue composition admitted town paint")
		}
		if err := a.HeadlessKey("escape"); err != nil {
			t.Fatal(err)
		}
		gatePaintFrame(t, a)
		if gateDialogueOpen(s) || s.sqEpisode("tavern").Frame != 1 || *rolls != beforeRolls+2 || s.sqPaintLast() != *now {
			t.Fatal("pager close did not readmit the already-due hub at its retained timestamp")
		}
	})
	t.Run("pager-and-screen-exit-controls", func(t *testing.T) {
		f, a, s, now, _ := gatePaintApp(t)
		gatePaintFrame(t, a)
		pressGate(t, a, squareGatePoint(t, f))
		if err := a.HeadlessKey("escape"); err != nil {
			t.Fatal(err)
		}
		if gateDialogueOpen(s) || !s.AtTownSquare() || a.Screen() != ui.ScreenTown {
			t.Fatal("pager did not restore square")
		}
		exteriorPointer(t, a, 10)
		*now = now.Add(68 * time.Millisecond)
		gatePaintFrame(t, a)
		if s.sqEpisode("tavern").Frame != 1 {
			t.Fatalf("square after pager close: tavern=%d", s.sqEpisode("tavern").Frame)
		}
		if err := a.HeadlessKey("escape"); err != nil {
			t.Fatal(err)
		}
		if a.Screen() == ui.ScreenTown {
			t.Fatal("square Escape did not leave town screen")
		}
		before := s.sqFrames()
		*now = now.Add(time.Second)
		gatePaintFrame(t, a)
		if s.sqFrames() != before {
			t.Fatal("off-screen town paint advanced")
		}
		t.Log("pager close resumes due square paint; screen exit blocks square composition")
	})
}
