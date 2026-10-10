// Command schoolcheck measures the skill school's column art against a lawful
// game install and prints every figure the 1015 story rests on.
//
//	schoolcheck [-assets DIR] [-png DIR]
//	schoolcheck [-assets DIR] -transparency-only
//
// The read-only -transparency-only census skips correlation and counts every
// pixel of all ten skills in all three states. It compares source RGB black
// with zero after standard RGB565/RGB555 channel truncation, not arbitrary
// runtime framebuffer masks. Non-black zero counts are reported, not hidden
// inside the pure-black population. Missing or invalid patches fail the census.
//
// WHAT IT MEASURES. The room background trnhall.bmp bakes one rotation frame of
// the training column into itself. The archive ships that column as sixteen
// separate frames, column/rt0000.bmp .. rt0015.bmp, and ten skill patches in
// three lit states each. Where each patch belongs on its own column face is not
// published by any claim for the mage class, so this tool derives it from the
// shipped art by correlation and cross-checks every answer against a second,
// independent instrument: the class mask's own colour-code bounding boxes.
//
// WHY A TEST CANNOT. Golden rule 2 forbids a Go test from reading an install,
// and every figure here is a property of installed art. cmd/paneldump carries
// the same reasoning for the unit panel. The output of this command is the
// evidence the story's closure.md quotes; the unit tests cover the production
// code paths with synthetic fixtures.
//
// THE TWO INSTRUMENTS.
//
// Instrument A, correlation: slide a patch over a destination picture and, at
// each offset, count the patch pixels whose per-channel absolute differences
// sum to at most matchTolerance. Pure-black patch pixels are excluded from both
// the numerator and the denominator, because pure black is this art's
// transparent colour (DIV-013's rule, keyBlack in pkg/game/shopart.go) and the
// build does not draw those pixels. A match is reported with the best fraction
// anywhere else in the offset space, so separation is a number rather than an
// assertion. The three lit states of one skill are an internal control: they
// must agree on one offset.
//
// Instrument B, the raster mask: each class ships mask.bmp, a paletted image
// whose five interactive regions are five colour codes. Each code's bounding
// box, placed at the class panel rectangle, is matched to the patch rectangle
// it overlaps most. The two rectangles are not nested in a fixed direction: on
// the fighter every mask box is smaller than its patch, on the mage every mask
// box is larger, so the test is intersection area over the smaller of the two
// areas, and the five-to-five assignment is required to be a bijection. That is
// the mask-code-to-skill correspondence, computed here rather than assumed.
//
// The two instruments are then checked against production: ui.SchoolSkillRect
// must equal instrument A's rectangle, and ui.TownSurfaceControlAt, run over
// the install's own mask, must answer instrument B's cell index. Any
// disagreement is a non-zero exit.
//
// It writes no game data. The optional -png directory receives the composed
// school surface for each class, our own composition of the install's pictures,
// and must be outside the repository.
package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"image"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"againrom/pkg/formats/bmp"
	"againrom/pkg/game"
	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

// matchTolerance is the per-pixel sum of the three absolute channel differences
// a match may carry.
const matchTolerance = 12

const columnPrefix = "graphics/interface/training/column/"

// restFrame is the rotation frame each class's face rests at.
//
// WHICH FRAME BELONGS TO WHICH CLASS IS MEASURED TWO WAYS, and the second was
// added at 1015's landing because the first cannot falsify this array. The
// tool prints every frame's agreement with the baked face at the established
// origin, which establishes the origin and the background's own baked class;
// it says nothing about the fighter. The `frame-of` lines correlate each
// class's five skill patches against all sixteen frames and require the winner
// to be the frame named here, so a wrong entry reddens the tool.
var restFrame = [2]int{0, 15}

var className = [2]string{"fighter", "mage"}

var skillName = [2][5]string{
	{"sword", "axe", "club", "pike", "bow"},
	{"fire", "water", "air", "earth", "astral"},
}

var stateName = [3]string{"on", "shine", "shine_on"}

var maskCodes = [5]uint8{0x37, 0x87, 0x9e, 0xd2, 0xff}

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "schoolcheck:", err)
		os.Exit(1)
	}
}

// pic is one decoded picture with its pure-black population already counted.
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
	out := pic{img: image.NewRGBA(image.Rect(0, 0, im.Width, im.Height))}
	for i, c := range im.Pix {
		o := i * 4
		out.img.Pix[o], out.img.Pix[o+1], out.img.Pix[o+2], out.img.Pix[o+3] = c.R, c.G, c.B, 0xff
		if c.R == 0 && c.G == 0 && c.B == 0 {
			out.black++
		}
	}
	return out, nil
}

type transparencyCounts struct {
	patches, pixels, black           int
	nonblackZero565, nonblackZero555 int
}

// patchTransparency reads source RGB, without production keyBlack. TOWN-150
// identifies the source-16-bit-zero key. TERR-LIGHT-019 supplies channel
// truncation; these two standard layouts do not assert the runtime masks.
func patchTransparency(raw []byte) (transparencyCounts, error) {
	// Bound the dimensions before Decode allocates. The school loader accepts
	// only positive patches inside the shared 80x36 envelope.
	if len(raw) >= bmp.HeaderLen {
		w := int32(binary.LittleEndian.Uint32(raw[18:22]))
		h := int32(binary.LittleEndian.Uint32(raw[22:26]))
		if w <= 0 || w > 80 || h <= 0 || h > 36 {
			return transparencyCounts{}, fmt.Errorf("skill image is %dx%d, outside the positive 80x36 envelope", w, h)
		}
	}
	im, err := bmp.Decode(raw)
	if err != nil {
		return transparencyCounts{}, err
	}
	out := transparencyCounts{patches: 1, pixels: len(im.Pix)}
	for _, c := range im.Pix {
		if c.R == 0 && c.G == 0 && c.B == 0 {
			out.black++
			continue
		}
		rgb565 := uint16(c.R>>3)<<11 | uint16(c.G>>2)<<5 | uint16(c.B>>3)
		rgb555 := uint16(c.R>>3)<<10 | uint16(c.G>>3)<<5 | uint16(c.B>>3)
		if rgb565 == 0 {
			out.nonblackZero565++
		}
		if rgb555 == 0 {
			out.nonblackZero555++
		}
	}
	return out, nil
}

func censusTransparency(src terrain.EntrySource, w io.Writer) error {
	var total transparencyCounts
	for class, skills := range skillName {
		for _, skill := range skills {
			for _, state := range stateName {
				addr := fmt.Sprintf("%s%s/%s/%s.bmp", columnPrefix, className[class], skill, state)
				raw, err := src.ReadFile(addr)
				if err != nil {
					return fmt.Errorf("%s: %w", addr, err)
				}
				p, err := patchTransparency(raw)
				if err != nil {
					return fmt.Errorf("%s: %w", addr, err)
				}
				total.patches += p.patches
				total.pixels += p.pixels
				total.black += p.black
				total.nonblackZero565 += p.nonblackZero565
				total.nonblackZero555 += p.nonblackZero555
			}
		}
	}
	// This census has no selection/filter mode: a partial population is not
	// evidence about all ten skills and all three states.
	if total.patches != 30 || total.pixels == 0 {
		return fmt.Errorf("incomplete transparency census: %d patches, %d pixels; want all 30 nonempty patches", total.patches, total.pixels)
	}
	fmt.Fprintln(w, "transparency scope: 10 skills x 3 states; standard RGB565/RGB555 channel truncation only")
	fmt.Fprintf(w, "transparency patches %d pixels %d pure-black %d nonblack-zero-rgb565 %d nonblack-zero-rgb555 %d\n",
		total.patches, total.pixels, total.black, total.nonblackZero565, total.nonblackZero555)
	fmt.Fprintln(w, "schoolcheck: transparency census complete")
	return nil
}

// score counts the patch's non-black pixels that agree with dst at offset
// (ox,oy), and the non-black population they are counted out of.
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

// correlation is one patch's best placement in one destination and the best
// fraction anywhere at least separation pixels away from it.
type correlation struct {
	at       image.Point
	fraction float64
	next     float64
	nextAt   image.Point
	total    int
}

const separation = 4

// correlate searches every offset at which patch fits inside dst.
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

// maskBoxes answers each colour code's bounding box in mask coordinates.
func maskBoxes(mask *image.Paletted) map[uint8]image.Rectangle {
	out := map[uint8]image.Rectangle{}
	b := mask.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			c := mask.ColorIndexAt(x, y)
			p := image.Rect(x, y, x+1, y+1)
			if r, ok := out[c]; ok {
				out[c] = r.Union(p)
			} else {
				out[c] = p
			}
		}
	}
	return out
}

func run(args []string, w io.Writer) error {
	fs := flag.NewFlagSet("schoolcheck", flag.ContinueOnError)
	assets := fs.String("assets", "", "lawful game install root (or AGAINROM_ASSETS)")
	pngDir := fs.String("png", "", "directory outside the repository for the composed school surfaces")
	transparencyOnly := fs.Bool("transparency-only", false, "read-only census of all 30 skill patches; skip correlation and game driving")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *transparencyOnly && (*pngDir != "" || fs.NArg() != 0) {
		return fmt.Errorf("-transparency-only accepts no -png output or positional patch selection")
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
	if *transparencyOnly {
		return censusTransparency(src, w)
	}
	fail := false

	hall, err := readPic(src, "graphics/interface/training/trnhall.bmp")
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "trnhall.bmp %dx%d black %d\n", hall.w(), hall.h(), hall.black)

	frames := make([]pic, 16)
	for i := range frames {
		if frames[i], err = readPic(src, fmt.Sprintf("%srt%04d.bmp", columnPrefix, i)); err != nil {
			return err
		}
	}
	fmt.Fprintf(w, "column frames 16, each %dx%d\n", frames[0].w(), frames[0].h())

	// The face origin comes from the frame the background bakes. Both rest
	// frames are searched over the whole background, so the winner and its
	// separation are measured rather than selected.
	var faceAt image.Point
	for _, i := range restFrame {
		c := correlate(hall.img, frames[i].img)
		fmt.Fprintf(w, "face rt%04d best %v frac %.4f next %.4f at %v over %d px\n",
			i, c.at, c.fraction, c.next, c.nextAt, c.total)
		if i == restFrame[1] {
			faceAt = c.at
		}
	}
	fmt.Fprintf(w, "face origin %v\n", faceAt)
	for i := range frames {
		m, t := score(hall.img, frames[i].img, faceAt.X, faceAt.Y)
		fmt.Fprintf(w, "frame rt%04d at origin frac %.4f (%d/%d) black %d\n",
			i, float64(m)/float64(t), m, t, frames[i].black)
	}

	// The build draws the class's own frame over the baked one, so what the
	// two disagree about at the established origin is what a player sees
	// change on the mage face. The bounding box says whether the disagreement
	// is scattered over the whole frame or concentrated in one region.
	for _, i := range restFrame {
		n, box, worst := disagreement(hall.img, frames[i].img, faceAt)
		fmt.Fprintf(w, "overdraw rt%04d differs in %d of %d px, box %v, worst channel sum %d\n",
			i, n, frames[i].w()*frames[i].h(), box, worst)
	}

	// Instrument A, per class, per skill, per state.
	var rect [2][5]image.Rectangle
	for class := 0; class < 2; class++ {
		face := frames[restFrame[class]]
		for slot := 0; slot < 5; slot++ {
			var at image.Point
			for state := 0; state < 3; state++ {
				addr := fmt.Sprintf("%s%s/%s/%s.bmp", columnPrefix, className[class], skillName[class][slot], stateName[state])
				p, err := readPic(src, addr)
				if err != nil {
					return err
				}
				c := correlate(face.img, p.img)
				fmt.Fprintf(w, "%s %s %s %dx%d black %d (%.4f) at %v frac %.4f next %.4f at %v\n",
					className[class], skillName[class][slot], stateName[state], p.w(), p.h(),
					p.black, float64(p.black)/float64(p.w()*p.h()), c.at, c.fraction, c.next, c.nextAt)
				if state == 0 {
					at = c.at
					rect[class][slot] = image.Rect(0, 0, p.w(), p.h()).Add(at).Add(faceAt)
					// The cross assignment. Correlating a patch against the
					// frame this class is already assumed to rest at reports
					// the best offset on that frame whatever the frame is, so
					// it cannot answer which frame the patch belongs to. Every
					// patch is correlated against all sixteen frames here and
					// the winner must be restFrame[class].
					best, bestFrac, runnerUp := -1, -1.0, -1.0
					for i := range frames {
						f := correlate(frames[i].img, p.img).fraction
						if f > bestFrac {
							best, runnerUp, bestFrac = i, bestFrac, f
						} else if f > runnerUp {
							runnerUp = f
						}
					}
					mark := "ok"
					if best != restFrame[class] {
						mark, fail = "FAIL", true
					}
					fmt.Fprintf(w, "frame-of %s %s best rt%04d frac %.4f next %.4f rest rt%04d %s\n",
						className[class], skillName[class][slot], best, bestFrac, runnerUp, restFrame[class], mark)
				} else if c.at != at {
					fmt.Fprintf(w, "FAIL %s %s: state %s lands at %v, state on at %v\n",
						className[class], skillName[class][slot], stateName[state], c.at, at)
					fail = true
				}
			}
		}
	}
	for class := 0; class < 2; class++ {
		for slot := 0; slot < 5; slot++ {
			got := ui.SchoolSkillRect(class, slot)
			mark := "ok"
			if got != rect[class][slot] {
				mark, fail = "FAIL", true
			}
			fmt.Fprintf(w, "rect %s %s measured %v production %v %s\n",
				className[class], skillName[class][slot], rect[class][slot], got, mark)
		}
	}

	// Instrument B, and production's own hit test over the install's mask.
	art, err := game.LoadTownSchoolArt(game.RoomDescription(arc.Base.Profile), src)
	if err != nil {
		return err
	}
	cells := make([]ui.TownSurfaceCell, 10)
	for i := range cells {
		cells[i].Enabled = true
	}
	for class := 0; class < 2; class++ {
		panel := ui.SchoolPanelRect(class)
		boxes := maskBoxes(art.Masks[class])
		fmt.Fprintf(w, "%s panel %v mask %v codes %d\n", className[class], panel,
			art.Masks[class].Bounds().Size(), len(boxes))
		v := ui.TownSurfaceView{Kind: ui.TownSurfaceSchool, SchoolArt: art, SchoolClass: class, Cells: cells}
		taken := [5]int{-1, -1, -1, -1, -1}
		for ci, code := range maskCodes {
			box, ok := boxes[code]
			if !ok {
				fmt.Fprintf(w, "FAIL %s mask code 0x%02x absent\n", className[class], code)
				fail = true
				continue
			}
			screen := box.Sub(art.Masks[class].Bounds().Min).Add(panel.Min)
			owner, share := -1, 0.0
			for slot := 0; slot < 5; slot++ {
				in := screen.Intersect(rect[class][slot])
				if in.Empty() {
					continue
				}
				small := area(screen)
				if a := area(rect[class][slot]); a < small {
					small = a
				}
				if f := float64(area(in)) / float64(small); f > share {
					owner, share = slot, f
				}
			}
			mid := image.Pt((screen.Min.X+screen.Max.X)/2, (screen.Min.Y+screen.Max.Y)/2)
			hit, hitOK := ui.TownSurfaceControlAt(v, mid)
			got := -1
			if hitOK && hit.Kind == ui.TownSurfaceControlCell {
				got = hit.Index - class*5
			}
			mark := "ok"
			// minShare keeps a chance overlap from being read as ownership;
			// the ten measured shares are all above 0.92.
			const minShare = 0.75
			if owner < 0 || share < minShare || got != owner {
				mark, fail = "FAIL", true
			}
			if owner >= 0 {
				if taken[owner] >= 0 {
					fmt.Fprintf(w, "FAIL %s codes 0x%02x and 0x%02x both claim %s\n",
						className[class], maskCodes[taken[owner]], code, skillName[class][owner])
					fail = true
				}
				taken[owner] = ci
			}
			fmt.Fprintf(w, "mask %s 0x%02x box %v screen %v overlaps %s %.4f hit-test %s %s\n",
				className[class], code, box, screen, name(class, owner), share, name(class, got), mark)
		}
	}

	// The game side, driven through the production town screen.
	if bad, err := driveSchoolRoom(root, w); err != nil {
		return err
	} else if bad {
		fail = true
	}

	if *pngDir != "" {
		if err := writeSurfaces(root, art, *pngDir, w); err != nil {
			return err
		}
	}
	if fail {
		return fmt.Errorf("the two instruments disagree; see the FAIL lines above")
	}
	fmt.Fprintln(w, "schoolcheck: ok")
	return nil
}

// disagreement counts the pixels at which dst and patch differ by more than
// matchTolerance when patch is placed at at, and answers their bounding box and
// the largest channel-sum difference seen.
func disagreement(dst, patch *image.RGBA, at image.Point) (int, image.Rectangle, int) {
	var box image.Rectangle
	n, worst := 0, 0
	for y := 0; y < patch.Bounds().Dy(); y++ {
		for x := 0; x < patch.Bounds().Dx(); x++ {
			p := patch.PixOffset(x, y)
			o := (at.Y+y)*dst.Stride + (at.X+x)*4
			s := diff(patch.Pix[p], dst.Pix[o]) + diff(patch.Pix[p+1], dst.Pix[o+1]) +
				diff(patch.Pix[p+2], dst.Pix[o+2])
			if s <= matchTolerance {
				continue
			}
			n++
			if s > worst {
				worst = s
			}
			r := image.Rect(x, y, x+1, y+1)
			if box.Empty() {
				box = r
			} else {
				box = box.Union(r)
			}
		}
	}
	return n, box, worst
}

func area(r image.Rectangle) int { return r.Dx() * r.Dy() }

func name(class, slot int) string {
	if slot < 0 || slot >= 5 {
		return "none"
	}
	return skillName[class][slot]
}

// writeSurfaces composes the production school surface for each class and
// writes it as a PNG, so the screen the owner reported on can be looked at
// without launching the game.
func writeSurfaces(root string, art *ui.TownSchoolArt, dir string, w io.Writer) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	f, err := game.NewFrontEnd(root)
	if err != nil {
		return err
	}
	for class := 0; class < 2; class++ {
		cells := make([]ui.TownSurfaceCell, 10)
		for c := 0; c < 2; c++ {
			for slot := 0; slot < 5; slot++ {
				cells[c*5+slot] = ui.TownSurfaceCell{Label: skillName[c][slot], Enabled: c == class}
			}
		}
		cells[class*5].Selected = true
		v := ui.TownSurfaceView{Kind: ui.TownSurfaceSchool, Cells: cells,
			SchoolArt: art, SchoolClass: class, HoverCell: class*5 + 1, Font: f.Font.Value()}
		// 1017: the school draws two buttons, not three.
		v.Buttons = []ui.TownSurfaceButton{
			{Label: f.Words.SchoolTrain, Value: "200", Enabled: true},
			{Label: f.Words.SchoolExit, Value: "500", Enabled: true},
		}
		out, err := os.Create(filepath.Join(dir, "school-"+className[class]+".png"))
		if err != nil {
			return err
		}
		if err := png.Encode(out, ui.ComposeTownSurface(v)); err != nil {
			out.Close()
			return err
		}
		if err := out.Close(); err != nil {
			return err
		}
		fmt.Fprintf(w, "png %s\n", out.Name())
	}
	return nil
}

// driveSchoolRoom opens a real campaign town through the production front end,
// walks into the school, and clicks the CENTRE OF EACH DRAWN SKILL RECTANGLE
// through the production hit test. It prints which skill the game then reports
// as selected and what it quotes for it.
//
// THE POINT COMES FROM THE DRAW SIDE AND THE ANSWER FROM THE MASK, which is
// what makes this a cross-check and not a tautology. ui.SchoolSkillRect says
// where the icon is painted; ui.TownSurfaceControlAt reads the install's own
// mask.bmp at that pixel; the game side turns the cell it answers into a hero
// skill and a price. A click that lands on the icon a player is looking at and
// selects a different skill is exactly the defect 1015 was opened for, and this
// is where the whole chain is measured rather than any one link of it.
//
// It drives whichever classes the started campaign's own party contains, and
// prints which those were. The class-specific half of the chain, the mask, is
// measured for both classes above and does not depend on this.
func driveSchoolRoom(root string, w io.Writer) (bool, error) {
	f, err := game.NewFrontEnd(root)
	if err != nil {
		return false, err
	}
	if f.Town == nil {
		return false, fmt.Errorf("drive: the front end started with no town")
	}
	f.Town.Arrive()
	screen := f.TownScreen()
	entered := false
	for i, row := range screen.Rows() {
		if row.Choosable && strings.Contains(strings.ToUpper(row.Text), "SCHOOL") {
			screen.Choose(i)
			entered = true
			break
		}
	}
	if !entered {
		return false, fmt.Errorf("drive: the town square offers no school row")
	}
	surface, ok := screen.(ui.TownSurfaceScreen)
	if !ok || !surface.AtTownSurface() {
		return false, fmt.Errorf("drive: the school room did not open")
	}

	bad := false
	seen := map[int]bool{}
	for member := 0; member < len(f.Carried); member++ {
		v := surface.TownSurface()
		class := v.SchoolClass
		if class < 0 || class > 1 || seen[class] {
			surface.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlNext}, false)
			continue
		}
		seen[class] = true
		fmt.Fprintf(w, "drive %s member %d of %d\n", className[class], member+1, len(f.Carried))
		for slot := 0; slot < 5; slot++ {
			r := ui.SchoolSkillRect(class, slot)
			mid := image.Pt((r.Min.X+r.Max.X)/2, (r.Min.Y+r.Max.Y)/2)
			c, hit := ui.TownSurfaceControlAt(surface.TownSurface(), mid)
			if !hit || c.Kind != ui.TownSurfaceControlCell {
				fmt.Fprintf(w, "FAIL drive %s %s: the centre of its own rectangle %v answers no cell\n",
					className[class], skillName[class][slot], r)
				bad = true
				continue
			}
			surface.TownSurfaceClick(c, false)
			after := surface.TownSurface()
			got, label := -1, "none"
			for i, cell := range after.Cells {
				if cell.Selected {
					got, label = i-class*5, cell.Label
				}
			}
			mark := "ok"
			if got != slot {
				mark, bad = "FAIL", true
			}
			fmt.Fprintf(w, "drive %s click %v on %s selects %s (cell %d) Train %q %s\n",
				className[class], mid, skillName[class][slot], label, c.Index,
				after.Buttons[0].Value, mark)
		}
		// TOWN-138: a step that MOVES the picker discards the pending skill
		// and its quote. A party of one refuses the step and reports
		// "nobody else is with you", and no member changed, so nothing is
		// stale and nothing is due to be cleared. Asserting a reset there
		// would be asserting against a step that did not happen.
		if len(f.Carried) < 2 {
			fmt.Fprintf(w, "drive picker step not exercised: the started campaign's party has %d member(s)\n",
				len(f.Carried))
			continue
		}
		surface.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlNext}, false)
		after := surface.TownSurface()
		still := -1
		for i, cell := range after.Cells {
			if cell.Selected {
				still = i
			}
		}
		mark := "ok"
		if still >= 0 || after.Buttons[0].Value != "0" || after.Buttons[0].Enabled {
			mark, bad = "FAIL", true
		}
		fmt.Fprintf(w, "drive picker step leaves selected cell %d Train %q enabled %v %s\n",
			still, after.Buttons[0].Value, after.Buttons[0].Enabled, mark)
	}
	if len(seen) == 0 {
		return bad, fmt.Errorf("drive: the started campaign's party owns no school panel")
	}
	return bad, nil
}
