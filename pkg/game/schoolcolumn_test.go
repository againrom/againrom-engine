package game

import (
	"fmt"
	"image"
	"image/color"
	"reflect"
	"strings"
	"testing"
	"time"

	"againrom/pkg/audio"
	"againrom/pkg/ui"
)

type schoolRotateRecorder struct {
	samples []audio.Sample
	places  []audio.Placement
}

func (r *schoolRotateRecorder) Play(s audio.Sample, p audio.Placement) {
	r.samples = append(r.samples, s)
	r.places = append(r.places, p)
}

func (r *schoolRotateRecorder) RequestSample(s audio.Sample, request audio.Request) audio.Voice {
	r.Play(s, request.Placement)
	return nil
}

type legacySoundRecorder schoolRotateRecorder

func (r *legacySoundRecorder) Play(s audio.Sample, placement audio.Placement) {
	(*schoolRotateRecorder)(r).Play(s, placement)
}

func columnSchool(t *testing.T) (*FrontEnd, *townScreen, *time.Time, *schoolRotateRecorder) {
	t.Helper()
	f := shellFrontEnd()
	member := f.Carried[0]
	member.Name, member.Mage = "Mage", true
	f.Carried = append(f.Carried, member, f.Carried[0])
	now := time.Unix(100, 0)
	f.TownAnimationNow = func() time.Time { return now }
	schoolArt, err := LoadTownSchoolArt(townSchoolSource())
	if err != nil {
		t.Fatal(err)
	}
	f.TownSchoolArt = resolved(schoolArt, nil)
	recorder := &schoolRotateRecorder{}
	f.SoundPlayer = recorder
	f.SoundBank = &SoundBank{named: map[string]soundCacheEntry{
		"town/school/rotate.wav": {sample: audio.Sample{Rate: audio.DeviceRate, PCM: []int16{17, -2}}, ok: true},
	}}
	s := f.townUI
	s.Back()
	s.Choose(2)
	return f, s, &now, recorder
}

func TestSchoolColumnStrictGateBothDirectionsAndNoCatchup(t *testing.T) {
	_, s, now, recorder := columnSchool(t)
	if s.schoolColumn.frame != 0 || s.schoolColumn.target != 0 || !s.schoolColumn.ready {
		t.Fatalf("fighter entry = %+v", s.schoolColumn)
	}
	s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlNext}, false)
	if len(recorder.samples) != 1 || s.schoolColumn.frame != 0 || s.schoolColumn.target != 15 {
		t.Fatal("class selection must start once without skipping the displayed frame")
	}
	for _, frame := range []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15} {
		*now = now.Add(83 * time.Millisecond)
		paintDiamondSchool(t, s)
		if s.schoolColumn.frame != frame-1 {
			t.Fatalf("exactly 83ms advanced to %d", s.schoolColumn.frame)
		}
		*now = now.Add(time.Millisecond)
		paintDiamondSchool(t, s)
		if s.schoolColumn.frame != frame {
			t.Fatalf("84ms frame = %d, want %d", s.schoolColumn.frame, frame)
		}
		paintDiamondSchool(t, s)
		if s.schoolColumn.frame != frame {
			t.Fatal("a second paint at the same instant advanced again")
		}
	}
	if s.TownSurface().SchoolClass != 1 {
		t.Fatal("the mage endpoint has no mage panel")
	}
	s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlPrevious}, false)
	for _, frame := range []int{14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0} {
		*now = now.Add(10 * time.Second)
		paintDiamondSchool(t, s)
		if s.schoolColumn.frame != frame {
			t.Fatalf("long paint gap frame = %d, want one step to %d", s.schoolColumn.frame, frame)
		}
	}
	before := s.schoolColumn
	*now = now.Add(time.Hour)
	paintDiamondSchool(t, s)
	if s.schoolColumn != before || len(recorder.samples) != 2 {
		t.Fatal("endpoint paints changed column state or replayed Rotate")
	}
	for _, p := range recorder.places {
		if p != (audio.Placement{Left: audio.GainUnit, Right: audio.GainUnit}) {
			t.Fatalf("rotation is not centered: %+v", p)
		}
	}
}

func TestSchoolColumnSelectionControlsAndRetargetPolicy(t *testing.T) {
	f, s, now, recorder := columnSchool(t)
	s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlCell, Index: 1}, false)
	if !s.TownSurface().Buttons[0].Enabled {
		t.Fatal("initial fighter skill is not selectable")
	}
	s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlNext}, false)
	hero, gold := f.Carried[1].Hero, f.Town.Gold()
	if s.schoolCell != schoolNoSelection || s.TownSurface().Buttons[0].Value != "0" {
		t.Fatal("class change retained a skill or price")
	}
	*now = now.Add(84 * time.Millisecond)
	paintDiamondSchool(t, s)
	view := s.TownSurface()
	if view.SchoolColumnFrame != 1 || view.SchoolClass != -1 {
		t.Fatal("intermediate frame published an endpoint panel")
	}
	for i, cell := range view.Cells {
		if cell.Enabled || cell.Selected {
			t.Fatalf("intermediate cell %d remains active", i)
		}
		s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlCell, Index: i}, false)
	}
	s.townSurfaceButton(0)
	if s.schoolCell != schoolNoSelection || f.Town.Gold() != gold || hero != f.Carried[1].Hero {
		t.Fatal("semantic selection/training bypassed the rotating panel guard")
	}
	// Go toward fighter from the displayed frame, not from either endpoint.
	s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlNext}, false)
	if s.schoolColumn.frame != 1 || s.schoolColumn.target != 0 || len(recorder.samples) != 2 {
		t.Fatal("rapid retarget jumped or failed to reverse")
	}
	last := s.schoolColumn.last
	s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlNext}, false) // fighter -> fighter
	if s.schoolColumn.last != last || len(recorder.samples) != 2 {
		t.Fatal("same-class member change rearmed or sounded the rotation")
	}
	*now = now.Add(84 * time.Millisecond)
	paintDiamondSchool(t, s)
	if s.schoolColumn.frame != 0 || s.TownSurface().SchoolClass != 0 {
		t.Fatal("reverse did not stop at fighter")
	}
	// A second click before the first frame cancels without a phantom sound.
	s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlNext}, false)
	s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlPrevious}, false)
	if s.schoolColumn.frame != 0 || s.schoolColumn.target != 0 || len(recorder.samples) != 3 {
		t.Fatal("zero-distance cancellation started a second rotation")
	}
}

func TestSchoolColumnReadsRoomBoundariesAndIndependentDiamond(t *testing.T) {
	_, s, now, recorder := columnSchool(t)
	s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlNext}, false)
	*now = now.Add(84 * time.Millisecond)
	paintDiamondSchool(t, s)
	before := s.schoolColumn
	*now = now.Add(time.Hour)
	for i := 0; i < 20; i++ {
		view := s.TownSurface()
		ui.TownSurfaceControlAt(view, image.Pt(240, 212))
		ui.ComposeTownSurface(view)
		s.Rows()
		s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlMode}, false)
	}
	if s.schoolColumn != before || len(recorder.samples) != 1 {
		t.Fatal("an inspection or statistics toggle advanced the column")
	}
	s.schoolDiamond.arm()
	s.room, s.dialogueBuilding = roomTalk, TownSchool
	paintDiamondSchool(t, s)
	if s.schoolColumn.frame != 2 || s.schoolDiamond.frame != 1 {
		t.Fatal("school dialogue did not paint both independent animations")
	}
	paintDiamondSchool(t, s)
	if s.schoolColumn.frame != 2 || s.schoolDiamond.frame != 2 {
		t.Fatal("the diamond inherited the column's millisecond gate")
	}
	s.room = roomSchool
	s.Back()
	before = s.schoolColumn
	*now = now.Add(time.Hour)
	s.Choose(0)
	paintDiamondSchool(t, s)
	if s.schoolColumn != before {
		t.Fatal("tavern paint advanced the hidden school column")
	}
	s.Back()
	s.Choose(2)
	if s.schoolColumn.frame != 15 || s.schoolColumn.target != 15 || len(recorder.samples) != 1 {
		t.Fatal("school reentry must initialize the selected class without a rotation")
	}
	s.resetForNewGame()
	if s.schoolColumn != (schoolColumnAnimation{}) {
		t.Fatal("new game retained the previous game's column")
	}
}

func TestSchoolColumnBackwardClockAndDegradedArt(t *testing.T) {
	for _, missing := range []string{"none", "art", "frames", "sound", "device"} {
		t.Run(missing, func(t *testing.T) {
			f, s, now, recorder := columnSchool(t)
			switch missing {
			case "art":
				f.TownSchoolArt = lazy[*ui.TownSchoolArt]{}
			case "frames":
				f.TownSchoolArt.Value().Column = [16]image.Image{}
			case "sound":
				f.SoundBank = nil
			case "device":
				f.SoundPlayer = nil
			}
			s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlNext}, false)
			before := s.schoolColumn
			*now = now.Add(-time.Hour)
			paintDiamondSchool(t, s)
			if s.schoolColumn != before {
				t.Fatal("backward clock advanced or rewound the animation")
			}
			if missing == "art" || missing == "frames" {
				if before.frame != 15 || before.target != 15 || len(recorder.samples) != 0 {
					t.Fatal("missing frames should retain the usable silent static fallback")
				}
			} else if before.target != 15 || before.frame != 0 {
				t.Fatal("missing sound prevented the animation")
			}
			if missing != "none" && len(recorder.samples) != 0 {
				t.Fatal("a silent degradation unexpectedly played sound")
			}
		})
	}
}

func TestLoadSchoolColumnAllFramesOrderedOpaqueAndValidated(t *testing.T) {
	art, err := LoadTownSchoolArt(townSchoolSource())
	if err != nil {
		t.Fatal(err)
	}
	for frame, pic := range art.Column {
		if got := pic.At(3, 3); got != (color.RGBA{R: uint8(0x40 + frame), G: 0x22, B: 0x11, A: 255}) {
			t.Fatalf("column frame %d = %v", frame, got)
		}
		for _, malformed := range []bool{false, true} {
			src := townSchoolSource()
			path := fmt.Sprintf("%scolumn/rt%04d.bmp", townSchoolArtPrefix, frame)
			if malformed {
				src[path] = synthBMP(147, 208, color.RGBA{A: 255})
			} else {
				delete(src, path)
			}
			if got, err := LoadTownSchoolArt(src); got != nil || err == nil || !strings.Contains(err.Error(), path) {
				t.Fatalf("bad frame %d: art=%v err=%v", frame, got, err)
			}
		}
	}
	if art.Faces[0] != art.Column[0] || art.Faces[1] != art.Column[15] {
		t.Fatal("static callers no longer share exact endpoint frames")
	}
	// Black is real column paint, not the key used for skill overlays.
	src := townSchoolSource()
	src[townSchoolArtPrefix+"column/rt0007.bmp"] = synthBMP(148, 208, color.RGBA{A: 255})
	art, err = LoadTownSchoolArt(src)
	if err != nil || art.Column[7].At(70, 90) != (color.RGBA{A: 255}) {
		t.Fatal("a pure-black rotation pixel lost its opaque alpha")
	}
}

func TestSchoolRotateNamedCacheAndSilentMissingBank(t *testing.T) {
	var nilBank *SoundBank
	if _, ok := nilBank.namedSample(schoolRotateSound); ok {
		t.Fatal("nil bank resolved a sound")
	}
	b := &SoundBank{}
	if _, ok := b.namedSample(schoolRotateSound); ok || len(b.named) != 1 {
		t.Fatal("missing named sound was not cached as absent")
	}
	_, s, _, rec := columnSchool(t)
	want := s.in.SoundBank.named["town/school/rotate.wav"].sample
	got, ok := s.in.SoundBank.namedSample(`Town\School\Rotate.wav`)
	if !ok || !reflect.DeepEqual(got, want) || len(rec.samples) != 0 {
		t.Fatal("named lookup must normalize the address without playing")
	}
}
