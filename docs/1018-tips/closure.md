# Closure — 1018-tips

## Result

The town square, the shop, the school, the tavern and the character generator's pre-create page
each show that screen's own shipped tip text in a floating panel, drawn from three shipped art
nodes (`interface/t_back.bmp`, `interface/t_border.bmp`, `interface/radiob.256`) and composed over
the screen's own content. The panel carries a close control and a labelled toggle and takes input
before the screen beneath it. Closing the panel dismisses it for the rest of the session on that
screen only; other screens' panels are unaffected. The toggle suppresses every screen's panel and
is persisted to `options.txt` beside `saves/`, read back at startup, defaulting to on. The shop's
pre-existing bare-text tip (story `1011`) is replaced by the bordered widget. Full behavioural
detail is in `spec.md`, canonicalized to the code as built.

## Twelve-aspect matrix

| aspect | status | evidence |
|---|---|---|
| data | PASS | `LoadTipPanelArt` (`pkg/game/tipart.go`) reads three shipped art nodes; `loadTip`/`ChargenSetup` read six shipped text nodes. `TestLoadTipPanelArtReadsTheThreeShippedNodes`, `TestChooseLoadsEachRoomsOwnTip`, both install-gated release tests, both roots. |
| runtime state | PASS | `townScreen.tipClosed [roomTalk+1]bool`, `ui.Chargen.tipClosed bool`, `ui.Chargen.tipSuppressed bool` (round 3, P-2 — re-applied on `Back()`, the generator's own re-entry), `FrontEnd.tipsOff` cached flag. `TestCloseTipDismissesOnlyTheOpenRoom`, `TestChargenCloseTipDismissesForTheVisit`, `TestToggleTipsFlipsSuppressionNotTheOpenPanel`, `TestChargenTipSuppressionSurvivesForwardAndBack`. |
| simulation | N/A | Nothing in this story imports or calls `pkg/sim`. Confirmed by `internal/archtest`'s import-graph check passing unchanged and by the missionrun census below reading identically to master's baseline. |
| player input | PASS | `TipPanelControlAt` hit-tests close and toggle and consumes any point inside the panel's rect; wired at all four call sites in `pkg/ui/app.go` (five screens, because `inSurface` serves both the tavern and the school), all four returning early on bare `consumed` (round-3 revert of round 2's narrowed `kind != TipControlNone` guard — DIV-162), so a showing panel swallows every press and release inside its own rect, on every screen, and only Close/Toggle act on it. Three of the four also clear the covered screen's own press state before returning, so a gesture crossing the panel's boundary completes nothing (hotfix `66ee8bf`, discharging pass 3's P-1; the town-square site is release-only and holds no latch). `TestTipPanelControlAt`, `TestTipPanelControlAtNotShowingConsumesNothing`, `pkg/ui/tippanel_overlap_test.go`'s five per-screen enumeration tests, `pkg/ui/tippanel_crossing_test.go`'s three boundary-crossing tests. |
| AI | N/A | Not touched. |
| UI/HUD | PASS | `ui.ComposeTipPanel` (`pkg/ui/tippanel.go`); `pkg/ui/tippanel_test.go`; visual composition witnessed against real art in all six install-gated release tests. |
| triggers/scripts | N/A | Not touched; the missionrun census (below) is unchanged from the baseline, which is the mechanical proof nothing in a shipped mission script now behaves differently. |
| inventory/equipment | N/A | Not touched. |
| persistence/save-load | PASS | `OptionsStore` (`pkg/game/tipstore.go`), a store beside `saves/`, distinct from `pkg/sim`'s campaign/save format — the toggle is a UI preference, not sim state. `TestOptionsStoreSetTipsModeRoundTrips`, `TestOptionsStoreSetTipsModePreservesOtherKeys`, `TestFrontEndLoadOptionsAndSetTipsOff`. |
| campaign/session | N/A | The panel's dismissal is per-screen for the session (behaviour 3); the toggle is per-install, not per-campaign. Neither reads or writes campaign/save state. |
| shipped content | PASS | Six of the eleven `text/tips/` nodes are read and shown; art pairing and text-capacity are verified against both preserved installs (`en`, `ru`) through the production composer, not synthetic fixtures alone. |
| interactions with existing mechanics | PASS | The shop's story-`1011` bare-text tip and its clip against `shopMessageRect` are replaced and re-verified (`TestShopTipRectExtendsPastTheMessageStrip`, `TestTipPanelOccludesTheMessageStripWhileShowing`); `DIV-132`/`DIV-133` amended rather than left describing retired code. The three per-gesture latches the panel now sits over — the town surface's press control, the generator's press control and the shop's drag machine — are cleared by a swallowed event (hotfix `66ee8bf`); at the reviewed sha a boundary-crossing gesture completed a `ShopDrag` from a stale origin, which is the one existing mechanic this story reached and did not hold. |

No aspect is a GAP. Two are N/A on the basis of a negative search stated above (grep across
`pkg/game`, `pkg/ui`, `cmd/` for any import of `pkg/sim` from the files this story touches, and
the unchanged missionrun census), not merely omission.

## Integration witness

**What was run.** Every one of the five call sites is driven through its real production entry
point against a real preserved install, on both `en` and `ru`, in
`pkg/game/tippanel_release_test.go` (six tests, `AGAINROM_ASSETS`-gated, golden rule 2):
`TownScreen()`'s own construction (town square), `Choose(0)`/`Choose(1)`/`Choose(2)` (tavern,
shop, school), a real `ui.Chargen` at the pre-create stage (generator), and
`TestReleaseTipPanelTextsDrawWholeOnBothClasses`, which reads both `ChargenFighterTipPath` and
`ChargenMageTipPath` directly to prove the fit against `ChargenTipRect` for each shipped node on
its own, independently of which one a given setup picks (`spec.md`, behaviour 2).
`TestReleaseChargenTipPanelFollowsTheLiveClassChoiceAgainstRealArt` (round 2, DIV-162) drives
`f.ChargenSetup()` → `ui.NewChargen` → `SelectPreChoice(1)` and requires the composed frame's own
tip-body pixels to change from the fighter text to the mage text, on both roots. Each of the first
five requires the panel's own rect to
differ against the same frame composed with the panel forced off, and requires at least one pixel
of the tip body's, the Close label's and the toggle label's own drawn colour inside their own
rects — the specific check that catches story `1017`'s "art paints, text does not" failure mode.
All six pass on both roots (`go test -run "TestRelease.*TipPanel"`, `AGAINROM_ASSETS` set to
`gameversions/en` then `gameversions/ru`).

**What this is not.** No headless JSON scenario (`scenarios/*.json`) currently visits any of these
five screens or has a command verb for a tip panel; extending that harness's vocabulary was not
undertaken here and stays out of this story's contracted scope. The witness above drives the same
production call chain (`FrontEnd` → `TownScreen`/`ShopScreen`/`ChargenSetup` → the composer) a
headless scenario would reach, against the same shipped assets, but it is not itself a scenario
file and is not run by `pipeline/check-scenarios.sh`. `pipeline/check-scenarios.sh` was run for
this story anyway, on both roots against this worktree, and reports the pre-existing 12 of 12
unaffected (below) — the correct result for a story that adds no scenario and changes no screen a
scenario currently drives through to a state a scenario asserts on.

**Instrument gap disclosed by the same route as the truncation finding.** `contract.md`'s own text
carried a hint ("a bordered panel may not need [the clip]") that, quantified rather than taken as
settled, surfaced the close/toggle-chrome truncation defect recorded in `DIV-162` and `spec.md`
behaviour 1: the four non-shop rects' researched 200px height did not hold their own screen's real
shipped text once this story's own close/toggle row was added, on either root, before the rects
were raised. No scenario or unit test would have caught this without a check comparing wrapped
line count against the real rect, which is what `ui.TipPanelFits` now is and what
`TestReleaseTipPanelTextsDrawWholeOnBothClasses` now runs against real content on both roots.

## Game-instrument check

Per the outer harness's own instruction, `cmd/missionrun` was built and run with `-trace -ticks 1`
against missions 10 and 20 on both roots, counting `UNSUPPORTED` lines:

| root | mission | UNSUPPORTED lines |
|---|---|---|
| en | 10 | 0 |
| en | 20 | 0 |
| ru | 10 | 0 |
| ru | 20 | 0 |

`pipeline/milestone-baseline.txt` carries no `cannot run` line for `m10` or `m20` on either root,
which is the same reading of 0 unsupported nodes recorded there. The full campaign census
(`pipeline/check-milestone.sh`, all 28 missions plus the escort drive, both roots) was also run
against this worktree and printed `ok the script gap and the drive are where they were recorded,
both roots`; its output is byte-identical to `pipeline/milestone-baseline.txt` once the script's
own leading indentation is stripped (`diff`, exit 0). This story's contract states simulation is
N/A (no aspect reaches `pkg/sim`); the census confirms the number did not move, rather than only
asserting it should not have. Re-run after round 2's fix (DIV-162) with the same 0/0/0/0 reading for
missions 10 and 20 on both roots — round 2 touches `pkg/ui` and `pkg/game`'s own chargen/tip files
only, none of which import `pkg/sim`. Re-run again after round 3's fix, same 0/0/0/0 reading on both
roots — round 3 touches the same two packages' own files (`pkg/ui/app.go`, `tippanel.go`,
`tippanel_overlap_test.go`, `pkg/ui/chargen.go`, `pkg/ui/chargen_tip_test.go`,
`pkg/game/tippanel_release_test.go`), plus the new `cmd/tippanelcheck`, none of which imports
`pkg/sim` (confirmed by `internal/archtest`'s import-graph check passing unchanged and by
`cmd/tippanelcheck`'s own registration in `internal/archtest/dag.go` naming only `pkg/game` and
`pkg/ui`).

## Divergence reconciliation

Allocated range `DIV-160`..`DIV-164`. Three spent, two returned unused:

- **`DIV-160`** (new, DEVIATION, ACCEPTED) — the tip suppression toggle persists to a flat file
  beside `saves/`, not to `HKEY_LOCAL_MACHINE\SOFTWARE\1C\Allods\TipsMode` (`TOWN-186`, High). This
  build is a Go program with no registry to write to on every platform it must run on; the
  departure is deliberate, not a research gap.
- **`DIV-161`** (CLOSED by hotfix `8fa10715`) — the close control and toggle now draw the exact
  installed captions named by `TOWN-185` and read by `TOWN-206`: `text/main.txt` lines 127 and 128.
  `InstallWords` performs the existing per-install text decode, `ui.Words` carries the result, and
  every town and character-generator tip view consumes the same pair. The EN fallback is used only
  by a diagnostic view without installed words. Both lawful-root release witnesses assert the
  selected strings before checking their composed pixels.
- **`DIV-162`** (amended round 3, UNKNOWN, OPEN) — the panel's fill/frame/toggle art pairing and
  every rect's own size past its sourced values are this lane's own inference and authorship,
  verified by composing against real shipped art and text on both roots, never by a claim.
  **Amended by round-2 adversarial review** (pass 1 of 3, two P findings): round 1's rects kept the
  researched width and raised only the height, which buried a pre-existing control on four of the
  five screens (tavern cells, school skill icons, the town square's school door and Gilded Statue,
  two of the generator's four portraits) — both invisible under the panel and unclickable through
  `TipPanelControlAt`'s own whole-rect `consumed=true`. Round 2's fix widened the rects and narrowed
  the early-return guard at three of the four call sites to `kind != TipControlNone`, so a point
  outside Close/Toggle fell through to the room's own hit test. The same review found the generator's tip
  text resolved once, at `ChargenSetup()` construction, to the fighter branch only — the class row's
  `Start` is never seeded by any caller, so the mage branch was unreachable through any player
  action. `ChargenSetup` now resolves both branches and `ui.Chargen.TipPanel()` picks between them
  by the player's own live pre-create portrait selection (`spec.md`, Behaviour 2) — this fix stands
  unchanged through round 3.
  **Amended again by round-3 adversarial review** (pass 2 of 3, two P findings, five D, one W).
  Round 2's own fix was the wrong direction: the narrowed guard let a click reach whatever control
  the widened rect happened to sit over, driven through the real dispatch to a door choice that left
  the town square, a class commit the player did not make, and a skill spend with nothing on screen
  to show it — story `1005`'s own shipped defect shape, a hit test answering for a pixel the player
  cannot see. Round 3 restores a uniform swallow at all four call sites (bare `consumed`, matching
  `inShop`'s own behaviour since round 1) and separately restores every non-shop rect's researched
  width and top-left corner, growing only the height: `TownTipRect` `(328,0)-(640,373)`,
  `TavernTipRect` `(0,0)-(312,322)`, `SchoolTipRect` `(0,0)-(456,271)`, `ChargenTipRect`
  `(0,280)-(312,480)`. Since a swallowed click costs the player nothing, the deviation this row now
  carries is HEIGHT ALONE, on four rects of the five, measured by `cmd/tippanelcheck` (new,
  committed this round): minimum fitting height at the researched width, worst root, is 295/363
  (town), 295/312 (tavern), 261/210 (school), each raised by a 10-row margin. The generator's rect
  is `TOWN-187`'s own, used whole — its minimum fit is 160 against the researched 200, so it takes
  no growth and carries no deviation at all.
  **Amended a third time at the landing** (pass 3 of 3, D-1 and D-2). Round 3 shipped
  `ChargenTipRect` as `(0,310)-(312,480)`, 30 rows lower and 30 rows shorter than `TOWN-187`'s own
  rectangle, while this bullet, `spec.md`, `DIVERGENCES.md` and the rect's own comment all said the
  top-left was researched: it was the one rect of the five SMALLER than the original popup, a
  direction nothing disclosed. It is the researched rectangle at the landing. The occlusion is now
  stated per control rather than as a live-pixel total, since a pixel total is the format's
  vocabulary and not the reader's — town square statue/menu 65.2% and door 2 50.8%; tavern cell 0
  50.0%; school cell 0 100%, cell 1 100%, cell 2 91.3%, so two of the school's three skill cells are
  completely covered while the tip shows; shop shelf picks 71.9% each, merchant 71.6%, table cells
  16.2%..46.2%; generator choice 0 97.9% and choice 2 13.1%. `cmd/tippanelcheck` prints those
  figures on either root and they are identical on both. Full arithmetic and the enumeration witness
  closing the regression class are in `spec.md`'s Behaviour 1.
- **`DIV-163`, `DIV-164`** — returned unused. The story's own findings surfaced three distinct
  facts, not five; per the contract's own instruction these ids stay retired rather than being
  reused, and are spoken for by the narrative paragraph in `DIVERGENCES.md` rather than by a row.
- **`DIV-154`** (CLOSED, moved to `DIVERGENCES-CLOSED.md`) — the town square's missing tip widget
  is now built, its rect's top-left and width matching `TOWN-165` exactly (round 2 had widened this
  rect to `(240,0)-(640,290)`, making the closed row's own sentence false for one round; round 3
  restores the researched corner and width, and `DIVERGENCES-CLOSED.md`'s own row is corrected to
  round 3's rect literal, `(328,0,640,373)`, not round 1's `384`).
- **`DIV-132`** (amended, narrowed) — the shop tip's construction is now gated on this build's own
  `TipsOff` (`DIV-160`), explicitly not established as the same mechanism as the original's
  `[L03631]`; that half of the row stays exactly as open as before.
- **`DIV-133`** (amended, reversed reasoning) — the panel is now bordered and opaque and draws
  last, which removes the reason `DIV-133`'s original clip-before-`shopMessageRect` was needed;
  the rect now extends past the strip instead, recovering the sourced 136-row text capacity once
  the close/toggle chrome is accounted for as additional floor space. The row stays UNKNOWN/OPEN:
  the original's own wrap break rule and its own layering of a message affordance against the tip
  widget remain undecoded.

Every claim cited in `spec.md` and `contract.md` (`TOWN-165`, `TOWN-015`, `TOWN-021`, `TOWN-025`,
`TOWN-184`, `TOWN-185`, `TOWN-186`, `TOWN-187`, `TOWN-188`, `SHOP-TIP-045`, `TEXT-API-007`,
`DLG-WRAP-009`, `MISSION-DOC-021`) was read whole, via `go run ./tools/claim <ID>`, in the prior
session that produced `contract.md` and the bulk of `spec.md`'s content; this session re-verified
their citations resolve at the current pin (`scripts/check-claim-citations.sh`, below) rather than
re-reading each row again, since no claim's content changed and B1 gives no reason to re-derive an
already-read fact.

## Contract corrections found during implementation

- **The 8-bit-BMP-format assumption.** `spec.md` behaviour 1 records that `interface/t_border.bmp`
  ships 8-bit paletted, not the 24-bit BI_RGB the contract's art table implied by pairing it with
  `t_back.bmp`'s own reader; `readChargenBMP` failed against both real installs before this was
  found and `readChargenMask` plus a new `paletteRGBA` helper were used instead.
- **The 200px/136-row sourced-height-is-sufficient assumption**, the more consequential of the
  two: the contract's claims give each non-shop rect a 200px height and the shop rect 136 rows,
  and nothing in the contract states whether that height holds the real shipped text once this
  story's own close/toggle row is added. It does not, on either root, for any of the five screens,
  until the rects are raised past their sourced height — see `DIV-162` and the truncation finding
  above. This was not a gap in what the contract asked for; it is a gap in what the contract's own
  cited claims established about capacity, disclosed as its own divergence row rather than folded
  silently into the rect constants.

## Round-3 documentation findings (D-1 through D-5, W-1)

Five D findings and one W finding from pass 2 of adversarial review, each fixed in place per the
owner's rule that neither class returns the story on its own:

- **D-1.** `ChargenTipRect` does not reach zero overlap; three documents said it did. Corrected in
  `pkg/ui/tippanel.go`'s own `ChargenTipRect` comment, `spec.md` Behaviour 1 and this document's
  `DIV-162` bullet to the measured figure — 14132 live pre-create pixels, all of it choice 0's own
  fighter portrait, which is the researched rect's own layout rather than this story's regression.
- **D-2.** The shop's rect does overlap live controls (`ShelfPick 0`, `ShelfPick 1`, `Merchant`,
  `TableCell 1..4`, 47422 pixels); "no rect/control overlap at any width" was false. Corrected in
  `spec.md` Behaviour 1, which now gives the per-control row-loss figures and the total, and in this
  document's `DIV-162` bullet.
- **D-3.** `pkg/ui/tippanel.go`'s own comments quoted minimum-fit figures for round 2's own widened
  rects (a 290-tall, wider-than-researched `TownTipRect` among them); the figures did not reproduce
  against the code as it then stood. Round 3 restores the researched width, which is a different rect
  than round 2's, so round 2's own minimum-fit figures do not apply to it either way; the comments now
  quote `cmd/tippanelcheck`'s own output against the round-3 rects (295/363 town, 295/312 tavern,
  261/210 school, 160/160 generator), reproduced by running the committed instrument rather than
  copied from a scratch program.
- **D-4.** `pkg/ui/tippanel_overlap_test.go` stated a false fact about which mask pixels
  `TownTipRect` covers (school-door pixels, when it in fact covers zero of them, only the gates, the
  shop door and 3 statue pixels). The file is fully rewritten this round (Behaviour 1 above); the
  false sentence is not carried into the replacement.
- **D-5.** `closure.md` and `docs/DIVERGENCES-CLOSED.md` (`DIV-154`'s own closed row) both quoted
  `TownTipRect` as `(328,0,640,384)`, round 1's own rect; round 2 had in fact built
  `(240,0)-(640,290)`, so the sentence was already false before this round, and `spec.md:71`'s claim
  that all four rects take their top-left from a researched rectangle was false for two of the four
  at round 2. Round 3 restores every rect's own researched top-left and width, and both documents are
  now corrected to the literal round-3 built the story with, `(328,0,640,373)`.
- **`docs/DIVERGENCES.md`'s own `DIV-162` row and narrative paragraph** were never amended in round 2
  despite two source files citing "DIV-162, amended 2026-08-19" for an amendment that had not been
  made. Both are amended this round, to the round-3 rects and the instrument's own numbers.
- **W-1.** `TestTavernTipRectStaysClearOfTheMercenaryCellGrid` witnessed only the cell grid, not the
  button wells or the character-preview controls at the tavern's own right edge. Production is
  correct — round 3's swallow makes the property moot for the tavern regardless, since every point in
  the rect is now consumed — and the new `TestTavernTipPanelSwallowsEveryCellItCovers` (and the
  enumeration's own sanity check against every button well) is the single witness for both the
  original property and the round-3 hit test; no second test was written.

## Pass-3 findings and the landing (D-1 through D-7)

Pass 3 of 3 found one P finding and seven D. The P finding is `P-1` below, discharged by hotfix
`66ee8bf` before this story's own merge; the seven D findings are fixed in place, at the landing, in
the same commits as the corrections they describe. Two further D findings were found at the landing
by the seat and are recorded with them: `D-8` and `D-9`.

- **D-1** — `ChargenTipRect` shipped `(0,310)-(312,480)` while its own comment, `spec.md` Behaviour
  1, this document's `DIV-162` bullet and `docs/DIVERGENCES.md` all said each rect was restored to
  its researched width and top-left. `TOWN-187` gives `left=0, top=0x118, right=0x138, bottom=0x1e0`
  — `(0,280)-(312,480)`. The rect was the one of the five *smaller* than the original popup, and it
  needed no growth to begin with: the minimum fit is 160 against the researched 200. **Fixed in the
  code, not disclosed**: the rect is `TOWN-187`'s own, used whole, so the story's stated rule now
  holds for all five rects. Coverage moves with it — choice 0 from 14132 to 17431 px (97.9%), and
  choice 2 from none to 2162 px (13.1%).
- **D-2** — `DIV-162` stated the occlusion as live-pixel totals, which is the format's vocabulary
  and not the reader's. `cmd/tippanelcheck` now prints coverage per control the room's own hit test
  names, as a percentage of that control's own hit-testable area, and `ui.PreCreateControlAt`
  returns the control's name beside the hit so the generator can be reported the same way. The
  figures are identical on both roots and reproduce the reviewer's own independent hand measurement
  exactly. They are measured from a fresh front end with no campaign loaded, so a control the room
  disables answers no hit: they are a lower bound on what a mid-campaign player sees covered, which
  the row and `spec.md` now say.
- **D-3** — `TOWN-185` carries three child-control rects for the original popup at High confidence
  and this build does not use them; nothing recorded that researched inner geometry existed and was
  set aside. Recorded in `spec.md` Behaviour 1 and in `DIV-162`: the outer rects are researched, the
  inner geometry is authored, because the chrome it positions is this story's own (`DIV-161`).
- **D-4** — the toggle is one-way from inside the game. The gem is reachable only on a showing
  panel, and no panel shows anywhere once tips are off, so the only reversible moment is the panel
  still on screen when the player turns them off. The player turns tips back on by editing or
  deleting `options.txt`. Production matches the owner's own request; the sentence was missing.
  Stated in `spec.md` Behaviour 4.
- **D-5** — `TipPanelFits` measures the glyph, not the shadow `ComposeTipPanel` draws at
  `(x+1,y+1)`, so the shop's 2-row EN margin is 1 row of glyph plus 1 of shadow. The shadow row
  lands inside `TipPanelTextRect`'s own 4-pixel gap above the button row and costs no shipped text a
  row on either root. Two related instrument facts recorded with it: `min-height` is searched with
  `ui.TipPanelFits` itself, so it is production's own fit query rather than a second instrument
  agreeing with it, and `TipPanelFits` returns true for a view that is not showing, so a fixture
  that fails to show would report a comfortable pass. Stated in the function's own doc and in
  `DIV-162`.
- **D-6** — `TipPanelToggleRect`'s doc said the label is left-aligned; `drawTownShellText` centres
  it. The nine-patch prose named `tipBorderCornerW`/`H` for constants called
  `tipPanelBorderCornerW`/`H`. Both corrected in place.
- **D-7** — `DIV-161` disclosed the two captions as authored but did not say that an RU player sees
  two English words on every tip panel. Stated in the row.
- **The font witness's own fallback**, raised with D-5 as a caution. `TestReleaseTipPanelTextsDrawWholeOnBothClasses`
  resolved `f3.ChargenAssets.Presentation.Font` when non-nil and fell back to `f3.Font` otherwise —
  the exact wrong font round 3 had just fixed, so the witness could silently degrade back into the
  instrument it was written to replace. The fallback is removed: an install that does not resolve
  the pre-create page's own font fails the test. Both roots resolve it, 33 of 33 unchanged.

**P-1 — the swallow leaves each screen's per-gesture press latch armed.** Three of the four call
sites return on bare `consumed` before their own screen's press and release bookkeeping, so a
gesture that starts on a live control and ends on the panel — or the reverse — is neither completed
nor cancelled, and the screen acts on a latch left over from the previous gesture. Measured through
the real dispatch: a shop drag released inside `ShopTipRect` leaves the drag machine armed and its
doll suppression pushed on the next idle frame, and a press inside the panel released on a live
control fires `ShopDrag`, `TownSurfaceClick` (Train/Hire) or a pre-create class commit. The town
square's site is release-only and holds no latch.

The story's own witness cannot express the defect: `tippanel_overlap_test.go`'s `clickAt` presses
and releases at the same point, and all five per-screen tests drive the dispatch through it. Under
the rule that a witness whose blindness produced the P finding is ordered with the fix, the fix
carries a two-point gesture helper.

**Pass 3 is this story's stated ceiling, so P-1 landed as a hotfix rather than as a fourth lane
cycle** (`docs/hotfix/LEDGER.md`), on the owner's own rule that reaching the ceiling is a scoping
diagnosis rather than a failure. The hotfix is `66ee8bf` with its ledger row in `8c0c017`, merged at
`a1f679a`. Each site clears what it owns on every swallowed press or release: `townSurfacePress`,
`chargenPress`, and the shop's drag machine through the existing `clearShopDrag`. Its witness is
`pkg/ui/tippanel_crossing_test.go`, whose `pressAt`/`releaseAt` split the shipped same-point
`clickAt`; each of the three added reset lines was reverted in isolation at this seat and reddened
its own test and no other. The hotfix row is marked folded, because `spec.md` Behaviour 1 absorbed
the rule in the same landing.

**D-8 — every document said five `TipPanelControlAt` call sites; `pkg/ui/app.go` has four.** The
count of screens was written as a count of sites, in `spec.md` Behaviour 1 (three places), in this
document's own player-input row and round-2/round-3 paragraphs, and in `pkg/ui/tippanel.go`'s own
comment, which further described round 2 as narrowing the guard at "four of app.go's five call sites"
while naming three branches. `inSurface` serves both the tavern and the school. Measured at every
round's own sha: four sites at `1ad6d6f`, at `0653f18` and at the merge. Corrected in place in all
three files.

**D-9 — `docs/hotfix/LEDGER.md`'s header asked for the row in the same commit as the fix, and no row
in it was ever written that way.** The first column carries the fix's own hash, which a commit cannot
contain before it exists; all seven earlier rows landed in the commit after their fix. The same
header named `scripts/check-hotfix-ledger.sh` as the live measure of the file, and named
`check-doc-budget.sh` and `check-sdd-audit.sh` in its Bounds section; all three were retired on
2026-08-15. Corrected in place: the rule is stated as the two-commit practice, and the falsifiable
`git log` command in the header is named as the whole remaining check. **The story was cut too large**: five behaviours across five
screens and two domains, with the panel taking input before every one of them. Each of the three
passes returned exactly one player-visible defect class, and each class was a consequence of the
same decision — that one widget sits over five screens whose input paths were written separately.
A contract for one screen, or for the widget alone with the five wirings following it, would have
been reviewable inside its ceiling.

## Surface not covered by any of the three passes

Named rather than silently closed, since pass 3 was the last inside the ceiling. None of these is a
known defect; each is a surface no pass reached, and each is carried into whatever work touches it
next.

- **`resetForNewGame`'s four call sites against a real save load.** The function's own behaviour is
  read and its doc is correct (`spec.md` Behaviour 3), and the three save-loading call sites are
  identified, but no test or scenario drives an original `.sav` or an `.ags` resume and then checks
  that the tip bits reset with it.
- **`options.txt`'s failure modes on disk.** The round trip, the default and unknown-key
  preservation are witnessed (`pkg/game/tipstore_test.go`). A missing file, a garbage file and a
  read-only directory are not.
- **The `t_back`/`t_border`/`radiob` pairing and the nine-patch corner fraction against the owner's
  photographs.** `DIV-162` carries this as authorship. It is a fidelity question no gate in this
  tree can answer and no test may read an install to answer.
- **Composed-frame diffs on the RU root** beyond what the 33 install-gated release tests assert.

## Mutation record

Every mutation was applied to the working file, run against the test(s) expected to catch it, and
restored to a byte-identical copy (`diff`, and for `tippanel.go` additionally a raw byte
comparison) before the next step:

| mutation | target test | first attempt |
|---|---|---|
| `chargenSize` on the border read forced wrong | `TestLoadTipPanelArtWrongSizedBorderFails` | reddened |
| `paletteRGBA` forced to return nil | `TestLoadTipPanelArtReadsTheThreeShippedNodes` | did NOT redden — `TipPanelArt`'s fields are the `image.Image` interface, so a nil `*image.RGBA` produces a non-nil interface value; a plain `== nil` check cannot see it. Test rewritten to type-assert to the concrete pointer before comparing to nil; re-run reddened. |
| `chargenTipPath` forced to always return the fighter path | `TestChargenTipPathSelectsByClassStart` | reddened |
| `TipPanel()`'s stage guard forced to `false` | `TestChargenTipPanelHidesPastPreCreate` | reddened |
| `TipPanelFits` forced to always `return true` | (none existed) | did NOT redden — the only caller was a release test whose fixed rects already satisfied real capacity, so `true` was indistinguishable from correct along that path. `TestTipPanelFitsTellsAFullRectFromATruncatingOne` added (a rect sized to hold N wrapped lines, then one pitch-line shorter); re-run reddened. |
| round 2 (DIV-162): town-square call site's `kind != TipControlNone` guard reverted to bare `consumed` | `TestTownSquareTipPanelDoesNotSwallowAClickOnTheDoorItVisuallyCovers` | reddened |
| round 2: `inSurface`/school call site's guard reverted | `TestSchoolTipPanelDoesNotSwallowAClickOnTheSkillIconItVisuallyCovers` | reddened |
| round 2: `stepPreCreate` call site's guard reverted | `TestPreCreateTipPanelDoesNotSwallowAClickOnAPortraitItVisuallyCovers` | reddened |
| round 2: `TipPanel()`'s class-selection branch removed (falls back to `TipText` unconditionally) | `TestPreCreateTipTextFollowsTheLiveClassChoice` (synthetic) | reddened |
| round 2: same mutation | `TestReleaseChargenTipPanelFollowsTheLiveClassChoiceAgainstRealArt` (real install, both roots) | reddened |
| round 3 (P-1, DIV-162): `app.go`'s `inSurface` call site (tavern/school) reverted to `consumed && kind != TipControlNone` | `TestTavernTipPanelSwallowsEveryCellItCovers` AND `TestSchoolTipPanelSwallowsEverySkillIconItCovers` | both reddened (shared call site); no other of the five tests reddened |
| round 3: town-square call site reverted the same way | `TestTownSquareTipPanelSwallowsEveryControlItCovers` | reddened alone |
| round 3: `inShop` call site reverted the same way, to confirm it is genuinely wired rather than untested | `TestShopTipPanelSwallowsEveryControlItCovers` | reddened alone |
| round 3: `stepPreCreate` call site reverted the same way | `TestPreCreateTipPanelSwallowsEveryPortraitItCovers` | reddened alone |
| round 3 (P-2): `Chargen.Back()`'s `c.tipSuppressed = !c.setup.TipsOn` line removed | `TestChargenTipSuppressionSurvivesForwardAndBack` | reddened ("TipPanel() after Forward and Back, tips off, is Showing() — suppression did not survive re-entry") |

Two of five round-1 mutations did not redden on the first attempt. Both gaps were in the synthetic
test suite, not in production code, and both are now closed by a strengthened or new test. All five
round-2 mutations reddened their target test on the first attempt. All five round-3 mutations
reddened their own target test(s) on the first attempt, and no mutation reddened a test outside its
own screen — the enumeration closes the regression class rather than four independent site fixes.

An earlier draft of the round-3 enumeration sampled a point inside `TavernTipRect` that happened to
land on `TipPanelToggleRect`, which round 2's own narrowed guard already let through correctly
(`kind != TipControlNone` is satisfied by a Toggle hit); that sample did not redden when the
`inSurface` mutation above was applied, and the gap was closed by restricting every enumeration
sample to `tipBackgroundZone(r)` — the panel's own rect minus its close/toggle row — before the
mutation sweep above was re-run and found to discriminate correctly.

## Gate numbers

### At the landing, on the merge commit

Measured at this seat on `2ed24d2`, master, with the story merged (`abbde62`), the research pin
bumped (`ab7112d`), the seven landing corrections applied and the tip-latch hotfix merged
(`a1f679a`). A gate result belongs to a commit, and this is the content neither the lane nor the
reviewer saw.

- `go build ./...`, `go vet ./...`: clean. `gofmt -l`: no output.
- `go test -count=1 -trimpath ./...`: 39 packages `ok`, no install present.
- `scripts/check-no-game-assets.sh`: `clean (tree scan)`.
- `scripts/check-claim-citations.sh`: `ok (1158 distinct citations resolve against 1369 claims and
  203 experiments under 782 prefixes)`. The merge introduces no new id: the same 1158 resolve at
  `68dfecf` and at `a1f679a`, checked by comparing the id sets at both shas.
- Research chain, `research/`: build, vet, gofmt clean; `go test` 7 packages `ok`;
  `check-claim-ids.sh` `ok (1369 ids)`; `check-retraction-status.sh` `ok (227 overturned ids)`.
- `pipeline/check-pin-forward.sh`: `ok (no unmerged lane branches)`, master pins `ded1827`,
  0 selected of 0 local and 0 on origin.
- `pipeline/check-preserved-installs.sh`: `ok — 162 file(s), both roots as recorded`.
- `pipeline/check-scenarios.sh`: `ok (12 of 12)` on `en` and on `ru`, 12 selected each.
- `pipeline/check-release-tests.sh`: `ok (33 of 33 install-gated tests ran and passed, 0 skipped)`
  on `en` and on `ru`, 33 selected each.
- `pipeline/check-milestone.sh`: exit 0, at `pipeline/milestone-baseline.txt` — 59 unrunnable script
  nodes per root across the 28 campaign maps, unchanged.
- Deletion set `git diff --diff-filter=D --name-only 68dfecf a1f679a`: empty.
- The hotfix's three mutations re-run at this seat: reverting each of the three reset lines in
  isolation reddened its own test in `pkg/ui/tippanel_crossing_test.go` and no other, and the file
  hash was verified identical after each restore.

### Round 3, in the lane's worktree

Re-run after round 3's fix (P-1, P-2, DIV-162 amended again) and the five D findings:

- `go build ./...`: clean.
- `go vet ./...`: clean.
- `gofmt -l $(git ls-files --cached --others --exclude-standard '*.go')`: no output.
- `go test -count=1 -trimpath ./...`: 39 packages `ok`, no game install present (`cmd/tippanelcheck`
  and `internal/archtest` both new/exercised this round).
- `scripts/check-no-game-assets.sh`: `clean (tree scan)`.
- `scripts/check-claim-citations.sh`: `ok (1150 distinct citations resolve against 1344 claims and
  198 experiments under 782 prefixes)` — unchanged; round 3 cites no claim not already cited.
- `pipeline/check-preserved-installs.sh`: `ok — 162 file(s), both roots as recorded`.
- `pipeline/check-scenarios.sh`, this worktree (`AGAINROM_IMPL`), `en`: `ok (12 of 12)`.
- `pipeline/check-scenarios.sh`, this worktree, `ru`: `ok (12 of 12)`.
- `pipeline/check-release-tests.sh`, this worktree, `en`: `ok (33 of 33 install-gated tests ran and
  passed, 0 skipped)` — unchanged from round 2 (`TestReleaseTipPanelTextsDrawWholeOnBothClasses` now
  reads the pre-create page's own font, `f3.ChargenAssets.Presentation.Font`, rather than the shared
  `f3.Font`; this is a fix to a pre-existing test defect, not a new test).
- `pipeline/check-release-tests.sh`, this worktree, `ru`: `ok (33 of 33 install-gated tests ran and
  passed, 0 skipped)`.
- `cmd/missionrun -mission {10,20} -trace -ticks 1`, `UNSUPPORTED` line count: 0 on both missions,
  both roots — unchanged from `pipeline/milestone-baseline.txt` and from round 1's and round 2's own
  readings; nothing in round 3 touches `pkg/sim`.
- Research pin: `eb92f537ac8ebee0d0176bc0c9b64ef0317af69f`, identical to `origin/master`'s own pin
  (`git ls-tree origin/master research`); no bump needed or made.

**A pre-existing test defect was found and fixed this round, outside the brief's own findings.**
`pkg/game/tippanel_release_test.go`'s `TestReleaseTipPanelTextsDrawWholeOnBothClasses` built its
`ui.TipPanelView` with `Font: f3.Font` — `FrontEnd.Font`, the shared town-room font ("font1",
`pkg/game/font.go`'s `DefaultFont`, loaded `pkg/game/frontend.go:595`). Production
`ui.Chargen.TipPanel()` resolves its own font from `c.setup.PreCreate.Art.Font`, which
`ChargenSetup()` sets from `FrontEnd.ChargenAssets.Presentation.Font` — a distinct resource,
"font2", loaded separately (`pkg/game/chargenassets.go:271`). The test's own wrong font was masked
by round 2's own taller, wider generator rect (font1's wider glyphs still fit inside it); round 3's
correctly-sized `ChargenTipRect` (170 rows, sized against font2, the real font) is genuinely too
short for font1's own wrapped output, and the test read `"chargen fighter/mage tip does not fit
ChargenTipRect (0,310)-(312,480)"` on the EN root before the fix. `cmd/tippanelcheck` was never
affected, since it drives the real `ui.Chargen.TipPanel()` path and so already resolves font2.
Fixed by reading `f3.ChargenAssets.Presentation.Font` when non-nil; re-run passes on both roots.

## Review status

Adversarial review pass 1 of 3 (ceiling for an ordinary story) returned this story to its lane with
two P findings and one D finding. Both P findings were addressed in round 2 (the occlusion class,
closed by widening the rects and narrowing the hit-test guard) and the D finding
(`spec.md`'s `resetForNewGame` wording) was fixed in place, `spec.md` Behaviour 3.

Adversarial review pass 2 of 3 returned this story to its lane again, with two P findings, five D
and one W. Round 2's own fix was the wrong direction — widening the rects and narrowing the guard let
a click reach a control the panel visually covered, story `1005`'s own shipped defect shape restated.
Round 3 reverts to a uniform swallow at all four call sites and restores every non-shop rect's own
researched width and top-left corner, growing only the height (P-1, `spec.md` Behaviour 1, closed by
a five-screen enumeration rather than four per-site tests); the generator's suppression toggle now
survives `Forward`/`Back` (P-2, `spec.md` Behaviour 4). All five D findings and the one W finding are
addressed above (Round-3 documentation findings).

Adversarial review pass 3 of 3 found one P finding and seven D. The P finding is `P-1` above, which
the ceiling rule carries as a hotfix rather than a fourth lane cycle; the seven D findings are fixed
in place at the landing, and one of them (D-1) is fixed in the code rather than disclosed. The pass
confirmed round 3's uniform swallow by reading all four call sites rather than the lane's account,
the committed instrument reproducing on both roots, the five per-screen enumeration tests driving the
real dispatch, the `tipSuppressed` fix by driving the real transition, and `DIV-162`'s amendment in
`DIVERGENCES.md`. It falsified one premise: "every rect returns to its researched width and top-left"
was false for `ChargenTipRect` on both halves. Every gate it ran matched the numbers reported here.

This document and `spec.md` are canonicalized to the as-built state at the landing.
