package ui

import (
	"bytes"
	"image/color"
	"testing"
)

// The school's shipped art carries its own heading: no placeholder title is
// drawn over it. Without the art the fallback box still states a title.
func TestSchoolSurfaceDrawsNoTitleOverItsArt(t *testing.T) {
	art := &TownSchoolArt{Background: uniform(480, 480, color.RGBA{R: 61, A: 255})}
	compose := func(art *TownSchoolArt, title string) []byte {
		v := TownSurfaceView{Kind: TownSurfaceSchool, SchoolArt: art, SchoolClass: -1, Title: title, Font: panelFont()}
		return ComposeTownSurface(v).Pix
	}
	if !bytes.Equal(compose(art, "SKILL SCHOOL"), compose(art, "")) {
		t.Error("a title is drawn over the school art")
	}
	if bytes.Equal(compose(nil, "SKILL SCHOOL"), compose(nil, "")) {
		t.Error("the artless fallback no longer states its title: the check is vacuous")
	}
}
