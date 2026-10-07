# Spec — main menu with map picker and NEW GAME

**Owner-directed revision — the defect pass.** The landed build was run by the owner against their
lawful install and came back with four defects: the EXIT button did nothing, the map picker showed no
map names, the list looked incomplete, and the build's run note carried no ready-to-run command. Two
of the fixes are honest reversals of decisions this spec previously recorded, and both are the owner's
call, made on the strength of that report:

- **The map list's order.** The retired `FR-2` ordered every row by its source text alone, which put
  all 28 nameless campaign maps ahead of the 10 named loose ones and filled the picker's first
  screenful with bare numeric file names and nothing else. **FR-2a** replaces it — loose files first,
  archive entries after, each group in the old case-insensitive order — and additionally requires the
  picker to state how long the list is and which part of it is on screen.
- **Button 8.** The retired `FR-9` left every brooch button but NEW GAME unbound, on the stated
  grounds that no per-button command was decoded. That reason did not hold for button 8:
  `MENU-STATE-007` decodes **button 8 → `WM_CLOSE`** by name. **FR-11** binds it, and **FR-9a**
  replaces FR-9's "the other seven do nothing" with "the remaining six".

`FR-2`, `FR-9` and `AC-1` are retired and never reused; `FR-2a`, `FR-9a`, `FR-11`, `AC-1a`, `AC-13`
and `AC-14` carry the contracts that replace and extend them.

The owner added two more items to the same pass while it was running, and neither contradicts a gated
contract, so both are **additions** rather than replacements: **FR-12** — the picker scrolls with the
mouse wheel, which the owner reached for and found dead — with **AC-15**; and a larger **default**
window, 2× the virtual frame and still exactly integer-scaled, which FR-7 already permits ("large
enough to show the whole frame unscaled") and which is therefore settled in `plan.md` DD16 rather than
here.

**Owner-directed revision — the placement markers.** The landed game draws terrain and nothing else.
The diagnostic marker overlays of 0008 (placed structures) and 0009 (placed units) exist and are
tested, but only `cmd/mapview` ever turned them on: this spec put them out of the front-end's scope,
and the shared load path both entry points call was built without them. The owner directed the
reversal and gave the reason. When this engine was built before, the markers were already on screen
when object sprites were first drawn, the sprites landed **two tiles off**, and it was the
marker-versus-sprite disagreement that made it visible immediately. The markers are therefore an
**instrument**, not decoration, and they have to be on screen before object art is. **FR-13**,
**AC-16** and **P-6** carry that contract and the out-of-scope entry that excluded them is withdrawn
below; no identifier is retired, because nothing this spec asserted becomes false.

**Provenance basis.** Two layers here are cleanly separable.

The **app shell** — the window, the screen flow, the map picker, and the reuse of the interactive
viewer — is the project's own design. Its data comes from prior stories: archives and their entries via
`pkg/formats/res` (0001), and each map's decode plus the name recorded in the map's own type-0 metadata
via `pkg/formats/alm` (0003, `ALM-META-010` — **High**, and *empty on most campaign maps*). The terrain
the map screen draws is 0004's pipeline behind 0005's camera.

The **menu asset contract** — which `main.res` bitmaps make up the menu, what the hit mask's byte values
mean, and where each button overlay is placed — **is decoded by the research submodule** and is reproduced
in *I/O examples* below. `MENU-ASSET-001`/`MENU-ASSET-002` (the 18-file asset set; exactly eight buttons),
`MENU-MASK-003`/`MENU-MASK-004` (the mask's palette is an identity grayscale ramp, so the raw 8-bit index
is the semantic; eight hot indices, everything else selects nothing) and
`MENU-GEOM-005`/`MENU-GEOM-006` (two static 8-entry `{x,y,w,h}` placement tables, menu origin `(0,0)`,
overlays drawn 1:1) are all **High**. `MENU-STATE-007` — the hover-vs-pressed overlay roles, a per-button
disable bitfield, and a click dispatcher posting a per-button command — is **Medium**, and this story's
scope is drawn to match: it requires the hover/pressed distinction, requires no disable behavior at all,
and binds exactly one per-button command — **button 8 → `WM_CLOSE`**, the one message id that claim
decodes. The claim's own words are that "message-id meanings beyond WM_CLOSE are undecoded", so the
remaining six buttons stay unbound because what they would do is unknown, not because binding a known
one would be beyond this story.

**NEW GAME is the top-left button of the brooch.** That identification is the **owner's own gameplay
knowledge of their lawful install** — it is not a research claim, is not attributed to one, and asserts
nothing about any decoded byte. What the research contributes is only the geometry: *which* rectangle the
top-left button occupies.

This story adds **no format decoding**. The bitmaps are standard Windows BMP, the archive is 0001's, and
the composition contract is the research's. Greenfield.

## Problem / current behavior

The project has format tools and two terrain viewers but no game front-end. `cmd/againrom` today resolves
the asset root from `-assets`/`AGAINROM_ASSETS`, prints it, and exits; it opens no window. The two viewers
are developer tools driven entirely from the command line: the standalone interactive viewer takes one
map path and opens it directly, and the PNG compositor renders one map to a file. Neither can list what
maps an install contains, and nothing in the tree draws the game's own menu. The camera model, the terrain
pipeline and the windowed run loop already live in library packages rather than inside the viewer command,
so a second front-end does not have to reimplement them — but the *loading* path that turns an asset root
plus a map path into a running viewer exists only inside the standalone viewer command.

The goal is to make `cmd/againrom` the game's entry point: it opens the original 640×480 brooch main menu,
and **NEW GAME** works — every installed map is listed, and the chosen one loads into the interactive map
screen. The menu can be built as the original composes it because the research has decoded that
composition contract; nothing here is re-derived from the game.

## What research and prior stories give

- `pkg/formats/res` (0001) — open `main.res`, `graphics.res` and `scenario.res` and read their entries.
- `pkg/formats/alm` (0003) — decode a map: its dimensions, its tile grid, and the map name recorded in its
  own metadata (`ALM-META-010`). A stock install has 10 loose maps plus 28 embedded in `scenario.res`
  (`ALM-LOC-007`), and most of the campaign ones record an empty name.
- 0004 / 0005 — the terrain tileset pipeline and the windowed viewer's camera, pan, edge-scroll, zoom and
  clamping.
- The research menu contract — `MENU-ASSET-001/002`, `MENU-MASK-003/004`, `MENU-GEOM-005/006` at High and
  `MENU-STATE-007` at Medium — reproduced in *I/O examples*.
- Standard Windows BMP decoding is a public, generic capability. *Which* bitmaps the menu is made of, and
  what each one means, is the decoded contract above — not this repo's own inspection of `main.res`.

## Functional requirements — app shell

- **FR-1** `cmd/againrom -assets <dir>` (or `AGAINROM_ASSETS`) MUST open a resizable window that shows
  exactly one screen at a time and moves between screens on the events below. The asset root comes from
  the flag or the environment and is never compiled in. The archives it requires under that root are
  `main.res` (menu art), `graphics.res` (terrain) and `scenario.res` (campaign maps). All three are
  required at startup, including `graphics.res`, which the menu itself does not read: an install missing a
  piece the app will need should say so before the user picks a map, not after.
- **FR-2a (map picker)** The picker MUST list every loose `*.alm` file in the asset root, in any letter
  case, plus every `.alm` entry inside `scenario.res`, as **one** list in a stable order: **every loose
  row before every archive row**, each of the two groups ordered case-insensitively by the source text
  each row displays, ties inside a group broken by comparing that same text case-sensitively. Each row
  MUST show its source — the file name, or the entry name inside the archive — together with the name the
  map records in its own metadata. A loose file and an archive entry whose source texts are equal are
  **two** rows, not one, the loose one first; nothing is deduplicated and nothing is emitted twice. That
  recorded name is **empty on most campaign maps**: an entry with no recorded name MUST still be listed
  and MUST still be choosable, identified by its source alone. A map whose metadata cannot be read MUST
  stay listed, be visibly distinguished from the choosable rows, and MUST NOT be choosable. The loose
  scan covers the asset root itself, not its subdirectories, and a loose row's source text is the bare
  file name.
  The picker MUST also state on screen **how many rows the list holds and which range of them is
  currently visible**, so a list longer than the screen is never mistaken for the whole list. That range
  MUST be the same range the picker draws and hit-tests, not a second count computed beside it.
- **FR-3** The picker MUST support click with the primary mouse button and Up/Down + Enter selection, and
  Esc to leave it for the screen it was entered from — in this story, the menu. Choosing a listed,
  choosable map MUST switch to the interactive map screen — terrain as 0004/0005 draw it — and Esc there
  MUST return to the picker, never exit the program. A chosen map that passes the metadata read but then
  fails to decode fully MUST report that failure, leave the user in the picker with that row now marked
  unusable, and MUST NOT crash or exit the program.
- **FR-4a (one viewer behind two entry points)** The interactive map screen reached through the front-end
  MUST offer the same observable camera behavior as the standalone viewer (0005) — key pan, edge-scroll,
  wheel zoom about the cursor, clamping at the map edges — and the two MUST NOT diverge: any later change
  to that camera behavior observable in one MUST be observable in the other. Adding this front-end MUST
  NOT change the standalone viewer's own observable behavior: its flag set, its `-check` summary text and
  its Esc-closes-the-window behavior stay exactly as they are.
- **FR-5** `cmd/againrom -assets <dir> -check` MUST load the map inventory and the menu assets headlessly,
  print a one-line summary — how many rows the map list holds, and for how many of the eight buttons the
  mask carries a non-empty hit region — and exit 0 without opening a window. The wording of that line is not
  pinned; both counts MUST be unambiguous in it. An asset failure under FR-10 applies in this mode too and
  overrides the exit-0 case; a button that is present but has **no** mask region is not such a failure —
  reporting it is precisely what this mode is for.

## Functional requirements — main-menu screen

- **FR-7 (the menu is the root screen)** The application MUST open on the main-menu screen, in a window
  large enough to show the whole frame unscaled. The menu and the picker are composed in a **640×480
  virtual frame**, and that finished frame — never its individual parts — is what the window shows: scaled
  by one uniform factor, centered, letterboxed, with no aspect distortion and nothing cropped. A window
  position MUST map to at most one frame pixel, and a position that lands in the letterbox maps to none.
  The map screen is the exception: it fills the window as the standalone viewer does, because its camera
  is sized by the window (FR-4a). Esc at the menu MUST exit the program; Esc on every other screen MUST
  return to the screen it was entered from and MUST NOT exit.
- **FR-8 (button selection and overlays)** Within that frame the base brooch bitmap is drawn first, over
  the whole 640×480. The cursor position MUST select **at most one** of the eight buttons, decided by the
  8-bit index the hit mask carries at the frame pixel under the cursor: the eight hot indices select
  buttons 1…8, and **every** other value — index 0, the anti-aliased edge ramp, any stray value — selects
  none, as does any cursor position that maps to no frame pixel. The selected button's **hover** overlay
  MUST be composited over the base at that button's normal rectangle — the table's hover entry — **1:1 in
  frame pixels** and as an **opaque** rectangle: the `(w,h)` there is the bitmap's own size, so nothing is
  resampled inside the frame, and the bitmaps carry no alpha and no decoded transparency rule.
  While the primary mouse button is held down, the brooch button it was pressed on stays latched and no
  other button may be selected until it is released: with the pointer still over the latched button its
  **pressed** overlay is composited at its pressed rectangle instead of the hover one; with the pointer
  held down but moved off it, **no** overlay is drawn at all. A press that selects no button latches
  nothing and changes nothing — hovering keeps tracking normally. Never two overlays at once; with no
  button selected the frame MUST equal the base bitmap.
- **FR-9a (NEW GAME)** Activating the **top-left** button of the brooch MUST open the map picker
  (FR-2a/FR-3). Activation is a press and a release of the primary mouse button both on that button;
  releasing anywhere else cancels it and changes nothing. The button MUST be identified by its **placement
  rectangle**, and MUST NOT be identified by a mask index value. The normative rectangle is the hover entry
  `112, 64, 212, 136` — the half-open frame area `(112,64)…(324,200)`. *That is the topmost entry of the
  left column, the two columns being told apart by x: the left column's entries start at x < 320, the right
  column's at x ≥ 320. Bare "topmost" does not identify it — the right column's top entry begins four
  pixels higher.* The remaining **six** buttons — every button but NEW GAME and EXIT (FR-11) — MUST
  highlight and press exactly as FR-8 requires and MUST do nothing else; a click on one of them is
  consumed and has no effect.
- **FR-11 (EXIT)** Activating the brooch's **EXIT** button MUST exit the program, with the same
  activation rule FR-9a defines: a press and a release of the primary mouse button both on that button,
  and releasing anywhere else cancels it and changes nothing. EXIT is **button 8** — the button the
  decoded click dispatcher binds to `WM_CLOSE` — whose placement rectangle is the bottom-right entry of
  the brooch, hover entry `324, 276, 208, 136`. Leaving this way MUST end the program with the same
  successful status Esc at the menu produces; the two are two ways out of one program, not two
  behaviors.
- **FR-12 (wheel scrolling in the picker)** The picker MUST also respond to the mouse wheel: rolling it
  away from the user moves towards the start of the list and rolling it towards the user moves towards
  the end, clamping at both ends and changing nothing beyond what row is selected and which rows are
  shown. It MUST NOT choose a row, and a wheel roll on any screen other than the picker MUST NOT reach
  the picker at all. This is in addition to FR-3's controls, not in place of any of them.
- **FR-13 (the placement markers on the game's map screen)** The map screen reached through the
  front-end MUST draw the 0008 structure and 0009 unit placement markers over the terrain of every map
  it opens. `cmd/againrom` MUST expose exactly one boolean flag, `-markers`, that turns **both** on or
  off together; it MUST default to **on**, and `-markers=false` MUST leave that screen exactly as it was
  before this revision. The name MUST NOT be `-objects`, `-units`, `-statics` or `-staticmarkers` — those four are
  the developer tools' and the object-art story's, and reusing one here would silently change what it
  means. Each marker's position MUST be derived from **its own placement record alone** — that record's
  stored anchor, and the centre of the cell it names — and from no other input: not a class field, not a
  sprite anchor, not a frame size, and never through a function shared with art placement, because a
  shared wrong anchor moves marker and art together and the screen then looks consistent while being
  wrong. The default is on **because nothing else is on screen yet**; it is expected to be reconsidered
  when real object art ships, and that is to be a recorded decision rather than a silent flip. The
  standalone viewer's own flags, summary and behaviour are unchanged by this (FR-4a).
- **FR-10 (incomplete or inconsistent assets)** Any of the following MUST be reported on the error stream
  before any window opens, with a non-zero exit status, in both the windowed and the `-check` mode: a
  required archive that cannot be opened; a menu asset that is absent; a base screen or hit mask whose
  pixel dimensions are not 640×480; a hover or pressed overlay bitmap whose pixel dimensions differ from
  its own row of the corresponding placement table. The application MUST NOT show a menu screen whose hit
  regions or overlays are only partly defined.

## Acceptance criteria (synthetic assets, no game files)

| ID | Level | GIVEN | WHEN | THEN |
|---|---|---|---|---|
| AC-1a | unit | a synthetic loose-file list and a synthetic archive of `.alm` entries — including maps that record an empty name, one source text present in both sources, entries differing only by letter case, and maps whose metadata cannot be read | the picker list is built | one list in the stable order FR-2a defines — every loose row before every archive row, each group ordered case-insensitively by source text — each row showing its source and its recorded name, each input contributing exactly one row; equal source texts from the two sources yield two distinct rows, the loose one first; an entry with an empty recorded name is still listed and still choosable; unreadable entries are distinguished and cannot be chosen |
| AC-2 | unit | window sizes wider than, taller than, and exactly 4:3 relative to 640×480 | the frame placement is computed | the 640×480 frame is centered and letterboxed with no stretching; window↔frame coordinates round-trip; a cursor outside the frame maps to no frame position |
| AC-3a | unit | the screen flow with a menu, a picker and a map screen | selection and Esc events, including a choice whose metadata read succeeded but whose full load fails | NEW GAME moves menu→picker; a loadable choice moves picker→map; a failing load reports and stays in the picker with that row now unusable, without exiting; Esc moves map→picker and picker→menu; Esc at the menu exits; no other screen exits |
| AC-4 | unit | a picker holding N entries | keyboard and mouse events | the selection moves and clamps at both ends; Enter and click choose only choosable entries; Esc reports leaving |
| AC-5a | unit | the standalone viewer's flag set and a `-check` run over a synthetic archive and map | invoked as it was before this story | its flags, its summary text and its exit behavior are character-for-character unchanged, and the front-end's map screen exposes the same camera operations |
| AC-6 | manual | a lawful game install | `againrom -assets <dir> -check` | prints the map count — 38 on a stock install, 10 loose plus 28 in `scenario.res` — and reports 8 of the 8 buttons as having a mask region, then exits 0 without opening a window; recorded in verification.md |
| AC-7a | manual | a lawful game install | run `againrom`, choose NEW GAME, pick a map | the menu appears; NEW GAME opens the picker, which lists the installed maps; the chosen map loads into the interactive map screen; Esc unwinds map→picker→menu and then exits; recorded in verification.md |
| AC-9 | unit | a synthetic 640×480 index mask carrying each of the eight hot indices, index 0, edge-ramp values `0x10`…`0x1e`, and assorted other values | the cursor is sampled at each of them, and at window positions that fall in the letterbox | the eight hot indices select buttons 1…8; every other value selects no button; every position mapping to no frame pixel selects no button |
| AC-10 | unit | a synthetic asset set plus the two placement tables | a button is hovered; then pressed; then the pointer is moved onto a *different* button while still held; then released there; then released back on the original | hovering composites that button's hover bitmap 1:1 at its normal rectangle; pressing composites its pressed bitmap at its pressed rectangle; moved off while held, the frame equals the base bitmap and the other button does not light; releasing off the latched button activates nothing; releasing on it activates it; never two overlays at once |
| AC-11 | manual | a lawful game install, on the menu screen | the cursor rests on the **top-left** brooch button, then clicks it; then each of the other seven is hovered in turn | the highlight appears over the **top-left** rectangle `(112,64)…(324,200)` and *not* over the bottom-left one, and the click opens the map picker; each other button lights its own rectangle and no other; recorded in verification.md |
| AC-12 | unit (error case) | a menu asset set with an entry missing; one whose base screen or mask is not 640×480; one whose overlay bitmap disagrees with its own placement-table row | startup, in both the windowed and the `-check` mode | each failure is reported on the error stream, no window opens, the exit status is non-zero, and no partly-defined menu screen is ever shown |
| AC-13 | unit | a picker holding more rows than fit on screen, and an empty one | the header line is rendered, before and after scrolling | it states the total number of rows and the 1-based range currently visible; that range is exactly the range the draw path iterates and the hit test derives from, and it tracks scrolling; an empty list says so rather than reporting a range |
| AC-14 | manual | a lawful game install, on the menu screen | the **bottom-right** brooch button is pressed and released on itself; then, on a fresh run, pressed on it and released away from it | the first exits the program with a zero status; the second does not exit and changes nothing; recorded in verification.md |
| AC-15 | unit | a picker holding more rows than fit on screen | the wheel is rolled away from the user and towards it, including past both ends | the selection and the visible window move towards the start and towards the end respectively, clamp at both ends, and no row is chosen; the same roll on the menu and on the map screen leaves the picker untouched |
| AC-16 | unit | a synthetic map carrying placed type-4 and type-6 records, plus a synthetic install | the map is loaded through the shared load path with the markers requested and with them not requested, and the front-end is built from the flag's default and from its off value | requested, both overlays are on and each holds exactly one anchor cell per placement record, each cell being that record's own stored anchor and nothing else; not requested, both are off; the front-end defaults to on and its off value reaches the loaded map screen; the standalone viewer's flags, summary and error text are unchanged |

## Derived properties

- **P-1** (invariant) The picker's selection and leave logic is a pure function of `(entries, events)`, and
  the screen transitions are deterministic.
- **P-2** (invariant) For any window at least 1×1, the whole virtual `(0,0)–(640,480)` frame maps inside
  the window and the mapping is invertible on its image.
- **P-3** (negative-invariant) For any mask index outside the eight hot values, and for any cursor position
  outside the 640×480 frame, no button becomes selected and no overlay is composited — the screen equals
  the base bitmap.
- **P-4** (negative-invariant) For any incomplete or dimension-inconsistent menu asset set, no menu screen
  is shown and no partly-defined hit map or overlay set is ever used.
- **P-5** (invariant) At most one button overlay is composited in any frame, and it is drawn at that
  button's own rectangle for the state it is in.
- **P-6** (invariant) A marker's cell is a pure function of its own placement record's stored anchor and
  of nothing else. No class, sprite or frame input can reach it, so a wrong art placement cannot move
  the marker along with it — which is the whole of what makes the two disagree visibly.

## I/O examples — the decoded menu contract

**Asset set** — the 18 entries of the `graphics/mainmenu/` subtree of `main.res`
(`MENU-ASSET-001`, `MENU-ASSET-002`):

| Entry | Dimensions | Bpp | Role |
|---|---|---|---|
| `menu_.bmp` | 640×480 | 24 | base brooch screen, drawn first |
| `menumask.bmp` | 640×480 | 8 | hit mask — read by raw palette **index**, never by color |
| `button1.bmp` … `button8.bmp` | per the table below | 24 | per-button hover overlay |
| `button1p.bmp` … `button8p.bmp` | per the table below | 24 | per-button pressed overlay |

**Hit mask → button** (`MENU-MASK-003`, `MENU-MASK-004`) — one byte per pixel of the 640×480 frame, menu
origin `(0,0)`:

| index | `0x80` | `0x90` | `0xa0` | `0xb0` | `0xc0` | `0xd0` | `0xe0` | `0xf0` |
|---|---|---|---|---|---|---|---|---|
| button | 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 |

Every other value selects **no** button: index 0 (background), the anti-aliased edge ramp
`0x10, 0x12, … 0x1e`, and every stray value. The mask's 256-entry palette is an identity grayscale ramp
and carries no meaning of its own.

**Overlay placement** (`MENU-GEOM-005`, `MENU-GEOM-006`) — `{x, y, w, h}` in the 640×480 frame, drawn 1:1
in frame pixels. Each `(w,h)` is the pixel size of *that* bitmap, so a button's hover and pressed rectangles
legitimately differ in both position and size; the two are separate images, not one image drawn twice:

| btn | hover `(x, y, w, h)` | pressed `(x, y, w, h)` |
|---:|---|---|
| 1 | 112, 64, 212, 136 | 116, 64, 208, 138 |
| 2 | 84, 88, 236, 148 | 88, 88, 236, 152 |
| 3 | 84, 236, 236, 152 | 88, 236, 236, 152 |
| 4 | 112, 276, 212, 136 | 116, 272, 208, 140 |
| 5 | 320, 60, 212, 140 | 320, 64, 212, 140 |
| 6 | 324, 88, 236, 148 | 324, 88, 232, 152 |
| 7 | 324, 236, 236, 152 | 324, 236, 236, 152 |
| 8 | 324, 276, 208, 136 | 320, 272, 212, 140 |

The layout is two columns of four — the left column's entries start at x < 320, the right column's at
x ≥ 320 — each column running top to bottom. These are **placement** rectangles, not hit regions: they are
bounding boxes of irregular brooch shapes, they overlap each other, and only the mask decides what the
cursor is over.

The **top-left** entry — the topmost of the left column — is the hover rectangle `112, 64, 212, 136`, the
half-open frame area `(112,64)…(324,200)`. That is the button FR-9a calls NEW GAME. Note that it is *not*
the entry with the smallest y: the right column's top entry starts at y = 60, four pixels higher, so a
rule that ranks by y alone picks the wrong button.

Entry **8** — `324, 276, 208, 136`, the bottom of the right column — is the button FR-11 calls EXIT.
`MENU-STATE-007` names it by number: the click dispatcher posts `WM_CLOSE` for button 8, and nothing for
any other button that the research was able to read.

## Constraints

- Asset paths come from `-assets`/`AGAINROM_ASSETS`; no install path is compiled in; no game data, raw or
  converted, enters the repo. Tests are built from synthetic archives, synthetic 8-bit masks and synthetic
  BMPs, never from a game file.
- The menu bitmaps are standard Windows BMP. The hit mask MUST be consumed as **raw 8-bit indices**: its
  palette is an identity grayscale ramp (`MENU-MASK-003`), so any reading that goes through palette colors
  is wrong by construction, not merely inefficient.
- **Why loose files come first (FR-2a).** On a stock install the 28 campaign maps inside `scenario.res`
  are named `10.alm`, `100.alm`, `101.alm` … `91.alm` and almost none records a name of its own, while
  the 10 loose files are named `Beast.ALM` … `Waters.alm` and every one of them does. Digits sort before
  letters, so ordering the whole list by source text alone put all 28 nameless rows ahead of all 10 named
  ones: the picker's first screenful was 25 bare numbers, with no name on any row and nothing on screen
  saying the list continued. Grouping by source is what puts the named maps where they are seen; the
  visible-range statement is what makes the rest discoverable. Both halves are required — either one
  alone still leaves a user able to conclude an install is short.
- **Why FR-9a is anchored to geometry and not to a mask index.** The research labels the mask's stored row
  order as *inferred* — concluded from each button's normal rectangle bracketing its mask region, not read
  directly. And the layout is nearly symmetric about the horizontal midline, so a vertically flipped read
  maps 1↔4, 2↔3, 5↔8 and 6↔7 onto each other within a few pixels and would very nearly satisfy that same
  bracketing check. An index-anchored requirement is therefore one the layout evidence cannot falsify:
  under one row order the visually top-left button answers to index `0x80`, under the other to `0xb0`.

  | Option | NEW GAME bound to | Observable trade-off |
  |---|---|---|
  | A | the mask index literal `0x80` | Correct only if the inferred row order is right; if it is not, the *bottom*-left button silently becomes NEW GAME and no layout check would have caught it. |
  | B *(chosen)* | the top-left entry of the normal-placement table | Independent of row order; a wrong row order is falsified the first time anyone hovers the top-left button (AC-11). |
  | C | nothing — defer NEW GAME until the row order is read directly | Leaves the front-end with no working button although the decoded contract is already sufficient to build one. |

- **Why FR-11 is anchored to a button *number* and not to a rectangle.** The reasoning above does not
  carry across, because the two identifications come from different places. "The top-left button is NEW
  GAME" is the owner's gameplay knowledge — a statement about a position on screen — so it is anchored to
  a position. "Button 8 quits" is decoded — `MENU-STATE-007` names the number, not a position — so it is
  anchored to that number, and the bottom-right rectangle is a *consequence* of it (`MENU-GEOM-005` puts
  entry 8 at `324, 276, 208, 136`), stated so a human can find the button, not so the code can. Deriving
  EXIT geometrically instead would invent a placement claim the research does not make. The exposure this
  leaves is the same one R-1 already carries and is the reason AC-14 exists: if the mask's inferred row
  order is wrong, a vertically flipped read maps 5↔8, and the button that quits would be the *top*-right
  one.
- The eight hot indices, the two placement tables and the 1:1 draw rule are High-confidence decoded facts
  and are the frozen contract the acceptance criteria pin. The hover-versus-pressed *roles* and the
  button-8 → `WM_CLOSE` binding rest on `MENU-STATE-007` at **Medium**; the spec requires both, and AC-11
  and AC-14 are where that Medium claim and the inferred row order are actually put in front of the game.
- **No transparency rule is decoded.** The research establishes that the overlays are opaque 24-bit
  rectangles and explicitly leaves any blend the original may apply outside what it has established. FR-8
  therefore composites them opaquely; a visible seam around a rectangle would be a discrepancy AC-11
  surfaces, not a licence to invent a colour key.
- Building the picker list reads no more of any map than its own metadata — a stock install holds 38 maps
  and the list is built on every start, including under `-check`.
- The map picker's text is placeholder debug text. Rendering text with the game's own fonts is a later
  story, and no font asset is read here.

## Out of scope

- The **six** remaining buttons' actions — every button but NEW GAME and EXIT. `MENU-STATE-007` decodes
  exactly one message id, button 8 → `WM_CLOSE`, and says in as many words that "message-id meanings
  beyond WM_CLOSE are undecoded". FR-11 binds the one that is decoded; the six stay unbound because what
  they would do is unknown, and guessing is what golden rule 4 forbids. *(An earlier revision of this
  spec bound none of them and gave "no command id is decoded" as the reason. That reason was false for
  button 8; the owner filed the dead EXIT button as a defect and directed the binding.)*
- The per-button disable bitfield — which buttons the original starts with disabled is not decoded, so
  every button here is treated as enabled and nothing suppresses an overlay.
- Keyboard navigation of the menu buttons. The menu is driven by the pointer; Esc is its only key. Keyboard
  selection belongs to the picker (FR-3), not to the brooch.
- Menu music and sound, animated menu transitions, custom cursor art, saved games, the options and
  multiplayer screens, and whatever campaign flow would follow a scenario map.
- Unit and structure **art**. The map screen draws terrain plus the markers FR-13 wires in; the sprites
  those markers stand for belong to a later story. *(An earlier revision kept the markers out of the
  front-end as well. The owner reversed that: they are the instrument that makes a wrong sprite
  placement visible the moment art lands, so they must be on screen before it is.)*
- Re-deriving anything about `main.res`, the BMPs, or the map format: the archive contract is 0001's, the
  map contract is 0003's, and the menu composition contract is the research's.

## Retired identifiers

`FR-4`, `FR-6`, `AC-3`, `AC-5`, `AC-7` and `AC-8` carried contracts this spec no longer states — the
menu-blocking prohibition and the criteria that assumed the picker was the root screen. `FR-2`, `FR-9`
and `AC-1` carried the two contracts the owner's defect report reversed: the source-text-only list order,
and "no per-button command is bound". Those numbers stay retired and are never reused; `FR-2a`, `FR-4a`,
`FR-7`, `FR-8`, `FR-9a`, `FR-10`, `FR-11`, `AC-1a`, `AC-3a`, `AC-5a`, `AC-7a` and `AC-9`…`AC-14` carry
the contracts that replaced them.

## Gate check

FR-1 → AC-2, AC-6, AC-7a, P-2 · FR-2a → AC-1a, AC-13, AC-7a · FR-3 → AC-3a, AC-4, AC-7a, P-1 ·
FR-4a → AC-5a, AC-7a · FR-5 → AC-6, AC-12 · FR-7 → AC-2, AC-3a, AC-7a, AC-9, AC-11, P-2 ·
FR-8 → AC-9, AC-10, AC-11, P-3, P-5 · FR-9a → AC-3a, AC-10, AC-11 · FR-10 → AC-12, P-4 ·
FR-11 → AC-14, and its dispatch half in AC-10 (the release-on/release-off rule it shares with FR-9a) ·
FR-12 → AC-15, AC-7a · FR-13 → AC-16, P-6.

FR-7's, FR-8's and FR-9a's *on-screen* half is carried by **AC-11**, the manual criterion: the unit
criteria pin the index mapping, the rectangles and the compositing rules against synthetic assets, none of
which can tell whether the real mask's rows are stored the way the research inferred. AC-11 is therefore
load-bearing, not a nicety.

## Verification mapping

AC-1a…AC-5a, AC-9, AC-10, AC-12, AC-13, AC-15 and AC-16, plus P-1…P-5 and P-6: unit criteria on synthetic assets —
CI-automatable, no game install needed. AC-6, AC-7a, AC-11 and AC-14: developer-run against a lawful
install, recorded in verification.md; no game bytes are committed. Until AC-11 and AC-14 run, the
hover/pressed roles, the top-left-button↔button-1 correspondence and the bottom-right-button↔button-8
correspondence rest on a Medium claim and an inferred row order.
