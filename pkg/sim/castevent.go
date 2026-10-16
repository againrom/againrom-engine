package sim

// Cast observations are immutable step return values for presentation. They
// enter neither World nor its byte form or digest; observing a step cannot
// change simulation. Script traces have a separate consumer and are independent.

// ScriptCastEvent is one temporary-caster spell that actually resolved. A
// script caster has no EntityID or owner, so its source and destination cells
// are carried directly. A refused or malformed pending script record produces
// no event.
type ScriptCastEvent struct {
	Produced     bool
	Records      []uint16
	Spell        uint16
	FromX, FromY int32
	ToX, ToY     int32
	// Victims is a Prismatic Spray's victim cells in selection order, as
	// CastEvent.Victims. Observation only; nothing reads it back.
	Victims []CellPoint
}

// CastEvent captures one applied cast and its endpoints at resolution time,
// before movement or removal can invalidate a later lookup. Weapon distinguishes
// an attack release from a book cast for the frontend's animation selection.
type CastEvent struct {
	Produced bool
	Records  []uint16
	Caster   EntityID
	Target   EntityID
	// Actor zero is valid; point delivery's zero is not an actor endpoint.
	AtCell bool `json:"-"`
	Spell  uint16
	School uint8

	// Owner is the CASTER's own roster slot, carried so a consumer can gate the
	// picture on whether that caster may be seen without looking the entity up
	// again — and a felled caster may be gone from a later read.
	Owner uint32
	// TargetOwner gates target-attached semantic feedback such as Heal. It is
	// captured with the target cell so later movement or removal cannot turn a
	// short-lived client effect into a stale entity lookup.
	TargetOwner uint32

	FromX, FromY int32
	ToX, ToY     int32

	// HealthRestored is the positive health delta this application produced.
	// It is semantic observation, not a guess from the spell id: a refused or
	// zero-outcome Heal emits no event, and the client can create one cosmetic
	// burst without reading mutable health back from the world.
	HealthRestored int32

	Weapon bool

	// Rider narrows Weapon to the FIGHTER's rider arm (weaponRiderApply,
	// spell.go). A weapon-borne release has two arms and they differ in the one
	// way a drawing tier cares about. The caster's replacement release runs on
	// an attack cycle the drawing tier is already animating: advanceAttack sets
	// AttackCasting for it, so the wind-up is a clock a live draw can hang off.
	// The rider arm never reaches AttackCasting — its carrier is not a mage, so
	// the phase is AttackCharging — and it applies at the blow. It has no
	// wind-up to draw across, which makes it a book cast's shape: an object that
	// travels caster to target on its own.
	//
	// Both arms set Weapon. Only the second sets this.
	Rider bool

	// Victims is the cast's own victim list, in the order the apply walked it,
	// and it is empty for every cast that reaches exactly one actor.
	//
	// ONE ROW HAS ONE: Prismatic Spray, whose arm applies to every hostile actor
	// within a power-scaled radius of the target (applyPrismatic, celleffect.go)
	// and whose picture draws one figure per victim, tagged with that victim's
	// own loop index (`MAGIC-BOLTLIST-071`). ToX/ToY still name the actor the cast
	// was directed at and the list is drawn from rather than added to it: the
	// directed target is itself on it.
	//
	// It is a fresh slice per event, so a consumer may hold it, and no simulation
	// state reads it — castevent.go's own rule for everything here.
	Victims []CellPoint

	Facing uint8
}

// castObs is the historical name of the report sink the step threads through
// cast and area-effect routines and reads once for the report. A NIL
// RECEIVER RECORDS NOTHING and every caller may hold one, which is what makes
// Step and StepObserved the same code path — recorders return rather than
// callers branching.
type castObs struct {
	scriptMessages []int32
	casts          []CastEvent
	scriptCasts    []ScriptCastEvent
	paints         []AreaPaint
	damages        []DamageEvent
	relocations    []Relocation
	// held names the casters whose charging book cast waited this step for
	// their turn.
	held             []EntityID
	flightStarts     []FlightStart
	projectilePhases []ProjectilePhase
	projectileMarks  []ProjectileMark
}

// ProjectileMark is one resolved actor target's picture-20 or picture-30
// callback. It is client observation and carries no simulation state.
type ProjectileMark struct {
	Target  EntityID
	Picture int32
	Binding SourceBinding
}

func (o *castObs) recordProjectileMark(mark ProjectileMark) {
	if o != nil {
		o.projectileMarks = append(o.projectileMarks, mark)
	}
}

// ProjectilePhase is one record's local callback boundary (ANIM-149).
type ProjectilePhase struct {
	ID                   uint16
	Picture, ActionPhase int32
	Owner                uint32
	CellX, CellY         int32
}

func (o *castObs) recordProjectilePhase(w *World, p SavedProjectile, d SavedProjectileDriver) {
	if o == nil || w.SecondGame() || p.Picture != 51 || p.ActionPhase != 8 {
		return
	}
	o.projectilePhases = append(o.projectilePhases, ProjectilePhase{ID: p.ID,
		Picture: p.Picture, ActionPhase: p.ActionPhase, Owner: d.Owner,
		CellX: p.X / 256, CellY: p.Y / 256})
}

// Relocation is one Teleport that moved its caster in this step: the caster,
// its roster slot and the two cells. A vetoed destination moves nothing and
// records nothing.
type Relocation struct {
	Caster       EntityID
	Owner        uint32
	FromX, FromY int32
	ToX, ToY     int32
}

// recordAdmission reports a cast at its admission: the caster's cell and the
// destination cell, which for a unit target is that unit's cell now.
func (o *castObs) recordAdmission(w *World, ci int, target EntityID, atCell bool, rule SpellRule, x, y int32) {
	if o == nil {
		return
	}
	a := w.entities[ci]
	ev := CastEvent{Caster: a.ID, Spell: rule.ID, School: rule.School, Owner: a.Owner, AtCell: atCell,
		FromX: a.X, FromY: a.Y, ToX: x, ToY: y, Facing: a.Facing}
	if !atCell {
		ev.Target = target
		if ti := indexOfEntity(w.entities, target); ti >= 0 {
			ev.TargetOwner = w.entities[ti].Owner
		}
	}
	o.casts = append(o.casts, ev)
}

func (o *castObs) recordRelocation(e Entity, fromX, fromY int32) {
	if o != nil {
		o.relocations = append(o.relocations, Relocation{Caster: e.ID, Owner: e.Owner,
			FromX: fromX, FromY: fromY, ToX: e.X, ToY: e.Y})
	}
}

func (o *castObs) recordScriptMessage(event int32) {
	if o != nil {
		o.scriptMessages = append(o.scriptMessages, event)
	}
}

func (o *castObs) recordScriptCast(c scriptCast, spell uint16, toX, toY int32, victims []CellPoint) {
	if o == nil {
		return
	}
	o.scriptCasts = append(o.scriptCasts, ScriptCastEvent{
		Spell: spell, FromX: int32(c.FromX), FromY: int32(c.FromY), ToX: toX, ToY: toY, Victims: victims,
	})
}

// DamageEvent is one causal damage-message health transition in application
// order. A zero unit strike carries equal levels (ANIM-125). Observation is
// transient and changes no save byte or digest. BeforeHP describes the server
// application; clients compare AfterHP against their own retained health.
type DamageEvent struct {
	Target            EntityID
	BeforeHP, AfterHP int32
}

// AreaPaint is one STAGE of one staged area effect, as the tick reports it
// (1003): the spell, whose cast it was, and every cell that stage accepted.
//
// IT IS THE ORIGINAL'S OWN MESSAGE. A staged effect sends one message per
// accepted cell inside its cell loop, and each message builds one transient
// client object of 16 ticks, 18 for `acid_stream` (`MAGIC-AREADRAW-049`,
// `MAGIC-RING-048`). Those objects are what makes a staged effect read as an
// explosion spreading, rocks landing across an area, or a cone growing: an
// earlier stage's cells are still drawn while a later stage lights up.
//
// A CLOUD AND A BLAST EMIT NOTHING HERE. A cloud creates no object at all and is
// drawn from its retained cell set; a blast's single centre object already
// arrives through CastEvent.
//
// IT IS NOT STATE. It is built inside one advance and handed back, castObs' own
// rule: no field of World carries it, the byte form does not know it, and a
// world advanced with a nil sink is the same world byte for byte.
//
// Cells is freshly allocated per stage, so a consumer may hold it.
type AreaPaint struct {
	Spell uint16
	Owner uint32
	Cells []CellPoint
}

// CellPoint is one map cell, in the exported coordinates CellEffect already
// uses.
type CellPoint struct{ X, Y int32 }

// recordPaint reports one staged stage's accepted cells. An empty stage is
// reported as nothing at all: Acid Stream's sixth stage is intentionally empty
// on an even orientation, and a stage with no cell sent no message.
func (o *castObs) recordPaint(w *World, e cellEffect, cells []uint16) {
	if o == nil || len(cells) == 0 {
		return
	}
	p := AreaPaint{Spell: e.Spell, Cells: make([]CellPoint, 0, len(cells))}
	if e.HasCaster {
		if ci := indexOfEntity(w.entities, e.Caster); ci >= 0 {
			p.Owner = w.entities[ci].Owner
		}
	}
	for _, k := range cells {
		x, y := keyCell(k)
		p.Cells = append(p.Cells, CellPoint{X: x, Y: y})
	}
	o.paints = append(o.paints, p)
}

func (o *castObs) record(w *World, ci, ti int, rule SpellRule, weapon bool) {
	if o == nil {
		return
	}
	a := w.entities[ci]
	o.recordFrom(w, ci, ti, rule, weapon, a.X, a.Y)
}

func (o *castObs) recordFrom(w *World, ci, ti int, rule SpellRule, weapon bool, fromX, fromY int32) {
	o.recordFromResult(w, ci, ti, rule, weapon, fromX, fromY, 0)
}

func (o *castObs) recordFromResult(w *World, ci, ti int, rule SpellRule, weapon bool, fromX, fromY, restored int32) {
	if o == nil {
		return
	}
	a, t := w.entities[ci], w.entities[ti]
	// THE ARM IS DERIVED AND NOT THREADED. weaponRiderSpellFor is defined as the
	// exact negation of weaponSpellFor's own isMage gate (spell.go), so the two
	// weapon-borne arms cannot both claim one strike and the predicate that
	// separates them is already isMage. Reading it here keeps one source of
	// truth: a second bool parameter through this recorder's three entry points
	// could go out of step with the gate that actually chose the arm.
	o.casts = append(o.casts, CastEvent{
		Caster: a.ID, Target: t.ID, Spell: rule.ID, School: rule.School, Owner: a.Owner, TargetOwner: t.Owner,
		FromX: fromX, FromY: fromY, ToX: t.X, ToY: t.Y, HealthRestored: restored, Weapon: weapon,
		Rider: weapon && !isMage(a), Facing: a.Facing,
	})
}

// recordVictims puts a victim list on the observation recordFromResult has just
// appended. It is a second call rather than a ninth parameter on all three
// recorders: exactly one spell row produces a list, and the two calls sit on
// adjacent lines at that row's only call site (castSpell, spell.go).
//
// A NIL SINK AND AN EMPTY LIST BOTH RECORD NOTHING, so the caller may hand over
// whatever its apply answered without asking.
func (o *castObs) recordVictims(victims []CellPoint) {
	if o == nil || len(victims) == 0 || len(o.casts) == 0 {
		return
	}
	o.casts[len(o.casts)-1].Victims = victims
}

// recordAt is the point-target counterpart of record. Point delivery carries
// no target actor in ROM1, but the drawing tier still needs the two cells and
// the caster identity. Target therefore remains zero while the explicit cell
// is carried in ToX/ToY.
func (o *castObs) recordAt(w *World, ci int, rule SpellRule, x, y int32) {
	if o == nil {
		return
	}
	a := w.entities[ci]
	o.casts = append(o.casts, CastEvent{
		Caster: a.ID, Spell: rule.ID, School: rule.School, Owner: a.Owner,
		AtCell: true,
		FromX:  a.X, FromY: a.Y, ToX: x, ToY: y, Facing: a.Facing,
	})
}
