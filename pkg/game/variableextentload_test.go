package game

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/vfs"
)

// A 40x40 map with water in columns 14..16 and a 3x2 bridge record of kind
// 0x21 across it at (14,10), authored ID 61. The definition table gives kind
// 0x21 the shipped 1x1 blocking row, so only the record's extension makes
// the rectangle.
const (
	extentBridgeCol, extentBridgeRow = 14, 10
	extentBridgeWidth                = 3
	extentBridgeHeight               = 2
	extentBridgeID                   = 61
	// extentWater is a tile word in the water index range: it closes the
	// cell to a ground mover.
	extentWater = 0x0200
)

func variableExtentFront(t *testing.T, objects []synth.ALMObject) *FrontEnd {
	t.Helper()
	f := missionFrontEnd(t)
	tiles := make([]byte, 2*40*40)
	for y := 0; y < 40; y++ {
		for x := 14; x <= 16; x++ {
			binary.LittleEndian.PutUint16(tiles[2*(y*40+x):], extentWater)
		}
	}
	b := synth.ALM(synth.ALMOptions{Width: 40, Height: 40, Type1Payload: tiles, Objects: objects})
	path := filepath.Join(t.TempDir(), ScenarioArchive)
	if err := os.WriteFile(path, synth.Archive([]synth.File{{Path: "10.alm", Data: b}, {Path: "npc.reg", Data: synth.NPCReg(nil)}}), 0o600); err != nil {
		t.Fatal(err)
	}
	fs, err := vfs.Open([]string{path}, nil)
	if err != nil {
		t.Fatal(err)
	}
	f.Archives.Containers = fs
	buildings := make(dbCollection, 34)
	buildings[1] = dbEntry{name: "Fixture", params: []int32{1, 1, 0, 100, 1, 1}}
	buildings[0x21] = dbEntry{name: "Bridge", params: []int32{1, 1, 0, 0, 1, 1}}
	f.Table = &mapload.Table{Buildings: buildings}
	f.SetDeterministicFrames(true)
	return f
}

func variableExtentBridgeObject() synth.ALMObject {
	return synth.ALMObject{X: extentBridgeCol << 8, Y: extentBridgeRow << 8, Kind: 0x21, Field12: extentBridgeID,
		Ext: []byte{extentBridgeWidth, 0, 0, 0, extentBridgeHeight, 0, 0, 0}}
}

func variableExtentRow(width, height uint8, attach, blocking uint32) sav.Building {
	return sav.Building{Class: "Building", AuthoredID: extentBridgeID, Col: extentBridgeCol, Row: extentBridgeRow,
		Kind: 0x21, Width: width, Height: height, Attach: attach, Blocking: blocking}
}

// variableExtentCells is one cell record per cell of the rectangle at the
// bridge anchor, each naming the fixture's first Building identity, 700.
func variableExtentCells(width, height int) []sav.DocumentCellData {
	var cells []sav.DocumentCellData
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			cells = append(cells, sav.DocumentCellData{Cell: uint16(extentBridgeRow+y)<<8 | uint16(extentBridgeCol+x), Building: 700})
		}
	}
	return cells
}

func variableExtentBridge(t *testing.T, w *sim.World) (sim.Structure, []sim.SavedStructureCell) {
	t.Helper()
	for _, s := range w.Structures() {
		if s.Col == extentBridgeCol && s.Row == extentBridgeRow {
			_, cells, _ := w.SavedStructures()
			var bound []sim.SavedStructureCell
			for _, c := range cells {
				if c.HasStructure && c.ID == s.ID {
					bound = append(bound, c)
				}
			}
			return s, bound
		}
	}
	t.Fatal("no structure at the bridge anchor")
	return sim.Structure{}, nil
}

func variableExtentDeckClosed(w *sim.World) int {
	block := w.CurrentPolicy().Terrain.Block
	closed := 0
	for y := extentBridgeRow; y < extentBridgeRow+extentBridgeHeight; y++ {
		for x := extentBridgeCol; x < extentBridgeCol+extentBridgeWidth; x++ {
			if block[y*40+x]&1 != 0 {
				closed++
			}
		}
	}
	return closed
}

// TestLoadRebuildsACollapsedVariableExtentFromTheMap: a document holding the
// map's bridge at 1x1 loads it with the map's rectangle, every cell bound and
// the deck open, as the map's fresh build gives it; the same document with
// the full rectangle, and a 1x1 document against a map without that bridge,
// load as saved.
func TestLoadRebuildsACollapsedVariableExtentFromTheMap(t *testing.T) {
	t.Run("collapsed bridge takes the map's rectangle and an open deck", func(t *testing.T) {
		f := variableExtentFront(t, []synth.ALMObject{variableExtentBridgeObject()})
		fresh, err := StartMission(f.Archives.Containers, 10, f.Table, mapload.DifficultyNormal, nil)
		if err != nil {
			t.Fatal(err)
		}
		if closed := variableExtentDeckClosed(fresh.World); closed != 0 {
			t.Fatalf("fresh build closes %d deck cells", closed)
		}
		save := structureFixtureSave([]sav.Building{variableExtentRow(1, 1, 1, 1)}, variableExtentCells(1, 1)...)
		ms, _, err := loadOriginalMission(f, save)
		if err != nil {
			t.Fatal(err)
		}
		s, cells := variableExtentBridge(t, ms.World)
		if s.Width != extentBridgeWidth || s.Height != extentBridgeHeight || s.Attach != ^uint32(0) || s.Blocking != 0 {
			t.Fatalf("loaded bridge %+v, want %dx%d, all attached, none blocking", s, extentBridgeWidth, extentBridgeHeight)
		}
		if len(cells) != extentBridgeWidth*extentBridgeHeight {
			t.Fatalf("loaded bridge binds %d cells, want %d", len(cells), extentBridgeWidth*extentBridgeHeight)
		}
		if closed := variableExtentDeckClosed(ms.World); closed != 0 {
			t.Fatalf("loaded deck has %d closed cells", closed)
		}
		sources, _, _ := ms.World.SavedStructures()
		if len(sources) != 1 || sources[0].Blocking != 0 {
			t.Fatalf("saved source %+v, want blocking 0", sources)
		}
	})
	t.Run("full rectangle loads as saved", func(t *testing.T) {
		f := variableExtentFront(t, []synth.ALMObject{variableExtentBridgeObject()})
		save := structureFixtureSave([]sav.Building{variableExtentRow(extentBridgeWidth, extentBridgeHeight, ^uint32(0), 0)},
			variableExtentCells(extentBridgeWidth, extentBridgeHeight)...)
		ms, _, err := loadOriginalMission(f, save)
		if err != nil {
			t.Fatal(err)
		}
		s, cells := variableExtentBridge(t, ms.World)
		if s.Width != extentBridgeWidth || s.Height != extentBridgeHeight || s.Attach != ^uint32(0) || s.Blocking != 0 || len(cells) != 6 {
			t.Fatalf("loaded bridge %+v with %d cells, want the saved 3x2 with 6", s, len(cells))
		}
		// The document holds no Block records, so the LOAD reads water under
		// the deck, as it did before the rebuild existed.
		if closed := variableExtentDeckClosed(ms.World); closed != 6 {
			t.Fatalf("full-rectangle deck has %d closed cells, want the document's 6", closed)
		}
	})
	t.Run("a document from another map loads as saved", func(t *testing.T) {
		other := synth.ALMObject{X: extentBridgeCol << 8, Y: extentBridgeRow << 8, Kind: 1, Field12: extentBridgeID}
		f := variableExtentFront(t, []synth.ALMObject{other})
		save := structureFixtureSave([]sav.Building{variableExtentRow(1, 1, 1, 1)}, variableExtentCells(1, 1)...)
		ms, _, err := loadOriginalMission(f, save)
		if err != nil {
			t.Fatal(err)
		}
		s, cells := variableExtentBridge(t, ms.World)
		if s.Width != 1 || s.Height != 1 || s.Attach != 1 || s.Blocking != 1 || len(cells) != 1 {
			t.Fatalf("loaded %+v with %d cells, want the saved 1x1 with 1", s, len(cells))
		}
		if closed := variableExtentDeckClosed(ms.World); closed != 6 {
			t.Fatalf("water under the record has %d closed cells, want 6", closed)
		}
	})
}
