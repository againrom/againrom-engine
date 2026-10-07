package sav

import "fmt"

// ActorHoldings is an exact all-Player projection, not Party's character view.
// The two held references and twelve sparse armor references stay separate.
// LoadState carries the actor's own container tail (ITEM-SAVE-014); item
// runtime blocks beyond Piece's own fields are not represented (DIV-746).
type ActorHoldings struct {
	Off                    int
	MapUnitID, Cell, HP    uint16
	Stage                  uint8
	HeldWeapon, HeldShield *Piece
	Worn                   [12]*Piece
	Items                  []Piece // A zero Piece is one explicit null inventory slot.
	LoadState              ActorLoadState
	Basis                  *ActorBasis
}

// ActorLoadState is the four stored actor words and inline container header.
// It is not a derive result. Zero load and absent container are independent.
//
// OwnWeight and Load are StatOwnWeight/StatLoad (+0x8e/+0x90): SAV-792 finds
// +0x8e incrementally maintained by real weight changes, with +0x90 a second,
// far-better-connected writer of ITEM-LOAD-005's own derive; SAV-794 shows the
// load import path reads +0x90 back and keeps it rather than recomputing it
// from the record's own container, so a caller must round-trip Load exactly
// as read, including a value the container alone could not reproduce.
type ActorLoadState struct {
	Present                          bool
	OwnWeight, Load, Capacity, Speed int16
	ContainerPresent                 bool
	InsertIndex                      uint32
	Accumulator                      int32
	HealthHundredths, ManaHundredths uint8
}

func actorLoadState(r *Record) ActorLoadState {
	return ActorLoadState{
		Present:   true,
		OwnWeight: int16(r.value("U8E")), Load: int16(r.value("U90")),
		Capacity: int16(r.value("Capacity")), Speed: int16(r.value("Speed")),
		ContainerPresent: r.value("HasInventory") != 0,
		InsertIndex:      r.value("Inventory1C"), Accumulator: int32(r.value("Inventory20")),
		HealthHundredths: uint8(r.value("UA2")), ManaHundredths: uint8(r.value("UA3")),
	}
}

// ActorHoldings follows every Player/group under one archive state. Repeated
// actor references name one owner, including across groups/Players. Distinct
// held/armor/inventory roles preserve every occurrence. Native graph bindings
// retain shared Item identity without selecting a primary ownership field.
func (f *File) ActorHoldings(current ...uint16) ([]ActorHoldings, error) {
	doc, _, err := f.exactDocument()
	if err != nil {
		return nil, err
	}
	return actorHoldings(doc, current)
}

func actorHoldings(doc *document, current []uint16) ([]ActorHoldings, error) {
	graph, err := currentActorGraph(doc.players, current, doc.dead)
	if err != nil {
		return nil, err
	}
	basis := make(map[int]*ActorBasis, len(graph.Actors))
	for _, actor := range graph.Actors {
		basis[actor.Off] = actor.Character.Basis
	}
	seen := make(map[*Record]bool)
	var out []ActorHoldings
	for _, actor := range currentActorRecords(doc.players, current, doc.dead) {
		if actor == nil || !groundClass(actor.Class, "Unit") {
			return nil, fmt.Errorf("sav: holdings actor is not a Unit")
		}
		if seen[actor] {
			continue
		}
		seen[actor] = true
		position := actor.Raw["Block12"]
		if len(position) != 12 {
			return nil, fmt.Errorf("sav: holdings actor at %d has no Position", actor.Off)
		}
		a := ActorHoldings{Off: actor.Off, MapUnitID: uint16(actor.value("T08")),
			Cell: u16(position, 0), HP: uint16(actor.value("Health")), Stage: uint8(actor.value("Stage")), LoadState: actorLoadState(actor)}
		var err error
		a.Basis = basis[actor.Off]
		if a.Basis == nil {
			a.Basis, err = actorBasisFields(actor)
		}
		if err != nil {
			return nil, err
		}
		project := func(item *Record) (*Piece, error) {
			if item == nil {
				return nil, nil
			}
			if !groundClass(item.Class, "Item") {
				return nil, fmt.Errorf("sav: actor %d holds %s, not Item", actor.Off, item.Class)
			}
			if len(item.Refs["Effects"]) != item.Counts["Effects"] {
				return nil, fmt.Errorf("sav: actor %d item has null Effect", actor.Off)
			}
			// A non-"Effect" class reaches piece() itself now, the same
			// per-item, never-whole-refusing bucket a nonzero Token state
			// already used (party.go's own class/state dispatch).
			p := piece(item)
			return &p, nil
		}
		for slot, site := range []string{"HeldWeapon", "HeldShield"} {
			refs := actor.Refs[site]
			if len(refs) > 1 {
				return nil, fmt.Errorf("sav: actor %d repeats %s", actor.Off, site)
			}
			if len(refs) == 0 {
				continue
			}
			p, err := project(refs[0])
			if err != nil {
				return nil, err
			}
			if slot == 0 {
				a.HeldWeapon = p
			} else {
				a.HeldShield = p
			}
		}
		if actor.Class != "Unit" && len(actor.WornSlots) != 12 {
			return nil, fmt.Errorf("sav: actor %d lacks sparse armor slots", actor.Off)
		}
		for slot, item := range actor.WornSlots {
			if slot >= len(a.Worn) {
				return nil, fmt.Errorf("sav: actor %d has excess armor slots", actor.Off)
			}
			p, err := project(item)
			if err != nil {
				return nil, err
			}
			a.Worn[slot] = p
		}
		inventory, present := actor.RefSlots["Inventory"]
		if !present {
			inventory = actor.Refs["Inventory"]
		}
		if len(inventory) != actor.Counts["Inventory"] {
			return nil, fmt.Errorf("sav: actor %d has incomplete inventory positions", actor.Off)
		}
		for _, item := range inventory {
			if item == nil {
				a.Items = append(a.Items, Piece{})
				continue
			}
			p, err := project(item)
			if err != nil {
				return nil, err
			}
			a.Items = append(a.Items, *p)
		}
		out = append(out, a)
	}
	return out, nil
}
