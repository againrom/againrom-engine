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

// unitMultiTier reports that unit index of the model is a creature whose
// class has more than one tier: the only placement the tier field is offered
// for.
func (s *mapInspectionEdits) unitMultiTier(index int) bool {
	m, err := s.model.Map()
	return err == nil && index >= 0 && index < len(m.AuthoredUnits) && len(s.inspector.tiersOf(m.AuthoredUnits[index])) > 1
}

// tierChoices are the tier field's choices, "<key>: <row name>" as every
// catalogue choice is written.
func tierChoices(tiers []unitTier) []ui.InspectionChoice {
	out := make([]ui.InspectionChoice, len(tiers))
	for i, t := range tiers {
		out[i] = ui.InspectionChoice{Value: int64(t.Face), Label: fmt.Sprintf("%d: %s", t.Face, t.Name)}
	}
	return out
}

// tierError refuses a secondary key that selects no Units row of the unit's
// class, naming the keys that do.
func (x *MapInspector) tierError(u alm.Unit) error {
	tiers := x.tiersOf(u)
	for _, t := range tiers {
		if t.Face == u.ClassSubID {
			return nil
		}
	}
	var keys []uint16
	for _, t := range tiers {
		keys = append(keys, t.Face)
	}
	if len(keys) == 0 {
		return fmt.Errorf("class %d has no Units rows, so it has no tier", u.ClassID)
	}
	return fmt.Errorf("tier %d names no Units row of class %d; its tiers are %v", u.ClassSubID, u.ClassID, keys)
}

// word is the install language's editor word id as UTF-8 text.
func (x *MapInspector) word(id string) string {
	lang := ""
	if x.archives != nil {
		lang = x.archives.Base.Profile.Language
	}
	return words.For(lang).Text(id)
}

// UnitTiers are the tiers a new unit of class may take, with the default the
// add takes when none is chosen; none for a class with one tier or none.
func (s *mapInspectionEdits) UnitTiers(class int64) ([]ui.InspectionChoice, int64) {
	if class < math.MinInt16 || class > math.MaxInt16 {
		return nil, 0
	}
	tiers := s.inspector.tiersOf(alm.Unit{ClassID: int16(class)})
	if len(tiers) < 2 {
		return nil, 0
	}
	face, err := s.inspector.secondaryKey(int16(class))
	if err != nil {
		return nil, 0
	}
	return tierChoices(tiers), int64(face)
}

// AddUnitTier is AddUnit with the secondary key tier: 0 takes the key of the
// class's first installed definition row (DIV-3209), any other value must
// select a Units row of the class.
func (s *mapInspectionEdits) AddUnitTier(cell image.Point, class, tier int64) (*ui.InspectionDocument, error) {
	return s.change(func(m *mapedit.Editor) (bool, error) {
		if err := insideMap(m, cell, "unit cell"); err != nil {
			return false, err
		}
		if class < math.MinInt16 || class > math.MaxInt16 {
			return false, fmt.Errorf("class key %d is outside %d..%d", class, math.MinInt16, math.MaxInt16)
		}
		if tier < 0 || tier > math.MaxUint16 {
			return false, fmt.Errorf("tier %d is outside 0..%d", tier, math.MaxUint16)
		}
		secondary := uint16(tier)
		if tier == 0 {
			key, err := s.inspector.secondaryKey(int16(class))
			if err != nil {
				return false, err
			}
			secondary = key
		} else if err := s.inspector.tierError(alm.Unit{ClassID: int16(class), ClassSubID: secondary}); err != nil {
			return false, err
		}
		view, err := m.Map()
		if err != nil {
			return false, err
		}
		owner := uint32(min(2, max(1, len(view.Groups))))
		x, y := cellWords(cell, newUnitFraction, newUnitFraction)
		rec, err := m.NewUnitRecord(x, y, int16(class), secondary, owner)
		if err != nil {
			return false, err
		}
		i, err := m.PlaceUnit(rec)
		if err != nil {
			return false, err
		}
		return true, s.checkPlacement(mapedit.KindUnit, m, i, image.Rectangle{})
	})
}
