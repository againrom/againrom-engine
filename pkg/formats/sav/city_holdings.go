package sav

import (
	"encoding/binary"
	"fmt"
	"reflect"
)

// CityItemGraph is a complete current pack, with local one-based references.
// It carries semantic records, never a World envelope or original file bytes.
// Shared or borrowed children remain outside this bounded city update.
type CityItemGraph struct {
	Objects     []DocumentRecordData
	Inventory   []uint16
	Present     bool
	InsertIndex uint32
	Accumulator int32
}

func cityItemGraphRoots(g *CityItemGraph) ([]*cityObject, map[uint16]*cityObject, error) {
	if g == nil || len(g.Objects) > maxCityDataObjects || len(g.Inventory) > maxCityDataObjects ||
		!g.Present && (len(g.Inventory) != 0 || g.InsertIndex != 0 || g.Accumulator != 0) {
		return nil, nil, fmt.Errorf("sav: invalid current city pack extent/presence")
	}
	if err := (&documentDataBudget{}).check(reflect.ValueOf(*g), 0); err != nil {
		return nil, nil, err
	}
	objects := make([]*Record, len(g.Objects))
	for i, r := range g.Objects {
		switch r.Class {
		case "Item", "Weapon", "Armor", "Shield", "Effect", "Spell":
		default:
			return nil, nil, fmt.Errorf("sav: current city pack contains %s", r.Class)
		}
		objects[i] = newRecord(r.Class, 0, 0)
	}
	for i, r := range g.Objects {
		if err := documentRecordFromData(r, objects[i], objects, false, 0); err != nil {
			return nil, nil, err
		}
	}
	seen := make(map[*Record]bool)
	var visit func(*Record, string) error
	visit = func(r *Record, class string) error {
		if r == nil || seen[r] || (class == "Item" && r.Class != "Item" && r.Class != "Weapon" && r.Class != "Armor" && r.Class != "Shield") || class != "Item" && r.Class != class {
			return fmt.Errorf("sav: current city pack has missing, aliased, or wrong-class %s", class)
		}
		seen[r] = true
		if class == "Item" {
			for _, child := range r.RefSlots["Effects"] {
				if err := visit(child, "Effect"); err != nil {
					return err
				}
			}
			for _, child := range r.RefSlots["WeaponSpell"] {
				if child != nil {
					if err := visit(child, "Spell"); err != nil {
						return err
					}
				}
			}
		}
		return nil
	}
	var roots []*Record
	for _, index := range g.Inventory {
		if index == 0 || int(index) > len(objects) {
			return nil, nil, fmt.Errorf("sav: current city pack root outside table")
		}
		r := objects[index-1]
		if err := visit(r, "Item"); err != nil {
			return nil, nil, err
		}
		roots = append(roots, r)
	}
	if len(seen) != len(objects) {
		return nil, nil, fmt.Errorf("sav: current city pack has orphan records")
	}
	// Both existing grammars validate this representation. The temporary
	// reference stream is discarded and is never an export input or DTO field.
	b, err := serializeArchiveReferences(roots)
	if err != nil {
		return nil, nil, err
	}
	r := newCityArchiveReader(&cityCursor{b: b})
	var out []*cityObject
	for range roots {
		o, err := r.reference("current pack")
		if err != nil {
			return nil, nil, err
		}
		out = append(out, o)
	}
	if r.c.p != len(b) {
		return nil, nil, fmt.Errorf("sav: trailing current pack fields")
	}
	return out, r.objects, nil
}

func ValidateCityItemGraph(g *CityItemGraph) error {
	_, _, err := cityItemGraphRoots(g)
	return err
}

// CityEquipmentGraph is a complete current worn-equipment set, in the same
// local one-based reference scheme as CityItemGraph. Slot order matches
// mapload.MemberItemEquipment: 0 is the weapon, 1 the shield, 2..11 the
// remaining armour and accessory slots. A zero Slots entry is an empty slot,
// not an omitted one; applyCityEquipment clears the base document's frozen
// reference for it exactly as it replaces a filled one.
type CityEquipmentGraph struct {
	Objects []DocumentRecordData
	Slots   [12]uint16
}

// cityEquipmentGraphRoots mirrors cityItemGraphRoots, but a zero Slots entry
// is a legitimate empty slot rather than a missing root: it is skipped from
// the visit/serialize pass and its output position stays nil.
func cityEquipmentGraphRoots(g *CityEquipmentGraph) ([12]*cityObject, map[uint16]*cityObject, error) {
	var empty [12]*cityObject
	if g == nil || len(g.Objects) > maxCityDataObjects {
		return empty, nil, fmt.Errorf("sav: invalid current city worn extent")
	}
	if err := (&documentDataBudget{}).check(reflect.ValueOf(*g), 0); err != nil {
		return empty, nil, err
	}
	objects := make([]*Record, len(g.Objects))
	for i, r := range g.Objects {
		switch r.Class {
		case "Item", "Weapon", "Armor", "Shield", "Effect", "Spell":
		default:
			return empty, nil, fmt.Errorf("sav: current city worn set contains %s", r.Class)
		}
		objects[i] = newRecord(r.Class, 0, 0)
	}
	for i, r := range g.Objects {
		if err := documentRecordFromData(r, objects[i], objects, false, 0); err != nil {
			return empty, nil, err
		}
	}
	seen := make(map[*Record]bool)
	var visit func(*Record, string) error
	visit = func(r *Record, class string) error {
		if r == nil || seen[r] || (class == "Item" && r.Class != "Item" && r.Class != "Weapon" && r.Class != "Armor" && r.Class != "Shield") || class != "Item" && r.Class != class {
			return fmt.Errorf("sav: current city worn set has missing, aliased, or wrong-class %s", class)
		}
		seen[r] = true
		if class == "Item" {
			for _, child := range r.RefSlots["Effects"] {
				if err := visit(child, "Effect"); err != nil {
					return err
				}
			}
			for _, child := range r.RefSlots["WeaponSpell"] {
				if child != nil {
					if err := visit(child, "Spell"); err != nil {
						return err
					}
				}
			}
		}
		return nil
	}
	var roots []*Record
	var order []int
	for slot, index := range g.Slots {
		if index == 0 {
			continue
		}
		if int(index) > len(objects) {
			return empty, nil, fmt.Errorf("sav: current city worn slot %d outside table", slot)
		}
		r := objects[index-1]
		if err := visit(r, "Item"); err != nil {
			return empty, nil, err
		}
		roots, order = append(roots, r), append(order, slot)
	}
	if len(seen) != len(objects) {
		return empty, nil, fmt.Errorf("sav: current city worn set has orphan records")
	}
	b, err := serializeArchiveReferences(roots)
	if err != nil {
		return empty, nil, err
	}
	c := newCityArchiveReader(&cityCursor{b: b})
	var out [12]*cityObject
	for i := range roots {
		o, err := c.reference("current worn")
		if err != nil {
			return empty, nil, err
		}
		out[order[i]] = o
	}
	if c.c.p != len(b) {
		return empty, nil, fmt.Errorf("sav: trailing current worn fields")
	}
	return out, c.objects, nil
}

func ValidateCityEquipmentGraph(g *CityEquipmentGraph) error {
	_, _, err := cityEquipmentGraphRoots(g)
	return err
}

// applyCityEquipment rebinds the weapon, shield and armour references to the
// current live capture, replacing whatever the base document froze there.
// pruneCitySoldObjects (called once per DocumentData/Marshal pass, after every
// applyCityHoldings/applyCityEquipment call) then drops any base-document
// object this unbinding leaves unreachable, before remintCityIdentities ever
// inspects its owner field.
func applyCityEquipment(d *cityDocument, actor *cityObject, g *CityEquipmentGraph) ([]*cityObject, error) {
	if g == nil {
		return nil, nil
	}
	if actor == nil || actor.unit == nil || len(actor.unit.equipment) != 13 {
		return nil, fmt.Errorf("sav: current worn owner is not an equipped Unit")
	}
	slots, objects, err := cityEquipmentGraphRoots(g)
	if err != nil {
		return nil, err
	}
	actor.unit.reference74 = slots[0]
	actor.unit.reference78 = slots[1]
	for i := 2; i < len(slots); i++ {
		actor.unit.equipment[i] = slots[i]
	}
	var next uint16
	for index := range d.objects {
		next = max(next, index)
	}
	var added []*cityObject
	for index := uint16(1); index != 0; index++ {
		o := objects[index]
		if o == nil {
			continue
		}
		if next >= 32766 {
			return nil, fmt.Errorf("sav: current city worn set exhausts object indices")
		}
		next++
		o.sourceIndex = next
		d.objects[next] = o
		added = append(added, o)
	}
	return added, nil
}

func applyCityHoldings(d *cityDocument, actor *cityObject, g *CityItemGraph) ([]*cityObject, error) {
	if g == nil {
		return nil, nil
	}
	if actor == nil || actor.unit == nil {
		return nil, fmt.Errorf("sav: current pack owner is not Unit")
	}
	roots, objects, err := cityItemGraphRoots(g)
	if err != nil {
		return nil, err
	}
	actor.unit.container = roots
	actor.unit.containerFlag = 0
	if g.Present {
		actor.unit.containerFlag = 1
	}
	actor.unit.containerTails = [2]uint32{g.InsertIndex, uint32(g.Accumulator)}
	var next uint16
	for index := range d.objects {
		next = max(next, index)
	}
	// Archive-local order is deterministic; no pointer or map order is an
	// allocation input. The ordinary city writer later remints all live keys.
	var added []*cityObject
	for index := uint16(1); index != 0; index++ {
		o := objects[index]
		if o == nil {
			continue
		}
		if next >= 32766 {
			return nil, fmt.Errorf("sav: current city pack exhausts object indices")
		}
		next++
		o.sourceIndex = next
		d.objects[next] = o
		added = append(added, o)
	}
	return added, nil
}

// applyCityCharacterGraphs grafts one character's current pack and worn set
// and returns every record they added, for clearUnresolvedCityOwners.
func applyCityCharacterGraphs(d *cityDocument, actor *cityObject, u CityCharacterUpdate) ([]*cityObject, error) {
	pack, err := applyCityHoldings(d, actor, u.Holdings)
	if err != nil {
		return nil, err
	}
	worn, err := applyCityEquipment(d, actor, u.Worn)
	if err != nil {
		return nil, err
	}
	return append(pack, worn...), nil
}

// clearUnresolvedCityOwners zeroes the owner Reference of a current graph's
// Item or Effect when that key names no object left in the document, which
// is the value the original's own load gives a missing key (SAV-PTRMAP-035).
// A resolving key is kept for remintCityIdentities to translate. It runs
// after pruning, so a key whose object the prune removed counts as missing.
func clearUnresolvedCityOwners(d *cityDocument, added []*cityObject) {
	if len(added) == 0 {
		return
	}
	present := make(map[uint32]bool, len(d.objects))
	for _, obj := range d.objects {
		if identity := cityObjectIdentity(obj); identity != 0 {
			present[identity] = true
		}
	}
	for _, obj := range added {
		var token []byte
		switch {
		case obj.item != nil && len(obj.item.token) == 37:
			token = obj.item.token
		case obj.effect != nil && len(obj.effect.token) == 37:
			token = obj.effect.token
		default:
			continue
		}
		if owner := binary.LittleEndian.Uint32(token[33:37]); owner != 0 && !present[owner] {
			binary.LittleEndian.PutUint32(token[33:37], 0)
		}
	}
}

// DocumentData rebuilds the generic semantic city representation without
// reminting keys or manufacturing a World. It is an import view of this source.
func (p *CityProvenance) DocumentData(holdings ...CityCharacterUpdate) (DocumentData, error) {
	if p == nil || p.document == nil {
		return DocumentData{}, fmt.Errorf("sav: missing city provenance")
	}
	d := cloneCityDocument(p.document)
	seen := make(map[uint32]bool)
	var added []*cityObject
	for _, u := range holdings {
		index, ok := p.characterSourceIndex[u.Identity]
		if !ok || seen[u.Identity] {
			return DocumentData{}, fmt.Errorf("sav: invalid current city pack owner")
		}
		seen[u.Identity] = true
		objects, err := applyCityCharacterGraphs(d, d.objects[index], u)
		if err != nil {
			return DocumentData{}, err
		}
		added = append(added, objects...)
	}
	if len(holdings) != 0 {
		if err := pruneCitySoldObjects(d); err != nil {
			return DocumentData{}, err
		}
	}
	clearUnresolvedCityOwners(d, added)
	body, err := serializeCityDocument(d)
	if err != nil {
		return DocumentData{}, err
	}
	a, err := parseArchiveDocument(body)
	if err != nil {
		return DocumentData{}, err
	}
	return saveDocumentToData(&saveDocument{version: p.version, archive: a, state: p.state, campaign: p.campaign})
}
