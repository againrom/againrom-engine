// Package shopitem is the one shop item of both games: a shelf entry or a
// deal place. Each game's rules over it stay with that game.
package shopitem

import (
	"againrom/pkg/data"
	"againrom/pkg/sim"
)

// Table is the second game's item table; zero in the first game.
type Table uint8

// Owner is the units of a place from one member's pack slot.
type Owner struct {
	PartyID   string
	Member    int
	PackIndex int
	Count     int32
}

// Item is one unit of an item as the bag holds it, its unit price and how
// many units it holds. The first game adds its city identity, the second
// its table identity; no document writes either.
type Item struct {
	CityID      sim.SavedObjectID `json:"-"`
	CityOrigins []string          `json:"-"`
	Owners      []Owner           `json:"-"`

	ObjectID        sim.SavedObjectID
	Code            data.ItemCode
	Kind            uint8
	Effects         []sim.ItemEffect
	Price           int32
	Count           int32
	Weight          int16
	WeightPresent   bool
	SourceEquipment sim.SourceEquipment

	Table    Table `json:"-"`
	Row      int   `json:"-"`
	Class    int   `json:"-"`
	Material int   `json:"-"`
	// Spell is a book's spell; an enchanted item's spell is an Effect.
	Spell int `json:"-"`
}

func (i Item) Instance() sim.ItemInstance {
	return sim.ItemInstance{ObjectID: i.ObjectID, Code: uint16(i.Code), Kind: i.Kind,
		Effects: append([]sim.ItemEffect(nil), i.Effects...), Price: i.Price, Weight: i.Weight, WeightPresent: i.WeightPresent, SourceEquipment: i.SourceEquipment}
}

func (i Item) Clone() Item {
	i.Effects = append([]sim.ItemEffect(nil), i.Effects...)
	i.CityOrigins = append([]string(nil), i.CityOrigins...)
	i.Owners = append([]Owner(nil), i.Owners...)
	return i
}
