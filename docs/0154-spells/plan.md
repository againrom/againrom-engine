# 0154 — two spells complete: plan

One lane implements this slice in its own context, so there is no `tasks.md`:
every `FR` below is built here and witnessed in `verification.md`.

## Shape of the change, by tier

### `pkg/data` — the table loader (FR-1)

`Spell` gains `Restorative`, set beside `Damaging` in the one expression that
already names the heal id. The two flags are complements over the heal row and
are never both true. `drainLifeSpellID` keeps refusing both.

### `pkg/sim` — the rule and the state (FR-2, FR-3, FR-4, FR-5, FR-7, FR-8, FR-9)

`SpellRule` gains `Restorative`, carried through `normaliseSpells` unchanged (it
is a flag, not a number, so it has no shape to refuse).

`Entity` gains four fields, all on the record's own tail so no offset in front of
them moves:

| field | width | meaning |
|---|---|---|
| `AutoSpell` | uint16 | the spell cast unbidden, 0 for none (FR-7) |
| `AutoCastWait` | uint8 | ticks before the next unbidden attempt (FR-7) |
| `SpellFX` | uint8 | ticks the effect mark still stands (FR-4) |
| `SpellFXSpell` | uint8 | which spell set it, 0 for none (FR-4) |

Record width 217 -> 222; `formatVersion` 44 -> 45.

`castSpell` forks after its shared refusals into the damage arm it already has
and a new restorative arm (FR-2, FR-3). Both end at one shared tail that pays the
mana, sets the marks (FR-5) and pays the award — so the two arms cannot come to
disagree about the economy.

`stepWorld` gains one phase before commands are applied: the autocast sweep
(FR-8), walking entities in world order, and the mark decay (FR-4). Target choice
is FR-9's two rules in one function.

### `pkg/mapload` — the table (FR-1)

`spellsFor` carries `Restorative` across into `sim.SpellRule` beside `Damaging`.

### `pkg/game` — the seam (FR-6, FR-10, FR-12, FR-13)

- the per-tick entity push carries the mark and the autocast id onto `MapEntity`;
- the spellbook push carries, per entry, its popup lines and whether it is the
  unit's autocast spell;
- the autocast toggle is a new seam function, `MapAutocast`, taking a unit and a
  spell id.

### `pkg/ui` — what the player sees and does (FR-6, FR-10, FR-11, FR-12)

- `SpellEntry` gains `Info []string` and `Autocast bool`;
- a spell popup, composed by the item popup's own builder;
- a dashed border on an autocasting cell, phase from `v.anim.Count()`;
- a spell-effect ring on a marked unit, in the school's colour;
- `Ctrl+A` in the app input, routed to the toggle.

### `pkg/game` — the scenario language (FR-14)

`HeadlessWorldSpec` gains a `spells` list and `HeadlessUnitSpec` gains `max_hp`
and the caster's four fields. `HeadlessStep` gains `spell`, and the order verb
set gains `cast` and `autocast`. `HeadlessUnitAssertion` gains `mana_at_least`,
`mana_at_most` and `autocast`. One scenario file ships with the story.

This arrived from `0155-headless`, which landed on master mid-story; the merge
is what made a drive cheaper than a second harness.

## Order of work

1. `pkg/data` + `pkg/mapload` — the flag, end to end, with its tests.
2. `pkg/sim` — the entity fields, the byte form, the digest, then the restorative
   arm, the mark and the autocast sweep, with tests.
3. `pkg/game` — the three pushes and the toggle seam.
4. `pkg/ui` — the popup, the dashed border, the ring, the key.
5. `pkg/game` again — the scenario vocabulary and the shipped scenario.
6. `builds/0154-spells` — the binary and the README naming `Ctrl+A`.

## Design decisions

**DD-1** The autocast is **simulation state, not front-end state**, and that makes
this story threshold High. A toggle held in the front end would be lost on save,
would not be covered by the digest, and would put a target-choosing rule in the
drawing tier where the movement law already refuses to go. Held in the world it
replays exactly, survives a save, and reaches the same cast arm a command does.

**DD-2 (AUTHORED)** **What a heal restores is ours.** No published claim states
the heal arm's magnitude. The row's own damage columns at the caster's own power
are used, because those columns are the only per-row magnitude a Heal row carries
and the same arithmetic already serves every damage row. The divergence, in the
reader's units: *a heal in this build restores about as much as Fire Arrow takes
off, and grows with the caster's Mind and his level in Life magic. If the original
heals by some other rule, this heals by the wrong amount — it never heals the
wrong unit, and it never heals across a battle line.*

**DD-3 (AUTHORED)** **The autocast toggle is the owner's, and everything under it
is ours.** The original's autocast is a property of a weapon carrying a spell,
which this tree already builds; a toggle on a book spell is a new affordance the
owner asked for. `Ctrl+A` is ours, the acting-on-the-selected-spell shape is
ours, the target rules of FR-9 are ours, and the wait is ours. The divergence, in
the reader's units: *no original save carries this setting, so a save written here
carries a switch the original game has no way to read, and a unit's autocast is
lost when its save is opened by the original.*

**DD-4 (AUTHORED)** **The rotating dashed border is the owner's design.** Its
period, its dash length and its colour are ours. It is drawn on the book cell and
nowhere else — not on the unit, not on the panel — because the cell is where the
player set it.

**DD-5** The mark is **per-entity, not a list of map objects**. The decoded even
picture creates no map object at all and gives the *caster* an action, so the
state the fork actually implies is on the actors. A list would be the odd
picture's shape, and the odd picture is SC-2.

**DD-6** The wait is reset only by an **applied** cast. Resetting it on a refusal
would make a caster with a target briefly out of range go quiet for the whole
period after it stepped back into range, which reads as the toggle not working.

**DD-7** The popup's lines are composed on the far side of the seam, beside the
item popup's own lines, rather than in the drawing tier from a widened entry.
The alternative — pushing cost, range, school and band as separate fields — puts
five numbers across a seam that one list of strings already crosses for items.

**DD-8** `Ctrl+A` rather than a bare letter. The owner asked for a combination,
`A` alone is free but a bare letter is one keystroke away from an accident that
silently spends a unit's whole mana pool, and every bare letter in this build is
already a toggle of something visible rather than something that costs.

## Success criteria

**SC-1** The gate is green from the worktree on a clean tree: `go build`,
`go vet`, `gofmt -l`, `go test -trimpath -count=1 ./...`, and
`check-no-game-assets.sh`, `check-doc-budget.sh`, `check-sdd-audit.sh` and
`check-hotfix-ledger.sh`.

**SC-2** `go test ./...` is green with no game install present. Every fixture
this story adds is authored in test code or in a `synthetic` scenario.

**SC-3** The shipped scenario, run the way the owner runs the game —
`againrom --headless scenarios/0154-synthetic-spells.json` — exits zero with no
`-assets` flag and no install, and its trace shows the heal, the two costs, the
toggle and an enemy felled by unbidden casts alone.

**SC-4** The script-gap census for missions 10 and 20 is measured on both sides
of the change and the two numbers are reported whether or not they moved.

**SC-5** Reverting the diplomacy clause in `castSpell` makes
`TestAHealAcrossAHostileRelationOrAtACorpseLeavesTheWorldByteIdentical` fail, and
reverting the modifier guard in `readInput` makes `TestTheAutocastKeyIsNotAlsoAPan`
fail. Both were checked by reverting rather than by reading the assertion.

**SC-6** The cuts hold: no third cast arm, nothing in flight, no resistance, no
`Effects` parse, no learning, no item cast, no shipped description text read, and
no autocast set on a unit the local player does not own.

## Risks

**R-1** The version bump is the sharp case every bump is: a version-44 buffer's
first record reads intact and every record after it reads four bytes into its
neighbour. Nothing but the version byte separates a correct decode from that, and
the refusal is what stands in for a migration.

**R-2** The autocast sweep runs before commands, so a player's own command on the
same tick lands after an unbidden cast and may find the mana already spent. That
is the intended order — an autocast is the unit acting on its own, and the player
overriding it is the later act.

**R-3** `Ctrl+A` must not collide. The build's existing bindings are checked
against it before it is wired.

## The second round

The four defects the owner reported, and where each is answered.

### R-3 — the observation is a return value, not state

FR-16 needs the caster and the target of an applied cast on the far side of the
simulation wall. Three shapes were considered:

1. **a field on `Entity`** naming who marked it. It is the shape a front end
   would find easiest to read, and it grows the record, bumps the byte form to 46
   and puts a presentation concern into hashed state;
2. **a field on `World`** holding the tick's events. Invisible to the byte-form
   and digest pins by construction, which is exactly what `nostate_test.go`'s
   literal field table exists to refuse;
3. **a value threaded down the step and returned**, which is `ScriptTrace`'s own
   shape and already sanctioned in this package.

The third is built. `castObs` is threaded from `stepWorld` to the two routines
that apply a spell, records nothing on a nil receiver, and is reached by
`StepObserved` alone. `Step` passes nil, so the two entry points are one code
path and an observed run is the same run.

The two recording sites are the two places a spell lands: `castSpell` past every
refusal, and `releaseWeaponSpell` past all five of its own. Nothing else in the
package applies a spell, so nothing can land unobserved.

### R-4 — FR-19's bolt has two producers and they are not the same shape

A weapon-borne cast has a wind-up the front end can already read: the caster
stands in the casting phase with a countdown, and `pkg/game` already places an
archer's shot from exactly those two numbers. So the bolt is interpolated live,
reaches the victim on the tick the release resolves, and is remembered nowhere.

A book cast has no wind-up: it resolves inside the command phase of the tick it
arrives on. So its bolt is spawned from the observation and flies afterwards,
held in `mapWorld.bolts`, which is presentation memory ruled cosmetic in 0143
FR-5 and dropped by a resume.

Both cross one seam. `ui.SpellBolt` carries a point in `ShotScale` units, the
caster's cell for the relief lift, the caster's owner for the fog gate, the
school for the colour, and a burst step; `pkg/ui` derives nothing further. A
burst step of zero is a travelling bolt and a positive one is a ring of that
width, so one value distinguishes the two and the drawing tier decides nothing.

### R-5 — FR-20's swing clock is one predicate in one place

`advanceSwings` tested `AttackCharging` in two places: the run restart and the
swing sound. `windUp` names both wind-up phases once, and both sites read it, so
the two cannot come to disagree about which phases are a swing. The divert case
— an attacker loaded toward one wind-up and now eligible for the other — is a
change between the two phases and restarts the run, which is right: the actor
returns to ready and loads a fresh wind-up on the next advance.

### R-6 — the heal arm is a priority list, not a branch

FR-18 has four clauses that interact. Written as nested conditions they would be
read once and maintained never. `autoCastOrder` returns the rows to try in
priority order and `autoCast` walks it, so each clause is one line of one list
and the fall-through is the loop rather than an extra rule. The list is empty for
an entity with nothing armed and an empty book, which is the cheap first test and
the state almost every entity is in.

Which row heals is the table's own `Restorative` flag throughout, so P-4 stands:
no spell id is compared to a literal anywhere in the simulation package.

### R-7 — FR-21: the toggle is a setting

`setAutocast` marked the unit commanded, which is the front end's permanent "the
player has taken this unit over" and drops that unit's scripted command track for
the rest of the mission. The command still travels the replayable queue; only the
mark is removed.

### R-8 — FR-15 and FR-17 are one-line restorations, and the choice is where

Both defects are a value that was never carried rather than a mechanism that was
never built, so the whole design question is which side of a seam restores it.

**FR-15**, the restored spellbook. `restoredDefinition` already builds the
`data.HumanDef` the book lives on and already returns five values off it; the
book becomes the sixth. The alternative — reading the save's own `Spellbook` — is
not available: `sav` decodes the presence flag and the element count and stops,
and what a `Spell` record's `+0x08` names is an open Unknown. So the row is not a
fallback chosen for convenience, it is the only source that exists, and spec DD-8
discloses what that costs.

**FR-17**, the weapon-borne mark. `releaseWeaponSpell` is the second of the two
routines that apply a spell, and it took neither the mark nor the observation.
Both statements go at its tail, past all five of its refusals, on `castSpell`'s
own precedent — so "applied" means the same thing on both paths and a release
cannot leave a half-recorded cast.

### R-9 — FR-22 and FR-23 put the run's clock in the front end, not the world

A cast run is a picture. It is per-entity, it is not hashed, and it must survive
frames that carry no tick, which is exactly `mapWorld.swing`'s own shape one
field over. So `castRun` joins it there and is ruled cosmetic in 0143 FR-5: a
resumed world simply draws no run until the next cast.

**FR-22** needs the run open for a caster that holds no victim, because a book
cast sets no attack target and the drawn-swing gate reads one. Widening that gate
to ask `casting(e.ID)` beside `HasAttackTarget` is the whole change; the frame
selection underneath it is `SelectAttackFrame`, untouched.

**FR-23** scales the run rather than retiming the simulation. The alternative,
holding a cast in flight so the damage lands when the picture arrives, is world
state and a byte-form version bump, and it would have bought a worse answer: the
cadence would still be the weapon's. Scaling is arithmetic in one function,
`scaleRun`, so a cast cadence and a swing cadence cannot come apart. It is
applied to a cast alone — a melee blow keeps its one-frame-per-tick clock,
because the owner's rule is about magic projectiles and widening it would change
a picture already accepted.

### R-10 — FR-24's floor is a refusal in the simulation, not a filter in the draw

The floor could have been held in either tier. Held in the front end it would
drop pictures, which is the swing-without-a-projectile FR-23 forbids in the other
direction; held in the simulation it holds the CAST, which is what the owner
asked for. It is therefore a refusal in `castSpell`, before the cost, so a
throttled cast spends nothing and rolls nothing and P-3 still stands.

It needs no new state. `AutoCastWait` already carried a per-entity cast wait at a
byte of the record; it is renamed `CastWait` because it is now every cast's wait
and not the unbidden one's, and the separate `autoCastPeriod` is deleted rather
than kept beside it — two periods would be two rules for one cadence. Same
offset, same width, same byte form, and no version bump.

A weapon-borne release is deliberately outside it. Its cadence is the attack
cycle's own wind-up, which always has a swing behind it, so the floor would only
delete projectiles that already had one.
