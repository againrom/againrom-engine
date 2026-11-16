package game

import (
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/mod"
	"againrom/pkg/sim"
)

// importOriginalCellPlanes runs only on a private original-LOAD candidate.
// SAV-LOAD-057/TERR-PASS-053 construct ALM cost/height first, then overlay only
// static/dynamic Blocks. SAV-CELLLOAD-109/110 do not recompute these planes or
// copy Cell.Cost into current cost. Later ordinary cell operations own those
// writes. This does not replay the still-unknown pre-entry bootstrap tick.
// A held document, which no claim admits as complete, supplies the same block
// rows, so its saved cloud areas bind to the planes.
func importOriginalCellPlanes(ms *Mission, source entrySource, groups *mod.TerrainData, held *heldDocument) error {
	if ms == nil || ms.World == nil || ms.savedDocument == nil {
		return nil
	}
	doc := ms.savedDocument.Document
	if doc == nil && held != nil && held.doc.World != nil && len(held.doc.World.Effects) > 0 {
		doc = held.doc
	}
	if doc == nil || doc.World == nil {
		return nil
	}
	base, err := readOriginalCellPlanes(ms.Map, source, groups)
	if err != nil || base == nil {
		return err
	}
	blocks := make([]sim.SavedActorBlock, 0, len(doc.World.Blocks))
	for _, block := range doc.World.Blocks {
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

func readOriginalCellPlanes(m *alm.Map, source entrySource, groups *mod.TerrainData) (*sim.SavedCellPlanes, error) {
	if source == nil {
		return nil, nil
	}
	costs, ok, err := terrainCostTable(source, filesOf("").planeCosts)
	if err != nil || !ok {
		return nil, err // Preserve absent authority on partial/custom inputs.
	}
	return mapload.OriginalTerrainPlanesWith(m, costs, groups)
}
