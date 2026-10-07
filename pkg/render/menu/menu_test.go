package menu_test

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"strings"
	"sync"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/render/menu"
)

// ---------------------------------------------------------------------------
// The contract, transcribed from docs/0010-main-menu/spec.md
// ---------------------------------------------------------------------------

const specFrameW, specFrameH = 640, 480

// The graphics/mainmenu/ subtree of main.res — spec "Asset set"
// (MENU-ASSET-001, MENU-ASSET-002), plan DD3 — addressed under main.res's
// own identity segment, which is what makes one string name the entry across
// an install's containers.
const specPrefix = "main/graphics/mainmenu/"

const (
	specBaseName = "menu_.bmp"
	specMaskName = "menumask.bmp"
)

// Hit mask -> button (MENU-MASK-003, MENU-MASK-004). Index 0x80 selects button 1
// through 0xf0 selecting button 8; every other value selects none.
var specHot = [8]byte{0x80, 0x90, 0xa0, 0xb0, 0xc0, 0xd0, 0xe0, 0xf0}

// The anti-aliased edge ramp the spec names explicitly as selecting nothing.
var specEdgeRamp = []byte{0x10, 0x12, 0x14, 0x16, 0x18, 0x1a, 0x1c, 0x1e}

// Assorted stray indices — near-misses around the hot values and the extremes.
var specStrays = []byte{0x01, 0x7f, 0x81, 0x8f, 0xff}

// Overlay placement, {x, y, w, h} in the frame, as half-open rectangles
// (MENU-GEOM-005, MENU-GEOM-006).
//
//	btn | hover (x, y, w, h)  | pressed (x, y, w, h)
//	  1 | 112,  64, 212, 136  | 116,  64, 208, 138
//	  2 |  84,  88, 236, 148  |  88,  88, 236, 152
//	  3 |  84, 236, 236, 152  |  88, 236, 236, 152
//	  4 | 112, 276, 212, 136  | 116, 272, 208, 140
//	  5 | 320,  60, 212, 140  | 320,  64, 212, 140
//	  6 | 324,  88, 236, 148  | 324,  88, 232, 152
//	  7 | 324, 236, 236, 152  | 324, 236, 236, 152
//	  8 | 324, 276, 208, 136  | 320, 272, 212, 140
var (
	specHover = [8]image.Rectangle{
		rect(112, 64, 212, 136),
		rect(84, 88, 236, 148),
		rect(84, 236, 236, 152),
		rect(112, 276, 212, 136),
		rect(320, 60, 212, 140),
		rect(324, 88, 236, 148),
		rect(324, 236, 236, 152),
		rect(324, 276, 208, 136),
	}
	specPressed = [8]image.Rectangle{
		rect(116, 64, 208, 138),
		rect(88, 88, 236, 152),
		rect(88, 236, 236, 152),
		rect(116, 272, 208, 140),
		rect(320, 64, 212, 140),
		rect(324, 88, 232, 152),
		rect(324, 236, 236, 152),
		rect(320, 272, 212, 140),
	}
)

// The columns are told apart by x: left column x < 320, right column x >= 320
// (spec FR-9a).
const specColumnSplit = 320

// EXIT is button 8 — the one per-button command MENU-STATE-007 decodes
// (button 8 -> WM_CLOSE), named by number rather than by corner.
const specExitButton = 8

// The normative NEW GAME rectangle: the hover entry 112, 64, 212, 136, i.e.
// the half-open frame area (112,64)...(324,200).
var specNewGameRect = image.Rect(112, 64, 324, 200)

// rect converts a {x, y, w, h} table row to a half-open rectangle.
func rect(x, y, w, h int) image.Rectangle { return image.Rect(x, y, x+w, y+h) }

// specEntries is the eighteen full archive paths in plan DD28's validation
// order: the base, the mask, then for i = 1...8 the hover overlay and then the
// pressed overlay.
var specEntries = buildSpecEntries()

func buildSpecEntries() []string {
	e := []string{specPrefix + specBaseName, specPrefix + specMaskName}
	for i := 1; i <= 8; i++ {
		e = append(e, fmt.Sprintf("%sbutton%d.bmp", specPrefix, i))
		e = append(e, fmt.Sprintf("%sbutton%dp.bmp", specPrefix, i))
	}
	return e
}

func hoverName(n int) string   { return fmt.Sprintf("button%d.bmp", n) }
func pressedName(n int) string { return fmt.Sprintf("button%dp.bmp", n) }

// ---------------------------------------------------------------------------
// An EntrySource over a map, and the shared valid asset set
// ---------------------------------------------------------------------------

// errNoEntry deliberately does NOT name the entry it was asked for: if a Load
// error message contains an entry name, that name came from the package under
// test and not from this stub. Otherwise the "names the offending entry"
// assertions would pass vacuously.
var errNoEntry = errors.New("synthetic source: entry absent")

type mapSource map[string][]byte

func (m mapSource) ReadFile(name string) ([]byte, error) {
	b, ok := m[name]
	if !ok {
		return nil, errNoEntry
	}
	return append([]byte(nil), b...), nil
}

// validSet is the one full-size asset set every case derives from; building the
// eighteen 640x480-scale bitmaps is the expensive part, so it happens once.
var validSet = sync.OnceValue(func() map[string][]byte {
	return synth.MenuFiles(synth.MenuOptions{
		Prefix:    specPrefix,
		Hover:     specHover,
		Pressed:   specPressed,
		MaskIndex: specHot,
	})
})

// filesWith copies the valid set and applies edits: a non-nil value replaces an
// entry, a nil value removes it.
func filesWith(edits map[string][]byte) mapSource {
	src := validSet()
	out := make(mapSource, len(src))
	for k, v := range src {
		out[k] = v
	}
	for k, v := range edits {
		if v == nil {
			delete(out, k)
			continue
		}
		out[k] = v
	}
	return out
}

func mustLoad(t *testing.T, files mapSource) *menu.Assets {
	t.Helper()
	a, err := menu.Load(files)
	if err != nil {
		t.Fatalf("Load of a complete, consistent set failed: %v", err)
	}
	if a == nil {
		t.Fatal("Load returned a nil *Assets and a nil error")
	}
	return a
}

// blankMask returns a 640x480 mask of index 0 on the given palette.
func blankMask(pal color.Palette) *image.Paletted {
	return image.NewPaletted(image.Rect(0, 0, specFrameW, specFrameH), pal)
}

func setPix(m *image.Paletted, x, y int, v byte) {
	m.Pix[y*m.Stride+x] = v
}

type probe struct {
	p    image.Point
	idx  byte
	want int
	what string
}

// probeSet lays every index the spec names at a distinct frame position.
func probeSet() []probe {
	var out []probe
	add := func(idx byte, want int, what string) {
		k := len(out)
		out = append(out, probe{
			p:    image.Pt(12+k*21, 33+k*13),
			idx:  idx,
			want: want,
			what: what,
		})
	}
	for i, idx := range specHot {
		add(idx, i+1, fmt.Sprintf("hot index %#02x", idx))
	}
	add(0x00, 0, "index 0 (background)")
	for _, idx := range specEdgeRamp {
		add(idx, 0, fmt.Sprintf("edge-ramp index %#02x", idx))
	}
	for _, idx := range specStrays {
		add(idx, 0, fmt.Sprintf("stray index %#02x", idx))
	}
	return out
}

func TestButtonAt(t *testing.T) {
	probes := probeSet()
	for i := range probes {
		p := probes[i].p
		if p.X < 0 || p.X >= specFrameW || p.Y < 0 || p.Y >= specFrameH {
			t.Fatalf("probe %d at %v is outside the frame; the probe layout is wrong", i, p)
		}
	}

	build := func(pal color.Palette) mapSource {
		m := blankMask(pal)
		for _, pr := range probes {
			setPix(m, pr.p.X, pr.p.Y, pr.idx)
		}
		return filesWith(map[string][]byte{specPrefix + specMaskName: synth.BMP8(m)})
	}

	gray := mustLoad(t, build(synth.GrayRamp()))

	t.Run("indices", func(t *testing.T) {
		for _, pr := range probes {
			if got := gray.ButtonAt(pr.p); got != pr.want {
				t.Errorf("ButtonAt(%v) with %s = %d, want %d", pr.p, pr.what, got, pr.want)
			}
		}
		// The hot indices in order: 0x80 -> 1 ... 0xf0 -> 8.
		for i, pr := range probes[:8] {
			if got := gray.ButtonAt(pr.p); got != i+1 {
				t.Errorf("hot index %#02x selects button %d, want %d", pr.idx, got, i+1)
			}
		}
		// An untouched pixel is index 0 and selects nothing.
		for _, p := range []image.Point{image.Pt(600, 470), image.Pt(0, 0), image.Pt(639, 479)} {
			if got := gray.ButtonAt(p); got != 0 {
				t.Errorf("ButtonAt(%v) on untouched background = %d, want 0", p, got)
			}
		}
	})

	t.Run("outside the frame", func(t *testing.T) {
		outside := []image.Point{
			image.Pt(-1, 100), image.Pt(100, -1), image.Pt(-1, -1),
			image.Pt(specFrameW, 100), image.Pt(100, specFrameH),
			image.Pt(specFrameW, specFrameH),
			image.Pt(-5000, -5000), image.Pt(5000, 5000),
			image.Pt(-1, 0), image.Pt(0, -1),
		}
		for _, p := range outside {
			if got := gray.ButtonAt(p); got != 0 {
				t.Errorf("ButtonAt(%v) outside the frame = %d, want 0", p, got)
			}
		}
	})

	// The mask's palette is an identity grayscale ramp and carries no meaning of
	// its own (spec Constraints, MENU-MASK-003): the same Pix on an all-black
	// palette must give the same answers. A decoder resolving indices through
	// the palette cannot satisfy both.
	t.Run("palette independence", func(t *testing.T) {
		black := mustLoad(t, build(synth.BlackPalette()))
		for _, pr := range probes {
			g, b := gray.ButtonAt(pr.p), black.ButtonAt(pr.p)
			if g != b {
				t.Errorf("ButtonAt(%v) with %s: gray ramp = %d, black palette = %d", pr.p, pr.what, g, b)
			}
			if b != pr.want {
				t.Errorf("ButtonAt(%v) with %s on a black palette = %d, want %d", pr.p, pr.what, b, pr.want)
			}
		}
		for _, p := range []image.Point{image.Pt(600, 470), image.Pt(-1, -1), image.Pt(640, 480)} {
			if g, b := gray.ButtonAt(p), black.ButtonAt(p); g != b {
				t.Errorf("ButtonAt(%v): gray ramp = %d, black palette = %d", p, g, b)
			}
		}
	})

	// Row-order sensitivity (plan DD7, R-1): the research labels the mask's
	// stored row order as *inferred*. A vertically mirrored mask must change
	// which button a fixed point selects, otherwise the manual game-side check
	// (AC-11) could not falsify that inference.
	t.Run("row order sensitivity", func(t *testing.T) {
		const probeX, probeY = 200, 100
		const bandH = 20
		mirrorY := specFrameH - 1 - probeY // 379

		fill := func(m *image.Paletted, y0 int, v byte) {
			for y := y0; y < y0+bandH; y++ {
				for x := 180; x < 220; x++ {
					setPix(m, x, y, v)
				}
			}
		}
		upright := blankMask(synth.GrayRamp())
		fill(upright, probeY-bandH/2, specHot[0])    // 0x80 -> button 1, over the probe
		fill(upright, mirrorY-bandH/2+1, specHot[3]) // 0xb0 -> button 4, over its mirror
		mirrored := blankMask(synth.GrayRamp())
		for y := 0; y < specFrameH; y++ {
			copy(mirrored.Pix[y*mirrored.Stride:y*mirrored.Stride+specFrameW],
				upright.Pix[(specFrameH-1-y)*upright.Stride:(specFrameH-1-y)*upright.Stride+specFrameW])
		}

		up := mustLoad(t, filesWith(map[string][]byte{specPrefix + specMaskName: synth.BMP8(upright)}))
		mi := mustLoad(t, filesWith(map[string][]byte{specPrefix + specMaskName: synth.BMP8(mirrored)}))

		p := image.Pt(probeX, probeY)
		a, b := up.ButtonAt(p), mi.ButtonAt(p)
		// Non-vacuity: both readings must name a real button, or the witness
		// proves nothing about row order.
		if a == 0 || b == 0 {
			t.Fatalf("VACUOUS row-order witness: ButtonAt(%v) = %d upright, %d mirrored; "+
				"both must be non-zero for the mirror to witness anything", p, a, b)
		}
		if a == b {
			t.Errorf("a vertically mirrored mask selects the same button at %v (%d): "+
				"the decode is not sensitive to stored row order, so AC-11 cannot falsify "+
				"the research's inferred row order", p, a)
		}
		if a != 1 || b != 4 {
			t.Errorf("upright/mirrored at %v = %d/%d, want 1/4 from the placed bands", p, a, b)
		}
	})
}

func TestNewGameButton(t *testing.T) {
	if menu.FrameW != specFrameW || menu.FrameH != specFrameH {
		t.Errorf("FrameW, FrameH = %d, %d, want %d, %d", menu.FrameW, menu.FrameH, specFrameW, specFrameH)
	}
	if menu.ButtonCount != 8 {
		t.Errorf("ButtonCount = %d, want 8", menu.ButtonCount)
	}
	if menu.ColumnSplit != specColumnSplit {
		t.Errorf("ColumnSplit = %d, want %d", menu.ColumnSplit, specColumnSplit)
	}

	ng := menu.NewGameButton
	if ng < 1 || ng > 8 {
		t.Fatalf("NewGameButton = %d, want a 1-based button in [1,8]", ng)
	}

	// The normative rectangle: hover entry 112, 64, 212, 136.
	if got := menu.HoverRects[ng-1]; got != specNewGameRect {
		t.Errorf("HoverRects[NewGameButton-1] = %v, want %v", got, specNewGameRect)
	}

	leftTop, leftBottom := -1, -1
	for i, r := range menu.HoverRects {
		if r.Min.X >= specColumnSplit {
			continue
		}
		if leftTop < 0 || r.Min.Y < menu.HoverRects[leftTop].Min.Y {
			leftTop = i
		}
		if leftBottom < 0 || r.Min.Y > menu.HoverRects[leftBottom].Min.Y {
			leftBottom = i
		}
	}
	if leftTop < 0 || leftBottom < 0 {
		t.Fatalf("no left-column entry found in HoverRects = %v", menu.HoverRects)
	}
	if menu.HoverRects[ng-1].Min.X >= specColumnSplit {
		t.Errorf("NewGameButton %d starts at x = %d, not in the left column (x < %d)",
			ng, menu.HoverRects[ng-1].Min.X, specColumnSplit)
	}
	if ng != leftTop+1 {
		t.Errorf("NewGameButton = %d, want %d — the smallest-Min.Y left-column entry", ng, leftTop+1)
	}
	if ng == leftBottom+1 {
		t.Errorf("NewGameButton = %d is the BOTTOM-left entry (Min.Y = %d); "+
			"a flipped row order has silently made the wrong button NEW GAME",
			ng, menu.HoverRects[leftBottom].Min.Y)
	}

	// The trap the spec's Constraints section names: bare "topmost" does not
	// identify it, because the right column's top entry begins four pixels
	// higher. Computed here as its own rule, so the warning is a regression test.
	globalTop := 0
	for i, r := range menu.HoverRects {
		if r.Min.Y < menu.HoverRects[globalTop].Min.Y {
			globalTop = i
		}
	}
	if globalTop+1 == ng {
		t.Errorf("the smallest-Min.Y-over-the-whole-table rule picks button %d, the same as "+
			"NewGameButton; the spec requires them to differ (the right column's top entry "+
			"starts at y = 60, four pixels above the left column's y = 64)", globalTop+1)
	}
	if globalTop+1 != 5 {
		t.Errorf("the smallest-Min.Y-over-the-whole-table rule picks button %d, want 5 "+
			"(hover entry 320, 60, 212, 140)", globalTop+1)
	}
	if d := menu.HoverRects[ng-1].Min.Y - menu.HoverRects[globalTop].Min.Y; d != 4 {
		t.Errorf("NEW GAME starts %d px below the globally topmost entry, want 4", d)
	}

	// Both placement tables, entry by entry.
	for i := 0; i < 8; i++ {
		if got := menu.HoverRects[i]; got != specHover[i] {
			t.Errorf("HoverRects[%d] = %v, want %v (button %d)", i, got, specHover[i], i+1)
		}
		if got := menu.PressedRects[i]; got != specPressed[i] {
			t.Errorf("PressedRects[%d] = %v, want %v (button %d)", i, got, specPressed[i], i+1)
		}
	}

	t.Run("EXIT is button 8, and its rectangle is the bottom-right one", func(t *testing.T) {
		if menu.ExitButton != specExitButton {
			t.Fatalf("ExitButton = %d, want %d — MENU-STATE-007 binds WM_CLOSE to button 8",
				menu.ExitButton, specExitButton)
		}
		if menu.ExitButton < 1 || menu.ExitButton > menu.ButtonCount {
			t.Fatalf("ExitButton = %d, outside [1, %d]", menu.ExitButton, menu.ButtonCount)
		}
		if menu.ExitButton == menu.NewGameButton {
			t.Fatalf("ExitButton and NewGameButton are both %d; one brooch button cannot be both",
				menu.ExitButton)
		}

		got := menu.HoverRects[menu.ExitButton-1]
		if want := specHover[specExitButton-1]; got != want {
			t.Errorf("HoverRects[ExitButton-1] = %v, want %v", got, want)
		}

		// The bottom entry of the RIGHT column, computed here as its own rule.
		rightBottom := -1
		for i, r := range menu.HoverRects {
			if r.Min.X < specColumnSplit {
				continue
			}
			if rightBottom < 0 || r.Min.Y > menu.HoverRects[rightBottom].Min.Y {
				rightBottom = i
			}
		}
		if rightBottom < 0 {
			t.Fatalf("no right-column entry found in HoverRects = %v", menu.HoverRects)
		}
		if menu.ExitButton != rightBottom+1 {
			t.Errorf("ExitButton = %d, but the bottom entry of the right column is button %d "+
				"(%v); AC-14 asks a human to press the bottom-right gem, so the two must agree",
				menu.ExitButton, rightBottom+1, menu.HoverRects[rightBottom])
		}
	})

	t.Run("entries", func(t *testing.T) {
		if menu.EntryPrefix != specPrefix {
			t.Errorf("EntryPrefix = %q, want %q", menu.EntryPrefix, specPrefix)
		}
		if menu.BaseEntry != specBaseName {
			t.Errorf("BaseEntry = %q, want %q", menu.BaseEntry, specBaseName)
		}
		if menu.MaskEntry != specMaskName {
			t.Errorf("MaskEntry = %q, want %q", menu.MaskEntry, specMaskName)
		}

		// HoverEntry/PressedEntry: the spec fixes the *name*; whether the
		// accessor carries the prefix is not stated (BaseEntry/MaskEntry are
		// bare, Entries() is documented as full paths), so both forms pass.
		for n := 1; n <= 8; n++ {
			hv, pr := menu.HoverEntry(n), menu.PressedEntry(n)
			if hv != hoverName(n) && hv != specPrefix+hoverName(n) {
				t.Errorf("HoverEntry(%d) = %q, want %q or %q", n, hv, hoverName(n), specPrefix+hoverName(n))
			}
			if pr != pressedName(n) && pr != specPrefix+pressedName(n) {
				t.Errorf("PressedEntry(%d) = %q, want %q or %q", n, pr, pressedName(n), specPrefix+pressedName(n))
			}
		}

		got := menu.Entries()
		if len(got) != 18 {
			t.Fatalf("Entries() returned %d paths, want 18", len(got))
		}
		seen := make(map[string]bool, len(got))
		for _, e := range got {
			if seen[e] {
				t.Errorf("Entries() repeats %q", e)
			}
			seen[e] = true
			if !strings.HasPrefix(e, specPrefix) {
				t.Errorf("Entries() holds %q, which lacks the prefix %q", e, specPrefix)
			}
		}
		for _, want := range specEntries {
			if !seen[want] {
				t.Errorf("Entries() is missing %q", want)
			}
		}
		// plan DD28: base, mask, then per button the hover then the pressed
		// overlay — the order Load validates in.
		for i, want := range specEntries {
			if i < len(got) && got[i] != want {
				t.Errorf("Entries()[%d] = %q, want %q (DD28 validation order)", i, got[i], want)
			}
		}
	})
}

func TestLoadRejects(t *testing.T) {
	t.Run("complete set loads", func(t *testing.T) {
		a := mustLoad(t, filesWith(nil))
		if a.Base == nil {
			t.Fatal("Base is nil after a successful Load")
		}
		if w, h := a.Base.Bounds().Dx(), a.Base.Bounds().Dy(); w != specFrameW || h != specFrameH {
			t.Errorf("Base is %dx%d, want %dx%d", w, h, specFrameW, specFrameH)
		}
		if a.Mask == nil {
			t.Fatal("Mask is nil after a successful Load")
		}
		if w, h := a.Mask.Bounds().Dx(), a.Mask.Bounds().Dy(); w != specFrameW || h != specFrameH {
			t.Errorf("Mask is %dx%d, want %dx%d", w, h, specFrameW, specFrameH)
		}
		for i := 0; i < 8; i++ {
			if a.Hover[i] == nil {
				t.Errorf("Hover[%d] is nil", i)
			} else if w, h := a.Hover[i].Bounds().Dx(), a.Hover[i].Bounds().Dy(); w != specHover[i].Dx() || h != specHover[i].Dy() {
				t.Errorf("Hover[%d] is %dx%d, want %dx%d", i, w, h, specHover[i].Dx(), specHover[i].Dy())
			}
			if a.Pressed[i] == nil {
				t.Errorf("Pressed[%d] is nil", i)
			} else if w, h := a.Pressed[i].Bounds().Dx(), a.Pressed[i].Bounds().Dy(); w != specPressed[i].Dx() || h != specPressed[i].Dy() {
				t.Errorf("Pressed[%d] is %dx%d, want %dx%d", i, w, h, specPressed[i].Dx(), specPressed[i].Dy())
			}
		}
	})

	// Guards: the two dimension-mismatch cases below borrow another button's row,
	// which only proves anything if the rows genuinely differ.
	if specHover[1].Size() == specHover[2].Size() {
		t.Fatalf("VACUOUS: hover rows 2 and 3 are the same size %v", specHover[1].Size())
	}
	if specPressed[4].Size() == specPressed[5].Size() {
		t.Fatalf("VACUOUS: pressed rows 5 and 6 are the same size %v", specPressed[4].Size())
	}

	rgba := func(w, h int) []byte { return synth.BMP24(image.NewRGBA(image.Rect(0, 0, w, h))) }
	paletted := func(w, h int) []byte {
		return synth.BMP8(image.NewPaletted(image.Rect(0, 0, w, h), synth.GrayRamp()))
	}

	cases := []struct {
		name    string
		edits   map[string][]byte
		named   string // must appear in the error message
		unnamed string // must NOT appear (first-failure order)
	}{
		{
			name:  "missing base",
			edits: map[string][]byte{specPrefix + specBaseName: nil},
			named: specBaseName,
		},
		{
			name:  "missing mask",
			edits: map[string][]byte{specPrefix + specMaskName: nil},
			named: specMaskName,
		},
		{
			name:  "missing hover overlay",
			edits: map[string][]byte{specPrefix + hoverName(5): nil},
			named: hoverName(5),
		},
		{
			name:  "missing pressed overlay",
			edits: map[string][]byte{specPrefix + pressedName(6): nil},
			named: pressedName(6),
		},
		{
			name:  "base not 640x480",
			edits: map[string][]byte{specPrefix + specBaseName: rgba(320, 240)},
			named: specBaseName,
		},
		{
			name:  "mask not 640x480",
			edits: map[string][]byte{specPrefix + specMaskName: paletted(specFrameW, specFrameH-1)},
			named: specMaskName,
		},
		{
			// Button 2's hover overlay given button 3's hover dimensions
			// (236x152 instead of 236x148): each overlay is checked against its
			// OWN placement row.
			name:  "hover overlay disagrees with its own row",
			edits: map[string][]byte{specPrefix + hoverName(2): rgba(specHover[2].Dx(), specHover[2].Dy())},
			named: hoverName(2),
		},
		{
			// Button 5's pressed overlay given button 6's pressed dimensions
			// (232x152 instead of 212x140).
			name:  "pressed overlay disagrees with its own row",
			edits: map[string][]byte{specPrefix + pressedName(5): rgba(specPressed[5].Dx(), specPressed[5].Dy())},
			named: pressedName(5),
		},
		{
			name:  "entry is not a valid BMP",
			edits: map[string][]byte{specPrefix + hoverName(4): []byte("this is not a BMP stream at all")},
			named: hoverName(4),
		},
		{
			// DD28: the base is validated before the mask.
			name: "two defects: missing base and a wrong-sized mask",
			edits: map[string][]byte{
				specPrefix + specBaseName: nil,
				specPrefix + specMaskName: paletted(320, 240),
			},
			named:   specBaseName,
			unnamed: specMaskName,
		},
		{
			// DD28: buttons are validated in ascending order.
			name: "two defective hover overlays on different buttons",
			edits: map[string][]byte{
				specPrefix + hoverName(3): nil,
				specPrefix + hoverName(6): nil,
			},
			named:   hoverName(3),
			unnamed: hoverName(6),
		},
		{
			// DD28: within a button, the hover overlay precedes the pressed one,
			// and button 2 precedes button 5 — so button 2's PRESSED defect is
			// reported before button 5's hover defect.
			name: "two defects: button 2 pressed and button 5 hover",
			edits: map[string][]byte{
				specPrefix + pressedName(2): nil,
				specPrefix + hoverName(5):   nil,
			},
			named:   pressedName(2),
			unnamed: hoverName(5),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a, err := menu.Load(filesWith(tc.edits))
			if err == nil {
				t.Fatalf("Load succeeded on a defective set (%s); want a non-nil error", tc.name)
			}
			if a != nil {
				t.Errorf("Load returned a non-nil *Assets (%p) alongside its error; "+
					"P-4 forbids any partly-defined asset set", a)
			}
			msg := err.Error()
			if !strings.Contains(msg, tc.named) {
				t.Errorf("error %q does not name the offending entry %q", msg, tc.named)
			}
			if tc.unnamed != "" && strings.Contains(msg, tc.unnamed) {
				t.Errorf("error %q names %q; the first failure in Entries() order is %q",
					msg, tc.unnamed, tc.named)
			}
		})
	}
}

// regionRects lays eight disjoint, differently sized regions in the frame, one
// per button, so each button's expected pixel count is its own rectangle's area.
func regionRects() [8]image.Rectangle {
	var r [8]image.Rectangle
	for i := 0; i < 8; i++ {
		x0, y0 := 20, 10+i*40
		r[i] = image.Rect(x0, y0, x0+11+i, y0+20)
	}
	return r
}

// maskFromRegions paints each present button's hot index over its rectangle.
func maskFromRegions(r [8]image.Rectangle, present [8]bool) *image.Paletted {
	m := blankMask(synth.GrayRamp())
	for i := 0; i < 8; i++ {
		if !present[i] {
			continue
		}
		for y := r[i].Min.Y; y < r[i].Max.Y; y++ {
			for x := r[i].Min.X; x < r[i].Max.X; x++ {
				setPix(m, x, y, specHot[i])
			}
		}
	}
	return m
}

func loadWithRegions(t *testing.T, r [8]image.Rectangle, present [8]bool) *menu.Assets {
	t.Helper()
	m := maskFromRegions(r, present)
	return mustLoad(t, filesWith(map[string][]byte{specPrefix + specMaskName: synth.BMP8(m)}))
}

func TestMaskRegions(t *testing.T) {
	rects := regionRects()
	all := [8]bool{true, true, true, true, true, true, true, true}

	// Sanity: the regions must be disjoint and inside the frame, or the counts
	// below are not the areas.
	frame := image.Rect(0, 0, specFrameW, specFrameH)
	for i := 0; i < 8; i++ {
		if !rects[i].In(frame) {
			t.Fatalf("region %d %v is not inside the frame", i+1, rects[i])
		}
		for j := i + 1; j < 8; j++ {
			if rects[i].Overlaps(rects[j]) {
				t.Fatalf("regions %d and %d overlap: %v, %v", i+1, j+1, rects[i], rects[j])
			}
		}
	}

	var want [8]int
	for i := 0; i < 8; i++ {
		want[i] = rects[i].Dx() * rects[i].Dy()
	}

	t.Run("all regions present", func(t *testing.T) {
		got := loadWithRegions(t, rects, all).MaskRegions()
		for i := 0; i < 8; i++ {
			if got[i] <= 0 {
				t.Errorf("MaskRegions()[%d] = %d, want > 0 (button %d has %d pixels of index %#02x)",
					i, got[i], i+1, want[i], specHot[i])
			}
			if got[i] != want[i] {
				t.Errorf("MaskRegions()[%d] = %d, want %d", i, got[i], want[i])
			}
		}
	})

	t.Run("one region omitted", func(t *testing.T) {
		const omit = 5 // 1-based
		present := all
		present[omit-1] = false
		got := loadWithRegions(t, rects, present).MaskRegions()
		if got[omit-1] != 0 {
			t.Errorf("MaskRegions()[%d] = %d with button %d's region omitted, want 0",
				omit-1, got[omit-1], omit)
		}
		for i := 0; i < 8; i++ {
			if i == omit-1 {
				continue
			}
			if got[i] != want[i] {
				t.Errorf("MaskRegions()[%d] = %d with button %d omitted, want %d (unchanged)",
					i, got[i], omit, want[i])
			}
		}
	})

	// DD29: "non-empty" is at least one pixel — nothing stronger, no bounding
	// box and no contiguity requirement.
	t.Run("a single pixel is a region", func(t *testing.T) {
		one := rects
		one[2] = image.Rect(500, 400, 501, 401) // button 3, exactly one pixel
		got := loadWithRegions(t, one, all).MaskRegions()
		if got[2] != 1 {
			t.Errorf("MaskRegions()[2] = %d for a single pixel of index %#02x, want 1", got[2], specHot[2])
		}
		for i := 0; i < 8; i++ {
			if i == 2 {
				continue
			}
			if got[i] != want[i] {
				t.Errorf("MaskRegions()[%d] = %d, want %d (unchanged)", i, got[i], want[i])
			}
		}
	})
}

// TestLoadGameIsTheTopRightButtonAndNothingElse pins the third bound button.
func TestLoadGameIsTheTopRightButtonAndNothingElse(t *testing.T) {
	n := menu.LoadGameButton
	if n < 1 || n > menu.ButtonCount {
		t.Fatalf("LoadGameButton = %d, outside [1, %d]", n, menu.ButtonCount)
	}

	r := menu.HoverRects[n-1]
	if r.Min.X < menu.ColumnSplit {
		t.Errorf("LoadGameButton %d starts at x=%d, left of the column split %d — "+
			"it must be in the RIGHT column", n, r.Min.X, menu.ColumnSplit)
	}
	for i, other := range menu.HoverRects {
		if i == n-1 || other.Min.X < menu.ColumnSplit {
			continue
		}
		if other.Min.Y < r.Min.Y {
			t.Errorf("button %d starts at y=%d, above LoadGameButton %d at y=%d — "+
				"LOAD GAME must be the TOP of the right column", i+1, other.Min.Y, n, r.Min.Y)
		}
	}

	// Three bound buttons, three distinct gems. A collision here would give one
	// rectangle two actions and silently drop one of them.
	if n == menu.NewGameButton {
		t.Fatalf("LoadGameButton and NewGameButton are both %d", n)
	}
	if n == menu.ExitButton {
		t.Fatalf("LoadGameButton and ExitButton are both %d — pressing LOAD would quit the game", n)
	}
}
