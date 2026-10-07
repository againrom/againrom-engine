# Provenance — 0098

Research pin: **`4920d6d`**, read through the submodule at `research/`. Every row below was read at
that pin with `go run ./tools/claim <ID>`; no ledger was opened by hand.

## Claims relied on

| Clause | Claim | Confidence |
|---|---|---|
| FR-1 the group assignment is rewritten every evaluation and cleared when nothing scores | `AI-SCORE-069` | High for the structure and each cited site; **Medium** as a completeness claim (module-scoped instrument) |
| FR-10 strictly-less comparison, seed `0xffffff`, first candidate wins a tie, list order decides | `AI-SCORE-069` | High — the seed and the `JGE` are the routine's own instructions |
| FR-1, FR-4 clearing the assignment ends the engage because the guard arm then writes `ord+0x08` — walk home, or the idle turn at the post | `AI-GRPGUARD-074` | High (one routine read end to end); its `ord+0x00`-unwritten parenthesis is **retracted**, see below |
| FR-5, FR-6 the stand-ground arm writes `ord+0x08 = 0xb` for an AI owner and sends a human participant's unit to `R0015` with no order write | `AI-STAND-076` | High for the behaviour; **Medium** for the name |
| FR-5 stand ground is the player's own `Self` slot and guard is everyone else's | `AI-STAND-076`, `AI-AUTHOR-015` | High |
| FR-3, out-of-scope nothing inside a pursuit arm ends one — no time, health or distance | `AI-PURSUE-040` | High |
| FR-3 what ends a pursuit is the `actor+0x50` arm rewriting `ord+0x08`, and a group under order 1 or 3 never evaluates `actor+0x50` | `AI-BREAK-041`, `AI-ORDER-010` | High for both; **Medium** on each one's completeness sweep |
| FR-8, FR-9 the scorer tests only the low byte of the candidate count, and of the group's member count | `AI-CANDBYTE-110` | **Medium** — both instructions cited and the asymmetry not in doubt; reachability unmeasured |
| FR-8 `AImgr+0xbb4` is the candidate collection's element count, so the count tested is the post-clip count | `AI-CANDCOUNT-105` | High |
| FR-10 list order, and that a group seeing only corpses still has a list | `AI-GROUPSEE-068` | High for the routine; **Medium** for caller completeness |
| the pursuit re-issues unconditionally, and death does not release a target | `AI-REISSUE-077` | High for the mechanism; **Medium** for "no other mechanism ends a group pursuit" |
| FR-11 the scorer's veto and the past-reach refusal both return the seed | `AI-PREF-070`, `AI-REACH-072` | High |

## What moved under this story's feet

`AI-GRPGUARD-074` is **amended** at this pin and `retracted.md` carries the overturn: its clause
*"for a group under order 1 from load nothing has ever written `ord+0x00`"* is **refuted** by
`AI-POST-095` — the load-time guard setter `R0125` writes the post itself, `0x76` bytes
before the `grpAI+0x20 = 1`. `AI-POST-096` retracts `AI-POST-042`'s complete-writer-set clause and
`AI-POST-097` retracts *"for a guard it never moves"*.

`pkg/sim/engage.go`'s header repeats the refuted clause as live fact. It is corrected by this story
because this story rewrites the paragraph it sits in, not because the walk home comes into scope —
it does not. The arm's three outcomes, which are what FR-1 and FR-4 rest on, are unaffected by the
overturn and the amended row says so explicitly.

## Ours by choice

- **Standing still in place of the walk home and the idle turn.** Both are out of scope and neither
  has the state it needs here. FR-4 discloses it.
- **The release drops the destination as well as the victim.** The law replaces the pursuit order;
  this build erases it, and erasing only the victim would leave the approach's own destination
  behind and the member would finish walking to where its quarry stood. No claim covers this —
  it is a consequence of this build's phase order, and the choice is the one that reproduces the
  law's *effect*.
- **FR-2, the release is a no-op on a member holding no victim.** The law's arm writes an order for
  every member on the clear path regardless of what it held. Restricting ours to members that held a
  victim gives up nothing this build has (the orders it would write are out of scope) and buys
  independence from any story about commanded units.
- **FR-8 reproduced rather than dropped.** The alternative was to test `len(cands) == 0` and record
  the byte edge as an unimplemented limit. Reproducing it is one conversion, keeps this build's
  standing habit of carrying a decoded field's width, and keeps the law's two-site shape.
- **FR-5 expressed through stance rather than through owner.** The two select the same members on
  every world this build can construct; stance is the property `decide` already holds. FR-6 is the
  seam.

## Open

- **`R0015`** — the branch a human participant's unit takes when its group assignment is
  cleared — is unread at this pin. If it writes `ord+0x08`, FR-5 is too generous and a stand-ground
  member should release too. The conservative reading is the one this build already has, so nothing
  is blocked; a research row on that routine would settle it.
- **`AI-CANDBYTE-110`'s reachability** is unmeasured: whether `0x100` simultaneously visible
  candidates occurs on any shipped map. FR-9 asserts it does not for maps this build can load and
  offers no census. A census would close it; the limit stands either way.
- **The remembered-attacker memory** (`AI-GROUPSEE-068` phase (b), `ord+0x58`/`ord+0x5a`, 20 group
  ticks) is a **named omission**. This build releases sooner than the law does for a struck AI-owned
  member. It is per-member state with its own clock and belongs to its own story.

## Removed

Nothing was removed from the contract during planning. Two things were considered and left out
deliberately: giving a released member a facing (there is nothing to face), and clearing on the
stand-ground stance (FR-5, and the reason is in *Open* above).
