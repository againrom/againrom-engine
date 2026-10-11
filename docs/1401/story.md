# The second game's first town built by the town composer

## Intent

On a second-game root, town 1 is a picture: the square with its gate,
guards, tavern, sign, stars, weather vane, birds and wildlife, and the inn
as a page with its panels, candle, cauldron, taverner, seated speakers and
Talk and Exit buttons. The town composer (`pkg/town`) builds both from one
description, `pkg/game/towns/rom2-town1.json`; the second game's campaign
answers its hooks. Towns 2 and 3 and the destination list keep the row
screen. The first game's town is unchanged.

Base: `6b002c8a`. Knowledge pin: k238.

## Authority

- Owner: one town composer; one builder per kind; a second-game screen is
  its own screen, with second-game evidence only.
- Square: `R2-ENGINE-239` through `R2-ENGINE-244`, `R2-ENGINE-231` (music
  b14). Inn: `R2-ENGINE-245` through `R2-ENGINE-248`. Room exit and square
  re-entry: `R2-ENGINE-250`. Gate flag: `R2-SESSION-128` (flag 0x301 is bank
  slot 769, set by the stage-10 topic-10 talk). Departure: `R2-ENGINE-144`.
  Every cite list in the description names at least one `R2-` claim; no
  first-game claim is cited.
- Rows: DIV-2962 through DIV-2968 are new; DIV-2390 and DIV-2635 are
  narrowed; DIV-2636 (kinds 1 and 2) and DIV-2844 (RU words) stand.
  DIV-2969 is unused.

## As built

- `pkg/town`: a room may carry `seats` (a `SeatSpec` grid: first rectangle,
  column and row steps, columns, rows, back and actor art, the selection
  sound slot, the selected seat's frame period) and `buttons` (`ButtonSpec`:
  name, rectangle, off and on art, caption slot). `Vocabulary.Pages` names
  the page kinds a game draws; a description naming another is refused.
  `Description.Room` and `Scene.Sound` are new accessors.
- New page kind `inn`. The room's static panels are scene layers, the
  candle, cauldron and taverner scene actors, in paint groups `back` and
  `front`. The roster seats and the buttons are room data drawn by the
  page widget in `pkg/game/secondtown.go` between the two groups. A new kind
  was needed because the existing tavern page (`ui.TownSurfaceView`) holds
  first-game geometry in code (`ShopButtonRect`, the tavern portrait grid);
  carrying the R2 rectangles there would either change the first-game
  tavern or put second-game numbers in first-game code.
- `pkg/game/secondtown.go` is the second game's adapter: the vocabulary
  (hooks `set-room`, `gate-closed`, `navigate`; condition `gate-open`; page
  `inn`), the description by town ID (`secondTownDescriptions`), the
  answers of the one scene adapter `townSceneHost` (the screen implements its `sceneCampaign` interface, as the first game's town screen does), art loaded once per screen through the
  install's art library, and the square and inn views behind the ui's
  composed-square seam (`AtTownSquare`, `TownSquareView`, `TownSquarePointer`,
  `TownSquareActive`, `AdvanceTownSquareAnimation`). The composed
  presentation state lives on the screen (`secondTownState`); the campaign
  room stays in `secondCampaign.room`, which the SAV records.
- Rows (the list fallback and the headless seam): the square lists TAVERN and
  GATES from the hotspot rows; the inn lists TALK per speaker, then EXIT.
  Choosing a square row runs its hotspot's click program. A TALK row selects
  its seat and talks. EXIT and Escape leave the inn for the square.
- Gate: `gateOpen` reads bank slot 769 for town 1 (`R2-SESSION-128`). The
  closed gate is choosable and opens the town text section `plagatguard`;
  the open gate departs.
- `pkg/ui`: `TownSquareControlSelect` (a seat click selects without
  choosing, through `TownSquareSelector`), `DrawTownCaption`, and the
  headless square route syncs music as a click through `Update` does.
- `pkg/base`: the second edition no longer names the first game's room
  description (`Rooms`), so a second-game install loads no first-game room
  art. `RoomDescription` answers nil there; the world-map art loader
  tolerates that.
- `pkg/town` validation (story1377 note 3): a wait compare other than
  `greater` or `at-least`, a wait or pick form other than empty or `scaled`,
  a negative wait or pick, a draw from an unknown source, a selector that
  never loops back or stops, and a frame cue beyond its art are refused at
  decode.

## Carried from the unlanded first-town candidate

- Kept, as adapter code: the campaign binding (which town, which room), talk
  (`npc%dtalk%d`, then TalkTo; speakers of kinds 0 and 3, first match,
  option order), the SAV room and cold LOAD into it, and the gate rule, now
  on bank 769 with the closed gate opening `plagatguard`.
- Dropped: the first-game town screen as the second game's town, its room
  sync calls, the second tavern surface, the first-game school code and row
  alignment, and the install-load branch for second-game tavern art. The
  candidate's DIV-2672 is superseded by the owner's ruling. Its other rows
  are re-homed: the shop (DIV-2963), speakers and inn presentation
  (DIV-2964, DIV-2635), gates and exit (DIV-2966). The school needs no row:
  tip 234 and no action match `R2-ENGINE-241`. Hiring stays under DIV-2636.

## Proof

- `pkg/game` `TestRom2TownDescriptionEveryValueIsCited`: every object
  stating a value carries a cite; every cite list holds an `R2-` claim.
- Release witnesses on the second-game roots (`secondtownsquare_release_test.go`):
  the mask census (`R2-ENGINE-241`); layer order, destinations and frame
  counts (`R2-ENGINE-240`); per-selector hover and click, school tip 234
  with no action, closed gate `plagatguard`; gate and guard frames under
  gate hover for both flag values (`R2-ENGINE-243`); the tavern episode
  under the synthetic clock and no step at 67 ms (`R2-ENGINE-242`); the inn
  rows, seat, button and panel rectangles (`R2-ENGINE-246`, `R2-ENGINE-247`);
  TALK 517 runs TalkTo; EXIT and Escape return; a SAV written in the inn
  loads cold into the inn. Results are in the lane return.
- RU: the square claims cover both locales, so the RU run asserts the same
  census, layers and selectors. The inn rectangles are EN-measured
  (`R2-ENGINE-245`); on RU the witness asserts the description's EN values,
  not RU geometry (DIV-2965).
- The second-game release files that drive town 1 add an EXIT step before
  GATES from the inn.
- The first-game town trace, room trace and town, tavern, shop, school,
  city, world-map and gate release files are unchanged (lane return).

## Story1377 review notes

1. Refused LOAD and page state: unchanged. No refused LOAD path mutates a
   page; the shared map objects only hide a change no path makes.
2. Screen copies sharing pages: unchanged. The one reachable case arms the
   school diamond before a failed city-book update, as the purchase already
   arms it by design (TOWN-379).
3. Validation: fixed, see As built; `pkg/town` `seats_test.go` refuses each
   probe from the review.
4. Trace coverage: unchanged. The trace is the step-1 instrument; the
   merchant's no reaction is covered by `shopinterior_test.go`.

## Open debt

- The shop page (DIV-2963) and the tips panel on entry (DIV-2968).
- Inn presentation unknowns (DIV-2964): actor art, portrait and stats, the
  lower-right image, press state.
- Towns 2 and 3 are row screens.
- Square and inn values the claims leave open: DIV-2962, DIV-2965,
  DIV-2966, DIV-2967.
