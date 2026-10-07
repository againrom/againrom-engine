package sim

import (
	"encoding/binary"
	"fmt"
	"slices"
	"sort"
)

// SavedStructureClass tags the archive class independently of the graphics kind.
// Subclasses use the base physical/area/inspection policy, not invented services.
type SavedStructureClass uint8

const (
	SavedBuilding SavedStructureClass = iota + 1
	SavedOutpost
	SavedTavern
	SavedShop
	GeneratedBuilding
	GeneratedOutpost
	GeneratedTavern
	GeneratedShop
)

func (c SavedStructureClass) Generated() bool { return c >= GeneratedBuilding && c <= GeneratedShop }
func (c SavedStructureClass) BaseClass() SavedStructureClass {
	if c.Generated() {
		return c - 4
	}
	return c
}

// SavedStructure is typed retained source state beside a live Structure. Health,
// geometry and footprint masks belong to the live Structure. SourceKey is
// never a native handle. AuthoredIndex binds a compiled ALM script reference only
// when HasAuthored is true; absent ALM objects cannot resurrect on that path.
type SavedStructure struct {
	ID                        StructureID
	Class                     SavedStructureClass
	SourceKey                 uint32
	ArchiveIndex              uint16
	AuthoredID, AuthoredIndex uint32
	HasAuthored               bool
	Position                  [12]byte
	RuntimeID                 uint32
	Token0C                   uint8
	Token0E, Token18          uint16
	Token1C, Reference        uint32
	Base52                    [22]byte
	Kind                      uint8
	Field46                   uint16
	Field48                   uint8
	Blocking                  uint32
	Tavern9C, Shop70          uint32
	OutpostWords              [4]uint32
	OutpostRecords            [][8]byte
}

// SavedStructureCell is the final cell overlay, in ascending packed-key order.
// A present cell with HasStructure=false preserves an explicit null overwrite.
// Baselines are retained for future recomputation, not mistaken for a current
// block plane. Dynamic actor occupancy remains the simulation's own state.
type SavedStructureCell struct {
	Cell                         uint16
	BaselineCost, BaselineStatic uint8
	ID                           StructureID
	HasStructure                 bool
}

func cloneSavedStructures(src []SavedStructure) []SavedStructure {
	out := slices.Clone(src)
	for i := range out {
		out[i].OutpostRecords = slices.Clone(out[i].OutpostRecords)
	}
	return out
}

// SavedStructures distinguishes absent legacy state from a saved empty roster.
func (w *World) SavedStructures() ([]SavedStructure, []SavedStructureCell, bool) {
	return cloneSavedStructures(w.savedStructures), slices.Clone(w.savedStructureCells), w.hasSavedStructures
}

func validateSavedStructures(structures []Structure, source []SavedStructure, cells []SavedStructureCell) error {
	if len(structures) != len(source) {
		return fmt.Errorf("saved structures: roster/source count mismatch")
	}
	keys := make(map[uint32]bool, len(source))
	archives := make(map[uint16]bool, len(source))
	authored := make(map[uint32]bool, len(source))
	authoredIDs := make(map[uint32]bool, len(source))
	for i, s := range source {
		if s.ID != structures[i].ID || (i > 0 && s.ID <= source[i-1].ID) {
			return fmt.Errorf("saved structures: invalid native ordering at %d", i)
		}
		if s.Class < SavedBuilding || s.Class > GeneratedShop || s.SourceKey == 0 || keys[s.SourceKey] || (!s.Class.Generated() && s.ArchiveIndex == 0) || (s.Class.Generated() && s.ArchiveIndex != 0) || s.ArchiveIndex != 0 && archives[s.ArchiveIndex] {
			return fmt.Errorf("saved structures: invalid class/source identity at %d", i)
		}
		keys[s.SourceKey], archives[s.ArchiveIndex] = true, true
		if (!s.HasAuthored && s.AuthoredIndex != 0) ||
			(s.HasAuthored && (s.AuthoredID == 0 || authored[s.AuthoredIndex] || uint32(s.ID) != s.AuthoredIndex)) ||
			(s.AuthoredID != 0 && authoredIDs[s.AuthoredID]) {
			return fmt.Errorf("saved structures: ambiguous authored binding at %d", i)
		}
		if s.HasAuthored {
			authored[s.AuthoredIndex] = true
		}
		if s.AuthoredID != 0 {
			authoredIDs[s.AuthoredID] = true
		}
		st := structures[i]
		if st.Col != int32(s.Position[0]) || st.Row != int32(s.Position[1]) {
			return fmt.Errorf("saved structures: position disagrees at %d", i)
		}
		if s.Base52[14] != st.Width || s.Base52[15] != st.Height || binary.LittleEndian.Uint32(s.Base52[18:]) != s.Blocking {
			return fmt.Errorf("saved structures: overlapping base fields disagree at %d", i)
		}
		if s.Class.BaseClass() != SavedOutpost && (s.OutpostWords != [4]uint32{} || len(s.OutpostRecords) != 0) {
			return fmt.Errorf("saved structures: Outpost state on another class")
		}
		if s.Class.BaseClass() != SavedShop && s.Shop70 != 0 {
			return fmt.Errorf("saved structures: Shop state on another class")
		}
		if s.Class.BaseClass() != SavedTavern && s.Tavern9C != 0 {
			return fmt.Errorf("saved structures: Tavern state on another class")
		}
	}
	for i, c := range cells {
		if i > 0 && c.Cell <= cells[i-1].Cell {
			return fmt.Errorf("saved structures: cell order is not canonical")
		}
		if (c.HasStructure && indexOfStructure(structures, c.ID) < 0) || (!c.HasStructure && c.ID != 0) {
			return fmt.Errorf("saved structures: invalid cell %04x binding", c.Cell)
		}
	}
	return nil
}

// ConstructSavedStructures attaches semantic authority to the existing fresh
// constructor roster. Its geometry and attachment mask already own the grid;
// this never applies LOAD's terrain replacement or replays attachment.
func (w *World) ConstructSavedStructures(source []SavedStructure, cells []SavedStructureCell) error {
	if w == nil || w.tick != 0 || w.hasSavedStructures {
		return fmt.Errorf("saved structures: constructor requires a fresh unbound world")
	}
	for _, s := range source {
		if !s.Class.Generated() {
			return fmt.Errorf("saved structures: constructor requires generated provenance")
		}
	}
	if err := validateSavedStructures(w.structures, source, cells); err != nil {
		return err
	}
	w.savedStructures, w.savedStructureCells, w.hasSavedStructures = cloneSavedStructures(source), slices.Clone(cells), true
	return nil
}

// ImportOriginalStructures replaces the roster and its explicit cell occupancy
// atomically on an unadvanced construction candidate. The caller supplies a
// terrain-only baseline with the saved static block overlay already applied.
// It deliberately never invokes the constructor footprint attachment routine.
func (w *World) ImportOriginalStructures(structures []Structure, source []SavedStructure, cells []SavedStructureCell, grid []byte) error {
	if w == nil || w.tick != 0 {
		return fmt.Errorf("saved structures: requires an unadvanced world")
	}
	if len(grid) != len(w.grid) {
		return fmt.Errorf("saved structures: block plane extent mismatch")
	}
	for _, b := range grid {
		if b&(gridReserved&^blockStaticObject) != 0 {
			return fmt.Errorf("saved structures: reserved block bits")
		}
	}
	cs, ss, cc := slices.Clone(structures), cloneSavedStructures(source), slices.Clone(cells)
	sort.Slice(cs, func(i, j int) bool { return cs[i].ID < cs[j].ID })
	sort.Slice(ss, func(i, j int) bool { return ss[i].ID < ss[j].ID })
	// Input cells are archive-ordered; duplicate keys have last-write semantics.
	last := make(map[uint16]SavedStructureCell, len(cc))
	for _, c := range cc {
		last[c.Cell] = c
	}
	cc = cc[:0]
	for _, c := range last {
		cc = append(cc, c)
	}
	sort.Slice(cc, func(i, j int) bool { return cc[i].Cell < cc[j].Cell })
	if err := validateSavedStructures(cs, ss, cc); err != nil {
		return err
	}
	w.structures, w.savedStructures, w.savedStructureCells = cs, ss, cc
	w.hasSavedStructures = true
	w.grid = slices.Clone(grid)
	w.rebuildStructureSlots()
	return nil
}
