package game

import (
	"sort"

	"againrom/pkg/render/terrain"
	"againrom/pkg/rules"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// Attached effects open at 0xffff; projectile pictures 20/30 refresh at 32.
// MAGIC-MARK-060 establishes kind/countdown pairing and by-kind replacement.
// One engine tick is one rebuild; native elapsed-time cadence remains unknown.
const markElementStart = 0xffff

// markElement is one open element: the actor it stands on, its kind (the
// geometry program), its picture, and its countdown. They are held in one flat
// slice in canonical (entity, kind) order so the rebuild's output order is a
// function of the set alone.
type markElement struct {
	Entity    sim.EntityID
	Kind      int
	Picture   int
	Count     uint16
	Transient bool
	Seed      uint32
	Identity  markTargetIdentity
}

// heldMark is the held mark a lasting effect draws: the geometry program that
// builds its records, where and on which clock, and the picture whose sheet
// draws them.
type heldMark struct{ program, picture int }

// secondHeldMarks selects the second game's held geometry and picture by arm
// (R2-ASSET-177/179, R2-ENGINE-693/698).
var secondHeldMarks = map[uint16]heldMark{
	5:  {terrain.MarkProtectionFire, 16},
	8:  {terrain.MarkPoisonCloud, 20},
	10: {terrain.MarkProtectionWater, 24},
	16: {terrain.MarkProtectionAir, 34},
	22: {terrain.MarkProtectionEarth, 46},
	23: {terrain.MarkProgramSecondBless, 48},
	18: {terrain.MarkProgramSecondShield, 62},
	27: {terrain.MarkProgramSecondCurse, 64},
}

// heldMarkOf is the held mark of an effect whose spell runs arm: the game's
// own row, else the first game's, where the arm's kind is both the program
// and the picture (MAGIC-MARK-061).
func (g *gameFiles) heldMarkOf(arm uint16) heldMark {
	if m, ok := g.heldMarks[arm]; ok {
		return m
	}
	kind := terrain.MarkKind(int(arm))
	return heldMark{kind, kind}
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
	files := mw.files()
	live := make(map[sim.EntityID]map[heldMark]bool)
	for _, e := range mw.world.ActiveEffects() {
		mark := files.heldMarkOf(mw.world.SpellArm(e.Spell))
		if live[e.Target] == nil {
			live[e.Target] = make(map[heldMark]bool)
		}
		live[e.Target][mark] = true
	}

	kept := mw.markElements[:0]
	held := make(map[sim.EntityID]map[heldMark]bool, len(live))
	for _, el := range mw.markElements {
		mark := heldMark{el.Kind, el.Picture}
		if el.Transient {
			if !mw.markTargetPresent(el) || el.Count <= 1 {
				continue
			}
		} else if !live[el.Entity][mark] {
			continue
		}
		el.Count--
		kept = append(kept, el)
		if held[el.Entity] == nil {
			held[el.Entity] = make(map[heldMark]bool)
		}
		held[el.Entity][mark] = true
	}
	for entity, marks := range live {
		for mark := range marks {
			if held[entity][mark] {
				continue
			}
			kept = append(kept, markElement{Entity: entity, Kind: mark.program, Picture: mark.picture,
				Count: markElementStart})
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
	if rules.PictureCapabilityOf(mw.world.SpellGame(), m.Picture).Mark == rules.TargetMarkNone {
		return
	}
	e, ok := mw.entity(m.Target)
	if !ok || e.OffMap || targetMarkIdentity(e) != sourceMarkIdentity(m.Binding) {
		return
	}
	el := markElement{Entity: m.Target, Kind: kind, Picture: kind, Count: healBurstLife, Transient: true,
		Seed: mw.presentationSeed(presentationIdentity{rule: seedTargetMark, target: m.Target, spell: (kind - 8) / 2}), Identity: targetMarkIdentity(e)}
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
		sheet := mw.projectiles.Sheet(el.Picture)
		if sheet == nil {
			continue
		}
		records := terrain.EffectMarkRecords(el.Kind, el.Picture, el.Count, tileSize)
		if el.Transient {
			if !mw.markTargetPresent(el) {
				continue
			}
			records = showerMarkRecords(healBurst{picture: el.Picture, seed: el.Seed, tile: tileSize,
				drain: rules.PictureCapabilityOf(mw.world.SpellGame(), int32(el.Kind)).Mark == rules.TargetMarkDrainLife,
				age:   healBurstLife - int(el.Count)})
		}
		for _, m := range records {
			out = append(out, ui.UnitMark{Sheet: sheet, Mark: m})
		}
	}
	return out
}
