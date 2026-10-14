package ui

import (
	"image"
	"testing"

	"againrom/pkg/audio"
)

// pressSoundRecorder keeps every request with its parameters.
type pressSoundRecorder struct{ requests []audio.Request }

func (r *pressSoundRecorder) Play(audio.Sample, audio.Placement) {}
func (r *pressSoundRecorder) RequestSample(_ audio.Sample, q audio.Request) audio.Voice {
	r.requests = append(r.requests, q)
	return nil
}

func wantClicks(t *testing.T, r *pressSoundRecorder, n int, what string) {
	t.Helper()
	if len(r.requests) != n {
		t.Fatalf("%s: %d request(s), want %d", what, len(r.requests), n)
	}
	for i, q := range r.requests {
		if q.Selector != "registry:2" || q.Priority != 220 || q.Group != audio.EffectsChannel || q.Repeat {
			t.Fatalf("%s: request %d is %s priority %d group %v repeat %v, want click01 at 220", what, i, q.Selector, q.Priority, q.Group, q.Repeat)
		}
	}
}

// clickingTipTown is a first-game town whose presses click.
type clickingTipTown struct{ fakeTipSurfaceEnumTown }

func (*clickingTipTown) TownClickSounds() bool { return true }

// TestTownTipClosePressRequestsClick01: the room tip's Close press requests
// click01 each time, the release and a press elsewhere nothing (TOWN-547).
func TestTownTipClosePressRequestsClick01(t *testing.T) {
	tip := TipPanelView{Rect: SchoolTipRect, Text: "the school teaches skills", Art: tipTestArt(), Font: shopTipTestFont()}
	town := &clickingTipTown{fakeTipSurfaceEnumTown{view: TownSurfaceView{Kind: TownSurfaceSchool, SchoolClass: -1, Tip: tip}}}
	a := latchTownApp(t, town)
	rec := &pressSoundRecorder{}
	a.SetAudio(rec, fixedUISoundBank())
	inside, _ := sampleInside(TipPanelCloseRect(tip.Rect))
	pointer := func(action string, p image.Point) {
		t.Helper()
		if err := a.HeadlessPointer(action, p.X, p.Y); err != nil {
			t.Fatal(err)
		}
	}
	pointer("press", image.Pt(600, 300))
	pointer("release", image.Pt(600, 300))
	wantClicks(t, rec, 0, "a press outside the tip")
	pointer("press", inside)
	wantClicks(t, rec, 1, "the Close press")
	pointer("release", inside)
	wantClicks(t, rec, 1, "the Close release")
	if town.closed != 1 {
		t.Fatalf("the Close release closed %d time(s)", town.closed)
	}
}

// TestMissionTipClosePressRequestsClick01 is the mission popup's Close.
func TestMissionTipClosePressRequestsClick01(t *testing.T) {
	v := &Viewer{}
	rec := &pressSoundRecorder{}
	v.SetAudio(rec, fixedUISoundBank())
	dress := MissionTipDress{Rect: image.Rect(100, 100, 412, 300), Art: tipTestArt(), Font: shopTipTestFont()}
	v.SetMissionTipDress(dress)
	v.ShowMissionTip("a mission tip")
	r := TipPanelCloseRect(v.MissionTip().Rect)
	p := r.Min.Add(r.Max).Div(2)
	// A dress without CloseClick, the second game's, requests nothing.
	v.missionTipGesture(p, true, false)
	wantClicks(t, rec, 0, "a Close press without CloseClick")
	dress.CloseClick = true
	v.SetMissionTipDress(dress)
	v.ShowMissionTip("a mission tip")
	v.missionTipGesture(image.Pt(5, 5), true, false)
	v.missionTipGesture(image.Pt(5, 5), false, true)
	wantClicks(t, rec, 0, "a press outside the mission tip")
	v.missionTipGesture(p, true, false)
	wantClicks(t, rec, 1, "the mission Close press")
	v.missionTipGesture(p, false, true)
	wantClicks(t, rec, 1, "the mission Close release")
	if v.MissionTip().Showing() {
		t.Fatal("the Close release left the mission tip showing")
	}
}

// TestTownDialoguePagePressRequestsClick01: a town dialogue's page button
// press requests click01; a mission dialogue's requests nothing here
// (DIALOGUE-080 reads the town's panel).
func TestTownDialoguePagePressRequestsClick01(t *testing.T) {
	for _, tc := range []struct {
		surface string
		want    int
		clicks  bool
	}{{"town", 1, true}, {"town", 0, false}, {"mission", 0, true}} {
		f := newDialoguePointerFixture(t, tc.surface)
		if town, ok := f.app.flow.town.(*pointerDialogueTown); ok {
			town.clicks = tc.clicks
		}
		rec := &pressSoundRecorder{}
		f.app.SetAudio(rec, fixedUISoundBank())
		p := f.button.Min.Add(f.button.Max).Div(2)
		f.pointer(t, "press", p, 0)
		wantClicks(t, rec, tc.want, tc.surface+" page press")
		f.pointer(t, "release", p, 1)
		wantClicks(t, rec, tc.want, tc.surface+" page release")
	}
}

// TestFirstGameTownSurfaceButtonsRequestNoClick: a first-game tavern or
// school button release requests no click01 (TOWN-548, TAVERN-028,
// TOWN-551); a town without the first game's click model still clicks.
func TestFirstGameTownSurfaceButtonsRequestNoClick(t *testing.T) {
	for _, first := range []bool{false, true} {
		plain := fakeTipSurfaceEnumTown{view: TownSurfaceView{Kind: TownSurfaceSchool, SchoolClass: -1,
			Buttons: []TownSurfaceButton{{Enabled: true}, {Enabled: true}}}}
		var town TownScreen = &plain
		if first {
			town = &clickingTipTown{plain}
		}
		a := latchTownApp(t, town)
		rec := &pressSoundRecorder{}
		a.SetAudio(rec, fixedUISoundBank())
		r := TownSurfaceButtonRect(TownSurfaceSchool, 0)
		p := r.Min.Add(r.Max).Div(2)
		for _, edge := range []string{"press", "release"} {
			if err := a.HeadlessPointer(edge, p.X, p.Y); err != nil {
				t.Fatal(err)
			}
		}
		want := 1
		if first {
			want = 0
		}
		wantClicks(t, rec, want, "a school button press and release")
	}
}
