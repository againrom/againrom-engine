package game

import (
	"encoding/binary"
	"fmt"
	"math"
	"slices"
	"strings"

	"againrom/pkg/data"
	"againrom/pkg/formats/reg"
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

type SnapshotSAVWorldEffects struct {
	Version       uint32
	SpellNodes    []SnapshotSAVSpellNode
	Areas         []SnapshotSAVArea
	Projectiles   bool
	ProjectileIDs []uint16
	Unavailable   string
}

// The DTO indices are actual archive-object bindings. A Token identity key
// only resolves cell-layer keys after uniqueness has been proved separately.
type SnapshotSAVArea struct {
	ID                      uint32
	ObjectIndex, ChildIndex uint16
}

const detachedProjectileTarget = sim.EntityID(math.MaxUint32)

func areaLayerIndex(spell uint16) (uint8, bool) {
	for i, id := range [...]uint16{3, 7, 8, 19, 12, 17} {
		if spell == id {
			return uint8(i), true
		}
	}
	return 0, false
}

func importOriginalWorldEffects(ms *Mission, src entrySource) error {
	state := ms.savedDocument
	if state == nil || state.Document == nil {
		return nil
	}
	projectiles := ms.World.SavedProjectiles()
	if len(state.Document.World.Effects) == 0 && len(projectiles.Items) == 0 {
		return nil
	}
	meta := &SnapshotSAVWorldEffects{Version: 1}
	drivers := &sim.SavedWorldEffects{}
	incoming := savedDocumentIncoming(state.Document)
	identityCount := map[uint32]int{}
	for _, object := range state.Document.Objects {
		if identity, err := savedStructureValue(&object, "Identity"); err == nil && identity != 0 {
			identityCount[identity]++
		}
	}
	var graph *sim.SavedSpellGraph
	if needsSpellGraph(state.Document) {
		var err error
		graph, err = bindSavedSpellGraph(state, ms.World, meta, drivers)
		if err != nil {
			return err
		}
	}
	for root, index := range state.Document.World.Effects {
		if graph != nil {
			break
		}
		object := &state.Document.Objects[index-1]
		if object.Class != "AreaEffect" || incoming[index] != 1 {
			meta.unavailable("SpellTransport/PointEffect scheduling, PE44 and shared world-effect lifetimes remain unbound")
			continue
		}
		refs, ok := savedObjectRefs(object, "AE44")
		if !ok || len(refs) != 1 || refs[0] == 0 || incoming[refs[0]] != 1 {
			meta.unavailable("AreaEffect has absent/shared payload lifetime")
			continue
		}
		child := &state.Document.Objects[refs[0]-1]
		ref, err := savedStructureValue(object, "Reference")
		if err != nil {
			return err
		}
		childRef, err := savedStructureValue(child, "Reference")
		if err != nil {
			return err
		}
		if ref != 0 || childRef != 0 {
			meta.unavailable("AreaEffect caster/reference lifecycle remains unbound")
			continue
		}
		mode, err := savedStructureValue(object, "T08")
		if err != nil {
			return err
		}
		spell, err := savedStructureValue(object, "T0C")
		if err != nil {
			return err
		}
		identity, err := savedStructureValue(object, "Identity")
		if err != nil {
			return err
		}
		position, err := savedMotionRaw(object, "Block12", 12)
		if err != nil {
			return err
		}
		d := sim.SavedAreaDriver{ID: uint32(root + 1), Root: int32(root), Identity: identity, Key: binary.LittleEndian.Uint16(position), Mode: sim.AreaModeBlast, Spell: uint16(spell), Layer: 255}
		if mode&2 != 0 {
			d.Mode = sim.AreaModeRing
		} else if mode&1 != 0 {
			d.Mode = sim.AreaModeCloud
		}
		if d.Mode == sim.AreaModeRing && spell != 4 && spell != 9 && spell != 21 {
			meta.unavailable("AreaEffect has no established staged program")
			continue
		}
		if d.Mode == sim.AreaModeCloud {
			layer, known := areaLayerIndex(d.Spell)
			if !known || identity == 0 || identityCount[identity] != 1 {
				meta.unavailable("AreaEffect cloud layer/identity is not uniquely bound")
				continue
			}
			d.Layer = layer
			for _, cell := range ms.World.SavedCellRecords() {
				if cell.SpellEffects[layer] == identity {
					d.Cells = append(d.Cells, cell.Cell)
				}
			}
		}
		if d.Spell != 19 && !sim.SavedAreaPayloadSupported(ms.World.SavedSpellEffects()[root].AE44) {
			meta.unavailable("AreaEffect payload application is unbound: EDD48 field bridge or unsupported ordinary effect")
		}
		drivers.Areas = append(drivers.Areas, d)
		meta.Areas = append(meta.Areas, SnapshotSAVArea{ID: d.ID, ObjectIndex: index, ChildIndex: refs[0]})
	}
	if len(projectiles.Items) > 0 {
		meta.Projectiles = true
		var definitions *data.Projectiles
		registryTried := false
		runtimeIDs := map[uint32]sim.EntityID{}
		duplicates := map[uint32]bool{}
		for _, binding := range state.Actors {
			if binding.Retired {
				continue
			}
			id, err := savedStructureValue(&state.Document.Objects[binding.ObjectIndex-1], "RuntimeID")
			if err != nil {
				return err
			}
			if existing, exists := runtimeIDs[id]; exists && existing != binding.EntityID {
				duplicates[id] = true
			}
			runtimeIDs[id] = binding.EntityID
		}
		for _, dead := range ms.World.OriginalDeadActors() {
			id := dead.Source.State.RuntimeID
			if id == 0 {
				continue
			}
			if existing, exists := runtimeIDs[id]; exists && existing != dead.ID {
				duplicates[id] = true
			}
			runtimeIDs[id] = dead.ID
		}
		for _, p := range projectiles.Items {
			if !completeSavedProjectileRecord(state.Document, p.ID) {
				meta.unavailable("projectile constructor-default or ambiguous source leaves remain unbound")
				continue
			}
			if p.Action < 0 || p.Action > 255 || p.Dir < 0 || p.Dir > 255 || p.ActionDir < 0 || p.ActionDir > 255 || p.ActionTarget < 0 || p.ActionTarget > 65535 || p.Picture < 0 || p.Z != 0 || p.ActionZ != 0 || p.ActionPhase < 0 || int64(p.ActionPhase)+int64(p.ActionSegments) >= 2147483647 || p.ActionSegments < 0 || p.ActionSegments > 65535 {
				meta.unavailable("projectile altitude or out-of-domain action continuation is unbound")
				continue
			}
			if !registryTried {
				registryTried = true
				raw, err := src.ReadFile(ProjectileRegistry)
				if err == nil {
					registry, parseErr := reg.Parse(raw)
					if parseErr == nil {
						definitions, _ = data.LoadProjectiles(registry)
					}
				}
			}
			d := sim.SavedProjectileDriver{ID: p.ID}
			if definitions == nil {
				// Without the registry a record is armed with no phase count,
				// as the live cast producer builds it from the same install.
				meta.unavailable("projectile phase registry unavailable; continuation is armed without a phase count")
			} else if definition, found := definitions.ByID(p.Picture); found {
				if definition.Phases <= 0 || definition.Phases > 65535 {
					meta.unavailable("projectile registry phase domain is unsupported")
					continue
				}
				d.Phases = uint16(definition.Phases)
			}
			if p.ActionTarget != 0 {
				id, found := runtimeIDs[uint32(p.ActionTarget)]
				if duplicates[uint32(p.ActionTarget)] {
					meta.unavailable("projectile target runtime ID has no unique current actor")
					continue
				}
				if !found {
					id = detachedProjectileTarget
				}
				d.Target, d.HasTarget = id, true
				_, materialized := entityIn(ms.World.Entities(), id)
				d.TargetDetached = !materialized
			}
			drivers.Projectiles = append(drivers.Projectiles, d)
			meta.ProjectileIDs = append(meta.ProjectileIDs, d.ID)
			// DIV-1814: these sixteen pictures' consumers are not bound.
			if p.Picture == 13 || p.Picture == 51 || p.Picture == 36 || p.Picture == 18 || p.Picture == 24 || p.Picture == 28 || p.Picture == 40 || p.Picture == 44 || p.Picture == 48 || p.Picture == 52 || p.Picture == 54 || p.Picture == 56 || p.Picture == 62 || p.Picture == 64 || p.Picture == 20 || p.Picture == 30 {
				meta.unavailable(unboundProjectileConsumers)
			}
			meta.unavailable("native removed-target policy retains the last current aim; original target lifetime remains unproved")
			meta.unavailable("projectile direction-sector and target-altitude arithmetic remain unproved; retained direction fields are unchanged")
		}
		slices.Sort(meta.ProjectileIDs)
		slices.SortFunc(drivers.Projectiles, func(a, b sim.SavedProjectileDriver) int { return int(a.ID) - int(b.ID) })
	}
	if len(drivers.Areas)+len(drivers.Projectiles) > 0 || graph != nil {
		if err := ms.World.ImportOriginalWorldEffectDrivers(drivers); err != nil {
			return err
		}
	}
	if graph != nil {
		if err := ms.World.ImportSavedSpellGraph(graph); err != nil {
			return err
		}
	}
	state.WorldEffects = meta
	return nil
}

func cloneSavedWorldEffects(src *SnapshotSAVWorldEffects, doc *sav.DocumentData) (*SnapshotSAVWorldEffects, error) {
	if src == nil {
		return nil, nil
	}
	fail := func() (*SnapshotSAVWorldEffects, error) {
		return nil, fmt.Errorf("invalid saved world-effect metadata")
	}
	if src.Version != 1 || doc == nil || doc.World == nil || len(src.Areas) > 0x7fff || len(src.Unavailable) > 4096 || strings.ContainsRune(src.Unavailable, 0) {
		return fail()
	}
	out := *src
	out.Areas = slices.Clone(src.Areas)
	out.SpellNodes = slices.Clone(src.SpellNodes)
	if len(out.SpellNodes) > 65534 {
		return fail()
	}
	spellSeen := map[uint16]bool{}
	for _, row := range out.SpellNodes {
		if int(row.ObjectIndex) > len(doc.Objects) || int(row.PayloadIndex) > len(doc.Objects) || row.ObjectIndex != 0 && spellSeen[row.ObjectIndex] {
			return fail()
		}
		if row.ObjectIndex != 0 {
			spellSeen[row.ObjectIndex] = true
		}
	}
	out.ProjectileIDs = slices.Clone(src.ProjectileIDs)
	if len(out.ProjectileIDs) > 65535 || len(out.ProjectileIDs) > 0 && !out.Projectiles {
		return fail()
	}
	for i, id := range out.ProjectileIDs {
		if i > 0 && out.ProjectileIDs[i-1] >= id {
			return fail()
		}
	}
	incoming := savedDocumentIncoming(doc)
	seen := map[uint16]bool{}
	for i, row := range out.Areas {
		if row.ID == 0 || i > 0 && out.Areas[i-1].ID >= row.ID || int(row.ObjectIndex) > len(doc.Objects) || int(row.ChildIndex) > len(doc.Objects) {
			return fail()
		}
		if row.ObjectIndex == 0 {
			if row.ChildIndex != 0 {
				return fail()
			}
			continue
		}
		if seen[row.ObjectIndex] || len(out.SpellNodes) == 0 && (row.ChildIndex == 0 || seen[row.ChildIndex] || incoming[row.ObjectIndex] != 1 || incoming[row.ChildIndex] != 1) {
			return fail()
		}
		seen[row.ObjectIndex], seen[row.ChildIndex] = true, true
		object := &doc.Objects[row.ObjectIndex-1]
		refs, ok := savedObjectRefs(object, "AE44")
		if object.Class != "AreaEffect" || len(out.SpellNodes) == 0 && !slices.Contains(doc.World.Effects, row.ObjectIndex) || !ok || !slices.Equal(refs, []uint16{row.ChildIndex}) {
			return fail()
		}
	}
	return &out, nil
}

func remapSavedWorldEffects(state *SnapshotSAVDocument, permutation []uint16) error {
	if state.WorldEffects == nil {
		return nil
	}
	for i := range state.WorldEffects.SpellNodes {
		r := &state.WorldEffects.SpellNodes[i]
		for _, index := range []*uint16{&r.ObjectIndex, &r.PayloadIndex} {
			if int(*index) >= len(permutation) || *index != 0 && permutation[*index] == 0 {
				return fmt.Errorf("implicit spell graph retirement")
			}
			*index = permutation[*index]
		}
	}
	for i := range state.WorldEffects.Areas {
		row := &state.WorldEffects.Areas[i]
		for _, index := range []*uint16{&row.ObjectIndex, &row.ChildIndex} {
			if int(*index) >= len(permutation) || *index != 0 && permutation[*index] == 0 {
				return fmt.Errorf("implicit world-effect retirement")
			}
			*index = permutation[*index]
		}
	}
	return nil
}

func validateSavedWorldEffectsWorld(state *SnapshotSAVDocument, world *sim.World) error {
	if err := validateSavedSpellGraph(state, world); err != nil {
		return err
	}
	drivers := world.SavedWorldEffectDrivers()
	if drivers == nil {
		if state != nil && state.WorldEffects != nil && len(state.WorldEffects.Areas)+len(state.WorldEffects.ProjectileIDs) > 0 &&
			(len(world.SavedSpellEffects()) != 0 || len(world.SavedProjectiles().Items) != 0) {
			return fmt.Errorf("native world-effect drivers lost")
		}
		return nil
	}
	if state == nil && onlyNativeProjectiles(world, drivers) {
		return nil
	}
	if state == nil || state.Document == nil || state.WorldEffects == nil || len(state.WorldEffects.Areas) != len(drivers.Areas) || len(state.WorldEffects.ProjectileIDs) != len(drivers.Projectiles) {
		return fmt.Errorf("native world-effect bindings lost")
	}
	for i, d := range drivers.Areas {
		if world.SavedSpellGraph() != nil {
			break
		}
		row := state.WorldEffects.Areas[i]
		if row.ID != d.ID || (row.ObjectIndex == 0) != (d.Root < 0) {
			return fmt.Errorf("native area identity/retirement differs")
		}
		if d.Root < 0 {
			continue
		}
		if int(d.Root) >= len(state.Document.World.Effects) || state.Document.World.Effects[d.Root] != row.ObjectIndex {
			return fmt.Errorf("native area root order differs")
		}
		e := world.SavedSpellEffects()[d.Root]
		object := &state.Document.Objects[row.ObjectIndex-1]
		identity, _ := savedStructureValue(object, "Identity")
		spell, _ := savedStructureValue(object, "T0C")
		rawMode, _ := savedStructureValue(object, "T08")
		mode := uint8(sim.AreaModeBlast)
		if rawMode&2 != 0 {
			mode = sim.AreaModeRing
		} else if rawMode&1 != 0 {
			mode = sim.AreaModeCloud
		}
		pos, err := savedMotionRaw(object, "Block12", 12)
		if err != nil || binary.LittleEndian.Uint16(pos) != d.Key || identity != d.Identity || spell != uint32(d.Spell) || mode != d.Mode {
			return fmt.Errorf("native area Token binding differs")
		}
		se40, _ := savedStructureValue(object, "SE40")
		se41, _ := savedStructureValue(object, "SE41")
		if se40 != uint32(e.SE40) || se41 != uint32(e.SE41) {
			return fmt.Errorf("native area common state differs")
		}
		if e.AE44 == nil {
			return fmt.Errorf("native area payload lost")
		}
		child := &state.Document.Objects[row.ChildIndex-1]
		rootRef, err := savedStructureValue(object, "Reference")
		childRef, childErr := savedStructureValue(child, "Reference")
		if err != nil || childErr != nil || rootRef != 0 || childRef != 0 {
			return fmt.Errorf("native area unbound caster/reference introduced")
		}
		if child.Class != e.AE44.Class {
			return fmt.Errorf("native area payload class differs")
		}
		for _, field := range []struct {
			name string
			want uint32
		}{{"E0C", uint32(e.AE44.E0C)}, {"E3C", uint32(e.AE44.E3C)}, {"E3D", uint32(e.AE44.E3D)}, {"E40", e.AE44.E40}} {
			got, err := savedStructureValue(child, field.name)
			if err != nil || got != field.want {
				return fmt.Errorf("native area payload %s differs", field.name)
			}
		}
		if child.Class == "Effect_DirectDamage" {
			block, err := savedMotionRaw(child, "EDD48", 24)
			if err != nil || !slices.Equal(block, e.AE44.DirectDamage[:]) {
				return fmt.Errorf("native area direct payload differs")
			}
		}
		counter, err := savedStructureValue(object, "AE4C")
		if err != nil || counter != uint32(e.AE4C) {
			return fmt.Errorf("native area countdown differs")
		}
		block, err := savedMotionRaw(object, "AE48", 4)
		if err != nil || !slices.Equal(block, e.AE48[:]) {
			return fmt.Errorf("native area stage differs")
		}
	}
	for i, d := range drivers.Projectiles {
		if state.WorldEffects.ProjectileIDs[i] != d.ID {
			return fmt.Errorf("native projectile identity differs")
		}
		if d.Retired || !d.HasTarget {
			continue
		}
		var target *sim.SavedProjectile
		for _, p := range world.SavedProjectiles().Items {
			if p.ID == d.ID {
				copy := p
				target = &copy
				break
			}
		}
		if target == nil {
			return fmt.Errorf("native projectile missing")
		}
		found := false
		if d.TargetStructure {
			// A structure keeps its World id as the key until the writer
			// gives the leaf the structure's runtime identity.
			for _, s := range world.Structures() {
				found = found || sim.EntityID(s.ID) == d.Target && int32(s.ID) == target.ActionTarget
			}
			if !found {
				return fmt.Errorf("native projectile structure target differs")
			}
			continue
		}
		if e, live := entityIn(world.Entities(), d.Target); live && (e.SourceBinding.Class == 0 || e.SourceBinding.RuntimeID == 0) {
			// A native actor holds no source runtime identity; the writer
			// assigns its wire identity at SAVE and rewrites the leaf.
			found = !d.TargetDetached && sim.ProjectileTargetKey(e) == target.ActionTarget
		}
		for _, a := range state.Actors {
			if found {
				break
			}
			if a.EntityID == d.Target {
				if a.Retired {
					found = d.TargetDetached
					break
				} // bounded tombstone: retain the last aim point
				id, err := savedStructureValue(&state.Document.Objects[a.ObjectIndex-1], "RuntimeID")
				if err == nil && id == uint32(target.ActionTarget) {
					found = true
				}
				break
			}
		}
		if !found {
			for _, dead := range world.OriginalDeadActors() {
				if dead.ID != d.Target || dead.Source.State.RuntimeID != uint32(target.ActionTarget) {
					continue
				}
				for _, index := range state.Document.DeadActors {
					object := &state.Document.Objects[index-1]
					identity, _ := savedStructureValue(object, "Identity")
					runtime, _ := savedStructureValue(object, "RuntimeID")
					if identity == dead.Source.Identity && (runtime == dead.Current.RuntimeID || runtime == dead.Source.State.RuntimeID) {
						found = true
						break
					}
				}
			}
		}
		if !found && d.TargetDetached {
			_, live := entityIn(world.Entities(), d.Target)
			found = !live
		}
		if !found {
			return fmt.Errorf("native projectile runtime target differs")
		}
	}
	if state.WorldEffects.Projectiles {
		return validateSavedProjectileDocument(state.Document, world.SavedProjectiles(), state.WorldEffects.ProjectileIDs)
	}
	return nil
}

// unboundProjectileConsumers names behaviour the engine does not run for a
// projectile in flight of a listed picture. It describes missing consumers, not
// a missing field: the record itself is written from current state or the
// loaded Document, so it never refuses a save (DIV-1814).
const unboundProjectileConsumers = "projectile picture-specific notification, view-slot, quake and chain-link consumers are not bound"

func (m *SnapshotSAVWorldEffects) unavailable(reason string) {
	if !strings.Contains(m.Unavailable, reason) {
		if m.Unavailable != "" {
			m.Unavailable += "; "
		}
		m.Unavailable += reason
	}
}

// onlyNativeProjectiles reports a world whose only carried effects are records
// this engine released at native actors. No Document binds them, and a world
// written without a Document carries them in its own form.
func onlyNativeProjectiles(world *sim.World, drivers *sim.SavedWorldEffects) bool {
	if len(drivers.Areas) != 0 || world.SavedSpellGraph() != nil {
		return false
	}
	for _, d := range drivers.Projectiles {
		if d.Retired {
			continue
		}
		if !d.HasTarget || d.TargetStructure {
			// A burst record names no target, and a structure is no actor; a
			// loaded record has a Document.
			continue
		}
		e, live := entityIn(world.Entities(), d.Target)
		if d.TargetDetached || !live || e.SourceBinding.Class != 0 && e.SourceBinding.RuntimeID != 0 {
			return false
		}
	}
	return true
}
