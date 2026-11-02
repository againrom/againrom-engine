package game

import (
	"fmt"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// Each slot's home in an actor record's equipment fields is sav's one slot
// table; a slot width other than the simulation's does not compile. Writers
// and readers that disagree at the edges pass policy as data.
var _ [sim.EquipSlots]sav.EquipmentPlace = sav.EquipmentPlaces()

// savWornRule decides whether a site writes or reads a record's Worn field.
type savWornRule uint8

const (
	savWornByClass savWornRule = iota // the record's class
	savWornByField                    // the record's present Worn field
)

// The writers' and the binding check's Worn rules, one per site.
var (
	cityProjectionWorn   = savWornByField
	terminalBodyWorn     = savWornByClass
	currentItemsWorn     = savWornByClass
	savedItemsWorn       = savWornByClass
	itemBindingCheckWorn = savWornByClass
)

func (rule savWornRule) carries(r *sav.DocumentRecordData) bool {
	if rule == savWornByClass {
		return sav.CarriesWorn(r.Class)
	}
	_, ok := savedObjectRefs(r, sav.WornField)
	return ok
}

// writeSavEquipment sets the equipment fields from one ref per slot, in
// slot order, through the site's own setter.
func writeSavEquipment(refs [sim.EquipSlots]uint16, carriesWorn bool, set func(field string, refs []uint16)) {
	var worn [sim.EquipSlots]uint16
	for slot, place := range sav.EquipmentPlaces() {
		if place.Field == sav.WornField {
			worn[place.Ordinal] = refs[slot]
			continue
		}
		set(place.Field, []uint16{refs[slot]})
	}
	if carriesWorn {
		set(sav.WornField, worn[:])
	}
}

// savHeldShape is the held-field shape a reader takes a ref from.
type savHeldShape uint8

const (
	savHeldFirst  savHeldShape = iota // the first ref of a nonempty field
	savHeldSingle                     // the ref of a field with exactly one ref
	savHeldExact                      // as savHeldSingle; any other shape is a fault
)

// savWornLow is what a reader does with Worn positions 0 and 1.
type savWornLow uint8

const (
	savLowIgnored    savWornLow = iota
	savLowFillsEmpty            // fills a held slot without a nonzero ref
	savLowFillsZero             // fills a held slot whose field is one zero ref
	savLowExclusive             // fills a zero held slot; beside a held ref, a fault
	savLowRefused               // a nonzero position is a fault
)

// savWornHigh is what a reader does with Worn positions past the last slot.
type savWornHigh uint8

const (
	savHighIgnored     savWornHigh = iota
	savHighRefused                 // a nonzero position is a fault
	savWornExactLength             // a present Worn field has exactly one position per slot
)

type savEquipmentPolicy struct {
	Worn savWornRule
	Held savHeldShape
	Low  savWornLow
	High savWornHigh
}

// The readers' policies, one per site.
var (
	cityTopologyEquipment = savEquipmentPolicy{Worn: savWornByField, Held: savHeldExact, Low: savLowExclusive, High: savWornExactLength}
	wornMaskEquipment     = savEquipmentPolicy{Worn: savWornByField, Held: savHeldFirst, Low: savLowFillsEmpty, High: savHighIgnored}
	passiveBodyEquipment  = savEquipmentPolicy{Worn: savWornByClass, Held: savHeldSingle, Low: savLowIgnored, High: savHighIgnored}
	savedItemEquipment    = savEquipmentPolicy{Worn: savWornByField, Held: savHeldExact, Low: savLowRefused, High: savHighRefused}
	cityPartyEquipment    = savEquipmentPolicy{Worn: savWornByField, Held: savHeldFirst, Low: savLowFillsZero, High: savHighIgnored}
)

type savEquipmentFault uint8

const (
	savWornShapeFault savEquipmentFault = iota + 1
	savHeldShapeFault
	savOverlapFault
)

type savEquipmentError struct {
	Fault savEquipmentFault
	Field string
	Err   error
}

func (e *savEquipmentError) Error() string {
	return fmt.Sprintf("SAV equipment %s fault %d: %v", e.Field, e.Fault, e.Err)
}

// savEquipmentView is one ref per slot; Located marks a slot whose field has
// a position for it.
type savEquipmentView struct {
	Refs    [sim.EquipSlots]uint16
	Located [sim.EquipSlots]bool
}

type savRefsGetter func(field string) ([]uint16, bool, error)

func savedObjectRefsGetter(r *sav.DocumentRecordData) savRefsGetter {
	return func(field string) ([]uint16, bool, error) {
		refs, ok := savedObjectRefs(r, field)
		return refs, ok, nil
	}
}

// readSavEquipment reads a record's slots under one policy: the Worn field's
// shape first, then each held field, then the worn positions.
func readSavEquipment(r *sav.DocumentRecordData, p savEquipmentPolicy, get savRefsGetter) (savEquipmentView, error) {
	var v savEquipmentView
	var worn []uint16
	if p.Worn == savWornByField || sav.CarriesWorn(r.Class) {
		refs, ok, err := get(sav.WornField)
		if err != nil || ok && p.High == savWornExactLength && len(refs) != sim.EquipSlots {
			return v, &savEquipmentError{Fault: savWornShapeFault, Field: sav.WornField, Err: err}
		}
		worn = refs
	}
	for slot, place := range sav.EquipmentPlaces() {
		if place.Field == sav.WornField {
			continue
		}
		refs, ok, err := get(place.Field)
		switch {
		case p.Held == savHeldFirst && place.Ordinal < len(refs):
			v.Refs[slot], v.Located[slot] = refs[place.Ordinal], true
		case p.Held != savHeldFirst && err == nil && ok && len(refs) == 1:
			v.Refs[slot], v.Located[slot] = refs[0], true
		case p.Held == savHeldExact:
			return v, &savEquipmentError{Fault: savHeldShapeFault, Field: place.Field, Err: err}
		}
		var at uint16
		if slot < len(worn) {
			at = worn[slot]
		}
		switch p.Low {
		case savLowFillsEmpty:
			if v.Refs[slot] == 0 && at != 0 {
				v.Refs[slot], v.Located[slot] = at, true
			}
		case savLowFillsZero:
			if len(refs) == 1 && refs[0] == 0 && at != 0 {
				v.Refs[slot] = at
			}
		case savLowExclusive:
			if v.Refs[slot] != 0 && at != 0 {
				return v, &savEquipmentError{Fault: savOverlapFault, Field: place.Field}
			}
			if v.Refs[slot] == 0 {
				v.Refs[slot] = at
			}
		}
	}
	for at, ref := range worn {
		refused := sav.WornOverlap(at) && p.Low == savLowRefused ||
			at >= sim.EquipSlots && p.High == savHighRefused
		if refused && ref != 0 {
			return v, &savEquipmentError{Fault: savOverlapFault, Field: sav.WornField}
		}
	}
	for slot, place := range sav.EquipmentPlaces() {
		if place.Field == sav.WornField && place.Ordinal < len(worn) {
			v.Refs[slot], v.Located[slot] = worn[place.Ordinal], true
		}
	}
	return v, nil
}
