# 1012 — spec (as-built)

## Subject

How many notice panels this build shows for a script message raised more than once in short
succession, and what ROM1 evidence that behaviour is measured against.

## Mechanism, as built (unchanged by this story)

`pkg/sim.World.scriptPass` (`pkg/sim/script.go`) evaluates every trigger once per script pass
(every 16 ticks, `scriptCycle`). For a trigger that is not one-shot (`Once == false`), the pass
clears the trigger's latch, re-evaluates its condition, and — if the condition holds — sets the
latch back to 1 and runs every instant in the trigger's slot list again, including a message
instant. This is unchanged by this story: **the simulation genuinely re-runs a repeating trigger's
message instant on every pass its condition holds**, which is the ROM1 send-side rate
`TRIG-MSG-023` describes.

`pkg/game.Announcer.Sample` (`pkg/game/announce.go`) is called once per world step, above the
simulation, and reports only the 0-to-1 transition of each trigger's latch (`announce.go:111`,
`now && !a.prev[latch]`). A repeating trigger whose latch is already 1 from the previous pass
reports nothing on the next pass, even though the simulation ran its instants again. The result: a
non-once trigger held true across two or more **consecutive** passes is reported as one notice for
the whole span it holds, not once per pass. This merge is `announce.go:32-38`'s own documented
concession (comment "THE ONE DIVERGENCE, stated because it is real").

Two **different** triggers (distinct map latches) raising the same message number are each
sampled independently and each produce their own rising edge; this build reports two notices,
matching the original. `TestTwoNodesRaisingOneNumberProduceTwoAnnouncements`
(`pkg/game/announcereport_test.go:267`) witnesses this and is unaffected by this row.

## What the cited claim establishes, and what it does not

`TRIG-MSG-023` is a **send-side** claim: it reads the instant's own arm, the shared static packet
it writes, and the broadcast decision (three routines read at instruction level, High). Its own
confidence line states plainly that "the instrument is those three routines, so it cannot see what
a recipient does with the packet" (Medium for "no simulation state"). It supports "the original
re-broadcasts once per pass a repeating trigger holds" — the SEND rate — and does not support "the
original shows one panel per broadcast" — a CLIENT rate, which `TRIG-MSG-023` never measures.

`DLG-LIFE-005` is the claim that answers the client-side question. It establishes, at High
confidence, that nothing ends the display but player input, nothing queues a second announcement,
and one arriving while any dialogue panel is open is discarded — not deferred, not shown as a
second panel. The recipient's gate is a single instruction, `L03312 TEST byte ptr
[EBP+0x3dc],0x8`, tested against the panel's own open bit.

Combined: for a repeating trigger whose condition holds across two consecutive passes (roughly 16
ticks apart, well under the time a player takes to read and dismiss a panel), the ORIGINAL's own
behaviour is: broadcast, open panel 1; broadcast again while panel 1 is still open, discard. The
visible result is one panel, not two. This build's merge (one notice for the whole held span) and
the original's discard-while-open rule produce the **same visible panel count** for this case. The
divergence row's premise that ROM1 shows "one panel each" for this case is not supported by the
claim it cites, and the claim that does answer it says the opposite.

## Where a residual divergence remains

The two mechanisms are not identical in every case, only in the common one. If a player dismisses
panel 1 and the SAME repeating trigger's condition is still true (or becomes true again) on a LATER
pass, the original broadcasts again with no panel open and shows a second panel; this build's
rising-edge memory (`a.prev[latch]` stays `true` for as long as the latch stays `true`) never
re-reports that trigger until its condition first drops to false and rises again. This is the
narrow case `DIV-011` is sharpened to describe below. It requires: a shipped, non-once,
message-raising trigger whose condition remains true, or returns true, after its own panel has
already been shown and dismissed once.

## Shipped-content measurement

Instrument: a standalone tool built against this worktree (`againrom/pkg/game`,
`againrom/pkg/mapload`), loading each of the 28 campaign maps
(`10 20 30 31 40 41 50 51 60 61 70 71 80 81 90 91 100 101 110 111 120 121 130 131 140 141 150 151`
— the same list `pipeline/check-milestone.sh` and `scripts/campaign-sweep.sh` use) the same way
`cmd/missionrun` does (`game.OpenArchives` → `game.LoadDefinitions` → `game.StartMission`), then
walking each map's compiled `*sim.Script` for triggers whose instant list includes a message
instant (`Op == 2`).

Counted per preserved root:

```
EN: 231 message-raising triggers, all 231 one-shot (Once == true), 0 repeating.
RU: 231 message-raising triggers, 230 one-shot, 1 repeating (mission 140, trigger 5, latch 5).
```

The one RU candidate is not inert (its checks are both supported opcodes) and carries two used
pairs (`nearest <= const`, `groupcount > const`) — a conditional trigger, not an unconditional
"always" one. A second, dynamic instrument — `cmd/missionrun -trace -census -ticks 2000` (unattended,
no waypoints, matching `scripts/campaign-sweep.sh`'s own methodology) — was run over all 28
missions on both roots; that trigger never fired during any of those runs, so its condition was
never observed to hold at all in an unattended drive, let alone across two consecutive passes.

Conclusion: the shipped population at risk of the residual divergence above is 0 of 231
message-raising triggers on EN, and at most 1 of 231 on RU — a candidate not confirmed to fire in
an unattended drive, and even if it does, the residual divergence requires the ADDITIONAL condition
that a player dismiss its panel and the trigger's condition hold true again afterward. This is not
"many" by `contract.md`'s decision rule, and the code is left unchanged.

## Cut list

Nothing is cut. No behaviour changes.

## Divergence disposition

`DIV-011` stays `OPEN`, re-cited to `DLG-LIFE-005` alongside `TRIG-MSG-023`, with its `Reason` cell
carrying the measured population above. `DIV-134` is returned unused: correcting an existing row's
citation and evidence is an edit, not a second divergence.
