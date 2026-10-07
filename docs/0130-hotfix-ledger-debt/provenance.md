# 0130 — the hotfix ledger pays its debt: provenance

This story decodes nothing and builds on no research fact. It moves prose, folds
clauses that were already landed as code, and adds a script. The ledger it
rewrites is the evidence for everything it moves, and each moved cell keeps its
own commit hash beside it.

## Research facts

Nothing is consumed. Three claim ids appear in the archive **only as corrections
to statements that have gone stale**, never as the basis of a clause:

| Claim | In the pin `0908589`? | What it corrects |
|---|---|---|
| `ITEM-ARMSLOT-031` | **yes** | `c9d0a2c`'s *"`ITEM-CODE-029` publishes B=3..13 → `Armor` without saying which of the eleven a helm takes — undecoded"*. The destination is a column of the piece's own `Armors` row, parameter 4, titled `Slot`. |
| `ITEM-CORPSE-034` | **no — research `master` only** | `c9d0a2c` and `026e936` on the boots being *"still open"*. Worn armour does reach the sack, through a `vt+0x44` call the death claim's published sequence steps over. |
| `ITEM-SUIT-035` | **no — research `master` only** | `026e936`'s *"that is a NAME and not a decode — the claim still grades the column's meaning Unknown"*. `sutableFor` is a two-bit mask over consumer classes. |

**The pin gap is disclosed rather than closed.** Two of the three ids were
published after this story's submodule pin and are not readable from it. The pin
is frozen mid-story by design and two sibling lanes are live on the same one, so
this story does not bump it. What it does with those two ids is the narrowest
thing possible: it records, in a frozen archive, that a sentence written on
2026-08-09 is no longer the state of the decode. **No clause folded into any
landed `spec.md` depends on either**, and no code is written from them. A reader
who needs them reads them from research `master`, and the correction lines say
so.

The consequence for the pipeline is worth one sentence, and it is a question for
the orchestrator rather than a finding: the ledger's *"undecoded"* notes are the
kind of statement that goes stale silently, and the archive freezing them means
the next such correction has nowhere to land. That is deliberate — a frozen
record of what was believed is worth more than a live one nobody maintains — but
it means the open questions those rows named (**the boots**, **the death gold**,
**the pre-contact attack cycle**, **the targeting question**, **the town**) now
live only in the archive, and nothing schedules them.

## Ours by choice

- **The archive is frozen and not appended to.** `pipeline/archive/`'s precedent:
  *nothing needs to read these*. A later hotfix writes its detail in its commit
  message and one line in the ledger.
- **A fold amends an existing `FR` wherever one can honestly carry the rule**,
  and mints a new `FR` only where none can. Two of twenty-three rows mint.
- **`FR` only.** Forced by `scripts/check-sdd-audit.sh`, verified by probe rather
  than by reading: an `AC` or `P` added to a landed `spec.md` must be witnessed in
  that story's `verification.md`, and an id removed anywhere is refused as
  dangling. A new `FR` carries no witness demand but **does** demand a mention in
  that story's `plan.md` and `tasks.md` — measured on `0126-sound`, which went
  red with `plan.md accounts for no: FR99` and green again once both named it.

## Open

- Whether the four open questions the archived rows name should become stories is
  the orchestrator's call, not this story's.
- `ITEM-CORPSE-034` says worn armour **does** reach the sack. This tree leaves the
  ten humanoid armour fields on the corpse, which `c9d0a2c` chose on the death
  claim's then-published sequence. That is now a known divergence from the
  original and it owes a story; this one does not fix it (spec, *Out of scope*).
