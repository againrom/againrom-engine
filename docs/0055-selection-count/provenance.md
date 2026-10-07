# Provenance — an authored label over a number the build already knows

Claim ids are the research submodule's, read at this tree's pin
(`acb8fb085757f5436ebb5f737255fe1630662829`).

## The question this story opened with

Owner play against the original reported a selection count on screen. `0053` had shipped a panel
that describes one unit and no count, citing `UNIT-PANEL-010`, so the first thing owed was whether
the owner had **falsified a published claim** or merely found this tree incomplete.

**It is the second, and no claim moves.** The clause is `UNIT-PANEL-010`'s closing sentence — "**No
multi-selection arm**: one `this`, one code, one face, no loop and no container" — and its subject
is `R0590`, the routine that **fills a 25-store value cache off a prototype actor**. It says
that routine caches one unit; it makes no statement about a selection count, and none about what any
drawing loop puts on screen. The row's own grades say so: High is claimed for "the copy map, the key
and the arithmetic", while "**that this cache is what the display reads**" is graded **Medium** and
is explicitly "bounded by `UNIT-PANEL-011`".

`UNIT-PANEL-011` then answers the question outright, as a negative about the instrument at High:
"which cached value appears at which position, **and whether the display has a multi-selection
mode**, cannot be settled without the index's own table and the drawing loop" — the interface layer,
scoped out — and the positive question is "**Unknown** by construction". A count readout is
therefore not something the ledgers deny. It is something they positively record as unread.

**One correction is owed upstream, and it is ours, not research's.** `docs/0053-unit-info-panel/`'s
`provenance.md` backs FR-2 with the row "the original's information display describes **one**
subject … `UNIT-PANEL-010` (the no-multi-selection-arm clause) | **High**". That reads a
routine-scoped clause as a display-scoped fact and re-grades to High what the claim itself grades
Medium. The same file's *Ours by choice* and *Open* sections state the position correctly — "whether
it has a multi-selection *mode* at all is unsettled (`UNIT-PANEL-011`)" — so the document contradicts
itself and the wrong half is the one in the backing table. Nothing in this story is built on that
row; it is reported here so the correction is made where the record is, and not by quietly citing
something else.

## Backing

| Spec anchor | Source | Grade |
|---|---|---|
| The panel's layout, row set and label text are ours to author, behind a named substitution point | `UNIT-PANEL-011` (the layout is on evidence of nothing; the interface layer is unopened) | High as a negative about the instrument |
| Nothing published settles whether the original states a count on this panel, elsewhere, or at all | `UNIT-PANEL-011` (Unknown by construction) | — |
| The values the panel already states are the actor's own live numbers, not a template's | `UNIT-PANEL-010` (the copy map) | High for the copy map |

**Threshold.** Presentation only. This story reaches no hashed simulation state, adds no entity
field, takes no branch inside the determinism wall and moves no world's byte form, so the ordinary
confidence threshold applies rather than High.

## Ours by choice

| Choice | Why it is ours |
|---|---|
| **The label's wording.** | The owner transcribed what they read in the original; a transcription from play is testimony and a question, not a decoded resource, and hardcoding it would ship a guess at the original's string dressed as evidence. The only published pointer at a place such a string could live is `SPR16A-TXT-023`'s incidental note that `MAIN.RES` holds UI strings — no claim decodes them, indexes them, or connects one to this readout. So the label is authored, disclosed, and replaceable at the same one value the rest of the panel's appearance is. |
| **That the count is stated only from two up.** | Absent below two is what makes the single-unit panel byte-identical to the one before this story, so the count cannot regress what `0053` shipped. Nothing decoded says what the original does with a selection of one. |
| **That the described unit stands, and the count is stated beside it.** | The alternative — replacing the description with a summary once several are selected — would invent a second presentation on top of an already authored one. `UNIT-PANEL-011` leaves both unevidenced; stating the count in addition is the smaller of the two inventions and leaves `0053`'s subject rule untouched. |
| **Its position in the authored row order** (first, above the name). | Layout is ours by the row above. |

## Open, and deliberately not consumed

- **Whether the original's display has a multi-selection mode at all**, and what it shows in it.
  `UNIT-PANEL-011`, Unknown by construction, with the two things that would settle it named:
  `R0082`'s index table and its drawing loop. **The owner's observation is a live question for
  research**, and a good one: it is testimony that the interface layer holds a readout, which is
  evidence about where to look and not a decoded fact.
- **Where the original's UI strings are held and how they are indexed.** `SPR16A-TXT-023` observes
  Russian UI strings in the RU `MAIN.RES` while establishing something else entirely; no claim
  decodes that container's string layout, so no wording can be cited from it today.
