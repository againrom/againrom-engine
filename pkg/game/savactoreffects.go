package game

import (
	"fmt"
	"slices"
	"strings"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// Additive native metadata: nil means a predecessor did not restore actor
// attachments. Numeric Token keys are never identity. An expired exclusive
// child keeps its binding with ObjectIndex zero after explicit retirement.
type SnapshotSAVActorEffects struct {
	Version     uint32
	Rows        []SnapshotSAVActorEffect
	Unavailable string
}

type SnapshotSAVActorEffect struct {
	Entity      sim.EntityID
	Spell       uint16
	ObjectIndex uint16
}

func originalActorEffect(value sim.SavedEffectObject, target sim.EntityID) (sim.ActiveEffect, error) {
	e := sim.ActiveEffect{Target: target, Spell: uint16(value.E0C), Magnitude: int32(int16(value.Value.Operand)), Remaining: uint16(value.Value.Operand >> 16)}
	e.Mode = sim.EffectMode(value.Value.Mode)
	kinds := map[uint8]sim.EffectKind{6: sim.EffectHealth, 8: sim.EffectHealthRegeneration, 11: sim.EffectManaRegeneration, 16: sim.EffectAbsorption, 17: sim.EffectSpeed, 19: sim.EffectScanRange, 21: sim.EffectProtectionFire, 22: sim.EffectProtectionWater, 23: sim.EffectProtectionAir, 24: sim.EffectProtectionEarth, 38: sim.EffectInvisible, 39: sim.EffectBless, 40: sim.EffectCurse}
	e.Kind = kinds[value.Value.Kind]
	if value.Value.Mode == 0 || value.Value.Mode > 7 || e.Spell > 28 || e.Kind == sim.EffectNone || e.Spell == 0 && (value.Value.Mode > 2 || e.Kind != sim.EffectHealthRegeneration && e.Kind != sim.EffectManaRegeneration && e.Kind != sim.EffectAbsorption) {
		return e, fmt.Errorf("unsupported actor Effect id%d/kind%d/mode%d", value.E0C, value.Value.Kind, value.Value.Mode)
	}
	return e, nil
}

func importOriginalActorEffects(ms *Mission) error {
	state := ms.savedDocument
	if state == nil || state.Document == nil {
		return nil
	}
	metadata := &SnapshotSAVActorEffects{Version: 1}
	incoming := savedDocumentIncoming(state.Document)
	var effects []sim.ActiveEffect
	for _, actor := range state.Actors {
		if actor.Retired {
			continue
		}
		refs, found := savedObjectRefs(&state.Document.Objects[actor.ObjectIndex-1], "Effects")
		if !found {
			return fmt.Errorf("actor %d lacks Effects references", actor.EntityID)
		}
		seen := map[uint16]bool{}
		start := len(effects)
		unsupported := false
		for _, index := range refs {
			if index == 0 || int(index) > len(state.Document.Objects) {
				return fmt.Errorf("actor %d has invalid Effect reference", actor.EntityID)
			}
			value, err := savedEffectRecord(&state.Document.Objects[index-1])
			if err != nil {
				metadata.Unavailable = fmt.Sprintf("actor %d Effect%d: %v", actor.EntityID, index, err)
				unsupported = true
				continue
			}
			effect, err := originalActorEffect(value, actor.EntityID)
			if err != nil || incoming[index] != 1 || seen[effect.Spell] {
				metadata.Unavailable = fmt.Sprintf("actor %d Effect%d requires shared/unsupported attachment lifecycle: %v", actor.EntityID, index, err)
				unsupported = true
				continue
			}
			seen[effect.Spell] = true
			effects = append(effects, effect)
			metadata.Rows = append(metadata.Rows, SnapshotSAVActorEffect{Entity: actor.EntityID, Spell: effect.Spell, ObjectIndex: index})
		}
		// An unsupported sibling can affect the same actor's expiry law. Keep
		// its complete source graph explicit, without partially arming that actor.
		if unsupported {
			effects = effects[:start]
			metadata.Rows = metadata.Rows[:start]
		}
	}
	if err := ms.World.ImportOriginalAttachedEffects(effects); err != nil {
		return err
	}
	state.ActorEffects = metadata
	return nil
}

func cloneSavedActorEffects(src *SnapshotSAVActorEffects, doc *sav.DocumentData, actors []SnapshotSAVActor) (*SnapshotSAVActorEffects, error) {
	if src == nil {
		return nil, nil
	}
	if src.Version != 1 || doc == nil || len(src.Rows) > len(actors)*29 || len(src.Unavailable) > 600 || strings.ContainsRune(src.Unavailable, 0) {
		return nil, fmt.Errorf("invalid saved actor-effect metadata")
	}
	byActor := map[sim.EntityID]uint16{}
	for _, a := range actors {
		byActor[a.EntityID] = a.ObjectIndex
	}
	seen := map[[2]uint32]bool{}
	objects := map[uint16]bool{}
	incoming := savedDocumentIncoming(doc)
	for _, row := range src.Rows {
		index, found := byActor[row.Entity]
		key := [2]uint32{uint32(row.Entity), uint32(row.Spell)}
		if !found || row.Spell > 28 || seen[key] || int(row.ObjectIndex) > len(doc.Objects) {
			return nil, fmt.Errorf("invalid saved actor-effect identity")
		}
		seen[key] = true
		if row.ObjectIndex == 0 {
			continue
		}
		if objects[row.ObjectIndex] || doc.Objects[row.ObjectIndex-1].Class != "Effect" {
			return nil, fmt.Errorf("aliased saved actor Effect")
		}
		objects[row.ObjectIndex] = true
		refs, _ := savedObjectRefs(&doc.Objects[index-1], "Effects")
		if !slices.Contains(refs, row.ObjectIndex) {
			return nil, fmt.Errorf("saved actor Effect lost owner edge")
		}
		if incoming[row.ObjectIndex] != 1 {
			return nil, fmt.Errorf("saved actor Effect acquired a shared owner")
		}
		id, err := savedStructureValue(&doc.Objects[row.ObjectIndex-1], "E0C")
		if err != nil || id != uint32(row.Spell) {
			return nil, fmt.Errorf("saved actor Effect ID differs")
		}
	}
	if src.Unavailable == "" {
		for _, actor := range actors {
			if actor.Retired {
				continue
			}
			refs, _ := savedObjectRefs(&doc.Objects[actor.ObjectIndex-1], "Effects")
			for _, index := range refs {
				if !objects[index] {
					return nil, fmt.Errorf("saved actor Effect lost its binding")
				}
			}
		}
	}
	next := *src
	next.Rows = slices.Clone(src.Rows)
	return &next, nil
}

func remapSavedActorEffects(state *SnapshotSAVDocument, permutation []uint16) error {
	if state.ActorEffects == nil {
		return nil
	}
	for i := range state.ActorEffects.Rows {
		row := &state.ActorEffects.Rows[i]
		if int(row.ObjectIndex) >= len(permutation) || row.ObjectIndex != 0 && permutation[row.ObjectIndex] == 0 {
			return fmt.Errorf("actor effect binding unexpectedly retired")
		}
		row.ObjectIndex = permutation[row.ObjectIndex]
	}
	return nil
}

func validateSavedActorEffectsWorld(state *SnapshotSAVDocument, world *sim.World) error {
	if state == nil || state.ActorEffects == nil {
		return nil
	}
	active := map[[2]uint32]sim.ActiveEffect{}
	for _, e := range world.ActiveEffects() {
		active[[2]uint32{uint32(e.Target), uint32(e.Spell)}] = e
	}
	for _, row := range state.ActorEffects.Rows {
		e, found := active[[2]uint32{uint32(row.Entity), uint32(row.Spell)}]
		if state.ActorEffects.Unavailable == "" {
			for _, actor := range state.Actors {
				if actor.EntityID != row.Entity {
					continue
				}
				mask, err := savedStructureValue(&state.Document.Objects[actor.ObjectIndex-1], "U144")
				if err != nil || (mask&(uint32(1)<<row.Spell) != 0) != (row.ObjectIndex != 0) || found != (row.ObjectIndex != 0) {
					return fmt.Errorf("saved actor Effect mask/presence differs from current World")
				}
			}
		}
		if row.ObjectIndex == 0 {
			continue
		} // a later new cast is not this retired object
		value, err := savedEffectRecord(&state.Document.Objects[row.ObjectIndex-1])
		if err != nil {
			return err
		}
		want, err := originalActorEffect(value, row.Entity)
		if err != nil || !found || e.Kind != want.Kind || e.Mode != want.Mode || int32(int16(e.Magnitude)) != want.Magnitude || e.Remaining != want.Remaining {
			if state.ActorEffects.Unavailable != "" {
				continue
			}
			return fmt.Errorf("saved actor%d Effect%d differs from current World", row.Entity, row.ObjectIndex)
		}
	}
	return nil
}

func projectSavedActorEffects(state *SnapshotSAVDocument, world *sim.World) error {
	if state.ActorEffects == nil {
		// A predecessor may retain historic actor effects with no live timers.
		// Native LOAD never imports them. Name the gap only when an effect
		// exists on either side; effect-free predecessors remain byte-stable.
		present := len(world.ActiveEffects()) != 0
		for _, actor := range state.Actors {
			refs, _ := savedObjectRefs(&state.Document.Objects[actor.ObjectIndex-1], "Effects")
			present = present || len(refs) != 0
		}
		if present {
			state.ActorEffects = &SnapshotSAVActorEffects{Version: 1, Unavailable: "legacy native actor effects have no exact current bindings"}
		}
		return nil
	}
	next, err := cloneSavedDocument(state)
	if err != nil {
		return err
	}
	actors := map[sim.EntityID]uint16{}
	for _, a := range next.Actors {
		actors[a.EntityID] = a.ObjectIndex
	}
	active := map[[2]uint32]sim.ActiveEffect{}
	for _, e := range world.ActiveEffects() {
		active[[2]uint32{uint32(e.Target), uint32(e.Spell)}] = e
	}
	bound := map[[2]uint32]bool{}
	for _, row := range next.ActorEffects.Rows {
		if row.ObjectIndex != 0 {
			bound[[2]uint32{uint32(row.Entity), uint32(row.Spell)}] = true
		}
	}
	created := false
	for _, e := range world.ActiveEffects() {
		key := [2]uint32{uint32(e.Target), uint32(e.Spell)}
		index, actor := actors[e.Target]
		if !actor || bound[key] {
			continue
		}
		record, err := newEffectAttachment(e)
		if err != nil {
			next.ActorEffects.Unavailable = err.Error()
			continue
		}
		if len(next.Document.Objects) >= 65534 {
			return fmt.Errorf("new actor Effect exceeds archive object limit")
		}
		next.Document.Objects = append(next.Document.Objects, record)
		child := uint16(len(next.Document.Objects))
		owner := &next.Document.Objects[index-1]
		refs, _ := savedObjectRefs(owner, "Effects")
		savedObjectSetRefs(owner, "Effects", append(slices.Clone(refs), child), true)
		reused := false
		for i := range next.ActorEffects.Rows {
			row := &next.ActorEffects.Rows[i]
			if row.Entity == e.Target && row.Spell == e.Spell {
				row.ObjectIndex = child
				reused = true
			}
		}
		if !reused {
			next.ActorEffects.Rows = append(next.ActorEffects.Rows, SnapshotSAVActorEffect{Entity: e.Target, Spell: e.Spell, ObjectIndex: child})
		}
		created = true
	}
	var retired []uint16
	for i := range next.ActorEffects.Rows {
		row := &next.ActorEffects.Rows[i]
		if row.ObjectIndex == 0 {
			continue
		}
		e, found := active[[2]uint32{uint32(row.Entity), uint32(row.Spell)}]
		actor := &next.Document.Objects[actors[row.Entity]-1]
		mask, err := savedStructureValue(actor, "U144")
		if err != nil {
			return err
		}
		if !found {
			refs, _ := savedObjectRefs(actor, "Effects")
			refs = slices.DeleteFunc(slices.Clone(refs), func(index uint16) bool { return index == row.ObjectIndex })
			savedObjectSetRefs(actor, "Effects", refs, true)
			retired = append(retired, row.ObjectIndex)
			row.ObjectIndex = 0
			savedObjectSetValue(actor, "U144", mask & ^(uint32(1)<<row.Spell))
			continue
		}
		current, err := newEffectAttachment(e)
		if err != nil {
			state.ActorEffects.Unavailable = err.Error()
			return nil
		}
		for _, name := range []string{"E0C", "E3C", "E3D", "E40"} {
			value, _ := savedStructureValue(&current, name)
			savedObjectSetValue(&next.Document.Objects[row.ObjectIndex-1], name, value)
		}
		savedObjectSetValue(actor, "U144", mask|uint32(1)<<row.Spell)
	}
	if len(retired) > 0 || created {
		doc, permutation, err := sav.RetireDocumentData(*next.Document, retired)
		if err != nil {
			return err
		}
		next.Document = &doc
		if err := remapSavedSackDocument(next, permutation); err != nil {
			return err
		}
	}
	var oldTarget, oldSpell uint32
	const oldCasterMarker = "current world SAV unavailable: actor %d effect %d caster persistence is not established; preserve this state in .ags"
	_, parseErr := fmt.Sscanf(next.ActorEffects.Unavailable, oldCasterMarker, &oldTarget, &oldSpell)
	obsolete := fmt.Sprintf(oldCasterMarker, oldTarget, oldSpell)
	if parseErr == nil && next.ActorEffects.Unavailable == obsolete {
		next.ActorEffects.Unavailable = ""
	}
	if err := validateSavedActorEffectsWorld(next, world); err != nil {
		return err
	}
	*state = *next
	return nil
}
