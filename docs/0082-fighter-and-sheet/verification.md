# Verification — the hero is a real fighter, and his numbers are on screen

Five task commits, `SDD-Task: 0082-fighter-and-sheet/T1..T5`, on `impl/0082-fighter-and-sheet`,
**rebased onto master `2a15d03`** — `0079`, `0080` and a research pin bump — with the gate below run
**after** the rebase. Research pinned at `414bc29`, `git submodule status` showing no leading
character; it showed a leading `+` immediately after the rebase, which is the gitlink moving in the
index while the working tree stayed at the old pin, and `git submodule update --init research`
cleared it.

**Two textual conflicts, both resolved by keeping both sides.** `pkg/game/frontend.go`: `0080`'s
attack-pointer clause sits above this story's renamed hero clause. `pkg/game/world.go`: `0079`'s
`faces` parameter and this story's character lookup are both arguments to the same two calls. The
suite is what says they composed, and it is green below.

**Every claim was re-checked at the new pin rather than carried forward.** All fourteen are still
`● active`; `UNIT-PANEL-010` carries a supersession whose consequence runs in this story's favour and
is recorded in `provenance.md`.

## The gate

Run in this seat, branching on the exit code and not on a grep:

```
go build ./... && go vet ./... && gofmt -l $(git ls-files '*.go') && go test -count=1 -trimpath ./...
```

`EXIT=0`, 31 packages `ok`, `gofmt` and `vet` silent.

```
bash scripts/check-no-game-assets.sh ; bash scripts/check-doc-budget.sh ; bash scripts/check-sdd-audit.sh
```

All three `EXIT=0`. `check-sdd-audit` reports 0082 with no note of its own; its note/warning **count**
is not quoted, because this lane's `builds/` holds one story and the orchestrator's holds many.

**No deletion.** `git diff --diff-filter=D --name-only 2a15d03..HEAD` is empty.

**`formatVersion` is 13 and no bump was needed** — with its reason and its falsifier. Every field
this story moves was already on the entity and already in the record: the eight a blow reads, and
`Speed`, which has been in the byte form since it existed. A statistic reaches no simulation field at
all, and the panel is above the determinism wall. What changed is the *value* of existing fields,
which a digest reflects and a layout does not. `pkg/mapload`'s round trip asserts the version byte
**13 by number**, so a bump that was actually needed would have failed it. **15 was allocated to this
story if one were needed; it is unused.**

## What his numbers became, on BOTH ROOTS

`gameversions/en` and `gameversions/ru`, 2026-08-03. G1 needs both; both were run and they agree
value for value.

```
EN: againrom: 38 map rows, 8 of 8 buttons have a mask region; hero Body 43, Reaction 26,
    Mind 15, Spirit 15, Blade 10, Iron Short Sword 10-16, to-hit 49, defence 8
RU: the same clause after `34 map rows`
```

| | before | now |
|---|---|---|
| Body / Reaction / Mind / Spirit | 25 / 25 / 25 / 25 | 43 / 26 / 15 / 15 |
| damage | 7-10 | **10-16** |
| to-hit | 39 | **49** |
| defence | 8 | 8 |
| speed | 10 (a class default) | **17** (derived from Reaction) |

**IT LANDS ON THE MEASURED BAND.** `0078` recorded the owner's four band edges — 7-10 at Body 15,
8-12 at 32, 9-14 at 39, **10-16 at 43**. The rule bought Body 43, and 10-16 is what this tree derives
from the shipped table on both roots. His measurement and this arithmetic meet at a number neither
was fitted to the other on.

**And he hits harder and moves faster for it.** The mission-10 kill `0078` measured at 68 ticks, on
a walk that took 2194:

```
missionrun -mission 10 -waypoint p0:13:49:2 -attack p0:u30 -ticks 30000   (identical on both roots)
waypoint 1  p0 -> (13,49) r2 : reached (11,50), Chebyshev 2, after 1283 ticks
attack 1  p0 -> u30 : FELLED it after 30 ticks, victim at -3 hp
```

The milestone still fires on both roots, and sooner: `TestTheTenthMissionIsDrivenToAWin` passes with
`outcome won at tick 2608` where it was 3504, and no waypoint stops short. **The kill time 57 -> 30
is not claimed as a consequence of anything this story derived**: the cadence is unchanged at 7/4, so
what moved is which draws the hit rolls landed on after arriving 900 ticks earlier. Reported, not
attributed.

**One number moved that this story did not aim at, and it is reported rather than explained away.**
Before speed was derived, the waypoint arrival was 2185 ticks where `0078` recorded 2194, and it was
2185 with the attack flag removed too — so that 9-tick drift is in the walk, not in the blow. The
hero's own to-hit and defence are inputs to combat resolution, which consumes the world's shared draw
stream, so a different hero changes what later draws see. **I did not isolate which draw moved**, and
that is a limitation of this evidence rather than a conclusion.

## What witnesses what

**SC-1 — the rules generation enforces (FR-1, FR-2, FR-4; AC-1..AC-8, AC-12; P-1).**
`pkg/data/chargen_test.go`. `TestTheCostOfAStatisticIsCumulativeAndEscalates` pins T(15)=2, T(25)=10,
T(45)=164 and both ends of the escalation, 22 into 45 and 1 in the low twenties (AC-1, AC-2).
`TestARefundIsTheStepThatReachedIt` walks every `v` in 16..45 (AC-3).
`TestHowHighAStatisticCanActuallyBeBought` **finds** 42, 43 and 34 by search rather than asserting
them, and shows the click ceiling of 45 is unreachable — which is what makes the two bounds two
bounds (AC-4, AC-5, AC-6). `TestThePoolAndTheBudgetAreOneNumber` (AC-7).
`TestASpreadOutsideEitherBoundIsRefused` covers under-floor, over-ceiling and the case that matters —
**every statistic inside the range and the spread still illegal**, so the budget is doing its own work
(AC-8; P-1's completeness, together with the exhaustive searches above).
`TestTheGenerationStartSpendsFortyAndLeavesTheWholePool` and
`pkg/data/hero_test.go`'s `TestTheChargenStartWithItsStartingWeapon` — the start still derives
`7-10 / 39 / 8`, unedited (AC-12). `TestNewHeroWritesTheSpreadAndOneSlot` asserts the start **is**
`NewHero` over the initialiser, so the two cannot drift.

**SC-2 — the spread (FR-3, FR-5; AC-9, AC-10, AC-11).** `pkg/game/hero_test.go`.
`TestThePartySpreadIsTheRulesUniqueAnswer` searches **all 31⁴ = 923 521 spreads**: the spread is
legal, both floors hold, no legal spread has a higher Body, none with that Body and both floors has a
higher Reaction, and exactly one spread satisfies the three together. It also asserts the leftover
point is **unspendable** — neither Body 44 nor Reaction 27 is legal — and, as its last case, that
Body-maximal plus Reaction-maximal **without** the floors admits more than one spread, which is why
the rule's first step is a step (AC-9). `TestAnInstalledTableBecomesTheHerosBand` runs a table's bytes
to `10-16` with nothing stubbed (AC-10, AC-11).

**SC-3 — the panel (FR-6, FR-7; AC-13..AC-16, AC-14a; P-4, P-5).** `pkg/ui/sheet_test.go`.
`TestAPanelStatesACharacterInTheDecodedOrder` asserts all fifteen rows as an **ordered** list, so a
panel listing the four statistics in storage order fails (AC-13).
`TestAUnitWithNoCharacterStatesOnlyItsNumbers` (AC-14), including that an untrained bare hero drops
the skill and weapon rows **one at a time** while keeping his statistics.
`TestASubjectToldNeitherGroupComposesThePanelItWas` is AC-14a as an identity: composed against a
layout holding only the four pre-story rows, **pixel for pixel**.
`TestTheDamageRowIsTheSheetsOwnComposition` kills `10-6` (AC-15).
`TestTheAlwaysHitsRowAppearsOnlyWhenTheMarkIsSet`, and that setting it adds a row rather than
replacing one (AC-16). `TestTheSharedFieldSpaceStaysDisjoint` walks all thirteen new numbers and all
twelve readout numbers, requires each resolver to refuse the other's, and refuses any of ours
inside `16..31`.
`TestAChangedNumberRebuildsThePictureAndAnUnchangedOneDoesNot` is P-5 through `panelBuilds`: a
statistic, a combat number and a weapon each rebuild; an identical frame does not. P-4 is unchanged
and is why the two composes above are comparable at all — `composePanel` still takes a layout, a font
and a subject, and this story added no viewer setter.

**SC-4 — the values arrive (AC-17).** `pkg/game/sheet_test.go`, an internal test because the driver's
lookup and its push are what is being asserted. `TestTheEntityPushCarriesEveryUnitsOwnEightNumbers`
uses two entities whose numbers are **all distinct**, so a push carrying one under the other's id
could not pass. `TestOnlyTheEntityTheLoadKnowsACharacterForStatesOne` covers the hero, a unit with no
entry, and a driver with no lookup at all. `TestPartyCharactersPairsTheStartsOwnSlices` covers an
axe-trained member (the slot is read **off the hero**, not assumed), a bare member naming no weapon,
an untrained member naming no skill, and three empty shapes.
`pkg/mapload/hero_test.go`'s `TestAStartReportsTheEntityIdOfEachMember` checks the recorded ids
against the ids the world's entities **actually carry**, by id, cell and class together.
And on the install: the panel above, composed with the game's own font.

**SC-7 — the speed a Reaction buys (FR-10; AC-21, AC-22, AC-23).**
`pkg/data/chargen_test.go`'s `TestSpeedIsDerivedFromReactionAtABranchOfTwelve` samples 0, 11, 12, 26,
45 and 50, then **walks every Reaction to the cap** — a table of samples near 12 would pass with the
branch one either side, and the walk will not (AC-21). It also pins that the cap binds first and that
the other three statistics move speed nowhere, which is the same census the eight are held to.
`pkg/mapload/hero_test.go`: a started member carries his hero's speed and explicitly **not** the unit
table's 10 (AC-22), and the zero-value member reaches 0 by the same arithmetic with no fallback.
`pkg/ui/sheet_test.go` states the row and drops it with the rest when nothing was told (AC-23).

**HIS SPEED IS 17, NOT THE OWNER'S 19, AND THAT IS A FINDING RATHER THAN A MISS.** The law at
Reaction 26 gives `26/5 + 12 = 17`. **19 requires a Reaction of 35..39** — which is legal, well
inside the budget, and simply a different character from the damage-maximal fighter the rule chose.
The highest speed *any* legal generated spread can reach is **20**, at Reaction 43. So his figure
does not contradict the law at all: it discriminates between builds, and it says his own character
was carrying a Reaction in the middle thirties where ours carries 26. Nothing was tuned toward 19.

**And it is visible in the drive, which is the strongest single number in this document.** The
mission-10 waypoint took **2185** ticks at the old class-default speed of 10 and takes **1283** at
17. `2185 × 10/17 = 1285.3` against a measured **1283** — 0.2 % — so the derived speed reached the
mover, and the 9-tick drift reported below is exactly the size of the residue that leaves. The
milestone now fires at tick **2608** where it fired at 3504, identical on both roots.

**SC-5 — the check line (FR-8; AC-18).** `pkg/game/frontend_test.go` pins the wording as a literal
**and** asserts the four numbers are `MissionParty`'s own member's, derived. That second assertion is
the one that matters: a check line built from a second construction of the hero would have printed
the generation start's `7-10 / 39` while every mission placed `10-16 / 49`, and both are correct
answers to different questions. `cmd/againrom/main_test.go` asserts the whole line over a synthetic
install, and the bare arm now requires the character **and** the reason.

**SC-6 — nothing simulated moved (FR-9; AC-19, AC-20; P-2, P-3).**
`pkg/mapload/hero_test.go`'s `TestAWorldHoldingAnArmedPartyRoundTrips` — equal bytes, equal hash,
version **13 by number** (AC-19). `pkg/game/sheet_test.go`'s `TestPushingTheCharacterMovesNoWorldState`
— a world pushed with a character hashes what one without hashes, and two pushes move neither the
digest nor the tick (P-2 at this seam).
**P-2 —** `git diff --stat 63769f0..HEAD -- pkg/sim` is empty: no rule, no field, no version.
`internal/archtest`'s source scan over `pkg/sim` is green above, and the one float this story adds
lives in `pkg/data`, below the wall.
**P-3 —** `internal/archtest`'s allow-map test is green. No package gained an intra-module import:
`pkg/game` already imported `pkg/data`, `pkg/ui`, `pkg/mapload` and `pkg/sim`, and `pkg/ui` gained
nothing at all — the window tier still names no simulation, format or data type, and both new types
are its own, of builtins.

**AC-20, and it is the criterion this story could most easily have faked.** The whole suite is green
with no game present. **Exactly four expectations were edited, and each states the party hero's own
numbers or the line that reports them**: `pkg/game/frontend_test.go`'s two check lines,
`pkg/game/hero_test.go`'s party assertion and its band, and `cmd/againrom/main_test.go`'s two.
`pkg/data/hero_test.go` was touched for **a comment only** — no expectation in it moved, and the
generation start still asserts the identical eight numbers. **No placement, spawn, digest, byte-form,
panel or readout expectation was edited at all**; in particular not one of `pkg/ui`'s existing tests
was touched, which is AC-14a seen from the other side.

## The panel, measured against the lawful install

Composed through the exported `RenderPanel` seam with each install's own font, by a throwaway program
that was deleted afterwards:

```
en: font height 15, speed 17, panel box (255, 283)
ru: font height 15, speed 17, panel box (255, 283)
```

Byte-identical on the two roots. **R-2 discharged against the real font**, not only the synthetic one:
255 × 283 in a 1024 × 768 window at a 12-pixel margin. The rendered sheet ships in
`builds/0082-fighter-and-sheet/` as `sheet-en.png` / `sheet-ru.png`.

The shipped sword's cadence is **7/4**, not the 9/5 the synthetic fixtures use — the panel states the
install's own numbers, and the difference is a fixture's, not a defect.

## What was NOT verified, and why

- **That the owner's character should be this one.** It is authored and it is his call. The rule that
  produced it and the seam that holds it are `plan.md` DD-2 and DD-3, and the seam is one function
  body.
- **The floors' cost.** Mind reaches sight and Spirit reaches mana and the five protections in the
  original. None is derived in this tree, so flooring both costs nothing **today**; a story that
  derives one inherits a weak hero and the open item that says so.
- **A screenshot of the running window.** The panel was composed through the same function the viewer
  composes it with, against the real font, and dumped; nothing drove the ebiten window.
- **That a shipped campaign takes the ordinary generation arm.** Research grades it Medium. `0078`
  carried the grade and this story carries it unchanged and unexamined.
- **The original's own panel layout.** Closed against the instrument rather than against the game:
  past `+0x14a` the block is addressed by computed index, so no sweep can name a consumer. The row
  order of the four statistics and the damage row's composition are reproduced from elsewhere; the
  rest is ours.
- **The 9-tick walk drift above.** Reported, bounded to the walk, not isolated to a draw.
