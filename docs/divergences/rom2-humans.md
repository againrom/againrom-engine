# Divergences -- Second game roots -- Human derive

2 row(s). Part of the split described in `docs/DIVERGENCES.md`: read that file first for what a row means, what each column holds, and how a row is found.

## Divergences

Implementation and researched ROM1 behaviour differ, or the owner ruled against a claim.

| ID | Subsystem | Owner directive | ROM1 behaviour (claims) | Implemented behaviour | Type | Reason | Revisit condition | Status |
|---|---|---|---|---|---|---|---|---|
| DIV-2786 | ROM2 Human derive / statistic caps | - | `R2-ENGINE-290` (High for the arithmetic): the ROM2 recompute caps Body, Agility, Mind and Spirit at 52/50/48/46 for a male fighter, 50/52/46/48 for a female fighter, 48/46/52/50 for a male mage and 46/48/50/52 for a female mage | A ROM2 Human goes through the one Human derive (`data.DeriveHuman`), which caps each statistic at 50 plus its modifier term (`HERO-CAP-015`, ROM1) | FIDELITY-DEBT | The derive takes no game-specific cap; no ROM2 input carries the class-and-sex caps | A per-game cap input to the derive with a ROM2 control on the four templates | OPEN |

## Authored where research is silent

No ROM2 claim answers the question, so the implementation authored one.

| ID | Subsystem | Owner directive | ROM1 behaviour (claims) | Implemented behaviour | Type | Reason | Revisit condition | Status |
|---|---|---|---|---|---|---|---|---|
| DIV-2785 | ROM2 Human derive / overload penalty and speed modifier order | - | `R2-ENGINE-290`: the ROM2 speed is Agility below 12, else Agility/5 + 12, plus 10 for type 0x13 or 0x15, minus the load term, minimum 6, and item modifiers follow (High for the arithmetic). The load term itself is Unknown there; no ROM2 claim states its operands, the capacity or the clear of a negative sum | A ROM2 native Human runs the ROM1 rule (`SAV-1116`, `HERO-SPEED-008`): load at or above capacity Body x 10 + 1 subtracts load/capacity before the modifier, with the floor of 6, and a negative sum clears the modifier (`rules.HumanSpeed`) | UNKNOWN | The ROM2 track builds ROM2 behaviour from ROM2 evidence; the load term has none, so the ROM1 rule stands in | A ROM2 claim on the load term and the modifier clear, with an overloaded ROM2 control | OPEN |
