# 0093 — analysis

Intensity: **spec-first / static**. Terrain: **brownfield** in `pkg/sim` (the script runtime
exists and its behaviour must not move) and in `cmd/missionrun` (an existing tool gains a flag);
**greenfield** for the trace types themselves.

## What we did not know

The first milestone — one campaign mission from its start to a win — has been red since `1c5a7a1`.
Every account of *why* has come from outside the script: a unit intercepted, a corpse on a route,
a sight stamp. Two of those were excluded by measurement and one was retracted outright. What no
account rested on was the script's own report, because **nothing in the tree can produce one**.

The runtime decides a mission through two counters and a reporter. Neither says which authored arm
moved a counter, and the four accessors a consumer has — `ScriptCounters`, `ScriptRegister`,
`ScriptLatched`, `Outcome` — answer *what the state is*, never *what happened*. So a mission that
ends the wrong way ends it invisibly, and the question "which arm fired" had no instrument.

## What we looked at

**The runtime, read before designing against it.** `scriptPass`, `triggerHolds`, `runInstant` and
`scriptReport` behave exactly as the pass/latch/AND/exactly-one description says. One thing that
description omits, and it is the thing that mattered:

> **A loss need not come from a trigger at all.** `runCheck`'s `ScriptCheckVIP` arm increments the
> lose counter itself, from the check loop, before any trigger is walked. A design that recorded
> only "which trigger fired, and which instants it ran" would have observed the tenth mission
> being lost with **nothing having fired**.

This is `MISSION-VIP-004`'s shape stated as behaviour, and it is why the trace has three event
kinds rather than one.

**The second silent path.** A check whose unit reference does not resolve writes no register — and
its readers are **not** inert, because inertness is derived from unimplemented *opcodes* alone. So
a live trigger can compare a register that no measurement ever wrote against an authored value. An
`alive` check in that state leaves its register at 0, which reads as *dead*. Nothing anywhere said
so, and it is the failure the trace's silence record exists to make visible.

**Whether a new `cmd/` tool was wanted.** `cmd/missionrun` already starts a mission as the
front-end does, issues only ordinary orders, stops when the world decides and prints the outcome
with its tick. A second tool would have had to repeat all of that to say one more sentence about
the same run. It hosts the readout.

**Where the observer could live.** Three shapes were considered and two rejected:

- a **hook stored on the world** — refused: `nostate_test.go` pins `World`'s field set for exactly
  this reason, and a callback field is state beside the digest;
- **reconstruction from outside**, by snapshotting latches and registers around each pass — it
  *can* name the trigger (a non-inert, non-skipped trigger fired iff its latch reads 1 after the
  pass), but the register values it recovers are the values **after** the whole pass, and an
  earlier trigger's `setvar` moves them. An instrument that is wrong exactly when the script is
  interesting is not an instrument;
- a **value threaded through the step and returned** — no field, no callback, exact values.

## What the instrument then measured

Mission 10, both roots, the milestone's own two waypoints:

```
tick 262  check 12 vip(u21) COUNTED A LOSS: its unit is not alive (-1 hp)
tick 262  counters won=0 lost=1
tick 271  REPORT won=0 lost=1 -> lost
```

No trigger fires. The deciding arm is a **check**, and the loss is the map's own escort objective
firing correctly on a unit that should not have died. `-1 hp` is *killed outright*, not downed.

One further probe, with the same drive minus u21's own waypoint: **u21 never dies and the mission
is still undecided after 1351 ticks.** So the death follows from walking u21, not from the mission
running. What kills it is a combat/AI question and is outside this story.

## Unresolved

- Which blow fells u21, and between which two ticks (the passes bracket it at 247..262).
- Whether the original's tenth mission expects the escorted unit to survive that route unaided.
  It is an escort objective either way; whether our walk is the walk the mission intends is a
  question for the story that reads it.
- Check op 14 and instant ops 2, 6, 20 and 28 are authored by `10.alm` and unimplemented here.
  Trigger 4 is inert because of op 14. None of them is on the losing path, but the win chain has
  not been shown to be free of them.
