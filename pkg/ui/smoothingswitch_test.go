package ui

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"image"
	"io"
	"strings"
	"testing"
	"time"

	"againrom/pkg/video"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestSmoothingKeyChangesEveryScreen(t *testing.T) {
	for _, screen := range []Screen{ScreenMenu, ScreenTown, ScreenMap} {
		t.Run(fmt.Sprint(screen), func(t *testing.T) {
			a, v, _ := atOnMap(t)
			a.flow.screen = screen
			if screen == ScreenTown {
				a.SetTown(&fakeTown{})
			}
			var values GameOptionValues
			values[GameOptionSmoothing] = 1
			writes := 0
			a.SetGameOptionControls(GameOptionControls{
				Read: func(bool) GameOptionValues { return values },
				Write: func(_ bool, o GameOption, value int) error {
					values[o] = value
					v.SetGraphicsOptions(GraphicsOptions{Smoothing: value != 0})
					writes++
					return nil
				},
			})
			for _, on := range []bool{false, true} {
				if err := a.HeadlessKey("ctrl-o"); err != nil {
					t.Fatal(err)
				}
				if a.FrameSmoothing() != on || (values[GameOptionSmoothing] != 0) != on {
					t.Fatalf("switch on %v: App %v, stored %d", on, a.FrameSmoothing(), values[GameOptionSmoothing])
				}
			}
			if writes != 2 {
				t.Fatalf("got %d preference writes, want 2", writes)
			}
		})
	}
}

func TestSmoothingKeyKeepsTheMissionTick(t *testing.T) {
	ticks := func(toggle bool) int {
		a, v, _ := atOnMap(t)
		count := 0
		a.flow.tick = func() { count++ }
		in := atFrame(v.frameW/2, v.frameH/2)
		in.Smoothing = toggle
		a.step(in, atAt.Add(time.Second))
		return count
	}
	want := ticks(false)
	if got := ticks(true); want == 0 || got != want {
		t.Fatalf("Ctrl+O tick count %d, ordinary %d", got, want)
	}
}

func TestSmoothingDrawingReadsTheCurrentPreference(t *testing.T) {
	a := newTestApp(t, appRows(1), okLoader(t))
	value := 1
	a.SetGameOptionControls(GameOptionControls{
		Current: func(GameOption) int { return value },
		Read: func(bool) GameOptionValues {
			t.Fatal("frame read the settings page's store route")
			return GameOptionValues{}
		},
	})
	if !a.FrameSmoothing() {
		t.Fatal("current on value was ignored")
	}
	value = 0
	if a.frameFilter() != filterSharp {
		t.Fatal("current off value was ignored")
	}
}

func TestSmoothingOnFramesKeepTheirPixels(t *testing.T) {
	for _, kind := range []string{"menu", "town", "film", "mission", "zoomed mission"} {
		t.Run(kind, func(t *testing.T) {
			a := newTestApp(t, appRows(1), okLoader(t))
			switch kind {
			case "town":
				a = latchTownApp(t, &fakeTown{})
			case "mission", "zoomed mission":
				z := 1.0
				if kind == "zoomed mission" {
					z = 1.2
				}
				a.flow.viewer = zoomViewer(t, false, true, z)
				a.flow.screen = ScreenMap
			case "film":
				a.cutscene = video.NewPlayer(io.NopCloser(bytes.NewReader(syntheticMovie())))
				t.Cleanup(a.cutscene.Close)
				deadline := time.Now().Add(time.Second)
				for a.cutscene.FrameNumber() == 0 && time.Now().Before(deadline) {
					a.cutscene.Advance(time.Now())
					time.Sleep(time.Millisecond)
				}
				if a.cutscene.FrameNumber() != 1 {
					t.Fatal("synthetic film has no first frame")
				}
			}
			a.SetFrameSmoothing(true)
			a.Layout(900, 675)
			pix, _, err := a.HeadlessPicture(HeadlessRequest{Part: HeadlessPartFrame, Scaled: true})
			if err != nil {
				t.Fatal(err)
			}
			got := fmt.Sprintf("%x", sha256.Sum256(pix.Pix))
			t.Logf("%s %s", kind, got)
			if want := smoothFrameHashes[kind]; got != want {
				t.Fatalf("frame %s, main %s", got, want)
			}
		})
	}
}

var smoothFrameHashes = map[string]string{
	"menu":           "5491a7339a1b53e1f125051320b113a7c5d47fbf1358d4592e1750764fb40d0e",
	"town":           "0f7cf1691b535aa480cf6a86edca4acf3ecc39eb4283b672c8fb1d5c771e852a",
	"film":           "69d4fdb20e9972422f7512c95628b98be4670b8f283f74aa6aa7c922f948529a",
	"mission":        "f5ac8ddda7af9abaf529671107170d511f8230107a47c25f7f785023839b4974",
	"zoomed mission": "0ca6ec90c694c02bbed9b8aba0b02d72c9390e91a6e6c5b65d5baae86c935f06",
}

func TestFilmSmoothingOffUsesHardPixels(t *testing.T) {
	a := newTestApp(t, appRows(1), okLoader(t))
	a.cutscene = &video.Player{}
	a.Layout(900, 675)
	calls := recordFinalBlits(t)
	for _, on := range []bool{true, false} {
		a.SetFrameSmoothing(on)
		*calls = nil
		a.drawCutscene(ebiten.NewImage(900, 675))
		want := "catmullrom"
		if !on {
			want = "nearest"
		}
		path, _ := pathOf(*calls, func(c finalBlit) bool { return c.src == a.cutsceneCanvas })
		if path != want {
			t.Fatalf("on %v: film path %q, want %q", on, path, want)
		}
	}
}

func TestEditorFrameStaysSmoothWithSmoothingOff(t *testing.T) {
	e := NewMapEditor(nil, nil, nil)
	e.doc = &InspectionDocument{Viewer: geometryViewer(t)}
	e.doc.Viewer.SetFrameSmoothing(false)
	e.window, e.deviceScale = image.Pt(640, 480), 1.5
	if e.frameFilter() != filterSmooth {
		t.Fatalf("editor frame filter %v, want smooth", e.frameFilter())
	}
}

func TestTownAndMenuFramesFollowTheSharedSmoothing(t *testing.T) {
	for _, town := range []bool{false, true} {
		a := newTestApp(t, appRows(1), okLoader(t))
		if town {
			a = latchTownApp(t, &fakeTown{})
		}
		a.Layout(900, 675)
		var values GameOptionValues
		values[GameOptionSmoothing] = 1
		a.SetGameOptionControls(GameOptionControls{
			Read:  func(bool) GameOptionValues { return values },
			Write: func(_ bool, o GameOption, value int) error { values[o] = value; return nil },
		})
		calls := recordFinalBlits(t)
		for _, on := range []bool{true, false, true} {
			if a.FrameSmoothing() != on {
				if err := a.HeadlessKey("ctrl-o"); err != nil {
					t.Fatal(err)
				}
			}
			*calls = nil
			a.Draw(ebiten.NewImage(900, 675))
			path, _ := pathOf(*calls, func(c finalBlit) bool { return c.src.Bounds().Dx() >= 640 })
			want, note := "catmullrom", "catmull-rom"
			if !on {
				want, note = "sharp", "sharp bilinear"
			}
			_, extras, err := a.HeadlessPicture(HeadlessRequest{Part: HeadlessPartFrame, Scaled: true})
			if path != want || err != nil || !strings.Contains(extras.Note, note) {
				t.Fatalf("town %v on %v: GPU path %q, CPU %q, err %v", town, on, path, extras.Note, err)
			}
		}
	}
}

func TestDialogueReferenceKeepsTheMapOnTheTextSafeCanvas(t *testing.T) {
	v := zoomViewer(t, false, true, 1.2)
	v.SetFrameSmoothing(false)
	v.SetFont(panelFont())
	v.SetNotice("words", NoticeDialogue)
	_, stages, err := v.headlessMissionFrame(image.Pt(1920, 1080))
	if err != nil {
		t.Fatal(err)
	}
	note := strings.Join(stages, "; ")
	if !strings.Contains(note, "-> canvas: nearest") || !strings.Contains(note, "-> window 1920x1080: sharp bilinear") {
		t.Fatalf("dialogue reference bypassed the shared canvas route: %s", note)
	}
}
