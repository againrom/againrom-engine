package game

import (
	"fmt"
	"slices"
	"strings"

	"againrom/pkg/sim"
)

const actorManifestVersion uint32 = 1

// SnapshotActor contains only missing construction/presentation bindings.
// Class, definition selection, facing and every live gameplay value remain
// canonical World state, never a second copy in this manifest.
type SnapshotActor struct {
	ID          sim.EntityID
	Name        string
	Constructed bool
}

type SnapshotActorManifest struct {
	Version         uint32
	Actors          []SnapshotActor
	NativeNamesOnly bool
}

func cloneActorManifest(in *SnapshotActorManifest) *SnapshotActorManifest {
	if in == nil {
		return nil
	}
	out := *in
	out.Actors = append([]SnapshotActor(nil), in.Actors...)
	return &out
}

func validateActorManifest(in *SnapshotActorManifest, world *sim.World) error {
	if in == nil {
		if world != nil {
			for _, e := range world.Entities() {
				if e.SourceBinding.Class != 0 {
					return fmt.Errorf("saved source actor %d has no reconstruction manifest", e.ID)
				}
			}
		}
		return nil
	}
	if in.Version != actorManifestVersion || world == nil {
		return fmt.Errorf("saved actor manifest has an unsupported version or no world")
	}
	entities := world.Entities()
	if len(in.Actors) > len(entities) {
		return fmt.Errorf("saved actor manifest exceeds the world population")
	}
	seen := make(map[sim.EntityID]bool, len(in.Actors))
	for _, actor := range in.Actors {
		if seen[actor.ID] {
			return fmt.Errorf("saved actor manifest repeats entity %d", actor.ID)
		}
		seen[actor.ID] = true
		if strings.ContainsRune(actor.Name, '\x00') || len(actor.Name) > 4096 {
			return fmt.Errorf("saved actor manifest has unsupported metadata for entity %d", actor.ID)
		}
		i, found := slices.BinarySearchFunc(entities, actor.ID, func(e sim.Entity, id sim.EntityID) int {
			if e.ID < id {
				return -1
			}
			if e.ID > id {
				return 1
			}
			return 0
		})
		if !found || !manifestActorAvailable(entities[i]) || in.NativeNamesOnly && entities[i].SourceBinding.Class != 0 {
			return fmt.Errorf("saved actor manifest entity %d has no matching actor class", actor.ID)
		}
	}
	for _, e := range entities {
		if in.NativeNamesOnly && e.SourceBinding.Class != 0 {
			return fmt.Errorf("native actor names cannot supply a source manifest")
		}
		if e.SourceBinding.Class != 0 && !seen[e.ID] {
			return fmt.Errorf("saved actor manifest omits entity %d", e.ID)
		}
	}
	return nil
}

func manifestActorAvailable(e sim.Entity) bool {
	return e.SourceBinding.Class != 0 || e.ActorLoad.Source.Class == 0 && e.NativeBasis.HasValues()
}

func restoreActorManifest(ms *Mission, in *SnapshotActorManifest) error {
	if ms == nil {
		return fmt.Errorf("saved actor manifest requires a mission")
	}
	if err := validateActorManifest(in, ms.World); err != nil {
		return err
	}
	manifest := cloneActorManifest(in)
	if manifest != nil {
		slices.SortFunc(manifest.Actors, func(a, b SnapshotActor) int {
			if a.ID < b.ID {
				return -1
			}
			if a.ID > b.ID {
				return 1
			}
			return 0
		})
	}
	ms.ActorManifest = manifest
	return nil
}

func snapshotActorManifest(in *SnapshotActorManifest, world *sim.World) (*SnapshotActorManifest, error) {
	if in == nil {
		return nil, validateActorManifest(nil, world)
	}
	if world == nil {
		return nil, fmt.Errorf("saved actor manifest requires a world")
	}
	live := make(map[sim.EntityID]bool)
	for _, e := range world.Entities() {
		live[e.ID] = manifestActorAvailable(e)
	}
	out := cloneActorManifest(in)
	out.Actors = slices.DeleteFunc(out.Actors, func(actor SnapshotActor) bool { return !live[actor.ID] })
	if err := validateActorManifest(out, world); err != nil {
		return nil, err
	}
	return out, nil
}

func (mw *mapWorld) installActorManifest(in *SnapshotActorManifest) {
	if in == nil {
		return
	}
	mw.actorNames = make(map[sim.EntityID]string, len(in.Actors))
	if mw.tiers == nil {
		mw.tiers = make(map[sim.EntityID]int)
	}
	for _, actor := range in.Actors {
		mw.actorNames[actor.ID] = actor.Name
		if e, ok := mw.entity(actor.ID); ok && e.SourceBinding.ActorClass() == 1 {
			mw.tiers[actor.ID] = int(e.SourceBinding.Face)
		}
	}
	// newMapWorldWith made the initial projection before these metadata existed.
	// Refresh before the candidate viewer can become the live mission.
	if mw.view != nil {
		mw.view.SetEntities(mw.entityDraws())
	}
}
