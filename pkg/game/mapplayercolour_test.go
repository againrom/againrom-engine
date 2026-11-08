package game

import (
	"image/color"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/render/terrain"
)

func TestAuthoredPlayerColoursSelectIndependentCachedFrames(t *testing.T) {
	frame := ownerShadeFrame()
	class := &terrain.UnitClass{OwnerShaded: true}
	set := &terrain.UnitSet{HasOwnerPalettes: true}
	for i := range set.OwnerPalettes {
		set.OwnerPalettes[i][55] = color.RGBA{R: uint8(i), A: 255}
	}
	mw := &mapWorld{units: set}
	ordinal := mw.ownerFrame(class, frame, 2, false)
	if got := ordinal.RGBA().RGBAAt(0, 0).R; got != 2 {
		t.Fatalf("campaign colour %d", got)
	}
	m := &alm.Map{Groups: []alm.Group{{Color: 9}, {Color: 7}, {Color: 7}}}
	mw.usePlayerColours(m)
	m.Groups[1].Color = 12
	authored := mw.ownerFrame(class, frame, 2, false)
	if authored == ordinal || authored.RGBA().RGBAAt(0, 0).R != 7 {
		t.Fatal("authored colour reused the ordinal frame or retained mutable map input")
	}
	if mw.ownerFrame(class, frame, 3, false) != authored || mw.ownerFrame(class, frame, 2, false) != authored {
		t.Fatal("the same chosen shade did not reuse its frame")
	}
	if mw.ownerFrame(class, frame, 18, false) != ordinal {
		t.Fatal("an owner outside the authored roster lost the ordinal fallback")
	}
	mw.usePlayerColours(m)
	if next := mw.ownerFrame(class, frame, 2, false); next == authored || next.RGBA().RGBAAt(0, 0).R != 12 {
		t.Fatal("a replaced roster retained the previous cached shade")
	}
}

func TestShippedSecondGameOwnerTakesItsMapPlayerColour(t *testing.T) {
	shipped := missionIdentity{}
	if usesAuthoredColours(shipped, &firstGameFiles) {
		t.Fatal("a shipped first-game mission left the owner ordinal rule")
	}
	if !usesAuthoredColours(shipped, &secondGameFiles) {
		t.Fatal("a shipped second-game mission ignored its map player colours")
	}
	set := &terrain.UnitSet{HasOwnerPalettes: true}
	for i := range set.OwnerPalettes {
		set.OwnerPalettes[i][55] = color.RGBA{R: uint8(i), A: 255}
	}
	mw := &mapWorld{units: set}
	m := &alm.Map{}
	for _, c := range []uint32{1, 1, 2, 3, 4, 5} {
		m.Groups = append(m.Groups, alm.Group{Color: c})
	}
	if usesAuthoredColours(shipped, &secondGameFiles) {
		mw.usePlayerColours(m)
	}
	class := &terrain.UnitClass{OwnerShaded: true}
	if got := mw.ownerFrame(class, ownerShadeFrame(), 4, false).RGBA().RGBAAt(0, 0).R; got != 3 {
		t.Fatalf("owner 4 draws table %d, want its file colour 3", got)
	}
}
