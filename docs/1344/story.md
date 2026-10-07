# 1344 — Troll refusal and inherited hints

## Result

No player-visible change on EN or RU. Both rows keep their behaviour: the k173
claims update DIV-1316's authority and Unknown columns and narrow DIV-1885's
Unknown; no behaviour changes for either. Neither program binary
changes, so no version moves.

## Authority

- `AI-394` (High for the enumeration; Medium for "only" and the composed rule):
  the static full search leaves its list empty on start equal to goal, on picker A
  answering 0 or the start cell, and, not excluded, on the extractor's 1000-step
  cap. Picker A scans rings 1 to `(D>>2)+3` around an unlabelled goal. Border
  probes alias: Unknown.
- `AI-395` (High; Medium for persistence): order 0xb never restarts pursuit; a
  refused order stays idle until a group decision reissues one (`AI-REISSUE-077`).
- `AI-384`, `AI-373`: no give-up counter in the routines read; the flag follows an
  empty list; a waypoint search that finds nothing retries and runs the full
  search once `mover+9` exceeds a third of the static count plus one.
- `MENU-096`, `TEXT-105`, `TEXT-106`, with `MENU-068`, `MENU-069`,
  `TEXT-HOVERSET-049`: counts and rules for inherited hints; no claim names an
  engine control as receiver.

## As built

- DIV-1316. The engine's far search already settles on the first labelled ring
  within `(D>>2)+3` of the ordered cell, the start cell included, and refuses on
  none (`settleRings`, `settleFor`). That is `AI-394`'s inside-map rule, read from
  the code and not newly pinned. The stalled-tick count, the 16-tick whole-map
  re-search, the human/AI split and the refusal of an out-of-reach settled cell
  stay; refusal and pursuit behaviour is owner-fixed as landed. The row's
  authority and Unknown columns carry `AI-394` and `AI-395`.
- Mission 121: `TestReleaseMission121BridgeTrollKeepsItsVictimWhenItsRouteIsRefused`
  is unchanged and passes on EN and RU: the pinned-pair troll is refused 5 ticks
  after it stops, idles on 294 of the next 294 ticks, and strikes within 300 ticks
  once the pair leaves. `AI-395` agrees with the idle order and with a refused AI
  attacker retrying at each group decision.
- DIV-1885. No inherited hint is bound: no claim names the engine control that
  receives a Sound Options, Game Options or `L06385` dialog hint, and the Sound
  Options pairs' final value is not stated. The row carries the k173 counts and a
  narrower evidence-needed column.

## Proof

- Mission 121 release test on EN and RU, unchanged expectations.
- Documents only: no Go file changes.

## Open debt

DIV-1316 and DIV-1885 stay OPEN. Evidence needed is in each row.
