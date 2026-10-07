// Command plaquescreens composes the four screens the owner reported against
// a lawful install through production code paths, and writes each as a PNG
// so the fix can be looked at without launching the game.
//
//	plaquescreens [-assets DIR] -png DIR
//
// School and tavern are composed through ui.ComposeTownSurface with the
// production art (game.NewFrontEnd's TownSchoolArt/TownTavernArt), the same
// entry point cmd/schoolcheck already uses for the school. The character
// generator is composed through ui.ComposeChargenFrame after selecting a
// pre-create choice and moving to the detailed stage, the same two calls the
// shell itself makes. The shop is composed through ui.ComposeShopScreen
// after a real campaign session opens the town and the square's own SHOP
// door is pressed, the same path cmd/shopdump uses.
//
// It writes no game data, only PNGs under the caller's own -png directory.
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
		fmt.Fprintln(os.Stderr, "plaquescreens:", err)
		os.Exit(1)
	}
}

func run() error {
	assets := flag.String("assets", "", "game install root (or AGAINROM_ASSETS)")
	pngDir := flag.String("png", "", "directory to write the four PNGs into")
	flag.Parse()

	root := game.ResolveAssetRoot(*assets, os.Getenv("AGAINROM_ASSETS"))
	if root == "" {
		return fmt.Errorf("no asset root: pass -assets or set AGAINROM_ASSETS")
	}
	if *pngDir == "" {
		return fmt.Errorf("-png is required")
	}
	if err := os.MkdirAll(*pngDir, 0o755); err != nil {
		return err
	}

	f, err := game.NewFrontEnd(root)
	if err != nil {
		return err
	}

	if err := writeSchool(f, *pngDir); err != nil {
		return fmt.Errorf("school: %w", err)
	}
	if err := writeTavern(f, *pngDir); err != nil {
		return fmt.Errorf("tavern: %w", err)
	}
	if err := writeChargen(f, *pngDir, 0, "chargen.png"); err != nil {
		return fmt.Errorf("chargen: %w", err)
	}
	// The mage detailed page too (round-2 adversarial review item 5): every
	// prior chargen render here and every chargen_release_test.go case
	// composed SelectPreChoice(0), the fighter, only — the blindness that
	// let the mage-only fire-icon overlap (owner item 4) survive two
	// review rounds with every other gate green.
	if err := writeChargen(f, *pngDir, 1, "chargen-mage.png"); err != nil {
		return fmt.Errorf("chargen (mage): %w", err)
	}
	if err := writeShop(f, *pngDir); err != nil {
		return fmt.Errorf("shop: %w", err)
	}
	fmt.Println("wrote 4 PNGs under", *pngDir)
	return nil
}

func writeSchool(f *game.FrontEnd, dir string) error {
	if f.TownSchoolArt.Value() == nil {
		return fmt.Errorf("no school art (%v)", f.TownSchoolArt.Err())
	}
	cells := make([]ui.TownSurfaceCell, 10)
	for c := 0; c < 2; c++ {
		for slot := 0; slot < 5; slot++ {
			cells[c*5+slot] = ui.TownSurfaceCell{Label: "skill", Enabled: c == 0}
		}
	}
	cells[0].Selected = true
	panes := characterPanes(f)
	v := ui.TownSurfaceView{
		Kind: ui.TownSurfaceSchool, Cells: cells,
		SchoolArt: f.TownSchoolArt.Value(), SchoolClass: 0, HoverCell: 1, Font: f.Font.Value(),
		Buttons: []ui.TownSurfaceButton{{Label: f.Words.SchoolTrain, Value: "200", Enabled: true}, {Label: f.Words.SchoolExit, Value: "500", Enabled: true}},
		Hero:    ui.TownCharacterView{FigurePane: panes.Figure, StatsPane: panes.Stats},
	}
	return writePNG(filepath.Join(dir, "school.png"), ui.ComposeTownSurface(v))
}

func writeTavern(f *game.FrontEnd, dir string) error {
	if f.TownTavernArt.Value() == nil {
		return fmt.Errorf("no tavern art (%v)", f.TownTavernArt.Err())
	}
	cells := []ui.TownSurfaceCell{{Label: "mercenary 1", Enabled: true}, {Label: "mercenary 2", Enabled: true}}
	panes := characterPanes(f)
	v := ui.TownSurfaceView{
		Kind: ui.TownSurfaceTavern, Title: "TAVERN", Cells: cells,
		TavernArt: f.TownTavernArt.Value(), SchoolClass: -1, HoverCell: -1, Font: f.Font.Value(),
		Buttons: []ui.TownSurfaceButton{
			{Label: "Sleep", Enabled: true},
			{Label: f.Words.TavernHire, Value: "100", Enabled: true},
			{Label: f.Words.TavernTalk, Enabled: true},
			{Label: f.Words.TavernExit, Enabled: true},
		},
		Hero: ui.TownCharacterView{FigurePane: panes.Figure, StatsPane: panes.Stats},
	}
	return writePNG(filepath.Join(dir, "tavern.png"), ui.ComposeTownSurface(v))
}

func writeChargen(f *game.FrontEnd, dir string, class int, name string) error {
	c := ui.NewChargen(f.ChargenSetup())
	c.SelectPreChoice(class)
	c.Forward()
	if c.Stage() != ui.DetailedStage {
		return fmt.Errorf("Forward did not reach the detailed stage (stage=%v)", c.Stage())
	}
	return writePNG(filepath.Join(dir, name), ui.ComposeChargenFrame(c))
}

func writeShop(f *game.FrontEnd, dir string) error {
	n, err := openTown(f)
	if err != nil {
		return err
	}
	_ = n
	screen := f.TownScreen()
	if err := enterShop(screen); err != nil {
		return err
	}
	shop, ok := screen.(ui.TownShopScreen)
	if !ok || !shop.AtTownShop() {
		return fmt.Errorf("shop room did not open")
	}
	v := shop.ShopScreen()
	pic := ui.ComposeShopScreen(v, image.Point{}, false, nil, false)
	return writePNG(filepath.Join(dir, "shop.png"), pic)
}

// openTown finishes the first campaign mission whose successor the registry
// marks as offering the town — the same derivation cmd/shopdump uses.
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
	for i := 0; i < 32; i++ {
		if _, shown := d.TownDialogue(); !shown {
			return nil
		}
		d.AdvanceTownDialogue()
	}
	return fmt.Errorf("the merchant's greeting did not end")
}

func characterPanes(f *game.FrontEnd) game.TownCharacterPaneArt {
	if f == nil || f.Archives == nil {
		return game.TownCharacterPaneArt{}
	}
	p, err := game.LoadTownCharacterPaneArt(f.Archives.Containers)
	if err != nil {
		return game.TownCharacterPaneArt{}
	}
	return p
}

func writePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}
