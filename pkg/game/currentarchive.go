package game

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"slices"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

type currentArchiveCoordinate struct {
	Object     uint16
	Native     uint16
	Generated  bool      `json:",omitempty"`
	ModeAnchor *[32]byte `json:",omitempty"`
}

type currentActorGroupCoordinate struct {
	Object uint16
	Player uint16
	Inline uint32
	Native uint32
	Anchor [32]byte
}

func currentGroupSiteAnchor(doc *sav.DocumentData, site currentGroupSite) [32]byte {
	var fields []sav.DocumentValueData
	var raw []sav.DocumentRawData
	if site.Player != 0 && int(site.Player) <= len(doc.Objects) && uint64(site.Inline) < uint64(len(doc.Objects[site.Player-1].Groups)) {
		g := &doc.Objects[site.Player-1].Groups[site.Inline]
		fields, raw = g.Values, g.Raw
	}
	value, _ := json.Marshal(struct {
		Values []sav.DocumentValueData
		Raw    []sav.DocumentRawData
	}{fields, raw})
	return sha256.Sum256(value)
}

type currentGroupSite struct {
	Player uint16
	Inline uint32
}

// The final occurrence owns an actor; repeated Player roots do not replay
// their Groups. This is the same relationship exposed by ActorGraph.
func currentActorGroupSites(doc *sav.DocumentData) map[uint16]currentGroupSite {
	out, seen := map[uint16]currentGroupSite{}, map[uint16]bool{}
	for _, player := range doc.Players {
		if player == 0 || seen[player] {
			continue
		}
		seen[player] = true
		for inline, group := range doc.Objects[player-1].Groups {
			actors, _ := savedObjectRefs(&group, "Actors")
			for _, actor := range actors {
				if actor != 0 {
					out[actor] = currentGroupSite{player, uint32(inline)}
				}
			}
		}
	}
	return out
}

func currentArchiveModeAnchor(r *sav.DocumentRecordData) [32]byte {
	// These fields constrain the existing generated absence mode. They are
	// anchors only; any edit retains the new ordinary imported mode and values.
	v := struct {
		Class  string
		Fields []sav.DocumentValueData
	}{Class: r.Class}
	for _, f := range r.Values {
		if f.Name == "Identity" || f.Name == "RuntimeID" || f.Name == "Stage" {
			v.Fields = append(v.Fields, f)
		}
	}
	raw, _ := json.Marshal(v)
	return sha256.Sum256(raw)
}

// All typed archive holders join through existing exact object bindings.
// Coordinates are not scalar address keys, native IDs or owner slots.
func currentArchiveObjects(state *SnapshotSAVDocument, w *sim.World) (map[uint16]currentArchiveCoordinate, error) {
	out := map[uint16]currentArchiveCoordinate{}
	doc := state.Document
	add := func(object, archive uint16, generated bool) error {
		if object == 0 || int(object) > len(doc.Objects) {
			return fmt.Errorf("current archive coordinate lacks an exact object")
		}
		if old, ok := out[object]; ok {
			if old.Native != archive {
				return fmt.Errorf("current archive object %d has conflicting coordinates %d and %d", object, old.Native, archive)
			}
			generated = generated || old.Generated
		}
		out[object] = currentArchiveCoordinate{Object: object, Native: archive, Generated: generated}
		return nil
	}
	actors, unresolved := map[sim.EntityID]uint16{}, map[uint16]uint16{}
	for _, b := range state.Actors {
		if !b.Retired {
			actors[b.EntityID] = b.ObjectIndex
		}
	}
	for _, e := range w.Entities() {
		if e.SourceBinding.Class != 0 {
			if err := add(actors[e.ID], e.SourceBinding.ArchiveIndex, e.SourceBinding.Generated()); err != nil {
				return nil, err
			}
		}
	}
	for _, d := range w.OriginalDeadActors() {
		class := sourceActorDocumentClass(d.Source.Class)
		if class == "" {
			return nil, fmt.Errorf("current archive dead actor %d has no SAV class binding", d.ID)
		}
		object, err := currentTypedIdentityIndex(doc, class, d.Source.Identity)
		if err != nil {
			return nil, err
		}
		if err := add(object, d.Source.ArchiveIndex, (sim.SourceBinding{Class: d.Source.Class}).Generated()); err != nil {
			return nil, err
		}
		if held := d.Source.HeldWeapon; held.Present {
			object, err := currentTypedIdentityIndex(doc, "Weapon", held.Identity)
			if err != nil {
				return nil, err
			}
			if err := add(object, held.ArchiveIndex, false); err != nil {
				return nil, err
			}
		}
	}
	structures, _, _ := w.SavedStructures()
	for _, s := range structures {
		object, err := currentTypedIdentityIndex(doc, savedStructureClass(s.Class), s.SourceKey)
		if err != nil {
			return nil, err
		}
		if err := add(object, s.ArchiveIndex, s.Class.Generated()); err != nil {
			return nil, err
		}
	}
	if state.GroupBindings == nil {
		return out, nil
	}
	for _, m := range state.GroupBindings.Members {
		if m.Bound {
			actors[m.EntityID] = m.ObjectIndex
		} else {
			unresolved[m.UnresolvedHandle] = m.ObjectIndex
		}
	}
	bindings := map[uint32]SnapshotSAVGroupBinding{}
	for _, b := range state.GroupBindings.Groups {
		bindings[b.ID] = b
	}
	groups, _, _ := w.SavedGroups()
	for _, g := range groups {
		b, ok := bindings[g.ID]
		if !ok {
			return nil, fmt.Errorf("current archive Group lacks exact binding")
		}
		for _, ref := range []struct {
			object  uint16
			archive uint16
		}{{b.Reference.ObjectIndex, g.Reference.Archive}, {b.Owner.ObjectIndex, g.Owner.Archive}} {
			if ref.archive != 0 {
				if err := add(ref.object, ref.archive, false); err != nil {
					return nil, err
				}
			}
		}
		for _, m := range g.Members {
			object := unresolved[m.Archive]
			if m.Bound {
				object = actors[m.Entity]
			}
			if object != 0 || m.Bound || m.Archive != 0 {
				if err := add(object, m.Archive, false); err != nil {
					return nil, err
				}
			}
		}
	}
	return out, nil
}

func captureCurrentArchiveCoordinates(doc *sav.DocumentData, state *SnapshotSAVDocument, w *sim.World, a *currentActionData) error {
	objects, err := currentArchiveObjects(state, w)
	if err != nil {
		return err
	}
	for _, row := range objects {
		if row.Generated {
			anchor := currentArchiveModeAnchor(&doc.Objects[row.Object-1])
			row.ModeAnchor = &anchor
		}
		a.ArchiveCoordinates = append(a.ArchiveCoordinates, row)
	}
	slices.SortFunc(a.ArchiveCoordinates, func(x, y currentArchiveCoordinate) int { return int(x.Object) - int(y.Object) })
	sites := currentActorGroupSites(doc)
	entities := map[sim.EntityID]sim.Entity{}
	for _, e := range w.Entities() {
		entities[e.ID] = e
	}
	for _, b := range state.Actors {
		if e, ok := entities[b.EntityID]; ok && !b.Retired && e.SourceBinding.Class != 0 {
			site := sites[b.ObjectIndex]
			a.ActorGroups = append(a.ActorGroups, currentActorGroupCoordinate{b.ObjectIndex, site.Player, site.Inline, e.SourceBinding.GroupIndex, currentGroupSiteAnchor(doc, site)})
		}
	}
	return validateCurrentArchiveCoordinates(doc, a)
}

func validateCurrentArchiveCoordinates(doc *sav.DocumentData, a *currentActionData) error {
	if len(a.ArchiveCoordinates) > 65535 || len(a.ActorGroups) > 32767 {
		return fmt.Errorf("current archive coordinate population exceeds bounds")
	}
	objects, coordinates := map[uint16]bool{}, map[uint16]bool{}
	for _, row := range a.ArchiveCoordinates {
		if row.Object == 0 || int(row.Object) > len(doc.Objects) || objects[row.Object] || row.Native != 0 && coordinates[row.Native] || row.Generated != (row.ModeAnchor != nil) || row.Generated && (row.Native != 0 || *row.ModeAnchor == ([32]byte{})) {
			return fmt.Errorf("current archive coordinate is malformed or aliased")
		}
		objects[row.Object], coordinates[row.Native] = true, true
	}
	objects = map[uint16]bool{}
	for _, row := range a.ActorGroups {
		if row.Object == 0 || int(row.Object) > len(doc.Objects) || objects[row.Object] || int(row.Player) > len(doc.Objects) || row.Player == 0 && row.Inline != 0 || row.Player != 0 && doc.Objects[row.Player-1].Class != "Player" || row.Anchor == ([32]byte{}) {
			return fmt.Errorf("current actor Group coordinate is malformed")
		}
		objects[row.Object] = true
	}
	return nil
}

// Resolve coordinates before native carrier restoration removes constructor
// rows. New ordinary objects retain their own coordinates unless that number
// is already reserved by an explicit native identity.
func prepareCurrentArchiveCoordinates(ms *Mission, a *currentActionData) ([]sim.CurrentArchiveCoordinate, error) {
	if len(a.ArchiveCoordinates) == 0 {
		return nil, nil
	}
	actual, err := currentArchiveObjects(ms.savedDocument, ms.World)
	if err != nil {
		return nil, err
	}
	targets, used := map[uint16]uint16{}, map[uint16]bool{}
	for _, row := range a.ArchiveCoordinates {
		if row.Generated && currentArchiveModeAnchor(&ms.savedDocument.Document.Objects[row.Object-1]) != *row.ModeAnchor {
			continue
		}
		if current, ok := actual[row.Object]; ok && current.Native != 0 {
			targets[current.Native], used[row.Native] = row.Native, true
		}
	}
	var objects []int
	for object := range actual {
		objects = append(objects, int(object))
	}
	slices.Sort(objects)
	var out []sim.CurrentArchiveCoordinate
	next := uint32(1)
	for _, object := range objects {
		current := actual[uint16(object)].Native
		if current == 0 {
			continue
		}
		value, ok := targets[current]
		if !ok {
			value = current
			if used[value] {
				for next <= 65535 && used[uint16(next)] {
					next++
				}
				if next > 65535 {
					return nil, fmt.Errorf("current archive coordinate namespace exhausted")
				}
				value = uint16(next)
			}
			used[value] = true
		}
		out = append(out, sim.CurrentArchiveCoordinate{From: current, To: value})
	}
	return out, nil
}

func restoreCurrentArchiveCoordinates(ms *Mission, a *currentActionData, rows []sim.CurrentArchiveCoordinate) error {
	sites := currentActorGroupSites(ms.savedDocument.Document)
	actors := map[uint16]sim.EntityID{}
	for _, b := range ms.savedDocument.Actors {
		if !b.Retired {
			actors[b.ObjectIndex] = b.EntityID
		}
	}
	groups := map[sim.EntityID]sim.CurrentSourceGroup{}
	current, _, _ := ms.World.SavedGroups()
	byID := map[uint32]sim.SavedGroup{}
	for _, g := range current {
		byID[g.ID] = g
	}
	for _, row := range a.ActorGroups {
		id, ok := actors[row.Object]
		if !ok {
			continue
		}
		site := sites[row.Object]
		index := uint32(0)
		matched := site == (currentGroupSite{row.Player, row.Inline}) && row.Anchor == currentGroupSiteAnchor(ms.savedDocument.Document, site)
		if matched {
			index = row.Native
		} else if bindings := ms.savedDocument.GroupBindings; bindings != nil {
			for _, b := range bindings.Groups {
				if b.PlayerObject == site.Player && b.InlineIndex == site.Inline {
					index = b.ID
				}
			}
		}
		value := sim.CurrentSourceGroup{Index: index, Absent: index == 0, AdoptSelector: !matched}
		if g, ok := byID[index]; ok {
			value.Record = &g
		}
		groups[id] = value
	}
	if err := ms.World.RestoreCurrentArchiveCoordinates(rows, groups); err != nil {
		return err
	}
	if bindings := ms.savedDocument.GroupBindings; bindings != nil {
		mapping := map[uint16]uint16{}
		for _, row := range rows {
			mapping[row.From] = row.To
		}
		for i := range bindings.Members {
			m := &bindings.Members[i]
			if !m.Bound {
				if value, ok := mapping[m.UnresolvedHandle]; ok {
					m.UnresolvedHandle = value
				}
			}
		}
		slices.SortFunc(bindings.Members, savedGroupMemberCompare)
	}
	return nil
}
