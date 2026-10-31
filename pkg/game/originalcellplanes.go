package game

import (
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// importOriginalCellPlanes runs only on a private original-LOAD candidate.
// SAV-LOAD-057/TERR-PASS-053 construct ALM cost/height first, then overlay only
// static/dynamic Blocks. SAV-CELLLOAD-109/110 do not recompute these planes or
// copy Cell.Cost into current cost. Later ordinary cell operations own those
// writes. This does not replay the still-unknown pre-entry bootstrap tick.
func importOriginalCellPlanes(ms *Mission, source entrySource) error {
	if ms == nil || ms.World == nil || ms.savedDocument == nil || ms.savedDocument.Document == nil || ms.savedDocument.Document.World == nil {
		return nil
	}
	base, err := readOriginalCellPlanes(ms.Map, source)
	if err != nil || base == nil {
		return err
	}
	blocks := make([]sim.SavedActorBlock, 0, len(ms.savedDocument.Document.World.Blocks))
	for _, block := range ms.savedDocument.Document.World.Blocks {
		blocks = append(blocks, sim.SavedActorBlock{Cell: block.Cell, Dyn: block.Dyn, Static: block.Static})
	}
	planes := ms.World.CellPlanes(sim.CellPlaneStart{Base: base, Document: true, Blocks: blocks})
	if err := ms.World.ImportOriginalCellPlanes(planes); err != nil {
		return err
	}
	// MOVE-086: the original keeps terrain ingest's byte in a loaded layered
	// cell until a later recompute. The import has no cost plane, so the engine
	// rewrites those cells here.
	ms.World.RewriteImportedLayerCosts()
	return nil
}

func readOriginalCellPlanes(m *alm.Map, source entrySource) (*sim.SavedCellPlanes, error) {
	if source == nil {
		return nil, nil
	}
	costs, ok, err := terrainCostTable(source, filesOf("").planeCosts)
	if err != nil || !ok {
		return nil, err // Preserve absent authority on partial/custom inputs.
	}
	return mapload.OriginalTerrainPlanes(m, costs)
}
