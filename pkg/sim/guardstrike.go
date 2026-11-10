package sim

import (
	"encoding/binary"
	"fmt"
)

// SavedStrike is the second game's blow record and pursuit stamp on a
// member's command: the striker at +0x94, the fine clock at the blow at
// +0x98, the cell +0x9c on the line from the striker beyond the member, and
// the server word +0 the stamp wrote to +0x16 (R2-ENGINE-845,
// R2-ENGINE-846). Held marks a blow, Stamped a stamp; an unstamped +0x16
// reads as the constructor's 65500.
type SavedStrike struct {
	Attacker EntityID
	Clock    uint32
	Away     uint16
	Stamp    uint16
	Held     bool
	Stamped  bool
}

const (
	// unstampedCommand is the constructor's cmd +0x16 (R2-ENGINE-846).
	unstampedCommand = 65500
	// strikeAwayCells scales the line from the striker to +0x9c
	// (R2-ENGINE-845, DIV-4503).
	strikeAwayCells = 10
)

// groupClock is server word +0, which counts group ticks, and fineClock the
// server dword +4 (R2-SESSION-343). The word is the count of group passes,
// tick / scriptCycle; the dword is the World tick (DIV-4503).
func (w *World) groupClock() uint16 { return uint16(w.tick / scriptCycle) }
func (w *World) fineClock() uint32  { return uint32(w.tick) }

// recordBlow writes the blow's record on victim ti's command o: the striker
// ai, the fine clock and the cell strikeAwayCells beyond the victim on the
// line from the striker, held inside the map (R2-ENGINE-845).
func (w *World) recordBlow(ai, ti int, o *SavedActorOrder) {
	a, v := w.entities[ai], w.entities[ti]
	dx, dy := v.X-a.X, v.Y-a.Y
	x, y := v.X, v.Y
	if d := isqrt(dx*dx + dy*dy); d > 0 {
		x += roundedQuotient(strikeAwayCells*dx, d)
		y += roundedQuotient(strikeAwayCells*dy, d)
	}
	x, y = min(max(x, 0), w.bounds.Width-1), min(max(y, 0), w.bounds.Height-1)
	o.Strike.Attacker, o.Strike.Clock, o.Strike.Held = a.ID, w.fineClock(), true
	o.Strike.Away = uint16(uint8(x)) | uint16(uint8(y))<<8
}

// CommandStamp is cmd +0x16 as k holds it: the stamp, else the
// constructor's 65500.
func (k SavedStrike) CommandStamp() uint16 {
	if k.Stamped {
		return k.Stamp
	}
	return unstampedCommand
}

// roundedQuotient is n / d rounded half away from zero, for d > 0.
func roundedQuotient(n, d int32) int32 {
	if n < 0 {
		return -((-n*2 + d) / (2 * d))
	}
	return (n*2 + d) / (2 * d)
}

// stampMissedApproach writes +0x16 for attacker i whose approach search
// missed goal and settled for sub, the zero cell for none: a member of a
// Group running the stamp window, with cmd +8 = 5, when the gap from sub
// to goal exceeds its attack range (R2-ENGINE-873, R2-ENGINE-878).
func (w *World) stampMissedApproach(i int, sub, goal cell) {
	e := w.entities[i]
	o, g := w.savedOrder(e.ID), w.savedGroupFor(e.ID)
	if o == nil || g == nil || o.Raw[8] != 5 || !w.game.groupRoute.runsStampWindow(g.AI[0x20]) {
		return
	}
	if approachGap(sub, goal, footprintSide(e.TokenSize)) > w.attackRange(i) {
		o.Strike.Stamp, o.Strike.Stamped = w.groupClock(), true
	}
}

// approachGap is the stamp's per-axis distance from a to b for a mover of
// side size: b - a - size + 1, floored at 0, on an axis where a < b, else
// a - b; the larger axis (R2-ENGINE-878).
func approachGap(a, b cell, size int32) int64 {
	axis := func(a, b int32) int64 {
		if a < b {
			return max(int64(b)-int64(a)-int64(size)+1, 0)
		}
		return int64(a) - int64(b)
	}
	return max(axis(a.x, b.x), axis(a.y, b.y))
}

// strikeWindow is guard B's cases 1 and 2: |+0x16 - server word +0| below
// 30 while hurt, else below 10; neither tests the blow (R2-ENGINE-844,
// R2-ENGINE-883).
func (w *World) strikeWindow(i int, o *SavedActorOrder) bool {
	d := int32(o.Strike.CommandStamp()) - int32(w.groupClock())
	if d < 0 {
		d = -d
	}
	e := w.entities[i]
	return e.HP != e.MaxHP && d < 30 || d < 10
}

// strikeRecent is guard B's case 4: a hurt member whose blow's +0x98 is
// within 10 of the fine clock (R2-ENGINE-844).
func (w *World) strikeRecent(i int, o *SavedActorOrder) bool {
	e := w.entities[i]
	d := int64(o.Strike.Clock) - int64(w.fineClock())
	return o.Strike.Held && e.HP != e.MaxHP && d > -10 && d < 10
}

// LoadedStrike is a member's blow record, stamp and alarm as a second-game
// command block holds them (R2-ENGINE-845, R2-ENGINE-846): Alarm, Notice
// and Scans are +0x54, +0x58 and +0x5a; Struck marks a nonzero +0x94 and
// Attacker is the striker the World holds for it, 0 for none.
type LoadedStrike struct {
	Entity, Attacker EntityID
	Struck           bool
	Stamp            uint16
	Alarm, Scans     uint8
	Notice           uint16
	Clock            uint32
	Away             uint16
}

// ImportLoadedStrikes writes each row into its member's order. A missing
// order is built by the stance setter for a Group that buildsMissingOrders
// when the row holds a blow or alarm, or a stamp its Group's case 2 reads
// (R2-ENGINE-883). Other rows leave a missing order absent until dispatch.
func (w *World) ImportLoadedStrikes(rows []LoadedStrike) error {
	for _, r := range rows {
		blow := r != (LoadedStrike{Entity: r.Entity, Stamp: r.Stamp})
		if !blow && r.Stamp == unstampedCommand {
			continue
		}
		i := indexOfEntity(w.entities, r.Entity)
		if i < 0 || r.Attacker != 0 && indexOfEntity(w.entities, r.Attacker) < 0 {
			return fmt.Errorf("sim: loaded strike names an actor the World does not hold")
		}
		o := w.savedOrder(r.Entity)
		if o == nil {
			g := w.savedGroupFor(r.Entity)
			if g == nil || !w.buildsMissingOrders(g) || !blow && !w.game.groupRoute.runsStampWindow(g.AI[0x20]) {
				continue
			}
			if o = w.constructMissingOrder(i); o == nil {
				continue
			}
		}
		o.Raw[0x54], o.Raw[0x5a] = r.Alarm, r.Scans
		binary.LittleEndian.PutUint16(o.Raw[0x58:], r.Notice)
		o.Strike = SavedStrike{Attacker: r.Attacker, Clock: r.Clock, Away: r.Away, Held: r.Struck}
		if r.Stamp != unstampedCommand {
			o.Strike.Stamp, o.Strike.Stamped = r.Stamp, true
		}
	}
	return nil
}

// holdStrikeWindow is guard B's cases 1 and 2 for member i. A member that is
// no human's unit keeps its unit target with its pursuit idle and its
// search forgotten, so the target case attacks it again after the window
// and its approach searches afresh (R2-ENGINE-874); any other target is
// released. A member holding a blow walks to +0x9c; one holding none
// holds its cell (DIV-4544).
func (w *World) holdStrikeWindow(i int, o *SavedActorOrder, human bool) {
	e := &w.entities[i]
	if !human && e.HasAttackTarget {
		if e.AttackTargetKind == AttackTargetUnit {
			e.AcquirePursuit, e.PursuitIdle, e.Pursuit = false, true, PursuitSearch{}
		} else {
			w.releaseAttack(i)
		}
	}
	if e.HasAttackTarget && !e.PursuitIdle {
		return
	}
	if !o.Strike.Held {
		if e.HasTarget {
			w.clearOrder(i)
		}
		return
	}
	w.walkToCell(i, int32(o.Strike.Away&0xff), int32(o.Strike.Away>>8), true)
	if w.entities[i].HasTarget {
		w.syncSavedDestination(i)
	}
}
