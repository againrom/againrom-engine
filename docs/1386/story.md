# One builder for picture-plaque buttons; the town latches are the kit latch

## Intent

The generator's Accept, Reset and Back, its stat plus and minus pictures,
the tavern and school command plaques, the shop's four command plaques and
the Save dialog's buttons draw through the kit push button, and the four
town press latches are the kit latch. Base: `f7f7dc24`.

Owner direction: frames must be unified, "and buttons too" (architecture
audit `14b4f7e3`, rows 8 and 16). One builder per kind. No pixel moves.

## Authority

Owner direction only. The plaque art, its states and its press rules keep
the claims the former painters cited: MENU-138, MENU-139, TOWN-182,
TOWN-183, TOWN-260, TOWN-383, TOWN-391, TOWN-392, SHOP-050; the latch keeps
MENU-116. DIV-1404 and DIV-159 are unchanged. No divergence row is added.

## One builder

Kind: push button. Its one builder is `drawPushButton`
(`pkg/ui/widgetbutton.go`). It is extended, not doubled: a `pushButton`
with a `Face` is a picture plaque. The state fields (`Hover`, `Pressed`,
`Inside`, `Disabled`) and the sunk rule (`Pressed && Inside`) are the drawn
button's.

Data in `plaqueFace`:

- `Pictures`: one picture per state (rest, hover, down, disabled). A state
  with no picture shows the rest picture.
- `Over`: composite over the art below; else an opaque copy. The picture is
  placed at the button's corner and clipped to its rectangle.
- `Bare`: draw the town shell box when the button has no picture.
- `Captions`: text lines, each centred in its rectangle; `Fit` trims the
  text to the rectangle less six pixels.
- `Ink`: caption colour at rest, hovered and disabled. `plaqueCommandInk` is
  the former `buttonInk`; the generator's `chargenCommandInk` differs only
  in its rest colour.
- `Sink`: the caption displacement while sunk. `plaqueSink` is the former
  `buttonTextOffset`.

The picture is the disabled picture when the button is disabled and the
face has one, else the down picture while sunk, else the hover picture
while hovered, else rest. A disabled plaque's captions never sink. The
four-command tavern passes its press only for an enabled button, as its
painter drew its bitmap only then. `pkg/ui/buttonfeedback.go` is removed.

Kind: press latch. Its one builder is `latch.Latch` (`buttonLatch`).

## As built

### Former painters

| former painter | built now |
|---|---|
| `drawChargenCommands` | `chargenCommandButtons`, three `pushButton` values drawn by `drawPushButton` |
| `detailedStatButton` (picked a picture) | `detailedStatButton` returns the stat `pushButton` |
| `ComposeTownSurface` command plaques | `townSurfaceButton` |
| `ComposeShopScreen` captions and pressed pictures | `shopCommandButton` |
| `saveDialogPaint` fontless fallback | `drawPushButton` with no font, label queued for the debug font |

The generator's commands and the tavern's original three-plaque panel use
the same off/on pair face (`plaquePair`). The tavern's four-command panel
and the shop use a face with a down picture only, composited over the
plaques their upper picture carries.

The shop's plaques now draw face and captions in one call where the
pressed pictures drew before; the captions moved ahead of the arrows, the
interior scene, the cells and the book, none of which reaches the command
rectangles (x 483..623, y 15..212).

### Former latches

| former latch | built now |
|---|---|
| `App.townSurfacePress` (a control) | `buttonLatch` keyed by `townSurfaceLatchID`; `townSurfacePressed` reads it back |
| `App.townTipPress` (owner and armed flag) | `buttonLatch` plus `townTipPressOwner`, the tip it was taken on |
| `townRoomPointerState.tipPress` | `buttonLatch` plus `tipOwner` |
| `townRoomPointerState.shopPress` (the drag origin) | `buttonLatch`; `App.shopButtonPress` holds the shop command buttons |

Behaviour kept: a new town surface press replaces the held one (the latch
is cleared first); the tip latch is dropped by a lost focus, Escape, Enter,
a right press, a cutscene or a changed tip; the town surface latch by a
lost focus or a left room. The shop's command buttons no longer arm the
drag machine: a press on a live button latches `shopButtonPress`, a release
on the same button activates it, and `clearShopDrag` drops it with the
drag. A lost focus does not drop it: the drag machine kept the press across
a focus loss on the base, and the kit latch keeps that (measured on the
base by `TestTownLatchesRunTheKitCases`).

### Ratchet

`widgetButtonDebt` holds the two second-game town list entries.
`widgetLatchDebt` is empty.

## Proof

- Frame census `TestReleasePlaqueButtonCensus`
  (`pkg/game/plaquebutton_census_release_test.go`): the generator through
  App input (both pages, its tip close button, each command and each stat
  button through hover, press, move out and back, release outside), the
  tavern, the school and the shop through App input (rest, tip close, each
  command plaque, disabled plaques), the tavern, school and shop composed
  with each plaque hovered and pressed, and the Save dialog's five buttons.
  221 pictures per root. The hash lists of the base (`f7f7dc24`) and of
  this branch are equal on EN and RU; so are the 31 pictures of
  `TestReleaseFrameAndButtonCensus`.
- `TestTownLatchesRunTheKitCases` (`pkg/ui/widgetlatch_town_test.go`): the
  town surface button, the room tip close button and the shop command
  button through App input: press and release inside fires once, release
  outside cancels, a move out and back fires once, a lost focus cancels the
  town surface and tip latches and keeps the shop latch, as on the base.

## Open debt

- The shop's command button latch survives a lost focus, unlike every other
  kit latch (MENU-116). Kept as on the base; aligning it is a one-line
  change in `dropWidgetLatches` that needs an owner decision.
- Without the install font the Save dialog's buttons now draw the kit bevel
  with the debug-font label, as the mod screens do, instead of a filled box.
  Only hand-assembled debug apps lack the font.
- A disabled shop plaque's captions no longer sink while it is pressed; the
  base moved them one pixel. App input cannot reach that state: a press
  latches only a live button and nothing changes a button's state while it
  is held. The town plaques already kept a disabled caption still.

- The second game's town list button (`drawTownButton`, `pkg/ui/app.go`) and
  its image twin in `composeTownList` (`pkg/ui/townlist.go`) stay outside
  the kit until composer step 2 (second-game town 1 as data).
- The plaque art loaders (audit row 7) stay: the inn plaque pictures are
  loaded by the generator's and the tavern's own loaders.
