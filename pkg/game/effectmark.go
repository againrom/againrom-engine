package game

import (
	"sort"

	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// Attached effects open at 0xffff; projectile pictures 20/30 refresh at 32.
// MAGIC-MARK-060 establishes kind/countdown pairing and by-kind replacement.
// One engine tick is one rebuild; native elapsed-time cadence remains unknown.
const markElementStart = 0xffff

// markElement is one open element: the actor it stands on, its kind, and its
// countdown. They are held in one flat slice in canonical (entity, kind) order
// so the rebuild's output order is a function of the set alone.
type markElement struct {
	Entity    sim.EntityID
	Kind      int
	Count     uint16
	Transient bool
	Seed      uint32
	Identity  markTargetIdentity
}

type markTargetIdentity struct {
	Class             uint8
	Identity, Runtime uint32
}

func sourceMarkIdentity(b sim.SourceBinding) markTargetIdentity {
	return markTargetIdentity{b.Class, b.Identity, b.RuntimeID}
}

func targetMarkIdentity(e sim.Entity) markTargetIdentity { return sourceMarkIdentity(e.SourceBinding) }

func (mw *mapWorld) markTargetPresent(el markElement) bool {
	e, ok := mw.entity(el.Entity)
	return ok && !e.OffMap && targetMarkIdentity(e) == el.Identity
}

// advanceEffectMarks decrements old elements, then admits this tick's branch
// observations at 32. Projectile retirement does not close a target mark.
func (mw *mapWorld) advanceEffectMarks(marks ...sim.ProjectileMark) {
	live := make(map[sim.EntityID]map[int]bool)
	for _, e := range mw.world.ActiveEffects() {
		kind := terrain.MarkKind(int(mw.world.SpellArm(e.Spell)))
		if live[e.Target] == nil {
			live[e.Target] = make(map[int]bool)
		}
		live[e.Target][kind] = true
	}

	kept := mw.markElements[:0]
	held := make(map[sim.EntityID]map[int]bool, len(live))
	for _, el := range mw.markElements {
		if el.Transient {
			if !mw.markTargetPresent(el) || el.Count <= 1 {
				continue
			}
		} else if !live[el.Entity][el.Kind] {
			continue
		}
		el.Count--
		kept = append(kept, el)
		if held[el.Entity] == nil {
			held[el.Entity] = make(map[int]bool)
		}
		held[el.Entity][el.Kind] = true
	}
	for entity, kinds := range live {
		for kind := range kinds {
			if held[entity][kind] {
				continue
			}
			kept = append(kept, markElement{Entity: entity, Kind: kind, Count: markElementStart})
		}
	}
	mw.markElements = kept
	for _, m := range marks {
		mw.refreshProjectileMark(m)
	}
	kept = mw.markElements
	sort.Slice(kept, func(i, j int) bool {
		if kept[i].Entity != kept[j].Entity {
			return kept[i].Entity < kept[j].Entity
		}
		return kept[i].Kind < kept[j].Kind
	})
	mw.markElements = kept
}

func (mw *mapWorld) refreshProjectileMark(m sim.ProjectileMark) {
	kind := int(m.Picture)
	if kind != terrain.MarkHeal && kind != terrain.MarkDrainLife {
		return
	}
	e, ok := mw.entity(m.Target)
	if !ok || e.OffMap || targetMarkIdentity(e) != sourceMarkIdentity(m.Binding) {
		return
	}
	el := markElement{Entity: m.Target, Kind: kind, Count: healBurstLife, Transient: true,
		Seed: castSeed(0, mw.visualActorLabel(m.Target), (kind-8)/2), Identity: targetMarkIdentity(e)}
	for i, old := range mw.markElements {
		if old.Entity == m.Target && old.Kind == kind {
			mw.markElements[i] = el
			return
		}
	}
	mw.markElements = append(mw.markElements, el)
}

// markDraws sends every kind through its shared builder and actor mark passes.
func (mw *mapWorld) markDraws(id sim.EntityID, tileSize int) []ui.UnitMark {
	var out []ui.UnitMark
	for _, el := range mw.markElements {
		if el.Entity != id {
			continue
		}
		sheet := mw.projectiles.Sheet(el.Kind)
		if sheet == nil {
			continue
		}
		records := terrain.EffectMarkRecords(el.Kind, el.Count, tileSize)
		if el.Transient {
			if !mw.markTargetPresent(el) {
				continue
			}
			records = showerMarkRecords(healBurst{picture: el.Kind, seed: el.Seed, tile: tileSize,
				drain: el.Kind == terrain.MarkDrainLife, age: healBurstLife - int(el.Count)})
		}
		for _, m := range records {
			out = append(out, ui.UnitMark{Sheet: sheet, Mark: m})
		}
	}
	return out
}
