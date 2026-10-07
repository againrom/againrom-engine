# Plan — 0104

Three tasks, bottom up, one tier each. T1 gives the data tier a range it can already almost compute.
T2 gives the simulation a reach field, a distance and a version, and changes no placement — so its
own tests build entities directly and the byte-form churn is isolated in one commit. T3 wires the
two together, so every derived value is witnessed through a real definition table.

## The shape of the change

`pkg/data` already resolves a weapon name to its numbers, `Range` among them, and already carries
`UnitDef.Reach` defaulted to 1 with a comment saying no column writes it. `pkg/mapload` already runs
that resolver over a placed person's equipment strings. `pkg/sim` already re-reads an approach's
victim every turn and stops on the same predicate the strike uses. What is missing is three joins
and one field: the unit arm never reads its row's strings, the resolver refuses a ranged row
outright, nothing carries reach into the simulation, and the strike compares against a constant.

## FR by FR

**FR-1** `pkg/data/weapon.go` gains `WeaponRange(name string, shapes, materials ScaleTable, weapons
Collection) (int32, bool)`. It is `ResolveWeapon`'s prefix walk with a different tail: strip, take
the shape prefix, take the material prefix, `findByName`, then `cellOr(p[weaponRangeSlot],
defaultRange)`. Its own length guard is `len(p) <= weaponRangeSlot`, not `weaponRelaxSlot` — a row
that carries a range and no cadence is a row this function can answer.

**`ResolveWeapon` KEEPS ITS RANGED REFUSAL.** That refusal is not a limitation to clean up in
passing: `firstWeapon` in `pkg/mapload/spawn.go` uses it as its "is this cell a weapon" predicate,
and loosening it would hand a person a ranged weapon's numbers down the melee arm — silently, and
in a story about reach. FR-1 exists as a second entry point for exactly this reason.

**FR-2** The `{…}` strip is a small unexported helper in `weapon.go` called as the first statement
of both `ResolveWeapon` and `WeaponRange`. It removes from the first `{` to the end and trims what
is left. `ResolveWeapon`'s behaviour changes only for a name carrying such a suffix, and no name it
is handed today does — the two shipped ones are unit rows, which it never sees.

**FR-3** `data.Combat` (in `hero.go`) gains `Reach int32`. `UnitDef.Combat()` copies `d.Reach`.
`HumanDef.Combat(w *Weapon)` already takes the weapon: it sets `Reach` to `w.Range` when `w` is
non-nil and to 1 otherwise. That is FR-5 in full — no other file changes for it.

**FR-4** `pkg/mapload/spawn.go` gains `unitReach(names []string, t *Table) int32`, a sibling of
`firstWeapon` with the same shape and the same three-collection guard: loop the strings, skip the
empty ones, return the first `WeaponRange` that resolves, and 1 when none does. `definitionFor`
calls it with `c.EntryStrings(r.Index)` and assigns `d.Reach` before returning. The unit collection
carries two strings per row and at most one is ever non-empty, so "the first that resolves" and
"the one there is" agree over the whole shipped table.

**FR-5** Carried entirely by FR-3's change to `HumanDef.Combat`. `blockFor` in `fromalm.go` already
passes the weapon.

**FR-6** `Adjust` in `spawn.go` scales health, damage and nothing else; it needs no change and a
test asserts it leaves reach alone. `blockFor`'s unresolved arm substitutes `data.UnitDefaults()`
whole, whose `Reach` is already 1.

**FR-7** `pkg/sim/world.go`: `Reach uint8` beside `ScanRange`, with a doc comment in the voice of
its neighbours saying what the byte is and why 1 is its floor. `reachFault(e Entity) error` joins
`transitFault` and `decayFault` — one predicate, two callers. **The split follows the package's own
rule, stated above `transitFault`: the constructor NORMALISES and the decoder REFUSES.** So
`newWorld` folds a zero reach to 1 and `reachFault` refuses a stored zero. That split is load
bearing for the size of this story: every entity literal in the package's tests names no reach, and
a constructor that refused zero would turn all of them red for no reason a reader could learn
anything from.

**FR-8** `pkg/sim/combat.go`: `const reach = 1` and its comment go; `strikeDistance(a, t Entity)
int32` takes their place, written as the spec's three lines with the footprint term spelled out and
its two size-1 values named as the constants they are. `inReach` keeps its name and its position
and becomes `strikeDistance(a, t) <= int32(a.Reach)`.

**FR-9** No new call site. The two existing `inReach` tests in `combat.go` — the strike scheduler's
and `advanceAttack`'s — are FR-9 once `inReach` reads the attacker's own field.

**FR-10** No new call site either. `approach`'s stop arm already calls `inReach`; what changes is
the paragraph of its doc comment claiming "a Chebyshev distance of at most one cell".

**FR-11** `pkg/sim/binary.go`: `formatVersion` becomes **23**, `entityLen` becomes **119**, the byte
is written and read at offset 118 of the record, the offset table gains its row and the version
comment gains a paragraph in the voice of the ones above it. `reachFault` is called on the decoded
entity beside the other faults.

## Decisions and divergences

**DD-1 The distance keeps the original's arithmetic, not the arithmetic it collapses to.** At one
cell each the footprint term is zero and the whole function equals `max(1, Chebyshev)`. Writing that
instead would be smaller and would delete the shape the footprint plugs into, so the story that
gives an actor a footprint would have to rediscover it. The equality is asserted by a test over a
range of separations, so the long form is measured against the short one rather than trusted.

**DD-2 This build assigns the weapon's range where the original adds `range − 1` to a reach of 1.**
The sum is the same over every shipped row: a unit row carries at most one equipment string and no
shipped one names two. The divergence becomes visible only when something equips twice, which needs
the equipment channel.

**DD-3 Zero reach is normalised on the way in and refused on the way back.** The original's field is
a byte and its distance floors at 1, so a reach of 0 is an actor that can never strike; nothing in
the corpus produces one. The upper bound is the byte's own.

**DD-4 The group-order layer consults no reach.** The original's second target scorer refuses a
candidate outright once the distance term passes 1, which is why its members never chase. This
build's group layer has no such test today, so widening reach changes no group decision — and adding
it is a group-behaviour story, not this one.

**DD-5 The weapon's cadence columns are NOT applied.** The same weapon row carries a charge and a
relax column, and the original's equip assigns both over the class template whenever the cell is not
empty. A lane implementing this story is holding that weapon and must leave those two columns alone:
they move the attack period on a large minority of the roster, which is a combat-pacing change and
belongs to its own story with its own measurement.

**DD-6 Every actor's footprint is one cell.** The definition tier reads a footprint column and the
simulation has no field for it. For a two- or three-cell actor the original subtracts up to two
whole cells from the separation, so this build's siege engines and dragons strike from nearer than
the original's. Owed by the story that gives the simulation a footprint.

**DD-7 There is no projectile.** A blow at reach 20 lands on the tick it resolves, exactly as a blow
at reach 1 does. Nothing flies, nothing travels and nothing can be intercepted.

## Risk

The version bump moves every literal digest and every byte-form offset pin in `pkg/sim` and
`pkg/mapload`. That is loud, mechanical and confined to T2. The one quiet risk is the tenth
mission's drive: eighteen class rows gain reach, so hostiles that used to close will now stop and
shoot, and the outcome may move in either direction. AC-6 records it; this story does not chase it.
