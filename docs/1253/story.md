# Hold Position during a running strike

## Intent and authority

A Hold Position order given while a strike runs keeps the strike and replaces
the order waiting behind it. The pinned authority is k106.

`AI-351` (High): both Stand Ground setters store the pending order 0 for every
member and store no progress. `AI-352` (High for the stores, Medium for the next
ticks): the setter does not cancel progress in flight. `AI-354` (High for the
body): a strike or cast body applies from the active fields, and clearing the
pending order alone does not cancel it. `AI-350` (High): the executor runs no arm
for pending 0, so the actor idles once the strike ends. `AI-353` (Medium): the
next group evaluation reissues the attack for a victim in reach.
`AI-STAND-076` and `AI-REACH-072` fix what the evaluation assigns.

Outside Hold the story-1249 rules stand: release, pickup, manual cast and scroll
wait beside the strike, and an explicit Retreat replaces a pending order.

## As-built behaviour

Hold ordered during a strike:

- The loaded strike keeps its victim and phase and lands on that victim. The
  engine's earlier retention (`standDown`, DIV-1582) is unchanged.
- A pending release, a pending pickup and a manual cast row not yet installed are
  replaced (`holdReplacesPendingRow`, DIV-1663). The replaced row never runs and
  is absent from the saved World and from SAV.
- An installed cast row and a pickup completion record are kept (DIV-1664).
- A waiting scroll is cancelled by the group command arm, as for every group
  order (DIV-1665).
- After the strike ends the actor stands. The group order 3 evaluation attacks
  what stands in reach, as before.
- A second order after Hold queues behind the strike as it would without Hold.
  An explicit Retreat after Hold replaces any pending order and keeps the strike
  body, as it does without Hold.

The driver already dropped its own pickup intent on any group order. After this
change the canonical pending pickup is gone too, so LOAD cannot restore a pickup
the player replaced.

## Proof

`TestHoldStrike*` in `pkg/sim` cover a queued actor cast and cell cast, a queued
pickup and a release marker replaced by Hold, with the loaded blow landing once,
cold round-trip equality and a no-Hold control that runs the cast. They also
cover Hold with no pending row, Hold then a second order, and Retreat after Hold.

`TestReleaseHoldStrikeReplacesPickupSAVAppColdLoad` (EN and RU) orders the
attack, queues a pickup, presses the command panel's Hold cell, saves through F2
and loads cold through the main menu. The loaded world equals the live one for
240 ticks, the sack stays and the purse is unchanged. The no-Hold control is
`TestReleaseMidStrikePickupCurrentSAVAppColdLoad`, where the pickup transfers.

## Open debt

DIV-1582: the common-tail replacement of the active victim is not modeled for
Hold, and native reachability during a running strike is Unknown. DIV-1663: no
claim says whether the pickup transfer's two native passes can be separated by a
Hold. DIV-1664: the consumer of the completion word the setter zeroes is not
established. DIV-1665: native survival of a waiting scroll pointer after Hold is
not established. DIV-1666 is unused.
