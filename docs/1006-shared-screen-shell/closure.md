# 1006 — shared screen shell — closure

## State

The story implementation is complete. The branch merged implementation master `7fc09de`, including
story 1005, and retained its shop doll hover, plain unequip, drag, suppression and cancellation paths
through the shared character region.

## Twelve-aspect matrix

| Aspect | Status | Evidence |
|---|---|---|
| Data | PASS | `ReadCampaign` reads `MercenaryCount`, `Mercenaries` and `EnableMercenary`; `NPCDefs` keeps independent `PriceA` and `PriceB`; production EN/RU routes resolve every tavern and school asset. |
| Runtime state | PASS | `Town` owns separate pool, unlock and hired arrays. One town-wide member index feeds tavern, school and shop. Cell, mode, Book and press/double-click state remain presentation state. |
| Simulation | N/A | No `pkg/sim` source, command, world-hash fixture or simulation binary-form version changed. Siege hires are converted at `StartMission` through their exact Units row before they enter the existing world. The outer save payload did change and is accounted for under Persistence. |
| Player input | PASS | Shared upper-before-left routing, same-control press ownership, 350 ms same-cell double-click, member arrows, mode, Book, class raster masks and modal capture are exercised through production dispatch. The production headless seam resolves the same visible surface cells and buttons, including the legacy NPC activation as cell selection followed by Talk. |
| AI | N/A | No AI policy or unit-order decision changed. |
| UI/HUD | PASS | One `DrawTownCharacterRegion` paints tavern, school and shop; the detailed generator shares the regions but keeps preview-only semantics. Tavern and school use shipped art with a total text fallback. |
| Triggers/scripts | N/A | No trigger producer or opcode changed. Mission 10 and 20 each report zero unsupported instants on both roots. |
| Inventory/equipment | PASS | The shop member, pack, table, Book and shared character projection pass. Story 1005's hover, plain unequip, both drag directions, carried icon, ordinary and suppressed masks, cancellation and weapon-materialization paths remain single production paths. |
| Persistence/save-load | PASS | Envelope version 1 and nested simulation form 53 remain distinct. Version 1 carries pool, unlock and hired arrays through additive gob fields. The exact pre-field envelope decodes, the current writer is digest-pinned and its output decodes, and pre-field saves infer unlocks and hired types from completed missions and the restored party. Hero level, per-slot XP, purse and party containers survive the production path. `DIV-095` records that the outer byte form changed without an envelope-version change. |
| Campaign/session | PASS | Completion applies mercenary unlocks; tavern and school dialogues remain repeatable and modal; room exit returns to the square; member changes cannot target a stale hero. |
| Shipped content | PASS | `TestReleaseSharedTownRoute` passes separately on EN and RU. It loads the campaign, NPC definitions, school masks/icons, tavern child panels/unit sprites, dialogues, shop, save envelope, siege templates and both chargen stages. |
| Interactions with existing mechanics | PASS | Existing shop trade arithmetic, table ownership, selected-member rebinding, town offers, mission entry, chargen validation and every merged 1005 shop-doll interaction pass together. |

No matrix cell has an in-scope gap.

## Acceptance mapping

| Acceptance | Witness |
|---|---|
| AC-1, AC-3, AC-3a | `TestTownShellUpperOverlapAndTavernBoundary`, `TestTownShellCharacterControlsShareOneGeometry`, the shared `townCharacterView`, and existing shop picker tests. |
| AC-2 | `TestTownDialogueOwnsInputAboveTheSharedSurface`; existing shop-dialogue tests cover the shop arm. |
| AC-4 | `TestTavernWholeSquadHireAndReturnAreImmediate` and `TestTavernOfferRequiresMissionUnlockAndStockAndUnaffordableHireIsAtomic`. |
| AC-5 | The EN/RU route opens, renders, closes and reopens shipped NPC dialogue and authored mercenary Talk. |
| AC-6 | `TestTownSchoolUsesClassSpecificRasterMask`, `TestLoadTownSchoolArtReadsBothClassMappings`, fixed price witnesses, selected-member training and the unaffordable mutation guard. General is absent from all ten view cells. |
| AC-7 | `TestSchoolPricesAndTrainingPersistThroughSave` and the EN/RU route's shipped school dialogue and second save/load boundary. |
| AC-8 | Full `pkg/game` and `pkg/ui` suites; `TestBookIsIndependentAndKeepsTheTradeTableAcrossMembers`. |
| AC-9 | Post-merge `pkg/ui` Shop/Doll/Inventory tests and `pkg/game` Shop/Doll/Weapon/SharedTown tests. `TestTheCharacterPanelIsTheBorrowedRegionAndTheFigureFillsIt` fails when the shared painter call is removed. |
| AC-9a | The Book test above plus the production shop table → Book → member change → EXIT route. |
| AC-10, AC-11, AC-12, AC-12a | The existing full chargen suite plus EN/RU pre-create → detailed → rejected reserved Accept → Back → valid Accept route. |
| AC-13 | Nil-art shell fixtures and disabled-control tests; startup carries optional school/tavern art errors rather than failing. |
| AC-14 | EN and RU production routes and all eleven production scenarios below. |
| AC-15 | Diff census finds no `pkg/sim` source, sim command, world-hash fixture or simulation format-version change. The separate outer save-byte change is pinned and disclosed under Persistence and `DIV-095`. |

## Production witness

These commands were run after merging implementation master `7fc09de` and pinning research master
`fcdb02e`:

```text
AGAINROM_ASSETS=<seat>\gameversions\en go test -trimpath -count=1 ./pkg/game -run ^TestReleaseSharedTownRoute$ -v
AGAINROM_ASSETS=<seat>\gameversions\ru go test -trimpath -count=1 ./pkg/game -run ^TestReleaseSharedTownRoute$ -v
AGAINROM_IMPL=<seat>\wt-1006 bash pipeline/check-scenarios.sh <seat>\gameversions\en
AGAINROM_IMPL=<seat>\wt-1006 bash pipeline/check-scenarios.sh <seat>\gameversions\ru
```

Both routes pass, and each scenario run selects and passes eleven of eleven files. The scenario gate
initially exposed the shared shell swallowing the old headless row activation: four campaign routes
stopped at the selected tavern NPC instead of opening Talk. The repaired headless path resolves the
visible shell cell and button through `clickTownSurface`; EN and RU then both passed 11/11.

Each route uses `NewFrontEnd` and a real campaign. It resolves both school masks and all
thirty class-icon states, all three tavern child surfaces and type 1/type 15 roster art; pages and
reopens a shipped tavern dialogue; opens mercenary Talk twice; hires a whole squad; saves and
restores it; preserves a live shop table through Book and member change; clears it on EXIT; returns
the squad; trains exactly one class skill and saves/restores it; opens a shipped school mission
dialogue; mints Catapult and Ballista into mission 10 from their Units rows; rejects `Self`; then
completes a valid detailed generation and reaches the map.

No synthetic pointer input was sent to the owner's desktop. The route drives the production
front-end seams headlessly; an observed window remains the owner's build witness after landing.

## Milestone census

The implementation was built as `cmd/missionrun` and driven with the owner's `AGAINROM_ASSETS`
environment form:

```text
en mission 10 UNSUPPORTED=0
en mission 20 UNSUPPORTED=0
ru mission 10 UNSUPPORTED=0
ru mission 20 UNSUPPORTED=0
```

`pipeline/milestone-baseline.txt` carries no `cannot run` row for mission 10 or mission 20 on either
root, so the before value is also zero. The census is unchanged, as expected: this story's concrete
result is the visible town/generation composition and its economy interactions, not a trigger gap.

## Falsification

The full implementation suite passes with `go test -trimpath -count=1 ./...`. Four mutations were
run one at a time against the committed production line, observed red, and restored from `HEAD`
without a shared stash:

| Claim | Mutation | Failure |
|---|---|---|
| Unlock is an independent tavern filter | Removed `MercenaryEnabled` from `tavernMercenaries` | `TestTavernOfferRequiresMissionUnlockAndStockAndUnaffordableHireIsAtomic`: locked type 3 became visible. |
| An unaffordable Train is atomic | Bypassed `Town.spend` | `TestSchoolUnaffordableTrainingIsAtomic`: skill 1 changed from 10 to 11 and gained Carry XP at 199 gold. |
| School input consumes the class mask permutation | Changed fighter mask byte `0xff` from slot 0 to slot 1 | `TestTownSchoolUsesClassSpecificRasterMask`: the painted fighter point no longer resolved its enabled cell. |
| Hired state is persisted, not reconstructed accidentally | Replaced the three mercenary arrays written by `snapshotTown` with zero arrays | `TestTavernWholeSquadHireAndReturnAreImmediate`: restored purse/party survived but hired state became false. |
| Shop composition uses the shared character painter | Removed `DrawTownCharacterRegion` from `ComposeShopScreen` | `TestTheCharacterPanelIsTheBorrowedRegionAndTheFigureFillsIt`: the figure pixel remained the background value 12 instead of 77. |
| The hostile-gob test locates the nested Swing field after Snapshot grew | Changed the Swing-field needle byte from `0x02` to `0x03` | `TestDecodeRejectsHugeGobMapCountBeforeAllocation` stopped at its locator assertion with `current Snapshot gob lacks the released residue suffix`. |
| Production headless activation consumes the shared tavern surface | Changed the NPC semantic prefix from `NPC` to `NPCX` | `TestHeadlessActivateUsesSharedTownSurfaceControls`: the dialogue advance count remained 0 instead of 1 because Talk was never pressed. |

## Research and divergence reconciliation

The implementation consumes research pin `fcdb02e3f38e610ac6b54148056dee37836dfd64`. It uses the
three-set roster (`MERC-SHELF-002`), whole-squad type semantics (`MERC-TYPE-001`,
`MERC-HIRE-003`), normal price formula (`MERC-PRICE-004`), exact human/siege templates
(`MERC-LEVEL-005`), school art and mask geometry (`TOWN-017`, `TOWN-018`, `TOWN-019`, `TOWN-061`,
`TOWN-067`, `TOWN-068`), and the purchase mutation/price (`HERO-SKILLBUY-076`).
`TOWN-GENERAL-106` establishes five selectable school skills, no General and ROM1's
`1,2,4,3,5` mapping. `TOWN-GENERAL-107` and `TOWN-GENERAL-108` close the former producer Unknown.
`HERO-GENERAL-086` establishes General as slot 0 with no maintained base copy. The new PathMap,
spell-visual and combat-control claims on this pin do not change the town-room, shop or detailed
character-generation contracts.

`DIV-017` closes because the screen-local shop character painter is gone. `DIV-015`, `DIV-016`,
`DIV-018` and `DIV-020` remain accurate and are not duplicated. `DIV-113` through `DIV-119` record
within-chapter unlock timing, alternate flat price mode, detailed-chargen composition,
rejected-Accept destination, generic mercenary Talk, shop Book and tavern double-click. `DIV-120`
remains returned unused. `DIV-121` records the owner's school order against ROM1's decoded order.
The former purchase-producer row is withdrawn because the pin resolves it. `DIV-095` now records
the version-1 outer gob byte-form change and its compatibility witnesses. No other reader-visible
mismatch was found.

## Remaining work

The implementation lane has no remaining gap. Independent adversarial review is a separate
post-push gate and is not self-certified here.
