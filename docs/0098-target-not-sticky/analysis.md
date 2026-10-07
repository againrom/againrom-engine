# Analysis — an acquisition is not a life sentence

## What was reported

The owner, playing the build: everything drifts and eventually collides in one brawl, and *the
village does not kill squirrels in the original game*. `0095` was opened for that and refuted its
own premise — freezing the notice radius changed nothing measurable on shipped content (28 missions,
2000 ticks, no commands: 2357 entities, 79 moved, 501 cells, 35 died, identical before and after and
on both roots). Its lane named the mechanism instead: `pkg/sim/combat.go:230` re-aims an attacker at
its victim every tick, so **one acquisition is a standing instruction to close the distance**, and
this build has nothing that withdraws one.

## What we did not know, and what the pin says

**Is this build's group layer sticky?** Literally, no. `AI-SCORE-069`'s `ord+0x20` is a **field**;
this build's group choice is the local `at` inside `decide`, born and dead inside one call. What is
sticky is `AttackTarget`, which stands in for the law's **actor** order `ord+0x0c`
(`AI-PURSUE-040`), written by `R0009` out of `ord+0x20`.

So the real question is not whether `ord+0x20` is cleared but **what the clear causes**, and that is
in the arms rather than in the scorer:

- **Order 1, guard** (`AI-GRPGUARD-074`, read end to end): per member, `ord+0x20 != 0` gives the
  engage; else, away from the post or not idle, `ord+0x08 = 1` and `ord+0x0a = post`; else, at the
  post and idle, an AI owner takes `ord+0x08 = 0xb` (idle turn) and a **human participant's** unit
  `R0015`, with no order write.
- **Order 3, stand ground** (`AI-STAND-076`, read end to end): `ord+0x20 != 0` gives the engage;
  else an AI owner takes `ord+0x08 = 0xb`, and a human participant's unit `R0015`, no order
  write. No walk of any kind.
- **Nothing else ends a pursuit under these two orders.** `AI-PURSUE-040`: nothing inside either
  pursuit arm measures time, health or distance travelled. `AI-BREAK-041`: what ends one is the
  `actor+0x50` arm rewriting `ord+0x08` — and `AI-ORDER-010` states that *a group under order 1 or
  3 never evaluates `actor+0x50` at all*. So for both stances this build implements, **the group
  arm's own rewrite is the only break-off there is.**

That closes the question `0086` left open, and it puts `AI-BREAK-041` out of scope for a stronger
reason than "this tree has no per-actor machine": the law never runs that machine under either of
these orders.

## The consequence, and where it does not apply

The clear ends the engage exactly where the arm then writes `ord+0x08`, which is **every member
except a human participant's**. In this build the only human participant is `SelfSlot`, and
`stance()` derives stand ground from `owner == SelfSlot` alone, so *stand ground* and *human
participant's unit* name the same population here and part company only on a world this build
cannot construct. That is a seam, not a simplification, and the spec names it.

One caution the pin will not settle: `R0015` is unread. Whether it rewrites `ord+0x08` is
**open**. It does not block, because the choice it would decide — leave a stand-ground member's
victim alone — is what this build already does, so the open question costs a divergence only if the
answer is the surprising one.

## What the handed enumeration missed

Handed two sites in `decide`. Enumerating `AttackTarget`'s surface ourselves (one setter,
`orderAttack`; clearers at `combat.go:299`, `group.go:144`, `step.go:270`, `step.go:818`,
`step.go:1155`, `world.go:1061/1099/1123`) turned up two things the brief did not have.

1. **Clearing the victim is not enough.** `approach` writes a destination toward the victim through
   `walkTo` every tick. `engagementPass` runs before the move loop, so a member whose victim is
   dropped by a decision still holds last tick's destination at the victim's old cell, and
   `step.go:437` no longer calls `approach` to update or retract it — it walks there and stops.
   The law has no such residue: its arm **replaces** the pursuit order rather than erasing its
   target. So the release must drop the walk as well, and it is exactly `orderAttack`'s inverse.
2. **The two clearing sites collapse to one, and then a third reappears.** With a release on
   *this member scored nothing*, the `len(cands) == 0` early return is redundant — an empty list
   leaves every member unscored, which is the same case. But `AI-CANDBYTE-110` (published after
   `AI-SCORE-069`, and not in the brief) states the scorer tests only the **low byte** of the
   candidate count against the order-5 gate's dword, so a count that is a non-zero multiple of
   `0x100` scores nothing and clears everyone. Keeping an explicit narrowed count test restores the
   law's own two-site shape and costs one conversion. It is Medium and its reachability is
   unmeasured; it is carried as a limit (G2), not as behaviour anyone expects to see.

## Composition with `0097`

`0097` removes a unit **holding a destination and no victim** from the decider set. `0098` changes
what happens to a member that **is** scored and scores nothing, and its release is a no-op on a
member holding no victim. **The two populations are disjoint by construction**, so neither makes the
other unnecessary and neither weakens the other: `0097` protects a player's order from being
destroyed by an acquisition, `0098` ends an acquisition nothing was ending. They meet in one place —
`0097`'s FR-5 says a commanded member contributes no sight, so a group that could see only through
it has no candidate list, and under `0098` that group now **releases** rather than holds. That is
the intended composition, not a collision. Textually both edit `decide`'s member loop and will
conflict on merge; the resolution is `0097`'s skip outside `0098`'s release.

## What this build releases sooner than the law does

`AI-GROUPSEE-068` phase (b), verified at the pin: for a member whose owner is **not** a human
participant and whose `ord+0x58` is non-zero, the remembered attacker's **cell** is incremented into
the same visibility map, `ord+0x5a` counts up, and at `ord+0x5a > 0x14` — **20** — `ord+0x58` is
cleared. So an AI-owned member that has been struck keeps that cell inside the stamp for up to 20
group ticks. That reading holds, with one narrowing worth keeping: what is remembered is a **cell**,
not an actor, so an attacker that steps off it is not restored by this path — the memory prolongs a
chase only while the quarry stays put or the strike is repeated.

This build has neither field. **It therefore releases sooner than the law does**, and that
divergence points the same way as the defect's fix rather than against it: it will make the census
fall further than a faithful build would. It is a **named omission** — two per-member fields with
their own clock, and a story whose whole change is a release in `decide` should not acquire them by
side effect. `verification.md` must not claim a release the moment sight is lost; the law's release
is *at most 20 group ticks later* for a struck AI-owned member, and this build's is immediate.
