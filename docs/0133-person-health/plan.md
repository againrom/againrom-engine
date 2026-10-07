# plan — 0133

## Approach

The derived-stat graph already computes what FR-1 asks for; the placement arm does not call it. So
the change is two small ones and a correction between them: the graph's health arm stops consulting
the column flag and starts gating on its own product (FR-4); the definition gains one accessor that
runs the graph for its own row (FR-2); the placement arm mints from that accessor instead of from
the streamed column (FR-1, FR-3). Everything else is the report that makes the result visible
(FR-9, FR-10) and the re-pinning of the tests that assert today's numbers.

## Facts verified during planning

* The placement arm's person rung mints the entity's health pair from the definition's streamed
  maximum, in one composite literal, and nothing writes the pair again. The difficulty adjustment is
  a function of a **creature** definition, so it cannot reach a person; the party mint is a separate
  loop over members no map places.
* The graph's health arm gates its first term on the column flag and lets the other two stand; its
  mana arm gates its whole derivation on the mana column. The two arms are neighbours in one
  function.
* The column flag has exactly one reader outside the graph: the party mint, which uses it — ahead of
  a positivity test — to keep a member built off no row at the party's own starting health.
* The graph's statistic ceiling is a flat 50 and the graph carries no modifier term, which is what a
  placement needs and no more.
* A definition already exposes its character (statistics and skills) and its combat block; the
  combat accessor is itself defined in terms of the graph, so the graph is already this collection's
  only source for a derived number.
* **The definition's profile expression already sets the class flag exactly when the row states no
  mana column**, and the graph's health arm already doubles on that flag. FR-5 therefore describes
  the shipped condition and the shipped direction, and needs no change to either. What is stale is
  the expression's own doc, which says the reading is authored and that no source decides it: since
  `HERO-CLASS-013`'s superseding entry and `HERO-HP-072`, the condition is decoded for the placement
  path — the flag is set at spawn by the mana column being positive, and the doubling belongs to the
  arm where it is clear.
* The graph's health arm reads nothing off a loadout. A derived maximum therefore needs no
  equipment argument, and a caller that passes an empty one is passing the only thing it could.
* The class-dump tool's definition-table verb resolves every placement of a named map, builds the
  world once, and prints a per-placement line whose health maximum and speed are read off the
  entity. It has the owning slot in hand on the same entity and does not print it.
* The serialized simulation form carries a health pair already; no field is added and no version
  constant is spent (FR-11).
* Seven landed tests assert what this story changes, measured by applying it and reverting it: five
  in the placement package, one in the definition package, one in the class-dump tool. Two of the
  seven move on the graph's correction alone; the other five move on the placement arm's.

## Files to touch

| Path | Intent |
|---|---|
| `pkg/data/recompute.go` | MODIFY — the health arm's gate; the profile type's and the graph's own doc |
| `pkg/data/humandef.go` | MODIFY — add the definition's derived-maximum accessor |
| `pkg/data/chargenbase.go` | MODIFY — doc only: what the class-flag expression now rests on |
| `pkg/data/recompute_test.go` | MODIFY |
| `pkg/mapload/fromalm.go` | MODIFY — the person rung's health |
| `pkg/mapload/start.go` | MODIFY — doc only: the party mint's own gate, whose reason changes |
| `pkg/mapload/fromalm_test.go`, `human_test.go`, `pools_test.go` | MODIFY |
| `cmd/classdump/databin.go` | MODIFY — the owning slot on the placement line; the collection census |
| `cmd/classdump/databin_test.go` | MODIFY |

## Design decisions

* **DD-1 — the derivation is a method on the definition, not an expression in the loader.** The
  definition already answers "what character is this row" and "what combat numbers does this row
  have", the second of them through the graph. A third question of the same shape belongs beside
  them. *Rejected:* building the character and the profile inside the placement arm — it would put
  the graph's inputs in a loader and create a second site that has to agree with the definition's
  own combat accessor about which profile a row carries (FR-2).
* **DD-2 — that method runs the graph with the definition's own profile expression**, the same one a
  generated character goes through, and takes no equipment argument. That expression is what carries
  FR-5: its class flag is already the mana column's absence, so FR-5 is met by calling it rather than
  by a change to the multiplier's condition, and the only edit the expression needs is to its doc.
  *Rejected:* a second profile builder for the placement path, which is exactly the two-expressions
  failure FR-2 forbids; and re-keying the health arm's multiplier off the mana-column flag directly,
  which would give the same numbers here while silently changing every caller that passes a profile
  built some other way.
* **DD-3 — the health arm's gate becomes its own product and covers all three terms.** *Rejected:*
  keeping the column flag and adding the product as a second condition — it would preserve a
  dependence the contract says must not exist (FR-4) and would still derive a maximum from
  experience alone for a character with no row.
* **DD-4 — the column flag stays on the profile and keeps its one reader.** *Rejected:* removing the
  field, which would leave the party mint with a positivity test alone; under the corrected arm a
  member with no row derives a large positive maximum, so that test would mint him at it and break
  FR-8. The field's meaning narrows from "the arm's gate" to "the row states a maximum at all", and
  its doc says so.
* **DD-5 — the mana arm is not touched and a person's mana pair stays the row's column.**
  *Rejected:* deriving the mana maximum in the same change. The same routine does derive it, and its
  own column gate is genuinely there — but the reading that licenses it would be taken sideways off a
  claim published about health, and it would move every caster's pool on this story's evidence rather
  than on its own. The cost of not doing it is named in `spec.md`'s out-of-scope list rather than
  hidden: a person's two pools come from two different places until a story closes it.
* **DD-6 — the collection census lives in the definition-table verb's own table report**, so it runs
  whether or not a map is named. *Rejected:* a new command over the same collection — a second tool
  reading the same rows is a second answer waiting to disagree.
* **DD-7 — the owning slot is read off the built entity**, beside the health and the speed the line
  already reads off it. *Rejected:* reading it off the map record — the line's other numbers come
  off the world, and a field that did not would be free to disagree with them about which placement
  it describes.
* **DD-8 — no characterization tasks are added.** The brownfield overlay asks that every changed
  unit be pinned before it changes; the six tests named above are exactly those pins, authored in
  earlier stories in contexts that could not have known this diff. Each is re-pinned at the
  corrected number with the reason it moved, and none is deleted. *This is a recorded deviation from
  the overlay's "pin before change" ordering*, taken because writing new tests to freeze a behaviour
  already frozen — and already known to be wrong — would assert something no one intends to keep.
* **DD-9 — the census reports figures that discriminate rather than figures that describe.** The two
  sums, the three counts and the largest-ratio row are chosen because each comes out visibly wrong
  under an inverted class multiplier, a gate in the wrong place, or a dropped truncation. The
  zero-column rule FR-10 states is held by taking the ratio comparison over rows with a positive
  column only, and counting the excluded rows beside it. *Rejected:* treating a zero column as a
  ratio of infinity, which would make one row the permanent answer and hide the collection.

## Risks

* **R-1 — the correction moves hashed simulation state for every person on every map.** That is the
  intent, and it is also the largest blast radius in the story. *Mitigation:* the census (FR-10) is
  run against both lawful roots and compared against the figures `HERO-HP-072` publishes for the same
  collection — two sums, a largest ratio and the one row on which the two roots disagree. Agreement
  is a strong discriminator; disagreement stops the story before it lands.
* **R-2 — the graph is shared with the party path.** A generated character with no base row derives a
  real maximum where he derived 2. *Mitigation:* FR-8 keeps the mint on the column flag rather than
  on positivity, and AC-7 witnesses it.
* **R-3 — a person could be born with no health at all** if a shipped row's Body is 0 while its
  column is positive. AC-2 makes that the specified answer, so detecting it settles nothing on its
  own — what is wanted is whether any shipped row reaches it. *Mitigation:* the census's zero-derived
  count is exactly that number. A nonzero count is not a defect this story may resolve by softening
  FR-4; the remedy is to stop and hand the case back for a contract decision, and the count is
  reported either way so the answer is on the record rather than assumed.
* **R-4 — the class multiplier's direction.** Inverted, it would move nearly every person's maximum
  by a factor close to two while still looking plausible, and no unit test over one profile can see
  it. *Mitigation:* the same census against `HERO-HP-072`'s published figures; the largest ratio and
  both sums separate the two readings, and an inverted multiplier misses both sums by thousands.

## Success criteria

| # | Condition | Kind |
|---|---|---|
| SC-1 | a test minting a person from a row whose column and derived maximum differ finds the derived one on both halves of the pair | automated |
| SC-2 | a test deriving a zero-Body character finds a maximum of 0 | automated |
| SC-3 | a test deriving one character under both settings of the column flag finds one number | automated |
| SC-4 | a test deriving a nonzero-Body character whose column is 0 finds a positive maximum | automated |
| SC-5 | the landed difficulty test still finds one health across all three settings | automated |
| SC-6 | the landed creature and unresolved-placement tests still find their own values | automated |
| SC-7 | the landed party test still mints a member with no row at the party's starting health | automated |
| SC-8 | a test placing a short row finds the build refused | automated |
| SC-9 | a test deriving one definition twice finds the same number | automated |
| SC-10 | the definition-table report over mission 20, on both roots, prints each person's derived maximum and the slot owning it | developer-run |
| SC-11 | the collection census, on both roots, prints two sums, three counts and a largest-ratio row, and they agree with what research published | developer-run |
| SC-12 | build, vet, gofmt, the full test run, and the four repository check scripts | release-gate |

## Traceability

FR-1 → DD-1, DD-2 → SC-1, SC-10 · FR-2 → DD-1, DD-2 → SC-1 · FR-3 → DD-1 → SC-1 · FR-4 → DD-3 →
SC-2, SC-3, SC-4 · FR-5 → DD-2 (`chargenbase.go`) → SC-11 · FR-6 → SC-5 · FR-7 → SC-6 · FR-8 → DD-4
(`start.go`) → SC-7 ·
FR-9 → DD-7 → SC-10 · FR-10 → DD-6, DD-9 → SC-11 · FR-11 → SC-12 · P-1 → DD-1 → SC-1 · P-2 → SC-8 ·
P-3 → SC-9 · P-4 → DD-1 → SC-10.
