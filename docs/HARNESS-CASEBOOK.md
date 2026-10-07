# Harness casebook

This file preserves the incidents behind the compact rules in `AGENTS.md`. It is **not** startup
context. Read the relevant case when a rule appears inapplicable, a review finds the same shape, or a
new rule is being proposed.

The rules remain normative in `AGENTS.md`; this file explains their failure boundaries.

## Closure is as-built, not a review diary

Story 1005 reached 1,730 lines of closure, mostly one section per adversarial pass. The reader who
needs to decide what shipped needs the matrix, integration witness, reconciliation, and open items.
The pass sequence is useful history, but git and `pipeline/LOG.md` already preserve it. A closure so
large that nobody finishes it stops functioning as evidence.

## Sweep producers, not data rows

Story 1004 counted 48 item rows carrying a weapon spell and called the feature covered. Delivery,
however, selected an arm from the **carrier's** mana pool. Both Fire Ball rows in the sweep were the
same carrier that the fix could not reach: the player saw the explosion, but no projectile flew.

The row count was correct and the population was wrong. A change to what is drawn or delivered must
enumerate production paths, not records that happen to contain the relevant data.

## A persisted fact invalidates every old derivation

Story 1005 added `PartyMember.WeaponMaterialized` because current equipment and pack contents cease to
answer historical questions once an item is sold or left behind. The first sweep covered the mission
and shop dolls. Later passes found three non-screen readers that still guessed the old fact:

- `buildInventorySubject` seeded opening state from present equipment;
- `mapload.PartyLoadout` re-derived it at the next mission's construction;
- `recomputeRaisedSkills` re-derived it every tick for a non-current party member.

Each site had a locally plausible comment whose premise the new persisted field invalidated. The
correct population is every reader of the state the new record replaces, not only the screens named
in the story.

## Expectations must be independent

The Heal hotfix asserted particle growth as `(age+1) * healSpawnsPerTick` and the cap as
`healParticleLife * healSpawnsPerTick`. Changing `healParticleLife` to 1 destroyed the requested
behaviour while the test stayed green: expectation and implementation moved together.

Write the externally required number or relation independently. Mutation must change the production
line a maintainer would actually edit, not a hand-built stand-in that can fail while the real
constant remains unguarded.

## Mutation kill does not validate the fixture

The 1005 doll and shop drag guards compared release position with drag origin. Live drag state masked
the origin slot, but the tests never installed that suppression. Reverting the guard reddened the
assertions, so the assertions were load-bearing; the same tests nevertheless passed against both the
broken and corrected live-state interaction.

For a defect-reproduction claim, name the production state the fixture installs and show that the
mutation becomes observable only once that state is in force.

## Observe the value production emitted

A headless `figure` witness computed `equipmentSlots(mw.currentFigureEquipment())`, while the doll
actually composed from `member.Worn`. Every scenario compared the live world array with itself and
could not see the disagreement. A first correction called the composer a second time, which still
recomputed an expected equivalent instead of capturing the actual composition.

A witness must carry out the value emitted by the production path under test. A second derivation,
even a pure one with identical arguments today, cannot disagree when routing changes tomorrow.
## A control cannot report that its region was painted over

The town's DOLL/STATS toggle blanked the whole screen outside the character column for two
stories. `ComposeShopScreen` called `drawTownStatisticsSurface` in Statistics mode, filling
`TownContentRegion`; story `1022` removed the same call from the tavern and the school and left
the shop's arm carrying it. The package's own tests agreed with the defect throughout, because
each one asked a control what it answered rather than what the frame showed, and the hit-test
gates had been written to match the blanking. Restoring the blanking as a mutation moves 189,607
pixels and reddens no hit-test assertion in `pkg/ui`.

Two properties made it invisible for that long. The mode had no entry in any screen enumeration:
`cmd/screenshot` knew nine screens and each one was a composer, not a state, so the only way to
see this mode was to play. And the story that introduced the card had no frame-level witness at
all for the shop, only per-control ones.

For a change to what a mode draws, compare the mode's composed frame with the other mode's and
require the difference to fall inside the rectangle the mode owns. The expected frame costs
nothing to build: it is the same view with one field flipped. Assert the owned rectangle did
change as well, or a compositor that ignores the mode passes too. Register the mode wherever
screens are enumerated in the same change.


## Why review chains are bounded

Story 1005 took thirteen passes. Four discovered one latch class site by site, and two restarted the
lane for witness/document findings even though production was correct. That history produced the
current P/W/D classes, class-wide enumeration rule, and pass ceilings.

The ceiling is not permission to ignore a production defect. It prevents review from substituting an
unbounded sequence of fresh anecdotes for one population argument and diagnoses when the contract was
too broad to close coherently.
