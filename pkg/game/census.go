package game

import (
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
)

// UnitCounts is one map's census: every entity of FromALM's world classified
// by the bundle's three answers. The three counts PARTITION the entities —
// each entity lands in exactly one — so their sum is the world's entity
// count, which FromALM makes the map's placed-unit count.
type UnitCounts struct {
	Sprites int // an entry with a drawn frame: the sprite the map screen draws
	NoClass int // no entry at all — an id naming no class, at either sign
	NoFrame int // a frameless entry — a resolved class without drawable art
}

// UnitCensus builds FromALM's world for m and counts, per entity, which of
// the bundle's three answers its class id gets.
//
// The world is FromALM's OWN — the same transform the map screen runs,
// sign extension included — so what is counted is exactly what a front-end
// over this map would resolve, entity for entity. The classification is the
// seam's lookup split three ways where entityDraws collapses it to two: at
// the draw seam an id naming no class and a class without art paint the same
// square, while AC-11 must record them apart — they are different facts
// about an install. An entry that is present but nil counts NoFrame: it is
// an entry that cannot draw, the answer UnitPlace gives it. "Without art" is
// len(Frames) == 0: the loader leaves an excluded class's Frames nil, and a
// class with any frame at all resolves.
//
// A nil or empty set answers "no entry" for every id, so a census with no
// bundle is all NoClass — total, never an error. Nothing here mutates m or
// set: the world is read through the copy-handing Entities, and the set is
// only looked up in.
func UnitCensus(m *alm.Map, set *terrain.UnitSet) UnitCounts {
	var classes map[int32]*terrain.UnitClass
	if set != nil {
		classes = set.Classes
	}
	var n UnitCounts
	for _, e := range mapload.FromALM(m).Entities() {
		switch c, ok := classes[e.Class]; {
		case !ok:
			n.NoClass++
		case c == nil || len(c.Frames) == 0:
			n.NoFrame++
		default:
			n.Sprites++
		}
	}
	return n
}
