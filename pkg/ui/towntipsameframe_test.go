package ui

import (
	"image"
	"testing"
	"time"
)

// clickingTipShopTown is a first-game shop whose presses click.
type clickingTipShopTown struct{ fakeTipShopEnumTown }

func (*clickingTipShopTown) TownClickSounds() bool { return true }

// sameFrameClick is a click whose press and release reach one update, as a
// quick tap does.
func sameFrameClick(a *App, now time.Time, p image.Point) {
	a.step(appInput{CursorX: p.X, CursorY: p.Y, PrimaryPressed: true, PrimaryReleased: true}, now)
}

// tipRoom is one room's tip fixture and the counters it is judged by.
type tipRoom struct {
	name   string
	rect   image.Rectangle
	town   func(tip TipPanelView, clicks bool) TownScreen
	closed func(TownScreen) int
	toggle func(TownScreen) int
	passed func(TownScreen) int
}

// TestTownTipSameFrameClickActsOnItsControl: a click whose press and release
// arrive in one frame closes the tip on Close and toggles it on the checkbox,
// once each, in the shop, the tavern and the school, for a town with the
// first game's click model and one without it. The click reaches nothing
// under the panel.
func TestTownTipSameFrameClickActsOnItsControl(t *testing.T) {
	surface := func(name string, kind TownSurfaceKind, rect image.Rectangle) tipRoom {
		return tipRoom{
			name: name,
			rect: rect,
			town: func(tip TipPanelView, clicks bool) TownScreen {
				plain := fakeTipSurfaceEnumTown{view: TownSurfaceView{Kind: kind, SchoolClass: -1, Tip: tip}}
				if clicks {
					return &clickingTipTown{plain}
				}
				return &plain
			},
			closed: func(s TownScreen) int { return surfaceTown(s).closed },
			toggle: func(s TownScreen) int { return surfaceTown(s).toggled },
			passed: func(s TownScreen) int { return len(surfaceTown(s).surfaceClicks) },
		}
	}
	shop := func(s TownScreen) *fakeTipShopEnumTown {
		if c, ok := s.(*clickingTipShopTown); ok {
			return &c.fakeTipShopEnumTown
		}
		return s.(*fakeTipShopEnumTown)
	}
	rooms := []tipRoom{
		{name: "shop", rect: shopTipRect,
			town: func(tip TipPanelView, clicks bool) TownScreen {
				if clicks {
					return &clickingTipShopTown{fakeTipShopEnumTown{tip: tip}}
				}
				return &fakeTipShopEnumTown{tip: tip}
			},
			closed: func(s TownScreen) int { return shop(s).closed },
			toggle: func(s TownScreen) int { return shop(s).toggled },
			passed: func(s TownScreen) int { return len(shop(s).clicked) }},
		surface("school", TownSurfaceSchool, SchoolTipRect),
		surface("tavern", TownSurfaceTavern, TavernTipRect),
	}
	for _, r := range rooms {
		for _, clicks := range []bool{false, true} {
			tip := TipPanelView{Rect: r.rect, Text: "the room's tip", Art: tipTestArt(), Font: shopTipTestFont()}
			town := r.town(tip, clicks)
			a := latchTownApp(t, town)
			rec := &pressSoundRecorder{}
			a.SetAudio(rec, fixedUISoundBank())
			now := time.Unix(1_700_000_000, 0)

			closeAt, ok := sampleInside(TipPanelCloseRect(r.rect))
			if !ok {
				t.Fatalf("%s: the Close rectangle is empty", r.name)
			}
			sameFrameClick(a, now, closeAt)
			if r.closed(town) != 1 || a.townTipPress.Holds() {
				t.Errorf("%s clicks=%v: a same-frame Close click closed %d time(s), latch held %v; want 1 and clear", r.name, clicks, r.closed(town), a.townTipPress.Holds())
			}
			want := 0
			if clicks {
				want = 1
			}
			wantClicks(t, rec, want, r.name+" same-frame Close click")

			toggleAt, ok := sampleInside(TipPanelToggleRect(r.rect))
			if !ok {
				t.Fatalf("%s: the checkbox rectangle is empty", r.name)
			}
			sameFrameClick(a, now, toggleAt)
			if r.toggle(town) != 1 {
				t.Errorf("%s clicks=%v: a same-frame checkbox click toggled %d time(s), want 1", r.name, clicks, r.toggle(town))
			}
			body, _ := sampleInside(tipBackgroundZone(r.rect))
			sameFrameClick(a, now, body)
			if r.passed(town) != 0 {
				t.Errorf("%s clicks=%v: a same-frame click on the tip's body reached %d control(s) under it", r.name, clicks, r.passed(town))
			}
		}
	}
}

func surfaceTown(s TownScreen) *fakeTipSurfaceEnumTown {
	if c, ok := s.(*clickingTipTown); ok {
		return &c.fakeTipSurfaceEnumTown
	}
	return s.(*fakeTipSurfaceEnumTown)
}
