# 1350 — ROM2 window and menu

## Result

The game window opens a second-game root at the second game's own main menu,
drawn from its installed menu bitmaps at their own sizes. New Game opens
mission 10 in the window: terrain, objects, units, the camera, selecting a unit
and ordering it to move. The starter selects a second-game root and launches
the game into it. A first-game root builds the same menu, screens and state as
before.

## Authority

Pin k175. No `R2-*` claim covers the menu or the mission screen, so the menu
layout is format work on the installed archives, recorded in `DIV-2368`. Where
second-game behaviour is unread the first game's behaviour runs and a row says
so: `DIV-2368` to `DIV-2371` in `docs/divergences/rom2.md`. The script and
campaign gaps remain `DIV-2354` to `DIV-2357`.

## Derived menu facts

Instrument: the installed `main.res` of `rom2-ru` and `rom2-en` read with
`pkg/formats/res`; each bitmap compared with the base bitmap.

- Base `menu_.bmp` and mask `menumask.bmp` are 640x480; the mask carries the
  eight hot indices 0x80 to 0xf0, in the first game's order. Each hot region's
  bounding box lies inside its overlay rectangle.
- Hover and pressed bitmaps of button n have one size: 104x96, 108x76, 96x88,
  100x100, 88x100, 84x88, 96x84, 72x80.
- Each hover bitmap equals the base over 44 to 70 percent of its pixels at one
  offset: (204,52), (124,156), (124,252), (208,340), (340,52), (424,152),
  (412,260), (344,348). Pressed bitmaps share those offsets by a border-ring
  match.
- `text1` to `text8` are 180x80 captions (New Game, Multi Player, Cut Scenes,
  Credits, Load Game, Server, Hall of Fame, Exit). Each matches the base over
  51 to 79 percent of its pixels at (232,200).
- Both builds hold identical bitmap sizes, mask statistics and offsets.

## As built

- `pkg/render/menu`: `LoadSecond` validates the eighteen bitmaps at the second
  game's sizes plus eight captions. `Assets` keeps the first game's rectangle
  tables when built by `Load`; `Overlay` and `Compose` use the second game's
  rectangles and draw the selected button's caption over the brooch centre.
- `pkg/game`: `NewFrontEnd` no longer refuses a second-game root and loads the
  menu by game. `LoadMapViewerFor` reads a map by the game's layout; the window's
  picker and mission doors use it. `LoadMapViewer` is `LoadMapViewerFor` for
  the first game.
- Saving: `ExportCurrentSave`, the one producer behind menu Save, quick save, timed and mission-entry autosave, refuses a second-game base with `saving is not available for this game yet`; no file is written.
- Starter: no code change. It already named a second-game profile and passed
  `-base rom2-*`; its tests now cover launching one.

## Proof

Synthetic: `pkg/render/menu` (second-game placement, captions, loss control that
only overlay and caption pixels change, refusals, first-game rectangles kept),
`cmd/starter` (profile named, row, status, launch arguments).

Release, one root each with `AGAINROM_ASSETS=<root> go test -run TestReleaseSecondGame ./pkg/game`:
`TestReleaseSecondGameMenuDrawsInstalledArt`, `...WindowStartsMissionTen`
(New Game reaches the map screen on mission 10 with 30 entities; the world
advances; the camera pans; a selected unit stays still for 60 frames and moves
after a ground click), `...MenuButtonsDoNotCrash`, `...WindowArtLoads`, `...WritesNoSave` (menu Save, quick save, timed and mission-entry autosave write no file and show `saving is not available for this game yet`).
First-game identity: `check-release-tests.sh` on the first-game EN and RU roots.

Witness (uncommitted): `review/story1350/` holds menu and hover frames and the
mission 10 terrain render. The mission screen has no CPU composite; no window
was opened.

## Open debt

- Menu behaviour of Multi Player and Server; second-game movies (`DIV-2369`,
  `DIV-2370`).
- Script operations, campaign driver, mission text, saves (`DIV-2354` to `DIV-2357`).
- Second-game mission screen layout (`DIV-2371`).
- The window shows no message naming the skipped script operations.
