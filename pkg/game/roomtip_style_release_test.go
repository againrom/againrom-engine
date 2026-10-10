package game

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"againrom/pkg/formats/spr256"
	"againrom/pkg/render/backdrop"
	"againrom/pkg/render/text"
	"againrom/pkg/ui"
)

type roomTipRaster struct {
	name          string
	view          ui.TipPanelView
	with, without *image.RGBA
	calls         []text.DrawCall
}

func releaseRoomTipRaster(t *testing.T, room string) roomTipRaster {
	t.Helper()
	f := releaseFront(t)
	s, ok := f.TownScreen().(*townScreen)
	if !ok {
		t.Fatal("TownScreen did not return a town screen")
	}
	var v ui.TipPanelView
	var compose func(bool) *image.RGBA
	switch room {
	case "town-square":
		x := s.TownSquareView()
		v = x.Tip
		compose = func(show bool) *image.RGBA {
			if !show {
				x.Tip = ui.TipPanelView{}
			}
			return ui.ComposeTownSquare(x)
		}
	case "shop":
		s.Choose(1)
		x := s.ShopScreen()
		v = x.TipPanel
		compose = func(show bool) *image.RGBA {
			if !show {
				x.TipPanel = ui.TipPanelView{}
			}
			return ui.ComposeShopScreen(x, image.Point{}, false, nil, false)
		}
	case "tavern", "school":
		door := 0
		if room == "school" {
			door = 2
		}
		s.Choose(door)
		x := s.TownSurface()
		v = x.Tip
		compose = func(show bool) *image.RGBA {
			if !show {
				x.Tip = ui.TipPanelView{}
			}
			return ui.ComposeTownSurface(x)
		}
	default:
		t.Fatalf("unknown room %q", room)
	}
	if !v.Showing() {
		t.Fatal("installed room tip is not showing")
	}
	text.ResetCapture()
	text.SetCapture(false)
	with := compose(true)
	text.StopCapture()
	calls := append([]text.DrawCall(nil), text.Captured()...)
	text.ResetCapture()
	return roomTipRaster{room, v, with, compose(false), calls}
}

func releaseTipFramePieces(t *testing.T) [9]*image.RGBA {
	t.Helper()
	f := releaseFront(t)
	raw, err := f.Archives.Containers.ReadFile(graphicsPrefix + "interface/lm.256")
	if err != nil {
		t.Fatal(err)
	}
	s, err := spr256.Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !s.HasPalette || len(s.Frames) < 18 {
		t.Fatal("installed lm sheet has no room-tip frame bank")
	}
	var pieces [9]*image.RGBA
	for i := range pieces {
		f := s.Frames[9+i]
		want := image.Pt(32, 32)
		if i == 0 || i == 2 || i == 7 {
			want.X = 48
		}
		if image.Pt(f.Width, f.Height) != want {
			t.Fatalf("tip frame %d size = %dx%d, want %v", 9+i, f.Width, f.Height, want)
		}
		pic := image.NewRGBA(image.Rectangle{Max: want})
		for n, p := range f.Pixels {
			if p.Opaque {
				c := s.Palette[p.Index]
				pic.SetRGBA(n%f.Width, n/f.Width, color.RGBA{c.R, c.G, c.B, 255})
			}
		}
		pieces[i] = pic
	}
	return pieces
}

type releaseTipTile struct {
	piece int
	at    image.Point
}

func releaseTipTiles(body image.Rectangle) []releaseTipTile {
	l, t, r, b := body.Min.X, body.Min.Y, body.Max.X, body.Max.Y
	list := []releaseTipTile{{1, image.Pt(l, t)}, {3, image.Pt(r-32, t)}, {6, image.Pt(l, b-32)}, {8, image.Pt(r-32, b-32)}}
	for x := 0; x < (body.Dx()-64)/48; x++ {
		list = append(list, releaseTipTile{2, image.Pt(l+32+x*48, t)}, releaseTipTile{7, image.Pt(l+32+x*48, b-32)})
	}
	for y := 0; y < (body.Dy()-64)/32; y++ {
		list = append(list, releaseTipTile{4, image.Pt(l, t+32+y*32)}, releaseTipTile{5, image.Pt(r-32, t+32+y*32)})
	}
	for y := 0; y < (body.Dy()-64)/32; y++ {
		for x := 0; x < (body.Dx()-64)/48; x++ {
			list = append(list, releaseTipTile{0, image.Pt(l+32+x*48, t+32+y*32)})
		}
	}
	return list
}

func assertReleaseTipFrame(t *testing.T, v roomTipRaster, pieces [9]*image.RGBA) {
	t.Helper()
	body := image.Rectangle{Min: v.view.Rect.Min, Max: v.view.Rect.Max.Sub(image.Pt(8, 8))}
	want := image.NewRGBA(v.without.Bounds())
	draw.Draw(want, want.Bounds(), v.without, v.without.Rect.Min, draw.Src)
	lookup, err := backdrop.NewLevel(backdrop.RGB565, backdrop.Full, 6)
	if err != nil {
		t.Fatal(err)
	}
	tiles := releaseTipTiles(body)
	for _, tile := range tiles {
		if tile.piece != 3 && tile.piece != 5 && tile.piece != 6 && tile.piece != 7 && tile.piece != 8 {
			continue
		}
		pic, at := pieces[tile.piece], tile.at.Add(image.Pt(8, 8))
		for y := 0; y < pic.Rect.Dy(); y++ {
			for x := 0; x < pic.Rect.Dx(); x++ {
				p := at.Add(image.Pt(x, y))
				if p.In(want.Bounds()) && pic.RGBAAt(x, y).A != 0 {
					want.SetRGBA(p.X, p.Y, lookup.Color(want.RGBAAt(p.X, p.Y)))
				}
			}
		}
	}
	fill := pieces[0].RGBAAt(24, 16)
	if fill.A != 255 || fill.G <= fill.R || fill.B <= fill.R {
		t.Fatalf("installed tip fill sample = %v, want opaque green-teal", fill)
	}
	for _, tile := range tiles {
		pic := pieces[tile.piece]
		draw.Draw(want, pic.Bounds().Add(tile.at), pic, pic.Bounds().Min, draw.Over)
	}
	textRect := image.Rect(v.view.Rect.Min.X+20, v.view.Rect.Min.Y+24, v.view.Rect.Max.X-28, v.view.Rect.Max.Y-36)
	closeRect := image.Rect(v.view.Rect.Max.X-120, v.view.Rect.Max.Y-40, v.view.Rect.Max.X-40, v.view.Rect.Max.Y-22)
	toggleRect := image.Rect(v.view.Rect.Min.X+40, v.view.Rect.Max.Y-40, v.view.Rect.Max.X-124, v.view.Rect.Max.Y-24)
	if ui.TipPanelTextRect(v.view.Rect) != textRect || ui.TipPanelCloseRect(v.view.Rect) != closeRect || ui.TipPanelToggleRect(v.view.Rect) != toggleRect {
		t.Errorf("room tip child rectangles differ from the list/checkbox/Close layout")
	}
	for _, r := range []image.Rectangle{closeRect, toggleRect} {
		if !r.In(body) {
			t.Errorf("control %v escapes body %v", r, body)
		}
	}
	if closeRect.Overlaps(toggleRect) {
		t.Error("Close overlaps toggle")
	}
	textCells := releaseTipBodyPaintedCells(v, textRect)
	checked, shadow := 0, 0
	for y := v.view.Rect.Min.Y; y < v.view.Rect.Max.Y; y++ {
		for x := v.view.Rect.Min.X; x < v.view.Rect.Max.X; x++ {
			p := image.Pt(x, y)
			if !p.In(want.Bounds()) || textCells[p] || p.In(closeRect.Inset(-5)) || p.In(toggleRect.Inset(-5)) {
				continue
			}
			checked++
			if !p.In(body) && want.RGBAAt(x, y) != v.without.RGBAAt(x, y) {
				shadow++
			}
			if got := v.with.RGBAAt(x, y); got != want.RGBAAt(x, y) {
				t.Fatalf("frame/fill/shadow at %v = %v, want %v", p, got, want.RGBAAt(x, y))
			}
		}
	}
	if checked == 0 || shadow == 0 {
		t.Fatalf("frame witness sampled %d pixels and %d visible shadow pixels", checked, shadow)
	}
}

func releaseTipBodyPaintedCells(v roomTipRaster, r image.Rectangle) map[image.Point]bool {
	fontGlyphs := map[*text.Glyph]bool{}
	for i := range v.view.Font.Glyphs {
		fontGlyphs[&v.view.Font.Glyphs[i]] = true
	}
	ink, shadow := color.RGBA{185, 159, 73, 255}, color.RGBA{8, 8, 8, 255}
	cells := map[image.Point]bool{}
	for _, c := range v.calls {
		if !fontGlyphs[c.Glyph] || c.Glyph.Width <= 0 || c.Y < r.Min.Y || c.Y > r.Max.Y || !((c.Color == ink && !c.Flat) || (c.Color == shadow && c.Flat)) {
			continue
		}
		for n, pixel := range c.Glyph.Pixels {
			if !pixel.Painted {
				continue
			}
			at := image.Pt(c.X+n%c.Glyph.Width, c.Y+n/c.Glyph.Width)
			if at.In(v.with.Bounds()) && (c.Clip.Empty() || at.In(c.Clip)) {
				cells[at] = true
			}
		}
	}
	return cells
}

func tipGlyphRight(g *text.Glyph, spacing int) int {
	if g == nil || g.Width <= 0 {
		return 0
	}
	right := g.Advance + spacing
	for i, p := range g.Pixels {
		if p.Painted && i%g.Width+1 > right {
			right = i%g.Width + 1
		}
	}
	return right
}

type releaseTipParagraph struct {
	glyphs []*text.Glyph
	breaks map[int]bool
}

func releaseTipParagraphGlyphs(font *text.Font, source string) []releaseTipParagraph {
	space := font.GlyphFor(' ')
	var paragraphs []releaseTipParagraph
	letter := func(b byte) bool {
		return b >= 'A' && b <= 'Z' || b >= 'a' && b <= 'z' || b >= 0x80 && b <= 0xaf || b >= 0xe0 && b <= 0xef
	}
	for _, p := range strings.Split(source, "\r\n") {
		var glyphs []*text.Glyph
		breaks := map[int]bool{}
		for i := 0; i < len(p); i++ {
			if p[i] == '-' && i > 0 && letter(p[i-1]) {
				j := i + 1
				for j < len(p) && (p[j] == ' ' || p[j] == '\t') {
					j++
				}
				if j > i+1 && j < len(p) && letter(p[j]) {
					breaks[len(glyphs)] = true
					i = j - 1
					continue
				}
			}
			g := font.GlyphFor(p[i])
			if g != nil && g != space {
				glyphs = append(glyphs, g)
			}
		}
		if len(glyphs) != 0 {
			paragraphs = append(paragraphs, releaseTipParagraph{glyphs, breaks})
		}
	}
	return paragraphs
}

func assertReleaseTipText(t *testing.T, v roomTipRaster) {
	t.Helper()
	r := ui.TipPanelTextRect(v.view.Rect)
	ink, shadow := color.RGBA{185, 159, 73, 255}, color.RGBA{8, 8, 8, 255}
	lines := map[int][]text.DrawCall{}
	type key struct {
		glyph *text.Glyph
		x, y  int
	}
	seen := map[key]int{}
	bodyCalls := map[int]bool{}
	fontGlyphs := map[*text.Glyph]bool{}
	for i := range v.view.Font.Glyphs {
		fontGlyphs[&v.view.Font.Glyphs[i]] = true
	}
	faces := 0
	for i, c := range v.calls {
		if c.Color == shadow && c.Flat {
			seen[key{c.Glyph, c.X - 1, c.Y - 1}] = i
		}
		// The claimed list overlaps the control row; captions are not body text.
		at := image.Pt(c.X, c.Y)
		if c.Color != ink || c.Flat || !fontGlyphs[c.Glyph] || c.Y < r.Min.Y || c.Y >= r.Max.Y ||
			at.In(ui.TipPanelCloseRect(v.view.Rect)) || at.In(ui.TipPanelToggleRect(v.view.Rect)) {
			continue
		}
		if prior, ok := seen[key{c.Glyph, c.X, c.Y}]; !ok || prior >= i {
			t.Fatalf("tip glyph at (%d,%d) has no preceding flat shadow at (+1,+1)", c.X, c.Y)
		} else {
			bodyCalls[prior] = true
		}
		bodyCalls[i] = true
		painted := false
		for _, pixel := range c.Glyph.Pixels {
			painted = painted || pixel.Painted
		}
		if painted && (c.X < r.Min.X || c.X+tipGlyphRight(c.Glyph, v.view.Font.Spacing) > r.Max.X+1 || c.Y+c.Glyph.Height+1 > r.Max.Y+1) {
			t.Fatalf("tip glyph at (%d,%d) clips outside text rectangle %v", c.X, c.Y, r)
		}
		lines[c.Y] = append(lines[c.Y], c)
		faces++
	}
	if faces == 0 || !ui.TipPanelFits(v.view) {
		t.Fatal("tip text is missing or does not fit")
	}
	ys := make([]int, 0, len(lines))
	for y := range lines {
		ys = append(ys, y)
	}
	sort.Ints(ys)
	if ys[0] != r.Min.Y {
		t.Errorf("first tip baseline = %d, want %d", ys[0], r.Min.Y)
	}
	justified := 0
	paragraphs := releaseTipParagraphGlyphs(v.view.Font, v.view.Text)
	paragraph, consumed := 0, 0
	space, hyphen := v.view.Font.GlyphFor(' '), v.view.Font.GlyphFor('-')
	indent := v.view.Font.GlyphFor('@').Advance
	for i, y := range ys {
		if paragraph >= len(paragraphs) {
			t.Fatal("rendered tip has lines after the installed source ends")
		}
		line := lines[y]
		lastNonSpace := len(line) - 1
		for lastNonSpace >= 0 && line[lastNonSpace].Glyph == space {
			lastNonSpace--
		}
		start := r.Min.X
		if consumed == 0 {
			start += indent
		}
		if line[0].X != start {
			t.Errorf("line at y=%d starts at %d, want %d", y, line[0].X, start)
		}
		if i != 0 && y-ys[i-1] != v.view.Font.Height()+2 {
			t.Errorf("line pitch = %d, want font height plus 2", y-ys[i-1])
		}
		for j, call := range line {
			if call.Glyph == space {
				continue
			}
			if paragraph >= len(paragraphs) || consumed >= len(paragraphs[paragraph].glyphs) {
				t.Fatal("rendered tip contains glyphs after the installed source ends")
			}
			if j == lastNonSpace && call.Glyph == hyphen && paragraphs[paragraph].glyphs[consumed] != hyphen {
				if !paragraphs[paragraph].breaks[consumed] {
					t.Fatalf("line at y=%d inserts a hyphen outside installed paragraph %d markers at glyph %d", y, paragraph, consumed)
				}
				continue
			}
			if call.Glyph != paragraphs[paragraph].glyphs[consumed] {
				t.Fatalf("line at y=%d differs from installed paragraph %d at glyph %d", y, paragraph, consumed)
			}
			consumed++
		}
		if lastNonSpace < 0 {
			t.Fatal("installed tip contains a glyphless rendered line")
		}
		last := line[lastNonSpace]
		paragraphEnd := consumed == len(paragraphs[paragraph].glyphs)
		if !paragraphEnd {
			if absTipInt(last.X+tipGlyphRight(last.Glyph, v.view.Font.Spacing)-r.Max.X) > 1 {
				t.Errorf("non-final paragraph line at y=%d does not reach the right margin", y)
			}
			justified++
		} else {
			for j := 1; j < len(line); j++ {
				previous := line[j-1]
				advance := previous.Glyph.Advance + v.view.Font.Spacing
				if previous.Glyph == space {
					advance += previous.Glyph.Height / 2
				}
				if line[j].X != previous.X+advance {
					t.Errorf("final paragraph line at y=%d stretches a gap", y)
				}
			}
			paragraph++
			consumed = 0
		}
	}
	if paragraph != len(paragraphs) || consumed != 0 {
		t.Error("tip omitted installed paragraph glyphs")
	}
	if justified == 0 {
		t.Error("no installed tip line reaches both text margins")
	}
	wantPixels := map[image.Point]color.RGBA{}
	for i, c := range v.calls {
		if !bodyCalls[i] {
			continue
		}
		for n, pixel := range c.Glyph.Pixels {
			if !pixel.Painted {
				continue
			}
			at := image.Pt(c.X+n%c.Glyph.Width, c.Y+n/c.Glyph.Width)
			if at.In(v.with.Bounds()) {
				wantPixels[at] = c.CellColor(pixel.Level)
			}
		}
	}
	for at, want := range wantPixels {
		if got := v.with.RGBAAt(at.X, at.Y); got != want {
			t.Fatalf("tip glyph or shadow at %v = %v, want %v", at, got, want)
		}
	}
	// TOWN-516's heights; a text the engine's wrap cannot fit grows the
	// panel to the tile-aligned height (DIV-2719).
	claimed := map[string]int{"town-square": 200, "tavern": 200, "school": 200, "shop": 136}[v.name]
	minHeight := ys[len(ys)-1] - v.view.Rect.Min.Y + v.view.Font.Height() + 1 + 36
	wantHeight := claimed
	if minHeight > claimed {
		wantHeight = max(104, 72+((max(0, minHeight-72)+31)/32)*32)
	}
	if v.view.Rect.Dy() != wantHeight {
		t.Errorf("panel height = %d, want %d for %d lines (claimed %d)", v.view.Rect.Dy(), wantHeight, len(lines), claimed)
	}
}

func absTipInt(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func assertReleaseTipControls(t *testing.T, v roomTipRaster) {
	t.Helper()
	ink := color.RGBA{185, 159, 73, 255}
	for _, control := range []struct {
		name, label string
		rect        image.Rectangle
	}{
		{"Close", v.view.CloseLabel, ui.TipPanelCloseRect(v.view.Rect)},
		{"toggle", v.view.ToggleLabel, ui.TipPanelToggleRect(v.view.Rect)},
	} {
		var faces []text.DrawCall
		for _, c := range v.calls {
			if c.Color == ink && !c.Flat && image.Pt(c.X, c.Y).In(control.rect) {
				faces = append(faces, c)
			}
		}
		if len(faces) != len(control.label) {
			t.Fatalf("%s caption draws %d glyphs, want %d installed bytes", control.name, len(faces), len(control.label))
		}
		visible := 0
		for i, c := range faces {
			if c.Glyph != v.view.Font.GlyphFor(control.label[i]) {
				t.Fatalf("%s caption glyph %d differs from the installed label", control.name, i)
			}
			for n, pixel := range c.Glyph.Pixels {
				if !pixel.Painted {
					continue
				}
				at := image.Pt(c.X+n%c.Glyph.Width, c.Y+n/c.Glyph.Width)
				if !at.In(control.rect) || !c.Clip.Empty() && !at.In(c.Clip) {
					t.Fatalf("%s caption clips a painted cell at %v", control.name, at)
				}
				if at.In(control.rect.Inset(2)) && v.with.RGBAAt(at.X, at.Y) == c.NativeColor(pixel.Level, n) {
					visible++
				}
			}
		}
		if visible == 0 {
			t.Fatalf("%s caption has no surviving installed glyph pixels", control.name)
		}
	}
	gem := v.view.Art.GemOff
	if v.view.ToggleOn {
		gem = v.view.Art.GemOn
	}
	if gem == nil || gem.Bounds().Size() != image.Pt(16, 16) {
		t.Fatal("tip toggle does not resolve its installed 16x16 square")
	}
	at := ui.TipPanelToggleRect(v.view.Rect).Min.Add(image.Pt(1, 0))
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			want := color.RGBAModel.Convert(gem.At(gem.Bounds().Min.X+x, gem.Bounds().Min.Y+y)).(color.RGBA)
			if want.A != 0 && v.with.RGBAAt(at.X+x, at.Y+y) != want {
				t.Fatalf("tip toggle sprite differs at local (%d,%d)", x, y)
			}
		}
	}
}

func writeReleaseTipRender(t *testing.T, v roomTipRaster) {
	t.Helper()
	dir := os.Getenv("AGAINROM_TIP_PANEL_RENDER_DIR")
	if dir == "" {
		return
	}
	dir = releaseTipResolvedOutput(t, dir)
	for _, name := range []string{"AGAINROM_ASSETS", "AGAINROM_ROOT"} {
		root := os.Getenv(name)
		if root == "" {
			continue
		}
		root = releaseTipResolvedOutput(t, root)
		rel, err := filepath.Rel(root, dir)
		if err != nil || rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))) {
			t.Fatal("tip render output must be outside the install")
		}
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	locale := "en"
	if v.view.Font.Selector == text.SelectorConverting {
		locale = "ru"
	}
	path := filepath.Join(dir, locale+"-"+v.name+".png")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	err = png.Encode(f, v.with)
	closeErr := f.Close()
	if err != nil {
		t.Fatal(err)
	}
	if closeErr != nil {
		t.Fatal(closeErr)
	}
	t.Logf("render %s", path)
}

func releaseTipResolvedOutput(t *testing.T, path string) string {
	t.Helper()
	path, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	var tail []string
	for {
		if _, err = os.Lstat(path); err == nil {
			break
		}
		if !os.IsNotExist(err) {
			t.Fatal(err)
		}
		tail = append(tail, filepath.Base(path))
		parent := filepath.Dir(path)
		if parent == path {
			t.Fatal("tip render output has no existing parent")
		}
		path = parent
	}
	path, err = filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatal(err)
	}
	for i := len(tail) - 1; i >= 0; i-- {
		path = filepath.Join(path, tail[i])
	}
	return path
}

func TestReleaseRoomTipStyleAndLayoutFromInstall(t *testing.T) {
	pieces := releaseTipFramePieces(t)
	for _, room := range []string{"town-square", "tavern", "school", "shop"} {
		t.Run(room, func(t *testing.T) {
			v := releaseRoomTipRaster(t, room)
			writeReleaseTipRender(t, v)
			assertReleaseTipFrame(t, v, pieces)
			assertReleaseTipText(t, v)
			assertReleaseTipControls(t, v)
		})
	}
}
