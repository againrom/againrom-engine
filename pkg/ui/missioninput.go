package ui

// THE MISSION INPUT CONTRACT.
//
// This file holds the three tables the mission map's own input is built from,
// and nothing that draws. Each is a pure function of the viewer's own state so
// that every arm is reachable in a test with no window.
//
// THE THREE ARE ONE CHAIN AND THAT IS THE POINT. The armed mode selects a
// cursor, the cursor selects an order, so a panel cell, a key and a bare
// hover all reach the world through one table rather than through three that
// agree today.

// The hover mask bits, named as `AI-CURSOR-231` numbers them. Four of the seven
// are reachable in this build and three are not; the three are named here
// anyway so that the arms that read them are transcriptions of the decoded
// cascade rather than a smaller cascade written from what happens to be
// implementable (`DIV-284`).
import "slices"

const (
	// hoverMaskUnit is `0x1`, set for a hit whose exact class name is `CUnit`.
	// Every drawable this build's hover hit test can reach is one.
	hoverMaskUnit uint32 = 0x1
	// hoverMaskAirUnit is `0x2`, a hit `CAirUnit`. NEVER SET HERE: this build
	// draws no separate flying-unit class and `MapEntity` carries no field to
	// distinguish one (`DIV-284`). Every arm below that reads `0x23` therefore
	// reduces to `0x1` plus `0x2`'s absence, which is the same set this build
	// can produce.
	hoverMaskAirUnit uint32 = 0x2
	// hoverMaskHostile is `0x4`, set for every class the hit test names when
	// the local participant is hostile to the hit object's owner.
	hoverMaskHostile uint32 = 0x4
	// hoverMaskStructure is `0x20`, set for EVERY hit `CStructure`
	// (`AI-CURSOR-231`). Visible inspection pixels provide a tagged identity.
	hoverMaskStructure uint32 = 0x20
	// hoverMaskDrawable is `0x40`: the hovered cell's four corners OR to
	// `0xc000` and the cell carries a drawable on the `CMapView+0x98` plane.
	// This build reads it as a ground sack the fog lets the player see
	// (`DIV-283`).
	hoverMaskDrawable uint32 = 0x40
	// hoverMaskFogged is `0x400`, the same comparison as `0x40` read the other
	// way: the four corners do NOT OR to `0xc000`. `AI-CURSOR-231` reads that
	// at Medium as no corner currently visible. The two are mutually exclusive
	// by construction here as they are there.
	hoverMaskFogged uint32 = 0x400
	// hoverMaskStructureCell is `0x800`, a hit `CStructure` whose cell record's
	// `+0x8c` is non-zero, carried IN ADDITION to `0x20`. Only the implemented
	// fountain/lever classes admit this capability (DIV-284).
	hoverMaskStructureCell uint32 = 0x800
)

// hoverMaskUnitAny is `0x23`, the mask the cascade tests to mean "a drawable
// with an actor identity was hit" — `CUnit`, `CAirUnit` or `CStructure`.
const hoverMaskUnitAny = hoverMaskUnit | hoverMaskAirUnit | hoverMaskStructure

// The selection-summary bits, named as `AI-PANEL-061` numbers `view+0x144`.
// Four are filled; the other two (`0x2` `CAirUnit`, `0x8` the primary's own
// `+0x18c` bit `0x1`) have no counterpart here (`DIV-286`).
const (
	// selSummaryUnit is `0x1`, set after a `strcmp` against `"CUnit"` on the
	// primary selected object.
	selSummaryUnit uint32 = 0x1
	// selSummaryForeign is `0x4`, set when the primary selected object's own
	// `CPlayer` is not the local participant's. It disables the command panel
	// outright (`AI-PANEL-060`) and, with `0x20`, forces the neutral cursor
	// pair in the cascade (`AI-CURSOR-203`).
	selSummaryForeign uint32 = 0x4
	// selSummaryStructure is `0x20`, a selected `CStructure`: set while a
	// plain click's structure is the selection and no unit is.
	selSummaryStructure uint32 = 0x20
	// AI-SPELLCAP-288
	selSummarySpell uint32 = 0x200
)

// selSummaryNeutral is `0x24`, the mask two independent routines test on
// `view+0x144` to reach a neutral outcome — the command panel's own blank
// notify and the cursor cascade's second arm (`AI-CURSOR-203`).
const selSummaryNeutral = selSummaryForeign | selSummaryStructure

const (
	modeNone   uint8 = 0
	modeAttack uint8 = 1
	modeMove   uint8 = 2
	modeDefend uint8 = 4
	modeCast   uint8 = 5
	modeSwarm  uint8 = 6
	modePatrol uint8 = 8
)

// missionModeCursor is the armed-mode table (`AI-PANEL-053`): the cursor an
// armed mode puts up, which is then the cursor a click is dispatched by.
//
// modeDefend IS IN THE TABLE AND UNREACHABLE IN THIS BUILD. Nothing here arms
// it, because this build has no Defend order for the arm to spend
// (`DIV-288`); the entry is written anyway so that the table is the decoded
// table and the gap is one missing writer rather than a missing row.
func missionModeCursor(mode uint8) (string, bool) {
	switch mode {
	case modeAttack:
		return "attack", true
	case modeMove:
		return "move", true
	case modeDefend:
		return "defend", true
	case modeCast:
		return "cast", true
	case modeSwarm:
		return "swarm", true
	case modePatrol:
		return "patrol", true
	}
	return "", false
}

// missionMode is which of the original's eight modes this viewer currently has
// armed, derived from the three pieces of arm state this build already keeps
// rather than replacing them: `v.armed` (the attack key and the panel's Attack
// cell), `v.aimed` (the panel's Move, Swarm and Patrol cells and their keys)
// and `v.spellArmed` (independent of the book's current spell).
//
// THE ORDER OF THE ARMS IS THE PRECEDENCE command.go ALREADY HAD, moved here
// and not changed: `v.aimed` above `v.armed` above the book. The original keeps
// ONE number in `view+0x99c` and so has no precedence to state; this build has
// three writers that already lower one another (`armAttack`, `armCommand`), and
// this chain is what makes the remaining overlap — a spell selected while a
// mode is armed — a defined single answer.
func (v *Viewer) missionMode() uint8 {
	switch {
	case v.aimed == commandPatrol:
		return modePatrol
	case v.aimed == commandSwarm:
		return modeSwarm
	case v.aimed == commandMove:
		return modeMove
	case v.aimed == commandDefend:
		return modeDefend
	case v.spellModeLive() || v.itemCast != nil:
		return modeCast
	case v.attackMode():
		return modeAttack
	}
	return modeNone
}

// selectionSummary is this build's `view+0x144` (`AI-PANEL-061`): a
// selection-derived flags word, rebuilt on every read rather than cached,
// because the selection and the entity snapshot it is derived from are both
// replaced whole every tick and a cache would need a key naming both.
func (v *Viewer) selectionSummary() uint32 {
	present := presentSelected(v.sel, v.entities)
	if len(present) == 0 {
		if _, ok := v.selectedStructure(); ok {
			return selSummaryStructure
		}
		return 0
	}
	// The PRIMARY is `view+0x138`, the first object the rebuild loop accepted
	// (`AI-CURSOR-202`), which here is the lowest present id: the selection is
	// ascending by construction and presentSelected preserves that order.
	primary := present[0]
	m := selSummaryUnit
	if v.localOwner != 0 && primary.Owner != v.localOwner {
		m |= selSummaryForeign
	}
	for _, e := range present {
		if e.CastCapable || !e.SpellStateKnown && e.MaxMana > 0 {
			m |= selSummarySpell
		}
	}
	return m
}

// hoverMask is the hover hit test restated over the surfaces this build has:
// the entity under the pointer, the sack under the pointer's cell, and
// whether the pointer's cell is under fog.
//
// IT ANSWERS FOR THE MAP SURFACE ALONE. A pointer that is not over the drawn
// map has no cell and hits nothing, which is `hoverMaskFogged` set and nothing
// else — the same answer a fully fogged cell gives, and the answer that makes
// every cursor arm below fall to the ground arm rather than to a unit arm.
//
// THE ENTITY HIT IS THE ATTACK PRESS'S POPULATION, not the selection hit:
// targetAt includes a body in the -1 through -9 finishing band and excludes an
// untargetable -10 corpse. Selection continues to use topAt and cannot select a
// body. The fog gate is the entity's own: an entity the party cannot see must
// not change the cursor. The sack's gate is fogGateSack, its own counterpart.
func (v *Viewer) hoverMask(x, y int) (mask uint32, hit uint32, hasHit bool) {
	col, row, inside := v.groundCellAt(float64(x), float64(y))
	if !inside {
		return hoverMaskFogged, 0, false
	}
	if !v.fogGateSack(col, row) {
		mask |= hoverMaskFogged
	} else {
		mask |= hoverMaskDrawable & sackMaskAt(v, col, row)
	}
	if id, ok := targetAt(v.entities, v.entityPickRect, float64(x), float64(y), false); ok {
		for _, e := range v.entities {
			if e.ID != id {
				continue
			}
			if !v.fogGateEntity(e.Owner, e.Cell.X, e.Cell.Y) {
				break
			}
			mask |= hoverMaskUnit
			if e.Hostile {
				mask |= hoverMaskHostile
			}
			hit, hasHit = id, true
			break
		}
	}
	if !hasHit {
		if ref, ok := v.inspectionAt(x, y); ok && ref.Kind == InspectionStructure && v.structureHoverHit(ref) {
			mask |= hoverMaskStructure
			if v.usableStructure(ref) {
				mask |= hoverMaskStructureCell
			}
		}
	}
	return mask, hit, hasHit
}

// structureHoverHit skips a class that is indestructible and not usable
// (AI-CURSOR-231, REG-STR-080).
func (v *Viewer) structureHoverHit(ref InspectionSubject) bool {
	c := v.structureInfo[ref.ID]
	return c == nil || c.Usable || !c.Indestructible
}

// sackMaskAt answers `hoverMaskDrawable` when this build's own sack registry
// holds a sack at (col, row), and zero otherwise.
func sackMaskAt(v *Viewer, col, row int) uint32 {
	for _, s := range v.sacks {
		if s.Cell.X == col && s.Cell.Y == row {
			return hoverMaskDrawable
		}
	}
	return 0
}

// pickupGate is `AI-CURSOR-242`'s `[EBP-0x68]`, whole: the single stack slot
// both `pickup` arms of the ordinary cascade read, built once before either arm
// runs.
//
// THREE ANDed TESTS ENDING IN A TWO-ARMED OR. Exactly one object selected and
// that object a `CUnit` (`view+0x140 == 1` and `view+0x144 & 0x1`); that
// object's own `+0x18c` bit `0x1`, which `PARTY-FLAG-003` reads at Medium as
// the player-character flag; and then either the hover mask equal to EXACTLY
// `0x40` — no other of its seven bits set, so no actor was hit — or the hit
// object being the selection itself with `0x40` also set.
//
// THE TWO GATES THE CASCADE ALREADY IMPOSED ARE NOT REPEATED HERE. `view+0x144
// & 0x24` clear and mask bit `0x4` clear are arms of the cascade above this
// call, not terms of the slot, and asking them twice would make the two copies
// able to disagree.
//
// `+0x18c` BIT `0x1` IS `MapEntity.PlayerCharacter` (the world's own guarded
// set, `pkg/game/world.go`'s guardedEntities). It is the same field and bit
// `AI-CURSOR-209` cites for its own `town` composite.
func pickupGate(present []MapEntity, mask, hit uint32, hasHit bool) bool {
	if len(present) != 1 {
		return false
	}
	sel := present[0]
	if !sel.PlayerCharacter {
		return false
	}
	if mask == hoverMaskDrawable {
		return true
	}
	return hasHit && hit == sel.ID && mask&hoverMaskDrawable != 0
}

// townGate is `AI-CURSOR-209`'s composite, amended by `AI-CURSOR-242` and
// `PARTY-FLAG-003`: exactly one `CUnit` selected, that unit's own `+0x18c` bit
// `0x1` set, and hover mask bit `0x800` set.
//
// The capability bit is supplied only for implemented usable structures.
func townGate(present []MapEntity, mask uint32) bool {
	if len(present) != 1 || !present[0].PlayerCharacter {
		return false
	}
	return mask&hoverMaskStructureCell != 0
}

// AI-CURSOR-052, DIV-262, DIV-270, UNIT-HOVER-020
func hoverCursorName(present []MapEntity, summary, mask, hit uint32, hasHit, ctrl, alt bool) string {
	if len(present) == 0 {
		if mask&hoverMaskUnitAny != 0 {
			return "select"
		}
		return "default"
	}
	if summary&selSummaryNeutral != 0 {
		if mask&hoverMaskUnitAny != 0 {
			return "select"
		}
		return "default"
	}
	if mask&hoverMaskHostile != 0 {
		if alt {
			return "move"
		}
		if mask&hoverMaskStructure != 0 {
			return "select"
		}
		return "attack"
	}
	if mask&hoverMaskUnitAny != 0 {
		if alt {
			return "move"
		}
		if ctrl {
			return "attack"
		}
		if pickupGate(present, mask, hit, hasHit) {
			return "pickup"
		}
		if townGate(present, mask) {
			return "town"
		}
		return "select"
	}
	if ctrl {
		return "swarm"
	}
	if alt {
		return "move"
	}
	if pickupGate(present, mask, hit, hasHit) {
		return "pickup"
	}
	return "move"
}

// cursorOrderKind is `AI-CLICK-050`'s dispatch: which order a click made under
// this cursor produces. It returns one of the orderKind values below.
//
// A CURSOR THIS TABLE DOES NOT NAME PRODUCES NOTHING, which is the routine's
// own shape — it compares the current cursor against each registered cursor's
// own handle, one arm per cursor, and a cursor with no arm falls out of the
// chain. The eight edge arrows, the five minimap cursors, `default`,
// `backpack`, `wait`, `cantput` and `dice` are all in that position.
type orderKind uint8

const (
	// orderKindNone is a cursor with no arm, and the zero value.
	orderKindNone orderKind = iota
	orderKindSelect
	// orderKindMove is `move` → opcode `0x16`.
	orderKindMove
	// orderKindAttack is `attack` → `0x19` at a qualifying target, `0x16`
	// otherwise. The class test at the qualifying arm is a runtime-class test
	// and no diplomacy is consulted at click time.
	orderKindAttack
	// orderKindSwarm is `swarm` → `0x1a`.
	orderKindSwarm
	// orderKindPatrol is `patrol` → `0x1d`.
	orderKindPatrol
	// orderKindDefend is `defend` → `0x1b` at an actor, and NOTHING AT ALL
	// with no actor under the cursor. Unreachable in this build: nothing arms
	// modeDefend (`DIV-288`).
	orderKindDefend
	// orderKindPickup is `pickup` → `0x21`.
	orderKindPickup
	// orderKindTown is `town` → `0x24`, one fountain or lever use.
	orderKindTown
	// orderKindCast is `cast` → `0x25`/`0x1e` at a unit and `0x1f`/`0x26` at a
	// cell.
	orderKindCast
)

func cursorOrderKind(cursor string) orderKind {
	switch cursor {
	case "select":
		return orderKindSelect
	case "move":
		return orderKindMove
	case "attack":
		return orderKindAttack
	case "swarm":
		return orderKindSwarm
	case "patrol":
		return orderKindPatrol
	case "defend":
		return orderKindDefend
	case "pickup":
		return orderKindPickup
	case "town":
		return orderKindTown
	case "cast":
		return orderKindCast
	}
	return orderKindNone
}

// -------------------------------------------------------------- the key
// forms of the selection routine (`AI-SELECT-122`, `AI-KEY-125`)

// selectAllOwnedUnits is the `E` key: "select every owned CUnit; deselect
// other classes and owners" (`AI-KEY-125`), which `AI-SELECT-122` states as
// "E selects every owned exact-name `CUnit`".
//
// EXACT-NAME `CUnit` IS READ HERE AS "a unit and not a structure". This build
// has no class name on `MapEntity` and no structure entities in the entity
// snapshot at all -- structures are terrain placements on a separate list --
// so every member of that snapshot is already the class this key selects. The
// filter that remains is ownership, which is the clause the same two rows
// state twice. Recorded as `DIV-284`, the row that carries this build's whole
// absence of a class name on `MapEntity`.
//
// A LOCAL PARTICIPANT OF ZERO TAKES EVERYTHING, which is `ownedCandidate`'s
// own reading of an unestablished participant and is what the developer viewer
// and every fixture with no roster run as.
func (v *Viewer) selectAllOwnedUnits() {
	next := make(selection, 0, len(v.entities))
	for _, e := range v.entities {
		if !selectableEntity(e) {
			continue
		}
		if v.localOwner != 0 && e.Owner != v.localOwner {
			continue
		}
		next = append(next, e.ID)
	}
	if len(next) == 0 {
		v.sel = nil
		return
	}
	slices.Sort(next)
	v.sel = next
}

// groupKey is one press of the digit row or of the numpad's own digits, which
// `AI-KEY-125` gives three actions and one precedence:
//
//	plain      select group n, replacing the selection
//	Shift      select group n, AUGMENTING the selection rather than replacing
//	Ctrl       assign the current owned selection to group n
//	Alt        select group n and centre the camera at the member mean (0x406)
//	Ctrl+Alt   Ctrl wins
//
// AN ASSIGNMENT STORES THE OWNED MEMBERS ONLY -- "assign current owned
// selection to group" -- so a foreign object a plain click put in the
// selection is not carried into a group. A recall of an unassigned or emptied
// group leaves the selection alone rather than clearing it, on
// `AI-SELECT-122`'s own rule for a rectangle that qualifies nobody: a
// selection form with no qualifier preserves.
func (v *Viewer) groupKey(n int, ctrl, alt, shift bool) {
	if n < 0 || n >= len(v.groups) {
		return
	}
	if ctrl {
		owned := make(selection, 0, len(v.sel))
		for _, id := range v.sel {
			if ownedCandidate(v.entities, id, v.localOwner) {
				owned = append(owned, id)
			}
		}
		v.groups[n] = owned
		return
	}

	recalled := presentSelected(v.groups[n], v.entities)
	if len(recalled) == 0 {
		return
	}
	next := make(selection, 0, len(recalled)+len(v.sel))
	if shift {
		next = append(next, v.sel...)
	}
	for _, e := range recalled {
		if !slices.Contains(next, e.ID) {
			next = append(next, e.ID)
		}
	}
	slices.Sort(next)
	v.sel = next

	if alt {
		// The member mean, in world coordinates, which is what message 0x406
		// carries the camera to.
		var sx, sy float64
		for _, e := range recalled {
			cx, cy := v.cellWorldCentre(e.Cell)
			sx, sy = sx+cx, sy+cy
		}
		n := float64(len(recalled))
		v.cam.CenterOn(sx/n, sy/n)
	}
}
