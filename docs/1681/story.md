# A guard placed by a map's script keeps the distance the map gives it

## Result

When a map's script orders a group to guard, the group now takes the guard distance the script gives, as in
the original. In the first game a distance of 1 to 255 replaces the old minimum of eight cells; a script with no
distance keeps eight. In the second game the group guards as the second game's own guards do, and a narrow group
with no distance guards twenty cells around it. A player's Guard order and the guards a mission starts with are
unchanged.

## Authority

- Owner: «обе игры как в оригинале, DIV-4905 пересмотреть»
  (both games as the original; revisit DIV-4905).
- `AI-792` (High / Medium): the first game's script Guard calls the guard setter with the low byte of Par1.
- `AI-793` (High / Medium): that setter gives max(widest sum, n) for a nonzero n and max(widest sum, 8)
  for 0.
- `R2-ENGINE-1252` (High / Medium): the second game's script Guard calls the mode-18 setter with record byte +0xc;
  the argument is read at Medium and is built as the claim states.
- `R2-ENGINE-1253` (High / Medium / Unknown): that setter writes 20 below the floor 8 for r = 0, max(sum, r) else.
- `DIV-4905` rewritten, `DIV-425` restated for both games and scrolls, `DIV-4977` added.

## Rules

1. A first-game script Guard with Par1 low byte 0 sets the guard radius to max(widest sum, 8).
2. A first-game script Guard with Par1 low byte 1 to 255 sets it to max(widest sum, that byte), with no floor.
3. A second-game script Guard sets programme 18 and radius max(sum, n) for n > 0; for n = 0, the sum or 20 below 8.
4. The setter's stores follow in both games: +0x38 equals +0x2d, +0x39 is cleared, the geometry is recomputed.
5. A player's Guard order and the mission-start group pass keep their rows and radii.

## Code

| kind | its one builder | change |
|---|---|---|
| group guard words | `setGroupWords` row `rowStanceGuard` | a script Guard takes it with its range value; `rowScriptGuard` removed |
| group route profile | `groupRouteProfile.spreadRadius` | a nonzero range gives max(sum, range); the floor rule is unchanged |
| scenario group commands | `cmdSavedGroupOrder` | passes the low byte of Par1 as the range |

## Proof

| test | rules |
|---|---|
| `TestScriptGuardTakesItsRange` | 1, 2, 3, 4 |
| `TestGroupWordsRecord` (second-game script Guard lines refreshed: programme 18) | 3, 5 |
| `TestGroupRouteProfilePerGame`, `TestSubCommandOneReFreezesTheNoticeBase` | 5 |

Shipped script Guards: first game mission 80 node 9, Par1 13, no trigger (EN and RU); second game mission 53
node 28, Par1 6, trigger «Druid has been returned» (EN and RU). Evidence: the review report.

Release tests at the release: `TestReleaseSecondGameSupportCensus` and
`TestReleaseSecondGuardStruckWalksAwayInsideTheStampWindow` on ROM2 EN/RU; the scenario chain of both games.

## Open debt

- The second game's unclassified mode-18 callers and programme-1 callers' arguments stay Unknown (`DIV-4905`).
- No original run has recorded +0x2d after either shipped script Guard.
