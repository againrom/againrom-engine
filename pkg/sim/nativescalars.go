package sim

import "fmt"

const (
	ScalarT0C = iota
	ScalarT08High
	ScalarT18
	ScalarT1C
	ScalarReference
	ScalarU4B
	ScalarU4C
	ScalarU6C
	ScalarU8E
	ScalarU60
	ScalarU61
	ScalarUA0
	ScalarUA4
	ScalarU130
	ScalarU136
	ScalarU138
	ScalarU148
	ScalarU144
	ScalarU50
	ScalarU54
	ScalarU58
	ScalarCount
)

const nativeScalarKnown = uint32(1<<ScalarCount - 1)
const nativeBlockKnown = uint16(1<<10 - 1)

func (b NativeActorBasis) hasScalarValues() bool {
	return b.ScalarsPresent || b.BlockPresent
}

func (b NativeActorBasis) ScalarIsKnown(index int) bool {
	return index >= 0 && index < ScalarCount && b.ScalarsPresent && b.ScalarKnown&(uint32(1)<<index) != 0
}

func (b NativeActorBasis) BlockByteKnown(index int) bool {
	return index >= 0 && index < len(b.Block) && b.BlockPresent && b.BlockKnown&(uint16(1)<<index) != 0
}

// NativeBasisNow overlays maintained fields without replacing independent bits
// or changing the availability of an observed native scalar.
func NativeBasisNow(e Entity) NativeActorBasis {
	b := e.NativeBasis
	if !e.Humanoid && e.TypeID >= 0x1a && b.ScalarIsKnown(ScalarT1C) {
		b.Scalars[ScalarT1C] = uint32(e.XPValue)
	}
	if e.ActionClock.Known && b.ScalarIsKnown(ScalarU138) {
		b.Scalars[ScalarU138] = e.ActionClock.End
	}
	if b.ScalarIsKnown(ScalarUA4) {
		b.Scalars[ScalarUA4] = b.Scalars[ScalarUA4]&255 | uint32(e.ScanRange)<<8
	}
	if b.ScalarIsKnown(ScalarU4C) {
		flags := b.Scalars[ScalarU4C] &^ 8
		if e.OffMap {
			flags |= 8
		}
		if e.NativeClass.Present {
			flags &^= 4
			if !e.NativeClass.Fighter {
				flags |= 4
			}
		}
		b.Scalars[ScalarU4C] = flags
	}
	if e.Stride.Present {
		x, y := e.X*256+128, e.Y*256+128
		if e.Transit > 0 {
			elapsed := int32(e.TransitTotal - e.Transit)
			x = e.Stride.FromX*256 + 128 + int32(e.Stride.StepX)*elapsed
			y = e.Stride.FromY*256 + 128 + int32(e.Stride.StepY)*elapsed
		}
		for index, value := range [4]byte{byte(x >> 8), byte(y >> 8), byte(x), byte(y)} {
			if b.BlockByteKnown(index) {
				b.Block[index] = value
			}
		}
	}
	if terminalRegistryActor(e) {
		b = nativeTerminalScalarBasis(b)
	}
	return b
}

func nativeTerminalScalarBasis(b NativeActorBasis) NativeActorBasis {
	for _, slot := range []int{ScalarU50, ScalarU54} {
		if b.ScalarIsKnown(slot) {
			b.Scalars[slot] = 16
		}
	}
	return b
}

func (w *World) nativeBasisNow(e Entity) NativeActorBasis {
	b := NativeBasisNow(e)
	if b.ScalarIsKnown(ScalarU50) {
		if order := w.savedOrder(e.ID); order != nil {
			o := *order
			if !e.Alive() && LivingOnlyActorState(o.State) {
				o.State = uint32(e.ActorState)
			}
			b.Scalars[ScalarU50] = ProjectActorRetreatOrder(e, o).State
		} else if w.savedGroups == nil {
			if order, patrol := PatrolOrder(e, [144]byte{}); patrol {
				b.Scalars[ScalarU50] = order.State
			}
		}
		if !e.Alive() && LivingOnlyActorState(b.Scalars[ScalarU50]) {
			b.Scalars[ScalarU50] = uint32(e.ActorState)
		}
	}
	if b.ScalarIsKnown(ScalarU54) {
		motion := w.motionFor(e.ID)
		if motion != nil || w.savedMotion == nil {
			m := SavedActorMotion{Entity: e.ID}
			if motion != nil {
				m = *motion
			}
			if !m.Current && m.Issue == "" {
				if current, err := ProjectActorMotion(e, m, w.Route(e.ID), w.savedMotion == nil); err == nil {
					m = current
				}
			}
			b.Scalars[ScalarU54] = m.ActorAction
		}
		if e.HasAttackTarget && e.Alive() && e.Transit == 0 && !e.Turning() && !w.motionActive(e.ID) && !w.nativeLaterActorAction(e) {
			action := uint32(3)
			order := w.savedOrder(e.ID)
			if e.AttackPhase == AttackBoundaryTwo || e.AttackPhase == AttackBoundaryOne && motion != nil && motion.Current && motion.ActorAction == 0 && order != nil && order.Raw[9] == 0 {
				action = 0
			}
			b.Scalars[ScalarU54] = action
		}
	}
	if terminalRegistryActor(e) {
		b = nativeTerminalScalarBasis(b)
	}
	return b
}

func (w *World) nativeLaterActorAction(e Entity) bool {
	if e.Retreat.Known && e.Retreat.Progress != 2 || e.PendingOrder.RowAdmitted {
		return true
	}
	if i, ok := w.bookCastIndex(e.ID); ok && w.bookCasts[i].Phase != bookApproach {
		return true
	}
	if _, ok := w.scrollIndex(e.ID); ok && e.PendingOrder.Kind != PendingScroll {
		return true
	}
	if e.AttackPhase == AttackCasting {
		if i := indexOfEntity(w.entities, e.ID); i >= 0 && w.equipment[i][0].SourceEquipment.Spell.Present {
			return true
		}
	}
	return false
}

func (w *World) nativeRemovedBasisNow(row NativeActorBasisRecord) NativeActorBasis {
	for _, terminal := range w.currentTerminalActors {
		if terminal.ID == row.ID && terminal.HP <= -10 {
			return nativeTerminalScalarBasis(row.Basis)
		}
	}
	for _, dead := range w.originalDead {
		if dead.ID == row.ID && dead.terminal.HP <= -10 && dead.terminal.Stage >= uint8(DecayBones) {
			return nativeTerminalScalarBasis(row.Basis)
		}
	}
	return row.Basis
}

func (b NativeActorBasis) validateScalars() error {
	if b.ScalarKnown & ^nativeScalarKnown != 0 || !b.ScalarsPresent && (b.ScalarKnown != 0 || b.Scalars != ([ScalarCount]uint32{})) {
		return fmt.Errorf("sim: invalid native scalar availability")
	}
	if b.BlockKnown & ^nativeBlockKnown != 0 || !b.BlockPresent && (b.BlockKnown != 0 || b.Block != ([10]byte{})) {
		return fmt.Errorf("sim: invalid native Block availability")
	}
	for slot, value := range b.Scalars {
		if !b.ScalarIsKnown(slot) {
			continue
		}
		switch slot {
		case ScalarT0C, ScalarU4B, ScalarU4C, ScalarU6C, ScalarU60, ScalarU61, ScalarU136:
			if value > 255 {
				return fmt.Errorf("sim: native scalar %d exceeds byte width", slot)
			}
		case ScalarT08High, ScalarT18, ScalarU8E, ScalarUA0, ScalarUA4:
			if value > 65535 {
				return fmt.Errorf("sim: native scalar %d exceeds word width", slot)
			}
		}
	}
	return nil
}
