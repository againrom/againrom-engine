# Plan — main menu with map picker and NEW GAME

Reading key: `FR-x`/`AC-x`/`P-x` → `spec.md`. This file fixes the package boundaries, the API contracts,
every design decision, and the success criteria the work is verified against. It is derivable from the
spec alone.

**Revised for the owner-directed defect pass.** The spec's FR-2a and FR-11 replaced two contracts this
plan had already designed against, and FR-12 added one; four design decisions here are **reversals**,
written as such rather than as if they had always read this way:

- **DD19** ordered the map list by source text alone. It now groups loose rows ahead of archive rows.
  The old rule was not wrong about ties — it is where the "loose first" idea already lived — it was
  wrong about which key was primary, and on a stock install that put 28 nameless campaign maps in front
  of the 10 named loose ones.
- **DD13** gave the picker a fixed header string. It now derives the header from `Visible()`, so the
  screen states how long the list is and which part of it is showing.
- **DD14** made the *absence* of any activation entry point other than NEW GAME the guard on "button 8
  is not wired to quit". That guard is withdrawn: **DD31** binds button 8, because `MENU-STATE-007`
  decodes that one command and the reason for leaving it dead was therefore false.
- **DD16** opened the window at exactly 640×480, picking the *smallest* size at which the scale is an
  integer. It now opens at 1280×960 — the *largest* integer multiple that fits a 1080-line desktop.
  What was right about the old decision (integer scale, never a fractional one) is kept; what was
  wrong was preferring "smallest" for its own sake.

**DD32** is new, not a reversal: the wheel arrives in the same input snapshot as every other input and
scrolls the picker.

**Revised again for FR-13 — the placement markers.** One reversal and one addition:

- **DD20** put the shared load path in `pkg/game` and left `cmd/mapview` "its own overlay wiring". That
  half is withdrawn. The overlay wiring moves **into** the shared path, because a front-end that has to
  remember to add the markers is a front-end that did not: the whole of FR-13's defect is that the one
  place map bytes become a running viewer never learned about them, so both callers built the same
  viewer with different overlays and only one of them was the game. What DD20 got right — the summary
  wording, the flag resolution and the `decode <path>` wrapping staying in the command — is unchanged.
- **DD33** is new: how the game exposes the markers, and why the derivation must not be unified with the
  art placement 0017 will add.

Everything else in this plan stands as gated.

## Approach

Build the front-end as **four layers, each testable at its own level**, and put every layer that can be
pure into the stdlib-only render tier so the acceptance criteria run without a window.

1. **`pkg/render/frame` (new, stdlib-only)** — the 640×480 virtual frame: fit a window, centre and
   letterbox at one uniform scale, and map window↔frame coordinates both ways in **exact integer
   arithmetic** (FR-7, AC-2, P-2).
2. **`pkg/render/menu` (new, stdlib-only)** — the menu contract: the eighteen entry names, the two
   8-entry placement tables, the eight hot mask indices, the 24-bpp and 8-bpp BMP readers, the startup
   validation FR-10 requires, the top-left-by-rectangle identification of NEW GAME, the hover/pressed
   latch state machine, and opaque composition of the base plus **at most one** overlay (FR-8, FR-9a,
   FR-10, AC-9, AC-10, AC-12, P-3, P-4, P-5).
3. **`pkg/ui` (extended)** — the screens: a pure picker model (rows, selection, clamping, scrolling,
   hit-testing), a pure screen-flow core (menu ↔ picker ↔ map, Esc semantics, a failing load), and the
   Ebitengine `App` that draws the frame and dispatches one input snapshot (FR-3, FR-7, AC-3a, AC-4,
   P-1). The existing `Viewer` gains an input-snapshot seam so **one** method advances the camera for
   both entry points; its own `Update`, flag-facing behaviour and Esc-terminates stay exactly as they
   are (FR-4a).
4. **`pkg/game` (extended)** — the wiring: resolve the asset root from flag **or** environment, open the
   three required archives, build the map inventory from loose files plus `scenario.res`, own the
   **single** map-bytes→running-viewer path both entry points call, assemble the front-end, and produce
   the headless `-check` summary (FR-1, FR-2a, FR-4a, FR-5, FR-10, AC-1a, AC-6).

`cmd/againrom` becomes a thin front (`-assets`, `-check`, error reporting, exit status).
`cmd/mapview` is re-expressed through the shared load path with **no observable change**:
character-for-character the same summary, the same flags, the same error strings, the same
Esc-closes-the-window (AC-5a). `pkg/formats/alm` gains one additive, metadata-only entry point so the
picker's listing pass decodes strictly less than a full load — which is what makes FR-2a's "metadata
cannot be read" and FR-3's "passes the metadata read but fails to decode fully" two distinguishable
states rather than one. `internal/synth` (new, unpoliced test-helper tier) holds the synthetic archive,
BMP and map builders the acceptance tests need in four different packages.

Two new packages enter the DAG, so `internal/archtest`'s allow-map and `docs/ARCHITECTURE.md` are
updated as part of adding them — the check is fail-closed and an unregistered package fails it
immediately.

## Facts verified during planning (baseline, frozen)

**The shipped tree.**

- `cmd/againrom/main.go` parses `-assets`, falls back to `AGAINROM_ASSETS` via
  `game.ResolveAssetRoot(*assets, os.Getenv("AGAINROM_ASSETS"))`, prints `againrom: asset root = <root>`
  and returns; with no root configured it prints a message to stderr and **returns zero**. It has no
  test file, so no assertion pins that text, that exit status, or the environment fallback.
- `pkg/game` contains only `ResolveAssetRoot(flagValue, envValue string) string` — precedence flag >
  env > `""`, whitespace trimmed — tested in `assetroot_test.go`. The DAG already permits `pkg/game` to
  import any `pkg/*`, and `cmd/againrom` to import any `pkg/*`.
- `cmd/mapview` owns the whole load path inside `load(assetsFlag, graphicsFlag, mapPath string,
  showObjects, showUnits bool) (*ui.Viewer, string, error)`: `resolveArchive` → `res.Open` →
  `terrain.LoadTileset` → `os.ReadFile` → `alm.Open` → title (`m.Name`, else `filepath.Base(mapPath)`) →
  `ui.NewViewer`, then the summary
  `mapview: <title> <W>x<H> cells (<N>), tile slots <L>/<S>` with `, objects N` / `, units M` appended
  under their flags; `run()` afterwards appends `, ` + the water cadence clause. Its two user-visible
  wrappings are `open <archivePath>: <err>` and `decode <mapPath>: <err>`.
- `cmd/mapview/main_test.go` holds **nine** `func Test…`. They pin the summary text, the flag behaviour,
  the cadence clause and the failure paths — **all nine run under `-check`**, so `Viewer.Run()` and
  therefore the Esc-closes-the-window behaviour are exercised by **none** of them. The two failure
  assertions substring-match `open` and `decode` rather than the whole message.
- `pkg/ui.Viewer.Update` reads Ebitengine globals directly: `inpututil.IsKeyJustPressed(KeyEscape)` sets
  the (write-only) `quit` field and returns `ebiten.Termination`; then `advanceAnimation(time.Now())`,
  `panIntent()`, then `ebiten.Wheel()` → `cam.ZoomAbout`. `panIntent` decides "cursor inside the window"
  by comparing `ebiten.CursorPosition()` against **the camera's** `ViewW`/`ViewH`, not against anything
  the engine reports.
- **`Viewer.Update` runs headlessly.** Every Ebitengine global it touches returns a deterministic zero
  value with no graphics context, so `Update` is callable and deterministic in a test today: the cursor
  reads `(0,0)`, which is inside the view, so each call edge-scrolls by `(−PanSpeed, −PanSpeed)`.
  `pkg/ui/viewer_test.go` pins construction, `Layout`, the visible range, sub-cell bounds and the
  animation cadence — and pins **nothing** about `Update`.
- **What is and is not callable headlessly** (verified in this environment against `ebiten v2.9.9` with
  no graphics context): `ebiten.NewImage`, `NewImageFromImage`, `(*Image).WritePixels`,
  `(*Image).DrawImage`, `(*Image).Fill` and `ebitenutil.DebugPrintAt` all **work** — so an `App` holding
  an offscreen image is constructible and drawable in a test; `(*Image).ReadPixels` and `(*Image).At`
  **panic** with `ui: ReadPixels cannot be called before the game starts`. Engine draw calls may
  therefore be exercised, but an `*ebiten.Image`'s pixels may never be read back in a test.
- `pkg/ui` is `package ui` for its own tests, so new unexported types in the package are directly
  testable. No existing `pkg/ui` test references `ebiten.`.
- `pkg/render/camera.Camera` exposes `Pan`, `ZoomAbout`, `Clamp`, `SetZoom`, `ScreenToWorld`,
  `WorldToScreen`, `VisibleTiles`, and the fields `X, Y, Zoom, ViewW, ViewH, Cols, Rows`. It guards its
  own float→int conversions against non-finite values.
- `ui.NewViewer` seeds the camera at `DefaultWindowW×DefaultWindowH` = 1024×768 until `Layout` is
  called, so any comparison between two viewers must call `Layout` identically on both.
- `pkg/render/terrain` is stdlib-only and its allow-map entry is `{}` — it imports **no** intra-module
  package. It exposes `EntrySource` (`ReadFile(name string) ([]byte, error)`), `LoadTileset(src)`,
  `Grid`, `CellSize = 32`, `SlotCount`, and `DecodeBMP8`, an 8-bpp Windows-BMP reader returning
  `*image.Paletted` with **raw indices preserved**, top-down, that **rejects** a non-positive height.
- `pkg/formats/res.Archive.Entries()` returns entries in registry order with `Path` **normalised to
  lower case and `/` separators**; `ReadFile` is case-insensitive, does **not** strip directories, and
  returns an `fs.ErrNotExist`-wrapped error for a missing name.
- `pkg/formats/alm.Open(data)` validates the 20-byte file header, walks ten 20-byte record headers that
  must **tile exactly to EOF** with distinct `typeId`s in `{0..9}`, then decodes type-0 (632 bytes: `W`,
  `H`, angle, scalars and counts, name `@+0x30`, description `@+0x78`), the three `W×H` grids and the
  type-4/5/6 tables. Because the walk requires exact tiling, a **physically truncated** file fails the
  walk itself; the grid check (`payloadSize` vs `2·W·H`) is a *separate*, later failure that a
  well-framed file can reach.
- `alm.Map.Name` is `""` on most campaign maps; `Map.Width`/`Height` are `int`.
- `internal/archtest.allow` is **fail-closed**; `"pkg/ui": {"pkg/render", "pkg/render/"}` already matches
  any package under `pkg/render/` by the trailing-slash prefix form, so a new `pkg/render/<x>` needs its
  **own** entry but needs no edit to `pkg/ui`'s. `externalAllowed` permits Ebitengine (and its
  subpackages, including `ebitenutil`) to `pkg/ui`, packages under `pkg/ui/`, and every `cmd/`.
  `archtest.Load` **skips every package under `internal/`**, so a helper package there needs no
  allow-map entry. Nothing checks `docs/ARCHITECTURE.md` against the allow-map.
- The repo has **no shared test-fixture package**. `buildArchive` exists separately in
  `pkg/formats/res/res_test.go` and `cmd/mapview/main_test.go`; `buildBMP8` in
  `pkg/render/terrain/bmp_test.go`, `cmd/mapview/main_test.go` and `cmd/terraintool/main_test.go`;
  `buildALM` in the two cmd test files. **There is no 24-bpp BMP builder anywhere in the tree.**
- `github.com/hajimehoshi/ebiten/v2/ebitenutil` ships `DebugPrintAt(img, str, x, y)`, drawing an
  embedded bitmap font on a fixed **6×16** glyph cell (with a `+1` px x offset per glyph) for runes
  U+0000–U+00FF. It is part of the already-required `ebiten/v2` module, so using it adds **no** module
  requirement and `internal/notices`' go.mod↔`THIRD_PARTY_NOTICES.md` equality test is unaffected.
- `image/draw` is standard library; `draw.Draw(dst, r, src, sp, draw.Src)` is an opaque copy with no
  blending.
- `scripts/check-no-game-assets.sh` keys on tracked paths and an extension list that includes neither
  `bmp` nor `png`; `.gitignore` does not ignore them either.

**The install — developer-run observation of the owner's lawful install through the shipped
`cmd/restool`, recorded as measurements only.** These are not research claims and carry no `MENU-*`/
`ALM-*` attribution; they are this repo's own reading of an install, of the same kind
`verification.md` records, and they are written here because three of them decide an API the spec
leaves to the plan.

- `main.res` holds exactly eighteen entries under the path prefix **`graphics/mainmenu/`** — as the
  `.res` reader normalises them: `menu_.bmp`, `menumask.bmp`, `button1.bmp`…`button8.bmp`,
  `button1p.bmp`…`button8p.bmp`. The prefix is part of the key; `ReadFile` folds case but not
  directories. *(This is the fact the spec's I/O example implies — "the 18 entries of the
  `graphics/mainmenu/` subtree" — read back off the archive to fix the exact key form.)*
- `scenario.res` holds **31** entries: three `.reg` files and **28** `.alm` entries, all at the archive
  root with bare names (`10.alm`, `100.alm`, …). A flat scan of `Entries()` therefore yields exactly the
  28 campaign maps, and an archive row's source text carries no directory component.
- The asset root holds **10** loose `.alm` files in mixed case (`Beast.ALM`, `Cross.ALM`,
  `Forester.alm`, `Horror.alm`, `Islands.alm`, `Kids.alm`, `Kids2.ALM`, `LuMoir.alm`, `Tomb.ALM`,
  `Waters.alm`), and also a file whose name ends `.LM` rather than `.alm`. 10 + 28 = 38.
- Every menu bitmap's header is a 14-byte file header + a **40-byte** `BITMAPINFOHEADER`, `BI_RGB`,
  planes 1, **positive** height (rows stored bottom-up), `clrUsed = 0`. `menu_.bmp` is 640×480×24 with
  `bfOffBits = 54`; `menumask.bmp` is 640×480×8 with `bfOffBits = 1078` and a palette that begins
  `(0,0,0) (1,1,1) (2,2,2)…` in BGRX order.
- **Every one of the eighteen files carries exactly two bytes beyond the minimum the header implies**
  (e.g. `menu_.bmp`: `54 + 640·3·480 = 921 654`, file size `921 656`). A reader that requires the stream
  length to *equal* the computed size would reject all eighteen; the reader must key on `bfOffBits` and
  require only that enough bytes are present.
- Each overlay bitmap's stored `(width, height)` equals its placement-table `(w, h)` — checked on
  `button2`…`button8`, `button1p`, `button5p`.

## Files to touch

| Path | Intent | Why |
|---|---|---|
| `pkg/render/frame/frame.go` | ADD | `W`/`H` = 640/480, `Placement`, `Fit`, `Valid`, `WindowToFrame`, `FrameToWindow`, `Scale`, `Origin` (DD5, DD26). Stdlib only. |
| `pkg/render/frame/frame_test.go` | ADD | AC-2 / P-2 on synthetic window sizes. |
| `pkg/render/menu/menu.go` | ADD | Entry prefix and the eighteen names, `ButtonCount`, the two placement tables, `ColumnSplit`, `NewGameButton`, `EntrySource`, `Assets`, `Load`, `ButtonAt`, `MaskRegions` (DD2, DD3, DD6, DD7, DD8, DD28, DD29). Stdlib only. |
| `pkg/render/menu/bmp.go` | ADD | The package's own 24-bpp and 8-bpp Windows-BMP readers (DD4, DD7). |
| `pkg/render/menu/state.go` | ADD | `State`, `Selection` (the hover/pressed latch machine), `Overlay`, `Compose` (DD9, DD10). |
| `pkg/render/menu/menu_test.go`, `bmp_test.go`, `state_test.go` | ADD | AC-9, AC-10, AC-12, FR-9a's rule, P-3, P-4, P-5 — synthetic bytes only. |
| `internal/synth/synth.go` | ADD | Shared synthetic fixture builders: `Archive`, `BMP8`, `BMP24`, `ALM`, `MenuArchive` (DD30). Unpoliced tier; imported only from tests. |
| `pkg/formats/alm/alm.go` | MODIFY | Extract the record walk and the type-0 payload decode into unexported helpers; add `Info` and `OpenInfo` over them. `Open`'s behaviour, signature and errors unchanged (DD11). |
| `pkg/formats/alm/info_test.go` | ADD | `OpenInfo` agrees with `Open` where both succeed, and succeeds on the DD11 fixture where `Open` fails. |
| `pkg/ui/update_pin_test.go` | ADD | **Characterization pin, landed before the change** (DD24): today's observable `Viewer.Update` behaviour. |
| `pkg/ui/viewer.go` | MODIFY | Add `Input`, `readInput()`, `(*Viewer).step(in Input, now time.Time)`; `Update` becomes its existing Esc branch plus `step(readInput(), time.Now())` (DD12). |
| `pkg/ui/picker.go` | ADD | `PickerRow`, `Picker`: selection, clamping, scroll window, frame hit-test, `Select`, `Choose`, `SetUnusable`, `Visible`, the layout constants (DD13). |
| `pkg/ui/flow.go` | ADD | `Screen`, `MapLoader`, `flow`: the pure screen state machine, its initial screen, the failing-load path and its message (DD14). |
| `pkg/ui/app.go` | ADD | `App` — the Ebitengine `Game`: the 640×480 offscreen frame, per-screen draw, the `appInput` seam, the bindings, `Run` (DD15, DD16, DD27). |
| `pkg/ui/picker_test.go`, `flow_test.go`, `input_test.go`, `app_test.go` | ADD | AC-3a, AC-4, AC-5a's camera half, P-1. |
| `pkg/game/archives.go` | ADD | `Archives`, `OpenArchives(root)`, `OpenTileset(path)` (DD17, DD20, DD25). |
| `pkg/game/maplist.go` | ADD | `MapEntry`, `NamedReader`, `BuildMapList`, `DirMaps`, `ArchiveMaps` (DD18, DD19). |
| `pkg/game/mapload.go` | ADD | `MapView`, `LoadMapViewer` — the one place map bytes become a running viewer (DD20). |
| `pkg/game/frontend.go` | ADD | `FrontEnd`, `NewFrontEnd(root)`, `CheckLine`, `App()` — assembly, the row conversion and the `-check` summary (DD17, DD21, DD29). |
| `pkg/game/maplist_test.go`, `mapload_test.go`, `frontend_test.go`, `archives_test.go` | ADD | AC-1a, AC-12, FR-5 on synthetic archives and synthetic map bytes. |
| `cmd/againrom/main.go` | MODIFY | `-assets` **and** the `AGAINROM_ASSETS` fallback, `-check`, FR-10 reporting on stderr with a non-zero exit, otherwise open the front-end (DD22). |
| `cmd/againrom/main_test.go` | ADD | The headless `-check` line, both asset-root sources, the error paths, no window (AC-6's synthetic half, AC-12). |
| `cmd/mapview/flagset_test.go` | ADD | The exact flag-name set and the exact `usage` string (DD24, AC-5a). |
| `cmd/mapview/main.go` | MODIFY | `load()` calls `game.OpenTileset` + `game.LoadMapViewer`; the summary literal, both error wrappings, the flag set, `usage`, the package doc and the Esc behaviour are **untouched** (DD20, DD25). |
| `internal/archtest/dag.go` | MODIFY | Register `pkg/render/frame` and `pkg/render/menu`, both `{}`. |
| `docs/ARCHITECTURE.md` | MODIFY | Two tier rows and two DAG rows for the new packages. |

**Not touched:** `pkg/render/terrain` (DD4), `pkg/render/camera`, `pkg/formats/res`, `pkg/sim`,
`pkg/vfs`, `pkg/data`, `pkg/mapload`, `cmd/terraintool` and every other `cmd`. **No existing test file is
edited** — `cmd/mapview/main_test.go`, `pkg/ui/viewer_test.go`, `pkg/ui/overlay_test.go`,
`pkg/ui/unit_overlay_test.go` and `pkg/formats/alm/alm_test.go` are characterization pins for the
brownfield areas and must pass **unmodified**. Work that finds it needs one of them edited has changed
shipped behaviour and must stop rather than adjust the pin. No `go.mod` change, so
`THIRD_PARTY_NOTICES.md` is untouched.

**The defect pass touches only these**, and adds no file and no package:

| Path | Intent | Why |
|---|---|---|
| `pkg/render/menu/menu.go` | MODIFY | Add `ExitButton` (DD31). Nothing else in the package changes. |
| `pkg/render/menu/menu_test.go` | MODIFY | `TestExitButton` — the constant against the decoded contract (SC-17). |
| `pkg/ui/app.go` | MODIFY | `stepMenu` reports exit on `menu.ExitButton` (DD31); the picker header comes from `Picker.HeaderText()` (DD13); `appInput.WheelY` scrolls the picker (DD32); the window opens at 2× (DD16). |
| `pkg/ui/app_test.go` | MODIFY | The EXIT arms, the six-unbound-buttons arm and the wheel arm of `TestAppDispatch` (SC-17, SC-18). |
| `pkg/ui/picker.go` | MODIFY | `HeaderText()` derived from `Visible()` (DD13). |
| `pkg/ui/picker_test.go` | MODIFY | `TestPickerHeaderText` (SC-16). |
| `pkg/ui/flow_test.go` | MODIFY | The FR-9a citation in the no-other-activation subtest; the flow itself is unchanged. |
| `pkg/game/maplist.go` | MODIFY | The sort key becomes `(kindRank, fold(Source), Source)` (DD19). |
| `pkg/game/maplist_test.go` | MODIFY | The hand-derived expected list, re-derived under the new rule (SC-1). |

`pkg/ui/flow.go` is deliberately **not** changed — DD31 explains why EXIT is dispatched in `App`.

**The marker revision touches only these**, and adds no package:

| Path | Intent | Why |
|---|---|---|
| `pkg/game/mapload.go` | MODIFY | `Markers`, `MarkerCells`, and `LoadMapViewer` taking the selection and doing the wiring (DD33). |
| `pkg/game/frontend.go` | MODIFY | `FrontEnd.Markers`, defaulted to both-on in `NewFrontEnd` and passed on every load (DD33). |
| `cmd/againrom/main.go` | MODIFY | `-markers` (default on) parsed into the front-end, and the usage line (DD33). |
| `cmd/mapview/main.go` | MODIFY | `load()` passes its two flags to `LoadMapViewer` instead of converting and setting them itself; its signature, summary, flags, `usage` and error wrappings are **untouched** (DD20 as revised). |
| `pkg/ui/viewer.go` | MODIFY | Two **read-only** accessors for the overlay state, mirroring `Animation()`. No behaviour, geometry or draw-path change. |
| `pkg/game/mapload_test.go`, `pkg/game/frontend_test.go`, `cmd/againrom/main_test.go` | ADD/MODIFY | SC-19. |

`pkg/render/terrain` and the rest of `pkg/ui` are **not** touched: the marker geometry, the clipping and
the draw order are 0008's, 0009's and 0015's contracts and this revision reuses them exactly as they
stand. `cmd/mapview/main_test.go` remains a pin and is not edited.

## Design decisions

- **DD1 — The menu contract lives under `pkg/render/`, and that is forced, not preferred.** `pkg/ui` may
  import `pkg/render` and anything under it, and **nothing else** intra-module. The menu's composition
  contract must therefore live under `pkg/render/` for the screen layer to use it at all. That also puts
  it in a stdlib-only tier, so every FR-8/FR-9a/FR-10 criterion runs with no engine and no window.
  *Rejected:* `pkg/formats/menu` — the formats tier decodes *file containers*; the menu contract is a
  composition over an already-decoded container (0001's `.res`), and `pkg/ui` is forbidden to import
  `pkg/formats/*`, so the screen layer could not reach it. *Rejected:* putting it inside `pkg/ui` — it
  would bury High-confidence decoded facts behind the engine tier and make AC-9/AC-10/AC-12 need a
  windowing context.

- **DD2 — Two new packages, split by what the spec attributes to whom.** `pkg/render/menu` holds the
  material the spec attributes to the research submodule — the entry names, the mask semantics and the
  two placement tables — together with the FR-8/FR-9a/FR-10 behaviour defined **over** them.
  `pkg/render/frame` holds the project's **own** virtual-frame design (a uniform scale, centring,
  letterboxing), for which the spec claims no provenance and which the picker uses as much as the menu
  does. The boundary is "does this arithmetic index a decoded table?", not "is every line in this
  package a decoded fact" — see DD9, which places a spec-defined state machine in `pkg/render/menu` for
  cohesion with the tables it indexes.
  *Rejected:* one package holding both — it would put our own letterbox arithmetic in the same file as
  the decoded placement tables, where a later reader could take the first for the second.

- **DD3 — `menu.Load` takes an entry source, and the eighteen keys are written down here.**
  `type EntrySource interface { ReadFile(name string) ([]byte, error) }`, mirroring `terrain.EntrySource`
  exactly, so `*res.Archive` satisfies it and no formats type crosses into the render tier. The keys are
  the constant prefix **`graphics/mainmenu/`** followed by `menu_.bmp`, `menumask.bmp`,
  `button1.bmp`…`button8.bmp` and `button1p.bmp`…`button8p.bmp` — eighteen in total, exactly as the
  `.res` reader normalises them (see *Facts*). Writing them here is load-bearing: because the test
  `EntrySource` is keyed by whatever the package asks for, **no synthetic test can detect a wrong
  prefix** — only the developer-run criterion could, and only after the fact.
  *Rejected:* passing `*res.Archive` — a DAG violation; *rejected:* passing eighteen `[]byte`s — the
  call site would then own the entry-name contract, which is precisely the decoded fact this package
  exists to hold.

- **DD4 — `pkg/render/menu` carries its own BMP readers; `pkg/render/terrain.DecodeBMP8` is not shared
  and not moved.** The menu needs a 24-bpp reader (base and sixteen overlays → `*image.RGBA`) and an
  8-bpp reader (`*image.Paletted`, raw indices preserved). Sharing would mean either a new
  `pkg/render/bmp` that `pkg/render/terrain` imports — turning a shipped tier documented in AGENTS.md
  and `docs/ARCHITECTURE.md` as importing **no** intra-module package into one that does, and pulling
  0004 into this story's blast radius — or a sibling-to-sibling edge from `pkg/render/menu` to
  `pkg/render/terrain`, coupling the menu to the tileset package. Both are worse than ~70 duplicated
  lines of header validation that are separately tested and whose outputs differ (RGBA vs Paletted).
  **Recorded consequence:** the repo will hold two BMP readers with *different accepted subsets* for
  bitmaps from the same install — terrain's rejects a negative height, the menu's accepts it (DD7). That
  divergence is deliberate and is written down so a later extraction is a decision rather than a
  discovery.
  *Rejected:* exporting a 24-bpp decoder from `pkg/render/terrain` — the terrain package would then own
  menu-art decoding.

- **DD5 — The virtual frame maps in exact integer arithmetic; floats are used only to draw.**
  `frame.W, frame.H = 640, 480`. `Fit(winW, winH)` stores the window size and the scale as an **exact
  rational** `num/den`: `num, den = winW, W` when `winW·H ≤ winH·W` (width-limited), else
  `num, den = winH, H`. A window with a non-positive dimension gives `num = 0` and `Valid() == false`,
  and every mapping then reports "no frame pixel" — there is no division anywhere on that path.
  With `ox = (winW·den − W·num) / (2·den)` and `s = num/den`, frame pixel `fx` owns the half-open window
  interval `[ox + fx·s, ox + (fx+1)·s)`. Multiplying through by `2·den·num` removes every float:

  - `WindowToFrame(x, y) (image.Point, bool)` — `fx = ⌊(2·den·x − den·winW + W·num) / (2·num)⌋` (and the
    same shape on y), `ok` only when both land inside `[0,640)×[0,480)`. Floor division on signed
    integers, computed in `int64`.
  - `FrameToWindow(p image.Point) (x, y int, ok bool)` — `x = ⌈(winW·den − W·num + 2·fx·num) / (2·den)⌉`,
    the **first** window pixel of that frame pixel, with `ok` iff
    `2·den·x < winW·den − W·num + 2·(fx+1)·num`, i.e. iff that first candidate really lies inside the
    interval. `ok == false` is then **exactly** "this frame pixel is not in the image of
    `WindowToFrame`", which happens only when the frame is scaled below 1:1.

  Three properties follow **by construction**, not by tolerance: a letterbox or out-of-window position
  maps to no frame pixel and every window position maps to at most one (FR-7); `WindowToFrame` after
  `FrameToWindow` is the identity wherever `ok` (P-2's invertibility on the image); and containment is
  exact — `ox ≥ 0` and `ox + W·s ≤ winW` both reduce to `W·num ≤ winW·den`, which is what `Fit` selects
  for. `Scale() float64` and `Origin() (float64, float64)` exist **only** to build the draw transform,
  where a 1-ulp error is invisible; no decision is ever taken on them.
  **What the floats do and do not promise, measured.** `Origin` is formed as a single correctly-rounded
  division of the exact integer numerator and denominator, never as `(win − extent·Scale())/2` — that
  form rounds the scale first and then subtracts two nearly equal quantities, which drives the origin a
  few ulp **negative** on **16 629 of the 923 601** window sizes in 640×480…1920×1200 (1.8 %), 700×500
  and 642×481 among them. Formed the first way it is non-negative everywhere, and it matches the
  correctly-rounded exact origin bit-for-bit.
  What no formulation can provide is `Origin + Scale()·extent ≤ win` in `float64`: the product rounds
  independently of the origin, so at 700×500 and 642×481 the far edge overshoots by 5.68e-14 px **even
  with the origin at exactly 0**. Substituting the exact origin leaves that inequality failing on
  precisely the same 16 629 sizes — the overshoot lives entirely in `s·extent`.
  *Rejected, with that measurement as the reason:* asserting containment on the float draw transform,
  which is what an earlier revision of this plan's criterion required. It is not a property any correct
  implementation can carry, so a criterion demanding it is a criterion that can only be satisfied by a
  tolerance nobody can justify. Containment is asserted instead on the **exact rational**, where it
  holds by construction and where it is what actually decides anything; the draw transform is held to
  being within one ulp of the exact origin, which pins centring **more** tightly than the inequality
  did. **This changes only how the plan verifies the contract, not the contract**: `spec.md` P-2 speaks
  of the *mapping* placing the whole frame inside the window, and the exact mapping does so with no
  rounding at all. No spec revision is implied and none is made.
  *Rejected:* the same formulas in float64 — worked through and measured, it fails: at roughly a fifth of
  window sizes in 640×480…1920×1200 at least one frame pixel round-trips off by one (700×500, frame row
  408, is a witness), and `ox` goes slightly negative for a large family of sizes, so the containment
  assertion fails too. *Rejected:* an integer-only scale (`max(1, min(winW/640, winH/480))`) — it
  satisfies "no aspect distortion" but not "the whole frame maps inside the window" for any window
  narrower than 640, which P-2 requires for **any** window at least 1×1. *Rejected:* a float mapping
  with an epsilon tolerance — it trades a provable property for a tuned constant.

- **DD6 — The mask is read as raw 8-bit indices through `Pix`, and the tests prove the palette is not
  consulted.** `Assets.Mask` is an `*image.Paletted`; `ButtonAt(p image.Point) int` reads
  `Mask.Pix[y*Stride + x]` and maps it through an explicit 256-entry table built from the eight decoded
  literals — `0x80→1 … 0xf0→8`, every other value → `0`. A point outside `[0,640)×[0,480)` → `0`.
  Writing the eight literals rather than the equivalent `idx/16 − 7` formula keeps the rule falsifiable
  by inspection and total by construction.
  **The non-hollow witness:** a test decodes a mask whose palette is deliberately **not** the identity
  ramp (every entry black) and asserts the button lookup is unchanged. A colour-based implementation
  fails that test; an index-based one cannot.
  *Rejected:* `Mask.At(x, y)` / `color.Gray` — goes through the palette, which the spec's constraint says
  carries no meaning; *rejected:* the arithmetic form — it would silently accept values the research
  does not list as hot.

- **DD7 — Row order: the public BMP convention, stated once, in one place.** The readers treat a
  **positive** DIB height as bottom-up storage and normalise to top-down display rows, and accept a
  negative height as already-top-down. Both readers key on `bfOffBits` and require only that enough
  bytes are present — never that the stream length equals the computed minimum, because every shipped
  menu bitmap carries two bytes beyond it (see *Facts*). This is **the** decision the research's
  *inferred* row order bears on, so it is written in exactly one function, named in the package doc, and
  pointed at from the manual criterion. A unit test asserts the code is **sensitive** to it — a
  vertically mirrored synthetic mask changes which button a fixed point selects — so the manual
  criterion is capable of falsifying the inference rather than passing vacuously.
  *Rejected:* rejecting negative heights as `pkg/render/terrain` does — three lines buy a startup
  failure mode for a legal input; *rejected:* deriving the order from the placement tables at runtime
  (auto-flipping until the rects bracket the regions) — that is inventing a format fact the research
  explicitly left open, and it would make the inference unfalsifiable forever.

- **DD8 — NEW GAME is computed from the placement table by the FR-9a rule; the answer is never a
  literal.** `newGameButton()` splits the hover table at `ColumnSplit = 320` (left column: `x < 320`),
  then takes the entry with the smallest `y` among the left column, ties broken by smallest `x` then
  lowest index so the result is total and deterministic.
  Three assertions carry it: the resulting rectangle equals `(112,64)…(324,200)`; it is **not** the
  bottom-left entry; and **a rule that ranks by `y` alone over the whole table picks a different
  button** — the test computes that rule too and asserts the two disagree, which is the only way the
  spec's warning about the right column starting four pixels higher becomes a regression test rather
  than a comment.
  **Independence from the mask is structural, and deliberately not asserted by a test.** `newGameButton`
  is a pure function of the placement table and `ColumnSplit`; the index→button table and every
  `Assets` field are simply not in scope where it is computed. A test that permuted the mask table and
  observed `NewGameButton` unchanged would pass by construction whatever the code did — it is the
  vacuous shape the green-but-hollow audit exists to reject — so the guarantee is carried by the
  function's inputs and by the two rules above, not by a test written to look like evidence.
  *Rejected:* a literal index constant — exactly the option the spec's Constraints table rejects as
  unfalsifiable; *rejected:* "the topmost entry" — it selects the wrong button.

- **DD9 — The hover/pressed latch is a pure state machine, placed with the tables it indexes.**
  `menu.Selection` holds `hit` (the button under the cursor, `0` = none) and `latched` (`0` = nothing
  latched). Events: `Move(hit)`, `Press(hit)` (latches `hit`, which may be 0 and then latches nothing),
  `Release(hit) (activated int)` (returns `latched` when `hit == latched` and `latched != 0`, then
  clears the latch). `State()` resolves to `{Selected, Pressed}`: latched≠0 and hit==latched →
  `{latched, true}`; latched≠0 and hit≠latched → `{0, false}`; latched==0 → `{hit, false}`. That
  reproduces every clause of FR-8 including "a press that selects no button latches nothing and hovering
  keeps tracking normally", because only a **non-zero latch** suppresses hover.
  **Provenance, stated precisely:** the latch and drag-off rules are the **spec's own requirements**
  (FR-8, FR-9a); the spec attributes to research only the hover-vs-pressed *roles* this machine selects
  between. It is placed in `pkg/render/menu` for cohesion — it is a pure function over the button set
  that package owns and it produces the `State` that package's `Compose` consumes, so FR-8's whole
  contract stays decidable in one stdlib-only package.
  *Rejected:* keeping a separate `held bool` — it would make "held with nothing latched" a state that
  suppresses hover, which FR-8 forbids. *Rejected:* placing it in `pkg/ui` — not for testability
  (`pkg/ui` tests are fully headless), but because the state and the composition it drives would then
  sit either side of a package boundary for no gain, and AC-10 would have to reach across it.

- **DD10 — Composition is `base` then **at most one** overlay, copied opaquely.**
  `Assets.Overlay(s State) (img *image.RGBA, at image.Rectangle, ok bool)` returns the single overlay to
  draw — the hover bitmap at the hover rect, or the pressed bitmap at the pressed rect — or `ok == false`.
  `Assets.Compose(s State) *image.RGBA` copies the base then, if `ok`, `draw.Draw(dst, at, img,
  image.Point{}, draw.Src)`. **`draw.Src`, not `draw.Over`**: the source is opaque either way, but `Src`
  states in the code that no blend is applied, because no blend is decoded. P-5 is structural — there is
  no code path that draws two overlays — and AC-10 checks it at the pixel level as well.
  *Rejected:* any colour-key or alpha rule — not decoded, and inventing one violates golden rule 4;
  *rejected:* compositing on the GPU from base and overlay textures — the composed frame would then not
  be a value a test can assert (an `*ebiten.Image`'s pixels cannot be read back in a test at all), and
  "never two overlays at once" would become an unobservable property of draw-call ordering.

- **DD11 — `alm` gains one additive, metadata-only entry point; `Open` is not changed, and 0003 needs no
  spec revision.** `Info{Width, Height int; Name, Description string; FormatVersion uint32}` and
  `OpenInfo(data []byte) (*Info, error)` decode the file header, the ten record headers, and the type-0
  payload — and stop. `Open` and `OpenInfo` share the same unexported record walk and the same
  unexported type-0 payload decoder, so the two can never disagree about the layout, and `Open`'s
  exported signature, error strings and behaviour are unchanged (`alm_test.go` is the pin).
  **The reachable state, stated precisely so the fixture is right:** because the record walk requires the
  ten records to tile *exactly* to EOF, a physically truncated file fails `OpenInfo` too. The state
  where `OpenInfo` succeeds and `Open` fails is a **well-framed** file whose type-1 record declares a
  `payloadSize` inconsistent with `2·W·H` — the file still tiles, type-0 still decodes, and the failure
  lands in the grid check. That is the fixture `info_test.go` builds and the shape FR-3's "passed the
  metadata read, then failed to decode fully" describes.
  **Reconciliation, since `pkg/formats/alm` is another story's spec-anchored package:** 0003's spec
  states the `.alm` **byte layout**, and `OpenInfo` changes none of it, adds no claim, and reads no
  offset 0003 does not already specify — it is a second entry point over the decode 0003 already
  documents. No 0003 spec revision is therefore required. This story's own spec is what authorises the
  addition: its Constraint that "building the picker list reads no more of any map than its own
  metadata" is unsatisfiable through `Open`, and FR-3's two-state distinction is unreachable through it.
  *Rejected:* using `alm.Open` for the listing pass — FR-3's clause becomes unreachable, the picker
  fully decodes 38 maps (grids, objects, units, triggers) on every start including `-check`, and a map
  with a bad grid vanishes from the list instead of being listed-then-reported;
  *rejected:* re-implementing the header walk in `pkg/game` — it would put `.alm` layout knowledge
  outside the package that owns it.

  > **Corrected 2026-08-01, at pin `130bb79`. DD11 above is left as written and its two "ten
  > records" clauses are false — the second of them was load-bearing for a fixture's rationale.**
  >
  > `.alm` never required ten records and this reader has not required them since `0003`'s T9. The
  > loader's **only** count gate is `recordCount >= 3` (`ALM-REQ-055`); the open path requires
  > `{type1, type2}` and nothing else, type-3 is manufactured, type-5 is synthesised, and every
  > other id is optional (`ALM-REQ-056`); a repeated id overwrites, an id at or above 10 is seeked
  > past, and the physical order is unenforced though three precedence relations inside it are real
  > (`ALM-ORD-057`). `ru/Horror.alm` is a four-record map the engine accepts (`ALM-CORP-060`), and
  > this tree measured the same thing from the other side: over both preserved roots, 44 names / 72
  > walked files, **71 carry ten records and one carries four** (`docs/0003-alm-container/analysis.md`).
  > So one story here required an invariant another story in the same tree had measured false.
  >
  > **Nor is the tiling exact.** The walk's loop bound is `recordCount`, so bytes past the last
  > record are never reached and are not an error. What `walkRecords` rejects is narrower and each
  > rejection is the loader's own: a record header that does not fit before EOF, a payload that
  > overruns it, a tag or `hdrLen` that is not the constant, and then type-0, type-1 and type-2.
  >
  > **The conclusion DD11 drew survives its reason.** A physically truncated file does still fail
  > `OpenInfo` — not because ten records must tile, but because the truncation lands as a header
  > that does not fit or a payload that overruns. And the state DD11 needed is still exactly the
  > state it describes: a **well-framed** file whose type-1 record declares a `payloadSize`
  > inconsistent with `2·W·H`, which `OpenInfo` passes and `Open` fails in the grid check. So
  > `info_test.go`'s fixture is right and no criterion moves.
  >
  > **`OpenInfo` does not restrict anything to ten**, so there is no local compatibility restriction
  > to declare as ours: it shares `walkRecords` with `Open` by construction, which is what DD11's
  > own "the two can never disagree about the layout" was for. `analysis.md` carries the same
  > wording at its own pin ("ten record headers that must tile to EOF") and is left as written,
  > being a dated record of what `pkg/formats/alm` did in July. `0003` and `0023` were audited for
  > the same sentence and are clean: `0023` cascaded the correction through its own contract, and
  > `0003`'s surviving uses are fixture descriptions and shipped-corpus observations, both of which
  > are true — every shipped EN map does carry ten records that tile exactly.

- **DD12 — One camera step for both entry points, behind an input snapshot that carries only raw engine
  reads.** `Input{PanLeft, PanRight, PanUp, PanDown bool; CursorX, CursorY int; WheelY float64}` is read
  from the engine by `readInput()`; `(*Viewer).step(in Input, now time.Time)` performs the animation
  advance, the pan and the wheel zoom. **`Input` carries no "cursor inside" flag**: insideness is a
  function of the *camera's* `ViewW`/`ViewH`, which `readInput()` cannot see, and `step` decides it
  exactly as `panIntent` does today. Putting the flag in the snapshot would have meant either a dead
  field or a changed edge-scroll rule — the divergence FR-4a forbids.
  `Viewer.Update()` becomes exactly the Esc check it already had plus `v.step(readInput(), time.Now())`;
  the front-end's map screen calls the **same** `step` after handling Esc its own way. Divergence would
  require editing one method, and `step` is reachable headlessly, so the shared camera behaviour is
  **tested** rather than asserted.
  *Rejected:* an `escapeExits bool` field on `Viewer` — it changes a shipped type's behaviour on a flag,
  which is exactly the divergence FR-4a forbids; *rejected:* the front-end reimplementing pan/zoom — the
  divergence FR-4a forbids, verbatim.

- **DD13 — The picker is a pure model with a scroll window, an absolute selection setter and stated
  layout constants.** A stock install lists 38 rows and the frame cannot show them all, so the model
  owns a scroll window and the draw path and the hit-test share it. The constants, in frame pixels:
  `pickerLeft = 8`, `pickerHeaderY = 8`, `pickerTop = 40`, `pickerLine = 16` (the debug font's cell
  height), `pickerVisible = 25` (the list occupies `y ∈ [40, 440)`), `pickerMessageY = 452`, and
  `pickerCols = 104` runes (`8 + 104·6 = 632 ≤ 640`), to which the draw path clips each row's text.
  `Picker` holds `rows []PickerRow{Text string; Choosable bool}`, `sel int` and `top int`. The API is:
  - `Move(d int)` — relative; clamps `sel` to `[0, len−1]` and scrolls `top` **minimally** to keep `sel`
    visible. Drives Up/Down.
  - `Select(i int) bool` — **absolute**; sets `sel` to `i` when in range (scrolling the same way) and
    reports whether it did. This is what a click needs, and its absence would leave FR-3's "MUST support
    click" and AC-4's "click chooses" inexpressible.
  - `RowAt(p image.Point) (int, bool)` — maps a frame position to a row using **only `p.Y`**:
    `top + (p.Y − pickerTop)/pickerLine`, rejected when above the list, past the visible window, or past
    the last row. Ignoring `p.X` is deliberate: any x inside the frame selects the row, which makes the
    debug font's per-glyph `+1` px x offset irrelevant to hit-testing.
  - `Choose() (int, bool)` — yields the selection only when it is choosable.
  - `SetUnusable(i int)` — clears a row's `Choosable`; what FR-3's failed load calls.
  - `Visible() (top, n int)` — what the draw path iterates, so the two cannot drift apart.
  - `HeaderText() string` — the one-line header, **derived from `Visible()`**: a fixed title, then
    `maps <first>-<last> of <total>` with 1-based row numbers, then the key hints, clipped to
    `pickerCols`. An empty list reads `no maps` instead of a range, which is the only branch.

  **The selected row is visibly marked**: the draw path prefixes the selected row with `"> "` and every
  other row with two spaces, so the marker costs no horizontal alignment and needs no colour. Without it
  a human could not perform AC-7a.
  **The header states the list's extent** (FR-2a, AC-13). This is a **revision**: the header used to be
  a constant string, and on a stock install a 38-row list drawn 25 rows at a time then looked exactly
  like a 25-row list — nothing on screen said otherwise, and the owner read it as an incomplete
  inventory. Deriving the numbers from `Visible()` rather than from `sel`/`top`/`len` directly is the
  point: the range a user reads is by construction the range the draw path renders and the hit test
  accepts, so the screen cannot claim rows it will not show or hide rows it would still click.
  *Rejected:* a scrollbar — it needs pixels and a second hit region on a screen whose text the spec
  calls placeholder, and it says how far you are but not how many there are;
  *rejected:* printing the count in the failed-load message line — that line is empty in the normal
  case, which is exactly when the count is needed.
  *Rejected:* two columns — it doubles the hit-test's failure modes for a screen the spec calls
  placeholder; *rejected:* shrinking the text — the debug font is a fixed bitmap; *rejected:* expressing
  a click as `Move(target − current)` — arithmetic on a clamped relative operation is exactly where an
  off-by-one hides.

- **DD14 — The screen flow is a pure state machine over an injected loader, and it starts on the menu.**
  `MapLoader func(index int) (*Viewer, error)`; `flow{screen Screen; picker *Picker; load MapLoader;
  viewer *Viewer; msg string}`. **`newFlow` sets `screen = ScreenMenu`** — FR-7's "the application MUST
  open on the main-menu screen", stated explicitly because the retired `AC-3`/`AC-5`/`AC-7` assumed the
  opposite and the mistake is otherwise invisible to every automated check.
  `activateNewGame()` moves menu→picker. `choose()` on a choosable row calls `load`; on success it
  switches to the map screen, on failure it **stays** in the picker, sets `msg` to the error text and
  calls `picker.SetUnusable(i)` — never exits, never crashes. `escape() (exit bool)` returns `true` only
  from the menu; from the map screen it returns to the picker and from the picker to the menu. `msg` is
  cleared on any successful transition.
  **NEW GAME is the only activation entry point on the *flow*.** `flow` exposes no per-button command
  and no table of them, so a release on any of the six unbound buttons is consumed by the menu screen and
  changes nothing — FR-9a's "consumed and has no effect".
  **Revised:** that absence used to double as the guard on "button 8 is not wired to quit". It no longer
  does, and could not: EXIT does not change screen, it ends the program, and the flow's own way of
  saying that is `escape()`'s return value rather than a transition. DD31 puts the binding in the menu
  dispatch and leaves `flow` unchanged — so the six unbound buttons are still guarded by there being no
  entry point for them, while the one decoded command is bound where the button numbers already live.
  *Rejected:* letting `App` hold the *screen transitions* inline — the whole of AC-3a and P-1 would then
  need an engine context; *rejected:* a `map[int]command` dispatch table "ready for later buttons" — it
  is still the shape that invites binding commands nobody has decoded, and DD31 binds exactly one by
  name instead.

- **DD15 — One 640×480 offscreen image is "the frame"; only it is scaled to the window.** `App` keeps an
  `*ebiten.Image` of exactly 640×480 (constructible with no graphics context — see *Facts*). The menu
  screen composes `menu.Compose` on the CPU **only when the `menu.State` changed since the last
  compose** and uploads it with `WritePixels`. The picker screen fills the frame and draws, with
  `ebitenutil.DebugPrintAt`: the DD13 header line at `pickerHeaderY`, the `Visible()` rows from `pickerTop` at
  `pickerLine` pitch each prefixed by its DD13 selection marker and clipped to `pickerCols`, and — when
  `flow.msg` is non-empty — that message at `pickerMessageY`. **That message line is where FR-3's "MUST
  report that failure" is discharged**; a failed load that set `msg` and drew nothing would satisfy no
  requirement.
  `Draw` then blits that single image to the screen with `GeoM.Scale(Scale, Scale)`,
  `GeoM.Translate(Origin)` and `FilterNearest`, so the finished frame — never its parts — is what the
  window shows, at one uniform factor, centred and letterboxed. The map screen is the deliberate
  exception: it bypasses the frame and calls `viewer.Draw(screen)`, and `App.Layout` forwards the window
  size to `viewer.Layout` so the camera is sized by the window (FR-7, FR-4a).
  *Rejected:* recomposing the menu every frame — 1.2 MB per frame for a screen that changes only on
  hover; *rejected:* drawing base and overlay separately onto the screen with two scaled blits — it
  would scale the parts, not the frame, and re-open the resampling FR-8 forbids; *rejected:* reporting
  the load failure on stderr instead — the user is at a window, and a line on a stream they cannot see
  is not a report.

- **DD16 — The window opens at an *integer* multiple of the frame, resizable. The multiple is 2 —
  1280×960.** What the gated version of this decision got right was that the startup size must be one
  where the scale is exactly an integer, so the brooch art is reproduced pixel-for-pixel rather than
  resampled; the viewer's 1024×768 default would put it at 1.6× under nearest-neighbour, i.e. uneven
  pixel doubling. What it got wrong was picking 1× on the grounds that it was the smallest such size:
  640×480 is a quarter of a 1080p desktop and the owner reported it as too small to use. 2× is the
  largest integer multiple that still fits vertically under a 1080-line desktop with its title bar
  (960 + chrome < 1080); 3× would be 1440 lines and would not.
  `MenuWindowW/H = frame.W*2, frame.H*2` and the scale falls out of `frame.Fit` as the exact rational
  `1280/640 = 2` — the letterbox arithmetic is untouched, and so is every other size, since resizing
  stays enabled. The map screen fills whatever window it is given, so it simply starts with a larger
  camera view.
  *Rejected:* a fractional default sized to the desktop — it resamples the brooch on the one screen
  whose art we reproduce exactly, and DD5's mapping would still be correct while looking worse;
  *rejected:* `ui.DefaultWindowW/H` for consistency — consistency with a developer tool is worth less
  than a pixel-exact menu; *rejected:* querying the monitor and choosing the largest integer multiple
  that fits — it makes the startup size depend on the machine, so no test and no bug report describes
  the same window twice.

- **DD17 — All three archives are opened up front, `graphics.res` included, and the tileset is built
  once.** `OpenArchives(root)` opens `main.res`, `graphics.res` and `scenario.res` **in that fixed
  order** and returns the **first** failure with the path in the message — FR-1's stated fail-early rule.
  The front-end then builds the terrain tileset **once, at startup**, with
  `terrain.LoadTileset(archives.Graphics)`, and reuses it for every map choice: `graphics.res` is read
  exactly once per process, and `Archives.Graphics` has a consumer. A picker→map→Esc→picker→map cycle
  re-decodes only the chosen map.
  *Rejected:* opening `graphics.res` lazily on the first map load — it is the option FR-1 legislates
  against by name ("an install missing a piece the app will need should say so before the user picks a
  map"), and under `-check` it would never be opened at all, so the mode would pass on an install the
  windowed run then fails. *Rejected:* rebuilding the tileset per map choice — a whole-archive re-read
  and re-decode on every Esc→pick cycle, for a tileset that cannot have changed.

- **DD18 — The map list is built from two abstract sources, so AC-1a needs no filesystem.**
  `NamedReader{ Names() []string; Read(name string) ([]byte, error) }` with two adapters — `DirMaps(root)`
  (**non-recursive** `os.ReadDir`, keeping regular files whose extension case-folds to `.alm`, source
  text = the bare file name, so the asset root's subdirectories are not scanned) and
  `ArchiveMaps(a *res.Archive)` (a **flat** scan of `Entries()` keeping paths that case-fold to a `.alm`
  suffix, source text = the entry path as the reader reports it — bare names on the shipped
  `scenario.res`; a nested archive stored as an entry is one opaque blob and is not descended into).
  `BuildMapList(loose, archive NamedReader) []MapEntry` is a pure function of the two sources and is
  where the ordering, the metadata read and the choosability live.
  *Rejected:* a single `BuildMapList(root string)` — AC-1a's "one source text present in both sources"
  and "entries differing only by letter case" cases would need a real directory and a real archive.

- **DD19 — The ordering rule and the row text, stated once.** Rows sort by
  `(kindRank, fold(Source), Source)` where `kindRank` is `0` for loose and `1` for archive, `fold` is
  `strings.ToLower`, and the final term is a plain byte comparison of the same source text. Loose
  entries are appended before archive entries and `sort.SliceStable` is used, so even two rows equal on
  all three keys keep a defined order. Nothing is deduplicated: a loose file and an archive entry with
  equal source texts are two rows, the loose one first, exactly as FR-2a requires.
  **This is a revision, and the change is which key is primary.** The gated rule was
  `(fold(Source), kindRank, Source)` — source text first, loose-before-archive only as a tiebreak. It
  satisfied every synthetic case in SC-1 and was still wrong on the shipped install: `scenario.res`'s 28
  campaign maps are named `10.alm`…`91.alm`, digits sort before letters, so all 28 landed ahead of the
  10 loose maps — and since campaign maps record no name of their own, the picker's 25 visible rows were
  25 bare numbers. Promoting `kindRank` to the primary key puts every named map on the first screenful.
  *Rejected:* sorting on "does this row have a recorded name" — the name is decoded at runtime, an
  unreadable row has none, and one campaign map (`81.alm`, "Panic") does, so the groups would be neither
  stable nor explicable from a row's source alone; `kindRank` is a property of where the row came from
  and needs no bytes read to know. *Rejected:* two visually separate lists — it doubles the hit-test's
  cases for a screen the spec calls placeholder, and FR-2a says one list.
  Each row is `MapEntry{Source string; FromArchive bool; Name string; Err error}`; `Choosable()` is
  `Err == nil`; `Text()` is `Source`, plus `" - " + Name` when the recorded name is non-empty, plus
  `"  [unreadable]"` when `Err != nil`. The text is ASCII because the debug font covers U+0000–U+00FF
  and the spec calls this text placeholder; the draw path clips it to `pickerCols` (DD13). An empty
  recorded name yields a row that is still listed and still choosable — it just shows its source alone.
  *Rejected:* `strings.EqualFold`-style folding — it is an equality test, not a sort key, and FR-2a needs
  a total order; `strings.ToLower` is total, map names are ASCII in practice, and archive paths arrive
  already lower-cased from the reader. *Rejected:* one precomputed comparison string per row — the three
  terms are not concatenable without inventing a separator that could itself appear in a source text.
  *Rejected:* showing the recorded name first and the source second — FR-2a requires the source, and a
  row whose recorded name is empty (most campaign maps) would then start with a blank.

- **DD20 — The shared load path lives in `pkg/game`, and `cmd/mapview` keeps every character of its
  output.** `OpenTileset(path string) (*terrain.Tileset, error)` (`res.Open` then
  `terrain.LoadTileset`, wrapping a failure as `open <path>: <err>`) and
  `LoadMapViewer(tiles *terrain.Tileset, data []byte, fallbackTitle string) (*MapView, error)` — where
  `MapView{Viewer *ui.Viewer; Map *alm.Map; Title string}` — are the **only** place map bytes become a
  running viewer. `cmd/mapview.load()` keeps its own flag resolution, its own summary literal and its
  own `decode <mapPath>: <err>` wrapping, and calls these two; `cmd/againrom`
  calls `LoadMapViewer` with the tileset DD17 built once. The summary text stays in the command because
  it is presentation, not loading, which is how "character-for-character unchanged" is achieved without
  freezing a library API around a developer tool's wording.
  *Rejected:* returning a formatted summary from `pkg/game` — mapview's exact wording would become a
  library contract, and the front-end would inherit a summary it does not want;
  *rejected:* leaving the path in `cmd/mapview` and duplicating it in `pkg/game` — the divergence FR-4a
  exists to prevent.

- **DD21 — `-check` does the whole startup and prints one line.** `NewFrontEnd(root)` opens the
  archives, loads and validates the menu assets, builds the tileset and builds the map list;
  `CheckLine()` renders `againrom: <N> map rows, <K> of 8 buttons have a mask region`. Both counts are
  unambiguous and the wording is fixed here rather than left open downstream. A button present but with
  an **empty** mask region is reported by `K < 8` and is **not** a failure — reporting it is what the
  mode is for — while any FR-10 failure surfaces from `NewFrontEnd` and overrides the exit-0 case in
  both the windowed and the `-check` mode.
  *Rejected:* two lines, or a machine-readable form — FR-5 says one line and pins nothing else; a
  second consumer can parse two integers out of this one. *Rejected:* accumulating and printing every
  FR-10 failure — see DD28; `-check` reports the same first failure the windowed mode does, so the two
  modes cannot disagree about what is wrong with an install.

- **DD22 — `cmd/againrom` reads flag **and** environment, reports on stderr, and exits non-zero on every
  startup failure.** `run(args []string, getenv func(string) string, stdout, stderr io.Writer) int`
  keeps `main` a three-liner and makes both the exit status and the environment fallback testable
  without mutating the process. It resolves the root through the existing
  `game.ResolveAssetRoot(flagValue, getenv("AGAINROM_ASSETS"))`, so the flag > env > unset precedence
  stays in the one place that already owns it and is already tested. **This preserves FR-1's environment
  fallback, which the command has today**; "replaced wholesale" applies to the *print-and-return-zero*
  behaviour, not to how the root is resolved. No root configured is reported on stderr with a non-zero
  exit. No test pins the old text.
  *Rejected:* `main` calling `os.Exit` directly — the exit status would be untestable, and FR-10 makes
  it contract. *Rejected:* reading `os.Getenv` inside `run` — the environment fallback would then be
  testable only by mutating the process environment, and SC-11 would be order-dependent.

- **DD23 — Nothing this story ships writes a bitmap, and no menu art is ever converted to a file.** The
  front-end decodes menu bitmaps into memory and draws them; there is no export path, no `-out` flag and
  no on-disk cache. The asset guard's extension list does not cover `.bmp`/`.png` (baseline), so the
  protection here is that **no code path produces such a file at all**.
  *Rejected:* a decoded-frame cache on disk to speed startup — it would write converted game art into
  the filesystem for a startup cost the measurement in R-5 does not justify, and it is exactly the
  artefact golden rule 1 exists to keep out of a working tree. *Rejected:* widening the guard's
  extension list to `bmp`/`png` in passing — it is 0000's contract, and changing another story's
  mechanism as a side effect of this one is the silent scope growth S-6 forbids; the gap is reported
  instead (R-8).

- **DD24 — The brownfield areas are pinned before they are changed, and each pin is a new file.**
  `Viewer.Update` is restructured (DD12) and today nothing pins it, although it *is* callable headlessly
  (see *Facts*). A characterization test is therefore written and landed **before** the restructuring,
  in `pkg/ui/update_pin_test.go`: with the camera placed away from its clamp bounds, successive
  `Update()` calls move it by exactly the shipped edge-scroll amount and return `nil`. Likewise
  `cmd/mapview/flagset_test.go` pins the exact flag-name set and the exact `usage` string before
  `main.go` is touched, catching a flag change that the summary tests would miss.
  **Two things stay honestly unpinned by any automated test.** Esc-closes-the-window cannot be driven
  headlessly (`inpututil.IsKeyJustPressed` has no test seam), and `Viewer.Run()` is exercised by none of
  the nine existing tests. Their protection is that the Esc branch of `Update` and the whole of `Run`
  are **textually unchanged** — a `git diff` of those hunks against the pre-story baseline must be empty
  — plus the live criterion. Neither is claimed as automated evidence.
  *Rejected:* adding an Esc seam so the branch becomes testable — it restructures the one part of the
  shipped behaviour AC-5a names explicitly, to buy a test of a two-line branch.

- **DD25 — The error-string contract is stated, not assumed.** `cmd/mapview`'s two user-visible
  wrappings, `open <archivePath>: <err>` and `decode <mapPath>: <err>`, are **contract** and are
  preserved character-for-character: the first is produced by `game.OpenTileset`, which has the path;
  the second stays in `cmd/mapview`, because `LoadMapViewer` takes bytes and has no path to name.
  `LoadMapViewer` therefore returns the underlying `alm`/`ui` error unwrapped, and each caller labels it
  with its own source. The existing failure tests only substring-match `open`/`decode`, so this is
  stated here rather than left to a test that would not notice a degraded message.

- **DD26 — `pkg/render/frame` exposes no rectangle helper beyond what the draw transform needs.** The
  API is `Fit`, `Valid`, `WindowToFrame`, `FrameToWindow`, `Scale`, `Origin` and the two size constants.
  There is deliberately no `Dest()`-style convenience rectangle: every consumer either takes a decision
  (and must use the exact integer mappings) or draws (and needs only `Scale`/`Origin`).

- **DD27 — `App` takes its input as one snapshot, and the bindings are named here.**
  `appInput{Viewer Input; CursorX, CursorY int; PrimaryPressed, PrimaryReleased, Escape, Up, Down,
  Enter bool}` carries one tick of everything the front-end reads; `(*App).step(in appInput, now
  time.Time) (exit bool)` performs the whole dispatch, and `App.Update()` gathers the snapshot from the
  engine and calls it — `Update` holds no logic. The bindings, so no task invents them:
  **primary mouse button** = `ebiten.MouseButtonLeft` (just-pressed / just-released edges);
  **Esc** = `ebiten.KeyEscape`; **Up/Down** = `ebiten.KeyArrowUp`/`ebiten.KeyArrowDown`;
  **Enter** = `ebiten.KeyEnter`. Dispatch per screen:
  - *menu* — the cursor goes through `frame.WindowToFrame` then `menu.ButtonAt`; a position in the
    letterbox yields `0`. `Press`/`Release` drive `menu.Selection`; a release that activates the
    `NewGameButton` calls `flow.activateNewGame()`, a release that activates `menu.ExitButton` reports
    exit (DD31), and a release that activates any of the other six does nothing. **Esc is the menu's
    only key** — Up, Down and Enter are ignored there, which is the spec's out-of-scope "no keyboard
    navigation of the menu buttons".
  - *picker* — Up/Down call `Picker.Move(∓1)`, Enter calls `flow.choose()`, a primary release inside the
    frame calls `Picker.RowAt` then `Picker.Select` then `flow.choose()`, and Esc returns to the menu.
  - *map* — Esc returns to the picker; otherwise `viewer.step(in.Viewer, now)`.

  Every screen transition, every latch event and the map screen's camera call are therefore reachable in
  a test with no window.
  *Rejected:* `App` calling `readInput()` and the engine's mouse/key helpers inline — the flow, the
  latch dispatch and the map-screen routing would all become live-only, which is most of what this
  story does.

- **DD28 — `menu.Load` validates the whole set and fails on the **first** defect, in a fixed order.**
  The order is: `menu_.bmp`, then `menumask.bmp`, then for `i = 1…8` the hover overlay and then the
  pressed overlay. For each entry: present → decodes → dimensions. The base and the mask must be
  640×480; each overlay's decoded `(w, h)` must equal its own row of the corresponding placement table.
  The first failure returns `nil, err` with the offending entry named; no partial `Assets` is ever
  constructed, let alone returned (P-4).
  *Rejected:* accumulating every defect into one joined message — AC-12 supplies one defect per asset
  set so both behaviours pass it, but a multi-error format is a thing nothing specifies, and it would
  make `-check` and the windowed mode able to disagree about which failure to name unless both joined
  identically. First-failure matches DD17's archive behaviour, so an operator sees one shape of message
  wherever the install is broken.

- **DD29 — "a non-empty hit region" is defined as at least one pixel.** `Assets.MaskRegions() [8]int`
  returns, for each button `1…8`, the **count of frame pixels whose raw mask index is that button's hot
  value** — one pass over the 640×480 mask. `CheckLine`'s `K` is how many of the eight counts are
  greater than zero. This is the whole of FR-5's "for how many of the eight buttons the mask carries a
  non-empty hit region", and it is worth pinning because it is the entire content of the developer-run
  criterion's second number.
  *Rejected:* a bounding box, or a contiguity requirement — both would call a real but scattered region
  empty, and neither is decoded; the research's own statement is about which index values are present.

- **DD30 — Synthetic fixtures live in one unpoliced helper package.** `internal/synth` exports
  `Archive(entries)` (a `.res` container), `BMP8(w, h int, palette [256][3]byte, rows []byte)`,
  `BMP24(w, h int, pix []byte)`, `ALM(...)` and `MenuArchive(...)` (an eighteen-entry menu archive at
  the DD3 keys, with per-entry overrides so AC-12's defective sets are one call away). `internal/` is
  **not policed by `archtest`** and the package is imported only from `_test.go` files, so it enters no
  binary and no DAG row. It exists because this story needs a **24-bpp** builder — which the tree does
  not have — in `pkg/render/menu`, `pkg/game` and `cmd/againrom`, plus a `.res` builder in two of them
  and a `.alm` builder in three; four hand-copied 24-bpp writers would be four chances to disagree
  about the same public format.
  **What it deliberately does not do:** it builds **inputs** only. No expected image, no expected
  rectangle and no expected ordering lives there, so a shared builder cannot make a test agree with the
  implementation by construction — the separate-context oracles stay independent.
  *Rejected:* the house convention of per-test-package copies — it is fine for one builder in two
  packages and stops being fine at five builders in four; *rejected:* a `pkg/...` fixtures package — it
  would be a shipped library tier needing a DAG row, for code no binary uses.

- **DD31 — EXIT is bound to the decoded button *number*, in the menu dispatch, and it is the only
  per-button command bound.** `pkg/render/menu` gains `const ExitButton = 8` and nothing else; `App`'s
  menu dispatch gains one arm — a release that activates `menu.ExitButton` makes `App.step` report
  `exit`, which `App.Update` turns into `ebiten.Termination`, exactly the value Esc at the menu already
  produces. So there is **one** exit path through the engine, not two, and `cmd/againrom`'s exit status
  is unchanged by construction.
  **Why a number and not a rectangle, when DD8 insists on a rectangle for NEW GAME.** The two facts have
  different provenance. NEW GAME's identification is the owner's gameplay knowledge — a claim about a
  position on screen — so it is anchored to a position, and `newGameButton()` computes it. EXIT's is
  decoded: `MENU-STATE-007` says the click dispatcher posts `WM_CLOSE` for **button 8**, naming a
  number. Computing "the bottom-right entry" and calling that EXIT would be inventing a geometric claim
  the research does not make; the rectangle (`324, 276, 208, 136`, `MENU-GEOM-005` entry 8) is written
  down as a *consequence* so a human can find the button for AC-14, and a test asserts that consequence
  holds — but no code derives the button from it.
  **The exposure, stated rather than mitigated.** This binding runs through the same inferred mask row
  order as everything else: `ButtonAt` answers with a mask-derived number, and if the row order is
  flipped (R-1) the bottom-right region answers `5` and the *top*-right one answers `8`. There is no
  design that removes that — a geometric anchor would move the problem, not solve it, since the mask is
  the only thing that says what the cursor is over. AC-14 is what settles it, and until it runs the
  binding is asserted only against synthetic assets.
  **Why not in `flow`.** `flow`'s vocabulary is screens; EXIT changes no screen. `escape()` already
  carries "should the program end" as a return value, and `App.step` already propagates one. Putting the
  binding beside `NewGameButton` in the menu dispatch keeps every button-number↔action pair in one
  place, and leaves `flow`'s six-unbound-buttons guard (DD14) intact.
  *Rejected:* a `map[int]func()` command table — the shape DD14 rejects, and it would invite binding the
  six ids `MENU-STATE-007` says are undecoded; *rejected:* posting a synthetic `WM_CLOSE` — we are not
  reimplementing the original's message loop, and Ebitengine's termination value is the platform-neutral
  equivalent; *rejected:* wiring EXIT to the same code path as the window's close button — that path is
  Ebitengine's own and takes no route through `App.step`, so the behaviour would be unreachable in a
  test.

- **DD32 — The wheel scrolls the picker by moving the *selection*, and it arrives in the same input
  snapshot as everything else.** `appInput` gains `WheelY float64`, read once per tick in
  `readAppInput()` from `ebiten.Wheel()`; `App.stepPicker` turns a non-zero `WheelY` into
  `Picker.Move(∓pickerWheelRows)` — away from the user is towards row 0 — and does nothing else, so the
  wheel never chooses. `pickerWheelRows = 3`. The wheel arm sits beside Up/Down in the same `switch`, so
  a wheel roll on the menu or the map screen cannot reach the picker: those screens' dispatch simply
  does not read it (the map screen's camera zoom keeps taking its own wheel value out of `in.Viewer`,
  which is where it already was).
  **Why the selection and not an independent view offset.** `Picker` maintains one invariant that the
  whole screen rests on: `top` is whatever keeps `sel` visible, and `Visible()` is the single value the
  draw path, the hit test and now the header all derive from. A view offset that moved `top` on its own
  would break it — the selection could scroll off screen, `RowAt` would name rows the marker is not on,
  and DD13's "a row that is drawn is a row that can be clicked" would need a second rule. Moving the
  selection keeps one invariant and reuses `Move`'s clamping and minimal scrolling as they are.
  *Rejected:* a free-scrolling view with the selection left behind — the extra rule above, for a screen
  the spec calls placeholder; *rejected:* one row per notch — 38 rows in a 25-row window makes a
  one-row notch feel broken; *rejected:* scaling the step by `|WheelY|` — the value is platform-dependent
  (a notch is 1 on Windows and fractional on some trackpads) and would make the step untestable as a
  fixed number; the sign is all that is read.

- **DD33 — The shared load path owns the marker wiring; the game exposes it as one flag, on by
  default; and the marker's derivation is never unified with art placement.** `LoadMapViewer` takes a
  `game.Markers{Objects, Units bool}` and does the wiring itself, through `game.MarkerCells(*alm.Map)`
  — the one conversion from a decoded map's placement records to marker cells, `terrain.AnchorCell` of
  each record's own stored anchor and nothing else. `FrontEnd` carries a `Markers` field that
  `NewFrontEnd` sets to both-on, so the default lives in exactly one place; `cmd/againrom` overrides it
  from `-markers` (default on, `-markers=false` off), and `cmd/mapview` passes its existing `-objects`
  and `-units` through unchanged, keeping its two independent toggles and its two summary tokens.
  **Why the value is a parameter and not a method on the result.** A `MapView.SetMarkers` the caller
  invokes afterwards is precisely the shape that produced the defect: the wiring exists, the front-end
  does not call it, and nothing says so. A parameter cannot be forgotten — a new front-end has to state
  what it wants — and the derivation is unreachable except through the one path.
  **Why one flag on the game and two on the viewer.** The two toggles are a developer tool's surface
  (0009 FR-4 requires each overlay usable without the other, and mapview's summary reports each count
  on its own flag). The game has one question — is the instrument on — and answering it with two flags
  would put a diagnostic control panel in a game front-end.
  **Why `-markers` and not a name that already exists.** `-objects` and `-units` mean the 0008 and 0009
  overlays on `cmd/mapview`; `-statics` and `-staticmarkers` are the object-art story's, which gives
  `cmd/againrom` neither. Reusing any of the four here would leave one flag name with two meanings
  depending on which binary it was typed at, which is worse than a fifth name.
  **The default is on because the screen is otherwise empty, and that is stated as temporary.** When
  real object art ships, "markers on by default" is a choice to re-make, not a state to discover: the
  contract (FR-13) records that so the flip is a decision with a reason attached.
  **Why the derivation stays independent — the load-bearing half.** A marker is placed from its
  placement record's anchor cell and the centre of that cell. It reads no class field, no sprite anchor
  and no frame geometry, and it goes through no function shared with art placement. Unifying the two
  behind one "where does this thing stand" helper is the tempting refactor and it would destroy the
  instrument: a wrong shared anchor moves marker and art together, the screen stays self-consistent, and
  the disagreement that catches a two-tile error becomes unrepresentable. The art story makes the same
  call from its own side, so this is a boundary both stories hold, not a preference of one.
  *Rejected:* markers unconditionally on inside the load path with no parameter — it would take
  `cmd/mapview`'s `-objects`/`-units` away and change a shipped tool's observable behaviour (FR-4a);
  *rejected:* two flags on `cmd/againrom` mirroring the viewer — see above; *rejected:* reporting the
  marker state in the `-check` line — AC-6 pins that line's two counts, and the marker wiring is
  witnessed by SC-19 without spending the one line the headless mode prints.

## Success criteria

Each is a named test (automated) or a named developer-run observation (manual), against the requirement
it serves.

1. **SC-1 — the map list (AC-1a, FR-2a).** *Automated.* `TestBuildMapList` in `pkg/game`: over two
   synthetic `NamedReader`s holding maps with recorded names, maps with an empty recorded name, one
   source text present in **both** sources, two loose names differing only by letter case, and bytes
   whose metadata will not decode — the result is one list in the DD19 order, **every loose row before
   every archive row** and each group ordered case-insensitively by source text; every input contributes
   **exactly one** row; the two equal source texts give two rows with the loose one first; the empty-name
   row is present and choosable; the undecodable rows are present, distinguishable in `Text()` and not
   choosable. Plus `TestDirMapsScan`: on a temp dir, a `.ALM` in any case is kept, a `.LM` near-miss is
   not, and a `.alm` inside a **subdirectory** is not.
2. **SC-2 — the virtual frame (AC-2, FR-7, P-2).** *Automated.* `TestFitAndMapping` in
   `pkg/render/frame`: at window sizes wider than, taller than, exactly 4:3 with, and smaller than
   640×480 — including the sizes where a float implementation provably fails (700×500, 644×494,
   642×481) — the frame is centred and wholly inside the window with no stretching; `WindowToFrame`
   reports no frame pixel for every letterbox position and every position outside the window;
   `WindowToFrame ∘ FrameToWindow` is the identity wherever `FrameToWindow` reports `ok`, and `ok` is
   false exactly where no window pixel maps to that frame pixel; a 1×1 window still places the whole
   frame inside itself; and a non-positive window is `Valid() == false` with every mapping refusing.
   **Every mapping assertion is exact — no tolerance.** Containment and centring are asserted on the
   **exact rational** the test derives for itself, where they hold by construction; `Scale()` is
   asserted exactly and `Origin()` is asserted non-negative exactly and within one ulp of the exact
   origin, because the draw transform's far edge cannot be exactly bounded in `float64` (DD5).
3. **SC-3 — the screen flow (AC-3a, FR-3, FR-7, FR-9a, P-1).** *Automated.* `TestFlow` in `pkg/ui`: a
   fresh flow **starts on the menu**; NEW GAME moves menu→picker; choosing a loadable row moves
   picker→map; choosing a row whose loader **fails** leaves the flow in the picker with a non-empty
   message and that row no longer choosable, without exiting; Esc moves map→picker and picker→menu; Esc
   at the menu reports exit; Esc on no other screen does; the same event sequence from the same start
   state always gives the same result; and **the flow exposes no activation for any button other than
   NEW GAME**, so a release on one of the six unbound buttons leaves the screen, the row set and the
   message unchanged. (EXIT is dispatched in `App`, not in `flow` — DD31 — and is SC-17's subject.)
4. **SC-4 — the picker model (AC-4, FR-3).** *Automated.* `TestPicker` in `pkg/ui`, exercised at 38 rows
   where the list is longer than the window: `Move` clamps at both ends; the scroll window follows the
   selection minimally and never shows a row that does not exist; `Select` sets an absolute index,
   scrolls it into view and refuses an out-of-range one; `RowAt` agrees with `Visible()` at the first
   row, the last visible row, one line above the list, one line past it, and past the last row; the
   click path `RowAt → Select → Choose` reaches exactly the row under the cursor; `Choose` yields
   nothing on an unchoosable row and the row's index on a choosable one; `SetUnusable` demotes a
   previously choosable row.
5. **SC-5 — the standalone viewer is unchanged (AC-5a, FR-4a).** *Automated, plus one mechanical diff
   check.* The nine existing `cmd/mapview` tests pass **unmodified**, pinning the summary text, the flag
   behaviour, the cadence clause and the failure paths; `TestFlagSet` in `cmd/mapview` pins the exact
   flag-name set and the exact `usage` string; `git diff` against the pre-story baseline is empty for
   `cmd/mapview/main_test.go` and for the Esc branch of `Viewer.Update` and the whole of `Viewer.Run`.
   *Recorded limitation:* Esc-closes-the-window has **no** automated pin and never had one (DD24); it is
   carried by SC-14.
6. **SC-6 — one camera step, reachable headlessly (AC-5a, FR-4a).** *Automated; two parts, and the
   second is deliberately modest.* `TestViewerStep` in `pkg/ui` drives `step` with synthetic `Input` and
   asserts the *behaviour*: pan on each key, edge-scroll only while the cursor is inside the view and
   with the right axis and sign at all four edges, wheel zoom about the cursor within the zoom limits,
   clamping at the world edges, and the water counter advancing from the injected timestamp — none of
   which had any automated coverage before. `TestMapScreenRoutesThroughStep` then puts an `App` on the
   map screen, calls `App.step` with an `appInput`, and asserts the resulting camera state equals that
   of a standalone `Viewer` given the same `Input` and the same `Layout` call. That second test is a
   **structural** witness — it proves the front-end routes camera input through the same method and
   would fail the moment the front-end grew its own camera handling — not an independent proof of
   camera behaviour, which is the first test's job.
7. **SC-7 — mask selection (AC-9, FR-8, P-3).** *Automated.* `TestButtonAt` in `pkg/render/menu`: a
   synthetic 640×480 index mask carrying each of the eight hot indices, index 0, the edge ramp
   `0x10`…`0x1e` and assorted strays selects buttons 1…8 at the hot indices and **nothing** at every
   other value; every point outside the frame selects nothing; the same mask decoded with a
   **non-identity (all-black) palette** gives identical results (DD6's palette-independence witness);
   and a **vertically mirrored** mask changes which button a fixed point selects (DD7's row-order
   sensitivity witness).
8. **SC-8 — overlays and the latch (AC-10, FR-8, FR-9a, P-5).** *Automated.*
   `TestSelectionAndCompose` in `pkg/render/menu`: hovering composites that button's hover bitmap 1:1 at
   its hover rect; pressing composites its pressed bitmap at its **pressed** rect; moving off while held
   leaves the frame equal to the base and lights no other button; releasing off the latched button
   activates nothing; releasing on it activates it and reports **which** button, so a caller can tell
   NEW GAME from EXIT and both from the six unbound buttons; a press on no button latches nothing while
   hover keeps tracking; `Overlay` never yields two images; and with nothing selected `Compose` is
   pixel-identical to the base.
9. **SC-9 — NEW GAME by rectangle (FR-9a, AC-11's automatable half).** *Automated.* `TestNewGameButton`
   in `pkg/render/menu`: the computed button's hover rect is `(112,64)…(324,200)`; it is in the left
   column and is the smallest-`y` entry of that column; it is not the bottom-left entry; a
   smallest-`y`-over-the-whole-table rule, computed independently in the test, picks a **different**
   button; and both placement tables match the decoded values entry by entry. "Never identified by a
   mask index" is **structural, not asserted** — see DD8: no mask value is in scope where the button is
   computed, so any test of it would pass by construction.
10. **SC-10 — incomplete or inconsistent assets (AC-12, FR-10, P-4).** *Automated.* `TestLoadRejects` in
    `pkg/render/menu` and `TestOpenArchives`/`TestFrontEndRejects` in `pkg/game`: a missing entry, a base
    or mask that is not 640×480, an overlay whose decoded dimensions disagree with its own
    placement-table row, and each of the three missing archives yield a non-nil error and a **nil**
    asset set or front-end — never a partial one — with the offending entry or path named; and the
    failure reported is the **first** in the DD28/DD17 order, asserted by a set with two defects.
11. **SC-11 — the headless mode and the asset-root contract (FR-1, FR-5, AC-6's automatable half,
    AC-12).** *Automated.* `TestCheck` in `cmd/againrom`: over a synthetic install (three synthetic
    archives and synthetic map bytes on a temp dir) `-check` prints the DD21 line with both counts
    correct, exits 0 and opens no window; **the same run works with the root supplied by `-assets` and
    by `AGAINROM_ASSETS`, and the flag wins when both are set**; a mask with one button region empty
    prints `7 of 8` and still exits 0; no asset root at all is reported and exits non-zero; and each
    FR-10 failure prints to **stderr** and exits non-zero in both the windowed and the `-check` mode.
12. **SC-12 — the architecture stays enforced.** *Project-mechanics gate (AGENTS.md), not a spec
    criterion — recorded as such.* `internal/archtest` is green with `pkg/render/frame` and
    `pkg/render/menu` registered as stdlib-only, `pkg/ui` gaining no import outside
    `pkg/render*`/Ebitengine, and `pkg/game`/`cmd/againrom` within their existing wildcards. The second
    half — `docs/ARCHITECTURE.md` listing both new packages in both of its tables — is a **reviewed
    edit, not a test**, because nothing checks the doc against the allow-map.
13. **SC-13 — headless evidence on a lawful install (AC-6).** *Developer-run.*
    `againrom -assets <root> -check` prints **38** map rows (10 loose + 28 in `scenario.res`) and
    **8 of 8** buttons with a mask region, and exits 0 without opening a window. The evidence this
    criterion produces is the observed line verbatim and the observed wall time (R-5). It has no "or a
    limitation" branch: the install is available and the run is headless.
14. **SC-14 — live front-end evidence (AC-7a).** *Developer-run, needs a human at a window.* Running
    `againrom -assets <root>`: the menu appears; NEW GAME opens the picker listing the installed maps;
    a chosen map loads into the interactive map screen; Esc unwinds map→picker→menu and then exits.
    **If it is not run**, it is recorded as an explicit pending limitation naming what was not observed
    — the whole engine-facing dispatch and presentation layer (R-9) and Esc-closes-the-window (SC-5's
    recorded gap) — AC-7a is not marked passed, and no conclusion rests on it.
15. **SC-15 — the load-bearing manual criterion (AC-11).** *Developer-run, needs a human at a window.* A
    human rests the cursor on the **visually top-left** brooch button and confirms the highlight appears
    over `(112,64)…(324,200)` and **not** over the bottom-left one; clicks it and confirms the picker
    opens; and hovers each of the other seven confirming each lights its own rectangle and no other.
    This is the only place the *inferred* mask row order and the **Medium** hover/pressed roles meet the
    game. **If it is not run**, the pending declaration must state precisely what stays unproven — that
    the visually top-left button is the one that lights, and that hover and pressed are not swapped —
    AC-11 is not marked passed, and it is never marked passed on the strength of a synthetic test,
    because no synthetic test can distinguish either case.
16. **SC-16 — the picker states its extent (AC-13, FR-2a).** *Automated.* `TestPickerHeaderText` in
    `pkg/ui`: at 38 rows the header names the total and the **1-based** range `Visible()` reports, and
    the two agree at the top of the list, after scrolling to the middle, and at the bottom; the numbers
    are read out of the header text and compared against `Visible()` rather than against a literal, so
    the header cannot claim a range the draw path does not render; a list shorter than the window
    reports its whole self; and an empty list reports no range at all. The boundary that matters is the
    1-based/0-based one, and it is asserted at both ends of a scrolled window.
17. **SC-17 — EXIT quits (FR-11, DD31, AC-14's automatable half).** *Automated.* `TestAppDispatch` in
    `pkg/ui` gains three arms: a press and a release both on `menu.ExitButton` makes `App.step` report
    exit; a press on it released **away** from it does not, and changes nothing; and every one of the
    six buttons that is neither NEW GAME nor EXIT is pressed and released with no exit and no screen
    change. `TestExitButton` in `pkg/render/menu` pins the constant against the decoded contract: it is
    in `[1, ButtonCount]`, it is not `NewGameButton`, and its hover rectangle is entry 8 —
    `(324,276)…(532,412)`, the bottom entry of the right column — which is the consequence AC-14 asks a
    human to confirm on the real brooch.
18. **SC-18 — the wheel scrolls the picker (AC-15, FR-12, DD32).** *Automated.* `TestAppDispatch` in
    `pkg/ui` gains a wheel arm: on the picker, a positive `WheelY` moves the selection towards row 0 by
    `pickerWheelRows` and a negative one towards the last row, with `Visible()` following; rolling past
    either end clamps rather than wrapping and chooses nothing (the screen is still the picker); and the
    same input on the **menu** and on the **map** screen leaves the picker's selection and window
    untouched, which is the boundary that would catch the wheel being read one level too high.
19. **SC-19 — the markers reach the game's map screen (AC-16, FR-13, P-6, DD33).** *Automated.*
    `TestLoadMapViewerMarkers` in `pkg/game`, over a synthetic map carrying placed type-4 and type-6
    records: asked for them, the loaded viewer reports both overlays on holding exactly one cell per
    record; asked for neither, both are off; and `TestMarkerCells` asserts each cell is the record's own
    stored anchor shifted down by 8 and nothing else — including a record whose anchor is not on a cell
    boundary, where any extra term would show. `TestFrontEndMarkers` (internal, over a synthetic
    install) drives the front-end's own loader and asserts the default is on and that a `Markers{}`
    front-end loads a map screen with both off — the boundary that would catch the wiring landing in the
    library but never being reached from the game. `TestMarkerFlag` in `cmd/againrom` asserts the parsed
    default is on, that the off form parses, and that the value reaches the front-end.
    *Recorded limitation:* that the markers are visible **on screen** is not automatable here — the same
    R-9 gap the rest of the front-end has; the overlays' own draw path and geometry are 0008/0009/0015's
    and are untouched by this revision.

## Risks (product)

- **R-1 — the mask's stored row order is inferred, and a wrong inference silently swaps top for bottom.**
  Under a flipped read the visually *bottom*-left brooch button becomes the one that opens the picker,
  the visually *top*-right one becomes the one that quits, and every synthetic criterion still passes.
  **What the geometry anchoring does and does not buy, stated precisely.** `ButtonAt` answers with a
  mask-derived number and every downstream decision — which overlay rectangle lights, and which action a
  release performs — keys off that number. So under a flip the screen area `(112,64)…(324,200)` would
  light the *bottom*-left overlay and would open nothing: it is **not** true that "a wrong row order
  changes only which art lights". What DD8 actually buys is **falsifiability**: because NEW GAME is
  bound to the table row rather than to an index literal, a flip produces a visible mismatch — the gem
  you hover lights a rectangle somewhere else — instead of silently and consistently relabelling the
  brooch, which is what option A would have done. *Mitigations:* the decode convention is written in one
  place (DD7); SC-7's mirrored-mask case proves the code is sensitive to row order so SC-15 can falsify
  it; and until SC-15 and AC-14 are run against the game, the correspondence is declared unproven.
- **R-2 — the hover/pressed roles rest on a Medium claim.** If the hover and pressed bitmaps are
  swapped, the menu shows the pressed art on hover and vice versa, and no synthetic test can tell.
  *Mitigation:* SC-15 is the only witness and is declared pending until run; nothing else in the story
  depends on which of the two is which.
- **R-3 — opaque composition may show a seam.** No transparency rule is decoded, so an overlay is copied
  as an opaque rectangle. If the original blends, a rectangle edge may be visible. *Mitigation:* this is
  a discrepancy SC-15 surfaces and reports, **not** a licence to invent a colour key (DD10); a real seam
  becomes a research question, not a patch.
- **R-4 — moving the load path could change the standalone viewer.** Any drift in `cmd/mapview`'s
  summary, flags, error strings or Esc behaviour violates FR-4a. *Mitigations:* the nine existing tests
  are the pin and are edited by nothing; the summary literal and the `decode` wrapping never leave the
  command (DD20, DD25); a flag-set snapshot test catches a flag change the summary tests would miss; the
  `Update` characterization pin lands before the restructuring (DD24). **Residual:**
  Esc-closes-the-window has no automated pin in this repo and never had one; it is carried by textual
  identity plus SC-14.
- **R-5 — startup reads 38 maps.** The list is built on every start, including `-check`. *Mitigation:*
  the listing pass decodes only the type-0 metadata (DD11), not the grids or content tables; the
  observed `-check` wall time is part of SC-13's evidence, so the claim is measured, not assumed.
- **R-6 — 38 rows do not fit the frame.** A picker that draws only what fits and hit-tests what it drew
  can silently make late rows unreachable, and a picker with no visible selection marker cannot be
  operated at all. *Mitigation:* one scroll window shared by the draw path and the hit-test through
  `Visible()`, plus an explicit selection marker (DD13), asserted at five boundary positions at 38 rows
  (SC-4).
  **This risk fired, and the mitigation above did not cover the failure mode that mattered.** Every row
  was reachable and the scroll window was correct; what was missing was any sign *on screen* that there
  were more rows. With the pre-revision ordering the first 25 rows were also the 25 rows with no
  recorded name, so the screen was a complete-looking list of nameless entries and the owner reported
  both "no map names" and "not all 38 maps are there". *Added mitigations:* named rows are ordered onto
  the first screenful (DD19), and the header states the total and the visible range (DD13, SC-16).
  **Recorded lesson:** "every row is reachable" is a property of the model; "a user can tell there are
  more rows" is a property of the screen, and only the first had a criterion.
- **R-7 — the letterbox mapping must not lose a pixel row.** A round-trip that fails at an edge would
  make a button unclickable along one row. *Mitigation:* the mapping is exact integer arithmetic with
  the inverse defined as membership in the forward map's image, so the property holds by construction
  rather than by tolerance (DD5), and SC-2 asserts it at the sizes where the float form provably fails.
- **R-8 — a decoded bitmap could be written to the tree.** Menu art is game data. *Mitigation:* no code
  path this story ships writes an image file (DD23); the guard runs in tree and `--history` mode before
  every push. **Residual, recorded honestly:** the guard's extension list covers neither `.bmp` nor
  `.png`, so it would not catch a developer who extracted menu art by hand. That gap predates this story
  and closing it would change 0000's contract; it is reported rather than silently widened.
- **R-9 — the engine-facing shell is exercised but not observable in tests.** Ebitengine draw calls
  (`NewImage`, `WritePixels`, `DrawImage`, `Fill`, `DebugPrintAt`) all **run** without a graphics
  context, so an `App` is constructible and `App.Draw` callable in a test; what is impossible is
  **reading pixels back** from an `*ebiten.Image` (`ReadPixels`/`At` panic before the game starts), and
  `App.Run` opens a real window. *Mitigation:* every decidable thing — the frame mapping, the flow, the
  picker, the composition, the latch, the dispatch — is pure and asserted below or beside the engine
  (`menu.Compose` returns an `*image.RGBA`, which *is* readable; `App.step` takes an injected snapshot),
  and `app_test.go` calls `App.Draw` only as a no-panic smoke check. What SC-14 exercises is the thin
  presentation layer alone; if SC-14 does not run, that is the layer declared unevidenced, rather than
  the whole front-end.
