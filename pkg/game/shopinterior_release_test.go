package game

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"againrom/pkg/formats/bmp"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestReleaseShopInterior1122InstalledAppFramesAndNative(t *testing.T) {
	f := releaseFront(t)
	art := f.shopArt()
	if art == nil {
		t.Fatal("installed shop art did not load")
	}
	read := func(path string) []byte {
		t.Helper()
		payload, err := f.Archives.Containers.ReadFile(path)
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		return payload
	}
	decode := func(path string) image.Image {
		t.Helper()
		decoded, err := bmp.Decode(read(path))
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		pic := image.NewRGBA(image.Rect(0, 0, decoded.Width, decoded.Height))
		for i, c := range decoded.Pix {
			alpha := uint8(0xff)
			if c.R == 0 && c.G == 0 && c.B == 0 {
				alpha = 0
			}
			pic.SetRGBA(i%decoded.Width, i/decoded.Width, color.RGBA{R: c.R, G: c.G, B: c.B, A: alpha})
		}
		return pic
	}

	// Literal addresses and counts are the oracle. No production animation
	// loader, family slice or selected-frame helper contributes a picture.
	var racks [4][]image.Image
	for rack := range racks {
		for file := 1; file <= 11; file++ {
			racks[rack] = append(racks[rack], decode(fmt.Sprintf("graphics/interface/shopanim/%02d/%d.bmp", 4-rack, file)))
		}
	}
	base := decode("movies/shopanim/pose2-3/1.bmp")
	var idle, yes, no []image.Image
	for file := 2; file <= 29; file++ {
		idle = append(idle, decode(fmt.Sprintf("movies/shopanim/pose2-3/%d.bmp", file)))
	}
	for file := 2; file <= 12; file++ {
		yes = append(yes, decode(fmt.Sprintf("movies/shopanim/yes/%d.bmp", file)))
		no = append(no, decode(fmt.Sprintf("movies/shopanim/no/%d.bmp", file)))
	}
	for rack := range racks {
		got := art.Scene[fmt.Sprintf("rack%d", rack)]
		if len(got) != len(racks[rack]) {
			t.Fatalf("installed rack %d count = %d, want %d", rack, len(got), len(racks[rack]))
		}
		for i := range racks[rack] {
			if !imagesEqual(got[i], racks[rack][i]) {
				t.Fatalf("installed rack %d frame %d differs from literal decode", rack, i+1)
			}
		}
	}
	for name, pair := range map[string]struct {
		got  []image.Image
		want []image.Image
	}{"idle": {art.Scene["idle"], idle}, "yes": {art.Scene["yes"], yes}, "no": {art.Scene["no"], no}} {
		if len(pair.got) != len(pair.want) {
			t.Fatalf("installed merchant %s count = %d, want %d", name, len(pair.got), len(pair.want))
		}
		for i := range pair.want {
			if !imagesEqual(pair.got[i], pair.want[i]) {
				t.Fatalf("installed merchant %s frame %d differs from literal decode", name, i+2)
			}
		}
	}
	if merchant := art.Scene["merchant"]; len(merchant) != 1 || !imagesEqual(merchant[0], base) {
		t.Fatal("installed merchant base differs from literal decode")
	}

	chapter := releaseMercenaryChapter(t, f.Campaign.Value())
	town := NewTown(f.Campaign.Value())
	for mission := range f.Campaign.Value().Chapters {
		if mission < chapter {
			town.Won(mission)
		}
	}
	f.Town = town
	f.Carried = f.NextParty()
	f.arriveInTown()
	snapshot, label, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := EncodeSave(snapshot, label)
	if err != nil {
		t.Fatal(err)
	}
	store := SaveStore{Dir: t.TempDir()}
	if _, err := store.Write(time.Unix(200, 0), payload); err != nil {
		t.Fatal(err)
	}

	now := time.Unix(200, 0)
	f.TownAnimationNow = func() time.Time { return now }
	f.ShopRandom = func(n int) int {
		if n != 5 {
			t.Fatalf("shop random range = %d", n)
		}
		return 0
	}
	app := f.App("1122-installed-shop")
	app.Layout(640, 480)
	app.SetSaveSeams(agsSaveSeams(f, store, OriginalStore{}, nil))
	if err := app.HeadlessActivate("load game"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessActivate("@first"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessActivate("SHOP"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	s := f.TownScreen().(*townScreen)
	for n := 0; s.room == roomTalk && n < 32; n++ {
		if err := app.HeadlessActivate("dialogue"); err != nil {
			t.Fatal(err)
		}
	}
	if s.room != roomShop {
		t.Fatalf("App entered room %d, want shop", s.room)
	}
	s.CloseTip()

	mw, _ := readoutWorld(t, sim.Entity{ID: 1122, X: 4, Y: 5})
	f.live = mw
	worldBytes, worldHash := marshalWorld(t, mw.world), mw.world.Hash()
	before, _, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	nativeBefore, err := EncodeSave(before, "same")
	if err != nil {
		t.Fatal(err)
	}

	writeFrame := func(name string, frame *image.RGBA) {
		t.Helper()
		dir := os.Getenv("AGAINROM_STORY1122_FRAMES")
		if dir == "" {
			return
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		rootName := filepath.Base(filepath.Clean(f.Archives.Root))
		out, err := os.Create(filepath.Join(dir, rootName+"-"+name+".png"))
		if err != nil {
			t.Fatal(err)
		}
		if err = png.Encode(out, frame); err != nil {
			out.Close()
			t.Fatal(err)
		}
		if err = out.Close(); err != nil {
			t.Fatal(err)
		}
	}
	check := func(name string, actual *image.RGBA, visible map[int]int, merchant image.Image) {
		t.Helper()
		view := s.ShopScreen()
		scene := literalShopScene{merchant: merchant}
		for rack, index := range visible {
			scene.racks[rack] = racks[rack][index]
		}
		view.Scene = scene
		want := ui.ComposeShopScreen(view, image.Point{}, false, nil, false)
		if !imagesEqual(actual, want) {
			for y := 0; y < actual.Bounds().Dy(); y++ {
				for x := 0; x < actual.Bounds().Dx(); x++ {
					if actual.RGBAAt(x, y) != want.RGBAAt(x, y) {
						t.Fatalf("%s first differing pixel %d,%d = %v, want %v", name, x, y, actual.RGBAAt(x, y), want.RGBAAt(x, y))
					}
				}
			}
			t.Fatalf("%s differs from independent literal-frame composition", name)
		}
		writeFrame(name, actual)
	}

	frame := drawShopInteriorApp(t, app, s, &now, 0)
	check("entry-rack0-file2", frame, map[int]int{0: 1}, base)
	if action := s.ShopClick(ui.ShopControl{Kind: ui.ShopControlShelfPick, Index: 2}); action.Msg != "" {
		t.Fatal("installed rack selection posted a shelf count caption")
	}
	frame = drawShopInteriorApp(t, app, s, &now, 0)
	check("switch-old10-new1", frame, map[int]int{0: 9, 2: 0}, base)
	frame = drawShopInteriorApp(t, app, s, &now, shopInteriorStep)
	check("rack2-file2-yes2", frame, map[int]int{2: 1}, yes[0])

	setShopModes(s, 0)
	shopMerchant(s).Index = 0
	s.shopPage().SetClock("idle", now.Add(-shopInteriorIdleBase))
	s.shopPage().SetClock("step", now.Add(-shopInteriorStep))
	frame = drawShopInteriorApp(t, app, s, &now, 0)
	check("idle-file2", frame, map[int]int{2: 2}, idle[0])
	setShopModes(s, shopMerchantNo)
	shopMerchant(s).Index = 0
	s.shopPage().SetClock("step", now.Add(-shopInteriorStep))
	frame = drawShopInteriorApp(t, app, s, &now, 0)
	check("no-file2", frame, map[int]int{2: 3}, no[0])

	after, _, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	nativeAfter, err := EncodeSave(after, "same")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(nativeBefore, nativeAfter) {
		t.Fatal("installed shop presentation changed native save bytes")
	}
	if mw.world.Hash() != worldHash || !bytes.Equal(worldBytes, marshalWorld(t, mw.world)) {
		t.Fatal("installed shop presentation changed simulation hash or binary state")
	}
	t.Logf("installed shop: racks 4x%d; merchant base + idle%d yes%d no%d; five App paints matched literal independent frames; no sound-name or audible-output claim",
		len(racks[0]), len(idle), len(yes), len(no))
}

// literalShopScene composes the shop centre from literal ROM1 placements:
// four racks and the merchant, each composited over at its fixed point.
type literalShopScene struct {
	racks    [4]image.Image
	merchant image.Image
}

func (l literalShopScene) Paint(dst *image.RGBA, group string) {
	if group != "interior" {
		return
	}
	at := []image.Point{{353, 108}, {197, 108}, {313, 20}, {201, 20}}
	for rack, pic := range l.racks {
		if pic != nil {
			b := pic.Bounds()
			draw.Draw(dst, b.Add(at[rack].Sub(b.Min)), pic, b.Min, draw.Over)
		}
	}
	if l.merchant != nil {
		b := l.merchant.Bounds()
		draw.Draw(dst, b.Add(image.Pt(277, 112).Sub(b.Min)), l.merchant, b.Min, draw.Over)
	}
}
