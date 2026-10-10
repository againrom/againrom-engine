package ui

import (
	"image"
	"image/color"
	"testing"
)

// groupScene paints each named group with its own function; it stands for a
// room page in composer tests.
type groupScene map[string]func(dst *image.RGBA)

func (g groupScene) Paint(dst *image.RGBA, group string) {
	if paint := g[group]; paint != nil {
		paint(dst)
	}
}

// TestSchoolPaintsTheSceneGroupsInOrder: the school composer asks its room
// scene for the movies, the column and the diamond, once each and in that
// order (TOWN-428, TOWN-154).
func TestSchoolPaintsTheSceneGroupsInOrder(t *testing.T) {
	var order []string
	record := func(name string) func(*image.RGBA) {
		return func(*image.RGBA) { order = append(order, name) }
	}
	art := &TownSchoolArt{Background: uniform(480, 480, color.RGBA{R: 61, A: 255})}
	scene := groupScene{"movies": record("movies"), "column": record("column"), "diamond": record("diamond"), "centre": record("centre")}
	ComposeTownSurface(TownSurfaceView{Kind: TownSurfaceSchool, SchoolArt: art, SchoolClass: -1, HoverCell: -1, Scene: scene})
	if len(order) != 3 || order[0] != "movies" || order[1] != "column" || order[2] != "diamond" {
		t.Fatalf("painted groups = %v, want movies, column, diamond", order)
	}
}

func TestSchoolWithoutASceneLeavesTheRoomVisible(t *testing.T) {
	art := &TownSchoolArt{Background: uniform(480, 480, color.RGBA{R: 61, A: 255})}
	got := ComposeTownSurface(TownSurfaceView{Kind: TownSurfaceSchool, SchoolArt: art, SchoolClass: -1, HoverCell: -1})
	if got.RGBAAt(200, 60) != (color.RGBA{R: 61, A: 255}) {
		t.Fatal("a school without a scene obscured the background")
	}
	// Optional art still permits fixture and damaged-install text fallbacks.
	ComposeTownSurface(TownSurfaceView{Kind: TownSurfaceSchool, Scene: groupScene{}})
}

// fill paints a uniform picture at a point.
func fill(pic image.Image, at image.Point) func(*image.RGBA) {
	return func(dst *image.RGBA) {
		b := pic.Bounds()
		r := b.Add(at.Sub(b.Min)).Intersect(TownContentRegion)
		for y := r.Min.Y; y < r.Max.Y; y++ {
			for x := r.Min.X; x < r.Max.X; x++ {
				dst.Set(x, y, pic.At(b.Min.X+x-r.Min.X, b.Min.Y+y-r.Min.Y))
			}
		}
	}
}
