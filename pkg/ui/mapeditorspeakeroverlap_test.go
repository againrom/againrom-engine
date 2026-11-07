package ui

import (
	"image"
	"strings"
	"testing"
)

// A big unit that shows speaker 5 covers a 3x3 block; a small unit that shows
// no speaker stands inside the block. The click picks the big unit wherever
// the map's own selection finds it, on the small unit's cell too.
func TestSpeakerPickReachesAUnitUnderAnotherUnitsCell(t *testing.T) {
	big, small := image.Pt(30, 10), image.Pt(31, 11)
	for _, lang := range []string{"en", "ru"} {
		t.Run(lang, func(t *testing.T) {
			e, _ := speakerPickEditor(t, lang)
			// Host unit 0 shows speaker 5; host unit 1 shows none.
			e.doc.Records = append(e.doc.Records,
				InspectionRecord{Kind: "Unit", Index: 0, Spatial: true, Cell: big, Size: image.Pt(3, 3), Player: 1},
				InspectionRecord{Kind: "Unit", Index: 1, Spatial: true, Cell: small, Size: image.Pt(1, 1), Player: 1})
			speaker := func() string { return e.obj.buf[1] }
			for _, cell := range []image.Point{small, big, big.Add(image.Pt(2, 2))} {
				e.obj.buf[1] = "-1"
				pressSpeakerPick(t, e, 0)
				clickCell(e, cell)
				if e.pick != pickNone || speaker() != "5" {
					t.Fatalf("a click at %v left pick %q, speaker %q", cell, e.pick, speaker())
				}
			}
		})
	}
}

// A click on a unit that shows no speaker, with no unit that does under it,
// leaves the speaker as it was, keeps the pick armed and says why.
func TestSpeakerPickSaysWhyOnARefusedUnit(t *testing.T) {
	e, _ := speakerPickEditor(t, "en")
	pressSpeakerPick(t, e, 0)
	clickCell(e, cellUnshownUnit)
	if e.pick != pickParamUnit || e.obj.buf[1] != "-1" {
		t.Fatalf("a refused click left pick %q, speaker %q", e.pick, e.obj.buf[1])
	}
	if !strings.Contains(e.statusText(), e.word("editor.param.nospeakerunit")) || e.pickWhy == "" {
		t.Fatalf("the refusal is silent: status %q, why %q", e.statusText(), e.pickWhy)
	}
}
