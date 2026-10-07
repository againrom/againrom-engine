package ui

import (
	"image"
	"image/color"
	"testing"
)

// windowFace is a speaker's 160x240 picture that is transparent except for one
// opaque block, so most of its window shows whatever lies under it.
func windowFace(block image.Rectangle, c color.RGBA) *image.RGBA {
	pic := image.NewRGBA(image.Rect(0, 0, 160, 240))
	for y := block.Min.Y; y < block.Max.Y; y++ {
		for x := block.Min.X; x < block.Max.X; x++ {
			pic.SetRGBA(x, y, c)
		}
	}
	return pic
}

// The pane's backdrop is cut at the speaker's own window and lies under the
// speaker's picture (DIV-1485): a transparent face pixel shows the backdrop's
// pixel at the same picture coordinate, an opaque one shows the face, and no
// pixel outside the window changes.
func TestThePortraitBackdropIsCutAtTheSpeakersWindow(t *testing.T) {
	f := panelFont()
	green := color.RGBA{G: 0xff, A: 0xff}
	for _, tc := range []struct {
		name string
		win  image.Rectangle
	}{
		{"a stated window", NoticeFaceWindow(34, 21)},
		{"the default window", image.Rectangle{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bare := noticePaneLayout().WithFaceWindow(tc.win)
			bare.Frame = &DialogFrame{}
			backed := bare
			backed.Frame = &DialogFrame{PortraitBack: coordPicture()}
			w := backed.PortraitWindow
			face := windowFace(image.Rect(w.Min.X+10, w.Min.Y+30, w.Min.X+40, w.Min.Y+60), green)
			without := RenderNotice(bare, f, "hello", face)
			with := RenderNotice(backed, f, "hello", face)
			at := backed.Portrait.Min.Add(backed.PortraitAt)
			window := image.Rectangle{Min: at, Max: at.Add(w.Size())}
			backdrop, figure := 0, 0
			for y := 0; y < with.Bounds().Dy(); y++ {
				for x := 0; x < with.Bounds().Dx(); x++ {
					pt, got := image.Pt(x, y), with.RGBAAt(x, y)
					if !pt.In(window) {
						if want := without.RGBAAt(x, y); got != want {
							t.Fatalf("(%d,%d) outside the window = %v, want the unchanged %v", x, y, got, want)
						}
						continue
					}
					src := pt.Sub(at).Add(w.Min)
					want := color.RGBA{R: uint8(src.X), G: uint8(src.Y), A: 0xff}
					if face.RGBAAt(src.X, src.Y).A == 0xff {
						want = green
						figure++
					} else {
						backdrop++
					}
					if got != want {
						t.Fatalf("picture pixel (%d,%d) = %v, want %v", src.X, src.Y, got, want)
					}
				}
			}
			if backdrop == 0 || figure == 0 {
				t.Fatalf("vacuous window: %d backdrop and %d figure pixels", backdrop, figure)
			}
		})
	}
}

// The map screen's dialogue takes the backdrop from the viewer's own frame,
// so a mission speaker stands on it as a town speaker does.
func TestAMissionSpeakerStandsOnThePortraitBackdrop(t *testing.T) {
	v := noticeViewer(t)
	v.SetDialogFrame(&DialogFrame{PortraitBack: coordPicture()})
	v.SetDialogue(Dialogue{Text: "hello", Portrait: true, Speaks: true,
		Face: image.NewRGBA(image.Rect(0, 0, 160, 240)), FaceWindow: NoticeFaceWindow(40, 10)})
	pic, _, _, ok := v.noticePresent()
	if !ok {
		t.Fatal("no dialogue composed")
	}
	l := v.noticeLayout()
	// The supplied engine canvas policy descends the window and surface rows.
	// Native final exposure remains Unknown.
	at := l.Portrait.Min.Add(image.Pt(8, 5))
	if got, want := pic.RGBAAt(at.X, at.Y), (color.RGBA{R: 41, G: 8, A: 0xff}); got != want {
		t.Errorf("the pane's window corner = %v, want the backdrop's %v", got, want)
	}
	if got, want := pic.RGBAAt(at.X, at.Y-1), (color.RGBA{A: 0xff}); got != want {
		t.Errorf("the row above the window = %v, want the pane's black ground %v", got, want)
	}
}
