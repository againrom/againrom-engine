package sim

import (
	"encoding/binary"
	"fmt"
	"slices"
	"sort"
)

// ActionContinuations is a typed boundary for unfinished native actions. It
// contains neither world bytes nor a second entity population. The receiving
// world must already own the actors, terrain, item registry and spell rules.
// Entity and object handles are local addresses; the SAV adapter translates
// them through explicit document bindings before calling RestoreActions.
type ActionContinuations struct {
	SessionVersion uint32
	ActorTraversal *[]EntityID `json:",omitempty"`
	Actors         []ActorContinuation
	Books          []BookContinuation
	Scrolls        []ScrollCast
	Scripts        []ScriptCast
	Reservations   ActionReservations
	Healing        []ActionHealingPolicy
	EffectCasters  []AttachedEffectCaster
	StructureUses  []StructureUse `json:",omitempty"`
}

type AttachedEffectCaster struct {
	Target, Caster EntityID
	Spell          uint16
	HasCaster      bool
}
type ActionHealingPolicy struct {
	Present bool
	Percent uint32
}

type ActorContinuation struct {
	Entity                                                      EntityID
	X, Y, TargetX, TargetY                                      int32
	HasTarget, OffMap                                           bool
	Stall                                                       uint8
	Transit, TransitTotal                                       uint16
	Stride                                                      NativeStride
	GroupSpeed, Facing, DesiredFacing, TurnRemaining, TurnTotal uint8
	ImportedMotion                                              bool
	MotionIssue                                                 string
	Route                                                       [][2]int32
	ActorState                                                  uint8
	Retreat                                                     *RetreatContinuation `json:",omitempty"`
	AttackTarget                                                EntityID
	AttackTargetKind                                            AttackTargetKind
	HasAttackTarget                                             bool
	PendingAttackTarget                                         EntityID         `json:",omitempty"`
	PendingAttackTargetKind                                     AttackTargetKind `json:",omitempty"`
	HasPendingAttackTarget                                      bool             `json:",omitempty"`
	PendingOrder                                                PendingOrder     `json:",omitempty"`
	AcquirePursuit                                              bool             `json:",omitempty"`
	PursuitIdle                                                 bool             `json:",omitempty"`
	AttackPhase                                                 AttackPhase
	AttackCountdown                                             int32
	CastWait                                                    uint8
	AutoSpell                                                   uint16
	ActionClock                                                 ActionClock
	Withdraw, Wimpy                                             int32
	PatrolHeadX, PatrolHeadY, PatrolTailX, PatrolTailY          int32
	PatrolLeg                                                   uint8
	PostX, PostY                                                int32
	EscortTarget                                                EntityID
	HasEscortTarget                                             bool
	EscortRange                                                 uint8
	CommandGroup                                                uint32
	Order                                                       *ActionOrderContinuation
	ProfileBasis                                                *CurrentProfileBasis
	Current                                                     *ActorCurrentContinuation
}
type ActorCurrentContinuation struct {
	Class             *int32 `json:",omitempty"`
	RotationSpeed     int32
	WeaponSpell       uint16
	WeaponSpellLevel  int32
	WeaponSpellSource WeaponSpellSource
	SpellFX           uint16
	SpellFXSpell      uint8
}
type ActionOrderContinuation struct {
	State                               uint32
	Authored                            bool
	RepairStage, Inner, Progress, Retry uint8
	Ordinal                             *uint32 `json:",omitempty"`
	AbsentSpellKey                      *uint32 `json:",omitempty"`
}

// BookPhaseApproach is the BookContinuation phase of a creature's retained
// cast order that is out of range and walking toward its victim.
const BookPhaseApproach = uint8(bookApproach)

type BookContinuation struct {
	Caster, Target                   EntityID
	Spell                            uint16
	X, Y                             int32
	Remaining, Phase, Progress       uint8
	Complete, AtCell, Retained, Paid bool
}

// Only the objects exclusively held by an unfinished action are carried here.
// Children also reachable from ordinary holdings retain their shared identity
// through the adapter's binding map, rather than being copied on LOAD.
type ActionReservations struct {
	ItemRoots []SavedItemRoot
	Items     []SavedItemObject
	Effects   []SavedEffectObject
	Spells    []SavedSpellObject
}

func (w *World) Actions() ActionContinuations {
	a := ActionContinuations{SessionVersion: 2, Scrolls: w.ScrollCasts(), Scripts: w.ScriptCasts(), EffectCasters: make([]AttachedEffectCaster, 0, len(w.attached)), StructureUses: w.StructureUses()}
	traversal := w.ActorTraversal()
	if traversal == nil {
		traversal = []EntityID{}
	}
	a.ActorTraversal = &traversal
	ordinals := map[EntityID]uint32{}
	if w.savedGroups != nil {
		for i, row := range w.savedGroups.Orders {
			ordinals[row.Entity] = uint32(i)
		}
	}
	for _, e := range w.attached {
		if indexOfEntity(w.entities, e.Target) < 0 {
			continue
		}
		a.EffectCasters = append(a.EffectCasters, AttachedEffectCaster{e.Target, e.Caster, e.Spell, e.HasCaster})
	}
	for _, p := range w.autoHealing {
		a.Healing = append(a.Healing, ActionHealingPolicy{p.Present, p.Percent})
	}
	for _, e := range w.entities {
		var retreat *RetreatContinuation
		if e.Retreat != (RetreatContinuation{}) {
			control := e.Retreat
			retreat = &control
		}
		basis, class := e.CurrentProfileBasis, e.Class
		m := w.motionFor(e.ID)
		issue := ""
		if m != nil {
			issue = m.Issue
		}
		var order *ActionOrderContinuation
		if o := w.savedOrder(e.ID); o != nil {
			ordinal := ordinals[e.ID]
			order = &ActionOrderContinuation{State: o.State, Authored: o.Authored, RepairStage: o.RepairStage, Inner: o.Raw[8], Progress: o.Raw[9], Retry: o.Raw[0x15], Ordinal: &ordinal}
			_, casting := w.bookCastIndex(e.ID)
			if (casting || e.PendingOrder.Kind == PendingActorCast || e.PendingOrder.Kind == PendingCellCast) && binary.LittleEndian.Uint32(o.Raw[0x30:]) == 0 {
				absent := uint32(0)
				order.AbsentSpellKey = &absent
			}
		}
		a.Actors = append(a.Actors, ActorContinuation{
			Entity: e.ID, X: e.X, Y: e.Y, TargetX: e.TargetX, TargetY: e.TargetY,
			HasTarget: e.HasTarget, OffMap: e.OffMap, Stall: e.Stall, Retreat: retreat,
			Transit: e.Transit, TransitTotal: e.TransitTotal, Stride: e.Stride,
			GroupSpeed: e.GroupSpeed, Facing: e.Facing, DesiredFacing: e.DesiredFacing,
			TurnRemaining: e.TurnRemaining, TurnTotal: e.TurnTotal, ImportedMotion: m != nil && m.Current, MotionIssue: issue,
			Route: w.Route(e.ID), ActorState: e.ActorState, AttackTarget: e.AttackTarget,
			AttackTargetKind: e.AttackTargetKind, HasAttackTarget: e.HasAttackTarget,
			PendingAttackTarget: e.PendingAttackTarget, PendingAttackTargetKind: e.PendingAttackTargetKind, HasPendingAttackTarget: e.HasPendingAttackTarget,
			PendingOrder:   e.PendingOrder,
			AcquirePursuit: e.AcquirePursuit, PursuitIdle: e.PursuitIdle,
			AttackPhase: e.AttackPhase, AttackCountdown: e.AttackCountdown,
			CastWait: e.CastWait, AutoSpell: e.AutoSpell, ActionClock: e.ActionClock,
			Withdraw: e.Withdraw, Wimpy: e.Wimpy, PatrolHeadX: e.PatrolHeadX, PatrolHeadY: e.PatrolHeadY,
			PatrolTailX: e.PatrolTailX, PatrolTailY: e.PatrolTailY, PatrolLeg: e.PatrolLeg,
			PostX: e.PostX, PostY: e.PostY, EscortTarget: e.EscortTarget,
			HasEscortTarget: e.HasEscortTarget, EscortRange: e.EscortRange, CommandGroup: e.CommandGroup, Order: order, ProfileBasis: &basis,
			Current: &ActorCurrentContinuation{Class: &class, RotationSpeed: e.RotationSpeed, WeaponSpell: e.WeaponSpell, WeaponSpellLevel: e.WeaponSpellLevel, WeaponSpellSource: e.WeaponSpellSource, SpellFX: e.SpellFX, SpellFXSpell: e.SpellFXSpell},
		})
	}
	a.SessionVersion = 2
	for _, c := range w.bookCasts {
		a.Books = append(a.Books, BookContinuation{c.Caster, c.Target, c.Spell, c.X, c.Y, c.Remaining, uint8(c.Phase), c.Progress, c.Complete, c.AtCell, c.Retained, c.Paid})
	}
	if r := w.SavedObjects(); r != nil {
		children := map[SavedObjectID]bool{}
		for _, row := range r.Items {
			session := false
			for _, at := range r.Locations(row.ID) {
				if at.Owner.Kind == SavedOwnerSession {
					session = true
					a.Reservations.ItemRoots = append(a.Reservations.ItemRoots, SavedItemRoot{row.ID, at.Owner})
				}
			}
			if !session {
				continue
			}
			a.Reservations.Items = append(a.Reservations.Items, row)
			for _, id := range row.Effects {
				children[id] = true
			}
			children[row.Spell] = true
		}
		for _, row := range r.Effects {
			if children[row.ID] {
				a.Reservations.Effects = append(a.Reservations.Effects, row)
			}
		}
		for _, row := range r.Spells {
			if children[row.ID] {
				a.Reservations.Spells = append(a.Reservations.Spells, row)
			}
		}
	}
	return a
}

// RemapActors translates every typed endpoint, including removed targets. The
// boolean argument distinguishes the structure namespace from actor IDs.
func (a *ActionContinuations) RemapActors(ref func(EntityID, bool) (EntityID, error)) error {
	apply := func(id *EntityID, structure bool) error {
		n, err := ref(*id, structure)
		if err == nil {
			*id = n
		}
		return err
	}
	if a.ActorTraversal != nil {
		for i := range *a.ActorTraversal {
			if err := apply(&(*a.ActorTraversal)[i], false); err != nil {
				return err
			}
		}
	}
	for i := range a.Actors {
		v := &a.Actors[i]
		if err := apply(&v.Entity, false); err != nil {
			return err
		}
		if v.HasAttackTarget {
			if err := apply(&v.AttackTarget, v.AttackTargetKind == AttackTargetStructure); err != nil {
				return err
			}
		}
		if v.HasPendingAttackTarget {
			if err := apply(&v.PendingAttackTarget, v.PendingAttackTargetKind == AttackTargetStructure); err != nil {
				return err
			}
		}
		if v.PendingOrder.Kind == PendingActorCast {
			if err := apply(&v.PendingOrder.Target, false); err != nil {
				return err
			}
		}
		if v.HasEscortTarget {
			if err := apply(&v.EscortTarget, false); err != nil {
				return err
			}
		}
	}
	for i := range a.Books {
		c := &a.Books[i]
		if err := apply(&c.Caster, false); err != nil {
			return err
		}
		if !c.AtCell {
			if err := apply(&c.Target, false); err != nil {
				return err
			}
		}
	}
	for i := range a.EffectCasters {
		e := &a.EffectCasters[i]
		if err := apply(&e.Target, false); err != nil {
			return err
		}
		if e.HasCaster {
			if err := apply(&e.Caster, false); err != nil {
				return err
			}
		}
	}
	for i := range a.Scrolls {
		c := &a.Scrolls[i]
		if err := apply(&c.Caster, false); err != nil {
			return err
		}
		if !c.AtCell {
			if err := apply(&c.Target, false); err != nil {
				return err
			}
		}
	}
	for i := range a.Scripts {
		if a.Scripts[i].AtUnit {
			if err := apply(&a.Scripts[i].Target, false); err != nil {
				return err
			}
		}
	}
	for i := range a.StructureUses {
		use := &a.StructureUses[i]
		if err := apply(&use.Entity, false); err != nil {
			return err
		}
		structure := EntityID(use.Structure)
		if err := apply(&structure, true); err != nil {
			return err
		}
		use.Structure = StructureID(structure)
	}
	return nil
}

// RestoreActions installs a complete action population atomically, after the
// SAV actors/objects have been imported and before a gameplay tick. Validation
// uses the same native invariants as an ordinary current World, not a LOAD-time
// action completion or a recomputation from speed, mana or donor operands.
func (w *World) RestoreActions(a ActionContinuations, objects map[SavedObjectID]SavedObjectID) error {
	if len(a.Actors) != len(w.entities) {
		var got, want []EntityID
		for _, e := range a.Actors {
			got = append(got, e.Entity)
		}
		for _, e := range w.entities {
			want = append(want, e.ID)
		}
		return fmt.Errorf("sim: incomplete current action actor population: %v, want %v", got, want)
	}
	n := *w
	if a.EffectCasters != nil {
		if len(a.EffectCasters) != len(w.attached) {
			return fmt.Errorf("sim: incomplete current effect caster population")
		}
		n.attached = slices.Clone(w.attached)
		seen := map[[2]uint32]bool{}
		for _, row := range a.EffectCasters {
			key := [2]uint32{uint32(row.Target), uint32(row.Spell)}
			if seen[key] {
				return fmt.Errorf("sim: invalid current effect caster")
			}
			seen[key] = true
			found := false
			for i := range n.attached {
				e := &n.attached[i]
				if e.Target == row.Target && e.Spell == row.Spell {
					e.Caster, e.HasCaster = row.Caster, row.HasCaster
					found = true
				}
			}
			if !found {
				return fmt.Errorf("sim: current effect caster lost its attachment")
			}
		}
	}
	if len(a.Healing) != len(n.autoHealing) {
		return fmt.Errorf("sim: incomplete current autohealing policy")
	}
	for i, p := range a.Healing {
		if !p.Present && p.Percent != 0 {
			return fmt.Errorf("sim: absent current autohealing policy has a value")
		}
		n.autoHealing[i] = autoHealingPolicy{p.Present, p.Percent}
	}
	n.entities, n.routes = slices.Clone(w.entities), slices.Clone(w.routes)
	n.savedMotion = cloneActorMotions(w.savedMotion)
	n.savedCellRecords = slices.Clone(w.savedCellRecords)
	n.savedGroups = cloneSavedGroups(w.savedGroups)
	n.savedObjects = w.savedObjects.Clone()
	seen := map[EntityID]bool{}
	for _, v := range a.Actors {
		i := indexOfEntity(n.entities, v.Entity)
		if i < 0 || seen[v.Entity] {
			return fmt.Errorf("sim: invalid current action actor binding")
		}
		seen[v.Entity] = true
		e := &n.entities[i]
		if v.ProfileBasis != nil {
			if *v.ProfileBasis > ProfileNativeRetired {
				return fmt.Errorf("sim: invalid current profile basis")
			}
			e.CurrentProfileBasis = *v.ProfileBasis
		}
		if v.Current != nil {
			c := v.Current
			if c.Class != nil {
				e.Class = *c.Class
			}
			e.RotationSpeed, e.WeaponSpell, e.WeaponSpellLevel, e.WeaponSpellSource = c.RotationSpeed, c.WeaponSpell, c.WeaponSpellLevel, c.WeaponSpellSource
			e.SpellFX, e.SpellFXSpell = c.SpellFX, c.SpellFXSpell
		}
		e.X, e.Y, e.TargetX, e.TargetY = v.X, v.Y, v.TargetX, v.TargetY
		e.HasTarget, e.OffMap, e.Stall = v.HasTarget, v.OffMap, v.Stall
		e.Transit, e.TransitTotal, e.Stride = v.Transit, v.TransitTotal, v.Stride
		e.GroupSpeed, e.Facing, e.DesiredFacing = v.GroupSpeed, v.Facing, v.DesiredFacing
		e.TurnRemaining, e.TurnTotal = v.TurnRemaining, v.TurnTotal
		e.ActorState, e.AttackTarget, e.AttackTargetKind, e.HasAttackTarget = v.ActorState, v.AttackTarget, v.AttackTargetKind, v.HasAttackTarget
		e.Retreat = RetreatContinuation{}
		if v.Retreat != nil {
			e.Retreat = *v.Retreat
		}
		e.PendingAttackTarget, e.PendingAttackTargetKind, e.HasPendingAttackTarget = v.PendingAttackTarget, v.PendingAttackTargetKind, v.HasPendingAttackTarget
		e.PendingOrder = v.PendingOrder
		e.AcquirePursuit, e.PursuitIdle = v.AcquirePursuit, v.PursuitIdle
		e.AttackPhase, e.AttackCountdown, e.CastWait, e.AutoSpell = v.AttackPhase, v.AttackCountdown, v.CastWait, v.AutoSpell
		e.ActionClock, e.Withdraw, e.Wimpy = v.ActionClock, v.Withdraw, v.Wimpy
		e.PatrolHeadX, e.PatrolHeadY, e.PatrolTailX, e.PatrolTailY, e.PatrolLeg = v.PatrolHeadX, v.PatrolHeadY, v.PatrolTailX, v.PatrolTailY, v.PatrolLeg
		e.PostX, e.PostY, e.EscortTarget, e.HasEscortTarget, e.EscortRange, e.CommandGroup = v.PostX, v.PostY, v.EscortTarget, v.HasEscortTarget, v.EscortRange, v.CommandGroup
		if v.Order != nil {
			if o := n.savedOrder(e.ID); o != nil {
				if v.Order.Authored {
					o.authorNative()
				}
				o.State, o.Authored, o.RepairStage = v.Order.State, v.Order.Authored, v.Order.RepairStage
				o.Raw[8], o.Raw[9], o.Raw[0x15] = v.Order.Inner, v.Order.Progress, v.Order.Retry
				if anchor := v.Order.AbsentSpellKey; anchor != nil && *anchor == binary.LittleEndian.Uint32(o.Raw[0x30:]) {
					binary.LittleEndian.PutUint32(o.Raw[0x30:], 0)
				}
			}
		} else if n.savedGroups != nil {
			n.savedGroups.Orders = slices.DeleteFunc(n.savedGroups.Orders, func(o SavedActorOrder) bool { return o.Entity == e.ID })
		}
		n.routes[i] = nil
		for _, p := range v.Route {
			n.routes[i] = append(n.routes[i], cell{p[0], p[1]})
		}
		if m := n.motionFor(e.ID); m != nil {
			if v.ImportedMotion {
				m.Current = true
				m.Issue = v.MotionIssue
			} else {
				if v.MotionIssue == "" {
					n.savedMotion.Motions = slices.DeleteFunc(n.savedMotion.Motions, func(m SavedActorMotion) bool { return m.Entity == e.ID })
					for j := range n.savedMotion.Cells {
						c := &n.savedMotion.Cells[j]
						for layer := 0; layer < 2; layer++ {
							slot := motionSlot(c, layer)
							if slot.Bound && slot.Entity == e.ID {
								if slot.Key == 0 || slot.Key != binary.LittleEndian.Uint32(c.Payload[4+4*layer:]) ||
									(e.SourceBinding.Identity != 0 && slot.Key != e.SourceBinding.Identity) {
									return fmt.Errorf("sim: saved actor cell %04x has an invalid typed slot", c.Cell)
								}
								*slot = SavedActorSlot{Key: slot.Key}
								n.syncCurrentCellActor(c.Cell, layer, *slot)
							}
						}
					}
				} else {
					m.Current, m.Active, m.Issue = false, false, v.MotionIssue
				}
			}
		} else if v.ImportedMotion {
			return fmt.Errorf("sim: current action lost its imported motion")
		}
		if err := transitFault(*e); err != nil {
			return err
		}
		if err := strideFault(*e); err != nil {
			return err
		}
		if err := turnFault(*e); err != nil {
			return err
		}
		if !e.AttackPhase.defined() || e.AttackTargetKind > AttackTargetStructure || !e.ActionClock.Known && e.ActionClock.End != 0 {
			return fmt.Errorf("sim: invalid current action operands")
		}
		if e.AcquirePursuit && (!e.HasAttackTarget || e.AttackTargetKind != AttackTargetUnit || indexOfEntity(n.entities, e.AttackTarget) < 0 || e.AttackTarget == e.ID) {
			return fmt.Errorf("sim: acquisition pursuit requires a current unit victim")
		}
		if e.PursuitIdle && (!e.HasAttackTarget || e.AttackTargetKind != AttackTargetUnit || e.AcquirePursuit) {
			return fmt.Errorf("sim: idle pursuit requires a current unit victim")
		}
	}
	if err := restoreActionOrderOrdinals(n.savedGroups, a.Actors); err != nil {
		return err
	}
	n.bookCasts = nil
	for _, v := range a.Books {
		c := bookCast{v.Caster, v.Target, v.Spell, v.X, v.Y, v.Remaining, bookPhase(v.Phase), v.Progress, v.Complete, v.AtCell, v.Retained, v.Paid}
		if fault := bookCastFault(c); fault != "" {
			return fmt.Errorf("sim: current book: %s", fault)
		}
		if indexOfEntity(n.entities, c.Caster) < 0 {
			return fmt.Errorf("sim: current book caster is absent")
		}
		n.bookCasts = append(n.bookCasts, c)
	}
	sort.Slice(n.bookCasts, func(i, j int) bool { return n.bookCasts[i].Caster < n.bookCasts[j].Caster })
	for i := 1; i < len(n.bookCasts); i++ {
		if n.bookCasts[i-1].Caster == n.bookCasts[i].Caster {
			return fmt.Errorf("sim: repeated book caster")
		}
	}
	n.casts = nil
	for _, c := range a.Scripts {
		if c.FromX < 0 || c.FromX > 255 || c.FromY < 0 || c.FromY > 255 || c.ToX < 0 || c.ToX > 255 || c.ToY < 0 || c.ToY > 255 || c.Spell == 0 || c.Spell > 255 {
			return fmt.Errorf("sim: invalid pending script cast")
		}
		n.casts = append(n.casts, scriptCast{FromX: uint8(c.FromX), FromY: uint8(c.FromY), ToX: uint8(c.ToX), ToY: uint8(c.ToY), Spell: uint8(c.Spell), Power: c.Power, Target: c.Target, AtUnit: c.AtUnit})
	}
	if err := n.restoreActionReservations(&a, objects); err != nil {
		return err
	}
	n.scrollCasts = a.Scrolls
	sort.Slice(n.scrollCasts, func(i, j int) bool { return n.scrollCasts[i].Caster < n.scrollCasts[j].Caster })
	n.structureUses = slices.Clone(a.StructureUses)
	slices.SortFunc(n.structureUses, func(a, b StructureUse) int {
		if a.Entity < b.Entity {
			return -1
		}
		if a.Entity > b.Entity {
			return 1
		}
		return 0
	})
	for i, use := range n.structureUses {
		actor := indexOfEntity(n.entities, use.Entity)
		structure := indexOfStructure(n.structures, use.Structure)
		if i > 0 && n.structureUses[i-1].Entity == use.Entity || actor < 0 || structure < 0 || !n.entities[actor].Alive() || n.entities[actor].OffMap {
			return fmt.Errorf("sim: invalid current structure use")
		}
	}
	buf := make([]byte, n.scrollSectionLen())
	n.encodeScrolls(buf, 0)
	if _, _, err := decodeScrolls(buf, n.entities, n.bounds, n.spells); err != nil {
		return err
	}
	if a.ActorTraversal != nil {
		if err := n.restoreActionActorTraversal(*a.ActorTraversal); err != nil {
			return err
		}
	} else {
		n.RebuildLoadedActorTraversal()
	}
	for _, e := range n.entities {
		p := e.PendingOrder
		if p.Kind != PendingActorCast && p.Kind != PendingCellCast {
			continue
		}
		if order := n.savedOrder(e.ID); order != nil {
			target, reach := uint32(0), uint8(0)
			if p.Kind == PendingActorCast {
				target = n.currentActionKey(p.Target, AttackTargetUnit)
			}
			if p.Spell >= 1 && p.Spell <= 28 {
				values := BookSlotValues(n.rules, e, n.spells)
				reach = values[p.Spell-1].Range
			}
			spell := binary.LittleEndian.Uint32(order.Raw[0x30:])
			order.Raw = ProjectCastOrderOperands(order.Raw, target, spell, p.X, p.Y, reach, p.Kind == PendingCellCast)
		}
	}
	if _, err := n.MarshalBinary(); err != nil {
		return fmt.Errorf("sim: restored current actions: %w", err)
	}
	*w = n
	return nil
}

func (w *World) restoreActionReservations(a *ActionContinuations, shared map[SavedObjectID]SavedObjectID) error {
	r := w.savedObjects
	if a.SessionVersion > 2 {
		return fmt.Errorf("sim: invalid session root version")
	}
	if len(a.Reservations.Items) == 0 {
		for i, c := range a.Scrolls {
			if c.Item.ObjectID != 0 {
				id := shared[c.Item.ObjectID]
				if r == nil || id == 0 {
					return fmt.Errorf("sim: reserved scroll has no object")
				}
				row, ok := r.Item(id)
				handle := c.Reservation
				if handle == 0 && a.SessionVersion < 2 {
					handle = r.sessionHandle(id)
				}
				if !ok || !r.HasRoot(id, SavedObjectOwner{Kind: SavedOwnerSession, SessionHandle: handle}) || row.Value.Count != 1 {
					return fmt.Errorf("sim: reserved scroll has no session owner")
				}
				a.Scrolls[i].Item = row.Value.Instance()
				a.Scrolls[i].Reservation = handle
			}
		}
		return nil
	}
	if r == nil {
		return fmt.Errorf("sim: reservation has no object registry")
	}
	ids := map[SavedObjectID]SavedObjectID{}
	for old, id := range shared {
		ids[old] = id
	}
	mint := func(old SavedObjectID) (SavedObjectID, error) {
		if old == 0 || ids[old] != 0 {
			return 0, fmt.Errorf("sim: duplicate reserved object")
		}
		id, err := r.mint()
		if err == nil {
			ids[old] = id
		}
		return id, err
	}
	for _, row := range a.Reservations.Effects {
		if id := ids[row.ID]; id != 0 {
			v := r.effect(id)
			if v == nil || v.Retired || v.Value != row.Value {
				return fmt.Errorf("sim: reserved shared Effect differs")
			}
			continue
		}
		id, err := mint(row.ID)
		if err != nil {
			return err
		}
		row.ID, row.Origin, row.ExternalReferences = id, SavedObjectOrigin{Kind: SavedObjectGenerated}, 0
		r.Effects = append(r.Effects, row)
	}
	for _, row := range a.Reservations.Spells {
		if id := ids[row.ID]; id != 0 {
			v := r.spell(id)
			if v == nil || v.Retired || v.Value != row.Value {
				return fmt.Errorf("sim: reserved shared Spell differs")
			}
			continue
		}
		id, err := mint(row.ID)
		if err != nil {
			return err
		}
		row.ID, row.Origin, row.ExternalReferences = id, SavedObjectOrigin{Kind: SavedObjectGenerated}, 0
		r.Spells = append(r.Spells, row)
	}
	legacyRoots := []SavedItemRoot{}
	for _, row := range a.Reservations.Items {
		if row.Value.Count != 1 || row.Retired || row.InFlight != 0 {
			return fmt.Errorf("sim: invalid reserved item lifecycle/count")
		}
		oldID := row.ID
		if a.SessionVersion < 2 && row.Owner.Kind == SavedOwnerSession {
			legacyRoots = append(legacyRoots, SavedItemRoot{oldID, row.Owner})
			row.Owner = SavedObjectOwner{}
		}
		if row.Owner != (SavedObjectOwner{}) {
			return fmt.Errorf("sim: current reservation carries legacy owner")
		}
		id := ids[oldID]
		if id != 0 {
			v := r.item(id)
			expected := row.Value.Clone()
			expected.ObjectID = id
			if v == nil || !StackStateEqual(v.Value, expected) {
				return fmt.Errorf("sim: shared reserved Item differs")
			}
			continue
		}
		var err error
		id, err = mint(oldID)
		if err != nil {
			return err
		}
		row.ID, row.Value.ObjectID, row.Origin = id, id, SavedObjectOrigin{Kind: SavedObjectGenerated}
		row.Effects = slices.Clone(row.Effects)
		for i, old := range row.Effects {
			row.Effects[i] = ids[old]
			if row.Effects[i] == 0 {
				return fmt.Errorf("sim: missing reserved Effect")
			}
		}
		if row.Spell != 0 {
			row.Spell = ids[row.Spell]
			if row.Spell == 0 {
				return fmt.Errorf("sim: missing reserved Spell")
			}
		}
		r.Items = append(r.Items, row)
	}
	for _, root := range append(slices.Clone(a.Reservations.ItemRoots), legacyRoots...) {
		if root.Owner.Kind != SavedOwnerSession || ids[root.ID] == 0 {
			return fmt.Errorf("sim: invalid reservation root")
		}
		if !r.HasRoot(ids[root.ID], root.Owner) {
			if err := r.addRoot(ids[root.ID], root.Owner); err != nil {
				return err
			}
		}
	}

	for i := range a.Scrolls {
		c := &a.Scrolls[i]
		if c.Item.ObjectID == 0 {
			continue
		}
		c.Item.ObjectID = ids[c.Item.ObjectID]
		if c.Reservation == 0 && a.SessionVersion < 2 {
			c.Reservation = r.sessionHandle(c.Item.ObjectID)
		}
		if c.Item.ObjectID == 0 || !r.HasRoot(c.Item.ObjectID, SavedObjectOwner{Kind: SavedOwnerSession, SessionHandle: c.Reservation}) {
			return fmt.Errorf("sim: missing reserved scroll object")
		}
	}
	sort.Slice(r.Items, func(i, j int) bool { return r.Items[i].ID < r.Items[j].ID })
	sort.Slice(r.Effects, func(i, j int) bool { return r.Effects[i].ID < r.Effects[j].ID })
	sort.Slice(r.Spells, func(i, j int) bool { return r.Spells[i].ID < r.Spells[j].ID })
	return r.ValidateNoInFlight()
}
