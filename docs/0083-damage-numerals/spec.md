# Spec — a blow leaves a number behind

**Intensity: spec-anchored / static. Terrain: brownfield** for the mission tool's health reporting,
which ships today and answers wrongly, and for the front end's `L` binding; **greenfield** for the
numeral itself, which nothing in this tree has a counterpart for.

A fight in this build resolves blows that leave no mark but a shrinking bar. The original answers a
landed blow with exactly two things — a numeral that flies out of the struck unit and vanishes, and
two sounds. This story builds the numeral. It builds no sound, because this tree has no audio layer
of any kind: no device, no channel, no per-class sound array reader. That is stated rather than
approximated.

## Functional requirements

- **FR-1 — the figure is the FRONT END'S OWN SUBTRACTION, and no simulation state is added.**
  What the front end is told is a unit's health, which it is already told. It remembers the health
  it last saw for each entity, and a strictly smaller one is a blow whose figure is
  `remembered - current`, drawn as a decimal numeral and nothing else — no sign, no unit, no word.
  Nothing is written into the simulation, no field is added to any record, and the byte form's
  version does not move.

  **The remembered health is updated on every push, whatever else this story does.** It is not
  conditional on the display, on the figure being positive, or on the entity being alive.

  An entity the front end has never seen before is remembered at the health it arrives holding and
  produces no figure: arriving is not a wound.

- **FR-2 — TWO GATES, and both must pass before a figure exists at all.** The display must be on
  (FR-9), and the health must have strictly DECREASED. A health that rose, or did not move,
  produces nothing.

  **The display gate is on CREATION and not on drawing.** With the display off no record is made, so
  turning it on shows nothing until the next blow — it does not reveal figures that were accruing
  invisibly.

  *Folded from hotfix `9729459` — see `docs/hotfix/ARCHIVE.md#9729459`.* The decrease is measured
  against the health the front end already held, and no figure exists at all when that already-held
  health was at or below zero — the killing blow still shows, a corpse's decay never does.

- **FR-3 — a second blow inside the window is ONE figure, not two.** A record already standing for
  an entity takes the new damage **added into its own number**. Its birth, its offset and its drift
  sign are untouched — so the merged figure rises from where the first one had reached, shows the
  sum, and vanishes on the FIRST blow's clock rather than being renewed by the second.

- **FR-4 — the colour is the VICTIM OWNER'S, never the attacker's.** Two units of different owners
  struck by one attacker carry two colours; one unit struck by two attackers of different owners
  carries one. An entity owned by nobody has its own colour, distinct from every owner's.

  The colour VALUES are ours: the rule is that the colour is a function of the struck unit's owner,
  and the palette this build uses is authored.

- **FR-5 — NO SEVERITY COLOUR IS COMPUTED OR DRAWN, and the absence is the requirement.** The
  original picks one of three colours on every landed blow by the victim's remaining health —
  at or above half its maximum, at or above a quarter, below — and then reads that choice nowhere:
  three writes and no read, in a routine read whole. A build that rendered one would add something
  the original computes and deliberately discards. Nothing here computes a health band, and no
  drawn property of a figure depends on how badly the victim is hurt.

- **FR-6 — the life is 1000 ms of WALL CLOCK.** A record is drawn while no more than 1000
  milliseconds have passed since it was made — a record at exactly 1000 ms is still drawn — and past
  that it is gone.

  **It is wall clock and not the tick**, so it is unmoved by the game's speed, by the player's pause
  and by a popup holding the map: figures made before any of those still vanish a second later. The
  consequence is deliberate and is the decoded behaviour, not a defect: at a slow cadence a figure
  drifts a shorter distance before it expires.

- **FR-7 — the drift is TICK-PACED and has two axes.** Once per tick of the map's ambient animation
  clock — the clock whose period the speed setting sets — every live record moves **up by 2** and
  **sideways by 1**. The vertical step is unconditional. The horizontal sign is the OWNERSHIP one:
  a figure over a unit belonging to the local participant steps one way, a figure over anyone
  else's steps the other, and the two are opposite. So a figure travels diagonally.

  A clock that advanced several ticks between two frames drifts a record by that many steps; a frame
  on which the clock did not advance drifts nothing.

- **FR-8 — the initial offset is not zero, and its horizontal sign is the DRIFT'S.** A record is
  born offset from the struck unit's own drawn position: `16` horizontally, carrying the same sign
  FR-7 gives that unit's ownership, and `48` upward. So a figure starts out of the unit and to the
  side it will continue toward, rather than starting on it.

  **A decoded term has no counterpart here and is named rather than substituted.** The original
  scales both offsets by a per-unit scalar this tree cannot compute, that scalar's source being
  undecoded. This build takes it as one, which is what makes the offsets exactly the numbers above.
  If it is later decoded to be anything else, these two numbers are what change.

- **FR-9 — the display has a toggle, it DEFAULTS TO ON, and its key is Ctrl+L.** A map screen opens
  showing figures without anything being pressed. Ctrl+L flips it, on a press edge, on the map
  screen and nowhere else.

  **`L` alone remains the chip key and Ctrl+L no longer chips.** The two are exclusive: a press of
  `L` reaches exactly one of them, decided by whether Ctrl is held. This narrows a binding that
  earlier stories set on our own authority; the decoded half is the modifier and the letter, and it
  is honoured.

- **FR-10 — a figure is drawn TWICE, a shadow under a face.** The shadow's offset is ours; that
  there are two issues of the same text at offset positions is not.

- **FR-11 — a figure hangs off the struck unit's OWN drawn position and follows it.** Its place is
  that unit's position as this frame draws it — carrying the same sub-tick displacement, the same
  relief lift and the same camera transform every other mark on that unit carries — plus the record's
  own accumulated offset. A unit that walks takes its figures with it.

  A record whose unit is not in this frame's entities is **not drawn and not destroyed**: it expires
  on its own clock like every other.

- **FR-12 — the mission tool distinguishes ABSENT from DEAD.** Health of an entity the world does
  not hold is not a number, and no report line may present it as one.

  **A drive naming a unit that is not there FAILS**, before any order is issued, saying which
  reference it could not resolve. It never reports an outcome for it. This replaces the answer the
  tool gives today, where an absent entity reads as health zero, zero reads as fallen, and an
  attack order onto a party slot the map has no member for is reported as a kill on its first tick.

- **FR-13 — nothing else changes.** No route, no rate, no blow, no order, no digest, no byte form
  and no version. A run that draws figures and a run that does not are the same run, tick for tick
  and digest for digest. `pkg/ui` still names no simulation type.

## Out of scope, each with its reason

- **The two sounds a blow plays** — the attacker's swing and the victim's grunt, both positional,
  both indexed out of a per-class array, one of them throttled. This tree has **no audio layer at
  all**, so there is nothing to half-build against; and the sound's own health bands and its `-10`
  bound gate the sound, never the figure, so nothing about them belongs to a numeral.
- **The severity colour** (FR-5). Not merely unbuilt: deliberately absent.
- **The suppression flag.** The original can hold a record alive while skipping its draw, on a flag
  of the struck unit whose meaning is undecoded. Nothing here reproduces it.
- **The second half of the merge key.** The original keys a merge on the victim and on one further
  undecoded value; this keys on the victim alone.
- **The swing, the facing, the attack pointer and the stat panel** — all 0080's, 0081's and 0082's,
  and none of them is a consequence of a blow LANDING.

## Acceptance criteria

- **AC-1** A push whose health is below the remembered one makes a figure of exactly the difference;
  a push at the same health and one above it make none; the first push an entity is ever seen in
  makes none whatever its health.
- **AC-2** The remembered health follows every push, including one taken with the display off, one
  that raised the health and one of a unit that is not alive.
- **AC-3** With the display off no record exists after a blow, and turning it on afterwards draws
  nothing; the next blow after it is on draws one figure of that blow's own damage alone.
- **AC-4** Two blows inside the window leave ONE record whose number is the sum, whose birth is the
  first blow's and whose offset is the first blow's continued — and it expires on the first blow's
  clock, not the second's.
- **AC-5** Figures over units of two different owners take two different colours; two figures over
  units of one owner take one; the colour does not change with the attacker, and an unowned unit's
  differs from every owner's.
- **AC-6** No drawn property of a figure differs between a victim above half health, between a
  quarter and a half, and below a quarter, at equal damage.
- **AC-7** A record is present at 999, 1000 and absent at 1001 milliseconds after its birth; the
  same holds with the ambient clock never advanced at all.
- **AC-8** One tick of the ambient clock moves a record up by 2 and sideways by 1; N ticks between
  frames move it by N steps; zero ticks move it by none; the horizontal sign is opposite for a
  local-participant victim and a foreign one, and matches that record's own initial offset's sign.
- **AC-9** A fresh record's offset is `(+16, -48)` over a local-participant victim and `(-16, -48)`
  over a foreign one.
- **AC-10** With no local participant established, an UNOWNED victim reads as the local
  participant's and takes the `+` sign while every owned one takes the `-`; nothing panics or
  divides by zero at any owner value.
- **AC-11** Ctrl+L flips the display on the map screen and bare `L` does not; bare `L` still chips
  and Ctrl+L does not; neither does anything on the menu or picker screen; a map screen opens with
  the display on.
- **AC-12** A drawn figure yields two issues of its text at different positions.
- **AC-13** A figure's screen position is the struck unit's own placed position plus the record's
  offset, under a moved camera, a zoom, a sub-tick displacement and a relief lift; a record whose
  unit is absent from the frame yields no drawn figure and is still present until it expires.
- **AC-14** The mission tool reports an absent entity's health as absent and never as a number;
  a drive naming an absent unit returns an error naming that reference and issues no order; a drive
  naming a present one behaves exactly as it does today.
- **AC-15** No simulation state is added: the determinism package is untouched, its byte-form
  version literal still reads 14, and the import-graph check still refuses the front end any
  simulation type — so no figure, no remembered health and no display flag has anywhere to reach a
  digest from. (An A/B schedule run is NOT the witness here and would be an overstatement: the
  records live behind a method the tier that owns a world cannot call, so the two legs of such a
  run could not differ in what this criterion is about.)
- **AC-16** Against both lawful roots the mission tool fells a real unit and reports its health, and
  refuses a party slot the map holds no member for.

## Properties

- **P-1** A record's whole life is total: no input of health, ownership, tick count or elapsed time
  panics, divides by zero, or leaves a record neither drawn nor expiring.
- **P-2** The number a figure shows is always positive. A record with a non-positive number cannot
  be made and cannot be reached by merging.
- **P-3** Exactly one authority answers what a figure shows. The front end's remembered health is
  the only source of a damage figure in the tree, and nothing else derives one.
- **P-4** Nothing this story holds reaches the world: the records, the remembered health and the
  display flag are invisible to the byte form and to the digest.
- **P-5** The full local gate is clean: build, vet, gofmt, the test suite, the three repo scripts.
