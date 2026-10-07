package game

import (
	"math"
	"sort"

	"againrom/pkg/render/terrain"
)

// UnitAnimRow is one class's audit line: the descriptor's predicted sheet
// total against the sheet's own frame count, and the full selection domain
// split into the selections the bounds guard left unchanged and the ones it
// refused. Every tuple of the domain lands in exactly one of the two counts,
// so InRange + Guarded is the domain's size. A Predicted that differs from
// Frames — or a non-zero Guarded — is DATA about the install, recorded and
// never an error: the predicted total is a computed property, not an
// assumption (spec, "The sheet contract").
type UnitAnimRow struct {
	ID        int32 // the class id the bundle keys by
	Predicted int   // the descriptor's Total — a property of the class alone
	Frames    int   // len(Frames) — the sheet's own count, 0 for a frameless entry
	InRange   int   // selections the sheet's own count left unchanged
	Guarded   int   // selections the bounds guard refused
}

// UnitAnimAudit sweeps, per class of the set, the FULL selection domain —
// both states, all 8 octants, every step of each track — through
// terrain.SelectUnitFrame, and reports one row per class, ascending by id
// (AC-10).
//
// The rows are SORTED so the caller's output is deterministic over the map's
// own iteration order. A nil entry is skipped — it carries no class and no
// descriptor, so there is nothing to sweep — and a nil or empty set audits to
// no rows at all. Nothing here reads a frame's pixels: the sheet enters the
// audit as its count alone, exactly as it enters the selector.
func UnitAnimAudit(set *terrain.UnitSet) []UnitAnimRow {
	if set == nil {
		return nil
	}
	var rows []UnitAnimRow
	for id, c := range set.Classes {
		if c == nil {
			continue
		}
		rows = append(rows, auditUnitClass(id, c))
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].ID < rows[j].ID })
	return rows
}

// auditUnitClass sweeps one class's whole selection domain and counts each
// tuple in range or guarded.
//
// THE DOMAIN IS RECTANGULAR: both states x 8 octants x one full period of the
// LONGER track (at least one tick, for the trackless class whose selections
// are tick-independent). The tick range is shared by both states because the
// fallback chain crosses them — a mover failing its gate takes the idle
// selection, so the moving state can read the idle track — and a range
// covering the longer period reaches every step of BOTH tracks whatever the
// gates select: a track of period p repeats under any tick run of length
// >= p, and tick enters selection only as tick mod p.
//
// ONE INDEX DRIVES BOTH CLOCKS. The two arms no longer read the same number
// — the moving arm takes a distance walked and the idle arm a tick — so
// the sweep passes the index as the tick AND as the distance that stands on
// that timeline step. The domain keeps its shape and its size, and every
// step of both tracks is still reached.
//
// GUARDED IS MEASURED, NOT RE-DERIVED. Each tuple is selected twice through
// the layer's ONE selection path — once against the sheet's own count,
// once against an unbounded one — and counted guarded exactly when the
// sheet's count CHANGED the answer, or when the sheet holds no frame at all,
// where the guard refuses every index ([0, 0) is empty) and the seam draws
// the square. The one blind spot is inherent in measuring through the
// interface: a selection whose raw index is negative answers sheet frame 0
// at EVERY count, indistinguishable from a genuine frame-0 answer, and is
// counted in range — it draws the same pixels at every count, so the count
// changed nothing observable.
func auditUnitClass(id int32, c *terrain.UnitClass) UnitAnimRow {
	row := UnitAnimRow{ID: id, Predicted: c.Anim.Total, Frames: len(c.Frames)}
	period := max(1, len(c.Anim.MoveTrack), len(c.Anim.IdleTrack))
	for _, moving := range [2]bool{true, false} {
		for oct := 0; oct < 8; oct++ {
			for tick := 0; tick < period; tick++ {
				odo := terrain.WalkStepOdometer(tick)
				frame, mirror := terrain.SelectUnitFrame(c.Anim, row.Frames, moving, oct, tick, odo)
				raw, rawMirror := terrain.SelectUnitFrame(c.Anim, math.MaxInt, moving, oct, tick, odo)
				if row.Frames <= 0 || frame != raw || mirror != rawMirror {
					row.Guarded++
				} else {
					row.InRange++
				}
			}
		}
	}
	return row
}
