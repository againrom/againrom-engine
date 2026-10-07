package ui

import (
	"reflect"
	"testing"
	"time"
)

func TestCurrentAnimationRestoresFrameAndNextCadenceWithoutLoadTime(t *testing.T) {
	live := applicationViewer(t)
	live.anim.AdvanceMicros(5*live.anim.Period() + 17000)
	live.SetPhase(17000, 62000)
	captured := live.SaveAnimation()
	cold := applicationViewer(t)
	cold.last = time.Unix(10, 0)
	if err := cold.RestoreAnimation(captured); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(captured, cold.SaveAnimation()) || !cold.last.IsZero() {
		t.Fatal("LOAD changed frame/phase or retained wall epoch")
	}
	live.anim.AdvanceMicros(live.anim.Period() - 17000)
	cold.anim.AdvanceMicros(cold.anim.Period() - 17000)
	if live.SaveAnimation() != cold.SaveAnimation() || cold.AnimationCounter() != captured.Count+1 {
		t.Fatal("restored phase changed next animation boundary")
	}
	for _, invalid := range []int{-1, 1000000} {
		before := cold.SaveAnimation()
		if cold.RestoreAnimation(AnimationState{Count: 77, RemainderUS: invalid}) == nil {
			t.Fatal("invalid remainder admitted")
		}
		if cold.SaveAnimation() != before {
			t.Fatal("invalid remainder changed clock")
		}
	}
}

func TestSavedAnimationEpochIsAdoptionAndHeldCommandTime(t *testing.T) {
	h := newHaltFix(t, haltOpts{})
	spy := &saveDialogSpy{}
	spy.install(h.a)
	h.v.anim.Restore(7, 17000)
	before := h.v.SaveAnimation()
	h.v.last = haltAt(0)
	h.frame(appInput{SaveGame: true})
	if h.a.Screen() != ScreenSave || h.v.SaveAnimation() != before || h.v.last != h.w.now {
		t.Fatal("ordinary F2 did not consume its held epoch without advancing the saved phase")
	}
	cold := applicationViewer(t)
	if err := cold.RestoreAnimation(before); err != nil {
		t.Fatal(err)
	}
	if err := h.a.HeadlessSaveAction("cancel"); err != nil {
		t.Fatal(err)
	}
	if err := h.a.HeadlessGameMenuAction("return"); err != nil {
		t.Fatal(err)
	}
	h.a.SetQuickSaveControls(QuickSaveControls{Load: func() (MapOpener, bool, error) {
		return func() (*Viewer, MapTick, MapOrder, MapCadence, MapAffect, MapAdvance, MapAttack, MapGrab, MapStance, MapMarch, error) {
			return cold, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil
		}, false, nil
	}})
	h.frame(appInput{QuickLoad: true})
	if h.a.flow.viewer != cold || cold.last != h.w.now || cold.SaveAnimation() != before {
		t.Fatal("ordinary F9 adoption changed saved phase or retained a foreign epoch", h.a.Screen(), h.a.HeadlessMessage(), h.a.flow.viewer == cold, cold.last, h.w.now, cold.SaveAnimation(), before)
	}
	now := h.w.now.Add(45 * time.Millisecond)
	h.a.step(appInput{CursorX: -1, CursorY: -1, Viewer: Input{CursorX: -1, CursorY: -1}}, now)
	if got := cold.SaveAnimation(); got.Count != 8 || got.RemainderUS != 0 {
		t.Fatal("first ordinary post-load frame lost the known 45ms span", got)
	}
}
