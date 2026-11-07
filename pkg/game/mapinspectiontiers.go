package game

import (
	"fmt"
	"image"
	"math"
	"sync"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
	"againrom/pkg/words"
)

// unitTier is one Units row a creature's class offers: the secondary key
// (`+0x0a`, the row's face column) that selects it, the row, and the row's
// name in the install's own bytes.
type unitTier struct {
	Face  uint16
	Index int
	Name  string
}

// tiersOf lists, in key order, each secondary key 0..255 that resolves u's
// class through the Units arm to a row no lower key already reached
// (ALM-CLS-052, UNIT-PROTO-008). It is empty for a human placement, one that
// another arm takes, and a missing table.
func (x *MapInspector) tiersOf(u alm.Unit) []unitTier {
	if x.table == nil {
		return nil
	}
	// Only the keys resolution reads remain in the cache key.
	key := alm.Unit{ClassID: u.ClassID, Flags: u.Flags, DefID: u.DefID, ServerID: u.ServerID}
	if c := x.tiers; c != nil {
		c.mu.Lock()
		defer c.mu.Unlock()
		if tiers, ok := c.byKey[key]; ok {
			return tiers
		}
	}
	var out []unitTier
	seen := map[int]bool{}
	for k := 0; k <= math.MaxUint8; k++ {
		probe := u
		probe.ClassSubID = uint16(k)
		r := mapload.Resolve(probe, x.table)
		if r.Arm != mapload.ArmUnits || !r.Found() || seen[r.Index] {
			continue
		}
		seen[r.Index] = true
		out = append(out, unitTier{Face: uint16(k), Index: r.Index, Name: x.table.Units.EntryName(r.Index)})
	}
	if c := x.tiers; c != nil {
		if c.byKey == nil {
			c.byKey = map[alm.Unit][]unitTier{}
		}
		c.byKey[key] = out
	}
	return out
}

// tierCache holds tiersOf's answers by the keys resolution reads. The table
// it was filled from is the inspector's, which never changes.
type tierCache struct {
	mu    sync.Mutex
	byKey map[alm.Unit][]unitTier
}

// word is the install language's editor word id as UTF-8 text.
func (x *MapInspector) word(id string) string {
	lang := ""
	if x.archives != nil {
		lang = x.archives.Base.Profile.Language
	}
	return words.For(lang).Text(id)
}

// AddUnitIdentity adds a unit at cell who is the identity choice value: a
// new record with that identity's class and secondary keys, keyed to it by
// the identity field's writer, as one edit, owned by the map's second roster
// slot when it has one and the first otherwise (DIV-3209).
func (s *mapInspectionEdits) AddUnitIdentity(cell image.Point, value int64) (*ui.InspectionDocument, error) {
	return s.change(func(m *mapedit.Editor) (bool, error) {
		if err := s.admitCell(m, cell, "unit cell", ui.KindUnit, admitAdd); err != nil {
			return false, err
		}
		id, ok := s.identityOf(value)
		if !ok {
			return false, fmt.Errorf("choose who the new unit is: a creature, a person or a named character")
		}
		keys, ok := mapload.Place(alm.Unit{}, id, s.inspector.table, s.mission())
		if !ok {
			return false, fmt.Errorf("the installed tables cannot key a unit to that choice")
		}
		i, err := placeNewUnit(m, cell, keys.ClassID, keys.ClassSubID)
		if err != nil {
			return false, err
		}
		if _, err := s.keyIdentity(m, i, id); err != nil {
			return false, err
		}
		return true, s.checkPlacement(admissionOf(ui.KindUnit, admitAdd), mapedit.KindUnit, m, i, image.Rectangle{})
	})
}

// placeNewUnit appends a new unit record with the class and secondary keys
// at cell, owned by the map's second player (its first when it has one).
func placeNewUnit(m *mapedit.Editor, cell image.Point, class int16, secondary uint16) (int, error) {
	view, err := m.Map()
	if err != nil {
		return -1, err
	}
	owner := uint32(min(2, max(1, len(view.Groups))))
	x, y := cellWords(cell, newUnitFraction, newUnitFraction)
	rec, err := m.NewUnitRecord(x, y, class, secondary, owner)
	if err != nil {
		return -1, err
	}
	return m.PlaceUnit(rec)
}
