# 1004 — closure

## Twelve aspects

| Aspect | Verdict | |
|---|---|---|
| Data | PASS | The two trail sheets are loaded by constructed path in `LoadProjectiles`, the same call the front end makes. On the EN install both decode: `smoke sheets loaded: true true`. |
| Runtime state | PASS | `spellBolt` gains `delay`, `seed` and `tag`, all client-side and none serialized. |
| Simulation | PASS | One field on `sim.CastEvent`, `Rider`, naming which of the two weapon-borne arms an observation came from. It is a return value, not world state: `CastEvent` and `castObs` appear in none of `hash.go`, `castbinary.go`, `tailbinary.go` or `scriptbinary.go`, and `castObs` is not a `World` field. `formatVersion` stays 53 and the digest and round-trip tests are unchanged and green. |
| Player input | N/A | No input path is touched. |
| AI | N/A | No decision is touched. |
| UI/HUD | PASS | The figure, the trail and the burst's wait are all in `boltDraws`, whose list the viewer already draws. |
| Triggers/scripts | N/A | No script instant reaches this. |
| Inventory/equipment | PASS | An equipped weapon carrying a `castSpell` releases on the swing, and `weaponBoltDraws` is the second producer of a cast picture. It now hands its object to the same figure and trail producers a book cast uses (FR-7). No inventory rule, slot or tooltip is changed. |
| Persistence/save-load | N/A | No new field is serialized; a save written before this story loads unchanged. |
| Campaign/session | N/A | |
| Shipped content | PASS | On the shipped book exactly two spells draw a path (13, 14), exactly two leave a trail (1, 2), and exactly one has both a travelling cast picture and a burst sheet (2). Measured through `cmd/spellcheck` on the EN install. Every campaign mission on both roots places 79 carriers of a weapon spell, on both arms; see the sweep below, which walks the placed actors rather than counting item rows. |
| Interactions with existing mechanics | PASS | Three interactions. The burst's wait changes when the burst object is drawn, not whether, so the heal feedback, the teleport pair and the area draw are unaffected and their existing tests are unchanged and green. The second is the caster's replacement release, FR-7. The third is the fighter's rider arm, FR-8, which now spawns a cast object and whose burst now waits for it. |

No in-scope GAP.

## Integration witness

`cmd/spellcheck` drives the production loader, the production `observeCasts` and the production
`boltDraws` over the EN install's own art (`game.InspectCastFigure`). It opens no window and
converts nothing to disk.

```
go run ./cmd/spellcheck -assets <install> -spell 13 -cells 8
```

Lightning, all 13 ticks:

```
  age  0 of 13:  22 stamps, 0 trail, 0 burst, (0,0)..(2048,0), max offset 24 units
  age  1 of 13:  19 stamps, 0 trail, 0 burst, (0,0)..(2048,0), max offset 24 units
  ...
  age 12 of 13:  21 stamps, 0 trail, 0 burst, (0,0)..(2048,0), max offset 24 units
```

Every tick spans (0,0) to (2048,0), which is the caster's cell to the target's eight cells away.
The stamp count moves between 19 and 23 tick to tick, so the figure is regenerated rather than
held. The maximum offset from the straight line is 16 to 24 ShotScale units, which is the decoded
±3 pixel clamp.

Fire Ball, spell 2 over eight cells:

```
  age  0 of  5:   1 stamps, 1 trail, 0 burst, (409,0)..(409,0)
  age  4 of  5:   1 stamps, 5 trail, 0 burst, (2048,0)..(2048,0)
  age  5 of 27:   0 stamps, 0 trail, 1 burst
  age 26 of 27:   0 stamps, 0 trail, 1 burst
```

The projectile crosses in five ticks with a growing trail; the burst then runs for exactly 22.

Fire Arrow, spell 1 over eight cells, holds the queue at its bound: the trail grows 1, 2, 3, 4, 5,
6 and stays at 6 for the remaining four ticks.

## The weapon-borne producer on the real install

`game.InspectWeaponRelease` drives `weaponBoltDraws` over an attacker holding a staff whose
`castSpell` is the given spell, mid wind-up, with its victim eight cells away, on the EN install's
own art. It opens no window and converts nothing to disk.

```
go run ./cmd/spellcheck -assets <install> -spell 13 -cells 8
```

Lightning from a staff, every swing position of an eight-tick charge:

```
  swing  0 of  8:  22 stamps, 0 trail, (0,0)..(2048,0), max offset 24 units
  swing  1 of  8:  19 stamps, 0 trail, (0,0)..(2048,0), max offset 24 units
  ...
  swing  8 of  8:  23 stamps, 0 trail, (0,0)..(2048,0), max offset 24 units
```

Fire Arrow from a staff, the same charge:

```
  swing  0 of  8:   1 stamps, 1 trail, (0,0)..(0,0), max offset 0 units
  swing  5 of  8:   1 stamps, 6 trail, (1280,0)..(1280,0), max offset 0 units
  swing  8 of  8:   1 stamps, 6 trail, (2048,0)..(2048,0), max offset 0 units
```

The staff's Lightning stamp counts match the book cast's tick for tick, because both producers seed
the same walk from the same three ids. The staff's Fire Arrow travels and fills its six-entry queue,
which is what the book cast does. Before this round both printed one stamp and no trail.

## Weapon-borne sweep

A cast picture has three producers, not two. `boltDraws` walks the client's cast objects, which come
from a book cast. `weaponBoltDraws` rebuilds a caster's replacement release from the attacker's
attack cycle every frame. The third is the fighter's rider arm, which produces no live draw at all
and needs a cast object.

**The arm is a property of the CARRIER, not of the item.** `weaponSpellFor` refuses unless the
carrier has a mana pool; `weaponRiderSpellFor` is its exact negation. A staff row in a data file
therefore does not say which arm it reaches: the same staff in a mage's hands takes the replacement
arm and in a fighter's takes the rider. Counting item rows cannot answer the question, and round
two's sweep counted item rows.

`cmd/weaponspellcheck -scan` walks the placed actors of every campaign mission through
`game.StartMission` and reports each carrier's own mana pool and the arm it selects. Both install
roots, all 28 campaign missions:

| Arm | Carriers per root | Spells carried | Drawn before this story | Drawn now |
|---|---|---|---|---|
| Caster's replacement (`MaxMana > 0`) | 76 | 1, 13, 14, 20 | one interpolated sprite, no figure, no trail | FR-7: the figure, the trail, the phase |
| Fighter's rider (`MaxMana == 0`) | 3 | 2 (Fire Ball) | nothing at all, with the burst still appearing at the target | FR-8: a cast object that travels, with a trail, and a burst that waits for it |

The three rider carriers, identical on both roots:

```
mission entity  spell level  mana      arm
     90    152      2    70     0/0    rider (fighter)
    111      0      2    40     0/0    rider (fighter)
    140    165      2    40     0/0    rider (fighter)
```

That count independently reproduces `pipeline/LOG.md`'s `1001` entry, which records three shipped
Boulder Throwers throwing the Fire Ball they were always carrying. It is measured here through
production mission loading rather than read from the ledger.

**The item rows, corrected.** Round two reported 48 `castSpell` rows per root over 10 distinct staff
names. The instrument was a name-extracting expression that assumed a space before the brace, and
two rows have none. Re-counted by group over the pinned research tree's brace scan
(`research/experiments/EXP-0142-item-names/evidence/databin-brace-scan.csv`):

| Group | Rows per root | Distinct names | Spells |
|---|---|---|---|
| `F humans` / `Humans` (items) | 46 | 8 staff names | Fire Arrow 22, Lightning 14, Prismatic Spray 8, Stone Curse 2 |
| `E units` / `Units` (unit templates) | 2 | `Boulder Thrower` | Fire Ball 2 |

So the staff count is **8**, not 10 and not 13. The two Fire Ball rows are not staves at all: they
are unit templates, and they are exactly the carriers that take the rider arm. Round two's closure
offered those two rows as evidence that the figure and the trail reach a staff, which they do not —
100% of its Fire Ball evidence was the one carrier the round-two fix could not reach.

Two facts about this sweep are outside this story's contract and are stated rather than changed.
`weaponBoltDraws` gates on `data.CastFlies` alone, so it draws a release the simulation refuses to
apply (`weaponSpellApply` refuses ids 14 and 25); that gate is unchanged by this story and predates
it. And `gameversions/` was read through the ordinary loader and not written to.

## Revert witnesses

The path fork removed from `boltDraws`:

```
--- FAIL: TestAPathLeavesTheStraightLine
    every stamp of every tick stood on the straight line — the figure is an interpolation
--- FAIL: TestAPathPictureStandsStillAndSpansItsWholeSegment
    at age 0 the figure is 1 stamps, want a path
```

`burstDelay` forced to 0:

```
--- FAIL: TestFireBallsBurstWaitsForItsProjectile
    the burst waits 0 of 6 ticks and lives 22, want 6 and 28
```

The FR-7 routing removed from `weaponBoltDraws`, leaving the single `spellDraw` call it had before:

```
--- FAIL: TestAWeaponBorneLightningDrawsTheSameFigureABookCastDoes
    at swing 0 the staff drew 1 stamps, want a path
--- FAIL: TestAWeaponBorneFigureIsRegeneratedAcrossTheSwing
    4 of 4 swing ticks redrew the previous figure unchanged
--- FAIL: TestAWeaponBorneTravellingPictureLeavesItsTrail
    at swing 0 the staff left 0 trail puffs, want 1
--- FAIL: TestAWeaponBorneReleaseAgreesWithABookCastOnWhatEachPictureDraws
    spell 13: a staff drew 1 cast stamps, figure=false, want figure=true
    spell 14: a staff drew 1 cast stamps, figure=false, want figure=true
    spell 1: a staff drew 0 trail stamps and a book drew 2
```

All four redden, and the fourth compares the two producers by running each one rather than by
restating what either does. `TestAWeaponBorneFigureIsRegeneratedAcrossTheSwing` did not redden on
its first form, which compared whole draw lists: a single travelling sprite also changes position
each tick. It now compares the figure's interior points, which do not exist without the figure.

The rider's own spawn, `observeCasts`' `|| ev.Rider`, put back to the form that skipped every
weapon-borne event:

```
--- FAIL: TestARiderReleaseSpawnsAnObjectThatFlies
    the rider's object never moved — it is a travelling picture and must cross
```

`burstDelay`'s `&& !ev.Rider`, put back to returning 0 for every weapon-borne event:

```
--- FAIL: TestARiderBurstWaitsForItsProjectile
    a rider burst waits 0 ticks, want its projectile's flight
```

The two are witnessed separately because they are two lines and two halves of the defect: with the
first alone the burst appears with nothing having flown to it, and with the second alone the
explosion fires while the fireball is still in the air.

The rider tests build a real world and take a real `StepObserved`, rather than writing a
`sim.CastEvent` by hand. Which arm the simulation chooses is the whole of what FR-8 turns on, and a
hand-written event would assert the choice instead of observing it.

The lines were removed and restored from a copy outside the repository, not with `git stash`: the
stash is one ref shared by every worktree of this repository.

## Research reconciliation

Implemented as decoded: the two-picture gate, the walk's structure and every one of its constants,
the whole-list replacement, the stationarity and the 13-tick countdown, the empty end, the stamp at
each point, picture 36's `tag * 5`, the six-entry trail with its oldest-first expiry, and the trail
art's address outside the registry.

Three mismatches remain and each is a ledger row:

- `DIV-080` — the walk's parametric reading is `MAGIC-BOLTSHAPE-070`'s own Medium, the list's
  origin endpoint is its Unknown, and the midpoint insertion and stamp spacing are authored.
- `DIV-081` — the thirteen frame-ramp values, the trail's frame and the chain victim set are not
  published; no claim states whether the original draws a weapon-borne release through the same
  projectile arm as a book cast, on either of the two arms; and no claim names the record tag a
  weapon-borne release carries, so this build uses the attacker's own id modulo seven. Drawing all
  three producers the same way is the owner's directive read forward, not a decoded fact.
- `DIV-082` — the burst's wait is a client-side correction for a simulation that carries no flight
  time.

## Census

Run at the pushed sha, after both master merges:

```
go build -o mr ./cmd/missionrun
AGAINROM_ASSETS=<en install> ./mr -mission 10 -trace -ticks 1 | grep -c UNSUPPORTED   # 0
AGAINROM_ASSETS=<en install> ./mr -mission 20 -trace -ticks 1 | grep -c UNSUPPORTED   # 0
```

Both are 0, which is what master carried before this story. The count is unchanged, which is the
expected result: nothing here touches a script node. This story's result is in `builds/current/`,
not in that census.

## Master merges

Two landings arrived while this story was open and both were merged before the re-push.

`a9322c8`, story `1003`, area overlays. Only `docs/DIVERGENCES.md` conflicted, on adjacent rows:
master's `DIV-077` through `DIV-079` against this story's `DIV-080` through `DIV-082`. Both sides
were kept in id order.

`246a7ee`, story `1002` (effect marks on the actor) and one ledger commit. No conflict:
`docs/DIVERGENCES.md` auto-merged because `1002` edited only `DIV-073`'s own columns, and the two
files this story might have met — `internal/archtest/dag.go` and `pkg/game/world.go` — are untouched
by it. This story adds no `cmd/` package, so the fail-closed allow-map needs no entry. After both
merges the ledger holds 69 rows, `DIV-073` through `DIV-082` in order, no id twice.

`1003` made `ui.EffectGroundPoint` the single owner of a cell's centre: it adds `terrain.CellSize/2`
on both axes and subtracts the sheet's centre, so a producer must supply no half cell of its own.
Checked rather than assumed. Every `ui.SpellBolt` this story produces goes through
`spellArtPlacements`, which applies `EffectGroundPoint` once. `boltPath` converts a cell to
`ui.ShotScale` units with no half-cell term, and `trailDraws` positions through `shotPoint`, which
has none either; a search of `pkg/game/spellbolt.go`, `pkg/game/spellpath.go` and
`pkg/game/world.go` finds no `ShotScale/2` outside comments. `pkg/game/areaoverlay1003_test.go`'s
own centring tests are unchanged and green.

## Gate

`go build`, `go vet`, `gofmt -l`, `go test -trimpath -count=1 ./...` (all packages ok) and
`scripts/check-no-game-assets.sh` all clean on a clean tree at the pushed sha. The research
submodule pin is unchanged at `1875e6d`.
