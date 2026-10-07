# Plan — the hero is a real fighter, and his numbers are on screen

## Decisions

- **DD-1 — The point-buy arithmetic lives in `pkg/data`, in its own file beside the fold.** It is
  arithmetic over a definition; it needs `math.Pow`, which `pkg/sim` forbids; and it sits below the
  determinism wall, so only truncated integers cross. A new file rather than more of `hero.go`,
  because generation and the derive are two subjects: one decides what a hero *is*, the other what a
  hero *does*. (FR-1, FR-2, P-2)

- **DD-2 — The spread is chosen by a stated rule, and the rule is what is authored.** Floor Mind and
  Spirit; maximise Body; then maximise Reaction with what remains. That yields **43 / 26 / 15 / 15**
  at a cost of 139. The rule is the fighter reading of `HERO-STATDMG-036` — only Body reaches damage,
  only Body and Reaction reach to-hit — turned into a search over the legal space, and it is chosen
  over the two rivals for reasons that are stated rather than felt:
  - *balanced (34/34/34/34)* spends the whole budget and buys the worst damage of the three,
    because the cost escalates and Body is bought last;
  - *accurate (39/38/15/15)* trades a band step for eleven to-hit, against a mission-10 defence of
    about 10 — the eleven buys nothing observable;
  - *heavy (43/26/15/15)* buys the top of the reachable damage range.

  **The three steps are ordered and the floors are step one**, because the maximalities alone do not
  pick a point: `T(16)` is 3, so 43/26/16/15 and 43/26/15/16 are also legal, Body-maximal and
  Reaction-maximal. The floors are what make the answer unique, and the leftover point is what that
  costs — it buys no step on either axis a blow reads. (FR-3)

- **DD-3 — The seam is one function in `pkg/game`, not a variable and not a `pkg/data` constant.**
  `PartySpread()` returns the authored value; `PartyHero()` composes it with the already-authored
  skill slot. A **function** rather than a package variable, for `MissionParty`'s own reason: a
  variable is writable from anywhere. It sits in `pkg/game` beside `PartySkillSlot`, where this
  tree's authored answers to *"generation is a screen we do not have"* already live — in `pkg/data`
  an authored choice would look like a decoded default. (FR-3)

- **DD-4 — `NewCampaignHero` is kept, and is re-expressed as the general constructor applied to the
  generation start.** `NewHero(spread, slot)` becomes the one construction implementation and
  `NewCampaignHero(slot)` becomes `NewHero(ChargenSpread(), slot)`. Deleting it would destroy the
  home of a decoded fact — 25/25/25/25 is `R0834`'s own initialiser — and defining the two
  side by side would let them drift. Its doc comment is corrected in place to say it is the
  generation **start** rather than "the character a new campaign starts with", which is a statement
  that stops being true in this story. (FR-4, FR-5)

- **DD-5 — The sheet goes on the unit information panel, and extends its closed field set rather
  than opening a second box.** `panel.go` already names this as its extension point. The debug
  readout was rejected, and **not** because it is hidden — it is *shown* by default and `F1` hides
  it — but because its own file states that nothing in it asserts anything about the game: a box
  carrying a frame rate and a world digest is not where a player reads his build. A third box was
  rejected because the panel already answers *"tell me about this unit"*. (FR-6, FR-7, P-4)

- **DD-6 — Both halves cross on the ONE per-tick entity seam; the character is resolved at open and
  held on the driver, exactly as a tier is.** The eight are simulation state and are read off the
  entity on the tick they are stated. The four statistics, the trained skill and the weapon are a
  per-entity constant fixed when the map opens, so they are resolved once into a lookup on the
  driver — born with the map, dropped with it, keyed by entity id, read inside `entityDraws` and
  never ranged — and the **resolved value** crosses beside the eight. **An id with no entry states no
  character**, which is the tier lookup's own rule.

  A second, viewer-side, id-keyed push was designed and **rejected**. What made it look attractive
  was a misreading: `MapEntity` already carries three per-entity constants — name, owner, speed — so
  "a constant must not cross a per-tick seam" is not a cost here at all. It would have bought three
  real ones: a lookup on the far side of a seam whose rule is that the window tier derives nothing,
  its own replace-or-accumulate and cache-invalidation rules, and a correctness holding only while
  every open makes a fresh viewer, which nothing enforces. The full argument sits at the fields.

  The lookup is a **constructor argument**, because the driver's tick-0 push is the only push a
  mission opened and left stopped ever gets. (FR-6, FR-7, P-3, P-4, P-5)

- **DD-6a — Each group carries a presence flag, and an unsupplied group is NOT TOLD.** The field
  space `PanelField` is **shared with the debug readout**, whose own set starts at **16** and
  reserves `4..15` for the panel to grow into. This story spends that reservation exactly: six
  character fields at `4..9`, six combat fields at `10..15`. **The reservation is now full** — the
  readout owns `16..31`, and a thirteenth panel field takes **32**. The readout's `PanelFieldCadence`
  already holds 16, so the panel's cadence row is named for the *attack* cadence it states, and the
  two resolvers stay total and disjoint.

  Each group carries a bool beside its numbers, because the zero value of six integers is
  indistinguishable from six nobody filled in. It is the readout's own *"the zero value is not told"*
  rule, applied one box over. (FR-6, FR-7)

- **DD-6b — A skill's name is the shipped table's warrior half, composed in `pkg/data`.** That
  package owns the slot constants, and the window tier may name no data type, so it receives a
  composed string like every other name it draws. (FR-7)

- **DD-7 — Reproduce the two layout facts that are decoded and author the rest.** Those two are the
  four statistics' row order and the damage row's `base-(base + spread)` composition. Row set,
  ranking, labels, colours and corner are ours, and that is a verdict rather than a hold. The labels
  are the game's own key names, in English because the original's come from a registry this tree does
  not read for text. (FR-6, FR-7)

- **DD-8 — The party's entity ids are recorded by the start, and the mission keeps the party it was
  started with.** The start already knows the ids it minted — it appends the party after the map's
  placements — so it records them beside the cells it already records; the mission carries the
  member slice it was handed, so the two halves of a character (which id, and whose statistics) meet
  in one place. The alternative, `len(placements) + i` recomputed in `pkg/game`, is a second copy of
  a rule that lives one package away and is exactly the class of duplication the entity seam's own
  comments reject. A viewer nobody pushes characters to holds none and states none, and a map opened
  twice gets a fresh viewer, so an id-keyed lookup cannot outlive the world that minted the ids.
  (FR-7)

- **DD-9 — The check line gains the character, keeps its shape, and is re-expressed through the ONE
  seam.** One clause, still one line, still printed on every run: statistics, skill, weapon, band,
  to-hit, defence. Printing the numbers without the spread would leave the line unable to say *which
  build* produced them, which is the one thing this story changed.

  **It builds no hero of its own.** Today it constructs the generation start independently of
  `MissionParty`, and left alone it would compile, pass, and print 7-10 / 39 for a party that places
  10-16 / 49 — a divergence this story is uniquely bad at noticing, because both figures are correct
  answers to different questions and one of them is a criterion. So the clause is derived from
  `PartyHero()`, and the criterion is that the line's numbers **equal the party member's own**, not
  merely that the line has the right shape. (FR-3, FR-8)

- **DD-10 — No version bump, and the reason is structural rather than observed.** `formatVersion`
  stays **13**. The eight fields are already on the entity and already in the record; a statistic is
  a loader input and reaches no simulation field; and the panel is above the wall entirely. What
  changes is the *value* of eight fields that already exist, which a digest reflects and a layout
  does not. The falsifier is the round-trip test asserting the version byte **by number**: if a bump
  were needed it would fail. **15 is this story's number if one is ever needed; 14 belongs to
  `impl/0081-attack-legible`.** (FR-9)

- **DD-11 — Speed is derived in `pkg/data` beside the fold, applied in the start, and is NOT one of
  the eight.** `Hero.Speed()` stands beside `Derive` rather than inside it: `Combat` is *the eight
  numbers a blow reads*, and a ninth field on it would make every consumer of a blow's numbers carry
  a movement term. The start writes it onto the entity where it already wrote the class default.

  **Uniformly, with no fallback for a hero who has no statistics.** A zero-value member's speed
  becomes 0 rather than the table's 10, and 0 is what the law gives at Reaction 0. The start's own
  doc already prefers *"the same arithmetic rather than a fallback"*; a floor here would be the one
  place a party number came from somewhere else. `rateOf` floors the resulting rate at 1, so such a
  member still moves. Two pinned tests move with it, which is those pins doing their job.

  **The two unimplemented terms are omitted on evidence, not on convenience.** The `+10` arm is
  gated on type ids `0x13`/`0x15`; humans are `0x21..0x24` inside a family bounded at `0x21..0x3f`,
  so **a hero cannot reach that arm** — the claim's standing *Unknown* is about which class does
  reach it, which is not a question this story has to answer. The load penalty is gated on
  `load >= capacity` and this tree has no inventory. (FR-10)

- **DD-12 — The sheet's speed row reuses the seam field that already exists and takes number 32.**
  `MapEntity.Speed` has crossed since `0060` for the readout to state, so nothing is added to the
  seam; the subject carries it and the row is gated on the **combat** flag, because both come off the
  same entity in the same push and a second flag would be a second thing to forget. The field number
  is **32**, not 16..31 — `PanelFieldSpeed` at 21 is the readout's and this one is
  `PanelFieldMoveSpeed` — which is the allocation rule DD-6a wrote down being obeyed the first time
  it bound. (FR-10)

## Risks

- **R-1 — The spread is min-maxed, and the floors are a bill a later story pays.** Mind reaches
  sight and Spirit reaches mana and the five protections in the original. None is derived here, so
  the floors cost nothing today; a story that derives one will find this hero weak in it. Mitigated
  by DD-3: the answer is one function body.
- **R-2 — The panel grows from four rows to fifteen and could run off a small window.** The box is
  fit-to-content and anchored bottom-left, so height grows upward from the margin. Measured rather
  than argued: the composed height is asserted in a test and checked against the default window.
- **R-3 — Three lanes are live in files this story touches.** `pkg/ui/overlay.go`,
  `pkg/game/world.go` and `pkg/game/frontend.go` are all reachable from another lane's story.
  Mitigated by appending fields at the end of a struct rather than inserting, by adding no viewer
  setter at all, and by rebasing on `origin/master` before pushing.
- **R-4 — The hero becoming stronger could move an existing test's outcome.** His band goes from
  7-10 to 10-16 and his to-hit from 39 to 49, so anything that pins a kill time or a tick count
  moves. Mitigated by running the whole suite and by reading the failures rather than adjusting the
  numbers until they pass.

## Success criteria

- **SC-1 — The cost function and the legal space reproduce every consequence research states of
  them.** T(15), T(25), T(45), the 22-point last step, the refund symmetry, and the three reachable
  maxima 42 / 43 / 34. (FR-1, FR-2; AC-1, AC-2, AC-3, AC-4, AC-5, AC-6, AC-7, AC-8; P-1)
- **SC-2 — The party's spread is legal, is the rule's own answer, and its numbers land on the
  owner's measured band.** (FR-3, FR-4, FR-5; AC-9, AC-10, AC-11, AC-12)
- **SC-3 — The panel states a character and the eight numbers, and states nothing for a unit that
  has none.** (FR-6, FR-7; AC-13, AC-14, AC-15, AC-16; P-4, P-5)
- **SC-4 — The values arrive on screen from the world and the loader, on both roots.** (FR-6, FR-7;
  AC-17)
- **SC-5 — The check line states the build.** (FR-8; AC-18)
- **SC-7 — The hero's speed is the law's answer to his Reaction, and it is on the sheet.** Including
  what it comes to, reported against the owner's own figure rather than fitted to it. (FR-10; AC-21,
  AC-22, AC-23)
- **SC-6 — Nothing simulated moved.** Version 13 by number, equal bytes, equal hash, the suite green
  and `pkg/sim` untouched. (FR-9; AC-19, AC-20; P-2, P-3)
