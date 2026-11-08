package sim

import (
	"encoding/binary"
	"fmt"
)

// BuildingCaster is one caster a map binds to a building through a type-9
// record of mode mask 4 (R2-ENGINE-1006, R2-ENGINE-1009). Owner is the roster
// slot of the building; Cell is the building's location cell word, which is
// what removal matches (R2-ENGINE-1010). X and Y are the record's own cell.
type BuildingCaster struct {
	X, Y         uint8
	Spell, Power uint8
	Reach        uint8
	Owner        uint32
	Cell         uint16
}

// StructureCellWord is a structure's location cell word.
func StructureCellWord(s Structure) uint16 { return cellKey(s.Col, s.Row) }

// DeclareBuildingCasters installs the map's caster list. A world receives its
// game with its script, so a list declared before that waits outside the byte
// form; a first-game World drops it and its digest does not move.
func (w *World) DeclareBuildingCasters(list []BuildingCaster) {
	if w.game.buildingCasters {
		w.buildingCasters = append([]BuildingCaster(nil), list...)
		return
	}
	w.authoredBuildingCasters = append([]BuildingCaster(nil), list...)
}

// settleBuildingCasters applies the profile the script just gave the world.
func (w *World) settleBuildingCasters() {
	if !w.game.buildingCasters {
		w.buildingCasters, w.authoredBuildingCasters = nil, nil
		return
	}
	if w.authoredBuildingCasters != nil {
		w.buildingCasters, w.authoredBuildingCasters = w.authoredBuildingCasters, nil
	}
}

// BuildingCasters returns the world's caster list as a fresh slice.
func (w *World) BuildingCasters() []BuildingCaster {
	return append([]BuildingCaster(nil), w.buildingCasters...)
}

// BuildsCasters reports whether this world's game has building casters.
func (w *World) BuildsCasters() bool { return w.game.buildingCasters }

// buildingCasterExcludedEffect is the effect whose bit 0x1000 of a unit's
// active-effect word keeps it from a caster's survivors (R2-ENGINE-1008,
// R2-ENGINE-054). Attached effects here are keyed by spell id (DIV-4718).
const buildingCasterExcludedEffect uint16 = 12

// buildingCasterPass runs once per turn after the group tick, every caster
// once, in list order (R2-ENGINE-1007, R2-ENGINE-778). The consumer holds no
// timer, cooldown or cost (R2-ENGINE-1008).
func (w *World) buildingCasterPass() {
	for _, c := range w.buildingCasters {
		w.buildingCasterAct(c)
	}
}

// buildingCasterAct picks one unit for one caster and queues its cast. The
// candidates are the units around the record's cell within the fill byte, read
// as a reach in cells; an attack spell keeps units hostile to the owner and a
// defensive spell the others (DIV-4718). One draw chooses among the
// survivors; none means no cast. A spell index with no table row casts
// nothing.
func (w *World) buildingCasterAct(c BuildingCaster) {
	rule, ok := w.findSpell(uint32(c.Spell))
	if !ok {
		return
	}
	var picks []int
	centre := cell{int32(c.X), int32(c.Y)}
	for i := range w.entities {
		e := &w.entities[i]
		if !e.Alive() || e.OffMap || !e.OrdinaryTargetable() || centre.chebyshevTo(cellOf(e)) > int64(c.Reach) {
			continue
		}
		if w.relations.Hostile(c.Owner, e.Owner) == rule.Defensive || w.hasAttachedSpell(e.ID, buildingCasterExcludedEffect) {
			continue
		}
		picks = append(picks, i)
	}
	if len(picks) == 0 {
		return
	}
	target := w.entities[picks[w.rng.aiRange(int32(len(picks)))]].ID
	// The same temporary caster instant 24 builds: no actor casts, so no mana
	// is paid and the cast request's script row is the one it meets.
	w.casts = append(w.casts, scriptCast{FromX: c.X, FromY: c.Y, Spell: c.Spell, Power: uint16(c.Power), Target: target, AtUnit: true})
}

// removeBuildingCastersOn removes every caster whose location cell word is the
// structure's, on a hit whose computed value is not below the structure's
// health (R2-ENGINE-1010). It runs before the subtraction. The list is rebuilt
// rather than filtered in place, so a World copy sharing it is untouched.
func (w *World) removeBuildingCastersOn(s *Structure, value int32) {
	if len(w.buildingCasters) == 0 || s.MaxHealth == 0 || uint16(value) < s.Field42 {
		return
	}
	word := StructureCellWord(*s)
	kept := make([]BuildingCaster, 0, len(w.buildingCasters))
	for _, c := range w.buildingCasters {
		if c.Cell != word {
			kept = append(kept, c)
		}
	}
	w.buildingCasters = kept
}

const (
	buildingCasterFormVersion byte = 135
	buildingCasterRecordLen        = 9
)

func (w *World) appendBuildingCasters(b []byte) []byte {
	if len(w.buildingCasters) == 0 {
		return b
	}
	ws := beginWrapper(b)
	b = binary.LittleEndian.AppendUint32(b, uint32(len(w.buildingCasters)))
	for _, c := range w.buildingCasters {
		b = append(b, c.X, c.Y, c.Spell, c.Power, c.Reach)
		b = binary.LittleEndian.AppendUint16(b, uint16(c.Owner))
		b = binary.LittleEndian.AppendUint16(b, c.Cell)
	}
	return endWrapper(b, ws, buildingCasterFormVersion)
}

func (w *World) unmarshalBuildingCasters(data []byte, proof formProof) error {
	fail := func() error { return fmt.Errorf("malformed building caster section") }
	env, ok := readWrapper(data, buildingCasterFormVersion)
	payload := env.section()
	if !ok || len(payload) < 4 {
		return fail()
	}
	count := uint64(binary.LittleEndian.Uint32(payload))
	if count == 0 || uint64(len(payload)) != 4+buildingCasterRecordLen*count {
		return fail()
	}
	next := *w
	if err := next.unmarshalForm(env.base(), proof); err != nil {
		return err
	}
	next.buildingCasters = make([]BuildingCaster, count)
	for i := range next.buildingCasters {
		r := payload[4+buildingCasterRecordLen*i:]
		next.buildingCasters[i] = BuildingCaster{X: r[0], Y: r[1], Spell: r[2], Power: r[3], Reach: r[4],
			Owner: uint32(binary.LittleEndian.Uint16(r[5:])), Cell: binary.LittleEndian.Uint16(r[7:])}
	}
	if !next.marshalsTo(data, proof) {
		return fail()
	}
	*w = next
	return nil
}
