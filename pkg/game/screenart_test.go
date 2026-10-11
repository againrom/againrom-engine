package game

import (
	"image"
	"image/color"
	"sort"
	"strings"
	"testing"

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
	src := chargenSource{}
	for key, size := range map[string]image.Point{
		shopArtPrefix + "shopmenu.bmp": {176, 238}, shopArtPrefix + "invarrow1.bmp": {32, 88},
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
}
