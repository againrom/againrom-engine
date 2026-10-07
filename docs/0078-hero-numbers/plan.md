# Plan — the player's own units can fight

## Decisions

- **DD-1 — The fold lives in `pkg/data`, not in `pkg/sim` and not in `pkg/mapload`.** It is arithmetic
  over a definition, which is what that tier is; it needs `math.Pow`, which `pkg/sim` forbids and
  `pkg/data` does not; and it sits *below* the simulation, so only integers ever cross the
  determinism wall. Putting it in `pkg/mapload` would have made it unreachable to a later hero panel
  and to the character sheet without a second copy. (FR-2, P-2, P-5)

- **DD-2 — The nine doubles are decoded in `pkg/formats/databin`, by the tier that owns the format.**
  A new `EntryDoubles` reads the record's nine little-endian `float64`. The bytes are still kept; the
  package's *"slots are undecoded"* sentence is corrected in place, because `ITEM-LADDER-019` decoded
  them. `internal/synth` gains the matching writer so a fixture can carry real factors. (FR-3)

- **DD-3 — The name split is LONGEST MATCHING PREFIX ON A WORD BOUNDARY, and it is OURS.** Research
  states the *shape* of the parse and not the matching rule, and `Uncommon Magic Wood Short Bow`
  proves a material may be two words, so a rule is needed and none is published. Longest-prefix is
  chosen because it is the only rule under which a table holding both `Wood` and `Magic Wood`
  resolves that literal at all; table order breaks a tie. Its falsification is total and cheap: the
  five ordinary literals must resolve against the shipped file, which the build measures. (FR-3)

- **DD-4 — The shape and material tables are searched by the WEAPON LITERAL's words, never by an
  expected name.** Nothing asks whether `Common` is at index 0 or whether `Iron` is in the table; the
  literal's first word either matches an entry or does not. An absent shape leaves index 0, which is
  what the item constructor's own default byte does. This is what keeps the story from encoding a
  belief about the shipped file's contents. (FR-3, AC-5)

- **DD-5 — A ranged weapon is refused at RESOLUTION, so the fold has one arm.** `ResolveWeapon`
  rejects an attack type of 10 or above, naming the row. The alternative — a fold with a second arm
  that drops the damage on the floor — would give a bow-armed hero a silent zero, which is the exact
  defect this story exists to remove. Precedent: `NewUnitDef` already refuses the two damage-routing
  arms it does not model. (FR-4, AC-7, P-3)

- **DD-6 — The party takes the ORDINARY chargen arm, and the Medium is carried.** `HERO-START-039`
  grades *"a shipped single-player campaign takes the skill-10 arm"* Medium, resting on the absence of
  an immediate-2 writer rather than on reading the menu chain. Both arms' five fighter literals are
  written down, so the high arm is one constant away; only the ordinary one is used. The owner's own
  sheet listing the sword at 5-8 is the ordinary arm's blade weapon, which is corroboration from the
  game and is recorded as such, not as the basis. (FR-6)

- **DD-7 — `PartyMember` carries a hero and a weapon POINTER, and the fold runs in the start.** A
  pointer is what tells "bare" apart from "a weapon whose numbers are all zero", and the two differ:
  a weapon **assigns** the cadence, so a zero-valued weapon would give a hero a cadence of `0 / 0`.
  Nil is bare. The derive runs where the entity is built, so a member's numbers cannot be set by a
  caller and then disagree with its hero. (FR-1, FR-5)

- **DD-8 — The zero value is the answer and not a fallback.** A zero hero with a nil weapon derives
  five zeroes and the cadence `8 / 4` — byte for byte what the party carries today — through the same
  arithmetic every other hero takes. No branch tests for it. This is what makes every existing
  placement, digest and round-trip test pass unchanged, and it is why the change to `start.go` is
  small. (FR-1, AC-13, AC-18)

- **DD-9 — `LoadTable` keeps its signature; a new `LoadDefinitions` beside it does the extra work.**
  The definition table is parsed once and yields both the placement collections and the resolved
  starting weapon (P-4). `LoadTable` becomes a two-line wrapper, so its seven call sites and every
  synthetic fixture behind them are untouched — a fixture that carries no `Weapons` rows simply gets
  no weapon, which is a state this story models. (FR-9, P-4, AC-18)

- **DD-10 — A weapon that will not resolve is CARRIED, not fatal.** `FontErr`'s precedent: the front
  end opens every map and plays, and the reason travels beside the nil. Fatal was considered and
  rejected — the two shipped roots differ byte for byte in `Data.bin`, and an install that opens the
  archive but names its weapons differently must not stop being a game. What makes it non-silent is
  the check line, which reports the band on **every** run rather than only on failure: the band is
  this story's whole deliverable, so burying it would defeat the report. (FR-9, AC-15)

- **DD-11 — `MissionParty` takes the weapon as an argument rather than becoming a method.**
  `cmd/missionrun` calls it too, and a method would have tied the mission tool to a whole front end.
  A function of the weapon keeps both callers one line long. (FR-6, FR-10)

- **DD-12 — `math.Pow`, with the truncation margin ASSERTED.** The image calls the CRT `pow` on the
  double nearest 1.1; two implementations may differ by an ULP, and these values are truncated and
  then hashed. Rather than assume, a test sweeps Body and Reaction over `0..100` and requires every
  term to stand at least `1e-6` from an integer boundary. The measured worst is `8.98e-3` on the
  damage term and `2.05e-4` on the to-hit term — a margin of order `1e12` ULPs. (P-1, AC-16)

- **DD-13 — The damage pair is byte-wide and WRAPS, because the image's stores are byte-wide.**
  `ftol` produces a 32-bit value and the store is `MOV byte ptr`, on the weapon's fill and on the
  actor's derive alike. Modelling the width is one conversion; refusing the overflow instead would
  invent a rule the original does not have. No shipped weapon and no legal stat reaches it, which the
  margin test's own range shows. (FR-3, FR-2)

- **DD-14 — The attack instrument is a FLAG on `missionrun`, not a new tool.** It already resolves a
  reference to an entity id, already drives a world to a tick ceiling and already prints an outcome;
  an attack is one command on the queue it already builds. A second binary would have duplicated
  every one of those. Default behaviour with no flag is byte-identical. (FR-10, AC-17)

## Success criteria

- **SC-1** The fold reproduces the four measured band edges and their step points, and a bare hero's
  four values, from `pkg/data` alone. (FR-2, FR-5; AC-2, AC-3, AC-4)
- **SC-2** A weapon literal resolves out of a synthetic definition table to the scaled pair, through
  the product of two factors, with the spread subtracting the rounded base. (FR-3; AC-5, AC-6, AC-9)
- **SC-3** Both refusals fire and name what they refused. (FR-4; AC-7, AC-8, P-3)
- **SC-4** The cap, the stat census and the skill's asymmetry hold. (FR-2; AC-10, AC-11, AC-12)
- **SC-5** A started party carries the derived numbers; the zero-value member carries today's; the
  world round-trips at version 12. (FR-1, FR-7, FR-8; AC-1, AC-13, AC-14, AC-18)
- **SC-6** The front end resolves the weapon from the one parse and states the band, or states why
  there is none. (FR-9; AC-15, P-4)
- **SC-7** The margin sweep passes over the whole legal stat range. (P-1; AC-16)
- **SC-8** The mission tool orders an attack and reports the fall; with no flag its output is
  unchanged. (FR-10; AC-17)

## Risks

- **R-1 — The shape/material split rule is ours and could be wrong on a name no literal exercises.**
  Bounded by DD-3's falsifier: all five ordinary literals are resolved against both installed roots
  in the build stage, and a failure is reported rather than absorbed. It cannot reach a placed unit,
  which resolves by class key and never by name.
- **R-2 — A wrong ladder would reproduce a plausible number.** This is the defect that produced the
  retractions. Mitigated by choosing a check that DISCRIMINATES: the owner's four bands are 7-10 /
  8-12 / 9-14 / 10-16 under the corrected ladder and 25-42 / 26-44 / 27-46 / 28-48 under the rival,
  so agreement is evidence and not a fit. The figures are re-derived from the install, never typed
  in.
- **R-3 — `LoadDefinitions` could diverge from `LoadTable`.** Removed rather than mitigated:
  `LoadTable` is implemented as a call to `LoadDefinitions`, so there is one parse and one path.
- **R-4 — The party's numbers reach hashed state, so a change moves every digest a test pins.** The
  zero-value member derives today's numbers (DD-8), so only a test that supplies a hero moves. Every
  existing digest, round-trip and placement test is run unchanged as the check.
