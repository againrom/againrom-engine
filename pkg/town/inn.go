package town

import (
	"fmt"
	"slices"
)

// OfferRow is one tavern or inn option: kind, topic and NPC record, and the
// predicates that admit it; an empty predicate admits. Kind 3 is a talk
// that makes mission Topic available.
type OfferRow struct {
	Kind  int `json:"kind"`
	Topic int `json:"topic"`
	NPC   int `json:"npc"`
	// Town is the town the row is offered in; 0 offers it in every town.
	Town int `json:"town,omitempty"`
	// Stages are the stages the row is offered at; empty offers it at
	// every stage.
	Stages []int `json:"stages,omitempty"`
	// After, when positive, offers the row only at a stage above it.
	After int `json:"after,omitempty"`
	// Zero, Nonzero and One are bank slots that must hold zero, a value
	// other than zero, and one; Any are slots of which at least one must
	// hold a value other than zero.
	Zero    []int `json:"zero,omitempty"`
	Nonzero []int `json:"nonzero,omitempty"`
	One     []int `json:"one,omitempty"`
	Any     []int `json:"any,omitempty"`
}

// OfferTable names a catalog's row shape by its mod table: a tavern row is
// one chapter of a catalog town, its mission the topic; an inn row takes
// the whole shape.
type OfferTable string

const (
	TavernTable OfferTable = "tavern"
	InnTable    OfferTable = "inn"
)

const (
	OfferTalk    = 0
	OfferMission = 3
)

// OfferKey names a row: town, chapter and NPC in a tavern, topic and NPC in
// an inn.
type OfferKey struct {
	Table                     OfferTable
	Town, Chapter, Topic, NPC int
}

func (t OfferTable) Key(r OfferRow) OfferKey {
	if t == TavernTable {
		return OfferKey{Table: t, Town: r.Town, Chapter: r.Chapter(), NPC: r.NPC}
	}
	return OfferKey{Table: t, Topic: r.Topic, NPC: r.NPC}
}

func (k OfferKey) String() string {
	if k.Table == TavernTable {
		return fmt.Sprintf("town %d chapter %d npc %d", k.Town, k.Chapter, k.NPC)
	}
	return fmt.Sprintf("topic %d npc %d", k.Topic, k.NPC)
}

// Chapter is the row's one stage, else 0.
func (r OfferRow) Chapter() int {
	if len(r.Stages) != 1 {
		return 0
	}
	return r.Stages[0]
}

func MissionKind(mission int) int {
	if mission != 0 {
		return OfferMission
	}
	return OfferTalk
}

// ChapterOffer is the tavern row of npc in chapter of town id.
func ChapterOffer(id, chapter, npc, mission int) OfferRow {
	return OfferRow{Kind: MissionKind(mission), Topic: mission, NPC: npc, Town: id, Stages: []int{chapter}}
}

func (r OfferRow) Equal(o OfferRow) bool {
	return r.Kind == o.Kind && r.Topic == o.Topic && r.NPC == o.NPC && r.Town == o.Town && r.After == o.After &&
		slices.Equal(r.Stages, o.Stages) && slices.Equal(r.Zero, o.Zero) && slices.Equal(r.Nonzero, o.Nonzero) &&
		slices.Equal(r.One, o.One) && slices.Equal(r.Any, o.Any)
}

func (r OfferRow) Gates() [][]int { return [][]int{r.Zero, r.Nonzero, r.One, r.Any} }

// OfferError is a refusal of row Row.
type OfferError struct {
	Row OfferKey
	Err error
}

func (e *OfferError) Error() string { return e.Err.Error() }
func (e *OfferError) Unwrap() error { return e.Err }

func OfferErrorf(key OfferKey, format string, args ...any) error {
	return &OfferError{Row: key, Err: fmt.Errorf(format, args...)}
}

// validateOffers checks each row's bounds and key against its table; a
// chapter names an NPC record once. A refusal is an OfferError.
func (c *Catalog) validateOffers(towns map[int]bool) error {
	t := c.OfferTable
	switch {
	case t == "" && c.Offers == nil:
		return nil
	case t != TavernTable && t != InnTable:
		return fmt.Errorf("town catalog %s: offer table %q is neither %q nor %q", c.Catalog, t, TavernTable, InnTable)
	}
	tavern := t == TavernTable
	minNPC, topic := 1, "topic"
	if tavern {
		minNPC, topic = 0, "mission"
	}
	seen := map[OfferKey]bool{}
	speaks := map[[2]int]OfferKey{}
	for _, r := range c.Offers {
		k := t.Key(r)
		bad := func(format string, args ...any) error {
			return OfferErrorf(k, "town catalog %s: %s row %v: "+format, append([]any{c.Catalog, t, k}, args...)...)
		}
		switch {
		case (tavern || r.Town != 0) && !towns[r.Town]:
			return bad("town %d is not a town of the catalog", r.Town)
		case tavern && len(r.Stages) != 1:
			return bad("%d chapters are stated; a tavern row stands in one", len(r.Stages))
		case tavern && r.Chapter() <= 0:
			return bad("chapter %d is not positive", r.Chapter())
		case r.NPC < minNPC || r.NPC > maxTavernValue:
			return bad("npc %d is outside the range %d to %d", r.NPC, minNPC, maxTavernValue)
		case r.Topic < 0 || r.Topic > maxTavernValue:
			return bad("%s %d is outside the range 0 to %d", topic, r.Topic, maxTavernValue)
		case r.Kind != OfferTalk && r.Kind != OfferMission:
			return bad("kind %d is neither a talk (%d) nor a mission talk (%d)", r.Kind, OfferTalk, OfferMission)
		case tavern && r.Kind != MissionKind(r.Topic):
			return bad("kind %d does not follow mission %d; a tavern row is a mission talk exactly when it gives a mission", r.Kind, r.Topic)
		case tavern && (r.After != 0 || slices.ContainsFunc(r.Gates(), func(l []int) bool { return len(l) > 0 })):
			return bad("a tavern row states no stage floor or bank gate")
		case r.After < 0:
			return bad("after %d is negative", r.After)
		case !tavern && seen[k]:
			return bad("the row is named twice")
		}
		for _, s := range r.Stages {
			if s <= 0 {
				return bad("stage %d is not positive", s)
			}
		}
		for _, slots := range r.Gates() {
			for _, s := range slots {
				if s < 0 {
					return bad("bank slot %d is negative", s)
				}
			}
		}
		if tavern {
			at := [2]int{r.Chapter(), r.NPC}
			if prior, ok := speaks[at]; ok {
				return bad("npc %d already speaks in chapter %d, in the row %v; a chapter names an NPC record once", r.NPC, r.Chapter(), prior)
			}
			speaks[at] = k
		}
		seen[k] = true
	}
	return nil
}

// MercenaryRule is a tavern's mercenary stock walk and squad price rule as
// data. The roster walks the stock units of types 1 to StockTypes; with no
// unit ids from a LOAD, type t's unit id is StockIDPhase + t - 1. A squad's
// unit price factor at mission m is PriceLadder[(m-PriceFrom)/PriceStep],
// 0 below PriceFrom and past the ladder; its level is the Level of the last
// of Levels whose From m reaches, 1 below them all.
type MercenaryRule struct {
	StockTypes   int              `json:"stock-types"`
	StockIDPhase int              `json:"stock-id-phase"`
	PriceFrom    int              `json:"price-from"`
	PriceStep    int              `json:"price-step"`
	PriceLadder  []int            `json:"price-ladder"`
	Levels       []MercenaryLevel `json:"levels"`
}

// MercenaryLevel is the squad level from mission From.
type MercenaryLevel struct {
	From  int `json:"from"`
	Level int `json:"level"`
}

// validate checks the rule's own bounds: positive stock types, price step
// and levels, no negative phase, start or factor, and levels in rising
// order of From. A nil rule states none.
func (r *MercenaryRule) validate(catalog string) error {
	if r == nil {
		return nil
	}
	bad := func(format string, args ...any) error {
		return fmt.Errorf("town catalog %s: mercenaries: "+format, append([]any{catalog}, args...)...)
	}
	switch {
	case r.StockTypes <= 0:
		return bad("stock-types %d is not positive", r.StockTypes)
	case r.StockIDPhase < 0 || r.PriceFrom < 0:
		return bad("stock-id-phase %d or price-from %d is negative", r.StockIDPhase, r.PriceFrom)
	case r.PriceStep <= 0:
		return bad("price-step %d is not positive", r.PriceStep)
	}
	for _, f := range r.PriceLadder {
		if f < 0 {
			return bad("price factor %d is negative", f)
		}
	}
	for i, l := range r.Levels {
		if l.Level <= 0 || i > 0 && l.From <= r.Levels[i-1].From {
			return bad("level %+v is not positive or does not rise above the one before", l)
		}
	}
	return nil
}

// UnitPrice is the squad unit price factor at mission.
func (r *MercenaryRule) UnitPrice(mission int) int {
	if r == nil || mission < r.PriceFrom {
		return 0
	}
	if i := (mission - r.PriceFrom) / r.PriceStep; i < len(r.PriceLadder) {
		return r.PriceLadder[i]
	}
	return 0
}

// Level is the squad level at mission.
func (r *MercenaryRule) Level(mission int) int {
	level := 1
	if r != nil {
		for _, l := range r.Levels {
			if mission >= l.From {
				level = l.Level
			}
		}
	}
	return level
}
