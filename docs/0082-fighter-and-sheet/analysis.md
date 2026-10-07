# Analysis — the hero is a real fighter, and his numbers are on screen

Intensity: **spec-anchored / static**. Terrain: **brownfield** in `pkg/data/hero.go`,
`pkg/game/hero.go`, `pkg/ui/panel.go`, `pkg/game/world.go` and `pkg/game/frontend.go` — each has
shipped behaviour this story moves; **greenfield** for the point-buy arithmetic, which is new code
against no existing behaviour. Threshold **High** for the build (it reaches hashed simulation state),
**Medium** for the display (it reaches nothing).

## The baseline, read rather than remembered

`pkg/data/hero.go:99` — `NewCampaignHero` writes `Body: 25, Reaction: 25, Mind: 25, Spirit: 25` from
the one constant `ChargenStat`. `pkg/game/hero.go:101` hands that to every party member. So the
player's hero is a flat **25/25/25/25**, and the file's own neighbouring comment already says what is
wrong with that: *"chargen floors every stat at 15"*. Twenty-five is the **initialiser** — the value
`R0834` writes into all four rows before the player has spent a point — and the party has been
carrying the screen's opening state as though it were a character.

It is not only cosmetic. `HERO-STATDMG-036` establishes by a read census inside the sole writer that
only **Body** reaches damage and only **Body and Reaction** reach to-hit. A flat spread therefore puts
half its points where no blow reads them.

## Is the legal space actually decoded? Yes, and the two rows agree

Four claims fix it, and they are not independent statements of one number — one is derivable from
another, which is what makes the agreement worth checking rather than assuming.

- `HERO-COST-002` gives the cost function `T(n) = ftol(0.349 x 1.15^(n-1) + 0.5)`, **cumulative**:
  the total cost of one statistic standing at `n`, not the price of a step.
- `HERO-BUY-003` gives the two click bounds — a `+` refuses at `v >= 45`, a `-` refuses at `v <= 15` —
  and the start: four stats at 25, pool 100.
- `HERO-BUDGET-004` gives the accepting side's test, `sum T(stat) <= 140`, and states five
  consequences.
- `HERO-CAP-015` is the *post*-generation clamp, `min(stat, 50 + modifier)`. It is a different bound
  from the click cap and does not bind here: no legal spread reaches 50.

`HERO-BUDGET-004`'s consequences were **recomputed from `HERO-COST-002`'s formula** before anything
was built on either, because a claim quoting another claim's consequences is exactly where a
transcription error hides. All eight figures reproduce:

```
T(15)=2  T(25)=10  T(45)=164        4*T(25)+100 = 140
one stat alone, others at 25 -> 42 (cost 108 of 110)
one stat alone, others at 15 -> 43 (cost 124 of 134)
all four together            -> 34 (total 140 of 140)
last step 44 -> 45 costs 22
```

`T(45) = 164 > 134` is why the click cap of 45 is unreachable in generation, and the 22-point last
step is `HERO-COST-002`'s own quoted escalation. Two rows written from two experiments agree to the
unit.

## What is not decoded: which spread

Nothing in the corpus says what the owner's character should be. The budget bounds the space; it does
not pick a point in it. That is an **AUTHORED** verdict — absence established positively, not an
AMBER hold — and the decision belongs in `plan.md` with the rule that produced it, not here.

Worth recording that the space is small and its extremes are sharp: with Mind and Spirit floored, the
budget buys **Body 43 / Reaction 26** or **Body 39 / Reaction 38** but nothing between them that
raises both, because the cost escalates. The choice is therefore a real choice about the character
and not a rounding.

## Does the decode give us a panel to copy?

Partly, and the split is unusually clean.

`UNIT-PANEL-010` establishes the **value set** of the original's own unit information display: the
four statistics, the health pair, mana, to-hit, defence, absorption, the damage pair, speed, sight and
the two runs of five resistances and protections — a straight-line block of named `MOV`s behind a
one-hit call enumeration, High.

`UNIT-PANEL-011` then establishes, as a **negative about the instrument**, that the *layout* cannot be
settled at all: from `+0x14a` on, the block is addressed by computed index, so no displacement sweep
can name which cached value appears where. Its own words — *"a consumer reproducing the panel
therefore has the value set and its arithmetic on evidence, and its layout on nothing"*.

So: what to show is decoded, where to show it is ours. Two smaller layout facts survive that
negative and are usable, because neither comes from the unit panel:

- `HERO-STAT-001` — the chargen panel reads the four back in **display order** `0x138, 0x13b, 0x139,
  0x13a` into rows 0..3, i.e. **Body, Reaction, Mind, Spirit**. That is a decoded row order for the
  four statistics.
- `HERO-SHEET-038` — the sheet's damage line is the literal `%d-%d` over `base` and `base + spread`.
  That is a decoded *format* for one row, and it is the composition this tree's check line already
  uses.

## Where the display could go, and what was rejected

The tree has two boxes. The **debug readout** is the wrong one, and the reason is not the one I was
handed. It is **shown by default** — `readoutHidden` is stored inverted, so the zero value is *shown*
and `F1` is a **hide** (`readoutPresent`, `pkg/ui/readout.go`; read rather than assumed, after the
brief asserted the opposite). So "the owner cannot see it" is false: he has been looking at that box
all along and it still told him nothing about his character. The real reason is what the file itself
says — *"this is a development instrument, and no line of it asserts anything about the game"* — and a
box carrying a frame rate and a world digest is not where a player reads his own build.
The **unit information panel**
already describes the selected unit and its own comment already names the extension point: *"A later
story that makes another value real adds a constant here and a case in `panelText` — one switch, one
place."* The panel is also what `UNIT-PANEL-010` describes. It is the right box.
