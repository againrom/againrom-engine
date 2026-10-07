package game

import (
	"fmt"
	"slices"

	"againrom/pkg/sim"
)

type SnapshotPendingQueue struct {
	Commands []sim.Command
	Ignored  []bool
}

type currentPendingCommand struct {
	Command sim.Command
	Issuer  *currentActionBinding `json:",omitempty"`
	Target  *currentActionBinding `json:",omitempty"`
	Ignored bool                  `json:",omitempty"`
}

type currentPendingQueue struct {
	Commands  []currentPendingCommand
	Commanded []currentActionBinding
}

// pendingPlayerScoped reports a command that addresses a roster slot rather
// than an actor, so it carries no issuer binding and never loses an endpoint.
func pendingPlayerScoped(c sim.Command) bool {
	return c.Kind == sim.KindPlayerParameter || c.Kind == sim.KindPlayerDropGold
}

func pendingKindValid(c sim.Command) bool {
	return c.Kind <= sim.KindPickUp || pendingPlayerScoped(c)
}

func pendingTarget(c sim.Command) (present, structure bool) {
	switch c.Kind {
	case sim.KindAttack, sim.KindCast, sim.KindUseScroll, sim.KindGroupDefend:
		return true, false
	case sim.KindAttackStructure, sim.KindUseStructure:
		return true, true
	}
	return false, false
}

func validateSnapshotPending(q *SnapshotPendingQueue) error {
	if q == nil {
		return nil
	}
	if len(q.Commands) > 65536 || len(q.Ignored) != len(q.Commands) {
		return fmt.Errorf("pending command population is invalid")
	}
	for _, c := range q.Commands {
		if !pendingKindValid(c) {
			return fmt.Errorf("pending command kind %d is invalid", c.Kind)
		}
	}
	return nil
}

func validateCurrentPending(q *currentPendingQueue) error {
	if q == nil {
		return nil
	}
	if len(q.Commands) > 65536 || len(q.Commanded) > 65536 {
		return fmt.Errorf("pending command population is invalid")
	}
	check := func(b *currentActionBinding, id sim.EntityID, structure bool) error {
		if b == nil || b.ID != id || b.Structure != structure || b.Missing != (b.Object == 0) {
			return fmt.Errorf("pending command binding is invalid")
		}
		return nil
	}
	for _, p := range q.Commands {
		c := p.Command
		if !pendingKindValid(c) {
			return fmt.Errorf("pending command kind %d is invalid", c.Kind)
		}
		if pendingPlayerScoped(c) {
			if p.Issuer != nil {
				return fmt.Errorf("player command has an actor binding")
			}
		} else if err := check(p.Issuer, c.Entity, false); err != nil {
			return err
		}
		if target, structure := pendingTarget(c); target {
			if err := check(p.Target, sim.EntityID(uint32(c.X)), structure); err != nil {
				return err
			}
		} else if p.Target != nil {
			return fmt.Errorf("pending scalar has an endpoint binding")
		}
	}
	seen := map[sim.EntityID]bool{}
	for _, b := range q.Commanded {
		if err := check(&b, b.ID, false); err != nil || seen[b.ID] {
			return fmt.Errorf("pending commanded binding is invalid or repeated")
		}
		seen[b.ID] = true
	}
	return nil
}

func projectCurrentPending(r SnapshotResidue, actors, structures map[sim.EntityID]uint16) (*currentPendingQueue, error) {
	if err := validateSnapshotPending(r.PendingQueue); err != nil || r.PendingQueue == nil {
		return nil, err
	}
	bind := func(id sim.EntityID, structure bool) *currentActionBinding {
		objects := actors
		if structure {
			objects = structures
		}
		object, ok := objects[id]
		return &currentActionBinding{ID: id, Structure: structure, Object: object, Missing: !ok || object == 0}
	}
	q := &currentPendingQueue{Commands: make([]currentPendingCommand, 0, len(r.PendingQueue.Commands))}
	for i, c := range r.PendingQueue.Commands {
		p := currentPendingCommand{Command: c, Ignored: r.PendingQueue.Ignored[i]}
		if !pendingPlayerScoped(c) {
			p.Issuer = bind(c.Entity, false)
		}
		if target, structure := pendingTarget(c); target {
			p.Target = bind(sim.EntityID(uint32(c.X)), structure)
		}
		q.Commands = append(q.Commands, p)
	}
	ids := slices.Clone(r.Commanded)
	slices.Sort(ids)
	for _, id := range ids {
		q.Commanded = append(q.Commanded, *bind(sim.EntityID(id), false))
	}
	return q, validateCurrentPending(q)
}

func (mw *mapWorld) restoreCurrentPending(q *currentPendingQueue, actors map[uint16]sim.EntityID) error {
	if err := validateCurrentPending(q); err != nil {
		return err
	}
	structures := map[uint16]sim.EntityID{}
	ss, _, _ := mw.world.SavedStructures()
	for _, s := range ss {
		object, err := currentTypedIdentityIndex(mw.mission.state.savedDocument.Document, savedStructureClass(s.Class), s.SourceKey)
		if err != nil {
			return err
		}
		if object != 0 {
			structures[object] = sim.EntityID(s.ID)
		}
	}
	resolve := func(b *currentActionBinding) (sim.EntityID, bool, error) {
		if b.Missing {
			return b.ID, false, nil
		}
		objects := actors
		if b.Structure {
			objects = structures
		}
		id, ok := objects[b.Object]
		if !ok {
			return 0, false, fmt.Errorf("pending command object %d is absent", b.Object)
		}
		return id, true, nil
	}
	commands, ignored := make([]sim.Command, 0, len(q.Commands)), make([]bool, 0, len(q.Commands))
	for _, p := range q.Commands {
		c, skip := p.Command, p.Ignored
		if p.Issuer != nil {
			id, ok, err := resolve(p.Issuer)
			if err != nil {
				return err
			}
			c.Entity, skip = id, skip || !ok
		}
		if p.Target != nil {
			id, ok, err := resolve(p.Target)
			if err != nil {
				return err
			}
			c.X, skip = int32(uint32(id)), skip || !ok
		}
		commands, ignored = append(commands, c), append(ignored, skip)
	}
	commanded := map[sim.EntityID]bool{}
	for _, b := range q.Commanded {
		id, ok, err := resolve(&b)
		if err != nil {
			return err
		}
		if ok {
			commanded[id] = true
		}
	}
	mw.pending, mw.pendingIgnored, mw.commanded = commands, ignored, commanded
	return nil
}

func (mw *mapWorld) pendingEndpointAbsent(c sim.Command) bool {
	if pendingPlayerScoped(c) {
		return false
	}
	if _, ok := mw.world.Entity(c.Entity); !ok {
		return true
	}
	if target, structure := pendingTarget(c); target {
		id := sim.EntityID(uint32(c.X))
		if !structure {
			_, ok := mw.world.Entity(id)
			return !ok
		}
		for _, s := range mw.world.Structures() {
			if sim.EntityID(s.ID) == id {
				return false
			}
		}
		return true
	}
	return false
}
