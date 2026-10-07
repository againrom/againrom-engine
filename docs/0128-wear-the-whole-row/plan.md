# plan — 0128 a person wears his whole row

How the contract is built, and which decisions the contract left to this document.

## What already exists, and is therefore NOT built

The twelve-slot worn set, its byte form, its digest and its container are all already in `pkg/sim`,
and a placed weapon already reaches slot 1 through the loader's own starting loadout. **P-1 is
therefore a statement about work not done**: no simulation field is added, no encoder changes, and
the byte form's version does not move — the number allocated for one goes unspent, which
`verification.md` says rather than leaving it looking forgotten.

What is missing is entirely above the wall: nine of the ten cells are never resolved at all.

## The shape of the change

Four seams, in dependency order.

### D-1 — the parse and the two new resolvers, in the data tier

The tier that already turns a weapon's name into a weapon gains the same for a shield and an
armour. The parse is lifted out of the weapon resolver into one shared walk (P-5) implementing
FR-2a and FR-2b, and the three resolvers differ only by their own two documented steps.

**The existing weapon path is REPLACED BY THE SHARED WALK, not left beside it.** The walk the weapon
resolver carries today is a longest-word-boundary-prefix match; the contract's is a descending
substring find with a positional rebuild. The two agree on every name either has ever been given —
which is a claim this story MEASURES rather than assumes (FR-11 prints both answers where they
differ). Keeping the old one would leave two parses in a tree whose spec says there is one, and
the whole reason the second was written was that the first had never been derived from the original.

**A shield's and an armour's resolved value carries a row, a code and — for armour — its Slot
column, and nothing else.** No damage, no defence, no absorption: the contract's P-2, and the reason
these two types are much smaller than the weapon's.

**A refusal by name is an error and a refusal by slot is not.** The caller must tell FR-5 apart from
FR-6, and a piece whose row resolved is a real piece however its Slot column reads — so the resolver
errors on the name alone and hands the Slot back for the caller to judge, rather than inventing a
second error kind for it.

### D-2 — the loadout, in the loader tier

One function turns a row's ten cells into a worn set and a container overflow. It is the loader's
because the loader is the tier that may read both the item collections and the simulation's widths;
the determinism wall keeps it out of the simulation, and the data tier may not name a slot array.

**It replaces the weapon-only search rather than wrapping it.** That search exists precisely because
only one cell could be read; with all ten read, its "first cell that resolves wins" predicate has
nothing left to do. The weapon POINTER it returns is still needed — the combat fold reads it — so
the new function returns the weapon beside the worn set, which is the shape the caller already has.

**The table gains the two collections it lacks**, and the existing "all of them or none" rule
(P-4) extends to them: a table missing the armour collection resolves no armour cell, and one
missing the shield collection resolves no shield cell, each independently of the other, because a
missing collection is a fact about one class and not about the walk.

**The corpse gate (FR-8) is the same test on the same name**; only the slot count moves, because the
slot it used to name was the only one ever filled. **Cell order is the loop's order (FR-9)**, and the
loop says so, because the displacement reads what an earlier cell wrote.

### D-3 — the panel row, in the window tier

A new panel field, a new subject member holding the already-formatted list of names, and a row in
the authored layout. The value is absent when the list is empty, which is the layout's existing rule
for a row with nothing to state, and is what makes AC-8's second half true without a second code
path: a subject nobody filled composes exactly the picture it composed before.

The row sits directly under the weapon row, because the two answer the same question about the same
subject and a reader who finds one expects the other beside it.

### D-4 — the instrument, and the wiring that lets it run

A developer tool under `cmd/` reads a lawful install and prints the per-class resolution counts, each
resolved cell's destination and every anomaly in full. It is the only thing here that may read a game
file, and its output is what `verification.md` records (AC-6).

The panel instrument is fed here too: it already starts a real mission, so it holds both the world
and the collections, and turning a worn code into a row name is a lookup it can do and the running
window cannot (DD-3).

## Decisions this plan makes that the contract does not

**D-5 — the shared walk lives beside the weapon resolver, not in a new package.** It is three
functions on collections the data tier already declares interfaces for, and moving it would mean a
new tier boundary for no new concept.

**D-6 — the armour and shield resolvers return their own small types rather than a common one.** A
common item type would have to carry the union of what the three classes state, and two of its three
arms would be empty for every value ever built. The one thing all three share — the code — is
already a type of its own.

**D-7 — the tool counts, and separately prints.** Counts are the measurement; printed rows are how a
reader checks the counts are of what they claim. Anomalies print in full, ordinary rows as a bounded
sample, because the collection is shipped game data this repository commits none of.

**D-8 — no fixture in any test reads an install.** Every collection a test resolves against is built
in the test, and the names in them are the test's own inventions, not the shipped ones. The shipped
answer is the tool's job and lives in `verification.md`.

## Risks

**The parse replacement is the one change that can move an already-landed answer.** A generated
character's weapon is resolved through the same walk, and the build already measures that every
literal character generation can name resolves against a shipped file. If the new walk moves any of
those, the walk is wrong — that measurement is the falsifier and it runs in the existing tests.

**AC-4's cases have no shipped witness.** No shipped armour uses a Slot outside the range, so those
rules are tested synthetically alone — which is why the contract states them as rules.

## Design decisions — the rationale behind the contract

The contract states WHAT; each note below says why that clause and not the neighbouring one.

**DD-1 — an unresolvable name is dropped rather than carried.** The original keeps an object it
could not name and puts it in the backpack. This build holds an item as a code, and a code whose row
field is zero names nothing that could later be drawn, dropped or re-equipped, so carrying one would
be carrying a hole. The cost is measured by FR-11 rather than assumed, and disclosed.

**DD-2 — an armour whose Slot column is above twelve is refused into the container.** The original
keeps the value and would address past its own array. That is out-of-bounds behaviour rather than a
rule, so it is not reproduced; no shipped row reaches it, which FR-11 also measures.

**DD-3 — the panel row is filled by the developer tool and not by the running window.** The seam that
pushes an entity to the window every frame carries no collection with which to turn a code into a
name, and giving it one would allocate a string per slot per entity per frame. The panel gains the
ability to state the set; which producers fill it is a separate question, and the one this story
needs is the instrument of FR-11.

**DD-4 — the pieces are named by their ROW names alone.** The panel states `Chain Mail`, not
`Uncommon Steel Chain Mail`. The shape and the material are scaling words; the row is the piece's
identity, and it is the one field of a code already read for a name elsewhere in this tree.

**DD-7 — FR-2b's left-trim is an INFERENCE and is marked as one.** The routine that re-attaches the
implied shape word calls one further method on the subject, taking no arguments, immediately before
it prepends. What that method is is not stated by the evidence this story rests on. Read as a
left-trim it takes the shipped corpus from 902 of 909 cells resolved to 908, with no cell moving the
other way and no other reading tried that does as well; every one of the six is one authored name
carrying a double space, which FR-2a's positional rebuild preserves. The instruction is the image's
and the identification is ours, and it is stated here rather than folded silently into FR-2b's
sentence so that a later decode can overturn it as a single named claim.

**DD-6 — the weapon path takes the new walk too, and two landed tests change with it.** The parse
this tree carried was authored rather than derived, and it differs from FR-2a in two visible ways: it
matched only at a word boundary and only at the start. Two existing tests assert that authored rule —
one directly, one through a fixture whose shape table omits the very entry a real one carries — and
both are corrected rather than worked around. The correction is safe by MEASUREMENT and not by
argument: no row name of the shipped weapon, shield or armour collections contains any entry of
either prefix table, so no shipped name can reach FR-2a's positional rebuild at all. That measurement
is `verification.md`'s and is the falsifier for this decision.

**DD-5 — the shield's slot is not read from a column.** A shield's destination is fixed by its class,
so it is the class constant and not a per-row lookup. A shield collection carries no Slot column to
read even if one wanted to.

## Which seam accounts for which requirement

D-1 carries FR-2, FR-2a, FR-2b and the composition FR-3 reads a destination out of; P-3 and P-5 are
true or false there. D-2 carries FR-1, FR-4, FR-5, FR-6, FR-7, FR-8 and FR-9, and is the only seam
P-1, P-2 and P-4 can break at; AC-1 to AC-5 and AC-7 are its tests. D-3 carries FR-10 and AC-8. D-4
carries FR-11 and turns AC-6 from a claim into a number.

## Order

D-1, then D-2, then D-3, then D-4. Each is one commit.
