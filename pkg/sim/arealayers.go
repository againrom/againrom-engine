package sim

import (
	"encoding/binary"
	"fmt"
	"slices"
	"sort"
)

// MAGIC-MAPLAYER-040 and CLOUDOWNER-155: six current pointers, not six
// objects and not an owner stack. Native objects have no invented SAV pointer.
var areaLayerSpells = [...]uint16{3, 7, 8, 19, 12, 17}

func areaLayerConflict(incoming, old uint16) bool {
	if !layerSpell(old) {
		return false
	}
	if incoming == old {
		return true
	}
	switch incoming {
	case 2, 3:
		return old == 7 || old == 8
	case 7:
		return old == 3
	case 12, 17:
		return old == 12 || old == 17
	}
	return false
}

func (w *World) nativeAreaLayerPresent(key, spell uint16) bool {
	for _, e := range w.effects {
		if e.Mode == areaModeCloud && e.Spell == spell && containsKey(e.Cells, key) {
			return true
		}
	}
	return false
}

func (w *World) areaLayerPresent(key, spell uint16) bool {
	return w.nativeAreaLayerPresent(key, spell) || w.retainedAreaLayerPresent(key, spell)
}

func (w *World) retainedAreaLayerPresent(key, spell uint16) bool {
	for layer, id := range areaLayerSpells {
		if spell != id {
			continue
		}
		at := sort.Search(len(w.savedCellRecords), func(i int) bool { return w.savedCellRecords[i].Cell >= key })
		return at < len(w.savedCellRecords) && w.savedCellRecords[at].Cell == key && w.savedCellRecords[at].SpellEffects[layer] != 0
	}
	return false
}

func (w *World) nativeAreaCellPresent(key uint16) bool {
	for _, spell := range areaLayerSpells {
		if w.nativeAreaLayerPresent(key, spell) {
			return true
		}
	}
	return false
}

// Each independent pulse scans the full square in x/y order, using any current
// same-spell pointer as its gate and its own payload (MAGIC-CLOUDVISIT-156).
func (w *World) cloudPulseCells(key, spell uint16, radius int32) []uint16 {
	x, y := keyCell(key)
	var out []uint16
	for cx := max(0, x-radius); cx <= min(255, w.bounds.Width-1, x+radius); cx++ {
		for cy := max(0, y-radius); cy <= min(255, w.bounds.Height-1, y+radius); cy++ {
			key := cellKey(cx, cy)
			if w.areaLayerPresent(key, spell) {
				out = append(out, key)
			}
		}
	}
	return out
}

// The read-only admission gate resolves every current-cell dependency before
// a cast can clear an old layer or spend its caster. A wall's two-cell paint
// extent is independent of Radius; include both bounded extents here.
func (w *World) areaPaintIssue(rule SpellRule, x, y int32) string {
	if !layerSpell(rule.ID) && rule.ID != 2 {
		return ""
	}
	if err := w.savedWorldEffectsFault(); err != nil {
		return err.Error()
	}
	radius := int32(rule.Radius)
	if rule.Distribution == distributionWall {
		radius = max(radius, 2)
	}
	for cx := max(0, x-radius); cx <= min(255, w.bounds.Width-1, x+radius); cx++ {
		for cy := max(0, y-radius); cy <= min(255, w.bounds.Height-1, y+radius); cy++ {
			key := cellKey(cx, cy)
			c := w.motionCell(key)
			at := sort.Search(len(w.savedCellRecords), func(i int) bool { return w.savedCellRecords[i].Cell >= key })
			if at < len(w.savedCellRecords) && w.savedCellRecords[at].Cell == key {
				r := w.savedCellRecords[at]
				if r.SpellEffects != ([6]uint32{}) {
					if c == nil {
						return "area layer lacks its current Cell payload"
					}
					if c.Payload[2] != r.LayerCount {
						return "area current layer count owners differ"
					}
					for layer, identity := range r.SpellEffects {
						if binary.LittleEndian.Uint32(c.Payload[20+4*layer:]) != identity {
							return "area current layer payload owners differ"
						}
					}
				}
			}
			if c != nil {
				if issue := w.savedCellRecomputeIssue(*c); issue != "" {
					return issue
				}
			} else if w.savedCellPlanes != nil && w.savedCellPlanes.CostKnown[key] == 0 {
				return "new area cell construction lacks known current cost"
			}
			if w.savedCellPlanes == nil {
				for _, spell := range areaLayerSpells {
					if w.areaLayerPresent(key, spell) && !w.nativeAreaLayerPresent(key, spell) {
						return "area layer lacks saved current cell authority"
					}
				}
			}
		}
	}
	return ""
}

// Clear only the addressed current pointer. Independent driver identity and
// countdown survive an overwrite. Caller preflights all owners before writes.
func (w *World) clearSavedAreaLayer(key uint16, layer int) {
	defer w.holdLayerCosts()()
	at := sort.Search(len(w.savedCellRecords), func(i int) bool { return w.savedCellRecords[i].Cell >= key })
	if at == len(w.savedCellRecords) || w.savedCellRecords[at].Cell != key || w.savedCellRecords[at].SpellEffects[layer] == 0 {
		return
	}
	r := &w.savedCellRecords[at]
	identity := r.SpellEffects[layer]
	r.SpellEffects[layer], r.LayerCount = 0, 0
	for _, p := range r.SpellEffects {
		if p != 0 {
			r.LayerCount++
		}
	}
	c := w.motionCell(key)
	binary.LittleEndian.PutUint32(c.Payload[20+4*layer:], 0)
	c.Payload[2] = r.LayerCount
	if s := w.savedWorldEffects; s != nil {
		for i := range s.Areas {
			d := &s.Areas[i]
			if int(d.Layer) == layer && d.Identity == identity {
				d.Cells = slices.DeleteFunc(d.Cells, func(k uint16) bool { return k == key })
			}
		}
	}
	w.recomputeSavedCell(key)
}

// MAGIC-CLOUDEND-163: after recomputation an empty node restores cost and the
// static baseline, preserving bit4. Dynamic retains the recomputation result.
// The actor-detach helper has a different Dynamic tail; reuse only its unlink.
func (w *World) deleteEmptyAreaCell(key uint16) bool {
	defer w.holdLayerCosts()()
	if w.nativeAreaCellPresent(key) || w.savedCellPlanes == nil {
		return false
	}
	dynamic := w.savedCellPlanes.Dynamic[key]
	if !w.deleteEmptySavedCell(key) {
		return false
	}
	w.savedCellPlanes.Dynamic[key] = dynamic
	w.savedCellRecords = slices.DeleteFunc(w.savedCellRecords, func(c SavedCellRecord) bool { return c.Cell == key })
	return true
}

// Native effects have not been loaded through an original archive graph.
func (w *World) HasNativeAreaEffects() bool { return len(w.effects) != 0 }

// Native form92 records have one owner per spell/cell. Reject hostile duplicate
// coverage rather than silently canonicalizing a current SAVE or LOAD.
func (w *World) areaOwnershipFault() error {
	owners := map[[2]uint16]bool{}
	for _, e := range w.effects {
		if e.Mode > areaModeCloud || e.Direction > 7 {
			return fmt.Errorf("sim: invalid area mode/direction")
		}
		if e.Mode != areaModeCloud {
			continue
		}
		for _, key := range e.Cells {
			pair := [2]uint16{e.Spell, key}
			if owners[pair] || w.retainedAreaLayerPresent(key, e.Spell) {
				return fmt.Errorf("sim: duplicate current area layer owner at %04x spell %d", key, e.Spell)
			}
			owners[pair] = true
		}
	}
	return nil
}
