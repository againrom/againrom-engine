# Plan — the group a check counts

One new arm, and the state it needs carried from the map to the register file. Two tasks: everything
that changes the byte form first, with the arm still unimplemented; then the arm.

## Design decisions

**DD-1 — the identifier is carried on the entity, not resolved to a member list at compile time.**
The rival is to have the binder turn a group reference into the list of entities that were in it and
store that list in the compiled check. It is rejected on two counts. The compiled check is a
fixed-width record and a member list is not, so it would need a fourth counted array in the script
section for a fact the entity can hold in one field; and the engine's own rule is an **equality on
the placed record's group word**, evaluated when a unit is spawned, so an identifier on the member is
what is being reconstructed rather than a stand-in for it. Carrying it also leaves a customised map
free to regroup without the compiled program changing shape.

**DD-2 — the check's group is a reference with a presence flag, not a plain parameter.** The binder
packs plain integer parameters in encounter order, and admitting the group type into that set would
shift the parameters of every *other* node that carries one — including arms this build does not
evaluate, whose compiled records would silently change. So the group joins the two unit references
instead, with a presence flag of its own, because the identifier's zero is a real group and no value
is free to mean absent (FR-2, FR-4). It is a **direct copy of the node's value**, not a table lookup:
a unit reference is resolved through the caller's id table because three id bands name three
different things, and a group reference names exactly one — the placed record's own word. No groups
table is added to the reference struct.

**DD-3 — one version bump, both record widenings, one commit.** The byte form takes version **11**.
The entity record grows from 83 to 87 bytes and the check record from 58 to 63 — four bytes for the
identifier and, on the check, one more for its presence byte. Both go to the **tail** of their
record, as every widening before them has, so no existing offset moves. Splitting the two would leave
an intermediate commit whose form is a third shape wearing one of the two version numbers.

Two consequences are named because they are easy to miss. The record widths and the version number
are pinned in **hand-written** byte-shape assertions — deliberately, so they do not assert the
encoder against itself — and those pins live in both `pkg/sim` and `pkg/mapload`, several per file,
each spelling the old width or the old version as a literal. Every one moves together or the pin
stops meaning anything. And the decoder's "a felled unit cannot carry this" family of refusals must
**not** grow a clause for the identifier: membership outlives death here by construction, which is
precisely what the count's transition through zero depends on.

**DD-4 — the count is a linear scan of the world's entities, in the order the world holds them.**
There is no membership index and none is built. The world's entities are already ordered and already
scanned by other arms; an index would be a second representation of the same fact, and the fact it
represents is one nothing in this tree mutates. The largest group in the shipped corpus is 133
members of at most a few hundred entities, once per check per pass.

**DD-5 — the supported-arm table stays the single answer.** The arm becomes supported by adding it to
the one table that both the compile-time report and the runtime dispatch consult, so the report and
the dispatch cannot come to disagree. The dispatch arm is added beside the arms that take no unit
reference, because this one takes none either.

**DD-6 — the loader change is one field at the one site that builds entities from placed records.**
Every world in the tree is built through that site, so nothing else needs to learn about groups. A
world built with no map keeps the field's zero, which DD-2's reasoning already makes a real value.

**DD-7 — dead is the runtime's existing predicate, unchanged.** The arm calls the same one-place
definition of dead every other arm uses, so a downed unit counts as dead here for the same reason it
does there, and a later change to that definition reaches this arm without an edit.

## Tasks

T1 — the group identifier is carried, from the placed record to the byte form, with the arm still
unimplemented and its readers still inert.

T2 — the arm counts, and the first mission's win chain runs.

## Success criteria

**SC-1** A world built from a map whose placed units carry distinct group identifiers, and one built
from no map, give the identifiers AC-1 states (AC-1).

**SC-2** A check node with a group parameter, one without, and one with a group parameter beside
plain parameters compile to the references and the plain-parameter slots AC-2 states (AC-2).

**SC-3** A world holding grouped entities — **including a dead one still carrying its group** — and
a group check marshals and reads back record for record; two worlds differing only in one identifier
have different digests; a form at version 10 is refused; a truncated widened record is refused; a
group presence byte outside its value set is refused (AC-5, P-3).

**SC-4** With T1 landed and T2 not, the unsupported report still names the group arm and its readers
are still inert. With T2 landed, a check of the arm carrying no group reference leaves a preset
register untouched while its trigger is evaluated as a live one (AC-4, FR-6).

**SC-5** The arm's register takes the five values AC-3 enumerates, and two checks of the arm naming
different groups in one pass write their own registers (AC-3).

**SC-6** A hand-built script mixing the group arm with the sack arm reports only the sack arm and
leaves only its readers inert (AC-6, FR-6).

**SC-7** A synthetic world reproducing the first mission's chain reaches a win when the named group's
last member dies (AC-7).

**SC-8** The full suite passes and the import-graph and determinism-wall checks are clean, with no
pinned digest edited except where a widened record makes it a different world (P-1, P-2).

## Risks

**R-1 — the count's meaning is the story's one unforced choice.** If the arm counts every member
rather than the living ones, every trigger this story arms is armed with the wrong number and none of
them can ever hold. Mitigated by SC-5 asserting the transition through zero rather than a single
value, so the behaviour is pinned and a later correction is one predicate and a failing test rather
than an archaeology.

**R-3 — the byte-form pins are literals in two packages, not derivations.** A widening that misses
one leaves a test that either false-passes or spuriously fails, and neither says which. Mitigated by
DD-3 naming the pins as a set that moves together, and by SC-3 testing that the field reaches the
digest rather than that a digest changed.

**R-2 — widening two records at once moves every digest the suite pins.** A pinned digest that
changes for the *right* reason is indistinguishable from one that changes for the wrong one.
Mitigated by SC-3's differential — two worlds differing only in an identifier — which tests the field
reaches the digest rather than that the digest changed.

## Traceability

| requirement | criterion | task |
|---|---|---|
| FR-1 | SC-1, SC-3 | T1 |
| FR-2 | SC-2, SC-3 | T1 |
| FR-3 | SC-5, SC-7 | T2 |
| FR-4 | SC-4 | T2 |
| FR-5 | SC-3 | T1 |
| FR-6 | SC-4, SC-6 | T1, T2 |
| AC-1, AC-2 | SC-1, SC-2 | T1 |
| AC-3 | SC-5 | T2 |
| AC-4 | SC-4 | T2 |
| AC-5 | SC-3 | T1 |
| AC-6 | SC-6 | T2 |
| AC-7 | SC-7 | T2 |
| P-1, P-2 | SC-8 | T2 |
| P-3 | SC-3 | T1 |
| DD-1, DD-2, DD-3, DD-6 | SC-1, SC-2, SC-3 | T1 |
| DD-4, DD-5, DD-7 | SC-5, SC-6, SC-7 | T2 |
