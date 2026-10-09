// Command plaqueseams measures the room upper-plaque 16-column gap DIV-166
// tracks and renders every candidate pairing between a 16-wide shipped strip
// and a room's 160-wide buttonsarea.bmp, so the pairing can be read from a
// picture rather than a file name.
//
//	plaqueseams [-assets DIR] -png DIR
//
// WHAT IT MEASURES. TownWideUpperRegion (pkg/ui/townshell.go) is 176 pixels
// wide; the school's, the tavern's and the character generator's own
// buttonsarea.bmp are each 160 wide and are drawn into the narrower
// TownUpperRegion, leaving columns x:[464,480) uncovered. The shop's own
// ShopMenu.bmp is 176 wide and needs no addition. DIV-166 records eight
// 16-pixel-wide bitmaps under graphics.res's interface/ tree, of which three
// share the 238-row upper-region height: inn/luover.bmp, inn/ruover.bmp and
// chrgen/rollstatsr.bmp. No 16-wide bitmap was found under training/.
//
// This tool first re-runs the DIV-166 census independently — every BMP
// entry under interface/ narrower than 32 pixels, on whichever root it is
// pointed at — then composes every {candidate strip} x {room buttonsarea.bmp}
// pair at (464,0), 176 wide, and writes each composite so the frame
// ornament's continuity across x=480 can be read from the PNG.
//
// WHY A TEST CANNOT. Golden rule 2 forbids a Go test from reading an
// install, and every figure here is a property of installed art. The output
// is the evidence a hotfix ledger row quotes; no game data is written, only
// PNGs under the caller's own -png directory.
package main

import (
	"flag"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"againrom/pkg/formats/bmp"
	"againrom/pkg/game"
	"againrom/pkg/vfs"
)

const ifacePrefix = "graphics/interface/"

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "plaqueseams:", err)
		os.Exit(1)
	}
}

func run(args []string, w io.Writer) error {
	fs := flag.NewFlagSet("plaqueseams", flag.ContinueOnError)
	assets := fs.String("assets", "", "lawful game install root (or AGAINROM_ASSETS)")
	pngDir := fs.String("png", "", "directory outside the repository for the composed seam pictures")
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

	if err := census(src, w); err != nil {
		return err
	}

	rooms := []struct {
		label, addr string
	}{
		{"school", ifacePrefix + "training/buttonsarea.bmp"},
		{"tavern", ifacePrefix + "inn/buttonsarea.bmp"},
		{"chrgen", ifacePrefix + "chrgen/buttonsarea.bmp"},
	}
	strips := []string{
		ifacePrefix + "inn/luover.bmp",
		ifacePrefix + "inn/ruover.bmp",
		ifacePrefix + "chrgen/rollstatsr.bmp",
	}

	for _, room := range rooms {
		plaque, err := readPic(src, room.addr)
		if err != nil {
			return err
		}
		fmt.Fprintf(w, "\n%s buttonsarea %dx%d\n", room.label, plaque.Width, plaque.Height)
		for _, sAddr := range strips {
			strip, err := readPic(src, sAddr)
			if err != nil {
				fmt.Fprintf(w, "  %s: %v\n", sAddr, err)
				continue
			}
			if strip.Height != plaque.Height {
				fmt.Fprintf(w, "  %-40s height %d != plaque height %d, skipped\n", sAddr, strip.Height, plaque.Height)
				continue
			}
			score := seamScore(strip, plaque)
			fmt.Fprintf(w, "  %-40s seam mean-abs-diff %.2f\n", sAddr, score)
			if *pngDir != "" {
				if err := writeComposite(*pngDir, room.label, sAddr, strip, plaque); err != nil {
					return err
				}
			}
		}
	}

	if err := family242(src, w, *pngDir); err != nil {
		return err
	}
	if err := familyLeftColumn238(src, w); err != nil {
		return err
	}
	if err := shopmenuSelfCheck(src, w); err != nil {
		return err
	}
	return nil
}

// namedPic pairs a short label (used in output and PNG file names) with the
// archive address it reads.
type namedPic struct {
	label, addr string
}

// family242 is round-1 adversarial review's own required extension: the
// 242-row column-pane population — every candidate body/seam pair this
// story's own character panel could draw — scored in BOTH join directions,
// not only the 238-row upper-plaque population census() and the room loop
// above already covered. A right-column layout puts the 16-wide strip at
// columns 0-15 and the 160-wide body at columns 16-175, so the border sits
// between the strip's own rightmost column and the body's own leftmost
// column: seamScore(strip, body). A left-column layout is the mirror,
// seamScore(body, strip). Printing both for every body/strip pair in this
// population is what let this story's own landing tell the two shipped
// families apart by rendering rather than by file name (DIV-175).
func family242(src *vfs.FS, w io.Writer, pngDir string) error {
	bodies := []namedPic{
		{"humanbackr", ifacePrefix + "humanbackr.bmp"},
		{"textbackr", ifacePrefix + "textbackr.bmp"},
		{"fullstatsl", ifacePrefix + "chrgen/fullstatsl.bmp"},
		{"leftpicture", ifacePrefix + "inn/leftpicture.bmp"},
	}
	strips := []namedPic{
		{"humanbackl", ifacePrefix + "humanbackl.bmp"},
		{"textbackl", ifacePrefix + "textbackl.bmp"},
		{"fullstatsr", ifacePrefix + "chrgen/fullstatsr.bmp"},
		{"ldover", ifacePrefix + "inn/ldover.bmp"},
		{"tav_09", ifacePrefix + "inn/tav_09.bmp"},
	}
	fmt.Fprintf(w, "\n242-row column-pane population, both join directions\n")
	for _, b := range bodies {
		body, err := readPic(src, b.addr)
		if err != nil {
			fmt.Fprintf(w, "  %s: %v\n", b.addr, err)
			continue
		}
		for _, s := range strips {
			strip, err := readPic(src, s.addr)
			if err != nil {
				fmt.Fprintf(w, "  %s: %v\n", s.addr, err)
				continue
			}
			if strip.Height != body.Height {
				fmt.Fprintf(w, "  %s|%s: height %d != %d, skipped\n", s.label, b.label, strip.Height, body.Height)
				continue
			}
			right := seamScore(strip, body) // right column: strip|body
			left := seamScore(body, strip)  // left column: body|strip
			fmt.Fprintf(w, "  %-12s|%-12s right-column(strip|body) %.2f  left-column(body|strip) %.2f\n",
				s.label, b.label, right, left)
			if pngDir != "" {
				if err := writeComposite(pngDir, "right-"+b.label, s.addr, strip, body); err != nil {
					return err
				}
				if err := writeComposite(pngDir, "left-"+b.label, s.addr, body, strip); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// familyLeftColumn238 is family242's own 238-row sibling: the stat-plate and
// tavern-stats candidates this story's own row 2 and row 11 fixes draw,
// scored the same way. leftup.bmp sits under main/graphics/chrgen/, not
// interface/, so it is read with its own prefix rather than census()'s.
func familyLeftColumn238(src *vfs.FS, w io.Writer) error {
	pairs := []struct{ body, strip namedPic }{
		{namedPic{"leftstats", ifacePrefix + "inn/leftstats.bmp"}, namedPic{"luover", ifacePrefix + "inn/luover.bmp"}},
		{namedPic{"rollstatsl", ifacePrefix + "chrgen/rollstatsl.bmp"}, namedPic{"rollstatsr", ifacePrefix + "chrgen/rollstatsr.bmp"}},
		{namedPic{"leftup", "main/graphics/chrgen/leftup.bmp"}, namedPic{"rollstatsr", ifacePrefix + "chrgen/rollstatsr.bmp"}},
	}
	fmt.Fprintf(w, "\n238-row left-column population\n")
	for _, p := range pairs {
		body, err := readPic(src, p.body.addr)
		if err != nil {
			fmt.Fprintf(w, "  %s: %v\n", p.body.addr, err)
			continue
		}
		strip, err := readPic(src, p.strip.addr)
		if err != nil {
			fmt.Fprintf(w, "  %s: %v\n", p.strip.addr, err)
			continue
		}
		if strip.Height != body.Height {
			fmt.Fprintf(w, "  %s|%s: height %d != %d, skipped\n", p.body.label, p.strip.label, strip.Height, body.Height)
			continue
		}
		fmt.Fprintf(w, "  %-12s|%-12s left-column(body|strip) %.2f\n", p.body.label, p.strip.label, seamScore(body, strip))
	}
	return nil
}

// shopmenuSelfCheck decomposes the shop's own shipped interface/shopmenu.bmp
// (176x238) — the one production slot this story's whole population table
// names as already correct and needing no addition — into its own left 16
// columns and right 160 columns, and reports the mean per-channel difference
// of each half against inn/ruover.bmp and chrgen/buttonsarea.bmp over the
// WHOLE sub-image, not only the border column seamScore reads.
func shopmenuSelfCheck(src *vfs.FS, w io.Writer) error {
	menu, err := readPic(src, ifacePrefix+"shopmenu.bmp")
	if err != nil {
		return err
	}
	ruover, err := readPic(src, ifacePrefix+"inn/ruover.bmp")
	if err != nil {
		return err
	}
	buttons, err := readPic(src, ifacePrefix+"chrgen/buttonsarea.bmp")
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "\nshopmenu.bmp self-check (%dx%d)\n", menu.Width, menu.Height)
	fmt.Fprintf(w, "  left 16 columns vs inn/ruover.bmp:          mean-abs-diff %.2f\n", regionDiff(menu, 0, ruover))
	fmt.Fprintf(w, "  right 160 columns vs chrgen/buttonsarea.bmp: mean-abs-diff %.2f\n", regionDiff(menu, 16, buttons))
	return nil
}

// regionDiff is the mean per-channel absolute difference between src's own
// sub-image starting at column xOff and ref, over ref's whole width and
// height (not one border column, unlike seamScore).
func regionDiff(src *bmp.Image, xOff int, ref *bmp.Image) float64 {
	if src.Height != ref.Height || src.Width < xOff+ref.Width {
		return -1
	}
	var sum, n int
	for y := 0; y < ref.Height; y++ {
		for x := 0; x < ref.Width; x++ {
			a := src.At(xOff+x, y)
			b := ref.At(x, y)
			sum += absInt(int(a.R)-int(b.R)) + absInt(int(a.G)-int(b.G)) + absInt(int(a.B)-int(b.B))
			n += 3
		}
	}
	return float64(sum) / float64(n)
}

// census re-derives DIV-166's own population: every BMP entry under
// interface/ narrower than 32 pixels, on the root this run is pointed at.
// It reads no shipped bytes into the repository; it prints path, width and
// height only, through the archive's own listing rather than a remembered
// file list.
func census(src *vfs.FS, w io.Writer) error {
	var addrs []string
	for _, e := range src.Entries() {
		if strings.Contains(e.Address, ifacePrefix) && strings.HasSuffix(strings.ToLower(e.Address), ".bmp") {
			addrs = append(addrs, e.Address)
		}
	}
	sort.Strings(addrs)
	fmt.Fprintf(w, "census: %d .bmp entries under %s\n", len(addrs), ifacePrefix)
	narrow := 0
	for _, a := range addrs {
		raw, err := src.ReadFile(a)
		if err != nil {
			return fmt.Errorf("%s: %w", a, err)
		}
		im, err := bmp.Decode(raw)
		if err != nil {
			// Not every interface/ entry is a 24bpp bitmap this decoder
			// accepts (masks and paletted art are excluded by design); a
			// decode failure here is not a census failure, only a skip.
			continue
		}
		if im.Width < 32 {
			narrow++
			fmt.Fprintf(w, "  narrow: %-50s %dx%d\n", a, im.Width, im.Height)
		}
	}
	fmt.Fprintf(w, "census: %d entries under 32px wide\n", narrow)
	return nil
}

func readPic(src *vfs.FS, addr string) (*bmp.Image, error) {
	raw, err := src.ReadFile(addr)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", addr, err)
	}
	im, err := bmp.Decode(raw)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", addr, err)
	}
	return im, nil
}

// seamScore is the mean per-channel absolute difference between the strip's
// own rightmost column (x=15) and the plaque's own leftmost column (x=0) —
// the two columns that sit immediately either side of x=480 once the strip
// is placed at (464,0). A low score is a border ornament that continues
// across the seam; a high score is a strip whose right edge disagrees with
// what it would sit against.
func seamScore(strip, plaque *bmp.Image) float64 {
	if strip.Height != plaque.Height {
		return -1
	}
	var sum, n int
	for y := 0; y < strip.Height; y++ {
		a := strip.At(strip.Width-1, y)
		b := plaque.At(0, y)
		sum += absInt(int(a.R)-int(b.R)) + absInt(int(a.G)-int(b.G)) + absInt(int(a.B)-int(b.B))
		n += 3
	}
	return float64(sum) / float64(n)
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// writeComposite writes the 176-wide seam alone (strip at (0,0), plaque at
// (16,0)) under -png/seam-<room>-<strip-file>.png, at 4x nearest-neighbour
// scale so the seam is legible without a viewer that zooms.
func writeComposite(dir, room, stripAddr string, strip, plaque *bmp.Image) error {
	room = sanitise(room)
	stripFile := sanitise(filepath.Base(stripAddr))
	h := plaque.Height
	seam := image.NewRGBA(image.Rect(0, 0, 176, h))
	draw.Draw(seam, image.Rect(0, 0, 16, h), strip.RGBA(), image.Point{}, draw.Src)
	draw.Draw(seam, image.Rect(16, 0, 176, h), plaque.RGBA(), image.Point{}, draw.Src)
	scaled := scale4x(seam)
	name := filepath.Join(dir, fmt.Sprintf("seam-%s-%s.png", room, stripFile))
	f, err := os.Create(name)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, scaled)
}

func scale4x(src *image.RGBA) *image.RGBA {
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx()*4, b.Dy()*4))
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			c := src.At(b.Min.X+x, b.Min.Y+y)
			for dy := 0; dy < 4; dy++ {
				for dx := 0; dx < 4; dx++ {
					dst.Set(x*4+dx, y*4+dy, c)
				}
			}
		}
	}
	return dst
}

func sanitise(s string) string {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '.', r == '-':
			out = append(out, r)
		default:
			out = append(out, '_')
		}
	}
	return string(out)
}
