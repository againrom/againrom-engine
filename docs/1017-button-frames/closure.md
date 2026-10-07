# 1017 — closure

As-built. What was measured, what witnesses it, what remains open.

## The twelve aspects

| Aspect | Verdict | Evidence |
|---|---|---|
| Data | PASS | `LoadTownSchoolArt` (`pkg/game/townschoolart.go`) reads and size-asserts four new nodes (`b1off`, `b1on`, `b2off`, `b2on`, 140x46 each); `LoadTownTavernArt` (`pkg/game/towntavernart.go`) reads `buttonsarea.bmp` (160x238, previously unread) and six new nodes (`button{1,2,3}{off,on}.bmp`); `LoadChargenAssets` (`pkg/game/chargenassets.go`) reads one new node, `interface/chrgen/buttonsarea.bmp` (160x238). No format change, no new archive kind. `TestLoadTownSchoolArtReadsBothClassMappings` and siblings, `TestLoadChargenAssets`, `TestLoadTownTavernArtIsAtomic` (`pkg/game/towntavernart_test.go`, round 2: the tavern's own loader had no atomicity fixture before this — fails whole on any one of the four pictures, six buttons or fifteen roster sheets missing; round 3 (D-4) widened its `cases` list to range over the fixture's own 25 keys, so all 25 are checked by name, not the six round 2 picked by hand) |
| Runtime state | PASS | `App.townSurfacePress` (`pkg/ui/app.go`) tracks the button a mouse-down began on, for FR-5's press-and-hover gate; cleared on release. No persisted counterpart |
| Simulation | N/A | `pkg/sim` is untouched. No hashed field, no serialized form, no `formatVersion` change |
| Player input | PASS | `townSurfaceButtonRect` moved from the wide widget's own origin to the shipped picture's, for both draw and hit test (one function, both callers). `TestTownShellButtonWellsSitInsideTheSharedOrigin`, and production drive through `cmd/buttonframecheck` (below) |
| AI | N/A | Not reached |
| UI / HUD | PASS | The whole subject: FR-1 through FR-6 in `spec.md`. `pkg/ui/townshell_test.go`, `pkg/game/townschoolart_test.go`, `pkg/game/towntavernart_test.go`, `pkg/game/chargenassets_test.go`, `pkg/game/townbuttons_release_test.go` (install-gated). Round 3 (D-5) closes the gap round 2 left open, that FR-4's composed chargen output had no install-gated witness at all: `TestReleaseChargenDetailedNavArtIsDrawnUnmodified` (`pkg/game/chargen_release_test.go`) checks the composed nav region against `NavArt` pixel for pixel, outside the three labelled wells, against the real install. Round 3 also adds `TestReleaseSchoolAndTavernUpperRegionMatchesShippedArtOutsideButtonWells` (`pkg/game/townbuttons_release_test.go`, P-2), checking both rooms' wide upper region against the shipped background/upper picture outside the button wells |
| Triggers / scripts | N/A | No script opcode is touched. Measured below (`missionrun`'s `UNSUPPORTED` count), not assumed |
| Inventory / equipment | N/A | Not reached |
| Persistence / save-load | PASS | FR-6's gate is `Town.taken` alone; `Snapshot.Taken []SnapshotOffer` (pre-existing) carries it through save and reload, checked by `TestATownSaveRoundTripsThroughDisk`. Round 1 and round 2 added a second, redundant latch, `Town.shown`/`Snapshot.Shown`; round 3 (D-1) found it provably dead at both its production read sites (mutating `Shown()` to a constant `false` changed the result of exactly one test, on its own direct assertions, and no install-gated scenario) and removed `Town.shown`, `Shown()`, `MarkShown()`, `Snapshot.Shown`, `restoreTown`'s Shown-restoring loop and `sortOffers(s.Shown)`; `TestPreWeaponMaterializedSaveFixtureStillDecodesItsOuterEnvelope` confirms the pre-1017 fixture still decodes. `s.Taken` is sorted (`sortOffers`, `pkg/game/save.go`) before being written, closing a Go map-iteration nondeterminism `s.Taken` carried before this story; round 3 adds `TestSaveTakenOffersEncodeDeterministicallyAcrossRepeatedSnapshots` (`pkg/game/save_test.go`, W-2): three taken offers, twenty `Snapshot`+`EncodeSave` calls, all twenty payloads byte-identical; reddens within a few runs when `sortOffers` is neutered, since Go's own map iteration order is randomized per call |
| Campaign / session | PASS | FR-6: the school's and shop's offer dialogue is accepted once per `(mission, building, index)` and not again; declining it (Escape) leaves it re-offerable rather than stuck. `TestTownOfferAcceptedOnceEscapedRepeats` (`pkg/game/town_test.go`, round 2) exercises auto-open, accept, decline-then-reopen and the merchant-press path together; round 1's citation here named a test that does not exist in this repository (D-1, round-2 adversarial review) |
| Shipped content | PASS | Every art correlation and segmentation is measured against both preserved installs and reported identical. The tavern's production wells are checked against the executable rectangles while the differing background-art peaks remain visible (`cmd/buttonframecheck`, EN and RU roots — see "The two roots' output", below) |
| Interactions with existing mechanics | PASS | The tavern's Talk dispatch, the shop's four button actions, and the character generator's Play/Reset/Back are unmodified logic, only their button geometry and (for the tavern) their background moved. `pipeline/check-scenarios.sh`: 12 of 12 (below) |

No aspect is GAP.

## The integration witness

`cmd/buttonframecheck` opens a real campaign town through `game.NewFrontEnd`, walks into the school
and the tavern through production's own row-list `Choose(i)` (the door dispatch, unrelated to and
unaffected by FR-1/FR-2/FR-5), reads each room's `TownSurfaceView` through the production
`TownSurfaceArtScreen` interface, and — for every button — hit-tests the well's own centre through
production's own `ui.TownSurfaceControlAt`, requiring a disabled button to answer no hit and an
enabled one to answer its own index, exactly as `ComposeTownSurface`'s hit test does for a live
player.

```
$ go build -o <outside-repo>/buttonframecheck ./cmd/buttonframecheck
$ AGAINROM_ASSETS=<root>/gameversions/en <outside-repo>/buttonframecheck
```

On both `gameversions/en` and `gameversions/ru` (identical output on both, byte for byte outside the
printed root path):

```
school area graphics/interface/training/buttonsarea.bmp 160x238
school button 0 graphics/interface/training/buttons/b1off.bmp 140x46 at (4,71) frac 0.1008 next 0.0363 at (4,65) ratio 2.8x black 0/6440
school button 0 graphics/interface/training/buttons/b1on.bmp 140x46 at (4,71) frac 0.2531 next 0.0634 at (4,76) ratio 4.0x black 0/6440
school button 0 well art (4,71)-(144,117) production (4,71)-(144,117) ok
school button 1 graphics/interface/training/buttons/b2off.bmp 140x46 at (4,117) frac 0.0984 next 0.0379 at (4,64) ratio 2.6x black 0/6440
school button 1 graphics/interface/training/buttons/b2on.bmp 140x46 at (4,117) frac 0.2533 next 0.0627 at (9,117) ratio 4.0x black 0/6440
school button 1 well art (4,117)-(144,163) production (4,117)-(144,163) ok
tavern area graphics/interface/inn/buttonsarea.bmp 160x238
tavern button 0 graphics/interface/inn/button1off.bmp 140x46 at (4,44) frac 0.1185 next 0.0388 at (5,172) ratio 3.1x black 0/6440
tavern button 0 graphics/interface/inn/button1on.bmp 140x46 at (4,44) frac 0.2562 next 0.0548 at (4,65) ratio 4.7x black 0/6440
tavern button 0 well art (4,44)-(144,90) decoded (4,44)-(144,90) production (4,44)-(144,90) ok
tavern button 1 graphics/interface/inn/button2off.bmp 140x46 at (4,90) frac 0.0978 next 0.0360 at (3,170) ratio 2.7x black 0/6440
tavern button 1 graphics/interface/inn/button2on.bmp 140x46 at (4,90) frac 0.2530 next 0.0607 at (4,96) ratio 4.2x black 0/6440
tavern button 1 well art (4,90)-(144,136) decoded (4,91)-(144,137) production (4,91)-(144,137) ok
tavern button 2 graphics/interface/inn/button3off.bmp 140x46 at (4,137) frac 0.0989 next 0.0379 at (3,170) ratio 2.6x black 0/6440
tavern button 2 graphics/interface/inn/button3on.bmp 140x46 at (4,137) frac 0.2530 next 0.0604 at (4,131) ratio 4.2x black 0/6440
tavern button 2 well art (4,137)-(144,183) decoded (4,138)-(144,184) production (4,138)-(144,184) ok
shop area graphics/interface/shopmenu.bmp 176x238
shop button 0 graphics/interface/shopbutton1.bmp 120x52 at (30,15) frac 0.2337 next 0.0789 at (35,15) ratio 3.0x black 716/6240
shop button 0 rect measured (494,15)-(614,67) production (494,15)-(614,67) ok
shop button 1 graphics/interface/shopbutton2.bmp 140x46 at (19,67) frac 0.1644 next 0.1065 at (18,116) ratio 1.5x black 0/6440
shop button 1 rect measured (483,67)-(623,113) production (483,67)-(623,113) ok
shop button 2 graphics/interface/shopbutton3.bmp 140x46 at (19,114) frac 0.1292 next 0.0621 at (19,64) ratio 2.1x black 0/6440
shop button 2 rect measured (483,114)-(623,160) production (483,114)-(623,160) ok
shop button 3 graphics/interface/shopbutton4.bmp 120x52 at (30,160) frac 0.1801 next 0.0688 at (25,160) ratio 2.6x black 804/6240
shop button 3 rect measured (494,160)-(614,212) production (494,160)-(614,212) ok
chargen area graphics/interface/chrgen/buttonsarea.bmp 160x238
chargen well y=[20,65) x-segments=[[26 127]]
chargen well y=[69,111) x-segments=[[6 137]]
chargen well y=[115,160) x-segments=[[6 137]]
chargen well y=[163,208) x-segments=[[24 121]]
drive SCHOOL: 2 buttons
drive SCHOOL button 0 "Train" enabled false well (484,71)-(624,117) centre (554,94) hit-test ui.TownSurfaceControl{Kind:0x0, Index:0} ok
drive SCHOOL button 1 "EXIT" enabled true well (484,117)-(624,163) centre (554,140) hit-test ui.TownSurfaceControl{Kind:0x1, Index:1} ok
drive TAVERN: 3 buttons
drive TAVERN button 0 "Hire" enabled false well (484,44)-(624,90) centre (554,67) hit-test ui.TownSurfaceControl{Kind:0x0, Index:0} ok
drive TAVERN button 1 "Talk" enabled true well (484,91)-(624,137) centre (554,114) hit-test ui.TownSurfaceControl{Kind:0x1, Index:1} ok
drive TAVERN button 2 "EXIT" enabled true well (484,138)-(624,184) centre (554,161) hit-test ui.TownSurfaceControl{Kind:0x1, Index:2} ok
buttonframecheck: ok
```

"Train" and "Hire" are correctly disabled on fresh room entry (no skill selected, no squad
selected), and `TownSurfaceControlAt`'s pre-existing rule for a disabled button — no hit at all,
unchanged by this story — is exercised, not bypassed: the drive requires no hit for those two and a
hit for the three enabled buttons.

The correlation ratio (winning fraction over next-best) is markedly lower here (1.5x-4.7x) than the
figures `contract.md`'s premise table carried (38x-102x) for the same button bitmaps. The winning
OFFSET agrees exactly in every case — the position is the fact the wells and the production hit test
depend on, and it matches both the contract's premises and production's own table. The ratio itself
is not consumed by any assertion; the difference is attributed to a different next-best exclusion
radius or scoring method between whatever produced the contract's table and this tool's own
`separation=4` constant (`cmd/schoolcheck`'s convention, reused here), and is recorded as an honest
discrepancy between two measurement methods, not resolved further.

## Research reconciliation

`spec.md`'s FR-1 and FR-5 cite facts read at the coordinator's own research branch tip during this
story, not carried in this story's frozen pin (`4f2651a`) and not cited by claim id here, per this
project's pin-discipline and B1 rules. Four facts were reported; all four are addressed:

1. **The widget's rect is the wide origin; the area picture is drawn at the picture's own origin,
   sixteen pixels inside it.** Matches this build exactly after the fix (FR-1): `TownUpperRegion` for
   the picture and the button wells, `TownWideUpperRegion` naming the wider widget rect alone. This
   settles what was `contract.md`'s premise 2 and closes the "unread destination routine" item the
   contract's own divergence list expected; no row is opened for it. Round 2 additionally drew an
   outline at `TownWideUpperRegion` to mark the widget's own edge on screen — a display choice this
   story added, not a fact research reported — and round 3 (P-2) found that choice painted a line over
   bare background sixteen pixels left of the school's own picture; the outline call is removed from
   both rooms (FR-2, above).
2. **Draw position and hit rectangle are the same rectangle.** Already true by construction here:
   `townSurfaceButtonRect` is the one function both `ComposeTownSurface`'s draw loop and
   `TownSurfaceControlAt`'s hit test call. No divergence.
3. **The ON bitmap needs two conditions: a hover index and a second field whose writer is unread.**
   This build's press-and-hover gate (FR-5) satisfies the known two-condition shape; the second
   condition's own identity is authored, not confirmed against the original's unread field
   (`DIV-155`).
4. **Exactly one of ON/OFF is blit per button, never both, after one background blit.** Confirmed
   already true here (FR-5): one `draw.Draw` call per button, background drawn once per composition.
   No divergence.

Two further, related facts were found independently during this story's own measurement, not
reported by research: the blit convention differs between rooms (school and tavern opaque, shop
keyed — `DIV-158`), and which shipped bitmap maps to which button label is authored from file and
well order in every room this story touches (`DIV-159`).

Five divergence rows are opened, `DIV-155` through `DIV-159`; none returned unused
(`docs/DIVERGENCES.md`). The contract's own premise 3, that the shop's button art was unlocated, was
wrong: the art (`shopmenu.bmp`, `shopbutton{1..4}.bmp`) and its loading and drawing
(`pkg/game/shopart.go`, `pkg/ui/shopscreen.go`) already existed before this story. This is not a
divergence — implementation and research do not disagree, a premise in this story's own contract
did — and is recorded here rather than as a ledger row.

This is the second time in this story that a review confined to a crop of the upper region alone
missed a defect visible only in the full screen frame. Round 2's own review of the character
generator's four wells worked from a segmentation run over `buttonsarea.bmp` alone, at a threshold
that dropped the fourth well below its own row-count minimum; the full-frame screenshot in that same
pass showed four identical plaques, and the count was corrected once a reviewer looked at the whole
picture rather than the segmenter's own cropped output. Round 3's P-2 outline defect was the same
shape one level up: `TownWideUpperRegion`'s outline call was checked against a crop of
`TownUpperRegion` alone in every prior test and screenshot, where the outline's own sixteen pixels of
overreach fall entirely outside the crop; only a full-frame screenshot showed the line landing on bare
background left of the school's picture. Both install-gated witnesses this round (P-1's and P-2's)
check their own region, `TownUpperRegion` and `TownWideUpperRegion` respectively, against the
shipped art at every pixel rather than against a further-cropped sub-rectangle, so a future authored
line or fill anywhere in either region now reddens a test without needing a full-frame screenshot to
be looked at by eye.

## Round 2 (adversarial review, pass 1)

Three player-visible findings, fixed in this pass:

- **P-1, no button label drawn.** The school/tavern draw loop (`pkg/ui/townshell.go`) blitted a
  button's art and skipped the label/value text unconditionally. Fixed by drawing the text after
  the art (or the fallback box) in every case. Chargen (`pkg/ui/chargen_page.go`) already drew its own
  labels before this story and was unaffected; the shop draws one number per command button and no
  word, matching `SHOP-SCREEN-035`, so it had no label to lose;
  the defect was confined to the loop this story newly wrote. Round 1's byte-exact release test,
  which compared the composed well to the raw bitmap over its whole area, positively forbade a
  label from ever being drawn there and was replaced, not merely relaxed:
  `TestReleaseSchoolAndTavernButtonArtDrawsExactlyOverItsOwnWell` is removed;
  `TestReleaseSchoolAndTavernButtonArtIsPlacedAndLabelled` and
  `TestReleaseSchoolAndTavernButtonPressedArtIsPlacedAndLabelled`
  (`pkg/game/townbuttons_release_test.go`) assert the composed pixels match the raw bitmap outside
  the label rows (placement), differ from it somewhere inside the label rows (a label is drawn),
  and that no two of a room's buttons are pixel-identical over their own wells.
- **P-2, chargen well count.** Covered above (FR-4, `DIV-156` amended): threshold 33 dropped a
  19-row well under its own 20-row minimum; threshold 40 resolves all four, and the topmost is
  drawn then suppressed rather than wired to an invented action.
- **P-3, unresearched outline on the tavern.** `ComposeTownSurface`'s tavern branch drew the
  school's `TownWideUpperRegion` outline, researched for the school (`1015`) only. Removed for the
  tavern; the picture blit itself is unchanged.

Fixed alongside, not player-visible on its own: the decline-strands-mission latch (FR-6,
`DIV-157` amended, witnessed by `TestTownOfferAcceptedOnceEscapedRepeats`); the `s.Taken`/`s.Shown`
save-order nondeterminism (`sortOffers`, `pkg/game/save.go`); the D-1 stale test citation in the
campaign/session aspect row above; and D-2, the tavern loader's atomicity fixture
(`pkg/game/towntavernart_test.go`), also cited above.

## The census and the game

```
$ go build -o <outside-repo>/mr ./cmd/missionrun
$ for m in 10 20; do AGAINROM_ASSETS=<root>/gameversions/en <outside-repo>/mr -mission $m -trace -ticks 1 | grep -c UNSUPPORTED; done
```

Mission 10: `0`. Mission 20: `0`. `pipeline/milestone-baseline.txt` carries no `cannot run` line for
either mission on either root, so the count is unchanged, as expected: this story is a UI/rendering
story and touches no script opcode.

## Gate

```
go build ./...                                                       clean
go vet ./...                                                         clean
gofmt -l $(git ls-files --cached --others --exclude-standard '*.go') clean (no output)
go test -trimpath -count=1 ./...                                     ok, 56 packages (39 ok, 17 [no test files])
bash scripts/check-no-game-assets.sh                                 check-no-game-assets: clean (tree scan)
```

`internal/archtest`'s `TestLiveTreeClean` required registering `cmd/buttonframecheck` in the DAG
allow-map (`internal/archtest/dag.go`), on `cmd/schoolcheck`'s own precedent: `pkg/game`, `pkg/ui`,
`pkg/formats/bmp`, `pkg/render/terrain`.

```
$ AGAINROM_IMPL=<this worktree> bash pipeline/check-scenarios.sh <root>/gameversions/en
check-scenarios: selected 12 scenario(s)
check-scenarios: ok (12 of 12)
```

None of the 12 scenarios activates a school or tavern button by name; the tavern rows five of them
activate (`0152-save666.json`, `0163-mission-to-town.json`, `1013-world-map-one-click.json`,
`1005-doll-carry-over-worn.json`, `1005-doll-and-shop.json`) select the tavern's own door or an NPC
cell, neither of which this story's button changes touch.

```
$ AGAINROM_IMPL=<this worktree> bash pipeline/check-release-tests.sh <root>/gameversions/en
check-release-tests: selected 25 install-gated tests, by variable:
         22 AGAINROM_ASSETS
          1 AGAINROM_ORIGINAL_SAVES
          2 AGAINROM_SAVE_666
check-release-tests: ok (25 of 25 install-gated tests ran and passed, 0 skipped)
```

Round 1 landed at 22 (19 `AGAINROM_ASSETS` + 1 + 2), against a 19-test master baseline. Round 2
removed one test (`TestReleaseSchoolAndTavernButtonArtDrawsExactlyOverItsOwnWell`, replaced for
P-1) and added two (`TestReleaseSchoolAndTavernButtonArtIsPlacedAndLabelled`,
`TestReleaseSchoolAndTavernButtonPressedArtIsPlacedAndLabelled`), a net +1 on `AGAINROM_ASSETS`,
19 to 20, giving 23 overall. Round 3 added two more, `TestReleaseChargenDetailedNavArtIsDrawnUnmodified`
(D-5) and `TestReleaseSchoolAndTavernUpperRegionMatchesShippedArtOutsideButtonWells` (P-2), 20 to
22, giving 25 overall. `pkg/game/towntavernart_test.go`'s `TestLoadTownTavernArtIsAtomic` and
`pkg/game/save_test.go`'s `TestSaveTakenOffersEncodeDeterministicallyAcrossRepeatedSnapshots` (W-2)
read no install (synthetic fixtures) and do not appear in this count. Confirmed identical on the RU
root (25 of 25).

Round 2's two new/renamed release tests were each checked against the defect they are meant to
catch. `TestReleaseSchoolAndTavernButtonArtIsPlacedAndLabelled`, with the label/value drawing
calls in `pkg/ui/townshell.go` temporarily removed (the P-1 regression restored), reddens on the
real EN install for every one of the five school and tavern buttons, each failing the
label-differs-from-raw-bitmap assertion. `TestReleaseSchoolAndTavernButtonPressedArtIsPlacedAndLabelled`
(successor to round 1's `...PressShowsItsOwnOnState`) reddens the same way with
`ComposeTownSurface`'s pressed branch temporarily read as always drawing `Buttons[i][0]` regardless
of `v.Press` (the FR-5 gate neutralized): every button reports its pressed well as pixel-identical
to the released one. Both files were restored from the pre-mutation source after each check, and
the full `pkg/game` suite plus
`AGAINROM_ASSETS=<root>/gameversions/en go test -run TestReleaseSchoolAndTavernButton -count=1
./pkg/game/...` were rerun clean afterward.

Round 3's own witnesses and fixes were each checked the same way, against the real EN install unless
noted, and every mutation was reverted and the affected suite rerun clean before moving on.
`TestReleaseChargenDetailedNavArtIsDrawnUnmodified` (P-1/D-5) reddens with the old dark-fill-and-border
suppression reinstated. `TestReleaseSchoolAndTavernUpperRegionMatchesShippedArtOutsideButtonWells`
(P-2) reddens with the school's outline call reinstated (824 mismatched pixels at the school's own
wide region). The seven mutations W-1 names against `townshell.go`'s text layout (label moved,
label/value swapped, border-coloured text, one-character truncation, disabled styling removed, no
text drawn, label drawn twenty pixels above its well) each redden
`TestReleaseSchoolAndTavernButtonArtIsPlacedAndLabelled`/`...PressedArtIsPlacedAndLabelled` against
the oracle-built rewrite. `TestSaveTakenOffersEncodeDeterministicallyAcrossRepeatedSnapshots` (W-2,
synthetic, no install) reddens within a few runs with `sortOffers(s.Taken)` neutered, consistent with
comparing against Go's own randomized map iteration rather than a fixed order. `Town.Shown()`
temporarily forced to return a constant `false` (D-1) changed the result of exactly one test,
`TestTownOfferAcceptedOnceEscapedRepeats`, at its own direct assertions on `Shown()`; no other unit
test and no install-gated scenario changed, which is the evidence `Town.shown` was dead in
production before its removal.

## Open items

- **The ON-bitmap second condition's own writer** (`DIV-155`) and **the blit slot a button reaches,
  in any room** (`DIV-158`) both need a claim not yet published to any pin this story used.
- **Which shipped bitmap is which button, in the school and the tavern** (`DIV-159`), and **the
  character generator's own well-to-action mapping** (`DIV-156`, amended round 3) are both authored
  from file and well order; no claim addresses either room's button dispatch. The fourth chargen
  well's own function in the original is separately undecoded (`DIV-156`): round 3 stopped covering
  it with an authored fill and now draws the shipped picture unmodified, leaving the well inert
  rather than assigning it an invented action.
- **ROM1's own re-entry behaviour after a school or shop offer has been shown or declined**
  (`DIV-157`, amended round 3) is unresearched; the once-accepted rule here is owner-directed for
  the shown case, and this pass's own reading of the decline case (re-offerable rather than stuck)
  is not yet confirmed against the owner's original ruling — he is being asked separately and may
  overrule it. The row now describes the mechanism as `Town.taken` alone (round 3, D-1); the
  behaviour it describes is unchanged by that amendment.
- **The school's and tavern's button labels are hardcoded English text on both roots**
  (`DIV-165`, round 3): pre-existing before this story and out of its scope. The label mechanism
  itself is undecoded (`DIV-165`, `spec.md`). Corrected at the landing (pass 3, D-1): the shop's
  four command buttons carry one number each and no word, matching `SHOP-SCREEN-035`'s own paint
  routine, and are not part of that row.
- **Everything `ComposeShopScreen` draws as text is unwitnessed** (pass 3, W-2). Its four
  command-button numbers can be removed and the hardcoded English `"Book"` caption
  (`pkg/ui/shopscreen.go`, pre-existing from `1006`) deleted, and the whole suite including every
  install-gated test stays green. The behaviour is correct; only the witness is missing. Not fixed
  here: the shop's text predates this story and this story changed none of it. A story touching the
  shop's own composition owes it.
- **The room's old release oracle shared production's well table** (pass 3, observed).
  `TestTavernButtonsUseDecodedExecutableRectangles` now holds the three executable-backed literals
  independently and checks the draw corners, button hits, and the two one-row gaps. Restoring the
  former lower-button tops to 90 and 137 fails that test. `cmd/buttonframecheck` still measures the
  shipped art on both roots, but now reports the art peaks separately from the executable-backed
  tavern expectation and compares production with the latter. Hotfix `2bf71055` closes `DIV-167`.
- **The rooms' 176-pixel plaque vocabulary is not drawn from shipped art** (`DIV-166`, opened at
  the landing). Out of this story's scope; it is the measurable half of the composition the owner
  reported on 2026-08-19.
- **The tip windows** (`main/text/tips/*.txt`) remain out of scope, per `contract.md` — story
  `1018`'s contract already exists on `master` as of this story's own branch point.
- **The tavern's own NPC-selection Talk flow, the shop's four button actions, and the character
  generator's window layout beyond the nav frame** are unmodified by this story, per `contract.md`'s
  own scope.
- Round 2's own adversarial review, pass 1, found three player-visible defects (P-1 through P-3,
  above); all three were fixed in round 2. A further adversarial review, pass 2, found the chargen
  fourth-well fill did not follow the plaque's own shape (a second P-1) and a stray outline line on
  the school (a second P-2, raised mid-pass), and found the label witness (W-1), the save-determinism
  fix (W-2) and the `Town.shown` mechanism (D-1) each unwitnessed or, for D-1, provably dead in
  production; every finding is addressed above and in `spec.md` (round 3).
- **Pass 3 found no player-visible defect and the story landed on it.** Three passes is the ceiling
  a story of this size states; the contract named four behaviours over three domains with no reach
  into hashed simulation state, and the chain ended one pass inside it. Pass 3 returned three
  document findings, all corrected at the landing, and two witness findings. It also re-ran, from
  the reviewer's own seat, the seven label mutations pass 2 named, all seven reddening the two
  install-gated button tests; the round-3 lane had reported them from a compacted session's
  carried-forward summary rather than from its own measurement, and that attribution is now backed
  by measurement. Its adversarial pass over the two new round-3 witnesses found neither vacuous:
  reinstating the removed school outline reddens the upper-region witness at 824 pixels with the
  first at (464,0), and the chargen witness covers the whole navigation region rather than a crop.
- **W-1 was closed with a witness rather than a ledger row**, at the landing:
  `TestReleaseChargenDetailedNavLabelsAreDrawn` (`pkg/game/chargen_release_test.go`) asserts, per
  navigation control, at least one pixel in the label colour, a bounding box centred in the
  control's own rectangle, and a drawn width agreeing with what the production font measures for
  that control's own string. Mutation-verified from this seat against `pkg/ui/chargen_page.go`:
  drawing no text, drawing the label in black, and swapping Back and Play each redden it, with the
  file restored to a byte-identical sha256 after each. It passes on both roots, where the shipped
  wording is `Back`/`Reset`/`Accept` and `Назад`/`Сбросить`/`Принять`, measuring 24/30/36 and
  31/48/42 pixels. The class it closes is the one this story shipped in round 1: art drawn, label
  not.
