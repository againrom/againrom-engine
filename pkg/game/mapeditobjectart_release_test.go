package game

import (
	"bytes"
	"image"
	"strings"
	"testing"

	"againrom/internal/firstgame"
)

// The Objects brush paints each listed code with that code's own art: the big
// stones, the rocks, the small stones and the statues each differ from every
// other of their group, in the picture of the map.
func TestReleaseMapEditorObjectBrushPaintsEachCodesOwnArt(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	e := x.Editor()
	if err := e.Open("scenario/10.alm"); err != nil {
		t.Fatal(err)
	}
	e.Layout(1280, 800)
	rows := objectEdits(t, e).BrushObjects()
	groups := map[string][]int{}
	for _, r := range rows {
		n := strings.ToLower(r.Name)
		switch {
		case strings.HasPrefix(n, "big stone"):
			groups["big stone"] = append(groups["big stone"], r.Code)
		case strings.HasPrefix(n, "rock"):
			groups["rock"] = append(groups["rock"], r.Code)
		case strings.HasPrefix(n, "statue"):
			groups["statue"] = append(groups["statue"], r.Code)
		}
	}
	for _, g := range []string{"big stone", "rock", "statue"} {
		if len(groups[g]) < 2 {
			t.Fatalf("the list names %d %s objects", len(groups[g]), g)
		}
	}
	doc := e.Document()
	cell := image.Pt(doc.Width/2, doc.Height/2)
	e.ShowCell(cell)
	if err := e.Key("objects"); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 8; i++ {
		_ = e.Key("radiusdown")
	}
	at := 0
	crop := func(code int) []byte {
		for rows[at].Code != code {
			if rows[at].Code < code {
				_ = e.Key("down")
				at++
			} else {
				_ = e.Key("up")
				at--
			}
		}
		pointerStroke(e, []image.Point{cell})
		pic, _, err := e.HeadlessFrame()
		if err != nil {
			t.Fatal(err)
		}
		p := e.CellPoint(cell)
		box := image.Rect(p.X-64, p.Y-110, p.X+64, p.Y+40).Intersect(pic.Bounds())
		var out []byte
		for y := box.Min.Y; y < box.Max.Y; y++ {
			out = append(out, pic.Pix[pic.PixOffset(box.Min.X, y):pic.PixOffset(box.Max.X, y)]...)
		}
		return out
	}
	for g, codes := range groups {
		seen := map[int][]byte{}
		for _, code := range codes {
			pix := crop(code)
			for other, p := range seen {
				if bytes.Equal(p, pix) {
					t.Errorf("%s: codes %d and %d paint the same picture", g, other, code)
				}
			}
			seen[code] = pix
		}
	}
}
