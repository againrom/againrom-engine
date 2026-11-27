package ui

import (
	"image"
	"image/color"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

func trackLayerImages(t *testing.T) (allocs, disposes, live func() int) {
	t.Helper()
	var made, freed int
	prevNew, prevDispose := newLayerImage, disposeLayerImage
	newLayerImage = func(w, h int) *ebiten.Image {
		made++
		return prevNew(w, h)
	}
	disposeLayerImage = func(img *ebiten.Image) {
		freed++
		prevDispose(img)
	}
	t.Cleanup(func() {
		newLayerImage, disposeLayerImage = prevNew, prevDispose
	})
	return func() int { return made }, func() int { return freed }, func() int { return made - freed }
}

func TestLayerImagesDisposeOnResize(t *testing.T) {
	t.Run("windowResize", func(t *testing.T) {
		v := columnViewer(t)
		v.SetPackShown(true)
		v.SetSpellBookShown(true)
		v.SetSpellbook(1, []SpellEntry{{ID: 1, Name: "Fire"}})

		allocs, disposes, live := trackLayerImages(t)
		sizes := []image.Point{
			{1024, 768},
			{1280, 720},
			{1440, 720},
			{1600, 768},
			{1920, 768},
		}
		var firstLive, firstAllocs int
		for i, sz := range sizes {
			v.Layout(sz.X, sz.Y)
			v.Draw(ebiten.NewImage(sz.X, sz.Y))
			if i == 0 {
				firstLive, firstAllocs = live(), allocs()
			}
		}
		if allocs() <= firstAllocs {
			t.Fatalf("resizes made no new layer images: allocs=%d firstAllocs=%d", allocs(), firstAllocs)
		}
		if live() != firstLive {
			t.Fatalf("live layer images grew across %d resizes: live=%d want=%d (allocs=%d disposes=%d)",
				len(sizes), live(), firstLive, allocs(), disposes())
		}
	})

	t.Run("contentResize", func(t *testing.T) {
		screen := ebiten.NewImage(MissionFrameW, MissionFrameH)

		t.Run("message", func(t *testing.T) {
			v := columnViewer(t)
			allocs, disposes, live := trackLayerImages(t)
			for _, msg := range []string{"A", "Longer message line", "Third even longer message line"} {
				v.ClearMessages()
				v.PostMessage(msg, MessageInk(0), time.Minute)
				v.layerMessage(screen, &windowPass{})
			}
			if allocs() < 3 || live() != 1 {
				t.Fatalf("message: allocs=%d disposes=%d live=%d, want allocs>=3 live=1", allocs(), disposes(), live())
			}
		})

		t.Run("readout", func(t *testing.T) {
			v := columnViewer(t)
			v.SetReadout(Readout{Tick: 1})
			allocs, disposes, live := trackLayerImages(t)
			for _, w := range []int{120, 180, 240} {
				v.readoutLayout.MinWidth = w
				v.readoutSerial++
				v.layerReadout(screen, &windowPass{})
			}
			if allocs() < 3 || live() != 1 {
				t.Fatalf("readout: allocs=%d disposes=%d live=%d, want allocs>=3 live=1", allocs(), disposes(), live())
			}
		})

		t.Run("notice", func(t *testing.T) {
			v := columnViewer(t)
			allocs, disposes, live := trackLayerImages(t)
			v.SetNotice("Short", NoticeDialogue)
			v.paintNotice(screen)
			v.SetNotice("Longer notice\nsecond line\nthird line\nfourth line", NoticeOutcome)
			v.paintNotice(screen)
			v.SetNotice("Different size text for a third notice box", NoticeDialogue)
			v.paintNotice(screen)
			if allocs() < 2 || live() != 1 {
				t.Fatalf("notice: allocs=%d disposes=%d live=%d, want allocs>=2 live=1", allocs(), disposes(), live())
			}
		})

		t.Run("numerals", func(t *testing.T) {
			v := columnViewer(t)
			allocs, disposes, live := trackLayerImages(t)
			now := time.Now()
			for _, dmg := range []int{1, 9999, 1234567} {
				v.numerals = nil
				v.receiveDamageNumeral(soundEntity{ID: 1, Owner: 1, Cell: image.Pt(4, 4)}, 1, dmg, now)
				v.layerNumerals(screen, &windowPass{})
			}
			if allocs() < 3 || live() != 1 {
				t.Fatalf("numerals: allocs=%d disposes=%d live=%d, want allocs>=3 live=1", allocs(), disposes(), live())
			}
		})

		t.Run("dragItem", func(t *testing.T) {
			v := columnViewer(t)
			v.dragActive = true
			v.hasCursor = true
			allocs, disposes, live := trackLayerImages(t)
			for _, side := range []int{16, 24, 32} {
				v.dragIcon = solidPic(side, side, color.RGBA{R: 0xff, A: 0xff})
				v.drawPointer(screen, 1, 0, 0)
			}
			if allocs() < 3 || live() != 1 {
				t.Fatalf("dragItem: allocs=%d disposes=%d live=%d, want allocs>=3 live=1", allocs(), disposes(), live())
			}
		})

		t.Run("missionTip", func(t *testing.T) {
			v := columnViewer(t)
			v.ShowMissionTip("Tip text")
			art := tipTestArt()
			allocs, disposes, live := trackLayerImages(t)
			for _, r := range []image.Rectangle{
				image.Rect(10, 10, 130, 90),
				image.Rect(10, 10, 170, 110),
				image.Rect(10, 10, 210, 130),
			} {
				v.SetMissionTipDress(MissionTipDress{Rect: r, Art: art, Font: panelFont()})
				v.layerMissionTip(screen, &windowPass{})
			}
			if allocs() < 3 || live() != 1 {
				t.Fatalf("missionTip: allocs=%d disposes=%d live=%d, want allocs>=3 live=1", allocs(), disposes(), live())
			}
		})

		t.Run("columnFiller", func(t *testing.T) {
			v := columnViewer(t)
			allocs, disposes, live := trackLayerImages(t)
			for _, h := range []int{30, 46, 60} {
				v.SetCharacterPaneFillerArt(TownPane{
					Body: solidPic(160, h, color.RGBA{B: 0x80, A: 0xff}),
					Seam: solidPic(characterPaneSeamW, h, color.RGBA{R: 0x80, A: 0xff}),
				})
				v.layerColumnFiller(screen, &windowPass{})
			}
			if disposes() != 2 || v.fillerImg == nil {
				t.Fatalf("columnFiller: allocs=%d disposes=%d live=%d, want disposes=2 and a current image", allocs(), disposes(), live())
			}
		})

		t.Run("cursorTexture", func(t *testing.T) {
			var ct cursorTexture
			allocs, disposes, live := trackLayerImages(t)
			for _, side := range []int{16, 32, 44, 64} {
				ct.upload(solidPic(side, side, color.RGBA{G: 0xff, A: 0xff}))
			}
			if allocs() != 4 || live() != 1 {
				t.Fatalf("cursorTexture: allocs=%d disposes=%d live=%d, want allocs=4 live=1", allocs(), disposes(), live())
			}
		})
	})

	t.Run("everyViewerLayerReplacesOld", func(t *testing.T) {
		screen := ebiten.NewImage(MissionFrameW, MissionFrameH)
		cases := []struct {
			name string
			run  func(t *testing.T)
		}{
			{"panel", func(t *testing.T) {
				v := columnViewer(t)
				v.panelImg = newLayerImage(1, 1)
				v.layerPanel(screen, &windowPass{})
			}},
			{"spellbook", func(t *testing.T) {
				v := columnViewer(t)
				v.SetSpellBookShown(true)
				v.SetSpellbook(1, []SpellEntry{{ID: 1, Name: "Fire"}})
				v.spellbookImg = newLayerImage(1, 1)
				v.layerSpellbook(screen, &windowPass{})
			}},
			{"readout", func(t *testing.T) {
				v := columnViewer(t)
				v.SetReadout(Readout{Tick: 1})
				v.readoutImg = newLayerImage(1, 1)
				v.layerReadout(screen, &windowPass{})
			}},
			{"frameRate", func(t *testing.T) {
				v := columnViewer(t)
				v.ToggleFPS()
				v.fpsImg = newLayerImage(1, 1)
				v.layerFrameRate(screen, &windowPass{})
			}},
			{"minimap", func(t *testing.T) {
				v := columnViewer(t)
				v.minimapImg = newLayerImage(1, 1)
				v.layerMinimap(screen, &windowPass{})
			}},
			{"missionCard", func(t *testing.T) {
				v := columnViewer(t)
				v.missionCardImg = newLayerImage(1, 1)
				v.layerMissionCard(screen, &windowPass{})
			}},
			{"structureReadout", func(t *testing.T) {
				v := columnViewer(t)
				v.structImg = newLayerImage(1, 1)
				v.structFresh = true
				f := &windowPass{
					readoutShown: true,
					readoutPic:   solidPic(80, 40, color.RGBA{R: 0x40, A: 0xff}),
				}
				v.layerStructureReadout(screen, f)
			}},
			{"worn", func(t *testing.T) {
				v := columnViewer(t)
				layoutViewport(v, MenuWindowW, wornBoxFixtureH)
				v.invImg = newLayerImage(1, 1)
				v.layerWorn(ebiten.NewImage(v.frameW, v.frameH), &windowPass{})
			}},
			{"packBar", func(t *testing.T) {
				v := columnViewer(t)
				v.SetPackShown(true)
				v.packImg = newLayerImage(1, 1)
				v.layerPackBar(screen, &windowPass{})
			}},
			{"commandPanel", func(t *testing.T) {
				v := columnViewer(t)
				v.commandPanelImg = newLayerImage(1, 1)
				v.layerCommandPanel(screen, &windowPass{})
			}},
			{"gold", func(t *testing.T) {
				v := columnViewer(t)
				v.gold.open, v.gold.text = true, "10"
				v.goldImg = newLayerImage(1, 1)
				v.layerGold(screen, &windowPass{})
			}},
			{"message", func(t *testing.T) {
				v := columnViewer(t)
				v.PostMessage("hello", MessageInk(0), time.Minute)
				v.messageImg = newLayerImage(1, 1)
				v.layerMessage(screen, &windowPass{})
			}},
			{"tooltip", func(t *testing.T) {
				v := columnViewer(t)
				v.SetSpellBookShown(true)
				v.SetSpellbook(1, []SpellEntry{{ID: 1, Name: "Fire", Info: []string{"Fireball"}}})
				bar, cols, ok := v.spellbookBar()
				if !ok {
					t.Fatal("spellbookBar not visible")
				}
				cell := bookCellRects(bar, cols)[0]
				pt := image.Pt(cell.Min.X+4, cell.Min.Y+4)
				v.cursorX, v.cursorY, v.hasCursor = pt.X, pt.Y, true
				target := v.tooltipTarget()
				v.tooltip = &tooltipController{delay: 0}
				v.tooltip.observe(time.Now(), pt, target.key(), false)
				v.itemPopupImg = newLayerImage(1, 1)
				v.layerTooltip(screen, &windowPass{})
			}},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				allocs, disposes, live := trackLayerImages(t)
				tc.run(t)
				if allocs() < 2 {
					t.Fatalf("%s: did not allocate replacement image (allocs=%d)", tc.name, allocs())
				}
				if live() != 1 {
					t.Fatalf("%s: old layer image not disposed: allocs=%d disposes=%d live=%d", tc.name, allocs(), disposes(), live())
				}
			})
		}
	})
}
