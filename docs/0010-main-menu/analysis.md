# Analysis — main menu with map picker and NEW GAME

## Intensity & terrain (declared for the whole work item)

- **Intensity: `spec-first / static`.** Per `SDD/PROFILE.md`, a front-end/UI delivery is "other engine
  work" — a bounded delivery whose behavior settles quickly; upfront clarity helps, and the spec is a
  thinking-and-alignment tool for *this* delivery rather than a durable external contract. It is not
  `no-spec`: the screen flow, the picker's ordering rule, the letterbox mapping, the latch semantics and
  the geometry-not-index anchoring of NEW GAME are all choices a reader could not recover from the diff,
  and the delivery spans five packages and two commands. It is not `spec-anchored` either: nothing
  outside this repo consumes the front-end's shape, no second team is aligned by it, and the one durable
  contract in play — the `main.res` menu composition — lives **upstream in the research submodule**
  (`formats/menu/format.md`), not here. No watcher tool exists in this repo, so synchronization is
  **static** — discipline, not tooling — and is labelled that way.
  *Sibling check:* 0005, 0006, 0008 and 0009 all declared `spec-first / static` for viewer/UI work and
  none of their specs has since needed maintenance. **The one honest pull toward `spec-anchored`** is
  FR-4a: a two-entry-point non-divergence invariant is exactly the sort of thing a maintained spec
  usually guards. It does not flip the call, because the invariant is discharged **structurally** — both
  entry points construct the same viewer type through one shared load path, so divergence becomes a
  compile-or-test failure rather than a document nobody re-reads. If a later story finds itself
  re-litigating FR-4a in prose, that is the signal this call was wrong, and it should be recorded then.
- **Size is not an intensity axis.** This is the largest story the project has attempted, and that
  changes the *task decomposition* budget, not the intensity: the playbook's signal is the class of
  ambiguity, not the code size.

**Terrain: declared per changed area, not per ticket** — this story genuinely holds both.

| Area | Terrain | Why |
|---|---|---|
| New menu/frame render packages, the front-end screens, `cmd/againrom` | **greenfield** | No behavior to preserve; `cmd/againrom` today resolves an asset root, prints it and exits. |
| `cmd/mapview` + the load path FR-4a shares | **brownfield** | FR-4a states the standalone viewer's flag set, `-check` summary text and Esc behavior stay exactly as they are. Any restructuring that pulls its load path into a library tier changes shipped code whose behavior is contractually frozen. AC-5a is the contract-level pin and `cmd/mapview/main_test.go` the executable one; both must stay green **unmodified**. |
| `pkg/ui`'s existing `Viewer` | **brownfield** | The front-end needs the viewer's camera behavior *without* its Esc-terminates-the-window behavior (FR-7 vs FR-4a). Whatever separates the two touches a shipped run loop; `pkg/ui/viewer_test.go` and `pkg/ui/overlay_test.go` are the pin. |
| `pkg/formats/alm` | **brownfield, if touched at all** | FR-2/FR-3 distinguish "the metadata read" from "the full decode" (see *A consequence the spec forces*, below). If the plan satisfies that by adding a metadata-only entry point to the map reader, that is an **additive** change to a spec-anchored package (0003) whose `alm_test.go` is the characterization pin. Whether it is touched is a plan decision; `analysis.md` is not where a design decision is made. |

For the brownfield areas the discipline of `sdd/brownfield.md` applies: every changed unit is pinned by
its existing tests **before** the change, and no existing test is edited to accommodate a new one.

## Staleness check — EXP-0030 does not reach this story

Checked explicitly rather than assumed, because the EXP-0030 container-framing correction (`-8` on every
pre-EXP-0030 `.alm` payload offset) forced a documentation relabel on 0007 and 0009 at this repo's last
two stories:

- **Clear.** `spec.md` for 0010 carries **no submodule pin hash and no `+0x` byte offsets**. The menu
  contract it reproduces (`MENU-*`) is about `main.res` bitmaps, a hit mask and two `rom.exe` placement
  tables — none of which the `.alm` framing touches.
- The story's only `.alm` dependence is `ALM-META-010` (the recorded map name) and `ALM-LOC-007` (10
  loose + 28 archived maps), consumed **through `pkg/formats/alm` as landed**, which is already at the
  corrected framing (0003, relabelled and re-verified in the 0007/0009 boundary work). `ALM-META-010`'s
  own row carries its `-8` amendment explicitly (name `@+0x30`, description `@+0x78`), and the promoted
  spec `research/formats/alm/format.md` agrees; there is nothing unreconciled and nothing for this story
  to edit.

## Source & confidence

All menu facts below are game-derived by the research team, not by this repo's own reverse-engineering.

- **Submodule pin:** `research/` at `da54e6d` (`againrom-research`, module `rom1research`) — research
  `master` HEAD at this story's boundary, and **frozen for the story's duration** (S-5). This story does
  not touch the submodule or the research repo.
- **Experiments:** `EXP-0026-mainmenu-assets` (the 18-file inventory and the loader),
  `EXP-0027-menu-hitmask` (the palette, the index-to-button map, the interaction state),
  `EXP-0028-menu-overlay-geometry` (the two static placement tables and the 1:1 draw rule). All three are
  promoted into `research/formats/menu/format.md`.
- Supporting, consumed only through already-shipped code: `EXP-0007` / `EXP-0010` / `EXP-0030` (the
  `.alm` container and its type-0 metadata) and `EXP-0001`/`EXP-0017` (the `.res` container).

## Claim inventory

| Claim | Statement | Confidence | Used by |
|---|---|---|---|
| `MENU-ASSET-001` | The main menu is the **18-file** `graphics/mainmenu/` subtree of `main.res`: `menu_.bmp` (640x480, 24 bpp base), `menumask.bmp` (640x480, **8 bpp** hit mask), `button1..8.bmp` (24 bpp hover) and `button1..8p.bmp` (24 bpp pressed). All 18 sha256-distinct; the loader `R1151` in `rom.exe` requests exactly these names | **High** | The asset set FR-1/FR-10 require and validate; the entry names the loader asks the archive for |
| `MENU-ASSET-002` | There are **exactly 8 buttons** — the loader's `sprintf` loop runs `n = 1..8` (a do/while with start 1 and bound `n < 8`), storing hover overlays at `this+0x6c`, pressed at `this+0x80`, mask at `this+0xbc`, background at `this+0xb8` | **High** | The fixed count of 8 in FR-5, FR-8, FR-9, AC-9, AC-10 |
| `MENU-MASK-003` | The 256-entry palette of `menumask.bmp` is the **identity grayscale ramp** `palette[i] = (i,i,i)` for all 256 entries — it carries no colour meaning; the raw **8-bit index** is the semantic. `rom.exe` loads it via a distinct 8-bit loader (`R1152`) that keeps raw indices (`w*h` bytes at `maskObj+0x10`) and never renders it to colour | **High** | The spec's hard constraint that the mask is consumed as raw indices; why a colour-based read is wrong *by construction* rather than merely slow |
| `MENU-MASK-004` | The hit-test `R1148` reads `idx = maskBuf[(y-top)*640 + (x-left)]` and maps **0x80 to 1, 0x90 to 2, 0xa0 to 3, 0xb0 to 4, 0xc0 to 5, 0xd0 to 6, 0xe0 to 7, 0xf0 to 8** (asset number = `idx/16 - 7`). Index 0 and **every** other value — the anti-alias edge ramp `0x10,0x12,...,0x1e` and stray pixels — hit the switch default = no button. Only 8 of the 43 index values present in the shipped mask are hot; each hot region is bracketed by its button's normal placement rect, 8/8 | **High** | FR-8's selection rule, AC-9, P-3 |
| `MENU-GEOM-005` | Overlay placement is **two static `rom.exe` tables**, contiguous, 8 entries x 16 B = `{x,y,w,h}` int32 LE: normal/hover at `L06179` (file `0x198ed8`), pressed at `L06180`. `R1153` places button *i* at `(x,y)` through `(x+w,y+h)` offset by the menu origin; **origin = (0,0)** | **High** | The two 8-entry tables the spec reproduces; the frame origin FR-8 samples the mask at |
| `MENU-GEOM-006` | Overlays draw **1:1** — each table entry's `(w,h)` equals the corresponding BMP's own pixel dimensions, **16/16** exact across both tables; and each normal rect **brackets** its button's mask region, **8/8** | **High** | FR-8's "1:1 in frame pixels, nothing resampled"; FR-10's per-overlay dimension check, which is exactly this identity turned into a startup assertion |
| `MENU-STATE-007` | `button%d.bmp` is the **hover** overlay and `button%dp.bmp` the **pressed** overlay: in `R1148`, not-pressed and hovering *i* draws `this+0x6c[i]` at normal rect `this+0x94[i]`; mouse-down while still hovering the latched *i* draws `this+0x80[i]` at pressed rect `this+0xa8[i]`. `this+0xe4` is a per-button **disable bitfield** (bit set suppresses the overlay). The click dispatcher `R0820` posts a per-button command message; button 8 posts `WM_CLOSE` (quit); the other message ids are undecoded | **Medium** | The hover-vs-pressed **roles** and the latch semantics FR-8 requires — the story's one Medium dependency. The disable bitfield and the per-button commands are the two pieces the spec deliberately scopes **out** |
| `ALM-LOC-007` | `scenario.res` is an `&YA1` archive holding **28 embedded campaign maps** (`N.alm`) plus 3 nested `&YA1` nodes; **38 maps total** conform to the container model | **High** | FR-2's inventory and AC-6's headless count (10 loose + 28 archived) |
| `ALM-META-010` | Type-0 `+0x30` = ASCII **map name** (64-B NUL-terminated, at most 21 B used, **empty on most campaign maps**); `+0x78` = description (64-B, ASCII or CP1251). Offsets carry the EXP-0030 `-8` amendment | **High** | The recorded name each picker row shows, and why FR-2 must treat an empty name as normal rather than as an error |
| `RES-*` (0001) | The `&YA1` container: header, registry nodes, CP866 names, payload ranges | **High** | Opening `main.res`, `graphics.res` and `scenario.res` and reading their entries |

Supporting evidence recorded by EXP-0026 that the story consumes indirectly (as the *shape* FR-10
validates, never as literals compiled into shipped source): every overlay BMP's stored dimensions equal
its placement-table `(w,h)`; the base and the mask are both 640x480; the mask is the only 8-bpp file of
the eighteen.

## What this story consumes, and what it deliberately does not

**Consumed.** The eighteen entry names; the eight hot mask indices and the "everything else selects
nothing" default; the two 8-entry placement tables; the 1:1 draw rule; the (0,0) frame origin; the
hover-vs-pressed role assignment. Plus, through already-shipped code, `.res` entry enumeration and the
`.alm` type-0 map name.

**Not consumed, and not invented.**

- **No transparency or blend rule.** The research establishes that the overlays are opaque 24-bit
  rectangles and explicitly leaves any blend the original may apply outside what it has established.
  Nothing in `EXP-0026`/`EXP-0028` records a colour key, an alpha channel or a raster-op. The spec
  therefore composites opaquely and treats a visible seam as a *discrepancy the manual criterion
  surfaces*, not as licence to invent a colour key. Inventing one would be a golden-rule-4 violation.
- **No per-button command binding.** `MENU-STATE-007` decodes only button 8 to `WM_CLOSE`, at Medium. The
  spec binds no command at all, so Esc at the menu stays the single way out of the program.
- **No disable bitfield source.** The *existence* of `+0xe4` is decoded; *which* buttons start disabled
  is listed under "Open / not established" in `research/formats/menu/format.md`. Every button here is
  therefore treated as enabled and nothing suppresses an overlay.
- **No font, no menu text.** The picker's text is placeholder debug text; the `.16` font glyph grammar
  is an open research item (`SPR16A-FONT-007`).
- **No unit or structure art.** The map screen draws terrain only.

## Decoded-vs-open reconciliation

1. **The mask's stored row order is *inferred*, and this is the story's sharpest open edge.**
   `research/formats/menu/format.md` section *Open / not established* says plainly: "The DIB loader's row
   order (top-down vs flip) is inferred from the 8/8 placement bracket rather than read directly; the
   screen y here is top-down." The inference is that each button's normal placement rectangle brackets
   its mask region — which holds 8/8 under the assumed order.
   **Why that check is weak here, specifically.** The layout is near-symmetric about the horizontal
   midline: a vertically flipped read maps 1-4, 2-3, 5-8 and 6-7 onto each other to within a few
   pixels, so the flipped reading would very nearly satisfy the same bracketing test. The evidence
   therefore under-determines the row order, and any requirement anchored to a **mask index literal**
   would be one the layout evidence cannot falsify. That is the whole reason the spec anchors NEW GAME to
   the **placement rectangle** instead, and why its AC-11 exists: hovering the visually top-left brooch
   button in the real game is the first observation that can falsify the inferred order.
   *Second-order note for whoever implements the decode:* a standard BMP with a positive DIB height
   stores rows bottom-up and a conforming decoder normalises them to top-down. The raw-index loader of
   the original may or may not perform that flip. Our decode must therefore state which convention it
   uses, and AC-11 is where the choice meets the game.
2. **The hover/pressed roles rest on a Medium claim.** `MENU-STATE-007` is the only Medium claim the
   story depends on, and it is Medium as a whole — the field offsets are read from the disassembly, but
   the *roles* ("this array is hover, that one is pressed") are read off the branch structure of one
   function. If the two were swapped, every unit criterion in the spec would still pass: the synthetic
   assets cannot tell hover art from pressed art. AC-11 is again the only place this is testable.
3. **A consequence the spec forces, worth naming before planning starts.** FR-2 lists a map "whose
   metadata cannot be read" as unreadable-but-listed, while FR-3 separately handles "a chosen map that
   passes the metadata read but then fails to decode fully". Those two clauses are only distinguishable
   if the listing pass decodes **strictly less** than a full map load. With the shipped reader, a full
   `alm` decode is all-or-nothing, so FR-3's clause would be unreachable. This is a real design
   consequence of the spec, not a defect in it — the spec is stating an observable contract, and it is
   the plan's job to choose how the listing pass reads less. The options are visible from here (a
   metadata-only entry point in the map reader; or a listing pass that accepts the full decode and
   leaves FR-3's clause defensive) and are **not** chosen in this file.
4. **The "empty on most campaign maps" half of `ALM-META-010` is load-bearing, not a footnote.** 28 of
   the 38 maps are the archived campaign maps, and most record no name. A picker that treated an empty
   name as a failure would drop the majority of a stock install's maps; FR-2 says so explicitly, and
   AC-1 pins it.
5. **Nothing about *how the menu looks* is a research claim.** The research decodes which bitmaps compose
   the menu and where they go. That the visually top-left brooch button is NEW GAME is the **owner's own
   gameplay knowledge of their lawful install** — the spec attributes it that way and it asserts nothing
   about any decoded byte. The letterboxing, the picker's layout, the debug text and the window sizing
   are the project's own design and no fidelity claim is made for any of them.

When the research team reads the DIB row order directly, closes the disable bitfield, or decodes the
per-button command ids, the flow is: repull the submodule (`git submodule update --remote research`),
derive the newly decoded fact here, then revise `spec.md` from the earliest affected stage per the SDD
revision process.

## Code baseline (what exists at this story's boundary)

Recorded because the spec's *current behavior* section asserts it and because three of the four changed
areas are brownfield. Verified against the tree, not from memory.

- `cmd/againrom/main.go` is 22 lines: it parses `-assets`, falls back to `AGAINROM_ASSETS` through
  `game.ResolveAssetRoot`, prints `againrom: asset root = <root>` and returns. It opens no window,
  imports no engine, and — worth noting — **returns rather than exiting non-zero** when no root is
  configured.
- `pkg/game` holds exactly one function, `ResolveAssetRoot(flag, env) string`. The tier is otherwise
  empty, and the DAG already lets it import any `pkg/*`.
- `cmd/mapview` owns the entire load path: resolve `graphics.res` from the asset root, `res.Open` it,
  `terrain.LoadTileset`, `os.ReadFile` plus `alm.Open` the map, pick a window title (`Map.Name`, else
  the file's base name), `ui.NewViewer`, then build a one-line summary and either print-and-exit
  (`-check`) or `Run()`. Nothing in a library package performs that sequence.
- `pkg/ui.Viewer` is the windowed viewer: an Ebitengine `Game` whose `Update` handles Esc (returning
  `ebiten.Termination`), water-tick advance, key/edge pan and wheel zoom, and whose `Layout` adopts the
  window size as the camera view size. Esc-terminates is **inside** `Update`, not injected.
- `pkg/render/camera` is the pure camera model (stdlib only); `pkg/render/terrain` is the pure terrain
  pipeline (stdlib only), including `DecodeBMP8`, an 8-bpp Windows-BMP decoder that returns an
  `*image.Paletted` in top-down display order with **raw palette indices preserved** and rejects
  negative (top-down-stored) heights.
- `pkg/formats/res.Archive` exposes `Entries() []Entry` and `ReadFile(name)`; **entry paths are
  normalised to lower case with `/` separators**, and lookup is case-insensitive. So an archive row's
  source text is already lower-cased by the reader, and CP866 node names are decoded to UTF-8.
- `pkg/formats/alm.Open(data)` performs a full decode — header, ten record headers that must tile to
  EOF, the 632-byte type-0 metadata (`Width`, `Height`, `Name`, `Description`, counts), three
  `W x H` grids, and the type-4/5/6 content tables. There is no partial or metadata-only entry point.
- `internal/archtest` is **fail-closed**: every module package except those under `internal/` must
  appear in its allow-map or the test fails. `pkg/ui` may already import `pkg/render` and anything under
  `pkg/render/`; `pkg/game` and `cmd/againrom` may import any `pkg/*`. Ebitengine is permitted only to
  `pkg/ui`, packages under it, and the `cmd/` tier.
- `docs/ARCHITECTURE.md` documents the same DAG in prose and states that the check is authoritative when
  the two disagree.
- The asset guard `scripts/check-no-game-assets.sh` keys on **tracked paths** with a fixed extension
  list (`res|reg|alm|16a|256|spr|pal|fnt|fon|wav|snd|ogg|mp3|smk|bik|mpg|avi|exe|dll`). **`.bmp` and
  `.png` are not in that list**, and `.gitignore` does not ignore them either. That is a pre-existing
  gap, not one this story creates (0004 already handles BMPs inside `graphics.res`), but this is the
  first story whose subject matter is *loose* bitmap art, so it is recorded here rather than discovered
  later.
- A stock install's asset root holds **10** loose `.alm` files in mixed case (`Beast.ALM`, `Cross.ALM`,
  `Forester.alm`, `Horror.alm`, `Islands.alm`, `Kids.alm`, `Kids2.ALM`, `LuMoir.alm`, `Tomb.ALM`,
  `Waters.alm`) alongside `main.res`, `graphics.res`, `scenario.res` and other archives — and also a
  file named `KIDS.LM`, which is **not** an `.alm` and must not be listed. The case mix and that
  near-miss name are why FR-2's "in any letter case" clause has teeth.
