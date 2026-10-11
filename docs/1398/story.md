# One screen art loader

## Intent

Every screen that drew install pictures from its own Go loader, except the
six files `ScreenArtDebt` lists (Open debt), now names its art as data, a
manifest of entries, and gets the decoded pictures from one loader,
`pkg/screenart`. One install picture several screens draw is read and
validated once, by one rule. Nothing a player sees or a save holds changes. No
CHANGELOG line: nothing is player-visible. Base: public main `a65abc1d`, merged
with `f64d5b17`; knowledge pin unchanged (k238).

## Authority

Owner decision 5 on the architecture audit `pipeline/reviews/arch-audit-34ae6dac.md`
(row 7; `arch-audit-db226edb.md`, row 6): one composer scene runtime, then one
screen art loader, before composer step 2. Owner rule "one builder per kind":
the screen art loader is the one builder of a screen's pictures. The present
behaviour is the specification; the town square, town room and generator
traces and the screens' release witnesses are the oracle. No ROM1 behaviour
changes, so no claim is cited anew. One divergence row is added, DIV-2932, for
the second game's documents panel art, which the engine already read at the
first game's addresses; DIV-2933..2939 are unused.

## As built

### The loader

`pkg/screenart` imports only the standard library and holds no game fact
(`TestLoaderHoldsNoGameFact`). An `Entry` names a picture's install key, its
format (`picture`, `series`, `sprites`, `frame`, `mask`), its size (`size`) or
size envelope (`max-size`), its key colour (`transparent: "black"`), whether
it is required, and its use on the screen: one or two indices (`index`, by
count and base or by listed values), a placement inside a rectangle (`at`,
`inside`), a crop, a mask's required indices and a frame of a sheet. A `bound`
entry names a picture the screen learns at run time (the documents page, the
world map's mission marker); `Library.Bind` loads it under the entry's rule.

`screenart.Library` is one install's picture memo. A key is read once per
format under its rule (size, envelope, key colour); a second entry naming the
key in the same format under another rule is an error naming both screens.
The same key read in two formats (a picture and a mask) is two reads. Sheet
size and count and a mask's required indices are checked per entry, not
stored with the rule. `Archives` carries one
library per install (`OpenArchives`), handed to every loader through
`Archives.artSource()`. The world map keeps a private counted library over
the same source for its read-count tests (`pkg/game/worldmap.go`). The
install source, `screenArtSource` in
`pkg/game/screenart.go`, decodes a 24-bit bitmap, an 8-bit mask and a `.256`
or `.16a` sheet through the existing decoders; `pkg/formats` is unchanged.

### Where each manifest lives

| Screen | Manifest | Why there |
|---|---|---|
| town square | `pkg/game/towns/rom1.json`, top-level scene `art` | the composer description already held it |
| tavern, school, shop, world map (gates) | `pkg/game/towns/rom1.json`, the room's new `art` (page art); its scene keeps scene `art` | the room is the screen; its page and its scene are one description |
| generator | `pkg/game/generators/*.json`, unchanged; `generatorArt` turns its fields into entries | the generator description already names every picture with its placement |
| bottom HUD, command panel, character pane, pane corners, pane filler, tip panel, documents panel | `pkg/game/screens/rom1.json` and `screens/rom2.json` (new) | no composer or generator description names these screens; `Edition.Screens` selects the file |

The second game's screens have their own file, `screens/rom2.json`, cited by
DIV-2371, `R2-ASSET-081`, `R2-ENGINE-313` and DIV-2932; no ROM1 path is
branched. The second game's rooms read `towns/rom1.json` because its edition
already names it (`Rooms: "rom1"`), as it did for scene art before this story.

### Loader files replaced

Each holds no path, size check or decode call now; it maps a manifest's
entries to the screen's art struct.

- `bottomhud.go`, `commandpanelart.go`, `characterpane.go`, `tipart.go`,
  `docsart.go`: read `screens/<edition>.json`.
- `towntavernart.go`, `townschoolart.go`, `shopart.go`: read the room's page
  art through `roomPageArt` (`roompage.go`).
- `worldmap.go`: reads the gates room's page art; the mission marker is a
  bound entry.
- `chargenassets.go`: its picture helpers (`chargenArt`, `chargenPane`,
  `chargenSize`, `chargenMaskOf` and the rest) are gone; `LoadChargenAssets`
  loads `generatorArt(l)`.
- `townsquare.go`, `roompage.go`: `town.LoadArt` reads through the install's
  library (`town.Loader` is `screenart.Source`).
- `heroart.go` loads unit body sheets, the unit loader's kind, and stays out of
  scope; `missionart.go` holds no path.

`readInstallBMP`, `readInstallMask`, `decodeInstallBMP` (formerly
`readChargenBMP`, `readChargenMask`, `chargenRGBA`) and `keyBlack` moved into
`pkg/game/screenart.go`.

### Shared pictures

`TestShippedScreenArtHoldsEachPictureToOneRule` reads every manifest of both
editions: ROM1 16 screens, 605 pictures, 20 readings of a picture another
screen names; ROM2 15 screens, 544 pictures, 21 readings. Each shared picture
has one rule in every screen. ROM1: `backinv.bmp`, `invarrow1..4.bmp` and
`invframe.bmp` (bottom HUD, shop); `humanbackl.bmp`, `humanbackr.bmp`
(character pane, generator); `inn/ruover.bmp` (school, tavern);
`shopbutton1..4.bmp`, `shopmenu.bmp` (generator, shop, tavern). ROM2 adds
`inn/buttonsarea.bmp` and `inn/button1..3{on,off}.bmp` (generator, tavern).

### Differences between the former loaders

Each is now data of the one loader or one rule of it. On shipped data each
draws what it drew before (the traces and screen witnesses below).

1. The shop checked no size; every shop bitmap was keyed black. Its entries
   that share a key with another screen (`shopmenu`, `shopbutton1..4`,
   `invarrow1..4`, `backinv`, `invframe`) now state that screen's size; the
   shop's own pictures keep no size rule. All keep `transparent: "black"`
   and stay optional: a mis-sized shared picture is absent and named in the
   set's problems, where it was drawn before.
2. Sheets were decoded four ways (the shop coin and the world-map flags
   inline from `.16a`, the tip frame and gems one `.256` frame each). Now a
   `.16a` sheet goes through `effectFrames` and a `.256` sheet through
   `decodeCursor256Frames`; a sheet with a frame of no positive size or with
   no palette refuses the whole sheet, not one frame. Census over en, ru,
   rom2-en and rom2-ru (inn `unit1..64`, hero sheets, world-map `flag`,
   `flag1`, `cross`, `money`, `sparkle`, `lm.256`, `radiob.256`,
   `shopframe.256`): no such frame or sheet.
3. The tavern's command plaque sizes came from `ui.ShopButtonRect`; they are
   the manifest's sizes now, the same 120x52 and 140x46.
4. When several school pictures fail at once, the first error is now the page
   art's, then the scene's.
5. A room without a description is an error (`room <name>: no page art`)
   where a nil loader skipped it.
6. The world map keeps its own library over a counting source, because its
   tests count each address it reads.
7. The tip back in `menupanel.go` keeps its inline 160x240 check; that file is
   debt (below).

`DecoderComposerDebt`: `pkg/game/shopart.go` and `pkg/game/worldmap.go` held
their pixel conversions (the coin, the frame sheet, the flags); those pictures
are manifest entries now and both files left the list. `cmd/schoolcheck` and
`cmd/townsquarecheck` stay.

### The ratchet

`internal/archtest/screenart.go` scans production files under `pkg/`: a string
literal naming a picture (`.bmp`, `.256`, `.16a`, a trailing `.16`) or a call
to a picture decoder or reader is a finding outside the loader (`pkg/screenart`,
`pkg/game/screenart.go`, `pkg/formats`). Seventeen files read pictures of
another kind and are listed with it (cursors, fonts, item and spell icons,
unit, sack and static sheets, projectiles, terrain, the hero figure, campaign
marker names). `ScreenArtDebt` lists six screens, each still holding a finding,
and may only fall.

The install share check (`installshare_test.go`) rereads every picture the
shared library read into a fresh library (`Library.Reread`) and compares them,
so a screen that wrote a shared picture fails the package.

## Proof

- Loader tests (`pkg/screenart/load_test.go`): a wrong size fails with the key
  and both sizes; a missing required picture fails with its key, a missing
  optional one is a problem; a shared picture is read once and both screens
  hold the same decode; a second rule for one picture fails naming both
  screens; every format; `Reread`.
- `TestScreensSharingAPictureReadItOnce`: the tavern and the shop read
  `shopmenu.bmp` once, the shop and the bottom HUD `invarrow1.bmp` once.
- `TestReleaseTownSquareTraceIsUnchanged`, `TestReleaseTownRoomTraceIsUnchanged`,
  `TestReleaseChargenTraceIsUnchanged`: pass on ru and en; no golden moved.
- 130 release tests of the town, tavern, shop, school, generator, world map,
  HUD, character pane, documents and tip families (`go test -list` over
  `pkg/game`): pass on ru and en; one skips for its unset input
  (`AGAINROM_SHOP_CLICK_INPUT`).
- `TestReleaseSecond*` (53): 52 pass on rom2-ru and rom2-en.
  `TestReleaseSecondPhysicalSyntheticCityAppRoute` loads a first-game corpus
  save and fails on both second-game roots; it passes on en.

## Open debt

- `ScreenArtDebt`: the hall of fame (`ending.go`), the options screen
  (`gameoptions.go`), the menu media (`media_menu.go`), the menu panels
  (`menupanel.go`, which reads the tip panel's pictures a second time), and
  the main menu renderer (`pkg/render/menu/menu.go`, `second.go`).
- `cmd/` witnesses are not scanned and keep their own paths.
- The second game's rooms read the first game's room description by the
  edition's choice; a second-game room description is composer step 2.
- DIV-2932: the second game's documents panel art is not evidenced.
