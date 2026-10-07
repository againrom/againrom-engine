# Analysis — map-editor E2: the edit / document model

## Intensity & terrain

| Axis | Declaration |
|---|---|
| Intensity | **spec-anchored / static** — the edit surface over a shipped format leaf, consumed by every later editor story; no watcher tool, so static + discipline |
| Threshold | **Medium** — nothing here reaches hashed sim state; the model is headless and format-tier-adjacent |
| Terrain — `pkg/mapedit` | **greenfield**: a new package, no behavior to preserve |
| Terrain — `pkg/formats/alm` | **brownfield, additive**: the shipped reader and `Document` are the reference; an editing seam lands beside them, nothing existing moves |

Research pin: `3d95f2a`; the revision re-read at `03a9448`. `claims/retracted.md` read first.

## The baseline describes an editor over a package this tree does not have

Every factual sentence of the staged baseline was checked against `pkg/formats/alm` as landed:

- **"Author CP1251 `@0x78..0x278` (512 B)" — false, and the most expensive one.** `+0x78` is a
  **64-byte description**; the 448 bytes at `+0xb8` are a separate region the shipped `Meta`
  carries as `Slots`. They are 7 x 64-byte text slots that campaign maps fill with live trigger
  and quest strings. The baseline's FR-4 rewrites "the entire fixed-width field" with NUL fill —
  applied to a 512-byte Author that erases all seven slots on every name edit. Scoped to 64 bytes
  here; the slot region is named as carried.
- **"Name CP1251" — false.** The reader decodes the name with ASCII and only the description with
  Windows-1251. A single-codec setter would write bytes the reader does not read back, in one
  direction or the other.
- **"Sections", `Parse`, `ParseDocument`, `SectionIDs`, `SectionBody` — none exist.** The
  container is ten *records*, the entry points are `Open`/`OpenDocument`, and navigation is
  `Version`/`RecordTypeIDs`/`RecordPayload`.
- **"On a map that already contains a Units section", "the absence of the addressed grid section"
  — unreachable.** All ten records always exist; `Open` rejects a stream missing any typeId. A
  map with no units has an empty type-6 payload, which appends exactly like any other. Two of the
  baseline's error branches, and its "creating an absent section" out-of-scope item, fall.

  **Re-grounded 2026-07-30, and only the ground moved.** Right about the reader, wrong about the
  format: ten records was our `Open`'s rule, not the container's, and 0003 held the
  counter-evidence — the count gates at `>= 3` with only `{type1, type2}` required
  (`ALM-REQ-055`), a four-record map ships (`ALM-CORP-060`), and an absent type-3 is manufactured
  rather than refused (`ALM-REQ-056`). On the owner's ruling that our reader accept at least what
  the engine does, `Open` widened and the branches became reachable. Which ones: **type-3 absent**
  and **type-6 absent**, the latter distinct from an empty type-6 payload this story always
  covered. Type-1 and type-2 stay unreachable, still required.
- **"X low-16-of-u32 ... the high halves are preserved" — false.** X and Y are whole `u32`
  fixed-point words (tile = `value>>8`), and the shipped `Unit` exposes them as `uint32`. There
  is no low/high split to preserve: a move rewrites both 4-byte words.
- **"Heights `int8`" / "Objects" — the shipped names and widths are `Altitudes []uint8` and
  `Overlay []uint8`.**
- **"`pkg/mapedit` imports `golang.org/x/text/encoding/charmap`" — a DAG violation here.** The
  import-graph check grants `golang.org/x/text` to `pkg/formats/*` alone and is fail-closed on
  any unregistered package, so `pkg/mapedit` needs a registration and cannot carry the codec.
- **"General's `UseTiles`/`TimeOfDay`/`Darkness`/`Contrast`" — labels our own decode contradicts
  or does not support.** The `+0x18` word is read into a local and discarded by the map loader;
  `+0x0c/+0x10/+0x14` are stored with no meaning established. The shipped `Meta` already named
  them by offset, and the setters follow it.

## What the 70-byte unit record actually holds

The published read map covers all 70 bytes, but the shipped `Unit` exposes only `+0x00..+0x13`.
The other 50 bytes are real, live state: the owner slot at `+0x14` (a 1-based index into the
type-5 roster), a `+0x18` index the loader range-checks, two `0xFF`-sentinel runs at `+0x35` and
`+0x3b`, and the unit and group id words at `+0x40`/`+0x42` that type-7 triggers resolve their
targets against.

So `PlaceUnit(u alm.Unit)` — the baseline's signature — cannot be honest: it would fabricate
those 50 bytes as zeros, producing a record with no valid owner, no sentinel pattern, and id 0
shared with every other placed unit. It decodes cleanly and is game-hostile, which is the worst
combination. The story takes a whole 70-byte record instead and interprets nothing but X/Y.

## Where the string encoder can live

The one hard architectural question. The formats tier holds the `golang.org/x/text` grant, the
DAG check is fail-closed, and the decode/encode symmetry the spec requires is exactly the kind of
pair that drifts when its two halves live in different packages. Both facts point the same way,
so the spec states the requirement (same codec, byte-identical or rejected) and the import
boundary (no external module in `pkg/mapedit`) and leaves the placement to the plan.

## What we looked at

`pkg/formats/alm` (`doc.go`, `alm.go`, `document.go`, all four test files), `internal/synth`'s
ALM builder, `internal/archtest`'s allow-map and external-import rules, `docs/ARCHITECTURE.md`,
`docs/0003-alm-container/spec.md` and `docs/0023-alm-roundtrip-writer/` as landed; in research at
`3d95f2a`: `claims/retracted.md` first, then `claims/alm.md`
(`ALM-META-008`/`009`/`010`/`024`/`025`/`026`/`027`/`028`, `ALM-CNT-017`, `ALM-GRID-012`,
`ALM-GRID-032`, `ALM-FRAME-031`, `ALM-UNIT-018`/`040`/`048`, `ALM-CLS-038`, `ALM-OWN-039`,
`ALM-TRIG-046`); and the staged baseline, read as a hypothesis set.

Measured rather than assumed: of the 256 single-byte Windows-1251 inputs, exactly one — `0x98` —
does not survive decode-then-encode. It decodes to U+FFFD and no encoder can put it back. That is
why the spec's string rule says *byte-identical or rejected*, never *byte-identical*.

## Open rather than guessed

- **What the type-0 scalars mean.** `+0x0c/+0x10/+0x14/+0x70/+0x74` are stored by the loader with
  no meaning established, and `+0x18` is discarded outright. The setters write raw values at
  offsets the reader already names; no story asserts a semantics.
- **Whether a placed unit needs a fresh `+0x40` id, a valid owner slot, or an in-grid
  coordinate.** All three are real constraints of the *game*, none is a constraint of the *file*
  the reader accepts. Validation is a later story; this model rejects only what it can decide.
- **The file header's `dataSize`.** Left carried, as 0023 left it. No in-scope mutation changes W
  or H, so the only identity ever observed for that word is untouched either way.
