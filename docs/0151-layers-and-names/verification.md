# 0151 — verification

## Result

The shipped build uses the decoded held-layer swap, paints paired equipment on both sides, keeps a
crossing unit above sacks and corpses, names items from the selected install, shows a framed hover
popup, and returns a double-clicked worn item to the pack.

There is no `tasks.md`; one lane carried the retrospective slice. Every FR, AC, P, DD and SC is
accounted for below.

T1's unconditional weapon-last rule was superseded by T7's conditional weapon/shield swap. T3's
origin-row rule was corrected by T9's later-of-origin-and-destination rule. The shipped result is T7
and T9, not the earlier rules.

## Gate

Run from the story worktree at its branch head:

| Command | Result |
|---|---|
| `go build ./...` | clean |
| `go test -count=1 -trimpath ./...` | all packages passed |
| `bash scripts/check-sdd-audit.sh` | no FAIL |
| `bash scripts/check-doc-budget.sh` | ok |

The orchestrator seat also established `go vet ./...`, `gofmt -l $(git ls-files '*.go')`,
`scripts/check-no-game-assets.sh` and `scripts/check-hotfix-ledger.sh` as clean on the merged code at
`0239a7f`. They were not re-run in this artifact lane. The deletion set `dfbc7eb..0239a7f` is empty.

`check-sdd-audit`'s note and warning count is not comparable from a worktree without `builds/`; only
the FAIL set is enforced.

## Figure composition and depth

**FR-1, AC-1, AC-2, AC-3, P-1, DD-1, DD-2 and DD-3.**
`TestFigureDrawOrderIsAPermutationWhicheverHeldSlotIsLast` proves all twelve slots occur exactly
once and exercises both held-order arms. `TestFigureHeldLastMatchesTheSixLiteralBodyNames` covers
the six matches, a one-handed name, empty slot, nil list and out-of-range row.
`TestComposeUnitFigurePaintsInFigureDrawOrder` and
`TestComposeInventorySubjectPaintsEveryOccupiedSlotInFigureDrawOrder` identify the last layer by
its pixel at a shared coordinate. `TestComposeUnitFigurePaintsWhicheverHeldSlotIsLast` and
`TestComposeUnitFigurePaintsTheWeaponLastForATwoHandedBody` exercise the swap in the map composer.
`TestComposeUnitFigurePaintsTheSecondSideOfAPairedSlot` and
`TestComposeUnitFigureLeavesAnUnpairedSlotAtItsPrimaryAlone` distinguish the four-slot secondary
boundary.

**FR-2, AC-4, P-2 and DD-9.** `TestAMovingUnitStaysAboveTheSackItLeaves`,
`TestAMovingUnitDrawsAboveTheSackItEnters`, `TestAMovingUnitStaysAboveTheSackSideways`,
`TestAMovingUnitStaysAboveTheCorpseItLeaves` and `TestAStationaryUnitAboveTheSackIsUnchanged` cover
both row directions, an equal-row crossing, both object classes and the unchanged stationary tie.
The southward test was run by its implementation lane against the T3-only tree and failed there; it
does not pass independently of the correction.

## Stored names and popup

**FR-3, AC-5, AC-6, AC-9, P-3, P-4, DD-4 and DD-5.** The
`pkg/formats/itemname` tests cover little-endian positional pairing, CRLF and LF, empty lines, both
shorter-input directions, empty input and byte identity through values in `0x80..0xAF`,
`0xE0..0xEF` and the unmoved high block. Source inspection establishes that integer division ignores
an odd final key byte and map assignment makes the last non-empty duplicate win; no named test
isolates those two malformed-input cases. `TestReadItemNamesPairsBothFiles` verifies the archive
join; `TestReadItemNamesIsSilentWhenEitherFileIsMissing` verifies non-fatal absence.
`TestItemNameConsultsTheStoredTableBeforeWeaponFromCode` and
`TestItemNameRecoversAnArmourCodeFromTheStoredTable` prove stored-name priority.

T12 was also checked by the orchestrator seat against the lawful Russian install. Six shipped names
spread across the key range were drawn through the same font path as the game, with the selector
read from that install; all six were legible Russian. The installed `text/itemname.txt` measured
9,582 bytes: 1,547 in `0x00..0x7F`, 6,038 in `0x80..0xAF`, 1,997 in `0xE0..0xEF`, and zero in
`0xB0..0xDF` and `0xF0..0xFF`. Its high bytes occupy exactly the two source blocks moved by the
existing converter. This is an orchestrator-seat measurement, not a test result.

**FR-4, AC-7, AC-8, DD-6 and DD-7.** `TestItemInfoLinesStatesAResolvableWeaponsDamageAndCombat`
requires the damage bounds and both combat fields. The armour, unresolved-code and no-table tests
cover their narrower line sets. Slot and pack tests require index-for-index text.
`TestHoveredItemInfoAtNamesTheWornCellUnderTheCursor` and its pack counterpart exercise both hit
tests. Empty cell, absent cursor and no-info tests exercise popup refusal.
`TestComposeItemPopupPaintsTheInventoryFrame` identifies border and fill pixels; presentation tests
place the picture at the cursor seam. Direct source inspection confirms the three-pixel clearance,
the fourteen-pixel offset, far-edge reduction, zero floor for an oversized box, and the final draw
position after the pickup log. The worn and pack hit tests are the only successful info routes.
Source inspection also shows the popup reads cursor and info arrays and issues no command.

## Unequip and state boundary

**FR-5, AC-10, AC-11, AC-12, P-5 and DD-8.** `TestUnequipMovesTheSlotsCodeIntoTheContainer` covers
append and merge. `TestUnequipTouchesNoOtherSlotAndNoOtherEntity` bounds mutation.
`TestUnequipRefusesAnAbsentEntityAnOutOfRangeSlotAndAnAlreadyEmptySlot` compares every refusal's
canonical bytes and digest with a quiet step, and includes a live control whose digest moves.
`TestAWorldHoldingAnUnequipRoundTripsByteIdentically` covers marshal/load/re-marshal.

The worn-box UI tests cover the last accepted frame, one frame beyond it, different slots, an empty
geometric cell and independence from the pack click window.
`TestUnequippingAnArmourPieceDropsItsDefenceAndAbsorption` equips armour with non-zero contributions,
unequips it through the command queue and requires both contributions to disappear while weapon
damage stays fixed.

**FR-6 and DD-10.** The implementation adds no serialized field and does not modify the format
version. The round-trip and refused-command byte comparisons cover the simulation boundary. Name
and popup structures occur outside `pkg/sim`; render-only changes do not enter the digest. The
pickup text builder remains bare and the new frame is local to the hover popup.

## Disclosures

- **Mage layer gap.** `HERO-FIGURE-059` gives mages a separate composition order in which the head is
  inserted among equipment layers, and mage slot 9 is tagged but not colour-blitted. The code does
  not disclose or implement that split; it paints the base, then applies the common order to every
  figure. A player can see equipment painted over a mage's head where it should pass behind it, and
  can see slot-9 equipment that the original would leave invisible.
- **Body-list overrun.** `heropicture.txt` can be indexed past its end. What the original does is
  Unknown. This build treats that as an unresolved body name and paints slot 2, the shield, last.
  This is an authored answer to an undecoded case.
- **Incomplete characteristics.** A shield gets its name alone. A class-14 magic item resolves
  through neither the weapon nor armour table and gets no characteristics line.
- **Damage fields.** A weapon states `DamageBase` through `DamageBase + DamageSpread`, matching the
  character sheet's DMG composition. `DamageBase` and `DamageSpread` are not a minimum and maximum.
- **Paired-sheet interleave.** The build paints a secondary sheet immediately after its primary.
  The fully decoded compositor places some secondary sheets later and has the separate mage order
  above. This is an authored simplification, not an exact reproduction of every inter-layer edge.
- **Font capacity.** The byte-wide font subscript can select at most 224 shipped records on selectors
  other than 1 and 160 distinguishable glyphs on selector 1. Lifting the selector-1 aliasing changes
  the meaning of shipped Russian bytes and requires rewriting that text; adding atlas records alone
  does not lift it.

## Success accounting

| Id | Witness |
|---|---|
| SC-1 | Figure and depth suites above. |
| SC-2 | Name/popup suites and six-name Russian run above. |
| SC-3 | Unequip simulation, UI and game tests above. |
| SC-4 | Gate table and empty deletion set above. |

AC-1 through AC-12, P-1 through P-5, FR-1 through FR-6 and DD-1 through DD-10 are each named beside
the evidence that witnesses them; the ranges in this sentence are summaries, not their witnesses.

## Limits of the evidence

No screen capture of the full inventory window was made in this lane. Popup geometry, frame pixels,
line content and hover routing are witnessed synthetically. The six Russian names were observed by
the orchestrator seat through the real font path, not through the inventory window itself.
