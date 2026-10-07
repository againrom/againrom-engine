// Command shopdump composes the town shop screen against a lawful game install
// and reports what it painted.
//
//	shopdump [-assets DIR] [-shelf N] [-steps N] [-hover X,Y] [-png DIR]
//
// It is the measuring instrument for 0157 round 3. The story's promised result
// is what the screen looks like, and a synthetic test cannot read the install's
// own pictures (golden rule 2), so the pixel evidence comes from a developer
// tool and its output is what closure.md records.
//
// THE SESSION IS A REAL CAMPAIGN SESSION. The front end is built by
// game.NewFrontEnd, the town is opened by finishing the campaign mission the
// registry marks as offering it, and the shop room is entered by pressing the
// square's own SHOP door through ui.TownScreen.Choose. Nothing here constructs
// a shop, a party or a stock list of its own.
//
// It writes no game data. The PNG it can emit is the composed screen — the
// install's pictures arranged by this build's own code — and it goes to a
// directory the caller names, outside the repo.
package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"againrom/pkg/game"
	"againrom/pkg/ui"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "shopdump:", err)
		os.Exit(1)
	}
}

func run() error {
	assets := flag.String("assets", "", "game install root (or AGAINROM_ASSETS)")
	shelf := flag.Int("shelf", 1, "which of the four room shelves to open, -1 for none")
	steps := flag.Int("steps", 0, "how many times to press the party picker's next arrow")
	hover := flag.String("hover", "", "pointer position X,Y for the characteristics box")
	pngDir := flag.String("png", "", "directory to write shop.png into")
	flag.Parse()

	root := game.ResolveAssetRoot(*assets, os.Getenv("AGAINROM_ASSETS"))
	if root == "" {
		return fmt.Errorf("no asset root: pass -assets or set AGAINROM_ASSETS")
	}
	f, err := game.NewFrontEnd(root)
	if err != nil {
		return err
	}

	town, err := openTown(f)
	if err != nil {
		return err
	}
	fmt.Printf("town opened after mission %d, chapter %d, purse %d\n", town, f.Town.Chapter(), f.Town.Gold())

	screen := f.TownScreen()
	if err := enterShop(screen); err != nil {
		return err
	}
	shop, ok := screen.(ui.TownShopScreen)
	if !ok {
		return fmt.Errorf("the town screen carries no shop seam")
	}
	if !shop.AtTownShop() {
		return fmt.Errorf("the shop room did not open")
	}
	msg := ""
	if *shelf >= 0 {
		msg = shop.ShopClick(ui.ShopControl{Kind: ui.ShopControlShelfPick, Index: *shelf}).Msg
	}
	for i := 0; i < *steps; i++ {
		msg = shop.ShopClick(ui.ShopControl{Kind: ui.ShopControlPickerNext}).Msg
	}

	v := shop.ShopScreen()
	// The answer to the last press, carried exactly as pkg/ui's own
	// applyTownAction carries it into the next composed frame.
	v.Msg = msg
	fmt.Printf("message: %q\n", msg)
	report(v)

	at, hasHover := image.Point{}, false
	if *hover != "" {
		if at, err = parsePoint(*hover); err != nil {
			return err
		}
		hasHover = true
		if lines, ok := ui.ShopHoverLines(v, at); ok {
			fmt.Printf("hover %v: %s\n", at, strings.Join(lines, " | "))
			for _, s := range lines {
				fmt.Printf("  bytes %X\n", s)
			}
		} else {
			fmt.Printf("hover %v: nothing\n", at)
		}
	}

	pic := ui.ComposeShopScreen(v, at, hasHover, nil, false)
	fmt.Printf("black pixels: %s\n", blackReport(pic))
	if *pngDir != "" {
		out := filepath.Join(*pngDir, "shop.png")
		if err := writePNG(out, pic); err != nil {
			return err
		}
		fmt.Println("wrote", out)
	}
	return nil
}

// openTown finishes the first campaign mission whose successor the registry
// marks as offering the town, which is the one path FrontEnd opens a town on.
func openTown(f *game.FrontEnd) (int, error) {
	party := f.NextParty()
	for n := 1; n < 200; n++ {
		offer, ok := f.Campaign.Value().NextMission(n)
		if !ok || !offer.Town {
			continue
		}
		f.FinishMission(n, party, nil, nil)
		if f.Town == nil || f.Shop == nil {
			return 0, fmt.Errorf("mission %d declares a town and none opened", n)
		}
		return n, nil
	}
	return 0, fmt.Errorf("this install's campaign declares no town")
}

// enterShop presses the square's own SHOP door and pages the merchant's
// greeting to its end, which is where the shop room itself is standing.
func enterShop(t ui.TownScreen) error {
	pressed := false
	for i, row := range t.Rows() {
		if !row.Choosable || !strings.HasPrefix(strings.TrimSpace(row.Text), "SHOP") {
			continue
		}
		t.Choose(i)
		pressed = true
		break
	}
	if !pressed {
		return fmt.Errorf("the town square lists no shop door")
	}
	d, ok := t.(ui.TownDialogueScreen)
	if !ok {
		return nil
	}
	for n := 0; n < 32; n++ {
		if _, shown := d.TownDialogue(); !shown {
			return nil
		}
		d.AdvanceTownDialogue()
	}
	return fmt.Errorf("the merchant's greeting did not end")
}

// report prints the composed state: which member is shown, what each grid holds,
// and which background and plaque every occupied cell took.
func report(v ui.ShopScreenView) {
	name, _ := ui.PanelSubjectName(v.Character.Subject)
	fmt.Printf("member %d of %d: %s\n", v.Member+1, v.MemberCount, name)
	fmt.Printf("shelf %d (%s), purse %d buy %d sell %d total %d\n",
		v.Chosen, v.ShelfName, v.Purse, v.Buy, v.Sell, v.Total)
	fmt.Printf("tip: %d bytes, hex %X\n", len(v.TipPanel.Text), v.TipPanel.Text)
	fmt.Printf("figure: %s\n", picSize(v.Figure))
	printGrid("shelf", v.Shelf[:])
	printGrid("table", v.Table[:])
	printGrid("pack", v.Pack[:])
}

func printGrid(name string, cells []ui.ShopCell) {
	for i, c := range cells {
		if !c.Occupied() && !c.Money {
			continue
		}
		fmt.Printf("  %s[%d] back=%d money=%v count=%d price=%d icon=%s\n",
			name, i, c.Back, c.Money, c.Count, c.Price, picSize(c.Icon))
	}
}

func picSize(pic *image.RGBA) string {
	if pic == nil {
		return "none"
	}
	b := pic.Bounds()
	return fmt.Sprintf("%dx%d", b.Dx(), b.Dy())
}

// blackReport counts pure-black pixels over the whole screen and over each of
// the two regions the owner named, which is the number the round has to move.
func blackReport(pic *image.RGBA) string {
	regions := []struct {
		name string
		r    image.Rectangle
	}{
		{"screen", pic.Bounds()},
		{"buttons", image.Rect(464, 0, 640, 238)},
		{"pack", image.Rect(0, 390, 480, 480)},
		{"corner", image.Rect(480, 238, 640, 480)},
	}
	var out []string
	for _, reg := range regions {
		n, total := 0, 0
		for y := reg.r.Min.Y; y < reg.r.Max.Y; y++ {
			for x := reg.r.Min.X; x < reg.r.Max.X; x++ {
				c := pic.RGBAAt(x, y)
				total++
				if c.R < 16 && c.G < 16 && c.B < 16 {
					n++
				}
			}
		}
		out = append(out, fmt.Sprintf("%s %d/%d (%d%%)", reg.name, n, total, 100*n/total))
	}
	return strings.Join(out, ", ")
}

func parsePoint(s string) (image.Point, error) {
	var x, y int
	if _, err := fmt.Sscanf(s, "%d,%d", &x, &y); err != nil {
		return image.Point{}, fmt.Errorf("hover %q: want X,Y", s)
	}
	return image.Pt(x, y), nil
}

func writePNG(path string, pic *image.RGBA) error {
	fh, err := os.Create(path)
	if err != nil {
		return err
	}
	defer fh.Close()
	return png.Encode(fh, pic)
}
