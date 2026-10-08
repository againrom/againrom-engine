package game

import (
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"fmt"
)

// applyOriginalStructures installs the exact source roster and saved cell
// overlay. ALM is only an authored-script binding and terrain baseline.
func applyOriginalStructures(ms *Mission, source []sav.Building, present bool,
	table *mapload.Table, report *OriginalSaveResume, files ...*sav.File) error {
	return applyOriginalStructuresCurrent(ms, source, present, table, report, nil, files...)
}

func applyOriginalStructuresCurrent(ms *Mission, source []sav.Building, present bool,
	table *mapload.Table, report *OriginalSaveResume, current *currentActionData, files ...*sav.File) error {
	if !present {
		return nil
	}
	if ms == nil || ms.Map == nil || ms.World == nil {
		return fmt.Errorf("original structures: mission has no map/world")
	}
	subjects, err := currentStructureSubjects(current, ms, source)
	if err != nil {
		return err
	}
	byMapID := make(map[uint32][]int)
	for i, object := range ms.Map.Objects {
		if object.Field12 != 0 {
			byMapID[uint32(object.Field12)] = append(byMapID[uint32(object.Field12)], i)
		}
	}
	var structures []sim.Structure
	var sources []sim.SavedStructure
	var counts originalStructureCounts
	seen := make(map[uint32]bool)
	byKey := make(map[uint32]sim.StructureID)
	matched := make(map[int]bool)
	for i, s := range source {
		if subjects == nil && s.AuthoredID != 0 && seen[s.AuthoredID] {
			return fmt.Errorf("original structures: ambiguous authored ID %d", s.AuthoredID)
		}
		if s.AuthoredID != 0 {
			seen[s.AuthoredID] = true
		}
		targets := byMapID[s.AuthoredID]
		if subjects == nil && len(targets) > 1 {
			return fmt.Errorf("original structures: ambiguous map authored ID %d", s.AuthoredID)
		}
		id := sim.StructureID(len(ms.Map.Objects) + i)
		ss := sim.SavedStructure{SourceKey: s.Identity, ArchiveIndex: s.ArchiveIndex,
			AuthoredID: s.AuthoredID, Position: s.Position, RuntimeID: s.RuntimeID,
			Token0C: s.Token0C, Token0E: s.Token0E, Token18: s.Token18, Token1C: s.Token1C,
			Reference: s.Reference, Base52: s.Base52, Kind: s.Kind, Field46: s.Field46,
			Field48: s.Field48, Blocking: s.Blocking, Tavern9C: s.Tavern9C, Shop70: s.Shop70,
			OutpostWords: s.OutpostWords, OutpostRecords: s.OutpostRecords}
		if subjects != nil {
			row := subjects[s.Identity]
			id, ss.HasAuthored, ss.AuthoredIndex = row.ID, row.HasAuthored, row.AuthoredIndex
			if row.RuntimeID != nil && row.RuntimeID.Wire == s.RuntimeID {
				ss.RuntimeID = row.RuntimeID.Value
			}
			if row.HasAuthored {
				matched[int(row.AuthoredIndex)] = true
			}
		} else if len(targets) == 1 {
			id = sim.StructureID(targets[0])
			ss.HasAuthored, ss.AuthoredIndex = true, uint32(targets[0])
			matched[targets[0]] = true
		} else if s.AuthoredID == 0 {
			counts.Unbound++
		} else {
			counts.Unmatched++
		}
		switch s.Class {
		case "Building":
			ss.Class = sim.SavedBuilding
		case "Outpost":
			ss.Class = sim.SavedOutpost
			counts.Subclasses++
		case "Tavern":
			ss.Class = sim.SavedTavern
			counts.Subclasses++
		case "Shop":
			ss.Class = sim.SavedShop
			counts.Subclasses++
		default:
			return fmt.Errorf("original structures: unsupported archive class %q", s.Class)
		}
		if subjects != nil && subjects[s.Identity].Class.Generated() {
			ss.Class, ss.ArchiveIndex = subjects[s.Identity].Class, 0
		}
		ss.ID = id
		if _, exists := byKey[s.Identity]; exists || s.Identity == 0 {
			return fmt.Errorf("original structures: ambiguous source identity %#x", s.Identity)
		}
		byKey[s.Identity] = id
		structures = append(structures, sim.Structure{ID: id, Field42: s.Health, MaxHealth: s.MaxHealth,
			Col: int32(s.Col), Row: int32(s.Row), Width: s.Width, Height: s.Height, Attach: s.Attach, Blocking: s.Blocking})
		mapload.StructureUseMetadata(&structures[len(structures)-1], uint16(s.Kind), table)
		sources = append(sources, ss)
	}
	// TERR-STRUCT-068/072: terrain ingest has no Building attachment arm,
	// and archive construction does not attach. Actor occupancy is not static.
	grid := mapload.Passability(ms.Map)
	for i := range grid {
		// Original domain 2 reads static bit2 (scenery/border), not bit1.
		if grid[i]&2 != 0 || (i < len(ms.Map.Overlay) && ms.Map.Overlay[i] != 0) {
			grid[i] |= 8
		}
	}
	var cells []sim.SavedStructureCell
	if len(files) != 0 && files[0] != nil {
		f := files[0]
		rawCells, hasCells, err := f.StructureCells()
		if err != nil {
			return err
		}
		if !hasCells || f.World == nil {
			return fmt.Errorf("original structures: world cells absent")
		}
		last := make(map[uint16]int, len(rawCells))
		for i, c := range rawCells {
			last[c.Cell] = i
		}
		for i, c := range rawCells {
			if last[c.Cell] != i {
				continue
			}
			x := sim.SavedStructureCell{Cell: c.Cell, BaselineCost: c.BaselineCost, BaselineStatic: c.BaselineStatic}
			if c.BuildingKey != 0 {
				var ok bool
				x.ID, ok = byKey[c.BuildingKey]
				if !ok {
					return fmt.Errorf("original structures: cell %04x has unresolved Building identity %#x", c.Cell, c.BuildingKey)
				}
				x.HasStructure = true
			}
			cells = append(cells, x)
		}
		for _, b := range f.World.Blocks {
			x, y := b.Col(), b.Row()
			if x >= ms.Map.Width || y >= ms.Map.Height {
				continue
			}
			var block byte
			if b.Static&1 != 0 {
				block |= 1
			}
			if b.Static&2 != 0 {
				block |= 2
			}
			if b.Static&4 != 0 {
				block |= 8
			}
			grid[y*ms.Map.Width+x] = block
		}
	}
	if err := ms.World.ImportOriginalStructures(structures, sources, cells, grid); err != nil {
		return err
	}
	counts.Restored, counts.Absent = len(structures), len(ms.Map.Objects)-len(matched)
	if report != nil {
		report.Structures = counts
	}
	return nil
}

type originalStructureCounts struct {
	Restored, Subclasses, Unbound, Unmatched, Topology, Absent int
}
