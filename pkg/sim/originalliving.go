package sim

import "fmt"

type OriginalLivingActor struct {
	Entity Entity
	New    bool
}

// ImportOriginalLivingActors publishes a detached, complete actor batch. New
// IDs must be above every existing/reserved native ID. Existing entities retain
// their parallel route/stock slots; unrelated session state is not rebuilt.
func (w *World) ImportOriginalLivingActors(batch []OriginalLivingActor) error {
	return w.importLivingActors(batch, nil)
}

// Current actor presence is validated against the exact archive registry before
// admission. Native positions may be outside terrain bounds, and a zero
// footprint is retained; its consumers use one cell.
func (w *World) ImportCurrentLivingActors(batch []OriginalLivingActor, current []EntityID) error {
	ids := make(map[EntityID]bool, len(current))
	for _, id := range current {
		if ids[id] {
			return fmt.Errorf("sim: repeated current actor admission identity")
		}
		ids[id] = true
	}
	return w.importLivingActors(batch, ids)
}

func (w *World) importLivingActors(batch []OriginalLivingActor, current map[EntityID]bool) error {
	if w == nil || w.tick != 0 {
		return fmt.Errorf("sim: original actor admission requires an unadvanced candidate")
	}
	staged := *w
	staged.entities = append([]Entity(nil), w.entities...)
	staged.routes = append([][]cell(nil), w.routes...)
	staged.carried = append([][]ItemStack(nil), w.carried...)
	staged.equipment = append([][EquipSlots]ItemInstance(nil), w.equipment...)
	// Action cleanup can touch these owners before final wire validation
	// refuses a batch. Detach them as well as the entity/route/stock slots.
	staged.savedGroups = cloneSavedGroups(w.savedGroups)
	staged.savedMotion = cloneActorMotions(w.savedMotion)
	staged.bookCasts = append([]bookCast(nil), w.bookCasts...)
	staged.scrollCasts = append([]ScrollCast(nil), w.scrollCasts...)
	staged.casts = append([]scriptCast(nil), w.casts...)
	seen := map[EntityID]bool{}
	for _, item := range batch {
		e := item.Entity
		if e.ActorLoad.Source.Class != 0 {
			e.NativeClass = NativeClass{}
			e.NativeBasis = NativeActorBasis{}
			e.SpeedModifier = 0
		}
		e.liftEffectiveSkills(w.rules)
		if seen[e.ID] || e.SourceBinding.Class == 0 || (!e.Alive() && !originalDyingEntity(e)) || !e.Domain.defined() || !current[e.ID] && (!e.OffMap && (e.X < 0 || e.Y < 0 || e.X >= w.bounds.Width || e.Y >= w.bounds.Height) || e.TokenSize == 0) {
			return fmt.Errorf("sim: unsupported original actor %d", e.ID)
		}
		seen[e.ID] = true
		i := indexOfEntity(staged.entities, e.ID)
		if item.New {
			next, ok := staged.NextEntityID()
			if !ok || i >= 0 || e.ID < next {
				return fmt.Errorf("sim: original actor ID %d collides", e.ID)
			}
			e.ActorState, e.PostX, e.PostY = actorStateGuard, e.X, e.Y
			e.DesiredFacing = e.Facing
			if !e.OffMap {
				staged.appendActorTraversal(e.ID)
			}
			staged.entities = append(staged.entities, e)
			staged.routes = append(staged.routes, nil)
			staged.carried = append(staged.carried, nil)
			staged.equipment = append(staged.equipment, [EquipSlots]ItemInstance{})
		} else {
			if i < 0 {
				return fmt.Errorf("sim: original actor ID %d missing", e.ID)
			}
			if !staged.entities[i].OffMap && e.OffMap {
				staged.unlinkActorTraversal(e.ID)
			} else if staged.entities[i].OffMap && !e.OffMap {
				staged.appendActorTraversal(e.ID)
			}
			staged.entities[i] = e
			if !e.SourceBinding.Generated() {
				staged.carried[i], staged.equipment[i] = nil, [EquipSlots]ItemInstance{}
			}
			if !e.SourceBinding.Generated() && e.WeaponSpellSource == WeaponSpellItem {
				staged.entities[i].WeaponSpellSource, staged.entities[i].WeaponSpell, staged.entities[i].WeaponSpellLevel = WeaponSpellNone, 0, 0
			}
		}
	}
	for id := range current {
		if !seen[id] {
			return fmt.Errorf("sim: current actor admission identity is absent")
		}
	}
	for _, item := range batch {
		if originalDyingEntity(item.Entity) {
			staged.clearFelledActions(indexOfEntity(staged.entities, item.Entity.ID))
			staged.clearInvalidTargetReferences(item.Entity.ID)
		}
	}
	staged.groups = freezeGroups(staged.entities)
	encoded, err := staged.MarshalBinary()
	if err != nil {
		return err
	}
	var checked World
	checked.ghost = w.ghost
	checked.sourceDerive = w.sourceDerive
	checked.burst.phases = w.burst.phases
	checked.rules = w.rules
	checked.diary = w.diary
	// STORIES 1131..1135 primed this staging round trip's fresh checked
	// receiver by hand for savedCellRecords, savedSpellEffects,
	// savedProjectiles and savedDiaries: UnmarshalBinary's own composite
	// literal used to carry each one across a decode from the RECEIVER's prior
	// value, so a fresh receiver like checked always decoded to zero regardless
	// of what w (staged's own source, staged := *w above) already held.
	if err := checked.UnmarshalBinary(encoded); err != nil {
		return err
	}
	*w = checked
	return nil
}
