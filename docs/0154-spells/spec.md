# 0154 — two spells complete: specification

**Fire Arrow and Heal, taken end to end: the cast the player can see, a popup
that states what a spell does before he spends the mana, and a toggle that puts
one spell on repeat with a rotating dashed border on its icon.** This contract is
self-contained.

Story 0127 shipped the first spell and cut delivery, every non-damage arm, area
shape, resistance and learning. This story lifts exactly two of those cuts — the
restorative arm and the picture a cast puts on the map — and leaves the rest cut.

## Functional requirements

### The restorative arm

**FR-1** A spell row is `Restorative` when its damage pair is positive and its id
is the heal id, and that flag is set by the table loader and by nothing else. It
is the exact complement of the id-named exclusion `Damaging` already carries:
Heal is refused from the damage arm because its roll is healing, so the same
comparison that refuses it there admits it here. Drain Life stays excluded from
both — it moves its roll from the victim to the caster and this build has no arm
for that. `Restorative` and `Damaging` are never both true of one row.

**FR-2** A cast of a `Restorative` row raises the target's health instead of
lowering it, and its refusals differ from a damage cast's in exactly three
places:

1. the target may be the caster himself, where a damage cast refuses self;
2. the target must **not** be hostile to the caster — a heal aimed across a
   hostile relation is refused whole, no mana spent and nothing rolled;
3. the target must be alive; a downed or dead body is refused.

Every other clause of the cast is unchanged and shared: the caster must be held,
alive, a mage and know the spell; the row must exist and target a unit; the
target must be within the row's `MaxRange` by the same Chebyshev distance; the
cost is subtracted once and only if it is affordable; the school is trained by
the same award.

**FR-3** The amount restored is the same arithmetic a damage cast rolls —
`base + U[0, spread]` off the row's own damage columns at the caster's own power
— and the result is **clamped at the target's maximum health**, which is the one
thing a damage cast does not do. Health never rises above the maximum the entity
was built with, and a target already at maximum takes the whole cast — the mana
is still spent and the school is still trained, because the refusals in FR-2 are
the whole of what refuses a heal.

### The picture a cast puts on the map

**FR-4** A world holds a **spell effect mark** on an entity: how many ticks of it
are left, and which spell set it. Both are per-entity state. They serialize with
the world, are covered by the digest, and fall by one on every tick until they
reach zero, at which point the spell id is cleared too — so a world with no live
mark is byte-identical to one that never had one.

**FR-5** A cast that is **applied** — one that passed every refusal and paid its
cost — sets that mark on the caster and on the target. A cast at oneself sets one
mark, not two.

**FR-5's SECOND CLAUSE AND SC-2 ARE REVERSED BY FR-19 AND FR-22.** DD-7.

**FR-6** The mark reaches the front end on the same per-tick seam that already
carries an entity's health, and the front end draws it: a ring on the marked
unit, in the colour of the marking spell's own school. Nothing about the drawing
is decoded and its whole specification is one sentence: **the player sees which
unit a spell has just touched, and in what element**.

### The autocast toggle

**FR-7** An entity carries **one autocast spell** — a spell id it casts unbidden,
0 for none — and a **wait**, the ticks before it may cast again. Both are
per-entity state, serialize with the world and are covered by the digest. An
autocast id of 0 is "no autocast" and is the state every entity is built in.

**FR-8** On every tick, before commands are applied, an entity whose autocast id
is nonzero and whose wait has reached zero attempts one cast of that spell. It
resolves through **exactly the same arm a commanded cast does** — the same
refusals, the same cost, the same roll, the same award, the same mark — so an
autocast can do nothing a player could not have ordered by hand. The wait is
reset to its full period only when the cast was **applied**; a refused attempt
leaves the wait at zero and is retried on the next tick, and costs nothing.

**FR-9** What an unbidden cast aims at is decided by the row's own kind and by a
total order, so two worlds stepped from the same state pick the same target:

- a `Damaging` row aims at the **nearest hostile living entity in range**, ties
  broken by the lower entity id;
- a `Restorative` row aims at the **living entity in range, not hostile to the
  caster, whose health is furthest below its own maximum**, ties broken by the
  lower entity id; a caster whose own health is down is a candidate like any
  other, and a candidate already at full health is not a candidate at all;
- a row that is neither aims at nothing and the attempt is refused.

An attempt that finds no target is refused and costs nothing.

**FR-10** The player toggles it: with a unit selected and a spell selected in
that unit's book, **`Ctrl+A`** makes that spell the unit's autocast spell, and the
same combination on the same spell clears it. Toggling a different spell replaces
whatever was set. The key does nothing with no unit selected, no spell selected,
or over a unit the local player does not own.

**FR-11** An autocasting spell's cell in the book carries a **rotating dashed
border**: a dashed outline whose dashes travel around the cell, driven by the
ambient animation counter the rest of the picture animates from, so it turns at
the speed the player has set and stops when he pauses. It is drawn over the
cell's border and on top of the icon, and is the only mark that says a spell is
on autocast.

### The spellbook popup

**FR-12** Hovering a spell's cell shows a popup stating that spell: its name,
mana cost, range in cells, school, the band it rolls (`4-8` for Fire Arrow),
whether that band is damage or healing, and, when it is the hovered unit's
autocast spell, that it is on autocast. A spell the world's table does not hold
shows its name alone.

It is a **hover and issues nothing**: it reads the item popup's own per-frame
cursor, touches no click state and gates no press, and uses that popup's own
composition, so the two boxes differ only in the lines they are handed.

**FR-13** The lines are composed on the far side of the drawing seam, beside the
item popup's own, from the world's spell table and the hovered unit's state. The
drawing tier holds no spell knowledge: it is handed an id, a name, an icon and a
list of lines per entry, and can no more compute a mana cost than an item's
weight.

### Driving it without a window

**FR-14** The scenario language reaches this story's behaviours, so the slice can
be driven through the shipped binary rather than through test code alone. An
authored world states a **spell table** — one row per spell, with the arm named
as one word (`damage` or `heal`) rather than as two flags a file could set
together — and a unit states the four fields a caster needs: its mana pool, its
Mind, the spell ids it knows, and the spell it casts unbidden. A unit also states
`max_hp`, so a **wounded** unit can be authored, which is what a heal is aimed
at; a file that states none keeps the health it states as its maximum.

Two orders are added. `cast` names a target and a spell. `autocast` names a spell
alone, `0` clearing it — the field is required rather than defaulted, so a step
with none is refused rather than read as a clear. Both issue exactly the
production commands a player's own click and key produce. Three assertions are
added: a unit's mana, in the `at_least`/`at_most` shape the health assertion
already uses, and its autocast spell.

The story ships one scenario using them. It is `synthetic`, so it needs no
install and runs under `go test`.

## Acceptance criteria

**AC-1** The shipped `Spells` collection loads with Heal `Restorative` and not
`Damaging`, Fire Arrow `Damaging` and not `Restorative`, and Drain Life neither.
No other row is `Restorative`.

**AC-2** A caster with mana at least the cost, healing a wounded ally in range,
loses exactly the cost once and the ally's health rises by an amount inside
`[base, base+spread]`, never above its maximum.

**AC-3** A heal aimed at a hostile unit spends no mana, rolls nothing and changes
no health: the world after is byte-identical to the world before. The same is
true of a heal aimed at a dead body.

**AC-4** A heal aimed at the caster himself is applied. A damage cast aimed at
the caster himself is still refused.

**AC-5** An applied cast leaves a mark on the caster and on the target naming the
cast spell, the mark falls by one per tick, and a world whose marks have all
expired is byte-identical to the same world stepped without a cast having
happened. A refused cast leaves no mark.

**AC-6** A world carrying an autocast id, a wait or a live mark round-trips
through the byte form unchanged; two worlds differing only in one of those three
have different digests; and a byte form declaring the previous version is
refused.

**AC-7** An entity with a damaging autocast spell, mana and a hostile unit in
range casts at it without any command being sent, marks it, and does not cast
again until the wait has run out. With the hostile unit out of range it casts
nothing and spends nothing.

**AC-8** An entity with a restorative autocast spell picks the ally furthest
below full health, and picks the lower entity id when two are equally hurt. With
every ally at full health it casts nothing.

**AC-9** An autocast that cannot afford its cost, whose caster is not a mage, or
whose caster does not know the spell is refused exactly as the commanded cast is,
and the wait is not reset — so the attempt is made again on the next tick.

**AC-10** `Ctrl+A` with a unit and a spell selected sets that unit's autocast to
that spell; the same combination again clears it; a different spell replaces it.
With nothing selected it changes nothing.

**AC-11** The book cell of an autocasting spell draws a dashed border whose dash
offset advances with the ambient counter, and the cell of every other spell does
not. Two frames one ambient step apart draw the dashes in different places.

**AC-12a** The shipped scenario runs to completion through the production
scenario runner with no install present: the heal raises a wounded ally and costs
its mana exactly once, the damage cast wounds an enemy and costs its own, the
toggle sets and clears the autocast id, and the unbidden casts alone fell the
enemy. A spell row naming an arm this build has no name for, a spell id of 0, a
repeated spell id, a `cast` with no spell, an `autocast` with no spell, and an
`hp` above the `max_hp` it is measured against are each refused by name.

**AC-12** Hovering a spell cell yields the popup lines for that spell, including
its band and its cost; hovering an empty cell or a cell past the end of the book
yields none; and the popup issues no order and swallows no press.

## Properties

**P-1** An autocast reads and writes only the caster, its chosen target and the
world's generator — the same reach a commanded cast has. It touches no route, no
group and no script register.

**P-2** Two worlds stepped from the same seed and the same commands stay
byte-identical, autocasts included. Nothing on this path reads a clock, the
environment or a package-global generator, and no float value exists inside the
simulation package.

**P-3** A refused cast — commanded or unbidden — is indistinguishable from no cast
at all, including in the generator's state and including the mark: nothing is
rolled and nothing is marked before every refusal has passed.

**P-4** No spell id is compared to a literal anywhere in the simulation package.
The restorative arm is driven by the row's own flag, the autocast by the row's
own flag, and the mark by the row's own id. The one place an id is named is the
table loader, where the heal id already lived.

**P-5** The drawing tier holds no spell knowledge. It receives ids, names, icons,
lines and a flag per entry, and derives no cost, range, band or school of its own.

## Out of scope

The other twenty-six spells. Each keeps exactly the 0127 behaviour: a
damaging row casts, every other row is refused by the table's own flags. No new
arm is written for any of them.

**SC-2 IS WITHDRAWN.** It cut everything that flies. A cast now animates its
caster (FR-22, decoded) and draws a bolt and a burst (FR-19, authored). 0127 SC-1
— the shipped `projectiles.reg` art itself — stays cut and is a story of its own.

Every elemental resistance, area shape and distribution, the `Effects` string,
learning, and the cast from an item. 0127 SC-2, SC-3, SC-4, SC-5 and SC-6 all
stand unchanged.

The shipped `text/spell.txt` description lines. The popup states the numbers this
tree holds; reading the installed description text is a later story.

Autocast for anyone but a unit the local player owns. A monster carries the field
— it is per-entity state and a save must carry it for every entity — and nothing
in this build ever sets it on one.

A `frontend`-stage scenario reaching the spellbook, the popup or the autocast
key. FR-14's additions are all `mission`-stage: the two surfaces are witnessed in
`pkg/ui`'s own tests and the rule is witnessed through the scenario.

An autocast that survives a mission boundary or a campaign hop. It is world state
and it lives as long as the world does.

## The second round

The owner rejected the story from his screen: no autocast at all, no graphics; no
fire arrow animation when the mage fights with his staff; and the staff animation
not corresponding to the moments the spell is applied. Three of the four are FR-5
and SC-2 carried out faithfully; the fourth had a cause upstream of everything
this story built. He then authored FR-18, FR-23 and FR-24.

### The book a saved character knows

**FR-15** A party member restored from a save knows the spells his own `Humans`
row names — the sixth value re-derived from the row the character's head points
at, beside the profile, figure, face, mage flag and derived-stat template. An
empty book draws no spellbook bar at all (FR-10), so this is what puts the bar,
the spell selection and the autocast key within a saved party's reach.

The save's own spellbook is not read: what a `Spell` record's `+0x08` names is an
open Unknown. DD-8 discloses what that costs.

### The observation

**FR-16** A step can be asked what casts it applied: the caster, target, row,
school, the caster's owner and the two cells the cast ran between, for every cast
past every refusal. A refused cast is reported as nothing.

It is a **return value and not state**: nothing is stored on a world, it enters no
byte of the form and no bit of the digest, and an observed and an unobserved step
of one world produce the same world byte for byte. It exists because the mark of
FR-4 is symmetric — it says only that a unit was touched — so the two ends of a
cast cannot be told apart in the world's state.

**FR-17** A weapon-borne release marks both actors and is observed, as an applied
book cast is.

### What an unbidden cast chooses

**FR-18** Authored by the owner. An entity attempts, in this order, stopping at
the first row applied:

1. the spell it has on autocast, when that row is restorative — in battle and out
   of it. This is the only way a heal outranks an attack while a fight is on, and
   the whole of what arming a heal buys;
2. out of battle, the lowest-id restorative row it **knows**, armed or not.
   Knowing the spell is the whole condition;
3. whatever it has on autocast, so in battle an armed attacking row is what runs.

An entity with nothing armed and an empty book attempts nothing. A row that finds
no target, and one the cast arm refuses, fall through to the next.

**In battle** is: the entity holds an attack target, or a living hostile stands
within its own `ScanRange` floored at the minimum guard range the engagement pass
already reads. DD-9 discloses this as a reading.

The unarmed out-of-battle heal keeps **a quarter** of the caster's mana pool in
reserve; an armed one keeps none. In his units: *a mage heals the party between
fights and will not spend below a quarter of his mana doing it; arm the heal
yourself and he spends it all, and heals before he strikes.*

Every other clause of an unbidden cast is FR-8's and FR-9's, unchanged.

### The cast the player sees

**FR-19** An applied cast is drawn as a bolt crossing the ground and a burst
where it lands, in the colour of the casting spell's own school.

- a book cast has no wind-up — it resolves inside the command phase of the tick
  it is ordered on — so its bolt is spawned from the observation (FR-16), flies
  over the ticks after, then bursts;
- a weapon-borne cast is drawn live off the attack cycle, by the swing clock over
  the charge, so it reaches the victim on the tick the release resolves.

The picture is authored and is not the shipped art; DD-7 and DD-11 disclose it.
The mark of FR-6 stands beneath it. A bolt takes the caster's own relief lift and
fog gate.

**FR-20** The drawn attack run restarts whenever an attacker enters **either**
wind-up phase — the charging one, which resolves a blow, and the casting one,
which releases a weapon-borne spell — and the swing's voicing is gated on the
same pair.

**FR-21** Arming an autocast queues its command and marks the unit nothing else.
In particular it does not mark it as one the player has taken over, which would
permanently drop that unit's scripted command track.

**FR-22** An applied cast makes its caster play that caster's own attack run,
once, over the interval FR-23 scales it to. A weapon-borne cast already does, its
caster holding an attack target; a book cast sets none, so the front end holds
the run open for one. `MAGIC-CASTANIM-029`, DD-7.

### One projectile, one swing

**FR-23** **Authored by the owner, 2026-08-14.** Every magic projectile has its
own visible swing behind it; the animation cadence follows the cast rate.

A cast's drawn attack run is **scaled** to the interval its projectile crosses in
— the wind-up for a weapon-borne cast, the caster's attack charge for a book
cast. A longer track is compressed and a shorter one stretched, so one complete
swing covers one projectile at any cadence. A cast arriving inside a run restarts
it. A **melee blow is not scaled** and keeps its one-frame-per-tick clock.

**FR-24** **Authored by the owner, 2026-08-14.** A swing is never shorter than
**eight ticks** and a book cast's cadence is floored at the same eight: a cast,
commanded or unbidden, is refused while the caster's wait stands, so the cast is
held to the swing rather than the swing dropped.

The floor is a named constant in both tiers and is **not** a registry value,
though it agrees with two decoded figures: the shortest expanded `Attack`
timeline over 34 classes is 8 frames (`MAGIC-CASTANIM-029`) and the original's
wind-up `actor+0x134` defaults to 8 (`MAGIC-CASTTICK-030`).

A **weapon-borne release is not held to the floor**: its cadence is the attack
cycle's own wind-up, which always has a swing behind it, and refusing one would
leave the swing-without-a-projectile FR-23 forbids. This supersedes 0127 FR-5's
claim that a slice carrying one cast twice pays twice; it now pays once.

### Acceptance criteria, second round

**AC-13** A restored character knows exactly the spells his `Humans` row names;
a row naming none, and a non-`Human` record, give an empty book.

**AC-14** An observed step reports one event per applied cast naming the caster,
target, row, school, owner and the two cells; reports nothing for a refused cast;
and leaves the world byte-identical to the same world stepped unobserved.

**AC-15** An applied weapon-borne release marks caster and victim with the
released row and is observed as a weapon-borne cast.

**AC-16** Out of battle, an unarmed caster that knows a restorative row heals an
ally below full health, or itself. In battle an armed attacking row runs and no
ally is healed; an armed restorative row heals and no attack is cast. A caster
below its reserve makes no unarmed attempt, and makes it anyway with the row
armed.

**AC-17** A book cast spawns one bolt and a weapon-borne cast none. A bolt
travels its span, bursts through widening rings on the target's cell, then
expires. A weapon-borne bolt stands at the caster's cell at the start of the
wind-up and at the victim's at its end; an attacker charging, holding no victim,
carrying no weapon spell or dead draws none.

**AC-18** Entering either wind-up phase restarts the swing run; a second tick in
the same phase advances it.

**AC-19** Arming an autocast queues one autocast command, leaves the world's hash
unchanged, and marks no unit commanded.

**AC-20** An applied book cast puts its caster into a run, which holds the drawn
attack clock open for a caster with no victim and ends on its last tick. A
weapon-borne cast starts no second run.

**AC-21** A span longer than the track holds each frame for several ticks, a
shorter one skips frames, and a clock past its span shows the last frame. A
weapon-borne cast scales to its wind-up, a book cast to its run's span, a melee
blow to nothing.

**AC-22** A book cast's run is never shorter than eight ticks whatever the
caster's charge. A second cast inside eight ticks spends and rolls nothing, and
the same command applies once the wait has run out.

### Design decisions, second round

**DD-7 — what a cast draws, now that `EXP-0163` has answered.**

Decoded and built (FR-22), `MAGIC-CASTANIM-029` (High): every cast writes
`2*spellId + 8`, even for every id, so the parity guard always takes the even
branch — which gives the **caster** action code 8 and a countdown of its class's
expanded `Attack` length. A cast plays the caster's Attack run, once, one frame
per tick, 8 to 28 ticks over 34 classes. A weapon-borne cast uses the same sender
and produces the identical picture.

Decoded and **not** built, `MAGIC-BURST-031`: `2*id + 9` is reached only through
the `AreaEffect` object, chosen by `Distribution system`, and 10 of 28 spells
build one. **Fire Arrow is not among them.** This build has no area arm (0127
SC-3 stands) and paints no ring of cells. Still open, `MAGIC-SENDER-032`: opcode
`0x87` and `R0635`'s eleven call sites are unread.

Shown to the player, in his units: *the caster swings when he casts, as the
original makes him* — FR-22, decoded — and *a coloured mark leaves him and opens
into a ring on whoever it hit* — FR-19, **ours**. The original flies a projectile
only where the caster's class carries a `Projectile` key (`REG-PROJ-086`) and
bursts only for the 10 area spells; this build marks every cast, and draws a
coloured square because nothing here loads `projectiles.reg`.

**DD-11 — the two clocks are tied, and the original's are not.**
`MAGIC-CASTTICK-030` (High): the original applies the effect on tick
`actor+0x134`, default 8, while the client spawns the projectile on `ShootDelay`,
and nothing equates them. FR-23 ties them anyway, on the owner's ruling; nothing
in that claim is doubted or retracted, and the divergence is authored.

A book cast is the one place they cannot coincide: the simulation resolves a
commanded cast inside the tick it arrives on, so its mark runs over the ticks
after. In his units: *a spell cast from the book lands when you order it, and its
picture runs just after.*

**DD-8 — a restored character's book is his row's, not his save's.** Nothing here
teaches a spell (0127 SC-5 stands), so no character this tree produces can know
more than his class starts with; the divergence is against the original's own
saves alone. In his units: *a hero loaded from a save knows the spells his class
starts with; if you taught him more in the original game, this build does not
know it.*

**DD-9 — "no battle" is a reading, not a decoded fact.** Nothing on an entity
says a fight is on. An attack target and a living hostile inside the guard radius
are the nearest words this package has, and they keep "in battle" the distance
the AI already fights at.

**DD-10 — the reserve is a share, not a count.** A quarter of the caster's own
maximum, so it scales with the character. Authored: the original has no
book-spell autocast to take a reserve from.
