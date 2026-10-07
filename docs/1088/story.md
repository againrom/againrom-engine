# Victory returns through the world map

## Intent

Victory must remove the completed mission scroll and return the party to town
automatically when the campaign destination is town. Continue must not complete
the mission. Direct campaign successors, including M10 -> M20, stay direct.

## Behaviour

The completion boundary updates campaign, purse and carried party once. It then
replaces the departed map screen's cached offers, starts travel from the finished
mission's registry point to MapObject 0, and opens the square on arrival. No
mission or town scroll is drawn or actionable away from home. A map click can
skip the remaining return animation; it cannot reopen the finished mission.

TOWN-121 supplies destination-zero arrival and TOWN-122 supplies the home-only
scroll gate. SAV-CAMPAIGN-077..080/086 separate completion, offers, selection and
marker history. Owner direction supplies the automatic post-Victory return
trigger and finished-mission origin; their exact original producer remains
Unknown (DIV-137). Registry-less diagnostic fixtures fall back to the square.

Native saves made during return retain the pending mission and reveal progress,
without a live mission world or a second completion/payment. Older native saves
and ordinary town imports keep their existing square-entry behaviour. If an
imported away relation is later opened on the map, it returns home through the
same route; this entry policy is also authored under DIV-137. Escape completes
the homeward presentation through the same arrival writer.
Defeat and a refused outgoing mission opener also return home, retaining the
accepted offers and purse; neither path invokes winning completion or payment.

## Proof and debt

Focused tests cover immediate/delayed Victory, repeated completion refusal,
route arrival, Escape, native continuation, malformed restore atomicity and
imported first-point persistence. The pure compositor has an independently
constructed pixel oracle for absent scrolls and the home destination cross.
The exact pre-change synthetic `.ags` envelope remains a decode-only fixture;
the additive field changes neither envelope version nor simulation version.

`TestReleaseVictory1088AppReturnsHomeAndResumesPendingTravel` passes EN/RU,
four cases each: immediate/continued Victory with and without F2/F3. Actual App
input travels from `(386,166)` to `(215,234)` over 227 route coordinates, restores
prefix 8, arrives without another click, keeps the carried party and pays once.
M30 disappears; the unrelated M31 offer survives. Completion is a controlled
synthetic victory, not an ordinary M30 win. This story does not claim an ordinary
M20 playthrough or repair `0163-mission-to-town`'s unrelated initial notice wait.

The sole fresh adversarial pass returned two non-winning exit regressions:
defeat and a refused outgoing opener could strand the away map behind hidden
scrolls. The single correction routes both home without completion/economy
changes. The durable report is seat `pipeline/reviews/1088-adversarial-review.md`;
its independent probes are rerun as correction verification, not another review.
All independent probes pass on correction `365514dd`, including installed EN/RU
defeat/retry and synthetic missing-ALM recovery. Final merge gates and the exact
playable build are recorded in the seat's implementation landing journal.
