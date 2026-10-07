# Provenance — the domain column, and what each mask is stopped by

Claim ids are the research submodule's, read at this tree's pin. Grades are the ledger's own:
**High** only where the evidence rules the alternatives out, corpus agreement alone capped at
**Medium**. Where a cited row carries both, the split is named.

## Backing

| Spec anchor | Source | Grade |
|---|---|---|
| FR-1 — the domain is a **column of the definition table**, reaching the mover at spawn from the same row the other numbers come from, and an empty cell leaves the ground value standing | `TERR-MOVE-057` | High — the streamer calls, their source, the ordered slot reads and the `−1` gate are named instructions |
| FR-1 — which slot, and its place in the ordered walk | `UNIT-STREAM-001`, `DAT-SCHEMA-004` | High |
| FR-2 — the three codes and the three stop-sets they select, and that the selector is the mover's only passability state, installed once, at spawn | `TERR-PASS-051`, `MOVE-DOM-025` | High — the predicate, the selector's three arms, its default and the occupancy writers are each named instructions, and the bitmask-vs-enum rival is excluded by the test's own form |
| FR-2 — that a code outside the three **stores nothing**, so the constructor's ground selector stands | `MOVE-DOM-025` | High — the fall-through is a named instruction at a named address, not an absence |
| FR-3 — what each domain is permitted to enter, terrain and non-terrain counted apart: the ground domain stopped by terrain **and** objects, buildings and the border; the middle domain by objects, buildings and the border and by **no** terrain; the air domain by **the border alone** | `MOVE-DOM-026`, `TERR-PASS-051` | High for the rule and for the discriminator — the air bit is set on the border cells and on **none** by any other arm, an exhaustive re-execution rather than a sample / Medium for the cell **figures**, one corpus on one root |
| FR-4 — which occupancy plane each domain contends on: the ground and middle domains together, the air domain apart | `MOVE-PLANE-005`, `TERR-PASS-051` | High — the two occupancy bits' complete writer set, keyed by the same selector |
| FR-5 — that a placed unit's health is its **class entry's** column rather than a constant, that the maximum is written to both fields, and that the scenario setting is applied once at placement with a value that leaves the definition unchanged | `UNIT-STREAM-001`, `UNIT-CTOR-004`, `UNIT-GATE-013`, `UNIT-GATE-012` | High — the slot's destination, the constructor's default, the three-way arithmetic and the value set are each named instructions |
| FR-6 — that the shipped table exercises all three codes and no fourth, and which named classes carry the two non-ground ones | `MOVE-DOM-028`, `TERR-MOVE-057` | **Medium** (a) |

**(a)** A corpus census on one root, capped there by rule. Nothing normative rests on it: it sizes
what the fall-through of FR-2 costs on a lawful install and it tells the owner artifact which
classes to look for. Were it wrong, no clause of this contract would change.

**Threshold.** This story reaches hashed simulation state twice over: a resolved domain becomes an
entity's canonical domain byte, and a resolved health maximum becomes its health pair — both carried
by the byte form and both entering the digest. The five clauses that decide the domain byte — the
column's identity, the empty-cell rule, the three codes' selectors, the out-of-range fall-through,
and which stop-set and which occupancy plane each selector names — are each **High**. The four that
decide the health — the slot's destination, the constructor's default, the adjustment arithmetic and
the resolution rule — were each graded **High** when that number was first loaded, and this story
moves none of them; it only carries the result to a caller that had not been asking for it. Nothing
Medium reaches either.

**Speed is deliberately not carried.** The movement-rate law is not published at this pin, so the
column stays loaded and unread; wiring it would mean inventing a conversion, which is the one thing
a decode may not do.

## Ours by choice

| Choice | Why it is ours |
|---|---|
| The three simulation domains are this tree's own enum, with ground as its zero value | Landed with the domains themselves and unchanged here. The table's codes are the file's numbering and the two are mapped rather than shared, so a table whose codes moved could not silently renumber a shipped byte form. |
| The domain is written at the **same statement** that already writes the resolved health | There is one place a placement becomes an entity, and a second consumer of the same resolution would be a second place for the two to come to disagree. |
| A placement that resolves to no definition is a **ground** mover | It is what a world built with no table gives it, so the arm that yields no stats yields no domain either, and the two halves of "unresolved" stay one case. |
| The front-end **requires** the archive holding the table | The alternative is a front-end that runs without it and gives every unit the ground domain — the exact defect this story removes, restored silently on any install that had lost the file. The three archives already required are required on the same argument. |
| The reporting verb prints the domain beside the arm it already prints | The join is a per-placement fact and the tool that resolves placements is where a reader can see it refuted. |

## Divergence, disclosed

**The middle domain is stopped by less here than in the game, and this story does not close the
gap.** The engine stops that mover on every static object and building; the block plane this tree
derives carries no bit for one, so here it is stopped by the map border alone — the same terrain
set the air domain has, and the two differ in this build by their occupancy plane and by nothing
else. `MOVE-DOM-026` sizes the disagreement at **83 203** of 880 704 shipped cells. The gap is the
plane's rather than the domain's; it was disclosed when the domains landed and it is inherited
unchanged. Concretely: a ghost and a bee cross water here, as they do in the game, and here they
also pass a tree, which in the game they do not.

**Every unit is commandable here.** `MOVE-DOM-028` records that no shipped roster hands a player a
non-ground class, and grades **Unknown** whether a trigger can. This tree models no ownership at
all, so the owner can select and order any placed unit. That is a property of what has not been
built yet rather than a decision of this story, and it is what makes the deliverable watchable.

## Open — deliberately assigned no meaning

- **The static-object bit of the block plane.** Derived nowhere in this tree, and the contract of
  the story deriving structures beside this one refuses any byte that sets it. Until a story
  derives it, the middle domain's object half has no representation and is not approximated by one.
- **The two remaining codes' further consumers.** `MOVE-DOM-024` names six consumers of the domain
  byte and passability is one; the other five — among them a distance term in target acquisition —
  are read by nothing here and are given no meaning.
- **The footprint column that travels beside the domain.** It is streamed from the neighbouring
  slot into the neighbouring byte of the same actor, and this tree has no footprint larger than a
  cell. Carried on the definition, read by nothing, unchanged by this story.

## Removed — what a reader might expect and this story does not assert

| Dropped | Why |
|---|---|
| The two non-ground codes as *two grades of flight* | `TERR-MOVE-057`'s own gloss reads that way and `retracted.md` records it as SUPERSEDED for exactly that reason. They are not a ladder: the two disagree on which block bits they carry, and only one of them passes an object. |
| A confidence claim on the **word** *domain* | `TERR-PASS-051` grades the naming Medium, restored from Unknown. Nothing here rests on the name; the contract is written from the selector and its stop-sets, which are High, and would be identical if the word were wrong. |
| The cell figures as a contract clause | They are one corpus on one root and appear above only to size a disclosed gap. |

## Not consulted

No third-party reimplementation, port, decompilation, format schema or web source was read for any
fact here. The measurements quoted in the analysis were taken with this tree's own loader over the
two lawful installs.
