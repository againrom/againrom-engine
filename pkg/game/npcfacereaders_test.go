package game

import (
	"fmt"
	"strings"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/data"
	"againrom/pkg/mapload"
)

// npcFaceFixture is a table with a catalog record, a catalog record on a
// fallback key, and a Humans row naming key 517 and the player's key 21.
func npcFaceFixture(g base.Game) *mapload.Table {
	params := make([]int32, 38)
	for i := range params {
		params[i] = -1
	}
	params[16], params[17], params[18], params[24] = 24, 4, 1, 517
	humans := &fixtureCollection{names: []string{"", "Mage"}, params: [][]int32{nil, params}, raw: [][]byte{nil, nil}}
	registry := map[int32]data.NPCFace{
		7:                             {Kind: data.NPCPortrait, Face: 2},
		517:                           {Kind: data.NPCPortrait, Face: 9},
		data.FallbackSpeakerFirst + 1: {Kind: data.NPCPortrait, Face: 5},
	}
	return readUnder(g, &mapload.Table{Humans: humans, NPCCatalog: &data.NPCCatalog{Faces: registry}})
}

func npcFaceProbeKeys() []int32 {
	return []int32{0, 7, 21, 517, 600, data.FallbackSpeakerFirst, data.FallbackSpeakerFirst + 1,
		data.FallbackSpeakerLast, data.FallbackSpeakerLast + 1}
}

// npcFaceAnswers is what one reader answers for every probe key.
func npcFaceAnswers(faces map[int32]data.NPCFace) string {
	var b strings.Builder
	for _, k := range npcFaceProbeKeys() {
		rec, held := faces[k]
		fmt.Fprintf(&b, "%d:%v:%d:%d:%v;", k, held, rec.Kind, rec.Face, rec.Tokens)
	}
	return b.String()
}

// The answers below were recorded from the readers as they stood before they
// shared one source: the person window read the catalog's faces, the NPC page
// and the game read the speaker table, the mod mission and the scenario
// announcements read the catalog's portraits.
const (
	baseFirstAnswers  = "0:false:0:0:0;7:true:2:2:0;21:false:0:0:0;517:true:2:9:0;600:false:0:0:0;81:false:0:0:0;82:true:2:5:0;335:false:0:0:0;336:false:0:0:0;"
	baseSecondCatalog = baseFirstAnswers
	baseSecondPage    = "0:false:0:0:0;7:true:2:2:0;21:true:0:0:9;517:true:2:9:0;600:false:0:0:0;81:true:1:1:32790;82:true:2:5:0;335:true:1:255:32790;336:false:0:0:0;"
)

// TestNPCFaceReadersShareOneSource holds each reader's answer for every probe
// key against the answer recorded before the change.
func TestNPCFaceReadersShareOneSource(t *testing.T) {
	first := npcFaceFixture(base.GameROM1)
	second := npcFaceFixture(base.GameROM2)
	cases := []struct {
		name, got, want string
	}{
		{"first game person window", npcFaceAnswers(speakerFaces(first)), baseFirstAnswers},
		{"first game NPC page", npcFaceAnswers(speakerFaces(first)), baseFirstAnswers},
		{"first game mod mission and announcements", npcFaceAnswers(CatalogFaces(first)), baseFirstAnswers},
		{"second game NPC page", npcFaceAnswers(speakerFaces(second)), baseSecondPage},
		{"second game person window shows the page's faces", npcFaceAnswers(speakerFaces(second)), baseSecondPage},
		{"second game mod mission and announcements", npcFaceAnswers(CatalogFaces(second)), baseSecondCatalog},
	}
	for _, tc := range cases {
		if tc.got != tc.want {
			t.Errorf("%s:\n got %s\nwant %s", tc.name, tc.got, tc.want)
		}
	}
	if CatalogFaces(nil) != nil || speakerFaces(nil) != nil {
		t.Fatal("a nil table has no faces")
	}
}
