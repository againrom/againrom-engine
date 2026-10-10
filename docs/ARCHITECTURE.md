# Architecture

`againrom` is an independent reimplementation of the Rage of Mages / Allods engine
in Go. This document describes the module layout and the one-directional
dependency DAG. **The prose here documents the design; the enforcement is the
import-graph test in `internal/archtest`** (spec FR-2). If this document and that
check ever disagree, the check is authoritative and this file is the bug.

## Module and tiers

Go module `againrom`. Library packages live under `pkg/`, executables under
`cmd/`, and non-tier build/test helpers under `internal/`.

| Tier | Package | Responsibility |
|---|---|---|
| formats | `pkg/formats/res` | `.res` archive container |
| formats | `pkg/formats/reg` | `.reg` registry |
| formats | `pkg/formats/alm` | `.alm` map |
| formats | `pkg/formats/spr256` | `.256` paletted sprite decoder |
| formats | `pkg/formats/spr16` | `.16a`/`.16` 16-bit sprite decoders |
| formats | `pkg/formats/databin` | the placeable-definition table: eight groups, eleven collections, framed verbatim |
| formats | `pkg/formats/pal` | the 256-entry colour table a per-class palette file carries, and the one blue-green-red entry layout every decoder reads |
| formats | `pkg/formats/bmp` | uncompressed 24-bit and 8-bit Windows bitmaps |
| formats | `pkg/formats/wav` | the RIFF WAVE chunk walk to PCM samples |
| formats | `pkg/formats/winicon` | the icon group of a Windows executable: every image of it decoded to pixels, for the window's icon |
| vfs | `pkg/vfs` | virtual filesystem over `.res` archives |
| data | `pkg/data` | typed game data (registries, tables) |
| sim | `pkg/sim` | deterministic simulation core |
| random | `pkg/random` | the session's random service: named streams, the seeded and original generators, the bounded-draw primitives |
| mapload | `pkg/mapload` | assemble a playable map from `.alm` |
| mapedit | `pkg/mapedit` | headless edit model over one `.alm` map: typed mutations, undo/redo, every untargeted byte carried |
| render | `pkg/render` | draw the world |
| render | `pkg/render/terrain` | terrain tile graphics: strip slice, tile-word mapping, map composite |
| render | `pkg/render/camera` | the map viewer's camera model: position, pan, zoom, clamping, visible tiles |
| render | `pkg/render/frame` | the fixed 640x480 virtual frame: fit a window, centre, letterbox, map window↔frame coordinates |
| render | `pkg/render/menu` | the main-menu composition contract: asset set, hit mask, placement tables, selection state, frame composition |
| render | `pkg/render/text` | the font model, the placement rule and the blit, as plain data plus arithmetic |
| audio | `pkg/audio` | resampling, positional gain and stereo mixing, as plain data plus arithmetic |
| video | `pkg/video` | bounded presentation-frame transport; decodes in-process through `pkg/video/smacker` as the ordinary path. The windows/386 `cutscenehelper` driving an installed `smackw32.dll` survives only as the `-cutscene-check` gate-time oracle, not an ordinary part of play; no game or simulation dependencies |
| video | `pkg/video/smacker` | pure-Go port of libsmacker: Smacker (`.smk`) bitstream/Huffman-tree/DPCM video and audio decode; no game or simulation dependencies |
| words | `pkg/words` | the engine's own words: one strings table per language, keyed by message id, in the mod text layout |
| locale | `pkg/locale` | the install languages: one record per language (archive entry, selector, table code, base id, code page, font remap) |
| ui | `pkg/ui` | user interface |
| game | `pkg/game` | wire the engine together; top-level config |
| cmd | `cmd/againrom` | the game entry point |
| cmd | `cmd/restool`, `cmd/regtool`, `cmd/almtool`, `cmd/sprtool` | per-format developer dump/view tools |
| cmd | `cmd/classdump` | dump the graphics registries' typed classes; sweep the classes maps place |
| cmd | `cmd/terraintool` | headless terrain render harness (archive + map wiring) |
| cmd | `cmd/mapview` | windowed developer map viewer |
| cmd | `cmd/texttool` | measure and render a shipped font against a lawful install |
| cmd | `cmd/missionrun` | drive a campaign mission headlessly against a lawful install and report its outcome |
| cmd | `cmd/dlgtool` | census the campaign's event-text corpus against a lawful install: the dialogue window's two speaker tests, measured separately |

## Dependency DAG

Each package may import only the standard library plus the intra-module packages
listed. This table is the human-readable form of the `internal/archtest`
allow-map; the two must stay identical.

| Package | May import (intra-module) | Notes |
|---|---|---|
| `pkg/formats/res` | — | stdlib + `golang.org/x/text` only |
| `pkg/formats/reg` | — | **stdlib only**, enforced (`.reg` stores bytes and the parser converts none of them, so it takes no text-encoding dependency) |
| `pkg/formats/alm` | — | stdlib + `golang.org/x/text` only |
| `pkg/formats/spr256` | `pkg/formats/pal` | **stdlib only** (`.256` carries no strings) |
| `pkg/formats/spr16` | `pkg/formats/pal` | **stdlib only**, enforced (neither 16-bit grammar carries text, so it takes no text-encoding dependency) |
| `pkg/formats/databin` | — | **stdlib only**, enforced (the table's names are handed back as the file's own bytes and no character encoding is applied, so it takes no text-encoding dependency) |
| `pkg/formats/pal` | — | **stdlib only**, enforced (a colour table carries no strings, so it takes no text-encoding dependency) |
| `pkg/formats/bmp` | `pkg/formats/pal` | stdlib only |
| `pkg/formats/wav` | — | **stdlib only**, enforced (a WAV carries no text the engine reads) |
| `pkg/formats/winicon` | — | **stdlib only**, enforced (an icon carries no strings, so it takes no text-encoding dependency) |
| `pkg/formats/itemname` | — | **stdlib only**, enforced (0151 T12: a line is stored as the shipped bytes with no code-page conversion — pkg/render/text applies the conversion per drawn byte instead — so this format takes no text-encoding dependency) |
| `pkg/vfs` | `pkg/formats/res` | |
| `pkg/data` | `pkg/vfs`, `pkg/formats/reg` | |
| `pkg/sim` | `pkg/random` | stdlib and the random leaf (determinism wall) |
| `pkg/random` | — | **stdlib only**: a leaf that reads no clock and no operating-system state |
| `pkg/mapload` | `pkg/formats/alm`, `pkg/data`, `pkg/sim`, `pkg/random` | |
| `pkg/mapedit` | `pkg/formats/alm` | stdlib and that leaf only: **no external module**, the `golang.org/x/text` grant stays the formats tier's, which is where both halves of the `.alm` string-field codecs live |
| `pkg/render` | `pkg/sim` (read-only), `pkg/vfs` | |
| `pkg/render/terrain` | `pkg/formats/bmp` | the mapping and composite pieces take decoded images and integers; tile bitmaps decode through the bitmap leaf |
| `pkg/render/camera` | — | **stdlib only**: the camera arithmetic is engine-free and unit-testable without a window |
| `pkg/render/frame` | — | **stdlib only**: exact integer window↔frame mapping; floats appear only in the draw transform |
| `pkg/render/menu` | `pkg/formats/bmp` | takes archive bytes through a one-method `EntrySource` and decodes them through the bitmap leaf |
| `pkg/render/text` | `pkg/locale` | a LEAF by contract (0050 P-5). It holds the font model, the placement rule and the blit as plain data plus arithmetic, and a loader outside it fills that data. Its one edge is the stdlib-only locale table, which says under which selector the font remaps a byte |
| `pkg/audio` | `pkg/formats/wav` | resampling, positional gain (`Place`) and stereo mixing (`Stereo`) as plain data plus arithmetic; no slot, archive, listener or device is a concept this package knows, and pkg/ui supplies the one concrete `Player` this tree ships. WAV bytes decode through the WAV leaf |
| `pkg/video` | `pkg/video/smacker` | stdlib-only leaf for ARV2 frames and bounded playback, decoding in-process through its own `pkg/video/smacker` port; the windows/386 installed-DLL adapter remains only as the `-cutscene-check` gate-time oracle; no UI, archive, game or simulation types |
| `pkg/video/smacker` | — | third-party-derived codec leaf (libsmacker port): pure bitstream/Huffman/DPCM decode, no knowledge of a player, a stream protocol or a game; cannot import `pkg/video` back |
| `pkg/ui` | `pkg/render` and anything under `pkg/render/`, plus `pkg/audio`, `pkg/video`, `pkg/random`, `pkg/words` and `pkg/locale` | audio devices and video-frame players are presentation leaves; neither can reach game or simulation state. `pkg/words` gives the screens the engine's own words in the install's language; `pkg/locale` the install language's code page and font remap |
| `pkg/words` | `pkg/mod`, `pkg/locale` | reads its embedded tables through the mod text lookup (`mod.Lookup`), the one lookup engine and mod words share; no UI, game or simulation types |
| `pkg/locale` | — | **stdlib only**: the table of install languages, one record per language |
| `cmd/cutscenehelper` | `pkg/video` | separately built Windows/386 native adapter; the normal game remains amd64 |
| `pkg/game` | any `pkg/*` | top library tier |
| `cmd/againrom` | any `pkg/*` | the game |
| `cmd/restool` | `pkg/formats/res`, `pkg/vfs` | |
| `cmd/regtool` | `pkg/formats/reg`, `pkg/formats/res` | `.reg` registries live as entries inside `.res` archives, so the tool reads both |
| `cmd/almtool` | `pkg/formats/alm`, `pkg/mapload` | its `pass` verb censuses the passability plane a map describes, counting through the derivation's own classifier rather than a copy of it, so the tool reaches the tier that owns it |
| `cmd/classdump` | `pkg/data`, `pkg/formats/reg`, `pkg/formats/res`, `pkg/formats/alm`, `pkg/formats/databin`, `pkg/vfs`, `pkg/mapload` | the registries live as entries inside `graphics.res` and the placements it resolves them against live in `.alm` maps, so the tool reads both containers itself — addressing each directly, the per-format developer tools' exemption 0027 leaves out of scope. Its `-databin` verb reaches the definition table through the vfs tier, walks it with the table's own parser, and resolves a map's placements through the map-loading tier so the health it reports is the one a world carries rather than a second application of the arithmetic |
| `cmd/sprtool` | `pkg/formats/res`, `pkg/formats/spr256`, `pkg/formats/spr16` | |
| `cmd/terraintool` | `pkg/render/terrain`, `pkg/formats/res`, `pkg/formats/alm`, `pkg/game` | the archive + map wiring for the terrain render lives here, not in the render tier |
| `cmd/mapview` | `pkg/ui`, `pkg/render/terrain`, `pkg/formats/res`, `pkg/formats/alm`, `pkg/game` | the windowed developer viewer; its loading path lives in `pkg/game` so the game front-end shares it |
| `cmd/texttool` | `pkg/game`, `pkg/render/text` | measures and renders a font, and names only the loader's tier and the drawing tier: the container filesystem reaches it as a return value, so the tool never has to name the format or vfs tiers to open an install |
| `cmd/missionrun` | `pkg/game`, `pkg/mapload`, `pkg/sim` | starts a campaign mission through the front-end's own loading path and drives it headlessly with ordinary move orders; it names the map-loading tier for a script identifier, a party slot and the passability plane, and the simulation tier for a command and an outcome |
| `cmd/dlgtool` | `pkg/game` | censuses every event text of a root against the two tests that decide whether a dialogue window has a portrait pane and whose face is in it. It names the loader's tier alone, for `cmd/texttool`'s reason: the container filesystem reaches it as a return value, so listing and reading an entry costs it no import of the vfs or format tiers. It prints counts only and writes no file |

**Two** external modules are permitted, each confined to one tier, and nothing
else is:

- `golang.org/x/text` (CP866 decoding) for the `formats` tier — minus
  `pkg/formats/reg`, `pkg/formats/spr16`, `pkg/formats/databin`,
  `pkg/formats/pal`, `pkg/formats/wav`, `pkg/formats/itemname` and
  `pkg/formats/winicon`, which
  are denied it by name
  and held to the standard library, so a format that converts no text
  cannot quietly acquire a text-encoding dependency.
- `github.com/hajimehoshi/ebiten/v2` (windowing/rendering) for `pkg/ui` and the
  `cmd/` tier that launches it. Keeping it out of everything else is what stops
  the engine leaking into the formats, sim and render tiers, which must stay
  headless and unit-testable without a window.

The check classifies an import as intra-module by the `againrom/` path prefix, is
fail-closed on any package not listed above (so a package added later cannot
silently escape the DAG), and holds `pkg/sim` to stdlib-only imports including in
its tests. Acyclicity follows by construction: the allow-map is a strict layering.

## Determinism wall

`pkg/sim` is intended to advance only through a pure `Step(state, commands) ->
state` on a fixed integer tick — no IO, no wall-clock, no floats — so that a
simulation is byte-for-byte reproducible from its inputs.

The **structural** half is the import check: `pkg/sim` imports only the standard
library and no other tier. That check permits every standard-library import and
looks at import paths alone, so it says nothing about the hazards inside the
standard library (`os`, `time`, `math/rand`) and nothing about floats, which need
no import at all. It is not evidence of behavioural determinism and is not cited
as such.

The **behavioural** half is a second check beside it in `internal/archtest`, a
**source scan** over `pkg/sim`'s non-test `.go` files. It parses them and reports:

- an import of `os`, `time` or `math/rand`, or of any package under one of them,
  so `math/rand/v2` is no escape;
- any `float32`, `float64`, `complex64` or `complex128` identifier, declared or
  used (complex being float storage under another name);
- any floating-point or imaginary literal.

It reads the parsed syntax, not the file text, so the same names written in a
comment or a string are not findings — `pkg/sim/rng.go` names `math/rand` in
prose to say why it is not the generator, and that explanation is worth more than
a grep's convenience. An empty file set is itself a violation, so a scan that has
stopped finding sources reads as a failure rather than as a pass. The negative
cases run against synthetic sources, one banned thing each: the real `pkg/sim` is
clean, so scanning it alone could never show the check can fail.

What the scan proves is **lexical**: no such import, type name or literal occurs
in those files. It does not prove that nondeterminism cannot reach the simulation
some other way — a float arriving as another package's untyped constant
(`math.Pi`), or whatever a permitted standard-library package does inside itself,
is invisible to it. Reproducibility itself is exercised separately by `pkg/sim`'s
own tests: equal worlds stepped against equal commands reach equal digests.

## Testing convention

The suite is synthetic and self-contained: `go test ./...` runs green with **no
game installation present**. Fixtures are byte streams built in test code from
the documented format contracts; no test reads a game install. When a fixture
must represent a CP866 string, it is written as bytes/hex — never as literal
non-ASCII text. Verification against real game files is a separate developer tool
under `cmd/`, run by a developer, and its evidence is recorded in a work item's
`verification.md`; the evidence ships, the assets never do.
