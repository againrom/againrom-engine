# Analysis — health in the world, and the two bodies a killed unit leaves behind

## Intensity & terrain

| Axis | Declaration |
|---|---|
| Intensity | **spec-anchored / static** — the profile names the save format and simulation determinism at this tier, and this story raises both; no watcher tool exists, so it is discipline |
| Terrain — the health fields, the three states, the two new orders, the health bar | **greenfield**: none of it exists in any spelling |
| Terrain — the byte form, the digest, the occupancy seed, the move loop, the front-end selection and its seam | **brownfield**: every one is shipped, pinned, and touched here |

## Three baselines were folded, and the fold removes a field rather than adding one

Taken apart, `0033` adds a `Dead` bool at one codec version and `0034` deletes it at the next.
Merged, the deleted field never exists: **`HP` and `MaxHP` are the whole of the state and one version
bump carries them**. `0039` rides along because it consumes that same state from the front-end.

## The baselines name a tree that is not this one

Each sentence was re-derived against the code rather than translated:

| The baselines say | What is actually here |
|---|---|
| codec **v7**, then **v8**; `unitRecordLen` 35 becomes `35 - 1 + 8 = 42` | the shipped version is **4** (`pkg/sim/binary.go:19`), so the next is **5**, and the record is `entityLen = 26` becoming 34 |
| `Unit.Dead`, `Unit.HP`, `StallTicks`, hashed "**like `Flying`**" | the type is `sim.Entity` with `Stall`; **no `Flying` field exists anywhere in the tree** |
| a command kind `cmdKill` beside `cmdMoveTo`; `amount` rides `Command.X` "mirroring how `cmdMoveTo` reuses `X`/`Y`" | `Command` is `{Entity, X, Y}` and says in its own doc comment that **there is no kind discriminator**; neither name exists |
| `pkg/game` reads the state at three sites — `unit_sprites.go`, `sim_overlay.go`, `command.go` | none of the three files exists. The one site is `entityDraws` in `pkg/game/world.go:480` |
| `WorldFromALM`; `SpawnUnit` gains HP fields; "the scene spawn path" | the loader is `mapload.FromALM`; there is **no `SpawnUnit` and no second spawn path** |
| the four selection gates `selectedCell`, `pickUnitAt`, `unitsInRect` in `box_select.go`, `revalidate` | **none of the five names exists.** The gates are `decide`'s tap and box branches and `presentSelected` (`pkg/ui/command.go:107`, `:167`) |
| `barVisible`, `healthBarFill`, `buildUnitLayer`, `aliveUnresolved`/`downedUnresolved`, `g.sim.sprites` | none exists; there is no health bar and no per-placement state flag anywhere |
| `formationTargets(cx, cy, len(movable), b)` sizes a formation to the movable members | **there is no formation.** 0030 C-3 decided against one by name: every ordered unit takes the same cell, and a placement policy in the front-end was rejected |
| a `ColorScale` tint marks a dead unit | the entity pass draws through `UnitPlace` and the shared blit; no draw path in either front-end takes a colour scale per placement |
| `docs/0016-data-classes/spec.md` line 88 puts `Dying` at a valid class id "in every shipped registry" | 0016 lists `Dying` as an `int32` key and **asserts no meaning for it at all**, which is that spec's own stated rule for every key but six |

**Two decline-by-pointing clauses were chased.** `0034` declines per-class HP by pointing at a stat
table to be decoded later — the table is real and undecoded here (`pkg/data.UnitClass` carries 37
registry keys and no health among them), so it stays a non-goal on a true premise. `0039` declines
sizing a group order by pointing at a formation — **that one is false**, so the clause becomes
nothing at all rather than a non-goal: with one shared destination there is no size to fix.

## What the decoded death arm says, and where this tree cannot follow it

`HERO-DEATH-026` reads death as a **three-stage arm of the actor's own tick**: health at or below
zero starts a per-class `dyingTime` countdown, and **the corpse keeps its cells until that countdown
expires** — occupancy is released at the teardown and the runtime id later still, at a decay stage
far below zero health (`MOVE-ID-016`). So the baselines' "a corpse contributes no occupancy from the
killing blow" is the far end of that arm collapsed onto its first instant, and the middle — a body
that is down, immobile and still in the way — is exactly what the folded three-state model needs.

Two things stop this story reproducing the arm: the countdown's length is a `Data.bin` column no
package here decodes, and a countdown is per-tick canonical state of its own. What the fold can do
is keep the **stages** and drive the transition by a second blow instead of by time. That the
decoded engine also treats zero health as death — the arm's own test is `<= 0` — is the sharper
divergence, and it is the owner's own correction rather than an oversight.

Nothing in the decoded arm needs a flag: every stage the client draws is a **threshold on the health
number itself** (`ANIM-DEATH-007`: stages at -10, -20, -40, and below -600). A signed integer health
is therefore the field a later animation story reads, which is why this one may not clamp it at zero.

## Left open for the plan to settle

Where the amount of a debug chip is decided, and on which side of the seam; whether a non-alive unit
carrying a target is normalised or refused, and by which of the constructor and the decoder; and
what a health bar's geometry is a function of. None is a contract question, and each is decided in
`plan.md`.
