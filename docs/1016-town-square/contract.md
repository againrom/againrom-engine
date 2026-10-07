# 1016 — contract

## What this story settles

The owner reported (2026-08-18, chat): "Собираем экран города вместо нашего черного экрана" — we
are assembling the town screen instead of our own black screen. `drawTown` (`pkg/ui/app.go`) fills
the canvas with a flat colour and paints four hand-drawn rectangles labelled TAVERN, SHOP, SCHOOL and
GATES. The install ships the real picture for this screen and has since story 1006; nothing before
this story reads it.

## What will work after this story

1. **The town square draws the shipped picture.** `graphics/interface/town/townmain.bmp` (640x480,
   24-bit) is the base; `graphics/interface/town/town_add.bmp` (552x92, 24-bit) is composited over
   it. The flat fill and the four rectangles are removed.
2. **A click enters the building under the cursor by raster mask**, reusing the school's own
   mechanism (`TOWN-003`, `TOWN-017`) rather than a second one: `graphics/interface/town/
   townmask.bmp` (640x480, 8-bit indexed) is read one pixel at a time, exact-code, no tolerance.
3. **Three door labels draw at their own measured positions**: `shop_l.bmp` (52x76), `tavern_l.bmp`
   (28x64), `trener_l.bmp` (140x116), all under `graphics/interface/town/`.
4. **The gate region opens the world map.** Wiring only — `townScreen.enterWorldMap()` already
   exists (story 1013).
5. **The statue region opens save/load**, through the existing in-game mini-menu
   (`App.SetSaveSeams`, story 0143 already carries SAVE and LOAD). Wiring only.

## The observable result

`builds/current/`: the town square shows the shipped picture with working doors and a working
statue, in place of the four coloured rectangles. Measured rather than asserted, by a new developer
tool `cmd/townsquarecheck`:

- `town_add.bmp`'s own compositing offset, and the three labels' own placements, each reported as a
  correlation match fraction against the next-best offset over the whole search window;
- the five interactive mask codes' identity, cross-checked two ways — which label sits over which
  code, and the shipped tip text `main/text/tips/town.txt` — against production's own
  `ui.TownSquareControlAt` run over the install's own mask;
- a drive of a real campaign town through `game.NewFrontEnd`/`TownScreen()`: each of the five
  regions is clicked through the production hit test, the four doors are fed to production's own
  `Choose(i)`, and the resulting room is read back from `Header()`.

`pipeline/check-milestone.sh`'s script-gap census is not expected to move — no script opcode is
touched — and is measured in `closure.md`, not assumed.

## Claims this story is built on

| Claim | Confidence | What it supplies |
|---|---|---|
| `TOWN-003` | High | The school room's own raster-mask hit test: one colour code per interactive region, exact match, no tolerance. This story's `TownSquareControlAt` reuses the mechanism rather than inventing a second one. |
| `TOWN-017` | High | The two-panel raster-mask hit test and the per-slot enabled flag, at the school. Same reuse. |

No claim publishes the town square's own mask-code-to-door mapping, its label placements, or its
overlay's compositing offset. All three are measured here from shipped art and shipped text, not
read from the executable — recorded as `DIV-148` (UNKNOWN), not built as if they were research.

## Aspects that apply

| Aspect | Applies |
|---|---|
| Data | Yes — five new archive nodes read for the first time: the base picture, the overlay, the mask, three labels. No format changes. |
| Runtime state | No new persisted or session field. |
| Simulation | No — `pkg/sim` is untouched. |
| Player input | Yes — the raster-mask hit test is the whole of FR-2. |
| AI | No. |
| UI / HUD | Yes — the whole subject. |
| Triggers / scripts | No script opcode is touched. |
| Inventory / equipment | No. |
| Persistence / save-load | No new field; FR-5 opens the pre-existing save/load menu, it does not add a save slot. |
| Campaign / session | No — the four doors and the gate already existed as row choices; this story changes how they are reached, not what they do. |
| Shipped content | Yes — every figure is measured against both preserved installs (`ROM.EXE` and this cosmetic art are shared bytes on both roots, `EXP-0123`). |
| Interactions with existing mechanics | Yes — `Choose(i)` and `Rows()` are unchanged; the raster click feeds the same `Select`/`chooseTown` seam the row-button grid already used, so every headless scenario driving the square keeps working unmodified. |

## Domains touched

**Client** (`pkg/ui/townsquare.go`, `pkg/ui/town.go`, `pkg/ui/app.go`, `pkg/game/townsquareart.go`,
`pkg/game/townscreen.go`, `pkg/game/frontend.go`). One domain, on `1015`'s own precedent for the
same room family.

## The ceiling this contract sets

**Three adversarial passes.** Five behaviours under one contract, no reach into hashed simulation
state, one domain touched.

## Divergence allocation

`DIV-148` through `DIV-152` are reserved.

- `DIV-148` — the mask code-to-door mapping (which colour code is which door, and which is the
  statue) is derived here from art correlation and the shipped tip text, not from the executable.
  **UNKNOWN.**
- `DIV-149` through `DIV-152` — spare, per the standing rule that a limit met at its ceiling is
  raised; returned unused if this story's own findings do not need them.

## Concurrency

A background research task on `EXP-0196` (town decoration: the animated/random sign, door, star,
weathervane and townbird sprites, and the PRNG-placed extras) is open in parallel and is not told
anything this contract measures. Its subject is explicitly out of scope below.

No other implementation lane is open.

## Out of scope

- Every animated or PRNG-placed decoration: `sign/V%.2d.bmp` (x10), `door/T%.2d.bmp` (x9),
  `stars/S%.2d.bmp` (x9), `fluger/F%.2d.bmp` (x8), the `townbirds/` sheets, and the PRNG-placed
  horse/baba/dervish extras. `EXP-0196`'s own subject; none of it is guessed here.
- The per-door hint text `squareRows()` currently prints ("N mission(s) available"), the room
  header and the gold/party footer line, all cut from the composed picture (spec `SC-1`..`SC-3`).
  `Rows()`, `Header()` and `Footer()` still answer every one of them for a caller not drawing this
  art.
- The tavern's, shop's and school's own room interiors — untouched by this story, already built by
  earlier stories.
