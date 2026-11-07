package sim

import (
	"encoding/binary"
	"fmt"
)

// DeathItemRoll is a Units row's death item columns 41..44 (R2-ASSET-224):
// the chance, the price bounds and the kind mask of the draw.
type DeathItemRoll struct {
	Chance, Min, Max int32
	Mask             uint32
}

// DeathItemDraw answers the list a death's item draw generates for roll,
// drawing on rnd, one rand() core value per call, in list order. An entry
// the bag cannot hold has code 0 and keeps its place. The game layer
// supplies it as install-derived input (SetDeathItemDraw); the second
// game's is its shop draw (R2-ENGINE-941).
type DeathItemDraw func(roll DeathItemRoll, rnd func() int) []ItemInstance

// SetDeathItemDraw binds the death item draw. Like the rules it is
// install-derived input, carried across a decode and never encoded.
func (w *World) SetDeathItemDraw(d DeathItemDraw) { w.deathItems = d }

// DeathItemDraw is the bound death item draw, nil when none is bound.
func (w *World) DeathItemDraw() DeathItemDraw { return w.deathItems }

// rollsAtDeath is the death roll's gate (HERO-KILL-027, R2-ENGINE-940): no
// reward mark under the profile, and a type id above 0x40, or 0x40 and above
// under the second game's profile.
func (w *World) rollsAtDeath(e Entity) bool {
	floor := int32(0x41)
	if w.game.deathRollAt64 {
		floor = 0x40
	}
	return !w.paysNoReward(e) && e.TypeID >= floor
}

// rollsDeathItem reports whether e's death can draw an item.
func (w *World) rollsDeathItem(e Entity) bool {
	return w.game.deathItemRoll && e.DeathItem.Chance > 0
}

// deathRoll is the Units-row death roll (HERO-KILL-027, R2-ENGINE-940). The
// gold draw is rand(100) against the chance, then the minimum plus a draw
// over the span. The second game's profile then draws rand(100) against the
// item chance; a pass runs the draw and takes entry rand(n-1) of its list
// as one unit. rng.uniform is U[0,n], the original's inclusive rand(n), and
// the uint32 keeps the purse and sack arithmetic's wrapping width.
func (w *World) deathRoll(i int) (gold uint32, item ItemInstance, rolled bool) {
	e := w.entities[i]
	if !w.rollsAtDeath(e) {
		return 0, ItemInstance{}, false
	}
	if e.GoldChance > w.rng.uniform(100) {
		gold = uint32(e.TreasureMin + w.rng.uniform(e.TreasureMax))
	}
	if !w.game.deathItemRoll || e.DeathItem.Chance <= w.rng.uniform(100) || w.deathItems == nil {
		return gold, ItemInstance{}, false
	}
	list := w.deathItems(e.DeathItem, func() int { return int(w.rng.raw()) })
	if len(list) == 0 {
		return gold, ItemInstance{}, false
	}
	item = list[w.rng.uniform(int32(len(list)-1))]
	return gold, item.Clone(), !item.Empty()
}

// Form134 wraps only Worlds with an actor holding a death item roll. Older
// forms and the first game's Worlds hold none and keep their bytes.
const deathItemFormVersion byte = 134

// deathItemRecordLen is one record: the entity index and the four columns.
const deathItemRecordLen = 20

func (w *World) appendDeathItems(b []byte) []byte {
	ws := beginWrapper(b)
	le := binary.LittleEndian
	for i, e := range w.entities {
		if r := e.DeathItem; r != (DeathItemRoll{}) {
			b = le.AppendUint32(b, uint32(i))
			b = le.AppendUint32(b, uint32(r.Chance))
			b = le.AppendUint32(b, uint32(r.Min))
			b = le.AppendUint32(b, uint32(r.Max))
			b = le.AppendUint32(b, r.Mask)
		}
	}
	if len(b) == ws.at {
		return b
	}
	return endWrapper(b, ws, deathItemFormVersion)
}

func (w *World) unmarshalDeathItems(data []byte, proof formProof) error {
	fail := func() error { return fmt.Errorf("malformed death item section") }
	env, ok := readWrapper(data, deathItemFormVersion)
	if !ok {
		return fail()
	}
	s := env.section()
	if len(s) == 0 || len(s)%deathItemRecordLen != 0 {
		return fail()
	}
	next := *w
	if err := next.unmarshalForm(env.base(), proof); err != nil {
		return err
	}
	entities := append([]Entity(nil), next.entities...)
	le := binary.LittleEndian
	prior := int64(-1)
	for at := 0; at < len(s); at += deathItemRecordLen {
		index := le.Uint32(s[at:])
		if int64(index) <= prior || uint64(index) >= uint64(len(entities)) {
			return fail()
		}
		prior = int64(index)
		entities[index].DeathItem = DeathItemRoll{Chance: int32(le.Uint32(s[at+4:])), Min: int32(le.Uint32(s[at+8:])),
			Max: int32(le.Uint32(s[at+12:])), Mask: le.Uint32(s[at+16:])}
	}
	next.entities = entities
	if !next.marshalsTo(data, proof) {
		return fail()
	}
	*w = next
	return nil
}
