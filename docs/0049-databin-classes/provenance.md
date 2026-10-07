# Provenance — the definition table, and what each number rests on

Claim ids are the research submodule's, read at this tree's pin. Grades are the ledger's own:
**High** only where the evidence rules the alternatives out, corpus agreement alone capped at
**Medium**. Where a cited row carries both, the split is named below.

## Backing

| Spec anchor | Source | Grade |
|---|---|---|
| FR-1 - the group order, the wire primitives, the per-class entry payloads, and that the walk tiles the file | `DAT-GRAM-003`, `DAT-OBJ-002` | High |
| FR-1 - which collections are 1-based and why a count word is one larger than its entries | `DAT-GRAM-003`, `DAT-BLD-005` | High, carried twice and independently |
| FR-2 - slot *i* is column *i+1*; an empty cell is stored as −1 | `DAT-SCHEMA-004` | High / Medium (a) |
| FR-3 - the 38 slots, their destinations, widths and order; the −1 skip; slot 33 dropped and re-read on demand. Corroborated by the display's 25-store copy block | `UNIT-STREAM-001`; `UNIT-PANEL-010` | High |
| FR-3 - the constructor defaults a −1 cell leaves standing, the two regeneration periods among them | `UNIT-CTOR-004`, `HERO-REGEN-021` as amended | High |
| FR-3 - that a non-hero has **no** derive, so the columns are the stats; and that this file carries the actor's numbers while `units.reg` carries the drawable's | `UNIT-DERIVE-003`, `DAT-SCHEMA-007`, `REG-UNITS-049`, `REG-UNITS-061` | High |
| FR-4 - the pair, the spread, and the switch whose selector is a local **pre-set to zero** | `UNIT-STREAM-001` | High |
| FR-4 - that arms 1 and 2 are unexercised on both shipped roots | `UNIT-COMBAT-006` | **Medium** (b) |
| FR-5 - the three search arms, the byte truncation, the carve-out, skipped empty names, first match wins; the placement's own fields; the key's injectivity | `ALM-CLS-052`, `ALM-CLS-038`, `ALM-UNIT-040`, `UNIT-PROTO-008` | High / Medium (c) |
| FR-7 - the three-way adjustment, its two constants, health forced to the new maximum, the hero exemption, and that it runs once at construction. Also that 1 weakens a unit and 3 strengthens it | `UNIT-GATE-013` | High - the ordering read off the arithmetic, not off the control |
| FR-7 - the value set `{1,2,3}` and the constructed default `2` | `UNIT-GATE-012` | High for the value set and the writers; Medium only that its *reader* enumeration is exhaustive, which this tree does not need |

**(a)** High for the streamed slots; Medium for title-only slots of collections whose consumers are
unread. Nothing rests on that half: they are framed and given no meaning.
**(b)** A corpus census, capped there; it is why C-2's refusal costs nothing measurable, and nothing
else rests on it. **(c)** High but for two clauses this contract does not use: that `typeID` equals
the registry `ID` *as a general law*, and injectivity, held as a corpus property rather than a rule,
so first-match-wins stands and a file breaking it is not refused.

**Threshold.** This story reaches hashed simulation state: a resolved health maximum becomes an
entity's `MaxHP`, which enters the canonical byte form and the digest. The four clauses deciding
that number - slot 4's destination, the constructor's 30, the adjustment arithmetic and the
resolution rule - are each **High**. Nothing Medium reaches it.

## Ours by choice

| Choice | Why it is ours |
|---|---|
| A parameter, and every definition field, is `int32` | As the registry loader here already does, so a negative stays visibly negative. The file stores raw `u32`. |
| Protections and resistances are fixed 5-arrays in **slot order** | The engine indexes protections in a different order from the one the columns arrive in; storing column order and saying so keeps the two apart. |
| The nine-double record is kept as raw bytes | Its slots are undecoded; naming them would assert an interpretation this tree cannot use. |
| Entry 0 of a 1-based collection is materialised as an empty entry | A subscript is then the game's own index, not that index minus one. |
| The adjustment is integer, `×66/100` and `×3/2` | Over the 16-bit domain a stored maximum can take these agree exactly with the engine's constants and its truncation - and the result crosses into hashed state. |
| `difficulty`, and `Easy`/`Normal`/`Hard` | **Owner testimony.** `UNIT-GATE-014` fixes the control completely - which button, which value, which arithmetic, on which actors - and grades its **caption Unknown**: the three bitmaps carry no text at all. The word is the owner's, not the image's; the *ordering* is not testimony but arithmetic. |
| `FromALM` keeps its signature and its provisional health | A landed contract, and every digest recorded against it, stay put; the table arrives as a second entry point's argument. |
| The provisional health pair survives wherever the table cannot answer | Better one visible constant than a decoded-looking number with nothing behind it. |

## Open, and deliberately not consumed

- **The Humans slot list.** Only the five slots the search and the spawn footprint need are
  published in a claim; the complete list exists inside an experiment's evidence file. That is a
  request to research to publish, and until it is, no human placement gets a stat block.
- **Slots 34–36** - *meaning* graded Medium in `UNIT-STREAM-001`, resting on shipped column titles
  and one corroborating reader. Consumed, not carried.
- **The trailing string array's count word.** `DAT-OBJ-002` names it a string array and this file's
  title arrays are counted, but nothing states the entry's array separately - so the reading rests
  on the serializer's identity, and the whole-file tiling is what would falsify it. That is why
  residue is an error and why the tool run exists. The **length escape past the 16-bit form** is
  open the same way: the published primitive stops there, and a longer one is refused, not guessed.
  **2026-08-01 - FALSIFIED by the tool run, and closed.** There is no count word: an entry's
  strings are a fixed run, two on a Units row and ten on a Humans row. `DAT-OBJ-002`'s "a
  CStringArray of 2 / 10 strings" describes the entry's in-memory **field**; reading that as a
  counted wire array was ours, and it was wrong. The counted walk desynced inside Units entry 26
  and surfaced at `+0x4400` as an undefined length escape - the escape half of this row catching
  the array half. Repaired in the contract's I/O example; the walk then tiles 88 327 of 88 327
  with 0 residue and reproduces every count `DAT-GRAM-003` publishes. The escape stays open and is
  now measured unexercised on this root.
- The seven collections this tree only frames; the equipment, spell and treasure columns; and the
  buildings footprint masks, which this story makes reachable and does not take.

## Removed, and why

- **A second and third damage pair.** Modelling the two unexercised arms adds fields no consumer
  reads; dropping their values silently ships a monster with no damage. Refused loudly (C-2).
- **The equipment name grammar.** `UNIT-EQUIP-005` is **Medium** on it - a corpus fit over 26
  strings, the parsing routine unread - and the weapon it resolves would then *overwrite* the attack
  cadence and extend reach on 16 of 56 classes. Carrying template values and disclosing the
  divergence is honest; carrying a Medium grammar into hashed state is not.
- **The spellbook** - `UNIT-SPELL-007` is Medium on the probability scale and its second
  destination's consumer was not read. **The prototype cache and the information panel** -
  `UNIT-PANEL-011` states as a positive Unknown that which cached value appears where cannot be
  settled without the interface layer.
- **Writing the file, and the CSV fallback that regenerates it** - decoded, and out of bounds.

## Appended 2026-08-01 - pin `130bb79`: three cited rows moved, and one of them is contested

A provenance is a dated record; the table above is left as written and the four changes are set
beside it. The **Threshold** paragraph is re-checked against each and still holds.

**`UNIT-STREAM-001` is now `● active (contested)`.** The registry's new *Open contradictions*
section puts it in conflict **C-1** with `AI-SIGHT-006` over **who writes the sight byte
`actor+0xa5`**. `AI-SIGHT-006`'s `disp:a5` sweep finds two writes and neither is a streamer, so
under that arm sight is the constructor's constant **5** for every actor; under this row it is
this file's slot 10, per class. The registry itself notes why a `disp:` sweep would miss the
streamer - the stores go through a pushed address - which is the same blind spot this contract's
own FR-3 rests on being real. **No side is picked here.** What changes if the other arm wins:
slot 10 has no destination and a resolved sight is a constant, which reaches nothing in this
contract, because sight is streamed and never read here. The **Threshold** is untouched: the four
clauses deciding a hashed `MaxHP` are slot 4's destination, the constructor's 30, the adjustment
arithmetic and the resolution rule, and C-1 reaches none of them.

**One address, two attributions, and this story cites both sides.** `retracted.md` records that
`HERO-TARGET-024` calls `L04076` the streamer's store to `actor+0xa5` while `UNIT-STREAM-001`
calls it slot 13's local, pre-set to 0. Both rows are `● active` and both are High on the
instruction. FR-4 above cites the *local pre-set to zero* reading. That is one arm of a live
dispute and is disclosed as such; if the other arm wins, the selector's default has some other
origin and FR-4's "an absent `attackKind` means arm 0" needs a second reading. The **value** it
produces is corroborated independently by `HERO-DMG2-029`, re-tested on two shipped roots, so
what is at risk is the mechanism sentence and not the behaviour.

**`UNIT-PANEL-010` - SUPERSEDED, and the correction runs toward this story.** The row is cited
above as corroboration for FR-3's slot map. `retracted.md` now records that its **scope** was
wrong: the three-way arithmetic is not a display rule at all - the `.alm` spawner runs it on the
actor it has just built (`UNIT-GATE-013`), so the actor's own health, to-hit and defence are
scaled once at placement. That is exactly what FR-7 implements, from `UNIT-GATE-013` rather than
from this row, so the contract was already on the corrected side and the corroboration is now
stronger than it was graded.

**`UNIT-COMBAT-006` - REFUTED, on a figure this story does not use.** "30 of 56 classes have reach
exactly 1" is **38**; 30 was the unarmed count, written beside the measurement instead of from it.
FR-4 cites the row only for arms 1 and 2 being unexercised, at Medium (b), and *Removed* cites it
for the equipment grammar's reach effect on 16 of 56 classes - neither is the retracted figure.

**`ALM-CLS-038` - REFUTED, on the rival this contract already rejects.** "the live `CUnit` holds a
`units.reg` **section index** at `+0x20`" is false: it holds the **`ID`**, and nothing is
translated. FR-5 cites the row for the placement's own fields and resolves by the key unchanged,
which is the corrected reading. Nothing moves.
## Appended 2026-08-02 — pin `9ff259c`: `AI-SIGHT-006` is superseded, and not on the clause C-1 turns on

`AI-SIGHT-006` stays `● active (amended, contested)` and takes a further amendment:
`claims/retracted.md` classes two of its **locations** SUPERSEDED. `R0134` zeroes `0x1000`
**dwords** at `fog+0x24000`, the line-of-sight accumulator — not the `0x10000`-byte visibility map
at `world+0x82ef0`, which a different routine clears; and `fog+0x25450` is not a field but that
accumulator's **centre cell**, so the radius is a seed value and not a scalar to go looking for.
The retraction states in its own words that the clauses the row's High
was argued for stand, **the 10-hit `disp:a5` sweep among them** — and that
sweep is the whole of what C-1 above turns on. **C-1 is unchanged and the Threshold is untouched.**

Separately, `PAL-FACE-005` attaches a consumer to this contract's Units slot **30 `face`**: it is
`unit+0x24`, the subscript of the per-tier palette arm. Corroboration of a slot already decoded here.
