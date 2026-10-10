package sim

import (
	"encoding/binary"
	"fmt"
	"slices"
	"sort"
)

// SavedWorldEffects binds only proved consumers of the carried SAV records.
// Nil is the legacy native boundary: loading an old carrier never starts a
// timer from a historical Document. IDs are stable import ordinals, not Token
// addresses or native EntityIDs. A retired area has Root == -1.
type SavedWorldEffects struct {
	Areas       []SavedAreaDriver
	Projectiles []SavedProjectileDriver
}

type SavedAreaDriver struct {
	ID       uint32
	Root     int32
	Identity uint32
	Key      uint16
	Layer    uint8
	Mode     uint8
	Spell    uint16
	Cells    []uint16
}

type SavedProjectileDriver struct {
	ID             uint16
	Phases         uint16
	Target         EntityID
	HasTarget      bool
	Retired        bool
	TargetDetached bool // native removal policy; source actiontarget remains carried
	// TargetStructure marks Target as a StructureID: a shot at a structure
	// homes on it like a unit (SAV-1197).
	TargetStructure bool
	Owner           uint32 // draw-only: the caster's owner slot; no SAV field carries it
}

func cloneSavedWorldEffects(src *SavedWorldEffects) *SavedWorldEffects {
	if src == nil {
		return nil
	}
	out := *src
	out.Areas = slices.Clone(src.Areas)
	for i := range out.Areas {
		out.Areas[i].Cells = slices.Clone(out.Areas[i].Cells)
	}
	out.Projectiles = slices.Clone(src.Projectiles)
	return &out
}

func (w *World) SavedWorldEffectDrivers() *SavedWorldEffects {
	return cloneSavedWorldEffects(w.savedWorldEffects)
}

// ImportOriginalWorldEffectDrivers arms existing saved values, without using
// a fresh-cast constructor or applying their payload at LOAD.
func (w *World) ImportOriginalWorldEffectDrivers(src *SavedWorldEffects) error {
	next := *w
	next.savedWorldEffects = cloneSavedWorldEffects(src)
	if err := next.savedWorldEffectsFault(); err != nil {
		return err
	}
	w.savedWorldEffects = next.savedWorldEffects
	return nil
}

func (w *World) savedWorldEffectsFault() error {
	s := w.savedWorldEffects
	if s == nil {
		return nil
	}
	fail := func() error { return fmt.Errorf("sim: invalid retained world-effect continuation") }
	if len(s.Areas) > 0x7fff || len(s.Projectiles) > 0xffff {
		return fail()
	}
	roots := map[int32]bool{}
	identities := map[uint32]bool{}
	layers := map[[2]uint32][]uint16{}
	for _, cell := range w.savedCellRecords {
		for layer, key := range cell.SpellEffects {
			if key != 0 {
				k := [2]uint32{uint32(layer), key}
				layers[k] = append(layers[k], cell.Cell)
			}
		}
	}
	for i, a := range s.Areas {
		if a.ID == 0 || i > 0 && s.Areas[i-1].ID >= a.ID || a.Root < -1 || a.Root >= int32(len(w.savedSpellEffects)) || a.Mode < areaModeBlast || a.Mode > areaModeCloud || a.Spell == 0 || a.Spell > 28 || a.Mode == areaModeCloud && (a.Layer >= 6 || a.Identity == 0 || identities[a.Identity]) || len(a.Cells) > 65536 {
			return fail()
		}
		identities[a.Identity] = true
		if a.Mode != areaModeCloud && (a.Layer != 255 || len(a.Cells) != 0) {
			return fail()
		}
		if a.Root == -1 && len(a.Cells) == 0 {
			continue
		}
		if !slices.Equal(a.Cells, layers[[2]uint32{uint32(a.Layer), a.Identity}]) {
			return fail()
		}
		if a.Root >= 0 {
			if roots[a.Root] {
				return fail()
			}
			roots[a.Root] = true
		}
		e, bound := w.savedAreaValue(a)
		if bound && e.Class != "AreaEffect" || a.Mode == areaModeRing && ringStageCount(w.spellArm(a.Spell)) == 0 {
			return fail()
		}

		for j, key := range a.Cells {
			x, y := keyCell(key)
			if x >= w.bounds.Width || y >= w.bounds.Height || j > 0 && a.Cells[j-1] >= key {
				return fail()
			}
		}
		if err := w.savedAreaCellsFault(a); err != nil {
			return err
		}
	}
	for i, d := range s.Projectiles {
		if i > 0 && s.Projectiles[i-1].ID >= d.ID || !d.HasTarget && (d.Target != 0 || d.TargetDetached || d.TargetStructure) || d.TargetStructure && d.TargetDetached {
			return fail()
		}
		p := w.savedProjectile(d.ID)
		if d.Retired {
			if p != nil {
				return fail()
			}
			continue
		}
		if p == nil || p.Picture < 0 || p.ActionSegments < 0 || p.ActionSegments > 65535 || p.ActionPhase < 0 || int64(p.ActionPhase)+int64(p.ActionSegments) >= 2147483647 || p.ActionZ != 0 || p.Z != 0 || (p.ActionTarget != 0) != d.HasTarget {
			return fail()
		}
		if d.TargetStructure {
			if indexOfStructure(w.structures, StructureID(d.Target)) < 0 {
				return fail()
			}
			continue
		}
		if d.HasTarget {
			i := indexOfEntity(w.entities, d.Target)
			if d.TargetDetached {
				if i >= 0 {
					return fail()
				}
				continue
			}
			if i < 0 {
				return fail()
			}
			live := i >= 0 && (!w.entities[i].SourceBinding.hasRuntimeID() || w.entities[i].SourceBinding.RuntimeID == uint32(p.ActionTarget))
			dead := false
			for _, record := range w.originalDead {
				if record.ID == d.Target && record.Source.State.RuntimeID == uint32(p.ActionTarget) {
					dead = true
					break
				}
			}
			if !live && !dead {
				return fail()
			}
		}
	}
	return nil
}

func (w *World) savedProjectile(id uint16) *SavedProjectile {
	for i := range w.savedProjectiles.Items {
		if w.savedProjectiles.Items[i].ID == id {
			return &w.savedProjectiles.Items[i]
		}
	}
	return nil
}

// The default arm includes unused switch entries inside13..64. Picture-specific
// notification effects stay outside this local coordinate/clock consumer.
// ANIM-143, DIV-944.
func savedAttachedProjectile(picture int32) bool {
	switch picture {
	case 18, 24, 28, 40, 44, 48, 52, 54, 56, 62, 64, 20, 30:
		return true
	}
	return false
}

// A residual layer key does not supply the baseline or current occupants for
// TERR-STRUCT-078. Admit a cloud only with both current cell owners and planes.
// Check every dependency before retirement can publish even its first clear.
func (w *World) savedAreaCellsFault(d SavedAreaDriver) error {
	if d.Mode != areaModeCloud {
		return nil
	}
	if len(d.Cells) > 0 && w.savedCellPlanes == nil {
		return fmt.Errorf("retained cloud lacks saved plane authority")
	}
	owners := 0
	for _, r := range w.savedCellRecords {
		if r.SpellEffects[d.Layer] == d.Identity {
			owners++
		}
	}
	if owners != len(d.Cells) {
		return fmt.Errorf("retained world-effect residual cell population differs")
	}
	for _, key := range d.Cells {
		at := sort.Search(len(w.savedCellRecords), func(i int) bool { return w.savedCellRecords[i].Cell >= key })
		if at == len(w.savedCellRecords) || w.savedCellRecords[at].Cell != key || w.savedCellRecords[at].SpellEffects[d.Layer] != d.Identity {
			return fmt.Errorf("retained world-effect cell %04x lacks its residual layer owner", key)
		}
		r := w.savedCellRecords[at]
		c := w.motionCell(key)
		if c == nil {
			return fmt.Errorf("retained cloud cell %04x lacks its current Cell payload", key)
		}
		if c.Payload[2] != r.LayerCount {
			return fmt.Errorf("retained cloud cell %04x layer count owners differ", key)
		}
		for layer, identity := range r.SpellEffects {
			if binary.LittleEndian.Uint32(c.Payload[20+4*layer:]) != identity {
				return fmt.Errorf("retained cloud cell %04x layer payload owners differ", key)
			}
		}
		value, bound := w.savedAreaValue(d)
		if !bound || savedAreaContains(d, int32(value.AE48[1]), key) {
			if issue := w.savedCellRecomputeIssue(*c); issue != "" {
				return fmt.Errorf("retained cloud cell %04x: %s", key, issue)
			}
		}
	}
	return nil
}

func (w *World) savedAreaValue(d SavedAreaDriver) (SavedSpellEffect, bool) {
	if d.Root >= 0 && int(d.Root) < len(w.savedSpellEffects) {
		return w.savedSpellEffects[d.Root], true
	}
	if g := w.savedSpellGraph; g != nil && d.ID > 0 && int(d.ID) <= len(g.Nodes) {
		return g.Nodes[d.ID-1].Value, true
	}
	return SavedSpellEffect{}, false
}

func (w *World) retireSavedArea(index int) {
	s := w.savedWorldEffects
	d := &s.Areas[index]
	if err := w.savedAreaCellsFault(*d); err != nil {
		// An externally broken current owner stays intact and is refused by
		// MarshalBinary with the same diagnostic. Never save a partial cleanup.
		return
	}
	root := d.Root
	w.captureSavedGraphAreas()
	if w.savedSpellGraph != nil {
		defer w.refreshSavedSpellGraph()
		defer w.retireSavedSpellNode(d.ID)
	}
	value, _ := w.savedAreaValue(*d)
	radius := int32(value.AE48[1])
	if root >= 0 {
		w.savedSpellEffects = slices.Delete(w.savedSpellEffects, int(root), int(root)+1)
		for j := range s.Areas {
			if s.Areas[j].Root > root {
				s.Areas[j].Root--
			}
		}
	}
	if d.Mode == areaModeCloud {
		for _, key := range slices.Clone(d.Cells) {
			if savedAreaContains(*d, radius, key) {
				w.clearSavedAreaLayer(key, int(d.Layer))
				w.deleteEmptyAreaCell(key)
			}
		}
		w.refreshSavedPlaneBlocks()
	}
	d.Root, d.Cells = -1, nil
}

// SAV-1033: direct damage carries frozen base/spread/school at bytes19..21.
// Other combat operands have no constructor writer in the traced population.
func SavedAreaPayloadSupported(e *SavedEffect) bool {
	if e != nil && e.Class == "Effect_DirectDamage" {
		for _, v := range e.DirectDamage[:19] {
			if v != 0 {
				return false
			}
		}
		return e.DirectDamage[21] <= 5
	}
	_, _, ok := savedAreaPayload(e)
	return ok
}

func savedAreaContains(d SavedAreaDriver, radius int32, key uint16) bool {
	cx, cy := keyCell(d.Key)
	x, y := keyCell(key)
	return x >= cx-radius && x <= cx+radius && y >= cy-radius && y <= cy+radius
}

func savedAreaPayload(e *SavedEffect) (EffectKind, EffectMode, bool) {
	if e == nil || e.Class != "Effect" || e.E3D > 7 {
		return 0, 0, false
	}
	kinds := map[uint8]EffectKind{6: EffectHealth, 8: EffectHealthRegeneration, 11: EffectManaRegeneration, 16: EffectAbsorption, 17: EffectSpeed, 19: EffectScanRange, 21: EffectProtectionFire, 22: EffectProtectionWater, 23: EffectProtectionAir, 24: EffectProtectionEarth, 38: EffectInvisible, 39: EffectBless, 40: EffectCurse}
	kind := kinds[e.E3C]
	mode := EffectMode(e.E3D)
	if e.E3D == 2 {
		mode = EffectContinuous
	}
	return kind, mode, kind != EffectNone && e.E0C > 0 && e.E0C <= 28
}

func (w *World) applySavedAreaPayload(d SavedAreaDriver, e SavedSpellEffect, cells []uint16, caster ...EntityID) {
	if w.spellArm(d.Spell) == 19 {
		return
	}
	if e.AE44 != nil && e.AE44.Class == "Effect_DirectDamage" && SavedAreaPayloadSupported(e.AE44) {
		if rule, ok := w.findSpell(uint32(d.Spell)); ok {
			rule.Damaging, rule.Restorative = true, false
			rule.DamageMin = int32(e.AE44.DirectDamage[19])
			rule.DamageMax = rule.DamageMin + int32(e.AE44.DirectDamage[20])
			rule.School = e.AE44.DirectDamage[21]
			area := cellEffect{Key: d.Key, Spell: d.Spell, Mode: d.Mode}
			if len(caster) != 0 {
				area.Caster, area.HasCaster = caster[0], true
			}
			w.applyAreaCells(area, rule, cells)
		}
		return
	}
	kind, mode, known := savedAreaPayload(e.AE44)
	if !known {
		return
	}
	rule, ok := w.findSpell(uint32(e.AE44.E0C))
	if !ok {
		return
	}
	for _, key := range cells {
		x, y := keyCell(key)
		ground, air := w.cellLayerOccupants(x, y)
		groups := [][]int{ground}
		if d.Mode != areaModeCloud {
			groups = append(groups, air)
		}
		for _, group := range groups {
			for _, at := range group {
				if !spellTargetable(w.entities[at], rule) || len(caster) != 0 && !w.areaHitAllowed(indexOfEntity(w.entities, caster[0]), rule, w.entities[at]) {
					continue
				}
				applied := false
				if mode == 0 {
					_, ok := w.applyEffectDelta(at, kind, int32(e.AE44.E40))
					applied = ok || w.entities[at].ActorLoad.Source.Class == 0
					if applied {
						w.clearFelled(at)
					}
				} else {
					owner := ^EntityID(0)
					if len(caster) != 0 {
						owner = caster[0]
					}
					applied = w.attachEffect(w.entities[at].ID, owner, rule, kind, int32(int16(e.AE44.E40)), uint16(e.AE44.E40>>16), mode)
				}
				if applied {
					w.markSpellEffect(at, rule.ID)
					if rule.arm() == 17 && len(caster) != 0 && caster[0] != w.entities[at].ID {
						w.orderAttack(at, caster[0])
					}
				}
			}
		}
	}
}

func (w *World) tickSavedArea(i int, obs *castObs) {
	s := w.savedWorldEffects
	d := &s.Areas[i]
	if d.Root < 0 {
		return
	}
	e := &w.savedSpellEffects[d.Root]
	if e.SE40 != 0 {
		w.retireSavedArea(i)
		return
	}
	switch d.Mode {
	case areaModeCloud:
		if e.AE48[0] == 0 {
			w.paintSavedArea(i)
			return
		}
		if e.AE4C == 0 {
			w.retireSavedArea(i)
			return
		}
		e.AE4C--
		if e.AE4C%16 == 0 {
			w.applySavedAreaPayload(*d, *e, w.cloudPulseCells(d.Key, w.spellArm(d.Spell), int32(e.AE48[1])))
		}
	case areaModeRing:
		if e.AE4C > 0 {
			e.AE4C--
			return
		}
		e.AE4C = 2
		native := cellEffect{Key: d.Key, Spell: d.Spell, Mode: areaModeRing, Direction: e.AE48[2] >> 5}
		cells := w.ringStageCells(native, int(e.AE48[3]))
		obs.recordPaint(w, native, cells)
		w.applySavedAreaPayload(*d, *e, cells)
		e.AE48[3]++
		if int(e.AE48[3]) >= ringStageCount(w.spellArm(d.Spell)) {
			w.retireSavedArea(i)
		}
	case areaModeBlast:
		x, y := keyCell(d.Key)
		w.applySavedAreaPayload(*d, *e, w.blastCellsInMap(x, y, int32(e.AE48[1])))
		w.retireSavedArea(i)
		if d.Spell == fireBallSpell {
			w.releaseFireBallBursts(x, y, int32(e.AE48[1]), 0)
		}
	}
}

func (w *World) stepSavedWorldEffects(obs *castObs) {
	w.stepWorldSpellEffects(obs)
	w.releaseDueBursts()
	defer w.clearRetiredWorldEffectCarriers()
	s := w.savedWorldEffects
	if s == nil {
		return
	}

	for i := range s.Projectiles {
		w.stepSavedProjectile(&s.Projectiles[i])
	}
}

// stepSavedProjectile is one driver call of one carried projectile.
func (w *World) stepSavedProjectile(d *SavedProjectileDriver) {
	if d.Retired {
		return
	}
	p := w.savedProjectile(d.ID)
	if p.ActionSegments == 0 {
		w.savedProjectiles.Items = slices.DeleteFunc(w.savedProjectiles.Items, func(p SavedProjectile) bool { return p.ID == d.ID })
		w.savedProjectiles.IDs = slices.DeleteFunc(w.savedProjectiles.IDs, func(id uint16) bool { return id == d.ID })
		d.Retired = true
		return
	}
	resolved := false
	if d.TargetStructure {
		if at := indexOfStructure(w.structures, StructureID(d.Target)); at >= 0 {
			p.ActionX, p.ActionY = structureTargetPoint(w.structures[at])
			resolved = true
		}
	} else if d.HasTarget && !d.TargetDetached {
		deadTarget := false
		for _, dead := range w.OriginalDeadActors() {
			if dead.ID == d.Target {
				deadTarget = true
				if dead.Current.RuntimeID != 0 {
					p.ActionX, p.ActionY = int32(dead.Current.Cell&255)*256+int32(dead.Current.FineX), int32(dead.Current.Cell>>8)*256+int32(dead.Current.FineY)
					resolved = true
				}
				break
			}
		}
		if at := indexOfEntity(w.entities, d.Target); !deadTarget && at >= 0 {
			target := w.entities[at]
			p.ActionX, p.ActionY = w.savedProjectileTargetPoint(target)
			resolved = true
		}
	}
	// ANIM-139: an action-1 call that resolves its target aims actiondir from
	// the shot's pre-move point at the target, and every action-1 call copies
	// actiondir to dir; an unresolved target keeps the last actiondir.
	if p.Action == 1 {
		if resolved {
			p.ActionDir = ProjectileDirection(p.ActionX-p.X, p.ActionY-p.Y)
		}
		p.Dir = p.ActionDir
	}
	// ANIM-PROJ-025: signed division truncates toward zero; the final
	// positive segment reaches the current target, the next tick reaps it.
	p.ActionPhase++
	switch {
	case p.Picture == 34 || p.Picture == 36:
		// MAGIC-BOLTSTILL-072 / ANIM-BOLTRAMP-035: no position writes. An
		// actionphase outside 1..13 keeps the previous phase (MAGIC-281).
		ramp := [...]int32{4, 3, 2, 1, 0, 1, 2, 1, 0, 1, 2, 3, 4}
		if p.ActionPhase >= 1 && p.ActionPhase <= 13 {
			p.Phase = ramp[p.ActionPhase-1]
		}
	default:
		if savedAttachedProjectile(p.Picture) {
			p.X, p.Y, p.Z = p.ActionX, p.ActionY, p.ActionZ
		} else if p.Picture != teleportProjectilePicture {
			// ANIM-149: the picture-60 arm writes no position.
			p.X += (p.ActionX - p.X) / p.ActionSegments
			p.Y += (p.ActionY - p.Y) / p.ActionSegments
			p.Z += (p.ActionZ - p.Z) / p.ActionSegments
		}
		p.Phase = 0
		switch p.Picture { // ANIM-PHASECLOCK-028
		case 60:
			p.Phase = p.ActionPhase - 1
		case 51:
			p.Phase = p.ActionPhase
		case 20, 30:
			// ANIM-149: a call that finds its target sets phase 1.
			if resolved {
				p.Phase = 1
			} else if d.Phases > 0 {
				p.Phase = (p.ActionPhase / 2) % int32(d.Phases)
			}
		default:
			if d.Phases > 0 {
				p.Phase = (p.ActionPhase / 2) % int32(d.Phases)
			}
		}
	}
	p.LastAction = p.Action
	p.ActionSegments--
}

// Once every carried effect has left the world, its retired import ordinals
// cannot be reached by a later tick. Keep mixed live/retired carriers intact
// while the live roots still need their stable IDs. A retired projectile row
// leaves at once: a record names itself, and SAVE writes no retired record.
func (w *World) clearRetiredWorldEffectCarriers() {
	g, s := w.savedSpellGraph, w.savedWorldEffects
	dropped := false
	if s != nil {
		n := len(s.Projectiles)
		s.Projectiles = slices.DeleteFunc(s.Projectiles, func(d SavedProjectileDriver) bool { return d.Retired })
		dropped = len(s.Projectiles) != n
	}
	if g != nil && len(g.Roots) != 0 {
		return
	}
	residue := dropped || g != nil && len(g.Nodes) != 0
	if s != nil {
		residue = residue || len(s.Areas)+len(s.Projectiles) != 0
		for _, area := range s.Areas {
			if area.Root >= 0 || len(area.Cells) != 0 {
				return
			}
		}
		for _, projectile := range s.Projectiles {
			if !projectile.Retired {
				return
			}
		}
	}
	if !residue {
		return
	}
	w.savedSpellGraph, w.savedWorldEffects = nil, nil
	w.compactEffectOrder()
}

// Source current motion is authoritative when present. After native movement
// takes over, use the retained accepted stride and paid transit steps, never a
// newly derived speed or the already-published destination cell center.
func (w *World) savedProjectileTargetPoint(e Entity) (int32, int32) {
	if fx, fy, ok := w.ActorFinePosition(e.ID); ok {
		return e.X*256 + int32(fx), e.Y*256 + int32(fy)
	}
	if e.Stride.Present && e.Transit > 0 {
		elapsed := int32(e.TransitTotal - e.Transit)
		return e.Stride.FromX*256 + 128 + elapsed*int32(e.Stride.StepX), e.Stride.FromY*256 + 128 + elapsed*int32(e.Stride.StepY)
	}
	return e.X*256 + 128, e.Y*256 + 128
}

// Native removal policy (not a claim about ROM1's unresolved target lifetime):
// capture the actual current position before deleting the entity, preserve the
// source actiontarget key and finish the local countdown toward that last point.
func (w *World) detachSavedProjectileTargets(id EntityID) {
	if w.savedWorldEffects == nil {
		return
	}
	at := indexOfEntity(w.entities, id)
	if at < 0 {
		return
	}
	x, y := w.savedProjectileTargetPoint(w.entities[at])
	for _, dead := range w.OriginalDeadActors() {
		if dead.ID == id && dead.Current.RuntimeID != 0 {
			x, y = int32(dead.Current.Cell&255)*256+int32(dead.Current.FineX), int32(dead.Current.Cell>>8)*256+int32(dead.Current.FineY)
			break
		}
	}
	for i := range w.savedWorldEffects.Projectiles {
		d := &w.savedWorldEffects.Projectiles[i]
		if !d.Retired && d.HasTarget && !d.TargetDetached && !d.TargetStructure && d.Target == id {
			if p := w.savedProjectile(d.ID); p != nil {
				p.ActionX, p.ActionY = x, y
			}
			d.TargetDetached = true
		}
	}
}
