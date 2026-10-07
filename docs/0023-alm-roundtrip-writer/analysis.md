# Analysis — lossless ALM round-trip writer

## Intensity & terrain

| Axis | Declaration |
|---|---|
| Intensity | **spec-anchored / static** — a byte-identity contract over a shipped format leaf; no watcher tool, so static + discipline |
| Threshold | **Medium** — nothing here reaches hashed sim state; the writer is format-tier and purely additive |
| Terrain — `pkg/formats/alm` | **brownfield, additive**: the shipped reader is itself this story's reference; new entry points beside it, nothing existing moves |
| Terrain — `cmd/almtool` | **brownfield**: one new verb for the corpus check |

Research pin: `3d95f2a`, frozen for the story; `claims/retracted.md` read first.

## The baseline describes a reader this tree does not have

The staged baseline's factual sentences were checked one by one against `pkg/formats/alm/alm.go`:

- **"The reader leaves 12 bytes of every section header and 4 bytes of the file header unread" —
  false here.** This reader is EXP-0030-framed and reads every header byte: record-header
  `+0x00`/`+0x04` are validated constants (tag 7, hdrLen 20 — anything else is rejected), `+0x10`
  is decoded (the per-map float), and file `+0x08` (`dataSize`) is stored raw. Three of the
  baseline's four "unknown" fields are now named by `ALM-FRAME-031`.
- **Consequence: the baseline's AC-2 input is a rejected stream here.** Non-zero bytes in
  record-header `+0x00`/`+0x04` never enter a round trip. What actually varies freely across
  accepted inputs: `dataSize`, `formatVersion` (anything but the rejected 1000), each record's
  per-map-constant bits, the physical order of the ten records, and payload bytes the decode does
  not pin.
- **"Truncates structure/unit X/Y (and structure type id) to their low 16 bits" — false here.**
  `Object.X/Y/Kind` and `Unit.X/Y` are u32; the one narrowing, `ClassID int16`, is the loader's
  own MOVSX and bijective. The truncation AC falls.
- **"`RawSection.Body` keeps aliasing the input" — no `RawSection` exists.** Every decoded slice
  is a fresh copy (`cloneBytes`); the shipped `Map` aliases nothing. The ownership FR survives for
  the `Document`; the aliasing-preservation sentence falls.
- **"Sections of every kind including unknown/raw ids" — rejected here.** `Open` requires exactly
  ten records, typeIds {0..9} once each. The free structural variable is the permutation, not the
  roster.
- **"General name/author" — this tree's fields are the type-0 name and description** (CP1251),
  plus the type-5 group names. The post-NUL sentence is the one lossiness claim that survives
  re-verification.
- **Entry point and vocabulary.** `Parse` exists nowhere; the entry points are `Open`/`OpenInfo`
  over *records*. The document API is renamed accordingly.

## What the interpreted Map actually cannot rebuild

Re-derived from the landed decoder — the losses a `Map`-regenerating writer would hit:

- bytes at and after the first NUL of the type-0 name (+0x30) and description (+0x78), and of
  each type-5 name read (`rec+0x0c` to record end);
- type-5 record bytes +0x00..+0x07, never decoded;
- type-6 record bytes +0x14..+0x45 — 50 of 70 bytes, deliberately unexposed (0003's DD12);
- the exact bits behind the three float32 fields (`Angle`, `SelectorA`, per-record
  `PerMapConst`) — values survive, bit-level guarantees are nothing the typed view contracts.

Everything else round-trips from `Map` in principle (the 632-byte meta is otherwise fully
exposed, grids are lossless, type-4 covers all 20+8 bytes, type-7/8/9 bodies are raw copies) —
but the story does not build on that: a raw-backed document makes the whole inventory moot and
stays correct if a future reader drops more.

## The story's shape after re-derivation

A `Document` that retains the accepted input's bytes and a writer that reproduces them; the
identity is arithmetic over the record walk's own partition of the stream and consumes no new
format fact. The four header fields the baseline called unknown are preserved verbatim exactly as
before — what moved at this pin is their *names* (three decoded from the game's own loader); the
contract interprets none of them, and `dataSize` in particular is never recomputed (its corpus
identity is Medium and its `+72` unexplained).

## What we looked at

`pkg/formats/alm` (doc.go, alm.go, both test files), `cmd/almtool`, `internal/synth` (the
synthetic ALM builder), the consumer set (`pkg/mapload`, `pkg/game`, `pkg/render/terrain`, the
cmds) — untouched by this story; `docs/0003-alm-container` as landed; in research at `3d95f2a`:
`claims/retracted.md` first (five ALM rows overturned by EXP-0030, all pre-corrected-framing, all
already absorbed by 0003), then `alm.md` (`ALM-FRAME-031`, `ALM-HDR-001`, `ALM-SEC-002`/`003`,
`ALM-META-024`/`009`/`025`, `ALM-TRIG-044…050`); and the staged baseline, read as a hypothesis
set.

## Open rather than guessed

- **`dataSize` (+0x08):** the loader never reads it; the `4·W·H+72` identity is Medium. Preserved
  verbatim; never validated, never recomputed.
- **The per-map constant:** a header word with a 3-value corpus domain at Medium. Carried as raw
  bits; acceptance never inspects it.
- **The shipped record order `0,1,2,3,5,4,9,8,6,7`:** a writer convention at Medium — the loader
  dispatches on typeId. Preserved, not enforced.
- **The type-7/8/9 grammar** research has since closed over the corpus (Medium, no consuming
  instruction read): deliberately not adopted — bodies stay opaque; a lossless writer needs no
  grammar.
