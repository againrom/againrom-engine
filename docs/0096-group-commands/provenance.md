# Provenance — 0096, the mission script's group command

Research pin: submodule `research/` at `4920d6d`. Every row below was read through
`go run ./tools/claim <ID>` at that pin, never out of a ledger by hand.

## What the story rests on

| Claim | Confidence | What it gives this story |
|---|---|---|
| `TRIG-GROUP-005` | High | FR-1. Action opcode 6 is a **second dispatch on its own first parameter**, ten implemented cases (`1 2 3 4 5 10 11 14 15 17`), and a value outside them returns having done nothing. The shipped catalogue declares eleven signatures, so literal `18` is authorable and inert. That is why the gap report is per sub-command and not per opcode: one opcode number covers ten behaviours, of which this build runs five. |
| `AI-GROUPCMD-020` | High for the census and for each order byte | FR-2 and the whole census. 135 nodes over 20 campaign maps, 0 over the ten loose ones, by `Par0`: `1`×1, `2`×6, `3`×24, `4`×29, `5`×40, `10`×5, `11`×6, `14`×14, `15`×10. What each writes to `grpAI+0x20`: `2` → 2 **plus `grpAI+0x0a = (Y<<8)|X`**; `3` → 3; `4` and `5` through the two setters → 4 and 5; `1` delegates to the guard setter; and `10`, `11`, `14`, `15` leave it at **0**. So a shipped map's group order is whatever the last command left, not the load-time 1 or 3. |
| `AI-ORDER-010` | High for the table and the arm targets | FR-6 through FR-9, the dispatch. Eight live values on `grpAI+0x20`, a 6-entry PE-read table plus two out-of-table arms, and **which scorer each arm runs**: order 2 and the guard arm both reach `R0177`, order 3 reaches `R0153`. That pairing is what makes Swarm the third corner of the (clip, scorer) square this build has two corners of. |
| `AI-SWARM-022` | High, one routine read end to end | FR-7. Order 2 is *walk to the commanded cell and fight what you see*: per member, a target scored → engage; else current cell against the commanded one **and** idle, not both → an ordinary move order **to that cell**; both → the idle-turn arm or the heal. **"No formation, no spread, no per-member offset"** — the commanded cell reaches every member unchanged, which is the opposite of what the two setters do. |
| `AI-MOVE-023` | High, one routine read end to end | FR-8 and the scope of order 4. `R0154` **takes a second argument and never reads it** — the destinations reach the members from the setter. Its per-member body is arrival maintenance: arrived and idle and the latch clear → reach becomes the stop distance and the latch is set; latch set → the standing acquisition rule; else, with the group speed term non-zero, the move is re-issued. |
| `AI-SWARM2-024` | High for the body and the branch; its **Medium** on the fallback is **superseded** by the table below | FR-14. Order 5 is order 2 **without** the move-to-the-cell branch, and on `AImanager+0xbb4` zero it tail-calls order 4's arm with both arguments forwarded. |
| `AI-CMD-032` | High for the table and each arm's call | FR-8's reuse. Player order opcode `0x16` dispatches to `R0108(grp, col, row)` and `0x1a` reaches `R0157(grp, col, row)` — **the same two routines the script's `Par0 = 4` and `Par0 = 5` enter**, `callto:` 3/2 and 2/2. So the script's move is not a second distribution; it is this tree's existing one. |
| `MOVE-GATE-035` | High | That both setters run the identical body: the formation flag, the per-player mode gate, the Chebyshev spread gate against the AI manager's compile-time 2, the fork of the per-member loop and the gated rate store. The two differ in the order byte alone. |
| `MOVE-FORM-036`, `AI-SPREAD-038`, `AI-FORM-037` | High | Cited only to record that the distribution this story reuses is already built to them (0059), and that nothing here changes it. |
| `AI-CMD-033` | High for the allocation chain | DD-3. Every **player** order allocates a brand-new group at order 0 which the setter then moves; a **scenario** command does not. That is why this story leaves the placed group's order byte alone on the player path. |
| `AI-RADFREEZE-075` | High for both reader sets | FR-5. The guard setter's six callers are map load, the player's guard command and the script's `Par0 = 1`, and it is the only reader of the computed geometry. So `Par0 = 1` re-freezes the notice base, which is the third of the three freeze moments 0095 named as absent. |
| `AI-STAND-076` | High for the behaviour | That order 3's arm has no clip and no walk of any kind, so `Par0 = 3` needs no arm work — only a writer. |
| `AI-ACQUIRE-002` | High | What order 4's arm calls on an arrived member: everything visible, filtered to enemies, nearest first with a turn tiebreak, **discarded unless within reach**. The reach cap is the half this build can honour; DD-2 says which scorer stands in for it. |
| `AI-PATROL-017`, `AI-PATROL-018`, `AI-STATE-011`, `AI-ORDER-039` | High | The fence. Patrol writes `actor+0x50 = 0xa` and builds a waypoint ring on the **actor's** order record; the per-actor machine has 27 state arms and a 15-slot order switch, and this tree has neither. `AI-PATROL-017` also types the sub-command's parameters — `Par1 = X`, `Par2 = Y`, `Par9` a `Target_Group` — which the binder's encounter-order packing turns into `Args[1]`, `Args[2]` and the carried group. |
| `MISSION-M10-009` | High, one map read end to end | That mission 10's always-trigger starts **two patrols**, which is why the milestone's outcome is predicted unchanged. Measured independently from the compiled records. |
| `AI-ROAM-025` | High for the arm and for the corpus 0 | The negative keeping `Par0 = 17` out: 0 nodes over 38 maps. |

| `TRIG-GRPLIST-016` | High | FR-21. `RemoveMember`'s complete caller set is 4 sites / 4 owners / 0 orphan, and one of the four is **the change-of-owner routine `R0064`** — so an actor handed to another player *leaves its group* in the law as well. That is what makes 0095's `(owner, group)` keying faithful across a hand-over rather than an artefact of it, and it is why the pair a decision is taken for can name no record at all. |
| `AI-CANDCOUNT-105` | High | What `AImanager+0xbb4` **is**: the element count of the candidate collection at `+0xba8`, `list + 0xc`. `RemoveAll`'s own body fixes the layout, the same field is read through two different bases fourteen instructions apart inside one routine, and the owner is a cited constructor chain rather than a displacement guess. It closes this story's one open item, and it closes it as a *count* rather than as the assigned mode flag the alternative reading needed. |
| `AI-SWARM2GATE-107` | High for the mechanism; **not establishable from the image** how often the branch is taken in play | FR-14 and D-4, and it **overturns the bound the earlier draft carried**. The gate is read on the instruction after the builder returns, and the builder empties the collection *before* its own empty-group early exit, so nothing stale reaches it. Empty ⇒ tail-call to order 4's arm. The count includes **corpses**, so a group seeing only a body runs order 5's own body — AC-17. And the two zero-target cases differ *by a walk*: no candidate walks, vetoed candidates do not, because the veto branches are the idle turn and the heal and neither moves anyone — AC-18. |
| `AI-CMDSET45-108` | High; a **sharpening**, not a discovery | DD-9. The two setters are `0x2f0`-byte blocks `0x2f0` apart and, with the fourteen shared `rel32` targets folded, **exactly one byte differs** — the order immediate, `04` against `05`. EXP-0094 had already read both end to end; what was open was whether they agree *everywhere else*. They do, so lifting one body for both callers is fidelity rather than convenience, and every per-member state an order-4 group holds an order-5 group holds identically — which is what makes the fallback coherent instead of degenerate. |
| `AI-CANDBYTE-110` | **Medium** — the asymmetry is cited, its reachability is not measured | D-7, a **G2 limit**. The gate tests the candidate count as a dword while both scorers test only its low byte, and a group's own member count is read the same way, so at exactly `0x100` visible candidates the law enters order 5's body and scores nothing. **This build narrows no count to a byte** — not in the gap report, not in the group record, not in this story's two new fields — so the limit is recorded and not reproduced. The two bytes that *are* narrowed here are neither of them counts: the order itself, and 0095's notice base. |

## What is ours by choice

- **The gap report is keyed by `(opcode, sub-command)`.** `TRIG-GROUP-005` establishes that opcode
  6 is a second dispatch; nothing says a consumer must report it that way. Reporting per opcode
  would let five implemented arms silence the census of the five that are not, which is the
  loudness rule inverted.
- **Order 4's arm scores arrived members with the reach-vetoing scorer** rather than reproducing
  `AI-ACQUIRE-002`'s own ordering. The two agree on the reach cap and on nothing else. DD-2.
- **"Arrived and idle" is `!HasTarget && Transit == 0`.** The law tests the order's own cell field
  and a sub-cell centre predicate; this tree clears the target on arrival and carries a crossing
  counter, so the two conditions are already there under different names.
- **The load-time order is written into the record rather than derived at each decision.** The rule
  itself is unchanged (`AI-AUTHOR-015`, already built as 0086's stance): slot 0 stands its ground,
  every other slot guards.
- **Why D-1 fences the four order-0 sub-commands out rather than writing the byte.** The half this
  tree could build is the write; the half it could not is the arm that receives it. Building only
  the first would take a commanded group from whatever it was doing to nothing at all — so a
  command that today does nothing would become one that *removes* behaviour, which is a worse
  divergence than the one it fixes. The same argument decides FR-21 the other way: there the zero
  is not authored by a command, it is what the law's own zeroed allocation carries, so honouring it
  is fidelity rather than a choice.

## Open — named, not filled

- ~~**What fills `AImanager+0xbb4`.**~~ **Closed** by `AI-CANDCOUNT-105` and `AI-SWARM2GATE-107`.
  What it leaves open is a fact about *play* rather than about the image — how often a group under
  order 5 sees nothing — which no reading of the binary can settle and which this build's own
  measurement on shipped maps is the instrument for.
- **Whether `0x100` simultaneously visible candidates is reachable on any shipped map**
  (`AI-CANDBYTE-110`, Medium). It decides whether D-7's limit is theoretical or live. A census, not
  a further reading, and not this story's to run.
- **Whether the four order-0 sub-commands should write the order byte** with no arm to receive it.
  `AI-GROUPCMD-020` is High that they write it; this build does not, and D-2 says why. Closing it
  needs the per-actor state machine, not a further reading.

## Removed

Nothing. The handed scope named `Move`, `Swarm` and `Swarm 2`; reading the rows added `Guard` and
`Stand Ground` to the same dispatch — both already have arms here, both are one writer each, and
leaving them out would have left 25 of the 135 nodes running by accident of the load-time rule and
not because the map asked.
