# Analysis — placed-objects diagnostic overlay (ROM1)

## Intensity & terrain (declared for the whole work item)

- **Intensity: `spec-first / static`.** Per `SDD/PROFILE.md`, a render/UI-tier diagnostic overlay is
  "other engine work" — a bounded delivery whose behavior settles quickly; upfront clarity helps but the
  contract will not keep evolving. No watcher tool exists, so synchronization is static + discipline,
  labelled as such.
- **Terrain: greenfield.** The marker geometry (the pure cross generator, its clipping, its rasterizer)
  is entirely new code added to `pkg/render/terrain`, and the viewer overlay is new code in `pkg/ui`.
  The existing compositors (`Composite`, `CompositeLit`, 0004/0007) and the viewer draw path (0005/0006)
  are extended in an **additive, off-by-default** way: with the overlay disabled the terrain output is
  byte-for-byte the baseline (spec P-3), so no shipped library contract is silently changed and no
  brownfield characterization burden falls on those areas. The only pre-existing cmd behavior touched is
  the flag set and the enabled-only summary text of `terraintool`/`mapview`; the disabled summary keeps
  its existing shape (spec FR-5). Recorded here so the terrain declaration is per changed area, not per
  ticket.

## What the overlay actually consumes

This story adds **no format decoding**. Its entire dependence on the game format is two already-decoded
facts about `alm.Map.Objects` (story 0003), plus its count:

1. Each object's anchor coordinate `(X, Y)` as a `u32` fixed-point `/256` value; the integral anchor cell
   is `(X >> 8, Y >> 8)`, an integer shift.
2. The number of decoded objects, `len(alm.Map.Objects)`.

Nothing else — no `flags`/`kind`/`id`/`value`, no extension pair — is read or interpreted. The overlay
parses no ALM bytes and asserts no field meanings.

## Source & confidence

All facts below are game-derived by the research team, not by this repo's own reverse-engineering.

- **Submodule pin:** `research/` at `87a72f4` (`againrom-research`, module `rom1research`).
- **Experiment:** `research/experiments/EXP-0019` — the variable-content sections type4-9, promoted into
  `research/formats/alm/format.md` (section "Content sections type4-9 - EXP-0019", claim `ALM-OBJ-019`).

## Claim inventory (what the overlay's inputs are built on)

| Claim | Statement | Confidence | Used by overlay |
|---|---|---|---|
| ALM-OBJ-019 | type4 = placed objects/structures. Base record = **20 bytes**; `count = meta+0x28`. Per record: `X` @ `+0x08` (u32 `/256`), `Y` @ `+0x0c` (u32 `/256`), integer part = tile, low byte usually `0x80` = tile centre; a minority append an 8-byte `[coord][value]` **extension**. An adaptive base-20+extension walk consumes each payload exactly (36/37 maps; the 3 with in-stream extensions are Cross/Horror/scn:81) | High (record stride + `X`/`Y` offsets + `/256` fixed-point; corpus-exact walk) | The anchor cell `(X>>8, Y>>8)` each marker is drawn at |
| ALM-CNT-017 / ALM-META-025 | `#type4 == meta+0x28` - the object-record count = `type4_size / 20` (34/38; a few records carry the extension) | High | `len(Objects)` reported by FR-5 |

Supporting `alm` implementation facts (already shipped in 0003, no change this story):

| Fact | Statement | Used by overlay |
|---|---|---|
| `alm.Object` | Struct exposes `X, Y uint32` (fixed-point `/256`), plus raw `Flags`/`Kind`/`ID`/`Value` and an optional 8-byte `Ext []byte` | The overlay reads only `X`, `Y` (and the slice length); it never reads `Flags`/`Kind`/`ID`/`Value`/`Ext` |
| `alm.Map.Objects` | `[]Object`, one entry per decoded type4 record | The list the viewer wires into the marker geometry |

## Decoded-vs-open reconciliation

- **The overlay's inputs are fully decoded.** The anchor cell `(X>>8, Y>>8)` and the object count are both
  High-confidence, corpus-verified facts from `ALM-OBJ-019` / `ALM-CNT-017`, already implemented and
  tested in 0003. The overlay needs nothing further from the research: **there is no blocking research for
  this story, and no format fact is invented.**
- **The undecoded object fields are deliberately out of scope, not a blocker.** 0003's R-2 records that the
  meanings of `flags`/`kind`/`id`/`value` and the 8-byte extension pair are undecoded; `research/`'s own
  open-thread list (`format.md` Residuals: "the type4 record extension discriminator; per-record
  owner/template/item ids in type4/5/6") confirms it. The overlay **references** R-2 only to state plainly
  what it does **not** do: it interprets none of those fields. In particular the extension is an 8-byte
  `[coord][value]` pair of unknown meaning - it is **not** a width/height, so the overlay draws **no**
  object footprint; each object is marked by its anchor cell alone. Object identity, class/template, real
  sprites, and footprints need the class registries (`REG-*`, EXP-0006) and static-data formats, which are
  downstream - the spec scopes them out rather than guessing.

## Scope notes carried into the spec

The research decodes the object container; it deliberately leaves the field semantics open, and the spec
scopes them out rather than guessing:

- **Object art / footprints / identity** - need the class registries + static data (downstream); the
  overlay marks the anchor cell with the project's own diagnostic glyph, not game artwork.
- **Interpreting `kind`/`id`/`value`/extension** - 0003 R-2, not decoded; not touched here.
- **Units, groups, triggers, markers** - other type sections / other stories (units are 0009).

All marker geometry (the glyph shape, pixel math, clipping, camera transform, colour, and size) is the
project's **own** diagnostic-render design - not a claim about the original game's rendering, and the spec
makes no such claim. When the research team later decodes the object field semantics or the class
registries, the flow is: repull the submodule (`git submodule update --remote research`), derive the newly
decoded fact here, then revise `spec.md` from the earliest affected stage per the SDD revision process.

## Appended 2026-08-02 — pin `01c64e2`: `ALM-OBJ-019` is contested, on a field this overlay never reads

`ALM-OBJ-019` now reads `● active (amended, contested)` under the new live contradiction **C-7**
(`research/claims/registry.md`): the row has the `.alm` type-4 `+0x12` word **sign-extended into
`obj+0x10`**, and `TERR-STRUCT-075` reads `obj+0x10` as a **pointer** to a 12-byte position object,
stored at `L02083` and dereferenced by both footprint routines. EXP-0081 read that store and both
dereferences but not `+0x12`'s own, so it **picked neither side**; the destination is unsourced on both
sides at Medium. The field's *role* — a type-7 `Target_Structure` id (`ALM-TRIG-046`) — is untouched.

The claim inventory above records that this overlay reads **only `X`, `Y` and the slice length**, and
interprets none of `Flags`/`Kind`/`ID`/`Value`/`Ext`. `+0x12` is inside that untouched set, and the
contested fact concerns a live object this overlay never builds. Nothing moves.

Two figures in that inventory row are also older than this note and are left as the story's record: the
`X @ +0x08` / `Y @ +0x0c` offsets are the pre-EXP-0030 frame, the same bytes measured from a payload base
since corrected by 8 (now `X @ rec+0x00`, `Y @ rec+0x04`, carried in 0003's `spec.md`), and the "36/37
maps" walk figure is superseded by the `kind == 0x21` discriminator that tiles 38/38. The overlay's
inputs are unchanged by either.

## Appended 2026-08-02 — pin `acb8fb0`: `TERR-STRUCT-075`'s Medium clause is superseded upward

The note above cites `TERR-STRUCT-075` as the row reading `obj+0x10` as a **pointer**. That half was
always High and is untouched. What moved is the row's **Medium** half — which of
`L02090`/`L02091` is the low axis — now **High as amended by EXP-0091**, with a
`claims/retracted.md` row classed **SUPERSEDED**. Both of the old reasons are withdrawn and the
answer they reached is confirmed: the instructions do say (`ALM-OBJ-061`, a nine-hop chain from the
two `LEA`s to the cell key, with no permutation in it), and the "0 of 17 057 footprint cells
off-plane" statistic could never have discriminated, because 37 of the 38 shipped maps are square.
Re-made as connectivity the corpus does separate them — bridge joins 93 against 5, for the published
reading.

**C-7 is unchanged and still live**, and nothing here moves for either row. This overlay reads only
`X`, `Y` and the slice length, as the inventory above records, and builds no live object.
