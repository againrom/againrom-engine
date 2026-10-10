package sim

import (
	"encoding/binary"
	"fmt"
)

// Native X/Y names the accepted destination. SAV Position names the current
// near cell and fractional displacement (SAV-TOKENPOS-074). NativeStride is
// the retained request-time rate; recomputing a rate here would change a step
// after damage, equipment or terrain updates.
// fresh marks a World without a saved motions plane. Existing mover bytes
// remain authoritative when the plane is present or live carriers sync.
func ProjectActorMotion(e Entity, old SavedActorMotion, route [][2]int32, fresh bool) (SavedActorMotion, error) {
	m := old
	pack := func(x, y int32) (uint16, error) {
		if x < 0 || x > 255 || y < 0 || y > 255 {
			return 0, fmt.Errorf("actor %d position exceeds SAV cell", e.ID)
		}
		return uint16(x) | uint16(y)<<8, nil
	}
	here, err := pack(e.X, e.Y)
	if err != nil {
		return m, err
	}
	m.Position.Cell, m.Position.PackedCell = here, here
	m.Position.FineX, m.Position.FineY = 128, 128
	// SAV-1092: a mover whose RotationSpeed byte is 0 divides by zero on its
	// first turn. An actor with no bound source basis (a generated mission's
	// NPCs never populate ActorLoad.Source on the live World) has nothing for
	// SourceNow to overlay, so its own live turn rate is the value of record.
	moverSpeed := e.SourceNow().MoverSpeed
	if e.ActorLoad.Source.Class == 0 {
		moverSpeed = uint8(e.RotationSpeed)
	}
	m.Mover[0], m.Mover[1], m.Mover[10] = e.Facing, e.DesiredFacing, moverSpeed
	// TERR-PASS-051: the original writer sets the mover's block-test mask
	// from the same domain U4A already carries (Domain+1); a zero mask (this
	// array's own zero value) blocks nothing, not even the border. Every
	// actor needs a real mask to move at all, so this one repair applies
	// regardless of owner.
	if fresh && m.Mover[5] == 0 {
		m.Mover[5] = MoverPassabilityMask(uint32(e.Domain) + 1)
	}
	// SAV-1096 and DIV-1388 keep AI defaults separate from player bytes.
	// SAV-CELLFAIL-583 names the entry-position cache. An idle native AI's
	// occupied cell owns it; the guard post remains an independent order field.
	// Keep the existing constructor for moving, turning or terminal actors.
	if fresh && e.Owner != SelfSlot {
		if m.Mover[8] == 0 && m.Mover[9] == 0 {
			m.Mover[8], m.Mover[9] = 5, 255
		}
		if m.Mover[0x84] == 0 && m.Mover[0x85] == 0 {
			m.Mover[0x84], m.Mover[0x85] = 0x80, 0x80
		}
		if m.Mover[0x82] == 0 && m.Mover[0x83] == 0 {
			entry := uint16(e.PostX) | uint16(e.PostY)<<8
			if e.Alive() && !e.OffMap && e.Transit == 0 && !e.HasTarget && !e.HasAttackTarget && !e.Turning() {
				entry = here
			}
			binary.LittleEndian.PutUint16(m.Mover[0x82:], entry)
		}
	}
	for _, at := range []int{0x70, 0x80, 0xa6, 0xa8, 0xaa, 0xac, 0xae} {
		binary.LittleEndian.PutUint16(m.Mover[at:], 0)
	}
	m.Mover[0xb0], m.Mover[0xb1], m.Mover[0x9d], m.Mover[0xa4] = 0, 0, 0, 0
	binary.LittleEndian.PutUint32(m.Mover[0xa0:], 0)
	m.ActorAction = 0
	if terminalRegistryActor(&e) {
		m.ActorAction = 16
	}
	m.StaticRoute, m.DynamicRoute = nil, nil
	if e.HasTarget {
		target, err := pack(e.TargetX, e.TargetY)
		if err != nil {
			return m, err
		}
		binary.LittleEndian.PutUint16(m.Mover[0x70:], target)
	}
	if e.Transit > 0 {
		stride := e.Stride
		if e.Transit > e.TransitTotal || e.TransitTotal == 0 {
			return m, fmt.Errorf("actor %d transit has no retained accepted stride", e.ID)
		}
		elapsed := e.TransitTotal - e.Transit
		if stride.Present {
			x := stride.FromX*256 + 128 + int32(stride.StepX)*int32(elapsed)
			y := stride.FromY*256 + 128 + int32(stride.StepY)*int32(elapsed)
			near, err := pack(x>>8, y>>8)
			if err != nil {
				return m, err
			}
			m.Position.Cell, m.Position.PackedCell, m.Position.FineX, m.Position.FineY = near, near, byte(x), byte(y)
			binary.LittleEndian.PutUint16(m.Mover[0x80:], here)
			binary.LittleEndian.PutUint16(m.Mover[0xa6:], uint16(stride.FromX)|uint16(stride.FromY)<<8)
			binary.LittleEndian.PutUint16(m.Mover[0xa8:], uint16(stride.Rate))
			binary.LittleEndian.PutUint16(m.Mover[0xaa:], e.TransitTotal)
			binary.LittleEndian.PutUint16(m.Mover[0xac:], elapsed)
			binary.LittleEndian.PutUint16(m.Mover[0xae:], uint16(stride.Direction))
			m.Mover[0xb0], m.Mover[0xb1] = byte(stride.StepX), byte(stride.StepY)
		} else {
			// Historical unrated transit has no accepted fractional stride.
			// Preserve its current destination and interval, with zero unknown
			// direction/rate operands. The supplement retains that distinction.
			binary.LittleEndian.PutUint16(m.Mover[0x80:], here)
			binary.LittleEndian.PutUint16(m.Mover[0xaa:], e.TransitTotal)
			binary.LittleEndian.PutUint16(m.Mover[0xac:], elapsed)
		}
		m.ActorAction = 1
		// Dynamic list includes the center still owed by this crossing.
		m.DynamicRoute = append(m.DynamicRoute, here)
	}
	if e.Turning() {
		binary.LittleEndian.PutUint32(m.Mover[0xa0:], 1)
		m.Mover[0x9d], m.Mover[0xa4] = e.TurnTotal-e.TurnRemaining, e.TurnRemaining
		m.ActorAction = 1
	}
	if e.TurnState.Present {
		flag := uint32(0)
		if e.TurnState.Active {
			flag = 1
		}
		binary.LittleEndian.PutUint32(m.Mover[0xa0:], flag)
		m.Mover[0x9d] = e.TurnState.Counter
	}
	for _, cell := range route {
		key, err := pack(cell[0], cell[1])
		if err != nil {
			return m, err
		}
		if len(m.DynamicRoute) == 0 || m.DynamicRoute[len(m.DynamicRoute)-1] != key {
			m.DynamicRoute = append(m.DynamicRoute, key)
		}
	}
	nameClaimedRouteCell(&m.Mover)
	m.Current, m.Active, m.Issue = true, e.Transit > 0, ""
	return m, nil
}

// SavedMover is the mover block a SAV record carries for m. MOVE-CLAIM-007:
// the route step reads the next route cell into +06 and stores that cell as
// the claim +0x80, so while a claim is outstanding +06 names it. SAV-1119: a
// non-centred actor's drawable is created at +06, so a zero there puts it at
// cell 0,0. With no claim, +06 keeps the bytes m already holds.
func (m SavedActorMotion) SavedMover() [180]byte {
	nameClaimedRouteCell(&m.Mover)
	return m.Mover
}

func nameClaimedRouteCell(mover *[180]byte) {
	if claim := binary.LittleEndian.Uint16(mover[0x80:]); claim != 0 {
		binary.LittleEndian.PutUint16(mover[0x06:], claim)
	}
}

// MoverPassabilityMask is the original writer's own map from the domain byte
// (U4A, 1/2/3) to the mover's block-test mask (TERR-PASS-051): 1 -> 0x41, 2 ->
// 0x44, 3 -> 0x82. The base constructor leaves 0x41, so ground
// is the default for any other value this package ever produces.
func MoverPassabilityMask(domain uint32) byte {
	switch domain {
	case 2:
		return 0x44
	case 3:
		return 0x82
	default:
		return 0x41
	}
}
