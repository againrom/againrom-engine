package game

import (
	"fmt"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// Names remain ordinary UnitState fields. Only native manifest presence,
// membership, order and the construction marker lack an ordinary owner.
type currentActorManifest struct {
	Present bool
	Actors  []currentManifestActor
}

type currentManifestActor struct {
	Entity      sim.EntityID
	Constructed bool
}

func captureCurrentActorManifest(doc *sav.DocumentData, state *SnapshotSAVDocument, world *sim.World, source *SnapshotActorManifest) (*currentActorManifest, error) {
	manifest, err := snapshotActorManifest(source, world)
	if err != nil {
		return nil, err
	}
	policy := &currentActorManifest{Present: manifest != nil}
	if manifest == nil {
		return policy, nil
	}
	objects := map[sim.EntityID]uint16{}
	for _, binding := range state.Actors {
		if !binding.Retired {
			objects[binding.EntityID] = binding.ObjectIndex
		}
	}
	for _, actor := range manifest.Actors {
		object := objects[actor.ID]
		if object == 0 || int(object) > len(doc.Objects) {
			return nil, fmt.Errorf("current actor manifest lacks an ordinary binding")
		}
		record := &doc.Objects[object-1]
		if record.Class != "Unit" && record.Class != "Human" && record.Class != "Humanoid" {
			return nil, fmt.Errorf("current actor manifest has a non-actor binding")
		}
		found := 0
		for i := range record.Texts {
			if record.Texts[i].Name == "Name" {
				record.Texts[i].Value = actor.Name
				found++
			}
		}
		if found != 1 {
			return nil, fmt.Errorf("current actor manifest has no unique ordinary name")
		}
		policy.Actors = append(policy.Actors, currentManifestActor{actor.ID, actor.Constructed})
	}
	return policy, nil
}

func validateCurrentActorManifest(a *currentActionData) error {
	p := a.Manifest
	if p == nil {
		return nil
	}
	if len(p.Actors) > 32767 || !p.Present && p.Actors != nil {
		return fmt.Errorf("current actor manifest presence or population is invalid")
	}
	bound := map[sim.EntityID]bool{}
	for _, binding := range a.Bindings {
		if !binding.Structure && !binding.Missing {
			bound[binding.ID] = true
		}
	}
	seen := map[sim.EntityID]bool{}
	for _, actor := range p.Actors {
		if seen[actor.Entity] || !bound[actor.Entity] || !a.Values[actor.Entity].SourceBound {
			return fmt.Errorf("current actor manifest has a repeated or absent source actor")
		}
		seen[actor.Entity] = true
	}
	for id, value := range a.Values {
		if value.SourceBound && !seen[id] {
			return fmt.Errorf("current actor manifest omits a source actor")
		}
	}
	return nil
}

func restoreCurrentActorManifest(ms *Mission, a *currentActionData, table *mapload.Table) error {
	if a.Manifest == nil {
		return nil
	}
	var manifest *SnapshotActorManifest
	if a.Manifest.Present {
		manifest = &SnapshotActorManifest{Version: actorManifestVersion}
		objects := map[sim.EntityID]uint16{}
		for _, binding := range ms.savedDocument.Actors {
			if !binding.Retired {
				objects[binding.EntityID] = binding.ObjectIndex
			}
		}
		for _, actor := range a.Manifest.Actors {
			object := objects[actor.Entity]
			if object == 0 || int(object) > len(ms.savedDocument.Document.Objects) {
				return fmt.Errorf("current actor manifest lost its ordinary actor")
			}
			record := &ms.savedDocument.Document.Objects[object-1]
			name, found := "", 0
			for _, text := range record.Texts {
				if text.Name == "Name" {
					name, found = text.Value, found+1
				}
			}
			if found != 1 {
				return fmt.Errorf("current actor manifest lost its ordinary name")
			}
			if member, ok := originalPartyMember(ms, actor.Entity); ok && member.Hired() {
				member.Name = name
				name = ordinaryActorName(member, table)
			}
			manifest.Actors = append(manifest.Actors, SnapshotActor{ID: actor.Entity, Name: name, Constructed: actor.Constructed})
		}
	}
	if err := validateActorManifest(manifest, ms.World); err != nil {
		return err
	}
	ms.ActorManifest = manifest
	return nil
}

func projectCurrentHiredNames(doc *sav.DocumentData, a *currentActionData, table *mapload.Table) {
	objects := map[sim.EntityID]uint16{}
	for _, b := range a.Bindings {
		if !b.Structure && !b.Missing {
			objects[b.ID] = b.Object
		}
	}
	for _, rows := range [][]currentPartyMember{a.Party, a.Roster} {
		for _, p := range rows {
			member := p.restore()
			index := objects[p.Entity]
			if !member.Hired() || index == 0 || int(index) > len(doc.Objects) {
				continue
			}
			for i := range doc.Objects[index-1].Texts {
				text := &doc.Objects[index-1].Texts[i]
				if text.Name == "Name" {
					member.Name = text.Value
					text.Value = ordinaryActorName(member, table)
				}
			}
		}
	}
}
