package game

import (
	"fmt"
	"slices"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func observeLegacyNativeBasis(b sim.NativeActorBasis, r *sav.DocumentRecordData) (sim.NativeActorBasis, error) {
	if r == nil {
		return b, nil
	}
	for _, block := range []struct {
		name    string
		present bool
		dst     []byte
	}{{"U114", b.BasePresent, b.Base[:]}, {"UD4", b.ModifierPresent, b.Modifier[:]}} {
		if block.present {
			continue
		}
		raw, err := savedActorRaw(r, block.name, len(block.dst))
		if err != nil {
			return b, err
		}
		copy(block.dst, raw)
		if block.name == "U114" {
			b.BasePresent, b.BaseKnown = true, 0x00ffffff
		} else {
			b.ModifierPresent, b.ModifierKnown = true, ^uint64(0)
		}
	}
	if !b.BodyPresent {
		body, err := savedStructureValue(r, "Body")
		if err != nil {
			return b, err
		}
		b = b.WithBody(uint16(body))
	}
	for _, block := range []struct {
		name    string
		present bool
		mask    uint32
		dst     []byte
	}{{"UA6", b.AttackPresent, 0xffffff, b.Attack[:]}, {"UBE", b.DefencePresent, 0x3fffff, b.Defence[:]}} {
		if block.present {
			continue
		}
		raw, err := savedActorRaw(r, block.name, len(block.dst))
		if err != nil {
			return b, err
		}
		for n := range block.dst {
			if block.mask&(uint32(1)<<n) != 0 {
				block.dst[n] = raw[n]
			}
		}
		if block.name == "UA6" {
			b.AttackPresent, b.AttackKnown = true, block.mask
		} else {
			b.DefencePresent, b.DefenceKnown = true, block.mask
		}
	}
	return observeNativeScalars(b, r)
}

func restoreLegacyNativeObservations(ms *Mission, a *currentActionData) error {
	if a.NativeHistoryVersion == 1 {
		return nil
	}
	if a.NativeHistoryVersion != 0 {
		return fmt.Errorf("unsupported native history observation version")
	}
	var rows []sim.NativeActorBasisRecord
	var names []currentManifestActor
	observeNames := a.Manifest == nil || a.Manifest.NativeActors == nil
	if ms.actorRegistry == nil || ms.savedDocument == nil || ms.savedDocument.Document == nil {
		return fmt.Errorf("legacy native observations lack ordinary provenance")
	}
	for _, e := range ms.World.Entities() {
		if e.ActorLoad.Source.Class != 0 {
			continue
		}
		var record *sav.DocumentRecordData
		matches := 0
		for _, b := range ms.actorRegistry.actors {
			if b.ID != e.ID {
				continue
			}
			for _, subject := range ms.savedDocument.Actors {
				if subject.Retired || subject.EntityID != e.ID {
					continue
				}
				if subject.ObjectIndex == 0 || int(subject.ObjectIndex) > len(ms.savedDocument.Document.Objects) {
					return fmt.Errorf("legacy native observation lacks exact ordinary subject")
				}
				r := &ms.savedDocument.Document.Objects[subject.ObjectIndex-1]
				key, err := savedStructureValue(r, "Identity")
				if err != nil || key != b.Source.Identity || r.Class != b.Source.Class {
					return fmt.Errorf("legacy native observation subject conflicts")
				}
				record = r
				matches++
			}
		}
		if matches > 1 {
			return fmt.Errorf("legacy native observation subject is ambiguous")
		}
		if record == nil {
			continue
		}
		b, err := observeLegacyNativeBasis(e.NativeBasis, record)
		if err != nil {
			return err
		}
		rows = append(rows, sim.NativeActorBasisRecord{ID: e.ID, Basis: b})
		if observeNames {
			found := 0
			for _, text := range record.Texts {
				if text.Name == "Name" {
					found++
				}
			}
			if found != 1 {
				return fmt.Errorf("legacy native observation lacks a unique ordinary name")
			}
			names = append(names, currentManifestActor{Entity: e.ID})
		}
	}
	bindings, err := currentTerminalActorBindings(ms.savedDocument, ms.World)
	if err != nil {
		return err
	}
	removed := ms.World.RemovedNativeActorBases()
	for _, terminal := range ms.World.CurrentTerminalActors() {
		object := bindings[terminal.ID]
		if object == 0 {
			continue
		}
		var b sim.NativeActorBasis
		for _, row := range removed {
			if row.ID == terminal.ID {
				b = row.Basis
			}
		}
		b, err = observeLegacyNativeBasis(b, &ms.savedDocument.Document.Objects[object-1])
		if err != nil {
			return err
		}
		rows = append(rows, sim.NativeActorBasisRecord{ID: terminal.ID, Basis: b})
	}
	slices.SortFunc(rows, func(a, b sim.NativeActorBasisRecord) int {
		if a.ID < b.ID {
			return -1
		}
		if a.ID > b.ID {
			return 1
		}
		return 0
	})
	if err := ms.World.RestoreNativeActorBases(rows); err != nil {
		return err
	}
	if len(names) != 0 {
		policy := currentActorManifest{}
		if a.Manifest != nil {
			policy = *a.Manifest
		} else if ms.ActorManifest != nil {
			manifest, err := snapshotActorManifest(ms.ActorManifest, ms.World)
			if err != nil {
				return err
			}
			for _, actor := range manifest.Actors {
				entity, live := ms.World.Entity(actor.ID)
				if live && entity.SourceBinding.Class != 0 {
					policy.Present = true
					policy.Actors = append(policy.Actors, currentManifestActor{actor.ID, actor.Constructed})
				}
			}
		}
		policy.NativeActors = names
		a.Manifest = &policy
	}
	return nil
}
