# Analysis — what a mission script does to a group, and what this tree can carry of it

## What was not known going in

**Where the group order lives** — this build has none: `engage.go` derives a stance from the owner's
roster slot at every decision. **What the opcode-6 sub-commands do**, past the byte each writes. And
**whether running them moves anything on shipped content**, a measurement rather than a claim.

## The census reproduces, on both roots

Every `.alm` in each install, compiled through `pkg/mapload`, instant opcode 6 counted by its first
plain parameter:

    total op-6 nodes: 135
      Par0=1    1 node   over 1 map      Par0=10   5 nodes  over 4 maps
      Par0=2    6 nodes  over 4 maps     Par0=11   6 nodes  over 5 maps
      Par0=3   24 nodes  over 6 maps     Par0=14  14 nodes  over 8 maps
      Par0=4   29 nodes  over 10 maps    Par0=15  10 nodes  over 4 maps
      Par0=5   40 nodes  over 9 maps

EN and RU are identical, and it agrees with `AI-GROUPCMD-020` on all nine figures — which is what
makes the binder's packing safe to build on: `Par0` arrives at `Args[0]` and, where the catalogue
declares a coordinate pair, `Par1`/`Par2` at `Args[1]`/`Args[2]`. Measured on the `Par0 = 14` nodes
(`AI-PATROL-017` types them `X`/`Y`) and on `Par0 = 4`, whose cells are all in-bounds.

## Mission 10 cannot move, and not for the expected reason

The tick-6 trigger's two instant-6 nodes, which this build skips, were the reason to expect movement
here. **Both are `Par0 = 14`, Patrol** —

    scenario/10.alm instant#20 op6 args=[14 53 19 ...] group=18
    scenario/10.alm instant#21 op6 args=[14 54 13 ...] group=17

— which is `MISSION-M10-009`'s "trigger 0 ... starts two patrols" and `AI-PATROL-017`'s `scn:10 x2`,
confirmed from the compiled records on both roots. Patrol is out of scope.

It authors four other op-6 nodes — `Par0 = 11`, two `Par0 = 15`, and one `Par0 = 4` (`Move` group 2
to `(66,16)`, the escortee) — and **no trigger's action list names any of them**: ids 13, 14, 16
and 20 are authored and unreferenced, and only 22 and 23 are fired, by trigger 0. *(Reason
corrected 2026-08-07: this first read "sits on trigger 4, which `Script.Inert` marks", which
confused a compiled instant index with a node id. The conclusion held; the reason did not.)*

So on the milestone's own map, **this story implements zero of the group commands the mission can
reach**. Baseline, both roots, with the test's own two waypoints:

    waypoint 1  u21 -> (56,21) r3 : STOPPED BY THE WORLD DECIDING, short of (43,46), after 272 ticks
    outcome lost at tick 272                            (EN and RU, identical)

The prediction is therefore **`lost at 272`, unchanged, both roots** — a result, not an excuse: the
milestone's obstacle is elsewhere, and ten maps carry a `Par0` in `1..5` on a trigger that fires on
its first evaluation pass, `scn:151` and `scn:110` among them.

## The arms, enumerated — four things the handed list did not carry

**1. The scorer moves between the arms.** Orders 2 and 5 run the *guard* scorer, order 3 the
reach-vetoing one (`AI-ORDER-010`, `AI-SWARM2-024`, `AI-GRPGUARD-074`). So the arms are two
independent binary choices — **clip to the notice circle** and **which scorer** — of which this
build had two corners. Swarm and Swarm 2 are the third: no clip, guard scorer.

**2. Order 4's *setter* is the behaviour; its *arm* is maintenance.** The arm never reads the cell
handed to it; destinations arrive through the setter, which is the **same routine the player's own
move command runs** and which `pkg/sim/group.go` already implements (`AI-MOVE-023`, `AI-CMD-032`,
`MOVE-GATE-035`).

**3. `Par0 = 1` re-freezes the notice radius** (`AI-RADFREEZE-075`) — the third of the three freeze
moments 0095's D-1 calls absent. One shipped node.

**4. A `(owner, group)` pair can name no record at all** — found by reading 0095's *landed* code
rather than the Phase-3 spec this story was planned against, which grew an FR-9 later. The
hand-over arms rewrite an owner inside a tick, so a decision can be taken for a pair the constructor
never froze. `TRIG-GRPLIST-016` shows the law does the same — the change-of-owner routine is one of
`RemoveMember`'s four callers — and a fresh group object is zeroed, so **that pair's order is 0**.
This build has no order-0 arm, so the group takes no decision: D-1 by a second road, live on
mission 10 (triggers 2 and 3 hand group 2 over and back). `spec.md` FR-21 carries it.

## The Patrol / Follow fence survived, and the reason improved

The fence puts `10`, `11`, `14` and `15` out because they leave the group order at 0, the arm
handing each member to a per-actor state machine this tree lacks. That reading holds
(`AI-GROUPCMD-020`, `AI-PATROL-018`, `AI-STATE-011`, `AI-ORDER-039`), and it gains a positive reason
rather than an absence: the half this tree *could* build is the write, and building only that would
take mission 10's groups 17 and 18 from guarding to standing still. `provenance.md` has the
argument; the fence picks the smaller of two divergences.

## What is open

**`AImanager+0xbb4`.** `AI-SWARM2-024` grades Medium on whether order 5's fallback to order 4 is
taken in play; nothing establishes what fills that field, and forty nodes ride on it. This story
took the body, bounding the divergence at *"the two readings agree wherever the group has no
candidate"*. A research request, not a hold.

> **Appended 2026-08-07, after EXP-0126 answered the request. Not substituted — a dated record of
> what was known when the spec was written.** The field is the candidate collection's element count,
> and the bound above is wrong: *no candidate* walks, through order 4's arm; *vetoed candidates*
> stay in order 5's own body, which has no walk. The conclusion survives for a different reason —
> order 4's arm scores nothing against an empty list here. See `spec.md` FR-14, AC-17, AC-18, D-4.

**The player's group move does not write a group order**, deliberately: `AI-CMD-033` allocates a
*fresh* group at order 0 which the setter moves to 4, leaving the placed group's byte alone, and
this tree cannot allocate one. Symmetry with finding 4 — a fresh group is order 0 on both roads;
the player one is unmodelled because there is no allocation to model, the hand-over one modelled
because the pair genuinely stops naming a record.

## How this composes with 0098

`0098` rewrites the group-assigned target every evaluation and **clears it when nothing scores**
(`AI-SCORE-069`). Its branch and this story's Swarm 2 gate are neighbours, so a clean merge is not
a correct one.

They do not conflict: they sit on **opposite sides of one test**. The gate fires when there is
nothing to score *at all*, before any member is scored; `0098` acts when candidates exist and a
member scores none. That is EXP-0126's two-zero-target split — *no candidate* against *vetoed
candidates* — so `0098` owns the second, FR-14's body, not the gate. Watch FR-16 against it: FR-16
keeps a member's own **destination**, `AI-SCORE-069` clears the **group's assignment**. Different
fields, contradictory-looking English.
