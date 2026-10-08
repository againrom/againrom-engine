package game

import (
	"fmt"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

type currentStructureBinding struct {
	ID            sim.StructureID
	Object        uint16
	SourceKey     uint32
	Class         sim.SavedStructureClass
	Ordinal       uint16
	HasAuthored   bool
	AuthoredIndex uint32
	RuntimeID     *sim.ActorRuntimeCoordinate `json:",omitempty"`
}

func captureCurrentStructureBindings(doc *sav.DocumentData, w *sim.World, a *currentActionData) error {
	source, _, present := w.SavedStructures()
	if !present {
		return nil
	}
	rows := make([]currentStructureBinding, len(source))
	for i, s := range source {
		object, err := currentTypedIdentityIndex(doc, savedStructureClass(s.Class), s.SourceKey)
		if err != nil {
			return err
		}
		rows[i] = currentStructureBinding{ID: s.ID, Object: object, SourceKey: s.SourceKey, Class: s.Class,
			Ordinal: uint16(i), HasAuthored: s.HasAuthored, AuthoredIndex: s.AuthoredIndex}
	}
	a.StructureBindings = &rows
	return validateCurrentStructureBindings(doc, a)
}

func validateCurrentStructureBindings(doc *sav.DocumentData, a *currentActionData) error {
	if a.StructureBindings == nil {
		return nil
	}
	if doc == nil || doc.World == nil || len(*a.StructureBindings) > 32767 {
		return fmt.Errorf("current structures: invalid world/population")
	}
	roots := map[uint16]bool{}
	for _, object := range doc.World.Buildings {
		roots[object] = true
	}
	if len(roots) != len(*a.StructureBindings) {
		return fmt.Errorf("current structures: incomplete root coverage")
	}
	used, keys := map[uint16]bool{}, map[uint32]bool{}
	coordinates := map[uint16]currentArchiveCoordinate{}
	a.structureGenerated = map[uint32]bool{}
	for _, row := range a.ArchiveCoordinates {
		coordinates[row.Object] = row
	}
	for i, row := range *a.StructureBindings {
		if row.Object == 0 || int(row.Object) > len(doc.Objects) || !roots[row.Object] || used[row.Object] ||
			row.SourceKey == 0 || keys[row.SourceKey] || row.Ordinal != uint16(i) || i > 0 && row.ID <= (*a.StructureBindings)[i-1].ID ||
			row.Class < sim.SavedBuilding || row.Class > sim.GeneratedShop ||
			row.HasAuthored && uint32(row.ID) != row.AuthoredIndex || !row.HasAuthored && row.AuthoredIndex != 0 {
			return fmt.Errorf("current structures: aliased or invalid binding")
		}
		record := &doc.Objects[row.Object-1]
		if row.RuntimeID != nil && (row.RuntimeID.Wire == 0 || row.RuntimeID.Wire > 65535 || row.RuntimeID.Wire == row.RuntimeID.Value) {
			return fmt.Errorf("current structures: invalid runtime coordinate")
		}
		key, err := savedStructureValue(record, "Identity")
		if err != nil || record.Class != savedStructureClass(row.Class) || key != row.SourceKey {
			return fmt.Errorf("current structures: record/class/key differs")
		}
		for j, other := range doc.Objects {
			for _, v := range other.Values {
				if j != int(row.Object)-1 && (v.Name == "Identity" || v.Name == "This") && v.Value == row.SourceKey {
					return fmt.Errorf("current structures: cross-object identity collision")
				}
			}
		}
		coordinate, ok := coordinates[row.Object]
		if !ok || coordinate.Generated != row.Class.Generated() || row.Class.Generated() &&
			(coordinate.Native != 0 || coordinate.ModeAnchor == nil) {
			return fmt.Errorf("current structures: missing construction coordinate")
		}
		if row.Class.Generated() {
			a.structureGenerated[row.SourceKey] = *coordinate.ModeAnchor == currentArchiveModeAnchor(record)
		}
		used[row.Object], keys[row.SourceKey] = true, true
	}
	return nil
}

func currentStructureSubjects(a *currentActionData, ms *Mission, source []sav.Building) (map[uint32]currentStructureBinding, error) {
	if a == nil || a.StructureBindings == nil {
		return nil, nil
	}
	if len(source) != len(*a.StructureBindings) {
		return nil, fmt.Errorf("current structures: physical roster differs")
	}
	byKey := map[uint32]currentStructureBinding{}
	for _, row := range *a.StructureBindings {
		if row.Class.Generated() && !a.structureGenerated[row.SourceKey] {
			row.Class = row.Class.BaseClass()
		}
		byKey[row.SourceKey] = row
	}
	for _, s := range source {
		row, ok := byKey[s.Identity]
		if !ok || savedStructureClass(row.Class) != s.Class {
			return nil, fmt.Errorf("current structures: physical subject differs")
		}
		if row.HasAuthored && (uint64(row.AuthoredIndex) >= uint64(len(ms.Map.Objects)) || s.AuthoredID == 0 ||
			uint32(ms.Map.Objects[row.AuthoredIndex].Field12) != s.AuthoredID) {
			return nil, fmt.Errorf("current structures: authored coordinate differs")
		}
	}
	return byKey, nil
}
