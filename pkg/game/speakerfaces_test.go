package game

import (
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/data"
	"againrom/pkg/mapload"
)

// The second game's speaker table adds the Humans rows and the player's key
// beside the registry's records, which win; the first game's is the
// registry's alone.
func TestSpeakerFacesByGame(t *testing.T) {
	params := make([]int32, 38)
	for i := range params {
		params[i] = -1
	}
	params[16], params[17], params[18], params[24] = 24, 4, 1, 517
	humans := &fixtureCollection{names: []string{"", "Mage"}, params: [][]int32{nil, params}, raw: [][]byte{nil, nil}}
	registry := map[int32]data.NPCFace{7: {Kind: data.NPCPortrait, Face: 2}, 517: {Kind: data.NPCPortrait, Face: 9}}
	table := func(g base.Game) *mapload.Table {
		return readUnder(g, &mapload.Table{Humans: humans, NPCCatalog: &data.NPCCatalog{Faces: registry}})
	}
	first := speakerFaces(table(base.GameROM1))
	if len(first) != 2 || first[517].Face != 9 {
		t.Fatalf("first game = %+v", first)
	}
	second := speakerFaces(table(base.GameROM2))
	if second[517].Face != 9 || second[7].Face != 2 {
		t.Fatalf("registry records lost: %+v", second)
	}
	if me := second[21]; !me.Tokens.Has(data.NPCTokenMe) {
		t.Fatalf("player key = %+v", me)
	}
	delete(registry, 517)
	if rec := speakerFaces(table(base.GameROM2))[517]; rec.Kind != data.NPCFigure || rec.Face != 4 {
		t.Fatalf("row record = %+v", rec)
	}
}
