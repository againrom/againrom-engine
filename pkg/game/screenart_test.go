package game

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"sort"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/base"
	"againrom/pkg/screenart"
	"againrom/pkg/town"
)

// editionArt is every screen art manifest one edition reads, by screen: the
// screen art description's, the generator description's, and the square's
// and every room's of the town and room descriptions.
func editionArt(p base.Profile) map[string][]screenart.Entry {
	out := map[string][]screenart.Entry{}
	if s := ScreensDescription(p); s != nil {
		for _, m := range s.Manifests {
			out[m.Screen] = m.Art
		}
	}
	if l := GeneratorDescription(p); l != nil {
		out["generator"] = generatorArt(l)
	}
	if d := TownDescription(p); d != nil {
		out["square"] = d.SquareScene().Art
	}
	rooms := func(prefix string, d *town.Description) {
		for _, r := range d.Rooms {
			out[prefix+r.Name] = r.Art
			if r.Scene != nil {
				out[prefix+r.Name+" scene"] = r.Scene.Art
			}
		}
	}
	if d := RoomDescription(p); d != nil {
		rooms("", d)
	}
	if p.GameOf() == base.GameROM2 {
		for _, name := range secondTownDescriptions {
			d := townDescriptions[name]
			out[name+" square"] = d.SquareScene().Art
			rooms(name+" ", d)
		}
	}
	return out
}

// Every install picture the shipped descriptions of one edition name is held
// to one rule: two screens naming one picture state the same format, size
// and key colour, so the one library validates it once and never refuses the
// second screen.
func TestShippedScreenArtHoldsEachPictureToOneRule(t *testing.T) {
	for _, p := range []base.Profile{{}, {Game: base.GameROM2}} {
		art := editionArt(p)
		screens := make([]string, 0, len(art))
		for s := range art {
			screens = append(screens, s)
		}
		sort.Strings(screens)
		first := map[string]string{}
		rule := map[string]string{}
		shared := 0
		readers := map[string][]string{}
		for _, s := range screens {
			if err := screenart.Check(art[s]); err != nil {
				t.Errorf("%s: %s: %v", p.GameOf(), s, err)
			}
			for _, e := range art[s] {
				for _, key := range e.Keys() {
					if r, ok := rule[key]; ok {
						if first[key] != s {
							shared++
							readers[key] = append(readers[key], s)
						}
						if r != e.Rule() {
							t.Errorf("%s: %s reads %s as %s, %s as %s", p.GameOf(), first[key], key, r, s, e.Rule())
						}
						continue
					}
					first[key], rule[key] = s, e.Rule()
				}
			}
		}
		if len(screens) < 10 || shared == 0 {
			t.Errorf("%s: %d screens, %d shared readings; the census lost its population", p.GameOf(), len(screens), shared)
		}
		keys := make([]string, 0, len(readers))
		for k := range readers {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			t.Logf("%s: %s read by %s and %s as %s", p.GameOf(), k, first[k], strings.Join(readers[k], ", "), rule[k])
		}
		t.Logf("%s: %d screens, %d pictures, %d readings of a picture another screen names", p.GameOf(), len(screens), len(rule), shared)
	}
}

// The tavern's four-command body and the shop's menu, and the shop's pack
// arrows and the bottom bar's, are one picture each: through the install's
// one library they are read once, and the second screen draws the first's
// decode.
func TestScreensSharingAPictureReadItOnce(t *testing.T) {
	src := tipArtSource()
	for key, size := range map[string]image.Point{
		shopArtPrefix + "shopmenu.bmp": {176, 238}, shopArtPrefix + "invarrow1.bmp": {32, 88},
		shopArtPrefix + "crystalr.bmp": {160, 158}, shopArtPrefix + "crystall.bmp": {16, 158},
	} {
		src[key] = synthBMP(size.X, size.Y, color.RGBA{R: 7, A: 255})
	}
	reads := map[string]int{}
	lib := screenart.NewLibrary(screenArtSource{countedSource{src, reads}})
	art := installArt{EntrySource: src, lib: lib}
	pick := func(entries []screenart.Entry, keys ...string) []screenart.Entry {
		var out []screenart.Entry
		for _, e := range entries {
			for _, k := range keys {
				// An indexed family is narrowed to its first member.
				if e.Key == shopArtPrefix+k {
					e.Index, e.Key = nil, shopArtPrefix+strings.Replace(k, "%d", "1", 1)
					out = append(out, e)
				}
			}
		}
		return out
	}
	rooms := editionArt(base.Profile{})
	tavern, err := screenArtOf(art).Load("tavern", pick(rooms["tavern"], "shopmenu.bmp"))
	if err != nil {
		t.Fatal(err)
	}
	shop, err := screenArtOf(art).Load("shop", pick(rooms["shop"], "shopmenu.bmp", "invarrow%d.bmp"))
	if err != nil {
		t.Fatal(err)
	}
	hud, err := screenArtOf(art).Load("bottom-hud", pick(rooms["bottom-hud"], "invarrow%d.bmp"))
	if err != nil {
		t.Fatal(err)
	}
	if reads[shopArtPrefix+"shopmenu.bmp"] != 1 || reads[shopArtPrefix+"invarrow1.bmp"] != 1 {
		t.Fatalf("reads = %v, want each shared picture read once", reads)
	}
	if tavern.Picture("command-upper") == nil || tavern.Picture("command-upper") != shop.Picture("menu") {
		t.Fatal("the tavern and the shop hold different decodes of shopmenu.bmp")
	}
	if shop.Picture("pack-arrow") == nil || shop.Picture("pack-arrow") != hud.Picture("pack-arrow") {
		t.Fatal("the shop and the bottom bar hold different decodes of invarrow1.bmp")
	}
	minimap, err := loadScreen(art, rom1Screens, "mission-minimap", "background", "seam")
	if err != nil {
		t.Fatal(err)
	}
	panels, err := LoadMenuPanelArt(art, rom1Screens)
	if err != nil {
		t.Fatal(err)
	}
	if panels.Minimap == nil || panels.Minimap != minimap.Picture("background") || panels.MinimapSeam != minimap.Picture("seam") {
		t.Fatal("menu panels do not hold the mission minimap's canonical pictures")
	}
	if reads[shopArtPrefix+"crystalr.bmp"] != 1 || reads[shopArtPrefix+"crystall.bmp"] != 1 {
		t.Fatalf("shared minimap reads = %v, want one per picture", reads)
	}
	if reads[shopArtPrefix+"minimapdata.bmp"] != 0 {
		t.Fatal("menu panels depend on the mission minimap's lookup")
	}
}

func TestSelectedScreenArtRejectsAnUnknownEntry(t *testing.T) {
	if _, err := loadScreen(tipArtSource(), rom1Screens, "mission-minimap", "absent"); err == nil || !strings.Contains(err.Error(), "absent") {
		t.Fatalf("unknown entry error = %v", err)
	}
}

func TestGameOptionsKeepsReadableSelectedFrames(t *testing.T) {
	valid := synth.Frame256{Width: 2, Height: 2, Pixels: []synth.Pixel256{{Index: 1, Opaque: true}, {}, {}, {Index: 1, Opaque: true}}}
	for _, edition := range []base.Game{base.GameROM1, base.GameROM2} {
		for _, empty := range []int{4, 2} {
			t.Run(fmt.Sprintf("%s/empty-%d", edition, empty), func(t *testing.T) {
				frames := []synth.Frame256{valid, valid, valid, valid, valid}
				frames[empty] = synth.Frame256{}
				src := chargenSource{tipGemPath: synth.Sheet256(synth.Sheet256Options{
					Palette: []color.RGBA{{}, {R: 16, G: 32, B: 48}}, Frames: frames,
				})}
				set, err := loadScreen(src, ScreensDescription(base.Profile{Game: edition}), "game-options")
				if err != nil {
					t.Fatal(err)
				}
				for _, selected := range []struct {
					name  string
					frame int
				}{{"check/0", 2}, {"radio/0", 0}, {"check/1", 3}, {"radio/1", 1}} {
					want, oldErr := loadTipGemFrame(src, tipGemPath, selected.frame)
					got := rgbaOf(set.Picture(selected.name))
					if oldErr != nil {
						if got != nil {
							t.Errorf("%s: empty selected frame %d remained present", selected.name, selected.frame)
						}
						continue
					}
					if got == nil || got.Bounds() != want.Bounds() || !bytes.Equal(got.Pix, want.Pix) {
						t.Errorf("%s: readable frame %d changed when frame %d is empty", selected.name, selected.frame, empty)
					}
				}
			})
		}
	}
}

func TestMenuPanelKeepsPiecesWhenAnUnselectedFrameIsEmpty(t *testing.T) {
	frames := make([]synth.Frame256, 10)
	for i := 0; i < 9; i++ {
		frames[i] = synth.Frame256{Width: 2, Height: 2, Pixels: []synth.Pixel256{{Index: 1, Opaque: true}, {}, {}, {Index: 1, Opaque: true}}}
	}
	const key = "graphics/interface/lm.256"
	src := chargenSource{key: synth.Sheet256(synth.Sheet256Options{
		Palette: []color.RGBA{{}, {R: 16, G: 32, B: 48}}, Frames: frames,
	})}
	if _, err := loadTipGemFrame(src, key, 9); err == nil {
		t.Fatal("unselected fixture frame is not empty")
	}
	for _, edition := range []base.Game{base.GameROM1, base.GameROM2} {
		t.Run(string(edition), func(t *testing.T) {
			art, err := LoadMenuPanelArt(src, ScreensDescription(base.Profile{Game: edition}))
			if err != nil {
				t.Fatal(err)
			}
			for i, got := range art.Pieces {
				want, err := loadTipGemFrame(src, key, i)
				if err != nil {
					t.Fatal(err)
				}
				if got == nil || got.Bounds() != want.Bounds() || !bytes.Equal(got.Pix, want.Pix) {
					t.Errorf("piece %d changed when unselected frame 9 is empty", i)
				}
			}
		})
	}
}

func TestMenuPanelRejectsAnEmptyRequiredFrame(t *testing.T) {
	valid := synth.Frame256{Width: 2, Height: 2}
	frames := []synth.Frame256{valid, valid, valid, valid, {}, valid, valid, valid, valid}
	const key = "graphics/interface/lm.256"
	src := chargenSource{key: synth.Sheet256(synth.Sheet256Options{Palette: []color.RGBA{{}, {R: 16, G: 32, B: 48}}, Frames: frames})}
	if _, err := loadTipGemFrame(src, key, 0); err != nil {
		t.Fatalf("positive fixture frame is unreadable: %v", err)
	}
	if _, err := loadTipGemFrame(src, key, 4); err == nil {
		t.Fatal("required fixture frame is not empty")
	}
	for _, edition := range []base.Game{base.GameROM1, base.GameROM2} {
		art, err := LoadMenuPanelArt(src, ScreensDescription(base.Profile{Game: edition}))
		if err == nil || art != nil || !strings.Contains(err.Error(), key) {
			t.Errorf("%s: empty required frame returned art %v, error %v", edition, art, err)
		}
	}
}
