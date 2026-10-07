package game

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"slices"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

type currentEffectWidth struct {
	Object, Actor uint16
	Spell         uint8
	Wire          uint16
	Lift          int64
	Anchor        [32]byte
}

func currentEffectWidthAnchor(value sim.SavedEffectObject) [32]byte {
	raw := []byte{value.E0C, value.Value.Kind, value.Value.Mode}
	raw = binary.LittleEndian.AppendUint32(raw, value.Value.Operand)
	return sha256.Sum256(raw)
}

func matchingCurrentEffectWidths(doc *sav.DocumentData, rows []currentEffectWidth) ([]currentEffectWidth, error) {
	if len(rows) == 0 {
		return nil, nil
	}
	if doc == nil || len(rows) > len(doc.Objects) || len(rows) > 65534 {
		return nil, fmt.Errorf("current effect width population exceeds ordinary objects")
	}
	incoming := savedDocumentIncoming(doc)
	seen := map[uint16]bool{}
	var matched []currentEffectWidth
	for _, row := range rows {
		magnitude := int64(int16(row.Wire)) + row.Lift
		if row.Object == 0 || int(row.Object) > len(doc.Objects) || row.Actor == 0 || int(row.Actor) > len(doc.Objects) || seen[row.Object] || row.Spell > 28 || row.Anchor == ([32]byte{}) || row.Lift == 0 || row.Lift%65536 != 0 || row.Lift < -1<<31 || row.Lift > 1<<31 || magnitude < -1<<31 || magnitude > 1<<31-1 {
			return nil, fmt.Errorf("invalid current effect width binding or lift")
		}
		seen[row.Object] = true
		actor := &doc.Objects[row.Actor-1]
		if actor.Class != "Unit" && actor.Class != "Human" && actor.Class != "Humanoid" {
			return nil, fmt.Errorf("current effect width owner is not an actor")
		}
		value, err := savedEffectRecord(&doc.Objects[row.Object-1])
		if err != nil {
			return nil, err
		}
		refs, ok := savedObjectRefs(actor, "Effects")
		if !ok {
			return nil, fmt.Errorf("current effect width owner has no Effects slots")
		}
		if value.E0C != row.Spell || uint16(value.Value.Operand) != row.Wire || currentEffectWidthAnchor(value) != row.Anchor || !slices.Contains(refs, row.Object) || incoming[row.Object] != 1 {
			continue
		}
		if _, err := originalActorEffect(value, 0); err != nil {
			return nil, err
		}
		matched = append(matched, row)
	}
	return matched, nil
}

func captureCurrentEffectWidths(doc *sav.DocumentData, state *SnapshotSAVDocument, world *sim.World, a *currentActionData) error {
	a.EffectWidths = nil
	actors := map[sim.EntityID]uint16{}
	for _, actor := range state.Actors {
		if !actor.Retired {
			actors[actor.EntityID] = actor.ObjectIndex
		}
	}
	effects := map[[2]uint32]uint16{}
	if state.ActorEffects != nil {
		for _, row := range state.ActorEffects.Rows {
			if row.ObjectIndex != 0 {
				effects[[2]uint32{uint32(row.Entity), uint32(row.Spell)}] = row.ObjectIndex
			}
		}
	}
	for _, effect := range world.ActiveEffects() {
		lift := int64(effect.Magnitude) - int64(int16(effect.Magnitude))
		if lift == 0 {
			continue
		}
		object, actor := effects[[2]uint32{uint32(effect.Target), uint32(effect.Spell)}], actors[effect.Target]
		if object == 0 || actor == 0 || int(object) > len(doc.Objects) {
			return fmt.Errorf("current effect width has no exact ordinary binding")
		}
		value, err := savedEffectRecord(&doc.Objects[object-1])
		if err != nil {
			return err
		}
		ordinary, err := originalActorEffect(value, effect.Target)
		if err != nil || ordinary.Spell != effect.Spell || ordinary.Kind != effect.Kind || ordinary.Mode != effect.Mode || ordinary.Remaining != effect.Remaining || ordinary.Magnitude != int32(int16(effect.Magnitude)) {
			return fmt.Errorf("current effect width ordinary fields differ")
		}
		a.EffectWidths = append(a.EffectWidths, currentEffectWidth{Object: object, Actor: actor, Spell: value.E0C, Wire: uint16(effect.Magnitude), Lift: lift, Anchor: currentEffectWidthAnchor(value)})
	}
	matched, err := matchingCurrentEffectWidths(doc, a.EffectWidths)
	if err != nil {
		return err
	}
	if len(matched) != len(a.EffectWidths) {
		return fmt.Errorf("current effect width lost its ordinary anchor")
	}
	return nil
}

func matchCurrentEffectWidths(ms *Mission, a *currentActionData) error {
	rows, err := matchingCurrentEffectWidths(ms.savedDocument.Document, a.EffectWidths)
	if err != nil {
		return err
	}
	actors := map[uint16]sim.EntityID{}
	for _, actor := range ms.savedDocument.Actors {
		if !actor.Retired {
			actors[actor.ObjectIndex] = actor.EntityID
		}
	}
	for _, row := range rows {
		id, bound := actors[row.Actor]
		value, present := a.Values[id]
		if !bound || !present {
			return fmt.Errorf("current effect width has no current actor value binding")
		}
		value.EffectWidths = append(value.EffectWidths, sim.EffectNumericResidue{Spell: uint16(row.Spell), Wire: row.Wire, Lift: row.Lift})
		a.Values[id] = value
	}
	return nil
}
