# 0125 — experience from use: plan

## Shape

Five edits, bottom of the stack upward: the curve's inverse and the two accessors in
`pkg/data`; the state and its byte form in `pkg/sim`; the payment in `pkg/sim`; the fill in
`pkg/mapload`; the readout in `pkg/game` + `pkg/ui`.

## Design decisions

**DD-1 — the simulation carries experience, never a level.** FR-11. `S` needs `pow`, and
`pkg/sim` bans floats. Making the simulation hold experience integers and nothing else removes
the conversion from behind the wall entirely, so no curve table has to cross it and no
generated table has to be kept in step with its generator. Levels are wanted in exactly two
places, the derive and the panel, and both are already outside. The alternative — a 101-entry
table injected as rules data — was rejected because it adds a second copy of the curve that
can silently disagree with `pkg/data`'s.

**DD-2 — the fighter/mage branch is resolved at load into one slot number.** FR-2, FR-14. The
entity gets `XPSlot uint8` and no class bit. This tree already folds the weapon into the
combat numbers when a placement resolves, so reading the weapon again at the moment of the
blow would be a second seam that could disagree with the first. It also keeps the authored
half of the story to **one expression in one function**, which is what a later research item
has to move. Cost, stated: swapping weapons mid-mission would not move the credited slot —
this build has no such swap in the simulation.

**DD-3 — five new entity fields, no more.** FR-1, FR-3.
`SkillXP [6]int32`, `Mind int32`, `XPValue int32`, `XPSlot uint8`, `GainsXP bool` — 34 bytes, `entityLen` 149 → 183, form version **35**
(allocated to this lane; the live version this branch merges is **32**, and 33/34 are out to
other lanes, so the peel chain's innermost predecessor is 32). `GainsXP` is a
field of its own rather than a sentinel value of `XPSlot`, because "a monster" and "which
slot" are two facts and folding them would make the decoder unable to refuse an undefined slot
without also refusing a monster.

Re-pinning the form is by eye and not by whatever the suite reports. **Read every chained
peel expression**: the chains run through `pkg/sim/binary_test.go`,
`pkg/mapload/fromalm_test.go`, `gridform_test.go` and `routeform_test.go`, and each unwinds
to the innermost predecessor, which is **32**. `pkg/sim/corpseloot_test.go` carries a
deliberate tripwire on the version literal, pinned at 32 and correct twice already:
**re-pin it naming this story, never delete it.**

**DD-4 — the decoder refuses what the constructor cannot build.** A slot outside `0..5`, a
negative slot experience, and a `GainsXP` byte outside `{0,1}` are refused, in the one fault
function both sides call — the shape `attackFault` and `transitFault` already have. A negative
Mind or experience value is **not** refused: both are table columns carried whole, exactly as
the seven blow numbers already are.

**DD-5 — the payment is a method on `*World`, called from `resolveBlow`'s tail.** FR-4, FR-8.
It takes the two indices and the amount removed, and it is the only writer of `SkillXP`. The
"target was alive before the blow" test is read **before** the subtraction and passed in, not
recomputed after — after the subtraction the answer is gone.

**DD-6 — the two divisions are integer, and each is proved rather than asserted.** FR-6, FR-7, P-2,
P-3. `raw = (xpValue*removed)/(2*maxHP) + 1`: the multiply happens before the halving so an odd
experience value does not lose its half early, and `floor(x + 1) = floor(x) + 1` for any real
`x`, so the `+1` outside the truncation is exact. `gain = raw*(4*mind+30)/120`: this is **not**
bit-identical to `raw*(mind/30 + 0.25)`, because `mind/30` is inexact in binary. Both are
measured against a float reference in a **test file** — `pkg/sim`'s determinism scan reads
non-test sources only, so a test there may use `math` and `float64`, and `pkg/data`'s existing
pow-margin test is the precedent for measuring rather than asserting.

**DD-7 — the inverse is a bounded scan, not a solve.** FR-10, P-1. `SkillLevelFor(xp)` walks
`n` from 100 down to 0 and returns the first `n` with `S(n) <= xp`; `S(0) = 0` so it always
returns. It calls the same `pow11` and the same truncation the forward direction already uses,
so the two cannot come to disagree about a boundary. No tolerance, no float comparison against
a float — the comparison is between two truncated `int32`.

**DD-8 — the party member is seeded from his own levels.** FR-13. `Hero.Reward()` is a thin
accessor over `Recompute` in the shape `Speed()` and `Sight()` already have, returning the
capped Mind and the six per-slot experiences the derive already computes. The mint site calls
it beside `Derive`; nothing re-derives a stat outside `Recompute`.

**DD-9 — the weapon's slot rides on `data.Combat`.** `Combat.SkillSlot int32`, filled by
`Derive` from its own weapon argument. This is exactly what `Combat.Reach` already does and
for the same reason: the value is a function of the weapon, every construction path already
carries a `Combat`, and a second channel would need a second fallback.

**DD-10 — the panel row changes value, not identity.** FR-15. `PanelFieldSkill` stays the
field and stays in `AuthoredPanelLayout` at its row, paired with XP. What changes is
`UnitCharacter`: `Skill string` + `SkillLevel int` become `Skills [PanelSkillSlots]int`, and
the resolver states the six through a helper beside `panelFamilyText`, unconditionally when
`Known`. Keeping the field id and the row literal keeps the diff inside `pkg/ui` small, which
matters — two other lanes are live in that package. `PanelSkillSlots` is a local constant, so
no import edge is added.

**DD-11 — the panel's character block is overlaid from the entity.** FR-16, FR-17.
The block stays the load-time map it is; where the readout is built, the two experience-bearing members are
overwritten from the entity that tick, the way the combat block beside them is already read.
The map keeps the four statistics, the weapon name and the two families, which do not move.

## Work, in order

1. **`pkg/data`** — `SkillLevelFor(xp int32) int32` (DD-7) beside the forward direction;
   `Combat.SkillSlot` filled by `Derive` (DD-9); `Reward` and `Hero.Reward()` (DD-8). Tests:
   P-1 over the whole level range, and the seed's agreement with `Derived.SkillXP`.
2. **`pkg/sim` state** — the five fields (DD-3), the encode/decode pair, `entityLen`, the
   version constant and its doc block, the fault function (DD-4), and the hash. Tests: AC-2,
   the refusals, and the existing pinned-bytes test re-pinned.
3. **`pkg/sim` payment** — `payExperience` (DD-5), the two integer forms (DD-6), the five
   refusals of FR-5, called from `resolveBlow`. Tests: AC-1, AC-3, AC-4, AC-5, AC-6, P-2, P-3,
   and FR-9's draw count.
4. **`pkg/mapload`** — `spawnBlock` gains the three definition-borne numbers and the flag; the
   three arms of `blockFor` fill them (FR-12); the party mint fills its own from `Reward` and
   `Combat.SkillSlot` (FR-13, FR-14).
5. **`pkg/game` + `pkg/ui`** — `UnitCharacter.Skills` (DD-10), the resolver, the overlay
   (DD-11). Tests: AC-7, AC-8.

**SC-1 and SC-2 belong to the build stage, not to any of the five.** Neither can be asserted
from a synthetic fixture: they are what the owner does with a mouse against his own install, so
they are run and recorded there and nowhere else.

## Risks

- **Byte-form collision.** Another lane is widening the same record. This lane appends its
  block at the tail, takes version 35 and takes no other number; `origin/master` is merged
  before the final gate, not after.
- **`pkg/ui` collision.** Two lanes are live there. DD-10 keeps the change to one struct, one
  resolver arm and one helper.
- **A gain that never fires.** FR-5's fourth and fifth refusals can each suppress everything if
  their premise is wrong about how this tree seats owners. The tests in step 3 assert a gain
  **fires** under the ordinary configuration, not only that it is refused under the odd ones,
  and SC-1 is checked in the build.
