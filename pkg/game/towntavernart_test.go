package game

import (
	"encoding/binary"
	"fmt"
	"image/color"
	"sort"
	"strings"
	"testing"

	"againrom/pkg/ui"
)

// tavernUnitSheet builds a minimal one-frame .16a stream: a 1024-byte
// palette block, a 2x2 frame record carrying one literal pixel, and the
// trailer with the palette flag set. It is cursor_test.go's own curSheet
// (package game_test, unreachable from here) rebuilt at the format's own
// contract rather than shared, since the two packages cannot see each
// other's unexported test helpers.
func tavernUnitSheet() []byte {
	palette := make([]byte, 1024)
	block := make([]byte, 0, 2)
	block = binary.LittleEndian.AppendUint16(block, 0<<14|1) // one literal pixel
	block = binary.LittleEndian.AppendUint16(block, uint16(5<<1)|uint16(15<<9))

	out := append([]byte{}, palette...)
	out = binary.LittleEndian.AppendUint32(out, 2) // width
	out = binary.LittleEndian.AppendUint32(out, 2) // height
	out = binary.LittleEndian.AppendUint32(out, uint32(len(block)))
	out = append(out, block...)
	return binary.LittleEndian.AppendUint32(out, 1|0x80000000) // one frame, palette-bearing
}

// townTavernSource builds a complete synthetic tavern-art tree: the inn's
// original furniture and three button pairs, the shop's four-button panel
// now reused for Sleep (DIV-483), and one one-frame sheet per roster type.
func townTavernSource() chargenSource {
	src := chargenSource{
		townTavernArtPrefix + "leftpicture.bmp": synthBMP(160, 242, color.RGBA{R: 0x21, A: 0xff}),
		townTavernArtPrefix + "leftstats.bmp":   synthBMP(160, 238, color.RGBA{G: 0x21, A: 0xff}),
		townTavernArtPrefix + "centerarea.bmp":  synthBMP(320, 480, color.RGBA{B: 0x21, A: 0xff}),
		townTavernArtPrefix + "manback.bmp":     synthBMP(48, 64, color.RGBA{R: 0x18, G: 0x26, A: 0xff}),
		townTavernArtPrefix + "manbacktalk.bmp": synthBMP(48, 64, color.RGBA{G: 0x38, A: 0xff}),
		townTavernArtPrefix + "buttonsarea.bmp": synthBMP(160, 238, color.RGBA{R: 0x22, G: 0x22, A: 0xff}),
		townTavernArtPrefix + "ruover.bmp":      synthBMP(16, 238, color.RGBA{B: 0x24, A: 0xff}),
		townTavernArtPrefix + "luover.bmp":      synthBMP(16, 238, color.RGBA{G: 0x24, A: 0xff}),
		townTavernArtPrefix + "ldover.bmp":      synthBMP(16, 242, color.RGBA{R: 0x25, A: 0xff}),
		shopArtPrefix + "shopmenu.bmp":          synthBMP(176, 238, color.RGBA{R: 0x26, G: 0x18, A: 0xff}),
	}
	for _, name := range []string{"button1", "button2", "button3"} {
		for _, suffix := range []string{"off", "on"} {
			src[townTavernArtPrefix+name+suffix+".bmp"] = synthBMP(140, 46, color.RGBA{B: 0x33, A: 0xff})
		}
	}
	for i, size := range []struct{ w, h int }{{120, 52}, {140, 46}, {140, 46}, {120, 52}} {
		c := color.RGBA{R: uint8(0x30 + i), A: 0xff}
		if i == 0 {
			c = color.RGBA{A: 0xff} // pure black must become a keyed corner
		}
		src[shopArtPrefix+shopButtonFiles[i]] = synthBMP(size.w, size.h, c)
	}
	for typ := 1; typ <= 15; typ++ {
		src[fmt.Sprintf("%sunit%d/sprites.16a", townTavernArtPrefix, typ)] = tavernUnitSheet()
	}
	for family, spec := range tavernInteriorFamilySpecs {
		for i := 0; i < spec.loaded; i++ {
			c := color.RGBA{R: uint8(0x40 + family), G: uint8(i + 1), B: 0x24, A: 0xff}
			src[townTavernArtPrefix+fmt.Sprintf(spec.pattern, spec.first+i)] = synthBMP(32, 12, c)
		}
	}
	return src
}

// TestLoadTownTavernArtIsAtomic keeps D-2 for the room furniture, roster and
// controls.
//
// cases RANGES OVER townTavernSource()'s OWN KEYS (round 3, D-4): round 2
// named six of the fixture's 25 nodes by hand, and closure.md described the
// coverage as "each case checked by name" as if all 25 were. Ranging over the
// fixture's own map keeps the two in agreement by construction — a node added
// to townTavernSource() in a future round is a case here without a second
// edit, and the case count here is the same count closure.md's Data row now
// reports.
func TestLoadTownTavernArtIsAtomic(t *testing.T) {
	src := townTavernSource()
	got, err := LoadTownTavernArt(src)
	if err != nil {
		t.Fatal(err)
	}
	if got.LeftPicture == nil || got.LeftStats == nil || got.Center == nil || got.Upper == nil ||
		got.ManBack == nil || got.ManBackTalk == nil {
		t.Fatal("one of the tavern's four fixed pictures did not load")
	}
	if got.LeftStatsSeam == nil || got.LeftPictureSeam == nil {
		t.Fatal("one of the tavern's own left-column seams (P-1) did not load")
	}
	for i := 0; i < 3; i++ {
		if got.Buttons[i][0] == nil || got.Buttons[i][1] == nil {
			t.Fatalf("button %d did not load both states", i)
		}
	}
	if got.CommandUpper == nil {
		t.Fatal("four-command panel body did not load")
	}
	for i, button := range got.CommandButtons {
		if button == nil {
			t.Fatalf("four-command button %d did not load", i)
		}
	}
	if _, _, _, a := got.CommandButtons[0].At(0, 0).RGBA(); a != 0 {
		t.Fatalf("four-command keyed black alpha = %#x, want 0", a)
	}
	for typ := 1; typ < 16; typ++ {
		if got.Units[typ] == nil || len(got.UnitFrames[typ]) != 1 {
			t.Fatalf("unit type %d did not load", typ)
		}
	}
	// The talk sheets TOWN-470 lists and the two Hero sheets load when present
	// and are optional: the atomic loop below never deletes one.
	talk := townTavernSource()
	for _, typ := range tavernTalkSheets {
		talk[fmt.Sprintf("%sunit%d/sprites.16a", townTavernArtPrefix, typ)] = tavernUnitSheet()
	}
	talk[townTavernArtPrefix+"herofighter/sprites.16a"] = tavernUnitSheet()
	talk[townTavernArtPrefix+"heromage/sprites.16a"] = tavernUnitSheet()
	withTalk, err := LoadTownTavernArt(talk)
	if err != nil {
		t.Fatal(err)
	}
	for _, typ := range tavernTalkSheets {
		if withTalk.Units[typ] == nil || len(withTalk.UnitFrames[typ]) != 1 || got.Units[typ] != nil {
			t.Fatalf("talk-only sheet Unit%d did not load, or loaded from nothing", typ)
		}
	}
	if len(withTalk.HeroFrames[0]) != 1 || len(withTalk.HeroFrames[1]) != 1 || got.HeroFrames[0] != nil {
		t.Fatal("HeroFighter/HeroMage sheets did not load, or loaded from nothing")
	}
	if len(got.Interior.Candle) != tavernCandleLoaded || len(got.Interior.Cauldron) != tavernCauldronLoaded ||
		len(got.Interior.Breath) != tavernBreathLoaded || len(got.Interior.Drink) != tavernDrinkLoaded {
		t.Fatalf("interior family counts = %d/%d/%d/%d", len(got.Interior.Candle), len(got.Interior.Cauldron), len(got.Interior.Breath), len(got.Interior.Drink))
	}

	cases := make([]string, 0, len(src))
	for k := range src {
		cases = append(cases, k)
	}
	sort.Strings(cases)
	for _, missing := range cases {
		partial := townTavernSource()
		delete(partial, missing)
		if _, err := LoadTownTavernArt(partial); err == nil {
			t.Errorf("missing %s: LoadTownTavernArt returned no error", missing)
		} else if !strings.Contains(err.Error(), missing) {
			t.Errorf("missing %s: error %v does not name it", missing, err)
		}
	}
}

func TestTavernInteriorFamiliesDegradeIndependently(t *testing.T) {
	for _, tc := range []struct {
		name string
		spec tavernInteriorFamilySpec
		len  func(ui.TavernInteriorArt) int
	}{
		{"candle", tavernInteriorFamilySpecs[0], func(a ui.TavernInteriorArt) int { return len(a.Candle) }},
		{"cauldron", tavernInteriorFamilySpecs[1], func(a ui.TavernInteriorArt) int { return len(a.Cauldron) }},
		{"breath", tavernInteriorFamilySpecs[2], func(a ui.TavernInteriorArt) int { return len(a.Breath) }},
		{"drink", tavernInteriorFamilySpecs[3], func(a ui.TavernInteriorArt) int { return len(a.Drink) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			src := townTavernSource()
			missing := townTavernArtPrefix + fmt.Sprintf(tc.spec.pattern, tc.spec.first+tc.spec.loaded/2)
			delete(src, missing)
			got, err := LoadTownTavernArt(src)
			if err == nil || !strings.Contains(err.Error(), missing) {
				t.Fatalf("diagnostic = %v, want %s", err, missing)
			}
			if got == nil || got.Center == nil || tc.len(got.Interior) != 0 {
				t.Fatalf("family-local fallback lost room or retained partial family: %#v", got)
			}
			complete := 0
			for _, n := range []int{len(got.Interior.Candle), len(got.Interior.Cauldron), len(got.Interior.Breath), len(got.Interior.Drink)} {
				if n != 0 {
					complete++
				}
			}
			if complete != 3 {
				t.Fatalf("complete sibling families = %d, want 3", complete)
			}
		})
	}
}
