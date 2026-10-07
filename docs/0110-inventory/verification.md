# Verification — 0110

Four task commits, then this. Evidence run against **both** lawful installs wherever a criterion
says both; the tests themselves read no install and the repository holds no asset.

## Success criteria

**SC-1, SC-2 — the project gate.** Run one command at a time, so an exit code belongs to the command
that produced it rather than to a pipe.

```
build=0  vet=0  gofmt=(nothing)  test=0
check-no-game-assets=0  check-doc-budget=0
check-sdd-audit: FAIL set empty for every story
```

The audit's note and warning **count** is not comparable from a worktree — `builds/` is untracked, so
a lane emits none of those warnings until its own build stage exists. Only the FAIL set is compared,
and it is empty here and on `master`.

**SC-3 / AC-3 — every composed address is a shipped node, on both installs.** The one weapon literal
this front end's character generation hands out is `Iron Short Sword`, resolving to row 3 with shape
index 0 and material index 0, so its code is `0x0103` and its name `0001003`. Measured on each root:

```
                                       en    ru
graphics/inventory/0001003.16a          ok    ok      80x80,   1 frame
equipment/mfighter/1.256                ok    ok      160x240, 1 frame
equipment/mfighter/primary/0001003.256  ok    ok      160x240, 1 frame
```

The other four weapon literals the generation table can yield — `0101118`, `0101109`, `0101015`,
`0801120` — were measured the same way: **icon present on both roots, figure layer present under
`mfighter` and `ffighter` on both roots, absent under `mmage` and `fmage` on both roots.** Absent
under the two mage directories is the expected shape and not a miss: a mage never holds these.

Census, identical on both roots: `inventory` **416** nodes, `equipment` **987** — of which `primary`
784, `secondary` 144, and 59 face sheets directly inside the four figure directories.

**This is corpus agreement about field C, and nothing more (DD-1).** Field C is published **Unknown**
by `ITEM-APPEAR-023`; that every shape index this build composes names a sheet the archive holds does
not decode those three bits, and research itself caps corpus agreement at Medium. The criterion is
worth having because it can **fail** — a wrong C names a sheet that does not exist — not because
passing it promotes the choice to a fact.

**SC-4 / AC-10 — the milestone did not move.** `cmd/missionrun` built from this branch, driven with
`pipeline/check-milestone.sh`'s own argv, against each root:

```
en	outcome lost at tick 272
en	census: 4 of 36 unit(s) moved, 1 fell, over 272 tick(s)
ru	outcome lost at tick 272
ru	census: 4 of 36 unit(s) moved, 1 fell, over 272 tick(s)
```

Byte-identical to `pipeline/milestone-baseline.txt`.

**SC-5 / AC-9, FR-10 — the simulation did not move.** No file under `pkg/sim/` is touched by any of
the four commits (`git diff --name-only master..HEAD | grep ^pkg/sim/` is empty). The byte form's
version literal is **24**, unchanged; `pkg/sim`'s own pinned-digest and version criteria pass
unmodified. **Version 27, allocated to this story, is unspent** — this story adds no encoded field,
and everything it introduces is a loader value (DD-7).

**SC-6 — the runbook.** Rendered from a shipped install through the code that ships: `game.NewFrontEnd`
resolves the starting weapon, `pkg/data` composes the three addresses, and `ui.RenderInventory`
composes the window. The result shows the figure **holding the sword**, that sword's icon in the first
of twelve slot cells, and the pack area drawn and empty. The base sheet contributes 7 259 opaque
pixels and the weapon layer writes **389** of its own over them; the same base with no layer differs
from the composed figure in 1 503 bytes. Both roots produce identical numbers.

## Where each criterion is witnessed

The three that need an install are above; the rest are assertions in the suite, and the table names
the one that would go red if the clause stopped holding.

| | Witness |
|---|---|
| **AC-1** | `TestFieldsAtEveryBoundaryValue` and the 65 536-code walk beside it, recomposing each code from masks computed independently of the readers |
| **AC-2** | `TestNameSpellsBothFormsAtTheirBoundaryValues`, and a second walk of the whole domain asserting exactly seven digits every time |
| **AC-4** | `TestFillingOneSlotAnswersTheWholeCodeThatWasWritten` — a code with all four fields nonzero, read back from every slot |
| **AC-5** | `TestAnOccupiedFirstSlotDerivesTheEntryOneBeforeItsRow`, widened with codes carrying nonzero A, B and C; and `pkg/game`'s party-assembly and resolution criteria, whose assertions did not move |
| **AC-6** | `TestInventoryToggle` over every clause — closed at the start, opens only with its own subject alone selected, the second press closes it, cancel closes an open one and leaves a closed one closed, clearing the selection closes it — and `TestToggleInventoryOpensOnlyForItsOwnSubjectsSingleSelection` at the method |
| **AC-7** | `TestRenderInventory`: two layers on the figure, twelve cells with the first holding the icon, the pack at its fixed count and empty |
| **AC-8** | `TestRenderInventory`'s nil-cell subtest, `TestATransparentPictureLeavesTheGroundBeneathItOpaque`, and `TestBuildInventorySubjectReportsEachUnreadAddressAndLeavesTheRestWhole` for the report |
| **AC-12** | `internal/archtest`'s pinned allow-map entry for `pkg/ui`, fail-closed: the tier may reach `pkg/render` and nothing else, so no archive type can appear |
| **P-2** | the two whole-domain walks (AC-1, AC-2) and the totality cases in `pkg/game/inventory_test.go` — no party, no ids, no weapon, no source |
| **P-3** | AC-12's witness; `pkg/ui/inventory.go` imports `image` and `image/color` only |
| **P-4** | `TestInventoryTogglingIsInert` for the selection and the seam, and `TestOpenMissionBuildsTheInventorySubjectWithoutMovingTheWorldsDigest` for the digest |

## What the reverts killed

Sixteen reverts, each restored before the next. The one survivor is recorded because it was real.

| | Reverted | |
|---|---|---|
| M1 | `Name()` never takes the class-14 form | killed |
| M2 | field C widened from three bits to four | killed |
| M3 | field D widened from five bits to eight | killed |
| M4 | field A's shift moved one bit | killed |
| M5 | the icon extension becomes the sprite one | killed |
| M6 | the layer address takes `secondary`, not `primary` | killed |
| M7 | the composed code loses field C | killed |
| M8 | `HeroBodyFor` reads the whole code, not field D | killed |
| M9 | the open gate accepts any selection size | killed |
| M10 | the open gate stops checking the subject's own id | killed |
| M11 | the window never closes on its own | killed |
| M12 | the toggle opens without checking eligibility | **survived, then closed** |
| M13 | a transparent layer pixel is painted over the base | killed |
| M14 | the layer is painted with no base beneath it | killed |
| M15 | the unread list stops naming the icon address | killed |
| M16 | the builder ignores the party's own entity id | killed |

**M12 was unwitnessed, not equivalent.** `ToggleInventory` is exported and its own documented gate is
"a closed one opens only when the current selection is exactly its own subject's character". On the
map arm, `closeInventoryIfIneligible` runs later in the **same tick** with no early return between the
two, so a toggle that opened unconditionally is closed again before any `App.step` observer can see
it — and every criterion in the package went through `App.step`. A caller holding a `*Viewer` without
an `App`, which is what `cmd/mapview` holds, has no second statement to mask it.
`TestToggleInventoryOpensOnlyForItsOwnSubjectsSingleSelection` asserts the method directly and goes
red on the revert.

## The defect the evidence stage found

The first render of a shipped figure came back with a **picture-shaped hole** in the window.
`drawInventoryPicture` wrote the source pixel verbatim, and both pictures this window is handed are
mostly transparent — a figure sheet is a body on an empty canvas, an icon an object on one — so the
picture's own emptiness was stamped over the opaque frame, and the running world would have shown
through the figure box and through the one occupied cell. It is a straight breach of FR-7's "on the
cell's own ground", and **the acceptance tests beside it did not fail**: they asserted that an icon
reaches its cell and that a nil picture leaves the ground alone, and neither of those distinguishes
compositing from copying.

Fixed inside T3: the draw composites source-over on premultiplied channels.
`TestATransparentPictureLeavesTheGroundBeneathItOpaque` is written from the defect rather than from
the requirement's wording — a wholly transparent subject must leave the composed surface identical to
an empty one, and every pixel of the window must stay opaque whatever it is handed. It goes red on
the revert.

The reason a rendering criterion missed this is worth keeping: every criterion the package had was
about **where** a picture lands, and none about **what survives beneath it**.

## Reconciliation

Four things in the contract needed resolving in this stage rather than being followed as written.

**T2 had to touch `pkg/game/hero.go`, which its own fence assigned to T4.** D-3 renames
`Equipment.SetRow` to `SetCode`; `partyBody` calls it, so the rename does not compile without that
one line. `tasks.md`'s header requires each task to leave the tree building on its own, and the
header wins over the fence. T4 then needed no change there, and reports so.

**T3's fence made D-8 unbuildable.** D-8 puts the subject on **the viewer**, and every other
mission-lifetime value already arrives that way — `SetFont`, `ShowReadout`, `SetAttackPointer`;
`pkg/game`'s mission open holds a `*ui.Viewer` and no `*ui.App` at all. But `Viewer`'s fields are
declared in `viewer.go`, which the fence's three-file list excluded, so the first implementation put
the window on `App` and flagged it. Corrected inside T3: the state is on `Viewer`, beside the panel's.

**FR-3's third address takes no item code.** P-1 reads as though all three addresses are built over
the namer, but FR-3 itself says the figure base "is a face number directly inside the same directory".
`ItemFigureBasePath` therefore takes a directory and a face, not a code. Routing a face through the
item namer would invent the correspondence P-5 forbids in the other direction.

**`plan.md` accounted for no FR-1, FR-2, FR-10 or FR-11.** The audit cannot see this until every task
has landed — a story with an unlanded task returns before that check — so it surfaced here. The four
are now cited where they belong, in D-1, D-6 and SC-5. No decision changed.

## Landed tests that moved

Three, all in `pkg/data/equip_test.go`, and all forced by the type change rather than chosen:

- The accessor spelling throughout, and the fixture helper's parameter.
- `TestFillingOneSlotLeavesEveryOtherSlotEmpty` became
  `TestFillingOneSlotAnswersTheWholeCodeThatWasWritten` and now round-trips a code with all four
  fields nonzero — a slot that carried only field D through would have passed the old assertion.
- `TestARowNamingNoEntryProducesNoName` lost its "a row before the list's start" case, which is
  **unrepresentable** now: field D is five unsigned bits, so a stored row is always in `[0, 31]`. It
  was replaced by a case at D's own maximum. `HeroBodyFor` keeps its `i < 0` guard for totality and
  that guard is now unreachable from the package's public surface — stated here rather than left for
  someone to discover as dead code.

In `pkg/game`, three `data.Weapon` fixture literals gained a `Code:` field beside their `Row:`.
Without it every such fixture would read as an *empty* slot and
`TestMovingTheWeaponMovesTheDerivedBody`'s four distinct rows would collapse to one case. No
assertion moved.

## Open, and deliberately so

**FR-8's "reported once" is satisfied on the headless path only.** The builder returns the addresses
it could not read, as D-11 requires, and `FrontEnd.MissionLine` — what `-check -mission N` prints —
names them. The **windowed** path discards the list, because `MissionOpener`'s closure has no output
stream and giving it one reaches outside `pkg/game`. So a player who starts a mission in the window
with art missing sees the absence drawn and reads nothing about it. Nothing is reported per frame,
which is the clause FR-8 exists to prevent; nothing is reported at all on that path, which it does not
say is acceptable.

**`data.Weapon` states both `Row` and `Code`, and `Code`'s field D is that row.** FR-5 asks for the
code "beside the numbers it already yields", so this is the contract rather than a redundancy that
crept in; `Row` now has no consumer outside tests.

**Eleven slots stay empty and the four cut items stay cut** — the compositor, the paint order, the
`secondary` sheets, the two-handed swap. None of them is observable while one slot can be occupied,
and none was built.
