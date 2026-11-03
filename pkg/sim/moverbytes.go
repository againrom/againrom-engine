package sim

import "encoding/binary"

// This file is the one producer of Mover record (U154) bytes. Every writer
// of a Mover, in pkg/sim and outside it, passes typed values to it; no other
// file indexes, puts or copies into a Mover (internal/archtest/moverbytes.go).

// MoverLen is the width of the Mover record.
const MoverLen = 180

// MoverHead is the head of the Mover: the facing +0x00, the desired facing
// +0x01, the block-test mask +0x05 (TERR-PASS-051), the hostile search radius
// +0x08 (AI-690), the byte +0x09 the constructor stores 0xff in (MOVE-119) and
// the turn rate +0x0a.
type MoverHead struct {
	Facing, Desired, Mask, Radius, SiegeRoute, Rate byte
}

// MoverHeadOf reads the head of the Mover in m.
func MoverHeadOf(m []byte) MoverHead {
	return MoverHead{Facing: m[0], Desired: m[1], Mask: m[5], Radius: m[8], SiegeRoute: m[9], Rate: m[10]}
}

func (h MoverHead) put(m []byte) {
	m[0], m[1], m[5], m[8], m[9], m[10] = h.Facing, h.Desired, h.Mask, h.Radius, h.SiegeRoute, h.Rate
}

// Record is a Mover holding h and zero after it.
func (h MoverHead) Record() [MoverLen]byte {
	var m [MoverLen]byte
	h.put(m[:])
	return m
}

// EditMoverHead applies edit to the head of the Mover in m.
func EditMoverHead(m []byte, edit func(*MoverHead)) {
	h := MoverHeadOf(m)
	edit(&h)
	h.put(m)
}

// MoverKeep names head fields CopyMover keeps from its destination.
type MoverKeep uint8

const (
	KeepFacing MoverKeep = 1 << iota
	KeepRate
)

// CopyMover writes the Mover m into the record bytes dst, keeping from dst
// the head fields keep names.
func CopyMover(dst []byte, m [MoverLen]byte, keep MoverKeep) {
	old := MoverHeadOf(dst)
	copy(dst, m[:])
	EditMoverHead(dst, func(h *MoverHead) {
		if keep&KeepFacing != 0 {
			h.Facing = old.Facing
		}
		if keep&KeepRate != 0 {
			h.Rate = old.Rate
		}
	})
}

// MoverFrom is the Mover a record's 180 bytes b hold.
func MoverFrom(b []byte) [MoverLen]byte {
	var m [MoverLen]byte
	copy(m[:], b)
	return m
}

// constructorHead is the head the actor base constructor stores (MOVE-119):
// the ground mask, the search radius (AI-691, AI-692), 0xff and the rate 0x10.
var constructorHead = MoverHead{Mask: 0x41, Radius: ConstructorSearchRadius, SiegeRoute: 0xff, Rate: 0x10}

// placedMover is the Mover the constructor and the placement leave: for a
// placed actor its facing, the derived rate and the mask of its domain
// (MOVE-120); a carried member placed by the join walk holds the
// constructor's head alone (MOVE-122).
func placedMover(e Entity, joined bool) [MoverLen]byte {
	h := constructorHead
	if !joined {
		h.Facing, h.Desired, h.Rate = e.Facing, e.Facing, MoverRate(e)
		h.Mask = MoverPassabilityMask(uint32(e.Domain) + 1)
	}
	return h.Record()
}

// constructedMover is the Mover an actor holds once the original has built
// it: placedMover, then for a placed actor the occupy's step cost and entry
// cache (MOVE-121).
func (w *World) constructedMover(e Entity, joined bool) [MoverLen]byte {
	m := placedMover(e, joined)
	if !joined {
		putStepCost(&m, w.stepCostWord(e, cell{e.X, e.Y}))
		putEntryCache(&m, uint16(byte(e.X))|uint16(byte(e.Y))<<8, 0x80, 0x80)
	}
	return m
}

// SiegeHireMover is the Mover the city writer gives a hired Catapult or
// Ballista: the mask of domain, the constructor's radius and +0x09, and rate.
func SiegeHireMover(domain uint32, rate byte) [MoverLen]byte {
	return siegeHireHead(MoverPassabilityMask(domain), rate).Record()
}

func siegeHireHead(mask, rate byte) MoverHead {
	return MoverHead{Mask: mask, Radius: constructorHead.Radius, SiegeRoute: constructorHead.SiegeRoute, Rate: rate}
}

// TownConstructorMovers are the Movers the town writer's two constructors
// write for an actor of mask and rate: a Human's, the mask and the rate
// alone, and a siege hire's.
func TownConstructorMovers(mask, rate byte) [][MoverLen]byte {
	return [][MoverLen]byte{MoverHead{Mask: mask, Rate: rate}.Record(), siegeHireHead(mask, rate).Record()}
}

// MoverRate is the rate byte +0x0a an actor holds now: its source's rate,
// else its own turn rate (SAV-1092).
func MoverRate(e Entity) uint8 {
	if e.ActorLoad.Source.Class != 0 {
		return e.SourceNow().MoverSpeed
	}
	return uint8(e.RotationSpeed)
}

// putStepCost stores the occupy's and each crossing's step-cost word +0x72
// (MOVE-121, MOVE-128).
func putStepCost(m *[MoverLen]byte, cost uint16) {
	binary.LittleEndian.PutUint16(m[0x72:], cost)
}

// putEntryCache stores the entry cell and fraction cache +0x82..+0x85
// (SAV-CELLFAIL-583).
func putEntryCache(m *[MoverLen]byte, packed uint16, fineX, fineY byte) {
	binary.LittleEndian.PutUint16(m[0x82:], packed)
	m[0x84], m[0x85] = fineX, fineY
}

// defaultEntryCache gives an idle native AI's zero entry cache the centre
// fraction and entry, each half only where it is zero (SAV-CELLFAIL-583).
func defaultEntryCache(m *[MoverLen]byte, entry uint16) {
	if m[0x84] == 0 && m[0x85] == 0 {
		m[0x84], m[0x85] = 0x80, 0x80
	}
	if m[0x82] == 0 && m[0x83] == 0 {
		binary.LittleEndian.PutUint16(m[0x82:], entry)
	}
}

// putLeaveCache stores a successful detach's old Position bytes 0/1/4/5 at
// +0x86..+0x89 (SAV-CELLLEAVE-584).
func putLeaveCache(m *[MoverLen]byte, p SavedActorPosition) {
	binary.LittleEndian.PutUint16(m[0x86:], p.Cell)
	m[0x88], m[0x89] = p.FineX, p.FineY
}

// putWalkClaim stores the walk claim (walkClaim) at +0x74, +0x76 and +0x90.
func putWalkClaim(m *[MoverLen]byte, c walkClaim) {
	binary.LittleEndian.PutUint16(m[0x74:], c.Target)
	binary.LittleEndian.PutUint16(m[0x76:], c.Claim)
	arrived := uint32(0)
	if c.Arrived {
		arrived = 1
	}
	binary.LittleEndian.PutUint32(m[0x90:], arrived)
}

func readWalkClaim(m *[MoverLen]byte) walkClaim {
	return walkClaim{
		Target:  binary.LittleEndian.Uint16(m[0x74:]),
		Claim:   binary.LittleEndian.Uint16(m[0x76:]),
		Arrived: binary.LittleEndian.Uint32(m[0x90:]) != 0,
	}
}

// putMoverTarget stores the accepted destination +0x70.
func putMoverTarget(m *[MoverLen]byte, packed uint16) {
	binary.LittleEndian.PutUint16(m[0x70:], packed)
}

// nameClaimedRouteCell names an outstanding claim +0x80 at +0x06
// (MOVE-CLAIM-007, SAV-1119).
func nameClaimedRouteCell(m *[MoverLen]byte) {
	if claim := binary.LittleEndian.Uint16(m[0x80:]); claim != 0 {
		binary.LittleEndian.PutUint16(m[0x06:], claim)
	}
}

// moverStride is a crossing's state: the claim +0x80, the origin +0xa6, the
// rate +0xa8, the interval +0xaa, the elapsed count +0xac, the direction
// +0xae and the per-tick steps +0xb0, +0xb1.
type moverStride struct {
	Claim, From, Rate, Total, Elapsed, Direction uint16
	StepX, StepY                                 byte
}

func putMoverStride(m *[MoverLen]byte, s moverStride) {
	binary.LittleEndian.PutUint16(m[0x80:], s.Claim)
	binary.LittleEndian.PutUint16(m[0xa6:], s.From)
	binary.LittleEndian.PutUint16(m[0xa8:], s.Rate)
	binary.LittleEndian.PutUint16(m[0xaa:], s.Total)
	binary.LittleEndian.PutUint16(m[0xac:], s.Elapsed)
	binary.LittleEndian.PutUint16(m[0xae:], s.Direction)
	m[0xb0], m[0xb1] = s.StepX, s.StepY
}

// putStrideOrigin stores the origin +0xa6 a boundary hands to the occupancy
// marker (MOVE-STEP-010).
func putStrideOrigin(m *[MoverLen]byte, packed uint16) {
	binary.LittleEndian.PutUint16(m[0xa6:], packed)
}

// putStrideElapsed stores the elapsed count +0xac.
func putStrideElapsed(m *[MoverLen]byte, elapsed uint16) {
	binary.LittleEndian.PutUint16(m[0xac:], elapsed)
}

// endStride clears the rate, interval, elapsed count, claim and origin a
// centre arrival leaves (SAV-CROSSNEXT-585).
func endStride(m *[MoverLen]byte) {
	for _, at := range []int{0xa8, 0xaa, 0xac, 0x80, 0xa6} {
		binary.LittleEndian.PutUint16(m[at:], 0)
	}
}

// moverTurn is the turn state: the active flag DWORD +0xa0, the counter
// +0x9d and the estimate +0xa4 (MOVE-TURN-044).
type moverTurn struct {
	Active            uint32
	Counter, Estimate byte
}

func moverTurnOf(m *[MoverLen]byte) moverTurn {
	return moverTurn{Active: binary.LittleEndian.Uint32(m[0xa0:]), Counter: m[0x9d], Estimate: m[0xa4]}
}

func putMoverTurn(m *[MoverLen]byte, t moverTurn) {
	binary.LittleEndian.PutUint32(m[0xa0:], t.Active)
	m[0x9d], m[0xa4] = t.Counter, t.Estimate
}

// clearMoverMotion zeroes the target, the stride and the turn a projection
// writes again from the actor.
func clearMoverMotion(m *[MoverLen]byte) {
	putMoverTarget(m, 0)
	putMoverStride(m, moverStride{})
	putMoverTurn(m, moverTurn{})
}

// clearRetreatFailure lowers the retreat failure byte +0x98.
func clearRetreatFailure(m *[MoverLen]byte) {
	m[0x98] = 0
}

// clearPursuitVictim zeroes the four bytes +0x7c..+0x7f. The pursuit search
// state keeps at +0x7c the victim its last search was for (PursuitSearch), and
// the stand-ground mirror writes mover +0x7c zero (syncSavedStandGround).
func clearPursuitVictim(m *[MoverLen]byte) {
	clear(m[0x7c:0x80])
}
