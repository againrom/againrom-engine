package ui

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// dollPixelGolden is the slot, the tip and the composed tip every screen with
// a paper doll answered at every pointer position of its fixture, recorded
// before the screens shared one description of their doll. Each entry is
// hits/pixels:hash of the per-pixel answers.
var dollPixelGolden = map[string]string{
	"candidate/centred":            "52257/80906:24300b469d0562fe",
	"candidate/compose":            "8/8:279642f9d602cca4",
	"candidate/nil-mask":           "0/14412:d0308527e4be9e1b",
	"candidate/no-subject":         "0/14412:d0308527e4be9e1b",
	"candidate/not-tavern":         "0/8167:eecfe18e79f66c26",
	"candidate/other-rect":         "24579/68042:cb5ac06d278f5208",
	"candidate/small-mask":         "1226/14412:9c40b6bf90780142",
	"candidate/tiny-mask":          "0/14412:d0308527e4be9e1b",
	"candidate/uncentred":          "3902/14412:c95f2c17a5e53203",
	"generator/rom1.json/full":     "27572/41378:1d0054f80c652264",
	"generator/rom1.json/nil-mask": "0/8131:6bf6b2767548dd76",
	"generator/rom1.json/no-info":  "0/8131:6bf6b2767548dd76",
	"generator/rom1.json/small":    "427/8131:e1e251e284ea884d",
	"generator/rom2.json/full":     "27572/41378:1d0054f80c652264",
	"generator/rom2.json/nil-mask": "0/8131:6bf6b2767548dd76",
	"generator/rom2.json/no-info":  "0/8131:6bf6b2767548dd76",
	"generator/rom2.json/small":    "427/8131:e1e251e284ea884d",
	"hero/full":                    "25078/41378:9fbb89fd4b0a7a16",
	"hero/moved-pane":              "25078/41378:8f32623a38b496ec",
	"hero/nil-mask":                "0/8131:c864688fef940793",
	"hero/no-book":                 "2316/8131:67f969334961945c",
	"hero/no-subject":              "0/8131:c864688fef940793",
	"hero/small-mask":              "525/8131:a63d5dc19159601f",
	"hero/statistics":              "0/8131:c864688fef940793",
	"inventory/doll-hidden":        "27882/92317:9690afd85dd74622",
	"inventory/multi-unit":         "30208/92317:85cdbc9c280513af",
	"inventory/nil-mask":           "27882/92317:9690afd85dd74622",
	"inventory/no-figure":          "27882/92317:9690afd85dd74622",
	"inventory/ordinary":           "58090/92317:3d702a7578f11f83",
	"inventory/other-unit":         "0/92317:662366ee9baccb53",
	"inventory/small-mask":         "43611/92317:ee25d7c0a79479ce",
	"inventory/suppressed":         "65597/92317:72110ed682adc428",
	"inventory/worn-hidden":        "30208/92317:0198f649b3b4a794",
	"shop/compose":                 "6/6:447c0d74dc5eb072",
	"shop/compose-moved":           "6/6:76ca38ecd9680963",
	"shop/figure-fallback":         "2720/8107:9cbe349362acbab1",
	"shop/full":                    "27341/41054:37507b8d46f0214e",
	"shop/moved-pane":              "27328/41378:c9ffa7eaa6c9757b",
	"shop/nil-mask":                "0/8107:cd44c2120c875f13",
	"shop/no-book":                 "2720/8107:9cbe349362acbab1",
	"shop/no-character-layout":     "0/8107:cd44c2120c875f13",
	"shop/small-mask":              "1182/8107:9e746d980b0786c2",
	"shop/statistics":              "2720/8107:8d61b1a15806d96c",
	"shop/suppressed-mask":         "3414/8107:9defe7710ed7c67b",
	"square/moved-pane":            "47067/120110:7a3ebe94b2e9947c",
	"square/nil-mask":              "766/20669:c85c5b7e4f102039",
	"square/no-pane":               "766/20669:d4ab5801631dc675",
	"square/pane":                  "40634/120110:35a41341bcbe4bef",
	"square/statistics":            "766/20669:2c4c00b4fa3ff480",
}

type dollRecord struct {
	h     hash.Hash
	hits  int
	total int
}

func newDollRecord() *dollRecord { return &dollRecord{h: sha256.New()} }

func (r *dollRecord) add(hit bool, format string, args ...any) {
	r.total++
	if hit {
		r.hits++
	}
	fmt.Fprintf(r.h, format+"\n", args...)
}

func (r *dollRecord) addHash(hit bool, sum [32]byte) {
	r.total++
	if hit {
		r.hits++
	}
	r.h.Write(sum[:])
}

func (r *dollRecord) String() string {
	return fmt.Sprintf("%d/%d:%s", r.hits, r.total, hex.EncodeToString(r.h.Sum(nil))[:16])
}

func dollTip(t tooltipTarget) string {
	return fmt.Sprintf("%d|%s|%q|%v", t.kind, t.id, t.lines, t.font != nil)
}

// dollTipAt is the tip target get answers at p, asked at a fifth of the
// points: the target's identity is the same across a doll, and asking it is the
// costly step of the walk.
func dollTipAt(p image.Point, get func() tooltipTarget) string {
	if (p.X*7+p.Y*3)%5 != 0 {
		return "-"
	}
	return dollTip(get())
}

// dollPatternMask is a deterministic slot map with every slot 1..12 in blocks
// and transparent blocks and pixels between them.
func dollPatternMask(w, h, seed int) *SlotMask {
	m := &SlotMask{W: w, H: h, Slot: make([]uint8, w*h)}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			n := (x/5 + (y/7)*3 + seed) % 14
			if n >= 1 && n <= 12 && (x*31+y*17+seed)%11 != 0 {
				m.Slot[y*w+x] = uint8(n)
			}
		}
	}
	return m
}

// dollPatternInfo names every slot's lines after the slot, and leaves slot 7
// with none.
func dollPatternInfo(prefix string) [12][]string {
	var info [12][]string
	for i := range info {
		if i != 6 {
			info[i] = []string{fmt.Sprintf("%s%d", prefix, i), "#Value 1"}
		}
	}
	return info
}

func dollRegionPoints(rs []image.Rectangle, margin int, whole image.Rectangle, stride int) []image.Point {
	var pts []image.Point
	for _, r := range rs {
		r = r.Inset(-margin)
		for y := r.Min.Y; y < r.Max.Y; y++ {
			for x := r.Min.X; x < r.Max.X; x++ {
				pts = append(pts, image.Pt(x, y))
			}
		}
	}
	for y := whole.Min.Y; y < whole.Max.Y; y += stride {
		for x := whole.Min.X; x < whole.Max.X; x += stride {
			pts = append(pts, image.Pt(x, y))
		}
	}
	return pts
}

func dollInventoryWalk(v *Viewer, boxes []image.Rectangle) string {
	rec := newDollRecord()
	whole := image.Rect(0, 0, v.frameW, v.frameH)
	for _, p := range dollRegionPoints(boxes, 3, whole, 11) {
		w, wok := v.inventoryWornSlotAt(p.X, p.Y)
		d, dok := v.dollFigureSlotAt(p.X, p.Y)
		u, uok := v.dollFigureSlotAtUnsuppressed(p.X, p.Y)
		lines, lok := v.hoveredItemInfoAt(p.X, p.Y)
		v.hasCursor, v.cursorX, v.cursorY = true, p.X, p.Y
		rec.add(wok || dok || uok, "%d,%d|%d,%v|%d,%v|%d,%v|%q,%v|%s", p.X, p.Y, w, wok, d, dok, u, uok, lines, lok, dollTipAt(p, v.tooltipTarget))
	}
	return rec.String()
}

// dollWhole is the 640x480 frame every screen but the mission composes in.
var dollWhole = image.Rect(0, 0, 640, 480)

// dollScreenPoints is every pixel within one of the rectangles when dense, else
// every fifth pixel and the ring three wide around each edge, and in both cases
// every thirteenth pixel of the frame.
func dollScreenPoints(dense bool, rs ...image.Rectangle) []image.Point {
	if dense {
		return dollRegionPoints(rs, 1, dollWhole, 13)
	}
	var pts []image.Point
	for _, r := range rs {
		for y := r.Min.Y - 3; y < r.Max.Y+3; y++ {
			for x := r.Min.X - 3; x < r.Max.X+3; x++ {
				p := image.Pt(x, y)
				if x%5 == 0 && y%5 == 0 || !p.In(r.Inset(3)) {
					pts = append(pts, p)
				}
			}
		}
	}
	return append(pts, dollRegionPoints(nil, 0, dollWhole, 13)...)
}

func dollShopRecord(v ShopScreenView, pts []image.Point) string {
	rec := newDollRecord()
	for _, p := range pts {
		a, aok := shopDollSlotAt(v, v.SlotMask, p)
		b, bok := shopDollSlotAt(v, v.OrdinaryDollMask, p)
		c := shopDollAreaAt(shopCharacterView(v), p)
		g, gok := shopGridControlAt(v, p)
		lines, lok := ShopHoverLines(v, p)
		rec.add(aok || bok, "%v|%d,%v|%d,%v|%v|%+v,%v|%q,%v|%s", p, a, aok, b, bok, c, g, gok, lines, lok, dollTipAt(p, func() tooltipTarget { return shopTooltip(v, Words{}, p) }))
	}
	return rec.String()
}

func dollShopViews() map[string]ShopScreenView {
	info := dollPatternInfo("s")
	mk := func(mask *SlotMask) ShopScreenView {
		v := shopFullDollTestView(mask)
		v.SlotInfo = info
		return v
	}
	full := dollPatternMask(160, 240, 2)
	views := map[string]ShopScreenView{}
	views["full"] = mk(full)
	views["small-mask"] = mk(dollPatternMask(100, 200, 3))
	views["nil-mask"] = mk(nil)
	stats := mk(full)
	stats.Character.Statistics = true
	views["statistics"] = stats
	moved := mk(full)
	moved.Character.PaneRect = image.Rect(40, 30, 200, 272)
	views["moved-pane"] = moved
	nobook := mk(full)
	nobook.Character.NoBook = true
	views["no-book"] = nobook
	hidden := mk(full)
	layout := firstShopLayout
	layout.Character = false
	hidden.Layout = &layout
	views["no-character-layout"] = hidden
	fallback := mk(full)
	fallback.Figure = fallback.Character.Figure
	fallback.Character = TownCharacterView{}
	views["figure-fallback"] = fallback
	suppressed := mk(full)
	suppressed.OrdinaryDollMask = dollPatternMask(160, 240, 9)
	views["suppressed-mask"] = suppressed
	return views
}

func dollShopComposeRecord(v ShopScreenView) string {
	rec := newDollRecord()
	v.SlotInfo = [12][]string{}
	for i := range v.SlotInfo {
		v.SlotInfo[i] = []string{"A", "A"}
	}
	for _, p := range []image.Point{
		shopFigureRect.Min.Add(image.Pt(50, 100)), shopFigureRect.Min.Add(image.Pt(1, 1)),
		image.Pt(630, 470), image.Pt(5, 5), shopFigureRect.Min.Add(image.Pt(159, 239)), image.Pt(320, 240),
	} {
		frame := ComposeShopScreen(v, p, true, nil, false)
		rec.addHash(true, sha256.Sum256(frame.Pix))
	}
	return rec.String()
}

func dollSurfaceHeroViews() map[string]TownSurfaceView {
	info := dollPatternInfo("h")
	mk := func(edit func(*TownCharacterView), mask *SlotMask) TownSurfaceView {
		hero := TownCharacterView{HasSubject: true, Font: messageFont()}
		if edit != nil {
			edit(&hero)
		}
		return TownSurfaceView{Kind: TownSurfaceSchool, Font: messageFont(), Hero: hero, HeroSlotMask: mask, HeroSlotInfo: info}
	}
	full := dollPatternMask(160, 240, 4)
	return map[string]TownSurfaceView{
		"full":       mk(nil, full),
		"small-mask": mk(nil, dollPatternMask(90, 120, 5)),
		"nil-mask":   mk(nil, nil),
		"statistics": mk(func(h *TownCharacterView) { h.Statistics = true }, full),
		"no-subject": mk(func(h *TownCharacterView) { h.HasSubject = false }, full),
		"moved-pane": mk(func(h *TownCharacterView) { h.PaneRect = image.Rect(100, 20, 260, 262) }, full),
		"no-book":    mk(func(h *TownCharacterView) { h.NoBook = true }, full),
	}
}

func dollHeroRecord(t *testing.T, v TownSurfaceView, dense bool) string {
	t.Helper()
	rec := newDollRecord()
	s := &fakeSchoolAnimator{}
	s.view = v
	a := newTestApp(t, nil, nil)
	a.Layout(640, 480)
	a.SetTown(s)
	a.flow.showTown("")
	for _, p := range dollScreenPoints(dense, v.Hero.paneRect()) {
		lines, ok := TownHeroHoverLines(v, p)
		a.tooltipPoint = p
		rec.add(ok, "%v|%q,%v|%s", p, lines, ok, dollTipAt(p, a.tooltipTarget))
	}
	return rec.String()
}

func dollCandidateViews() map[string]TownSurfaceView {
	info := dollPatternInfo("c")
	figure := image.NewRGBA(image.Rect(0, 0, 160, 240))
	mk := func(mask *SlotMask, edit func(*TownSurfaceView)) TownSurfaceView {
		v := TownSurfaceView{Kind: TownSurfaceTavern, Font: messageFont(),
			Candidate:         TownCharacterView{HasSubject: true, Figure: figure, Font: messageFont(), CardFont: messageFont()},
			CandidateSlotMask: mask, CandidateSlotInfo: info}
		if edit != nil {
			edit(&v)
		}
		return firstTavernPage(v)
	}
	full := dollPatternMask(160, 240, 6)
	return map[string]TownSurfaceView{
		"centred":    mk(full, nil),
		"small-mask": mk(dollPatternMask(100, 200, 7), nil),
		"tiny-mask":  mk(&SlotMask{W: 4, H: 2, Slot: []uint8{1, 2, 0, 4, 5, 6, 7, 8}}, nil),
		"nil-mask":   mk(nil, nil),
		"uncentred": mk(full, func(v *TownSurfaceView) {
			v.Page.Person.Figure = RoomFigure{Rect: image.Rect(0, 238, 160, 480), Offset: image.Pt(3, 5)}
		}),
		"other-rect": mk(full, func(v *TownSurfaceView) {
			v.Page.Person.Figure = RoomFigure{Rect: image.Rect(20, 100, 150, 300), Centre: true, Offset: image.Pt(-2, 4)}
		}),
		"not-tavern": mk(full, func(v *TownSurfaceView) { v.Kind = TownSurfaceSchool }),
		"no-subject": mk(full, func(v *TownSurfaceView) { v.Candidate.HasSubject = false }),
	}
}

func dollCandidateRecord(t *testing.T, v TownSurfaceView, dense bool) string {
	t.Helper()
	rec := newDollRecord()
	s := &fakeSchoolAnimator{}
	s.view = v
	a := newTestApp(t, nil, nil)
	a.Layout(640, 480)
	a.SetTown(s)
	a.flow.showTown("")
	for _, p := range dollScreenPoints(dense, v.Page.Person.Figure.Rect, image.Rect(0, 238, 160, 480)) {
		lines, ok := TownCandidateHoverLines(v, p)
		a.tooltipPoint = p
		rec.add(ok, "%v|%q,%v|%s", p, lines, ok, dollTipAt(p, a.tooltipTarget))
	}
	return rec.String()
}

func dollCandidateComposeRecord() string {
	font := townShellRosterTestFont()
	figure := image.NewRGBA(image.Rect(0, 0, 4, 2))
	art := &TownTavernArt{
		Center:      uniform(320, 480, color.RGBA{R: 4, G: 5, B: 6, A: 0xff}),
		LeftStats:   uniform(160, 238, color.RGBA{R: 7, G: 8, B: 9, A: 0xff}),
		LeftPicture: uniform(160, 242, color.RGBA{R: 10, G: 11, B: 12, A: 0xff}),
	}
	info := [12][]string{}
	for i := range info {
		info[i] = []string{"Sword", "#Damage 3", "#Value 25"}
	}
	v := TownSurfaceView{Kind: TownSurfaceTavern, Font: font, TavernArt: art,
		Candidate: TownCharacterView{HasSubject: true, Figure: figure, Font: font, CardFont: font,
			Subject: PanelSubject{Name: "Candidate", HP: 4, MaxHP: 5}},
		CandidateSlotMask: &SlotMask{W: 4, H: 2, Slot: []uint8{1, 2, 3, 0, 5, 6, 7, 8}},
		CandidateSlotInfo: info,
	}
	rec := newDollRecord()
	for _, p := range []image.Point{{86, 359}, {87, 359}, {89, 359}, {86, 360}, {88, 360}, {85, 359}, {635, 475}, {10, 10}} {
		v.HasHover, v.Hover = true, p
		frame := ComposeTownSurface(firstTavernPage(v))
		rec.addHash(true, sha256.Sum256(frame.Pix))
	}
	return rec.String()
}

func dollSquareRecord(t *testing.T, name string, dense bool) string {
	t.Helper()
	info := dollPatternInfo("q")
	mask := dollPatternMask(160, 240, 8)
	town := paneSquareTown(2, true)
	town.view.HeroSlotMask, town.view.HeroSlotInfo = mask, info
	switch name {
	case "statistics":
		town.view.Hero.Statistics = true
	case "no-pane":
		town.view.Pane = false
	case "nil-mask":
		town.view.HeroSlotMask = nil
	case "moved-pane":
		town.view.Hero.PaneRect = image.Rect(100, 20, 260, 262)
	}
	town.view.Dolls = []TownSquareDoll{
		{At: image.Pt(11, 240), Clip: image.Rect(0, 238, 160, 480), Mask: dollPatternMask(100, 200, 2), Info: info},
		{At: image.Pt(200, 100), Clip: image.Rect(210, 110, 250, 150), Mask: mask, Info: info},
		{At: image.Pt(300, 50), Clip: image.Rect(0, 0, 640, 480), Mask: nil, Info: info},
	}
	a := paneSquareApp(t, town)
	rec := newDollRecord()
	for _, p := range dollScreenPoints(dense, town.view.Hero.paneRect(), image.Rect(0, 238, 160, 480), image.Rect(200, 100, 360, 340)) {
		hero, hok := town.view.heroHoverLines(p)
		var dolls string
		hit := hok
		for i, d := range town.view.Dolls {
			lines, ok := d.LinesAt(p)
			dolls += fmt.Sprintf("%d:%q,%v;", i, lines, ok)
			hit = hit || ok
		}
		a.tooltipPoint = p
		rec.add(hit, "%v|%q,%v|%s|%s", p, hero, hok, dolls, dollTipAt(p, a.tooltipTarget))
	}
	return rec.String()
}

func dollGenerator(t *testing.T, file string) *GeneratorDescription {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "game", "generators", file))
	if err != nil {
		t.Fatal(err)
	}
	d, err := DecodeGenerator(b)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func dollChargenRecord(t *testing.T, layout *GeneratorDescription, mask *SlotMask, info [12][]string, dense bool) string {
	t.Helper()
	stats := make([]ChargenStat, 4)
	for i := range stats {
		stats[i] = ChargenStat{Floor: 0, Ceiling: 50, Start: 10 + i}
	}
	figure := image.NewRGBA(image.Rect(0, 0, 160, 240))
	c := NewChargen(ChargenSetup{
		Name:      "Danath",
		PreCreate: &ChargenPreCreate{Art: &ChargenPresentation{Layout: layout, Font: messageFont()}},
		Choices:   []ChargenChoice{{Options: []string{"male", "female"}, Parent: -1}},
		Stats:     stats, Cost: triangular(50), Budget: 5000,
		Preview: func(ChargenResult) ChargenPreview {
			return ChargenPreview{Doll: figure, Worn: &ChargenDollHover{Mask: mask, Info: info}}
		},
	})
	c.Forward()
	if c.Stage() != DetailedStage {
		t.Fatal("fixture did not reach DetailedStage")
	}
	a := NewApp("chargen", nil, nil, nil)
	a.Layout(640, 480)
	a.flow.chargen, a.flow.screen = c, ScreenChargen
	rec := newDollRecord()
	for _, p := range dollScreenPoints(dense, layout.Detail.Doll.Rect.Rectangle()) {
		lines, n, ok := chargenDollHoverLines(c, p)
		rec.add(ok, "%v|%q,%d,%v|%s", p, lines, n, ok, dollTipAt(p, func() tooltipTarget { return a.chargenTooltip(p) }))
	}
	return rec.String()
}

// TestDollSlotAnswersAreTheRecordedOnes walks every pixel of each screen's
// doll, for both games where the screens differ, and compares the slot, the
// tip text and the tip's identity with what the screens answered before one
// description of the doll served them all.
func TestDollSlotAnswersAreTheRecordedOnes(t *testing.T) {
	got := map[string]string{}

	inv := map[string]func(v *Viewer, sub *InventorySubject){
		"ordinary": func(*Viewer, *InventorySubject) {},
		"multi-unit": func(v *Viewer, _ *InventorySubject) {
			v.sel = selection{5, 6}
		},
		"other-unit": func(v *Viewer, _ *InventorySubject) {
			v.sel = selection{6}
		},
		"suppressed": func(v *Viewer, sub *InventorySubject) {
			v.SetInventorySubject(*sub)
			v.SetDollSuppressedFigure(sub.ID, 4, solidPic(4, 4, color.RGBA{G: 0xff, A: 0xff}), dollPatternMask(160, 240, 5))
		},
		"small-mask": func(_ *Viewer, sub *InventorySubject) {
			sub.SlotMask = dollPatternMask(100, 200, 3)
		},
		"nil-mask": func(_ *Viewer, sub *InventorySubject) {
			sub.SlotMask = nil
		},
		"no-figure": func(_ *Viewer, sub *InventorySubject) {
			sub.Figure = nil
		},
		"doll-hidden": func(v *Viewer, _ *InventorySubject) {
			v.toggleHudPanel(hudPanelDoll)
		},
		"worn-hidden": func(v *Viewer, _ *InventorySubject) {
			v.toggleHudPanel(hudPanelWorn)
		},
	}
	for name, setup := range inv {
		v, _ := openInventoryAt(t)
		doll, dok := v.dollBox()
		worn, wok := v.wornBox()
		if !dok || !wok {
			t.Fatalf("inventory fixture draws no box: doll %v worn %v", dok, wok)
		}
		sub := v.invSubject
		sub.Figure = solidPic(4, 4, color.RGBA{R: 0xff, A: 0xff})
		sub.SlotMask = dollPatternMask(160, 240, 1)
		sub.SlotInfo = dollPatternInfo("w")
		setup(v, &sub)
		v.SetInventorySubject(sub)
		got["inventory/"+name] = dollInventoryWalk(v, []image.Rectangle{doll, worn})
	}

	shopPts := dollScreenPoints(true, shopFigureRect)
	shopSparse := dollScreenPoints(false, shopFigureRect)
	movedPts := dollScreenPoints(true, image.Rect(40, 30, 200, 272))
	for name, v := range dollShopViews() {
		pts := shopSparse
		switch name {
		case "full":
			pts = shopPts
		case "moved-pane":
			pts = movedPts
		}
		got["shop/"+name] = dollShopRecord(v, pts)
	}
	got["shop/compose"] = dollShopComposeRecord(dollShopViews()["full"])
	got["shop/compose-moved"] = dollShopComposeRecord(dollShopViews()["moved-pane"])

	for name, v := range dollSurfaceHeroViews() {
		got["hero/"+name] = dollHeroRecord(t, v, name == "full" || name == "moved-pane")
	}
	for name, v := range dollCandidateViews() {
		got["candidate/"+name] = dollCandidateRecord(t, v, name == "centred" || name == "other-rect")
	}
	got["candidate/compose"] = dollCandidateComposeRecord()

	for _, name := range []string{"pane", "statistics", "no-pane", "nil-mask", "moved-pane"} {
		got["square/"+name] = dollSquareRecord(t, name, name == "pane" || name == "moved-pane")
	}

	info := dollPatternInfo("g")
	for _, game := range []string{"rom1.json", "rom2.json"} {
		layout := dollGenerator(t, game)
		got["generator/"+game+"/full"] = dollChargenRecord(t, layout, dollPatternMask(160, 240, 10), info, true)
		got["generator/"+game+"/small"] = dollChargenRecord(t, layout, dollPatternMask(60, 100, 11), info, false)
		got["generator/"+game+"/nil-mask"] = dollChargenRecord(t, layout, nil, info, false)
		got["generator/"+game+"/no-info"] = dollChargenRecord(t, layout, dollPatternMask(160, 240, 10), [12][]string{}, false)
	}

	names := make([]string, 0, len(got))
	for name := range got {
		names = append(names, name)
	}
	sort.Strings(names)
	bad := false
	for _, name := range names {
		if dollPixelGolden[name] != got[name] {
			bad = true
			t.Errorf("%q: got %s, recorded %q", name, got[name], dollPixelGolden[name])
		}
	}
	for name := range dollPixelGolden {
		if _, ok := got[name]; !ok {
			t.Errorf("recorded case %q is no longer run", name)
		}
	}
	if bad {
		for _, name := range names {
			t.Logf("\t%q: %q,", name, got[name])
		}
	}
}
