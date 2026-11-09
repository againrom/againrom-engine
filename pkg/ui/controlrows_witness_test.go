package ui

import (
	"bufio"
	"fmt"
	"image"
	"os"
	"strconv"
	"strings"
	"testing"
)

// The control rows witness: for each surface whose controls read controlLayout
// rows, every control's rectangle and the answer of the surface's hit test on
// a 2-pixel grid of the 640x480 frame; for tips, the rectangle the placement
// gives at the page's four edges. The golden holds only numbers, so the
// rectangles a change moves name themselves as a number that differs.
const controlRowsGolden = "testdata/control_rows.golden"

// witnessRLE records a value per grid point as value,count pairs.
type witnessRLE struct {
	out []int
	val int
	n   int
}

func (w *witnessRLE) add(v int) {
	if w.n > 0 && v == w.val {
		w.n++
		return
	}
	w.flush()
	w.val, w.n = v, 1
}

func (w *witnessRLE) flush() {
	if w.n > 0 {
		w.out = append(w.out, w.val, w.n)
	}
	w.n = 0
}

func witnessGrid(hit func(p image.Point) int) []int {
	w := &witnessRLE{}
	for y := 0; y < 480; y += 2 {
		for x := 0; x < 640; x += 2 {
			w.add(hit(image.Pt(x, y)))
		}
	}
	w.flush()
	return w.out
}

func rectNums(r image.Rectangle) []int { return []int{r.Min.X, r.Min.Y, r.Max.X, r.Max.Y} }

func boolNum(b bool) int {
	if b {
		return 1
	}
	return 0
}

// controlRowsLines is the witness as lines of decimal numbers: the surface
// number, the record number, then the record's numbers.
func controlRowsLines() []string {
	var out []string
	emit := func(surface, record int, nums []int) {
		parts := []string{strconv.Itoa(surface), strconv.Itoa(record)}
		for _, n := range nums {
			parts = append(parts, strconv.Itoa(n))
		}
		out = append(out, strings.Join(parts, " "))
	}

	// 1: the tip panel, its two controls and its two hit tests.
	font := panelFont()
	panel := TipPanelView{Rect: image.Rect(200, 300, 400, 450), Text: "Tips", Art: &TipPanelArt{}, Font: font}
	emit(1, 0, rectNums(TipPanelCloseRect(panel.Rect)))
	emit(1, 1, rectNums(TipPanelToggleRect(panel.Rect)))
	emit(1, 2, witnessGrid(func(p image.Point) int {
		k, ok := TipPanelControlAt(panel, p)
		return int(k)*2 + boolNum(ok)
	}))
	for down := 0; down < 2; down++ {
		emit(1, 3+down, witnessGrid(func(p image.Point) int {
			k, ok := TipPanelEventAt(panel, p, down == 1)
			return int(k)*2 + boolNum(ok)
		}))
	}

	// 2: the drop gold dialog.
	gold := &Viewer{frameW: 640, frameH: 480}
	edit, action, cancel := gold.goldModalControls()
	emit(2, 0, rectNums(edit))
	emit(2, 1, rectNums(action))
	emit(2, 2, rectNums(cancel))
	emit(2, 3, witnessGrid(func(p image.Point) int {
		f, ok := gold.goldControlAt(p)
		return int(f)*2 + boolNum(ok)
	}))

	// 3: the documents panel.
	for i := docControlLeft; i <= docControlOK; i++ {
		emit(3, i, rectNums(docControlRect(i)))
	}
	emit(3, 3, witnessGrid(func(p image.Point) int {
		c, ok := documentControlAt(p.X, p.Y)
		return c*2 + boolNum(ok)
	}))

	// 4: the character creation page, each control and the mask-less hit test.
	pre := witnessChargen()
	for id := chargenName; id <= chargenRestore; id++ {
		emit(4, 100+int(id), rectNums(preControlRect(pre, id)))
	}
	emit(4, 1000, witnessGrid(func(p image.Point) int { return int(preControlAt(pre, p)) }))

	// 5: tip placement, the hover box at the page edges.
	frame := image.Rect(0, 0, 640, 480)
	font2 := panelFont()
	target := tooltipTarget{tooltipText, "witness", []string{"Long enough hover help text for the box", "second line"}, font2}
	edges := []image.Point{{0, 0}, {639, 0}, {0, 479}, {639, 479}, {320, 240}, {639, 240}, {320, 479}}
	for i, p := range edges {
		_, at, ok := tooltipPicture(target, p, frame, nil, nil)
		pic, _, _ := tooltipPicture(target, p, frame, nil, nil)
		emit(5, i, append(rectNums(pic.Bounds().Add(at)), boolNum(ok)))
		bpic, bat, bok := placeHover(hoverDescription{target.lines, font2, hoverStyleFor(hoverStyleShop, hoverParams{}), hoverAnchor{hoverBeside, p, frame}}, nil)
		emit(5, 100+i, append(rectNums(bpic.Bounds().Add(bat)), boolNum(bok)))
		inner := image.Rect(50, 50, 590, 430)
		ipic, iat, iok := placeHover(hoverDescription{target.lines, font2, nil, hoverAnchor{hoverAbove, p, inner}}, nil)
		emit(5, 200+i, append(rectNums(ipic.Bounds().Add(iat)), boolNum(iok)))
		jpic, jat, jok := placeHover(hoverDescription{target.lines, font2, nil, hoverAnchor{hoverBeside, p, inner}}, nil)
		emit(5, 300+i, append(rectNums(jpic.Bounds().Add(jat)), boolNum(jok)))
	}

	// 6: tip placement, the popup row, direction, portrait and column head
	// tips with their popup at each corner of the page.
	long := strings.Repeat("brigands gone mad ", 6)
	icon := image.NewRGBA(image.Rect(0, 0, 40, 40))
	dirs := []InspectionChoice{{Value: 1, Label: "North of the far gate"}, {Value: 2, Label: "South of the far gate"}, {Value: 3, Label: "East of the far gate"}, {Value: 4, Label: "West of the far gate"}}
	pad := []image.Point{{0, -1}, {0, 1}, {1, 0}, {-1, 0}}
	grid := func() *popupGrid {
		return &popupGrid{Side: relationSide, Rows: 2, Cols: 2, HeadW: 60, HeadH: 20, RowHeads: []string{"1: Self", "2: Enemy"}, ColHeads: []string{"one long column head " + long, "2"},
			Cells: []popupGridCell{{Inert: true}, {Label: "4"}, {Label: "5"}, {Inert: true}}}
	}
	at := []image.Point{{0, 0}, {280, 0}, {0, 300}, {280, 300}, {120, 360}}
	for c, pt := range at {
		s := popupSpec{Title: "T", At: pt, Width: 360, Bounds: frame, LabelWidth: 120,
			Rows: []popupRow{
				{Kind: popupText, Label: "Tip row", Value: "v", Tip: long},
				{Kind: popupText, Label: long, Value: "cut row"},
				{Kind: popupPick, Label: "Direction", Value: "1"},
				{Kind: popupText, Label: "Portrait", Value: "p", PortraitTip: long, Portrait: icon},
			}}
		s.Rows[2] = s.Rows[2].withChoices(dirs, pad, "1")
		s.Grid = grid()
		l := layoutPopup(s, font2, font2)
		pts := []image.Point{}
		for _, rr := range l.Rows {
			pts = append(pts, rr.Rect.Min.Add(rr.Rect.Size().Div(2)))
			for _, b := range rr.Buttons {
				pts = append(pts, b.Min.Add(b.Size().Div(2)))
			}
			if s.Rows[rr.Index].Portrait != nil {
				pts = append(pts, rowPortraitBox(rr).Min.Add(image.Pt(1, 1)))
			}
		}
		for _, r := range l.GridColHeads {
			pts = append(pts, r.Min.Add(image.Pt(1, 1)))
		}
		// The page, then bounds that end inside the popup at each edge, so
		// every pull, flip, push and clip of the placement runs.
		page := []image.Rectangle{frame,
			image.Rect(0, 0, 640, l.Rect.Max.Y-24),
			image.Rect(0, 0, l.Rect.Max.X-40, 480),
			image.Rect(l.Rect.Min.X+40, l.Rect.Min.Y+40, 640, 480),
			image.Rect(l.Rect.Min.X+30, l.Rect.Min.Y+30, l.Rect.Max.X-30, l.Rect.Max.Y-30)}
		// A page ending just under each row and column head flips its tip.
		// A sliver of page over a row pushes the tip back in and clips it.
		for _, rr := range l.Rows {
			page = append(page, image.Rect(0, 0, 640, rr.Rect.Max.Y+3), image.Rect(0, rr.Rect.Min.Y, 640, rr.Rect.Min.Y+6))
		}
		for _, r := range l.GridColHeads {
			page = append(page, image.Rect(0, 0, 640, r.Max.Y+3))
		}
		for b, bounds := range page {
			for j, p := range pts {
				tip, ok := popupTipAt(bounds, font2, font2, s, l, p, true)
				emit(6, c*10000+b*100+j, append(rectNums(tip.Rect), boolNum(ok), len(tip.Lines)))
			}
		}
	}
	return out
}

func witnessChargen() *Chargen {
	font := panelFont()
	l := testGenerator()
	pic := func(w, h int) image.Image { return image.NewRGBA(image.Rect(0, 0, w, h)) }
	art := &ChargenPresentation{Layout: l, Font: font, Forward: pic(40, 30)}
	for i := range art.Levels {
		art.Levels[i][0] = pic(60+i*5, 20)
	}
	for i := range art.Choices {
		art.Choices[i][0] = pic(50, 70+i*3)
	}
	c := NewChargen(ChargenSetup{PreCreate: &ChargenPreCreate{Art: art}, TipArt: &TipPanelArt{}, Tips: testChargenTips([3]string{"Tips"}, "", "", "")})
	return c
}

func readControlRowsGolden(t *testing.T) []string {
	t.Helper()
	f, err := os.Open(controlRowsGolden)
	if err != nil {
		t.Fatalf("control rows golden: %v", err)
	}
	defer f.Close()
	var out []string
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		if strings.Trim(sc.Text(), "0123456789 -") != "" {
			t.Fatalf("golden line holds more than numbers: %q", sc.Text())
		}
		out = append(out, sc.Text())
	}
	return out
}

func diffControlRows(want, got []string) string {
	if len(want) != len(got) {
		return fmt.Sprintf("%d golden lines, %d computed", len(want), len(got))
	}
	for i := range want {
		if want[i] != got[i] {
			return fmt.Sprintf("line %d:\n golden   %s\n computed %s", i+1, want[i], got[i])
		}
	}
	return ""
}

// TestControlRowsWitness holds every migrated surface's controls and hit
// answers and every tip placement byte equal to the golden recorded before
// any surface moved. AGAINROM_CONTROL_ROWS_RECORD=1 rewrites it.
func TestControlRowsWitness(t *testing.T) {
	got := controlRowsLines()
	if os.Getenv("AGAINROM_CONTROL_ROWS_RECORD") == "1" {
		if err := os.WriteFile(controlRowsGolden, []byte(strings.Join(got, "\n")+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	if d := diffControlRows(readControlRowsGolden(t), got); d != "" {
		t.Fatalf("a control rectangle, hit answer or tip placement moved: %s", d)
	}
}

// TestControlRowsWitnessFailsAPlantedFault checks the comparison itself: one
// number changed anywhere is a failure.
func TestControlRowsWitnessFailsAPlantedFault(t *testing.T) {
	want := readControlRowsGolden(t)
	got := append([]string(nil), want...)
	got[0] = strings.Replace(got[0], "0", "1", 1) + " 7"
	if diffControlRows(want, got) == "" {
		t.Fatal("a planted change to one witness number was not reported")
	}
}
