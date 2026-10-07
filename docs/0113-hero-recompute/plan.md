# Plan — a hero's stats are recomputed, whole and in order

## Approach

One new file, `pkg/data/recompute.go`, holds the whole graph as a single function taking a `Hero`, a
`Profile` and a `Loadout` and returning a `Derived`. Everything already in `pkg/data/hero.go` that
states a piece of that graph — `Derive`, `Speed`, `Sight` — is reduced to a call into it. The helpers
those three already share (`capStat`, `pow11`, `ftol`, `activeSkill`, `cellOr`) stay where they are
and are reused unchanged, so the diff to `hero.go` is the removal of three function bodies rather
than a rewrite of the package.

Above that, two consumers: `pkg/ui` grows three panel fields and the values they state, and `pkg/sim`
grows one mutator so a second recompute can land on a unit that is already in a world. Nothing else
moves.

## Facts verified during planning

**The call sites are three.** `pkg/data/humandef.go:225`, `pkg/game/frontend.go:435`,
`pkg/mapload/start.go:272`, plus test files. All three take the `Combat` return and nothing else, so
`Derive`'s signature can stand as a wrapper and no caller has to change.

**`pkg/data` is outside the determinism wall.** `internal/archtest`'s float ban is evaluated over
files named `pkg/sim/...` only; `pkg/data/hero.go` already imports `math` and calls `math.Pow`, and
builds. So the pools' logarithm may live here.

**Nothing in `pkg/sim` reads a protection or a resistance.** `resolveBlow` subtracts absorption flat
and stops, and its own doc block lists the elemental protections among what is not there. So the new
values are loader values and **no byte-form version is spent** — 27 stays unallocated.

**`data.UnitDef` already models `Protection [5]int32` and `Resistance [5]int32`**, deliberately in
column order rather than in the resolver's order and deliberately unnamed. The derived set uses the
same width and the same convention, so a later story that finds the permutation applies it in one
place.

**`data.Equipment` carries item codes, not item numbers.** It cannot be the recompute's equipment
argument on its own: resolving a code to the numbers it contributes is a tier above `pkg/data`. The
recompute therefore takes the *resolved* contribution.

**A party member's health is `SpawnHP` and always has been**, on `0078`'s recorded divergence.
Nothing about it changes here.

## Design decisions

**DD-1 One function, one value.** `func (h Hero) Recompute(p Profile, l Loadout) Derived`. `Derived`
is a value type of arrays and integers — comparable, copied by assignment — so a test can pin a whole
recompute with one `==` and a caller cannot half-apply one.

**DD-2 The profile is what the character does not carry.** `Profile{Fighter bool; HealthColumn,
ManaColumn bool}`. Its zero value is flag clear and both columns absent, which is exactly what a
generated character is known to be able to say about himself: nothing. Naming the field `Fighter`
follows the claim wording for the arm that takes the doubled health term; **which archetype actually
sets that flag is not decided here**, and the field's doc block says so rather than implying it.

**DD-3 The loadout is a weapon and a block, not one of them.** The weapon is not merely additive: it
**assigns** the cadence per half, the reach, and the active-skill slot. The other eleven slots are
purely additive. Folding the weapon into the modifier block would lose the assignments; passing only
a weapon is what the current signature does and is the defect. So `Loadout{Weapon *Weapon; Mod
EquipMod}`, and `EquipMod` is the fourteen-field additive block: `ToHit`, `DamageBase`,
`DamageSpread`, `Defence`, `Absorption`, `Protection [5]int32`, `Resistance [5]int32`.

**DD-4 The armour seam is a zero, not an absence.** `EquipMod` has the fields an armour or a shield
would fill; nothing in this build fills them, because the column that supplies them has no read
consumer anywhere. That is a seam a later story closes by writing one resolver, not by widening a
type. **No column is guessed and no value is fitted to a remembered sheet.**

**DD-5 `Derive`, `Speed` and `Sight` become accessors.** Each is one line: build the recompute, return
the member. `Derive(w)` passes `Profile{}` and `Loadout{Weapon: w}`, which changes nothing it
returns, because the profile reaches only the two pools. Keeping the three names is what keeps the
diff outside `pkg/data` to zero; deleting their **bodies** is what makes FR-1 true.

**DD-6 The two pools are produced and not wired.** `Derived` carries `HealthMax` and `ManaMax`
because the routine computes them and because the second load-bearing ordering point is about them.
No caller writes them onto an entity: a generated character's health column, mana column and class
flag are three inputs nobody in this tree can state, and inventing them to move a party member off
`SpawnHP` would be exactly the fit this project refuses. The seam is named in the doc block with the
missing fact spelled out.

**DD-7 The final clamp is a method, not two fields.** `Derived.ClampPools(health, mana int32)
(int32, int32)`. The derived set is what a recompute produces; a live health is state the caller
holds. Putting the clamp on the value keeps step 8 inside the one implementation without `Derived`
pretending to own a pool.

**DD-8 The five-wide arrays, and the sixth slot.** The cleared block holds six protections and six
damage-kind bytes; five of each are what a column fills and what the derivation writes. The arrays
here are five wide, matching `UnitDef`, and the doc block records that the block's index 0 in each
family is filled by no column and derived by nothing — so a reader knows the width is a fact and not
a truncation.

**DD-9 The re-derivation seam is one mutator.** `pkg/sim` gains `CombatBlock` — the eight numbers and
the reach, as builtins — and `func (w *World) SetCombat(id EntityID, c CombatBlock) bool`. It writes
those nine fields on the named entity and nothing else, answers false for an id no entity has, and is
documented as canonical-state mutation that must be applied at a deterministic point. **Where it is
called from is `0115`'s**; this story provides the door, not the caller.

**DD-10 Three panel rows, not thirteen.** `ui.UnitCharacter` gains `Experience int` and
`Protection [5]int`, `Resistance [5]int`. The layout gains `PanelFieldExperience`,
`PanelFieldProtection` and `PanelFieldResistance` — the next three free values in the shared space,
34, 35 and 36 — each stating its family on one line. Five separate protection rows would state one
number five times in a window this story is explicitly not allowed to redesign, and the claim's own
prediction — all five equal on an unequipped character — is more legible on one line than on five.

**DD-11 Speed and sight sit after the pools.** They read only capped statistics, so their position
among the terms is not load-bearing; they are placed where the original reads their inputs, and the
doc block says which of the four ordering points constrain them (only the first).

**DD-13 The restore is modelled with the bonus block it reads, and only one of the two clamps.**
`EquipMod` gains `SkillBonus [SkillSlots]int32` and `Derived` gains `Skill [SkillSlots]int32`: the
incoming levels are the base copy, the bonus is what a loadout adds, and the restored array is what
the active-skill terms read. The original clamps the same array twice inside one recompute, once at
the restore and once in the tail past the equipment fold; nothing between them can move it — the fold
reaches the two bytes below the array and the eight above it, never the array — so this tree clamps
once and says in the code that the second exists. `Derived.SkillXP` carries the per-slot experience
because that is the primary storage in the original and the total is the derived value; the inverse
direction is named in the doc block as a seam rather than derived mathematically, because what is
unread is the routine's own arithmetic and a plausible algebraic inverse is not a decoded one.

**DD-12 The logarithm's divergence is disclosed.** `log(1.1, x)` is computed as
`math.Log(x) / math.Log(1.1)`, truncated toward zero like every other intermediate. The original
calls the C runtime's log, and two implementations may differ by an ULP, which a truncation can
amplify to a whole unit. `pow11`'s doc block carries a measured margin for its own chain; **no such
margin is measured for the pools here**, and it does not have to be while nothing consumes them — but
it is the first thing owed by whoever wires DD-6's seam.

## Files to touch

| File | What |
|---|---|
| `pkg/data/recompute.go` | new: `Profile`, `EquipMod`, `Loadout`, `Derived`, `Recompute`, `ClampPools`, the experience sum and the skill restore |
| `pkg/data/hero.go` | `Derive`, `Speed`, `Sight` reduced to accessors; helpers unchanged |
| `pkg/data/recompute_test.go` | new: the order, the terms, the clamps, the equivalence with the old `Derive` |
| `pkg/sim/rearm.go`, `pkg/sim/rearm_test.go` | new: `CombatBlock`, `World.SetCombat` |
| `pkg/ui/panel.go`, `pkg/ui/panel_test.go` | three fields, three rows, three texts |
| `pkg/game/world.go`, `pkg/game/hero_test.go` | `partyCharacters` fills the three from one recompute |

## Risks

**A silent change to a shipped number.** The recompute must reproduce `Derive` exactly. Mitigated by
pinning the equivalence directly: the existing `hero_test.go` cases are left in place and keep
calling `Derive`, so if the wrapper's answer moves, they fail.

**Two spellings surviving.** The failure this story exists to end. Mitigated by deleting the bodies
rather than adding a second function beside them, and by an explicit success criterion that greps for
the arithmetic.

**Rebase against `0112`.** That lane edits the party-member literal in `pkg/mapload/start.go`. This
story does not touch `start.go` at all, which removes the conflict rather than managing it.

**Panel width.** Three new rows state five numbers each; the panel flows and has a minimum width, not
a maximum, so the frame grows. That is the layout doing its job, and compacting it is `0116`.

## Success criteria

**SC-1** `go build ./... && go vet ./...`, `gofmt -l` empty over tracked Go files, and
`go test -trimpath -count=1 ./...` all green.

**SC-2** `scripts/check-no-game-assets.sh`, `scripts/check-doc-budget.sh` and
`scripts/check-sdd-audit.sh` pass.

**SC-3** `grep -n 'pow11\|math\.Log' pkg/data/*.go` shows the graph's arithmetic in
`recompute.go` and in the shared helpers only — `hero.go` retains no derivation body.

**SC-4** The byte-form version constant in `pkg/sim` is unchanged against master, and
`git diff master --stat -- pkg/sim/binary.go` shows no field added to the form.

**SC-5** `git diff --diff-filter=D --name-only master..HEAD` is empty.

## Traceability

FR-1 -> DD-1, DD-5 -> SC-3. FR-2 -> DD-1, DD-2, DD-3 -> SC-1. FR-3 -> DD-1 -> SC-1.
FR-4, FR-5 -> DD-1, DD-11 -> SC-1. FR-6, FR-6a -> DD-1, DD-13 -> SC-1.
FR-7, FR-8 -> DD-2, DD-6, DD-12 -> SC-1.
FR-9, FR-10 -> DD-5 -> SC-1. FR-11 -> DD-8 -> SC-1. FR-12 -> DD-5, DD-11 -> SC-3.
FR-13, FR-14 -> DD-3, DD-4 -> SC-1. FR-15 -> DD-1 -> SC-1. FR-16 -> DD-7 -> SC-1.
FR-17 -> DD-10 -> SC-1. FR-18 -> DD-9 -> SC-4.
