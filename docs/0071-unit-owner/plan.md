# Plan — the owner a script hands over

One field carried from the map to the world, three references added to the compiled instant record, and
two dispatch arms. Two tasks: everything that widens a record first, with both arms still
unimplemented and the report still naming them; then the arms.

## Design decisions

**DD-1 — the owner rides on the entity as a plain number, beside the class key.** The rival is a roster
object that entities point into. It is rejected because there is nothing to point at: this build has no
player, no roster and no group object, and inventing one to hold a single word would put a second
representation of the map's roster in the tree before anything reads the first. The word the map wrote
is the whole of the state, exactly as the class key is (FR-1, FR-2). It also means the decoder change is
one field: the map document keeps its own bytes and writes them back untouched, so the round-trip is
unaffected by the field existing (AC-1).

**DD-2 — zero on the entity is the absence, and no presence flag is added.** The identifier space begins
at 1, so zero cannot be a roster entry and does not have to be told apart from one. This is deliberately
the opposite of the group identifier, whose zero is a real group and which therefore needs a flag on the
compiled *reference*; the two differ because the two identifier spaces do, not because two authors chose
differently (FR-2). The consequence to watch is the one below.

**DD-3 — a felled entity keeps its owner, and the constructor's not-alive block must not grow a clause
for it.** That block clears the order, the transit pair, the group rate term and the attack — everything
that is *residue of a state the unit has left*. An owner is not residue: the fallen stay on their
roster, exactly as they stay in their group, and both arms are specified to write them (FR-4). The
decoder's matching family of refusals must not grow one either, or a world this package can build
becomes one it cannot read back (P-3).

**DD-4 — the instant's three references are references with presence flags, not plain parameters.** The
binder packs plain integer parameters in encounter order, so admitting the unit, group or player types
into that set would shift the parameters of every *other* node that carries one — including the arms
this build does not run, whose compiled records would silently change under it. The three therefore sit
beside the plain parameters with a flag each, which is the shape the compiled check already uses for its
own references (FR-3, AC-3).

The group and the player are **direct copies of the node's value**: each names exactly one thing and
needs no table. The unit is **resolved**, through the same three identifier bands and the same caller
supplied table a check's unit reference uses, and an unresolvable one is added to the same report — the
binder already resolves and reports action-node unit references and then discards the result, so this is
a plumbing change and not a new mechanism.

**DD-5 — one version bump, both record widenings, one commit; and the number is not written here.**
Both new pieces of state are canonical (FR-6), so both cross the form. The
entity record grows by four bytes and the instant record by fifteen — four each for the unit, group and
player values and one each for their presence bytes — both at the **tail** of their record, as every
widening before them, so no existing offset moves. Splitting them across two commits would leave an
intermediate form that is a third shape wearing one of the two version numbers.

The version itself is deliberately *not* fixed in this document. A sibling story widens the same two
records and takes the next number, and two stories that each read "the next version" off the same
starting tree pick the same one. The number is allocated once, outside this story, and the task that
takes it is the only place it is written.

Two consequences are easy to miss and are named. The widths and the version are pinned in **hand-written**
byte-shape assertions, several per file and in more than one package, each spelling a literal — every one
moves together or the pin stops meaning anything. And a widened instant record must keep its presence
bytes **refused** outside their value set rather than read as truthy, or the form stops being injective
(AC-7, P-3).

**DD-6 — opcode 22 is a linear scan of the world's entities, and no membership index is built.** An index
would be a second representation of a fact this arm is the only thing in the tree that changes, and it
would have to be maintained by the arm that invalidates it. The world's entities are already ordered and
already scanned by other arms; the largest group in the shipped corpus is 133 members, and the arm runs
once per firing of a one-shot trigger. Scanning the world's own ordered slice is also what makes P-1's
storage-order independence a property of the data structure rather than of the arm.

**DD-7 — opcode 19 resolves through the world, not through the compiled reference alone.** The compiled
unit reference is an entity id fixed at compile time, and the world it runs against may no longer hold
that entity. The arm therefore looks the id up and does nothing on a miss, which is the same treatment
every check's unit reference already gets and the same answer FR-5 gives an absent reference (AC-5, P-2).

**DD-8 — the supported-arm table stays the single answer.** Both arms become supported by adding them to
the one table the compile-time report and the runtime dispatch both consult, so the report and the
dispatch cannot come to disagree about which arms exist. Nothing about inertness changes, and nothing
needs to: inertness is derived from unimplemented **checks**, and this story adds no check (FR-7).

**DD-9 — FR-7 is verified by a differential, not by an assertion that nothing happened.** "No trigger
changed state" is not observable from one run. The criterion is met by building one script twice over
identical triggers — once with these two opcodes and once with an opcode this build still does not run —
and comparing the inert set, the triggers evaluated and the latch vector after a pass. That comparison
fails if either arm is ever made to poison a register or to abort a trigger, which is the failure worth
guarding (AC-8).

**DD-10 — no reader is added anywhere, and the two premises this story falsifies are left standing.** The
far-search budget's owner term and the front end's unrestricted selection are both correct today *because*
no entity carries an owner; after this story both are correct only because nothing consults the field.
Changing either needs to know which roster entry the human participant is, which no map states, so
touching them here would be inventing the answer rather than deferring it. The existing prose that states
the premise is updated to say that an entity now carries an owner and that this arm still does not read
one — a premise that has quietly become false is worse than one that was never written.

## Tasks

T1 — the owner is carried, from the placed record to the entity, the compiled instant's three references
are carried with it, and the byte form takes its next version; both arms still unimplemented and still
reported.

T2 — the two arms hand ownership over, and the report stops naming them.

## Success criteria

**SC-1** A placed-unit record set with distinct owner words decodes to those words and writes back
byte-identical; a world built from that map gives each entity its record's owner and a world built from
no map gives every entity zero (AC-1, AC-2).

**SC-2** An instant node with all three reference kinds, one with none, and one with references beside
plain parameters compile to the references and the plain-parameter slots AC-3 states, and an
unresolvable action-node unit reference is reported (AC-3).

**SC-3** A world holding entities with distinct owners — **including a felled one still carrying its
owner** — and a script holding both arms marshals and reads back record for record; two worlds differing
only in one owner have different digests; a form at the previous version is refused; a truncated widened
record is refused; a reference presence byte outside its value set is refused (AC-7, P-3).

**SC-4** With T1 landed and T2 not, the unsupported report still names both arms and no owner changes
when a trigger carrying them fires.

**SC-5** Opcode 22 over a world of two groups with a dead member in the named one, over a group no entity
carries, and twice in one trigger for two groups, writes the owners AC-4 states (AC-4, P-1, P-2).

**SC-6** Opcode 19 over a world writes the one entity AC-5 states, and changes nothing for an entity the
world no longer holds (AC-5, P-2).

**SC-7** Each of FR-5's three no-reference cases leaves every owner unchanged over a world whose owners a
successful arm would have overwritten (AC-6).

**SC-8** The differential of DD-9 agrees on the inert set, the triggers evaluated and the latch vector
(AC-8), and the two-trigger hand-over choreography runs to the owners AC-9 states (AC-9).
