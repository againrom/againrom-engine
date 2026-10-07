package ui

import (
	"errors"
	"image"
	"reflect"
	"testing"
	"time"
)

func TestQuickKeysReachPhysicalVocabulary(t *testing.T) {
	h := newHaltFix(t, haltOpts{})
	for _, key := range []string{"f4", "f9"} {
		if err := h.a.HeadlessKey(key); err != nil {
			t.Errorf("%s: %v", key, err)
		}
	}
}

func TestQuickActionCancelsPendingInventoryGesture(t *testing.T) {
	for _, tc := range []struct {
		name       string
		save, load bool
	}{{"ordinary control", false, false}, {"F4", true, false}, {"refused F9", false, true}} {
		t.Run(tc.name, func(t *testing.T) {
			h := newHaltFix(t, haltOpts{})
			h.a.Layout(1024, 768)
			h.frame(appInput{Pause: true})
			h.v.hudHidden[hudPanelPack] = false
			h.v.SetEntities([]MapEntity{{ID: 5, Cell: image.Pt(1, 1)}})
			h.v.sel = selection{5}
			h.v.SetInventorySubject(packOf(5, image.NewRGBA(image.Rect(0, 0, 8, 8))))
			calls := 0
			h.a.SetQuickSaveControls(QuickSaveControls{
				Save: func(bool) error { calls++; return nil },
				Load: func() (MapOpener, bool, error) { calls++; return nil, false, errors.New("load refusal") },
			})
			x, y := packCellCenter(t, h.v, 0)
			press := func() {
				h.frame(appInput{PrimaryPressed: true, CursorX: x, CursorY: y, Viewer: Input{PrimaryDown: true, CursorX: x, CursorY: y}})
			}
			move := func() {
				h.frame(appInput{CursorX: x + 3, CursorY: y + 2, Viewer: Input{PrimaryDown: true, CursorX: x + 3, CursorY: y + 2}})
			}
			press()
			move()
			if _, _, visible := h.v.dragItemPresent(); !visible || !h.v.invGrab {
				t.Fatal("ordinary pack drag did not arm")
			}
			before, world, ticks := h.v.SaveApplication(), h.w.world, h.w.ticks
			h.frame(appInput{QuickSave: tc.save, QuickLoad: tc.load, CursorX: x + 3, CursorY: y + 2, Viewer: Input{PrimaryDown: true, CursorX: x + 3, CursorY: y + 2}})
			quick := tc.save || tc.load
			if quick {
				if calls != 1 || h.w.ticks != ticks || h.w.world != world || h.a.flow.viewer != h.v || !reflect.DeepEqual(before, h.v.SaveApplication()) {
					t.Fatal("quick action changed session/application or advanced a tick")
				}
				if h.v.dragActive || h.v.dragCandKind != dragNone || h.v.invGrab || h.v.held {
					t.Error("quick frame retained the pending gesture")
				}
				for range 2 {
					h.frame(appInput{CursorX: x + 6, CursorY: y + 4, Viewer: Input{PrimaryDown: true, CursorX: x + 6, CursorY: y + 4}})
					if h.v.dragging || h.v.dragActive || h.v.held || h.v.invGrab || !reflect.DeepEqual(before, h.v.SaveApplication()) {
						t.Errorf("held tail: dragging=%v active=%v held=%v inv=%v pointer=%d,%d before=%+v after=%+v", h.v.dragging, h.v.dragActive, h.v.held, h.v.invGrab, x, y, before, h.v.SaveApplication())
					}
				}
			} else if calls != 0 {
				t.Fatal("no-quick control dispatched a quick action")
			}
			h.frame(appInput{PrimaryReleased: true, CursorX: x, CursorY: y, Viewer: Input{CursorX: x, CursorY: y}})
			for frame := range 2 {
				if _, _, visible := h.v.dragItemPresent(); visible || h.v.dragCandKind != dragNone || h.v.invGrab || h.v.primaryDown {
					t.Errorf("released item persists at frame %d: active=%v candidate=%v inventory=%v down=%v", frame, h.v.dragActive, h.v.dragCandKind, h.v.invGrab, h.v.primaryDown)
				}
				if _, ok := h.v.TakeInventoryEquip(); ok {
					t.Error("interrupted gesture equipped an item")
				}
				if _, ok := h.v.TakeInventoryDollUnequip(); ok {
					t.Error("interrupted gesture unequipped an item")
				}
				if _, _, _, _, ok := h.v.TakeInventoryDrop(); ok {
					t.Error("interrupted gesture dropped an item")
				}
				h.frame(appInput{CursorX: x, CursorY: y, Viewer: Input{CursorX: x, CursorY: y}})
			}
			if !quick {
				for range 16 {
					h.frame(haltNeutral())
				}
			}
			press()
			move()
			if _, _, visible := h.v.dragItemPresent(); !visible || !h.v.invGrab {
				t.Fatalf("fresh subsequent gesture could not arm: active=%v cand=%d inv=%v app=%+v", h.v.dragActive, h.v.dragCandKind, h.v.invGrab, h.v.SaveApplication())
			}
			h.frame(appInput{PrimaryReleased: true, CursorX: x, CursorY: y, Viewer: Input{CursorX: x, CursorY: y}})
			if _, _, visible := h.v.dragItemPresent(); visible || h.a.suppressPrimaryRelease || h.v.invGrab {
				t.Fatal("fresh subsequent release remained suppressed")
			}
			if h.w.world != world {
				t.Fatal("paused pointer controls stepped the world")
			}
		})
	}
}

func TestQuickActionCancelsPendingMapGesture(t *testing.T) {
	for _, save := range []bool{true, false} {
		t.Run(map[bool]string{true: "F4", false: "refused F9"}[save], func(t *testing.T) {
			h := newHaltFix(t, haltOpts{})
			h.a.Layout(1024, 768)
			h.frame(appInput{Pause: true})
			h.v.Camera().Pan(200, 200)
			h.v.SetEntities([]MapEntity{{ID: 5, Cell: image.Pt(1, 1)}})
			h.v.sel = selection{5}
			h.a.SetQuickSaveControls(QuickSaveControls{Save: func(bool) error { return nil }, Load: func() (MapOpener, bool, error) { return nil, false, errors.New("load refusal") }})
			pointer := func(x, y int, down bool) appInput {
				return appInput{CursorX: x, CursorY: y, Viewer: Input{CursorX: x, CursorY: y, PrimaryDown: down, SecondaryDown: down}}
			}
			press := pointer(200, 200, true)
			press.PrimaryPressed, press.SecondaryPressed = true, true
			h.frame(press)
			h.frame(pointer(220, 210, true))
			if !h.v.dragging || !h.v.held || !h.v.rightDragging || !h.v.rightPanned {
				t.Fatal("ordinary map gestures did not arm")
			}
			before, world, ticks := h.v.SaveApplication(), h.w.world, h.w.ticks
			key := pointer(220, 210, true)
			key.QuickSave, key.QuickLoad = save, !save
			h.frame(key)
			if h.w.ticks != ticks || h.w.world != world || !reflect.DeepEqual(before, h.v.SaveApplication()) {
				t.Fatal("quick action changed application or advanced a tick")
			}
			for range 2 {
				h.frame(pointer(260, 240, true))
				if h.v.dragging || h.v.held || h.v.boxing || h.v.rightDragging || h.v.rightPanned || !reflect.DeepEqual(before, h.v.SaveApplication()) {
					t.Error("cancelled map gesture resumed or changed selection/camera")
				}
			}
			release := pointer(260, 240, false)
			release.PrimaryReleased, release.SecondaryReleased = true, true
			h.frame(release)
			h.frame(pointer(260, 240, false))
			if !reflect.DeepEqual(before, h.v.SaveApplication()) || h.a.suppressPrimaryRelease || h.a.suppressSecondaryRelease {
				t.Fatal("cancelled map release changed application or retained suppression")
			}
			if h.w.world != world {
				t.Fatal("pointer cancellation changed the paused world")
			}
			fresh := pointer(200, 200, false)
			fresh.SecondaryPressed, fresh.Viewer.SecondaryDown = true, true
			h.frame(fresh)
			fresh.SecondaryPressed = false
			fresh.CursorX, fresh.Viewer.CursorX = 210, 210
			h.frame(fresh)
			if reflect.DeepEqual(before, h.v.SaveApplication()) || !h.v.rightPanned {
				t.Fatalf("fresh subsequent map gesture could not pan: right=%v suppressed=%v app=%+v", h.v.rightPanned, h.a.suppressSecondaryRelease, h.v.SaveApplication())
			}
		})
	}
}

func TestQuickKeysOwnFrameAndUseTransactionalLoad(t *testing.T) {
	h := newHaltFix(t, haltOpts{})
	saves, loads, resets, polls := 0, 0, 0, 0
	failure := errors.New("injected load refusal")
	h.a.SetTimedAutosaveControls(TimedAutosaveControls{Reset: func() { resets++ }, Poll: func(*Viewer, bool, bool) error { polls++; return nil }})
	h.a.SetQuickSaveControls(QuickSaveControls{
		Save: func(onMap bool) error {
			saves++
			if !onMap {
				t.Fatal("wrong source screen")
			}
			return nil
		},
		Load: func() (MapOpener, bool, error) { loads++; return nil, false, failure },
	})
	before, tick, anim, x := h.v, h.w.world, h.v.AnimationCounter(), h.v.Camera().X
	h.frame(appInput{QuickSave: true, Pause: true, Grid: true, Viewer: Input{PanRight: true}})
	if saves != 1 || h.w.world != tick || h.v.AnimationCounter() != anim || h.v.Camera().X != x || h.v.GridOverlay() || h.a.flow.stopped {
		t.Fatal("quicksave leaked side input or advanced the world")
	}
	priorPoll, priorReset := polls, resets
	h.frame(appInput{QuickLoad: true, Pause: true, Grid: true})
	if loads != 1 || h.a.flow.viewer != before || h.w.world != tick || polls != priorPoll || resets != priorReset {
		t.Fatal("failed quickload changed the session or polled the due timer")
	}
	h.a.SetQuickSaveControls(QuickSaveControls{Load: func() (MapOpener, bool, error) {
		return func() (*Viewer, MapTick, MapOrder, MapCadence, MapAffect, MapAdvance, MapAttack, MapGrab, MapStance, MapMarch, error) {
			return nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, failure
		}, false, nil
	}})
	h.frame(appInput{QuickLoad: true})
	if h.a.flow.viewer != before || resets != priorReset || h.a.flow.loadUI.completed != 0 {
		t.Fatal("opener refusal released the old map")
	}
	h.a.SetTown(&stubTown{rows: []TownRow{{Text: "town", Choosable: true}}})
	h.a.SetQuickSaveControls(QuickSaveControls{Load: func() (MapOpener, bool, error) { return nil, true, nil }})
	h.frame(appInput{QuickLoad: true})
	if h.a.Screen() != ScreenTown || resets != priorReset+1 || h.a.flow.loadUI.completed != 1 {
		t.Fatal("quickload did not share ordinary load completion/reset")
	}
}

func TestQuickKeysRespectInputOwners(t *testing.T) {
	for _, save := range []bool{false, true} {
		for _, blocked := range []string{"focus", "standalone", "notice", "notice closes", "popup", "text", "save text", "load", "menu", "town dialogue", "cutscene"} {
			t.Run(blocked+map[bool]string{true: " save", false: " load"}[save], func(t *testing.T) {
				h := newHaltFix(t, haltOpts{})
				calls := 0
				h.a.SetQuickSaveControls(QuickSaveControls{Save: func(bool) error { calls++; return nil }, Load: func() (MapOpener, bool, error) { calls++; return nil, false, errors.New("refused") }})
				in := appInput{QuickSave: save, QuickLoad: !save}
				switch blocked {
				case "focus":
					in.Unfocused = true
				case "standalone":
					h.v.SetGameMenuContext(func() GameMenuContext { return GameMenuContext{} })
				case "notice":
					h.v.SetNotice("notice", NoticeDialogue)
				case "notice closes":
					h.v.SetNotice("notice", NoticeDialogue)
					in.Enter = true
				case "popup":
					h.v.menuUp = true
				case "text":
					h.a.flow.screen = ScreenChargen
				case "save text":
					h.a.flow.screen = ScreenSave
				case "load":
					h.a.flow.screen = ScreenLoad
				case "menu":
					h.a.flow.screen = ScreenMenu
				case "town dialogue":
					h.a.SetTown(&fakeTownDialogue{pic: image.NewRGBA(image.Rect(0, 0, 10, 10))})
					h.a.flow.showTown("")
				case "cutscene":
					h.a.cutsceneDrain = true
				}
				h.a.step(in, time.Unix(100, 0))
				if calls != 0 {
					t.Fatal("input owner admitted quick action")
				}
			})
		}
	}
}
