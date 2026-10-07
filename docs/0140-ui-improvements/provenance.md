# 0140 — provenance

## Claims this story rests on

| Claim | Confidence | What it carries here |
|---|---|---|
| `UNIT-COMBAT-015` | High | The per-class combat column set and its **column order**. Slots 19..23 are `prot Fire..Astral` and 24..28 are `res.Blade..res.Shooting`; that binding is what says the generation screen was printing the wrong family, and it is the order the five protection labels are drawn in. |
| `HERO-RESIST-012` | High | The five elemental protections are **Spirit halved**, clamped. Used only to know that the row **moves** when the player spends Spirit — which is what falsified the old note's "constant across every spread". No arithmetic of it is reimplemented here; `pkg/data`'s mint already owns that. |
| `HERO-APPEAR-051` | High | A figure sheet lives under `graphics\equipment\<figure directory>\`, and the figure directories are **four**, chosen by a mage bit and a sex bit. This is the whole basis for refusing to draw a doll for a non-human unit: there is no fifth directory to address. |
| `UNIT-PANEL-011` | — (a negative result) | That **which cached value the original's information display shows cannot be settled at all**: one cache, three writing routines, an indexed read. It is why this project's panel arrangement is authored and says so, and why 0140 could take the owner's photograph as a design without contradicting anything. |
| `AI-MINIMAP-062` | High | The original has a **second click surface** with six arms and five gates. Cited to bound what the minimap press reproduces: **none** of it. One gesture, chosen by the owner. |

## Claims cited as a bound, whose state matters

`UNIT-PANEL-010` — **amended, and the amended part is not the part used.** The row's *scope* was
superseded by EXP-0075 (it is not a display-only rule; the `.alm` spawner runs the same arithmetic
at placement). What survives at High is the **copy map** — the 25-store block and its file order,
including that a `sight` value is copied out of the actor. That is all `pkg/mapload/sheet.go` cites
it for, and the citation says in as many words that neither this row nor `UNIT-PANEL-011` says the
units collection's scan-range column *is* that value for a creature.

`UNIT-COMBAT-006` — **retracted, and the retraction is not the part used.** What was taken back is
the *reach-1 count* ("30 of 56"). The surrounding field↔column bindings are untouched and were
never inside the retracted cell. `pkg/ui/panel.go` cites the row for the order a damage resolver
indexes the protection five in — a binding, not the count — to say why relabelling those five
without reordering the array would put a true number under a false name.

## Ours by choice, not decoded

Everything about **where a box stands and how large it is**: `sidebarWidth` and its 300 pixels, the
margin of 12, both bars' cell sizes and gaps, the two-row book, the four switches' letters, their
size, their order and their colours, the scroll step of one cell, the doll box's and worn box's
proportions, and the whole of `hud.go`'s arithmetic.

The **window the game opens at**. That it is the monitor's rectangle, that it is undecorated, and
that a monitor reporting nothing usable falls back to a decorated `MenuWindowW x MenuWindowH`.

The **generation screen's place in the flow** — a picker mission row and `-mission N` open it, a
campaign successor does not. This is an owner ruling and reverses 0119 plan DD-18, which was also
ours.

That a creature's **scan-range column is what a sheet calls Sight**. Stated at the field.

A spellbook cell carrying an **abbreviation instead of an icon**. The original plainly draws one
picture per spell; nothing published names where that art lives.

The **five protection labels' spelling** (`Fire / Water / Air / Earth / Astral`) as English words.
The column order is `UNIT-COMBAT-015`'s; the words are ours. The other five —
`res.Blade..res.Shooting` — are printed as **one unnamed slash-separated row** on the generation
screen precisely because five of the ten protection/resistance columns have no published title, and
a wrong name on a screen reads as a decoded fact.

## Open — research owes this

**Where a non-human unit's doll picture lives.** `HERO-APPEAR-051` gives four figure directories and
they are the four human ones. The owner states that enemies have a doll picture in the original;
that is testimony and therefore a question, not a verdict. Until a claim names the address, the doll
box shows the selected unit's own **drawn world frame** — visibly not a doll, and disclosed at
`dollSubject` as a substitution rather than an implementation of the ruling.

## Removed

Nothing decoded was removed. Two **authored** clauses were: 0119 plan DD-18's "entering a mission
must not open generation", replaced by the owner's ruling above and by the successor boundary that
was the true content of it; and 0118 AC-12's "the minimap takes no press", which shipped with two
tests enforcing it and whose reversal the owner asked for. Both reversals are named where the
original decision was written, not only here.
