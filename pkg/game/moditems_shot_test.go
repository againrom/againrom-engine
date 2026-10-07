package game

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/modrt"
	"againrom/pkg/ui"
)

// TestReleaseModItemShopScreenshot composes the merchant's armour shelf with the
// example mod's item selected, offscreen, and writes it under
// AGAINROM_SHOT_DIR, or a temporary directory when none is named.
func TestReleaseModItemShopScreenshot(t *testing.T) {
	dir := os.Getenv("AGAINROM_SHOT_DIR")
	if dir == "" {
		dir = t.TempDir()
	}
	f := modItemFront(t)
	party := f.NextParty()
	opened := false
	for n := 1; n < 200 && !opened; n++ {
		offer, ok := f.Campaign.Value().NextMission(n)
		if !ok || !offer.Town {
			continue
		}
		f.FinishMission(n, party, nil, nil)
		opened = f.Town != nil
	}
	if !opened {
		t.Fatal("no town opened")
	}
	s := f.TownScreen().(*townScreen)
	s.room, s.packBase = roomShop, 0
	s.chooseRoomShelf(roomArmour)
	i := slices.IndexFunc(f.Shop.Shelf(s.shopShelf), func(item ShopItem) bool { return uint16(item.Code) == modItemCode })
	if i < 0 {
		t.Fatal("the armour shelf does not list the mod item")
	}
	s.shopFromShelf(i, false)
	pix, err := ui.ComposeTownScreen(s, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	name := filepath.Join(dir, "shop-armour-"+f.ModSet().Base+".png")
	out, err := os.Create(name)
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	if err := png.Encode(out, pix); err != nil {
		t.Fatal(err)
	}
}

func TestReleaseModItemNameDrawsInTheInstallAlphabet(t *testing.T) {
	f := modItemFront(t)
	text := "Gambeson"
	if modrt.LanguageFor(f.ModSet().Base) == "ru" {
		text = "Поддоспешник"
	}
	want, err := encodeSaveLabel(text, f.textSelector())
	if err != nil {
		t.Fatal(err)
	}
	code := data.ItemCode(modItemCode)
	if got := itemName(code, f.Table); got != want {
		t.Fatalf("itemName = %q, want the install bytes %q", got, want)
	}
	lines := itemInfoLines(code, f.Table)
	if len(lines) == 0 || lines[0] != want {
		t.Fatalf("information lines %q, want the name %q first", lines, want)
	}
	bounds := image.Rect(0, 0, 640, 480)
	at := image.Pt(320, 240)
	pic, _, ok := ui.ComposeTooltipHint(lines, f.tipFont(), at, bounds)
	if !ok {
		t.Fatal("the popup does not compose")
	}
	writeModShot(t, "item-name-popup", pic)
	named, _, ok := ui.ComposeTooltipHint(append([]string{want}, lines[1:]...), f.tipFont(), at, bounds)
	if !ok || !bytes.Equal(pic.Pix, named.Pix) {
		t.Fatal("the popup is not the picture of the encoded name")
	}
	if text != want {
		utf8Lines := append([]string{text}, lines[1:]...)
		loss, _, ok := ui.ComposeTooltipHint(utf8Lines, f.tipFont(), at, bounds)
		if ok && loss.Bounds() == pic.Bounds() && bytes.Equal(loss.Pix, pic.Pix) {
			t.Fatal("the UTF-8 name draws the same as the install bytes; the witness cannot tell them apart")
		}
	}
}
