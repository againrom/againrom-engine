# Story `1036` — closure

As-built evidence. Behaviour is `spec.md`; this file is the aspect matrix, the witnesses, the
reconciliation against the pin, and what is left open.

Base `1fcf7e0f`. Research pin `d7ee0c6`, unchanged through the story.

## Twelve-aspect matrix

| Aspect | Verdict | Evidence |
|---|---|---|
| Data | PASS | Nine corner bitmaps and the two mode bodies resolve from `graphics\interface\` through `readChargenBMP` with `keyBlack` (`pkg/game/characterpane.go`), cached on the front end once. No new file format, no new archive read in `pkg/ui`. `TOWN-356` names every entry; no size is asserted, because the blit size is an operand of the paint routine. |
| Runtime state | PASS | The presentation mode is `hudPanelDoll`, one of the four display switches, not a second field (`TOWN-351`'s `this+0x70`). Rect A and rect B read and write `hudPanelPack` and `hudPanelBook`. Rect F's request is one-shot (`TakeCharacterPaneMenu`). `paneCornerGrab` is the press-to-release latch. The panel cache key gained the mode, the figure and the two open flags, so a mode flip drops the cached picture. |
| Simulation | N/A | Nothing in this story reaches `pkg/sim`. No hashed state, no tick, no order. |
| Player input | PASS | Six rectangles on the primary release, in `TOWN-346`'s own order, every match firing (`DIV-310`); the Tab key as rect C's keyboard route (`TOWN-355`, `AI-KEY-125`); the held item pre-empting all six (`TOWN-348`); the corner/figure gesture split (`DIV-308`). Tab was unbound in `pkg/` before this story. |
| AI | N/A | No unit behaviour is touched. |
| UI/HUD | PASS | One composer for both screens (`DrawTownCharacterRegion`), reached by the mission through `panelPresent`. Three registered `GeometryTests` entries in `pkg/ui/screenregistry.go`, each mutation-proved at the production draw call. |
| Triggers/scripts | N/A | No script node, no trigger. The census is unmoved and measured below. |
| Inventory/equipment | PASS | The 160x200 slot map is untouched: `shopDollSlotAt` and `dollFigureSlotAt` answer over the whole painted crop exactly as before, which `TestShopDollHitAreaMatchesEveryVisibleSyntheticFigurePixel` holds. The separate authored doll box is removed and the worn box moves up with it (`DIV-316`). |
| Persistence/save-load | N/A | Nothing new is persisted. The presentation mode is a display switch, as it was before this story; the two original-save scenarios and the three save-gated release tests are green on both roots. |
| Campaign/session | PASS | `campaign+0x3dc` is carried as `TownCharacterView.Session`. The mission passes `CharacterPaneMission` (1) and every town-side screen `CharacterPaneShop` (2). Which value the tavern, school, square and generator carry is authored: `DIV-311`. |
| Shipped content | PASS | Both roots, against the lawful installs: 41 of 41 install-gated tests, 14 of 14 scenarios, and the two new pane screens captured through `cmd/screenshot` with real shipped art. The two captures are byte-identical between `en` and `ru`, which is what `EXP-0123` predicts for `interface/` art. |
| Interactions with existing mechanics | PASS, with one recorded collision | The drag surface, the shop's grids, the pack bar and the spellbook are unchanged. The shop's authored `Book` plaque and its `shopBookRect` were deleted in round 2 on the owner's own instruction; the shop's book control is now rect B's own decoded rectangle, drawn with the shipped `BookOpened.bmp` and `BookClosed.bmp`, and answers its click through `shopBookCornerHitAt`. One authored surface still meets decoded art: the statistics card's first left label is clipped by rect B's bitmap. It is a ledger amendment rather than a new row, because the reserved range is spent: `DIV-175`. |

No aspect is `GAP`. Every unbuilt behaviour named in `spec.md` section 9 is outside this story's
contract and carries its own row.

## Integration witness, in a real campaign mission

`cmd/screenshot`'s session drives one live `ui.App` the way a keyboard session drives it — root menu,
New Game, the first campaign map row, generation, Play — and lands on the campaign's own first
mission. Two screens were added to it, `mission-pane-doll` and `mission-pane-stats`, captured through
`App.HeadlessCharacterPane`, which composes the pane by the same call `Viewer.Draw` makes.

This is the only committed way to see the mission screen at all: `HeadlessFrame` refuses it, because
`Viewer.Draw` paints the ebiten canvas directly and composes no CPU frame. The pane is the one part
of that screen composed as an `*image.RGBA`.

Measured on both roots, at 640x480:

```
mission-pane-doll  captured -> mission-pane-doll.png
mission-pane-doll  note: the pane reported figure mode, subject Danath
mission-pane-stats captured -> mission-pane-stats.png
mission-pane-stats note: the pane reported statistics mode, subject Danath
```

The figure-mode capture shows the shipped `HumanBackR.bmp` body, the party member's painted figure,
the member-name row, rect A's `BackPackOp.bmp` at the pane's bottom left and rect F's `diskette.bmp`
at its bottom right. The statistics-mode capture shows `TextBackR.bmp` with all seventeen card rows
and the same two corners. Rects D and E are absent from both, which is `1 & 0x226 == 0`.

The session selects the first live party member before capturing, through `HeadlessSelectEntity` —
the same press the command path resolves — so the pane is photographed carrying a subject. A pane
with no subject still composes, and that is `panelPresent`'s own gate change.

`AGENTS.md` rule 7 asks a presentation mode to be witnessed against the other mode rather than
against its own controls. `TestMissionCharacterPaneModesDifferOnlyInWhatTheModeOwns` composes the
same view twice with the one field flipped and asserts the difference is confined to what the mode
owns and that the region did change. The two captures above are the same comparison with shipped art.

## What moved that someone can point at

**The mission has a doll at 640x480 and 800x600 for the first time.** Before this story the doll was
a separate authored box at `y` 488, 260 tall; `rightColumnBox` refuses `488+260` at both of the
resolutions the owner plays at, so the mission had no doll there at all. It is now the character
pane's own figure presentation mode, inside the 242 the panel already occupies, at every shipped
resolution. `builds/current/` shows it after the landing.

The worn box moved up with it, from `y` 756 to `y` 488, and draws at 1024x768 for the first time
(`DIV-316`).

## The script-node census

Unchanged, and this story was not meant to move it.

`pipeline/check-milestone.sh` with `AGAINROM_MILESTONE_DRIVE` set to a `missionrun` built from this
worktree: `the script gap and the drive are where they were recorded, both roots`, exit 0.
`pipeline/milestone-baseline.txt` carries 56 map rows and four drive rows and **no** `cannot run`
row, so the script gap is zero on both roots and stayed zero.

The two-mission count the lane brief asks for, `missionrun -mission N -trace -ticks 1 | grep -c
UNSUPPORTED`, is **0** for mission 10 and **0** for mission 20, on `en` and on `ru`, before and
after.

The drive is recorded, not asserted: `outcome lost at tick 240`, `census: 4 of 36 unit(s) moved, 1
fell, over 240 tick(s)`, identical on both roots and identical to the baseline. An unattended escort
loses by design.

## Gates

Run in `<seat>/wt-1036`.

| Gate | Result |
|---|---|
| `go build ./...` | exit 0 |
| `go vet ./...` | exit 0 |
| `gofmt -l` over tracked and untracked `*.go` | no output |
| `go test -trimpath -count=1 ./...` | exit 0, 43 packages `ok` |
| `scripts/check-claim-citations.sh` | exit 0 — 1268 distinct citations resolve against 1479 claims and 220 experiments under 787 prefixes |
| `scripts/check-no-game-assets.sh` | exit 0 — clean (tree scan) |
| `pipeline/check-release-tests.sh` `en` | exit 0 — selected 41 install-gated tests (38 `AGAINROM_ASSETS`, 1 `AGAINROM_ORIGINAL_SAVES`, 2 `AGAINROM_SAVE_666`); 41 of 41 ran and passed, 0 skipped |
| `pipeline/check-release-tests.sh` `ru` | exit 0 — same 41, 41 of 41 passed, 0 skipped |
| `pipeline/check-scenarios.sh` `en` | exit 0 — 14 of 14 |
| `pipeline/check-scenarios.sh` `ru` | exit 0 — 14 of 14 |
| `pipeline/check-milestone.sh` | exit 0 — census and drive at the recorded values, both roots |

The two glob lists were read rather than remembered: `scripts/check-*.sh` is two scripts, and
`research/scripts/check-*.sh` is three, none of which this story touches.

**One install-gated test went red and was repaired.** `TestReleaseStatisticsCardFitsAtTheProductionFont/town`
measures the card's ink by differencing the composed pane against the shipped body. Once the corner
bitmaps were drawn on the town pane, two of them stood inside the card's own row band and read as
text: the ink bounds moved from `(9,18)-(151,203)` to `(0,4)-(153,203)` and the name row's measured
centre moved 3.5 pixels. The repair is in the witness, not in production —
`cardInkMask` now excludes `ui.TownCharacterPersistentControls()`, which is the corners' own gate
rather than a second copy of their rectangles. The whole repository chain was green while this was
red, which is the failure `check-release-tests.sh`'s own header describes.

## Research reconciliation

Every claim was read whole against the pin with `go run ./tools/claim`.

Reproduced as decoded:

- `TOWN-346` — the six rectangles, panel-relative, each with its own `sess+0x3dc` gate, no early
  return between them. `characterPaneRects`, `CharacterPaneCornerLive`, `CharacterPaneCornersAt`.
- `TOWN-347` — what each rectangle posts. Each message has exactly one consumer in this build.
- `TOWN-348` — the held-item pre-emption before any point-in-rect call.
- `TOWN-351` — `this+0x70` as the presentation mode, forced to 1 by the constructor.
- `TOWN-352` — `campaign+0x3dc` as a bitmask.
- `TOWN-353` — one instance of the widget, re-parented; both screens carry the same six rectangles.
- `TOWN-354`, `TOWN-356` — the art bindings, the two mode bodies, and the corners' own destinations.
- `TOWN-355` — the two rectangle overlaps and the three art/hit gate disagreements, reproduced
  rather than repaired (`DIV-309`, `DIV-310`); and rect C's keyboard route.
- `MENU-ESC-010` — rect F raises exactly what Esc raises.
- `SHOP-PICKER-043` — rects D and E, whose four numbers this build already carried; they are now
  derived from `characterpane.go` rather than repeated as literals, and their gate is applied.
- `SHOP-FIGURE-041` — the id-7 slot the mission pane stands in.
- `AI-KEY-125` — `0x412` as the focused character panel's Tab message.

Confirmed an earlier inference: `DIV-175` had paired the oval body with the doll and the rectangular
body with the card by frame shape. `TOWN-354` plus `TOWN-356` decode that pairing and it is the one
this build already drew. The row is amended and stays OPEN for the 16-wide seam, which `TOWN-356`
does not tie.

Corrected a false absence claim: `DIV-217` asserted that no claim gives this widget's interactive
controls. Two of the six had been published since `EXP-0173`, and this build's own two constants were
that row's numbers. The row is amended, all six rectangles now exist, and it stays OPEN for the
coordinate-free right-button-up.

Not decoded, decided here under the owner's 2026-08-23 directive, each with a row: the map view's own
`0x40e`/`0x40f` arms (`DIV-307`), the session value each town screen passes (`DIV-311`), rect C's
height gate at 1024x768 (`DIV-312`), the `sess+0x6bc` picker suppression (`DIV-313`).

## Divergence rows

Eleven rows are spent. Ten came from the story's first reserved range,
`DIV-307`..`DIV-316`. That range ran out before the story finished and a second was allocated,
`DIV-325`..`DIV-327`; round 2 spent one of it. `DIV-325` and `DIV-327` are unspent and return.

| Id | Type | Subject |
|---|---|---|
| `DIV-307` | DEVIATION | Rects A and B consume `0x40e`/`0x40f` as this build's pack-bar and spellbook switches; the map view's own arms for those two messages are not decoded. |
| `DIV-308` | DEVIATION | Corner and figure share pixels and are separated by GESTURE, because this build reaches both from one button where the original uses three different messages. |
| `DIV-309` | DEVIATION / ACCEPTED | The art gates and the hit gates disagree for four of the six, and the disagreement is reproduced. |
| `DIV-310` | DEVIATION / ACCEPTED | Overlapping rectangles both fire, because the original has no early return between its six tests. |
| `DIV-311` | UNKNOWN | Which session mask the tavern, school, square and generator pass. 2 is authored. |
| `DIV-312` | UNKNOWN | Rect C's `flag` compares the pane's own rect; the original's accumulated vertical origin above `campaign+0xd4` was not read. |
| `DIV-313` | UNKNOWN | The `sess+0x6bc == 2` half of the picker art suppression. The parameter exists and every caller passes false. |
| `DIV-314` | FIDELITY-DEBT | Sound `0xdc` on rects C, D, E and F, and rect C's `[mapview+0xe0] = 1` redraw flag. |
| `DIV-315` | FIDELITY-DEBT | `TOWN-348`'s drop-slot choice on held-item values 1 and 2. |
| `DIV-316` | DEVIATION | The worn box moves from `y` 756 to `y` 488 and draws at 1024x768 for the first time. |
| `DIV-326` | DEVIATION | A selected unit with no known spells opens the same spellbook bar a caster does, empty. Owner-directed, given as round 2's adversarial return. |

Three existing rows were amended: `DIV-175` (the decoded body pairing, and the card-versus-corner-art
clip), `DIV-200` (what the emptied fourth slot now owes), `DIV-217` (all six rectangles exist).

## Open items

- The eighth interactive behaviour of `TOWN-344`, the coordinate-free `WM_RBUTTONUP` posting `0x405`
  to the map view. Its own story. `DIV-217`.
- The authored statistics card clips `BODY`'s first glyph column under rect B's shipped bitmap. The
  card cannot move without pushing its last row under rect F. A decode of the original's own
  statistics text layout would settle it.
- The divergence range ran out. One late finding, the clipped `BODY` label, is carried as an
  amendment to `DIV-175` instead of taking a row of its own. It is a real row in the ledger and
  is not lost, but the seat should know the range was spent before the story finished. No id is named for either finding here: a document
  that names an id spends it, because `pipeline/next-div-id.sh` answers one above the highest
  mention.
- `CharacterPaneShopInMission` (3) exists and no screen passes it. It becomes reachable when a shop
  can be opened from a mission, which is when both of `TOWN-355`'s overlaps first have two live
  members.

## Scope

The story stayed inside its contract's six behaviour groups. Its ceiling is four adversarial
passes, which is what `contract.md` states. `AGENTS.md`'s ladder gives four where a story
reaches hashed simulation state **or** touches more than three domains; the two conditions are
alternatives, not a conjunction. This story touches four domains, UI/HUD,
inventory/equipment, input and campaign/session, and reaches no hashed simulation state, so the
four-pass ceiling applies on the domain count alone.

Two adversarial passes ran. The chain ended at round 3 by owner directive rather than at the
ceiling or at a pass with no P finding: *«как закончат агенты так максимально быстро заканчиваем и мержим, все ошибки потом
хотфиксами отдельно доведем»*, 2026-08-23. The open items above are what that directive defers.
