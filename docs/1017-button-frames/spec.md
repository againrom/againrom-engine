# 1017 — spec

As-built. Canonicalized to the behaviour this story shipped.

## FR-1 — the school draws its two shipped buttons in their own wells

`interface/training/buttonsarea.bmp` (160x238) was already loaded as `TownSchoolArt.Upper`.
`LoadTownSchoolArt` (`pkg/game/townschoolart.go`) now also reads `interface/training/buttons/
b{1,2}{off,on}.bmp` (140x46 each) into `TownSchoolArt.Buttons[2][2]`; any one missing or mis-sized
fails the whole load, the same atomic-or-nothing rule the picture already used.

Before this story, `townSurfaceButtonRect` (`pkg/ui/townshell.go`) placed three 166x52 boxes at the
wide widget's own origin `TownWideUpperRegion.Min` = `(464,0)`, sixteen pixels left of the shipped
picture's own left edge `TownUpperRegion.Min` = `(480,0)`, and drew a third box (Talk) the shipped
picture has no well for. Research not carried in this story's pin — read at the coordinator's own
research branch tip, not merged to `master`, so cited here by fact and not by claim id — confirms
this from the executable: the paint routine's own `ButtonsArea.bmp` blit and the button boxes both
use the picture's own origin, and the pre-fix code took the wider widget's edge instead. This is
settled, not open; no divergence is recorded for it.

`townSurfaceButtonWells` now carries each button's own well, measured by correlating the shipped
button bitmap over the shipped area picture (`cmd/buttonframecheck`), local to the picture:

| Room | Button | Well (local to the area picture) |
|---|---|---|
| school | 0 (Train) | `(4,71)-(144,117)` |
| school | 1 (Exit) | `(4,117)-(144,163)` |

Both wells are 140x46, flush against each other (the well height), starting four pixels from the
picture's own left edge. `townSurfaceButtonRect(kind, i)` adds `TownUpperRegion.Min`, giving screen
rectangles `(484,71)-(624,117)` and `(484,117)-(624,163)`. The same function backs both the draw
call and `TownSurfaceControlAt`'s hit test — one function, two callers, so the draw rectangle and
the hit rectangle cannot disagree by construction.

The shipped `b1`/`b2` off/on bitmaps carry no baked text (round-2 adversarial review, pass 1, P-1):
every one of the four sampled directly from both preserved roots is a plain bordered plaque with no
glyph pixels in it. No claim addresses how the original itself draws its own button label; the gap
is flagged for the coordinator to allocate a divergence id rather than assigned one here.
`ComposeTownSurface` draws the button's own art (if present) or the pre-1017 flat box (if not), then
always draws `b.Label` and `b.Value` over it with the shell's own text drawing, at the same two
rectangles the pre-1017 box path used relative to the well. `TestReleaseSchoolAndTavernButtonArtIsPlacedAndLabelled` and
`TestReleaseSchoolAndTavernButtonPressedArtIsPlacedAndLabelled` (`pkg/game/townbuttons_release_test.go`)
witness this against the real install: exact placement outside the label/value rows, a real pixel
difference from the raw bitmap inside them, and no two buttons of a room pixel-identical over their
own wells.

The school has two buttons now, not three: `TownScreen.TownSurface()` (`pkg/game/townshell.go`)
returns a two-element `[]TownSurfaceButton{Train, EXIT}`, and `townSurfaceButton(i)` dispatches
index 0 to training and index 1 to `Back()` (EXIT moved from index 2 to index 1). The Talk box and
its dispatch case are gone from this path; see FR-6 for why removing it does not strand any shipped
content, and "Legacy list path" below for what still exists but is unreached.

Which shipped bitmap (`b1` or `b2`) is Train and which is Exit is not decoded (`DIV-159`): this
build reads them in file-number order, matching the well order top to bottom, the same convention
the tavern (FR-2) and the pre-1017 character generator dispatch (FR-4) already used.

## FR-2 — the tavern draws its three shipped buttons in their own wells

`TownSurfaceView.TavernArt` already existed before this story (`TownTavernArt.LeftPicture`,
`LeftStats`, `Center`, `Units`), loaded and drawn for the town content region — the left furniture
strip and the merchant's roster. What did not exist was its upper-region counterpart: no `Upper`
field and no `Buttons` array, so `ComposeTownSurface` fell back to flat placeholder boxes for the
whole upper region regardless of `TavernArt`'s own presence. `LoadTownTavernArt`
(`pkg/game/towntavernart.go`) now also reads `interface/inn/buttonsarea.bmp`
(160x238) into `TownTavernArt.Upper` and `interface/inn/button{1,2,3}{off,on}.bmp` (140x46 each)
into `TownTavernArt.Buttons[3][2]`, atomically as in FR-1 (`TestLoadTownTavernArtIsAtomic`,
`pkg/game/towntavernart_test.go`, added round 2 — no synthetic fixture had installed anything under
`interface/inn/` before). `ComposeTownSurface` draws `TavernArt.Upper` at `TownUpperRegion` with
`draw.Src`, the same function and origin the school's own draw uses. Round 2 added an
`outline(dst, TownWideUpperRegion, townShellBorder)` call after the school's own draw and omitted it
for the tavern, reasoning that `TownWideUpperRegion`'s own `464..640` widget rect is researched for
the school (`1015`) and not for the tavern, so painting it there would draw an authored line over the
shipped `centerarea.bmp` with no research behind it for this room (round-2 adversarial review, pass
1, P-3). Round 3 found the school's own outline call wrong on the same grounds: the school's shipped
`Upper` picture is 160 pixels wide, exactly filling `TownUpperRegion` (`480..640`); `TownWideUpperRegion`
is sixteen pixels wider, so the outline painted an authored border line over bare room background at
`x=464`, sixteen pixels left of the picture's own edge — visible in the full frame, invisible in a
crop of the upper region alone, which is how round 2's review missed it. The school's outline call is
removed; neither room draws one, and `TestReleaseSchoolAndTavernUpperRegionMatchesShippedArtOutsideButtonWells`
(`pkg/game/townbuttons_release_test.go`) checks every pixel of both rooms' wide upper region against
the shipped background/upper picture, outside the button wells, against the real install. Buttons
carry labels the same way FR-1 describes.

| Room | Button | Well (local to the area picture) |
|---|---|---|
| tavern | 0 (Hire) | `(4,44)-(144,90)` |
| tavern | 1 (Talk) | `(4,91)-(144,137)` |
| tavern | 2 (Exit) | `(4,138)-(144,184)` |

Screen rectangles: `(484,44)-(624,90)`, `(484,91)-(624,137)`, `(484,138)-(624,184)`. These are the
executable's stored rectangles (`TOWN-214`, corroborated by `TOWN-282`) under the exclusive-bottom
convention `TOWN-258` establishes. The shipped background's two lower recesses correlate one row
higher, at 90 and 137. `cmd/buttonframecheck` reports that art observation separately and checks the
production table against the executable rectangles. `TownSurface()` returns
`[]TownSurfaceButton{Hire, Talk, EXIT}`; the tavern keeps its Talk button and its existing dispatch
(select an NPC from the cells, press Talk to open that NPC's conversation), unchanged by this
story. Which shipped bitmap is which of Hire/Talk/Exit is authored the same way as the school's
(`DIV-159`).

## FR-3 — the shop draws its four shipped buttons in the same shipped frame

This behaviour was already built before this story: `pkg/game/shopart.go`'s `loadShopArt` already
read `graphics/interface/shopmenu.bmp` (176x238) and `graphics/interface/shopbutton{1..4}.bmp`, and
`pkg/ui/shopscreen.go`'s `ComposeShopScreen` already blitted them at `shopButtonRects`. The
contract's own premise that this art was unlocated was wrong; this story's contribution is
verification, not construction, plus one exported accessor, `ShopButtonRect(i)`
(`pkg/ui/shopscreen.go`), so `cmd/buttonframecheck` can measure it the same way as the school and
tavern.

| Button | Bitmap | Size | Screen rect |
|---|---|---|---|
| 0 (Undo) | `shopbutton1.bmp` | 120x52 | `(494,15)-(614,67)` |
| 1 (Buy) | `shopbutton2.bmp` | 140x46 | `(483,67)-(623,113)` |
| 2 (Sell) | `shopbutton3.bmp` | 140x46 | `(483,114)-(623,160)` |
| 3 (Exit) | `shopbutton4.bmp` | 120x52 | `(494,160)-(614,212)` |

Measured by correlating each bitmap over `shopmenu.bmp` and comparing the winning offset (plus
`TownWideUpperRegion.Min`) against `shopButtonRects`: exact agreement, both preserved roots. The
shop ships one bitmap per button, not an off/on pair: `shopbutton1.bmp` and `shopbutton4.bmp` carry
716/6240 (11.5%) and 804/6240 (12.9%) pure-black pixels; `shopbutton2.bmp` and `shopbutton3.bmp`
carry none. The shop's own pre-existing `keyBlack`/`blit` path (`draw.Over`, alpha zeroed on pure
black at load) keys them, unlike the school's and tavern's opaque `draw.Src` — see `DIV-158` for
what that difference means against the two decoded blit primitives.

## FR-4 — the character generator draws its buttons in the same shipped frame

`interface/chrgen/buttonsarea.bmp` (160x238) is new: `LoadChargenAssets`
(`pkg/game/chargenassets.go`) reads it into `ChargenPresentation.NavArt`, and
`composeChargenDetailedPage` (`pkg/ui/chargen_page.go`) draws it at `chargenNavBox` (=
`TownUpperRegion`) with `copyNative` (opaque, `draw.Src`), falling back to the pre-1017
`drawChargenFrame` box when `NavArt` is nil. **That fallback is not reachable through the shipped
load path.** `LoadChargenAssets` (`pkg/game/chargenassets.go`) requires the file at 160x238 and
`NewFrontEnd` (`pkg/game/frontend.go`) propagates the error, as it already does for every other
chargen asset in the same function, so `NavArt` is nil only in a hand-built fixture.

The generator ships no per-button bitmap: the recessed panels for Play, Reset, Back and one further
plaque are baked directly into the one area picture, unlike the school's and tavern's separate
per-button files. `detailedControlRegion` (`pkg/ui/chargen_page.go`) places three of the four wells
found by segmenting `buttonsarea.bmp`'s own rows and columns by brightness (`cmd/buttonframecheck`),
identically on both preserved roots:

| Well (screen) | Assigned to |
|---|---|
| `(506,20)-(607,65)` | none |
| `(486,69)-(617,111)` | Play |
| `(486,115)-(617,160)` | Reset |
| `(504,163)-(601,208)` | Back |

The segmentation threshold is chosen from the picture, not from the nav action count: round 1 used
33, at which the topmost well's run is 19 rows, one under the 20-row minimum, and the segmenter
dropped it — returning three wells that happened to agree with this build's three existing actions.
Round 1's `spec.md` read that agreement as evidence for the count, which is backwards; the visual
read of four identical plaques was right, and the reviewer's own re-run at threshold 40 (round-2
adversarial review, pass 1, P-2) resolves all four, identically on both roots. This build still has
only three nav actions, so the topmost well is drawn as shipped art, unmodified, the same way the
other three are (`composeChargenDetailedPage`, `pkg/ui/chargen_page.go`): a player is shown four
plaques and three of them respond to a click. Round 2 instead covered the fourth with
`drawChargenFrame`'s own dark-fill-and-border, the flat-fill treatment this page uses when it has no
art for a region at all; round 3 found the fill did not follow the plaque's own shape — three of its
four corners sampled onto warm brown wood outside the plaque rather than the plaque's dark interior,
on both preserved roots — so a player was shown an opaque rectangle that did not match the picture
under it. The fill is removed; the picture is drawn unmodified and the fourth plaque is left inert,
with no control rectangle over it, so a click there does nothing.
`TestReleaseChargenDetailedNavArtIsDrawnUnmodified` (`pkg/game/chargen_release_test.go`) checks the
composed nav region against `NavArt` pixel for pixel, outside the three labelled wells, against the
real install. Which of the four the original wires to a fourth action, if any, is undecoded
(`DIV-156`, amended round 3): the well count and rectangles are measured, reproducible facts; the
well-to-action assignment is authored, and the fourth well's own function is unknown rather than
assumed empty.

## FR-5 — a pressed button shows its own on state

Every school and tavern button loaded an off/on pair (FR-1, FR-2). `App.townSurfacePress`
(`pkg/ui/app.go`) records which button control the primary mouse button went down on
(`TownSurfaceControlAt` at `PrimaryPressed`) and clears it at `PrimaryReleased`. While the button is
down, `drawTown` re-resolves the current cursor position each frame and sets `surface.Press` to that
recorded control only if the cursor is still over it — an AND of two conditions: the control the
press began on, and the control currently under the cursor. `ComposeTownSurface`
(`pkg/ui/townshell.go`) draws a button's `Buttons[i][1]` (on) bitmap only when `v.Press.Kind ==
TownSurfaceControlButton && v.Press.Index == i`, else `Buttons[i][0]` (off); exactly one `draw.Draw`
call per button, never both bitmaps.

Research not carried in this story's pin (see FR-1) confirms the general shape: the original also
gates its own ON bitmap on two conditions both equalling the button index — a hover index and a
second field whose writer is not established (an explicit negative: six routines plus a whole-`.text`
scan of 461 unfilterable hits found no write site). This build's second condition (press began here,
cursor has not left) satisfies that shape and produces the requested click feedback, but is not
confirmed identical to the original's own unread second condition (`DIV-155`).

The shop has no separate on-state bitmap for any of its four buttons (FR-3), so this behaviour does
not apply there: a shop button shows the same bitmap pressed or not. The character generator's nav
wells (FR-4) are baked into the one area picture with no per-button art either, so the same applies
there.

## FR-6 — the school and shop offer their mission once per entering the room

Owner ruling, quoted verbatim in `contract.md`: "Кнопка talk в школе не нужна (как и в магазине),
потому что выдача задания происходит просто при заходе один раз когда задание можно выдать и диалог
запускается. Всего один раз на каждое задание." (the Talk button is not needed in the school, nor in
the shop, because the mission offer happens simply on entering, once, when the mission can be given,
and the dialogue fires; only once per mission.)

`Town.taken` (`map[offerRef]bool`, `pkg/game/town.go`), keyed `offerRef{mission, building, index}`
and pre-existing before this story, is the sole gate: `Town.Offers(b)` filters out any index the
map holds, and both open sites below act on `Offers`'s own result.

- `townScreen`'s room-enter handler (`pkg/game/townscreen.go`): on entering `roomShop` or
  `roomSchool`, if `Offers(building)` is non-empty, it calls `openTownDialogue` on the first offer
  (an install shipping no offer text stays silent and re-offers on the next entry — unchanged from
  before this story, still asserted by `TestMissingOrEmptyTownDialogueDoesNotOpenOrConsumeAnOffer`).
  `openTownDialogue` enters the modal room only when the shipped payload carries a first part
  (`EventPart(payload, 1, ...)`); a missing or empty file returns `false` with no state change.
- The shop's own `ShopControlMerchant` case (`pkg/game/shopview.go`), the merchant portrait press:
  gated the same way, on `len(Offers(TownShop)) > 0`, so a player who has already accepted the
  offer gets "he has no work for you today" from the merchant too. Before this story the merchant
  press had no gate at all and could reopen the same offer indefinitely.

Rounds 1 and 2 gated both sites on a second, separate latch, `Town.shown` (`Shown`/`MarkShown`),
written independently of `taken` to answer "has this offer's dialogue already been shown", with its
own write point moved once (round 1 wrote it at open time, which stranded a declined offer for the
rest of the chapter; round 2 moved the write to `talk`'s accept branch, immediately after `Take`
succeeds, removing the stranding). Round 3 found the second latch redundant in production: at both
read sites the offer being checked is drawn from `Offers`, which already excludes anything `taken`,
and `shown` is written only where `taken` is also written (the accept branch, right after `Take`
succeeds) — so `shown` is never true where `taken` is false, and every offer the two read sites ever
see is one `Offers` has already established is not taken, hence not shown either. `Shown()`'s answer
at both call sites is therefore always false, provably, without touching the tests: mutating
`Shown()` to return a constant `false` and running the full suite changed the result of exactly one
test, `TestTownOfferAcceptedOnceEscapedRepeats`, at its own direct assertions on `Shown()`, and no
other test and no install-gated scenario. `Town.shown`, `Shown()`, `MarkShown()`,
`Snapshot.Shown`, `restoreTown`'s Shown-restoring loop and the `sortOffers(s.Shown)` call are
removed; `taken` alone now gates both read sites, unchanged in externally observable behaviour.
`TestTownOfferAcceptedOnceEscapedRepeats` (`pkg/game/town_test.go`) exercises the full path for
both buildings against `Offers`'s own length, rather than against the removed latch: auto-open on
entry, decline via `Back`, confirmation the offer is still listed by `Offers`, reopen via the
merchant press, accept via `AdvanceTownDialogue`, and confirmation `Offers` is then empty and the
merchant answers "he has no work for you today".

The school's Talk box and its dispatch case are removed (FR-1); the shop never had one in this
surface (its own screen is `pkg/ui/shopscreen.go`, driven by the mouse alone, not
`TownSurfaceView.Buttons`). The tavern is unaffected: it keeps its Talk button, which selects among
several NPCs rather than firing one chapter offer, and gates on no offer-dialogue latch at all.

`Taken`/`Take` read the current chapter through `Town.ChapterData()`, so the state is per-mission by
construction: a school or shop that has consumed its offer for mission N still offers mission N+1's
on first entry after the chapter advances. `Snapshot.Taken []SnapshotOffer` (`pkg/game/save.go`)
carries the state through save and reload; `TestATownSaveRoundTripsThroughDisk` takes an offer before
saving and reads it back from a decoded snapshot after restore. `s.Taken` is sorted by chapter, then
building, then index (`sortOffers`, `pkg/game/save.go`) before being written: `t.taken` is a Go map
and ranges over its own randomized order, so an unsorted snapshot could serialize a save of the same
game state to different bytes across two runs (round-2 adversarial review, pass 1, live). Round 3
adds `TestSaveTakenOffersEncodeDeterministicallyAcrossRepeatedSnapshots` (`pkg/game/save_test.go`):
three taken offers, twenty `Snapshot`+`EncodeSave` calls, all twenty payloads required byte-identical.

The owner's ruling reproduced above says "once" without qualification; this build reads it as
"once accepted," and a declined offer is re-offered rather than lost, which is a narrower reading
than "shown once, never again." ROM1's own repeat behaviour after an offer dialogue is declined was
not researched for this story, and whether the owner intended the literal reading for the decline
case specifically is open (`DIV-157`, amended round 3). The mechanism describing it changed between
round 2 and round 3 (a dedicated `Shown` latch removed in favour of `taken` alone, above); the
behaviour it describes — an offer opens on each room entry or merchant press until accepted, then
never again for that mission — is unchanged by the amendment.

## Legacy list path

`pkg/ui/app.go`'s town input dispatch checks `townSurfaceScreen(a.flow.town)` first and returns
immediately once true — `townSurfaceScreen` is unconditionally true whenever `AtTownSurface()` is,
which the school and the tavern both are (established at `1016`). So the row-list `Rows()`/
`Choose(i)` methods are never reached by a real player's mouse or keyboard input, or by
`HeadlessActivate`, while at the school or the tavern: `headlessActivateTownSurface`
(`pkg/ui/headless.go`) matches only against `surface.Buttons` and `surface.Cells`, the same slices
FR-1/FR-2 populate, and dispatches through `clickTownSurface`, the identical call a real click makes.

`townScreen.openBuildingDialogue`'s `roomSchool` case (`pkg/game/townscreen.go`,
`case roomSchool: return t.openBuildingDialogue(TownSchool, i)`) still exists, still fires the
school's offer dialogue unconditionally for `i == 1` (no `taken` gate), and is not called from
anywhere production reaches: the switch it sits in belongs to the legacy row-list `Choose` method,
which no live input path selects for this room after FR-1/FR-2/FR-5. It stays for a caller that
invokes `Choose(i)` directly rather than through `app.go`, and this story does not remove it. **No
such caller exists today**: `cmd/schoolcheck`'s only `Choose(i)` call is the town-square door and
never reaches this arm.

## Readiness and fallback

`artSchool := v.Kind == TownSurfaceSchool && v.SchoolArt != nil` and the equivalent for the tavern
(`pkg/ui/townshell.go`) are the only readiness gates: `TownSurface()` (`pkg/game/townshell.go`)
assigns `v.SchoolArt`/`v.TavernArt` directly from `FrontEnd.TownSchoolArt`/`TownTavernArt`, which are
`nil` whenever `LoadTownSchoolArt`/`LoadTownTavernArt` failed at start-up (any one of the shipped
nodes missing or mis-sized). An install failing to ship any button bitmap falls back to the
pre-existing `drawTownShellBox`/text layout for the whole room, the same fallback FR-1 through FR-3
of `1015`'s school column and `1016`'s town square already use. A button whose own bitmap loaded but
whose room art did not (impossible given the atomic load, kept as a defensive per-button `nil`
check) falls back to the same box for that one button only.

## Divergence rows

`DIV-155` through `DIV-159`, all spent (`docs/DIVERGENCES.md`). None returned unused. Round 2
amended `DIV-156` (the chargen well count, FR-4) and `DIV-157` (the decline-latch reading, FR-6) in
place. Round 3 amends both again: `DIV-156` to state the fourth chargen well is drawn as shipped art
and left inert, its function undecoded (FR-4, above); `DIV-157` to state the mechanism is `taken`
alone, with the behaviour it describes unchanged (FR-6, above). No new id was allocated for either
amendment, since each describes a correction to the same subject the existing row already covers.

`DIV-165` (allocated by the coordinator) records what this round leaves alone rather than fixes: the
school's and tavern's button labels are hardcoded English text on both the `en` and `ru`
roots (`TownSurfaceButton.Label`/`.Value`, `pkg/ui/townshell.go`), pre-existing before this story
and out of its scope. **The shop's four command buttons are not among them**: they carry one number
each and no word, which is what `SHOP-SCREEN-035`'s own paint routine does. Its one hardcoded
English caption is the `"Book"` box (`pkg/ui/shopscreen.go`), pre-existing from `1006` and outside
the button panel. `TOWN-183` (High), published after this story's own pin was frozen and read at
the story boundary, establishes that the original draws its own text label onto each button after
the per-button blits: each of the paint routine's two iterations draws one text or number label
through a vtable `+0x14` slot on an object selected from `[L09922]` or `[L09928]`. That is
consistent with FR-1's own sampling that the shipped `b1`/`b2` bitmaps carry no baked glyph pixels.
The label mechanism itself — font, exact positioning, and whatever selects the label string per
locale — is not decoded, the `+0x14` slot being the undecoded half. Fixing the hardcoding is deferred to whichever story decodes that mechanism or is
directed to build a translation seam without it.
