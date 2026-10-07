package game

import (
	"bytes"
	"fmt"
	"slices"
	"sort"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

const cityObjectTopologyVersion uint32 = 1

// City topology owns identities and locations. Carry, equipment and books own
// current values. WornCount carries quantity only without a Pack count view.
type cityObjectTopology struct {
	Version uint32
	NextID  sim.SavedObjectID
	Items   []cityItemTopology
	Effects []sim.SavedObjectID
	Spells  []sim.SavedObjectID
	Roots   []cityPartyObjectRoots
	Books   []cityBookTopology
	// Current records retain native Token/raw operands alongside the topology.
	// Party values remain authoritative for mutable stack/equipment fields; the
	// record is the current state's source for fields the party view cannot name.
	ItemRecords   map[sim.SavedObjectID]sim.SavedItemObject
	EffectRecords map[sim.SavedObjectID]sim.SavedEffectObject
	SpellRecords  map[sim.SavedObjectID]sim.SavedSpellObject
	// HumanTails is rebuilt from ordinary SAV records. It is current document
	// residue rather than a second public actor value.
	HumanTails []cityHumanTails
}

type cityItemTopology struct {
	ID        sim.SavedObjectID
	Effects   []sim.SavedObjectID
	Spell     sim.SavedObjectID
	WornCount uint32
}

type cityPartyObjectRoots struct {
	PartyID []byte
	Pack    []sim.SavedObjectID
	Worn    [sim.EquipSlots]sim.SavedObjectID
}

type cityBookTopology struct {
	PartyID []byte
	Slots   [28]sim.SavedObjectID
}

type cityHumanTails struct {
	PartyID []byte
	Tails   [3][2]byte
}

type cityActorObjectBinding struct {
	Object  uint16
	PartyID string
}

// Book IDs are explicit because SavedObjects has no book-slot owner edges.
type cityActorEntityBinding struct {
	Entity  sim.EntityID
	PartyID string
	Books   [28]sim.SavedObjectID
}

func cityItemTopologyByID(items []cityItemTopology, id sim.SavedObjectID) *cityItemTopology {
	for i := range items {
		if items[i].ID == id {
			return &items[i]
		}
	}
	return nil
}

func cityHumanTailsByParty(rows []cityHumanTails, id string) ([3][2]byte, bool) {
	for _, row := range rows {
		if string(row.PartyID) == id {
			return row.Tails, true
		}
	}
	return [3][2]byte{}, false
}

func (g *cityObjectTopology) Clone() *cityObjectTopology {
	if g == nil {
		return nil
	}
	n := *g
	n.Items = slices.Clone(g.Items)
	for i := range n.Items {
		n.Items[i].Effects = slices.Clone(g.Items[i].Effects)
	}
	n.Effects, n.Spells = slices.Clone(g.Effects), slices.Clone(g.Spells)
	n.Roots = slices.Clone(g.Roots)
	for i := range n.Roots {
		n.Roots[i].PartyID = slices.Clone(g.Roots[i].PartyID)
		n.Roots[i].Pack = slices.Clone(g.Roots[i].Pack)
	}
	n.Books = slices.Clone(g.Books)
	for i := range n.Books {
		n.Books[i].PartyID = slices.Clone(g.Books[i].PartyID)
	}
	if g.ItemRecords != nil {
		n.ItemRecords = make(map[sim.SavedObjectID]sim.SavedItemObject, len(g.ItemRecords))
		for id, row := range g.ItemRecords {
			row.Value = row.Value.Clone()
			row.Effects = slices.Clone(row.Effects)
			n.ItemRecords[id] = row
		}
	}
	if g.EffectRecords != nil {
		n.EffectRecords = make(map[sim.SavedObjectID]sim.SavedEffectObject, len(g.EffectRecords))
		for id, row := range g.EffectRecords {
			n.EffectRecords[id] = row
		}
	}
	if g.SpellRecords != nil {
		n.SpellRecords = make(map[sim.SavedObjectID]sim.SavedSpellObject, len(g.SpellRecords))
		for id, row := range g.SpellRecords {
			n.SpellRecords[id] = row
		}
	}
	n.HumanTails = slices.Clone(g.HumanTails)
	for i := range n.HumanTails {
		n.HumanTails[i].PartyID = slices.Clone(g.HumanTails[i].PartyID)
	}
	return &n
}

func cityTopologyCount(total *int, count int) error {
	if count > sim.MaxSavedObjects-*total {
		return fmt.Errorf("city object topology exceeds element budget")
	}
	*total += count
	return nil
}

func cityTopologyPartyBytes(total *int, id []byte) error {
	const limit = 64 << 20
	if len(id) == 0 || len(id) > limit-*total {
		return fmt.Errorf("city object topology has absent/oversized party identity")
	}
	*total += len(id)
	return nil
}

// Null, repeated and shared edges are locations, not extra nodes. Unreferenced
// nodes may remain during a city transaction; they do not create live actors.
func (g *cityObjectTopology) Validate() error {
	if g == nil {
		return nil
	}
	if g.Version != cityObjectTopologyVersion || g.NextID == 0 {
		return fmt.Errorf("invalid city object topology header")
	}
	count := 0
	for _, n := range []int{len(g.Items), len(g.Effects), len(g.Spells), len(g.Roots), len(g.Books), len(g.HumanTails)} {
		if err := cityTopologyCount(&count, n); err != nil {
			return err
		}
	}
	for _, row := range g.Items {
		if err := cityTopologyCount(&count, len(row.Effects)+1); err != nil {
			return err
		}
	}
	partyBytes := 0
	for _, row := range g.Roots {
		if err := cityTopologyCount(&count, len(row.Pack)+len(row.Worn)); err != nil {
			return err
		}
		if err := cityTopologyPartyBytes(&partyBytes, row.PartyID); err != nil {
			return err
		}
	}
	for _, row := range g.Books {
		if err := cityTopologyCount(&count, len(row.Slots)); err != nil {
			return err
		}
		if err := cityTopologyPartyBytes(&partyBytes, row.PartyID); err != nil {
			return err
		}
	}
	kinds := make(map[sim.SavedObjectID]uint8)
	register := func(id sim.SavedObjectID, kind uint8) error {
		if id == 0 || id >= g.NextID || kinds[id] != 0 {
			return fmt.Errorf("invalid/repeated city object identity %d", id)
		}
		kinds[id] = kind
		return nil
	}
	for _, row := range g.Items {
		if err := register(row.ID, 1); err != nil {
			return err
		}
	}
	for _, group := range []struct {
		ids  []sim.SavedObjectID
		kind uint8
	}{{g.Effects, 2}, {g.Spells, 3}} {
		for _, id := range group.ids {
			if err := register(id, group.kind); err != nil {
				return err
			}
		}
	}
	edge := func(id sim.SavedObjectID, kind uint8) error {
		if id != 0 && kinds[id] != kind {
			return fmt.Errorf("city object edge %d has missing/wrong kind", id)
		}
		return nil
	}
	for _, row := range g.Items {
		for _, id := range row.Effects {
			if err := edge(id, 2); err != nil {
				return err
			}
		}
		if err := edge(row.Spell, 3); err != nil {
			return err
		}
	}
	parties := make(map[string]bool)
	for _, row := range g.Roots {
		if parties[string(row.PartyID)] {
			return fmt.Errorf("city object roots repeat party identity")
		}
		parties[string(row.PartyID)] = true
		for _, ids := range [][]sim.SavedObjectID{row.Pack, row.Worn[:]} {
			for _, id := range ids {
				if err := edge(id, 1); err != nil {
					return err
				}
			}
		}
	}
	books := make(map[string]bool)
	for _, row := range g.Books {
		id := string(row.PartyID)
		if !parties[id] || books[id] {
			return fmt.Errorf("city book has absent/repeated party identity")
		}
		books[id] = true
		for _, id := range row.Slots {
			if err := edge(id, 3); err != nil {
				return err
			}
		}
	}
	tails := make(map[string]bool)
	for _, row := range g.HumanTails {
		if err := cityTopologyPartyBytes(&partyBytes, row.PartyID); err != nil {
			return err
		}
		id := string(row.PartyID)
		if tails[id] {
			return fmt.Errorf("city Human tails repeat party identity")
		}
		tails[id] = true
	}
	return nil
}

func (g *cityObjectTopology) allocateID() (sim.SavedObjectID, error) {
	if g == nil {
		return 0, fmt.Errorf("city object topology is absent")
	}
	if err := g.Validate(); err != nil {
		return 0, err
	}
	return mintCityObjectID(&g.NextID)
}

func mintCityObjectID(next *sim.SavedObjectID) (sim.SavedObjectID, error) {
	if *next == 0 || *next == ^sim.SavedObjectID(0) {
		return 0, fmt.Errorf("city object identities exhausted")
	}
	id := *next
	*next++
	return id, nil
}

func (g *cityObjectTopology) sort() {
	sort.Slice(g.Items, func(i, j int) bool { return g.Items[i].ID < g.Items[j].ID })
	slices.Sort(g.Effects)
	slices.Sort(g.Spells)
	sort.Slice(g.Roots, func(i, j int) bool { return bytes.Compare(g.Roots[i].PartyID, g.Roots[j].PartyID) < 0 })
	sort.Slice(g.Books, func(i, j int) bool { return bytes.Compare(g.Books[i].PartyID, g.Books[j].PartyID) < 0 })
	sort.Slice(g.HumanTails, func(i, j int) bool { return bytes.Compare(g.HumanTails[i].PartyID, g.HumanTails[j].PartyID) < 0 })
}

func cityRecordKind(class string) uint8 {
	if savedItemClass(class) {
		return 1
	}
	switch class {
	case "Effect":
		return 2
	case "Spell":
		return 3
	case "Sack":
		return 4
	}
	return 0
}

func cityTopologyRefs(r *sav.DocumentRecordData, name string) ([]uint16, bool, error) {
	if len(r.RefSlots) > 128 {
		return nil, false, fmt.Errorf("city object record has too many reference sites")
	}
	var found []uint16
	ok := false
	for _, row := range r.RefSlots {
		if row.Name != name {
			continue
		}
		if ok || len(row.Objects) > sim.MaxSavedObjects {
			return nil, false, fmt.Errorf("city object record has repeated/oversized %s", name)
		}
		found, ok = row.Objects, true
	}
	return found, ok, nil
}

// The object/party join is explicit. Existing nonzero IDs are reserved before
// any allocation, including IDs on records outside the selected party closure.
// The returned map contains exactly the selected Item, Effect and Spell nodes.
// Native record supplements are attached only by a restore path that has a
// current-document boundary; direct topology capture remains a topology-only
// projection so ordinary value edits cannot alter graph identity.
func captureCityObjectTopology(doc *sav.DocumentData, actors []cityActorObjectBinding, owned []currentOwnedObject, nextID sim.SavedObjectID) (*cityObjectTopology, map[uint16]sim.SavedObjectID, error) {
	return captureCityObjectTopologyWithRecords(doc, actors, owned, nextID, false)
}

func captureCityObjectTopologyWithRecords(doc *sav.DocumentData, actors []cityActorObjectBinding, owned []currentOwnedObject, nextID sim.SavedObjectID, withRecords bool) (*cityObjectTopology, map[uint16]sim.SavedObjectID, error) {
	if doc == nil || doc.Version != sav.DocumentDataVersion || len(doc.Objects) > (1<<15)-1 {
		return nil, nil, fmt.Errorf("city topology needs a bounded ordinary document")
	}
	count := 0
	for _, n := range []int{len(actors), len(owned), len(doc.Objects)} {
		if err := cityTopologyCount(&count, n); err != nil {
			return nil, nil, err
		}
	}
	g := &cityObjectTopology{Version: cityObjectTopologyVersion, NextID: max(nextID, 1)}
	byObject, byID := make(map[uint16]sim.SavedObjectID), make(map[sim.SavedObjectID]bool)
	boundObjects := make(map[uint16]bool)
	for _, row := range owned {
		if row.Kind < 1 || row.Kind > 4 || row.ID == ^sim.SavedObjectID(0) || row.ID != 0 && byID[row.ID] {
			return nil, nil, fmt.Errorf("invalid/repeated city object binding")
		}
		if row.ID != 0 {
			byID[row.ID] = true
			g.NextID = max(g.NextID, row.ID+1)
		}
		if row.Object == 0 {
			continue
		}
		if int(row.Object) > len(doc.Objects) || boundObjects[row.Object] || cityRecordKind(doc.Objects[row.Object-1].Class) != row.Kind {
			return nil, nil, fmt.Errorf("city object binding has repeated/missing/wrong record")
		}
		boundObjects[row.Object], byObject[row.Object] = true, row.ID
	}
	selected := make(map[uint16]uint8)
	var visit func(uint16, uint8) error
	visit = func(ref uint16, kind uint8) error {
		if ref == 0 {
			return nil
		}
		if int(ref) > len(doc.Objects) || cityRecordKind(doc.Objects[ref-1].Class) != kind {
			return fmt.Errorf("city ordinary object edge %d has missing/wrong kind", ref)
		}
		if selected[ref] != 0 {
			return nil
		}
		selected[ref] = kind
		if kind != 1 {
			return nil
		}
		r := &doc.Objects[ref-1]
		effects, ok, err := cityTopologyRefs(r, "Effects")
		if err != nil || !ok {
			return fmt.Errorf("city Item lacks valid Effect edges: %v", err)
		}
		if err := cityTopologyCount(&count, len(effects)+1); err != nil {
			return err
		}
		for _, child := range effects {
			if err := visit(child, 2); err != nil {
				return err
			}
		}
		spell, present, err := cityTopologyRefs(r, "WeaponSpell")
		if err != nil || r.Class == "Weapon" && (!present || len(spell) != 1) || r.Class != "Weapon" && present {
			return fmt.Errorf("city Item has invalid weapon Spell site: %v", err)
		}
		if present {
			return visit(spell[0], 3)
		}
		return nil
	}
	type wireRoots struct {
		party string
		pack  []uint16
		worn  [sim.EquipSlots]uint16
		book  [28]uint16
	}
	var roots []wireRoots
	parties, objects := make(map[string]bool), make(map[uint16]bool)
	partyBytes := 0
	for _, actor := range actors {
		if err := cityTopologyPartyBytes(&partyBytes, []byte(actor.PartyID)); err != nil {
			return nil, nil, err
		}
		if parties[actor.PartyID] || actor.Object == 0 || int(actor.Object) > len(doc.Objects) || objects[actor.Object] {
			return nil, nil, fmt.Errorf("city topology has absent/repeated actor binding")
		}
		parties[actor.PartyID], objects[actor.Object] = true, true
		r := &doc.Objects[actor.Object-1]
		if r.Class != "Unit" && r.Class != "Humanoid" && r.Class != "Human" {
			return nil, nil, fmt.Errorf("city topology actor binding names %s", r.Class)
		}
		if r.Class == "Human" || r.Class == "Humanoid" {
			a, err := savedObjectRaw(r, "UA6", 24)
			if err != nil {
				return nil, nil, err
			}
			b, err := savedObjectRaw(r, "U114", 24)
			if err != nil {
				return nil, nil, err
			}
			m, err := savedObjectRaw(r, "UD4", 64)
			if err != nil {
				return nil, nil, err
			}
			g.HumanTails = append(g.HumanTails, cityHumanTails{PartyID: []byte(actor.PartyID),
				Tails: [3][2]byte{{a[22], a[23]}, {b[22], b[23]}, {m[40], m[41]}}})
		}
		root := wireRoots{party: actor.PartyID}
		var err error
		root.pack, _, err = cityTopologyRefs(r, "Inventory")
		if err != nil {
			return nil, nil, err
		}
		worn, hasWorn, err := cityTopologyRefs(r, "Worn")
		if err != nil || hasWorn && len(worn) != sim.EquipSlots {
			return nil, nil, fmt.Errorf("city actor has invalid Worn edges: %v", err)
		}
		copy(root.worn[:], worn)
		for slot, name := range []string{"HeldWeapon", "HeldShield"} {
			refs, ok, err := cityTopologyRefs(r, name)
			if err != nil || !ok || len(refs) != 1 {
				return nil, nil, fmt.Errorf("city actor lacks exact %s edge: %v", name, err)
			}
			if refs[0] != 0 {
				if root.worn[slot] != 0 {
					return nil, nil, fmt.Errorf("city actor has overlapping held/Worn location")
				}
				root.worn[slot] = refs[0]
			}
		}
		book, _, err := cityTopologyRefs(r, "Spells")
		if err != nil || len(book) > len(root.book) {
			return nil, nil, fmt.Errorf("city actor has invalid book edges: %v", err)
		}
		copy(root.book[:], book)
		if err := cityTopologyCount(&count, len(root.pack)+len(root.worn)+len(root.book)); err != nil {
			return nil, nil, err
		}
		for _, refs := range [][]uint16{root.pack, root.worn[:]} {
			for _, ref := range refs {
				if err := visit(ref, 1); err != nil {
					return nil, nil, err
				}
			}
		}
		for _, ref := range root.book {
			if err := visit(ref, 3); err != nil {
				return nil, nil, err
			}
		}
		roots = append(roots, root)
	}
	indices := make(map[uint16]sim.SavedObjectID)
	for i := range doc.Objects {
		ref := uint16(i + 1)
		if selected[ref] == 0 {
			continue
		}
		id := byObject[ref]
		if id == 0 {
			var err error
			id, err = mintCityObjectID(&g.NextID)
			if err != nil {
				return nil, nil, err
			}
		}
		indices[ref] = id
	}
	for ref, kind := range selected {
		id := indices[ref]
		switch kind {
		case 1:
			effects, _, _ := cityTopologyRefs(&doc.Objects[ref-1], "Effects")
			row := cityItemTopology{ID: id}
			count, err := savedStructureValue(&doc.Objects[ref-1], "F42")
			if err != nil || count == 0 {
				return nil, nil, fmt.Errorf("city Item has missing/zero ordinary quantity: %v", err)
			}
			row.WornCount = count
			for _, child := range effects {
				row.Effects = append(row.Effects, indices[child])
			}
			spells, _, _ := cityTopologyRefs(&doc.Objects[ref-1], "WeaponSpell")
			if len(spells) != 0 {
				row.Spell = indices[spells[0]]
			}
			g.Items = append(g.Items, row)
		case 2:
			g.Effects = append(g.Effects, id)
		case 3:
			g.Spells = append(g.Spells, id)
		}
	}
	if withRecords {
		// A current city document already carries the complete ordinary record.
		// Keep a detached parsed supplement so starting a mission can reuse native
		// Token/raw operands instead of manufacturing a second object state.
		docData := sav.DocumentData{Objects: doc.Objects}
		for ref, kind := range selected {
			id := indices[ref]
			switch kind {
			case 1:
				if source, err := savedItemRecord(&docData, ref); err == nil {
					source.ID, source.Value.ObjectID = id, id
					if node := cityItemTopologyByID(g.Items, id); node != nil {
						source.Effects, source.Spell = slices.Clone(node.Effects), node.Spell
					}
					if g.ItemRecords == nil {
						g.ItemRecords = make(map[sim.SavedObjectID]sim.SavedItemObject)
					}
					g.ItemRecords[id] = source
				}
			case 2:
				if source, err := savedEffectRecord(&doc.Objects[ref-1]); err == nil {
					source.ID = id
					if g.EffectRecords == nil {
						g.EffectRecords = make(map[sim.SavedObjectID]sim.SavedEffectObject)
					}
					g.EffectRecords[id] = source
				}
			case 3:
				if source, err := savedSpellRecord(&doc.Objects[ref-1]); err == nil {
					source.ID = id
					if g.SpellRecords == nil {
						g.SpellRecords = make(map[sim.SavedObjectID]sim.SavedSpellObject)
					}
					g.SpellRecords[id] = source
				}
			}
		}
	}
	for _, source := range roots {
		root, book := cityPartyObjectRoots{PartyID: []byte(source.party)}, cityBookTopology{PartyID: []byte(source.party)}
		for _, ref := range source.pack {
			root.Pack = append(root.Pack, indices[ref])
		}
		for i, ref := range source.worn {
			root.Worn[i] = indices[ref]
		}
		for i, ref := range source.book {
			book.Slots[i] = indices[ref]
		}
		g.Roots, g.Books = append(g.Roots, root), append(g.Books, book)
	}
	g.trimWornCounts()
	g.sort()
	if err := g.Validate(); err != nil {
		return nil, nil, err
	}
	return g, indices, nil
}

// This adapter carries registry edges and quantity without a native Worn
// field. Other scalar agreement belongs to the caller's live-value boundary.
func captureCityObjectTopologyFromSaved(objects *sim.SavedObjects, actors []cityActorEntityBinding) (*cityObjectTopology, error) {
	if objects == nil || objects.Version != sim.SavedObjectsVersion || objects.NextID == 0 {
		return nil, fmt.Errorf("city topology needs a current object registry")
	}
	count := 0
	for _, n := range []int{len(objects.Items), len(objects.Effects), len(objects.Spells), len(objects.Sacks), len(objects.Containers), len(objects.SackRoots), len(objects.ItemRoots), len(actors)} {
		if err := cityTopologyCount(&count, n); err != nil {
			return nil, err
		}
	}
	for _, row := range objects.Items {
		if err := cityTopologyCount(&count, len(row.Effects)+1); err != nil {
			return nil, err
		}
	}
	for _, row := range objects.Containers {
		if err := cityTopologyCount(&count, len(row.Items)); err != nil {
			return nil, err
		}
	}
	kinds := make(map[sim.SavedObjectID]uint8)
	register := func(id sim.SavedObjectID, kind uint8) error {
		if id == 0 || id >= objects.NextID || kinds[id] != 0 {
			return fmt.Errorf("invalid/repeated registry identity %d", id)
		}
		kinds[id] = kind
		return nil
	}
	items := make(map[sim.SavedObjectID]*sim.SavedItemObject)
	effects := make(map[sim.SavedObjectID]*sim.SavedEffectObject)
	spells := make(map[sim.SavedObjectID]*sim.SavedSpellObject)
	for i := range objects.Items {
		row := &objects.Items[i]
		if err := register(row.ID, 1); err != nil {
			return nil, err
		}
		if row.Owner != (sim.SavedObjectOwner{}) || row.InFlight != 0 || row.Retired != (len(objects.Locations(row.ID)) == 0) {
			return nil, fmt.Errorf("city registry Item ownership is not current")
		}
		items[row.ID] = row
	}
	for i := range objects.Effects {
		row := &objects.Effects[i]
		if err := register(row.ID, 2); err != nil {
			return nil, err
		}
		effects[row.ID] = row
	}
	for i := range objects.Spells {
		row := &objects.Spells[i]
		if err := register(row.ID, 3); err != nil {
			return nil, err
		}
		spells[row.ID] = row
	}
	for _, row := range objects.Sacks {
		if err := register(row.ID, 4); err != nil {
			return nil, err
		}
	}
	g := &cityObjectTopology{Version: cityObjectTopologyVersion, NextID: objects.NextID}
	byEntity, parties := make(map[sim.EntityID]int), make(map[string]bool)
	partyBytes := 0
	for _, actor := range actors {
		if err := cityTopologyPartyBytes(&partyBytes, []byte(actor.PartyID)); err != nil {
			return nil, err
		}
		if _, duplicate := byEntity[actor.Entity]; duplicate || parties[actor.PartyID] {
			return nil, fmt.Errorf("city topology has repeated live actor/party binding")
		}
		if err := cityTopologyCount(&count, sim.EquipSlots+len(actor.Books)); err != nil {
			return nil, err
		}
		byEntity[actor.Entity], parties[actor.PartyID] = len(g.Roots), true
		g.Roots = append(g.Roots, cityPartyObjectRoots{PartyID: []byte(actor.PartyID)})
		g.Books = append(g.Books, cityBookTopology{PartyID: []byte(actor.PartyID), Slots: actor.Books})
	}
	selected, containers := make(map[sim.SavedObjectID]bool), make(map[sim.EntityID]bool)
	for _, row := range objects.Containers {
		index, ok := byEntity[row.Owner.Entity]
		if row.Owner.Kind != sim.SavedOwnerActorPack || !ok {
			continue
		}
		owner := sim.SavedObjectOwner{Kind: sim.SavedOwnerActorPack, Entity: row.Owner.Entity}
		if row.Owner != owner || containers[owner.Entity] || !row.Present && len(row.Items) != 0 {
			return nil, fmt.Errorf("city topology has invalid/repeated live pack")
		}
		containers[owner.Entity] = true
		g.Roots[index].Pack = slices.Clone(row.Items)
		for _, id := range row.Items {
			if id == 0 {
				continue
			}
			if items[id] == nil || items[id].Retired {
				return nil, fmt.Errorf("city live pack has missing/wrong item owner")
			}
			selected[id] = true
		}
	}
	for _, root := range objects.ItemRoots {
		index, ok := byEntity[root.Owner.Entity]
		if !ok || root.Owner.Kind != sim.SavedOwnerActorWorn {
			continue
		}
		owner := sim.SavedObjectOwner{Kind: sim.SavedOwnerActorWorn, Entity: root.Owner.Entity, Slot: root.Owner.Slot}
		if root.Owner != owner || root.Owner.Slot == 0 || root.Owner.Slot > sim.EquipSlots || g.Roots[index].Worn[root.Owner.Slot-1] != 0 || items[root.ID] == nil || items[root.ID].Retired {
			return nil, fmt.Errorf("city live equipment has invalid/repeated slot")
		}
		g.Roots[index].Worn[root.Owner.Slot-1] = root.ID
		selected[root.ID] = true
	}

	usedEffects, usedSpells := make(map[sim.SavedObjectID]bool), make(map[sim.SavedObjectID]bool)
	addSpell := func(id sim.SavedObjectID) error {
		if id != 0 {
			if spells[id] == nil || spells[id].Retired {
				return fmt.Errorf("city live Spell edge has missing/retired child")
			}
			usedSpells[id] = true
		}
		return nil
	}
	for id := range selected {
		row := items[id]
		for _, child := range row.Effects {
			if child != 0 {
				if effects[child] == nil || effects[child].Retired {
					return nil, fmt.Errorf("city live Effect edge has missing/retired child")
				}
				usedEffects[child] = true
			}
		}
		if err := addSpell(row.Spell); err != nil {
			return nil, err
		}
		g.Items = append(g.Items, cityItemTopology{ID: id, Effects: slices.Clone(row.Effects), Spell: row.Spell, WornCount: row.Value.Count})
	}
	for _, book := range g.Books {
		for _, id := range book.Slots {
			if err := addSpell(id); err != nil {
				return nil, err
			}
		}
	}
	for id := range usedEffects {
		g.Effects = append(g.Effects, id)
	}
	for id := range usedSpells {
		g.Spells = append(g.Spells, id)
	}
	// The topology above is the identity/location projection. Keep the
	// selected registry records as a separate current-state supplement so a
	// city SAV can carry native Token and child operands that PartyMember does
	// not expose. The live registry remains untouched; these are detached
	// copies owned by the snapshot.
	if len(selected) != 0 {
		g.ItemRecords = make(map[sim.SavedObjectID]sim.SavedItemObject, len(selected))
		for id := range selected {
			row := *items[id]
			row.Value = row.Value.Clone()
			row.Effects = slices.Clone(row.Effects)
			g.ItemRecords[id] = row
		}
	}
	if len(usedEffects) != 0 {
		g.EffectRecords = make(map[sim.SavedObjectID]sim.SavedEffectObject, len(usedEffects))
		for id := range usedEffects {
			g.EffectRecords[id] = *effects[id]
		}
	}
	if len(usedSpells) != 0 {
		g.SpellRecords = make(map[sim.SavedObjectID]sim.SavedSpellObject, len(usedSpells))
		for id := range usedSpells {
			g.SpellRecords[id] = *spells[id]
		}
	}
	g.trimWornCounts()
	g.sort()
	if err := g.Validate(); err != nil {
		return nil, err
	}
	return g, nil
}

// publishSessionEntry is town entry's publication (SAV-1114, SAV-POSTLOAD-222):
// every Item a party member carries or wears has its pending pickup flag,
// Token.T08, stored as 0. The party is the recipient's own Humans.
func (g *cityObjectTopology) publishSessionEntry() {
	if g == nil {
		return
	}
	for _, roots := range g.Roots {
		for _, id := range append(slices.Clone(roots.Pack), roots.Worn[:]...) {
			if row, ok := g.ItemRecords[id]; ok && row.Token.T08 != 0 {
				row.Token.T08 = 0
				row.Coverage.Unknown &^= sim.SavedUnknownFlags
				g.ItemRecords[id] = row
			}
		}
	}
}
