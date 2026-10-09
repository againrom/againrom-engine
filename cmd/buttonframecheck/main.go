// Command buttonframecheck measures the shipped button frames against a
// lawful game install.
//
//	buttonframecheck [-assets DIR]
//
// The school and the tavern each ship a 160x238 area picture with the button
// wells baked into it and one 140x46 bitmap per button per state (off, on).
// This tool correlates each shipped button bitmap over its own room's area
// picture — the same method cmd/schoolcheck uses for the training column's
// skill patches — and the school compares that winner with production. The
// tavern reports its original three art winners and executable RECTs as
// provenance. DIV-483 makes production reuse the shop's complete four-button
// composition, so the shop correlation also verifies the tavern's panel
// rectangles and driveRooms checks their input.
package main

import (
	"flag"
	"fmt"
	"image"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"

	"againrom/pkg/formats/bmp"
	"againrom/pkg/game"
	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

const matchTolerance = 12
const separation = 4

type pic struct {
	img   *image.RGBA
	black int
}

func (p pic) w() int { return p.img.Bounds().Dx() }
func (p pic) h() int { return p.img.Bounds().Dy() }

func readPic(src terrain.EntrySource, addr string) (pic, error) {
	raw, err := src.ReadFile(addr)
	if err != nil {
		return pic{}, fmt.Errorf("%s: %w", addr, err)
	}
	im, err := bmp.Decode(raw)
	if err != nil {
		return pic{}, fmt.Errorf("%s: %w", addr, err)
	}
	out := pic{img: im.RGBA()}
	for _, c := range im.Pix {
		if c == (bmp.Color{}) {
			out.black++
		}
	}
	return out, nil
}

func diff(a, b uint8) int {
	if a > b {
		return int(a) - int(b)
	}
	return int(b) - int(a)
}

func score(dst, patch *image.RGBA, ox, oy int) (matched, total int) {
	pw, ph := patch.Bounds().Dx(), patch.Bounds().Dy()
	for y := 0; y < ph; y++ {
		po := y * patch.Stride
		do := (oy+y)*dst.Stride + ox*4
		for x := 0; x < pw; x++ {
			pr, pg, pb := patch.Pix[po], patch.Pix[po+1], patch.Pix[po+2]
			po += 4
			if pr == 0 && pg == 0 && pb == 0 {
				do += 4
				continue
			}
			total++
			d := diff(pr, dst.Pix[do]) + diff(pg, dst.Pix[do+1]) + diff(pb, dst.Pix[do+2])
			do += 4
			if d <= matchTolerance {
				matched++
			}
		}
	}
	return matched, total
}

type correlation struct {
	at       image.Point
	fraction float64
	next     float64
	nextAt   image.Point
	total    int
}

func correlate(dst, patch *image.RGBA) correlation {
	maxX := dst.Bounds().Dx() - patch.Bounds().Dx()
	maxY := dst.Bounds().Dy() - patch.Bounds().Dy()
	if maxX < 0 || maxY < 0 {
		return correlation{}
	}
	frac := make([][]float64, maxY+1)
	_, total := score(dst, patch, 0, 0)
	var wg sync.WaitGroup
	sem := make(chan struct{}, runtime.NumCPU())
	for y := 0; y <= maxY; y++ {
		wg.Add(1)
		sem <- struct{}{}
		go func(y int) {
			defer wg.Done()
			defer func() { <-sem }()
			r := make([]float64, maxX+1)
			for x := 0; x <= maxX; x++ {
				m, t := score(dst, patch, x, y)
				if t > 0 {
					r[x] = float64(m) / float64(t)
				}
			}
			frac[y] = r
		}(y)
	}
	wg.Wait()
	out := correlation{fraction: -1, next: -1, total: total}
	for y, r := range frac {
		for x, f := range r {
			if f > out.fraction {
				out.fraction, out.at = f, image.Pt(x, y)
			}
		}
	}
	for y, r := range frac {
		for x, f := range r {
			if abs(x-out.at.X) <= separation && abs(y-out.at.Y) <= separation {
				continue
			}
			if f > out.next {
				out.next, out.nextAt = f, image.Pt(x, y)
			}
		}
	}
	return out
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "buttonframecheck:", err)
		os.Exit(1)
	}
}

func run(args []string, w io.Writer) error {
	fs := flag.NewFlagSet("buttonframecheck", flag.ContinueOnError)
	assets := fs.String("assets", "", "lawful game install root (or AGAINROM_ASSETS)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	root := game.ResolveAssetRoot(*assets, os.Getenv("AGAINROM_ASSETS"))
	if root == "" {
		return fmt.Errorf("no asset root: pass -assets or set AGAINROM_ASSETS")
	}
	arc, err := game.OpenArchives(root)
	if err != nil {
		return err
	}
	src := arc.Containers
	fmt.Fprintf(w, "root %s\n", root)
	fail := false

	if bad, err := checkRoom(w, src, "school", ui.TownSurfaceSchool, "graphics/interface/training/buttonsarea.bmp",
		[]string{"graphics/interface/training/buttons/b1", "graphics/interface/training/buttons/b2"}, nil, true); err != nil {
		return err
	} else if bad {
		fail = true
	}
	if bad, err := checkRoom(w, src, "tavern", ui.TownSurfaceTavern, "graphics/interface/inn/buttonsarea.bmp",
		[]string{"graphics/interface/inn/button1", "graphics/interface/inn/button2", "graphics/interface/inn/button3"},
		[]image.Rectangle{
			image.Rect(4, 44, 144, 90),
			image.Rect(4, 91, 144, 137),
			image.Rect(4, 138, 144, 184),
		}, false); err != nil {
		return err
	} else if bad {
		fail = true
	}
	if bad, err := checkShop(w, src); err != nil {
		return err
	} else if bad {
		fail = true
	}
	if err := checkChargen(w, src); err != nil {
		return err
	}
	if bad, err := driveRooms(root, w); err != nil {
		return err
	} else if bad {
		fail = true
	}

	if fail {
		return fmt.Errorf("measured geometry disagrees with production; see the FAIL lines above")
	}
	fmt.Fprintln(w, "buttonframecheck: ok")
	return nil
}

// checkRoom correlates each button's off/on pair against the room's own area
// picture. If decoded is present it remains the original executable-backed
// answer. compareProduction decides whether that original/measured layout is
// also normative for the current production room.
func checkRoom(w io.Writer, src terrain.EntrySource, name string, kind ui.TownSurfaceKind, areaAddr string, buttonPrefixes []string, decoded []image.Rectangle, compareProduction bool) (bool, error) {
	if decoded != nil && len(decoded) != len(buttonPrefixes) {
		return false, fmt.Errorf("%s: %d decoded rectangles for %d buttons", name, len(decoded), len(buttonPrefixes))
	}
	area, err := readPic(src, areaAddr)
	if err != nil {
		return false, err
	}
	fmt.Fprintf(w, "%s area %s %dx%d\n", name, areaAddr, area.w(), area.h())
	fail := false
	for i, prefix := range buttonPrefixes {
		var at image.Point
		for state, suffix := range []string{"off.bmp", "on.bmp"} {
			addr := prefix + suffix
			p, err := readPic(src, addr)
			if err != nil {
				return false, err
			}
			c := correlate(area.img, p.img)
			ratio := "inf"
			if c.next > 0 {
				ratio = fmt.Sprintf("%.1fx", c.fraction/c.next)
			}
			fmt.Fprintf(w, "%s button %d %s %dx%d at %v frac %.4f next %.4f at %v ratio %s black %d/%d\n",
				name, i, addr, p.w(), p.h(), c.at, c.fraction, c.next, c.nextAt, ratio, p.black, p.w()*p.h())
			if state == 0 {
				at = c.at
			} else if c.at != at {
				fmt.Fprintf(w, "FAIL %s button %d: off lands at %v, on at %v\n", name, i, at, c.at)
				fail = true
			}
		}
		art := image.Rect(at.X, at.Y, at.X+140, at.Y+46)
		want := art
		if decoded != nil {
			want = decoded[i]
		}
		if !compareProduction {
			fmt.Fprintf(w, "%s original button %d well art %v decoded %v (production intentionally differs; DIV-483)\n",
				name, i, art, want)
			continue
		}
		production := ui.TownSurfaceButtonWell(kind, i)
		mark := "ok"
		if production != want {
			mark, fail = "FAIL", true
		}
		if decoded != nil {
			fmt.Fprintf(w, "%s button %d well art %v decoded %v production %v %s\n", name, i, art, want, production, mark)
		} else {
			fmt.Fprintf(w, "%s button %d well art %v production %v %s\n", name, i, art, production, mark)
		}
	}
	return fail, nil
}

func checkShop(w io.Writer, src terrain.EntrySource) (bool, error) {
	menu, err := readPic(src, "graphics/interface/shopmenu.bmp")
	if err != nil {
		return false, err
	}
	fmt.Fprintf(w, "shop area graphics/interface/shopmenu.bmp %dx%d\n", menu.w(), menu.h())
	fail := false
	origin := ui.TownWideUpperRegion.Min
	for i := 1; i <= 4; i++ {
		addr := fmt.Sprintf("graphics/interface/shopbutton%d.bmp", i)
		p, err := readPic(src, addr)
		if err != nil {
			return false, err
		}
		c := correlate(menu.img, p.img)
		ratio := "inf"
		if c.next > 0 {
			ratio = fmt.Sprintf("%.1fx", c.fraction/c.next)
		}
		fmt.Fprintf(w, "shop button %d %s %dx%d at %v frac %.4f next %.4f at %v ratio %s black %d/%d\n",
			i-1, addr, p.w(), p.h(), c.at, c.fraction, c.next, c.nextAt, ratio, p.black, p.w()*p.h())
		want := ui.ShopButtonRect(i - 1)
		got := image.Rect(c.at.X, c.at.Y, c.at.X+p.w(), c.at.Y+p.h()).Add(origin)
		mark := "ok"
		if got != want {
			mark, fail = "FAIL", true
		}
		fmt.Fprintf(w, "shop button %d rect measured %v production %v %s\n", i-1, got, want, mark)
		tavern := ui.TownSurfaceButtonRect(ui.TownSurfaceTavern, i-1)
		tavernMark := "ok"
		if tavern != want {
			tavernMark, fail = "FAIL", true
		}
		fmt.Fprintf(w, "tavern button %d reuses shop rect %v production %v %s\n",
			i-1, want, tavern, tavernMark)
	}
	return fail, nil
}

// checkChargen segments interface/chrgen/buttonsarea.bmp's baked wells by
// brightness. There is no per-button bitmap to correlate, so this reports the
// segmentation rather than comparing it with a production rectangle; the
// production placement it backs (pkg/ui/chargen_page.go detailedControlRegion)
// is an authored reading of this report, recorded as such in DIV-156. The
// fourth (topmost) well has no production control rectangle at all: the
// picture is drawn unmodified over it and a click there does nothing.
//
// The threshold (40) is chosen from the picture, not from the three nav
// actions this build already had: at 33 the topmost well's run is 19 rows,
// one under the 20-row minimum, and segments() drops it, returning three
// wells that happen to match the existing action count — an agreement with
// the expected answer rather than evidence for it (round 1 read it as the
// latter). At 40 all four visually identical plaques resolve, identically on
// both preserved roots.
func checkChargen(w io.Writer, src terrain.EntrySource) error {
	addr := "graphics/interface/chrgen/buttonsarea.bmp"
	p, err := readPic(src, addr)
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "chargen area %s %dx%d\n", addr, p.w(), p.h())
	const borderAvg = 40 // a row/column whose mean channel sum exceeds this is gilt trim, not a well interior
	rows := rowBrightness(p.img)
	for _, seg := range segments(rows, borderAvg, 20) {
		cols := colBrightness(p.img, seg[0], seg[1])
		colSegs := segments(cols, borderAvg, 40)
		fmt.Fprintf(w, "chargen well y=[%d,%d) x-segments=%v\n", seg[0], seg[1], colSegs)
	}
	return nil
}

func rowBrightness(img *image.RGBA) []float64 {
	b := img.Bounds()
	out := make([]float64, b.Dy())
	for y := 0; y < b.Dy(); y++ {
		var sum int
		o := y * img.Stride
		for x := 0; x < b.Dx(); x++ {
			sum += int(img.Pix[o]) + int(img.Pix[o+1]) + int(img.Pix[o+2])
			o += 4
		}
		out[y] = float64(sum) / float64(b.Dx()*3)
	}
	return out
}

func colBrightness(img *image.RGBA, y0, y1 int) []float64 {
	b := img.Bounds()
	out := make([]float64, b.Dx())
	for x := 0; x < b.Dx(); x++ {
		var sum int
		for y := y0; y < y1; y++ {
			o := y*img.Stride + x*4
			sum += int(img.Pix[o]) + int(img.Pix[o+1]) + int(img.Pix[o+2])
		}
		out[x] = float64(sum) / float64((y1-y0)*3)
	}
	return out
}

// driveRooms opens a real campaign town through the production front end,
// walks into the school and the tavern, and hit-tests the centre of every
// drawn button well through ui.TownSurfaceControlAt. It prints which index
// each well answers, so the whole chain — art placement, well table and hit
// test — is measured together rather than each in isolation.
func driveRooms(root string, w io.Writer) (bool, error) {
	f, err := game.NewFrontEnd(root)
	if err != nil {
		return false, err
	}
	if f.Town == nil {
		return false, fmt.Errorf("drive: the front end started with no town")
	}
	f.Town.Arrive()
	bad := false
	for _, name := range []string{"SCHOOL", "TAVERN"} {
		screen := f.TownScreen()
		screen.Back() // the screen is remembered across rooms; start at the square
		entered := false
		for i, row := range screen.Rows() {
			if row.Choosable && strings.Contains(strings.ToUpper(row.Text), name) {
				screen.Choose(i)
				entered = true
				break
			}
		}
		if !entered {
			return bad, fmt.Errorf("drive: the town square offers no %s row", name)
		}
		surface, ok := screen.(ui.TownSurfaceScreen)
		if !ok || !surface.AtTownSurface() {
			return bad, fmt.Errorf("drive: the %s room did not open", name)
		}
		v := surface.TownSurface()
		fmt.Fprintf(w, "drive %s: %d buttons\n", name, len(v.Buttons))
		for i, b := range v.Buttons {
			r := ui.TownSurfaceButtonRect(v.Kind, i)
			mid := image.Pt((r.Min.X+r.Max.X)/2, (r.Min.Y+r.Max.Y)/2)
			c, hit := ui.TownSurfaceControlAt(v, mid)
			mark := "ok"
			switch {
			case !b.Enabled:
				// A disabled button answers no control at all
				// (TownSurfaceControlAt's own rule, unchanged by this
				// story): the well is still where the art sits, but there
				// is nothing to hit-test here on a fresh room entry.
				if hit {
					mark, bad = "FAIL", true
				}
			case !hit || c.Kind != ui.TownSurfaceControlButton || c.Index != i:
				mark, bad = "FAIL", true
			}
			fmt.Fprintf(w, "drive %s button %d %q enabled %v well %v centre %v hit-test %#v %s\n",
				name, i, b.Label, b.Enabled, r, mid, c, mark)
		}
	}
	return bad, nil
}

// segments returns the [start,end) runs whose average is at or below thresh
// and whose length is at least min.
func segments(avg []float64, thresh float64, min int) [][2]int {
	var out [][2]int
	start := -1
	for i, v := range avg {
		below := v <= thresh
		if below && start < 0 {
			start = i
		} else if !below && start >= 0 {
			if i-start >= min {
				out = append(out, [2]int{start, i})
			}
			start = -1
		}
	}
	if start >= 0 && len(avg)-start >= min {
		out = append(out, [2]int{start, len(avg)})
	}
	return out
}
