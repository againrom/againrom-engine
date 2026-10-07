# Story `1035` — Valuable Documents: closure

**As built, 2026-08-23.** Branch `1035-valuable-documents`, base `1fcf7e0f`, research pin `d7ee0c6`.
Behaviour is in `spec.md`. Claim provenance is in `contract.md`. This document is the evidence.

## Twelve-aspect matrix

| Aspect | Verdict | Evidence |
|---|---|---|
| Data | PASS | `AddTextDocument` and `AddPictureDocum` read into `Chapter.TextDocuments` / `Chapter.PictureDocuments` through the existing `sectionIntSlice` reader (`pkg/game/campaign.go`). `data.RaisesDocuments` is the class-14 row-28-modulo-32 mask and `data.QuestDocumentCode` is `0x0e1c` (`pkg/data/itemcode.go`). Witnessed by `pkg/game/documents_test.go` and `pkg/data/raisesdocuments_test.go`, and on shipped content by `TestReleaseShippedCampaignDocumentsAllResolve`. |
| Runtime state | PASS | `Town.documents` and `Town.docMission` (`pkg/game/town.go`); the one-shot request on `Viewer` (`pkg/ui/viewer.go`); the five flow fields and `ScreenDocuments` (`pkg/ui/flow.go`). Witnessed by `pkg/game/documents_test.go` and `pkg/ui/flow_test.go`. |
| Simulation | PASS | The panel's own dispatch arm now runs the map-hold seam's three statements before `stepDocuments` (`holdMapUnderDocuments`, `pkg/ui/app.go`), so no held span reaches `sim.World` beyond the same bounded catch-up the in-game menu already pays. `pkg/sim` itself is still untouched (`git status --porcelain pkg/sim` empty against base `1fcf7e0f`) — this row was originally scored against that fact, which is a statement about the **package**, not about the **shipped behaviour**: a screen can reach hashed simulation state through `pkg/game`'s pacer without `pkg/sim` itself changing a line, and this aspect had exactly that gap until adversarial review pass 1 (P-1) found it. Witnessed by `TestDocumentsPanelHoldsTheMapLikeTheInGameMenu` (`pkg/ui/holddocs_test.go`), which drives the same held span under the panel and under the in-game menu and requires the same trace on the seam: zero world ticks and zero animation ticks during the hold, a declared stop, and the same bounded resume on the first frame back. |
| Player input | PASS | Double-click on the pack cell and drag-to-doll both reach `enqueueEquip` (`pkg/game/world.go`) and raise the request; three controls with press-latched release semantics and Escape (`stepDocuments`, `pkg/ui/app.go`). Witnessed by `pkg/ui/documents_test.go` for the controls and by `scenarios/1035-documents-mission10.json` for the double-click, the arrows, OK and Escape. |
| AI | N/A | No actor behaviour, order, or decision path is read or written. |
| UI/HUD | PASS | `ScreenDocuments` is registered in `pkg/ui/screenregistry.go` with its selection `Test` and two `GeometryTests` entries, both mutation-proved at the composer's own draw call sites (below). `cmd/screenshot -screens documents` composes it from a real install. |
| Triggers/scripts | N/A | No script check or instant reads or writes the collection; the grant is the campaign registry's, applied at the mission opener. The script-gap census is unchanged (below). |
| Inventory/equipment | PASS | The item is carried and never worn: `EquipSlotFor` refuses class 14 and `enqueueEquip` returns before enqueueing anything. `partyInputs.Documents` places it from `ChargenParty` and `MissionPartyAs`. Witnessed by `pkg/game/chargen_test.go` and by the scenario, which asserts the pack holds code 3612 and then re-opens the panel from the same cell after closing it, so the item is still there and was not worn. |
| Persistence/save-load | PASS | `Snapshot.Documents` and `Snapshot.DocumentMission`; `restoreTown` replays each element through `addDocument`, so a duplicate pair restores one element. Witnessed by `pkg/game/documents_test.go`, including a round trip, a duplicated pair and an old-payload decode, and by `pkg/game/save_test.go`'s envelope fixture. |
| Campaign/session | PASS | `Town.CollectDocuments(n)` is called once from `missionOpenerMode`, guarded monotonically and deduplicated on the `(value, kind)` pair. Witnessed by `pkg/game/documents_test.go` for replay, re-entry and the guard, and by `pkg/game/frontend_session_test.go` for the opener's own state population. |
| Shipped content | PASS | `TestReleaseShippedCampaignDocumentsAllResolve` resolves every value the campaign registry grants: **4 text and 1 picture elements over 24 `[Mission]` sections**, on both roots. `TestReleaseDocumentPanelArtAndFontLoad` loads the eleven bitmaps and font4 (**224 glyph records**, both roots). |
| Interactions with existing mechanics | PASS | The request is drained below the popup gate, so a notice standing over the map takes the double-click and the panel does not open behind it; the scenario dismisses the notice mission 10 raises and then opens the panel. Two pre-existing gaps this story made reachable were found and fixed. `Shop.Sell` sold zero-price places (`DIV-306`, closed at this landing), witnessed by `TestSellKeepsAPlaceTheShopWillNotPayFor`. The panel's own dispatch arm carried no equivalent of `holdMapUnderMenu` (adversarial review pass 1, P-1): a running mission's world clock and ambient animation clock both took the whole held span in one jump on dismissal instead of the bounded catch-up the in-game menu already enforces. This aspect's original PASS was scored against `pkg/sim` being untouched, a fact about the package rather than about the shipped behaviour, and it is why the gap was not caught at first landing. Fixed by `holdMapUnderDocuments` and a `docUp` half of `Viewer.popupOpen()` alongside `menuUp` (`pkg/ui/app.go`, `pkg/ui/popup.go`, `pkg/ui/flow.go`), witnessed by `TestDocumentsPanelHoldsTheMapLikeTheInGameMenu` (`pkg/ui/holddocs_test.go`). |

No in-scope GAP.

## Integration witness, in a real campaign mission

`scenarios/1035-documents-mission10.json`, 88 steps, run by `check-scenarios.sh` on both roots.

It generates a character, enters campaign mission 10, and asserts the collection is three elements
and the primary's pack carries item 3612. It dismisses the notice the mission raises, double-clicks
the pack cell through the production pointer edges, and then walks the panel: forward to the next
element, back to the previous element's first page, refused at the collection's first end, one page
forward and back inside an element, twenty steps forward to the last element, refused at the far end,
closed with OK, re-opened at the first element, and closed with Escape. Every panel assertion is
against the panel's own state, read through `HeadlessDocumentState`.

The scenario asserts element index, element count, page index and kind. It does not assert the page
count except on element 0, which is one page on both roots: the count is language-dependent, because
the panel wraps the install's own text with the install's own font, and element 1 fills four pages on
the English root and five on the Russian one.

**`cmd/screenshot -screens documents`** reaches the same panel through the same production
double-click and writes the composed frame. It was run on both roots. The English capture draws the
Royal Edict text and the Russian capture draws the same document in the Russian install's own words,
both on the shipped sheet, with the arrows and the OK plate in place.

## Shipped-content and code-path sweep

**Assets, both roots, listed with `cmd/restool list`.** `main.res` carries `text/docs/1.txt` through
`4.txt`. `graphics.res` carries `interface/docs/1.bmp`, `interface/docs/sheet.bmp`, three left
arrows, three right arrows and four OK bitmaps. The file sets are identical on the English and
Russian roots; the four text files differ in size and the eleven bitmaps are byte-identical.

**Registry corpus, both roots.** `[Mission10] AddTextDocument 1,2,3`, `[Mission50] AddTextDocument
4`, `[Mission60] AddPictureDocum 1`. Five elements over three of the 24 `[Mission<n>]` sections. The
granted set and the shipped file set are the same five values, so no shipped document file is
unreachable and no granted value is unresolvable.

**The picture arm is reached by a committed command.** No scenario reaches mission 60, so the
scenario walk exercises text elements only. `TestReleaseShippedCampaignDocumentsAllResolve` exercises
both arms and fails if the corpus stops carrying either kind, which is why it asserts the two counts
rather than only the resolutions.

**What the sweep cannot see.** It reads what `LoadDocumentPage` produces, not what the panel draws
from it. The one shipped picture is 464x344 against a 456-pixel content rectangle; the 8px overhang
is a spec decision (`spec.md`, B2), witnessed against the shipped bitmap by
`TestReleaseDocumentsPanelComposesTheShippedPictureElement` (`pkg/ui/documents_release_test.go`,
added in adversarial review round 2), which composes the real picture through the production
composer and confirms the overhang draws rather than clips, on both roots.

## Mutation proof

**The panel's geometry.** The two registered `GeometryTests` entries were proved at the composer's
own draw call sites in `composeDocumentsPanel`, not at the rectangle-returning helpers. Seven
mutations: the sheet origin, each of the three control destinations, the content origin's two axes,
and the line pitch. All seven were killed. `pkg/ui/documents.go` was restored byte-identical
afterwards, sha256 `362cbc0f29c49f52889695a755f7974ec1fb6ab523c1628c89ac7920e6ca18f1`.

**The rules behind the panel.** Six further mutations, one per production line this story adds or
changes outside `pkg/ui`. Each was applied, run, and reverted with a byte-identical sha256 check on
the file.

| Mutation | Test that reddened |
|---|---|
| `Shop.Sell`'s zero-price keep removed | `TestSellKeepsAPlaceTheShopWillNotPayFor` |
| `CollectDocuments`'s monotonic guard removed | `TestCollectDocumentsGrowsOnceAndOnlyForward` |
| the two grant loops swapped, so a picture is appended before a text | `TestCollectDocumentsGrowsOnceAndOnlyForward` |
| `addDocument` deduplicating on the value instead of on the pair | `TestTheCollectionDeduplicatesOnThePairAndNotOnTheValue` |
| `addPictureDocumentKey` spelled at its full length | `TestTheCampaignReadsBothDocumentGrantKeys` |
| `RaisesDocuments` narrowed to `c == QuestDocumentCode` | `TestRaisesDocumentsAdmitsTheEightRowsAndNothingElse` |

**One mutation survived and it is equivalent.** Weakening the guard from `n <= t.docMission` to
`n < t.docMission` changes nothing observable: the two differ only when a mission is collected at
exactly the guard's own value, and at that value the grants are the same set the deduplicating append
already absorbs. The guard's `<=` is redundant with the dedup and is kept because it states the rule
in one place. The mutation that does have an observable effect is removing the guard, which is the
row above.

**The first version of the guard test did not catch that mutation.** It re-collected mission 50 after
going back to mission 10, which restores the guard's value whether or not the guard exists. It was
found by running the mutation rather than by reading the assertion, and the test now reads the guard
on the statement after going back. This is recorded because the assertion looked correct.

## Gates

Run in this worktree, at the story's own tree.

| Gate | Result |
|---|---|
| `go build ./...`, `go vet ./...` | clean |
| `gofmt -l $(git ls-files --cached --others --exclude-standard '*.go')` | prints nothing |
| `go test -trimpath -count=1 ./...` | 43 packages ok, exit 0 |
| `scripts/check-claim-citations.sh` | exit 0, 1265 distinct citations resolve against 1479 claims and 220 experiments under 787 prefixes |
| `scripts/check-no-game-assets.sh` | exit 0, clean (tree scan) |
| `pipeline/check-div-claims.sh` | exit 0, 196 live rows, 272 cited claim ids, header declares 9 cells per row |
| `check-release-tests.sh`, English root | exit 0, 44 of 44 install-gated tests ran and passed, 0 skipped |
| `check-release-tests.sh`, Russian root | exit 0, 44 of 44 install-gated tests ran and passed, 0 skipped |
| `check-scenarios.sh`, English root | exit 0, 15 of 15 |
| `check-scenarios.sh`, Russian root | exit 0, 15 of 15 |
| `check-milestone.sh` | exit 0, the script gap and the drive are where they were recorded, both roots |

The install-gated population moved from 41 to 44 across this story's own three release tests, two
added at first landing and a third (`TestReleaseDocumentsPanelComposesTheShippedPictureElement`)
added in adversarial review round 2. That population is a checked-in manifest,
`internal/gatedtests/testdata/population.txt`, and `TestScanMatchesTheCheckedInPopulationList` fails
until a new gated test is listed in it, which is how each was added. `check-release-tests.sh`
measures the population itself and reported 44 on both roots, split 41 `AGAINROM_ASSETS`,
2 `AGAINROM_SAVE_666`, 1 `AGAINROM_ORIGINAL_SAVES`.

**Script-gap census.** `missionrun -mission <n> -trace -ticks 1`, UNSUPPORTED lines: mission 10 = 0
and mission 20 = 0, on both roots. `pipeline/milestone-baseline.txt` records mission 10 as 16 checks,
27 instants, 12 triggers and mission 20 as 14 checks, 15 instants, 11 triggers, all supported. This
story moves no census number and does not claim to: its pointable result is the screen in
`builds/current/`, reachable in a played campaign on both language installs.

## Research reconciliation

| Row | Taken as | Where this build differs |
|---|---|---|
| `MENU-DOC-009` | eleven bitmaps, three hit rectangles, 21-line pages, the page-then-element arrow order, the OK teardown | The panel is a screen rather than an overlay (`DIV-299`); the phase gate is this build's map arm (`DIV-298`); Escape is an added exit (`DIV-304`); the OK index sentence was taken over its own closing clause (`DIV-301`); the element is entered at its first page in both directions (`DIV-300`) |
| `MISSION-DOC-021` | the two path rules, the record's field set, the reader | The collection is written into this build's own `Snapshot` rather than the original's save location. The row is Medium on that location and the story does not claim the original's (`DIV-324`) |
| `REG-SCN-097` | the two registry keys, the kind assignment, the corpus | The "only grows" clause is Medium; this build enforces monotonic growth as a guard rather than asserting the original does |
| `ITEM-DOC-053` | the gate mask, the item identity, the one entry point | The item is placed at character creation, which no shipped producer does (`DIV-305`, DEVIATION, Owner directive). `R0240`'s equipment store has no counterpart here (`DIV-297`). The object-field third of its own gate is not applied (`DIV-323`) |
| `ITEM-DOC-054`, `DAT-DOC-021`, `SHOP-DOC-029` | the original's silence on placing the item, established positively | Same row, `DIV-305` |
| `ITEM-NAMEKEY-037` | the packed code `0x0e1c` and the name-key formatter | None |
| `TEXT-API-007`, `SPR16A-FONT-018` | font4 as this panel's font and the `.16a` atlas class | The palette index of each glyph pixel is dropped and the 4-bit level kept (`DIV-303`) |
| `DLG-WRAP-009` | the text is wrapped to the document rectangle | The break rule itself is undecoded; this build uses its own greedy word wrap (`DIV-302`) |
| `SHOP-SELL-010` | both halves of the sell test | The second half was unimplemented before this story (`DIV-306`, closed here) |
| `TOWN-352` | `campaign+0x3dc` as a bitmask | This build has no equivalent field (`DIV-298`) |

**Three Mediums were respected as Mediums.** "Nothing places it" (`ITEM-DOC-053`), "only grows"
(`REG-SCN-097`) and the save location (`MISSION-DOC-021`) are each written into their divergence rows
as scoped findings, not as settled facts. `DIV-305`'s revisit condition names one observation
consistent with a fourth producer family: the preserved original save `666 - mission 20` asserts
`documents: 1` in two pre-existing scenarios, so that playthrough's primary carries the access item.

## Divergence rows

Twelve. Ten within the reserved range `DIV-297`..`DIV-306`, allocated before implementation. Two
more, `DIV-323` and `DIV-324`, were allocated during adversarial review round 2 for two mismatches
the first landing left uncited; both are outside the reserved range. Eleven are open in
`docs/DIVERGENCES.md`; one (`DIV-306`) was opened and closed at this landing and is in
`docs/DIVERGENCES-CLOSED.md`.

| Id | Type | Subject |
|---|---|---|
| `DIV-297` | DEVIATION | the double-click raises the panel and does not also file the item |
| `DIV-298` | DEVIATION | the phase gate is this build's map arm below the popup gate |
| `DIV-299` | DEVIATION | the panel is a screen, not an overlay over the running mission |
| `DIV-300` | UNKNOWN | an element is entered at its first page in both directions |
| `DIV-301` | UNKNOWN | `Ok_l_off` is the unused OK bitmap, taken from the claim's index sentence |
| `DIV-302` | UNKNOWN | the line breaker is this build's own greedy word wrap |
| `DIV-303` | UNKNOWN | font4's palette index is dropped and its glyphs draw as a level ramp |
| `DIV-304` | DEVIATION | Escape closes the panel, which no claim gives the original |
| `DIV-305` | DEVIATION | the player is given the access item at character creation (Owner directive) |
| `DIV-306` | DEVIATION | a zero-price place was sold anyway (opened and closed here) |
| `DIV-323` | FIDELITY-DEBT | `ITEM-DOC-053`'s dropped object-field test in `RaisesDocuments` |
| `DIV-324` | FIDELITY-DEBT | the documents collection's save-file location against `DIV-095`'s envelope form |

## Open items

- **`DIV-301` is a coin the story called on a structural argument.** `MENU-DOC-009` contradicts
  itself about which OK bitmap is unused. The row states both readings and what would settle it.
- **The picture element is drawn by no scenario.** Mission 60 is not reached by any committed
  scenario, so no headless integration witness walks the panel's picture branch through
  `HeadlessDocumentState`. The composed frame itself is witnessed on shipped content by
  `TestReleaseDocumentsPanelComposesTheShippedPictureElement` (`pkg/ui/documents_release_test.go`,
  added in adversarial review round 2), which composes the real picture through the production
  composer and compares it against an independently built frame, on both roots.
- **The panel's entry point is Medium.** `MENU-DOC-009` is Medium on the panel having one shower in
  the image. This story builds the one entry point the row gives and does not close the question.
- **`DIV-297`'s second outcome is a no-op here.** The original's one routine both raises the panel
  and stores the item into an equipment slot array this build does not have. If a later story gives
  this build a carried-item slot array, that row is the one to re-read.

## Scope

Five behaviours under one contract, across five domains. `pkg/sim` itself is untouched, but the panel's
map-hold seam reaches `sim.World`'s own tick pacer the same way the in-game menu does — a fact
about the shipped behaviour, not about the package, and the distinction this story's own aspect
matrix originally missed (Simulation, above). The contract records the five behaviours and the five
domains. The story was not cut too large: no behaviour was dropped, and the two additions beyond the
contract were the shop sell fix and the map-hold fix, both pre-existing gaps this story's own change
made reachable.
