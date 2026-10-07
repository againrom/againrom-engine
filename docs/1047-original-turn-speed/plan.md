# Plan — story 1047 original turn speed

## Prerequisite gate

1. Wait for the seat to identify the reviewed story-1045 SHA after it lands on
   `master`.
2. Require a clean story-1047 tree. Merge the landed master tip normally with a
   merge commit. Do not rebase, stash or copy individual 1045 files.
3. Measure the merged action lifecycle, form version, entity width and readable
   migration population. Recount facing writers and all upstream attack/cast
   producers.
4. Stop and ask the seat if the next byte-form version is not free. Do not use a
   number inferred from the correction branch.

## Design decisions

### D1 — one canonical turn lifecycle

Add `DesiredFacing uint8` and `TurnRemaining uint8` beside `Facing` on
`sim.Entity`. Keep `Facing` as the current value used by every existing consumer.
The inactive canonical state is `(DesiredFacing == Facing, TurnRemaining == 0)`.

Put shortest-arc arithmetic, positive-rate duration and the nonpositive
compatibility arm in `pkg/sim/facing.go`. The helper accepts an already-derived
desired facing. It does not know whether movement, approach or cast requested
the turn.

The active-turn advancement helper decrements once per eligible actor tick. A
large turn copies desired to current at completion. A small turn copies at
start, then holds its one-tick gate. Both normalize desired to current when
progress reaches zero.

### D2 — movement requests after the production near search

In `pkg/sim/step.go`, advance an existing turn before book-cast, transit or
approach processing. After the near search selects `step[0]`, derive desired
facing and request the common turn before changing `X` or `Y`.

An arc above 32 with a positive rate clears `w.routes[i]` at turn start and
returns without committing the step. An arc at most 32 preserves the route and
returns for its one-tick hold. Equality falls through to the existing cell,
transit, occupancy and route-consumption statements unchanged.

The route-destruction test observes `World.Route`, then completion and the
production replan. It does not call the route helper directly.

### D3 — approach owns attack facing; strike owns none

Replace `approach`'s `face` write with the common turn request. Its existing
closed-on predicate remains the only site that requests attack facing. A zero
delta remains a no-op.

The later attack loop refuses positive turn progress. It also requires the
closed-on producer to have normalized desired and current facing before loading
or advancing action cadence. `advanceAttack` and `resolveBlow` receive no turn
code.

Order replacement clears active turn progress only when the destination or
victim actually changes. Same-victim AI reissue preserves both cadence and turn
progress. Death and invalid-victim normalization clear progress; Stone and
off-map gates freeze it.

### D4 — book admission fixes desired facing and gates final 1045 cadence

After merging 1045, replace the unit and cell admission direct-facing writes
with one common desired-facing request. Keep the admitted target/cell snapshot.
Do not move the existing priority, mana, visibility or target-validity guards.

The final 1045 per-tick book lifecycle must not decrement charge, recovery or
retry progress while the caster is turning. Retained player, AI and autocast
routes must reach this same record. A movement command admitted under `DIV-028`
does not replace the cast's desired facing.

Mutation checks change each unit and cell cast gate independently and require
the producer-equivalence matrix to fail.

### D5 — rate feeds and rearm

Do not add a second RotationSpeed field or data conversion. Exercise the
existing Units, Humans, mapload, equipment-effect, game-rearm and sim-rearm
chain. A turn reads `Entity.RotationSpeed` only at start. `SetDerived` during an
active turn changes the next turn, not the stored remaining count.

### D6 — byte form appends and migration normalizes

After the prerequisite measurement, append desired facing and remaining count
to the current entity record. Rename any live test whose name embeds the form
number before changing the number. Update independent literal widths and
offsets; do not derive expectations from `entityLen`.

The decoder validates the four state classes from the contract. The shared
upgrade path appends `(Facing, 0)` to every migrated entity after all earlier
upgrades have established current facing. It does not patch bytes in a command
or test.

Hash witnesses mutate each new byte separately and use the production form as
the observation. Resume witnesses marshal at every countdown boundary and
compare the next production event tick against an uninterrupted control.

### D7 — producer census fails closed

Add an AST-based facing writer census under `internal/archtest`. Pin runtime
current-facing writes, desired-facing writes and progress writes by function and
class. The scanner reports a new writer and a vanished expected writer. Its own
tests mutate a synthetic file population for both cases.

Constructor and decoder writes remain named. A new direct producer cannot be
accepted by adding its name alone; the diagnostic requires classifying it and
routing it through the common request.

### D8 — presentation reads canonical turn state

Expose only the read needed by the Client, preferably `Entity.Turning()` if the
entity copy already crosses the package boundary. In `pkg/game/world.go`, exclude
a turning actor from swing selection. Movement selection continues to depend on
cell displacement. Standing-frame selection uses current facing and adds no
presentation clock.

The presentation test builds one real `sim.World`, advances the production
turn, and renders through `mapWorld.entityDraws`. It proves current facing and
standing/swing selection at start, mid-turn and completion. A world stepped with
and without the viewer must hash identically.

### D9 — lawful-root withdrawal witness

Extend the existing `missionrun -withdrawal` path or add one narrowly named
turn report in the same command. Reuse its real placed ranged actor and
production withdrawal decision. Record, from world copies and events:

- outward current and desired facing;
- outward remaining count and first retreat-step tick;
- pursuer position at outward start and completion;
- return current and desired facing after reacquisition;
- return remaining count and first attack-application tick;
- pursuer position at return start and completion.

The formatter receives observed state. It does not recompute arc or expected
duration. Synthetic tests pin the output fields. The closure runs it with
`AGAINROM_ASSETS` on EN and RU and records exact lines. No GUI or ROM process is
launched.

## Implementation sequence after 1045 lands

1. Merge and rerun the producer, form and migration census.
2. Add failing story-specific arithmetic, movement, attack, cast and interruption
   tests. Add the failing producer-census tests.
3. Implement turn state and pure arithmetic in `facing.go` and entity
   normalization in `world.go`.
4. Wire movement and attack approach, including route destruction and
   replacement/death/Stone/off-map cases.
5. Wire final-1045 unit/cell cast admission and lifecycle gating.
6. Wire existing RotationSpeed feeds with boundary tests. Change production
   feed code only if a test proves a missing link.
7. Append the form, implement validation and legacy migration, and update all
   independent form witnesses.
8. Wire the Client standing/swing seam and the headless withdrawal report.
9. Reconcile `DIV-429` and the implementation text of `DIV-022`. Add no row
   unless the seat allocates one.
10. Write the as-built `spec.md`, `closure.md` and `verification.md`. Run the
    complete local and lawful-root gates on a clean commit, push the candidate
    and return it for fresh adversarial review.

## Test matrix

| Surface | Independent cases |
|---|---|
| Arc arithmetic | zero; 31, 32, 33 and 128; clockwise and counter-clockwise wrap; exact and rounded-up division; rates 1, 8 and 23; zero and negative compatibility |
| Movement | all eight next-step directions; small route retained; large route destroyed; replan differs; transit starts only at commit; blocked and give-up paths do not turn |
| Attack | all eight victim positions; zero delta; target moves mid-turn; target death/removal/invisibility; charge unchanged until gate; strike path has no facing write |
| Cast | unit and cell forms; player, AI and retained retry; zero delta; mana and visibility refusal write nothing; wind-up unchanged until gate; move attached under `DIV-028` |
| Interruption | replacement move; replacement victim; same order reissue; withdrawal; reacquisition; death/downed; Stone; off-map return; rearm during progress; removal |
| Persistence | inactive; snapped one-tick; large first/middle/last; desired and progress hash differences; malformed quantum/count/residue; every readable legacy form |
| Presentation | standing during turn; no swing; current facing before large completion; snapped facing during small hold; hash independent of viewer |
| Lawful roots | EN and RU placed positive-rate actors; outward and return intervals; pursuer advances in both |

Every expected tick and byte is literal in its test. No expectation calls the
production arc, duration, record-width or field-offset helper it is checking.

## Verification and review

Run the repository build, vet, format, test and check-script glob on the exact
candidate commit. Run the install-gated release and scenario gates on both roots
with `AGAINROM_IMPL` set to this worktree. Run mission 10 and mission 20
unsupported-node counts and record the master baseline and story result in
`verification.md`. Run the lawful-root withdrawal witness on both roots.

Push only a clean candidate. The seat supplies a fresh-context reviewer. The
review has a maximum of three passes and stops at the first pass with no class-P
finding and no remaining surface. The lane does not self-review.
