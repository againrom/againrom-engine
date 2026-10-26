package ui

import (
	"image"
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"againrom/pkg/render/terrain"
)

// R2-ENGINE-588, SESS-VIEW-028: the view frame is the view's span in cells.
func TestMinimapViewFrameIsTheViewSpan(t *testing.T) {
	for _, screen := range []MissionScreen{{}, {ColumnW: 176, NoticeOnScreen: true}} {
		for name, g := range map[string]terrain.Grid{"flat": bigGrid(), "displaced": bigAltGrid()} {
			for _, panel := range []bool{false, true} {
				v, err := NewViewer("view-frame", g, &terrain.Tileset{})
				if err != nil {
					t.Fatal(err)
				}
				v.SetMissionScreen(screen)
				v.SetPackShown(panel)
				v.SetSpellBookShown(false)
				v.Layout(MissionFrameW, MissionFrameH)
				v.cam.X += 7
				v.cam.Y += 11
				v.cam.Clamp()
				wantCols := (v.cam.ViewW + 31) / 32
				wantRows := (v.cam.ViewH + 31) / 32
				got := v.minimapViewCells()
				if got.Dx() != wantCols || got.Dy() != wantRows {
					t.Errorf("%s column %d panel %t: frame %v is %dx%d cells, want %dx%d",
						name, screen.ColumnW, panel, got, got.Dx(), got.Dy(), wantCols, wantRows)
				}
				if !panel && wantRows != 24 {
					t.Errorf("%s: closed-panel view is %d rows, want 24", name, wantRows)
				}
			}
		}
	}
}

// The right column's transparent art shows black.
func TestMissionColumnBackdropIsBlack(t *testing.T) {
	for _, screen := range []MissionScreen{{}, {ColumnW: 176, NoticeOnScreen: true}} {
		v := columnViewer(t)
		v.SetMissionScreen(screen)
		v.Layout(MissionFrameW, MissionFrameH)
		dst := ebiten.NewImage(v.frameW, v.frameH)
		t.Cleanup(dst.Dispose)
		v.Draw(dst)
		want := image.Rect(v.cam.ViewW, 0, v.frameW, v.frameH)
		found := false
		for _, op := range v.canvasLog.ops {
			if op.kind == pixelOver && op.pic == nil && op.layer == nil && op.rect == want {
				found = true
				if op.solid != (color.RGBA{A: 0xff}) {
					t.Errorf("column %d: backdrop %v, want opaque black", screen.ColumnW, op.solid)
				}
			}
		}
		if !found {
			t.Errorf("column %d: no backdrop over %v", screen.ColumnW, want)
		}
	}
}

// A hover box's lower-left corner is on the pointer, as both originals draw it.
func TestTooltipBoxStandsAtThePointerHotspot(t *testing.T) {
	font := townShellRosterTestFont()
	frame := image.Rect(0, 0, 1024, 768)
	for _, p := range []image.Point{{872, 257}, {889, 188}} {
		target := tooltipTarget{tooltipCommand, "attack", []string{"ATTACK <A>"}, font}
		pic, at, ok := tooltipPicture(target, p, frame, nil, nil)
		if !ok || pic.Bounds().Dy() != 19 || at != image.Pt(p.X, p.Y-19) {
			t.Errorf("pointer %v: box %v at %v, want height 19 at %v", p, pic.Bounds(), at, image.Pt(p.X, p.Y-19))
		}
	}
}

// The view frame is white; the local dot's colour is profile data.
func TestMinimapFrameIsWhiteAndLocalDotIsProfileData(t *testing.T) {
	if minimapViewColor != (color.RGBA{R: 255, G: 255, B: 255, A: 255}) {
		t.Errorf("view colour %v, want white", minimapViewColor)
	}
	blue := color.RGBA{R: 82, G: 121, B: 231, A: 0xff}
	for _, tc := range []struct {
		screen MissionScreen
		want   color.RGBA
	}{
		{MissionScreen{}, color.RGBA{}},
		{MissionScreen{MinimapLocal: blue}, blue},
	} {
		v, err := NewViewer("dot", grid(8, 8), &terrain.Tileset{})
		if err != nil {
			t.Fatal(err)
		}
		v.SetMissionScreen(tc.screen)
		v.SetLocalOwner(1)
		v.SetEntities([]MapEntity{{ID: 1, Owner: 1, Cell: image.Pt(2, 2), Life: LifeAlive}})
		v.SetFog(nil, 0, 0)
		marks := v.minimapMarks()
		if len(marks) != 1 || !marks[0].Local || marks[0].Colour != tc.want {
			t.Errorf("profile %+v: marks %+v, want one local mark of colour %v", tc.screen, marks, tc.want)
		}
	}
}
