package game

import (
	"fmt"
	"reflect"
	"slices"

	"againrom/pkg/sim"
)

// residueCarrier is one session residue carrier: a group of SnapshotResidue
// fields the front end captures from the running map, validates in its SAV
// rows, checks for actor bindings, restores and writes under its SAV key.
// Every site that handles session residue iterates residueCarriers, so a new
// carrier is one row. Each carrier keeps its own row codec and bounds in the
// functions its row names; a part the carrier lacks is a named empty part.
type residueCarrier struct {
	name string
	// fields are the SnapshotResidue fields the carrier owns.
	fields   []string
	capture  func(mw *mapWorld, r *SnapshotResidue)
	validate func(a *currentActionData) error
	actors   residueActors
	restore  residueRestore
	sav      residueSAVKey
}

// residueActors is the carrier's actor binding check.
type residueActors struct {
	// rows lists the actor IDs the carrier's SAV rows name, in row order.
	rows func(a *currentActionData) []sim.EntityID
	// unbound is the decode refusal of a row whose actor has no binding. Nil
	// leaves an unresolved actor to the carrier's own restore.
	unbound func(id sim.EntityID) error
	// early binds the actors with the action graph, before bodies and actor
	// values: the Bindings order every save this engine wrote carries.
	early bool
}

// residueRestore puts the carrier back: snapshot from an in-memory
// SnapshotResidue, saved from the decoded SAV rows of a cold LOAD.
type residueRestore struct {
	snapshot func(mw *mapWorld, r SnapshotResidue)
	saved    func(l residueLoad) error
}

// residueLoad is what a cold LOAD hands each carrier's restore.
type residueLoad struct {
	mw *mapWorld
	a  *currentActionData
	// ids maps a saved actor ID to the reopened World's actor.
	ids map[sim.EntityID]sim.EntityID
	// objects maps a SAV object index to the reopened World's actor.
	objects map[uint16]sim.EntityID
}

// residueSAVKey names the currentActionData fields the carrier writes and the
// projection that writes them. A carrier with no SAV key says why in unsaved.
// A key "Residue.<name>" is an entry of currentActionData.Residue, which
// takes a carrier without a field of its own.
type residueSAVKey struct {
	keys    []string
	write   func(p residueProjection) error
	unsaved string
}

// residueProjection is what a SAVE hands each carrier's write.
type residueProjection struct {
	r SnapshotResidue
	s Snapshot
	w *sim.World
	a *currentActionData
	// actors and structures map an entity to its SAV object index.
	actors, structures map[sim.EntityID]uint16
}

func residueNotCaptured(*mapWorld, *SnapshotResidue)    {}
func residueNoBounds(*currentActionData) error          { return nil }
func residueNoActors(*currentActionData) []sim.EntityID { return nil }
func residueNotRestored(*mapWorld, SnapshotResidue)     {}
func residueNotLoaded(residueLoad) error                { return nil }

var residueCarriers = []residueCarrier{
	{
		name:     "spell bolts",
		fields:   []string{"SpellBolts"},
		capture:  residueNotCaptured,
		validate: func(a *currentActionData) error { return validateSpellBolts(a.Bolts) },
		actors:   residueActors{rows: residueNoActors},
		restore:  residueRestore{snapshot: residueNotRestored, saved: residueNotLoaded},
		sav: residueSAVKey{keys: []string{"Bolts"}, write: func(p residueProjection) error {
			p.a.Bolts = p.r.SpellBolts
			return nil
		}},
	},
	{
		name:     "heal bursts",
		fields:   []string{"HealBursts"},
		capture:  func(mw *mapWorld, r *SnapshotResidue) { r.HealBursts = mw.captureHealBursts() },
		validate: func(a *currentActionData) error { return validateHealBursts(a.Heals) },
		actors:   residueActors{rows: residueNoActors},
		restore: residueRestore{
			snapshot: func(mw *mapWorld, r SnapshotResidue) { mw.restoreHealBursts(r.HealBursts) },
			saved: func(l residueLoad) error {
				l.mw.restoreHealBursts(l.a.Heals)
				return nil
			},
		},
		sav: residueSAVKey{keys: []string{"Heals"}, write: func(p residueProjection) error {
			p.a.Heals = p.r.HealBursts
			return nil
		}},
	},
	{
		name:     "cast runs",
		fields:   []string{"CastRuns"},
		capture:  func(mw *mapWorld, r *SnapshotResidue) { r.CastRuns = mw.captureCastRuns() },
		validate: func(a *currentActionData) error { return validateCastRuns(a.Runs) },
		actors: residueActors{rows: func(a *currentActionData) []sim.EntityID {
			return residueRowActors(a.Runs, func(r SnapshotCastRun) sim.EntityID { return r.Entity })
		}},
		restore: residueRestore{
			snapshot: func(mw *mapWorld, r SnapshotResidue) { mw.restoreCastRuns(r.CastRuns) },
			saved: func(l residueLoad) error {
				for i := range l.a.Runs {
					id, ok := l.ids[l.a.Runs[i].Entity]
					if !ok {
						return fmt.Errorf("current cast animation actor is absent")
					}
					l.a.Runs[i].Entity = id
				}
				l.mw.restoreCastRuns(l.a.Runs)
				return nil
			},
		},
		sav: residueSAVKey{keys: []string{"Runs"}, write: func(p residueProjection) error {
			p.a.Runs = p.r.CastRuns
			return nil
		}},
	},
	{
		name:     "target marks",
		fields:   []string{"TargetMarks"},
		capture:  func(mw *mapWorld, r *SnapshotResidue) { r.TargetMarks = mw.captureTargetMarks() },
		validate: func(a *currentActionData) error { return validateTargetMarks(a.TargetMarks) },
		actors: residueActors{
			rows: func(a *currentActionData) []sim.EntityID {
				return residueRowActors(a.TargetMarks, func(r SnapshotTargetMark) sim.EntityID { return r.Entity })
			},
			unbound: func(sim.EntityID) error { return fmt.Errorf("current target mark actor lacks a binding") },
		},
		restore: residueRestore{
			snapshot: func(mw *mapWorld, r SnapshotResidue) { mw.restoreTargetMarks(r.TargetMarks) },
			saved: func(l residueLoad) error {
				for i := range l.a.TargetMarks {
					id, ok := l.ids[l.a.TargetMarks[i].Entity]
					if !ok {
						return fmt.Errorf("current target mark actor is absent")
					}
					l.a.TargetMarks[i].Entity = id
				}
				l.mw.restoreTargetMarks(l.a.TargetMarks)
				return nil
			},
		},
		sav: residueSAVKey{keys: []string{"TargetMarks"}, write: func(p residueProjection) error {
			p.a.TargetMarks = p.r.TargetMarks
			return nil
		}},
	},
	{
		// Swing and Phase leave as the attack half of the Animation clocks;
		// the movement half comes from the snapshot's map motion.
		name:     "attack clocks",
		fields:   []string{"Swing", "Phase"},
		capture:  captureAttackClocks,
		validate: func(a *currentActionData) error { return validateCurrentAnimation(a.Animation, a.Runs) },
		actors: residueActors{
			rows: func(a *currentActionData) []sim.EntityID {
				return residueRowActors(a.Animation, func(r currentAnimationClock) sim.EntityID { return r.Entity })
			},
			unbound: func(sim.EntityID) error { return fmt.Errorf("current animation actor lacks a binding") },
		},
		restore: residueRestore{
			snapshot: restoreAttackClocks,
			saved:    func(l residueLoad) error { return l.mw.restoreCurrentAnimation(l.a.Animation, l.ids) },
		},
		sav: residueSAVKey{keys: []string{"Animation"}, write: func(p residueProjection) error {
			p.a.Animation = mergeCurrentMapMotion(captureCurrentAnimation(p.r), p.s.mapMotion)
			return nil
		}},
	},
	{
		name:     "death ages",
		fields:   []string{"deathAges"},
		capture:  func(mw *mapWorld, r *SnapshotResidue) { r.deathAges = mw.captureCurrentDeathAges() },
		validate: func(a *currentActionData) error { return validateCurrentDeathAges(a.DeathAges) },
		actors: residueActors{
			rows: func(a *currentActionData) []sim.EntityID {
				return residueRowActors(a.DeathAges, func(r currentDeathAge) sim.EntityID { return r.Entity })
			},
			unbound: func(id sim.EntityID) error { return fmt.Errorf("invalid current death age for actor %d", id) },
		},
		restore: residueRestore{
			snapshot: residueNotRestored,
			saved:    func(l residueLoad) error { return l.mw.restoreCurrentDeathAges(l.a.DeathAges, l.ids) },
		},
		sav: residueSAVKey{keys: []string{"DeathAges"}, write: func(p residueProjection) error {
			p.a.DeathAges = p.r.deathAges
			return nil
		}},
	},
	{
		name:   "visual identities",
		fields: []string{"VisualIdentities", "VisualNext"},
		capture: func(mw *mapWorld, r *SnapshotResidue) {
			r.VisualIdentities, r.VisualNext = mw.currentVisualIdentities()
		},
		validate: func(a *currentActionData) error { return validateVisualIdentities(a.VisualIdentities) },
		actors: residueActors{rows: func(a *currentActionData) []sim.EntityID {
			return residueRowActors(a.VisualIdentities, func(r SnapshotVisualIdentity) sim.EntityID { return r.Entity })
		}},
		restore: residueRestore{
			snapshot: func(mw *mapWorld, r SnapshotResidue) { mw.restoreVisualIdentities(r.VisualIdentities, r.VisualNext) },
			saved: func(l residueLoad) error {
				// An identity whose actor the reopened World lacks is dropped.
				var rows []SnapshotVisualIdentity
				for _, v := range l.a.VisualIdentities {
					if id, ok := l.ids[v.Entity]; ok {
						v.Entity = id
						rows = append(rows, v)
					}
				}
				l.mw.restoreVisualIdentities(rows, l.a.VisualNext)
				return nil
			},
		},
		sav: residueSAVKey{keys: []string{"VisualIdentities", "VisualNext"}, write: func(p residueProjection) error {
			p.a.VisualIdentities, p.a.VisualNext = p.r.VisualIdentities, p.r.VisualNext
			return nil
		}},
	},
	{
		// The queue binds its own issuers and targets by SAV object; the
		// commanded set rides in it.
		name:    "pending commands",
		fields:  []string{"PendingGameOptions", "PendingQueue", "Commanded"},
		capture: capturePendingCommands,
		validate: func(a *currentActionData) error {
			if err := validatePendingGameOptions(a.Options); err != nil {
				return err
			}
			return validateCurrentPending(a.Pending)
		},
		actors: residueActors{rows: residueNoActors},
		restore: residueRestore{
			snapshot: restorePendingCommands,
			saved: func(l residueLoad) error {
				if l.a.Pending != nil {
					return l.mw.restoreCurrentPending(l.a.Pending, l.objects)
				}
				for _, o := range l.a.Options {
					l.mw.applyGameOption(o.Option, o.Value)
				}
				return nil
			},
		},
		sav: residueSAVKey{keys: []string{"Options", "Pending"}, write: func(p residueProjection) (err error) {
			p.a.Options = p.r.PendingGameOptions
			p.a.Pending, err = projectCurrentPending(p.r, p.actors, p.structures)
			return err
		}},
	},
	{
		name:     "group tag",
		fields:   []string{"GroupTag"},
		capture:  func(mw *mapWorld, r *SnapshotResidue) { r.GroupTag = mw.groupTag },
		validate: residueNoBounds,
		actors:   residueActors{rows: residueNoActors},
		restore: residueRestore{
			snapshot: func(mw *mapWorld, r SnapshotResidue) { mw.groupTag = r.GroupTag },
			saved: func(l residueLoad) error {
				l.mw.groupTag = l.a.GroupTag
				return nil
			},
		},
		sav: residueSAVKey{keys: []string{"GroupTag"}, write: func(p residueProjection) error {
			p.a.GroupTag = p.r.GroupTag
			return nil
		}},
	},
	{
		// The second game's SAV carries the visibility plane here; the first
		// game's carries exploration in its application Fog leaves.
		name:    "fog",
		fields:  []string{"FogCols", "FogRows", "FogExplored", "FogVisible"},
		capture: captureFogResidue,
		validate: func(a *currentActionData) error {
			if a.Fog == nil {
				return nil
			}
			return a.Fog.validate()
		},
		actors: residueActors{rows: residueNoActors},
		restore: residueRestore{
			snapshot: func(mw *mapWorld, r SnapshotResidue) { mw.restoreNativeFog(r) },
			saved: func(l residueLoad) error {
				if l.a.Fog != nil && !l.mw.restoreNativeFog(l.a.Fog.residue()) {
					return fmt.Errorf("current visibility belongs to a different mission extent")
				}
				return nil
			},
		},
		sav: residueSAVKey{keys: []string{"Fog"}, write: func(p residueProjection) error {
			if p.w.Script().Dialect() == sim.ScriptROM2 {
				p.a.Fog = captureCurrentMissionFog(p.r)
			}
			return nil
		}},
	},
	{
		name:   "mission notices",
		fields: []string{"PendingMessages"},
		capture: func(mw *mapWorld, r *SnapshotResidue) {
			if mw.mission != nil {
				r.PendingMessages = slices.Clone(mw.mission.pendingMessages)
			}
		},
		validate: func(a *currentActionData) error {
			if len(a.PendingMessages) > 65536 {
				return fmt.Errorf("current mission notice queue exceeds its bound")
			}
			return nil
		},
		actors: residueActors{rows: residueNoActors},
		restore: residueRestore{
			snapshot: residueNotRestored,
			saved: func(l residueLoad) error {
				if l.mw.mission != nil {
					l.mw.mission.pendingMessages = slices.Clone(l.a.PendingMessages)
				}
				return nil
			},
		},
		sav: residueSAVKey{keys: []string{"PendingMessages"}, write: func(p residueProjection) error {
			p.a.PendingMessages = slices.Clone(p.r.PendingMessages)
			return nil
		}},
	},
	{
		name:     "departed characters",
		fields:   []string{"DepartedCharacters"},
		capture:  captureDepartedCharacters,
		validate: func(a *currentActionData) error { return validateCurrentDepartedCharacters(a.DepartedCharacters) },
		actors: residueActors{
			rows: func(a *currentActionData) []sim.EntityID {
				return residueRowActors(a.DepartedCharacters, func(id uint32) sim.EntityID { return sim.EntityID(id) })
			},
			unbound: func(sim.EntityID) error { return fmt.Errorf("current departed character lacks an actor binding") },
			early:   true,
		},
		restore: residueRestore{
			snapshot: restoreDepartedCharacters,
			saved: func(l residueLoad) error {
				return restoreCurrentDepartedCharacters(l.mw, l.a.DepartedCharacters, l.ids)
			},
		},
		sav: residueSAVKey{keys: []string{"DepartedCharacters"}, write: func(p residueProjection) error {
			p.a.DepartedCharacters = append([]uint32(nil), p.r.DepartedCharacters...)
			return nil
		}},
	},
	{
		// It follows the departed characters: a guarded character's
		// departure is part of the loss it latches.
		name:   "mission loss",
		fields: []string{"MissionLost"},
		capture: func(mw *mapWorld, r *SnapshotResidue) {
			r.MissionLost = mw.mission != nil && mw.mission.outcome == sim.OutcomeLost
		},
		validate: residueNoBounds,
		actors:   residueActors{rows: residueNoActors},
		restore:  residueRestore{snapshot: restoreMissionLoss, saved: residueNotLoaded},
		sav:      residueSAVKey{unsaved: "a SAV mission derives its loss from the saved World and script outcome"},
	},
}

func residueRowActors[T any](rows []T, actor func(T) sim.EntityID) []sim.EntityID {
	if len(rows) == 0 {
		return nil
	}
	out := make([]sim.EntityID, len(rows))
	for i, r := range rows {
		out[i] = actor(r)
	}
	return out
}

// writeResidueCarriers writes every carrier under its SAV key.
func writeResidueCarriers(p residueProjection) error {
	for _, c := range residueCarriers {
		if c.sav.write == nil {
			continue
		}
		if err := c.sav.write(p); err != nil {
			return err
		}
	}
	return nil
}

// validateResidueCarriers checks every carrier's SAV rows against its own
// bounds, and refuses a keyed entry no carrier owns.
func validateResidueCarriers(a *currentActionData) error {
	for key := range a.Residue {
		if !slices.ContainsFunc(residueCarriers, func(c residueCarrier) bool {
			return slices.Contains(c.sav.keys, "Residue."+key)
		}) {
			return fmt.Errorf("current residue carrier %q is unknown", key)
		}
	}
	for _, c := range residueCarriers {
		if err := c.validate(a); err != nil {
			return err
		}
	}
	return nil
}

// checkResidueActorBindings refuses a carrier row whose actor has no binding.
func checkResidueActorBindings(a *currentActionData, bound map[sim.EntityID]bool) error {
	for _, c := range residueCarriers {
		if c.actors.unbound == nil {
			continue
		}
		for _, id := range c.actors.rows(a) {
			if !bound[id] {
				return c.actors.unbound(id)
			}
		}
	}
	return nil
}

// bindResidueActors visits the actors of the carriers bound early or late, in
// table order.
func bindResidueActors(a *currentActionData, early bool, ref func(sim.EntityID, bool) (sim.EntityID, error)) error {
	for _, c := range residueCarriers {
		if c.actors.early != early {
			continue
		}
		for _, id := range c.actors.rows(a) {
			if _, err := ref(id, false); err != nil {
				return err
			}
		}
	}
	return nil
}

// loadResidueCarriers restores every carrier from its decoded SAV rows.
func loadResidueCarriers(l residueLoad) error {
	for _, c := range residueCarriers {
		if err := c.restore.saved(l); err != nil {
			return err
		}
	}
	return nil
}

// hasMissionState reports a residue that holds any carrier's state.
func (r SnapshotResidue) hasMissionState() bool {
	v := reflect.ValueOf(r)
	for i := range v.NumField() {
		if residueFieldHeld(v.Field(i)) {
			return true
		}
	}
	return false
}

// residueFieldHeld reports a SnapshotResidue field that holds state.
func residueFieldHeld(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Slice, reflect.Map:
		return v.Len() != 0
	case reflect.Pointer:
		return !v.IsNil()
	default:
		return !v.IsZero()
	}
}

func (mw *mapWorld) captureHealBursts() []SnapshotHealBurst {
	var out []SnapshotHealBurst
	for _, b := range mw.healBursts {
		out = append(out, SnapshotHealBurst{b.at, b.picture, b.owner, b.seed, b.tile, b.drain, b.age})
	}
	return out
}

func (mw *mapWorld) captureTargetMarks() []SnapshotTargetMark {
	var out []SnapshotTargetMark
	for _, el := range mw.markElements {
		if el.Transient && mw.markTargetPresent(el) {
			out = append(out, SnapshotTargetMark{el.Entity, uint32(el.Kind)<<16 | uint32(el.Count), el.Seed})
		}
	}
	return out
}

func (mw *mapWorld) captureCastRuns() []SnapshotCastRun {
	var out []SnapshotCastRun
	for id, c := range mw.castRun {
		out = append(out, SnapshotCastRun{id, c.left, c.span, mw.swing[id], mw.phase[id]})
	}
	slices.SortFunc(out, func(a, b SnapshotCastRun) int {
		if a.Entity < b.Entity {
			return -1
		}
		if a.Entity > b.Entity {
			return 1
		}
		return 0
	})
	return out
}

func captureAttackClocks(mw *mapWorld, r *SnapshotResidue) {
	r.Swing = make(map[uint32]int, len(mw.swing))
	r.Phase = make(map[uint32]uint8, len(mw.phase))
	for id, n := range mw.swing {
		r.Swing[uint32(id)] = n
	}
	for id, p := range mw.phase {
		r.Phase[uint32(id)] = uint8(p)
	}
}

func restoreAttackClocks(mw *mapWorld, r SnapshotResidue) {
	for id, n := range r.Swing {
		mw.swing[sim.EntityID(id)] = n
	}
	for id, p := range r.Phase {
		mw.phase[sim.EntityID(id)] = sim.AttackPhase(p)
	}
}

func capturePendingCommands(mw *mapWorld, r *SnapshotResidue) {
	r.PendingGameOptions = pendingGameOptions(mw.pending)
	r.PendingQueue = &SnapshotPendingQueue{Commands: append([]sim.Command(nil), mw.pending...)}
	if len(mw.pending) != 0 {
		r.PendingQueue.Ignored = make([]bool, len(mw.pending))
		copy(r.PendingQueue.Ignored, mw.pendingIgnored)
	}
	for id := range mw.commanded {
		r.Commanded = append(r.Commanded, uint32(id))
	}
}

func restorePendingCommands(mw *mapWorld, r SnapshotResidue) {
	if r.PendingQueue != nil {
		mw.pending = slices.Clone(r.PendingQueue.Commands)
		mw.pendingIgnored = slices.Clone(r.PendingQueue.Ignored)
	} else {
		for _, order := range r.PendingGameOptions {
			mw.applyGameOption(order.Option, order.Value)
		}
	}
	for _, id := range r.Commanded {
		mw.commanded[sim.EntityID(id)] = true
	}
}

func captureFogResidue(mw *mapWorld, r *SnapshotResidue) {
	if mw.fog != nil {
		r.FogCols, r.FogRows = mw.fog.cols, mw.fog.rows
		r.FogExplored = append([]byte(nil), mw.fog.explored...)
		r.FogVisible = append([]byte(nil), mw.fog.visible...)
	}
}

// Party order makes the envelope deterministic, without map iteration.
func captureDepartedCharacters(mw *mapWorld, r *SnapshotResidue) {
	if mw.mission == nil {
		return
	}
	for _, id := range mw.mission.guarded {
		if mw.mission.departed[id] {
			r.DepartedCharacters = append(r.DepartedCharacters, uint32(id))
		}
	}
}

func restoreDepartedCharacters(mw *mapWorld, r SnapshotResidue) {
	if mw.mission == nil {
		return
	}
	for _, id := range r.DepartedCharacters {
		if mw.isGuarded(sim.EntityID(id)) {
			if mw.mission.departed == nil {
				mw.mission.departed = make(map[sim.EntityID]bool)
			}
			mw.mission.departed[sim.EntityID(id)] = true
		}
	}
}

func restoreMissionLoss(mw *mapWorld, r SnapshotResidue) {
	if mw.mission == nil {
		return
	}
	if r.MissionLost || mw.guardedCharacterLost() || mw.world.Outcome() == sim.OutcomeLost {
		alreadyShown := mw.mission.outcome == sim.OutcomeLost && mw.mission.outcomeShown
		mw.mission.announced, mw.mission.outcome = true, sim.OutcomeLost
		if !alreadyShown {
			mw.showOutcome()
		}
	}
}
