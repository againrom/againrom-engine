# Provenance — typed data classes from the graphics registries

Pinned at research `e61153d`, frozen for the story; the **revision** at `9c01af7`, frozen in turn.
`claims/retracted.md` read first: it withdraws at the **High** it held `REG-KEY-044`'s clause that
the class arrays are section-index tables, `Parent` with them, and that a placed `ID` must be
**translated** — all read in `objects.reg`, the one registry where `ID == section index` on 82/82
and so the only one that cannot discriminate.

## Backing

| `spec.md` anchor | Claim | Confidence |
|---|---|---|
| `Parent` is an `ID`; inheritance eager, per key, at load, nearest ancestor winning; structures have none; arrays one hop (the parent's own section re-read) where scalars chain; `ByID` primary, the engine's arrays being `ID`-keyed and sparse (units 81 slots for 34, 47 NULL; structures 67/66; objects 82/82) | `REG-KEY-044` (amended 2026-07-27) | High — store sites and the `Parent` subscript at instruction level; the rival reading leaves 4/16 unit parents unappended and makes `Unit3` its own parent. One hop is High for the units loader, the only one whose array sites were read |
| Guard is **presence** for scalars, **length** for arrays; `0` overrides; an empty array key does not clear one; a wrong-kind scalar throws; a non-empty string for an array is a hard error | `REG-KEY-045` (new) | High — both accessors read off instructions; the one discriminating record (`Unit33`, kind 0 size 1, `Parent = 3`) pinned to measured bytes |
| A placed class id is the subscript unchanged; the type-3 code minus one is an `objects.reg` `ID` | `ALM-CLS-035`, `ALM-CLS-036`, `ALM-CLS-038` | High for which field selects a class and which registry it names — rivals fail a corpus domain fit / **Medium** for the type-4 table being `ID`-ordered and a live unit's `+0x20` an `ID` |
| Counts 34 / 82 / 66, `[Global]` names, `[Files]`, the key catalogues and their ASCII spelling; the record framing and kinds | `REG-UNITS-018`, `REG-OBJ-039`, `REG-STR-040`, `REG-VAL-029`, `REG-LOC-016`; `REG-REC-032`, `REG-KIND-033`/`034` | High — counts cross-checked twice on the corrected framing, names byte-present; `REG-UNITS-018`'s **withdrawn** track-length clause is unused |
| A class may resolve cleanly and have no sprite file behind it | `ALM-CLS-042` | Medium — a corpus measurement over another claim's key |
| **Revision.** `objects.reg` per-key defaults where no section on the chain sets the key — `-1`, `InMapEditor` `0` — and `File` not inheriting there, its default being unconditional | `REG-OBJ-046` | High — each key→offset pair a `PUSH`/`CALL`/`MOV` triple in a listing read end to end, `File`'s literal one named immediate. `-1` is **stated** for `File`, carried for the rest by `InMapEditor`'s "default **0**, not −1" and for `Parent` by the `Parent != -1` guard |
| **Revision.** `DeadObject` `-1` on 54 of 82, else a class subscript; `FireObject` `{-2 ×21, -1 ×61}` | `REG-OBJ-047` | High — both value spaces exhaustive over the only registry carrying the keys |

## Ours by choice

| What the spec fixes | What the evidence says |
|---|---|
| `units.reg` and `structures.reg` keep the Go zero where a key is set nowhere — **provisional**, *relabelled 2026-08-01, see below* | Their defaults are **not** decoded at either pin — `REG-OBJ-046` reads the objects loader alone — so the contract states the limit |
| An absent array key with no inheritable ancestor is **nil**, full stop | `REG-KEY-045`'s own **Unknown**: there the engine leaves its destination untouched and reuses two `CArray`s across six keys, so absent can read the previous pair's residue |
| Arrays resolve **one hop**, unobservably | The only depth-2 chain inherits no array; we follow the loader that was read |
| Sprite path construction — separator, prefix, `.256`, the overlay's `b` | No claim reads the engine's path formatter. Ours, corroborated by the archive index: Medium |
| An unresolved type-3 code is counted, never a sweep failure | `ALM-CLS-035` finds 64 of 71,099 nonzero cells past the array and grades whether the engine reads them **Unknown** |
| `All` in numeric section order; absent `DescText` and absent art both non-errors | The order is the owner's; the engine has none. The non-errors are forced by measured data (7 nameless, 7 artless) and `ALM-CLS-042` |

## Measured, not claimed

**Orchestrator pre-flight, 2026-07-27**: the key inventory; the `File`-never-omitted counts; the
depth histogram and single depth-2 chain; seven nameless object classes; 52 structures with an empty
`AnimFrame`; no forward `Parent`. **Our own re-derivation** (`cmd/regtool dump`, `cmd/restool list`;
asset root on the command line, output outside the repo) reproduces each and supplies everything
else the spec asserts without a claim. The revision adds
one: at the landed code `classdump` read `DeadObject = 0` on 33 classes and `FireObject = 0` on 54 —
the defect, not a disagreement with `REG-OBJ-047`.

## Open / undecoded

- **Every animation, combat and render key meaning.** `REG-LOC-016` gives the *names* at High; the
  baseline's meanings came from a third-party reference and appear nowhere in the spec. Not
  dispatched on the owner's instruction: nothing depends on the label (P-2).
- **What a `Parent` written as `-1` means** — the engine's guard is `Parent != -1`, so it reads as
  no parent; ours rejects it as unresolvable. No shipped class writes one.
- **Whether the engine's absent-array arm ever yields stale residue** — `REG-KEY-045`, Unknown.
- **The 64 type-3 cells past the registry, and what the engine draws at an artless class**
  — `ALM-CLS-035`, `ALM-CLS-042`, both Unknown. Reported categories, not failures.
- **`ShadowY`'s sentinel convention, `patch.res` layering** — untouched. `IconID`: the engine's object
  class record has no field for it, so it has no default and keeps the Go zero.

## Removed from the baseline and why

- **"`Unit33` clears its inherited attack animation with an empty `AttackAnimTime`"** — false
  (`REG-KEY-045`, High): the guard is the array's length after the read, so the empty value falls
  through and `Unit33` inherits `Unit0`'s track. The *rule* — an empty array key reads as absent —
  survives; **nothing spells "clear this array"**.
- **"Units inherit all keys including `File`; objects inherit all except `File`"** — deleted in the
  first pass as unsupported: no shipped record can contradict either half. The objects half is
  **since re-derived from the loader** (`REG-OBJ-046`) and is back on that evidence, not the
  baseline's; the units half is still unread.
- **"Units depth ≤1, objects depth ≤2" as a bound** — an observation, not a constraint; the depth-2
  half rests on one class, and the contract resolves an arbitrary acyclic chain instead.
- **The hedge on `Parent`** against the section-index reading — `REG-KEY-044` settles it; and
  **`[C, corpus-proven]` on sprite paths** — regraded Medium, ours with agreement as corroboration.
- **R-1 as a spec section** — the inventory is kept and each meaning marked unconfirmed in place; a
  request whose named source is a third-party reference is not one we can act on.

## Appended 2026-08-01 — two rows above are no longer true of the tree

A provenance is a dated record, so both are appended beside what was written
rather than over it: this story was authored against the readings below, and a
reader has to be able to see that.

**`units.reg` no longer keeps the Go zero.** The row above says `units.reg` and
`structures.reg` keep it where a key is set nowhere, because their defaults were
decoded at neither of this story's pins. `REG-UNITS-049` has since decoded
`units.reg`'s whole scalar inventory — seventeen keys to `-1`, eight to `0`,
`TileSize` to `1`, `InMapEditor` with no field at all, and `File` **inheriting**,
opposite to `objects.reg`'s — and **story 0024 implemented it** (`T1`,
`5d50161`), so `pkg/data`'s `unitDefaults` now carries those rows. The clause is
still exactly right for `structures.reg`, which is undecoded and keeps the zero.

This was carried as an open 0016 revision question in
`docs/0022-units-static-sprites/provenance.md`. It is closed, and closed by 0024
rather than here; nothing in this story's contract needs to move for it, because
the contract's own sentence is scoped to what was decoded at its pin.

**The `b` sibling is an overlay, and was called a shadow.** `SpritePath`'s
companion was `ShadowPath` in `spec.md`, `plan.md`, `tasks.md` and `pkg/data`,
and `classdump` printed a `shadow =` line. The row above discloses that the path
FORMATTER is ours at Medium, which was honest; what was not disclosed is that the
NAME made a second claim about the node's identity, and that the pin
**contradicts** it rather than leaving it open. `SPR256-OVL-014` measures the
b-variant as an overlay layer — one frame per base frame, ~12x sparser, indices
zero in its own palette and coloured in the base's — compositing over the base;
`EXP-0053` puts it at the table entry's `+0x08`, drawn by the same six-argument
lit signature as the base pass. The engine's shadow is a different mechanism on
the SAME sprite: those passes never read the source and recolour the destination
through the shroud table. There is no shadow file the old name could have named.

Renamed to `OverlayPath` throughout, and `classdump` prints `overlay =`. Nothing
about the string it returns changed — same base, same `b` before `.256` — so no
figure in `verification.md` moves; only the label above it does.

## Appended 2026-08-01 (second) — two cited rows are now REFUTED, on the readings this story rejected

`REG-STR-040` and `ALM-CLS-038` each gained a `retracted.md` row at pin `130bb79`, both classed
**REFUTED**, and in both cases the clause struck is the one the *Backing* table above was written
against rather than from.

- **`REG-STR-040`** — its parenthetical "1-based, unlike `units`/`objects`' 0-based `ID`" is false:
  **only `objects.reg` is 0-based**, and `units.reg` is sparse `1..80`. The row is cited above for
  the counts 34 / 82 / 66, which are untouched, and the *ID*-keyed picture this contract states is
  the corrected one — 81 slots for 34 units, 67 for 66 structures, 82 for 82 objects. A consumer
  that believed the parenthetical would index the units array off by one on every class; `ByID`
  does not.
- **`ALM-CLS-038`** — "the live `CUnit` holds a `units.reg` **section index** at `+0x20`" is false;
  it holds the **`ID`**, and nothing is translated. The row above already grades exactly that
  clause **Medium** and the header of this file already records `REG-KEY-044`'s matching
  withdrawal. What is new is the Kind: this was an error, not a label that moved, and the rival is
  now struck rather than merely unpreferred.

Neither moves a requirement. Recorded because a reader meeting these ids in `retracted.md` should
not have to re-derive which half of each row this story leaned on.

## Appended 2026-08-01 (third) — *ours by choice* was doing the work of two labels

A **product choice** is a local contract we intend to keep. A **provisional substitute** is a
placeholder standing in for data we do not have, and it is meant to disappear. Both have been
written here as "ours by choice", and only the first should be — an unlabelled substitute is one
nobody comes back to.

The Go-zero row above is the second kind, and this story proves it rather than argues it: its
`units.reg` half **already disappeared**, exactly as a provisional substitute should, when
`REG-UNITS-049` decoded that registry's whole scalar inventory and `0024` implemented it (the
first append above records the event). The `structures.reg` half is the same substitute waiting
for the same thing. It is relabelled, not changed: the contract still keeps the Go zero, and what
is new is that the row now says it is waiting.

## Appended 2026-08-02 — pin `acb8fb0`: the `structures.reg` substitute is no longer waiting, and that is a revision this sweep may not make

The third append above relabelled the Go-zero row as a **provisional substitute** and said its
`structures.reg` half was "the same substitute waiting for the same thing". **The thing has
arrived.** `REG-STR-080` reads that registry's loader `R1400` end to end — a `0xa4`-byte
class record, every field the `MOV` after its own key `PUSH` — and publishes the per-key defaults at
High: **`-1` for `ID` through `SelectionY2`, and `0` for `ShadowY` and the five flags.**

`pkg/data`'s `structureDefaults` is nil and its comment reads *"not decoded: a key set nowhere keeps
the Go zero"*. That sentence is now false, and the substitute is measurably wrong in one direction:
a structure class omitting any of `ID`…`SelectionY2` resolves to `0` here where the engine gives
`-1`. **Correcting it changes shipped Go and a contract clause, so it is a revision and not a sweep
edit** — recorded here and reported to the orchestrator, exactly as the `units.reg` half was left to
0024 rather than taken here.

**Two statements this contract makes on its own authority now have a claim behind them.** Both were
reasoned from the shipped corpus and are attested at High by the same experiment, which is a
strengthening and moves nothing:

- **A structure class inherits nothing.** `REG-STR-080` finds **no `Parent` key in that loader at
  all** — an absence over a routine read end to end, not a sweep — which is what `keys.go`'s
  `structureDesc{parentLegal: false}` already refuses.
- **`File` is a path, not an index.** `REG-STR-081`: the loader reads it into a `0x100` local and
  stores `"graphics\structures\" + File`; there is no `[Files]` table, and all 66 resolve under the
  whole-path fold. `structureDesc{hasFiles: false}` and `StructureClass.File`'s own comment are
  confirmed rather than merely chosen.

**And the one relation the contract validates on `AnimMask` is the right one.** `REG-STR-082`: the
loader allocates `FullHeight × TileWidth + 1` bytes, so a non-empty mask's length is that product on
**14/14** classes that spell one, while the `TileWidth × TileHeight` rival differs on 6 of the 14 and
fails all 6. Also worth carrying forward, because it is a trap a consumer of this data walks into:
ten further classes spell `Phases > 1` and then spell neither `AnimMask` nor `AnimTime`/`AnimFrame`,
so **`Phases > 1` alone does not mean animated**.

**`ALM-CLS-036` is amended, and not on either clause cited above.** The new text names the
extension's footprint override — file `+0x14` → `obj+0x60`, `+0x18` → `obj+0x61` (`ALM-OBJ-062`) —
and records that the arm is chosen by the **sum** of those two bytes rather than by the kind
(`TERR-STRUCT-090`). The *Backing* row cites the id for which field selects a class and which
registry it names; `verification.md` cites it for `3 141` type-4 records over `kind ∈ 1..66`. Both
stand. `ALM-CLS-063` bears on the second and in its favour: the `Shop` constructor reaches the
footprint resolver through its base class, so the resolver has three callers where a call-site count
showed two — and the published 3 141-placement figure "was right, but assumed rather than tested",
and is now tested.
