# Provenance — units as static sprites

Pinned at research `778c2a6`, frozen for the story; citations re-pointed to `03a9448`, 2026-07-30.
`claims/retracted.md` read first: `TERR-SPR-041` (the dispatch it named draws the HP bars) and
`REG-KEY-044`'s section-index clauses are this story's own ground. `TERR-SPR-048` and
`TERR-SPR-038` have fallen since, both at **High**; each is repaired where cited, and neither
reaches behaviour here.

## Backing

| `spec.md` anchor | Claim | Confidence |
|---|---|---|
| Class id — the record stores the class key; the 70-byte read map (FR-1, FR-2) | `ALM-UNIT-040`, `ALM-CLS-038` | High — the read map is instruction-anchored, summing to exactly 70; the key is a `units.reg` `ID`, the section-index rival failing 82 % of 8094 shipped records |
| Class id used directly, misses required — the class array is `ID`-keyed, 81 slots, 47 NULL | `REG-KEY-044` (amended) | High — three append sites at instruction level, discriminated twice more by the shipped registry; the 1..80 domain is a whole-registry measurement |
| Sheet — `units/` + `Files[class.File]` + `.256`; `File` inherits here | `REG-VAL-029`, `REG-ROSTER-052`, `REG-UNITS-049` | High — the loader's `File` read and its inherited default are named instructions; the restored pairing opens an existing node 34/34 |
| Frame — the standing block opens the sheet; 16/8 vs 9/5 by `Flip`; the standing frame is the facing index | `SPR256-UNIT-024`, `REG-UNITS-051`, `TERR-SPR-047` (amended) | High for the block arithmetic, the layout constants and the standing arm (the tiling exact on 33/34 sheets — the 34th under *Ours*); Medium for the corpus figure and the state names. Amended: the switch is duplicated; the copy read is the shadow's, identical arm for arm |
| Anchor — `(Center − canvas/2) + frame/2`, at the DRAWN frame's size (FR-4) | `TERR-SPR-040` (amended), `TERR-SPR-043`, `TERR-SPR-067`, `SPR256-FRAME-023` | High — six named instructions per axis, rivals excluded by which fields appear; the body pass measures the drawn frame, and all 34 unit sheets mix frame sizes |
| The engine's unit draw reads no terrain height: the body ignores arguments 1–2 and places from the unit's own fields | `TERR-SPR-065`, `TERR-SPR-067` | High — `TERR-SPR-048`'s `vt+0x2c` was the **shadow** (retracted at High); the body `vt+0x28` adds a `− unit+0x10` and reads its third argument, a light level. What maintains those fields is open |
| The lift borrowed for displaced mode — the cell's four-corner mean, the object path's own (FR-4) | `TERR-SPR-038` (in part), `TERR-SPR-039` (amended) | High for the mechanism (instruction-anchored, truncating toward zero); Medium that it stands art on the drawn ground, a corpus cross-check. What fell in `TERR-SPR-038` is its fifth push slot (`TERR-SPR-066`), not this destination |

## Ours by choice

| What the spec fixes | What the evidence says |
|---|---|
| Ground point = the entity's cell centre — **provisional**, *relabelled 2026-08-01, see below* | Our sim is cell-resolution (0019); the engine places from unit fields whose maintenance is undecoded. At a record's rest position (low byte `0x80`) the two coincide |
| The displaced lift on unit sprites (FR-4) | Decoded only for the OBJECT path; the unit draw reads no height. Chosen so sprite and cross coincide at tick 0 — the instrument. A later story giving an entity a pixel position must carry the height in exactly one place, or the lift is counted twice |
| Facing: direction index 0, constant | No type-6 field is established as a facing — the `+0x1c..+0x2b` roles are unresolved (`ALM-UNIT-040`, Medium). The constant lands on sheet frame 0 under either `Flip` layout |
| The class id raw in canonical state, unvalidated (FR-1, FR-2) | `+0x08` lies inside the loaded `ID` set on 8094/8094 shipped records (`ALM-CLS-038`); loader validation would make a save's digest depend on registry contents |
| Byte form version 2, its field order, the refusal discipline (FR-1) | Ours, as version 1 was in 0019: one version defined, every other refused |
| Layer order inside the entity layer | Our layer design, extending 0020's content-under-instruments order; no draw-order fact is claimed |
| The `Unit33` stance (the frame rule's "resolved `Flip`") | `SPR256-UNIT-024`'s one miss: `File = 21` with an inherited `Flip = 0`, the registry's own inconsistency. The contract follows the resolved `Flip`, as the engine does; at facing 0 both readings give frame 0, so the divergent indices are never reached |
| Startup fails on an unreadable unit registry (FR-7) | Mirrors the object bundle: an install missing a registry is reported before a window opens |
| Unit sprites unlit, as object art is | Decided against us at this pin: sprites ARE lit, at the cell's `+0xb0` level through the sheet's own palette (`TERR-LIGHT-059`…`064`, High). A disclosed divergence, no longer an open question |

## Open / undecoded

- **The definition database and the override paths.** The engine resolves a type-6 record through
  a named-node collection whose populator is unlocated, plus the `+0x10`/NPC-flag overrides
  (`ALM-CLS-038`); this story resolves by the class key alone.
- **The engine's unit draw position.** How `+0x60`/`+0x64` are maintained. `+0x68` is the
  ground-plane term and `+0x10` the lift (`TERR-SPR-041`, amended); nothing here claims either.
  **And the placement this story ships is right only IF `unit+0x10` is 0 for a map-loaded unit.**
  `TERR-SPR-067` gives the body `dstY = unit+0x64 − anchorY − unit+0x10 − unit+0x68`, so the lift
  is a term in the very expression the re-pointing landed; ours omits it, which is the same
  arithmetic exactly when the field is zero. **No writer of `unit+0x10` is decoded** — the claim
  says outright that no instruction in the block writes it and that its being a height above ground
  is the Medium half — so we cannot show it is zero at map load, only that nothing yet says it is
  not. Every corpus figure in `verification.md` is consistent with zero, which is agreement and not
  proof: a nonzero lift would move every unit by the same amount and no comparison here would see
  it. If the writer is ever decoded and the field is nonzero for a map-loaded unit, this story's
  placement is wrong by that many pixels and nothing else about it changes.
- **The compass meaning of frame 0.** Which direction frame 0 shows is not decoded; the facing
  constant is presentation, not an orientation claim.
- **~~`units.reg` absent-everywhere defaults.~~ CLOSED 2026-08-01, and it was not closed here.**
  This entry read: decoded as −1 / eight zeros / `TileSize` 1 (`REG-UNITS-049`), while `pkg/data`
  (0016) resolved them to the type's zero — of this story's keys only `Flip` could differ and both
  give 0 — filed as a 0016 revision question. **Story 0024 implemented the decoded table** (`T1`,
  `5d50161`), so `pkg/data`'s `unitDefaults` now carries all twenty-six rows, `File` inheriting
  included, and the premise of the question is gone. It is struck rather than deleted because an
  open-questions list that quietly loses rows cannot be audited; and it is recorded here because
  this file is where the question was raised, not where it was answered. `structures.reg` is still
  undecoded and still keeps the Go zero.

## Removed from the baseline and why

- **`Frames[class.Index]` as the unit's static frame (its R-1).** Refuted: the frame comes from a
  nine-state switch over a block-addressed sheet (`TERR-SPR-047`, `SPR256-UNIT-024`) and no arm
  consults `Index`. The standing-block frame replaces it.
- **The `CenterX`/`CenterY` research item (its R-2).** Closed: the anchor is decoded
  (`TERR-SPR-040`/`043`/`067`); the note corroborating the hotspot from a third-party
  reimplementation fell with it.
- **`DefinitionID` — a registry-validated id with a `NoDefinition = -1` sentinel, set in the
  map loader.** Replaced by the raw stored key (register above); the name went with it —
  the record's own `+0x10` IS a definition id of another, undecoded table, so the word invited
  that confusion.
- **`openrom`, `MapScene`, `PickerScene.launch`, `AttachSim`, `SpawnsFromALM`,
  `NewWorldFromSpawns`, `SampleHeight8p8`, `render.ObjectAnchor`.** None exists here; re-derived onto `pkg/game`'s
  front-end and world holder, `mapload.FromALM`, `pkg/ui`'s entity pass, and
  `terrain.StaticAnchor`.
- **The sim layer drawn topmost (its FR-4).** Refused as 0020 already refused it; see the layer
  row above.
- **`height = SampleHeight8p8` at the 8.8 cell centre.** No such sampler; the lift is the marker
  family's cell mean.

## Appended 2026-08-01 — pin `130bb79`: `ALM-CLS-038` is REFUTED, on the rival this story measured against

The *Backing* row above cites `ALM-CLS-038` for the 70-byte read map and the `+0x08` class key, and
states the reading it took: "the key is a `units.reg` `ID`, the section-index rival failing 82 % of
8094 shipped records". At this pin `retracted.md` strikes that rival outright — "the live `CUnit`
holds a `units.reg` **section index** at `+0x20`" is **REFUTED**; it holds the **`ID`** and nothing
is translated, the arrays being `ID`-keyed. The cost recorded is a consumer that "inserted a
translation step that must not exist, and resolved most shipped placements to the wrong class".

This story inserted none, so FR-1 and FR-2 are unchanged. What is new is only that the alternative
is now struck rather than out-scored, and that a reader meeting the id in `retracted.md` need not
work out which half was used here.

## Appended 2026-08-01 (second) — the ground point is a substitute, not a product choice

*Ours by choice* has been carrying two different things: a **product choice**, a local contract we
intend to keep, and a **provisional substitute**, a placeholder standing in for data we do not have
and meant to disappear. Only the first is really a choice, and an unlabelled substitute is one
nobody comes back to.

The ground-point row is the second kind, and its own wording is the tell — "at a record's rest
position (low byte `0x80`) the two coincide" is what a stand-in says, not what a decision says. The
data it stands in for now exists: `MOVE-STEP-010` puts movement at **sub-cell resolution, 1/256 per
axis, with `0x80` centred**, so the engine's placement is no longer undecoded — what is missing is
a sim that carries a fraction, which is `0019`'s cell resolution and another story's to change.
Relabelled only; the contract is unchanged and FR-1 does not move. The neighbouring rows are
product choices and stay as they are.
