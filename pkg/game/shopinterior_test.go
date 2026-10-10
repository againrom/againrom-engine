package game

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"testing"
	"time"

	"againrom/pkg/sim"
	"againrom/pkg/town"
	"againrom/pkg/ui"
	"github.com/hajimehoshi/ebiten/v2"
)

func shopInteriorPixel(c color.RGBA) *image.RGBA {
	pic := image.NewRGBA(image.Rect(0, 0, 1, 1))
	pic.SetRGBA(0, 0, c)
	return pic
}

// The shop scene as the ROM1 description states it: its step clock, idle wait
// and draw range, frame counts and merchant states.
const (
	shopInteriorStep       = 100 * time.Millisecond
	shopInteriorIdleBase   = 5 * time.Second
	shopInteriorIdleRange  = 5
	shopRackFrameCount     = 11
	shopMerchantIdleCount  = 28
	shopMerchantReactCount = 11
	shopMerchantIdle       = uint8(0x10)
	shopMerchantYes        = uint8(0x20)
	shopMerchantNo         = uint8(0x40)
)

// shopMerchantModes are the merchant's states in description order.
var shopMerchantModes = []uint8{shopMerchantIdle, shopMerchantYes, shopMerchantNo}

func shopRacks(s *townScreen) *town.Selector { return s.shopPage().Actor("racks").(*town.Selector) }

func shopMerchant(s *townScreen) *town.Priority {
	return s.shopPage().Actor("merchant").(*town.Priority)
}

// shopModes answers the merchant's raised states as a bit set.
func shopModes(s *townScreen) uint8 {
	var modes uint8
	for i, raised := range shopMerchant(s).Raised {
		if raised {
			modes |= shopMerchantModes[i]
		}
	}
	return modes
}

func setShopModes(s *townScreen, modes uint8) {
	for i, bit := range shopMerchantModes {
		shopMerchant(s).Raised[i] = modes&bit != 0
	}
}

func shopRackFrames(rack int) []image.Image {
	var frames []image.Image
	for frame := 0; frame < shopRackFrameCount; frame++ {
		frames = append(frames, shopInteriorPixel(color.RGBA{R: uint8(10 + rack*20 + frame), A: 0xff}))
	}
	return frames
}

func shopInteriorTestArt() *ui.ShopScreenArt {
	art := &ui.ShopScreenArt{Scene: map[string][]image.Image{
		"merchant": {shopInteriorPixel(color.RGBA{R: 1, G: 2, B: 3, A: 0xff})},
	}}
	for rack := 0; rack < 4; rack++ {
		art.Scene[fmt.Sprintf("rack%d", rack)] = shopRackFrames(rack)
	}
	for frame := 0; frame < shopMerchantIdleCount; frame++ {
		art.Scene["idle"] = append(art.Scene["idle"], shopInteriorPixel(color.RGBA{G: uint8(20 + frame), A: 0xff}))
	}
	for frame := 0; frame < shopMerchantReactCount; frame++ {
		art.Scene["yes"] = append(art.Scene["yes"], shopInteriorPixel(color.RGBA{B: uint8(40 + frame), A: 0xff}))
		art.Scene["no"] = append(art.Scene["no"], shopInteriorPixel(color.RGBA{R: uint8(70 + frame), G: 1, A: 0xff}))
	}
	return art
}

func shopInteriorFixture(t *testing.T) (*FrontEnd, *ui.App, *townScreen, *time.Time) {
	t.Helper()
	f := shellFrontEnd()
	f.Shop = NewShop(0)
	f.Shop.shelves[ShelfArmour] = []ShopItem{{Count: 1}}
	f.shopArtCache = resolved(shopInteriorTestArt(), nil)
	now := time.Unix(200, 0)
	f.TownAnimationNow = func() time.Time { return now }
	f.ShopRandom = func(n int) int {
		if n != shopInteriorIdleRange {
			t.Fatalf("shop random range = %d", n)
		}
		return 0
	}
	a := f.App("1122-shop-interior")
	a.Layout(640, 480)
	a.SetSaveSeams(nil, func() []ui.SaveEntry { return []ui.SaveEntry{{Name: "town.ags", Label: "Town"}} }, func(string) (ui.MapOpener, bool, error) {
		f.townUI.resetForNewGame()
		return nil, true, nil
	})
	if err := a.HeadlessKey("load"); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessActivate("@first"); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessActivate("SHOP"); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	if f.townUI.room != roomShop || !f.townUI.shopPage().Active() {
		t.Fatalf("shop entry = room%d active%v", f.townUI.room, f.townUI.shopPage().Active())
	}
	f.townUI.CloseTip()
	return f, a, f.townUI, &now
}

func drawShopInteriorApp(t *testing.T, a *ui.App, s *townScreen, now *time.Time, elapsed time.Duration) *image.RGBA {
	t.Helper()
	*now = now.Add(elapsed)
	a.Draw(ebiten.NewImage(640, 480))
	// App.Draw has published the selected pictures into the read-only view.
	// Ebiten forbids target readback before RunGame in unit tests, so compose
	// that exact view on the CPU without delivering another animation step.
	return ui.ComposeShopScreen(s.ShopScreen(), image.Point{}, false, nil, false)
}

func TestShopInteriorLivePaintGateRackSwitchAndNoCatchUp(t *testing.T) {
	_, app, s, now := shopInteriorFixture(t)

	frame := drawShopInteriorApp(t, app, s, now, 0)
	if got := shopRacks(s).Index[0]; got != 1 {
		t.Fatalf("entry paint rack index = %d, want first eligible step 1", got)
	}
	if got := frame.RGBAAt(353, 108); got != (color.RGBA{R: 11, A: 0xff}) {
		t.Fatalf("entry rack pixel = %v, want file2 fixture", got)
	}
	if _, err := ui.ComposeTownScreen(s, "read-only capture"); err != nil {
		t.Fatal(err)
	}
	if got := shopRacks(s).Index[0]; got != 1 {
		t.Fatalf("exported read-only composition advanced rack to %d", got)
	}
	drawShopInteriorApp(t, app, s, now, 0)
	if got := shopRacks(s).Index[0]; got != 1 {
		t.Fatalf("sub100 duplicate paint advanced to %d", got)
	}
	drawShopInteriorApp(t, app, s, now, 100*time.Millisecond)
	if got := shopRacks(s).Index[0]; got != 2 {
		t.Fatalf("100ms equality did not admit exactly one step: %d", got)
	}
	drawShopInteriorApp(t, app, s, now, time.Hour)
	if got := shopRacks(s).Index[0]; got != 3 {
		t.Fatalf("long gap caught up instead of one step: %d", got)
	}
	shopRacks(s).Index[0] = 8
	drawShopInteriorApp(t, app, s, now, 100*time.Millisecond)
	if got := shopRacks(s).Index[0]; got != 3 {
		t.Fatalf("steady loop 8 advanced to %d, want 3", got)
	}

	setShopModes(s, 0)
	shopMerchant(s).Index = 0
	s.shopPage().SetClock("idle", *now)
	if action := s.ShopClick(ui.ShopControl{Kind: ui.ShopControlShelfPick, Index: 1}); action.Msg != "" {
		t.Fatal("rack click posted a shelf count caption")
	}
	frame = drawShopInteriorApp(t, app, s, now, 0)
	if !shopRacks(s).Enabled[0] || shopRacks(s).Index[0] != 9 || !shopRacks(s).Enabled[1] || shopRacks(s).Index[1] != 0 {
		t.Fatalf("switch state = enabled%v index%v", shopRacks(s).Enabled, shopRacks(s).Index)
	}
	if frame.RGBAAt(353, 108) != (color.RGBA{R: 19, A: 0xff}) || frame.RGBAAt(197, 108) != (color.RGBA{R: 30, A: 0xff}) {
		t.Fatal("sub100 switch paint did not show old file10 beside new file1")
	}

	// Reaction requests add their bit and preserve the shared index. Repeating
	// the selected rack is a no-op even though the model's shelf stays usable.
	shopMerchant(s).Index = 5
	s.ShopClick(ui.ShopControl{Kind: ui.ShopControlShelfPick, Index: 1})
	if shopMerchant(s).Index != 5 {
		t.Fatal("same-rack click restarted the merchant")
	}
	frame = drawShopInteriorApp(t, app, s, now, 100*time.Millisecond)
	if shopRacks(s).Enabled[0] || shopRacks(s).Index[1] != 1 || shopMerchant(s).Index != 6 {
		t.Fatalf("post-switch eligible state = enabled%v rack%v merchant%d", shopRacks(s).Enabled, shopRacks(s).Index, shopMerchant(s).Index)
	}
	if got := frame.RGBAAt(277, 112); got != (color.RGBA{B: 45, A: 0xff}) {
		t.Fatalf("retained-index Yes pixel = %v, want Yes file7", got)
	}
}

func TestShopInteriorMerchantPriorityTriggersAndIdleCandidate(t *testing.T) {
	f, app, s, now := shopInteriorFixture(t)

	setShopModes(s, shopMerchantYes|shopMerchantNo)
	shopMerchant(s).Index = 0
	s.shopPage().SetClock("step", now.Add(-shopInteriorStep))
	frame := drawShopInteriorApp(t, app, s, now, 0)
	if shopMerchant(s).Index != 1 || frame.RGBAAt(277, 112) != (color.RGBA{B: 40, A: 0xff}) {
		t.Fatal("Yes did not advance/draw ahead of pending No")
	}
	for shopMerchant(s).Index < 11 {
		drawShopInteriorApp(t, app, s, now, shopInteriorStep)
	}
	frame = drawShopInteriorApp(t, app, s, now, shopInteriorStep)
	if shopModes(s) != shopMerchantNo || shopMerchant(s).Index != 0 || frame.RGBAAt(277, 112) != (color.RGBA{R: 1, G: 2, B: 3, A: 0xff}) {
		t.Fatalf("Yes completion = modes%x index%d pixel%v", shopModes(s), shopMerchant(s).Index, frame.RGBAAt(277, 112))
	}
	frame = drawShopInteriorApp(t, app, s, now, shopInteriorStep)
	if shopMerchant(s).Index != 1 || frame.RGBAAt(277, 112) != (color.RGBA{R: 70, G: 1, A: 0xff}) {
		t.Fatal("pending No did not begin after Yes completion")
	}

	setShopModes(s, 0)
	shopMerchant(s).Index = 0
	s.shopPage().SetClock("idle", now.Add(-shopInteriorIdleBase))
	s.shopPage().SetClock("step", now.Add(-shopInteriorStep))
	frame = drawShopInteriorApp(t, app, s, now, 0)
	if shopModes(s) != shopMerchantIdle || shopMerchant(s).Index != 1 || frame.RGBAAt(277, 112) != (color.RGBA{G: 20, A: 0xff}) {
		t.Fatal("eligible five-second candidate did not arm and advance Idle")
	}

	// The actual ShopClick routes add Yes/No without replacing stock, table or
	// messages. Merchant goods exercise both affordability arms; a player lot
	// exercises Sell.
	f.Shop.shelves[ShelfWeapons] = []ShopItem{{Code: 1, Price: 5, Count: 2}}
	if !f.Shop.TakeFromShelf(ShelfWeapons, 0, 1) {
		t.Fatal("merchant table fixture refused")
	}
	setShopModes(s, 0)
	shopMerchant(s).Index = 7
	f.Town.gold = 100
	if msg := s.ShopClick(ui.ShopControl{Kind: ui.ShopControlButton, Index: 1}).Msg; msg == "" || shopModes(s)&shopMerchantYes == 0 || shopMerchant(s).Index != 7 {
		t.Fatalf("affordable buy = %q modes%x index%d", msg, shopModes(s), shopMerchant(s).Index)
	}
	f.Shop.shelves[ShelfWeapons] = []ShopItem{{Code: 2, Price: 50, Count: 1}}
	if !f.Shop.TakeFromShelf(ShelfWeapons, 0, 1) {
		t.Fatal("refusal table fixture refused")
	}
	setShopModes(s, 0)
	shopMerchant(s).Index = 4
	f.Town.gold = 0
	if msg := s.ShopClick(ui.ShopControl{Kind: ui.ShopControlButton, Index: 1}).Msg; msg != "" || shopModes(s) != shopMerchantNo || shopMerchant(s).Index != 4 {
		t.Fatalf("unaffordable buy = %q modes%x index%d", msg, shopModes(s), shopMerchant(s).Index)
	}
	f.Shop.ClearTable()
	if !f.Shop.PutOnTable(ShopItem{Code: 3, Price: 7, Count: 1}) {
		t.Fatal("sell table fixture refused")
	}
	setShopModes(s, 0)
	shopMerchant(s).Index = 3
	if msg := s.ShopClick(ui.ShopControl{Kind: ui.ShopControlButton, Index: 2}).Msg; msg == "" || shopModes(s) != shopMerchantYes || shopMerchant(s).Index != 3 {
		t.Fatalf("sell = %q modes%x index%d", msg, shopModes(s), shopMerchant(s).Index)
	}
}

func TestShopInteriorLifecycleDegradationAndPresentationBoundary(t *testing.T) {
	f, app, s, now := shopInteriorFixture(t)
	drawShopInteriorApp(t, app, s, now, 0)
	before := shopRacks(s).Index[0]
	if err := app.HeadlessFocus(false); err != nil {
		t.Fatal(err)
	}
	if s.shopPage().Active() {
		t.Fatal("focus loss left shop controller active")
	}
	drawShopInteriorApp(t, app, s, now, time.Hour)
	if shopRacks(s).Index[0] != before {
		t.Fatal("unfocused live draw advanced shop")
	}
	if err := app.HeadlessFocus(true); err != nil {
		t.Fatal(err)
	}
	drawShopInteriorApp(t, app, s, now, 99*time.Millisecond)
	if shopRacks(s).Index[0] != before {
		t.Fatal("resume admitted less than 100ms")
	}
	drawShopInteriorApp(t, app, s, now, time.Millisecond)
	if shopRacks(s).Index[0] == before {
		t.Fatal("resume did not advance at rebased 100ms equality")
	}

	// A missing rack or reaction family falls back locally. Complete siblings
	// continue to resolve their own selected picture.
	art := f.shopArtCache
	scene := art.Value().Scene
	yesFrames := scene["yes"]
	incomplete := make([]image.Image, shopRackFrameCount)
	incomplete[0] = shopInteriorPixel(color.RGBA{R: 0xee, A: 0xff})
	scene["rack2"] = incomplete
	s.shopPage().Event("rack", 2)
	shopMerchant(s).Index = 1
	delete(scene, "yes")
	view := s.ShopScreen()
	frame := ui.ComposeShopScreen(view, image.Point{}, false, nil, false)
	if frame.RGBAAt(313, 20) != (color.RGBA{R: 0xee, A: 0xff}) || frame.RGBAAt(277, 112) != (color.RGBA{R: 1, G: 2, B: 3, A: 0xff}) {
		t.Fatal("missing rack/Yes family removed its static local fallback")
	}
	setShopModes(s, shopMerchantNo)
	shopMerchant(s).Index = 1
	frame = ui.ComposeShopScreen(s.ShopScreen(), image.Point{}, false, nil, false)
	if frame.RGBAAt(277, 112) != (color.RGBA{R: 70, G: 1, A: 0xff}) {
		t.Fatal("missing Yes family removed complete No sibling")
	}
	scene["yes"] = yesFrames
	delete(scene, "idle")
	setShopModes(s, shopMerchantIdle|shopMerchantYes)
	shopMerchant(s).Index = 1
	frame = ui.ComposeShopScreen(s.ShopScreen(), image.Point{}, false, nil, false)
	if frame.RGBAAt(277, 112) != (color.RGBA{R: 1, G: 2, B: 3, A: 0xff}) {
		t.Fatal("missing active Idle family exposed lower-priority Yes")
	}
	delete(scene, "yes")
	setShopModes(s, shopMerchantYes|shopMerchantNo)
	shopMerchant(s).Index = 1
	frame = ui.ComposeShopScreen(s.ShopScreen(), image.Point{}, false, nil, false)
	if frame.RGBAAt(277, 112) != (color.RGBA{R: 1, G: 2, B: 3, A: 0xff}) {
		t.Fatal("missing active Yes family exposed lower-priority No")
	}

	mw, _ := readoutWorld(t, sim.Entity{ID: 1, X: 4, Y: 5})
	f.live = mw
	worldBytes, worldHash := marshalWorld(t, mw.world), mw.world.Hash()
	beforeSnapshot, _, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	nativeBefore, err := EncodeSave(beforeSnapshot, "same")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 80; i++ {
		drawShopInteriorApp(t, app, s, now, shopInteriorStep)
	}
	afterSnapshot, _, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	nativeAfter, err := EncodeSave(afterSnapshot, "same")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(nativeBefore, nativeAfter) || mw.world.Hash() != worldHash || !bytes.Equal(worldBytes, marshalWorld(t, mw.world)) {
		t.Fatal("shop presentation changed native bytes, simulation hash or binary state")
	}

	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if s.room != roomSquare || s.shopPage().Ready() {
		t.Fatal("shop exit retained controller state")
	}
	if err := app.HeadlessActivate("SHOP"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	if !s.shopPage().Ready() || shopRacks(s).Selected != 0 || shopModes(s) != 0 || s.shopChosen != 0 {
		t.Fatalf("fresh reentry = %+v chosen%d", *shopRacks(s), s.shopChosen)
	}
	shopRacks(s).Index[0] = 5
	if err := app.HeadlessKey("f3"); err != nil {
		t.Fatal(err)
	}
	if app.Screen() != ui.ScreenLoad || s.shopPage().Active() {
		t.Fatalf("load menu = screen%v active%v", app.Screen(), s.shopPage().Active())
	}
	if err := app.HeadlessActivate("@first"); err != nil {
		t.Fatal(err)
	}
	if app.Screen() != ui.ScreenTown || s.room != roomSquare || s.shopPage().Ready() {
		t.Fatal("actual App load retained shop presentation")
	}
}

func TestShopInteriorMissingMerchantBaseStaysNilSafeOnLiveComposition(t *testing.T) {
	f, app, s, now := shopInteriorFixture(t)

	// loadShopArt over no archive is the supported no-archive production cache: its art
	// object exists while every optional picture, including Merchant, is nil.
	f.shopArtCache = resolved(loadShopArt(ROM1TownDescription(), nil), nil)
	if got := drawShopInteriorApp(t, app, s, now, 0).RGBAAt(277, 112); got == (color.RGBA{R: 1, G: 2, B: 3, A: 0xff}) {
		t.Fatalf("no-archive merchant drew the fixture's base %v", got)
	}

	// A missing base must not disable independently complete rack and reaction
	// families. Drive the same live App composition with Yes and rack0 file2.
	art := shopInteriorTestArt()
	delete(art.Scene, "merchant")
	f.shopArtCache = resolved(art, nil)
	copy(shopRacks(s).Enabled, []bool{true, false, false, false})
	copy(shopRacks(s).Index, []int{0, 0, 0, 0})
	setShopModes(s, shopMerchantYes)
	shopMerchant(s).Index = 0
	s.shopPage().SetClock("step", now.Add(-shopInteriorStep))
	frame := drawShopInteriorApp(t, app, s, now, 0)
	if got := frame.RGBAAt(353, 108); got != (color.RGBA{R: 11, A: 0xff}) {
		t.Fatalf("missing base suppressed complete rack: %v", got)
	}
	if got := frame.RGBAAt(277, 112); got != (color.RGBA{B: 40, A: 0xff}) {
		t.Fatalf("missing base suppressed complete Yes family: %v", got)
	}
}
