package sim

import "encoding/binary"

type attackNotice struct {
	Cell  uint16
	Scans uint8
}

// Original orders already own these fields. Fresh worlds keep the same
// small state separately until an explicit saved order is constructed.
func (w *World) attackNoticeAt(i int) attackNotice {
	if o := w.savedOrder(w.entities[i].ID); o != nil {
		return attackNotice{binary.LittleEndian.Uint16(o.Raw[0x58:]), o.Raw[0x5a]}
	}
	return w.entities[i].attackNotice
}

func (w *World) setAttackNotice(i int, n attackNotice) {
	if o := w.savedOrder(w.entities[i].ID); o != nil {
		binary.LittleEndian.PutUint16(o.Raw[0x58:], n.Cell)
		o.Raw[0x5a] = n.Scans
		w.entities[i].attackNotice = attackNotice{}
		return
	}
	w.entities[i].attackNotice = n
}

func (w *World) rememberAttacker(ai, ti int) {
	a := w.entities[ai]
	w.setAttackNotice(ti, attackNotice{Cell: uint16(a.Y)<<8 | uint16(uint8(a.X))})
	if o := w.savedOrder(w.entities[ti].ID); o != nil {
		o.Raw[0x54] = 1 // AI-RETAL-056's separate idle-turn alarm.
	}
}

// AI-GROUPSEE-068 counts candidate builds, rather than elapsed world ticks.
// Actor perception queries read the same cell without consuming the count.
// The player's fog and issued orders do not change when somebody hits them.
func (w *World) stampAttackNotices(stamp []byte, members []int, consume bool) {
	for _, i := range members {
		e := w.entities[i]
		if e.Owner == 0 || e.Owner == SelfSlot || !e.Alive() || e.OffMap {
			continue
		}
		n := w.attackNoticeAt(i)
		if n.Cell == 0 {
			continue
		}
		x, y := int32(n.Cell&255), int32(n.Cell>>8)
		if x < w.bounds.Width && y < w.bounds.Height {
			stamp[y*w.bounds.Width+x]++
		}
		if consume {
			n.Scans++
			if int8(n.Scans) > 20 {
				n.Cell = 0
			}
			w.setAttackNotice(i, n)
		}
	}
}
