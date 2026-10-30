package game

import (
	"againrom/pkg/sim"
	"fmt"
	"image"
	"slices"
)

// SnapshotSpellBolt is the wire form of a cast object an earlier build kept
// outside the World. Every object in flight is now a World record that SAVE
// writes as a Prj section, so nothing writes this form; an old envelope's rows
// are read and dropped. SnapshotHealBurst is the frontend's current shower.
type SnapshotSpellBolt struct {
	From, To         image.Point
	Picture          int
	Owner            uint32
	Age, Life, Delay int
	Seed             uint32
	Tag              int
	Facing           uint8
	Centered         bool
	// Launch is the start offset from From's centre (castOffset). An
	// envelope written before it existed restores zero: the object then
	// starts at its caster's cell centre for the rest of its short life.
	Launch image.Point
}
type SnapshotHealBurst struct {
	At          image.Point
	Picture     int
	Owner, Seed uint32
	Tile        int
	Drain       bool
	Age         int
}
type SnapshotTargetMark struct {
	Entity  sim.EntityID
	Element uint32
	Seed    uint32
}
type SnapshotCastRun struct {
	Entity            sim.EntityID
	Left, Span, Swing int
	Phase             sim.AttackPhase
}
type SnapshotVisualIdentity struct{ Entity, Label sim.EntityID }

// Native IDs supply the initial seed vocabulary. Once LOAD changes local IDs,
// these labels and their allocator remain independent of that address space.
// SAVE previews the same assignments without changing presentation state.
func (mw *mapWorld) currentVisualIdentities() ([]SnapshotVisualIdentity, sim.EntityID) {
	if mw.world == nil {
		return nil, mw.visualNext
	}
	var out []SnapshotVisualIdentity
	next := mw.visualNext
	localNext, _ := mw.world.NextEntityID()
	if mw.visualIDs == nil {
		next = localNext
	}
	for _, e := range mw.world.Entities() {
		label := e.ID
		if mw.visualIDs != nil {
			var ok bool
			label, ok = mw.visualIDs[e.ID]
			if !ok {
				label = next + e.ID - mw.visualLocalNext
			}
		}
		out = append(out, SnapshotVisualIdentity{e.ID, label})
	}
	if mw.visualIDs != nil {
		next += localNext - mw.visualLocalNext
	}
	return out, next
}
func (mw *mapWorld) restoreVisualIdentities(rows []SnapshotVisualIdentity, next sim.EntityID) {
	mw.visualIDs = nil
	mw.visualNext = next
	mw.visualLocalNext = 0
	if mw.world != nil {
		mw.visualLocalNext, _ = mw.world.NextEntityID()
	}
	if rows != nil {
		mw.visualIDs = map[sim.EntityID]sim.EntityID{}
		for _, v := range rows {
			mw.visualIDs[v.Entity] = v.Label
		}
	}
}
func (mw *mapWorld) visualActorLabel(id sim.EntityID) sim.EntityID {
	if mw.visualIDs == nil {
		return id
	}
	rows, next := mw.currentVisualIdentities()
	mw.restoreVisualIdentities(rows, next)
	if label, ok := mw.visualIDs[id]; ok {
		return label
	}
	return id
}
func (mw *mapWorld) visualCastSeed(caster, target sim.EntityID, spell int, atCell bool) uint32 {
	if mw.visualIDs == nil {
		return castSeed(caster, target, spell)
	}
	rows, next := mw.currentVisualIdentities()
	mw.restoreVisualIdentities(rows, next)
	if label, ok := mw.visualIDs[caster]; ok {
		caster = label
	}
	if !atCell {
		if label, ok := mw.visualIDs[target]; ok {
			target = label
		}
	}
	return castSeed(caster, target, spell)
}
func validateCastRuns(runs []SnapshotCastRun) error {
	seen := map[sim.EntityID]bool{}
	for _, r := range runs {
		if seen[r.Entity] || r.Left < 1 || r.Span < r.Left || r.Swing < -1 || r.Phase > sim.AttackBoundaryTwo {
			return fmt.Errorf("invalid current cast animation")
		}
		seen[r.Entity] = true
	}
	return nil
}
func (mw *mapWorld) restoreCastRuns(runs []SnapshotCastRun) {
	mw.castRun = map[sim.EntityID]castRun{}
	for _, r := range runs {
		mw.castRun[r.Entity] = castRun{r.Left, r.Span, &castRunIdentity{}}
		mw.swing[r.Entity] = r.Swing
		mw.phase[r.Entity] = r.Phase
	}
}
func validateSpellBolts(bolts []SnapshotSpellBolt) error {
	if len(bolts) > 65536 {
		return fmt.Errorf("current visual population exceeds bound")
	}
	for _, b := range bolts {
		if b.Age < 0 || b.Life <= 0 || b.Age >= b.Life || b.Delay < 0 || b.Picture < 0 {
			return fmt.Errorf("invalid current spell visual lifetime")
		}
	}
	return nil
}
func validateHealBursts(heals []SnapshotHealBurst) error {
	if len(heals) > 65536 {
		return fmt.Errorf("current visual population exceeds bound")
	}
	for _, b := range heals {
		if b.Age < 0 || b.Age >= healBurstLife || b.Picture < 0 || b.Tile < 1 {
			return fmt.Errorf("invalid current heal visual lifetime")
		}
	}
	return nil
}
func validateVisualIdentities(rows []SnapshotVisualIdentity) error {
	ids, labels := map[sim.EntityID]bool{}, map[sim.EntityID]bool{}
	for _, v := range rows {
		if ids[v.Entity] || labels[v.Label] {
			return fmt.Errorf("repeated current visual identity")
		}
		ids[v.Entity], labels[v.Label] = true, true
	}
	return nil
}
func (mw *mapWorld) restoreHealBursts(heals []SnapshotHealBurst) {
	mw.healBursts = nil
	for _, b := range heals {
		mw.healBursts = append(mw.healBursts, healBurst{b.At, b.Picture, b.Owner, b.Seed, b.Tile, b.Drain, b.Age})
	}
}

func validateTargetMarks(rows []SnapshotTargetMark) error {
	if len(rows) > 65536 {
		return fmt.Errorf("current target mark population exceeds bound")
	}
	seen := map[[2]uint32]bool{}
	for _, r := range rows {
		kind, count := r.Element>>16, uint16(r.Element)
		key := [2]uint32{uint32(r.Entity), kind}
		if (kind != 20 && kind != 30) || count == 0 || count > healBurstLife || seen[key] {
			return fmt.Errorf("invalid current target mark")
		}
		seen[key] = true
	}
	return nil
}

func (mw *mapWorld) restoreTargetMarks(rows []SnapshotTargetMark) {
	mw.markElements = slices.DeleteFunc(mw.markElements, func(el markElement) bool { return el.Transient })
	for _, r := range rows {
		e, ok := mw.entity(r.Entity)
		if !ok || e.OffMap {
			continue
		}
		kind := int(r.Element >> 16)
		mw.markElements = append(mw.markElements, markElement{Entity: r.Entity, Kind: kind, Picture: kind,
			Count: uint16(r.Element), Transient: true, Seed: r.Seed, Identity: targetMarkIdentity(e)})
	}
}
