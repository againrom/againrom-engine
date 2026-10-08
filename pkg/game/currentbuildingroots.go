package game

import (
	"fmt"
	"slices"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func (b *generatedDocumentBuilder) currentStructureRoots(w *sim.World) (bool, error) {
	sources, cells, present := w.SavedStructures()
	if !present {
		return false, nil
	}
	live := w.Structures()
	if len(live) != len(sources) {
		return false, fmt.Errorf("current Building live/source roster differs")
	}
	changed := false
	for i, source := range sources {
		if source.ID != live[i].ID || source.SourceKey == 0 {
			return false, fmt.Errorf("current Building lacks exact source ownership")
		}
		class := savedStructureClass(source.Class)
		index, err := currentTypedIdentityIndex(&b.doc, class, source.SourceKey)
		if err != nil {
			return false, err
		}
		for n, record := range b.doc.Objects {
			for _, value := range record.Values {
				if (value.Name == "Identity" || value.Name == "This") && value.Value == source.SourceKey && int(index) != n+1 {
					return false, fmt.Errorf("current Building has a cross-object Identity/This collision")
				}
			}
		}
		if index == 0 {
			record := mustNewRecord(class)
			mustSetValue(&record, "Identity", source.SourceKey)
			if err := projectSavedStructureRecord(&record, source, live[i]); err != nil {
				return false, err
			}
			index, err = b.append(record)
			if err != nil {
				return false, err
			}
			changed = true
		}
		if !slices.Contains(b.doc.World.Buildings, index) {
			b.doc.World.Buildings = append(b.doc.World.Buildings, index)
			changed = true
		}
	}
	_, currentCells, _, _ := w.SavedActorMotions()
	for _, cell := range cells {
		index := -1
		for i, prior := range b.doc.World.Cells {
			if prior.Cell == cell.Cell {
				index = i
			}
		}
		// DIV-2476.
		payload := sav.DocumentCellData{Cell: cell.Cell, Cost: cell.BaselineCost, Static: cell.BaselineStatic}
		complete := false
		for _, current := range currentCells {
			if current.Cell == cell.Cell {
				payload = sav.DocumentCellFromPayload(current.Cell, current.Payload)
				complete = true
				break
			}
		}
		if index < 0 {
			b.doc.World.Cells = append(b.doc.World.Cells, payload)
			changed = true
		} else if complete && b.doc.World.Cells[index] != payload {
			b.doc.World.Cells[index] = payload
			changed = true
		}
	}
	return changed, nil
}
