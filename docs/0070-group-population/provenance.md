# Provenance — the group a check counts

Research is the `research/` submodule at the pin this branch carries, **`f56bb38`**, unbumped for the
whole story. Claims are cited by **id** and never by experiment, so a later amendment or retraction
reaches this story through the ledger that carries it.

## Backing

| spec anchor | what it rests on | claim(s) | confidence |
|---|---|---|---|
| FR-1, the entity's identifier | the placed-unit record's `+0x42` word is the **group** id and `+0x40` the unit id; the earlier row had the two labelled the wrong way round | `ALM-UNIT-048` (which corrects `ALM-UNIT-040`) | **Medium** — the distinctness counts and the trigger-grammar resolution are whole-corpus and refute the rival, but the labels rest on the grammar rather than on an instruction dereferencing either field |
| FR-1, the identifier's width | the loader's case-6 read sequence takes four bytes into the struct field and keeps its running maximum | `ALM-UNIT-040`, `AI-GROUP-009` | **High** — a read boundary in a sequence that sums to exactly seventy |
| FR-2, which parameter names a group | parameter type code `2` is `Target_Group`, and 169 of 169 references over the shipped corpus resolve against the placed-unit record's group word | `ALM-TRIG-046` | **Medium** — whole-corpus exact against a domain another record type publishes; an id absent from the unit records would show and none does |
| FR-3, what a group is | the engine forms a runtime group at spawn by **equality of that same word**, matching the loader struct's field; the file-to-struct binding is a read boundary, not an inference | `AI-GROUP-009` | **High** for the mechanism; **Medium** for its corpus figures, which replay the rule rather than observe the engine |
| FR-3, that the arm exists and is arm 1 of 22 | the condition vocabulary is a 22-arm jump table, opcodes 1..22, read out of the image with its own bound, two arms dead and three structurally odd | `TRIG-COND-003` | **High** for the table, its extent and the dead arms |
| FR-3, what the count is *for* | the campaign's first mission decoded end to end: the win is a two-step escort whose first trigger is gated on this arm reaching zero | `MISSION-M10-009` | **High** — the whole type-7 payload is consumed exactly and every parameter resolves |
| FR-6, what a sack is and why the other arm is not answerable here | a sack is a `Token` in the world's own per-cell registry, holding gold and a container of the class a unit holds; it is made by a kill or by a mission overlay, and picked up through an order that names a cell | `ITEM-SACK-010`, `ITEM-CLASS-001`, `ITEM-STACK-003`, `ITEM-SPAWN-013`, `HERO-KILL-027`, `ITEM-CMD-007`, `ITEM-PICK-009`, `ITEM-PICK-016` | **High** across the ledger for what a sack *is* — and see *Open*, because none of it says what the **check** writes |

The corpus shape this story sizes itself against is `AI-GROUP-009`'s census over 38 maps and 8094
placed units: **2264 groups, mean 3.58 members, 686 singletons, largest 133, and only 5 units
carrying group id 0**. The last figure is why the identifier's zero is treated as a real group.

## Ours by choice

| what the spec fixes | why it is ours, and what it costs to change |
|---|---|
| the identifier lives **on the member**, not on a group object the world owns | nothing in this tree adds, removes or moves a member, so the two are one observable behaviour. The seam is the same one the rate term already names; a story that can change membership after a spawn moves the field and its readers together |
| a **presence flag** for the check's group reference rather than a reserved identifier value | the identifier's zero is a real group, so no value is free to mean "none" — the same reason the entity id needs a presence byte |
| the byte form's **next version**, with both widened records in one bump | one version per landed change to the form; the record widths are the form's own, not a claim's |
| the count iterates the world's own ordered entities | determinism: no map iteration reaches an answer that enters the digest |
| membership ignores the owning player | `AI-GROUP-009` keys on the pair; this tree carries no placed-unit owner at all, so the refinement has nowhere to live yet |

## Open

1. **What the arm's body reads, and whether it counts the dead.** `TRIG-COND-003` publishes the
   table, its extent and three arms' structural oddities; `TRIG-DIST-014` publishes the five distance
   arms. **Arm 1 is in neither**, and no claim at this pin states what it writes into its slot. The
   spec takes the living-members reading because it is the only one under which a shipped trigger
   comparing this count against zero can ever hold — but the whole-membership rival is not excluded
   by anything read. *A request to research: read arm 1 of `R0433` and publish what it counts.*
2. **`Target_Group` when one identifier has two owners.** `ai.md` carries this as an open item: the
   runtime group is keyed on the pair and the script names an identifier alone, and which runtime
   group the resolver returns was not read. One shipped patrol node is known to reach it.
3. **What check arm 14 writes into its slot.** The item ledger settles what a sack is, where one
   comes from and how one is picked up; none of it is a statement about the check. This is the
   narrow gap, against a ledger that was read.

## Removed

| statement dropped | why |
|---|---|
| that this story implements **both** unimplemented check arms of the campaign's first mission | the arm-14 half was measured out. Over both installed roots it moves no map into the set whose win instant is reachable, its only reader on that mission raises a message, and this tree's world holds nothing for it to measure — so any implementation of it would answer without measuring, which is what the inert-trigger rule exists to prevent. It stays reported and its readers stay inert |
| that implementing these checks makes the first mission winnable | the same measurement says the chain has a second gate that is not a check: the instant that hands the escortee to the player. Nothing here asserts a win becomes reachable |
