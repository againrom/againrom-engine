# 0127 — first spell: verification

Branch `0127-first-spell`. Six tasks, one commit each, plus three untrailered
commits: the contract, the task split, and the hero-book fix found at this stage.

## The gate

Run on the committed tree with no game install configured:

```
go build ./... && go vet ./... && gofmt -l $(git ls-files '*.go') && go test -trimpath -count=1 ./...
```

`go build`, `go vet` and `gofmt -l` each printed **nothing** — for `gofmt` that
empty output is the pass, and it is stated because three executors have reported
it clean while leaving files unformatted. `go test` exited 0 with 32 packages
`ok` and no `FAIL`. `bash scripts/check-no-game-assets.sh` printed
`check-no-game-assets: clean (tree scan)` and `--history` printed
`check-no-game-assets: clean (history scan)`. `bash scripts/check-doc-budget.sh`
and `bash scripts/check-sdd-audit.sh` are clean for this story; the audit's note
and warning counts are meaningless from a worktree, which has no `builds/`, so
only its FAIL set is read here.

No commit on the branch carries a `Co-Authored-By` trailer —
`git log --format='%h %(trailers:key=Co-Authored-By)' b93d793..HEAD` prints an
empty field on every line. `git diff --diff-filter=D --name-only b93d793..HEAD`
is **empty**: nothing was deleted.

## What was measured against the lawful installs, before any code was written

Both roots, `world.res::data/data.bin`, and the two agree on every number below.

**`MAGIC-DMG-005`'s corpus test, AC-2.** Re-executing the arithmetic at power 0
(`f = 1`) makes `base .. base+spread` reproduce each row's own
`damageMin .. damageMax`:

| id | spell | column | ours | rival |
|---|---|---|---|---|
| 1 | Fire Arrow | 4..8 | 4..8 ✓ | 4..12 ✗ |
| 2 | Fire Ball | 7..13 | 7..13 ✓ | 7..20 ✗ |
| 3 | Wall of Fire | 1..3 | 1..3 ✓ | 1..4 ✗ |
| 6 | Heal | 8..16 | 8..16 ✓ | 8..24 ✗ |
| 9 | Acid Stream | 10..14 | 10..14 ✓ | 10..24 ✗ |
| 11 | Drain Life | 3..5 | 3..5 ✓ | 3..8 ✗ |
| 13 | Lightning | 5..15 | 5..15 ✓ | 5..20 ✗ |
| 14 | Prismatic Spray | 5..15 | 5..15 ✓ | 5..20 ✗ |
| 21 | Meteor Storm | 5..25 | 5..25 ✓ | 5..30 ✗ |

**9 of 9 ours, 0 of 9 the rival**, on both roots. The same discrimination is a
synthetic test in `pkg/data/spell_test.go`, built from the nine numbers rather
than from the file.

**The castSpell census.** 215 shipped `Humans` rows, **46** carrying
`{castSpell=…}`, and **0 of the 46** with no mana column — so
`MAGIC-ITEM-007`'s fighter-gated melee trigger fires for no shipped person. Two
`Units` rows carry one (`Catapult`, `Ballista`); nothing in `Data.bin` states
their class. `PIPELINE-STATUS.md` had been calling that path the cheapest route
to a first spell; it is not a route at all, and the story built the real cast.

**The `knownSpells` column.** 48 rows state a positive mask; **0** set bit 0,
the id `MAGIC-SPELL-001` says the resolver refuses; **0** set any bit above 28,
where no row exists; **0** sit on a row with no mana column. That is the
discrimination for the bit-per-spell-id reading, and it is ours rather than a
research claim — `provenance.md` says so.

## FR-8: the integer form's divergence, disclosed rather than asserted away

`pkg/sim` admits no float, so `ftol(x × (power/30 + 1))` is integerised as
`x*(power+30)/30`. Swept over power 0..100 × column 0..1000 — 101 101 pairs —
the two forms disagree on **1234**. The first is `power=12, column=45`: the
integer form gives **63**, the float reference **62**, because `12.0/30` is
`0.39999999999999991118` in binary. The test logs the count and the first
disagreeing triple and asserts the measured **1234**, not zero. Every shipped
damage column is below 26, and no shipped disagreement was found in the
9-of-9 reproduction above; the divergence is real and it is stated.

## What was reverted, and what came back red

Every claim below was checked by removing the line and re-running, never by
reading the assertion.

| Line removed | Test | Failure |
|---|---|---|
| `caster.Mana -= rule.ManaCost` | `pkg/sim` cast | `caster's mana is 20 after the cast, want exactly 15` |
| the `+ uniform(spread)` roll | `pkg/sim` cast | `victim's health is 92 after the cast, want exactly 86` and `the generator drew 0 time(s), want exactly 1` |
| `- base` in `spellDamage`'s spread | `pkg/sim` arithmetic | `spellDamage(4,8,0) base+spread = 12, want 8`, 8 further cases |
| `PutUint32(b[o+183:o+187], e.KnownSpells)` | byte form | `the decoded entity's KnownSpells is 0x0, want 0x42` |
| the spell-table count write | byte form | `script section declares 0 check(s) … and carries 34` |
| `Y: int32(spell)` in `castAt` | AC-10 | queue held `Y:0`, want `Y:5` |
| the post-cast selection clear | affordance | `selectedSpell = 1 after the cast fired, want 0` |
| `armed: … \|\| v.selectedSpell != 0` | affordance | `the press produced {…attack:false…}, want one cast by 5 onto 9 naming spell 1` |
| `KnownSpells: p.KnownSpells` (party entity) | party book | `member 0's entity carries book 0, want 266306` |
| `KnownSpells: book` (`assembleParty`) | party book | `KnownSpells = 0, want 266306` |

Two of those tests did not exist until the revert showed they were needed: the
one for the selection clear, and the one for the arming (below).

## The defect this stage found, and how

**Driving a real mission, not a test.** A throwaway harness loaded mission 10
off each install, minted a mage party the way the generation screen does, walked
the caster at the nearest enemy and issued a `KindCast`. The first run reported
`36 entities, 0 with a mana pool, 0 with a book` — the placement path carried
`KnownSpells` and the **party** path never had. Mission 10 places thirty-six
units and not one is a caster, so the only unit a player can select knew no
spell: the story was complete, green, and invisible. `assembleParty` and the
start path now carry the book; both halves are in the revert table above.

## The cast, observed against the lawful installs

Same harness, mission 10, mage party. **Identical on both roots.**

```
party[0] KnownSpells = 266306
mission 10: 36 entities, 1 with a mana pool, 1 with a book, 28 spell rules
nearest enemy is 0 at Chebyshev 12; walking the caster at him
CAST at tick 75: entity 35 (mana 96/96, mind 35) -> 0 at Chebyshev 7
     spell 1 cost 3 school 1 range 7 damage 4..8
     BEFORE mana 96, victim health 15, digest 4a6f60240c127955
     AFTER  mana 94, victim health 11, digest 7fba8bd47020648d
```

Health taken **4**. Mind 35 gives power `clamp(35-30,0,100) = 5`, so
`base = 4*(5+30)/30 = 4` and `spread = 8*35/30 - 4 = 5`: the amount must lie in
4..9, and 4 is the roll's floor.

Mana read **-2** against a cost of 3, which is the one number here that had to be
chased rather than accepted. Three controls, each on the same tick from the same
saved bytes:

- **no command at all** → mana 96. It moves nothing, but it is *blind*: the pool
  is at its maximum, so the regeneration pass has nothing it can add.
- **from a pool the first cast had already opened** → no command leaves 94, a
  second cast leaves 91. Difference **3**.
- **the same full-pool tick with two casts in it** → 91, against 94 for one cast
  and 96 for none. A per-tick contribution is made once whatever the slice
  holds, so one-vs-two gives the per-cast cost directly: **3**.

So the cast subtracts exactly the column, twice measured independently, and the
tick contributed +1 to a pool the cast itself had just dropped below maximum.
That interaction is the regeneration pass's (0109), not this story's, and it is
recorded here rather than smoothed over.

## What only a test says

**Nobody has seen a spell cast in the game window.** `SetForegroundWindow` is
refused from this seat, so no keyboard or mouse reaches an ebiten window: the
spellbook strip has never been drawn to a screen anyone looked at, and the click
that selects a spell has only ever been driven through `Viewer.command` in a
test. What is witnessed is the whole path below the window — the table off the
real file, the book off the real row, the command, the arm, the byte form, the
digest — and the seam between `pkg/ui`'s click and `pkg/game`'s command, in
`pkg/game`'s own test. The pixels are not.

## The milestone

`bash pipeline/check-milestone.sh`'s own drive, run from this branch's binary on
both roots:

```
en  outcome lost at tick 272   census: 4 of 36 unit(s) moved, 1 fell, over 272 tick(s)
ru  outcome lost at tick 272   census: 4 of 36 unit(s) moved, 1 fell, over 272 tick(s)
```

Identical to `pipeline/milestone-baseline.txt`. **Unmoved.**

## The version

**36 was taken and is earned twice**: the entity record widened by
`KnownSpells` (183 to 187 bytes) and the world gained a spell table. The
tripwire in `pkg/sim/corpseloot_test.go` is re-pinned to 36 with the sentence
naming this story; it is neither deleted nor weakened. No test name spells the
number — the round-trip test has been version-free since 0106 DD-11.

## What was cut, and why each

The owner's ruling is breadth before fidelity, so sixteen of `MAGIC-ARM-014`'s
seventeen arms are not built. `spec.md` SC-1..SC-7 name them; the ones worth
repeating here are the ones a reader will otherwise assume are bugs:

- **No projectile.** A cast lands on the tick it is issued.
- **No area.** One spell hits one unit. Note for whoever builds the area half:
  `MAGIC-TARGET-017` says the collector consults no diplomacy, so **every area
  spell burns its own party** — that is the original's behaviour, not a defect
  to design out.
- **No resistance and no skill term in the power.** Neither an elemental
  protection nor a skill level reaches a simulation entity in this build. Both
  are named seams; `School` is carried into `sim.SpellRule` unread precisely so
  the arm that needs it costs no second version.
- **No learning.** A spell is known because the definition said so.
  `teachSpell` occurs **once** in the whole shipped table, in the effect-verb
  vocabulary, and **no shipped item uses it**: the five books and five scrolls
  carry a price and a weight and name no spell. The mechanism is decoded and
  exercised by nothing a player can buy, so nothing here invents what a book
  would teach.
- **No item cast**, measured dead for every shipped person, above.
- **No training on a cast.** `0125` landed the six slots it would credit, so it
  is one call away.

## Left undone, named for a cold reader

1. **The spellbook strip has never been seen.** It is drawn bottom-right, redrawn
   every frame rather than cached like the panel. Its geometry and palette are
   authored and disclosed in `pkg/ui`'s own doc block.
2. **A multi-unit selection under a selected spell issues the same spell id to
   every member**, exactly as a plain attack already does; the per-caster
   refusals sort out who actually casts. Nothing in `pkg/ui` restricts it.
3. **The default party is a fighter.** `defaultChargenAxes` answers "no class
   flag, male", so `missionrun` and any start that skips generation begin with
   PC_Danath, who states no spells. **A player must choose Mage on the
   generation screen to have a book at all.** That is the shipped data's own
   arrangement, not a limit added here, but it is the first thing to say to
   anyone who opens the build and finds no spells.
4. **`pkg/sim/world.go` and `pkg/game/world.go` were both touched here**, and a
   hotfix lane was working in `rearm.go`, `pkg/sim/world.go` and
   `pkg/game/world.go` at the same time. `SetCombat` writes nine named fields in
   place and rebuilds no entity, so `KnownSpells`, `Mind` and the mana pair
   survive a re-arm — checked, not assumed — but the two branches touch two of
   the same files and want sequencing at the merge.

## Where each criterion is witnessed

Each row names the assertion that fails if the thing stops being true. Every one
of them asserts a value; none asserts a sign.

| id | witness |
|---|---|
| AC-1 | `pkg/data` loads 28 spells, ids 1..28, `Fire Arrow` at 1 with cost 3, school 1, range 7, damage 4..8, unit-targeted |
| AC-2 | the 9-of-9 / 0-of-9 table above, as two counted assertions |
| AC-3 | `pkg/sim`: exact mana after and exact health after, against a fixed seed |
| AC-4 | `pkg/sim`: the whole `MarshalBinary` output compared before and after a cast one mana short |
| AC-5 | `pkg/sim`: no mana pool, and mana without the book, each a no-op |
| AC-5a | `pkg/data`: the `-1` cell loads empty; `266306` loads as bits 1, 6, 12, 18 |
| AC-6 | `pkg/sim`: a victim one cell past `MaxRange` costs and takes nothing |
| AC-7 | `pkg/sim`: an unknown id, a non-damaging row and a point-targeted row, three sub-cases |
| AC-8 | `pkg/sim`: a world with a table and a nonzero mask round-trips byte-identical; two worlds differing only in their tables hash differently |
| AC-9 | `pkg/sim`: a form declaring 35 is refused, including a real well-formed version-35 stream |
| AC-10 | `pkg/game`: a nonzero spell appends the cast command exactly, 0 appends the attack; and the book offered is the mask intersected with the table, in table order |
| P-1 | `pkg/sim`: a bystander, the registers, latches, groups and routes are untouched across a cast |
| P-2 | the determinism wall itself — `internal/archtest`'s source scan over `pkg/sim` passes, so no `os`, `time`, `math/rand`, float identifier or float literal exists on this path |
| P-3 | `pkg/sim`: the generator state compared across eleven distinct refusal causes, unchanged in each |
| P-4 | no spell id is compared to a literal in `pkg/sim`, `pkg/game` or `pkg/ui`; the armed spell is a table row's own id and the book is a mask intersected with the table |
| SC-1 | no delivery delay exists: the cast resolves inside the command phase, asserted by AC-3's same-tick health |
| SC-2 | `spellPower` reads Mind alone and no entity carries a protection field |
| SC-3 | the `Effects` column is carried as a string and read by nothing |
| SC-4 | the arm names one victim; no collector exists |
| SC-5 / SC-5a | nothing writes `KnownSpells` after construction; no training, price or monster-cast path exists |
| SC-6 | no item-cast path exists, and the census above says none would fire |
| SC-7 | no book is acquired; a book is the definition's own column |
