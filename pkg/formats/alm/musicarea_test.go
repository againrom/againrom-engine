package alm

import (
	"encoding/binary"
	"testing"
)

func musicRecord(v ...int32) []byte {
	out := make([]byte, 0, 28)
	for _, x := range v {
		out = binary.LittleEndian.AppendUint32(out, uint32(x))
	}
	return out
}

func TestMusicAreasReadTheHeadThenTheAreas(t *testing.T) {
	m := &Map{}
	if _, _, ok := m.MusicAreas(); ok {
		t.Fatal("a map without type-12 data answered areas")
	}
	m.Extension[2] = append(musicRecord(0, 0, 0, 3, -1, -1, -1), musicRecord(12, 34, 5, 1, 2, -1, 16)...)
	head, areas, ok := m.MusicAreas()
	if !ok || head != (MusicArea{Themes: [4]int32{3, -1, -1, -1}}) {
		t.Fatalf("head %+v ok %v", head, ok)
	}
	if len(areas) != 1 || areas[0] != (MusicArea{X: 12, Y: 34, Radius: 5, Themes: [4]int32{1, 2, -1, 16}}) {
		t.Fatalf("areas %+v", areas)
	}
	m.Extension[2] = m.Extension[2][:40]
	if _, _, ok := m.MusicAreas(); ok {
		t.Fatal("a partial record decoded")
	}
}
