# Story 1027 — closure

The statistics card composes at font2 in the tavern, the school and the shop. Every value the
subject states is drawn, all sixteen rows are inside the card, the name is centred, and the shop
draws nothing over it. Behaviour as built is `spec.md`.

Pointable result: `builds/current/`, press STATS on a party member in any town room.

## Twelve-aspect matrix

| Aspect | Verdict | Evidence |
|---|---|---|
| Data | N/A | No format, field or shipped byte is read differently. The story changes which font atlas an existing composer is handed. |
| Runtime state | PASS | `TownCharacterView.CardFont` is the one added field. `ShopScreenView.Book` keeps its value across the mode, and `shopCharacterView`'s fallback carries `CardFont` through. |
| Simulation | N/A | Nothing under `pkg/sim` is touched and no hashed state is reachable from a font or a draw call. `internal/archtest` unchanged. |
| Player input | PASS | `shopScreenControlAt` answers no control over the card. `TestShopStatisticsPlacesTheCardInTheCharacterPaneAndDisablesTheDoll` asserts both directions: a press on `shopBookRect` misses while Statistics is on and hits while it is off. The three persistent controls and the chevrons still answer. |
| AI | N/A | No unit behaviour is reachable. |
| UI/HUD | PASS | `TestReleaseStatisticsCardFitsAtTheProductionFont` (subtests `town` and `chargen`) and `TestReleaseShopDrawsNothingOverTheStatisticsCard`, both roots. |
| Triggers/scripts | N/A | No script opcode or trigger path is reached. |
| Inventory/equipment | PASS | The shop's grids, drag surfaces and doll suppression are unchanged, and the same test asserts them in the run that checks the `Book` change. |
| Persistence/save-load | N/A | Nothing is added to a save record. `CardFont` is resolved per frame from the loaded assets. |
| Campaign/session | PASS | `check-scenarios` 13 of 13 on both roots, including `0163-mission-to-town.json`, which enters the town rooms from a played mission, and `0163-chargen-mission10.json`, which enters mission 10 from the generator. |
| Shipped content | PASS | Both installs. The town card is measured against the shipped starting party member on each root, and the generator card against a played-through generator at budget exhaustion. |
| Interactions with existing mechanics | PASS | The generator's detailed page shares `CompactPanelLayout` and takes B4's centring; its own install-gated witnesses pass on both roots. The mission panel shares `layoutLines`; its truncation was measured, left unchanged and recorded as `DIV-218`. |

No in-scope GAP.

## Witnesses

`pkg/game/towncard_release_test.go` holds the three install-gated tests. They render through the
production composer at the production font against a real install, because a synthetic font cannot
witness a font budget: the assertion is about one atlas's advances against a fixed 160-pixel canvas.

Ink is read as the difference from the card's own shipped background bitmap, and the scan stops at
card-local y=205, above the three persistent controls. This is the instrument
`TestReleaseTownSquareOverlayDoesNotPaintBlackOverTheSky` already uses. An earlier form of the test
diffed against the same card composed with an empty `PanelSubject`. That is not a text-free
reference: an empty subject still resolves every row and draws a full card of zeros, so the mask was
the exclusive-or of two texts.

`pkg/ui/panel_test.go` holds `TestCompactPanelLayoutCentredNameDoesNotMoveTheAlignedColumn`, a
synthetic test for the two halves of B4 that needs no install.

## Mutation proofs

Each mutation was applied to the production use site a maintainer would change, not to a helper's
own return statement, and reverted with the same edit pair. Every revert was compared by sha256
against the file's pre-mutation bytes.

| | Site | Mutation | Witness | Result |
|---|---|---|---|---|
| M1 | `pkg/game/townshell.go` | `CardFont: t.f.tipFont()` to `t.f.Font` | `TestReleaseStatisticsCardFitsAtTheProductionFont` | KILLED: `row 2 ("") had its right value truncated from "145/145" to "1" at the production font` |
| M2 | `pkg/ui/townshell.go` | the composer reads `v.Font` instead of `v.cardFont()` | same | KILLED: `the card's text ink spans (9,21)-(160,205), outside its own padded box (9,18)-(151,224)` |
| M3 | `pkg/ui/shopscreen.go` | the `Book` chrome drawn in Statistics mode again | `TestReleaseShopDrawsNothingOverTheStatisticsCard` | KILLED: `differs from the shared character region at 425 pixel(s), first at card-local (68,4)` |
| M4 | `pkg/ui/shopscreen.go` | `ShopControlBook` back in the live-in-Statistics case list | `TestShopStatisticsPlacesTheCardInTheCharacterPaneAndDisablesTheDoll` | KILLED: `Book answered {Kind:12 ...} over the statistics card, want a miss` |
| M5 | `pkg/ui/panel.go` | the placement guard drops `!it.center` | the release test above, and `TestCompactPanelLayoutCentredNameDoesNotMoveTheAlignedColumn` | KILLED by both: `the name row's ink spans x=[73,108) ... centred at 90.5 against the card's own centre 80.0`, and `the centred short name spans a doubled centre of 212, want the card's own 160` |
| M6 | `pkg/ui/panel.go` | the edge-accumulation guard drops the centred skip | `TestCompactPanelLayoutCentredNameDoesNotMoveTheAlignedColumn` | KILLED: `row 1's value pen moved from x=68 to x=128 when only the centred name changed` |

M6 survived every install-gated test before the synthetic one was written. The shipped party cannot
witness it: a centred name moves the shared column only when its own measured width exceeds the
widest ordinary left cell's edge, and at font2 a shipped name's ink is 35 pixels against 58. The
synthetic test supplies a name wide enough, then asserts that the ordinary rows' pens do not move.
That test carries both name widths, because a name wider than the shared edge is never shifted by
the alignment pass and so cannot witness M5.

All six reverts were byte-identical, and the working tree after the mutation runs matched
`git diff --stat` taken before them.

## Measurements against the contract's

The contract's figures are the adversarial review's. These are this lane's, measured with
`ui.CharacterPanelReport` through `panelItems` and `layoutLines`, against the shipped starting party
member on the `en` root. Two disagree.

| | Contract | Measured | Instrument |
|---|---|---|---|
| left column's widest cell at font1 | 120 px | 120 px | `CharacterPanelReport` |
| shared right-column origin at font1 | 126 | 126 | same |
| right-column budget at font1 | 16 px | 16 px | same |
| right values truncated to the empty string | all twelve | 9 of 11 value-bearing cells, with 2 more shortened to one character | same |
| rows never drawn | SIGHT and SPEED | SIGHT at y=242 and SPEED at y=258 on a 242-pixel card | same |
| mission panel's widest row | 367 px | 299 px | `cmd/paneldump` |

The right-column count differs because three of the sixteen rows carry a right heading and no value
(HEALTH, MANA, RESISTANCE), so eleven right cells state a value rather than twelve, and
`panelFitValue` has no one-character floor: at a 16-pixel budget it left `1` for `145/145` and `0`
for `0/0` rather than emptying them. The mission panel figure differs because it is a different row.
299 px is the shipped starting party member's `WORN` row, measured on both roots.

## D-1, D-2, W-1

**D-1.** `AuthoredPanelLayout`'s `Size.X` is `sidebarWidth`, 300, not 0. Only its `Size.Y` is 0, and
`layoutLines`' truncation pass reads the X axis, so that pass has run on the mission panel since
story `1022` landed it. Measured before it was recorded, with `cmd/paneldump`, identically on the
`en` and `ru` roots: the shipped starting party member's `WORN Short Sword, Soft Mail, Soft Boots`
row measures 299 pixels against a 280-pixel budget and draws as `Short Sword, Soft Mail, ` on
missions 10, 20, 30 and 40. On mission 40 a placed unit loses more: its
`WEAPON Uncommon Steel Two Handed Sword` row (301 px) draws as `Uncommon Steel Two `, and its
seven-piece `WORN` row (884 px) draws as `Two Handed Sword, Pla`. `cmd/paneldump` now prints the cut
rows and each cut value's own measured width, so the figures come from a committed command. The
three documents that stated the false premise are corrected in place: the code comment in
`pkg/ui/panel.go`, and story `1022`'s `spec.md` and `closure.md`. The mission panel itself is
unchanged; the behaviour is recorded as `DIV-218`.

**D-2.** `DIV-191`'s justification cell asserted that the card's font is already fitted to each
page's other rows and that the card composes at the same width budget those rows use. That is the
premise that produced the illegible card. The cell is amended in place, dated and attributed,
quoting what it said and giving the measured refutation. The row's own decision, that the
arrangement stays the owner's, is unchanged.

**W-1.** Story `1022`'s statement that the generator cannot reach card overflow came from a program
deleted with its worktree. `TestReleaseChargenMaximumAllocationFitsTheCard` reproduces it. It drives
`ui.Chargen` through `SelectPreChoice`, `Forward` and round-robin `AdjustStat` until the budget
refuses, logs the derived subject, and asserts the composed card truncates nothing. On the `en` root
it reports `remaining=0 HP=91/91 Mana=0/0 Defence=17 ToHit=45`, matching the numbers `1022`'s
`spec.md` quoted. That spec now names the test beside them.

## Divergence ledger

`DIV-218` allocated with `pipeline/next-div-id.sh`; none returned. Type UNKNOWN, status OPEN, in the
"Authored where research is silent" table. It cites `UNIT-PANEL-011`, which is High as a negative
about its own instrument: the unit panel's cached value block is addressed by computed index from
`+0x14a`, so no claim states whether the original shortens a row that does not fit or does something
else. The row closes when the mission panel either draws the whole row or records a decoded reason
not to.

`DIV-191` amended, as above. No other row touched.

## Gates

Implementation, at the branch tip: `go build` 0, `go vet` 0, `gofmt -l` empty, `go test -count=1
-trimpath ./...` 0. `scripts/check-claim-citations.sh` 0, 1189 distinct citations resolve against
1400 claims. `scripts/check-no-game-assets.sh` 0, clean.

Research, at the pinned submodule: `go build` 0, `go vet` 0, `go test` 0,
`scripts/check-claim-ids.sh` 0 (1400 ids, 31 ledgers), `scripts/check-retraction-status.sh` 0 (229
overturned ids). This story does not move the pin.

| Seat gate | en | ru | baseline |
|---|---|---|---|
| `check-release-tests.sh` | selected 38, ok 38 of 38, 0 skipped | selected 38, ok 38 of 38, 0 skipped | 35 |
| `check-scenarios.sh` | ok 13 of 13 | ok 13 of 13 | 13 |

The release-test selection rose from 35 to 38, which is this story's three new install-gated tests
and nothing else. `internal/gatedtests/testdata/population.txt` lists the same 38.

## Game census

`missionrun -mission 10 -trace -ticks 1` and `-mission 20`, EN root: 0 and 0 `UNSUPPORTED` lines.
`pipeline/milestone-baseline.txt` records no `cannot run` line for `m10` or `m20` on either root, so
both were 0 before this story. Unchanged, as expected for a presentation-only story. This story's
result is the visible one, not a census movement.

## Open items

- `DIV-218`, the mission panel's truncated `WORN` row. Not this story's to fix.
- `panelFitValue` has no minimum, so at a small enough budget it can leave one character of a value.
  It is not reachable on the town card at font2, where nothing is shortened at all, and no divergence
  row is opened for it. A later layout that narrows the card will meet this behaviour.
- The card was not observed on screen by a person at this landing. It is witnessed by pixel
  comparison through the production composer against both installs.
