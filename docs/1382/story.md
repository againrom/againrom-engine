# ROM1 tips as the original

## Intent

The tip popups and the guided highlight cycles of the first game run the
original's state machine. The character generator's two pages show the step
texts the original shows, at its rectangles, advance them only on the clicks
the original reads, and run its highlight cycles. A mission dialogue tagged
`tips=N` raises its tip in the mission popup. One tip popup builder
(`ui.ComposeTipPanel`, `ui.TipPanelView`) serves every family, and one cycle
machine (`ui.guidedCycle`) serves both generator pages, with targets and art
as data.

Base: `6f05571d` (game 0.105.0). Knowledge pin k216. Reconciled main:
`ab0b7bdc` (release 0.106.0, after the rooms on the town composer and the one
game profile). The mission tip is ROM1 edition data (`Edition.MissionTips`).

## Authority

| Part | Authority |
|---|---|
| `TipsMode` word: default, writers, not in SAV; what it gates | MENU-135, MENU-136 |
| popup rectangles, controls, Close routing | MENU-137 |
| pre-create step field and texts chrsel1..3 | TOWN-518 |
| guided cycle timing, targets, order | TOWN-519 |
| pre-create mask regions and draw rectangles | TOWN-520 |
| pre-create state bits and art | TOWN-521 |
| detailed page tip and skill cycle | TOWN-522 |
| page cursors | TOWN-523 |
| statistic `+`/`-` art and disable rules | MENU-138 |
| Accept, Reset, Back art and states | MENU-139 |
| mission tip post and campaign arm | TRIG-TIPS-087, DIALOGUE-050 |
| shipped `tips=` census, fire-once latch | TRIG-TIPS-088 |
| room popups at every enter; shop2 | TOWN-516, TOWN-517 |
| owner observations of the original | review notes for this story |

## Gap census

| Claim | Engine before | Change |
|---|---|---|
| TOWN-518: pre-create popup shows chrsel1, chrsel2 after a portrait click at step 0, chrsel3 after a level click at step 1; step 0 at every enter | showed chrgen1f/m at the detailed rectangle; no step | step field on `ui.Chargen`; chrsel1..3 read; only the named pointer clicks advance it |
| MENU-137: pre-create popup at (232,0)-(640,136) | (160,280)-(472,480), shrunk to text | `PreCreateTipRect` (`DIV-2714` for the Medium origin) |
| TOWN-522: detailed popup shows chrgen1m/f by class, chrgen2 after the first skill click | chrgen2 at once on the detailed page | step field on the detailed page; class text at enter |
| TOWN-519: cycle 500 ms after hover, step at the first paint >300 ms after the last; order portraits, levels, amulet+OK; resumes after the hovered target; timers not reset at enter | none | `ui.guidedCycle`, painted from the App step with the frame clock |
| TOWN-522: skill cycle while the popup shows at step 0; reads no `TipsMode` | none | the same machine over the five skills, drawing `shine_on` on the chosen skill else `shine_off` |
| TOWN-521: portrait and level art by state bits (chosen, hovered) | every portrait drew on-art; keyed pixels | on, l, lon or nothing by the two bits |
| TOWN-520: only `Mask.bmp` decides a hit | keyed nonblack pixels overrode the mask | mask regions 20..180 only |
| TOWN-523: `select` cursor every pre-create paint; `default` every detailed paint | `default` on both | set on every paint (`DIV-2717` for the dice setter) |
| MENU-138: `+`/`-` draw nloff, loff, lon, disable; `nlon` never | `nlon` after a capture left the button | four states; lon while held over the button |
| MENU-138, MENU-066: `+` (cost) at x 107..127, `-` (refund) at x 132..152 | `-` left, `+` right | `+` left, `-` right |
| MENU-139: Accept/Reset/Back are `Inn\button{1,2,3}{on,off}.bmp` over `Inn\ButtonsArea.bmp`, font4 labels, on only while pressed and hovered | measured wells on `chrgen\buttonsarea.bmp`, font1 labels | the Inn art, rectangles and states (`DIV-156` closed; `DIV-169` keeps the ink) |
| TRIG-TIPS-087: a dialogue closing on its last page after a part set `tips=N` shows `m<mission>\tips<NN>` in popup `0x10` at (10,20)-(370,188), replacing an open one, while `TipsMode` is set | tags ignored in missions | mission popup on the viewer; raised by the dialogue close |
| TRIG-TIPS-088: once per mission run through fire-once triggers; LOAD restores the latch | none | no own latch; the trigger latch already rides the SAV |
| MENU-135: only the popup checkbox and Game Options OK write `TipsMode`; not in SAV | matched | unchanged; the mission popup's checkbox writes it on its press |
| MENU-136: `TipsMode` gates construction and steps, not shop2 or the skill cycle; clearing deletes no open popup | generator texts were not read at all with the flag clear | texts always read; the gate is tested at enter and at each step |
| TOWN-516: rooms build their popup at every enter while `TipsMode` is set, at fixed rectangles | enter, gate and Close matched; every popup fitted to its text's height | the description rectangle; the bottom edge moves only when the engine's wrap overflows it (`DIV-2719`) |
| TOWN-517: shop2 replaces shop1 once per shop activation on the idle check, without reading `TipsMode` | never read | the description's second text, read on the shop's paint pass when the table holds a place (`DIV-132` for the getter and the campaign bit) |

## As built

- `pkg/ui/guidedcycle.go`: the cycle machine. Its timers and index are
  process state; a page enter does not reset them. It draws nothing for 500 ms
  after a hover and steps at the first paint more than 300 ms after the last.
- `pkg/ui/chargen.go`, `chargen_page.go`, `app.go`: the step fields, the
  named-click advances, the state-bit art, mask-only hits, the cursors, the
  statistic and navigation button states. The App paints the cycle after each
  generator step with its frame clock.
- `pkg/ui/missiontip.go`: the mission popup on the viewer, drawn under the
  dialogue, its Close and checkbox taking their own presses. `pkg/game/world.go`
  keeps the last shown part's `tips=` value and raises the tip when the
  dialogue closes after its last page; the ROM2 dialogue is outside the claim.
- `pkg/game/chargenassets.go`: loads `Inn\ButtonsArea.bmp` and the six button
  pictures.
- `pkg/game/towns/rom1.json`, `pkg/town` `TipSpec`: each room's tip keeps its
  claimed rectangle; the shop's tip names its second text. `TipSpec.Fit` fits
  a popup to its text when a description asks for it; ROM1's does not.
  `pkg/game/tips.go` projects the popups and runs the shop's second-text
  latch, cleared at each shop enter.

## Proof

| Witness | Roots | Result |
|---|---|---|
| `TestGuidedCycleTiming` (pkg/ui): the cycle on a synthetic clock, hover freeze and resume | fixture | pass |
| `TestPreCreateTipStepsOnlyOnTheNamedClicks`, `TestChargenDetailedTipFollowsClassThenFirstSkillClick`, `TestChargenTipsModeOffStopsOnlyWhatTheClaimsSay` (pkg/ui) | fixture | pass |
| `TestReleaseGeneratorTipsCycleAndStep`: chrsel1..3 at their rectangle on the named clicks; cycle frames 320 ms apart after the wait; the class text, the skill cycle, chrgen2 and the stopped cycle; `+` art plain, hovered and held | EN, RU | pass |
| `TestReleaseMissionTipsFollowTheirTaggedDialogues`: every shipped tag in m10 (7) and m20 (2), start tips through the start triggers | EN, RU | pass |
| `TestReleaseMissionTipControlsAndTipsMode`: checkbox clears `TipsMode` at once and keeps the popup; no tip with the flag clear; Close on its release | EN, RU | pass |
| `TestReleaseMissionStartTipDoesNotReturnAfterSaveAndLoad`: F2 SAVE, cold LOAD, no start dialogue or tip | EN, RU | pass |
| `TestReleaseRoomTipsAtEveryEnterAndShopSecondText`: tavern, shop and school popups at their rectangles on two enters each, after Close; the town popup at its rectangle unless its text overflows; shop1 with an empty table, shop2 after a place, again after a new entry and with `TipsMode` cleared after the entry; no popup with `TipsMode` clear at the entry | EN, RU | pass |
| `TestReleaseRoomTipStyleAndLayoutFromInstall`: the four popups at the claimed heights, the RU town grown for its text | EN, RU | pass |
| `TestReleaseTownRoomTraceIsUnchanged`, `TestReleaseTownSquareTraceIsUnchanged`: re-recorded; only frame hashes move (the shop after a table item, the RU school popup), every save hash, message and sound line is unchanged | EN, RU | pass |
| `TestReleaseProfileWitnessIsUnchanged`: re-recorded; only the generator line's frame hash moves | EN, RU | pass |
| `TestReleaseChargenSoundsThroughAppInput`, `TestReleaseDifficultyLevelsInstalledArtAndPointer`, `TestReleaseChargenDetailedNavLabelsAreDrawn` updated to the claimed rectangles and mask | EN, RU | pass |

Owner renders: `TestReleaseMissionStartTipRender` and the generator witness
write PNG frames when `AGAINROM_TIPS_RENDER_DIR` is set.

## Open debt

- The RU town popup grows to 232 px because the engine's wrap needs 223 px;
  the original's overflow is Unknown (`DIV-2719`). The shop's tray getter and
  campaign bit gate are Unknown (`DIV-132`).
- Unknowns kept as rows: pre-create origin (`DIV-2714`), placement at larger
  resolutions (`DIV-2715`), the mission popup across SAVE, LOAD and mission end
  (`DIV-2716`), the dice cursor (`DIV-2717`), an unreadable tip file
  (`DIV-2718`), the label ink ramps (`DIV-169`).
- The cycle's wall-clock period depends on the paint rate (TOWN-519); the
  engine paints at its frame rate.
