// Command tippanelcheck measures the floating tip panel's own five rects
// against a lawful game install.
//
//	tippanelcheck [-assets DIR]
//
// WHAT IT MEASURES, per screen: the panel's own rect as production currently
// builds it; the minimum height that draws that screen's own shipped text
// whole at the rect's researched width (ui.TipPanelFits, searched upward
// from a small floor); the spare rows the current rect carries past that
// minimum; and how many of the current rect's own pixels sit over a control
// the room's own hit test would otherwise reach — the square scene's
// ControlAt for the town square's mask, TownSurfaceControlAt for the school and tavern's
// cells and button wells, ui.PreCreateControlAt for the generator's
// portraits, ShopControlAt for the shop.
//
// WHY A COMMAND AND NOT A TEST. Golden rule 2 forbids a Go test from reading
// an install, and every figure here is a property of installed art and
// text. Round 3 of this story's adversarial review found three committed
// documents quoting numbers from a scratch program deleted with its own
// worktree; this is the committed replacement, on cmd/buttonframecheck's and
// cmd/worldmapcheck's own precedent.
package main

import (
	"flag"
	"fmt"
	"image"
	"io"
	"os"
	"sort"

	"againrom/pkg/game"
	"againrom/pkg/ui"
)

var frame = image.Rect(0, 0, 640, 480)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "tippanelcheck:", err)
		os.Exit(1)
	}
}

func run(args []string, w io.Writer) error {
	fs := flag.NewFlagSet("tippanelcheck", flag.ContinueOnError)
	assets := fs.String("assets", "", "lawful game install root (or AGAINROM_ASSETS)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	root := game.ResolveAssetRoot(*assets, os.Getenv("AGAINROM_ASSETS"))
	if root == "" {
		return fmt.Errorf("no asset root: pass -assets or set AGAINROM_ASSETS")
	}
	f, err := game.NewFrontEnd(root)
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "root %s\n", root)

	scr := f.TownScreen()

	sq, ok := scr.(ui.TownSquareArtScreen)
	if !ok {
		return fmt.Errorf("TownScreen() does not implement TownSquareArtScreen")
	}
	sqv := sq.TownSquareView()
	if sqv.Scene == nil {
		return fmt.Errorf("town square art did not resolve from %s", root)
	}
	sqRect := sqv.Tip.Rect
	report(w, "town square", sqv.Tip, sqRect, topAnchor(sqRect.Min.X, sqRect.Min.Y, sqRect.Max.X), func(p image.Point) (string, bool) {
		c, hit := sqv.Scene.ControlAt(p)
		if !hit {
			return "", false
		}
		return squareLabel(c), true
	})

	// surface reports against v.Tip.Rect itself: production keeps the
	// room's description rectangle and moves only its bottom edge when the
	// text overflows it, so an anchor built from the resolved rect's own
	// corners is still a top anchor.
	surface := func(door int, name string) error {
		scr.Back()
		scr.Choose(door)
		s, ok := scr.(ui.TownSurfaceScreen)
		if !ok || !s.AtTownSurface() {
			return fmt.Errorf("%s: Choose(%d) did not open the surface room", name, door)
		}
		v := s.TownSurface()
		rect := v.Tip.Rect
		report(w, name, v.Tip, rect, topAnchor(rect.Min.X, rect.Min.Y, rect.Max.X), func(p image.Point) (string, bool) {
			c, hit := ui.TownSurfaceControlAt(v, p)
			if !hit {
				return "", false
			}
			return surfaceLabel(c), true
		})
		return nil
	}
	if err := surface(0, "tavern"); err != nil {
		return err
	}
	if err := surface(2, "school"); err != nil {
		return err
	}

	scr.Back()
	scr.Choose(1) // shop
	shop, ok := scr.(ui.TownShopScreen)
	if !ok || !shop.AtTownShop() {
		return fmt.Errorf("shop: Choose(1) did not open the shop room")
	}
	shopView := shop.ShopScreen()
	// shopRect reads shopView.TipPanel.Rect, not ui.ShopTipRect() (round-2
	// adversarial review, owner item "shop tip too tall"): pkg/game/shopview.go
	// now shrinks the shop's tip the same way the surface rooms already do
	// (see the comment on the surface() closure above), so the raw package
	// constant no longer matches what production draws and hit-tests.
	shopRect := shopView.TipPanel.Rect
	report(w, "shop", shopView.TipPanel, shopRect, topAnchor(shopRect.Min.X, shopRect.Min.Y, shopRect.Max.X), func(p image.Point) (string, bool) {
		c, hit := ui.ShopControlAt(p)
		if !hit {
			return "", false
		}
		return shopLabel(c), true
	})

	setup := f.ChargenSetup()
	c := ui.NewChargen(setup)
	pre := c.TipPanel()
	report(w, "chargen pre-create", pre, pre.Rect, topAnchor(pre.Rect.Min.X, pre.Rect.Min.Y, pre.Rect.Max.X), func(p image.Point) (string, bool) {
		return ui.PreCreateControlAt(c, p)
	})
	// The detailed page's popup carries the class text the pre-create choice
	// selects; fighter's and mage's own rects are reported apart because each
	// shrinks against its own text.
	detailed := func(choice int) ui.TipPanelView {
		c.SelectPreChoice(choice)
		c.Forward()
		v := c.TipPanel()
		c.Back()
		return v
	}
	fighter, mage := detailed(0), detailed(1)
	c.SelectPreChoice(0)
	chosen := fighter.Rect
	fighterRectAt := bottomAnchor(fighter.Rect.Min.X, fighter.Rect.Max.X, fighter.Rect.Max.Y)
	mageRectAt := bottomAnchor(mage.Rect.Min.X, mage.Rect.Max.X, mage.Rect.Max.Y)
	minFighter := searchHeight(fighter, fighterRectAt)
	minMage := searchHeight(mage, mageRectAt)
	chargenID := func(p image.Point) (string, bool) { return ui.PreCreateControlAt(c, p) }
	live := countLive(chosen, func(p image.Point) bool { _, hit := chargenID(p); return hit })
	fmt.Fprintf(w, "chargen fighter: rect=%v min-height=%d chosen-height=%d spare=%d live-pixels=%d\n",
		fighter.Rect, minFighter, fighter.Rect.Dy(), fighter.Rect.Dy()-minFighter, live)
	printCoverage(w, chosen, chargenID)
	fmt.Fprintf(w, "chargen mage: rect=%v min-height=%d chosen-height=%d spare=%d\n",
		mage.Rect, minMage, mage.Rect.Dy(), mage.Rect.Dy()-minMage)

	return nil
}

// topAnchor returns a candidate-rect function for a rect whose top edge and
// left/right edges are fixed and only the bottom edge moves.
func topAnchor(x0, y0, x1 int) func(h int) image.Rectangle {
	return func(h int) image.Rectangle { return image.Rect(x0, y0, x1, y0+h) }
}

// bottomAnchor returns a candidate-rect function for a rect whose bottom
// edge and left/right edges are fixed (the generator's own rect, anchored
// at the screen's own 480) and only the top edge moves.
func bottomAnchor(x0, x1, y1 int) func(h int) image.Rectangle {
	return func(h int) image.Rectangle { return image.Rect(x0, y1-h, x1, y1) }
}

// searchHeight is the minimum height, from a 20px floor up to the screen's
// own 480, at which ui.TipPanelFits(v with Rect replaced) first reports
// true. v's own Text, Font and Art are read from the real install; only
// Rect is replaced by each candidate this function builds.
func searchHeight(v ui.TipPanelView, rectAt func(h int) image.Rectangle) int {
	for h := 20; h <= 480; h++ {
		cand := v
		cand.Rect = rectAt(h)
		if ui.TipPanelFits(cand) {
			return h
		}
	}
	return -1
}

// countLive is how many of r's own pixels (clipped to the 640x480 frame)
// hit answers true for.
func countLive(r image.Rectangle, hit func(image.Point) bool) int {
	r = r.Intersect(frame)
	n := 0
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if hit(image.Pt(x, y)) {
				n++
			}
		}
	}
	return n
}

func report(w io.Writer, name string, v ui.TipPanelView, chosen image.Rectangle, rectAt func(h int) image.Rectangle, id func(image.Point) (string, bool)) {
	minH := searchHeight(v, rectAt)
	live := countLive(chosen, func(p image.Point) bool { _, hit := id(p); return hit })
	fmt.Fprintf(w, "%s rect=%v min-height=%d chosen-height=%d spare=%d live-pixels=%d showing=%v\n",
		name, chosen, minH, chosen.Dy(), chosen.Dy()-minH, live, v.Showing())
	printCoverage(w, chosen, id)
}

// printCoverage lists, per control the room's own hit test names, how much of
// that control the panel's rect covers.
//
// WHY PER CONTROL AND NOT ONE TOTAL (round-3 adversarial review, D-2): a
// live-pixel total is stated in the format's own vocabulary. "Two of the
// school's three skill cells are completely covered while the tip shows" is
// the sentence a reader planning around this can evaluate, and it is what
// docs/DIVERGENCES.md DIV-162 now quotes.
//
// THE FIGURES ARE THIS GAME STATE'S, NOT THE SCREEN'S. Each room is opened
// from a fresh front end with no campaign loaded, so a control a room
// disables answers no hit and is absent here. The counts are a lower bound on
// what a mid-campaign player would see covered, not a fixed property.
func printCoverage(w io.Writer, r image.Rectangle, id func(image.Point) (string, bool)) {
	covered := countByLabel(r, id)
	total := countByLabel(frame, id)
	labels := make([]string, 0, len(covered))
	for label := range covered {
		labels = append(labels, label)
	}
	sort.Slice(labels, func(i, j int) bool {
		fi := float64(covered[labels[i]]) / float64(total[labels[i]])
		fj := float64(covered[labels[j]]) / float64(total[labels[j]])
		if fi != fj {
			return fi > fj
		}
		return labels[i] < labels[j]
	})
	for _, label := range labels {
		fmt.Fprintf(w, "  covers %s %.1f%% (%d of %d px)\n",
			label, 100*float64(covered[label])/float64(total[label]), covered[label], total[label])
	}
	if len(labels) == 0 {
		fmt.Fprintln(w, "  covers no control this room's own hit test names")
	}
}

// countByLabel is how many of r's own pixels (clipped to the 640x480 frame)
// each named control answers for.
func countByLabel(r image.Rectangle, id func(image.Point) (string, bool)) map[string]int {
	r = r.Intersect(frame)
	n := map[string]int{}
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if label, hit := id(image.Pt(x, y)); hit {
				n[label]++
			}
		}
	}
	return n
}

// squareLabel, surfaceLabel and shopLabel name one control of their own room.
// The kind names are this tool's own words for the room's own constants; the
// index is the constant's own.
func squareLabel(c ui.TownSquareControl) string {
	if c.Kind == ui.TownSquareControlDoor {
		return fmt.Sprintf("door %d", c.Door)
	}
	return "statue/menu"
}

func surfaceLabel(c ui.TownSurfaceControl) string {
	switch c.Kind {
	case ui.TownSurfaceControlButton:
		return fmt.Sprintf("button %d", c.Index)
	case ui.TownSurfaceControlCell:
		return fmt.Sprintf("cell %d", c.Index)
	case ui.TownSurfaceControlPrevious:
		return "previous"
	case ui.TownSurfaceControlNext:
		return "next"
	case ui.TownSurfaceControlMode:
		return "mode"
	default:
		return fmt.Sprintf("kind %d index %d", int(c.Kind), c.Index)
	}
}

func shopLabel(c ui.ShopControl) string {
	switch c.Kind {
	case ui.ShopControlButton:
		return fmt.Sprintf("button %d", c.Index)
	case ui.ShopControlArrowUp:
		return "arrow up"
	case ui.ShopControlArrowDown:
		return "arrow down"
	case ui.ShopControlShelfCell:
		return fmt.Sprintf("shelf cell %d", c.Index)
	case ui.ShopControlTableCell:
		return fmt.Sprintf("table cell %d", c.Index)
	case ui.ShopControlPackCell:
		return fmt.Sprintf("pack cell %d", c.Index)
	case ui.ShopControlShelfPick:
		return fmt.Sprintf("shelf pick %d", c.Index)
	case ui.ShopControlMerchant:
		return "merchant"
	case ui.ShopControlPickerPrev:
		return "picker prev"
	case ui.ShopControlPickerNext:
		return "picker next"
	case ui.ShopControlCharacterMode:
		return "character mode"
	case ui.ShopControlBook:
		return "book"
	case ui.ShopControlDoll:
		return fmt.Sprintf("doll slot %d", c.Index)
	default:
		return fmt.Sprintf("kind %d index %d", int(c.Kind), c.Index)
	}
}
