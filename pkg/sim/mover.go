package sim

import (
	"cmp"
	"encoding/binary"
	"slices"

	"againrom/pkg/random"
)

// facingKeySeed is the seed of the start facing's keyed draw in seeded mode.
// It is the engine's own constant, not a game value: the keyed draw takes no
// value from any stream, so the placement stream and the World stream draw as
// they would without it.
const facingKeySeed uint64 = 0x666163696E67

// constructorFacingDraw is MOVE-119's draw: the actor base constructor takes
// one value over 0..0x80 from the shared stream. On an original stream r is
// that stream and one value is consumed; on a seeded stream the draw is keyed
// by the actor's identity and r is not touched.
func constructorFacingDraw(r *rng, id EntityID) int32 {
	if r.original {
		return r.uniform(0x80)
	}
	return random.Keyed(facingKeySeed, random.StartFacing, uint64(id), 0x80)
}

// constructorFacing is the facing the actor base constructor stores at Mover
// +0x00 and copies to +0x01 (MOVE-119): 0x40 plus the draw, 0x40..0xC0.
func constructorFacing(r *rng, id EntityID) uint8 {
	return uint8(0x40 + constructorFacingDraw(r, id))
}

// constructFacing gives e the constructor facing and wants no turn. Every path
// that constructs an actor in a first-game World calls it once, in the
// original's order of construction; a second-game World keeps its facing.
func (w *World) constructFacing(e *Entity) {
	if w.SecondGame() {
		return
	}
	e.Facing = constructorFacing(&w.rng, e.ID)
	e.DesiredFacing = e.Facing
}

// ConstructPlacementFacings is the type-6 spawner's share of MOVE-119: each
// map actor, in record order, is built by the actor base constructor, which
// draws its facing. d is the mission start's stream; on an original stream
// one value per actor is consumed before the join walk's draws (MOVE-122). A
// placement the loader refuses is constructed first and deleted after
// (MOVE-120, MOVE-141), so it draws its facing too and keeps no actor:
// withdrawnBefore lists, in record order, how many World actors precede each
// such placement. It runs on an unadvanced World only.
func (w *World) ConstructPlacementFacings(d *Draws, withdrawnBefore []int) {
	if w == nil || w.tick != 0 || w.SecondGame() || d == nil {
		return
	}
	next := 0
	discard := func(upTo int) {
		for next < len(withdrawnBefore) && withdrawnBefore[next] <= upTo {
			constructorFacingDraw(&d.r, 0)
			next++
		}
	}
	for i := range w.entities {
		discard(i)
		e := &w.entities[i]
		e.Facing = constructorFacing(&d.r, e.ID)
		e.DesiredFacing = e.Facing
	}
	discard(len(w.entities))
}

// moverRecord is the part of an actor's Mover a first-game World holds and
// SAVE cannot take from the actor itself: the block-test mask (+0x05) and
// whether the step-cost word +0x72 is still the zero the join walk leaves
// (MOVE-122) because the actor has made no crossing since. SAVE writes the
// facing, the rate, the step cost and the motion from the actor, and the
// constructor's +0x08/+0x09 (ApplyMover, ProjectActorMotion).
type moverRecord struct {
	Entity EntityID
	Mask   byte
	Joined bool
}

const moverRecordLen = 2

func (r moverRecord) appendHeld(b []byte) []byte {
	joined := byte(0)
	if r.Joined {
		joined = 1
	}
	return append(b, r.Mask, joined)
}

func moverRecordFrom(id EntityID, b []byte) (moverRecord, bool) {
	return moverRecord{Entity: id, Mask: b[0], Joined: b[1] == 1}, b[1] <= 1
}

// stepCostWord is the word the occupy and the crossing store at Mover +0x72
// (MOVE-121, MOVE-128): for a ground mover the speed word times eight over the
// cell's cost byte, for the other two domains the speed word. A cost byte of 0
// raises a divide error in the original; the engine stores 0 (DIV-3033).
func (w *World) stepCostWord(e Entity, c cell) uint16 {
	speed := int32(int16(e.SpeedWord()))
	if e.Domain != DomainGround {
		return uint16(speed)
	}
	if i, ok := w.cellIndex(c.x, c.y); !ok || i >= len(w.cost) {
		return 0
	}
	cost := int32(w.costAt(c))
	if cost == 0 {
		return 0
	}
	return uint16(speed * 8 / cost)
}

// constructedMover is the one Mover builder. It returns the Mover an actor
// holds once the original has built it: the constructor's bytes (MOVE-119),
// then for a placed actor its facing, the stat default and the derived rate,
// the mask of its domain and the occupy's stores (MOVE-120, MOVE-121). A
// carried member placed by the join walk holds the constructor's bytes alone
// (MOVE-122).
func (w *World) constructedMover(e Entity, joined bool) [180]byte {
	m := placedMover(e, joined)
	if !joined {
		binary.LittleEndian.PutUint16(m[0x72:], w.stepCostWord(e, cell{e.X, e.Y}))
		m[0x82], m[0x83], m[0x84], m[0x85] = byte(e.X), byte(e.Y), 0x80, 0x80
	}
	return m
}

// placedMover is constructedMover before the occupy's stores.
func placedMover(e Entity, joined bool) [180]byte {
	var m [180]byte
	m[5], m[8], m[9], m[10] = 0x41, 5, 0xff, 0x10
	if joined {
		return m
	}
	m[0], m[1], m[10] = e.Facing, e.Facing, moverRate(e)
	m[5] = MoverPassabilityMask(uint32(e.Domain) + 1)
	return m
}

// moverRate is the rate byte +0x0a an actor holds now: its source's rate,
// else its own turn rate.
func moverRate(e Entity) uint8 {
	if e.ActorLoad.Source.Class != 0 {
		return e.SourceNow().MoverSpeed
	}
	return uint8(e.RotationSpeed)
}

// heldMover is the record a World keeps of an actor's Mover bytes m.
func heldMover(e Entity, m *[180]byte, joined bool) moverRecord {
	return moverRecord{Entity: e.ID, Mask: m[5], Joined: joined}
}

// builtMover is the record the builder gives e.
func (w *World) builtMover(e Entity, joined bool) moverRecord {
	m := placedMover(e, joined)
	return heldMover(e, &m, joined)
}

func (w *World) moverIndex(id EntityID) (int, bool) {
	return slices.BinarySearchFunc(w.movers, id, func(r moverRecord, id EntityID) int { return cmp.Compare(r.Entity, id) })
}

func (w *World) moverFor(id EntityID) *moverRecord {
	if i, ok := w.moverIndex(id); ok {
		return &w.movers[i]
	}
	return nil
}

// moverExpected reports whether e holds a record the builder gives it from the
// current state: every actor of a first-game World except one whose Mover an
// original-SAV import holds.
func (w *World) moverExpected(e Entity) bool {
	return !w.SecondGame() && w.motionFor(e.ID) == nil
}

// setMover installs r, replacing e's record, on a fresh slice.
func (w *World) setMover(r moverRecord) {
	i, ok := w.moverIndex(r.Entity)
	next := slices.Clip(w.movers)
	if ok {
		next = slices.Clone(next)
		next[i] = r
	} else {
		next = slices.Insert(next, i, r)
	}
	w.movers = next
}

// constructMover builds e's record as e is constructed in the World.
func (w *World) constructMover(e Entity) {
	if w.moverExpected(e) {
		w.setMover(w.builtMover(e, false))
	}
}

// constructMovers gives every actor of a first-game World that has no record
// the one the builder gives it from its current state. A record the World
// holds is kept.
func (w *World) constructMovers() {
	if w.SecondGame() {
		return
	}
	for _, e := range w.entities {
		if w.moverExpected(e) && w.moverFor(e.ID) == nil {
			w.setMover(w.builtMover(e, false))
		}
	}
}

// ConstructMissionMovers builds every actor's record at mission start, once
// the mission World holds its actors as the spawner and the join walk leave
// them: a map actor as the spawner places it, a carried member in joined as
// the join walk replaces its Mover (MOVE-122). It runs on an unadvanced World.
func (w *World) ConstructMissionMovers(joined []EntityID) {
	if w == nil || w.tick != 0 || w.SecondGame() {
		return
	}
	w.movers = nil
	for _, e := range w.entities {
		if w.moverExpected(e) {
			w.movers = append(w.movers, w.builtMover(e, slices.Contains(joined, e.ID)))
		}
	}
	slices.SortFunc(w.movers, func(a, b moverRecord) int { return cmp.Compare(a.Entity, b.Entity) })
}

// moverCrossed records that a crossing has stored the step cost of the cell
// it enters (MOVE-128). The tick owns w.movers (stepWorld detaches it), so the
// store is in place.
func (w *World) moverCrossed(id EntityID) {
	if r := w.moverFor(id); r != nil {
		r.Joined = false
	}
}

// moverCell is the cell whose cost an actor's step-cost word was taken from:
// the cell it stands in, or during a crossing the cell its position is in.
func moverCell(e Entity) cell {
	if e.Transit > 0 && e.Stride.Present && e.Transit <= e.TransitTotal {
		return strideCell(e.Stride, int32(e.TransitTotal-e.Transit))
	}
	return cell{e.X, e.Y}
}

// dropMovers forgets the records of actors the World no longer holds, and
// of actors whose Mover an original-SAV import holds.
func (w *World) dropMovers() {
	if w.movers == nil {
		return
	}
	next := make([]moverRecord, 0, len(w.movers))
	for _, r := range w.movers {
		if indexOfEntity(w.entities, r.Entity) >= 0 && w.motionFor(r.Entity) == nil {
			next = append(next, r)
		}
	}
	w.movers = next
}

// ApplyMover writes the record of actor id and the step-cost word into Mover
// bytes m and reports whether the World holds a record. The step-cost word is
// the occupy's and every crossing's store (MOVE-121, MOVE-128), taken from the
// cell the actor is in and its speed word now; a carried member that has made
// no crossing holds the join walk's 0. +0x08/+0x09 hold the constructor's 5
// and 0xff (MOVE-119, MOVE-122), 0 for the controlled player's actor now
// (DIV-1388).
func (w *World) ApplyMover(id EntityID, m *[180]byte) bool {
	r := w.moverFor(id)
	i := indexOfEntity(w.entities, id)
	if r == nil || i < 0 {
		return false
	}
	e := w.entities[i]
	m[5], m[8], m[9] = r.Mask, 5, 0xff
	if e.Owner == SelfSlot {
		m[8], m[9] = 0, 0
	}
	cost := uint16(0)
	if !r.Joined {
		cost = w.stepCostWord(e, moverCell(e))
	}
	binary.LittleEndian.PutUint16(m[0x72:], cost)
	return true
}

// adoptMovers takes, for every actor whose Mover a LOAD of an engine SAV
// imported into loaded, an import the policy has dropped, the record of the
// loaded bytes: the World then holds the Mover as the World that wrote the
// SAV held it. Every other actor keeps or receives its built record.
func (w *World) adoptMovers(loaded *savedActorMotionState) {
	if w.SecondGame() || loaded == nil {
		return
	}
	for _, m := range loaded.Motions {
		if i := indexOfEntity(w.entities, m.Entity); i >= 0 {
			w.adoptMover(w.entities[i], &m.Mover)
		}
	}
	w.constructMovers()
}

// adoptMover takes e's record from the Mover bytes m an engine SAV wrote for
// it. A step-cost word of 0 where the cell's cost gives another is the join
// walk's, kept until the actor's first crossing. A Mover with no block-test
// mask holds no record (ProjectActorMotion repairs such a mask); e keeps the
// record it holds or receives the built one.
func (w *World) adoptMover(e Entity, m *[180]byte) {
	if m[5] == 0 {
		if w.moverFor(e.ID) == nil {
			w.constructMover(e)
		}
		return
	}
	cost := binary.LittleEndian.Uint16(m[0x72:])
	joined := cost == 0 && w.stepCostWord(e, moverCell(e)) != 0
	w.setMover(heldMover(e, m, joined))
}
