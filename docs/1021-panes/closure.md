# Closure — 1021-panes

## Result

B1 (shared column pane), B2 (per-root tip height) and B3 (character panel body) landed on branch
`1021-panes` and were returned twice by adversarial review: round-1 with four player-visible findings
(P-1 through P-4) and five documentation findings (D-1 through D-5); round-2 with one further
player-visible finding on the seam blit mode (P-1, keyed vs opaque, `TOWN-234`), the roster-cell
width regression P-1 had itself introduced (P-5), and four owner-reported items (brown doll
rectangle, oversized tip panel, mage skill-icon clip, near-black control boxes and the DOLL/STATS
label budget). This document reflects both fix passes, all on the same branch. Research pin unchanged
at `d1d350d` throughout both rounds (frozen per brief; `git submodule status research` shows no
leading `+`/`-`).

This story is cut too large: five behaviours after round-2's own shop-tip item, across four screens
(tavern, school, shop, generator) and nine call sites, against a three-pass ceiling contract read at
open as three behaviours in one domain (client presentation). Round-2 is this ceiling's second of
three; see Cut list below for what round-2 removed to land inside it, and Review status for the
ceiling accounting.

This story touches no `pkg/sim` file and no script-node dispatch, so neither milestone instrument
moves. Round-1 D-1: this section previously called a two-mission smoke check "the milestone
census"; the two are now reported separately (see Game-instrument check below) — `pipeline/
check-milestone.sh` is the census, walking all 28 shipped campaign maps per root.

This story's own pointable result is visual, not a census movement: the near-black character panel,
the shared worst-of-both-roots tip heights, the bare tavern left-column seam, the tavern roster cells
overwriting shipped furniture, and the generator's own doll box never reaching the fix at all — every
one of these, reported by the owner or found by round-1 review, is gone from every rendered screen
(described under Integration witness, below).

## Twelve-aspect matrix

| Aspect | Status | Note |
|---|---|---|
| Data | PASS | Round-1 review replaced the single `fullstatsl.bmp`/`fullstatsr.bmp` pair with two mode-switched shipped pairs, `humanbackr.bmp`/`humanbackl.bmp` and `textbackr.bmp`/`textbackl.bmp` (160x242/16x242 each), plus new reads of `inn/luover.bmp`/`inn/ldover.bmp` (tavern left-column seams) and `chrgen/rollstatsr.bmp` (generator plate seam, replacing the retired `NavSeamLower` field). All through the same `readChargenBMP`/`chargenSize` loader as every other chargen asset; no new file format. |
| Runtime state | PASS | `FrontEnd.characterPaneCache` is now `TownCharacterPaneArt` (Figure/Stats, was a single `ui.TownPane`), same `characterPaneLoaded bool` gate, same `tipArtCache` precedent. `TestResetSessionForNewGameDropsExactlyTheSessionPopulation`'s fixture updated to the new type. |
| Simulation | N/A | No `pkg/sim` file touched. Milestone census unchanged (see Result). |
| Player input | PASS | B2's shrunk rect is the SAME `TipPanelView.Rect` value both `ComposeTipPanel` draws and `TipPanelControlAt` hit-tests (one field, set once by `tipView`); no separate hit rectangle was introduced, so the swallow/draw mismatch `DIV-162` records from story 1018's round 3 is not reintroduced. Round-1 P-4's roster-cell relocation is also a hit-test change (`townSurfaceCellRect`'s own return value), covered by the same production function both the draw and `TownSurfaceControlAt` call. Round-2 P-5: the same function's cell width, shrunk to 70px by round-1's own fix, truncated roster labels; `townSurfaceCellRect` now derives both the bottom anchor and the 144px width from the room's own roster count `total`, so draw and hit-test move together again. |
| AI | N/A | Untouched. |
| UI/HUD | PASS | The core of this story. See Behaviours B1-B3, P-4 in `spec.md` and Integration witness below. |
| Triggers/scripts | N/A | Untouched; milestone census confirms. |
| Inventory/equipment | N/A | Shop's own inventory art is explicitly out of scope (contract.md). |
| Persistence/save-load | N/A | No save-file field touched. `characterPaneCache` is process-lifetime, not persisted (confirmed by the session-reset test's own `keepUnchanged` classification). |
| Campaign/session | PASS | `resetSessionForNewGame` unaffected (test above); scenario `0163-mission-to-town.json` (real campaign, notice-driven town entry, `TAVERN` activation, NPC dialogue) passes on both roots through the new code paths. |
| Shipped content | PASS | Round-1 review corrected the family: `humanbackr.bmp`/`textbackr.bmp` now draw on FOUR screens (tavern, school, shop, and the generator's own doll box — the generator was previously unreached, P-2), not three. `inn/luover.bmp`/`inn/ldover.bmp` newly drawn on the tavern's own left column. `chrgen/rollstatsr.bmp` newly drawn beside the generator's own stat plate. The shop's own upper-slot menu plaque now composes through the shared `drawTownPane` primitive (D-2) instead of a direct `blit` call. Round-2 P-1: all 11 `drawTownPane`/`PlateSeam`/`NavSeam` sites now blit their seam strip `draw.Over` a pre-keyed source instead of `draw.Src`, per `TOWN-234`'s decoded body-plus-keyed-border pattern; canvas-fill revealed is 0px at every site, confirmed by re-rendering and reverting through the production composers. |
| Interactions with existing mechanics | PASS | `DrawTownCharacterRegion`'s nil-pane fallback (a caller predating this story) is exercised and mutation-tested (`pkg/ui/townpane_test.go`). The generator's doll (`preview.Doll`) now composites with `draw.Over` (`copyNativeOver`) instead of `draw.Src`, fixing an alpha punch-through that had erased the pane body underneath it (P-2); this is a change to an existing draw call, not a new one, and is covered by the extended `chargen_release_test.go` witness. The shop's `art.Menu != nil` guard (D-2) prevents a typed-nil interface panic that a synthetic partial-art fixture (`pkg/ui/shopscreen_test.go`) surfaced during this round's own fix. Round-2 owner items: the doll's authored brown border rectangle and the dead `drawChargenFrame` helper are removed (`chargen_page.go`); the mage page's skill icons are clipped to `chargenColumnSkillClip` so they stop reaching the nav-seam column (measured 416/3808 mismatched pixels at `x:[464,472) y:[150,202)` before the clip, 0 after); the shop's and the chargen's own tip panels shrink through the same `TipPanelShrinkRect` shape B2 already established for the room screens; the near-black control boxes draw hit-testable but without an opaque fill, on shipped art; `townCharacterMode`'s DOLL/STATS box widens from 48px to 74px against a measured 52px/57px label budget. |

No in-scope GAP.

## Cut list

Four items round-2 review named; three are cut, one was found cheap and implemented instead of cut.

- **Generator centre re-layout — cut.** `chargenColumnDestination` stays `(300,0)-(480,480)`, 180
  wide. `TOWN-234` decodes the generator's own `+0x7c` centre child at `(160,0)-(480,480)`, 320 wide,
  a background blit followed by the four border-column blits this round now draws keyed. Widening
  the centre column to match is a composer restructure (`composeChargenDetailedPage`'s own column,
  card and message-box geometry all key off `chargenColumnDestination`), which item 1 of this round's
  own brief named explicitly out of scope ("do NOT restructure composers this round"). Tracked as its
  own story: `implementation/docs/1022-chargen-composition/contract.md` names the same rect and the
  same 320-wide target as its own B1.
- **`chargenCardBox` overpaint — cut.** `chargenCardBox`, `(0,207)-(300,480)`, draws last and
  unconditionally over two neighbours it partially overlaps: the stat plate `Plate`,
  `(0,0)-(160,240)`, by its own bottom 33 rows (`y:[207,240)`, `x:[0,160)`, 160x33 = 5,280px), and
  this round's own new `PlateSeam` strip, `chargenPlateSeamRegion` `(160,0)-(176,238)`, by its own
  bottom 31 rows (`y:[207,238)`, `x:[160,176)`, 16x31 = 496px). Both figures read directly from the
  four rects in `pkg/ui/chargen_page.go`, not measured pixel-by-pixel. `pkg/ui/chargen_page.go:172`
  already documents the card's own coverage of the seam's bottom rows as intentional, on the same
  precedent as `chargenDetailMessageBox`'s own upper rows. Reshaping the card to stop short of either
  neighbour is a card-geometry change this round's own brief did not ask for and was not built.
- **`shopmenu.bmp` split — cut.** The shop's own upper-slot plaque stays one opaque `Body` blit
  (`DIV-183`, new this round). `DIV-166` already gives a positive reason: the shop's own paint
  routine carries no `+0x10`-style addend and its bitmap is 176 wide, an exact fit, unlike the
  school's 160-wide bitmap in a 176-wide widget. Splitting it into a `Body`+`Seam` pair to match the
  other three screens' `drawTownPane` shape would change no shipped pixel, so it was recorded
  (`DIV-183`) rather than built.
- **Mage generator page parameterization ("if cheap") — NOT cut, implemented.** Item 5 of this
  round's own brief asked for the mage page's own skill-icon clip conditionally on cost.
  `chargen_release_test.go`'s own test already needed both pre-choice classes to witness the clip
  (only the mage page reaches the seam column), so parameterizing `TestReleaseChargenDetailedSeamColumnsDrawShippedStrips`
  over `t.Run("fighter", ...)`/`t.Run("mage", ...)` cost one loop, not a new test file. Built, not cut.

## Integration witness

**State-level, real campaign:** `pipeline/check-scenarios.sh` on both `en` and `ru`: 13 of 13
scenarios pass, including `0163-mission-to-town.json` (loads an original save, drives the map to a
notice-triggered town entry, activates `TAVERN`, activates an NPC, opens dialogue) and
`1005-doll-and-shop.json` (real campaign into the shop). Both exercise `townCharacterView()` and
`DrawTownCharacterRegion` through the real production dispatch this story rewired.

**Pixel-level, real campaign:** `cmd/plaquescreens`'s shop path (`writeShop`) opens a town through
`FrontEnd.FinishMission` — the same production campaign-completion call the game itself makes, not
a hand-built view — and composes the shop screen through `f.TownScreen()`. Rendered on `en`, the
shop's own character panel shows the shipped body's green ornate frame around the standing figure
(previously a solid near-black box). Rendered on both `en` and `ru`, output is visually identical
across roots except for the loaded party's own name/level (both roots share the same shipped panel
art).

**Pixel-level, hand-built view (disclosed limitation):** `cmd/plaquescreens`'s school and tavern
paths build their `ui.TownSurfaceView` directly rather than through a live campaign session (no
party is loaded), so B3's `HasSubject`/figure/name path is not exercised there; this predates this
story and is unchanged by it. Both gained a `characterPanes(f)` helper this story (renamed at
round-1 review from the singular `characterPane`) so their own `FigurePane`/`StatsPane` are
populated the same way `townCharacterView()` populates them in production, and both render the
correct-family body/seam:

- Tavern (`en`): the character panel (bottom right) shows the ornate frame in its Figure-mode
  presentation (`humanbackr.bmp`); the left column shows `LeftStats` (ornate frame, dark interior,
  now with its own `luover.bmp` seam) above `LeftPicture` (an oval fur-textured portrait frame, now
  with its own `ldover.bmp` seam) below, matching the B1/B3 order fix and the P-1 seam addition. The
  mercenary roster cells (P-4) now sit at `x:[176,480)`, clear of both the left column and its own
  seam. Confirmed on `ru` as well, pixel-identical panel art (both roots share the same shipped
  bitmaps).
- School (`en` and `ru`): the character panel (bottom right, below the Train/Exit button well)
  shows the same corrected frame.

**Pixel-level, generator, real production path (round-1 review witness):**
`pkg/game/chargen_release_test.go`'s `TestReleaseChargenDetailedSeamColumnsDrawShippedStrips`
reaches `composeChargenDetailedPage` through the same construction the generator's own live window
uses (`releaseFront(t)`, not a hand-built view), and checks the composed page's own pixels against
four shipped sources pixel for pixel: `NavArt`'s seam, the stat plate's new seam, the doll box's own
seam, and the doll box's own body (both excluding their own documented legitimate overlaps). All
four pass on `en`; run and confirmed clean on `ru` as well through `check-release-tests.sh`'s own
per-root invocation (Gate numbers, below). This is the P-2 fix's own witness: before it,
`preview.Doll`'s `draw.Src` composite erased the doll body pane wherever the doll sprite's own alpha
fell under 255, and no test read the composed page's pixels at that rectangle to catch it.

Screens rendered this session and inspected directly (not committed; ephemeral scratch output under
the lane's own temp directory, on `cmd/plaquescreens`' own precedent that PNGs are not repository
artifacts): `school.png`, `tavern.png`, `chargen.png`, `shop.png` on both `en` and `ru`, plus
targeted crops of the tavern's left column and the shop's/school's/tavern's character panel.

## Game-instrument check

Round-1 review D-1: this section previously used "the milestone census" for the two-mission smoke
check below; the two are distinct instruments and are reported separately here.

SMOKE CHECK (`go build -o mr ./cmd/missionrun`, then `AGAINROM_ASSETS=<root> ./mr -mission <10|20>
-trace -ticks 1 | grep -c UNSUPPORTED`, this worktree's own build):

| root | mission | this story | master (`pipeline/milestone-baseline.txt`) |
|---|---|---|---|
| en | 10 | 0 | 0 (no `cannot run` line) |
| en | 20 | 0 | 0 (no `cannot run` line) |
| ru | 10 | 0 | 0 (no `cannot run` line) |
| ru | 20 | 0 | 0 (no `cannot run` line) |

THE CENSUS (`pipeline/check-milestone.sh`, all 28 shipped campaign maps per root, reads a prebuilt
`implementation/builds/current/missionrun.exe` rather than this worktree): `pipeline/
milestone-baseline.txt` sums to 59 `UNSUPPORTED` nodes on `en` and 59 on `ru`, across all `cannot
run` lines in the file. `builds/current/` was not rebuilt from this branch, so this figure is
master's own, unchanged by this story, as expected — no `pkg/sim` file and no script-node dispatch
was touched.

## Divergence reconciliation

Allocated `DIV-175` through `DIV-182` at round-1, `DIV-183` through `DIV-188` at round-2 (via
`pipeline/next-div-id.sh`). **Round-1 adversarial review D-3, D-4, D-5 corrected this whole section**;
it is rewritten rather than amended in place, since the original entries describe a body/seam pairing
(`fullstatsl.bmp`/`fullstatsr.bmp`) this branch no longer ships anywhere.

**Round-2: two spent, four returned.**

- **`DIV-177`** (round-1 row, retyped `UNKNOWN`→`DEVIATION` at round-2 review): the tavern's own
  left-column seam strips are the same shape `TOWN-234` decodes for the generator's own left edge,
  `(160,0)-(176,238)`/`(160,238)-(176,480)` via `vt+0x38` (keyed) — corroborating, not silent, so the
  row no longer belongs with the rows research says nothing about. Not a new spend; retyped in place.
- **`DIV-183`** (new, round-2): the shop's upper region has no separate seam blit — the
  `shopmenu.bmp` split cut above.
- **`DIV-184`** (new, round-2): whether `TOWN-234`'s four border columns belong to the generator's
  centre child specifically or generalize to a shared side-panel mechanism — the open research
  question item 1 of this round's own brief asked to be recorded, not answered.
- **`DIV-185`–`DIV-188`** returned unused, on this ledger's own established precedent (`DIV-151`/
  `DIV-152`, `DIV-163`/`DIV-164`, `DIV-171`–`DIV-174`, `DIV-180`–`DIV-182`): round-2 review's own
  findings did not surface a further distinct technical fact beyond `DIV-177`'s retype and the two
  new rows above. Not reissued.

**Round-1 spend, unchanged this round:**

Five spent (all Type `UNKNOWN`, not `DEVIATION` — round-1 D-3: authored where research is silent is
not a deliberate departure from a stated fact), three returned unused:

- **`DIV-175`** (retyped `DEVIATION`→`UNKNOWN` at round-1 review, D-3; content rewritten, D-4): the
  shared character panel now carries TWO mode-switched bodies, `humanbackr.bmp`/`humanbackl.bmp`
  (Figure mode) and `textbackr.bmp`/`textbackl.bmp` (Stats mode), on FOUR screens (tavern, school,
  shop, generator — the generator reached only after P-2's `draw.Over` fix), not the single
  `fullstatsl.bmp` on three screens the story's original landing recorded. Chosen by a direct
  join-score render (`cmd/plaqueseams`, extended this round), not a decoded destination:
  `fullstatsr.bmp`/`fullstatsl.bmp`, the original pairing, scores 42.49 as the right-column join this
  rect needs — a visibly broken seam — where `humanbackl.bmp`/`humanbackr.bmp` scores 9.52 and
  `textbackl.bmp`/`textbackr.bmp` scores 6.30. Verified directly against the pin this round (`go run
  ./tools/claim SHOP-FIGURE-041`, `TOWN-284`): both still match the row's own paraphrase — neither
  claim names an archive entry for any of the four new bitmaps either.
- **`DIV-176`** (retyped `DEVIATION`→`UNKNOWN` at round-1 review, D-3; content otherwise unchanged):
  the tavern's left-column bitmap order, reversed from this build's prior order, authored from three
  converging signals against `TOWN-282`'s own unresolved blit coordinates. Verified directly against
  the pin this round (`go run ./tools/claim TOWN-282`): matches the row's own paraphrase exactly — no
  disagreement found.
- **`DIV-177`** (new, round-1 P-1; **retyped `UNKNOWN`→`DEVIATION` at round-2 review**, see above):
  the tavern's own left-column seam strips, `inn/luover.bmp` and `inn/ldover.bmp` at `x:[160,176)`,
  absent from the story's original landing. Chosen by the same join-score method as `DIV-176`:
  `leftstats.bmp`/`luover.bmp` 10.31, `leftpicture.bmp`/`ldover.bmp` 5.76, both low and both matching
  the existing height pairing. The round-2 retype does not change this choice; it changes only
  whether research is silent on the strips' own SHAPE (it is not, per `TOWN-234`).
- **`DIV-178`** (new, round-1 P-4): the tavern roster cells and room title, moved from `x<176` (where
  they overwrote `LeftPicture`/`LeftStats` and the new seam column) to `x:[176,480)`. Authored
  geometry; no claim gives the roster's own coordinates in the original.
- **`DIV-179`** (new, round-1 P-3): the tip panel's own text font, corrected from `font1` to `font2`
  for every town/school/tavern/shop tip, on the generator's own tip panel's established precedent
  (`c.setup.PreCreate.Art.Font`). No claim names either font for a tip popup.
- **`DIV-168`** (amended again, not a new spend; round-1 D-5): the row's own 2026-08-21 amendment,
  recording the seam column's widening from chargen-only to all four screens, now also reasons
  against both decoded geometries that name a related but different screen: `TOWN-234` decodes the
  GENERATOR's own `+0x7c` child alone and says nothing about the other three screens; `SHOP-FIGURE-041`
  is about the SHOP's own id-7 object and does not address a seam column at all. Neither is a claim
  that the seam is generator-only or shop-exempt; each is silent on the other three screens, not a
  negative on them.
- **`DIV-180`–`DIV-182`** returned unused, on this ledger's own established precedent (`DIV-151`/
  `DIV-152`, `DIV-163`/`DIV-164`, `DIV-171`–`DIV-174`): round-1 review's own findings did not surface
  a further distinct technical fact beyond the five rows and the one amendment above.

The preamble to the "Authored where research is silent" section (`docs/DIVERGENCES.md`) is corrected
at round-1 review to state the section's own type distribution (69 `UNKNOWN`, 6 `DEVIATION`, 7
`FIDELITY-DEBT`, counted directly from the section's own `Type` column this round) rather than
asserting every row is `UNKNOWN`, which `DIV-149`/`DIV-150`/`DIV-160`/`DIV-169`/`DIV-170` (all
`DEVIATION`) and seven `FIDELITY-DEBT` rows already contradicted before this story. The other
thirteen non-`UNKNOWN` rows in that section were not re-audited this round; only `DIV-175`/`DIV-176`,
the two round-1 review named specifically, were checked and retyped.

No claim cited in `docs/1021-panes/contract.md` was found to disagree with the contract's own
paraphrase when read directly against the pin this round (`TOWN-282`, `SHOP-FIGURE-041`, `TOWN-284`
read in full; `SHOP-TIP-045` and `TOWN-187` not re-read, since B2 leaves both rects unchanged and the
contract does not paraphrase them beyond the two numbers already carried in `DIV-162`).

## Open items (not in-scope GAPs)

- **Tavern cell overlap, RESOLVED at round-1 review (P-4).** The prior open item here described the
  mercenary cells at `x:[10,234)` overlapping the left column's own art. `townSurfaceCellRect`'s
  tavern case now returns `x:[176,480)`, clear of both the left column and its own new seam
  (`DIV-178`); no overlap remains in the rendered `tavern.png` on either root.
- **The shared character panel's own two body bitmaps stay authored (`DIV-175`).** A claim naming
  the archive entry behind `SHOP-FIGURE-041`'s two id-7 fields, or any claim resolving
  `TownCharacterRegion`'s own body destination and mode assignment, corrects this row.
- **The tavern's own bitmap order (`DIV-176`) stays authored.** A claim resolving
  `R0892`'s own two blit destinations to actual coordinates corrects it.
- **The tavern's own left-column seam strips stay authored (`DIV-177`)**, the generator's own doll
  seam family and tip font (`DIV-179`) likewise: each is a rendering choice with no decoded
  destination, per its own row.
- **The generator's own left-edge column, `x:[160,176)`, resolved for its upper half at round-2
  review, open for its lower half.** `TOWN-234` decodes two border blits on this edge:
  `(160,0,16,238)` and `(160,238,16,242)`. The upper one now draws (`PlateSeam`/
  `chargenPlateSeamRegion`, `DIV-177` retyped `DEVIATION`). The lower one, `y:[238,480)`, is still not
  drawn: the character card (`chargenCardBox`, `x:[0,300)`) covers `y:[207,480)` there, including
  this rect, unconditionally (see Cut list, `chargenCardBox` overpaint). Whether the card's own
  coverage of an otherwise-undrawn decoded seam is itself a divergence is left open; no ledger row
  was opened for it this round because `TOWN-234` names the blit but not the archive entry, so there
  is nothing yet to compare the card's coverage against.
- **Whether `TOWN-234`'s pattern is centre-child-specific or a shared side-panel mechanism is open
  research (`DIV-184`, new this round).** Recorded, not answered; no code change follows from it and
  `drawTownPane`'s seven call sites are unchanged pending an answer (round-2 review, item 1).

- **The generator's skill-icon hit test still gates on the wider rectangle (pass-3 review, W-1).**
  This round narrowed the skill patches' DRAW clip from `chargenColumnDestination` (`x:[300,480)`)
  to `chargenColumnSkillClip` (`x:[300,464)`), so a mage patch stops before the nav seam.
  `detailedControlAt` (`pkg/ui/chargen_page.go:271`) still gates its mask lookup on
  `chargenColumnDestination`, unchanged by this story and outside its diff. In production the shipped
  mask decides, not that rectangle, so the gate matters only if the shipped `ColumnMask` carries a
  skill's colour code inside `x:[464,480)`. That is a fact about decoded install data and was not
  established: `detailedControlAt` is unexported and `pkg/ui` may not import `pkg/game`, where the
  install-gated harness lives, so no probe exists for it. Worst case is a click in an 8-pixel sliver
  selecting a skill that is already selectable from its own visible icon; no wrong game state follows
  either way. **The follow-up is the probe, not the row**: an install-gated assertion of the hit test
  at `(464,150)` under the mage class, before and after the clip.

## Mutation record

`pkg/ui/townpane_test.go`'s four new tests, each with a killed mutation in the lane's worktree
(edited and reverted, no file-level copy):

| Assertion | Mutation | Result |
|---|---|---|
| `TestDrawTownPaneBlitsBodyAndSeamOpaque`, body pixel | body blit line replaced with `_ = b` | killed (`townpane_test.go:53`) |
| `TestDrawTownPaneBlitsBodyAndSeamOpaque`, seam pixel | seam blit line replaced with `_ = b` | killed (`townpane_test.go:54`) |
| `TestDrawTownCharacterRegionFallsBackToAuthoredFillWithNilPane` | fallback condition flipped (`!= nil` to `== nil`) | killed (`townpane_test.go:113`) |

`TestDrawTownPaneNilSeamLeavesSeamRectUntouched` and
`TestDrawTownPaneNilBodyLeavesBodyRectForCallersFallback` were not separately mutated: the first
shares its body-blit assertion with the mutation above (same line), and its seam-untouched
assertion is the negative of the seam mutation already killed; the second asserts nothing changed
under a no-op input, which no single-line mutation of `drawTownPane` can falsely satisfy without
also failing the opaque-blit test.

`go test -trimpath -count=1 ./...` is otherwise unchanged in structure beyond round-1 review's own
additions: this pass added `pkg/game/townpanes_release_test.go`, extended
`chargen_release_test.go`, and updated three synthetic fixtures to the new archive entries
(`pkg/game/chargenassets_test.go`, `cmd/againrom/main_test.go`, `pkg/game/towntavernart_test.go`).

Round-1 review's own two mutation tests, against the new install-gated witnesses (not `go test`
mutations; run and reverted in the lane's worktree this round):

| Mutation | Result | What it shows |
|---|---|---|
| `characterPaneFigureBodyPath` set to the wrong file (`fullstatsl.bmp`) | `TestReleaseTownColumnPanesDrawShippedBodyAndSeam` did NOT fail | The pixel-identity witness reads its own "expected" value from the same `characterPanes()` call production uses, so it proves wiring correctness, not "is this the right shipped file" — that claim is carried by the join-score measurement (`cmd/plaqueseams`), a known and accepted limitation of this witness class, not a defect to fix |
| `ComposeTownSurface`'s tavern draw order reverted (Center-first back to Center-last) | `TestReleaseTownColumnPanesDrawShippedBodyAndSeam` FAILED on both tavern-left subtests | Confirms the witness catches the P-1 rows 2/3 regression class it was built for |

Round-2 review's own mutation tests, against the round-2 rebuilt witnesses (install-gated; file backed
up to the lane's scratchpad before each edit, restored and re-diffed against the intended fix after):

| Mutation | Result | What it shows |
|---|---|---|
| `drawTownPane`'s seam blit reverted from `draw.Over` to `draw.Src` (the round-1 P-1 regression itself) | `townpanes_release_test.go`'s and `townbuttons_release_test.go`'s keyed subtests FAILED at every seam rectangle | `assertSeamRegion`'s keyed-aware comparison catches the exact defect class round-2 review's own item 1 fixed; a plain `assertRegion` against the raw source (the round-1 witness shape) would not have |
| `chargenColumnSkillClip` removed from the mage skill-icon draw loop | `chargen_release_test.go`'s `mage`/`"nav seam"` subtest FAILED, 416 of 3808 pixels mismatched at `x:[464,472) y:[150,202)`; the `fighter` subtest continued to pass | Confirms the class parameterization (item 5) is load-bearing: the fighter page carries no icon in that column and cannot redden on this regression, so the mage subtest is the only witness for this owner item |
| `townSurfaceCellRect`'s roster cell width reverted to round-1's own 70px | `TestTownShellRosterCellTextBudgetFitsEveryProductionString` FAILED 11 of 12 subtests (`NPC 12`-class labels truncated) | Confirms the derived-width assertion (`townSurfaceCellRect(...).Dx()`, not a duplicated literal) catches the P-5 regression it exists to catch |

## Gate numbers

Round-1 figures above are as measured at the round-1 fix pass. **Round-2 re-run**, all from the
lane's worktree (`wt-1021-panes`), against the tree as it stands after round-2's own fixes, committed
at `8e8240a` (no further code change after that commit; this document's own `git diff` against it is
limited to this section's own final edit):

- `go build ./... && go vet ./...`: clean.
- `gofmt -l $(git ls-files --cached --others --exclude-standard '*.go')`: no output.
- `go test -trimpath -count=1 ./...`: every package `ok`, no `FAIL` (39 test-bearing packages).
- `bash scripts/check-no-game-assets.sh`: `clean (tree scan)`.
- `bash scripts/check-claim-citations.sh`: `ok (1178 distinct citations resolve against 1390 claims
  and 206 experiments under 783 prefixes)` — re-run after this round's own `docs/DIVERGENCES.md`
  edits (`DIV-177` retype, `DIV-183`/`DIV-184` new rows); unchanged count, no citation broken or
  added.
- `pipeline/check-release-tests.sh` (`AGAINROM_IMPL` exported to this worktree, confirmed by the
  script's own printed `checkout` line — a bare run silently targets the seat's default
  `implementation/` checkout instead, which this round's own first attempt did and had to be
  re-run): `en` — `ok (35 of 35 install-gated tests ran and passed, 0 skipped)`; `ru` — same,
  `ok (35 of 35 install-gated tests ran and passed, 0 skipped)`. Selected count unchanged from
  round-1 (34→35 at round-1, unmoved at round-2): round-2's own new subtests and the chargen class
  loop all sit inside the same `Test...` functions the script already counted.
- `pipeline/check-scenarios.sh`: `en` — `ok (13 of 13)`; `ru` — `ok (13 of 13)`. Unchanged.
- `cmd/tippanelcheck`: rebuilt this round (see Behaviour B2 in `spec.md`) to read each room's own
  production-resolved tip rect instead of the package-level constant for shop and chargen, which had
  been silently reporting the pre-shrink spare even after the shrink was wired into production. Real
  measured output, both roots, post-fix: shop spare 10px (`en`/`ru`); chargen fighter spare 10px
  (`en`/`ru`); chargen mage spare 10px (`en`/`ru`, read via a separate `SelectPreChoice(1)` call, not
  a shared rect — `Chargen.TipPanel()`'s own resolved rect depends on which class is selected).
- `cmd/plaqueseams`: unchanged this round; see round-1's own figures below.
- Milestone smoke check (`cmd/missionrun -mission 10|20 -trace -ticks 1 | grep -c UNSUPPORTED`, this
  worktree's own build): 0 on both missions, both roots, re-run this round and unchanged from
  round-1. Census (`pipeline/check-milestone.sh`, all 28 shipped maps, reads `builds/current/`, a
  seat-level instrument not rebuilt from this branch): unchanged from round-1's own reading, 59
  `UNSUPPORTED` nodes on `en`, 59 on `ru` — this story touches no `pkg/sim` file and no script-node
  dispatch in either round.
- `git log --format='%h %(trailers:key=Co-Authored-By)' c5c4f00..HEAD`: `8e8240a` (round-2 commit,
  round-1's own `c5c4f00` as the base), empty trailer field — no trailer.

## Review status

Returned twice: round-1 adversarial review (four P findings, five D findings, one required witness
extension) and round-2 adversarial review (P-1 corrected from opaque to keyed per `TOWN-234`, P-5 —
round-1's own roster-cell fix regressed cell text width — plus four owner-reported items and two
documentation corrections). This document reflects both fix passes, on the same branch. Ceiling for
this story: 3 passes (one domain — client presentation — five behaviours after round-2's own shop-tip
item, no hashed simulation state). 2 of 3 consumed. This story is cut too large for that ceiling (see
Cut list): three items — the generator's centre re-layout, the character card's own overpaint of the
plate and its seam, and the `shopmenu.bmp` split — are removed from round-2's own scope rather than
built, one of them (the centre re-layout) already opened as its own story,
`implementation/docs/1022-chargen-composition/`.
