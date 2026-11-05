package sim

import "encoding/binary"

// groupWord names the AI words of a Group record an order writes. This file
// is the one writer of SavedGroup.AI; the byte form and the SAV seam copy the
// whole block (internal/archtest).
type groupWord uint16

const (
	wordOrder     groupWord = 1 << iota // +0x20, the programme byte
	wordActive                          // +0x45, the dispatch gate
	wordCommanded                       // +0x0a..+0x0b, the commanded cell
	wordRoam                            // +0x15, the roam counter
	wordGeometry                        // +0x24, +0x28, +0x2a..+0x2c by the centroid rule
	wordPost                            // +0x00..+0x01, a copy of the centre cell +0x28
	wordNotice                          // +0x38 by the notice rule
	wordRadius                          // +0x2d by the radius rule
	wordLatch                           // +0x30 latch and +0x34 counter, with radiusLatch
	wordSpeed                           // +0x44, the formation rate term
	wordNamed                           // +0x48, the script-named mark
)

// noticeRule is how an order sets the notice base +0x38.
type noticeRule uint8

const (
	// noticeGiven stores the value the write carries.
	noticeGiven noticeRule = iota
	// noticeCentroid stores noticeBase about the members' sub-cell mean.
	noticeCentroid
	// noticeFromRadius stores the working radius +0x2d and clears +0x39,
	// after the radius rule.
	noticeFromRadius
)

// radiusRule is how an order sets the working radius +0x2d.
type radiusRule uint8

const (
	// radiusNotice installs the notice base beside it (AI-741).
	radiusNotice radiusRule = iota
	// radiusSpread is the widest sum +0x2c raised to MinimalGuardRange.
	radiusSpread
	// radiusLatch is live Guard's latch block (guardLatch).
	radiusLatch
)

// groupOrder is one order's Group record write: the words it sets and the
// rule each derived word takes. programme takes the order byte through the
// game profile's programme map (gameProfile.groupRoute); bytewise stores the
// commanded cell as one byte per axis, not as the cell word; countsRoam adds
// one to the roam counter instead of storing it.
type groupOrder struct {
	words      groupWord
	centroid   centroidRule
	notice     noticeRule
	radius     radiusRule
	programme  bool
	bytewise   bool
	countsRoam bool
}

// groupValues are the given words of one write.
type groupValues struct {
	order, active, roam, notice, speed uint8
	commanded                          cell
}

// The order rows. Each writer of a Group record names its row.
var (
	// rowOrder is the order byte alone: a script Stand Ground, Patrol's
	// order 0, a stop, Roam's stop helper and a Group with no member list.
	rowOrder = groupOrder{words: wordOrder}
	// rowOrderToCell is a script Swarm, Move or Swarm 2, or Roam: the order
	// byte and the commanded cell.
	rowOrderToCell = groupOrder{words: wordOrder | wordCommanded}
	// rowScriptGuard is a script Guard: the order byte, the notice base and
	// the working radius beside it (AI-741); no latch or counter store.
	rowScriptGuard = groupOrder{words: wordOrder | wordNotice | wordRadius, notice: noticeCentroid, radius: radiusNotice}
	// rowCommandIssue is a command's new Group: order byte, gate, commanded
	// cell and notice base.
	rowCommandIssue = groupOrder{words: wordOrder | wordActive | wordCommanded | wordNotice, notice: noticeCentroid}
	// rowCommandGuard is the guard issue, which installs the working radius
	// beside the base (AI-741).
	rowCommandGuard = groupOrder{words: rowCommandIssue.words | wordRadius, notice: noticeCentroid, radius: radiusNotice}
	// rowGate is the dispatch gate +0x45: a constructed Group's, the
	// activity rebuild's of either game and a current SAV root Group's.
	rowGate = groupOrder{words: wordActive}
	// rowStanceStand is the constructed Stand Ground of a SelfSlot Group.
	rowStanceStand = groupOrder{words: wordOrder, programme: true}
	// rowStanceGuard is the load-time guard setter with no range argument
	// (AI-450): the members' fine centroid at +0x24 and its cell at +0x28 and
	// +0x00, their spread at +0x2a..+0x2c, +0x2d = +0x38 = +0x2c raised to
	// MinimalGuardRange and +0x39 zero. An empty Group keeps the centroid.
	rowStanceGuard = groupOrder{words: wordOrder | wordGeometry | wordPost | wordRadius | wordNotice, programme: true,
		centroid: fineSubCells, radius: radiusSpread, notice: noticeFromRadius}
	// rowNativeRecord carries a native group record into a constructed
	// Group: its order, base, roam counter and commanded cell.
	rowNativeRecord = groupOrder{words: wordOrder | wordNotice | wordRoam | wordCommanded, programme: true, notice: noticeGiven}
	// rowNativeGuard is rowNativeRecord for a Guard record, whose working
	// radius is its base.
	rowNativeGuard = groupOrder{words: rowNativeRecord.words | wordRadius, programme: true, notice: noticeGiven, radius: radiusNotice}
	// rowNamed is the mark of a Group the script names (AI-452).
	rowNamed = groupOrder{words: wordNamed}
	// rowLiveGeometry is live Guard's first act (AI-739, AI-RADFREEZE-075):
	// the whole-cell mean at +0x24, its cell at +0x28 and the spread.
	rowLiveGeometry = groupOrder{words: wordGeometry, centroid: wholeCells}
	// rowLiveLatch is live Guard's latch block (AI-741, AI-GUARD-021).
	rowLiveLatch = groupOrder{words: wordRadius | wordLatch, radius: radiusLatch}
	// rowRoamStep is a roam roll's cell and counter (AI-ROAM-025).
	rowRoamStep = groupOrder{words: wordCommanded | wordRoam, bytewise: true}
	// rowRoamCount is a Roam decision's counter step.
	rowRoamCount = groupOrder{words: wordRoam, countsRoam: true}
	// rowCommanded is the commanded cell alone: a structure use's stop.
	rowCommanded = groupOrder{words: wordCommanded}
	// rowSpeed is the formation rate term (MOVE-GATE-035).
	rowSpeed = groupOrder{words: wordSpeed}
)

// commandIssueRow is the row of a command's new Group under order.
func commandIssueRow(order uint8) groupOrder {
	if order == orderGuard {
		return rowCommandGuard
	}
	return rowCommandIssue
}

// nativeRecordRow is the row that carries native record r.
func nativeRecordRow(r *groupAI) groupOrder {
	if r.order == orderGuard {
		return rowNativeGuard
	}
	return rowNativeRecord
}

// setGroupWords writes order o into g's AI words over the entities at
// members, in one fixed word order. w may be nil only for a row that reads
// no World: no programme map, centroid, notice, radius or latch.
func setGroupWords(w *World, g *SavedGroup, members []int, o groupOrder, v groupValues) {
	if o.words&wordOrder != 0 {
		b := v.order
		if o.programme {
			b = w.game.groupRoute.programme(b)
		}
		g.AI[0x20] = b
	}
	if o.words&wordActive != 0 {
		g.AI[0x45] = v.active
	}
	if o.words&wordCommanded != 0 {
		if o.bytewise {
			g.AI[10], g.AI[11] = uint8(v.commanded.x), uint8(v.commanded.y)
		} else {
			binary.LittleEndian.PutUint16(g.AI[10:], uint16(v.commanded.x)|uint16(v.commanded.y)<<8)
		}
	}
	if o.words&wordRoam != 0 {
		if o.countsRoam {
			g.AI[0x15]++
		} else {
			g.AI[0x15] = v.roam
		}
	}
	// A zero low-byte count leaves the old geometry: the stance setter and
	// live Guard both test it before dividing.
	if o.words&wordGeometry != 0 && uint8(len(members)) != 0 {
		cx, cy := groupCentroid(w.entities, members, o.centroid)
		binary.LittleEndian.PutUint32(g.AI[0x24:], uint32(uint16(cy))<<16|uint32(uint16(cx)))
		centre := o.centroid.centre(cx, cy)
		binary.LittleEndian.PutUint16(g.AI[0x28:], uint16(centre.x)|uint16(centre.y)<<8)
		g.AI[0x2a], g.AI[0x2b], g.AI[0x2c] = groupSpread(w.entities, members, centre)
	}
	if o.words&wordPost != 0 {
		copy(g.AI[0x00:0x02], g.AI[0x28:0x2a])
	}
	if o.words&wordNotice != 0 {
		switch o.notice {
		case noticeGiven:
			g.AI[0x38] = v.notice
		case noticeCentroid:
			g.AI[0x38] = centroidNotice(w.entities, members)
		}
	}
	if o.words&wordRadius != 0 {
		switch o.radius {
		case radiusNotice:
			g.AI[0x2d] = g.AI[0x38]
		case radiusSpread:
			g.AI[0x2d] = max(g.AI[0x2c], minimalGuardRange)
		case radiusLatch:
			w.latchWords(g)
		}
	}
	if o.words&wordNotice != 0 && o.notice == noticeFromRadius {
		g.AI[0x38], g.AI[0x39] = g.AI[0x2d], 0
	}
	if o.words&wordSpeed != 0 {
		g.AI[0x44] = v.speed
	}
	if o.words&wordNamed != 0 {
		binary.LittleEndian.PutUint32(g.AI[0x48:], 1)
	}
}

// latchWords is radiusLatch: a Group with a first member and latch +0x30
// zero rolls working radius +0x2d = base + 4 + jitter, clears counter +0x34
// and sets the latch; with the latch set it counts. An empty Group with the
// latch set rolls without the 4 and clears both; with it clear it counts.
// The jitter is floor(3*rand()/D) - 1 over the random service, D = 0x8000
// (AI-RANGE-102), which AI-741 grades Medium in play. The radius wraps as a
// byte and is read by the next clip. Both fields are taken as dwords; the
// claims do not fix their width.
func (w *World) latchWords(g *SavedGroup) {
	latch, count := binary.LittleEndian.Uint32(g.AI[0x30:]), binary.LittleEndian.Uint32(g.AI[0x34:])
	populated := len(g.Members) != 0
	switch {
	case populated && latch == 0:
		g.AI[0x2d] = g.AI[0x38] + noticeMargin + uint8(w.rng.aiRange(3)-1)
		latch, count = 1, 0
	case latch != 0 && !populated:
		g.AI[0x2d] = g.AI[0x38] + uint8(w.rng.aiRange(3)-1)
		latch, count = 0, 0
	default:
		count++
	}
	binary.LittleEndian.PutUint32(g.AI[0x30:], latch)
	binary.LittleEndian.PutUint32(g.AI[0x34:], count)
}

// groupSpread is the members' widest Chebyshev distance to centre, widest
// sight and widest sum of the two, each a byte: the three fields wordGeometry
// stores at +0x2a..+0x2c for the stance setter and live Guard alike.
func groupSpread(ents []Entity, members []int, centre cell) (spread, sight, widest uint8) {
	for _, i := range members {
		d, s := uint8(cellOf(&ents[i]).chebyshevTo(centre)), ents[i].ScanRange
		spread, sight, widest = max(spread, d), max(sight, s), max(widest, d+s)
	}
	return spread, sight, widest
}

// centroidNotice is noticeCentroid: noticeBase about the members' sub-cell
// mean, the centre zero for no member. The native group record's base takes
// it too (freezeGroups).
func centroidNotice(ents []Entity, members []int) uint8 {
	var cx, cy int32
	if len(members) > 0 {
		cx, cy = groupCentroid(ents, members, subCellMean)
	}
	return noticeBase(ents, members, cx, cy)
}

// RootGroupRecord is the Group a current SAV's actor root builds for an
// actor no Group record holds: its ID and selector, the dispatch gate set.
func RootGroupRecord(id, selector uint32) SavedGroup {
	g := SavedGroup{ID: id, Selector: selector}
	setGroupWords(nil, &g, nil, rowGate, groupValues{active: 1})
	return g
}
