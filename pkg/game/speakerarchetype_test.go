package game

import (
	"bytes"
	"image"
	"image/color"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/data"
	"againrom/pkg/formats/reg"
)

func TestUnansweredHeroSpeakerUsesRegistryArchetype(t *testing.T) {
	section := func(name, flags string, face int32) synth.RegNode {
		children := []synth.RegNode{{Name: "Flags", Kind: 0, Str: flags}}
		if face != 0 {
			children = append(children, synth.RegNode{Name: "Face", Kind: kindInt, Int: face})
		}
		return synth.RegNode{Name: name, Kind: kindDir, Children: children}
	}
	registry, err := reg.Parse(synth.Reg(kindRoot, []synth.RegNode{
		section("MaleFighter", "", 5), section("MaleMage", "", 3),
		section("FemaleFighter", "", 1), section("FemaleMage", "", 1),
		section("npc21", "Hero,Me,Start", 0),
		section("npc22", "Hero,Mage,!MySex,Start", 0),
		section("npc23", "Hero,!Mage,!MySex,Start", 0),
		section("npc24", "Hero,!MyClass,MySex,Start", 0),
	}))
	if err != nil {
		t.Fatal(err)
	}
	src := missionSource{}
	for i, dir := range []data.FigureDir{data.FigureDirManFighter, data.FigureDirManMage, data.FigureDirWomanFighter, data.FigureDirWomanMage} {
		face := []int{5, 3, 1, 1}[i]
		src[graphicsPrefix+data.ItemFigureBasePath(dir, face)] = invSparseSheet(color.RGBA{R: uint8(40 + i*40)}, 0)
	}
	src[graphicsPrefix+"interface/heroback/backm.256"] = invSparseSheet(color.RGBA{B: 255}, 0, 1, 2, 3)
	src[graphicsPrefix+"interface/heroback/backf.256"] = invSparseSheet(color.RGBA{G: 255}, 0, 1, 2, 3)
	dirs := []data.FigureDir{data.FigureDirManFighter, data.FigureDirManMage, data.FigureDirWomanFighter, data.FigureDirWomanMage}
	wants := [4][4]int{{0, 3, 2, 1}, {0, 3, 2, 0}, {0, 1, 0, 3}, {0, 1, 0, 2}}
	for primary, dir := range dirs {
		r := speakerResolver{src: src, npcFaces: data.LoadNPCFaces(registry),
			playerFigure: image.NewRGBA(image.Rect(0, 0, 7, 7)),
			cast:         speakerCast{playerDir: dir, hasPlayer: true}}
		for offset, index := range wants[primary] {
			want, _ := composeInventoryPortrait(src, 0, data.Equipment{}, figureID{Dir: dirs[index], Face: []int{5, 3, 1, 1}[index], Hero: true})
			got, _, ok := r.SpeakerFace(21 + offset)
			if !ok || got == nil || got.Bounds() != want.Figure.Bounds() || !bytes.Equal(got.Pix, want.Figure.Pix) {
				t.Errorf("primary %s npc%d: not archetype %s/%d", dir, 21+offset, dirs[index], []int{5, 3, 1, 1}[index])
			}
		}
	}
}
