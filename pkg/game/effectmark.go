package game

import (
	"sort"

	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// THE CLIENT'S EFFECT-MARK LIST (1002; MAGIC-MARK-060).
//
// The original does not store the mark records. It stores, per actor, a list of
// one-dword elements — high 16 bits a kind, low 16 bits a countdown — which the
// simulation OPENS with opcode `0x88` and CLOSES with `0x89`, and it rebuilds
// the whole record array from that list on every rebuild. This file is that
// list and that rebuild.
//
// THE OPEN AND CLOSE ARE READ, NOT RECEIVED. This build has no message wire
// between the two halves; the simulation's own attached-effect set IS the set
// the two opcodes maintain there, so a kind present in sim.World.ActiveEffects
// is an element open and a kind absent is an element closed. A recast — which
// the original spells as `0x89` then `0x88` — is not visible in that set, so a
// recast keeps the element it had rather than restarting its phase, which is
// the one place this derivation is coarser than the wire it stands for.
//
// THE COUNTDOWN IS A PHASE CLOCK AND NOT A LIFETIME. An element opened by
// `0x88` starts at `0xffff` and is bounded by the closing message, and the
// rebuild decrements every element's low half once. Each builder takes its own
// modulus of it, so what the number does is select frames and angles; nothing
// expires by reaching zero on this path.

// markElementStart is the value the open message writes (MAGIC-MARK-060,
// `L05681`): the low half of a freshly opened element.
const markElementStart = 0xffff

// markElement is one open element: the actor it stands on, its kind, and its
// countdown. They are held in one flat slice in canonical (entity, kind) order
// so the rebuild's output order is a function of the set alone.
type markElement struct {
	Entity sim.EntityID
	Kind   int
	Count  uint16
}

// advanceEffectMarks is the rebuild's own list maintenance: open an element for
// every (actor, kind) the simulation now carries and this list does not, drop
// every element the simulation no longer carries, and decrement the rest.
//
// ONE REBUILD IS ONE TICK HERE. The original decrements once per client
// rebuild; this build advances the client's own per-tick state — bolts, heal
// bursts and cast runs — from one place, and this joins them, so a mark's phase
// runs at the world's tick rate rather than at the window's frame rate.
func (mw *mapWorld) advanceEffectMarks() {
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
		if !live[el.Entity][el.Kind] {
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
	sort.Slice(kept, func(i, j int) bool {
		if kept[i].Entity != kept[j].Entity {
			return kept[i].Entity < kept[j].Entity
		}
		return kept[i].Kind < kept[j].Kind
	})
	mw.markElements = kept
}

// markDraws is the rebuild itself: every element of one actor turned into
// records by the render tier's builders, each record beside the sheet its
// record index names.
//
// THE RECORD INDEX IS THE PICTURE ID. MAGIC-MARK-059 resolves a record's art
// through the same projectile record array MAGIC-PIC-026 addresses, by the byte
// at `record+0x06`, and that byte is the kind — which is `2*spellId + 8`, the
// even picture. So the lookup here is the projectile set's own subscript and no
// second table exists.
//
// A RECORD WHOSE PICTURE NAMES NO SHEET IS DROPPED, on spellDraw's own rule for
// the same three refusals: an install without the projectile registry draws no
// marks and opens its mission anyway.
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
		for _, m := range terrain.EffectMarkRecords(el.Kind, el.Count, tileSize) {
			out = append(out, ui.UnitMark{Sheet: sheet, Mark: m})
		}
	}
	return out
}

// stoneHeld reports whether this actor's animation frame is held, and marks the
// tick the hold began (1002; MAGIC-ACTOR-066).
//
// The unit draw tests for kind `0x30`, `stone_curse`, twice on its own account
// outside the mark passes, and the arm it gates holds the actor's animation
// frame. The live-selection clock stops at the value it had when the effect
// landed, while simulation supplies the owner-directed movement and attack
// gate and MapEntity.Stone supplies the decoded grayscale draw.
func (mw *mapWorld) stoneHeld(id sim.EntityID) (int, bool) {
	if !mw.world.HasEffectArm(id, 20) {
		delete(mw.stoneHold, id)
		return 0, false
	}
	if at, ok := mw.stoneHold[id]; ok {
		return at, true
	}
	if mw.stoneHold == nil {
		mw.stoneHold = make(map[sim.EntityID]int)
	}
	mw.stoneHold[id] = mw.scene
	return mw.scene, true
}
