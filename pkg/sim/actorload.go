package sim

import "fmt"

// ActorLoad is the actor's live weight bookkeeping. Load and Capacity remain
// Entity fields; Items remain World.carried. Present selects original signed
// word arithmetic, not whether the current load happens to be nonzero.
// An absent original container is distinct from a present empty container.
type ActorLoad struct {
	Present          bool
	OwnWeight        int16
	ContainerPresent bool
	InsertIndex      uint32
	Accumulator      int32
	Source           SourceActor
}

func (a ActorLoad) Validate() error {
	if err := a.Source.Validate(); err != nil {
		return err
	}
	if !a.Present && a != (ActorLoad{}) {
		return fmt.Errorf("sim: absent actor load has residue")
	}
	if a.Present && !a.ContainerPresent && (a.InsertIndex != 0 || a.Accumulator != 0) {
		return fmt.Errorf("sim: absent actor container has residue")
	}
	return nil
}

// CurrentLoad refreshes the signed word without inferring that a saved current
// value was already refreshed. ITEM-LOAD-005 and SAV-CITYSALE-513.
//
// SAV-793 (High for width/signedness/boundary) adds the integer facts this
// already matches: the 64000 compare is signed (a Go int32 Accumulator
// compared with >=, not cast unsigned first), the halved sum truncates into
// OwnWeight's own int16 width before it is added, and the sum sign-extends
// back out through int32(v) exactly as the original's own MOVSX does. A
// caller must not widen this to a 32-bit accumulate: SAV-793 executed that
// alternative and it predicts a different stored value at the boundary.
func (a ActorLoad) CurrentLoad() int32 {
	v := a.OwnWeight
	if a.ContainerPresent {
		if a.Accumulator >= 64000 {
			return 32000
		}
		v += int16(a.Accumulator / 2)
	}
	return int32(v)
}

type loadMutation struct{ worn, carried int32 }

// ActorLoadSnapshot transfers live state through a city boundary, where there
// is no World/Entity. It is mutable city state, not an immutable source DTO.
type ActorLoadSnapshot struct {
	Inventory             ActorLoad
	Load, Capacity, Speed int32
	Movement              HumanMovement
	// SpeedModifier is a native Human's Entity.SpeedModifier. Older native
	// snapshots omit it and read zero, so their whole Speed is the base.
	SpeedModifier int32 `json:",omitempty"`
	// Independent source regeneration residues survive the city boundary.
	// Older native snapshots omit both fields and deterministically read zero.
	HealthHundredths, ManaHundredths uint8
}

func (s ActorLoadSnapshot) Validate() error {
	if err := s.Inventory.Validate(); err != nil {
		return err
	}
	if !s.Inventory.Present {
		return fmt.Errorf("sim: live actor load lacks bookkeeping")
	}
	if s.Load != int32(int16(s.Load)) || s.Capacity != int32(int16(s.Capacity)) {
		return fmt.Errorf("sim: original actor load/capacity exceeds signed word")
	}
	if s.Movement.Present && (s.Movement.Load != s.Load || s.Movement.Capacity != s.Capacity || s.Movement.NativeSpeed != s.Speed) {
		return fmt.Errorf("sim: current load movement disagrees with actor")
	}
	if !s.Movement.Present && s.Movement != (HumanMovement{}) {
		return fmt.Errorf("sim: absent current movement has residue")
	}
	if s.Inventory.Source.Class != 0 && s.SpeedModifier != 0 {
		return fmt.Errorf("sim: source actor load carries a native speed modifier")
	}
	if s.Inventory.Source.Class == 0 && (s.HealthHundredths > 99 || s.ManaHundredths > 99) {
		return fmt.Errorf("sim: native actor load has invalid regeneration residue")
	}
	return nil
}

// DisplaySpeed is the same own statistic read by a live actor's card. It is
// deliberately not a terrain-adjusted or group-overridden movement rate.
func (s ActorLoadSnapshot) DisplaySpeed() int32 {
	if s.Movement.Present {
		return int32(s.Movement.RawSpeed)
	}
	word, _ := humanSpeedWord(s.Speed, s.SpeedModifier, s.Load, s.Capacity)
	return word
}

func (e Entity) CurrentActorLoad() *ActorLoadSnapshot {
	if !e.ActorLoad.Present {
		return nil
	}
	a := e.ActorLoad
	a.Source = e.SourceNow()
	return &ActorLoadSnapshot{Inventory: a, Load: e.Load, Capacity: e.Capacity, Speed: e.Speed, Movement: e.HumanMovement, SpeedModifier: e.SpeedModifier,
		HealthHundredths: e.HealthHundredths, ManaHundredths: e.ManaHundredths}
}

func (s ActorLoadSnapshot) apply(e *Entity) {
	e.ActorLoad, e.Load, e.Capacity, e.Speed, e.HumanMovement = s.Inventory, s.Load, s.Capacity, s.Speed, s.Movement
	e.SpeedModifier = s.SpeedModifier
	if s.Inventory.Source.Class != 0 {
		e.NativeBasis = NativeActorBasis{}
	}
	e.HealthHundredths, e.ManaHundredths = s.HealthHundredths, s.ManaHundredths
}

func (w *World) RestoreActorLoad(id EntityID, s ActorLoadSnapshot) error {
	i := indexOfEntity(w.entities, id)
	if i < 0 {
		return fmt.Errorf("sim: actor load names missing entity %d", id)
	}
	if err := s.Validate(); err != nil {
		return err
	}
	if !s.Inventory.ContainerPresent && len(w.carried[i]) != 0 {
		return fmt.Errorf("sim: absent actor container carries items")
	}
	if w.savedObjects != nil {
		if c := w.savedObjects.container(w.savedPackOwner(i)); c != nil &&
			(c.Present != s.Inventory.ContainerPresent || c.InsertIndex != s.Inventory.InsertIndex || c.Accumulator != s.Inventory.Accumulator) {
			return fmt.Errorf("sim: actor load restore differs from bound container bookkeeping")
		}
	}
	hp, maxHP := w.entities[i].HP, w.entities[i].MaxHP
	s.apply(&w.entities[i])
	if s.Inventory.Source.Class != 0 {
		w.publishSource(i, s.Inventory.Source)
		// Publishing decodes both words as signed 16-bit values. A current
		// value that is wider than its word keeps its high bits while the
		// source still carries the same words.
		if e, stats := &w.entities[i], s.Inventory.Source.Stats; uint16(hp) == stats[8] && uint16(maxHP) == stats[9] {
			e.HP, e.MaxHP = hp, maxHP
		}
	}
	return nil
}

func (w *World) beginLoadMutation(i int) loadMutation {
	return loadMutation{w.wornWeight(i), w.containerWeight(i)}
}

// finishLoadMutation applies this operation's weight delta to the stored
// accumulators. A retained discrepancy is not erased by an unrelated item.
func (w *World) finishLoadMutation(i int, before loadMutation) bool {
	defer w.syncSavedPack(i)
	e := &w.entities[i]
	if e.ActorLoad.Present {
		oldLoad := e.Load
		_, retained := e.RetainedHumanSpeed()
		e.ActorLoad.OwnWeight += int16(w.wornWeight(i) - before.worn)
		e.ActorLoad.Accumulator += w.containerWeight(i) - before.carried
		e.Load = e.ActorLoad.CurrentLoad()
		if e.ActorLoad.Source.Class != 0 {
			e.ActorLoad.Source = e.SourceNow()
		}
		if retained && e.Capacity != 0 && oldLoad/e.Capacity == e.Load/e.Capacity {
			e.HumanMovement.Load = e.Load
		} else if e.ActorLoad.Source.Class != 0 {
			e.HumanMovement.Load = e.Load
			if !w.deriveSource(i) {
				return false
			}
		} else {
			e.HumanMovement = HumanMovement{}
		}
		e.deriveNativeHumanSpeed()
		return true
	}
	w.recomputeLoad(i)
	w.entities[i].deriveNativeHumanSpeed()
	return true
}

// addCarried preserves existing element boundaries and inserts a new element
// at the container's stored next-index. A merge touches only its destination.
// The existing instance-retention equality policy remains in force.
func (w *World) addCarried(i int, item ItemStack, otherActive ...int) bool {
	item = w.joinForm(i, item)
	if w.savedObjects != nil {
		if k := w.firstMergeableHeld(i, item); k >= 0 && (item.ObjectID == 0 || w.carried[i][k].ObjectID == 0) {
			n := w.sourceMutationCopy(i)
			if !n.mergeIntoHeld(i, k, item, 0, otherActive...) {
				return false
			}
			copy(w.entities, n.entities)
			n.entities = w.entities
			*w = n
			return true
		}
	}
	if item.ObjectID == 0 && w.savedObjects != nil {
		n := w.sourceMutationCopy(i)
		if constructed, ok := n.constructSavedAcquisition(item); ok {
			if !n.putCarriedObject(i, constructed, otherActive...) {
				return false
			}
			// Node merges can update another actor's view. Publish the whole
			// candidate while keeping enclosing source-command actor pointers.
			copy(w.entities, n.entities)
			n.entities = w.entities
			*w = n
			return true
		}
	}
	if item.ObjectID != 0 {
		return w.putCarriedObject(i, item, otherActive...)
	}
	defer w.syncSavedPack(i)
	a := &w.entities[i].ActorLoad
	if a.Present && !a.ContainerPresent {
		return false
	}
	if k := w.firstMergeableHeld(i, item); k >= 0 && w.carried[i][k].ObjectID == 0 {
		w.carried[i][k].Count += item.Count
		return true
	}
	index := len(w.carried[i])
	if a.Present {
		index = int(min(a.InsertIndex, uint32(index)))
	}
	w.carried[i] = append(w.carried[i], ItemStack{})
	copy(w.carried[i][index+1:], w.carried[i][index:])
	w.carried[i][index] = item.Clone()
	return true
}

// firstMergeableHeld is the element of actor i's container that item joins,
// or -1: the first one CanMergeItemValues accepts, whether or not either side
// is a bound saved object (ITEM-MERGE-129 retains that destination).
func (w *World) firstMergeableHeld(i int, item ItemStack) int {
	var uses map[uint32]int
	if item.NativeRecord != nil {
		uses = w.nativeItemIdentityUses()
	}
	for k, held := range w.carried[i] {
		if held.NativeRecord != nil && uses[held.NativeRecord.Token.Identity] > 1 {
			continue
		}
		if held.Code != 0 && held.Count != 0 && CanMergeItemValues(held.Instance(), item.Instance()) {
			return k
		}
	}
	return -1
}

// joinForm is an unbound item written in the form of the first element of
// actor i's container that CanMergeItemValues accepts once so written
// (JoinForm), so the ordinary merge takes it, or else item itself. The
// world's declared constructors build a plain value.
func (w *World) joinForm(i int, item ItemStack) ItemStack {
	if item.ObjectID != 0 {
		return item
	}
	for _, held := range w.carried[i] {
		if held.Code == 0 || held.Count == 0 {
			continue
		}
		if v, ok := JoinForm(held.Instance(), item.Instance(), w.sourceConstructItem); ok && CanMergeItemValues(held.Instance(), v) {
			return StackItem(v, item.Count)
		}
	}
	return item
}

// mergeIntoHeld merges item into element k when at least one side has no
// saved object, which the registry's node insertion cannot. As in
// ITEM-MERGE-129 the destination is retained and only the incoming item is
// retired: a plain destination grows and stays unbound, and a bound
// destination absorbs the plain units with flags as their +0x08 word.
func (w *World) mergeIntoHeld(i, k int, item ItemStack, flags uint32, otherActive ...int) bool {
	held := w.carried[i][k].Clone()
	switch {
	case held.ObjectID == 0:
		if item.ObjectID != 0 && w.savedObjects.Dispose(item.ObjectID) != nil {
			return false
		}
		w.carried[i][k].Count += item.Count
		w.syncSavedPack(i)
		return true
	case item.ObjectID == 0:
		if w.savedObjects.absorbValue(held.ObjectID, item, flags) != nil {
			return false
		}
		return w.syncSavedItemViews(held.ObjectID, held, i, otherActive...)
	}
	return false
}

func (w *World) hasActorContainer(i int) bool {
	a := w.entities[i].ActorLoad
	return !a.Present || a.ContainerPresent
}

// Explicit occurrence owners use this insertion when scalar equality must
// not merge distinct roots. The current container cursor chooses the position.
func (w *World) addCarriedUnmerged(i int, item ItemStack) (int, bool) {
	if !w.hasActorContainer(i) {
		return 0, false
	}
	index := len(w.carried[i])
	if a := w.entities[i].ActorLoad; a.Present {
		index = int(min(a.InsertIndex, uint32(index)))
	}
	if item.ObjectID != 0 {
		return index, w.putCarriedObjectAt(i, item, index, 0)
	}
	w.carried[i] = append(w.carried[i], ItemStack{})
	copy(w.carried[i][index+1:], w.carried[i][index:])
	w.carried[i][index] = item.Clone()
	w.syncSavedPack(i)
	return index, true
}
