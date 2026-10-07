# A saved turn continues

A loaded actor finishes its admitted centered turn instead of remaining still.
Current facing advances by the saved RotationSpeed on each body update, with
the shorter-arc wrap and target clamp from MOVE-TURN-044. The complete active
dword matters; the estimate byte is recomputed from the pre-step arc and the
counter byte wraps. The next movement step waits until after turn completion.

The knowledge pin advances from k3 to k4. MOVE-TURN-031's route-destruction
clause is partially retracted there. Native turns therefore retain their
calculated route too. Their existing countdown and drawing interpolation stay
as a compatibility arm; this story does not replace the whole native turn clock.

Original LOAD preserves the mover bytes. The known local transition starts on
an executed simulation update. A clean crossing can hand over to a pending
turn; its dynamic-list arrival cleanup still occurs before that turn. On
completion, only an ordinary saved move order (kind 1, progress 0, no repair)
hands its remaining route to the existing bounded native continuation. A
standing or attack-facing turn does not start walking a stale carried route.
Original route planning and all-order scheduling remain outside this slice.

The existing motion block persists the state; no byte layout is added. Old
native saves carrying either exact turn-only refusal are reconsidered on their
next body update after admission validation. LOAD itself keeps their hash.
Other issues and zero RotationSpeed remain explicit refusals. Stone Curse and
off-map presence withhold the body update; a replacement command still owns
priority. Manual Teleport cancels the saved turn before casting.

SAV-TURNLOAD-822 proves raw transfer only inside the local serializer. Complete
post-LOAD hooks, first-dispatch chronology, speed callbacks and full original
world SAVE interoperability remain Unknown. DIV-802/950/952 narrow accordingly;
DIV-1019 records scheduling, native-clock and existing-form compatibility debt.
The stricter older readers may refuse a new issue-free pending turn in form 86;
this change preserves reading old saves, not downgrading to an older executable.

## Proof

Independent table cases cover fresh/active short arcs, all flag bytes, the
128 tie, both wrap directions, a terminal clamp and counter wrap. A six-call
turn is compared with a cold native reload before each call and retains both
routes byte for byte. Movement starts on the following body update. Legacy
turn refusals resume, unrelated callback debt stays deferred, zero rate stays
named, standing turns do not adopt a route, and manual Teleport still wins.

Five historical native-fixture cuts keep their frozen envelopes and exact LOAD
checks. Their 20 successor digests now include the admitted turns. Independent
literal checks assert the completed motion; comparison with main proves every
other pre-action-clock byte unchanged. The four document-bearing cuts differ
only in the expected actors' mover bytes 0, 0x9d and 0xa4 on all 20 successors.
The sole review found a wall closing a carried route during the turn. Handoff
now rechecks the complete footprint, retaining the goal but dropping a closed
route. The regression compares 24 post-reload ticks and reloads each successor.

The registered EN/RU witness uses preserved `2026-08-02/game0009.sav`, SHA256
`60267c82072c77446ab9b34913318e89eab8f70e49f3510ae64aaaf423819bd6`.
Actor 3 turns 28 -> 48 -> 68 -> 88 -> 96. Ordinary menu SAVE and LOAD at 68
preserve the world hash; subsequent game updates match an uninterrupted second
frontend, and the actor then moves along its continued order. The focused EN
run passes. Final paired release, milestone-2 compatibility and exact binary
witness results belong to the landing record.
