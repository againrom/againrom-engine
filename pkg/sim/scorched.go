package sim

import (
	"encoding/binary"
	"fmt"
	"slices"
)

const scorchedSpanLen = 4

// ScorchedCells is the persistent fire footprint, in packed-cell order.
// Rendering resolves installed scenery and water. These records do not alter
// collision, damage, or the RNG stream.
func (w *World) ScorchedCells() []uint16 { return slices.Clone(w.scorchedCells) }

func (w *World) scorchCells(spell uint16, cells []uint16) {
	if spell != 2 && spell != 3 {
		return
	}
	for _, key := range cells {
		x, y := keyCell(key)
		if x < 0 || y < 0 || x >= w.bounds.Width || y >= w.bounds.Height {
			continue
		}
		at, found := slices.BinarySearch(w.scorchedCells, key)
		if !found {
			w.scorchedCells = slices.Insert(w.scorchedCells, at, key)
		}
	}
}

func (w *World) appendScorched(b []byte) []byte {
	for _, key := range w.scorchedCells {
		b = binary.LittleEndian.AppendUint16(b, key)
	}
	return binary.LittleEndian.AppendUint32(b, uint32(2*len(w.scorchedCells)))
}

func splitScorched(data []byte) ([]byte, []uint16, error) {
	if len(data) < headerLen+scorchedSpanLen {
		return nil, nil, fmt.Errorf("sim: truncated scorched-cell footer")
	}
	end := len(data) - scorchedSpanLen
	span := uint64(binary.LittleEndian.Uint32(data[end:]))
	if span%2 != 0 || span > 2*65536 || span > uint64(end-headerLen) {
		return nil, nil, fmt.Errorf("sim: invalid scorched-cell span")
	}
	start := end - int(span)
	var cells []uint16
	for at := start; at < end; at += 2 {
		key := binary.LittleEndian.Uint16(data[at:])
		if len(cells) > 0 && key <= cells[len(cells)-1] {
			return nil, nil, fmt.Errorf("sim: scorched cells are not strictly ordered")
		}
		cells = append(cells, key)
	}
	return data[:start], cells, nil
}
