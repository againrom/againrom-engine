package game_test

import (
	"encoding/binary"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/game"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
)

func TestMapLoadSeparatesBlockedGroundFromRuntimeFire(t *testing.T) {
	const width = 32
	tiles := make([]uint16, width*width)
	overlay := make([]byte, len(tiles))
	tiles[16*width+16], overlay[16*width+16] = 0x2000, 1
	dead := &terrain.StaticClass{Frame: &terrain.StaticFrame{Width: 2, Height: 2}}
	live := &terrain.StaticClass{Frame: &terrain.StaticFrame{Width: 2, Height: 2}, Dead: dead}
	statics := &terrain.StaticSet{}
	statics.Classes[1] = live
	payload := make([]byte, 2*len(tiles))
	for i, word := range tiles {
		binary.LittleEndian.PutUint16(payload[2*i:], word)
	}
	b := synth.ALM(synth.ALMOptions{Width: width, Height: width, Type1Payload: payload, Overlay: overlay})
	mv, err := game.LoadMapViewer(&terrain.Tileset{}, b, "blocked ground", game.Markers{}, game.StaticLayer{Set: statics, Art: true}, game.StructureLayer{})
	if err != nil {
		t.Fatal(err)
	}
	if mv.Map.Tiles[16*width+16] != 0x2000 || mapload.Passability(mv.Map)[16*width+16]&1 == 0 {
		t.Fatal("render load changed the simulation's impassable cell")
	}
	if _, n := mv.Viewer.ScorchedScenery(); n != 0 {
		t.Fatalf("new map draws %d burned objects from raw impassability", n)
	}
	mv.Viewer.SetScorchedCells([]uint16{0x1010})
	if ground, objects := mv.Viewer.ScorchedScenery(); ground != 1 || objects != 1 {
		t.Fatalf("runtime fire absent: ground=%d objects=%d", ground, objects)
	}
	mv.Viewer.SetScorchedCells(nil)
	if ground, objects := mv.Viewer.ScorchedScenery(); ground != 0 || objects != 0 {
		t.Fatalf("cleared fire did not restore clean ground: %d/%d", ground, objects)
	}
}
