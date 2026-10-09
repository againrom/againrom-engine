// Command townsquarecheck measures the town square's shipped art against a
// lawful game install.
//
//	townsquarecheck [-assets DIR] [-png DIR]
//
// townmain.bmp is the base picture; town_add.bmp is an overlay strip
// composited over it; townmask.bmp is a paletted raster whose colour codes
// carry the five interactive regions (four doors and the statue); shop_l.bmp,
// tavern_l.bmp and trener_l.bmp are the three door labels. Where each label
// sits, and which mask code belongs to which door, is not published by any
// research claim, so this tool derives both from the shipped art by
// correlation and from the shipped tip text, then cross-checks every answer
// against production.
//
// DIV-148
package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"

	"againrom/pkg/formats/bmp"
	"againrom/pkg/game"
	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

// matchTolerance is the per-pixel sum of the three absolute channel
// differences a match may carry.
const matchTolerance = 12

const separation = 4

const townArtPrefix = "graphics/interface/town/"

// labelName is each label's file name; labelArt is the art entry the ROM1
// town description gives it, same order.
var labelName = [3]string{"shop_l", "tavern_l", "trener_l"}
var labelArt = [3]string{"label-shop", "label-tavern", "label-school"}

// layerAt answers where the ROM1 town description paints the named art.
func layerAt(art string) image.Point {
	for _, l := range game.ROM1TownDescription().Layers {
		if l.Art == art {
			return l.At.Pt()
		}
	}
	return image.Pt(-1, -1)
}

// labelAddr is each label's own archive address, same order.
var labelAddr = [3]string{
	townArtPrefix + "shop_l.bmp",
	townArtPrefix + "tavern_l.bmp",
	townArtPrefix + "trener_l.bmp",
}

// wantDoor is which door (ui.TownSquareControl's Door index, pkg/game's
// townDoors order: 0 tavern, 1 shop, 2 school, 3 gates) each label is
// expected to sit over, and wantCode is the mask code expected to dominate
// under it. Both are this tool's OWN independent reading of the shipped tip
// text (main/text/tips/town.txt) and the correlation below, kept apart from
// the ROM1 town description so the two can disagree.
var wantDoor = [3]int{1, 0, 2} // shop, tavern, school
var wantCode = [3]uint8{144, 128, 192}

const (
	codeTavern = 128
	codeShop   = 144
	codeGate   = 160
	codeStatue = 176
	codeSchool = 192
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "townsquarecheck:", err)
		os.Exit(1)
	}
}

func readPic(src terrain.EntrySource, addr string) (*image.RGBA, int, error) {
	raw, err := src.ReadFile(addr)
	if err != nil {
		return nil, 0, fmt.Errorf("%s: %w", addr, err)
	}
	im, err := bmp.Decode(raw)
	if err != nil {
		return nil, 0, fmt.Errorf("%s: %w", addr, err)
	}
	out := image.NewRGBA(image.Rect(0, 0, im.Width, im.Height))
	black := 0
	for i, c := range im.Pix {
		o := i * 4
		out.Pix[o], out.Pix[o+1], out.Pix[o+2], out.Pix[o+3] = c.R, c.G, c.B, 0xff
		if c.R == 0 && c.G == 0 && c.B == 0 {
			black++
		}
	}
	return out, black, nil
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

func diff(a, b uint8) int {
	if a > b {
		return int(a) - int(b)
	}
	return int(b) - int(a)
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

// codeAnchor answers a pixel inside CODE's own LARGEST connected component,
// the one nearest that component's own centroid.
//
// A BOUNDING BOX CENTRE IS NOT ENOUGH, and neither is a component-blind
// nearest-any-pixel search (round-2 adversarial review, W-1). The gate and
// tavern regions are thin stone-arch outlines carrying single-pixel
// same-code strays scattered across their own bounding box: code 128 (the
// tavern) has 15 connected components, 14 of them a single pixel; code 160
// (the gate) has 17, 16 of them a single pixel. Both strays stretch their
// own bounding box far past the real archway blob, and a nearest-any-pixel
// anchor landed on one of them — the tool's own hit-test and drive lines
// answered correctly there too, since the scene's ControlAt maps the colour
// index alone and answers identically for every pixel of a code wherever it
// lies, but that agreement demonstrated
// only that an isolated rounding artifact resolves correctly, not that a
// click on the VISIBLE graphic does.
//
// Connected components are found by 4-connected flood fill over the code's
// own pixels inside box, which already bounds every pixel of that code
// (maskBoxes). The largest component's own pixel count decides which one is
// "the graphic"; the returned anchor is always one of ITS OWN pixels, never
// a pixel outside it, even when that pixel is farther from box's raw
// midpoint than a stray would have been.
func codeAnchor(mask *image.Paletted, code uint8, box image.Rectangle) image.Point {
	visited := map[image.Point]bool{}
	var largest []image.Point
	for y := box.Min.Y; y < box.Max.Y; y++ {
		for x := box.Min.X; x < box.Max.X; x++ {
			p := image.Pt(x, y)
			if visited[p] || mask.ColorIndexAt(x, y) != code {
				continue
			}
			comp := floodFillCode(mask, code, p, box, visited)
			if len(comp) > len(largest) {
				largest = comp
			}
		}
	}
	if len(largest) == 0 {
		return image.Point{}
	}
	var sx, sy int
	for _, p := range largest {
		sx += p.X
		sy += p.Y
	}
	cx, cy := sx/len(largest), sy/len(largest)
	best, bestD := largest[0], -1
	for _, p := range largest {
		dx, dy := p.X-cx, p.Y-cy
		if d := dx*dx + dy*dy; bestD < 0 || d < bestD {
			bestD, best = d, p
		}
	}
	return best
}

// floodFillCode walks CODE's own 4-connected component containing start,
// within box, marking every visited pixel in visited (shared across calls so
// codeAnchor's outer scan never re-walks a component it already found).
func floodFillCode(mask *image.Paletted, code uint8, start image.Point, box image.Rectangle, visited map[image.Point]bool) []image.Point {
	comp := []image.Point{start}
	visited[start] = true
	stack := []image.Point{start}
	for len(stack) > 0 {
		p := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		for _, d := range [4]image.Point{{X: 1}, {X: -1}, {Y: 1}, {Y: -1}} {
			n := p.Add(d)
			if !n.In(box) || visited[n] || mask.ColorIndexAt(n.X, n.Y) != code {
				continue
			}
			visited[n] = true
			comp = append(comp, n)
			stack = append(stack, n)
		}
	}
	return comp
}

// maskBoxes answers each colour code's bounding box and pixel count.
func maskBoxes(mask *image.Paletted) (map[uint8]image.Rectangle, map[uint8]int) {
	boxes := map[uint8]image.Rectangle{}
	counts := map[uint8]int{}
	b := mask.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			c := mask.ColorIndexAt(x, y)
			counts[c]++
			p := image.Rect(x, y, x+1, y+1)
			if r, ok := boxes[c]; ok {
				boxes[c] = r.Union(p)
			} else {
				boxes[c] = p
			}
		}
	}
	return boxes, counts
}

func run(args []string, w io.Writer) error {
	fs := flag.NewFlagSet("townsquarecheck", flag.ContinueOnError)
	assets := fs.String("assets", "", "lawful game install root (or AGAINROM_ASSETS)")
	pngDir := fs.String("png", "", "directory outside the repository for the composed pictures")
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

	main1, black, err := readPic(src, townArtPrefix+"townmain.bmp")
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "townmain.bmp %dx%d black %d\n", main1.Bounds().Dx(), main1.Bounds().Dy(), black)

	mraw, err := src.ReadFile(townArtPrefix + "townmask.bmp")
	if err != nil {
		return err
	}
	mask, err := terrain.DecodeBMP8(mraw)
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "townmask.bmp %dx%d\n", mask.Bounds().Dx(), mask.Bounds().Dy())
	boxes, counts := maskBoxes(mask)
	fmt.Fprintf(w, "distinct codes %d\n", len(counts))
	// Code 0 is the background — the whole rest of the picture, always over
	// this threshold, and not one of the five interactive regions. It is
	// reported like every other code above the threshold but excluded from
	// the "how many interactive regions" count below.
	// SORTED BY CODE (round-3 review, W-5): counts is a Go map, and ranging it
	// directly printed these lines in map order, which varies run to run — a
	// prior closure claimed byte-identical output on both roots when the
	// wording could not have been checked with `diff` as this tool stood.
	codes := make([]uint8, 0, len(counts))
	for c := range counts {
		codes = append(codes, c)
	}
	sort.Slice(codes, func(i, j int) bool { return codes[i] < codes[j] })
	sigCodes := []uint8{}
	for _, c := range codes {
		n := counts[c]
		if n > 1000 {
			if c != 0 {
				sigCodes = append(sigCodes, c)
			}
			fmt.Fprintf(w, "code %3d count %6d box %v\n", c, n, boxes[c])
		}
	}
	if len(sigCodes) != 5 {
		fmt.Fprintf(w, "FAIL mask carries %d non-background codes over 1000px, want 5\n", len(sigCodes))
		fail = true
	}
	for _, want := range []uint8{codeTavern, codeShop, codeGate, codeStatue, codeSchool} {
		if _, ok := boxes[want]; !ok {
			fmt.Fprintf(w, "FAIL expected code %d absent from the mask\n", want)
			fail = true
		}
	}

	// Instrument A: town_add.bmp against the description's overlay layer.
	addPic, addBlack, err := readPic(src, townArtPrefix+"town_add.bmp")
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "town_add.bmp %dx%d black %d\n", addPic.Bounds().Dx(), addPic.Bounds().Dy(), addBlack)
	ac := correlate(main1, addPic)
	mark := "ok"
	if ac.at != layerAt("overlay") {
		mark, fail = "FAIL", true
	}
	fmt.Fprintf(w, "town_add best %v frac %.4f (%d total) next %.4f at %v production %v %s\n",
		ac.at, ac.fraction, ac.total, ac.next, ac.nextAt, layerAt("overlay"), mark)

	if *pngDir != "" {
		if err := os.MkdirAll(*pngDir, 0o755); err != nil {
			return err
		}
		writePNG(filepath.Join(*pngDir, "townmain.png"), main1)
		writePNG(filepath.Join(*pngDir, "town_add.png"), addPic)
		palette := map[uint8][3]uint8{
			codeTavern: {255, 0, 0},
			codeShop:   {0, 255, 0},
			codeGate:   {0, 128, 255},
			codeStatue: {255, 255, 0},
			codeSchool: {255, 0, 255},
		}
		overlay := image.NewRGBA(main1.Bounds())
		copy(overlay.Pix, main1.Pix)
		b := mask.Bounds()
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				col, ok := palette[mask.ColorIndexAt(x, y)]
				if !ok {
					continue
				}
				o := overlay.PixOffset(x, y)
				overlay.Pix[o] = uint8((uint16(overlay.Pix[o]) + uint16(col[0])) / 2)
				overlay.Pix[o+1] = uint8((uint16(overlay.Pix[o+1]) + uint16(col[1])) / 2)
				overlay.Pix[o+2] = uint8((uint16(overlay.Pix[o+2]) + uint16(col[2])) / 2)
			}
		}
		writePNG(filepath.Join(*pngDir, "mask-overlay.png"), overlay)
	}

	// Instrument A and B, per label: where it sits (correlation) and which
	// mask code dominates underneath it (the raster).
	labelDomCode := [3]uint8{}
	for i, addr := range labelAddr {
		p, pblack, err := readPic(src, addr)
		if err != nil {
			return err
		}
		fmt.Fprintf(w, "%s %dx%d black %d\n", labelName[i], p.Bounds().Dx(), p.Bounds().Dy(), pblack)
		c := correlate(main1, p)
		want := layerAt(labelArt[i])
		mark := "ok"
		if c.at != want {
			mark, fail = "FAIL", true
		}
		fmt.Fprintf(w, "%s best %v frac %.4f (%d total) next %.4f at %v production %v %s\n",
			labelName[i], c.at, c.fraction, c.total, c.next, c.nextAt, want, mark)

		codeCount := map[uint8]int{}
		for y := 0; y < p.Bounds().Dy(); y++ {
			for x := 0; x < p.Bounds().Dx(); x++ {
				mx, my := c.at.X+x, c.at.Y+y
				if mx < 0 || my < 0 || mx >= mask.Bounds().Dx() || my >= mask.Bounds().Dy() {
					continue
				}
				codeCount[mask.ColorIndexAt(mx, my)]++
			}
		}
		dom, domN := uint8(0), 0
		for code, n := range codeCount {
			if n > domN {
				dom, domN = code, n
			}
		}
		labelDomCode[i] = dom
		mark = "ok"
		if dom != wantCode[i] {
			mark, fail = "FAIL", true
		}
		fmt.Fprintf(w, "%s dominant mask code %d (%d of %d px) want %d %s\n",
			labelName[i], dom, domN, p.Bounds().Dx()*p.Bounds().Dy(), wantCode[i], mark)

		if *pngDir != "" {
			writePNG(filepath.Join(*pngDir, labelName[i]+"-label.png"), p)
			crop := image.NewRGBA(p.Bounds())
			for y := 0; y < p.Bounds().Dy(); y++ {
				do := (c.at.Y+y)*main1.Stride + c.at.X*4
				co := y * crop.Stride
				copy(crop.Pix[co:co+p.Bounds().Dx()*4], main1.Pix[do:do+p.Bounds().Dx()*4])
			}
			writePNG(filepath.Join(*pngDir, labelName[i]+"-basecrop.png"), crop)
			diffImg := image.NewRGBA(p.Bounds())
			for y := 0; y < p.Bounds().Dy(); y++ {
				po := y * p.Stride
				co := y * crop.Stride
				do := y * diffImg.Stride
				for x := 0; x < p.Bounds().Dx(); x++ {
					pr, pg, pb := p.Pix[po], p.Pix[po+1], p.Pix[po+2]
					cr, cg, cb := crop.Pix[co], crop.Pix[co+1], crop.Pix[co+2]
					switch {
					case pr == 0 && pg == 0 && pb == 0:
						diffImg.Pix[do], diffImg.Pix[do+1], diffImg.Pix[do+2], diffImg.Pix[do+3] = 0, 0, 0, 255
					case pr == cr && pg == cg && pb == cb:
						diffImg.Pix[do], diffImg.Pix[do+1], diffImg.Pix[do+2], diffImg.Pix[do+3] = 0, 128, 0, 255
					default:
						diffImg.Pix[do], diffImg.Pix[do+1], diffImg.Pix[do+2], diffImg.Pix[do+3] = 255, 0, 0, 255
					}
					po += 4
					co += 4
					do += 4
				}
			}
			writePNG(filepath.Join(*pngDir, labelName[i]+"-diff.png"), diffImg)
		}
	}

	// Instrument B against production, over the install's OWN mask loaded
	// through the production loader rather than this tool's decoder.
	art, err := game.LoadTownSquareArt(src)
	if err != nil {
		return err
	}
	base := art.Pictures("base")
	if len(base) == 0 || art.Mask == nil {
		return fmt.Errorf("LoadTownSquareArt returned an incomplete art set")
	}
	if !sameRGBA(main1, base[0]) {
		fmt.Fprintln(w, "FAIL production LoadTownSquareArt's Background differs from this tool's own decode of townmain.bmp")
		fail = true
	} else {
		fmt.Fprintln(w, "production Background matches this tool's own decode: ok")
	}

	expect := map[uint8]struct {
		kind ui.TownSquareControlKind
		door int
	}{
		codeTavern: {ui.TownSquareControlDoor, 0},
		codeShop:   {ui.TownSquareControlDoor, 1},
		codeSchool: {ui.TownSquareControlDoor, 2},
		codeGate:   {ui.TownSquareControlDoor, 3},
		codeStatue: {ui.TownSquareControlMenu, 0},
	}
	scene, err := squareScene(root)
	if err != nil {
		return err
	}
	for _, code := range []uint8{codeTavern, codeShop, codeGate, codeStatue, codeSchool} {
		box, ok := boxes[code]
		if !ok {
			continue
		}
		mid := codeAnchor(mask, code, box)
		got, hit := scene.ControlAt(mid)
		want := expect[code]
		mark := "ok"
		if !hit || got.Kind != want.kind || (want.kind == ui.TownSquareControlDoor && got.Door != want.door) {
			mark, fail = "FAIL", true
		}
		fmt.Fprintf(w, "hit-test code %3d at %v -> kind %d door %d, want kind %d door %d %s\n",
			code, mid, got.Kind, got.Door, want.kind, want.door, mark)
	}
	// SELF-CONSISTENCY CHECK, NOT A PRODUCTION CHECK (round-3 review, D-3): this
	// loop compares two tables THIS TOOL OWN AUTHORS — labelDomCode (measured by
	// correlation, above) against expect and wantDoor (both hand-written
	// constants in this file) — and never calls the scene's ControlAt or any
	// other production code. Under a mutation of the production tavern
	// mapping, this line stayed "ok" while the "hit-test" lines failed:
	// production's own code-to-door mapping is covered by those lines alone.
	// This one only catches this file's own two tables disagreeing with each
	// other.
	for i, code := range labelDomCode {
		want := wantDoor[i]
		got, ok := expect[code]
		mark := "ok"
		if !ok || got.kind != ui.TownSquareControlDoor || got.door != want {
			mark, fail = "FAIL", true
		}
		fmt.Fprintf(w, "self-check: label %s's own code %d, this tool's expect table door %d, want door %d %s\n",
			labelName[i], code, got.door, want, mark)
	}

	// The game side, driven through the production town screen: each door
	// code is fed to Choose(i) on a freshly started campaign town (Choose
	// mutates room state, so each door needs its own front end) and the
	// resulting room is read back from Header(), independently of the
	// ControlAt call that produced the index.
	for _, code := range []uint8{codeTavern, codeShop, codeSchool, codeGate} {
		box := boxes[code]
		mid := codeAnchor(mask, code, box)
		if bad, err := driveDoor(root, code, mid, w); err != nil {
			return err
		} else if bad {
			fail = true
		}
	}
	if bad, err := driveStatue(root, codeAnchor(mask, codeStatue, boxes[codeStatue]), w); err != nil {
		return err
	} else if bad {
		fail = true
	}

	if fail {
		return fmt.Errorf("the instruments disagree; see the FAIL lines above")
	}
	fmt.Fprintln(w, "townsquarecheck: ok")
	return nil
}

// squareScene answers the production square scene of a freshly started
// campaign town.
func squareScene(root string) (ui.TownSquareScene, error) {
	f, err := game.NewFrontEnd(root)
	if err != nil {
		return nil, err
	}
	if f.Town == nil {
		return nil, fmt.Errorf("the front end started with no town")
	}
	f.Town.Arrive()
	as, ok := f.TownScreen().(ui.TownSquareArtScreen)
	if !ok {
		return nil, fmt.Errorf("the town screen does not implement TownSquareArtScreen")
	}
	v := as.TownSquareView()
	if v.Scene == nil {
		return nil, fmt.Errorf("the town screen's square has no scene")
	}
	return v.Scene, nil
}

// sameRGBA compares two images pixel for pixel over their shared bounds size.
func sameRGBA(a, b image.Image) bool {
	ab, bb := a.Bounds(), b.Bounds()
	if ab.Dx() != bb.Dx() || ab.Dy() != bb.Dy() {
		return false
	}
	for y := 0; y < ab.Dy(); y++ {
		for x := 0; x < ab.Dx(); x++ {
			ar, ag, abl, aa := a.At(ab.Min.X+x, ab.Min.Y+y).RGBA()
			br, bg, bbl, ba := b.At(bb.Min.X+x, bb.Min.Y+y).RGBA()
			if ar != br || ag != bg || abl != bbl || aa != ba {
				return false
			}
		}
	}
	return true
}

// codeDoorName names each door code for the log lines below.
var codeDoorName = map[uint8]string{
	codeTavern: "tavern", codeShop: "shop", codeSchool: "school", codeGate: "gates",
}

// driveDoor starts one fresh campaign front end, arrives at the town square,
// resolves the given mask code through production's own scene ControlAt,
// and feeds the resulting door index to production's own Choose(i) — the
// same seam the row-button grid has always used. The room Choose(i) leaves
// the screen in is read back from Header(), a call this loop never fed the
// index into, so the check does not compare a value with itself.
func driveDoor(root string, code uint8, mid image.Point, w io.Writer) (bool, error) {
	f, err := game.NewFrontEnd(root)
	if err != nil {
		return false, err
	}
	if f.Town == nil {
		return false, fmt.Errorf("drive: the front end started with no town")
	}
	f.Town.Arrive()
	screen := f.TownScreen()
	as, ok := screen.(ui.TownSquareArtScreen)
	if !ok {
		return false, fmt.Errorf("drive: the town screen does not implement TownSquareArtScreen")
	}
	v := as.TownSquareView()
	if v.Scene == nil {
		return false, fmt.Errorf("drive: the town screen's square has no scene")
	}
	c, hit := v.Scene.ControlAt(mid)
	if !hit || c.Kind != ui.TownSquareControlDoor {
		fmt.Fprintf(w, "FAIL drive code %d at %v: production hit-test answers no door\n", code, mid)
		return true, nil
	}
	screen.Choose(c.Door)
	header := strings.ToLower(screen.Header())
	want := codeDoorName[code]
	bad := false
	mark := "ok"
	if !strings.Contains(header, want) {
		mark, bad = "FAIL", true
	}
	fmt.Fprintf(w, "drive code %d door %d Choose -> header %q contains %q %s\n", code, c.Door, header, want, mark)
	return bad, nil
}

// driveStatue confirms production's own hit test answers the mini-menu kind
// at the statue's own mask region, over a freshly started campaign town. The
// statue does not route through Choose — it is app.go's own escape seam
// (0143) — so this stops at the resolved kind, which pkg/ui/townsquare_test.go
// carries the rest of (TestTownSquareStatueClickOpensTheMiniMenu).
func driveStatue(root string, mid image.Point, w io.Writer) (bool, error) {
	f, err := game.NewFrontEnd(root)
	if err != nil {
		return false, err
	}
	if f.Town == nil {
		return false, fmt.Errorf("drive: the front end started with no town")
	}
	f.Town.Arrive()
	screen := f.TownScreen()
	as, ok := screen.(ui.TownSquareArtScreen)
	if !ok {
		return false, fmt.Errorf("drive: the town screen does not implement TownSquareArtScreen")
	}
	v := as.TownSquareView()
	if v.Scene == nil {
		return false, fmt.Errorf("drive: the town screen's square has no scene")
	}
	c, hit := v.Scene.ControlAt(mid)
	mark := "ok"
	bad := false
	if !hit || c.Kind != ui.TownSquareControlMenu {
		mark, bad = "FAIL", true
	}
	fmt.Fprintf(w, "drive statue at %v -> kind %d, want menu %s\n", mid, c.Kind, mark)
	return bad, nil
}

func writePNG(path string, img image.Image) {
	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	if err := png.Encode(f, img); err != nil {
		panic(err)
	}
	if err := f.Close(); err != nil {
		panic(err)
	}
}
