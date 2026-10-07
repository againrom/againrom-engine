# 1004 — projectile presentation

## The result

Lightning and Prismatic Spray draw a jagged figure spanning the whole caster-to-target segment
from their first frame, regenerated on every tick, and neither travels. Fire Arrow and Fire Ball
leave a six-puff smoke trail behind them. Fire Ball's explosion begins where the projectile
arrives instead of at the release tick. A spell released by a weapon draws the same way a spell cast
from a book does, on both of the two arms a weapon-borne release has: a staff replacing its wielder's
attack, and a spell riding a fighter's blows. Every campaign mission on both install roots places 79
carriers of a weapon spell — 76 on the first arm and 3 on the second — and before this story the
second drew nothing at all while its burst still appeared at the target.

Owner reports this answers: Lightning «молнии до сих пор не отрисованы нормально» and «что насчет
спрайтов молний, когда мы их отрисуем?» (2026-08-15, 2026-08-16); Fire Ball «огненный шар
проигрывает в конце какую-то странную анимацию после того как его спрайт отыграл» (2026-08-15).

The observable result is in `builds/current/`: cast Lightning and cast Fire Ball. It moves no
number in `pipeline/check-milestone.sh`'s script-gap census, which is at 0 UNSUPPORTED for
missions 10 and 20 before and after.

## Claims

| Claim | What it fixes here |
|---|---|
| `MAGIC-BOLTGATE-069` | Exactly two picture ids draw a path: 34 Lightning and 36 Prismatic Spray. |
| `MAGIC-BOLTSHAPE-070` | The figure is a bounded random walk in a canonical horizontal frame, rotated onto the segment. Constants: `rand()%7` with a random sign into a value clamped to ±3.0, `rand()%50` scaled by 0.01 into an abscissa ending past 0.7, rejection at 0.15 of the length with a whole re-run, a smoothing pass with the factor −0.5. |
| `MAGIC-BOLTLIST-071` | The list is replaced whole on every driver tick, and spans the whole segment from the first. |
| `MAGIC-BOLTSTILL-072` | The object never moves; its 13 ticks are a countdown, and its arm writes the sheet frame alone. |
| `MAGIC-BOLTEND-074` | Nothing is created when the figure ends. |
| `ANIM-CAST-027` | The path arms ignore the object's own position and stamp the sheet at each point, offset by 8 pixels; picture 36 adds a per-point phase of `tag * 5`. The default arm walks the trail after the sprite pass. |
| `MAGIC-TRAIL-073` | The trail belongs to pictures 10 and 12: a queue of at most six PAST positions, oldest dropped one at a time, appended from the position saved before the move. |
| `ANIM-PROJ-026`, `REG-PROJ-086` | The trail art is `graphics\projectiles\smoke%d\sprites.16a` for 0 and 1, loaded outside `projectiles.reg`, which never names it. |
| `MAGIC-BURSTLIFE-034`, `MAGIC-DELIVER-035` | A burst is stationary and its sender chooses its lifetime; the original's own cast routine computes a flight time for the four `Delivery System == 2` spells. |
| `AI-RAND-058` | The generator the shape routine calls is MSVC's LCG with `RAND_MAX` 0x7fff. |

## UNKNOWN

- The parametric mapping from the walk's internal units to screen pixels is `MAGIC-BOLTSHAPE-070`'s
  own Medium, and which endpoint the list starts from is its Unknown. → `DIV-080`.
- The thirteen values of the frame ramp are not published, nor is the frame the trail stamps, nor
  what fills the caster's victim array for picture 36 (`MAGIC-BOLTLIST-071`'s Unknown). → `DIV-081`.
- This build's simulation applies a cast at the release tick and carries no flight time. → `DIV-082`.
- No claim states whether the original draws a weapon-borne release through the same projectile arm
  as a book cast, on either of its two arms, nor which record tag a weapon-borne release carries. The
  owner's directive is that the spell be drawn, so both arms draw it the way a book cast does.
  → `DIV-081`.

## Aspects that apply

Data (the two trail sheets), runtime state (the arm a cast observation reports), simulation (the
observation field alone, which is client-facing and neither hashed nor serialized), UI/HUD (the whole
figure), shipped content (the 79 placed carriers of a weapon spell), inventory/equipment (a
weapon-borne release is what an equipped staff draws), interactions with existing mechanics (the
burst's timing against a travelling cast object). The other five are N/A: nothing here reaches input,
AI, triggers, persistence or the campaign.

## Domains

**Client** (8), **Assets** (1) for the two sheets loaded by constructed path, and **Simulation**
for one client-facing field on the cast observation, which reaches no hashed state and no saved
byte.

## Out of scope

Area overlays and staged rows (`1003`), per-actor effect marks (`1002`), and the Prismatic Spray
victim set. The bolt's shape is client presentation and must not reach hashed simulation state; it
does not. The one change in `pkg/sim` is a field on the cast observation, which is a return value
rather than world state: it is absent from the serializer, the digest and the `World` struct, so
`formatVersion` does not move.

## Expected divergence rows

`DIV-080`, `DIV-081`, `DIV-082`.
