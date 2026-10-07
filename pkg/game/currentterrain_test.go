package game

import (
	"testing"
	"testing/fstest"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func TestCurrentTerrainOwnsModeledFieldsWithoutRawPlanes(t *testing.T) {
	f, _, base := partialCurrentGraph(t)
	terrain := base.CurrentPolicy().Terrain
	terrain.Block[0], terrain.Cost[0], terrain.Height[0] = 3, 43, 19
	w, err := sim.NewRelatedWorld(11, base.Bounds(), sim.ModeCanonical, terrain, nil, nil, sim.Relations{})
	if err != nil {
		t.Fatal(err)
	}
	terrain.Block[0] = 11
	if err := w.ImportOriginalStructures(nil, nil, nil, terrain.Block); err != nil {
		t.Fatal(err)
	}
	hash := w.Hash()
	for _, source := range []entrySource{nil, fstest.MapFS{}, f.Archives.Containers} {
		p, err := currentSpatialPlanes(w, f.live.mission.state.Map, source)
		if err != nil || p.Cost[0] != 43 || p.Static[0]&7 != 7 || p.Dynamic[0]&7 != 7 || p.Height[0] != 19 {
			t.Fatal("constructor replaced current modeled terrain", p, err)
		}
		if w.CurrentPolicy().PlaneCarrier || w.Hash() != hash {
			t.Fatal("ordinary construction changed native authority")
		}
	}
	if _, err := currentSpatialPlanes(w, f.live.mission.state.Map, fstest.MapFS{worldPrefix + "data/map.reg": &fstest.MapFile{Data: []byte("bad registry")}}); err == nil {
		t.Fatal("malformed installed terrain was accepted")
	}
	doc := sav.DocumentData{World: &sav.DocumentWorldData{Blocks: []sav.BlockRecord{{Cell: 0, Static: 0x80, Dyn: 0x90}, {Cell: 0xffff, Static: 0x52, Dyn: 0x53}}}}
	projectCurrentTerrain(&doc, w, false)
	if doc.World.Blocks[0].Static != 0x87 || doc.World.Blocks[0].Dyn != 0x97 || doc.World.Blocks[len(doc.World.Blocks)-1] != (sav.BlockRecord{Cell: 0xffff, Static: 0x52, Dyn: 0x53}) {
		t.Fatal("modeled obstacle or unmodeled block residue changed")
	}
}
