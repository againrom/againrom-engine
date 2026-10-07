# Provenance — a decoded font and a decoded name, over an authored frame

Claim ids are the research submodule's, read at this tree's pin. Grades are the ledger's own:
**High** only where the evidence discriminates against the alternatives, corpus agreement alone
capped at **Medium**.

This story's balance is unusual and is stated once here so the three registers below read correctly.
Everything the panel **draws with** and everything it **says** is decoded. Everything about **how it
looks** is ours, because the interface layer is an unopened area — and that is a published finding
rather than a gap we found.

## Backing

| Spec anchor | Source | Grade |
|---|---|---|
| FR-1 — a unit class carries its own name text as a `units.reg` key, held as the registry's own bytes with no character encoding applied | `REG-UNITS-049` (`DescText` at `class+0x10c`, read as `0x20` bytes) | High |
| FR-1 — every one of the 34 unit classes carries one, and the set of them reproduces the roster | `REG-VAL-029`, `REG-ROSTER-052` | High |
| FR-2 — the **value cache** the display is fed is filled one subject at a time: one instance, one code, one face, no loop and no container | `UNIT-PANEL-010` (the no-multi-selection-arm clause, whose subject is `R0590`) | **Medium** — see the correction below |
| FR-3 — health and its maximum are the pair the display is handed, off the actor and not off the class registry | `UNIT-PANEL-010` (`+0xfc`/`+0x100` in the 25-store block) | High for the copy map |
| FR-3, FR-4 — a string is a byte string in the game's own arrangement: record `k` is character `32 + k`, walked one byte at a time, never transcoded | `SPR16A-FONT-015`, `SPR16A-FONT-018` | High |
| FR-4 — a glyph pixel's 4-bit value indexes a ramp of the caller's colour, and the write is opaque | `SPR16A-FONT-013` | High |
| FR-4 — the pen rule, the letter spacing 2, and that a measured box is the thing a frame is drawn around | `SPR16A-FONT-018`; the placement contract landed at `0050` | High |
| FR-7 — a font is two nodes under `<base>/`, and `font1` is the byte-control atlas identical across both shipped releases | `SPR16A-FONT-018`, `SPR16A-FONT-015`, `SPR16A-FONT-022` | High |

**Threshold.** This story reaches **no** hashed simulation state. It adds no field to an entity,
writes nothing a world carries, and takes no branch inside the determinism wall; it reads a
snapshot the tier above the wall already builds. The Medium-graded rows in the ledgers around it —
the tail definition-table slots, the equipment grammar — are not consumed here at all, because the
numbers this panel states are the simulation's own live values rather than any template's.

## Ours by choice

| Choice | Why it is ours |
|---|---|
| **The whole appearance**: the panel's rectangle, where on the screen it sits, its size rule, its background, its border, its colours, its row order and its label text. | `UNIT-PANEL-011` establishes as a **positive** finding that the layout cannot be settled from what has been read: past the first cached value the display indexes by a computed subscript, so a displacement sweep finds no reader, and which value appears at which position needs the index table and the drawing loop — the interface layer, deliberately scoped out. Its own words are that a consumer has the value set and the arithmetic on evidence and its **layout on nothing**. Owner's ruling that the panel is wanted; the divergence is disclosed and the substitution point is named. |
| **The caption is the class's `DescText`.** | A divergence, not a reconstruction. `UNIT-PANEL-010`'s reference enumeration over the class array establishes, at High, that the display path reads **no `units.reg` field at all** — so neither `DescText` nor `InfoPicture` reaches it, and the original's caption comes from a source nobody has read. `DescText` is the only name text this tree holds, it is per class and it reproduces the roster, so it is the honest stand-in; it is not what the original prints. |
| **Which unit, when several are selected**: the first in the selection's own order. | The original has no multi-selection arm (`UNIT-PANEL-010`) and whether it has a multi-selection *mode* at all is unsettled (`UNIT-PANEL-011`). Describing several would invent a presentation; describing none would waste the selection. Describing the first is the smallest choice consistent with a one-subject display, and the selection's order is already ascending by construction, so it is a rule and not an accident. |
| **Which numbers appear**, and the decision that a value with no live source is **omitted rather than filled**. | No source ranks the fields for a screen. The alternative — printing the definition table's columns — is refused on its own grounds under *Removed* below. |
| **No portrait, and no background art taken from the install.** | `REG-UNITS-049` names `InfoPicture` as a key and nothing published says what the string resolves to, where its node lives or where it is drawn. A guessed node is an invention; an authored flat frame is visibly ours. |
| **The layout is one value, and the panel is a function of it.** | Package and seam shape; nothing in the research speaks to it. It is what makes the substitution above a substitution. |

## Open, and deliberately not consumed

- **Where the original's caption and portrait come from.** `UNIT-PANEL-010` proves, at High, that it
  is not the class registry, and names no alternative. That negative is what this story's caption
  divergence is measured against.
- **Which cached value appears at which position, and whether the display has a multi-selection
  mode.** `UNIT-PANEL-011`, Unknown by construction, with the two things that would settle it named:
  `R0082`'s index table and its drawing loop.
- **The 25-store copy order** published by `UNIT-PANEL-010`. It is evidence about the **copy**, not
  about the screen; `UNIT-PANEL-011` is the row that says the second does not follow from the first.
  Reading it as a field order would be exactly the inference that claim exists to refuse.
- **The absolute colour of the game's text**, and which of its ramps a string gets. Open at `0050`
  and untouched here: the colour is an argument, and this story passes an authored one.
- **How the Russian release displays its CP866 strings** (`SPR16A-TXT-023`). Nothing here depends on
  it: a byte reaches a record and is never transcoded, in either release.

## Removed, and why

- **The definition table's stat block on the panel** — speed, the damage pair, to-hit, defence,
  absorption, the five protections, the five resistances, the primaries, mana, scan range, XP value.
  Refused on two independent grounds, either sufficient. None of them reaches a live entity:
  `mapload.FromALMWith` consumes a resolved definition's health maximum and nothing else. And the
  game reaches that entry point from nowhere — every playable world is built through `FromALM` with
  no table at all. A panel printing them would be printing a class template beside a live unit's
  health while labelling both as that unit's.
- **The three-way difficulty adjustment** (`UNIT-GATE-012`, `UNIT-GATE-013`). It is the correction a
  consumer must apply before printing a **template's** maximum, and `UNIT-GATE-013` is High that the
  spawner already applies it to the actor. This panel prints the actor's own live pair, so the
  adjustment is upstream of everything it reads and cannot be got wrong here — which is a reason it
  is absent, not an omission.
- **A hero's derived stats** (`HERO-HP-005` and the rows beside it). Nothing in this tree builds a
  hero, and no placement here takes the humans arm.
- **`HERO-DAMAGE-022`'s base-and-spread reading.** Correct, and cited by the story that decodes the
  columns; it becomes this panel's problem only when a damage row exists, which is a later story.

## Correction, 2026-08-02 — a routine-scoped Medium clause was cited as a display-scoped High fact

Found by the `0055-selection-count` lane the same day this story landed, after the owner selected
several units in the original and the game answered with a **count**. Recorded here rather than
argued: the backing row above is now corrected, and this section says what was wrong with it.

`UNIT-PANEL-010`'s closing clause reads *"**No multi-selection arm**: one `this`, one code, one face,
no loop and no container."* Its subject is `R0590`, the routine that fills a **25-store
per-unit value cache off a prototype actor**. It says that routine caches one unit. It says nothing
about a selection **count**, and nothing about what any drawing loop puts on screen.

The claim grades itself accordingly — **High** for the copy map, the key and the arithmetic;
**Medium** that this cache is what the display reads. The row above took the clause out of its
routine, promoted it to a statement about the display, and re-graded it to **High** on the way. Two
errors compounding: a scope widening and a grade the source does not carry.

`UNIT-PANEL-011` answers the owner's question directly and is the row that should have been read
first: *which* cached value appears at *which* position, **and whether the display has a
multi-selection mode**, cannot be settled without the index's own table and the drawing loop. The
ledgers therefore record a count readout as **unread**, not as **absent**.

**So no claim is falsified and no claim moves.** The owner found this tree incomplete, which is the
falsification channel working exactly as intended — and it found it within the hour, because the
story shipped something to look at.

The same over-read is in `builds/0053-unit-info-panel/README.md` step 4 ("the original's display has
no multi-selection arm"). `builds/` is untracked, so that copy is corrected in place and leaves no
history; this note is the record.

What this story built is untouched by the correction: describing one subject remains right, and
`0055` adds the count above it through this story's own `PanelLayout` seam without changing a line of
the single-unit picture.

## Correction, 2026-08-02 (second) — *Removed* now overstates, because `0052` closed the hole

The first *Removed* bullet gives two independent grounds for keeping the definition table's stat
block off the panel, and the second of them reads: *"the game reaches that entry point from nowhere
— every playable world is built through `FromALM` with no table at all."*

**That was true when this story shipped and stopped being true the same day.** `0052-flyers-fly`
FR-5 changed the front-end's call from `FromALM` to the with-table entry point, so a placed unit now
takes both its movement domain and its health maximum from its class entry. Verified in the
orchestrator seat against the EN install: on `Waters.alm` the health maximum now varies per class —
48, 77, 123, 82, 10 — where every unit was previously the provisional 100/100.

So the panel's `HP` line already tells the truth about a real class without this story changing, and
that is the change the reader should notice. **The first ground still stands and is the load-bearing
one**: none of the other columns — speed, the damage pair, to-hit, defence, absorption, the five
protections, the five resistances, the primaries, mana, scan range, XP — reaches a live entity, so a
panel printing them would still be printing a class template beside a live unit's health and
labelling both as that unit's. Health was the one column with a consumer, and it is the one that
moved.

The general point, which is why this is a correction and not an edit: **a document that argues from
the state of another package will go stale without anything failing.** No gate can catch it. What
caught it here was that the sibling story landed an hour later and the two were read together.
