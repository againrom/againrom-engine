package game

import (
	"image"
	"image/color"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/data"
	"againrom/pkg/formats/reg"
)

// A SPEAKER NO LIVE ACTOR ANSWERS FOR STANDS ON THE HERO BACKDROP when his
// record states `Hero`. The synthesised drawable takes the token as its hero bit
// (`TAVERN-TALKSTATS-017`), and the figure compositor draws the warrior backdrop
// behind a drawable whose hero bit is set and whose mage bit is clear, the
// woman's backdrop when the sex bit is set (`HERO-FIGURE-144`). The speaker is
// still bare: the drawable holds twelve empty equipment slots
// (`DLG-SPEAKER-022`).
//
// The records are built through the registry loader from the `Flags` spellings
// of the shipped Hero records that name a face, so the token set, the figure
// directory and the sheet number all reach the resolver the way an installed
// registry gives them. Every body sheet paints pixel (0,0) only and each
// backdrop sheet paints all four pixels in its own colour, so pixel (1,1) is
// that backdrop's colour or nothing. Two people who are no heroes stand on the
// sheets a hero's record also names, and the sequence returns to a hero after
// each of them, so a backdrop cached under a sheet would show on the wrong
// speaker.
func TestSynthesisedHeroSpeakerStandsOnTheHeroBackdrop(t *testing.T) {
	section := func(name, flags string, face int32) synth.RegNode {
		return synth.RegNode{Name: name, Kind: kindDir, Children: []synth.RegNode{
			{Name: "Flags", Kind: 0x00, Str: flags},
			{Name: "Face", Kind: kindInt, Int: face},
		}}
	}
	registry, err := reg.Parse(synth.Reg(kindRoot, []synth.RegNode{
		section("npc25", "Hero,Face,!Female,!Mage", 1),
		section("npc26", "Hero,Face,Mage,!Female", 4),
		section("npc27", "Hero,Face,!Female,!Mage", 7),
		section("npc28", "Hero,Face,Female,!Mage", 2),
		section("npc60", "Human,!Mage,!Female,Face", 1),
		section("npc61", "Human,!Mage,Female,Face", 2),
	}))
	if err != nil {
		t.Fatal(err)
	}

	body := color.RGBA{R: 0xff, A: 0xff}
	man := color.RGBA{B: 0xff, A: 0xff}
	woman := color.RGBA{G: 0xff, A: 0xff}
	sheet := invSparseSheet(color.RGBA{R: 0xff}, 0)
	src := missionSource{
		graphicsPrefix + data.ItemFigureBasePath(data.FigureDirManFighter, 1):   sheet,
		graphicsPrefix + data.ItemFigureBasePath(data.FigureDirManMage, 4):      sheet,
		graphicsPrefix + data.ItemFigureBasePath(data.FigureDirManFighter, 7):   sheet,
		graphicsPrefix + data.ItemFigureBasePath(data.FigureDirWomanFighter, 2): sheet,
		graphicsPrefix + "interface/heroback/backm.256":                         invSparseSheet(color.RGBA{B: 0xff}, 0, 1, 2, 3),
		graphicsPrefix + "interface/heroback/backf.256":                         invSparseSheet(color.RGBA{G: 0xff}, 0, 1, 2, 3),
	}
	mw := &mapWorld{
		mission:    &missionNotices{src: src},
		figurePics: map[figureCacheKey]*image.RGBA{},
		npcFaces:   data.LoadNPCFaces(registry),
	}

	for _, tc := range []struct {
		name    string
		speaker int
		behind  color.RGBA
	}{
		{"a man who is no hero, on Brian's own sheet", 60, color.RGBA{}},
		{"Brian's record: a man fighter hero", 25, man},
		{"the same man again", 60, color.RGBA{}},
		{"a man mage hero has no backdrop", 26, color.RGBA{}},
		{"the third man fighter hero", 27, man},
		{"a woman who is no hero, on the hero woman's sheet", 61, color.RGBA{}},
		{"a woman fighter hero stands on the woman's backdrop", 28, woman},
		{"the same woman again", 61, color.RGBA{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pic, _, ok := mw.SpeakerFace(tc.speaker)
			if !ok || pic == nil {
				t.Fatalf("npc%d answered no picture", tc.speaker)
			}
			if got := pic.RGBAAt(0, 0); got != body {
				t.Errorf("npc%d body pixel = %v, want the sheet's %v", tc.speaker, got, body)
			}
			if got := pic.RGBAAt(1, 1); got != tc.behind {
				t.Errorf("npc%d shows %v behind the body, want %v", tc.speaker, got, tc.behind)
			}
		})
	}

	// Standing on the backdrop adds no clothing: every speaker above is
	// composed from twelve empty slots.
	for _, speaker := range []int{25, 26, 27, 28, 60, 61} {
		if key := composeKey(t, mw, speaker); key.eq != (data.Equipment{}) {
			t.Errorf("npc%d composed with %+v, want twelve empty slots", speaker, key.eq)
		}
	}
}
