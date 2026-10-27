package game

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func creatureResistanceGlyph(o priceOracle, b byte, selector int) int {
	if selector == 1 {
		switch {
		case b >= 0x80 && b <= 0xaf:
			b += 0x30
		case b >= 0xe0 && b <= 0xef:
			b += 0x10
		}
	}
	index := int(b) - 32
	if index < 0 || index >= len(o.frames) {
		return 0
	}
	return index
}

func creatureResistanceWidth(o priceOracle, s string, selector int) int {
	pen, right := 0, 0
	for i := 0; i < len(s); i++ {
		index := creatureResistanceGlyph(o, s[i], selector)
		glyph := o.frames[index]
		for y := 0; y < glyph.Height; y++ {
			for x := 0; x < glyph.Width; x++ {
				if glyph.Pixels[y*glyph.Width+x].Painted {
					right = max(right, pen+x+1)
				}
			}
		}
		pen += o.advances[index] + 2
		if index == 0 {
			pen += glyph.Height / 2
		}
	}
	return max(pen, right)
}

func creatureResistancePaint(o priceOracle, dst *image.RGBA, s string, x, selector int) {
	ink := color.RGBA{R: 148, G: 89, A: 255}
	for i := 0; i < len(s); i++ {
		index := creatureResistanceGlyph(o, s[i], selector)
		glyph := o.frames[index]
		for y := 0; y < glyph.Height; y++ {
			for col := 0; col < glyph.Width; col++ {
				pixel := glyph.Pixels[y*glyph.Width+col]
				if pixel.Painted {
					level := int(pixel.Value)
					dst.SetRGBA(x+col, y, color.RGBA{R: uint8(int(ink.R) * level / 15), G: uint8(int(ink.G) * level / 15), A: 255})
				}
			}
		}
		x += o.advances[index] + 2
		if index == 0 {
			x += glyph.Height / 2
		}
	}
}

func creatureResistanceHeading(t *testing.T, rows []ui.PanelLineReport, y int) ui.PanelLineReport {
	t.Helper()
	for _, row := range rows {
		if row.Right && row.At.Y == y && row.Value == "" && row.RightValue == "" {
			return row
		}
	}
	t.Fatal("the installed card has no heading pair")
	return ui.PanelLineReport{}
}

func TestReleaseCreatureCardUsesInstalledResistanceHeading(t *testing.T) {
	f := releaseFront(t)
	f.Options = OptionsStore{}
	art := (viewerArtSource{in: &f.InstallResources, pr: &f.Presentation, pc: &f.PersistenceContext}).resolve()
	font := art.cardFont
	background, _ := art.paneStats.Body.(*image.RGBA)
	if font == nil || background == nil {
		t.Fatal("the install resolved no card font or background")
	}
	fontBase, configuredFont := DefaultFont, f.Font.Value()
	// The card draws in the generation screen's text font where the install
	// has one (R2-ASSET-148); the oracle reads that font's raw file.
	if a := f.ChargenAssets; a != nil && a.Presentation != nil && a.Presentation.Font != nil {
		fontBase = f.generator().Words.TextFont.Name
		configuredFont = a.Presentation.Font
	}
	if font != configuredFont {
		t.Fatal("the mission card font differs from the configured install/text font")
	}
	identity, err := f.Archives.Containers.ReadFile("main/id")
	if err != nil || len(identity) == 0 || identity[len(identity)-1] != '0' && identity[len(identity)-1] != '1' {
		t.Fatal("the install has no EN/RU language selector", err)
	}
	selector := int(identity[len(identity)-1] - '0')
	if font.Selector != selector {
		t.Fatal("the mission card font differs from the installed language selector")
	}
	oracle := newPriceOracle(t, f, fontBase)
	firstGame := f.Archives.Game() == base.GameROM1
	for _, slot := range []int{27, 28} {
		caption := f.Words.PanelCaptions[slot]
		if caption == "" {
			t.Fatalf("the install states no caption at main.txt slot %d", slot)
		}
		width, _ := font.Measure(caption)
		independent := creatureResistanceWidth(oracle, caption, selector)
		t.Logf("root=%s card-font=%s selector=%d main-slot=%d caption=%s card-width=%d raw-card-width=%d font-height=%d", f.Archives.Root, fontBase, selector, slot, decodeInstallText(caption), width, independent, font.Height())
		if width != independent {
			t.Errorf("slot %d: card font width %d differs from raw %s width %d", slot, width, fontBase, independent)
		}
	}
	layout := ui.CompactPanelLayout(background)
	app := cardCaptionMission(t, f, f.DirectNewGame(10))
	app.SetTooltipDelayPreference(0, nil)
	before := f.live.world.Hash()
	if err := app.HeadlessKey("shift-f4"); err != nil {
		t.Fatal(err)
	}
	var creature, human sim.EntityID
	for _, entity := range f.live.world.Entities() {
		if !entity.Alive() {
			continue
		}
		switch f.live.chars[entity.ID].Band {
		case ui.CharacterBandCreature:
			if creature == 0 {
				creature = entity.ID
			}
		case ui.CharacterBandPerson:
			if human == 0 && f.live.missionPartyMember(entity.ID) == nil {
				human = entity.ID
			}
		}
	}
	if creature == 0 || human == 0 || len(f.live.mission.ids) == 0 {
		t.Fatalf("shipped mission 10 lacks creature/Human/hero controls: creature=%d human=%d party=%d", creature, human, len(f.live.mission.ids))
	}
	controls := []struct {
		name string
		id   sim.EntityID
		band ui.CharacterBand
	}{
		{"creature", creature, ui.CharacterBandCreature},
		{"Human", human, ui.CharacterBandPerson},
		{"hero", f.live.mission.ids[0], ui.CharacterBandPerson},
	}
	for _, control := range controls {
		t.Run(control.name, func(t *testing.T) {
			subject, _ := enemyCardSubject(t, app, f.live, control.id)
			if subject.Char.Band != control.band {
				t.Fatalf("card band %d, want %d", subject.Char.Band, control.band)
			}
			if subject.PreserveCreatureSkillsHeading == firstGame {
				t.Fatal("the creature heading's edition scope differs from the installed game")
			}
			legacySubject := subject
			legacySubject.PreserveCreatureSkillsHeading = true
			legacy := creatureResistanceHeading(t, ui.CharacterPanelReport(layout, font, legacySubject), layout.Pad.Y+26+68)
			if legacy.Label != subject.Words.PanelCaptions[27] {
				t.Fatal("the main baseline does not draw the complete installed slot-27 heading")
			}
			row := creatureResistanceHeading(t, ui.CharacterPanelReport(layout, font, subject), legacy.At.Y)
			t.Logf("subject=%s actual-left=%s actual-right=%s right-x=%d legacy-left=%s legacy-right=%s legacy-right-x=%d", control.name,
				decodeInstallText(row.Label), decodeInstallText(row.RightLabel), row.At.X+row.RightX,
				decodeInstallText(legacy.Label), decodeInstallText(legacy.RightLabel), legacy.At.X+legacy.RightX)
			geometry := row
			geometry.Label = legacy.Label
			if !reflect.DeepEqual(geometry, legacy) {
				t.Fatalf("the heading changed its legacy right cell or geometry: actual=%+v legacy=%+v", row, legacy)
			}
			expected := legacy.Label
			changedCreature := firstGame && control.band == ui.CharacterBandCreature
			if changedCreature {
				expected = subject.Words.PanelCaptions[28]
			}
			if row.Label != expected {
				t.Fatalf("installed heading %q, want %q", decodeInstallText(row.Label), decodeInstallText(expected))
			}
			width := creatureResistanceWidth(oracle, expected, selector)
			if row.At.X != 6 || row.Width <= 0 || width > row.Width || row.At.X+width >= row.At.X+row.RightX {
				t.Fatalf("heading does not fit styled column: row=%+v raw width=%d", row, width)
			}
			pic, err := app.HeadlessMissionCard()
			if err != nil || pic == nil {
				t.Fatal("the shipped mission card did not render", err)
			}
			body := image.NewRGBA(image.Rectangle{Max: layout.Size})
			draw.Draw(body, body.Bounds(), pic, image.Pt(16, 0), draw.Src)
			old := ui.RenderCharacterPanel(layout, font, legacySubject)
			changed := image.Rect(row.At.X, row.At.Y, row.At.X+64, row.At.Y+font.Height())
			changes := 0
			for y := 0; y < layout.Size.Y; y++ {
				for x := 0; x < layout.Size.X; x++ {
					if body.RGBAAt(x, y) == old.RGBAAt(x, y) {
						continue
					}
					if !changedCreature || !image.Pt(x, y).In(changed) {
						t.Fatalf("the actual card changed outside its creature heading at %d,%d", x, y)
					}
					changes++
				}
			}
			if changedCreature {
				if changes == 0 || bytes.Equal(body.Pix, old.Pix) {
					t.Fatal("the creature card still draws the old skills heading")
				}
				glyphs := image.NewRGBA(image.Rect(0, 0, layout.Size.X, font.Height()))
				creatureResistancePaint(oracle, glyphs, expected, row.At.X, selector)
				ink := 0
				for y := 0; y < glyphs.Bounds().Dy(); y++ {
					for x := 0; x < glyphs.Bounds().Dx(); x++ {
						want := glyphs.RGBAAt(x, y)
						if want.A == 0 {
							continue
						}
						if body.RGBAAt(x, row.At.Y+y) != want {
							t.Fatalf("new heading glyph pixel %d,%d differs from raw %s", x, y, fontBase)
						}
						ink++
					}
				}
				if ink == 0 {
					t.Fatal("the independent heading glyphs paint no pixel")
				}
				if !subject.Combat.Known || subject.Words.Hover[188] == "" {
					t.Fatal("the live creature hover lacks its combat-row overlap control or installed slot 188")
				}
				if err := app.HeadlessSelectEntity(uint32(control.id)); err != nil {
					t.Fatal(err)
				}
				_, cardAt, err := app.HeadlessDrawnMissionCard()
				if err != nil {
					t.Fatal(err)
				}
				for _, dy := range []int{0, font.Height() - 1} {
					point := cardAt.Add(image.Pt(16+row.At.X+1, row.At.Y+dy))
					if err := app.HeadlessPointer("hover", point.X, point.Y); err != nil {
						t.Fatal(err)
					}
					selected, shown := f.live.view.InspectionPanel()
					if !shown || selected.ID != subject.ID {
						t.Fatalf("moving onto the card lost its selected creature subject: %d/%v, want %d", selected.ID, shown, subject.ID)
					}
					state, popup := app.HeadlessTooltip()
					if state.Target != fmt.Sprintf("stat/%d/%d", subject.ID, ui.PanelFieldSkillsHeading) || !state.Visible || popup == nil {
						t.Fatalf("live heading row %d has no visible heading tooltip: %+v", dy, state)
					}
					bounds := image.Rect(0, 0, 1024, 768)
					want, _, ok := ui.ComposeTooltipHint(strings.Split(subject.Words.Hover[188], "#"), font, point, bounds, f.HoverBall(), f.HoverHelp())
					oldHint, _, _ := ui.ComposeTooltipHint(strings.Split(subject.Words.Hover[168], "#"), font, point, bounds, f.HoverBall(), f.HoverHelp())
					if !ok || !samePicture(popup, want) || samePicture(popup, oldHint) {
						t.Fatalf("live heading row %d does not draw installed slot 188 instead of slot 168", dy)
					}
				}
			}
			t.Logf("mission=10 subject=%s id=%d heading=%s width=%d/%d right-heading-x=%d card-font=raw-%s font-height=%d changed-pixels=%d remaining-card=main-slot27", control.name, control.id, decodeInstallText(expected), width, row.Width, row.At.X+row.RightX, fontBase, font.Height(), changes)
		})
	}
	if f.live.world.Hash() != before {
		t.Fatal("display-only knowledge and card inspection changed the simulation hash")
	}
}
