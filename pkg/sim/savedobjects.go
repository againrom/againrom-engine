package sim

import (
	"fmt"
	"slices"
	"strings"
)

const SavedObjectsVersion uint32 = 2
const MaxSavedObjects = 1 << 20

// These are native identities. Token Identity/Reference and archive indices
// are never allocator inputs. A nil registry and ObjectID zero remain absent.
type SavedObjectOrigin struct {
	Kind   SavedObjectOriginKind
	Parent SavedObjectID
}

type SavedObjectOriginKind uint8

const (
	SavedObjectOriginal SavedObjectOriginKind = iota + 1
	SavedObjectGenerated
	SavedObjectSplit
)

type SavedObjectUnknown uint64

const (
	SavedUnknownPosition SavedObjectUnknown = 1 << iota
	SavedUnknownRuntimeID
	SavedUnknownDefinitionRow
	SavedUnknownT0E
	SavedUnknownFlags
	SavedUnknownT18
	SavedUnknownPrice
	SavedUnknownIdentity
	SavedUnknownReference
	SavedUnknownF45
	SavedUnknownF46
	SavedUnknownF47
	SavedUnknownF48
	SavedUnknownSpellInitialization
	SavedUnknownCountWidth
	SavedUnknownMergePolicy
	SavedUnknownContainerLoad
)

const savedUnknownAll = SavedUnknownContainerLoad*2 - 1
const SavedUnknownToken = SavedUnknownPosition | SavedUnknownRuntimeID | SavedUnknownDefinitionRow | SavedUnknownT0E | SavedUnknownFlags | SavedUnknownT18 | SavedUnknownPrice | SavedUnknownIdentity | SavedUnknownReference

// Unknown operands may retain a native value, but are not original-writer
// authority. Unsupported names a bounded ownership/producer limitation.
type SavedObjectCoverage struct {
	Unknown     SavedObjectUnknown
	Unsupported string
}

type SavedObjectToken struct {
	Position            [12]byte
	RuntimeID           uint32
	T0C                 uint8
	T0E                 uint16
	T08                 uint32
	T18                 uint16
	T1C                 uint32
	Identity, Reference uint32
}

type SavedOwnerKind uint8

const (
	SavedOwnerActorPack SavedOwnerKind = iota + 1
	SavedOwnerActorWorn
	SavedOwnerSack
	SavedOwnerSession
	SavedOwnerInFlight
	SavedOwnerRetired
)

// Slot is an equipment location, never identity. Pack ordering is held by
// Containers. SessionHandle names a paired game-session value, not a pointer.
type SavedObjectOwner struct {
	Kind          SavedOwnerKind
	Entity        EntityID
	Object        SavedObjectID
	Slot          uint32
	SessionHandle uint64
}

type SavedItemObject struct {
	ID            SavedObjectID
	Origin        SavedObjectOrigin
	Owner         SavedObjectOwner `json:",omitzero"` // Legacy decode operand; current registries leave it zero.
	Retired       bool
	InFlight      uint32
	Value         ItemStack
	Token         SavedObjectToken
	F45, F46, F47 uint8
	F48           uint16
	Effects       []SavedObjectID
	Spell         SavedObjectID
	Coverage      SavedObjectCoverage
}

// Child edges, not a made-up exclusive parent, own Effects and Spells.
// ExternalReferences retains incoming references outside this bounded owner.
type SavedEffectObject struct {
	ID                 SavedObjectID
	Origin             SavedObjectOrigin
	Token              SavedObjectToken
	Value              ItemEffect
	E0C                uint8
	ExternalReferences uint32
	Retired            bool
	Coverage           SavedObjectCoverage
}

type SavedSpellObject struct {
	ID                 SavedObjectID
	Origin             SavedObjectOrigin
	Value              SourceItemSpell
	This               uint32
	ExternalReferences uint32
	Retired            bool
	Coverage           SavedObjectCoverage
}

type SavedSackObject struct {
	ID       SavedObjectID
	Origin   SavedObjectOrigin
	Token    SavedObjectToken
	Gold     uint32
	Retired  bool
	Coverage SavedObjectCoverage
}

type SavedObjectContainer struct {
	Owner       SavedObjectOwner
	Present     bool
	InsertIndex uint32
	Accumulator int32
	Items       []SavedObjectID
	Coverage    SavedObjectCoverage
}

// Container occurrences are addressed by their exact ordinal. Worn and
// session locations use Index zero and are recorded in ItemRoots.
type SavedItemLocation struct {
	Owner SavedObjectOwner
	Index uint32
}

type SavedItemRoot struct {
	ID    SavedObjectID
	Owner SavedObjectOwner
}

// Book roots carry identity only. The actor's current book owns slot values.
type SavedBookRoot struct {
	Entity EntityID
	Slots  [28]SavedObjectID
}

// Values reuse the ordinary simulation types. CompareLive is mandatory at
// persistence/binding boundaries; these rows never repair the live views.
type SavedObjects struct {
	Version    uint32
	NextID     SavedObjectID
	Items      []SavedItemObject
	Effects    []SavedEffectObject
	Spells     []SavedSpellObject
	Sacks      []SavedSackObject
	SackRoots  []SavedObjectID
	Containers []SavedObjectContainer
	ItemRoots  []SavedItemRoot
	BookRoots  []SavedBookRoot
}

type SavedObjectBinding struct {
	ID    SavedObjectID
	Owner SavedObjectOwner
	Index uint32
	Value ItemStack
}

type SavedExternalItem struct {
	Handle uint64
	ID     SavedObjectID
	Value  ItemStack
}

type SavedMergePolicy uint8

const (
	SavedMergeNone SavedMergePolicy = iota
	SavedMergeNativeRetention
	SavedMergeOriginalPredicate // predicate only; native counts still use uint32
)

func (r *SavedObjects) Clone() *SavedObjects {
	if r == nil {
		return nil
	}
	n := *r
	n.Items = slices.Clone(r.Items)
	for i := range n.Items {
		n.Items[i].Value = n.Items[i].Value.Clone()
		n.Items[i].Effects = slices.Clone(n.Items[i].Effects)
	}
	n.Effects, n.Spells, n.Sacks = slices.Clone(r.Effects), slices.Clone(r.Spells), slices.Clone(r.Sacks)
	n.SackRoots = slices.Clone(r.SackRoots)
	n.ItemRoots = slices.Clone(r.ItemRoots)
	n.BookRoots = slices.Clone(r.BookRoots)
	n.Containers = slices.Clone(r.Containers)
	for i := range n.Containers {
		n.Containers[i].Items = slices.Clone(n.Containers[i].Items)
	}
	return &n
}

func savedCoverageFault(c SavedObjectCoverage) error {
	if c.Unknown & ^savedUnknownAll != 0 || len(c.Unsupported) > 512 || strings.ContainsRune(c.Unsupported, '\x00') {
		return fmt.Errorf("sim: invalid saved object coverage")
	}
	return nil
}

func savedOwnerFault(o SavedObjectOwner) error {
	switch o.Kind {
	case SavedOwnerActorPack:
		if o.Object == 0 && o.Slot == 0 && o.SessionHandle == 0 {
			return nil
		}
	case SavedOwnerActorWorn:
		if o.Object == 0 && o.Slot >= 1 && o.Slot <= EquipSlots && o.SessionHandle == 0 {
			return nil
		}
	case SavedOwnerSack:
		if o.Entity == 0 && o.Object != 0 && o.Slot == 0 && o.SessionHandle == 0 {
			return nil
		}
	case SavedOwnerSession:
		if o.Entity == 0 && o.Object == 0 && o.Slot == 0 && o.SessionHandle != 0 {
			return nil
		}
	case SavedOwnerInFlight, SavedOwnerRetired:
		if o.Entity == 0 && o.Object == 0 && o.Slot == 0 && o.SessionHandle == 0 {
			return nil
		}
	}
	return fmt.Errorf("sim: invalid saved object owner")
}

// D-1: every lookup accessor below shares one shape — index into a field of
// r without checking r itself first. A registry-less World (SourceTownEquipment
// builds one; see sourceequipmove.go) carries a nil *SavedObjects, so any of
// these reached directly (outside a mutate() transaction, which only ever
// hands its callback an already-cloned, guaranteed non-nil registry) would
// dereference a nil r. Guard each the same way the exported Item/Validate/
// Clone already do on this same receiver: nil in, nil out, no lookup.
func (r *SavedObjects) item(id SavedObjectID) *SavedItemObject {
	if r == nil {
		return nil
	}
	i, ok := slices.BinarySearchFunc(r.Items, id, func(v SavedItemObject, id SavedObjectID) int { return savedIDCompare(v.ID, id) })
	if !ok {
		return nil
	}
	return &r.Items[i]
}

func (r *SavedObjects) effect(id SavedObjectID) *SavedEffectObject {
	if r == nil {
		return nil
	}
	i, ok := slices.BinarySearchFunc(r.Effects, id, func(v SavedEffectObject, id SavedObjectID) int { return savedIDCompare(v.ID, id) })
	if !ok {
		return nil
	}
	return &r.Effects[i]
}

func (r *SavedObjects) spell(id SavedObjectID) *SavedSpellObject {
	if r == nil {
		return nil
	}
	i, ok := slices.BinarySearchFunc(r.Spells, id, func(v SavedSpellObject, id SavedObjectID) int { return savedIDCompare(v.ID, id) })
	if !ok {
		return nil
	}
	return &r.Spells[i]
}

func (r *SavedObjects) sack(id SavedObjectID) *SavedSackObject {
	if r == nil {
		return nil
	}
	i, ok := slices.BinarySearchFunc(r.Sacks, id, func(v SavedSackObject, id SavedObjectID) int { return savedIDCompare(v.ID, id) })
	if !ok {
		return nil
	}
	return &r.Sacks[i]
}

func savedIDCompare(a, b SavedObjectID) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

func (r *SavedObjects) container(owner SavedObjectOwner) *SavedObjectContainer {
	if r == nil {
		return nil
	}
	for i := range r.Containers {
		if r.Containers[i].Owner == owner {
			return &r.Containers[i]
		}
	}
	return nil
}

// Item returns an isolated current value; obtaining a view transfers nothing.
func (r *SavedObjects) Item(id SavedObjectID) (SavedItemObject, bool) {
	if r != nil {
		if row := r.item(id); row != nil {
			out := *row
			out.Value, out.Effects = row.Value.Clone(), slices.Clone(row.Effects)
			return out, true
		}
	}
	return SavedItemObject{}, false
}

// Validate admits in-flight transaction state, but rejects conflicting live
// owners, duplicated scalars, orphan children, or stale container edges.
func (r *SavedObjects) Validate() error {
	if r == nil {
		return nil
	}
	if r.Version != SavedObjectsVersion || r.NextID == 0 || len(r.Items)+len(r.Effects)+len(r.Spells)+len(r.Sacks)+len(r.Containers)+len(r.SackRoots)+len(r.ItemRoots)+len(r.BookRoots) > MaxSavedObjects {
		return fmt.Errorf("sim: invalid saved object registry header/count")
	}
	kinds := make(map[SavedObjectID]uint8)
	register := func(id SavedObjectID, previous SavedObjectID, kind uint8, origin SavedObjectOrigin, coverage SavedObjectCoverage) error {
		if id == 0 || id <= previous || id >= r.NextID || kinds[id] != 0 || origin.Kind < SavedObjectOriginal || origin.Kind > SavedObjectSplit || (origin.Kind == SavedObjectSplit) != (origin.Parent != 0) || origin.Parent >= id {
			return fmt.Errorf("sim: invalid saved object identity/origin %d", id)
		}
		if err := savedCoverageFault(coverage); err != nil {
			return err
		}
		kinds[id] = kind
		return nil
	}
	var previous SavedObjectID
	for _, row := range r.Items {
		if err := register(row.ID, previous, 1, row.Origin, row.Coverage); err != nil {
			return err
		}
		previous = row.ID
		if row.Owner != (SavedObjectOwner{}) || row.InFlight > MaxSavedObjects {
			return fmt.Errorf("sim: current Item carries legacy ownership or excessive transient roots")
		}
		if row.Value.ObjectID != row.ID || row.Value.Code == 0 || row.Value.Count == 0 || len(row.Effects) != len(row.Value.Effects) || row.Token.T1C != uint32(row.Value.Price) || row.Value.Count > 65535 && row.Coverage.Unknown&SavedUnknownCountWidth == 0 {
			return fmt.Errorf("sim: saved item %d has conflicting value", row.ID)
		}
		if err := row.Value.Instance().ValidateWeight(); err != nil {
			return err
		}
		if row.Value.SourceEquipment.Class != 0 && row.Token.T0C != row.Value.SourceEquipment.DefinitionRow {
			return fmt.Errorf("sim: saved item %d definition row differs", row.ID)
		}
		if (row.Spell != 0) != row.Value.SourceEquipment.Spell.Present {
			return fmt.Errorf("sim: saved item %d Spell presence differs", row.ID)
		}
	}
	previous = 0
	for _, row := range r.Effects {
		if err := register(row.ID, previous, 2, row.Origin, row.Coverage); err != nil {
			return err
		}
		previous = row.ID
		if row.Retired && row.ExternalReferences != 0 {
			return fmt.Errorf("sim: invalid saved Effect %d", row.ID)
		}
	}
	previous = 0
	for _, row := range r.Spells {
		if err := register(row.ID, previous, 3, row.Origin, row.Coverage); err != nil {
			return err
		}
		previous = row.ID
		if !row.Value.Present || row.Retired && row.ExternalReferences != 0 {
			return fmt.Errorf("sim: invalid saved Spell %d", row.ID)
		}
	}
	previous = 0
	sacks := make(map[SavedObjectID]bool)
	for _, row := range r.Sacks {
		if err := register(row.ID, previous, 4, row.Origin, row.Coverage); err != nil {
			return err
		}
		previous = row.ID
		if row.Retired && row.Gold != 0 {
			return fmt.Errorf("sim: retired saved Sack carries gold")
		}
		sacks[row.ID] = !row.Retired
	}
	for _, row := range r.Items {
		if row.Origin.Parent != 0 && kinds[row.Origin.Parent] != 1 {
			return fmt.Errorf("sim: saved item has missing split parent")
		}
	}
	for _, row := range r.Effects {
		if row.Origin.Parent != 0 && kinds[row.Origin.Parent] != 2 {
			return fmt.Errorf("sim: saved Effect has missing split parent")
		}
	}
	for _, row := range r.Spells {
		if row.Origin.Parent != 0 && kinds[row.Origin.Parent] != 3 {
			return fmt.Errorf("sim: saved Spell has missing split parent")
		}
	}
	for _, row := range r.Sacks {
		if row.Origin.Parent != 0 && kinds[row.Origin.Parent] != 4 {
			return fmt.Errorf("sim: saved Sack has missing split parent")
		}
	}
	rootCounts := make(map[SavedObjectID]int)
	for _, id := range r.SackRoots {
		if !sacks[id] {
			return fmt.Errorf("sim: saved Sack root names missing/retired object")
		}
		rootCounts[id]++
	}
	containers := make(map[SavedObjectOwner]bool)
	owned := make(map[SavedObjectID]uint64)
	edges := len(r.SackRoots)
	for _, c := range r.Containers {
		if err := savedOwnerFault(c.Owner); err != nil {
			return err
		}
		if err := savedCoverageFault(c.Coverage); err != nil {
			return err
		}
		if containers[c.Owner] || c.Owner.Kind != SavedOwnerActorPack && c.Owner.Kind != SavedOwnerSack || !c.Present && (len(c.Items) != 0 || c.InsertIndex != 0 || c.Accumulator != 0) || c.Owner.Kind == SavedOwnerSack && (!c.Present || !sacks[c.Owner.Object]) {
			return fmt.Errorf("sim: invalid saved object container")
		}
		containers[c.Owner] = true
		edges += len(c.Items)
		if edges > MaxSavedObjects {
			return fmt.Errorf("sim: too many saved object edges")
		}
		for _, id := range c.Items {
			if id == 0 {
				if c.Owner.Kind == SavedOwnerSack && c.Coverage.Unknown&(SavedUnknownContainerLoad|SavedUnknownMergePolicy) != SavedUnknownContainerLoad|SavedUnknownMergePolicy {
					return fmt.Errorf("sim: unbound saved container slot lacks coverage")
				}
				continue
			}
			row := r.item(id)
			if row == nil || row.Retired {
				return fmt.Errorf("sim: conflicting saved container item %d", id)
			}
			owned[id]++
		}
	}
	for _, row := range r.Sacks {
		if !row.Retired && (rootCounts[row.ID] == 0 || !containers[SavedObjectOwner{Kind: SavedOwnerSack, Object: row.ID}]) {
			return fmt.Errorf("sim: live saved Sack lacks root/container")
		}
	}
	uniqueOwners := make(map[SavedObjectOwner]bool)
	for i, root := range r.ItemRoots {
		row := r.item(root.ID)
		if row == nil || row.Retired || savedOwnerFault(root.Owner) != nil || root.Owner.Kind != SavedOwnerActorWorn && root.Owner.Kind != SavedOwnerSession || uniqueOwners[root.Owner] || i != 0 && savedRootCompare(r.ItemRoots[i-1], root) >= 0 {
			return fmt.Errorf("sim: invalid or duplicate Item root")
		}
		uniqueOwners[root.Owner] = true
		owned[root.ID]++
	}
	edges += len(r.ItemRoots)
	childRefs := make(map[SavedObjectID]int)
	for i, root := range r.BookRoots {
		if i != 0 && r.BookRoots[i-1].Entity >= root.Entity {
			return fmt.Errorf("sim: duplicate or unordered book root")
		}
		for slot, id := range root.Slots {
			if id == 0 {
				continue
			}
			child := r.spell(id)
			if child == nil || child.Retired || child.Value.ID != uint8(slot+1) {
				return fmt.Errorf("sim: book slot names missing or conflicting Spell")
			}
			childRefs[id]++
			edges++
		}
	}
	if edges > MaxSavedObjects {
		return fmt.Errorf("sim: too many saved book edges")
	}
	for _, row := range r.Items {
		if row.Retired != (owned[row.ID]+uint64(row.InFlight) == 0) {
			return fmt.Errorf("sim: Item lifecycle disagrees with its roots")
		}
		edges += int(row.InFlight)
		edges += len(row.Effects)
		if edges > MaxSavedObjects {
			return fmt.Errorf("sim: too many saved Effect edges")
		}
		live := !row.Retired
		for i, id := range row.Effects {
			child := r.effect(id)
			if child == nil || child.Value != row.Value.Effects[i] || live && child.Retired {
				return fmt.Errorf("sim: saved item %d Effect edge/value differs", row.ID)
			}
			if live {
				childRefs[id]++
			}
		}
		if row.Spell != 0 {
			child := r.spell(row.Spell)
			if child == nil || child.Value != row.Value.SourceEquipment.Spell || live && child.Retired {
				return fmt.Errorf("sim: saved item %d Spell edge/value differs", row.ID)
			}
			if live {
				childRefs[row.Spell]++
			}
		}
	}
	for _, row := range r.Effects {
		if row.Retired != (childRefs[row.ID] == 0 && row.ExternalReferences == 0) {
			return fmt.Errorf("sim: orphan/retired saved Effect %d", row.ID)
		}
	}
	for _, row := range r.Spells {
		if row.Retired != (childRefs[row.ID] == 0 && row.ExternalReferences == 0) {
			return fmt.Errorf("sim: orphan/retired saved Spell %d", row.ID)
		}
	}
	return nil
}

func (r *SavedObjects) mutate(fn func(*SavedObjects) error) error {
	if r == nil {
		return fmt.Errorf("sim: saved object registry is absent")
	}
	if err := r.Validate(); err != nil {
		return err
	}
	n := r.Clone()
	if err := fn(n); err != nil {
		return err
	}
	if len(n.ItemRoots) == 0 {
		n.ItemRoots = nil
	}
	if len(n.SackRoots) == 0 {
		n.SackRoots = nil
	}
	if err := n.Validate(); err != nil {
		return err
	}
	*r = *n
	return nil
}

func (r *SavedObjects) mint() (SavedObjectID, error) {
	if r.NextID == ^SavedObjectID(0) {
		return 0, fmt.Errorf("sim: saved object IDs exhausted")
	}
	id := r.NextID
	r.NextID++
	return id, nil
}

func (c *SavedObjectContainer) weight(item ItemStack, count uint32, add bool) {
	if !item.WeightPresent {
		c.Coverage.Unknown |= SavedUnknownContainerLoad
		return
	}
	delta := int32(uint32(int64(item.Weight) * int64(count)))
	if add {
		c.Accumulator += delta
	} else {
		c.Accumulator -= delta
	}
}

func (r *SavedObjects) detach(row *SavedItemObject, from SavedItemLocation, count uint32, whole bool) error {
	if !r.HasLocation(row.ID, from) {
		return fmt.Errorf("sim: take has no exact root")
	}
	switch from.Owner.Kind {
	case SavedOwnerActorPack, SavedOwnerSack:
		c := r.container(from.Owner)
		c.InsertIndex = from.Index
		c.weight(row.Value, count, false)
		if whole {
			c.Items = slices.Delete(c.Items, int(from.Index), int(from.Index)+1)
		}
	case SavedOwnerActorWorn:
		if !whole {
			return fmt.Errorf("sim: cannot split worn item")
		}
		r.ItemRoots = slices.DeleteFunc(r.ItemRoots, func(v SavedItemRoot) bool { return v.ID == row.ID && v.Owner == from.Owner })
	default:
		return fmt.Errorf("sim: take requires a live world root")
	}
	return nil
}

// TakeWhole is identity-preserving. No callback/Token mutation is inferred.
func (r *SavedObjects) TakeWhole(id SavedObjectID, owner SavedObjectOwner, expected ItemStack) (SavedObjectID, error) {
	from, err := r.uniqueLocation(id, owner)
	if err != nil {
		return 0, err
	}
	return r.TakeWholeAt(id, from, expected)
}

func (r *SavedObjects) TakeWholeAt(id SavedObjectID, from SavedItemLocation, expected ItemStack) (SavedObjectID, error) {
	err := r.mutate(func(n *SavedObjects) error {
		row := n.item(id)
		if row == nil || row.Retired || !n.HasLocation(id, from) || !StackStateEqual(row.Value, expected) {
			return fmt.Errorf("sim: stale saved item take")
		}
		if err := n.detach(row, from, row.Value.Count, true); err != nil {
			return err
		}
		row.InFlight++
		return nil
	})
	if err != nil {
		return 0, err
	}
	return id, nil
}

// TakeOne exposes exactly the one-unit split of ITEM-EFFSPLIT-074. It does not
// generalize that virtual into a requested-quantity constructor.
func (r *SavedObjects) TakeOne(id SavedObjectID, owner SavedObjectOwner, expected ItemStack) (SavedObjectID, error) {
	from, err := r.uniqueLocation(id, owner)
	if err != nil {
		return 0, err
	}
	return r.TakeOneAt(id, from, expected)
}

func (r *SavedObjects) TakeOneAt(id SavedObjectID, from SavedItemLocation, expected ItemStack) (SavedObjectID, error) {
	var result SavedObjectID
	err := r.mutate(func(n *SavedObjects) error {
		row := n.item(id)
		if row == nil || row.Retired || !n.HasLocation(id, from) || !StackStateEqual(row.Value, expected) {
			return fmt.Errorf("sim: stale saved item take")
		}
		whole := row.Value.Count == 1
		if err := n.detach(row, from, 1, whole); err != nil {
			return err
		}
		if whole {
			row.InFlight++
			result = id
			return nil
		}
		clone := *row
		clone.Value = row.Value.Clone()
		clone.Effects = nil
		var err error
		clone.ID, err = n.mint()
		if err != nil {
			return err
		}
		clone.Origin = SavedObjectOrigin{Kind: SavedObjectSplit, Parent: id}
		clone.Owner, clone.InFlight = SavedObjectOwner{}, 1
		clone.Value.ObjectID, clone.Value.Count = clone.ID, 1
		clone.F47 = 0
		clone.Coverage.Unknown |= SavedUnknownF47 | SavedUnknownToken
		// Native bookkeeping can copy these values; unknown marks explicitly
		// prevent that copy from claiming original constructor operands.
		clone.Token.Identity = 0
		for _, childID := range row.Effects {
			child := *n.effect(childID)
			child.ID, err = n.mint()
			if err != nil {
				return err
			}
			child.Origin = SavedObjectOrigin{Kind: SavedObjectSplit, Parent: childID}
			child.Token.Identity, child.ExternalReferences = 0, 0
			child.Coverage.Unknown |= SavedUnknownIdentity
			n.Effects = append(n.Effects, child)
			clone.Effects = append(clone.Effects, child.ID)
		}
		if row.Spell != 0 {
			child := *n.spell(row.Spell)
			child.ID, err = n.mint()
			if err != nil {
				return err
			}
			child.Origin = SavedObjectOrigin{Kind: SavedObjectSplit, Parent: row.Spell}
			child.This, child.ExternalReferences = 0, 0
			child.Value = SourceItemSpell{Present: true, ID: child.Value.ID}
			child.Coverage.Unknown |= SavedUnknownIdentity | SavedUnknownSpellInitialization
			clone.Coverage.Unknown |= SavedUnknownSpellInitialization
			clone.Value.SourceEquipment.Spell = child.Value
			clone.Spell = child.ID
			n.Spells = append(n.Spells, child)
		}
		// Count belongs to the node. Every container occurrence observes the
		// same decrement; only the addressed cursor records the take.
		for _, at := range n.Locations(id) {
			if at != from {
				if c := n.container(at.Owner); c != nil {
					c.weight(row.Value, 1, false)
				}
			}
		}
		row.Value.Count--
		n.Items = append(n.Items, clone)
		result = clone.ID
		return nil
	})
	if err != nil {
		return 0, err
	}
	return result, nil
}

func savedOriginalMerge(a, b ItemStack) bool {
	return a.Code == b.Code && (a.Kind == 3 || len(a.Effects) == 0) && (b.Kind == 3 || len(b.Effects) == 0)
}

func savedNativeMerge(a, b ItemStack) bool {
	return CanMergeItemValues(a.Instance(), b.Instance())
}

func (r *SavedObjects) dispose(id SavedObjectID) error {
	row := r.item(id)
	if row == nil || row.InFlight == 0 {
		return fmt.Errorf("sim: disposal requires an in-flight occurrence")
	}
	row.InFlight--
	row.Retired = r.rootCount(id) == 0
	r.RefreshChildLiveness()
	return nil
}

func (r *SavedObjects) Dispose(id SavedObjectID) error {
	return r.mutate(func(n *SavedObjects) error { return n.dispose(id) })
}

// Insert consumes an in-flight handle. Its equality policy is explicit;
// original predicate selection does not narrow native uint32 quantities.
func (r *SavedObjects) Insert(id SavedObjectID, destination SavedObjectOwner, policy SavedMergePolicy) (SavedObjectID, error) {
	return r.insertSelected(id, destination, policy, 0, false)
}

func (r *SavedObjects) insertSelected(id SavedObjectID, destination SavedObjectOwner, policy SavedMergePolicy, selected SavedObjectID, restrict bool) (SavedObjectID, error) {
	retained, _, err := r.insertSelectedAt(id, destination, policy, selected, restrict)
	return retained, err
}

func (r *SavedObjects) insertSelectedAt(id SavedObjectID, destination SavedObjectOwner, policy SavedMergePolicy, selected SavedObjectID, restrict bool) (SavedObjectID, uint32, error) {
	var retained SavedObjectID
	var position uint32
	err := r.mutate(func(n *SavedObjects) error {
		row := n.item(id)
		if row == nil || row.InFlight == 0 || policy > SavedMergeOriginalPredicate {
			return fmt.Errorf("sim: invalid saved item insertion")
		}
		if err := savedOwnerFault(destination); err != nil {
			return err
		}
		if destination.Kind == SavedOwnerActorWorn {
			if policy != SavedMergeNone {
				return fmt.Errorf("sim: equipment insertion cannot merge")
			}
			if err := n.addRoot(id, destination); err != nil {
				return err
			}
			row.InFlight--
			retained = id
			return nil
		}
		c := n.container(destination)
		if c == nil || !c.Present {
			return fmt.Errorf("sim: missing saved destination container")
		}
		for candidateIndex, candidate := range c.Items {
			// A split must not merge back into a parent with multiple roots.
			// Such a merge can undo an aliased Sack drain's
			// progress. This is native alias policy, not an original callback.
			if candidate == id || row.Origin.Kind == SavedObjectSplit && candidate == row.Origin.Parent && n.rootCount(candidate) > 1 || restrict && candidate != selected {
				continue
			}
			if candidate == 0 {
				row.Coverage.Unknown |= SavedUnknownMergePolicy
				continue
			}
			held := n.item(candidate)
			original, native := savedOriginalMerge(held.Value, row.Value), savedNativeMerge(held.Value, row.Value)
			if original != native && policy == SavedMergeNativeRetention || original && policy == SavedMergeNone {
				row.Coverage.Unknown |= SavedUnknownMergePolicy
			}
			merge := policy == SavedMergeNativeRetention && native || policy == SavedMergeOriginalPredicate && original
			if !merge {
				continue
			}
			sum := uint64(held.Value.Count) + uint64(row.Value.Count)
			if uint32(sum) == 0 {
				return fmt.Errorf("sim: saved merge would create an invalid zero native count")
			}
			if err := n.dispose(id); err != nil {
				return err
			}
			for _, at := range n.Locations(held.ID) {
				if other := n.container(at.Owner); other != nil {
					other.weight(row.Value, row.Value.Count, true)
				}
			}
			held.Value.Count = uint32(sum)
			held.Token.T08 |= row.Token.T08
			if row.Coverage.Unknown&SavedUnknownFlags != 0 {
				held.Coverage.Unknown |= SavedUnknownFlags
			}
			held.Coverage.Unknown |= row.Coverage.Unknown & SavedUnknownMergePolicy
			if sum > 65535 {
				held.Coverage.Unknown |= SavedUnknownCountWidth
			}
			retained, position = held.ID, uint32(candidateIndex)
			return nil
		}
		at := min(uint64(c.InsertIndex), uint64(len(c.Items)))
		c.Items = slices.Insert(c.Items, int(at), id)
		c.weight(row.Value, row.Value.Count, true)
		row.InFlight--
		retained, position = id, uint32(at)
		return nil
	})
	if err != nil {
		return 0, 0, err
	}
	return retained, position, nil
}

// absorbValue is insertSelectedAt's merge for an incoming item this registry
// never bound: the live Item id is the retained destination, its count grows
// by the value's count and every container holding it gains that weight
// (ITEM-MERGE-129). flags is the incoming Token +0x08 word, literal one for a
// pickup (ITEM-GROUNDMOVE-130) and zero for a value no Token describes.
func (r *SavedObjects) absorbValue(id SavedObjectID, value ItemStack, flags uint32) error {
	return r.mutate(func(n *SavedObjects) error {
		held := n.item(id)
		if held == nil || held.Retired || value.ObjectID != 0 || value.Count == 0 || !savedNativeMerge(held.Value, value) {
			return fmt.Errorf("sim: invalid saved value merge")
		}
		sum := uint64(held.Value.Count) + uint64(value.Count)
		if uint32(sum) == 0 {
			return fmt.Errorf("sim: saved merge would create an invalid zero native count")
		}
		for _, at := range n.Locations(id) {
			if c := n.container(at.Owner); c != nil {
				c.weight(value, value.Count, true)
			}
		}
		held.Value.Count = uint32(sum)
		held.Token.T08 |= flags
		if !savedOriginalMerge(held.Value, value) {
			held.Coverage.Unknown |= SavedUnknownMergePolicy
		}
		if sum > 65535 {
			held.Coverage.Unknown |= SavedUnknownCountWidth
		}
		return nil
	})
}

// StampPickup precedes the drain: even a later merge sees literal one, not
// the pre-pickup incoming flags (ITEM-GROUNDMOVE-130).
func (r *SavedObjects) StampPickup(sack SavedObjectID) error {
	return r.mutate(func(n *SavedObjects) error {
		c := n.container(SavedObjectOwner{Kind: SavedOwnerSack, Object: sack})
		if c == nil || !c.Present {
			return fmt.Errorf("sim: pickup names no live saved Sack")
		}
		for _, id := range c.Items {
			if id == 0 {
				continue
			}
			item := n.item(id)
			item.Token.T08 = 1
			item.Coverage.Unknown &^= SavedUnknownFlags
		}
		return nil
	})
}

// SetSackGold changes the named live operand only. It does not repair a
// retained Token.T1C cache or infer a value-recompute event.
func (r *SavedObjects) SetSackGold(id SavedObjectID, expectedGold, gold uint32) error {
	return r.mutate(func(n *SavedObjects) error {
		row := n.sack(id)
		if row == nil || row.Retired || row.Gold != expectedGold {
			return fmt.Errorf("sim: stale saved Sack gold")
		}
		row.Gold = gold
		return nil
	})
}

// RetireSack finishes an explicit empty-container drain. Cell detachment and
// any purse credit belong to the enclosing World transaction, not this core.
func (r *SavedObjects) RetireSack(id SavedObjectID, expectedGold uint32) error {
	return r.mutate(func(n *SavedObjects) error {
		row := n.sack(id)
		owner := SavedObjectOwner{Kind: SavedOwnerSack, Object: id}
		c := n.container(owner)
		if row == nil || row.Retired || row.Gold != expectedGold || c == nil || len(c.Items) != 0 {
			return fmt.Errorf("sim: saved Sack retirement requires an expected empty Sack")
		}
		n.SackRoots = slices.DeleteFunc(n.SackRoots, func(v SavedObjectID) bool { return v == id })
		n.Containers = slices.DeleteFunc(n.Containers, func(v SavedObjectContainer) bool { return v.Owner == owner })
		row.Retired, row.Gold = true, 0
		return nil
	})
}

func (r *SavedObjects) ExportExternal(id SavedObjectID, handle uint64) error {
	return r.mutate(func(n *SavedObjects) error {
		row := n.item(id)
		if row == nil || row.InFlight == 0 || handle == 0 {
			return fmt.Errorf("sim: invalid saved external export")
		}
		if err := n.addRoot(id, SavedObjectOwner{Kind: SavedOwnerSession, SessionHandle: handle}); err != nil {
			return err
		}
		row.InFlight--
		return nil
	})
}

func (r *SavedObjects) ImportExternal(handle uint64, expected ItemStack) (SavedObjectID, error) {
	var result SavedObjectID
	err := r.mutate(func(n *SavedObjects) error {
		for i, root := range n.ItemRoots {
			if root.Owner.Kind != SavedOwnerSession || root.Owner.SessionHandle != handle || handle == 0 {
				continue
			}
			row := n.item(root.ID)
			if !StackStateEqual(row.Value, expected) {
				return fmt.Errorf("sim: stale saved external import")
			}
			row.InFlight++
			n.ItemRoots = slices.Delete(n.ItemRoots, i, i+1)
			result = row.ID
			return nil
		}
		return fmt.Errorf("sim: saved session handle missing")
	})
	return result, err
}

func (r *SavedObjects) ValidateNoInFlight() error {
	if err := r.Validate(); err != nil {
		return err
	}
	if r != nil {
		for _, row := range r.Items {
			if row.InFlight != 0 {
				return fmt.Errorf("sim: unfinished saved Item transaction")
			}
		}
	}
	return nil
}

// CompareLive accounts for locations, so repeated references verify one node
// without suppressing a missing occurrence or accepting a conflicting value.
func (r *SavedObjects) CompareLive(bindings []SavedObjectBinding) error {
	if err := r.ValidateNoInFlight(); err != nil {
		return err
	}
	seen := make(map[SavedItemLocation]bool)
	for _, b := range bindings {
		if b.ID == 0 && b.Value.ObjectID == 0 {
			continue
		}
		at := SavedItemLocation{b.Owner, b.Index}
		row := r.item(b.ID)
		if row == nil || row.Retired || seen[at] || !r.HasLocation(b.ID, at) || !StackStateEqual(row.Value, b.Value) {
			return fmt.Errorf("sim: saved/live item %d disagrees", b.ID)
		}
		seen[at] = true
	}
	if r != nil {
		for _, row := range r.Items {
			for _, at := range r.Locations(row.ID) {
				if !seen[at] {
					return fmt.Errorf("sim: saved Item root lacks live binding")
				}
			}
		}
	}
	return nil
}

func (r *SavedObjects) ValidateExternal(pairs []SavedExternalItem) error {
	if err := r.ValidateNoInFlight(); err != nil {
		return err
	}
	seen := make(map[uint64]bool)
	for _, p := range pairs {
		row := r.item(p.ID)
		if row == nil || p.Handle == 0 || seen[p.Handle] || !r.HasRoot(p.ID, SavedObjectOwner{Kind: SavedOwnerSession, SessionHandle: p.Handle}) || !StackStateEqual(row.Value, p.Value) {
			return fmt.Errorf("sim: saved external pair differs")
		}
		seen[p.Handle] = true
	}
	if r != nil {
		for _, root := range r.ItemRoots {
			if root.Owner.Kind == SavedOwnerSession && !seen[root.Owner.SessionHandle] {
				return fmt.Errorf("sim: saved session handle lacks paired value")
			}
		}
	}
	return nil
}
