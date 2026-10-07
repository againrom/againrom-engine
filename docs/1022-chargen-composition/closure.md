# closure.md — 1022, the character generator's canonical composition

As-built, after round-3 adversarial review (the final round for this story, three-pass ceiling). The
aspect matrix, witness and reconciliation below describe the tree as it stands after round 3's fixes,
not a pass-by-pass account of what each round found; the findings themselves and their fixes are
named where they changed a row's own evidence.

## Twelve-aspect matrix

| Aspect | Result | Note |
|---|---|---|
| Data | PASS | Two new archive entries load and are size-validated: `chrgen/fullstatsl.bmp` (160x242, card background) and `chrgen/fullstatsr.bmp` (16x242, card seam), `pkg/game/chargenassets.go`. `TestLoadChargenAssets` covers both; `cmd/againrom`'s shared install fixture covers every subtest that builds a full synthetic install. |
| Runtime state | PASS | `ChargenPresentation.CardBackground`/`CardSeam`, `ChargenSetup.TipTextDetail` and `Chargen.message` (round-2 P-3) are populated at load, construction and `SetDetailMessage` respectively; no field is left at its zero value on a successful load. |
| Simulation | N/A | No file under `pkg/sim` is touched. SIGHT's fractional digit (canon shows `6.1`, `6.4`, `6.5`; this build prints a whole number) is explicitly out of scope for this reason: printing it means storing `Derived.Sight`'s dropped sub-cell remainder, which changes that field's own representation and reaches `pkg/game`'s `ScanRange` and the actor's own sight — hashed simulation state, a different contract from this story's card composition (round-2 coordinator scope note, 2026-08-21). Research bearing on the divisor is in progress at another seat on a branch this story's pin does not carry; no claim is cited here for it. |
| Player input | PASS | The detailed page swallows a press/release the tip panel covers (`stepChargenDetailed`); the town's cells and buttons answer in both DOLL and STATS mode; the shop's doll drag-and-drop (`shopGridControlAt`) and its hover/tooltip test (`ShopHoverLines`, round-2 P-2) both refuse a hit during Statistics, in the same order. Round-2 review found the two had disagreed: `ShopHoverLines` tested the doll slot before `Statistics`, so a populated slot's item-name tooltip still appeared over the card though the click already refused on the same pixel — fixed by reordering the check. |
| AI | N/A | Not touched. |
| UI/HUD | PASS | The story's primary aspect. B1-B6, `spec.md`. Round-2 fixed five player-visible defects in this aspect (P-1 through P-5, listed below its own row); round 3 fixed two more, both found only once the witness stopped hiding them behind a near-empty subject: the persistent chevron/mode-box/Book chrome and the member-name row drawing over the card's own now-populated rows in Statistics mode for a real party member (Defect A), and `AlignValues`' shared right edge overflowing the card's own 160px width for a real leveled character's three-digit HEALTH/MANA and two-digit DEFENSE/RESISTANCE values (Defect B, a regression inside round-2's own P-5). Both are mutation-proved at their use sites; see Witness. |
| Triggers/scripts | N/A | Not touched; mission 10/20 UNSUPPORTED counts are unchanged (Milestone census, below). |
| Inventory/equipment | PASS | The shop's own doll slot (equip/unequip surface) is disabled, not removed, during Statistics; `TestShopStatisticsPlacesTheCardInTheCharacterPaneAndDisablesTheDoll` and round-2's `TestShopHoverLinesMissesTheDollDuringStatistics` assert the miss on both the click and the hover surface. |
| Persistence/save-load | N/A | No byte-form field, no save-format change. |
| Campaign/session | N/A | Not touched. |
| Shipped content | PASS | `check-release-tests.sh` (35 of 35 install-gated tests selected and passed, both roots) and `check-scenarios.sh` (13 of 13, both roots) run against real installs; `cmd/plaquescreens` renders the detailed page for both classes through the production composer against the EN root (Witness, below). Round-2 found this row's own round-1 evidence overstated: the round-1 text claimed the border-strip pixel test covered "every one of the four border-column slots" (D-4) when the test's own subtest list was nav/plate/doll seam and doll body — three slots, not four, none of them the card seam. `TestReleaseChargenDetailedSeamColumnsDrawShippedStrips` now carries a fourth subtest, "card seam", closing the gap; see Witness. |
| Interactions with existing mechanics | PASS | The town-square NPC-prefix auto-Talk convenience (pre-existing) still fires correctly once cells are reachable during Statistics — `TestHeadlessActivateRoutesThroughTownStatisticsSurface`. The cross-family shop drag-release guard covers a drag started before a mid-drag mode toggle. |

No in-scope GAP remains after round 3. Round 1's blanket "No in-scope GAP" line (D-5) was false as
written: P-1 through P-5 (round 2) and the two round-3 findings above were each a real player-visible
gap in either Shipped content or UI/HUD, undisclosed at the landing that preceded their own fix. All
seven are fixed and carry their own evidence above and in `spec.md`. WEIGHT's absence (`DIV-209`) and
SIGHT's fractional digit are real gaps against the owner's canon screenshot, but both are explicitly
out of this story's own domain boundary (Client/Assets) rather than in-scope, and are recorded as
such rather than silently dropped.

## Card composition: rows this box drops, and why

`CompactPanelLayout` narrows `AuthoredPanelLayout`'s own row set to what the owner's 2026-08-21
screenshot shows fitting a 160x242 box (`pkg/ui/panel.go`, `CompactPanelLayout`'s own doc comment,
which names this document as where the list is recorded). Dropped, none of them on the screenshot:
`SELECTED` (`PanelFieldCount`), `WEAPON`, `WORN`, `SWING` paired with `ALWAYS HITS`, `CELL`, and the
`GENERAL` skill slot (present in `AuthoredPanelLayout`'s five-skill block as a sixth row for this
tree's own worn-general-skill affordance; the screenshot's skill block has five rows, not six).
WEIGHT is also not drawn, but is not in this "narrowed to the screenshot" list: it is on the
screenshot, at a non-zero, per-character value, and this build has no field to show (`DIV-209`).

`interface/chrgen/centerarea.bmp` (320x480) has no reference anywhere in this tree and is not drawn
by this story. It is out of scope: this story's own centre-column rect
(`chargenColumnDestination`) is 320 wide (`x:[160,480)`) at the decoded size, and no code path in
this build reads `centerarea.bmp` at all, decoded or otherwise — the same disclosure `spec.md`'s
"Out of scope" section carries.

## Witness

`cmd/plaquescreens -assets <en root> -png <dir>` composes the detailed page through
`ui.ComposeChargenFrame` after `SelectPreChoice`, the same call the shell itself makes, for both the
fighter (`chargen.png`) and the mage (`chargen-mage.png`), against the EN root's real shipped art.
Re-rendered after round 2's fixes (P-1 through P-5): the card's `BODY`/`HEALTH` row and the name now
sit inside the shipped frame's own ornate border rather than crowding its top-left corner, the name
is centred over the card rather than flush left, `MANA` reads `0/60` for a mage and `0/0` for a
fighter with no pool rather than being absent, and every column's own values (`HEALTH`, `MANA`,
`ABSORB`, `DEFENSE`, the five element rows, `SIGHT`, `SPEED`) sit at one shared right edge rather
than each at its own row's natural width.

**Pixel measurements (three independent readings, round-2 P-1).** A throwaway dev tool
(`game.LoadTownCharacterPaneArt`/`game.LoadChargenAssets`, not committed) dumped `fullstatsl.bmp`
and `textbackr.bmp` as PNGs from both preserved roots; a settled-edge scan (most frequent colour is
the fill, first column/row where a quarter of pixels match it is the settled edge) puts
`fullstatsl.bmp`'s own interior at column 18, row 16 on both EN and RU, and `textbackr.bmp`'s at
column 7, row 16 on both. `compactPanelInset` computes `Pad = settled + (2,2)`: `(20,18)` and
`(9,18)`. Cropping `cmd/plaquescreens`'s own composed `chargen.png` to `chargenCardBox` and running
the identical scan on the composite gives `(18,16)` — the same reading as the raw bitmap, confirming
the measured pad reaches the actual draw call rather than being computed and discarded. The owner's
own canon screenshots (`pipeline/archive/owner-ruling-2026-08-21-stats-card/canon-card-fergard.png`,
`canon-card-danath-named.png`) settle at column 23 in both, which agrees with each other but not
numerically with the composed frame's 18, because the owner's own screen crop includes a few pixels
of the card's visible gold border that `chargenCardBox`'s exact rect excludes; the three readings
are structurally consistent (fixed inset measured, applied, and reproduced in the render) rather
than pixel-identical against an owner crop of unknown origin.

`check-release-tests.sh`'s `TestReleaseChargenDetailedSeamColumnsDrawShippedStrips` is the border-strip
claim checked mechanically, pixel by pixel, against both roots. Round 2 added its fourth subtest,
"card seam" (`chargen_release_test.go`), closing `DIV-189`'s own closure condition, which round-1's
text had claimed met without it (D-3/D-4). `check-scenarios.sh`'s `0163-chargen-mission10.json`
drives the generator through `create_character` on both roots and reaches the map screen; it
exercises the PreCreate stage's own flow rather than the detailed page's card, tip panel or toggle
specifically. No scenario in this repository drives the town's DOLL/STATS toggle or the detailed
page's tip panel end to end; those are witnessed by the pixel-comparison unit tests named in
`spec.md` and by the two renders above, not by a scripted headless drive.

**This gap was disclosed as neutral coverage debt at round 2's landing and was not neutral: round-3
adversarial review found it was hiding both of that round's own P findings.** The two unit tests that
substitute for a scripted drive
(`TestShopStatisticsPlacesTheCardInTheCharacterPaneAndDisablesTheDoll`,
`TestTownStatisticsKeepsLeftContentAndPlacesTheCardInTheCharacterPane`) used
`panelSubjectFixture()`'s near-empty values (HP 63/100, two- and three-character fields) and excluded
the full `TownCharacterPanelControls()` region, including the member-name rectangle, from their own
pixel comparison. Neither choice was wrong on its own terms at round 2's landing, and together they
excluded exactly the region and exactly the value range in which both round-3 defects were visible: a
near-empty subject cannot overflow the card's right edge, and an excluded name rectangle cannot show
that a name is drawn there. Round 3 rewrote both tests against a realistic wide-valued subject
(`panelWideSubjectFixture()`/`panelWideFont()`, `pkg/ui/panel_test.go`) that does not exclude the name
rectangle, and both are mutation-proved at Defect A's use site (townshell.go/shopscreen.go's chrome
gates). The scenario-corpus gap itself remains open — no headless scenario drives the toggle end to
end, and a follow-up scenario would still close it without a code change — but the fixture and
exclusion choices that made the substitute unit tests blind to a real defect are fixed, not merely
disclosed again.

**Geometry witness (round 2, closing `cmd/screencensus`'s recorded gap).**
`TestDetailedPageDrawsTheProductionCompactCard` and `TestDetailedPageDrawsTheProductionMessageStrip`
are now `ScreenChargen`'s own `GeometryTests` entries in `pkg/ui/screenregistry.go`, both proved
mutation-sensitive at their draw-call use sites (AGENTS.md rule 6, reworded by story 1024): a +1px
shift on `chargenCardBox.Min` at its `copyNative` call site fails the first; the same shift on
`drawChargenMessage`'s own `x` computation fails the second, at 80 pixels, after the test was
rewritten to compare the composed frame against an expectation the test draws itself rather than
scanning only for "some pixel inside the rect, none outside" (the shape AGENTS.md rule 6 names as
insufficient). Both mutations were reverted byte-identical. `cmd/screencensus`'s own count: 0
unwitnessed composing screens, 0 known geometry gaps (was 1, `ScreenChargen`'s own
`GeometryGapNote`, before this round).

**Round-3 witness.** Two mutation-proof cycles, both reverted byte-identical
(`git diff` checked after restoring):

- Defect A (chrome/name overlay, `pkg/ui/townshell.go`, `pkg/ui/shopscreen.go`): disabling the three
  Statistics-mode gates (the name-row skip, the chevron/mode-box plate, the shop's Book plate) makes
  `TestShopStatisticsPlacesTheCardInTheCharacterPaneAndDisablesTheDoll` and
  `TestTownStatisticsKeepsLeftContentAndPlacesTheCardInTheCharacterPane` fail (exit 1); restoring
  makes both pass (exit 0).
- Defect B (value overflow, `pkg/ui/panel.go`): disabling `layoutLines`' two new truncation passes
  makes `TestCompactPanelLayoutRightValuesFitTheBox` and
  `TestCompactPanelLayoutTruncatesAWideValueRatherThanOverflow` fail (exit 1); restoring makes both
  pass (exit 0).

`TownCharacterPanelControls()` is renamed `TownCharacterPersistentControls()` (three rectangles, not
four) and `pkg/game/townpanes_release_test.go` (install-gated, `check-release-tests.sh`) is updated to
the new name. That test's own "tavern character panel stats mode" and "school character panel stats
mode" subtests compare a composed Statistics-mode frame against the shipped `panes.Stats.Body`
bitmap; both subtests set `HasSubject: false`, so `townCharacterMemberName` returns `""` and nothing
paints in the name rectangle in either mode (`drawTownShellText`'s own `s==""` guard) — before this
round the name rectangle was excluded from that comparison regardless, so this real-asset,
install-gated test could not have shown the overlay even though its own subject made the overlay a
no-op. Leaving that rectangle unexcluded now makes this test a live witness, on both preserved roots,
that nothing paints there in Statistics mode, for the tavern and the school — the two rooms
`ComposeTownSurface` shares (`TownSurfaceKind` has exactly two values). The shop's own equivalent
population is covered by `TestShopStatisticsPlacesTheCardInTheCharacterPaneAndDisablesTheDoll` above,
with `HasSubject: true` and a populated card, which is the state Defect A actually reproduced in.

## Milestone census

`go build -o <tmp>/mr ./cmd/missionrun`, then `AGAINROM_ASSETS=<root> <tmp>/mr -mission {10,20}
-trace -ticks 1 | grep -c UNSUPPORTED`, on both roots:

| Checkout | en m10 | en m20 | ru m10 | ru m20 |
|---|---|---|---|---|
| This branch, round 3 (`a63f56a`) | 0 | 0 | 0 | 0 |
| This branch, round 2 (`dcfe9cb`) | 0 | 0 | 0 | 0 |
| `master` (`1a257fe`) | 0 | 0 | 0 | 0 |

Unchanged in every direction. This story, including round 3, touches no script-execution code; the
two fixes in this round are both `pkg/ui` composition. `pipeline/milestone-baseline.txt`'s own per-
mission census carries no `cannot run` line for `m10` or `m20` on either root, consistent with 0.

## Research reconciliation

`TOWN-234`, `TOWN-313` and `TOWN-314` (`EXP-0207`) are consumed as decoded fact: the four
border-column blits and their order, the fourth child's own rect, and the tip popup's absolute rect.
`TOWN-312` is consumed for one of the four border-seam archive entries (`fullstatsr.bmp`, the card's
own seam) — round 2 (D-8) found the loader's own comment had mis-cited it for `fullstatsl.bmp`, the
card's background, which `TOWN-312` does not name; corrected in `pkg/game/chargenassets.go`.
`SHOP-FIGURE-041` is consumed for the shared 160x242 character-panel object B6 places the card into.
`UNIT-PANEL-011` is consumed as the negative that keeps the card's own row arrangement and order
owner-authored (`DIV-191`); round 2 (D-1) found this same claim had also been cited for whether the
WEIGHT row exists to be shown, which it does not address, and split that question into its own row
(`DIV-209`). No claim was requested from or received by research during this story; every cited row
was already on the pin the story opened against (`7491e5d`), read whole through
`go run ./tools/claim`. SIGHT's fractional digit is not reconciled here: the coordinator's scope
note excludes it from this story, and no claim is cited for it since the research bearing on it sits
on an unmerged branch this pin does not carry. Round 3 requests no new claim: both its findings are
engineering defects in this build's own compositing code, not open questions about ROM1 behaviour,
and `DIV-191` already carries the one owner ruling ("truncation is acceptable and the main
information must fit") either fix answers to.

## Divergence ledger

Opened: `DIV-190` (four body-slot archive entries, size-matched, no claim names the reader), `DIV-191`
(rewritten round 2: the card's field arrangement and row order, and its reused font — narrowed to
exclude the WEIGHT question), `DIV-193` (the DOLL/STATS toggle's card-replaces-doll behaviour,
uniform across tavern/school/shop; round 2 corrected its Implemented-behaviour and Reason cells,
D-2). `DIV-209` (new, round 2, D-1: the WEIGHT row's absence, OPEN, FIDELITY-DEBT, names story `1025`).
Closed: `DIV-189` (the lower-left border strip now draws, its closure condition actually met as of
round 2's card-seam subtest — round 1's own closing text had claimed the condition met before it
was, D-3) and `DIV-192` (the message box's dropped hover/refusal copy — round 2 gave it a
replacement channel, P-3). Returned unused: `DIV-194`..`DIV-198`. Detail in `spec.md` and in each
row's own text in `docs/DIVERGENCES.md`/`docs/DIVERGENCES-CLOSED.md`.

**The WEIGHT scope decision.** Round 1 folded the WEIGHT row's absence into `DIV-191` and recorded it
as accepted on the owner's word. The owner's actual ruling is the opposite: «нет, WEIGHT не
принимал» (2026-08-21) — dropping the row was never accepted. `DIV-209` now carries this on its own:
the row stays undrawn in this story because no field anywhere in `pkg/sim`/`pkg/game` carries a
carried-weight or encumbrance value, and building one reaches those packages, outside this story's
Client/Assets domains. The vertical space the row would occupy is left as the owner's own screenshot
shows it (a blank line between XP and SIGHT is not reproduced or removed by this story either way;
the row is simply not drawn, not re-flowed around). Story `1025` is named as the place that adds the
field and the row together.

**Round 3 amends `DIV-191` in place, opens no new row.** Its Implemented-behaviour cell now records
that the ruling's truncation half is actually enforced (`panelFitValue`, `layoutLines`'s two new
passes), not only the arrangement half round 2 implemented. Both round-3 P findings (the chrome/name
overlay and the value overflow) are covered by this one existing row; neither is a new mismatch with
ROM1 or a new owner-authored choice.

## Remaining open items

- The scenario corpus has no headless drive of the DOLL/STATS toggle or the detailed page's tip
  panel (Witness, above). A follow-up scenario would close this without a code change.
- `DIV-209` (WEIGHT) is OPEN, naming story `1025`.
- SIGHT's fractional digit is out of this story's scope (Simulation aspect, above); a divergence row
  for it is left to the seat once the pending research lands, per the coordinator's round-2 scope
  note.
- `DIV-190`'s four body-slot archive-entry choices are corroborated by exact size match, not decoded.
- Story `1023` (the mission screen's right column) is unblocked by this landing: it reuses the
  160x242 card `CompactPanelLayout` builds here (its own `contract.md` landed on master mid-story,
  merged in without conflict). Story `1023` uses `AuthoredPanelLayout` (`Size.X == 0`), not
  `CompactPanelLayout`, so round 3's truncation fix does not apply there; that screen's own field
  budget is that story's own scope.
  **Corrected 2026-08-22 by story `1027` (D-1):** `AuthoredPanelLayout`'s `Size.X` is
  `sidebarWidth`, 300, not 0, so round 3's truncation fix does apply to the mission panel and
  has applied since it landed. Measured with `cmd/paneldump` on both roots; recorded as
  `DIV-218`. Story `1023` plans on this corrected premise, not on the one this line stated.
- `layoutLines`' column-split algorithm can, in principle, still let a right-column LABEL (the fixed
  authored text, not the value round 3's fix clamps) push past the card's own right edge at a wide
  enough left column, independent of this round's fix. Not observed at the realistic font/subject
  this round tested (`panelWideFont()`/`panelWideSubjectFixture()`, whole-card ink extent x=155 of
  160), and not fixed: redesigning the column split is materially larger than a round-3
  regression-scoped fix. See spec.md, Card value truncation.
