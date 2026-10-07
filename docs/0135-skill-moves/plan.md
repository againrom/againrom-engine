# 0135-skill-moves — plan

## Shape

| # | Area | What it settles |
|---|---|---|
| T1 | `pkg/sim` state + byte form, `pkg/data`, `pkg/mapload` | FR-1..FR-4, FR-14, P-1, P-2 |
| T2 | `pkg/sim` award sink and the two feeds | FR-5..FR-11, P-2, P-3 |
| T3 | `pkg/game` re-derive and the announcement | FR-12, FR-13 |

## The decisions

**DD-1 — the level lives on `sim.Entity`, not beside it.** FR-1 makes it state
the tick reads: FR-8's raise happens inside a blow's own resolution and FR-11's
power reads it at a cast. A field on the entity is therefore the only place it
can be, and FR-2's "canonical" follows: byte form, digest, replay. It is
`[6]int32`, matching the width `pkg/data` already carries a level at, so no
conversion sits between the two.

**DD-2 — `S(n)` becomes a 101-entry `int32` table inside `pkg/sim`.** The curve
needs `pow`; `pkg/sim` bans floats (P-2), and it may not import `pkg/data`. The
alternative — hoisting the comparison out to a caller — was rejected because it
puts a per-blow decision outside the determinism wall, where two peers could
answer it differently. P-1 is the mechanism that keeps the table honest: the
forward curve is **exported once** from `pkg/data` (`SkillXPFor`), used by
`Recompute` and by `SkillLevelFor`, and compared against the table from
`pkg/mapload`, which may import both.

**DD-3 — the class discriminator is the mana pool, and it is the one already
there.** `isMage` (`pkg/sim/spell.go`) is `MaxMana > 0`; the game's own class
bit is set from the mana column being positive. FR-7 therefore adds no field
and no second predicate. A `Fighter` flag on the placement path was rejected on
that ground: it would be a second answer to a settled question.

**DD-4 — one sink, two feeds.** FR-5 to FR-8 are one function,
`(*World).awardSkill(ai int, named int32, amount int64, srcIdx int)`, and both
feeds call it. The refusals live there and not at the call sites, so "what a
blow pays" and "what a cast pays" cannot come to disagree about the gates. The
blow feed passes `named = 0` — the value the original's own weapon-arm caller
passes — and lets FR-7 do the routing; the cast passes the school.

**DD-5 — the sink returns whether a level moved, and nobody in `pkg/sim` reads
it.** FR-13's detector is in `pkg/game`, which holds no per-tick callback into
the simulation, so it compares the levels it last posted against the live ones
— `rearm`'s own compare-and-act shape, but over **every** character the load
knows rather than the inventory subject alone, because FR-13 announces a raise
wherever one happens. The return value exists for the package's own tests,
which is where the raise itself is pinned.

**DD-6 — FR-12 rides the existing door.** `mw.rearm` (`pkg/game/world.go`) is
the front end's one path to `SetCombat`, which is the only writer of derived
numbers onto a live entity; it runs once per map-screen frame and today acts
only on an equipment change. It gains a second trigger — the subject's own six
levels — and one re-derivation serves both, seeded with the levels read off the
entity. Nothing new is opened: widening the door to the pools, the sight and
the protections is its own story, and FR-12 names that limit rather than hiding
it. `refreshEquipment` is a different function and is not the one meant.

*Restated at the merge with `0136-armour-counts`, which moved the resolve, the
fold, the recompute and the `SetCombat` out of `mw.rearm` into an exported
`Rearm` in a new `pkg/game/rearm.go`.* The trigger and the seeding stay in
`mw.rearm`; `Rearm` takes its hero **by value**, so the levels are overridden
on a local copy at the call site and `Rearm` itself needs no argument, no field
and no knowledge of this story. The decision is unchanged and the seam it rides
is better than the one it was written against: a raise now re-folds the whole
worn set, not the weapon alone.

**DD-7 — the announcement is a `PickupRow` with the level in its text.** The
row type carries a `Count`, and `Count` renders as `" xN"` — "Blade x11" reads
as eleven blades. So the level goes in `Text` ("Blade 11") at `Count 1`, which
keeps `pkg/ui` naming nothing, exactly as it names no item code. The skill's
name comes from the mage set for an entity with a mana pool and the warrior set
otherwise — FR-7's own discriminator, not a second one.

**DD-8 — the placement path grows a sixth definition-borne number.** A placed
person's levels come off the same `Recompute` his combat block already comes
off, carried on the block struct beside the mana pair. A creature and an
unresolved placement carry zeros, which is what their rows say.

## Reconciliation — deliberately not a numbered decision, and not a task

Two landed contracts state things this story falsifies, and they are corrected
**in place, keeping their ids**, in the same commit as this story's own
artifacts: `0125` FR-8, FR-10, FR-11, FR-14, AC-6 and two of its cuts; `0127`
FR-3, FR-7 and SC-2. Minting a new id inside a landed story costs a line in
that story's plan **and** its tasks for every id, for no reader's benefit; the
repo's own precedent for a cross-story correction is the fold clause already
inside `0125` FR-13 and FR-14.

It carries no `DD-` number and no task on purpose. A `DD-` must be named by a
task, a task must be a commit, and a commit carrying a task trailer must touch
a file outside `docs/` — so a document-only reconciliation cannot be one
without lying about what it changed. Doing it here instead means the tree never
holds two contracts that disagree, not even for one commit.

**Do not spell another story's `FR-n` inside that story's own files.** The
audit extracts ids by pattern per folder, so `0135`'s `FR-11` written into
`0127/spec.md` mints a phantom `FR-11` for `0127` that no plan accounts for and
no task carries — measured, two FAILs, and the same shape `0130` recorded.

## Risks

**R-1 — the cap silently changes what a blow pays.** FR-6's cap is new to the
blow feed. At a level-0 slot it is 100, which is above every amount mission 10
produces, but it is not above every amount. Measured rather than assumed: T2
pins the boundary case.

**R-2 — a level in the byte form is 24 more bytes per entity.** Every pinned
digest in `pkg/sim` moves. That is the version bump's own cost and the pins are
re-taken; the check that it is *only* that is the pre-story-pin-plus-the-block
test the package already writes for every such crossing.

**R-3 — `Skill.General` on a placed person is 10 to 99.** Storing it makes it
visible in the panel where a derived level showed 0 before. That is the row's
own number and the correct one; it is called out because it changes what the
panel prints for every placed person on every map.

## Success criteria

**SC-1** The tenth mission driven on both lawful roots from `cmd/missionrun`
prints the party member's six levels before and after, and one of them moves.

**SC-2** The full gate is green: build, vet, gofmt, `go test -count=1
-trimpath ./...`, and the four repo scripts.

**SC-3** Reverting the raise statement alone makes a named test fail, and
reverting the slot-0 refusal alone makes a different one fail — the witness is
the failure, not the assertion.

## Traceability

| FR | DD | SC |
|---|---|---|
| FR-1, FR-2 | DD-1 | SC-2 |
| FR-3, FR-4 | DD-8 | SC-1 |
| FR-5, FR-6 | DD-2, DD-4 | SC-2, SC-3 |
| FR-7 | DD-3, DD-4 | SC-3 |
| FR-8 | DD-2, DD-4, DD-5 | SC-1, SC-3 |
| FR-9, FR-10, FR-11 | DD-4 | SC-1, SC-2 |
| FR-12 | DD-6 | SC-2 |
| FR-13 | DD-7 | SC-2 |
| FR-14 | DD-1 | SC-1 |
| P-1 | DD-2 | SC-2 |
| P-2 | DD-2 | SC-2 |
| P-3 | DD-4 | SC-3 |
